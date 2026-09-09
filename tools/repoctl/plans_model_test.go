package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modelPlanFixture = `---
plan_id: EP-TEST-001
plan_type: implementation
status: active
owner: test
last_verified: 2026-09-09
priority: 10
base_branch: master
merge_policy: guarded
---
# Test plan
`

func TestPlanMetadataStrictValidation(t *testing.T) {
	tests := []struct{ name, from, to, path string }{
		{"status path", "status: active", "status: paused", "active/test.md"},
		{"unknown type", "implementation", "automation", "active/test.md"},
		{"missing ID", "plan_id: EP-TEST-001\n", "", "active/test.md"},
		{"invalid ID", "EP-TEST-001", "test", "active/test.md"},
		{"string priority", "priority: 10", "priority: \"10\"", "active/test.md"},
		{"negative priority", "priority: 10", "priority: -1", "active/test.md"},
		{"missing priority", "priority: 10\n", "", "active/test.md"},
		{"missing base", "base_branch: master\n", "", "active/test.md"},
		{"unknown field", "priority: 10", "priority: 10\nprioritty: 2", "active/test.md"},
		{"scalar dependencies", "priority: 10", "priority: 10\ndepends_on: EP-TEST-002", "active/test.md"},
		{"numeric dependency", "priority: 10", "priority: 10\ndepends_on: [{plan_id: 12}]", "active/test.md"},
		{"duplicate dependency", "priority: 10", "priority: 10\ndepends_on: [{plan_id: EP-TEST-002}, {plan_id: EP-TEST-002}]", "active/test.md"},
		{"bad satisfaction", "priority: 10", "priority: 10\ndepends_on: [{plan_id: EP-TEST-002, satisfaction: complete}]", "active/test.md"},
		{"paused reason", "status: active", "status: paused", "paused/test.md"},
		{"draft criteria", "status: active", "status: draft", "draft/test.md"},
		{"abandoned reason", "status: active", "status: abandoned", "abandoned/test.md"},
		{"completed evidence", "status: active", "status: completed", "completed/test.md"},
		{"human kick", "implementation", "human-validation", "active/test.md"},
		{"invalid date", "2026-09-09", "2026-99-99", "active/test.md"},
		{"yaml duplicate", "priority: 10", "priority: 10\npriority: 20", "active/test.md"},
		{"new legacy bypass", "plan_id: EP-TEST-001\n", "", "completed/unregistered.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parsePlanMetadata([]byte(strings.Replace(modelPlanFixture, tt.from, tt.to, 1)), tt.path)
			if err == nil {
				t.Fatal("invalid metadata accepted")
			}
		})
	}
	p, legacy, err := parsePlanMetadata([]byte(modelPlanFixture), "active/test.md")
	if err != nil || legacy || p.PlanID != "EP-TEST-001" {
		t.Fatalf("valid metadata: %+v %v %v", p, legacy, err)
	}
}

func TestPlanMetadataLegacyIsBounded(t *testing.T) {
	data := []byte("---\nstatus: completed\nowner: historical\nlast_verified: 2026-01-01\n---\nOld plan\n")
	if _, legacy, err := parsePlanMetadata(data, "completed/agent-env-mvp.md"); err != nil || !legacy {
		t.Fatalf("historical exemption: %v %v", legacy, err)
	}
	if _, _, err := parsePlanMetadata(data, "completed/new-plan.md"); err == nil {
		t.Fatal("new plan bypassed schema")
	}
	if _, _, err := parsePlanMetadata(data, "active/agent-env-mvp.md"); err == nil {
		t.Fatal("active plan bypassed schema")
	}
}

