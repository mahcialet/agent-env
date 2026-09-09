package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func replayGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	// Isolate fixtures from developer signing, hooks and identity configuration.
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func replayFixture(t *testing.T, invalidStart bool) (string, string, string, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git unavailable")
	}
	root := t.TempDir()
	replayGit(t, root, "init", "-b", "master")
	replayGit(t, root, "config", "user.email", "test@example.invalid")
	replayGit(t, root, "config", "user.name", "Test")
	replayGit(t, root, "commit", "--allow-empty", "-m", "base")
	start := replayGit(t, root, "rev-parse", "HEAD")
	if invalidStart {
		replayGit(t, root, "checkout", "--orphan", "unrelated")
		replayGit(t, root, "commit", "--allow-empty", "-m", "unrelated root")
		start = replayGit(t, root, "rev-parse", "HEAD")
		replayGit(t, root, "checkout", "master")
	}
	replayGit(t, root, "checkout", "-b", "feature")
	planPath := "docs/exec-plans/completed/example.md"
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, filepath.FromSlash(planPath))), 0755); err != nil {
		t.Fatal(err)
	}
	document := "---\nstatus: completed\nowner: maintainers\n---\n\nStarting revision: `" + start + "`\n"
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(planPath)), []byte(document), 0644); err != nil {
		t.Fatal(err)
	}
	replayGit(t, root, "add", planPath)
	replayGit(t, root, "commit", "-m", "implementation and historical archive")
	head := replayGit(t, root, "rev-parse", "HEAD")
	replayGit(t, root, "checkout", "master")
	replayGit(t, root, "merge", "--no-ff", "feature", "-m", "merge feature")
	merge := replayGit(t, root, "rev-parse", "HEAD")
	return root, planPath, head, merge
}

func TestPlanReplayVerifiesGitHistoryWithoutInventingReview(t *testing.T) {
	root, plan, head, merge := replayFixture(t, false)
	var out bytes.Buffer
	if err := executePlanReplay(root, []string{"--plan", plan, "--merge", merge, "--pr", "12"}, &out); err != nil {
		t.Fatal(err)
	}
	var result planReplayResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.BranchHead != head || result.MergeSHA != merge || !result.ArchivedAtMerge || len(result.Commits) != 1 {
		t.Fatalf("wrong historical evidence: %+v", result)
	}
	if !strings.Contains(result.PRIdentityEvidence, "unverified") || !strings.HasPrefix(result.ReviewEvidence, "unknown") || result.Nodes[1].State != "unknown" || result.Nodes[2].State != "awaiting-reconciliation" {
		t.Fatalf("inferred unproved authorization: %+v", result)
	}
	before := out.String()
	out.Reset()
	if err := executePlanReplay(root, []string{"--plan", plan, "--merge", merge, "--pr", "12"}, &out); err != nil {
		t.Fatal(err)
	}
	if before != out.String() {
		t.Fatal("replay is nondeterministic")
	}
	if got := replayGit(t, root, "status", "--porcelain"); got != "" {
		t.Fatalf("replay mutated repository: %s", got)
	}
}

func TestPlanReplayRejectsUnprovenHistory(t *testing.T) {
	root, plan, head, merge := replayFixture(t, false)
	cases := [][]string{
		{"--plan", plan, "--merge", head},
		{"--plan", plan, "--merge", strings.Repeat("f", 40)},
		{"--plan", "docs/exec-plans/completed/missing.md", "--merge", merge},
		{"--plan", "docs/exec-plans/active/example.md", "--merge", merge},
		{"--plan", "docs/exec-plans/completed/../active/example.md", "--merge", merge},
		{"--plan", plan, "--merge", "HEAD"},
		{"--plan", plan, "--merge", merge, "--pr", "0"},
		{"--plan", plan, "--merge", merge, "--pr", "-2"},
		{"--plan", plan, "--merge", merge, "extra"},
	}
	for _, args := range cases {
		var out bytes.Buffer
		if err := executePlanReplay(root, args, &out); err == nil {
			t.Errorf("invalid replay accepted: %v", args)
		}
		if out.Len() != 0 {
			t.Errorf("invalid replay emitted success evidence: %v", args)
		}
	}
	replayGit(t, root, "checkout", head)
	if err := executePlanReplay(root, []string{"--plan", plan, "--merge", merge}, &bytes.Buffer{}); err == nil {
		t.Fatal("unreachable merge accepted")
	}
	badRoot, badPlan, _, badMerge := replayFixture(t, true)
	if err := executePlanReplay(badRoot, []string{"--plan", badPlan, "--merge", badMerge}, &bytes.Buffer{}); err == nil {
		t.Fatal("unrelated starting revision accepted")
	}
}

