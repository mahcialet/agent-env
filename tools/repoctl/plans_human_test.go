package main

import (
	"bytes"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func humanFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	data := strings.Replace(modelPlanFixture, "plan_type: implementation", "plan_type: human-validation\nexecution_mode: human-kick", 1)
	modelWritePair(t, root, "human", data)
	dir := filepath.Join(root, "docs", "exec-plans", "validation")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	c := humanContract{PlanID: "EP-TEST-001", Scenarios: []humanScenario{{ID: "EP-TEST-001-01", Purpose: "test", Actions: []string{"observe"}, Expected: "isolated", PassCriteria: "evidence"}}}
	raw, _ := json.Marshal(c)
	path := filepath.Join(dir, "EP-TEST-001.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return root, path
}
func TestHumanNoKickDoesNotReadOrWrite(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "bundle")
	err := executePlanHuman(root, []string{"preflight", "--plan", "EP-TEST-001", "--config", "missing", "--evidence-dir", dir}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--kick") {
		t.Fatalf("expected kick refusal: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("created evidence without kick")
	}
}
func TestHumanPreflightBlockedEvidence(t *testing.T) {
	root, path := humanFixture(t)
	var c humanContract
	if err := readHumanJSON(path, &c); err != nil {
		t.Fatal(err)
	}
	c.Executables = []string{"definitely-missing-human-fixture-command"}
	c.Endpoints = []string{"worker"}
	raw, _ := json.Marshal(c)
	os.WriteFile(path, raw, 0600)
	cfg := filepath.Join(root, "config.json")
	os.WriteFile(cfg, []byte(`{"endpoints":{}}`), 0600)
	dir := filepath.Join(root, "bundle")
	err := executePlanHuman(root, []string{"preflight", "--plan", "EP-TEST-001", "--kick", "--config", cfg, "--evidence-dir", dir}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("missing prerequisites passed")
	}
	var e humanEvidence
	if err := readHumanJSON(filepath.Join(dir, "evidence.json"), &e); err != nil {
		t.Fatal(err)
	}
	if e.Result != "BLOCKED" || len(e.Blockers) != 2 || e.ActionRequired != "paused-plan-update" {
		t.Fatalf("bad evidence: %+v", e)
	}
	if _, err := os.Stat(filepath.Join(dir, "evidence.md")); err != nil {
		t.Fatal(err)
	}
}
func TestHumanReadyIsNotScenarioPass(t *testing.T) {
	root, path := humanFixture(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	var c humanContract
	readHumanJSON(path, &c)
	c.Endpoints = []string{"worker"}
	raw, _ := json.Marshal(c)
	os.WriteFile(path, raw, 0600)
	cfg := filepath.Join(root, "config.json")
	raw, _ = json.Marshal(humanConfig{Endpoints: map[string]string{"worker": listener.Addr().String()}})
	os.WriteFile(cfg, raw, 0600)
	dir := filepath.Join(root, "bundle")
	out := &bytes.Buffer{}
	err = executePlanHuman(root, []string{"preflight", "--plan", "EP-TEST-001", "--kick", "--config", cfg, "--evidence-dir", dir}, out)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(filepath.Join(dir, "evidence.json"))
	if strings.Contains(string(raw), listener.Addr().String()) || !strings.Contains(string(raw), "READY") || strings.Contains(string(raw), "PASS") {
		t.Fatalf("misleading or sensitive result: %s", raw)
	}
}
func TestHumanRecordRejectsUnknownBlankAndUntrackedFinding(t *testing.T) {
	for _, kind := range []string{"unknown", "blank", "finding", "valid"} {
		t.Run(kind, func(t *testing.T) {
			root, _ := humanFixture(t)
			path := filepath.Join(root, "observation.json")
			body := `{"observation":"Worker cleanup matches expected absence","evidence_refs":["private sanitized log"]}`
			if kind == "blank" {
				body = `{"observation":" ","evidence_refs":[]}`
			}
			os.WriteFile(path, []byte(body), 0600)
			scenario := "EP-TEST-001-01"
			if kind == "unknown" {
				scenario = "EP-TEST-001-99"
			}
			result := "PASS"
			if kind == "finding" {
				result = "FINDING"
			}
			dir := filepath.Join(root, "bundle")
			err := executePlanHuman(root, []string{"record", "--plan", "EP-TEST-001", "--kick", "--scenario", scenario, "--result", result, "--evidence", path, "--evidence-dir", dir}, &bytes.Buffer{})
			if kind == "valid" {
				if err != nil {
					t.Fatal(err)
				}
				var e humanEvidence
				readHumanJSON(filepath.Join(dir, "evidence.json"), &e)
				if len(e.EvidenceSHA256) != 64 || e.Scenario != scenario {
					t.Fatal("missing proof")
				}
				if err = writeHumanEvidence(dir, e); err == nil {
					t.Fatal("overwrote old evidence")
				}
			} else {
				if err == nil {
					t.Fatal("invalid result accepted")
				}
				if _, err = os.Stat(dir); !os.IsNotExist(err) {
					t.Fatal("invalid input wrote evidence")
				}
			}
		})
	}
}

func TestHumanFindingRequiresTrackedReviewAndKeepsFollowUp(t *testing.T) {
	root, _ := humanFixture(t)
	review := strings.Replace(modelPlanFixture, "EP-TEST-001", "EP-TEST-002", 1)
	review = strings.Replace(review, "plan_type: implementation", "plan_type: review", 1)
	modelWritePair(t, root, "review", review)
	observation := filepath.Join(root, "observation.json")
	if err := os.WriteFile(observation, []byte(`{"observation":"Unexpected duplicate effect","evidence_refs":["sanitized receipt"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "bundle")
	err := executePlanHuman(root, []string{"record", "--plan", "EP-TEST-001", "--kick", "--scenario", "EP-TEST-001-01", "--result", "FINDING", "--follow-up-plan", "EP-TEST-002", "--evidence", observation, "--evidence-dir", dir}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	var e humanEvidence
	if err := readHumanJSON(filepath.Join(dir, "evidence.json"), &e); err != nil {
		t.Fatal(err)
	}
	if e.Result != "FINDING" || e.FollowUpPlan != "EP-TEST-002" || e.ActionRequired != "review-plan-update" {
		t.Fatalf("lost tracking: %+v", e)
	}
}

func TestHumanJSONRejectsOversizeAndTrailingData(t *testing.T) {
	first := `{"observation":"observed","evidence_refs":["receipt"]}`
	cases := map[string]string{
		"oversized whitespace":    first + strings.Repeat(" ", humanJSONLimit),
		"hidden second object":    first + strings.Repeat(" ", humanJSONLimit) + `{}`,
		"second object":           first + ` {}`,
		"trailing malformed data": first + ` unexpected`,
		"null":                    `null`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			root, _ := humanFixture(t)
			path := filepath.Join(root, "observation.json")
			if err := os.WriteFile(path, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			var observation humanObservation
			if _, err := readHumanJSONSnapshot(path, &observation); err == nil {
				t.Fatal("invalid snapshot accepted")
			}
			dir := filepath.Join(root, "bundle")
			if err := executePlanHuman(root, []string{"record", "--plan", "EP-TEST-001", "--kick", "--scenario", "EP-TEST-001-01", "--result", "PASS", "--evidence", path, "--evidence-dir", dir}, &bytes.Buffer{}); err == nil {
				t.Fatal("invalid evidence recorded")
			}
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Fatal("invalid evidence created bundle")
			}
		})
	}
}
func TestHumanJSONSnapshotRetainsValidatedBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "observation.json")
	original := []byte(" {\"observation\":\"observed\",\"evidence_refs\":[\"receipt\"]}\n")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	var observation humanObservation
	snapshot, err := readHumanJSONSnapshot(path, &observation)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(snapshot, original) || observation.Observation != "observed" {
		t.Fatal("validated snapshot changed")
	}
}
