package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sourceTestGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE=2024-01-02T03:04:05Z", "GIT_COMMITTER_DATE=2024-01-02T03:04:05Z")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func sourceTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	sourceTestGit(t, root, "init")
	sourceTestGit(t, root, "config", "user.name", "Release Test")
	sourceTestGit(t, root, "config", "user.email", "release@example.invalid")
	sourceTestGit(t, root, "config", "commit.gpgsign", "false")
	sourceTestGit(t, root, "config", "tag.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("initial\n"), 0644); err != nil {
		t.Fatal(err)
	}
	sourceTestGit(t, root, "add", "tracked.txt")
	sourceTestGit(t, root, "commit", "-m", "initial")
	return root
}

func TestReleaseSourceValidTags(t *testing.T) {
	for _, annotated := range []bool{false, true} {
		t.Run(map[bool]string{false: "lightweight", true: "annotated"}[annotated], func(t *testing.T) {
			root := sourceTestRepo(t)
			if annotated {
				sourceTestGit(t, root, "tag", "-a", "v0.1.0", "-m", "release annotation")
			} else {
				sourceTestGit(t, root, "tag", "v0.1.0")
			}
			sourceTestGit(t, root, "tag", "non-release-label")
			tag, timestamp, err := releaseVersion(root, "0.1.0")
			if err != nil {
				t.Fatal(err)
			}
			want := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
			if tag != "v0.1.0" || timestamp != want {
				t.Fatalf("got %s %s; want v0.1.0 %s", tag, timestamp, want)
			}
		})
	}
}

func TestReleaseSourceRejectsInvalidIdentity(t *testing.T) {
	cases := []struct {
		name      string
		prepare   func(*testing.T, string)
		requested string
	}{
		{"no tag", func(*testing.T, string) {}, "0.1.0"},
		{"malformed tag", func(t *testing.T, root string) { sourceTestGit(t, root, "tag", "v00.1.0") }, "0.1.0"},
		{"mismatch", func(t *testing.T, root string) { sourceTestGit(t, root, "tag", "v0.2.0") }, "0.1.0"},
		{"ambiguous", func(t *testing.T, root string) {
			sourceTestGit(t, root, "tag", "v0.1.0")
			sourceTestGit(t, root, "tag", "-a", "v0.2.0", "-m", "second version")
		}, "0.1.0"},
		{"HEAD beyond tag", func(t *testing.T, root string) {
			sourceTestGit(t, root, "tag", "v0.1.0")
			sourceTestGit(t, root, "commit", "--allow-empty", "-m", "later")
		}, "0.1.0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := sourceTestRepo(t)
			tc.prepare(t, root)
			if _, _, err := releaseVersion(root, tc.requested); err == nil {
				t.Fatal("invalid release identity accepted")
			}
		})
	}
	for _, version := range []string{"", "v0.1.0", "01.2.3", "1.02.3", "1.2.03", "1.2", "1.2.3-rc.1", "1.2.3+build", " 1.2.3", "1.2.3\n"} {
		t.Run("version "+version, func(t *testing.T) {
			if _, _, err := releaseVersion(t.TempDir(), version); err == nil || !strings.Contains(err.Error(), "invalid release version") {
				t.Fatalf("expected invalid version error before Git effects, got %v", err)
			}
		})
	}
	if _, _, err := releaseVersion(t.TempDir(), "0.1.0"); err == nil {
		t.Fatal("directory without Git identity accepted")
	}
}

