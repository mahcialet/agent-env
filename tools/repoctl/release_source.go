package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var releaseTag = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func gitOut(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"--no-replace-objects"}, args...)...)
	cmd.Dir = root
	cmd.Env = releaseGitEnv()
	b, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(b)), nil
}

// Keep normal executable lookup and authentication while preventing ambient Git
// repository routing (including routing injected through config) from overriding
// the explicitly selected repository.
func releaseGitEnv() []string {
	blocked := map[string]bool{
		"GIT_DIR": true, "GIT_COMMON_DIR": true, "GIT_WORK_TREE": true,
		"GIT_INDEX_FILE": true, "GIT_OBJECT_DIRECTORY": true,
		"GIT_ALTERNATE_OBJECT_DIRECTORIES": true, "GIT_NAMESPACE": true,
		"GIT_CEILING_DIRECTORIES": true, "GIT_DISCOVERY_ACROSS_FILESYSTEM": true,
		"GIT_CONFIG": true, "GIT_CONFIG_PARAMETERS": true, "GIT_CONFIG_COUNT": true,
	}
	env := []string{}
	for _, variable := range os.Environ() {
		key, _, _ := strings.Cut(variable, "=")
		key = strings.ToUpper(key)
		if !blocked[key] && !strings.HasPrefix(key, "GIT_CONFIG_KEY_") && !strings.HasPrefix(key, "GIT_CONFIG_VALUE_") {
			env = append(env, variable)
		}
	}
	return env
}

// privateReleaseSource checks out committed objects into an owned clone. Ignored
// files and index flags hiding working-tree edits cannot become compiler inputs.
func privateReleaseSource(root, commit, tag string) (string, func(), error) {
	noop := func() {}
	if !releaseTag.MatchString(tag) || !regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64})$`).MatchString(commit) {
		return "", noop, fmt.Errorf("private release source requires canonical tag and full commit hash")
	}
	head, err := gitOut(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", noop, err
	}
	if head != commit {
		return "", noop, fmt.Errorf("private release source commit must equal source HEAD")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", noop, err
	}
	workspace, err := os.MkdirTemp("", "agent-env-release-source-")
	if err != nil {
		return "", noop, err
	}
	cleanup := func() { _ = os.RemoveAll(workspace) }
	clone := filepath.Join(workspace, "source")
	fail := func(err error) (string, func(), error) { cleanup(); return "", noop, err }
	if _, err := gitOut(root, "clone", "--no-hardlinks", "--no-tags", "--no-checkout", "--", root, clone); err != nil {
		return fail(err)
	}
	if _, err := gitOut(clone, "-c", "core.autocrlf=false", "checkout", "--detach", commit); err != nil {
		return fail(err)
	}
	if _, err := gitOut(clone, "-c", "tag.gpgsign=false", "tag", tag, commit); err != nil {
		return fail(err)
	}
	if _, _, err := releaseVersion(clone, strings.TrimPrefix(tag, "v")); err != nil {
		return fail(err)
	}
	actual, err := gitOut(clone, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return fail(err)
	}
	if actual != commit {
		return fail(fmt.Errorf("private release source identity mismatch"))
	}
	return clone, cleanup, nil
}

// releaseVersion validates identity before callers create any release output.
// Untracked files are dirt; ignored build output is intentionally excluded.
func releaseVersion(root, requested string) (string, time.Time, error) {
	if !releaseTag.MatchString("v" + requested) {
		return "", time.Time{}, fmt.Errorf("invalid release version %q: expected canonical X.Y.Z", requested)
	}
	status, err := gitOut(root, "status", "--porcelain=v1", "--untracked-files=all", "--ignore-submodules=none")
	if err != nil {
		return "", time.Time{}, err
	}
	if status != "" {
		return "", time.Time{}, fmt.Errorf("working tree and index must be clean, including untracked files")
	}
	head, err := gitOut(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", time.Time{}, err
	}
	tags, err := gitOut(root, "tag", "--points-at", head)
	if err != nil {
		return "", time.Time{}, err
	}
	var matching []string
	for _, tag := range strings.Fields(tags) {
		if releaseTag.MatchString(tag) {
			matching = append(matching, tag)
		}
	}
	if len(matching) == 0 {
		return "", time.Time{}, fmt.Errorf("HEAD has no canonical vX.Y.Z release tag")
	}
	if len(matching) != 1 {
		return "", time.Time{}, fmt.Errorf("HEAD has multiple release tags: %s", strings.Join(matching, ", "))
	}
	tag := matching[0]
	if tag != "v"+requested {
		return "", time.Time{}, fmt.Errorf("requested version %q does not match HEAD release tag %q", requested, tag)
	}
	commit, err := gitOut(root, "rev-parse", "--verify", "refs/tags/"+tag+"^{commit}")
	if err != nil {
		return "", time.Time{}, err
	}
	if commit != head {
		return "", time.Time{}, fmt.Errorf("release tag commit does not equal HEAD")
	}
	ts, err := gitOut(root, "show", "-s", "--format=%ct", commit, "--")
	if err != nil {
		return "", time.Time{}, err
	}
	mtime, err := releaseTimestamp(ts)
	if err != nil {
		return "", time.Time{}, err
	}
	return tag, mtime, nil
}

func releaseTimestamp(raw string) (time.Time, error) {
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid tagged commit timestamp %q: %w", raw, err)
	}
	return time.Unix(seconds, 0).UTC(), nil
}
