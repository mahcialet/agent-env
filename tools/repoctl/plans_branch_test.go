package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestPlanBaseBranchMetadata(t *testing.T) {
	for _, branch := range []string{"HEAD~1", "HEAD", "master^", "master^{commit}", "master:path", "@{-1}", "topic..other", "topic.lock", "topic.lock/child", ".hidden", "topic/.hidden", "topic//child", "topic/", "/topic", "topic.", "-topic", "topic name", "topic\tname", "topic\x7fname", "topic?", "topic*", "topic[1]", "topic\\name"} {
		t.Run(branch, func(t *testing.T) {
			data := strings.Replace(modelPlanFixture, "base_branch: master", fmt.Sprintf("base_branch: %q", branch), 1)
			if _, _, err := parsePlanMetadata([]byte(data), "active/test.md"); err == nil || !strings.Contains(err.Error(), "base_branch") {
				t.Fatalf("invalid branch accepted or wrong error: %v", err)
			}
		})
	}
	for _, branch := range []string{"master", "feat/stacked-parent", "release/v1.2", "日本語", "@"} {
		data := strings.Replace(modelPlanFixture, "base_branch: master", fmt.Sprintf("base_branch: %q", branch), 1)
		if _, _, err := parsePlanMetadata([]byte(data), "active/test.md"); err != nil {
			t.Fatalf("valid branch %q rejected: %v", branch, err)
		}
	}
}

func TestPlanBaseRevisionUsesOnlyBranchRefs(t *testing.T) {
	root := t.TempDir()
	planTestGit(t, root, "init", "-b", "master")
	planTestGit(t, root, "config", "user.email", "tests@example.invalid")
	planTestGit(t, root, "config", "user.name", "Tests")
	planTestGit(t, root, "commit", "--allow-empty", "-m", "base")
	base := planTestGit(t, root, "rev-parse", "HEAD")
	planTestGit(t, root, "tag", "tag-only")
	planTestGit(t, root, "tag", "topic")
	planTestGit(t, root, "update-ref", "refs/remotes/origin/remote-only", base)
	planTestGit(t, root, "update-ref", "refs/remotes/origin/topic", base)
	planTestGit(t, root, "commit", "--allow-empty", "-m", "next")
	head := planTestGit(t, root, "rev-parse", "HEAD")
	planTestGit(t, root, "branch", "topic")
	for _, tc := range []struct{ branch, want string }{{"master", head}, {"topic", head}, {"remote-only", base}} {
		got, err := planBaseRevision(root, tc.branch)
		if err != nil || got != tc.want {
			t.Fatalf("branch %q: got %s, %v; want %s", tc.branch, got, err, tc.want)
		}
	}
	for _, branch := range []string{"tag-only", "HEAD~1", "HEAD", base, "missing"} {
		if got, err := planBaseRevision(root, branch); err == nil {
			t.Fatalf("non-branch %q resolved to %s", branch, got)
		}
	}
	planTestGit(t, root, "checkout", "topic")
	planTestGit(t, root, "branch", "-D", "master")
	planTestGit(t, root, "update-ref", "refs/remotes/origin/master", base)
	if got, err := planBaseRevision(root, "master"); err != nil || got != base {
		t.Fatalf("origin/master fallback: %s, %v", got, err)
	}
}
