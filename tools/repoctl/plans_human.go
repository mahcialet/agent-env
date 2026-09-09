package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type humanScenario struct {
	ID           string   `json:"id"`
	Purpose      string   `json:"purpose"`
	Actions      []string `json:"actions"`
	Expected     string   `json:"expected"`
	PassCriteria string   `json:"pass_criteria"`
}
type humanContract struct {
	Revision    string          `json:"-"`
	SHA256      string          `json:"-"`
	PlanID      string          `json:"plan_id"`
	Executables []string        `json:"executables"`
	Endpoints   []string        `json:"endpoints"`
	Scenarios   []humanScenario `json:"scenarios"`
}
type humanConfig struct {
	Endpoints map[string]string `json:"endpoints"`
}
type humanObservation struct {
	Observation  string   `json:"observation"`
	EvidenceRefs []string `json:"evidence_refs"`
}
type humanEvidence struct {
	ContractRevision string   `json:"contract_revision"`
	ContractSHA256   string   `json:"contract_sha256"`
	PlanID           string   `json:"plan_id"`
	Kind             string   `json:"kind"`
	Result           string   `json:"result"`
	Scenario         string   `json:"scenario,omitempty"`
	Timestamp        string   `json:"timestamp"`
	Blockers         []string `json:"blockers,omitempty"`
	EvidenceSHA256   string   `json:"evidence_sha256,omitempty"`
	ActionRequired   string   `json:"action_required,omitempty"`
	FollowUpPlan     string   `json:"follow_up_plan,omitempty"`
}

const humanJSONLimit = 1024 * 1024

func readHumanJSON(path string, target any) error {
	_, err := readHumanJSONSnapshot(path, target)
	return err
}

// Validation and evidence identity use one bounded snapshot, never a second read.
func readHumanJSONSnapshot(path string, target any) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open JSON input")
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, humanJSONLimit+1))
	if err != nil {
		return nil, fmt.Errorf("cannot read JSON input")
	}
	return decodeHumanJSONSnapshot(raw, target)
}

func decodeHumanJSONSnapshot(raw []byte, target any) ([]byte, error) {
	if len(raw) > humanJSONLimit {
		return nil, fmt.Errorf("JSON input exceeds size limit")
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("JSON input must contain exactly one object")
	}
	if err := validateHumanJSONKeys(json.NewDecoder(bytes.NewReader(raw))); err != nil {
		return nil, err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return nil, fmt.Errorf("invalid JSON input")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return nil, fmt.Errorf("JSON input must contain exactly one object")
	}
	return raw, nil
}

// Check every object, including objects inside arrays, before decoding can merge
// repeated members or replace an earlier value. Do not expose input keys in errors.
func validateHumanJSONKeys(d *json.Decoder) error {
	token, err := d.Token()
	if err != nil {
		return fmt.Errorf("invalid JSON input")
	}
	delim, container := token.(json.Delim)
	if !container {
		return nil
	}
	seen := map[string]bool{}
	for d.More() {
		if delim == '{' {
			key, err := d.Token()
			if err != nil {
				return fmt.Errorf("invalid JSON input")
			}
			name, ok := key.(string)
			if !ok {
				return fmt.Errorf("invalid JSON input")
			}
			if seen[name] {
				return fmt.Errorf("duplicate JSON object key")
			}
			seen[name] = true
		}
		if err := validateHumanJSONKeys(d); err != nil {
			return err
		}
	}
	if _, err := d.Token(); err != nil {
		return fmt.Errorf("invalid JSON input")
	}
	return nil
}

