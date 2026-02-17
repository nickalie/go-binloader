package binloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mholt/archives"
)

// Src defines executable source
type Src struct {
	url      string
	os       string
	arch     string
	execPath string
}

// BinLoader downloads and extracts platform-specific binaries.
type BinLoader struct {
	src      []*Src
	dest     string
	execPath string
	strip    int
	autoExe  bool
	mu       sync.Mutex
	loaded   bool
}

// NewSrc creates new Src instance
func NewSrc() *Src {
	return &Src{}
}

// URL sets a url pointing to a file to download.
func (s *Src) URL(value string) *Src {
	s.url = value
	return s
}

// Os tie the source to a specific OS. Possible values are same as runtime.GOOS
func (s *Src) Os(value string) *Src {
	s.os = value
	return s
}

// Arch tie the source to a specific arch. Possible values are same as runtime.GOARCH
func (s *Src) Arch(value string) *Src {
	s.arch = value
	return s
}

// ExecPath tie the src to a specific binary file
func (s *Src) ExecPath(value string) *Src {
	s.execPath = value
	return s
}

// NewBinLoader creates BinLoader instance
func NewBinLoader() *BinLoader {
	return &BinLoader{}
}

// Src adds a Src to BinLoader
func (b *BinLoader) Src(src *Src) *BinLoader {
	b.src = append(b.src, src)
	return b
}

// Dest accepts a path which the files will be downloaded to
func (b *BinLoader) Dest(dest string) *BinLoader {
	b.dest = dest
	return b
}

// Strip strips a number of leading paths from file names on extraction.
func (b *BinLoader) Strip(value int) *BinLoader {
	b.strip = value
	return b
}

// Path returns the full path to the binary.
// It is safe for concurrent use; the download/extraction is performed at most once.
// If the download fails with a transient error, subsequent calls will retry.
func (b *BinLoader) Path() (string, error) {
	if len(b.src) > 0 {
		b.mu.Lock()
		if !b.loaded {
			err := b.findExisting()
			if err != nil {
				b.mu.Unlock()
				return "", err
			}
			b.loaded = true
		}
		b.mu.Unlock()
	}

	return b.path(), nil
}

func (b *BinLoader) path() string {
	execPath := b.execPath

	src := osFilterObj(b.src)
	if src != nil && src.execPath != "" {
		execPath = src.execPath
		if b.autoExe && runtime.GOOS == "windows" {
			ext := strings.ToLower(filepath.Ext(execPath))
			if ext != ".exe" {
				execPath += ".exe"
			}
		}
	}

	if b.dest == "." {
		return b.dest + string(filepath.Separator) + execPath
	}

	return filepath.Join(b.dest, execPath)
}

// ExecPath define a file to use as the binary
func (b *BinLoader) ExecPath(execPath string) *BinLoader {

	if b.autoExe && runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(execPath))

		if ext != ".exe" {
			execPath += ".exe"
		}
	}

	b.execPath = execPath
	return b
}

// AutoExe adds .exe extension for windows executable path
func (b *BinLoader) AutoExe() *BinLoader {
	b.autoExe = true
	return b.ExecPath(b.execPath)
}

func (b *BinLoader) findExisting() error {
	_, err := os.Stat(b.path())

	if os.IsNotExist(err) {
		return b.download()
	} else if err != nil {
		return err
	} else {
		return nil
	}
}

func (b *BinLoader) download() error {
	src := osFilterObj(b.src)

	if src == nil {
		return errors.New("no binary found matching your system, it's probably not supported")
	}

	file, err := b.downloadFile(src.url)

	if err != nil {
		return err
	}

	err = b.extractFile(file)

	if err != nil {
		return err
	}

	if src.execPath != "" {
		b.ExecPath(src.execPath)
	}

	return nil
}

func (b *BinLoader) extractFile(file string) error {
	defer os.Remove(file)

	ctx := context.Background()

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	format, stream, err := archives.Identify(ctx, file, f)
	if err != nil {
		return fmt.Errorf("%s: failed to identify archive format: %w", file, err)
	}

	extractor, ok := format.(archives.Extractor)
	if !ok {
		return fmt.Errorf("%s: format does not support extraction", file)
	}

	err = extractor.Extract(ctx, stream, b.handleExtractedFile)

	if err != nil {
		return err
	}

	if b.strip == 0 {
		return nil
	}

	return b.stripDir()
}

