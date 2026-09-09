package remotesource

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/blobstore"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/policy"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
	"github.com/oklog/ulid/v2"
)

func materializedReviewFixture(t *testing.T) (Package, string, app.PlanOptions, app.SourceProvider) {
	t.Helper()
	repo, other := repository(t), repository(t)
	manifest(t, repo, other)
	cas, err := blobstore.New(filepath.Join(t.TempDir(), "cas"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Build(context.Background(), app.PlanOptions{Repository: repo, Stack: "local"}, cas)
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(t.TempDir(), "worker")
	opts, provider, err := Materialize(context.Background(), p, home, cas)
	if err != nil {
		t.Fatal(err)
	}
	return p, home, opts, provider
}

func TestMaterializedDigestReusedWithoutBundleAccess(t *testing.T) {
	p, home, opts, _ := materializedReviewFixture(t)
	empty, err := blobstore.New(filepath.Join(t.TempDir(), "empty-cas"))
	if err != nil {
		t.Fatal(err)
	}
	again, provider, err := Materialize(context.Background(), p, home, empty)
	if err != nil {
		t.Fatalf("valid cached digest required bundle extraction: %v", err)
	}
	if again.Repository != opts.Repository || provider == nil {
		t.Fatal("cached identity changed")
	}
	entries, err := os.ReadDir(filepath.Dir(opts.Repository))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".incoming-") {
			t.Fatal("reuse created extraction staging")
		}
	}
}

func TestRemoteProviderForceDestroyPreservesTrackedDiff(t *testing.T) {
	_, home, opts, provider := materializedReviewFixture(t)
	ctx := context.Background()
	plan, err := app.BuildPlan(ctx, opts, provider)
	if err != nil {
		t.Fatal(err)
	}
	id := ulid.Make().String()
	source := plan.Sources[0]
	source.WorktreePath = filepath.Join(home, "leases", id, "sources", source.Alias)
	if err = provider.Materialize(ctx, source); err != nil {
		t.Fatal(err)
	}
	lease := domain.Lease{ID: id, Owner: "fixture", Stack: "local", Mode: "task", Desired: "active", Observed: "ready", CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Hour), Sources: []domain.Source{source}}
	db, err := sqlite.Open(filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Reserve(ctx, lease, 8); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(source.WorktreePath, "data")
	t.Setenv("AGENT_ENV_FIXTURE_SECRET_TOKEN", "fixture-diff-secret-value")
	t.Setenv("GIT_EXTERNAL_DIFF", "nonexistent-external-diff-must-not-run")
	if err = os.WriteFile(data, []byte("changed tracked bytes fixture-diff-secret-value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	svc := app.Service{Home: home, Store: db, Source: provider, Policy: policy.Defaults()}
	if result, err := svc.Destroy(ctx, id, false, false); err == nil || result.Observed != "quarantined" {
		t.Fatalf("nonforce dirty cleanup: %+v %v", result, err)
	}
	if _, err = os.Stat(data); err != nil {
		t.Fatalf("nonforce removed tracked bytes: %v", err)
	}
	result, err := svc.Destroy(ctx, id, true, false)
	if err != nil || result.Observed != "released" {
		t.Fatalf("forced remote cleanup failed: %+v %v", result, err)
	}
	artifacts, err := db.Artifacts(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, artifact := range artifacts {
		if artifact.Kind == "tracked-diff" {
			found = true
			patch, err := os.ReadFile(artifact.Path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(patch), "changed tracked bytes") || strings.Contains(string(patch), "fixture-diff-secret-value") {
				t.Fatalf("missing or unredacted tracked evidence: %q", patch)
			}
		}
	}
	if !found {
		t.Fatal("forced cleanup did not register diff evidence")
	}
	if _, err = os.Stat(source.WorktreePath); !os.IsNotExist(err) {
		t.Fatalf("worktree retained after successful force: %v", err)
	}
}

func TestMaterializedReuseRejectsUnexpectedManagedPaths(t *testing.T) {
	for _, mode := range []string{"manifest", "missing-control", "missing-source", "source-symlink", "control-symlink", "digest-symlink"} {
		t.Run(mode, func(t *testing.T) {
			p, home, opts, _ := materializedReviewFixture(t)
			target := filepath.Join(opts.Repository, "sources", p.Sources[0].Alias)
			switch mode {
			case "manifest":
				target = filepath.Join(opts.Repository, "manifest.json")
				if err := os.WriteFile(target, []byte("unexpected user bytes"), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing-control", "control-symlink":
				target = filepath.Join(opts.Repository, "control")
			case "digest-symlink":
				target = opts.Repository
			}
			if mode != "manifest" {
				if err := os.Rename(target, target+"-saved"); err != nil {
					t.Fatal(err)
				}
				if strings.HasSuffix(mode, "symlink") {
					if err := os.Symlink(target+"-saved", target); err != nil {
						if runtime.GOOS == "windows" {
							t.Skipf("optional symlink privilege unavailable: %v", err)
						}
						t.Fatal(err)
					}
				}
			}
			empty, err := blobstore.New(filepath.Join(t.TempDir(), "empty-cas"))
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err = Materialize(context.Background(), p, home, empty); err == nil {
				t.Fatal("unexpected cached path trusted")
			}
			if strings.Contains(err.Error(), "empty-cas") {
				t.Fatalf("invalid cache triggered extraction before validation: %v", err)
			}
			if mode == "manifest" {
				b, err := os.ReadFile(target)
				if err != nil || string(b) != "unexpected user bytes" {
					t.Fatal("invalid cache was overwritten")
				}
			} else if strings.HasPrefix(mode, "missing-") {
				if _, err := os.Lstat(target); !os.IsNotExist(err) {
					t.Fatalf("missing cache component was silently recreated: %v", err)
				}
			} else if st, err := os.Lstat(target); err != nil || st.Mode()&os.ModeSymlink == 0 {
				t.Fatal("unexpected cache symlink was replaced")
			}
		})
	}
}

func TestRemoteTrackedDiffRefusesOverflowAndForeignWorktree(t *testing.T) {
	_, _, opts, provider := materializedReviewFixture(t)
	ctx := context.Background()
	plan, err := app.BuildPlan(ctx, opts, provider)
	if err != nil {
		t.Fatal(err)
	}
	source := plan.Sources[0]
	source.WorktreePath = filepath.Join(t.TempDir(), "worktree")
	if err = provider.Materialize(ctx, source); err != nil {
		t.Fatal(err)
	}
	diff, ok := provider.(app.SourceDiff)
	if !ok {
		t.Fatal("remote source does not implement diff evidence")
	}
	foreign := source
	foreign.WorktreePath = repository(t)
	if _, err = diff.Diff(ctx, foreign); err == nil {
		t.Fatal("foreign worktree accepted for destructive cleanup evidence")
	}
	if err = os.WriteFile(filepath.Join(source.WorktreePath, "data"), []byte(strings.Repeat("x", 17<<20)), 0600); err != nil {
		t.Fatal(err)
	}
	patch, err := diff.Diff(ctx, source)
	if err == nil || patch != "" || !strings.Contains(err.Error(), "output limit exceeded") {
		t.Fatalf("oversized diff returned usable incomplete evidence: bytes=%d err=%v", len(patch), err)
	}
	if _, err = os.Stat(source.WorktreePath); err != nil {
		t.Fatal("failed diff removed source")
	}
}
