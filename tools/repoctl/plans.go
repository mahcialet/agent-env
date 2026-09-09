package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Lifecycle plans alone admit structured metadata. All other documentation keeps
// the strict existing single-line scalar format and its negative fixtures.
func documentationTranslationMetadata(path string, data []byte) (map[string]string, error) {
	if !strings.HasPrefix(path, "docs/exec-plans/") {
		return translationMetadata(data)
	}
	_, legacy, err := parsePlanMetadata(data, path)
	if err != nil {
		return nil, err
	}
	if legacy {
		return translationMetadata(data)
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	end := 1
	for end < len(lines) && lines[end] != "---" {
		end++
	}
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &node); err != nil {
		return nil, err
	}
	fields := map[string]string{}
	for i := 0; i < len(node.Content[0].Content); i += 2 {
		k, v := node.Content[0].Content[i], node.Content[0].Content[i+1]
		if v.Kind == yaml.ScalarNode {
			fields[k.Value] = v.Value
		}
	}
	return fields, nil
}

func planGit(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func expectedPlanBranch(p planMetadata) string {
	if p.PlanID == "EP-OPS-001" {
		return "feat/execplan-lifecycle-orchestration"
	}
	prefix := "feat/"
	if p.PlanType == "review" {
		prefix = "fix/"
	}
	if p.PlanType == "human-validation" {
		prefix = "validate/"
	}
	return prefix + strings.ToLower(p.PlanID)
}

func planAncestor(root, ancestor, descendant string) bool {
	if !planCommitPattern.MatchString(ancestor) || !planCommitPattern.MatchString(descendant) {
		return false
	}
	_, err := planGit(root, "merge-base", "--is-ancestor", ancestor, descendant)
	return err == nil
}

func planRevision(root, ref string) (string, error) {
	sha, err := planGit(root, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	if err != nil && ref == "master" {
		return planGit(root, "rev-parse", "--verify", "refs/remotes/origin/master^{commit}")
	}
	return sha, err
}

// Readiness is derived from local Git observations, never an assertion in the
// dependency's prose. Fetching remote refs remains an explicit operator action.
func planGitReadiness(root string, g *planGraph) (planReadinessContext, error) {
	ctx := planReadinessContext{Merged: map[string]bool{}, Stacked: map[string]bool{}}
	worktrees, err := planGit(root, "worktree", "list", "--porcelain")
	if err != nil {
		return ctx, err
	}
	for _, p := range g.Plans {
		if p.Branch != "" && p.Branch != expectedPlanBranch(p) {
			return ctx, fmt.Errorf("%s: branch must be %s", p.PlanID, expectedPlanBranch(p))
		}
		if p.Status == "active" && strings.Contains("\n"+worktrees+"\n", "\nbranch refs/heads/"+expectedPlanBranch(p)+"\n") {
			ctx.Running = append(ctx.Running, p.PlanID)
		}
		for _, d := range p.DependsOn {
			dep := g.ByID[d.PlanID]
			key := p.PlanID + "/" + dep.PlanID
			base, err := planRevision(root, p.BaseBranch)
			if err != nil {
				continue
			}
			if d.Satisfaction == "merged" {
				if dep.Status != "completed" || !planAncestor(root, dep.MergeCommit, base) {
					continue
				}
				// The merged commit must contain the same identity, not just be an arbitrary
				// unrelated commit that happens to be reachable from base.
				ctx.Merged[key] = planMergeProof(root, dep, base) == nil
			} else if dep.Status == "active" || dep.Status == "completed" {
				head, err := planStackedHead(root, dep)
				if err == nil {
					consumer, consumerErr := planRevision(root, expectedPlanBranch(p))
					if consumerErr != nil {
						// Before creation, the declared base is the future branch start point.
						// An existing but invalid ref must not silently fall back.
						_, refErr := planGit(root, "show-ref", "--verify", "refs/heads/"+expectedPlanBranch(p))
						if refErr == nil {
							continue
						}
						consumer = base
					}
					ctx.Stacked[key] = planAncestor(root, head, consumer)
				}
			}
		}
	}
	return ctx, nil
}

func planIdentityAt(root, revision, id string) (bool, error) {
	paths, err := planGit(root, "ls-tree", "-r", "--name-only", revision, "--", "docs/exec-plans")
	if err != nil {
		return false, err
	}
	count := 0
	for _, path := range strings.Split(paths, "\n") {
		if !strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".ja.md") {
			continue
		}
		data, err := planGit(root, "show", revision+":"+path)
		if err != nil {
			return false, err
		}
		p, legacy, err := parsePlanMetadata([]byte(data), path)
		if err != nil {
			return false, err
		}
		if !legacy && p.PlanID == id {
			count++
		}
	}
	return count == 1, nil
}

func planProvenance(root string, p planMetadata, prBody string) error {
	branch, err := planGit(root, "branch", "--show-current")
	if err != nil {
		return err
	}
	if branch != expectedPlanBranch(p) {
		return fmt.Errorf("%s: expected branch %s, got %s", p.PlanID, expectedPlanBranch(p), branch)
	}
	if p.Branch != "" && p.Branch != branch {
		return errors.New("declared branch differs from deterministic branch")
	}
	g := &planGraph{ByID: map[string]planMetadata{}}
	if len(p.DependsOn) > 0 {
		g, err = loadPlanGraph(root)
		if err != nil {
			return err
		}
	}
	head, err := planRevision(root, "HEAD")
	if err != nil {
		return err
	}
	if err := planCommitProvenance(root, p, head, g, map[string]bool{}, true); err != nil {
		return err
	}
	if prBody != "" {
		count := 0
		for _, line := range strings.Split(documentProse(prBody), "\n") {
			if strings.HasPrefix(line, "ExecPlan:") {
				if strings.TrimSpace(strings.TrimPrefix(line, "ExecPlan:")) != p.PlanID {
					return errors.New("PR Plan-ID mismatch")
				}
				count++
			}
		}
		if count != 1 {
			return errors.New("PR requires one visible standalone ExecPlan: ID line")
		}
	}
	return nil
}

// Completed dependencies use immutable merge evidence, even after branch deletion.
// A squash proves only its resulting commit; it cannot prove ancestry of the old tip.
func planStackedHead(root string, p planMetadata) (string, error) {
	if p.Status == "active" {
		return planRevision(root, expectedPlanBranch(p))
	}
	if p.Status != "completed" {
		return "", errors.New("stacked dependency is not active or completed")
	}
	base, err := planRevision(root, p.BaseBranch)
	if err != nil {
		return "", err
	}
	if err := planMergeProof(root, p, base); err != nil {
		return "", err
	}
	parents, err := planGit(root, "show", "-s", "--format=%P", p.MergeCommit)
	if err != nil {
		return "", err
	}
	if list := strings.Fields(parents); len(list) == 2 {
		return list[1], nil
	}
	return p.MergeCommit, nil
}

func planCommitProvenance(root string, p planMetadata, head string, g *planGraph, visiting map[string]bool, requireOwn bool) error {
	if visiting[p.PlanID] {
		return errors.New("cyclic provenance dependency")
	}
	visiting[p.PlanID] = true
	defer delete(visiting, p.PlanID)
	base, err := planRevision(root, p.BaseBranch)
	if err != nil {
		return err
	}
	args := []string{"rev-list", "--no-merges", head, "^" + base}
	for _, d := range p.DependsOn {
		if d.Satisfaction != "stacked" {
			continue
		}
		dep, ok := g.ByID[d.PlanID]
		if !ok {
			return fmt.Errorf("unknown dependency %s", d.PlanID)
		}
		tip, err := planStackedHead(root, dep)
		if err != nil {
			return err
		}
		if !planAncestor(root, tip, head) {
			return fmt.Errorf("%s: stacked dependency is not inherited", d.PlanID)
		}
		ok, err = planIdentityAt(root, tip, dep.PlanID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%s: inherited history lacks Plan identity", dep.PlanID)
		}
		if dep.Status == "active" {
			if err := planCommitProvenance(root, dep, tip, g, visiting, false); err != nil {
				return err
			}
		}
		args = append(args, "^"+tip)
	}
	commits, err := planGit(root, args...)
	if err != nil {
		return err
	}
	if commits == "" {
		if requireOwn {
			return errors.New("no implementation commits beyond base and dependencies")
		}
		return nil
	}
	for _, sha := range strings.Split(commits, "\n") {
		body, err := planGit(root, "show", "-s", "--format=%B", sha)
		if err != nil {
			return err
		}
		cmd := exec.Command("git", "interpret-trailers", "--parse")
		cmd.Dir = root
		cmd.Stdin = strings.NewReader(body)
		b, err := cmd.Output()
		if err != nil {
			return err
		}
		found := 0
		for _, line := range strings.Split(string(b), "\n") {
			key, value, ok := strings.Cut(line, ":")
			if ok && strings.EqualFold(strings.TrimSpace(key), "ExecPlan") {
				if strings.TrimSpace(value) != p.PlanID {
					return fmt.Errorf("%s: conflicting ExecPlan trailer", sha)
				}
				found++
			}
		}
		if found != 1 {
			return fmt.Errorf("%s: requires exactly one ExecPlan: %s trailer", sha, p.PlanID)
		}
	}
	return nil
}

func writePlanJSON(out io.Writer, v any) error {
	e := json.NewEncoder(out)
	e.SetIndent("", "  ")
	return e.Encode(v)
}

func executePlans(root string, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("use repoctl plans list|check|graph|ready|provenance|gate|replay")
	}
	if args[0] == "replay" {
		return executePlanReplay(root, args[1:], out)
	}
	if args[0] == "gate" {
		return executePlanGate(root, args[1:], out)
	}
	if args[0] == "human" {
		return executePlanHuman(root, args[1:], out)
	}
	g, err := loadPlanGraph(root)
	if err != nil {
		return err
	}
	switch args[0] {
	case "list", "graph":
		if len(args) != 1 {
			return errors.New("command takes no arguments")
		}
		return writePlanJSON(out, g)
	case "check":
		if len(args) != 1 {
			return errors.New("command takes no arguments")
		}
		if err := docsCheck(root); err != nil {
			return err
		}
		if err := planIdentityHistory(root, g); err != nil {
			return err
		}
		for _, p := range g.Plans {
			if p.Branch != "" && p.Branch != expectedPlanBranch(p) {
				return fmt.Errorf("%s branch mismatch", p.PlanID)
			}
			if p.Status == "completed" {
				base, err := planRevision(root, p.BaseBranch)
				if err != nil {
					return err
				}
				if planMergeProof(root, p, base) != nil {
					return fmt.Errorf("%s completion is not proven merged into %s", p.PlanID, p.BaseBranch)
				}
			}
		}
		return writePlanJSON(out, map[string]any{"valid": true, "plans": len(g.Plans)})
	case "ready":
		if len(args) != 1 {
			return errors.New("ready takes no arguments; running plans derive from Git worktrees")
		}
		ctx, err := planGitReadiness(root, g)
		if err != nil {
			return err
		}
		return writePlanJSON(out, planReadiness(g, ctx))
	case "provenance":
		fs := flag.NewFlagSet("provenance", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		id := fs.String("plan", "", "Plan ID")
		bodyPath := fs.String("pr-body", "", "optional PR body path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return errors.New("unexpected arguments")
		}
		p, ok := g.ByID[*id]
		if !ok {
			return errors.New("unknown plan ID")
		}
		body := ""
		if *bodyPath != "" {
			b, err := os.ReadFile(filepath.Clean(*bodyPath))
			if err != nil {
				return err
			}
			body = string(b)
			if strings.TrimSpace(body) == "" {
				return errors.New("empty PR body")
			}
		}
		if err := planProvenance(root, p, body); err != nil {
			return err
		}
		return writePlanJSON(out, map[string]string{"plan_id": p.PlanID, "branch": expectedPlanBranch(p), "status": "valid"})
	default:
		return fmt.Errorf("unknown plans command %q", args[0])
	}
}

// Immutable identity is checked against each configured base, not filenames.
// A historical lifecycle ID cannot disappear merely because its title/path changes.
func planIdentityHistory(root string, g *planGraph) error {
	// The repository baseline is trusted independently of mutable Plan metadata.
	bases := map[string]bool{"master": true}
	for _, p := range g.Plans {
		bases[p.BaseBranch] = true
	}
	keys := make([]string, 0, len(bases))
	for base := range bases {
		keys = append(keys, base)
	}
	sort.Strings(keys)
	for _, base := range keys {
		rev, err := planRevision(root, base)
		if err != nil {
			return err
		}
		paths, err := planGit(root, "ls-tree", "-r", "--name-only", rev, "--", "docs/exec-plans")
		if err != nil {
			return err
		}
		for _, path := range strings.Split(paths, "\n") {
			if !strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".ja.md") {
				continue
			}
			data, err := planGit(root, "show", rev+":"+path)
			if err != nil {
				return err
			}
			p, legacy, err := parsePlanMetadata([]byte(data), path)
			if err != nil {
				return err
			}
			if !legacy {
				if _, ok := g.ByID[p.PlanID]; !ok {
					return fmt.Errorf("immutable Plan ID %s from %s disappeared; retain abandoned/completed record", p.PlanID, base)
				}
			}
		}
	}
	return nil
}

