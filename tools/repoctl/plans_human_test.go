package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	c := humanContract{PlanID: "EP-TEST-001", Executables: []string{"git"}, Endpoints: []string{"worker"}, Scenarios: []humanScenario{{ID: "EP-TEST-001-01", Purpose: "test", Actions: []string{"observe"}, Expected: "isolated", PassCriteria: "evidence"}}}
	raw, _ := json.Marshal(c)
	path := filepath.Join(dir, "EP-TEST-001.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	planTestGit(t, root, "init")
	planTestGit(t, root, "config", "user.name", "Test")
	planTestGit(t, root, "config", "user.email", "test@example.invalid")
	planTestGit(t, root, "add", "docs/exec-plans")
	commitHumanContract(t, root)
	return root, path
}
func commitHumanContract(t *testing.T, root string) {
	t.Helper()
	planTestGit(t, root, "add", "docs/exec-plans/validation")
	if planTestGit(t, root, "diff", "--cached", "--name-only") == "" {
		return
	}
	planTestGit(t, root, "commit", "-m", "scenario contract")
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
	commitHumanContract(t, root)
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
	commitHumanContract(t, root)
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

func TestHumanJSONRejectsDuplicateKeys(t *testing.T) {
	for name, body := range map[string]string{
		"top level":     `{"observation":"failed","observation":"passed","evidence_refs":["receipt"]}`,
		"escaped alias": `{"observation":"failed","\u006fbservation":"passed","evidence_refs":["receipt"]}`,
		"nested map":    `{"endpoints":{"worker":"first","worker":"second"}}`,
		"array object":  `{"scenarios":[{"id":"first","id":"second"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.json")
			if err := os.WriteFile(path, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			var target any
			if _, err := readHumanJSONSnapshot(path, &target); err == nil || !strings.Contains(err.Error(), "duplicate") {
				t.Fatalf("expected duplicate rejection: %v", err)
			}
			if target != nil {
				t.Fatal("duplicate input decoded into target")
			}
		})
	}
	// Names may recur in distinct objects; JSON member uniqueness is per object.
	path := filepath.Join(t.TempDir(), "valid.json")
	if err := os.WriteFile(path, []byte(`{"scenarios":[{"id":"first"},{"id":"second"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	var target any
	if err := readHumanJSON(path, &target); err != nil {
		t.Fatal(err)
	}
}

func TestHumanDuplicateInputHasNoEffects(t *testing.T) {
	for _, mode := range []string{"preflight", "record", "contract"} {
		t.Run(mode, func(t *testing.T) {
			root, contractPath := humanFixture(t)
			listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1")})
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			var contract humanContract
			if err := readHumanJSON(contractPath, &contract); err != nil {
				t.Fatal(err)
			}
			contract.Endpoints = []string{"worker"}
			raw, err := json.Marshal(contract)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "contract" {
				raw = bytes.Replace(raw, []byte(`"purpose":"test"`), []byte(`"purpose":"first","purpose":"test"`), 1)
			}
			if err := os.WriteFile(contractPath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			commitHumanContract(t, root)
			configPath := filepath.Join(root, "config.json")
			address, _ := json.Marshal(listener.Addr().String())
			config := `{"endpoints":{"worker":` + string(address) + `}}`
			if mode == "preflight" {
				config = `{"endpoints":{"worker":"127.0.0.1:1","worker":` + string(address) + `}}`
			}
			if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "bundle")
			args := []string{"preflight", "--plan", "EP-TEST-001", "--kick", "--config", configPath, "--evidence-dir", dir}
			if mode == "record" {
				observation := filepath.Join(root, "observation.json")
				if err := os.WriteFile(observation, []byte(`{"observation":"failed","observation":"passed","evidence_refs":["receipt"]}`), 0600); err != nil {
					t.Fatal(err)
				}
				args = []string{"record", "--plan", "EP-TEST-001", "--kick", "--scenario", "EP-TEST-001-01", "--result", "PASS", "--evidence", observation, "--evidence-dir", dir}
			}
			var out bytes.Buffer
			if err := executePlanHuman(root, args, &out); err == nil || !strings.Contains(err.Error(), "duplicate") {
				t.Fatalf("expected duplicate rejection: %v", err)
			}
			if out.Len() != 0 {
				t.Fatal("invalid input emitted evidence")
			}
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Fatal("invalid input created evidence bundle")
			}
			if err := listener.SetDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
				t.Fatal(err)
			}
			conn, err := listener.Accept()
			if err == nil {
				conn.Close()
				t.Fatal("invalid input probed endpoint")
			}
			if timeout, ok := err.(net.Error); !ok || !timeout.Timeout() {
				t.Fatalf("unexpected accept error: %v", err)
			}
		})
	}
}

func TestHumanContractRequiresCommittedIdentity(t *testing.T) {
	for _, kind := range []string{"dirty", "staged", "untracked", "committed"} {
		t.Run(kind, func(t *testing.T) {
			root, path := humanFixture(t)
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			changed := bytes.Replace(original, []byte(`"expected":"isolated"`), []byte(`"expected":"changed"`), 1)
			if kind != "committed" {
				if err := os.WriteFile(path, changed, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "staged" {
				planTestGit(t, root, "add", path)
			}
			if kind == "untracked" {
				planTestGit(t, root, "rm", "--cached", path)
				planTestGit(t, root, "commit", "-m", "remove contract")
			}
			observation := filepath.Join(root, "observation.json")
			if err := os.WriteFile(observation, []byte(`{"observation":"checked","evidence_refs":["receipt"]}`), 0600); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "bundle")
			var out bytes.Buffer
			err = executePlanHuman(root, []string{"record", "--plan", "EP-TEST-001", "--kick", "--scenario", "EP-TEST-001-01", "--result", "PASS", "--evidence", observation, "--evidence-dir", dir}, &out)
			if kind != "committed" {
				if err == nil {
					t.Fatal("mutable contract accepted")
				}
				if out.Len() != 0 {
					t.Fatal("mutable contract emitted evidence")
				}
				if _, err := os.Stat(dir); !os.IsNotExist(err) {
					t.Fatal("mutable contract created bundle")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var e humanEvidence
			if err := readHumanJSON(filepath.Join(dir, "evidence.json"), &e); err != nil {
				t.Fatal(err)
			}
			revision := planTestGit(t, root, "rev-parse", "HEAD")
			digest := fmt.Sprintf("%x", sha256.Sum256(original))
			if e.ContractRevision != revision || e.ContractSHA256 != digest {
				t.Fatalf("unbound evidence: %+v", e)
			}
			if err := os.WriteFile(path, changed, 0600); err != nil {
				t.Fatal(err)
			}
			commitHumanContract(t, root)
			archived, err := os.ReadFile(filepath.Join(dir, "evidence.md"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(archived), revision) || !strings.Contains(string(archived), digest) {
				t.Fatal("Markdown lost original contract identity")
			}
			committed, err := planGit(root, "show", e.ContractRevision+":docs/exec-plans/validation/EP-TEST-001.json")
			if err != nil || committed != string(original) {
				t.Fatalf("cannot recover original contract: %v", err)
			}
		})
	}
}

func TestHumanInvalidEvidenceDestinationNeverProbes(t *testing.T) {
	for _, kind := range []string{"existing", "missing-parent", "parent-file"} {
		t.Run(kind, func(t *testing.T) {
			root, path := humanFixture(t)
			listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1")})
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			var c humanContract
			if err := readHumanJSON(path, &c); err != nil {
				t.Fatal(err)
			}
			c.Endpoints = []string{"worker"}
			raw, _ := json.Marshal(c)
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			commitHumanContract(t, root)
			cfg := filepath.Join(root, "config.json")
			raw, _ = json.Marshal(humanConfig{Endpoints: map[string]string{"worker": listener.Addr().String()}})
			if err := os.WriteFile(cfg, raw, 0600); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "bundle")
			if kind == "existing" {
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "evidence.json"), []byte("retained"), 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				parent := filepath.Join(root, "parent")
				if kind == "parent-file" {
					if err := os.WriteFile(parent, []byte("file"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				dir = filepath.Join(parent, "bundle")
			}
			var out bytes.Buffer
			err = executePlanHuman(root, []string{"preflight", "--plan", "EP-TEST-001", "--kick", "--config", cfg, "--evidence-dir", dir}, &out)
			if err == nil || !strings.Contains(err.Error(), "evidence directory") {
				t.Fatalf("destination not rejected: %v", err)
			}
			if out.Len() != 0 {
				t.Fatal("invalid destination emitted evidence")
			}
			if kind == "existing" {
				raw, err := os.ReadFile(filepath.Join(dir, "evidence.json"))
				if err != nil || string(raw) != "retained" {
					t.Fatal("previous evidence changed")
				}
			}
			if err := listener.SetDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
				t.Fatal(err)
			}
			conn, err := listener.Accept()
			if err == nil {
				conn.Close()
				t.Fatal("invalid destination probed endpoint")
			}
			if timeout, ok := err.(net.Error); !ok || !timeout.Timeout() {
				t.Fatalf("unexpected accept error: %v", err)
			}
		})
	}
}

func TestHumanContractAcceptsCleanCRLFCheckout(t *testing.T) {
	root, path := humanFixture(t)
	var c humanContract
	if err := readHumanJSON(path, &c); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	commitHumanContract(t, root)
	planTestGit(t, root, "config", "core.autocrlf", "true")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	planTestGit(t, root, "checkout", "--", path)
	checkout, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(checkout, []byte("\r\n")) {
		t.Fatal("fixture did not produce CRLF checkout")
	}
	contract, err := loadHumanContract(root, "EP-TEST-001")
	if err != nil {
		t.Fatalf("clean converted checkout rejected: %v", err)
	}
	if contract.SHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) {
		t.Fatal("digest did not use committed LF bytes")
	}
}

func TestHumanRejectsUncommittedPlanAndEmptyPrerequisites(t *testing.T) {
	for _, kind := range []string{"untracked-plan", "dirty-plan", "empty-executables", "empty-endpoints"} {
		t.Run(kind, func(t *testing.T) {
			root, path := humanFixture(t)
			if kind == "untracked-plan" {
				planTestGit(t, root, "rm", "--cached", "docs/exec-plans/active/human.md", "docs/exec-plans/active/human.ja.md")
				planTestGit(t, root, "commit", "-m", "remove plan")
			} else if kind == "dirty-plan" {
				data := strings.Replace(modelPlanFixture, "plan_type: implementation", "plan_type: human-validation\nexecution_mode: human-kick", 1)
				data = strings.Replace(data, "priority: 10", "priority: 20", 1)
				modelWritePair(t, root, "human", data)
			} else {
				var c humanContract
				if err := readHumanJSON(path, &c); err != nil {
					t.Fatal(err)
				}
				if kind == "empty-executables" {
					c.Executables = nil
				} else {
					c.Endpoints = nil
				}
				raw, _ := json.Marshal(c)
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
				commitHumanContract(t, root)
			}
			observation := filepath.Join(root, "observation.json")
			if err := os.WriteFile(observation, []byte(`{"observation":"checked","evidence_refs":["receipt"]}`), 0600); err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"preflight", "record"} {
				dir := filepath.Join(root, mode+"-bundle")
				args := []string{mode, "--plan", "EP-TEST-001", "--kick", "--evidence-dir", dir}
				if mode == "preflight" {
					args = append(args, "--config", "not-read.json")
				} else {
					args = append(args, "--scenario", "EP-TEST-001-01", "--result", "PASS", "--evidence", observation)
				}
				var out bytes.Buffer
				err := executePlanHuman(root, args, &out)
				expected := "human plan must match contract revision"
				if strings.HasPrefix(kind, "empty-") {
					expected = "nonempty executable and endpoint prerequisites"
				}
				if err == nil || !strings.Contains(err.Error(), expected) {
					t.Fatalf("expected %s rejection before effects: %v", kind, err)
				}
				if out.Len() != 0 {
					t.Fatal("invalid input emitted evidence")
				}
				if _, err := os.Stat(dir); !os.IsNotExist(err) {
					t.Fatal("invalid input reserved evidence directory")
				}
			}
		})
	}
}