func TestReleaseSourceRejectsDirtRegardlessGitConfig(t *testing.T) {
	for _, kind := range []string{"tracked", "staged", "untracked", "deleted"} {
		t.Run(kind, func(t *testing.T) {
			root := sourceTestRepo(t)
			sourceTestGit(t, root, "tag", "v0.1.0")
			sourceTestGit(t, root, "config", "status.showUntrackedFiles", "no")
			name := "tracked.txt"
			if kind == "untracked" {
				name = "new.txt"
			}
			var err error
			if kind == "deleted" {
				err = os.Remove(filepath.Join(root, name))
			} else {
				err = os.WriteFile(filepath.Join(root, name), []byte("changed\n"), 0644)
			}
			if err != nil {
				t.Fatal(err)
			}
			if kind == "staged" {
				sourceTestGit(t, root, "add", name)
			}
			if _, _, err := releaseVersion(root, "0.1.0"); err == nil || !strings.Contains(err.Error(), "clean") {
				t.Fatalf("expected clean-tree error, got %v", err)
			}
		})
	}
}

func TestReleaseSourceRejectsHiddenIndexEntries(t *testing.T) {
	for _, flag := range []string{"--assume-unchanged", "--skip-worktree"} {
		for _, changed := range []bool{false, true} {
			t.Run(flag+map[bool]string{false: "/unchanged", true: "/modified"}[changed], func(t *testing.T) {
				root := sourceTestRepo(t)
				sourceTestGit(t, root, "tag", "v0.1.0")
				sourceTestGit(t, root, "update-index", flag, "tracked.txt")
				if changed {
					if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("hidden change\n"), 0644); err != nil {
						t.Fatal(err)
					}
				}
				if status := sourceTestGit(t, root, "status", "--porcelain"); status != "" {
					t.Fatalf("fixture is visibly dirty: %s", status)
				}
				before := sourceTestGit(t, root, "ls-files", "-v")
				if _, _, err := releaseVersion(root, "0.1.0"); err == nil || !strings.Contains(err.Error(), "index flags") {
					t.Fatalf("expected unsupported index flags error, got %v", err)
				}
				if after := sourceTestGit(t, root, "ls-files", "-v"); after != before {
					t.Fatal("validation changed caller index flags")
				}
			})
		}
	}
}

