package buildinfo

import (
	"strings"
	"testing"
)

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
