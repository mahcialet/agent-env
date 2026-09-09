package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
)

type planReplayCommit struct {
	SHA         string `json:"sha"`
	CommittedAt string `json:"committed_at"`
	Subject     string `json:"subject"`
}

type planReplayNode struct {
	ID        string   `json:"id"`
	Role      string   `json:"role"`
	State     string   `json:"state"`
	DependsOn []string `json:"depends_on"`
	Evidence  string   `json:"evidence"`
}

type planReplayStep struct {
	Name         string            `json:"name"`
	Evidence     string            `json:"evidence"`
	Hypothetical bool              `json:"hypothetical"`
	Graph        []planMetadata    `json:"graph"`
	Readiness    []planReadyResult `json:"readiness"`
}

type planReplayResult struct {
	SimulatedSteps     []planReplayStep   `json:"simulated_steps"`
	Kind               string             `json:"kind"`
	PlanPath           string             `json:"plan_path"`
	MergeSHA           string             `json:"merge_sha"`
	MergeBaseParent    string             `json:"merge_base_parent"`
	BranchHead         string             `json:"branch_head"`
	StartingRevision   string             `json:"starting_revision"`
	ArchivedAtMerge    bool               `json:"archived_at_merge"`
	ReachableFromHEAD  bool               `json:"reachable_from_head"`
	RequestedPR        int                `json:"requested_pr,omitempty"`
	PRIdentityEvidence string             `json:"pr_identity_evidence"`
	ReviewEvidence     string             `json:"review_evidence"`
	Commits            []planReplayCommit `json:"commits"`
	ProjectionNotice   string             `json:"projection_notice"`
	Nodes              []planReplayNode   `json:"projected_nodes"`
}

