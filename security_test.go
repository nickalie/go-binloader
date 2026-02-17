package binloader

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSafePath_DanglingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests not reliable on Windows")
	}

	dest := t.TempDir()
	b := &BinLoader{dest: dest}

	// Create a dangling symlink: dest/a -> /nonexistent/target
	symlinkPath := filepath.Join(dest, "a")
	if err := os.Symlink("/nonexistent/target", symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Attempt to extract a file through the dangling symlink
	_, err := b.safePath("a/b/file")
	if err == nil {
		t.Fatal("expected safePath to reject path through dangling symlink, but it succeeded")
	}
}

func TestSafePath_ValidSymlinkInsideDest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests not reliable on Windows")
	}

	dest := t.TempDir()
	b := &BinLoader{dest: dest}

	// Create a valid symlink that stays inside dest
	subdir := filepath.Join(dest, "realdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	symlinkPath := filepath.Join(dest, "link")
	if err := os.Symlink(subdir, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Path through a valid symlink that resolves inside dest should succeed
	_, err := b.safePath("link/file.txt")
	if err != nil {
		t.Fatalf("expected safePath to allow valid symlink inside dest, got: %v", err)
	}
}

func TestSafePath_SymlinkEscapeViaDanglingAncestor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests not reliable on Windows")
	}

	dest := t.TempDir()
	b := &BinLoader{dest: dest}

	// Create: dest/parent/link -> /nonexistent
	parent := filepath.Join(dest, "parent")
	if err := os.MkdirAll(parent, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	symlinkPath := filepath.Join(parent, "link")
	if err := os.Symlink("/nonexistent", symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Attempt to extract through the dangling symlink
	_, err := b.safePath("parent/link/subdir/file")
	if err == nil {
		t.Fatal("expected safePath to reject path through dangling ancestor symlink, but it succeeded")
	}
}

func TestSafePath_PathTraversal(t *testing.T) {
	dest := t.TempDir()
	b := &BinLoader{dest: dest}

	// Classic path traversal with ..
	_, err := b.safePath("../../../etc/passwd")
	if err == nil {
		t.Fatal("expected safePath to reject path traversal with .., but it succeeded")
	}
}

func TestSafePath_NormalFile(t *testing.T) {
	dest := t.TempDir()
	b := &BinLoader{dest: dest}

	path, err := b.safePath("subdir/file.txt")
	if err != nil {
		t.Fatalf("expected safePath to allow normal path, got: %v", err)
	}

	// safePath returns a resolved absolute path, so resolve dest too
	// (handles platforms where TempDir may contain symlinks, e.g. macOS /tmp -> /private/tmp)
	realDest, err := filepath.EvalSymlinks(dest)
	if err != nil {
		t.Fatalf("failed to resolve dest: %v", err)
	}
	expected := filepath.Join(realDest, "subdir", "file.txt")
	if path != expected {
		t.Fatalf("expected %s, got %s", expected, path)
	}
}

func TestResolveExistingPrefix_DanglingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests not reliable on Windows")
	}

	dir := t.TempDir()

	// Create a dangling symlink
	symlinkPath := filepath.Join(dir, "dangling")
	if err := os.Symlink("/nonexistent/target", symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// resolveExistingPrefix should return error for dangling symlink
	_, err := resolveExistingPrefix(filepath.Join(symlinkPath, "child", "file"))
	if err == nil {
		t.Fatal("expected resolveExistingPrefix to return error for dangling symlink, but it succeeded")
	}
}

func TestResolveExistingPrefix_ValidPath(t *testing.T) {
	dir := t.TempDir()

	// resolveExistingPrefix should succeed for a normal path with non-existent children
	result, err := resolveExistingPrefix(filepath.Join(dir, "nonexistent", "child"))
	if err != nil {
		t.Fatalf("expected resolveExistingPrefix to succeed for valid path, got: %v", err)
	}

	// The result should contain the resolved dir + the non-existent parts
	if !filepath.IsAbs(result) {
		t.Fatalf("expected absolute path, got: %s", result)
	}
}
