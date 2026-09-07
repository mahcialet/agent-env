package gitcli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/execx"
)

func TestManifestOriginTrackedDirtyUntrackedAndOutside(t *testing.T) {
	dir := repo(t)
	ctx := context.Background()
	client := Client{Runner: execx.OSRunner{}}
	path := filepath.Join(dir, "manifest space 日本語.yaml")
	if err := os.WriteFile(path, []byte("version: 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	head := git(t, dir, "rev-parse", "HEAD")
	commit, modified, err := client.Origin(ctx, path)
	if err != nil || commit != head || !modified {
		t.Fatalf("untracked origin %q %v %v", commit, modified, err)
	}
	git(t, dir, "add", "--", filepath.Base(path))
	git(t, dir, "commit", "-m", "control manifest")
	head = git(t, dir, "rev-parse", "HEAD")
	indexPath := filepath.Join(dir, ".git", "index")
	before, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	commit, modified, err = client.Origin(ctx, path)
	if err != nil || commit != head || modified {
		t.Fatalf("clean origin %q %v %v", commit, modified, err)
	}
	after, err := os.ReadFile(indexPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("origin changed index")
	}
	if err := os.WriteFile(path, []byte("version: 2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	commit, modified, err = client.Origin(ctx, path)
	if err != nil || commit != head || !modified {
		t.Fatalf("dirty origin %q %v %v", commit, modified, err)
	}
	outside := filepath.Join(t.TempDir(), "outside.yaml")
	if err := os.WriteFile(outside, []byte("version: 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	commit, modified, err = client.Origin(ctx, outside)
	if err != nil || commit != "" || !modified {
		t.Fatalf("outside origin %q %v %v", commit, modified, err)
	}
}

func TestManifestOriginIgnoredFileAndLinkedWorktree(t *testing.T) {
	dir := repo(t)
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("ignored.yaml\n"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ignored.yaml")
	if err := os.WriteFile(path, []byte("ignored"), 0600); err != nil {
		t.Fatal(err)
	}
	client := Client{Runner: execx.OSRunner{}}
	ctx := context.Background()
	_, modified, err := client.Origin(ctx, path)
	if err != nil || !modified {
		t.Fatalf("ignored treated committed %v %v", modified, err)
	}
	resolved, err := client.Resolve(ctx, dir, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	w, err := client.Materialize(ctx, resolved, filepath.Join(t.TempDir(), "control worktree"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Remove(ctx, w, true) })
	commit, modified, err := client.Origin(ctx, filepath.Join(w.Path, "tracked.txt"))
	if err != nil || modified || commit != resolved.Commit {
		t.Fatalf("linked origin %q %v %v", commit, modified, err)
	}
}

func TestManifestOriginRunnerContract(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "-manifest 日本語.yaml")
	if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	head := strings.Repeat("a", 40)
	runner := &fakeRunner{results: []execx.Result{{Stdout: head + "\n"}, {Stdout: "? -manifest 日本語.yaml\x00"}}}
	commit, modified, err := (Client{Runner: runner}).Origin(context.Background(), path)
	if err != nil || commit != head || !modified {
		t.Fatalf("origin %s %v %v", commit, modified, err)
	}
	want := [][]string{{"rev-parse", "--verify", "--end-of-options", "HEAD^{commit}"}, {"status", "--porcelain=v2", "-z", "--untracked-files=all", "--ignored=matching", "--", filepath.Base(path)}}
	for i, call := range runner.calls {
		if !reflect.DeepEqual(call.Args, want[i]) || call.Env["GIT_OPTIONAL_LOCKS"] != "0" || call.Env["GIT_LITERAL_PATHSPECS"] != "1" || call.Name != "git" {
			t.Fatalf("origin argv/env %+v", call)
		}
	}
	runner = &fakeRunner{err: errors.New("checkout failure")}
	if _, _, err := (Client{Runner: runner}).Origin(context.Background(), path); err == nil {
		t.Fatal("checkout failure mislabeled outside Git")
	}
}