func executePlanReplay(root string, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("plans replay", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	planPath := flags.String("plan", "", "archived English ExecPlan path")
	mergeSHA := flags.String("merge", "", "full two-parent merge commit SHA")
	pr := flags.Int("pr", 0, "optional PR number; identity verified separately")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("plans replay: unexpected arguments")
	}
	if !validPlanMergeSHA(*mergeSHA) {
		return fmt.Errorf("plans replay: --merge requires a full commit SHA")
	}
	if !strings.HasPrefix(*planPath, "docs/exec-plans/completed/") || path.Clean(*planPath) != *planPath || strings.ContainsAny(*planPath, "\\:\x00") || !strings.HasSuffix(*planPath, ".md") || strings.HasSuffix(*planPath, ".ja.md") {
		return fmt.Errorf("plans replay: --plan requires a canonical completed English plan path")
	}
	if *pr < 0 {
		return fmt.Errorf("plans replay: PR number must be positive when supplied")
	}
	suppliedPR := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "pr" {
			suppliedPR = true
		}
	})
	if suppliedPR && *pr == 0 {
		return fmt.Errorf("plans replay: PR number must be positive when supplied")
	}
	git := func(argv ...string) (string, error) {
		cmd := exec.Command("git", argv...)
		cmd.Dir = root
		data, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("plans replay: git %s failed: %w", argv[0], err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	line, err := git("rev-list", "--parents", "-n", "1", *mergeSHA)
	if err != nil {
		return err
	}
	parents := strings.Fields(line)
	if len(parents) != 3 || parents[0] != *mergeSHA {
		return fmt.Errorf("plans replay: expected an exact two-parent merge commit")
	}
	if _, err := git("merge-base", "--is-ancestor", *mergeSHA, "HEAD"); err != nil {
		return fmt.Errorf("plans replay: merge is not reachable from current HEAD: %w", err)
	}
	document, err := git("show", *mergeSHA+":"+*planPath)
	if err != nil {
		return fmt.Errorf("plans replay: archived plan missing at merge: %w", err)
	}
	document = strings.ReplaceAll(document, "\r\n", "\n")
	// Historical Plans need not carry modern lifecycle IDs. Read their original
	// metadata and exact starting revision without migrating historical content.
	metadataEnd := strings.Index(strings.TrimPrefix(document, "---\n"), "\n---")
	if !strings.HasPrefix(document, "---\n") || metadataEnd < 0 {
		return fmt.Errorf("plans replay: archived plan lacks frontmatter")
	}
	metadata := strings.TrimPrefix(document, "---\n")[:metadataEnd]
	completed := regexp.MustCompile(`(?m)^status: completed[ \t]*$`).MatchString(metadata)
	if !completed {
		return fmt.Errorf("plans replay: archived plan is not marked completed at merge")
	}
	match := regexp.MustCompile("(?m)^Starting revision: `([0-9a-f]{40}|[0-9a-f]{64})`").FindStringSubmatch(document)
	if len(match) != 2 {
		return fmt.Errorf("plans replay: original exact starting revision is missing")
	}
	start := match[1]
	for _, parent := range parents[1:] {
		if _, err := git("merge-base", "--is-ancestor", start, parent); err != nil {
			return fmt.Errorf("plans replay: starting revision is not an ancestor of both merge parents: %w", err)
		}
	}
	log, err := git("log", "--reverse", "--format=%H%x00%cI%x00%s", start+".."+parents[2])
	if err != nil {
		return err
	}
	commits := []planReplayCommit{}
	if log != "" {
		for _, line := range strings.Split(log, "\n") {
			parts := strings.SplitN(line, "\x00", 3)
			if len(parts) != 3 || !validPlanMergeSHA(parts[0]) {
				return fmt.Errorf("plans replay: invalid Git commit record")
			}
			commits = append(commits, planReplayCommit{SHA: parts[0], CommittedAt: parts[1], Subject: parts[2]})
		}
	}
	result := planReplayResult{
		Kind: "historical-git-replay", PlanPath: *planPath, MergeSHA: *mergeSHA, MergeBaseParent: parents[1], BranchHead: parents[2], StartingRevision: start, ArchivedAtMerge: true, ReachableFromHEAD: true, RequestedPR: *pr,
		PRIdentityEvidence: "unknown: Git merge ancestry does not verify GitHub PR identity",
		ReviewEvidence:     "unknown: merge and commit messages do not prove current-HEAD review approval",
		Commits:            commits,
		ProjectionNotice:   "Derived nodes are a lifecycle projection, not historical child ExecPlans. Earlier archival is recorded as historical behavior; current completion requires merge before archive. Parent acceptance and review cannot be inferred from Git ancestry.",
		Nodes: []planReplayNode{
			{ID: "implementation", Role: "implementation", State: "completed", DependsOn: []string{}, Evidence: "branch head is the second parent of the reachable merge; completed plan exists in that merge"},
			{ID: "review", Role: "review", State: "unknown", DependsOn: []string{"implementation"}, Evidence: "structured review evidence must be supplied independently"},
			{ID: "parent", Role: "parent-finalization", State: "awaiting-reconciliation", DependsOn: []string{"implementation", "review"}, Evidence: "merged child alone does not establish parent acceptance or authorize human validation"},
		},
	}
	if *pr > 0 {
		result.PRIdentityEvidence = "unverified caller-supplied PR #" + strconv.Itoa(*pr) + "; verify GitHub head and merge SHA independently"
	}
	steps, err := simulatePlanReplay(result.ArchivedAtMerge && result.ReachableFromHEAD, result.MergeSHA)
	if err != nil {
		return err
	}
	result.SimulatedSteps = steps
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// This projection runs the actual graph validator and selection engine. Only the
// implementation merge input is backed by historical Git observations. The
// final step is an explicit counterfactual, never evidence of a historical review.
func simulatePlanReplay(implementationMerged bool, mergeSHA string) ([]planReplayStep, error) {
	implementation := planMetadata{PlanID: "EP-REPLAY-001", PlanType: "implementation", Status: "active", Parent: "EP-REPLAY-003", Priority: 1, BaseBranch: "master", MergePolicy: "guarded"}
	review := planMetadata{PlanID: "EP-REPLAY-002", PlanType: "review", Status: "active", Parent: "EP-REPLAY-003", Priority: 2, BaseBranch: "master", MergePolicy: "guarded", DependsOn: []planDependency{{PlanID: implementation.PlanID, Satisfaction: "merged"}}}
	parent := planMetadata{PlanID: "EP-REPLAY-003", PlanType: "implementation", Status: "active", Priority: 3, BaseBranch: "master", MergePolicy: "guarded", DependsOn: []planDependency{{PlanID: implementation.PlanID, Satisfaction: "merged"}, {PlanID: review.PlanID, Satisfaction: "merged"}}}
	ctx := planReadinessContext{Merged: map[string]bool{}, Stacked: map[string]bool{}}
	steps := []planReplayStep{}
	appendStep := func(name, evidence string, hypothetical bool) error {
		graph := &planGraph{Plans: []planMetadata{implementation, review, parent}, ByID: map[string]planMetadata{implementation.PlanID: implementation, review.PlanID: review, parent.PlanID: parent}}
		if err := validatePlanGraph(graph); err != nil {
			return fmt.Errorf("plans replay: projected graph: %w", err)
		}
		steps = append(steps, planReplayStep{Name: name, Evidence: evidence, Hypothetical: hypothetical, Graph: graph.Plans, Readiness: planReadiness(graph, ctx)})
		return nil
	}
	if err := appendStep("before-implementation-merge", "Hypothetical lifecycle projection before the verified historical implementation merge; not a reconstruction of historical child Plan files.", true); err != nil {
		return nil, err
	}
	if implementationMerged {
		implementation.Status = "completed"
		implementation.MergeCommit = mergeSHA
		ctx.Merged[review.PlanID+"/"+implementation.PlanID] = true
		ctx.Merged[parent.PlanID+"/"+implementation.PlanID] = true
	}
	if err := appendStep("after-verified-implementation-merge", "Implementation dependency evidence comes from the verified historical merge and archive. Review remains unknown; projected review selection is not review approval.", false); err != nil {
		return nil, err
	}
	review.Status = "completed"
	// Deliberately do not invent a review commit SHA. This context is solely a
	// counterfactual input to show parent readiness, not persisted completion.
	ctx.Merged[parent.PlanID+"/"+review.PlanID] = true
	if err := appendStep("if-independent-review-merge-were-proven", "Counterfactual only: assume independently verified review merge. Parent becomes selectable for reconciliation but remains active, not completed. No historical review proof was obtained.", true); err != nil {
		return nil, err
	}
	return steps, nil
}
