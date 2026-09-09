package remotesource

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/blobstore"
)

func TestMaterializeThroughAncestorAlias(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("fixture needs optional symlink privilege: %v", err)
		}
		t.Fatal(err)
	}
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
	opts, provider, err := Materialize(ctx, p, filepath.Join(alias, "worker"), cas)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := app.BuildPlan(ctx, opts, provider)
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range plan.Sources {
		source.WorktreePath = filepath.Join(t.TempDir(), "worktree")
		if err := provider.Materialize(ctx, source); err != nil {
			t.Fatal(err)
		}
		observed, err := provider.Inspect(ctx, source)
		if err != nil || !observed.Exists || !observed.Registered || observed.Commit != source.Commit {
			t.Fatalf("aliased source inspection: %+v, %v", observed, err)
		}
		if err := provider.Remove(ctx, source, false); err != nil {
			t.Fatal(err)
		}
	}
	// A trusted ancestor alias must not authorize a symlink replacing a
	// managed repository, even when it points to an otherwise valid repository.
	sourcePath := filepath.Join(opts.Repository, "sources", p.Sources[0].Alias)
	if err := os.Rename(sourcePath, sourcePath+"-saved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sourcePath+"-saved", sourcePath); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Materialize(ctx, p, filepath.Join(alias, "worker"), cas); err == nil {
		t.Fatal("symlink replacing a managed source was accepted")
	}
}
