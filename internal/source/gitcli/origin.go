package gitcli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/execx"
)

// Origin records the selected control file's checkout HEAD, independently from
// any requested runtime ref. No checkout/fetch/index refresh is performed.
// Outside a Git checkout the file is a modified, uncommitted snapshot authority.
func (c Client) Origin(ctx context.Context, path string) (string, bool, error) {
	actual, err := CanonicalPath(path)
	if err != nil {
		return "", false, err
	}
	info, err := os.Stat(actual)
	if err != nil {
		return "", false, err
	}
	if !info.Mode().IsRegular() {
		return "", false, errors.New("manifest origin requires a regular file")
	}
	directory := filepath.Dir(actual)
	inside, err := hasCheckoutMarker(directory)
	if err != nil {
		return "", false, err
	}
	if !inside {
		return "", true, nil
	}
	head, err := c.originRun(ctx, directory, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
	if err != nil {
		return "", false, err
	}
	commit := strings.TrimSpace(head)
	if !validCommit(commit) {
		return "", false, errors.New("manifest checkout returned an invalid HEAD commit")
	}
	status, err := c.originRun(ctx, directory, "status", "--porcelain=v2", "-z", "--untracked-files=all", "--ignored=matching", "--", filepath.Base(actual))
	if err != nil {
		return "", false, err
	}
	return commit, status != "", nil
}

func hasCheckoutMarker(directory string) (bool, error) {
	for {
		_, err := os.Lstat(filepath.Join(directory, ".git"))
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return false, nil
		}
		directory = parent
	}
}

func (c Client) originRun(ctx context.Context, directory string, args ...string) (string, error) {
	if c.Runner == nil {
		return "", errors.New("Git command runner is required")
	}
	result, err := c.Runner.Run(ctx, execx.Command{Name: "git", Args: args, Dir: directory, Env: map[string]string{"GIT_TERMINAL_PROMPT": "0", "GIT_OPTIONAL_LOCKS": "0", "GIT_LITERAL_PATHSPECS": "1"}, UnsetEnv: []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_PREFIX", "GIT_GLOB_PATHSPECS", "GIT_NOGLOB_PATHSPECS", "GIT_ICASE_PATHSPECS"}, Timeout: 30 * time.Second})
	if err != nil {
		return "", fmt.Errorf("read manifest Git origin: %w", err)
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("read manifest Git origin: git %s exited %d", args[0], result.ExitCode)
	}
	return result.Stdout, nil
}
