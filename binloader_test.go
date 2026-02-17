package binloader_test

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nickalie/go-binloader"
	"github.com/stretchr/testify/assert"
)

func ExampleNewBinLoader() {
	base := "https://storage.googleapis.com/downloads.webmproject.org/releases/webp/"

	bin := binloader.NewBinLoader().
		Src(
			binloader.NewSrc().
				URL(base + "libwebp-1.6.0-mac-arm64.tar.gz").
				Os("darwin").
				Arch("arm64")).
		Src(
			binloader.NewSrc().
				URL(base + "libwebp-1.6.0-mac-x86-64.tar.gz").
				Os("darwin").
				Arch("x64")).
		Src(
			binloader.NewSrc().
				URL(base + "libwebp-1.6.0-linux-x86-64.tar.gz").
				Os("linux").
				Arch("x64")).
		Src(
			binloader.NewSrc().
				URL(base + "libwebp-1.6.0-linux-aarch64.tar.gz").
				Os("linux").
				Arch("arm64")).
		Src(
			binloader.NewSrc().
				URL(base + "libwebp-1.6.0-windows-x64.zip").
				Os("win32").
				Arch("x64").
				ExecPath("cwebp.exe")).
		Strip(2).
		Dest("vendor/cwebp").
		ExecPath("cwebp")

	file, err := bin.Path()

	fmt.Printf("file: %s\n", file)
	fmt.Printf("err: %v\n", err)
}

func TestNewBinWrapperNoError(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "cwebp")
	base := "https://storage.googleapis.com/downloads.webmproject.org/releases/webp/"

	bin := binloader.NewBinLoader().
		Src(
			binloader.NewSrc().
				URL(base + "libwebp-1.6.0-mac-arm64.tar.gz").
				Os("darwin").
				Arch("arm64")).
		Src(
			binloader.NewSrc().
				URL(base + "libwebp-1.6.0-mac-x86-64.tar.gz").
				Os("darwin").
				Arch("x64")).
		Src(
			binloader.NewSrc().
				URL(base + "libwebp-1.6.0-linux-x86-64.tar.gz").
				Os("linux").
				Arch("x64")).
		Src(
			binloader.NewSrc().
				URL(base + "libwebp-1.6.0-linux-aarch64.tar.gz").
				Os("linux").
				Arch("arm64")).
		Src(
			binloader.NewSrc().
				URL(base + "libwebp-1.6.0-windows-x64.zip").
				Os("win32").
				Arch("x64")).
		Strip(2).
		Dest(dest).
		ExecPath("cwebp").AutoExe()

	file, err := bin.Path()
	assert.Nil(t, err)

	output, err := exec.Command(file, "-version").CombinedOutput()
	assert.Nil(t, err)
	assert.True(t, strings.Contains(string(output), "1.6.0"), "expected version 1.6.0 in output: %s", string(output))
}
