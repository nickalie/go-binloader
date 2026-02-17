package binloader

import (
	"runtime"
	"slices"
)

func osFilterObj(values []*Src) *Src {
	platforms := currentPlatforms()
	arches := currentArches()

	for _, v := range values {
		if matchesSrc(v, platforms, arches) {
			return v
		}
	}

	return nil
}

func matchesSrc(v *Src, platforms, arches []string) bool {
	matchOS := v.os == "" || slices.Contains(platforms, v.os)
	matchArch := v.arch == "" || slices.Contains(arches, v.arch)
	return matchOS && matchArch
}

func currentPlatforms() []string {
	platforms := []string{runtime.GOOS}
	if runtime.GOOS == "windows" {
		platforms = append(platforms, "win32")
	}
	return platforms
}

func currentArches() []string {
	arches := []string{runtime.GOARCH}

	switch runtime.GOARCH {
	case "386":
		arches = append(arches, "x86")
	case "amd64":
		arches = append(arches, "x64")
	}

	return arches
}
