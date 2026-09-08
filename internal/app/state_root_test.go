package app

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/paths"
)

func assertStatePath(t *testing.T, root, path string) {
	t.Helper()
	rel, err := filepath.Rel(root, path)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("owned path %q escapes state root %q: %v", path, root, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("owned state absent: %s: %v", path, err)
	}
}

func stateTree(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			result[rel] = "directory"
			return nil
		}
		data, err := os.ReadFile(path)
		if err == nil {
			result[rel] = string(data)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestLifecyclePersistentStateStaysUnderOverride(t *testing.T) {
	for _, kind := range []string{"compose", "android"} {
		t.Run(kind, func(t *testing.T) {
			var s *Service
			var options PlanOptions
			if kind == "compose" {
				s, options, _, _, _ = lifecycleFixture(t)
			} else {
				s, options, _, _ = androidAppFixture(t)
			}
			defaultHome := t.TempDir()
			t.Setenv("HOME", defaultHome)
			t.Setenv("USERPROFILE", defaultHome)
			t.Setenv("LOCALAPPDATA", defaultHome)
			t.Setenv("XDG_STATE_HOME", defaultHome)
			t.Setenv("AGENT_ENV_HOME", s.Home)
			resolved, err := paths.Resolve()
			if err != nil || resolved != s.Home {
				t.Fatalf("state override: %q %v", resolved, err)
			}
			beforeRepo := stateTree(t, options.Repository)
			beforeDefault := stateTree(t, defaultHome)
			ctx := context.Background()
			lease, err := s.Create(ctx, options, CreateOptions{Owner: "state-audit"})
			if err != nil {
				t.Fatal(err)
			}
			assertStatePath(t, s.Home, filepath.Join(s.Home, "registry.sqlite"))
			assertStatePath(t, s.Home, filepath.Join(s.Home, "registry.sqlite-wal"))
			assertStatePath(t, s.Home, filepath.Join(s.Home, "leases", lease.ID, "environment.json"))
			for _, source := range lease.Sources {
				assertStatePath(t, s.Home, source.WorktreePath)
			}
			for _, runtime := range lease.Runtimes {
				if runtime.Android != nil {
					assertStatePath(t, s.Home, runtime.Android.AVDHome)
					assertStatePath(t, s.Home, runtime.Android.AVDPath)
				} else {
					assertStatePath(t, s.Home, runtime.ConfigPath)
				}
			}
			if _, err = s.Destroy(ctx, lease.ID, false, false); err != nil {
				t.Fatal(err)
			}
			artifacts, err := s.Store.Artifacts(ctx, lease.ID)
			if err != nil || kind == "compose" && len(artifacts) == 0 {
				t.Fatalf("no lifecycle evidence: %v", err)
			}
			for _, artifact := range artifacts {
				assertStatePath(t, s.Home, artifact.Path)
			}
			if got := stateTree(t, options.Repository); !reflect.DeepEqual(got, beforeRepo) {
				t.Fatalf("target repository modified: %v", got)
			}
			if got := stateTree(t, defaultHome); !reflect.DeepEqual(got, beforeDefault) {
				t.Fatalf("state leaked into default home: %v", got)
			}
		})
	}
}

func TestNamedCommandEvidenceStaysUnderStateRoot(t *testing.T) {
	s, db, lease := commandFixture(t)
	run, err := s.Test(context.Background(), lease.ID, "check")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(s.Home, "leases", lease.ID, "artifacts")
	assertStatePath(t, root, run.StdoutPath)
	assertStatePath(t, root, run.StderrPath)
	artifacts, err := db.Artifacts(context.Background(), lease.ID)
	if err != nil || len(artifacts) != 4 {
		t.Fatalf("command evidence missing: %v", err)
	}
	for _, artifact := range artifacts {
		assertStatePath(t, root, artifact.Path)
	}
	// Target-defined commands may intentionally build in their owned worktree;
	// app copies their selected outputs into the lease evidence directory.
	assertStatePath(t, lease.Sources[0].WorktreePath, filepath.Join(lease.Sources[0].WorktreePath, "reports", "result.txt"))
}
