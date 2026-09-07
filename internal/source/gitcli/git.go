// Package gitcli manages pinned, detached worktrees using the installed Git CLI.
package gitcli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/execx"
)

type Client struct{ Runner execx.Runner }
type Resolved struct{ RepositoryID, RepositoryPath, RequestedRef, Commit string }
type Worktree struct {
	Resolved
	Path string
}
type Inspection struct {
	Exists, Registered, Dirty, TrackedDirty bool
	Commit                                  string
}

func (c Client) run(ctx context.Context, dir string, args ...string) (string, error) {
	if c.Runner == nil {
		return "", errors.New("Git command runner is required")
	}
	r, err := c.Runner.Run(ctx, execx.Command{Name: "git", Args: args, Dir: dir,
		Env:      map[string]string{"GIT_TERMINAL_PROMPT": "0"},
		UnsetEnv: []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_PREFIX"},
		Timeout:  30 * time.Second})
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(r.Stderr))
	}
	return r.Stdout, nil
}

func CanonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

func (c Client) identity(ctx context.Context, repo string) (string, error) {
	out, err := c.run(ctx, repo, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	return CanonicalPath(strings.TrimRight(out, "\r\n"))
}

func (c Client) Resolve(ctx context.Context, repo, ref string) (Resolved, error) {
	var result Resolved
	canonical, err := CanonicalPath(repo)
	if err != nil {
		return result, err
	}
	id, err := c.identity(ctx, canonical)
	if err != nil {
		return result, err
	}
	if ref == "" {
		ref = "HEAD"
	}
	commit, err := c.run(ctx, canonical, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return result, err
	}
	commit = strings.TrimSpace(commit)
	if !validCommit(commit) {
		return result, errors.New("Git returned an invalid commit identity")
	}
	return Resolved{RepositoryID: id, RepositoryPath: canonical, RequestedRef: ref, Commit: commit}, nil
}

func validCommit(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func samePath(a, b string) bool {
	aa, ea := os.Stat(a)
	bb, eb := os.Stat(b)
	return ea == nil && eb == nil && os.SameFile(aa, bb)
}

func (c Client) Materialize(ctx context.Context, source Resolved, dest string) (Worktree, error) {
	w := Worktree{Resolved: source}
	if !validCommit(source.Commit) {
		return w, errors.New("invalid pinned commit")
	}
	id, err := c.identity(ctx, source.RepositoryPath)
	if err != nil {
		return w, err
	}
	if !samePath(id, source.RepositoryID) {
		return w, errors.New("repository identity changed before materialization")
	}
	abs, err := filepath.Abs(dest)
	if err != nil {
		return w, err
	}
	w.Path = abs
	if _, err = os.Lstat(abs); !errors.Is(err, os.ErrNotExist) {
		return w, fmt.Errorf("worktree destination must not exist: %s", abs)
	}
	_, err = c.run(ctx, source.RepositoryPath, "worktree", "add", "--detach", "--", abs, source.Commit)
	return w, err
}

func (c Client) Inspect(ctx context.Context, w Worktree) (Inspection, error) {
	var result Inspection
	info, err := os.Lstat(w.Path)
	if errors.Is(err, os.ErrNotExist) {
		listing, e := c.run(ctx, w.RepositoryID, "worktree", "list", "--porcelain", "-z")
		if e != nil {
			return result, e
		}
		for _, record := range strings.Split(listing, "\x00\x00") {
			var path, head string
			detached := false
			for _, field := range strings.Split(record, "\x00") {
				if strings.HasPrefix(field, "worktree ") {
					path = strings.TrimPrefix(field, "worktree ")
				}
				if strings.HasPrefix(field, "HEAD ") {
					head = strings.TrimPrefix(field, "HEAD ")
				}
				if field == "detached" {
					detached = true
				}
			}
			match := filepath.Clean(path) == filepath.Clean(w.Path)
			if runtime.GOOS == "windows" {
				match = strings.EqualFold(filepath.Clean(path), filepath.Clean(w.Path))
			}
			if path != "" && match {
				result.Registered = true
				result.Commit = head
				if !detached || head != w.Commit {
					return result, errors.New("missing worktree registration no longer matches pinned detached source")
				}
				break
			}
		}
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.Exists = true
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return result, errors.New("worktree path was replaced or is not a directory")
	}
	id, err := c.identity(ctx, w.Path)
	if err != nil {
		return result, err
	}
	if !samePath(id, w.RepositoryID) {
		return result, errors.New("worktree repository identity mismatch")
	}
	root, err := c.run(ctx, w.Path, "rev-parse", "--show-toplevel")
	if err != nil {
		return result, err
	}
	if !samePath(strings.TrimRight(root, "\r\n"), w.Path) {
		return result, errors.New("recorded worktree path is not a repository root")
	}
	listing, err := c.run(ctx, w.Path, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return result, err
	}
	for _, record := range strings.Split(listing, "\x00\x00") {
		var path, head string
		detached := false
		for _, field := range strings.Split(record, "\x00") {
			if strings.HasPrefix(field, "worktree ") {
				path = strings.TrimPrefix(field, "worktree ")
			}
			if strings.HasPrefix(field, "HEAD ") {
				head = strings.TrimPrefix(field, "HEAD ")
			}
			if field == "detached" {
				detached = true
			}
		}
		if path != "" && samePath(path, w.Path) {
			result.Registered = true
			result.Commit = head
			if !detached {
				return result, errors.New("managed worktree is no longer detached")
			}
			break
		}
	}
	if !result.Registered {
		return result, errors.New("worktree is not registered")
	}
	if result.Commit != w.Commit {
		return result, errors.New("worktree HEAD differs from pinned commit")
	}
	status, err := c.run(ctx, w.Path, "status", "--porcelain=v2", "-z", "--untracked-files=normal")
	if err != nil {
		return result, err
	}
	for _, record := range strings.Split(status, "\x00") {
		if record == "" {
			continue
		}
		result.Dirty = true
		if record[0] != '?' && record[0] != '!' {
			result.TrackedDirty = true
		}
	}
	return result, nil
}

func (c Client) Remove(ctx context.Context, w Worktree, force bool) error {
	state, err := c.Inspect(ctx, w)
	if err != nil {
		return err
	}
	if !state.Exists && !state.Registered {
		return nil
	}
	if state.TrackedDirty && !force {
		return errors.New("tracked worktree changes require quarantine or explicit force")
	}
	// Git otherwise refuses untracked build output, which the review policy permits.
	// --force is used only after pinned identity and tracked-change checks above.
	_, err = c.run(ctx, w.RepositoryID, "worktree", "remove", "--force", "--", w.Path)
	return err
}
