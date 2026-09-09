package remotesource

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/blobstore"
)

func TestRemoteSourceLifecycleBeyondWindowsLegacyPathLimit(t *testing.T) {
	ctx := context.Background()
	repo, other := repository(t), repository(t)
	manifest(t, repo, other)
	cas, err := blobstore.New(filepath.Join(t.TempDir(), "cas"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Build(ctx, app.PlanOptions{Repository: repo, Stack: "local"}, cas)
	if err != nil {
		t.Fatal(err)
	}
	// Real worker homes and immutable digest subdirectories can exceed the
	// legacy Windows 260-character limit. Keep every individual component short.
	home := t.TempDir()
	for len(home) < 300 {
		home = filepath.Join(home, strings.Repeat("nested", 8))
	}
	options, provider, err := Materialize(ctx, p, home, cas)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := app.BuildPlan(ctx, options, provider)
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range plan.Sources {
		source.WorktreePath = filepath.Join(home, "worktrees", source.Alias)
		if err := provider.Materialize(ctx, source); err != nil {
			t.Fatal(err)
		}
		observed, err := provider.Inspect(ctx, source)
		if err != nil || !observed.Exists || !observed.Registered || observed.Commit != source.Commit {
			t.Fatalf("long-path source inspection: %+v %v", observed, err)
		}
		if err := provider.Remove(ctx, source, false); err != nil {
			t.Fatal(err)
		}
	}
}
