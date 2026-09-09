//go:build windows

package remotesource

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/blobstore"
	"github.com/mahcialet/agent-env/internal/paths"
)

func TestWindowsSourceDerivedDirectoryPreflight(t *testing.T) {
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
	home := t.TempDir()
	for len(home) < 175 {
		home = filepath.Join(home, "nested")
	}
	if err := paths.ValidateExecutionDirectory(home); err != nil {
		t.Fatalf("test home must itself be supported: %v", err)
	}
	if _, _, err := Materialize(ctx, p, home, cas); !errors.Is(err, paths.ErrExecutionDirectoryUnsupported) {
		t.Fatalf("derived source directory was not rejected: %v", err)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("rejected source layout created state: %v", err)
	}
	deep := filepath.Join(home, strings.Repeat("long", 30))
	if _, err := git(ctx, deep, "--version"); !errors.Is(err, paths.ErrExecutionDirectoryUnsupported) {
		t.Fatalf("Git invocation skipped typed preflight: %v", err)
	}
	if _, err := Build(ctx, app.PlanOptions{Repository: deep, Stack: "local"}, cas); !errors.Is(err, paths.ErrExecutionDirectoryUnsupported) {
		t.Fatalf("source packaging skipped repository preflight: %v", err)
	}
}
