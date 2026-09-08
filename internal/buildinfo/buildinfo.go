// Package buildinfo exposes the stable identity of an agent-env executable.
package buildinfo

import (
	"regexp"
	"runtime"
	"strings"
)

// These variables are populated by release builds with -ldflags. Development
// binaries intentionally retain honest defaults.
var (
	Version = "devel"
	Commit  = "unknown"
	Dirty   = "unknown"
	// ReleaseRecord binds static artifact verification to the identity Current
	// actually reports, including when -trimpath omits linker flags from build info.
	ReleaseRecord = ""
)

const ReleaseRecordPrefix = "agent-env-release-v1|"

var releaseRecordPattern = regexp.MustCompile(`^agent-env-release-v1\|((0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*))\|([0-9a-f]{40}|[0-9a-f]{64})\|false\|end$`)

// ParseReleaseRecord accepts only complete, canonical clean-release records.
func ParseReleaseRecord(record string) (version, commit string, ok bool) {
	match := releaseRecordPattern.FindStringSubmatch(record)
	if match == nil {
		return "", "", false
	}
	return match[1], match[5], true
}

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
	commit, dirty := Commit, Dirty
	if releaseVersion, releaseCommit, ok := ParseReleaseRecord(ReleaseRecord); ok {
		version, commit, dirty = releaseVersion, releaseCommit, "false"
	}
	if version == "" {
		version = "devel"
	}
	return Info{Version: version, Commit: commit, Dirty: dirty, GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
}
