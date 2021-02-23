package binloader_test

import (
	"fmt"
	"github.com/nickalie/go-binloader"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func ExampleNewBinLoader() {
	base := "https://storage.googleapis.com/downloads.webmproject.org/releases/webp/"

	bin := binloader.NewBinLoader().
		Src(
		binloader.NewSrc().
				URL(base + "libwebp-0.6.0-mac-10.12.tar.gz").
				Os("darwin")).
		Src(
		binloader.NewSrc().
				URL(base + "libwebp-0.6.0-linux-x86-32.tar.gz").
				Os("linux").
				Arch("x86")).
		Src(
		binloader.NewSrc().
				URL(base + "libwebp-0.6.0-linux-x86-64.tar.gz").
				Os("linux").
				Arch("x64")).
		Src(
		binloader.NewSrc().
				URL(base + "libwebp-0.6.0-windows-x64.zip").
				Os("win32").
				Arch("x64").
				ExecPath("cwebp.exe")).
		Src(
		binloader.NewSrc().
				URL(base + "libwebp-0.6.0-windows-x86.zip").
				Os("win32").
				Arch("x86").
				ExecPath("cwebp.exe")).
		Strip(2).
		Dest("vendor/cwebp").
		ExecPath("cwebp")

	file, err := bin.Path()

	fmt.Printf("file: %s\n", file)
	fmt.Printf("err: %v\n", err)
}

func TestNewBinWrapperNoError(t *testing.T) {
	base := "https://storage.googleapis.com/downloads.webmproject.org/releases/webp/"

	bin := binloader.NewBinLoader().
		Src(
		binloader.NewSrc().
				URL(base + "libwebp-0.6.0-mac-10.12.tar.gz").
				Os("darwin")).
		Src(
		binloader.NewSrc().
				URL(base + "libwebp-0.6.0-linux-x86-32.tar.gz").
				Os("linux").
				Arch("x86")).
		Src(
		binloader.NewSrc().
				URL(base + "libwebp-0.6.0-linux-x86-64.tar.gz").
				Os("linux").
				Arch("x64")).
		Src(
		binloader.NewSrc().
				URL(base + "libwebp-0.6.0-windows-x64.zip").
				Os("win32").
				Arch("x64")).
		Src(
		binloader.NewSrc().
				URL(base + "libwebp-0.6.0-windows-x86.zip").
				Os("win32").
				Arch("x86")).
		Strip(2).
		Dest("deps/cwebp").
		ExecPath("cwebp").AutoExe()

	file, err := bin.Path()
	assert.Nil(t, err)
	_, err = os.Stat(file)
	assert.Nil(t, err)
}