func loadHumanContract(root, id string) (humanContract, error) {
	var c humanContract
	if !planIDPattern.MatchString(id) {
		return c, fmt.Errorf("invalid plan ID")
	}
	path := "docs/exec-plans/validation/" + id + ".json"
	revision, err := planGit(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return c, fmt.Errorf("human contract requires a committed HEAD")
	}
	blob, err := planGit(root, "rev-parse", "--verify", revision+":"+path)
	if err != nil {
		return c, fmt.Errorf("human contract must be tracked at HEAD")
	}
	sizeText, err := planGit(root, "cat-file", "-s", blob)
	size, parseErr := strconv.Atoi(sizeText)
	if err != nil || parseErr != nil || size > humanJSONLimit {
		return c, fmt.Errorf("invalid committed human contract size")
	}
	cmd := exec.Command("git", "-C", root, "cat-file", "blob", blob)
	raw, err := cmd.Output()
	if err != nil {
		return c, fmt.Errorf("cannot read committed human contract")
	}
	if _, err := decodeHumanJSONSnapshot(raw, &c); err != nil {
		return c, err
	}
	status, err := planGit(root, "status", "--porcelain", "--untracked-files=all", "--", path)
	if err != nil || status != "" {
		return c, fmt.Errorf("human contract must match committed HEAD; dirty contract rejected")
	}
	var working humanContract
	_, err = readHumanJSONSnapshot(filepath.Join(root, filepath.FromSlash(path)), &working)
	// Git may convert checkout line endings; compare the strictly decoded contract
	// while retaining committed bytes as the evidence digest authority.
	workingCanonical, workingErr := json.Marshal(working)
	committedCanonical, committedErr := json.Marshal(c)
	if err != nil || workingErr != nil || committedErr != nil || !bytes.Equal(workingCanonical, committedCanonical) {
		return c, fmt.Errorf("human contract must match committed HEAD")
	}
	sum := sha256.Sum256(raw)
	c.Revision = revision
	c.SHA256 = hex.EncodeToString(sum[:])
	if c.PlanID != id || len(c.Scenarios) == 0 {
		return c, fmt.Errorf("invalid human contract identity or scenarios")
	}
	seen := map[string]bool{}
	for _, s := range c.Scenarios {
		if !strings.HasPrefix(s.ID, id+"-") || seen[s.ID] || strings.TrimSpace(s.Purpose) == "" || len(s.Actions) == 0 || strings.TrimSpace(s.Expected) == "" || strings.TrimSpace(s.PassCriteria) == "" {
			return c, fmt.Errorf("invalid or duplicate scenario")
		}
		seen[s.ID] = true
		for _, a := range s.Actions {
			if strings.TrimSpace(a) == "" {
				return c, fmt.Errorf("empty scenario action")
			}
		}
	}
	for _, list := range [][]string{c.Executables, c.Endpoints} {
		if len(list) == 0 {
			return c, fmt.Errorf("human contract requires nonempty executable and endpoint prerequisites")
		}
		seen = map[string]bool{}
		for _, v := range list {
			if strings.TrimSpace(v) == "" || seen[v] {
				return c, fmt.Errorf("invalid prerequisite")
			}
			seen[v] = true
		}
	}
	return c, nil
}