func (b *BinLoader) safePath(nameInArchive string) (string, error) {
	destPath := filepath.Join(b.dest, filepath.Clean(nameInArchive))

	// Resolve the dest directory to its real path (following symlinks) so that
	// the prefix check cannot be bypassed by symlinked path components.
	absDest, err := filepath.Abs(b.dest)
	if err != nil {
		return "", err
	}
	if realDest, err := filepath.EvalSymlinks(absDest); err == nil {
		absDest = realDest
	}

	absPath, err := filepath.Abs(destPath)
	if err != nil {
		return "", err
	}
	// Resolve the target path: if intermediate components are symlinks,
	// the real path may fall outside the destination directory.
	if realPath, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = realPath
	} else {
		// Parts of the path may not exist yet (we're about to create them).
		// Walk up to find the deepest existing ancestor, resolve symlinks
		// there, then re-append the non-existent remainder. This catches
		// symlinks at any ancestor level, even when multiple child
		// directories are missing.
		resolved, resolveErr := resolveExistingPrefix(absPath)
		if resolveErr != nil {
			return "", fmt.Errorf("illegal file path in archive: %s: %w", nameInArchive, resolveErr)
		}
		absPath = resolved
	}

	if !strings.HasPrefix(absPath, absDest+string(filepath.Separator)) && absPath != absDest {
		return "", fmt.Errorf("illegal file path in archive: %s", nameInArchive)
	}

	// Return the resolved absolute path rather than the original joined path.
	// This narrows the TOCTOU window: an attacker would need to race between
	// this return and the caller's use of the path. The gap cannot be fully
	// closed without OS-level primitives (e.g. openat2 RESOLVE_NO_SYMLINKS).
	return absPath, nil
}

// resolveExistingPrefix walks up from path to find the deepest existing
// ancestor, resolves symlinks on that ancestor, and re-appends the
// remaining (non-existent) components. This ensures symlinks at any level
// of the path are detected, even when deeper components don't yet exist.
// Returns an error if a dangling symlink is encountered (Lstat succeeds
// but EvalSymlinks fails), which would indicate a potential path traversal.
func resolveExistingPrefix(path string) (string, error) {
	dir := path
	var remaining []string
	for {
		_, err := os.Lstat(dir)
		if err == nil {
			resolved, err := filepath.EvalSymlinks(dir)
			if err != nil {
				return "", fmt.Errorf("dangling symlink in path: %s: %w", dir, err)
			}
			parts := append([]string{resolved}, remaining...)
			return filepath.Join(parts...), nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("cannot access path %s: %w", dir, err)
		}
		remaining = append([]string{filepath.Base(dir)}, remaining...)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return path, nil
}

func (b *BinLoader) handleExtractedFile(_ context.Context, info archives.FileInfo) error {
	destPath, err := b.safePath(info.NameInArchive)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return os.MkdirAll(destPath, 0755)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	srcReader, err := info.Open()
	if err != nil {
		return err
	}
	defer srcReader.Close()

	destFile, err := os.OpenFile(destPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(destFile, srcReader)
	closeErr := destFile.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func (b *BinLoader) stripDir() error {
	dir, dirsToRemove, err := b.findStrippedDir()
	if err != nil {
		return err
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, v := range files {
		if err := os.Rename(filepath.Join(dir, v.Name()), filepath.Join(b.dest, v.Name())); err != nil {
			return err
		}
	}

	for _, v := range dirsToRemove {
		if err := os.RemoveAll(v); err != nil {
			return fmt.Errorf("failed to clean up directory %s: %w", v, err)
		}
	}

	return nil
}

func (b *BinLoader) findStrippedDir() (dir string, dirsToRemove []string, err error) {
	dir = b.dest

	for i := 0; i < b.strip; i++ {
		var files []os.DirEntry
		files, err = os.ReadDir(dir)
		if err != nil {
			return
		}

		nextDir := ""
		for _, v := range files {
			if !v.IsDir() {
				continue
			}
			nextDir = filepath.Join(dir, v.Name())
			dirsToRemove = append(dirsToRemove, nextDir)
			dir = nextDir
			break
		}

		if nextDir == "" {
			err = fmt.Errorf("strip level %d: no subdirectory found in %s", i+1, dir)
			return
		}
	}

	return
}

func fileNameFromURL(rawURL string) (string, error) {
	fileURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	if fileURL.Scheme != "http" && fileURL.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme: %s", fileURL.Scheme)
	}

	name := filepath.Base(fileURL.Path)
	if name == "" || name == "." || name == "/" {
		return "", fmt.Errorf("could not determine filename from URL: %s", rawURL)
	}

	return name, nil
}

func writeToFile(fileName string, r io.Reader) error {
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(file, r)
	closeErr := file.Close()
	if copyErr != nil {
		os.Remove(fileName)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(fileName)
		return closeErr
	}

	return nil
}

// httpClient is the HTTP client used for downloads, with a default timeout.
var httpClient = &http.Client{
	Timeout: 5 * time.Minute,
}

func (b *BinLoader) downloadFile(value string) (string, error) {
	if b.dest == "" {
		b.dest = "."
	}

	if err := os.MkdirAll(b.dest, 0755); err != nil {
		return "", err
	}

	name, err := fileNameFromURL(value)
	if err != nil {
		return "", err
	}
	fileName := filepath.Join(b.dest, name)

	resp, err := httpClient.Get(value)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unable to download %s: HTTP %d", value, resp.StatusCode)
	}

	return fileName, writeToFile(fileName, resp.Body)
}
