package main

import "testing"

func TestCompletedPlanDependencies(t *testing.T) {
	for _, tc := range []struct {
		name, mode, state                                         string
		inherited, wrongMerge, staleMetadata, outsideBase, wantOK bool
	}{
		{name: "merged complete", state: "completed", wantOK: true},
		{name: "merged active", state: "active"},
		{name: "merged abandoned", state: "abandoned"},
		{name: "merged false proof", state: "completed", wrongMerge: true},
		{name: "merged outside consumer base", state: "completed", outsideBase: true},
		{name: "stacked completed inherited", mode: "stacked", state: "completed", inherited: true, wantOK: true},
		{name: "stacked completed not inherited", mode: "stacked", state: "completed"},
		{name: "stacked active inherited", mode: "stacked", state: "active", inherited: true, wantOK: true},
		{name: "stacked active stale metadata", mode: "stacked", state: "active", inherited: true, staleMetadata: true},
		{name: "stacked abandoned", mode: "stacked", state: "abandoned", inherited: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := planTestRepo(t)
			initial := planTestGit(t, root, "rev-parse", "HEAD")
			planTestGit(t, root, "switch", "-c", "feat/ep-test-001")
			a := commitProvenancePlan(t, root, "a", "EP-TEST-001", "")
			start := initial
			if tc.inherited {
				start = "HEAD"
			}
			planTestGit(t, root, "switch", "-c", "feat/ep-test-002", start)
			mode := tc.mode
			if mode == "" {
				mode = "merged"
			}
			b := commitProvenancePlan(t, root, "b", "EP-TEST-002", "depends_on:\n  - plan_id: EP-TEST-001\n    satisfaction: "+mode+"\n")
			planTestGit(t, root, "switch", "master")
			if tc.state == "completed" {
				if tc.outsideBase {
					planTestGit(t, root, "switch", "-c", "other-base")
					a.BaseBranch = "other-base"
				}
				planTestGit(t, root, "merge", "--no-ff", "feat/ep-test-001", "-m", "merge dependency")
				a.MergeCommit = planTestGit(t, root, "rev-parse", "HEAD")
				planTestGit(t, root, "branch", "-d", "feat/ep-test-001")
				if tc.outsideBase {
					planTestGit(t, root, "switch", "master")
				}
			}
			a.Status = tc.state
			if tc.wrongMerge {
				a.MergeCommit = initial
			}
			if tc.staleMetadata {
				a.Priority++
			}
			planTestGit(t, root, "merge", "--no-ff", "feat/ep-test-002", "-m", "merge consumer")
			b.Status = "completed"
			b.MergeCommit = planTestGit(t, root, "rev-parse", "HEAD")
			// Completed consumer evidence cannot depend on this branch surviving.
			planTestGit(t, root, "branch", "-d", "feat/ep-test-002")
			g := &planGraph{Plans: []planMetadata{a, b}, ByID: map[string]planMetadata{a.PlanID: a, b.PlanID: b}}
			err := planCompletedDependencies(root, g, b)
			if (err == nil) != tc.wantOK {
				t.Fatalf("expected success %v, got %v", tc.wantOK, err)
			}
		})
	}
}
