package main

import (
	"reflect"
	"strings"
	"testing"
)

func passingPlanMergeGate() (PlanMergePolicy, PlanMergeEvidence) {
	head, base := strings.Repeat("a", 40), strings.Repeat("b", 40)
	p := PlanMergePolicy{Mode: "automatic", TrustedReviewers: []string{"reviewer"}}
	e := PlanMergeEvidence{HeadSHA: head, BaseSHA: base, ObservedHeadSHA: head, ObservedBaseSHA: base, Author: "author", Open: true, GraphValid: true, ProvenanceValid: true, AcceptanceComplete: true, NoBlocker: true, ThreadsComplete: true, ReviewsComplete: true, ChecksComplete: true, RulesetVerified: true, BaseUpToDate: true, Reviews: []PlanMergeReview{{Reviewer: "reviewer", HeadSHA: head, State: "APPROVED"}}}
	for _, category := range []string{"ci", "native", "integration", "docs"} {
		p.RequiredChecks = append(p.RequiredChecks, PlanRequiredCheck{Name: category, Category: category, AppID: 123})
		e.Checks = append(e.Checks, PlanMergeCheck{Name: category, Category: category, AppID: 123, HeadSHA: head, Conclusion: "success"})
	}
	return p, e
}

func TestPlanMergeGateAllowsOnlyCompleteCurrentEvidence(t *testing.T) {
	p, e := passingPlanMergeGate()
	if got := EvaluatePlanMergeGate(p, e); !got.Allow || len(got.Reasons) != 0 {
		t.Fatalf("complete evidence denied: %+v", got)
	}
	p.RequiredChecks = []PlanRequiredCheck{p.RequiredChecks[0], p.RequiredChecks[3]}
	p.ExemptCategories = map[string]string{"native": "documentation-only change", "integration": "documentation-only change"}
	if got := EvaluatePlanMergeGate(p, e); !got.Allow {
		t.Fatalf("explicit nonapplicability denied: %+v", got)
	}
}

