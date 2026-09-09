package main

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// PlanMergePolicy must come from trusted base policy, never PR-supplied evidence.
// Every category must have a required check or an explicit non-applicability
// rationale. CI and documentation checks cannot be exempted.
type PlanMergePolicy struct {
	Mode             string              `json:"mode"`
	TrustedReviewers []string            `json:"trusted_reviewers"`
	RequiredChecks   []PlanRequiredCheck `json:"required_checks"`
	ExemptCategories map[string]string   `json:"exempt_categories"`
}

type PlanRequiredCheck struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	AppID    int64  `json:"app_id"`
}

type PlanMergeCheck struct {
	Name       string `json:"name"`
	Category   string `json:"category"`
	AppID      int64  `json:"app_id"`
	HeadSHA    string `json:"head_sha"`
	Conclusion string `json:"conclusion"`
}

// Reviews are the complete set of effective (latest per reviewer) structured
// GitHub reviews. The adapter must retrieve all pages and resolve chronology;
// duplicates are ambiguous and rejected here. Dismissal is a structured state,
// not a comment. Even a stale changes-requested review blocks until superseded
// or dismissed. Review prose has deliberately no representation in this API.
type PlanMergeReview struct {
	Reviewer string `json:"reviewer"`
	HeadSHA  string `json:"head_sha"`
	State    string `json:"state"`
}

type PlanMergeEvidence struct {
	HeadSHA            string            `json:"head_sha"`
	BaseSHA            string            `json:"base_sha"`
	ObservedHeadSHA    string            `json:"observed_head_sha"`
	ObservedBaseSHA    string            `json:"observed_base_sha"`
	Author             string            `json:"author"`
	Open               bool              `json:"open"`
	Draft              bool              `json:"draft"`
	GraphValid         bool              `json:"graph_valid"`
	ProvenanceValid    bool              `json:"provenance_valid"`
	AcceptanceComplete bool              `json:"acceptance_complete"`
	NoBlocker          bool              `json:"no_blocker"`
	ThreadsComplete    bool              `json:"threads_complete"`
	ReviewsComplete    bool              `json:"reviews_complete"`
	ChecksComplete     bool              `json:"checks_complete"`
	RulesetVerified    bool              `json:"ruleset_verified"`
	BaseUpToDate       bool              `json:"base_up_to_date"`
	UnresolvedThreads  int               `json:"unresolved_threads"`
	Reviews            []PlanMergeReview `json:"reviews"`
	Checks             []PlanMergeCheck  `json:"checks"`
}

type PlanMergeDecision struct {
	Allow   bool     `json:"allow"`
	Reasons []string `json:"reasons"`
}

