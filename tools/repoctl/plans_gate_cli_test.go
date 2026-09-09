package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestPlanGateAdapterFailClosedWithInjectedGitHub(t *testing.T) {
	policy, e := passingPlanMergeGate()
	p := planMetadata{PlanID: "EP-TEST-001", PlanType: "implementation", Status: "active", MergePolicy: "automatic", BaseBranch: "master"}
	config, _ := json.Marshal(map[string]any{"version": 1, "policies": map[string]PlanMergePolicy{p.PlanID: policy}})
	pr := map[string]any{"state": "open", "draft": false, "merged": false, "body": "ExecPlan: EP-TEST-001", "mergeable_state": "clean", "user": map[string]string{"login": "author"}, "head": map[string]any{"sha": e.HeadSHA, "ref": expectedPlanBranch(p), "repo": map[string]string{"full_name": "owner/repo"}}, "base": map[string]any{"sha": e.BaseSHA, "ref": "master", "repo": map[string]string{"full_name": "owner/repo"}}}
	prData, _ := json.Marshal(pr)
	reviews := `[[{"id":1,"state":"APPROVED","commit_id":"` + e.HeadSHA + `","submitted_at":"2026-09-09T01:00:00Z","user":{"login":"reviewer"}}]]`
	runs := []map[string]any{}
	for _, c := range e.Checks {
		runs = append(runs, map[string]any{"name": c.Name, "head_sha": c.HeadSHA, "conclusion": c.Conclusion, "app": map[string]int64{"id": c.AppID}})
	}
	checks, _ := json.Marshal([]any{map[string]any{"check_runs": runs}})
	threads := `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[],"pageInfo":{"hasNextPage":false}}}}}}`
	runner := func(name string, args ...string) ([]byte, error) {
		if name == "git" {
			switch args[0] {
			case "remote":
				return []byte("git@github.com:owner/repo.git"), nil
			case "show":
				return config, nil
			case "rev-parse":
				return []byte(e.HeadSHA), nil
			}
		}
		if name != "gh" || args[0] != "api" {
			t.Fatalf("unexpected command %s %v", name, args)
		}
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "graphql"):
			return []byte(threads), nil
		case strings.Contains(joined, "/reviews?"):
			return []byte(reviews), nil
		case strings.Contains(joined, "/check-runs?"):
			return checks, nil
		default:
			return prData, nil
		}
	}
	result := collectPlanGate(p, "owner/repo", 12, runner, func(string) bool { return true })
	if result.Decision.Allow || len(result.Decision.Reasons) != 2 {
		t.Fatalf("expected only unproved acceptance and ruleset blockers: %+v", result)
	}
	reviews = `[[{"id":2,"state":"CHANGES_REQUESTED","commit_id":"` + strings.Repeat("c", 40) + `","submitted_at":"2026-09-09T00:00:00Z","user":{"login":"other"}},{"id":3,"state":"COMMENTED","commit_id":"` + e.HeadSHA + `","submitted_at":"2026-09-09T02:00:00Z","user":{"login":"other"}}]]`
	result = collectPlanGate(p, "owner/repo", 12, runner, func(string) bool { return true })
	if !strings.Contains(strings.Join(result.Decision.Reasons, " "), "blocking changes-requested") {
		t.Fatalf("comment erased blocking review: %+v", result)
	}
	threads = `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[],"pageInfo":{"hasNextPage":true}}}}}}`
	result = collectPlanGate(p, "owner/repo", 12, runner, func(string) bool { return true })
	if !strings.Contains(strings.Join(result.Decision.Reasons, " "), "review-thread retrieval is incomplete") {
		t.Fatal("partial threads accepted")
	}
	config = nil
	result = collectPlanGate(p, "owner/repo", 12, runner, func(string) bool { return true })
	if result.Decision.Allow {
		t.Fatal("missing policy accepted")
	}
}

func TestPlanGateAdapterMissingTrustAndWrongRepository(t *testing.T) {
	p := planMetadata{PlanID: "EP-TEST-001", MergePolicy: "guarded"}
	calledGH := false
	runner := func(name string, args ...string) ([]byte, error) {
		if name == "gh" {
			calledGH = true
		}
		return []byte("https://github.com/another/repo.git"), nil
	}
	r := collectPlanGate(p, "owner/repo", 1, runner, func(string) bool { return false })
	if r.Decision.Allow || calledGH {
		t.Fatal("foreign repository queried")
	}
	runner = func(name string, args ...string) ([]byte, error) { return nil, errors.New("unavailable") }
	r = collectPlanGate(p, "owner/repo", 1, runner, func(string) bool { return false })
	if r.Decision.Allow || len(r.Decision.Reasons) == 0 {
		t.Fatal("missing evidence accepted")
	}
}
