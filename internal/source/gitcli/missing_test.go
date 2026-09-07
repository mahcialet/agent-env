package gitcli

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mahcialet/agent-env/internal/execx"
)

func TestMissingWorktreeRegistrationIsObservedAndRemoved(t *testing.T) {
	testMissingWorktree(t, false)
}

func TestMissingWorktreeThroughParentAlias(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink regression; native temporary path spelling covered separately")
	}
	testMissingWorktree(t, true)
}

func testMissingWorktree(t *testing.T, alias bool) {
	t.Helper()
	root := t.TempDir()
	runner := execx.OSRunner{}
	for _, args := range [][]string{{"init"}, {"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--allow-empty", "-m", "fixture"}} {
		if _, err := runner.Run(context.Background(), execx.Command{Name: "git", Args: args, Dir: root}); err != nil {
			t.Fatal(err)
		}
	}
	client := Client{Runner: runner}
	source, err := client.Resolve(context.Background(), root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	if alias {
		link := filepath.Join(t.TempDir(), "parent alias")
		if err := os.Symlink(parent, link); err != nil {
			t.Fatal(err)
		}
		parent = link
	}
	w, err := client.Materialize(context.Background(), source, filepath.Join(parent, "missing checkout 日本語"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(w.Path); err != nil {
		t.Fatal(err)
	}
	o, err := client.Inspect(context.Background(), w)
	if err != nil || o.Exists || !o.Registered {
		t.Fatalf("stale registration hidden: %+v %v", o, err)
	}
	if err := client.Remove(context.Background(), w, false); err != nil {
		t.Fatal(err)
	}
	o, err = client.Inspect(context.Background(), w)
	if err != nil || o.Exists || o.Registered {
		t.Fatalf("registration remains: %+v %v", o, err)
	}
}