// EvaluatePlanMergeGate is pure eligibility evaluation, not merge authorization
// or a substitute for GitHub's atomic expected-HEAD and base/ruleset enforcement.
// The caller must re-observe immediately before an attempted merge and must not
// enable deferred auto-merge from this point-in-time decision.
func EvaluatePlanMergeGate(policy PlanMergePolicy, evidence PlanMergeEvidence) PlanMergeDecision {
	reasons := map[string]bool{}
	deny := func(reason string) { reasons[reason] = true }
	if policy.Mode != "automatic" {
		deny("merge policy does not permit automatic merge")
	}
	if !validPlanMergeSHA(evidence.HeadSHA) || evidence.HeadSHA != evidence.ObservedHeadSHA {
		deny("current HEAD evidence is missing or stale")
	}
	if !validPlanMergeSHA(evidence.BaseSHA) || evidence.BaseSHA != evidence.ObservedBaseSHA {
		deny("current base evidence is missing or stale")
	}
	flags := []struct {
		name string
		ok   bool
	}{
		{"pull request is not open", evidence.Open},
		{"graph validation is unproven", evidence.GraphValid},
		{"Git and PR provenance is unproven", evidence.ProvenanceValid},
		{"acceptance excluding merge/archive is incomplete", evidence.AcceptanceComplete},
		{"absence of explicit blockers is unproven", evidence.NoBlocker},
		{"review-thread retrieval is incomplete", evidence.ThreadsComplete},
		{"review retrieval is incomplete", evidence.ReviewsComplete},
		{"check retrieval is incomplete", evidence.ChecksComplete},
		{"GitHub ruleset enforcement is unproven", evidence.RulesetVerified},
		{"branch is not proven current with base", evidence.BaseUpToDate},
	}
	for _, flag := range flags {
		if !flag.ok {
			deny(flag.name)
		}
	}
	if evidence.Draft {
		deny("pull request is draft")
	}
	if evidence.Author == "" {
		deny("pull request author is unknown")
	}
	if evidence.UnresolvedThreads != 0 {
		deny("unresolved review threads or invalid thread count")
	}
	trusted := map[string]bool{}
	for _, reviewer := range policy.TrustedReviewers {
		if strings.TrimSpace(reviewer) == "" {
			deny("trusted reviewer identity is empty")
			continue
		}
		trusted[strings.ToLower(reviewer)] = true
	}
	if len(trusted) == 0 {
		deny("trusted reviewer policy is missing")
	}
	reviewers := map[string]bool{}
	approved := false
	for _, review := range evidence.Reviews {
		reviewer := strings.ToLower(review.Reviewer)
		if reviewer == "" || reviewers[reviewer] {
			deny("review identities are missing or ambiguous")
		}
		reviewers[reviewer] = true
		switch review.State {
		case "APPROVED":
			if trusted[reviewer] && !strings.EqualFold(review.Reviewer, evidence.Author) && review.HeadSHA == evidence.HeadSHA && validPlanMergeSHA(review.HeadSHA) {
				approved = true
			}
		case "CHANGES_REQUESTED":
			deny("a blocking changes-requested review remains")
		case "DISMISSED", "COMMENTED", "PENDING":
		default:
			deny("review state is unknown")
		}
	}
	if !approved {
		deny("trusted independent approval for current HEAD is missing")
	}
	categories := map[string]bool{"ci": true, "native": true, "integration": true, "docs": true}
	covered := map[string]bool{}
	required := map[string]bool{}
	if len(policy.RequiredChecks) == 0 {
		deny("required check policy is empty")
	}
	for _, check := range policy.RequiredChecks {
		if !categories[check.Category] || strings.TrimSpace(check.Name) == "" || check.AppID <= 0 {
			deny("required check identity or category is invalid")
		}
		key := fmt.Sprintf("%s/%d", check.Name, check.AppID)
		if required[key] {
			deny("required check policy contains ambiguous duplicate identities")
		}
		required[key] = true
		covered[check.Category] = true
		matches := 0
		success := false
		for _, observed := range evidence.Checks {
			if observed.Name == check.Name && observed.AppID == check.AppID {
				matches++
				success = observed.Category == check.Category && observed.HeadSHA == evidence.HeadSHA && observed.Conclusion == "success"
			}
		}
		if matches != 1 || !success {
			deny("required check is missing, ambiguous, stale or unsuccessful: " + key)
		}
	}
	for category, rationale := range policy.ExemptCategories {
		if !categories[category] || category == "ci" || category == "docs" || strings.TrimSpace(rationale) == "" || covered[category] {
			deny("check category exemption is invalid: " + category)
		}
	}
	for category := range categories {
		if !covered[category] && strings.TrimSpace(policy.ExemptCategories[category]) == "" {
			deny("check category has no requirement or explicit exemption: " + category)
		}
	}
	result := PlanMergeDecision{Allow: len(reasons) == 0, Reasons: []string{}}
	for reason := range reasons {
		result.Reasons = append(result.Reasons, reason)
	}
	sort.Strings(result.Reasons)
	return result
}

func validPlanMergeSHA(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}
