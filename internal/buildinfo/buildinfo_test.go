package buildinfo

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestCurrentVCSFallback(t *testing.T) {
	oldVersion, oldCommit, oldDirty, oldRecord := Version, Commit, Dirty, ReleaseRecord
	t.Cleanup(func() { Version, Commit, Dirty, ReleaseRecord = oldVersion, oldCommit, oldDirty, oldRecord })
	Version, ReleaseRecord = "devel", ""
	for _, test := range []struct {
		name, commit, dirty, wantCommit, wantDirty string
		settings                                   []debug.BuildSetting
	}{
		{name: "absent", commit: "unknown", dirty: "unknown", wantCommit: "unknown", wantDirty: "unknown"},
		{name: "empty defaults", wantCommit: "unknown", wantDirty: "unknown"},
		{name: "revision only", commit: "unknown", dirty: "unknown", wantCommit: "abc", wantDirty: "unknown", settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc"}}},
		{name: "modified only", commit: "unknown", dirty: "unknown", wantCommit: "unknown", wantDirty: "true", settings: []debug.BuildSetting{{Key: "vcs.modified", Value: "true"}}},
		{name: "invalid modified", wantCommit: "unknown", wantDirty: "unknown", settings: []debug.BuildSetting{{Key: "vcs.modified", Value: "invalid"}, {Key: "vcs.revision", Value: ""}}},
		{name: "partial override", commit: "explicit", dirty: "unknown", wantCommit: "explicit", wantDirty: "true", settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc"}, {Key: "vcs.modified", Value: "true"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			Commit, Dirty = test.commit, test.dirty
			got := current(test.settings)
			if got.Version != "devel" || got.Commit != test.wantCommit || got.Dirty != test.wantDirty {
				t.Fatalf("unexpected identity: %+v", got)
			}
		})
	}
}

func TestCurrentHasHonestDevelopmentIdentity(t *testing.T) {
	got := Current()
	if got.Version != Version || got.GoVersion == "" || got.GOOS == "" || got.GOARCH == "" {
		t.Fatalf("unexpected build info: %+v", got)
	}
}

func TestCurrentReleaseRecord(t *testing.T) {
	oldVersion, oldCommit, oldDirty, oldRecord := Version, Commit, Dirty, ReleaseRecord
	t.Cleanup(func() { Version, Commit, Dirty, ReleaseRecord = oldVersion, oldCommit, oldDirty, oldRecord })
	Version, Commit, Dirty = "devel", "unknown", "unknown"
	for _, length := range []int{40, 64} {
		commit := strings.Repeat("a", length)
		ReleaseRecord = ReleaseRecordPrefix + "0.1.0|" + commit + "|false|end"
		got := Current()
		if got.Version != "0.1.0" || got.Commit != commit || got.Dirty != "false" || got.GoVersion == "" || got.GOOS == "" || got.GOARCH == "" {
			t.Fatalf("unexpected release identity: %+v", got)
		}
	}
}

func TestInvalidReleaseRecordPreservesDevelopmentIdentity(t *testing.T) {
	oldVersion, oldCommit, oldDirty, oldRecord := Version, Commit, Dirty, ReleaseRecord
	t.Cleanup(func() { Version, Commit, Dirty, ReleaseRecord = oldVersion, oldCommit, oldDirty, oldRecord })
	Version, Commit, Dirty = "devel", "unknown", "unknown"
	valid := ReleaseRecordPrefix + "0.1.0|" + strings.Repeat("a", 40) + "|false|end"
	for _, record := range []string{
		"", "invalid", valid + "\n", valid + "extra", "extra" + valid,
		strings.Replace(valid, "v1|", "v2|", 1),
		strings.Replace(valid, "0.1.0", "01.1.0", 1),
		strings.Replace(valid, "0.1.0", "0.01.0", 1),
		strings.Replace(valid, "0.1.0", "0.1.00", 1),
		strings.Replace(valid, "0.1.0", "v0.1.0", 1),
		strings.Replace(valid, "0.1.0", "0.1.0-rc.1", 1),
		strings.Replace(valid, "0.1.0", "0.1.0+build", 1),
		strings.Replace(valid, strings.Repeat("a", 40), strings.Repeat("a", 39), 1),
		strings.Replace(valid, strings.Repeat("a", 40), strings.Repeat("g", 40), 1),
		strings.Replace(valid, "false", "true", 1),
		strings.Replace(valid, "false", "unknown", 1),
		strings.TrimSuffix(valid, "|end"),
	} {
		ReleaseRecord = record
		got := Current()
		if got.Version != "devel" || got.Commit != "unknown" || got.Dirty != "unknown" {
			t.Errorf("invalid record %q changed development identity: %+v", record, got)
		}
	}
	Version, Commit, Dirty, ReleaseRecord = "custom", "custom-commit", "true", ""
	if got := Current(); got.Version != Version || got.Commit != Commit || got.Dirty != Dirty {
		t.Fatalf("legacy linker identity lost: %+v", got)
	}
}
