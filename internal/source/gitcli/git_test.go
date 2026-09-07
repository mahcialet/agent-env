package gitcli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/execx"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	r, err := (execx.OSRunner{}).Run(context.Background(), execx.Command{Name: "git", Args: args, Dir: dir, Timeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, r.Stderr)
	}
	return strings.TrimSpace(r.Stdout)
}
func repo(t *testing.T) string {
	t.Helper()
	if _, err := execx.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	dir := filepath.Join(t.TempDir(), "repository 日本語 with spaces")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "init")
	git(t, dir, "config", "user.name", "Fixture")
	git(t, dir, "config", "user.email", "fixture@example.invalid")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("original\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "tracked.txt")
	git(t, dir, "commit", "-m", "fixture")
	return dir
}
func TestWorktreeLifecycle(t *testing.T) {
	dir := repo(t)
	c := Client{Runner: execx.OSRunner{}}
	ctx := context.Background()
	r, err := c.Resolve(ctx, dir, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	a, err := c.Materialize(ctx, r, filepath.Join(t.TempDir(), "lease A 日本語"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.Materialize(ctx, r, filepath.Join(t.TempDir(), "lease B 日本語"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Remove(ctx, a, true); _ = c.Remove(ctx, b, true) })
	linked, err := c.Resolve(ctx, a.Path, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if linked.RepositoryID != r.RepositoryID {
		t.Fatalf("identity mismatch %q %q", linked.RepositoryID, r.RepositoryID)
	}
	state, err := c.Inspect(ctx, a)
	if err != nil || !state.Registered || state.TrackedDirty {
		t.Fatalf("%+v %v", state, err)
	}
	if err := os.WriteFile(filepath.Join(a.Path, "generated 日本語.txt"), []byte("build output"), 0600); err != nil {
		t.Fatal(err)
	}
	state, err = c.Inspect(ctx, a)
	if err != nil || !state.Dirty || state.TrackedDirty {
		t.Fatalf("untracked: %+v %v", state, err)
	}
	if err := os.WriteFile(filepath.Join(a.Path, "tracked.txt"), []byte("modified"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = c.Remove(ctx, a, false); err == nil {
		t.Fatal("removed dirty tracked worktree")
	}
	if _, err := os.Stat(a.Path); err != nil {
		t.Fatal("dirty path lost", err)
	}
	if err = c.Remove(ctx, a, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(b.Path); err != nil {
		t.Fatal("other lease affected", err)
	}
	if err := os.WriteFile(filepath.Join(b.Path, "build-output"), []byte("ok"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = c.Remove(ctx, b, false); err != nil {
		t.Fatal(err)
	}
}

func TestRejectChangedIdentityEvenForce(t *testing.T) {
	dir := repo(t)
	other := repo(t)
	c := Client{Runner: execx.OSRunner{}}
	ctx := context.Background()
	r, err := c.Resolve(ctx, dir, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	w, err := c.Materialize(ctx, r, filepath.Join(t.TempDir(), "managed"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Remove(ctx, w, true) })
	forged := w
	forged.RepositoryID = filepath.Join(other, ".git")
	if err = c.Remove(ctx, forged, true); err == nil {
		t.Fatal("removed mismatched identity")
	}
	forged = w
	forged.Commit = strings.Repeat("a", 40)
	if err = c.Remove(ctx, forged, true); err == nil {
		t.Fatal("removed mismatched HEAD")
	}
	forged = w
	forged.Path = dir
	if err = c.Remove(ctx, forged, true); err == nil {
		t.Fatal("removed primary checkout")
	}
}

func TestInheritedGitDirectoryCannotRedirectSource(t *testing.T) {
	dir := repo(t)
	other := repo(t)
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_WORK_TREE", other)
	r, err := (Client{Runner: execx.OSRunner{}}).Resolve(context.Background(), dir, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := CanonicalPath(filepath.Join(dir, ".git"))
	if r.RepositoryID != want {
		t.Fatalf("inherited environment redirected repository: %+v", r)
	}
}

type fakeRunner struct {
	calls   []execx.Command
	results []execx.Result
	err     error
}

func (f *fakeRunner) Run(_ context.Context, c execx.Command) (execx.Result, error) {
	f.calls = append(f.calls, c)
	if f.err != nil {
		return execx.Result{Stderr: "diagnostic"}, f.err
	}
	r := f.results[0]
	f.results = f.results[1:]
	return r, nil
}
func TestResolveRunnerContract(t *testing.T) {
	dir := t.TempDir()
	id := filepath.Join(dir, ".git")
	if err := os.Mkdir(id, 0700); err != nil {
		t.Fatal(err)
	}
	sha := strings.Repeat("a", 40)
	f := &fakeRunner{results: []execx.Result{{Stdout: id + "\n"}, {Stdout: sha + "\n"}}}
	r, err := (Client{Runner: f}).Resolve(context.Background(), dir, "--some-option")
	if err != nil {
		t.Fatal(err)
	}
	if r.Commit != sha {
		t.Fatal(r)
	}
	want := [][]string{{"rev-parse", "--path-format=absolute", "--git-common-dir"}, {"rev-parse", "--verify", "--end-of-options", "--some-option^{commit}"}}
	for i, c := range f.calls {
		if c.Name != "git" || !reflect.DeepEqual(c.Args, want[i]) || c.Timeout != 30*time.Second || c.Env["GIT_TERMINAL_PROMPT"] != "0" {
			t.Fatalf("command %+v", c)
		}
		canon, _ := CanonicalPath(dir)
		if c.Dir != canon {
			t.Fatalf("cwd %s", c.Dir)
		}
	}
	f = &fakeRunner{err: errors.New("failure")}
	_, err = (Client{Runner: f}).Resolve(context.Background(), dir, "HEAD")
	if err == nil || !strings.Contains(err.Error(), "diagnostic") {
		t.Fatalf("error conversion: %v", err)
	}
}
