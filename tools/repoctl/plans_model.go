package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type planDependency struct {
	PlanID       string `yaml:"plan_id" json:"plan_id"`
	Satisfaction string `yaml:"satisfaction" json:"satisfaction"`
}

type planMetadata struct {
	Blockers          []string         `yaml:"blockers" json:"blockers,omitempty"`
	PlanID            string           `yaml:"plan_id" json:"plan_id"`
	PlanType          string           `yaml:"plan_type" json:"plan_type"`
	Status            string           `yaml:"status" json:"status"`
	Owner             string           `yaml:"owner" json:"owner"`
	LastVerified      string           `yaml:"last_verified" json:"last_verified"`
	Parent            string           `yaml:"parent" json:"parent,omitempty"`
	DependsOn         []planDependency `yaml:"depends_on" json:"depends_on,omitempty"`
	Priority          int              `yaml:"priority" json:"priority"`
	Workstreams       []string         `yaml:"workstreams" json:"workstreams,omitempty"`
	Conflicts         []string         `yaml:"conflicts" json:"conflicts,omitempty"`
	PauseReason       string           `yaml:"pause_reason" json:"pause_reason,omitempty"`
	ResumeWhen        string           `yaml:"resume_when" json:"resume_when,omitempty"`
	PromotionCriteria []string         `yaml:"promotion_criteria" json:"promotion_criteria,omitempty"`
	AbandonmentReason string           `yaml:"abandonment_reason" json:"abandonment_reason,omitempty"`
	MergePolicy       string           `yaml:"merge_policy" json:"merge_policy"`
	BaseBranch        string           `yaml:"base_branch" json:"base_branch"`
	Branch            string           `yaml:"branch" json:"branch,omitempty"`
	MergeCommit       string           `yaml:"merge_commit" json:"merge_commit,omitempty"`
	ExecutionMode     string           `yaml:"execution_mode" json:"execution_mode,omitempty"`
	Path              string           `yaml:"-" json:"path"`
}

type planGraph struct {
	Plans []planMetadata          `json:"plans"`
	ByID  map[string]planMetadata `json:"-"`
}

