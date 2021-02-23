package binloader

import (
	"errors"
	"fmt"
	"github.com/mholt/archiver"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Src defines executable source
type Src struct {
	url      string
	os       string
	arch     string
	execPath string
}

type BinLoader struct {
	src      []*Src
	dest     string
	execPath string
	strip    int
	autoExe  bool
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

// Path returns the full path to the binary
func (b *BinLoader) Path() (string, error) {
	if b.src != nil && len(b.src) > 0 {
		err := b.findExisting()

		if err != nil {
			return "", err
		}
	}

	return b.path(), nil
}

func (b *BinLoader) path() string {
	src := osFilterObj(b.src)

	if src != nil && src.execPath != "" {
		b.ExecPath(src.execPath)
	}

	if b.dest == "." {
		return b.dest + string(filepath.Separator) + b.execPath
	}

	return filepath.Join(b.dest, b.execPath)
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
		fmt.Printf("%s not found. Downloading...\n", b.path())
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
		return errors.New("No binary found matching your system. It's probably not supported")
	}

	file, err := b.downloadFile(src.url)

	if err != nil {
		return err
	}

	fmt.Printf("%s downloaded. Trying to extract...\n", file)

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
	err := archiver.Unarchive(file, b.dest)

	if err != nil {
		fmt.Printf("%s is not an archive or have unsupported archive format\n", file)
		return err
	}

	if b.strip == 0 {
		return nil
	}

	return b.stripDir()
}

func (b *BinLoader) stripDir() error {
	dir := b.dest

	var dirsToRemove []string

	for i := 0; i < b.strip; i++ {
		files, err := ioutil.ReadDir(dir)

		if err != nil {
			return err
		}

		for _, v := range files {
			if v.IsDir() {

				if dir != b.dest {
					dirsToRemove = append(dirsToRemove, dir)
				}

				dir = filepath.Join(dir, v.Name())
				break
			}
		}
	}

	files, err := ioutil.ReadDir(dir)

	if err != nil {
		return err
	}

	for _, v := range files {
		err := os.Rename(filepath.Join(dir, v.Name()), filepath.Join(b.dest, v.Name()))

		if err != nil {
			return err
		}
	}

	for _, v := range dirsToRemove {
		os.RemoveAll(v)
	}

	return nil
}

func (b *BinLoader) downloadFile(value string) (string, error) {
	if b.dest == "" {
		b.dest = "."
	}

	err := os.MkdirAll(b.dest, 0755)

	if err != nil {
		return "", err
	}

	fileURL, err := url.Parse(value)

	if err != nil {
		return "", err
	}

	path := fileURL.Path

	segments := strings.Split(path, "/")
	fileName := segments[len(segments)-1]
	fileName = filepath.Join(b.dest, fileName)
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)

	if err != nil {
		return "", err
	}

	defer file.Close()

	check := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	resp, err := check.Get(value)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if !(resp.StatusCode >= 200 && resp.StatusCode < 400) {
		return "", errors.New("Unable to download " + value)
	}

	_, err = io.Copy(file, resp.Body)

	return fileName, err
}