func TestReleaseTimestampRejectsMalformedOutput(t *testing.T) {
	for _, raw := range []string{"", "not a time", "123 extra", "123\n456", "9223372036854775808"} {
		if _, err := releaseTimestamp(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}

func TestPrivateReleaseSourceUsesOnlyCommittedFiles(t *testing.T) {
	root := sourceTestRepo(t)
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("injected.go\n"), 0644); err != nil {
		t.Fatal(err)
	}
	sourceTestGit(t, root, "add", ".gitignore")
	sourceTestGit(t, root, "commit", "-m", "ignore local file")
	sourceTestGit(t, root, "tag", "existing-label")
	sourceTestGit(t, root, "update-index", "--assume-unchanged", "tracked.txt")
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("hidden modification\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "injected.go"), []byte("package injected\n"), 0644); err != nil {
		t.Fatal(err)
	}
	commit := sourceTestGit(t, root, "rev-parse", "HEAD")
	before := sourceTestGit(t, root, "show-ref")
	if status := sourceTestGit(t, root, "status", "--porcelain=v1"); status != "" {
		t.Fatalf("expected hidden changes fixture, got %s", status)
	}
	clone, cleanup, err := privateReleaseSource(root, commit, "v0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	if _, err := os.Stat(filepath.Join(clone, "injected.go")); !os.IsNotExist(err) {
		t.Fatalf("ignored source included: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(clone, "tracked.txt"))
	if err != nil || string(data) != "initial\n" {
		t.Fatalf("clone did not use committed contents: %q %v", data, err)
	}
	if got := sourceTestGit(t, root, "show-ref"); got != before {
		t.Fatalf("caller refs changed: %s", got)
	}
	if _, _, err := releaseVersion(clone, "0.1.0"); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(root, "tracked.txt"))
	if err != nil || string(data) != "hidden modification\n" {
		t.Fatal("caller file changed")
	}
}

func TestReleaseGitIgnoresAmbientRepositoryRouting(t *testing.T) {
	root := sourceTestRepo(t)
	other := sourceTestRepo(t)
	sourceTestGit(t, other, "commit", "--allow-empty", "-m", "different identity")
	want := sourceTestGit(t, root, "rev-parse", "HEAD")
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_WORK_TREE", other)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(other, ".git", "index"))
	t.Setenv("GIT_OBJECT_DIRECTORY", filepath.Join(other, ".git", "objects"))
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.worktree")
	t.Setenv("GIT_CONFIG_VALUE_0", other)
	got, err := gitOut(root, "rev-parse", "HEAD")
	if err != nil || got != want {
		t.Fatalf("ambient routing affected identity: %s %v", got, err)
	}
	clone, cleanup, err := privateReleaseSource(root, want, "v0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	got, err = gitOut(clone, "rev-parse", "HEAD")
	if err != nil || got != want {
		t.Fatalf("ambient routing affected snapshot: %s %v", got, err)
	}
}

func TestReleaseCheckoutFilterHelper(t *testing.T) {
	mode := os.Getenv("AGENT_ENV_TEST_CHECKOUT_FILTER")
	if mode == "" {
		return
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(2)
	}
	if len(os.Args) > 0 && os.Args[len(os.Args)-1] == "clean" {
		data = []byte(strings.ReplaceAll(string(data), "filtered", "initial"))
	} else {
		data = []byte(strings.ReplaceAll(string(data), "initial", "filtered"))
	}
	if _, err = os.Stdout.Write(data); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestPrivateReleaseSourceIgnoresAmbientCheckoutFilters(t *testing.T) {
	for _, scope := range []string{"global", "system"} {
		t.Run(scope, func(t *testing.T) {
			root := sourceTestRepo(t)
			commit := sourceTestGit(t, root, "rev-parse", "HEAD")
			config := filepath.Join(t.TempDir(), "config")
			attrs := filepath.Join(t.TempDir(), "attributes")
			if err := os.WriteFile(attrs, []byte("tracked.txt filter=release-test\n"), 0644); err != nil {
				t.Fatal(err)
			}
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			// Git filter commands use Git's shell interface; this fixture executes the
			// native Go test binary, requiring no separate scripting runtime.
			quoted := "'" + strings.ReplaceAll(filepath.ToSlash(executable), "'", "'\"'\"'") + "'"
			for _, kv := range [][2]string{{"core.attributesFile", attrs}, {"filter.release-test.smudge", quoted + " -test.run=^TestReleaseCheckoutFilterHelper$ -- smudge"}, {"filter.release-test.clean", quoted + " -test.run=^TestReleaseCheckoutFilterHelper$ -- clean"}, {"filter.release-test.required", "true"}} {
				sourceTestGit(t, root, "config", "--file", config, kv[0], kv[1])
			}
			t.Setenv("AGENT_ENV_TEST_CHECKOUT_FILTER", "1")
			t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
			t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
			t.Setenv("GIT_CONFIG_NOSYSTEM", "0")
			if scope == "global" {
				t.Setenv("GIT_CONFIG_GLOBAL", config)
			} else {
				t.Setenv("GIT_CONFIG_SYSTEM", config)
			}
			// Prove the external filter actually changes bytes while status stays clean.
			control := filepath.Join(t.TempDir(), "control")
			sourceTestGit(t, root, "clone", "--no-hardlinks", "--", root, control)
			changed, err := os.ReadFile(filepath.Join(control, "tracked.txt"))
			if err != nil || string(changed) != "filtered\n" {
				t.Fatalf("inactive filter fixture: %q %v", changed, err)
			}
			if status := sourceTestGit(t, control, "status", "--porcelain"); status != "" {
				t.Fatalf("filter fixture is dirty: %s", status)
			}
			clone, cleanup, err := privateReleaseSource(root, commit, "v0.1.0")
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()
			got, err := os.ReadFile(filepath.Join(clone, "tracked.txt"))
			if err != nil || string(got) != "initial\n" {
				t.Fatalf("release source transformed by ambient %s filter: %q %v", scope, got, err)
			}
		})
	}
}