var planIDPattern = regexp.MustCompile(`^EP-[A-Z][A-Z0-9]*-[0-9]{3,}$`)
var planCommitPattern = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64})$`)
var legacyCompletedPlans = map[string]bool{
	"agent-env-mvp.md":                          true,
	"android-emulator-lease.md":                 true,
	"android-emulator-review-2.md":              true,
	"android-emulator-review.md":                true,
	"android-ui-observer.md":                    true,
	"bilingual-documentation-review.md":         true,
	"bilingual-documentation.md":                true,
	"browser-cdp-automation.md":                 true,
	"compose-provider-podman-review.md":         true,
	"compose-provider-podman.md":                true,
	"flutter-android-review.md":                 true,
	"flutter-android-runtime.md":                true,
	"multi-host-control-plane.md":               true,
	"multi-host-review-followup.md":             true,
	"multi-host-review-round-two.md":            true,
	"persistent-process-runtime.md":             true,
	"process-destroy-preview-review.md":         true,
	"reader-first-documentation-restructure.md": true,
	"repository-correctness-audit.md":           true,
	"repository-correctness-review.md":          true,
	"standalone-distribution-review.md":         true,
	"standalone-distribution.md":                true,
	"standalone-release-filter-review.md":       true,
	"standalone-release-finalization.md":        true,
	"standalone-release-review.md":              true,
	"standalone-verify-review.md":               true,
	"worker-android-capacity-review.md":         true,
}

var planStates = []string{"draft", "active", "paused", "completed", "abandoned"}

// parsePlanMetadata accepts unrelated document metadata, but lifecycle fields
// have exact YAML types. Historical completed documents without IDs are exempt.
func parsePlanMetadata(data []byte, path string) (planMetadata, bool, error) {
	var p planMetadata
	fail := func(s string) (planMetadata, bool, error) { return p, false, fmt.Errorf("%s: %s", path, s) }
	normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return fail("missing YAML frontmatter")
	}
	end := 1
	for end < len(lines) && lines[end] != "---" {
		end++
	}
	if end == len(lines) {
		return fail("unterminated YAML frontmatter")
	}
	var doc yaml.Node
	decoder := yaml.NewDecoder(strings.NewReader(strings.Join(lines[1:end], "\n")))
	if err := decoder.Decode(&doc); err != nil {
		return fail(err.Error())
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return fail("frontmatter must contain one YAML document")
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return fail("frontmatter must be a mapping")
	}
	node := doc.Content[0]
	fields := map[string]*yaml.Node{}
	for i := 0; i < len(node.Content); i += 2 {
		k, v := node.Content[i], node.Content[i+1]
		if k.Tag != "!!str" {
			return fail("metadata keys must be strings")
		}
		if _, ok := fields[k.Value]; ok {
			return fail("duplicate metadata key " + k.Value)
		}
		fields[k.Value] = v
	}
	state := filepath.Base(filepath.Dir(path))
	if fields["plan_id"] == nil && state == "completed" && legacyCompletedPlans[strings.TrimSuffix(filepath.Base(path), ".ja.md")+".md"] {
		return p, true, nil
	}
	if fields["plan_id"] == nil && state == "completed" && legacyCompletedPlans[filepath.Base(path)] {
		return p, true, nil
	}
	allowed := map[string]bool{}
	for _, key := range []string{"plan_id", "plan_type", "status", "owner", "last_verified", "parent", "depends_on", "priority", "workstreams", "conflicts", "pause_reason", "resume_when", "promotion_criteria", "abandonment_reason", "merge_policy", "base_branch", "branch", "merge_commit", "execution_mode", "translation_of", "source_sha256", "blockers"} {
		allowed[key] = true
	}
	for key := range fields {
		if !allowed[key] {
			return fail("unknown lifecycle metadata field " + key)
		}
	}
	scalarFields := []string{"plan_id", "plan_type", "status", "owner", "parent", "pause_reason", "resume_when", "abandonment_reason", "merge_policy", "base_branch", "branch", "merge_commit", "execution_mode"}
	for _, key := range scalarFields {
		if n := fields[key]; n != nil && (n.Kind != yaml.ScalarNode || n.Tag != "!!str") {
			return fail(key + " must be a string")
		}
	}
	if n := fields["last_verified"]; n != nil && (n.Kind != yaml.ScalarNode || (n.Tag != "!!str" && n.Tag != "!!timestamp")) {
		return fail("last_verified must be a date")
	}
	for _, key := range []string{"workstreams", "conflicts", "promotion_criteria", "blockers"} {
		if n := fields[key]; n != nil {
			if n.Kind != yaml.SequenceNode {
				return fail(key + " must be a string list")
			}
			seen := map[string]bool{}
			for _, v := range n.Content {
				if v.Kind != yaml.ScalarNode || v.Tag != "!!str" || strings.TrimSpace(v.Value) == "" || seen[v.Value] {
					return fail(key + " requires unique nonempty strings")
				}
				seen[v.Value] = true
			}
		}
	}
	if n := fields["priority"]; n != nil && (n.Kind != yaml.ScalarNode || n.Tag != "!!int") {
		return fail("priority must be an integer")
	}
	if n := fields["depends_on"]; n != nil {
		if n.Kind != yaml.SequenceNode {
			return fail("depends_on must be a list")
		}
		for _, d := range n.Content {
			if d.Kind != yaml.MappingNode {
				return fail("dependency must be a mapping")
			}
			for j := 0; j < len(d.Content); j += 2 {
				key, v := d.Content[j].Value, d.Content[j+1]
				if key != "plan_id" && key != "satisfaction" {
					return fail("unknown dependency field " + key)
				}
				if v.Kind != yaml.ScalarNode || v.Tag != "!!str" {
					return fail("dependency fields must be strings")
				}
			}
		}
	}
	if err := node.Decode(&p); err != nil {
		return fail(err.Error())
	}
	p.Path = filepath.ToSlash(path)
	if !planIDPattern.MatchString(p.PlanID) {
		return fail("invalid or missing plan_id")
	}
	if !planContains(planStates, p.Status) || p.Status != state {
		return fail("status must match lifecycle directory")
	}
	if !planContains([]string{"implementation", "review", "human-validation"}, p.PlanType) {
		return fail("invalid or missing plan_type")
	}
	if strings.TrimSpace(p.Owner) == "" {
		return fail("owner is required")
	}
	if _, err := time.Parse("2006-01-02", p.LastVerified); err != nil {
		return fail("last_verified must be YYYY-MM-DD")
	}
	if fields["priority"] == nil || p.Priority < 0 {
		return fail("priority must be a nonnegative integer")
	}
	if !planContains([]string{"automatic", "guarded", "manual"}, p.MergePolicy) {
		return fail("invalid or missing merge_policy")
	}
	if !validPlanBaseBranch(p.BaseBranch) {
		return fail("base_branch is required and must be a Git branch name")
	}
	if p.Parent != "" && !planIDPattern.MatchString(p.Parent) {
		return fail("invalid parent ID")
	}
	seen := map[string]bool{}
	for i := range p.DependsOn {
		d := &p.DependsOn[i]
		if !planIDPattern.MatchString(d.PlanID) || seen[d.PlanID] {
			return fail("invalid or duplicate dependency ID")
		}
		seen[d.PlanID] = true
		if d.Satisfaction == "" {
			d.Satisfaction = "merged"
		}
		if d.Satisfaction != "merged" && d.Satisfaction != "stacked" {
			return fail("invalid dependency satisfaction")
		}
	}
	if p.Status == "paused" && (strings.TrimSpace(p.PauseReason) == "" || strings.TrimSpace(p.ResumeWhen) == "") {
		return fail("paused requires pause_reason and resume_when")
	}
	if p.Status == "draft" && len(p.PromotionCriteria) == 0 {
		return fail("draft requires promotion_criteria")
	}
	if p.Status == "abandoned" && strings.TrimSpace(p.AbandonmentReason) == "" {
		return fail("abandoned requires abandonment_reason")
	}
	if p.Status == "completed" && !planCommitPattern.MatchString(p.MergeCommit) {
		return fail("completed requires full merge_commit SHA")
	}
	if p.PlanType == "human-validation" && p.ExecutionMode != "human-kick" {
		return fail("human-validation requires execution_mode: human-kick")
	}
	if p.PlanType != "human-validation" && p.ExecutionMode != "" {
		return fail("execution_mode only applies to human-validation")
	}
	return p, false, nil
}

func planContains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func loadPlanGraph(root string) (*planGraph, error) {
	g := &planGraph{ByID: map[string]planMetadata{}}
	for _, state := range planStates {
		dir := filepath.Join(root, "docs", "exec-plans", state)
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			if strings.HasSuffix(entry.Name(), ".ja.md") {
				if _, err := os.Stat(strings.TrimSuffix(path, ".ja.md") + ".md"); err != nil {
					return nil, fmt.Errorf("orphan Japanese plan %s", path)
				}
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			rel, _ := filepath.Rel(root, path)
			p, legacy, err := parsePlanMetadata(data, rel)
			if err != nil {
				return nil, err
			}
			if legacy {
				continue
			}
			jaPath := strings.TrimSuffix(path, ".md") + ".ja.md"
			ja, err := os.ReadFile(jaPath)
			if err != nil {
				return nil, fmt.Errorf("%s: Japanese pair: %w", rel, err)
			}
			jaRel, _ := filepath.Rel(root, jaPath)
			jp, legacy, err := parsePlanMetadata(ja, jaRel)
			if err != nil {
				return nil, err
			}
			jp.Path = p.Path
			if legacy || !reflect.DeepEqual(p, jp) {
				return nil, fmt.Errorf("%s: EN/JA lifecycle metadata differ", rel)
			}
			if old, ok := g.ByID[p.PlanID]; ok {
				return nil, fmt.Errorf("duplicate plan_id %s: %s and %s", p.PlanID, old.Path, p.Path)
			}
			g.Plans = append(g.Plans, p)
			g.ByID[p.PlanID] = p
		}
	}
	sort.Slice(g.Plans, func(i, j int) bool { return g.Plans[i].PlanID < g.Plans[j].PlanID })
	if err := validatePlanGraph(g); err != nil {
		return nil, err
	}
	return g, nil
}

func validatePlanGraph(g *planGraph) error {
	for _, relation := range []string{"parent", "dependency"} {
		marks := map[string]int{}
		var visit func(string) error
		visit = func(id string) error {
			if marks[id] == 1 {
				return fmt.Errorf("%s cycle at %s", relation, id)
			}
			if marks[id] == 2 {
				return nil
			}
			p, ok := g.ByID[id]
			if !ok {
				return fmt.Errorf("missing %s plan %s", relation, id)
			}
			marks[id] = 1
			edges := []string{}
			if relation == "parent" {
				if p.Parent != "" {
					edges = append(edges, p.Parent)
				}
			} else {
				for _, d := range p.DependsOn {
					edges = append(edges, d.PlanID)
				}
			}
			for _, edge := range edges {
				if err := visit(edge); err != nil {
					return err
				}
			}
			marks[id] = 2
			return nil
		}
		for _, p := range g.Plans {
			if err := visit(p.PlanID); err != nil {
				return err
			}
		}
	}
	return nil
}

type planReadinessContext struct {
	// Evidence must be computed against each consumer's configured base by caller.
	Merged  map[string]bool
	Stacked map[string]bool
	Running []string
}

type planReadyResult struct {
	PlanID   string   `json:"plan_id"`
	Runnable bool     `json:"runnable"`
	Selected bool     `json:"selected"`
	Reasons  []string `json:"reasons"`
}

func planReadiness(g *planGraph, ctx planReadinessContext) []planReadyResult {
	plans := append([]planMetadata(nil), g.Plans...)
	sort.Slice(plans, func(i, j int) bool {
		if plans[i].Priority != plans[j].Priority {
			return plans[i].Priority < plans[j].Priority
		}
		return plans[i].PlanID < plans[j].PlanID
	})
	results := make([]planReadyResult, 0, len(plans))
	selected := false
	for _, p := range plans {
		r := planReadyResult{PlanID: p.PlanID, Reasons: []string{}}
		if p.Status != "active" {
			r.Reasons = append(r.Reasons, "status: "+p.Status)
		}
		for _, blocker := range p.Blockers {
			r.Reasons = append(r.Reasons, "blocker: "+blocker)
		}
		if p.PlanType == "human-validation" {
			r.Reasons = append(r.Reasons, "explicit human kick required")
		}
		for _, d := range p.DependsOn {
			dep, ok := g.ByID[d.PlanID]
			key := p.PlanID + "/" + d.PlanID
			if !ok || (d.Satisfaction != "stacked" && (dep.Status != "completed" || !ctx.Merged[key])) || (d.Satisfaction == "stacked" && ((dep.Status != "active" && dep.Status != "completed") || !ctx.Stacked[key])) {
				r.Reasons = append(r.Reasons, "unsatisfied dependency: "+d.PlanID)
			}
		}
		for _, id := range ctx.Running {
			running, ok := g.ByID[id]
			if !ok {
				r.Reasons = append(r.Reasons, "unknown running plan: "+id)
				continue
			}
			if planWorkstreamConflict(p, running) {
				r.Reasons = append(r.Reasons, "workstream conflict: "+id)
			}
		}
		if len(ctx.Running) > 0 {
			r.Reasons = append(r.Reasons, "implementation concurrency limit reached")
		}
		r.Runnable = len(r.Reasons) == 0
		if r.Runnable && !selected {
			r.Selected = true
			selected = true
		}
		results = append(results, r)
	}
	return results
}

func planWorkstreamConflict(a, b planMetadata) bool {
	for _, w := range a.Workstreams {
		if planContains(b.Workstreams, w) || planContains(b.Conflicts, w) {
			return true
		}
	}
	for _, w := range b.Workstreams {
		if planContains(a.Conflicts, w) {
			return true
		}
	}
	return false
}
