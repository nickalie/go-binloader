# go-binloader

[![CI](https://github.com/nickalie/go-binloader/actions/workflows/ci.yml/badge.svg)](https://github.com/nickalie/go-binloader/actions/workflows/ci.yml)

A Go library for downloading and extracting platform-specific binaries. Automatically selects the correct binary for the current OS and architecture, downloads it, and extracts it from an archive.

## Installation

```bash
go get github.com/nickalie/go-binloader
```

## Usage

```go
package main

import (
	"fmt"
	"github.com/nickalie/go-binloader"
)

func main() {
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
		ExecPath("cwebp").
		AutoExe()

	file, err := bin.Path()
	if err != nil {
		panic(err)
	}

	fmt.Println(file)
}
```

## API

### Src

Defines a download source for a specific platform.

| Method | Description |
|--------|-------------|
| `URL(value string)` | URL pointing to the archive to download |
| `Os(value string)` | Target OS (`darwin`, `linux`, `windows`, `win32`) |
| `Arch(value string)` | Target architecture (`x86`, `x64`, `386`, `amd64`, `arm64`, etc.) |
| `ExecPath(value string)` | Path to the executable within the archive for this source |

OS values match `runtime.GOOS`, with `win32` as an alias for `windows`. Architecture values match `runtime.GOARCH`, with `x86` as an alias for `386` and `x64` for `amd64`.

### BinLoader

Downloads and extracts platform-specific binaries.

| Method | Description |
|--------|-------------|
| `Src(src *Src)` | Add a download source |
| `Dest(dest string)` | Directory to download and extract files into |
| `ExecPath(execPath string)` | Name of the executable binary |
| `Strip(value int)` | Strip leading path components from extracted files (like `tar --strip-components`) |
| `AutoExe()` | Automatically append `.exe` on Windows |
| `Path()` | Returns the full path to the binary, downloading if needed |

All methods (except `Path`) return the receiver for chaining.

## How It Works

1. `Path()` checks if the binary already exists at the destination
2. If not, it selects the matching `Src` for the current OS and architecture
3. Downloads the archive from the URL
4. Extracts the archive (supports tar.gz, zip, and other formats via [mholt/archives](https://github.com/mholt/archives))
5. Optionally strips leading directory components
6. Returns the path to the executable

Subsequent calls to `Path()` skip the download if the binary already exists.

## License

MIT
