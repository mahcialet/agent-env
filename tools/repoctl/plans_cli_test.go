package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func planTestGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	s, err := planGit(root, args...)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func planTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	planTestGit(t, root, "init", "-b", "master")
	planTestGit(t, root, "config", "user.email", "fixture@example.invalid")
	planTestGit(t, root, "config", "user.name", "Fixture")
	planTestGit(t, root, "commit", "--allow-empty", "-m", "base")
	return root
}
func TestPlanProvenanceRejectsWrongTrailerBranchAndPR(t *testing.T) {
	root := planTestRepo(t)
	p := planMetadata{PlanID: "EP-TEST-001", PlanType: "implementation", BaseBranch: "master"}
	planTestGit(t, root, "switch", "-c", expectedPlanBranch(p))
	planTestGit(t, root, "commit", "--allow-empty", "-m", "implementation", "-m", "ExecPlan: EP-TEST-001")
	if err := planProvenance(root, p, "ExecPlan: EP-TEST-001"); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"ExecPlan: EP-TEST-002", "```\nExecPlan: EP-TEST-001\n```", "<!-- ExecPlan: EP-TEST-001 -->", "ExecPlan: EP-TEST-001\nExecPlan: EP-TEST-001"} {
		if err := planProvenance(root, p, body); err == nil {
			t.Errorf("accepted %q", body)
		}
	}
	planTestGit(t, root, "commit", "--allow-empty", "-m", "untraced change")
	if err := planProvenance(root, p, ""); err == nil {
		t.Fatal("accepted missing trailer")
	}
	planTestGit(t, root, "switch", "master")
	if err := planProvenance(root, p, ""); err == nil {
		t.Fatal("accepted master branch")
	}
}
func TestPlanStackedReadinessChecksConsumerHead(t *testing.T) {
	root := planTestRepo(t)
	initial := planTestGit(t, root, "rev-parse", "HEAD")
	a := planMetadata{PlanID: "EP-TEST-001", PlanType: "implementation", Status: "active", BaseBranch: "master"}
	b := planMetadata{PlanID: "EP-TEST-002", PlanType: "implementation", Status: "active", BaseBranch: "stack-base", DependsOn: []planDependency{{PlanID: a.PlanID, Satisfaction: "stacked"}}}
	planTestGit(t, root, "switch", "-c", expectedPlanBranch(a))
	planTestGit(t, root, "commit", "--allow-empty", "-m", "dependency")
	planTestGit(t, root, "branch", "stack-base")
	planTestGit(t, root, "branch", expectedPlanBranch(b), initial)
	g := &planGraph{Plans: []planMetadata{a, b}, ByID: map[string]planMetadata{a.PlanID: a, b.PlanID: b}}
	ctx, err := planGitReadiness(root, g)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Stacked[b.PlanID+"/"+a.PlanID] {
		t.Fatal("base inclusion accepted despite stale consumer branch")
	}
	planTestGit(t, root, "switch", expectedPlanBranch(b))
	planTestGit(t, root, "merge", "--ff-only", expectedPlanBranch(a))
	ctx, err = planGitReadiness(root, g)
	if err != nil {
		t.Fatal(err)
	}
	if !ctx.Stacked[b.PlanID+"/"+a.PlanID] {
		t.Fatal("actual stacked ancestor not observed")
	}
}
func TestPlanIdentityCannotDisappearOnRename(t *testing.T) {
	root := planTestRepo(t)
	path := "docs/exec-plans/active/original.md"
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, modelPlanYAML("EP-TEST-001", "active", ""), 0644); err != nil {
		t.Fatal(err)
	}
	planTestGit(t, root, "add", ".")
	planTestGit(t, root, "commit", "-m", "existing Plan")
	g := &planGraph{Plans: []planMetadata{{PlanID: "EP-TEST-002", BaseBranch: "master"}}, ByID: map[string]planMetadata{"EP-TEST-002": {PlanID: "EP-TEST-002"}}}
	if err := planIdentityHistory(root, g); err == nil || !strings.Contains(err.Error(), "EP-TEST-001") {
		t.Fatalf("identity rewrite accepted: %v", err)
	}
	if err := planIdentityHistory(root, &planGraph{ByID: map[string]planMetadata{}}); err == nil {
		t.Fatal("deleting all lifecycle plans bypassed immutable IDs")
	}
	g.ByID["EP-TEST-001"] = planMetadata{PlanID: "EP-TEST-001", Path: "docs/exec-plans/abandoned/renamed.md"}
	if err := planIdentityHistory(root, g); err != nil {
		t.Fatal(err)
	}
}
func TestStructuredTranslationMetadataOnlyForPlans(t *testing.T) {
	data := modelPlanYAML("EP-TEST-001", "active", "workstreams:\n  - storage\n")
	if _, err := translationMetadata(data); err == nil {
		t.Fatal("legacy parser unexpectedly accepts nested YAML")
	}
	fields, err := documentationTranslationMetadata("docs/exec-plans/active/p.md", data)
	if err != nil {
		t.Fatal(err)
	}
	if fields["plan_id"] != "EP-TEST-001" {
		t.Fatal(fields)
	}
	if _, err := documentationTranslationMetadata("docs/design-docs/p.md", data); err == nil {
		t.Fatal("nonplan parser weakened")
	}
}