// A reachable commit containing a draft Plan is not sufficient merge evidence.
// Require the named merge/squash result to carry this Plan's branch provenance.
func planMergeProof(root string, p planMetadata, base string) error {
	if !planAncestor(root, p.MergeCommit, base) {
		return errors.New("merge result is not in base")
	}
	parents, err := planGit(root, "show", "-s", "--format=%P", p.MergeCommit)
	if err != nil {
		return err
	}
	list := strings.Fields(parents)
	source := p.MergeCommit
	if len(list) == 2 {
		source = list[1]
	} else if len(list) != 1 {
		return errors.New("unsupported merge parent count")
	}
	ok, err := planIdentityAt(root, source, p.PlanID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("merge source does not contain Plan identity")
	}
	body, err := planGit(root, "show", "-s", "--format=%B", source)
	if err != nil {
		return err
	}
	cmd := exec.Command("git", "interpret-trailers", "--parse")
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(body)
	b, err := cmd.Output()
	if err != nil {
		return err
	}
	count := 0
	for _, line := range strings.Split(string(b), "\n") {
		key, value, has := strings.Cut(line, ":")
		if has && strings.EqualFold(strings.TrimSpace(key), "ExecPlan") {
			if strings.TrimSpace(value) != p.PlanID {
				return errors.New("merge source trailer names another Plan")
			}
			count++
		}
	}
	if count != 1 {
		return errors.New("merge source lacks unique ExecPlan trailer")
	}
	return nil
}
