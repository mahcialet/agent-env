// Package buildinfo exposes the stable identity of an agent-env executable.
package buildinfo

import (
	"runtime"
	"strings"
)

// These variables are populated by release builds with -ldflags. Development
// binaries intentionally retain honest defaults.
var (
	Version = "devel"
	Commit  = "unknown"
	Dirty   = "unknown"
)

type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Dirty     string `json:"dirty"`
	GoVersion string `json:"go_version"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
}

func Current() Info {
	version := strings.TrimSpace(Version)
	if version == "" {
		version = "devel"
	}
	return Info{Version: version, Commit: Commit, Dirty: Dirty, GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
}
