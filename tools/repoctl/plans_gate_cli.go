package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type planGateCommand func(string, ...string) ([]byte, error)
type planGateReport struct {
	PlanID   string            `json:"plan_id"`
	Decision PlanMergeDecision `json:"decision"`
	HeadSHA  string            `json:"head_sha"`
	BaseSHA  string            `json:"base_sha"`
	Notice   string            `json:"notice"`
}

type planGatePR struct {
	State          string `json:"state"`
	Draft          bool   `json:"draft"`
	Merged         bool   `json:"merged"`
	Body           string `json:"body"`
	MergeableState string `json:"mergeable_state"`
	User           struct {
		Login string `json:"login"`
	} `json:"user"`
	Head struct {
		SHA  string `json:"sha"`
		Ref  string `json:"ref"`
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"head"`
	Base struct {
		SHA  string `json:"sha"`
		Ref  string `json:"ref"`
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"base"`
}

func executePlanGate(root string, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("plans gate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("plan", "", "Plan ID")
	pr := fs.Int("pr", 0, "PR number")
	repo := fs.String("repo", "", "owner/name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *pr <= 0 || !regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`).MatchString(*repo) {
		return errors.New("plans gate requires --plan ID --pr positive-number --repo owner/name")
	}
	g, err := loadPlanGraph(root)
	if err != nil {
		return err
	}
	p, ok := g.ByID[*id]
	if !ok {
		return errors.New("unknown plan ID")
	}
	run := func(name string, args ...string) ([]byte, error) {
		cmd := exec.Command(name, args...)
		cmd.Dir = root
		return cmd.Output()
	}
	report := collectPlanGate(p, *repo, *pr, run, func(body string) bool { return strings.TrimSpace(body) != "" && planProvenance(root, p, body) == nil })
	if err := writePlanJSON(out, report); err != nil {
		return err
	}
	if !report.Decision.Allow {
		return errors.New("merge gate BLOCKED; guarded/manual review remains required")
	}
	return nil
}

func collectPlanGate(p planMetadata, repo string, pr int, run planGateCommand, provenance func(string) bool) planGateReport {
	report := planGateReport{PlanID: p.PlanID, Notice: "Read-only eligibility inspection. No merge or auto-merge is executed. Atomic ruleset/base enforcement and machine acceptance proof are unavailable; guarded/manual fallback is required."}
	blocked := func(reason string) planGateReport {
		report.Decision = PlanMergeDecision{Reasons: []string{reason}, Allow: false}
		return report
	}
	remote, err := run("git", "remote", "get-url", "origin")
	if err != nil {
		return blocked("origin repository identity is unavailable")
	}
	url := strings.TrimSuffix(strings.TrimSpace(string(remote)), ".git")
	if url != "https://github.com/"+repo && url != "git@github.com:"+repo && url != "ssh://git@github.com/"+repo {
		return blocked("origin does not match requested GitHub repository")
	}
	endpoint := "repos/" + repo + "/pulls/" + strconv.Itoa(pr)
	readPR := func() (planGatePR, error) {
		var v planGatePR
		b, e := run("gh", "api", endpoint)
		if e != nil {
			return v, e
		}
		e = json.Unmarshal(b, &v)
		return v, e
	}
	current, err := readPR()
	if err != nil {
		return blocked("current PR evidence is unavailable")
	}
	report.HeadSHA = current.Head.SHA
	report.BaseSHA = current.Base.SHA
	if !validPlanMergeSHA(current.Head.SHA) || !validPlanMergeSHA(current.Base.SHA) || current.Base.Repo.FullName != repo || current.Head.Repo.FullName != repo {
		return blocked("PR repository or commit identity is invalid")
	}
	policyBytes, err := run("git", "show", current.Base.SHA+":.github/execplan-gates.json")
	if err != nil {
		return blocked("trusted base .github/execplan-gates.json policy unavailable; guarded/manual fallback")
	}
	var config struct {
		Version  int                        `json:"version"`
		Policies map[string]PlanMergePolicy `json:"policies"`
	}
	decoder := json.NewDecoder(bytes.NewReader(policyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return blocked("trusted policy is malformed")
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF || config.Version != 1 {
		return blocked("trusted policy version or trailing content is invalid")
	}
	policy, ok := config.Policies[p.PlanID]
	if !ok {
		return blocked("Plan has no trusted base merge policy")
	}
	if policy.Mode != p.MergePolicy {
		return blocked("working Plan merge policy differs from trusted base")
	}
	e := PlanMergeEvidence{HeadSHA: current.Head.SHA, BaseSHA: current.Base.SHA, Author: current.User.Login, Open: current.State == "open" && !current.Merged, Draft: current.Draft, GraphValid: true, NoBlocker: p.Status == "active" && len(p.Blockers) == 0, BaseUpToDate: current.MergeableState == "clean"}
	localHead, headErr := run("git", "rev-parse", "HEAD")
	e.ProvenanceValid = headErr == nil && strings.TrimSpace(string(localHead)) == current.Head.SHA && current.Head.Ref == expectedPlanBranch(p) && current.Base.Ref == p.BaseBranch && provenance(current.Body)
	// Reviews are REST structured objects, never free-form comment authorizations.
	var pages [][]struct {
		ID          int64  `json:"id"`
		State       string `json:"state"`
		CommitID    string `json:"commit_id"`
		SubmittedAt string `json:"submitted_at"`
		User        struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if data, err := run("gh", "api", "--paginate", "--slurp", endpoint+"/reviews?per_page=100"); err == nil && json.Unmarshal(data, &pages) == nil && len(pages) > 0 {
		all := map[string]struct {
			ID     int64
			At     string
			Review PlanMergeReview
		}{}
		e.ReviewsComplete = true
		for _, page := range pages {
			for _, r := range page {
				key := strings.ToLower(r.User.Login)
				_, timestampErr := time.Parse(time.RFC3339, r.SubmittedAt)
				if key == "" || r.ID <= 0 || timestampErr != nil {
					e.ReviewsComplete = false
					continue
				}
				if r.State != "APPROVED" && r.State != "CHANGES_REQUESTED" && r.State != "DISMISSED" && r.State != "COMMENTED" && r.State != "PENDING" {
					e.ReviewsComplete = false
				}
				old, exists := all[key]
				actionable := r.State == "APPROVED" || r.State == "CHANGES_REQUESTED" || r.State == "DISMISSED"
				oldActionable := old.Review.State == "APPROVED" || old.Review.State == "CHANGES_REQUESTED" || old.Review.State == "DISMISSED"
				if !exists || actionable && !oldActionable || actionable == oldActionable && (r.SubmittedAt > old.At || r.SubmittedAt == old.At && r.ID > old.ID) {
					all[key] = struct {
						ID     int64
						At     string
						Review PlanMergeReview
					}{r.ID, r.SubmittedAt, PlanMergeReview{Reviewer: r.User.Login, HeadSHA: r.CommitID, State: r.State}}
				}
			}
		}
		keys := []string{}
		for key := range all {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			e.Reviews = append(e.Reviews, all[key].Review)
		}
	}
	var checkPages []struct {
		CheckRuns []struct {
			Name       string `json:"name"`
			HeadSHA    string `json:"head_sha"`
			Conclusion string `json:"conclusion"`
			App        struct {
				ID int64 `json:"id"`
			} `json:"app"`
		} `json:"check_runs"`
	}
	if data, err := run("gh", "api", "--paginate", "--slurp", "repos/"+repo+"/commits/"+current.Head.SHA+"/check-runs?per_page=100&filter=latest"); err == nil && json.Unmarshal(data, &checkPages) == nil && len(checkPages) > 0 {
		e.ChecksComplete = true
		for _, page := range checkPages {
			for _, c := range page.CheckRuns {
				category := ""
				for _, required := range policy.RequiredChecks {
					if required.Name == c.Name && required.AppID == c.App.ID {
						category = required.Category
					}
				}
				e.Checks = append(e.Checks, PlanMergeCheck{Name: c.Name, Category: category, AppID: c.App.ID, HeadSHA: c.HeadSHA, Conclusion: c.Conclusion})
			}
		}
	}
	owner, name, _ := strings.Cut(repo, "/")
	query := `query($owner:String!,$name:String!,$number:Int!){repository(owner:$owner,name:$name){pullRequest(number:$number){reviewThreads(first:100){nodes{isResolved} pageInfo{hasNextPage}}}}}`
	var threads struct {
		Errors []json.RawMessage `json:"errors"`
		Data   struct {
			Repository *struct {
				PullRequest *struct {
					ReviewThreads *struct {
						Nodes []struct {
							IsResolved *bool `json:"isResolved"`
						} `json:"nodes"`
						PageInfo struct {
							HasNextPage *bool `json:"hasNextPage"`
						} `json:"pageInfo"`
					} `json:"reviewThreads"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	data, err := run("gh", "api", "graphql", "-f", "query="+query, "-f", "owner="+owner, "-f", "name="+name, "-F", fmt.Sprintf("number=%d", pr))
	if err == nil && json.Unmarshal(data, &threads) == nil && len(threads.Errors) == 0 && threads.Data.Repository != nil && threads.Data.Repository.PullRequest != nil && threads.Data.Repository.PullRequest.ReviewThreads != nil {
		t := threads.Data.Repository.PullRequest.ReviewThreads
		e.ThreadsComplete = t.PageInfo.HasNextPage != nil && !*t.PageInfo.HasNextPage
		for _, node := range t.Nodes {
			if node.IsResolved == nil {
				e.ThreadsComplete = false
			} else if !*node.IsResolved {
				e.UnresolvedThreads++
			}
		}
	}
	observed, err := readPR()
	if err == nil {
		e.ObservedHeadSHA = observed.Head.SHA
		e.ObservedBaseSHA = observed.Base.SHA
	}
	// GitHub "clean" does not prove strict base/merge-queue enforcement. No
	// ruleset permissions or free-form acceptance text are promoted into proof.
	e.RulesetVerified = false
	e.AcceptanceComplete = false
	report.Decision = EvaluatePlanMergeGate(policy, e)
	return report
}