func TestCompletedPlanQuotedIDStillRequiresSections(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Instructions\n"), 0644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "docs", "exec-plans", "completed", "quoted.md")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	data := strings.Replace(string(modelPlanYAML("EP-TEST-001", "completed", "merge_commit: "+strings.Repeat("a", 40)+"\n")), "plan_id:", "\"plan_id\":", 1) + "\n# Incomplete\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	err := documentStructureCheck(root, []string{path})
	if err == nil || !strings.Contains(err.Error(), "AGENTENV-DOC-005") {
		t.Fatalf("quoted ID bypassed section check: %v", err)
	}
}

func modelPlanYAML(id, status, extra string) []byte {
	return []byte("---\nplan_id: " + id + "\nplan_type: implementation\nstatus: " + status + "\nowner: test\nlast_verified: 2026-09-09\nbase_branch: master\npriority: 1\nmerge_policy: guarded\n" + extra + "---\n")
}

func TestPlanMergeProofRejectsUnrelatedContainingCommit(t *testing.T) {
	root := planTestRepo(t)
	p := planMetadata{PlanID: "EP-TEST-001", PlanType: "implementation", Status: "completed", BaseBranch: "master"}
	path := filepath.Join(root, "docs", "exec-plans", "active", "p.md")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, modelPlanYAML(p.PlanID, "active", ""), 0644); err != nil {
		t.Fatal(err)
	}
	planTestGit(t, root, "add", ".")
	planTestGit(t, root, "commit", "-m", "draft metadata only")
	p.MergeCommit = planTestGit(t, root, "rev-parse", "HEAD")
	if err := planMergeProof(root, p, p.MergeCommit); err == nil {
		t.Fatal("arbitrary containing commit treated as Plan merge")
	}
	planTestGit(t, root, "switch", "-c", expectedPlanBranch(p))
	planTestGit(t, root, "commit", "--allow-empty", "-m", "implementation", "-m", "ExecPlan: EP-TEST-001")
	planTestGit(t, root, "switch", "master")
	planTestGit(t, root, "merge", "--no-ff", expectedPlanBranch(p), "-m", "merge")
	p.MergeCommit = planTestGit(t, root, "rev-parse", "HEAD")
	planTestGit(t, root, "branch", "-d", expectedPlanBranch(p))
	if err := planMergeProof(root, p, p.MergeCommit); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleDirectoriesKeepBilingualStructureChecks(t *testing.T) {
	for _, state := range []string{"draft", "active", "paused", "completed", "abandoned"} {
		t.Run(state, func(t *testing.T) {
			root := fixture(t)
			old := filepath.Join(root, "docs", "exec-plans", "active", "plan.md")
			data, err := os.ReadFile(old)
			if err != nil {
				t.Fatal(err)
			}
			text := strings.Replace(string(data), "status: active", "status: "+state, 1)
			extra := ""
			switch state {
			case "draft":
				extra = "promotion_criteria: [Maintainer approval]\n"
			case "paused":
				extra = "pause_reason: Environment unavailable\nresume_when: Environment supplied\n"
			case "completed":
				extra = "merge_commit: " + strings.Repeat("a", 40) + "\n"
			case "abandoned":
				extra = "abandonment_reason: Replaced deliberately\n"
			}
			text = strings.Replace(text, "owner: maintainers", "owner: maintainers\n"+strings.TrimSuffix(extra, "\n"), 1)
			path := "docs/exec-plans/" + state + "/plan.md"
			dest := filepath.Join(root, filepath.FromSlash(path))
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				t.Fatal(err)
			}
			if state != "active" {
				if err := os.Rename(old, dest); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(strings.TrimSuffix(old, ".md")+".ja.md", strings.TrimSuffix(dest, ".md")+".ja.md"); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(dest, []byte(text), 0644); err != nil {
				t.Fatal(err)
			}
			pairFixtureDocument(t, root, path)
			if err := docsCheck(root); err != nil {
				t.Fatalf("valid %s pair: %v", state, err)
			}
			ja := strings.TrimSuffix(dest, ".md") + ".ja.md"
			b, err := os.ReadFile(ja)
			if err != nil {
				t.Fatal(err)
			}
			b = []byte(strings.Replace(string(b), "## Progress", "# Progress", 1))
			if err := os.WriteFile(ja, b, 0644); err != nil {
				t.Fatal(err)
			}
			if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "DOC-005") {
				t.Fatalf("%s Japanese required section bypass: %v", state, err)
			}
		})
	}
}
