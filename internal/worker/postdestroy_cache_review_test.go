package worker

import (
	"context"
	"github.com/mahcialet/agent-env/internal/blobstore"
	"os"
	"path/filepath"
	"testing"
)

func TestPostDestroyCacheReview(t *testing.T) {
	ctx := context.Background()
	e, op, process, db := executorFixture(t)
	executePrepared(t, e, op)
	before, err := db.Get(ctx, op.LeaseID)
	if err != nil {
		t.Fatal(err)
	}
	result := executePrepared(t, e, nextOperation(op, "destroy", Request{}))
	if !result.CleanupConfirmed {
		t.Fatal("destroy no proof")
	}
	for _, s := range before.Sources {
		if _, err := os.Stat(s.WorktreePath); !os.IsNotExist(err) {
			t.Fatalf("worktree still exists: %v", err)
		}
		if st, err := os.Lstat(s.RepositoryPath); err != nil || !st.IsDir() {
			t.Fatalf("bare cache removed: %v", err)
		}
	}
	// Subsequent operations must reuse the retained cache even without bundle data.
	empty, err := blobstore.New(filepath.Join(t.TempDir(), "empty-cas"))
	if err != nil {
		t.Fatal(err)
	}
	e.CAS = empty
	for _, kind := range []string{"logs", "artifact", "reconcile", "destroy"} {
		t.Run(kind, func(t *testing.T) {
			follow := nextOperation(op, kind, Request{})
			if kind == "destroy" {
				follow.ID += "-again"
			}
			if err := e.Prepare(ctx, follow); err != nil {
				t.Fatalf("postdestroy Prepare: %v", err)
			}
			result := e.Execute(ctx, follow)
			if result.State != "completed" {
				t.Fatalf("postdestroy Execute: %+v", result)
			}
			if kind == "reconcile" && !result.CleanupConfirmed {
				t.Fatal("reconcile lost absence proof")
			}
		})
	}
	if process.starts != 1 {
		t.Fatalf("recreated process: %d", process.starts)
	}
}
