// Package buildinfo exposes the stable identity of an agent-env executable.
package buildinfo

import (
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/mahcialet/agent-env/internal/assets"
)

// These variables are populated by release builds with -ldflags. Development
// binaries fall back to embedded Go VCS settings and otherwise retain honest
// unknown defaults. No runtime Git invocation or checkout discovery is needed.
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
	Version   string             `json:"version"`
	Commit    string             `json:"commit"`
	Dirty     string             `json:"dirty"`
	GoVersion string             `json:"go_version"`
	GOOS      string             `json:"goos"`
	GOARCH    string             `json:"goarch"`
	Assets    []assets.AssetInfo `json:"assets"`
}

func Current() Info {
	var settings []debug.BuildSetting
	if build, ok := debug.ReadBuildInfo(); ok {
		settings = build.Settings
	}
	return current(settings)
}

func current(settings []debug.BuildSetting) Info {
	version := strings.TrimSpace(Version)
	commit, dirty := Commit, Dirty
	for _, setting := range settings {
		switch setting.Key {
		case "vcs.revision":
			if (commit == "" || commit == "unknown") && setting.Value != "" {
				commit = setting.Value
			}
		case "vcs.modified":
			if (dirty == "" || dirty == "unknown") && (setting.Value == "true" || setting.Value == "false") {
				dirty = setting.Value
			}
		}
	}
	if commit == "" {
		commit = "unknown"
	}
	if dirty == "" {
		dirty = "unknown"
	}
	if releaseVersion, releaseCommit, ok := ParseReleaseRecord(ReleaseRecord); ok {
		version, commit, dirty = releaseVersion, releaseCommit, "false"
	}
	if version == "" {
		version = "devel"
	}
	return Info{Version: version, Commit: commit, Dirty: dirty, GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, Assets: assets.Inventory()}
}