// The kick authorizes only this invocation. Preflight never executes scenarios.
func executePlanHuman(root string, args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: plans human preflight|record --plan ID --kick --evidence-dir DIR")
	}
	mode := args[0]
	if mode != "preflight" && mode != "record" {
		return fmt.Errorf("unknown human command")
	}
	f := flag.NewFlagSet("plans human", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	id := f.String("plan", "", "Plan ID")
	kick := f.Bool("kick", false, "explicit human kick")
	dir := f.String("evidence-dir", "", "new evidence bundle directory")
	config := f.String("config", "", "endpoint config JSON")
	scenario := f.String("scenario", "", "scenario ID")
	result := f.String("result", "", "PASS/FINDING/BLOCKED")
	evidence := f.String("evidence", "", "observation JSON")
	followup := f.String("follow-up-plan", "", "tracked finding Plan ID")
	if err := f.Parse(args[1:]); err != nil || f.NArg() != 0 {
		return fmt.Errorf("invalid human command arguments")
	}
	if !*kick {
		return fmt.Errorf("explicit --kick required; no preflight or scenario was run")
	}
	if *dir == "" {
		return fmt.Errorf("explicit --evidence-dir required")
	}
	g, err := loadPlanGraph(root)
	if err != nil {
		return err
	}
	p, ok := g.ByID[*id]
	if !ok || p.PlanType != "human-validation" || p.ExecutionMode != "human-kick" || (p.Status != "active" && p.Status != "paused") {
		return fmt.Errorf("plan must be active or paused human-validation with human-kick")
	}
	c, err := loadHumanContract(root, *id)
	if err != nil {
		return err
	}
	if err := planMetadataMatchesRevision(root, c.Revision, p); err != nil {
		return fmt.Errorf("human plan must match contract revision: %w", err)
	}
	e := humanEvidence{PlanID: *id, ContractRevision: c.Revision, ContractSHA256: c.SHA256, Kind: mode, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	if mode == "preflight" {
		if *config == "" {
			return fmt.Errorf("explicit --config required")
		}
		var cfg humanConfig
		if err := readHumanJSON(*config, &cfg); err != nil {
			return err
		}
		for name := range cfg.Endpoints {
			if !planContains(c.Endpoints, name) {
				return fmt.Errorf("config contains an unapproved endpoint name")
			}
		}
		if err := reserveHumanEvidence(*dir); err != nil {
			return err
		}
		for _, name := range c.Executables {
			if _, err := exec.LookPath(name); err != nil {
				e.Blockers = append(e.Blockers, "missing executable: "+name)
			}
		}
		for _, name := range c.Endpoints {
			address := cfg.Endpoints[name]
			if _, _, err := net.SplitHostPort(address); err != nil {
				e.Blockers = append(e.Blockers, "missing or invalid endpoint: "+name)
				continue
			}
			conn, err := net.DialTimeout("tcp", address, 2*time.Second)
			if err != nil {
				e.Blockers = append(e.Blockers, "unreachable endpoint: "+name)
			} else {
				conn.Close()
			}
		}
		e.Result = "READY"
		if len(e.Blockers) > 0 {
			e.Result = "BLOCKED"
			e.ActionRequired = "paused-plan-update"
			e.FollowUpPlan = *id
		}
	} else {
		found := false
		for _, s := range c.Scenarios {
			if s.ID == *scenario {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("unknown scenario ID")
		}
		if !planContains([]string{"PASS", "FINDING", "BLOCKED"}, *result) {
			return fmt.Errorf("result must be PASS, FINDING or BLOCKED")
		}
		var observation humanObservation
		if *evidence == "" {
			return fmt.Errorf("observation evidence required")
		}
		raw, err := readHumanJSONSnapshot(*evidence, &observation)
		if err != nil {
			return err
		}
		if strings.TrimSpace(observation.Observation) == "" || len(observation.EvidenceRefs) == 0 {
			return fmt.Errorf("observation and evidence_refs are required")
		}
		for _, ref := range observation.EvidenceRefs {
			if strings.TrimSpace(ref) == "" {
				return fmt.Errorf("empty evidence reference")
			}
		}
		sum := sha256.Sum256(raw)
		e.EvidenceSHA256 = hex.EncodeToString(sum[:])
		e.Scenario = *scenario
		e.Result = *result
		if *result == "FINDING" {
			target, ok := g.ByID[*followup]
			if !ok || target.PlanType != "review" || (target.Status != "draft" && target.Status != "active") {
				return fmt.Errorf("FINDING requires --follow-up-plan naming a tracked draft or active review plan")
			}
			e.ActionRequired = "review-plan-update"
			e.FollowUpPlan = *followup
		}
		if *result == "BLOCKED" {
			e.ActionRequired = "paused-plan-update"
			e.FollowUpPlan = *id
		}
		if err := reserveHumanEvidence(*dir); err != nil {
			return err
		}
	}
	if err := writeReservedHumanEvidence(*dir, e); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s: %s (preflight is not scenario acceptance; recorded results are operator attestations)\n", e.PlanID, e.Result)
	if e.Result == "BLOCKED" {
		return fmt.Errorf("human validation blocked; see evidence bundle")
	}
	return nil
}
func reserveHumanEvidence(dir string) error {
	// Require a fresh directory: previous evidence is immutable and never overwritten.
	if err := os.Mkdir(dir, 0700); err != nil {
		return fmt.Errorf("evidence directory must be new with an existing parent")
	}
	return nil
}

func writeHumanEvidence(dir string, e humanEvidence) error {
	if err := reserveHumanEvidence(dir); err != nil {
		return err
	}
	return writeReservedHumanEvidence(dir, e)
}

func writeReservedHumanEvidence(dir string, e humanEvidence) error {
	raw, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(dir, "evidence.json"), append(raw, '\n'), 0600); err != nil {
		return err
	}
	text := fmt.Sprintf("# Human validation evidence\n\nPlan: `%s`\n\nKind: %s\n\nResult: %s\n\nPreflight does not prove scenario acceptance. Scenario results are explicit operator attestations; the evidence digest identifies the separately retained observation.\n", e.PlanID, e.Kind, e.Result)
	text += "\nContract revision: `" + e.ContractRevision + "`\n\nContract SHA-256: `" + e.ContractSHA256 + "`\n"
	if e.Scenario != "" {
		text += "\nScenario: `" + e.Scenario + "`\n\nObservation SHA-256: `" + e.EvidenceSHA256 + "`\n"
	}
	for _, b := range e.Blockers {
		text += "\n- " + b + "\n"
	}
	if e.ActionRequired != "" {
		text += "\nRequired follow-up: " + e.ActionRequired + " (`" + e.FollowUpPlan + "`). Update the tracked Plan; this command does not change its state.\n"
	}
	return os.WriteFile(filepath.Join(dir, "evidence.md"), []byte(text), 0600)
}