func TestPlanMergeGateFailsClosed(t *testing.T) {
	cases := []struct {
		name   string
		change func(*PlanMergePolicy, *PlanMergeEvidence)
	}{
		{"guarded", func(p *PlanMergePolicy, e *PlanMergeEvidence) { p.Mode = "guarded" }},
		{"manual", func(p *PlanMergePolicy, e *PlanMergeEvidence) { p.Mode = "manual" }},
		{"unknown mode", func(p *PlanMergePolicy, e *PlanMergeEvidence) { p.Mode = "" }},
		{"later commit invalidates review", func(p *PlanMergePolicy, e *PlanMergeEvidence) {
			e.HeadSHA = strings.Repeat("c", 40)
			e.ObservedHeadSHA = e.HeadSHA
			for i := range e.Checks {
				e.Checks[i].HeadSHA = e.HeadSHA
			}
		}},
		{"raced head", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.ObservedHeadSHA = strings.Repeat("c", 40) }},
		{"raced base", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.ObservedBaseSHA = strings.Repeat("c", 40) }},
		{"invalid SHA", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.BaseSHA = "main"; e.ObservedBaseSHA = "main" }},
		{"unknown author", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Author = "" }},
		{"self approval", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Author = "REVIEWER" }},
		{"untrusted approval", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Reviews[0].Reviewer = "other" }},
		{"empty trust", func(p *PlanMergePolicy, e *PlanMergeEvidence) { p.TrustedReviewers = nil }},
		{"comment is not authorization", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Reviews[0].State = "COMMENTED" }},
		{"LGTM is not authorization", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Reviews[0].State = "LGTM" }},
		{"dismissed approval", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Reviews[0].State = "DISMISSED" }},
		{"pending approval", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Reviews[0].State = "PENDING" }},
		{"changes requested even stale", func(p *PlanMergePolicy, e *PlanMergeEvidence) {
			e.Reviews = append(e.Reviews, PlanMergeReview{Reviewer: "other", State: "CHANGES_REQUESTED", HeadSHA: strings.Repeat("c", 40)})
		}},
		{"ambiguous review chronology", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Reviews = append(e.Reviews, e.Reviews[0]) }},
		{"unresolved thread", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.UnresolvedThreads = 1 }},
		{"invalid thread count", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.UnresolvedThreads = -1 }},
		{"empty required checks", func(p *PlanMergePolicy, e *PlanMergeEvidence) { p.RequiredChecks = nil }},
		{"missing category", func(p *PlanMergePolicy, e *PlanMergeEvidence) { p.RequiredChecks = p.RequiredChecks[:3] }},
		{"native cannot silently disappear", func(p *PlanMergePolicy, e *PlanMergeEvidence) {
			p.RequiredChecks = append(p.RequiredChecks[:1], p.RequiredChecks[2:]...)
		}},
		{"empty exemption", func(p *PlanMergePolicy, e *PlanMergeEvidence) { p.ExemptCategories = map[string]string{"native": ""} }},
		{"docs exemption forbidden", func(p *PlanMergePolicy, e *PlanMergeEvidence) {
			p.RequiredChecks = p.RequiredChecks[:3]
			p.ExemptCategories = map[string]string{"docs": "skip"}
		}},
		{"missing check", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Checks = e.Checks[:3] }},
		{"stale check", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Checks[0].HeadSHA = strings.Repeat("c", 40) }},
		{"failed check", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Checks[0].Conclusion = "failure" }},
		{"skipped check", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Checks[0].Conclusion = "skipped" }},
		{"wrong publisher", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Checks[0].AppID = 456 }},
		{"wrong category", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Checks[0].Category = "docs" }},
		{"ambiguous check", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Checks = append(e.Checks, e.Checks[0]) }},
		{"duplicate requirement", func(p *PlanMergePolicy, e *PlanMergeEvidence) {
			p.RequiredChecks = append(p.RequiredChecks, p.RequiredChecks[0])
		}},
		{"unknown publisher", func(p *PlanMergePolicy, e *PlanMergeEvidence) { p.RequiredChecks[0].AppID = 0 }},
		{"draft", func(p *PlanMergePolicy, e *PlanMergeEvidence) { e.Draft = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, e := passingPlanMergeGate()
			tc.change(&p, &e)
			got := EvaluatePlanMergeGate(p, e)
			if got.Allow || len(got.Reasons) == 0 {
				t.Fatalf("unsafe merge permitted: %+v", got)
			}
			if again := EvaluatePlanMergeGate(p, e); !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic decision")
			}
		})
	}
	// Every independent boolean proof is required, with zero-value false denying.
	for _, field := range []string{"Open", "GraphValid", "ProvenanceValid", "AcceptanceComplete", "NoBlocker", "ThreadsComplete", "ReviewsComplete", "ChecksComplete", "RulesetVerified", "BaseUpToDate"} {
		t.Run(field, func(t *testing.T) {
			p, e := passingPlanMergeGate()
			reflect.ValueOf(&e).Elem().FieldByName(field).SetBool(false)
			if got := EvaluatePlanMergeGate(p, e); got.Allow {
				t.Fatal("missing proof allowed merge")
			}
		})
	}
	if got := EvaluatePlanMergeGate(PlanMergePolicy{}, PlanMergeEvidence{}); got.Allow {
		t.Fatal("zero evidence allowed merge")
	}
}

func TestPlanMergeGateDismissedBlockerDoesNotAuthorize(t *testing.T) {
	p, e := passingPlanMergeGate()
	e.Reviews = append(e.Reviews, PlanMergeReview{Reviewer: "other", State: "DISMISSED"})
	if got := EvaluatePlanMergeGate(p, e); !got.Allow {
		t.Fatalf("dismissed blocker remains active: %+v", got)
	}
	e.Reviews = e.Reviews[1:]
	if got := EvaluatePlanMergeGate(p, e); got.Allow {
		t.Fatal("dismissal became approval")
	}
}