func TestPlanReplayExercisesDependencySelectionAndParentFinalization(t *testing.T) {
	steps, err := simulatePlanReplay(true, strings.Repeat("a", 40))
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 3 {
		t.Fatalf("wrong steps: %+v", steps)
	}
	before, after, hypothetical := steps[0], steps[1], steps[2]
	if !before.Readiness[0].Selected || before.Readiness[1].Runnable || before.Readiness[2].Runnable {
		t.Fatalf("premature dependency satisfaction: %+v", before)
	}
	if !after.Readiness[1].Selected || after.Readiness[2].Runnable || !strings.Contains(strings.Join(after.Readiness[2].Reasons, " "), "EP-REPLAY-002") {
		t.Fatalf("child merge auto-completed parent: %+v", after)
	}
	if !hypothetical.Hypothetical || !hypothetical.Readiness[2].Selected || hypothetical.Graph[2].Status != "active" {
		t.Fatalf("hypothetical review became parent completion: %+v", hypothetical)
	}
	if after.Graph[1].Parent != "EP-REPLAY-003" || after.Graph[1].DependsOn[0].PlanID != "EP-REPLAY-001" {
		t.Fatal("parent and dependency relations were conflated")
	}
	unproved, err := simulatePlanReplay(false, "")
	if err != nil {
		t.Fatal(err)
	}
	if unproved[1].Readiness[1].Runnable || unproved[2].Readiness[2].Runnable {
		t.Fatal("unproven implementation merge released dependent plan")
	}
}

func TestPlanReplayRequiresArchiveDeliveryBySourceParent(t *testing.T) {
	for _, tc := range []struct {
		name       string
		sourceEdit string
		mergeEdit  string
		wantError  string
	}{
		{name: "unrelated later merge", wantError: "did not introduce or change"},
		{name: "archive removed by source and restored at merge", sourceEdit: "remove", mergeEdit: "restore", wantError: "missing at source parent"},
		{name: "archive edited only at merge", mergeEdit: "edit", wantError: "differs from source parent"},
		{name: "source archive changed again at merge", sourceEdit: "edit", mergeEdit: "edit", wantError: "differs from source parent"},
		{name: "source updates existing archive", sourceEdit: "edit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, plan, _, _ := replayFixture(t, false)
			filename := filepath.Join(root, filepath.FromSlash(plan))
			original, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			write := func(data []byte) {
				t.Helper()
				if err := os.WriteFile(filename, data, 0644); err != nil {
					t.Fatal(err)
				}
			}
			replayGit(t, root, "checkout", "-b", "later-feature")
			switch tc.sourceEdit {
			case "remove":
				replayGit(t, root, "rm", plan)
			case "edit":
				write(append(append([]byte{}, original...), []byte("\nSource acceptance evidence.\n")...))
				replayGit(t, root, "add", plan)
			}
			replayGit(t, root, "commit", "--allow-empty", "-m", "later work")
			replayGit(t, root, "checkout", "master")
			replayGit(t, root, "merge", "--no-ff", "--no-commit", "later-feature")
			if tc.mergeEdit != "" {
				if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
					t.Fatal(err)
				}
				if tc.mergeEdit == "restore" {
					write(original)
				} else {
					// A trailing newline must count as a different blob, even though
					// replay's document reader trims surrounding whitespace.
					data, err := os.ReadFile(filename)
					if err != nil {
						t.Fatal(err)
					}
					write(append(data, '\n'))
				}
				replayGit(t, root, "add", plan)
			}
			replayGit(t, root, "commit", "-m", "later merge")
			merge := replayGit(t, root, "rev-parse", "HEAD")
			var out bytes.Buffer
			err = executePlanReplay(root, []string{"--plan", plan, "--merge", merge}, &out)
			if tc.wantError == "" {
				if err != nil || out.Len() == 0 {
					t.Fatalf("valid archive update rejected: %v; output: %s", err, out.String())
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantError) || out.Len() != 0 {
				t.Fatalf("want rejection %q without evidence, got error %v; output: %s", tc.wantError, err, out.String())
			}
		})
	}
}