func modelWritePair(t *testing.T, root, name, data string) {
	t.Helper()
	dir := filepath.Join(root, "docs", "exec-plans", "active")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{".md", ".ja.md"} {
		if err := os.WriteFile(filepath.Join(dir, name+suffix), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadPlanGraphPairsAndReferences(t *testing.T) {
	for _, kind := range []string{"valid", "missing dependency", "duplicate ID", "parity", "orphan", "missing Japanese"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			data := modelPlanFixture
			if kind == "missing dependency" {
				data = strings.Replace(data, "priority: 10", "priority: 10\ndepends_on: [{plan_id: EP-TEST-002}]", 1)
			}
			modelWritePair(t, root, "one", data)
			dir := filepath.Join(root, "docs", "exec-plans", "active")
			switch kind {
			case "duplicate ID":
				modelWritePair(t, root, "two", data)
			case "parity":
				if err := os.WriteFile(filepath.Join(dir, "one.ja.md"), []byte(strings.Replace(data, "priority: 10", "priority: 2", 1)), 0644); err != nil {
					t.Fatal(err)
				}
			case "orphan":
				if err := os.Remove(filepath.Join(dir, "one.md")); err != nil {
					t.Fatal(err)
				}
			case "missing Japanese":
				if err := os.Remove(filepath.Join(dir, "one.ja.md")); err != nil {
					t.Fatal(err)
				}
			}
			g, err := loadPlanGraph(root)
			if kind == "valid" {
				if err != nil || len(g.Plans) != 1 {
					t.Fatalf("valid graph: %v", err)
				}
			} else if err == nil {
				t.Fatal("invalid graph accepted")
			}
		})
	}
}

func modelGraph(plans ...planMetadata) *planGraph {
	g := &planGraph{Plans: plans, ByID: map[string]planMetadata{}}
	for _, p := range plans {
		g.ByID[p.PlanID] = p
	}
	return g
}

func TestPlanGraphCyclesAreSeparate(t *testing.T) {
	a := planMetadata{PlanID: "EP-TEST-001"}
	b := planMetadata{PlanID: "EP-TEST-002"}
	a.Parent = b.PlanID
	b.Parent = a.PlanID
	if err := validatePlanGraph(modelGraph(a, b)); err == nil {
		t.Fatal("parent cycle accepted")
	}
	a.Parent = ""
	b.Parent = ""
	a.DependsOn = []planDependency{{PlanID: b.PlanID}}
	b.DependsOn = []planDependency{{PlanID: a.PlanID}}
	if err := validatePlanGraph(modelGraph(a, b)); err == nil {
		t.Fatal("dependency cycle accepted")
	}
	b.DependsOn = nil
	b.Parent = a.PlanID
	if err := validatePlanGraph(modelGraph(a, b)); err != nil {
		t.Fatalf("separate parent/dependency relations conflated: %v", err)
	}
}

func TestPlanReadinessRequiresBaseSpecificMergeEvidence(t *testing.T) {
	dep := planMetadata{PlanID: "EP-TEST-001", Status: "completed"}
	a := planMetadata{PlanID: "EP-TEST-002", Status: "active", DependsOn: []planDependency{{PlanID: dep.PlanID, Satisfaction: "merged"}}}
	b := a
	b.PlanID = "EP-TEST-003"
	g := modelGraph(dep, a, b)
	r := planReadiness(g, planReadinessContext{Merged: map[string]bool{a.PlanID + "/" + dep.PlanID: true}})
	if !r[1].Selected || r[2].Runnable {
		t.Fatalf("base-specific evidence ignored: %+v", r)
	}
	dep.Status = "active"
	g = modelGraph(dep, a)
	r = planReadiness(g, planReadinessContext{Merged: map[string]bool{a.PlanID + "/" + dep.PlanID: true}})
	if r[1].Runnable {
		t.Fatal("uncompleted dependency accepted from ancestry alone")
	}
}

func TestPlanReadinessSelectsOneAndRequiresHumanKick(t *testing.T) {
	a := planMetadata{PlanID: "EP-TEST-002", Status: "active", Priority: 2}
	b := planMetadata{PlanID: "EP-TEST-001", Status: "active", Priority: 2}
	human := planMetadata{PlanID: "EP-TEST-003", Status: "active", PlanType: "human-validation", Priority: 0}
	g := modelGraph(a, b, human)
	r := planReadiness(g, planReadinessContext{})
	if r[0].Runnable || !r[1].Selected || r[2].Selected || !r[2].Runnable {
		t.Fatalf("selection not deterministic: %+v", r)
	}
	for _, r := range planReadiness(g, planReadinessContext{Running: []string{a.PlanID}}) {
		if r.Runnable {
			t.Fatal("concurrency cap ignored")
		}
	}
	a.DependsOn = []planDependency{{PlanID: b.PlanID, Satisfaction: "stacked"}}
	r = planReadiness(modelGraph(a, b), planReadinessContext{Stacked: map[string]bool{a.PlanID + "/" + b.PlanID: true}})
	if !r[1].Runnable {
		t.Fatal("explicit stacked evidence rejected")
	}
}
