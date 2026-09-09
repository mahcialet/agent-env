---
status: completed
owner: maintainers
last_verified: 2026-09-09
---

# Address PR #11 correctness review

[日本語](repository-correctness-review.ja.md)

Expected branch: `audit/repository-correctness`.
This plan is the execution authority for PR #11 review follow-up. The
[completed audit](../completed/repository-correctness-audit.md) preserves its
original evidence; this follow-up handles review findings without rewriting history.

## Purpose / Big Picture

Fix two PR #11 findings: a popup between page census and creation can exceed the
128-page limit; heading anchor extraction substitutes inline code text and rejects
valid links. Reply to and resolve both review threads after verified fixes are pushed.

## Progress

- [x] (2026-09-09) Confirm clean PR branch at6b13cd4 and read both unresolved threads.
- [x] (2026-09-09) Reproduced and repaired both defects; added malformed/missing-type and all-target absence regressions.
- [x] (2026-09-09) Independently review changes and run harness, race and relevant native CI.
- [x] (2026-09-09) Pushed f10ecd4/de6da4f/ab71b64; replied to both threads and confirmed resolved state.
- [x] (2026-09-09) Record outcomes and archive this bilingual plan.

## Surprises & Discoveries

- 2026-09-09 final checkpoint: all checks below have completed, including unchanged Windows rerun attempt2. Earlier pending statements record their historical checkpoint; no acceptance gate remains open.

- 2026-09-09: all PR Verify34298063439 jobs, both Browser native runs and Release preview34298063300 pass atab71b64. Duplicate push Verify34298060578 Windows1.27 failed unchanged TestRunnerReapsOrdinaryDescendants/timeout: the 300ms timeout expired before the helper emitted its readiness marker. This is a fixture startup precondition failure, not evidence that a descendant survived cleanup. The same-head PR Windows1.27 job passed. Rerun the failed job without changing product/tests and retain both results; the precise scheduling cause is not established.

- 2026-09-09 independent review: initial popup-race repair passed full harness/race and Linux Browser integration, but a malformed census item with the created ID and missing type was filtered out and falsely proved absence. A private negative failed0.024s. Require complete target identity/type and compare exact absence against all targets, not only page-filtered results. Revalidate this repair before push. Markdown independent race×5 passes2.663s; existing visible-source-link filtering remains unchanged. Command lock-probe race×10 passes4.390s with all original assertions.

- 2026-09-09: inline-code anchor regression failed four fixtures before repair; full repoctl package race passes8.915s after splitting block/inline filtering. An early full harness attempt overlapped unfinished CDP test formatting and stopped at format-check; rerun after files stabilize. PR Verify34296197727 Windows Go1.26 failed only the final fresh-lock release probe in TestNamedCommandFailuresRetainEvidence/sleep, after timed-out command/run/artifact assertions passed. The probe used1s TTL while the real command uses2min. Same-head push CI passed. Adjust only probe TTL to1min to avoid making lock availability verification depend on subsecond DB/scheduler latency; keep every assertion and production fence unchanged. Recheck native CI.

The current inline-span sanitizer replaces code contents with `code`, rather than
simply deleting them; the review correctly identifies a rendered-anchor mismatch.

## Decision Log

- 2026-09-09, implementation coordinator: post-create validation uses complete target census. Compensate only a returned identity absent from every original target (including workers), reverify native ownership before close, require close acknowledgement and exact absence among all targets. Only then return a confirmed failure through the existing confirmedError; failed ownership/identity/protocol/absence evidence remains unconfirmed. External popup pages are never cleanup targets. Full final harness and full race PASS; CDP8.233s, CLI5.624s. Real Linux Browser native race10.362s PASS. The original Windows job also passed unchanged on rerun attempt2, supporting the narrow probe-latency diagnosis; candidate Windows CI remains required.

- 2026-09-09, implementation coordinator: accept comments3963647055 and3963647059.
  Retain the existing CDP lifecycle ownership boundary and hidden-heading checks.
  Separate block filtering from inline-text handling so link/provenance extraction
  retains its existing safety behavior.

## Outcomes & Retrospective

Completed2026-09-09. Both PR #11 review threads are fixed, replied to and resolved.

- `f10ecd4`: preserve inline code text for heading anchors while retaining the
  existing hidden-block and provenance/link filtering. Four positive/negative
  docsCheck regressions failed before repair; independent focused race×5 passed2.663s.
- `de6da4f`: change only the post-command lock-release probe TTL from1s to1min.
  All status, evidence, reacquisition and release assertions remain. Focused race×10
  passed4.390s; the original failed Windows job also passed unchanged on rerun.
- `ab71b64`: recensus after page creation, and compensate only the new target on
  overflow or unverifiable census. Require native ownership, close acknowledgement
  and exact absence across all target types. Missing/ambiguous proof remains
  unconfirmed; existing targets and concurrent popups are preserved.

The original popup regression failed before repair. Independent review found
missing-type filtering could falsely prove absence; the corrected17-case test
and127/128/129 boundaries passed independent race×5 2.224s.
Full `repoctl check` and full `go test -race ./...` pass; final CDP race8.233s,
CLI race5.624s, real Linux Browser native race10.362s.

At exact product revision `ab71b64f3867ccced2a304640ade22db56706adf`, PR
Verify34298063439, push Verify34298060578 (attempt2), Browser native34298063305/
34298060667 and Release preview34298063300 all pass. This includes native
Windows/macOS/Linux and release archive smoke on all three OSes. The push
Windows1.27 first attempt failed an unchanged300ms helper-start precondition;
the same-head PR job and unchanged rerun passed. This is retained as a fixture
startup sensitivity, not evidence of surviving descendants or a silently weakened test.

Both threads received concrete commit/test replies and returned isResolved=true.
No accepted review finding remains open. This follow-up preserves the prior
audit's historical record, changes no production dependency or lifecycle owner,
and leaves only documentation completion after the validated product revision.
Future cleanup proofs must retain the full identity census before filtering for
presentation; otherwise an omitted type can be mistaken for absence.

## Context and Orientation

`internal/browser/cdp/client.go` owns page effects and census; app persists browser
run certainty. `tools/repoctl/main.go` computes fragments; `translations.go`
filters hidden Markdown content and provenance links. Tests must reach these consumers.

## Plan of Work

Reproduce popup growth using protocol counts and exact target IDs, then compensate
only the created target with ownership revalidation and honest uncertain errors.
Reproduce valid inline-code anchors and wrong substitute anchors through docsCheck;
reuse block filtering while preserving heading inline text. Validate and publish.

## Concrete Steps

Run `go test -race ./internal/browser/cdp ./tools/repoctl`,
`go run ./tools/repoctl check`, `go test -race ./...`, and the existing real Browser
native integration. Push the PR branch and inspect native CI; use the GitHub
thread reply and resolve mutations only after fixes are reviewable.

## Validation and Acceptance

- Page creation rechecks the census and removes only its own newly created target
  on overflow; concurrent popup/sibling pages survive. Failed cleanup remains an error.
- Valid inline-code anchors pass and nonexistent substitute anchors fail; fenced,
  commented and indented fake headings remain invalid.
- Full harness, relevant races, independent review and native Browser evidence pass.
- Both threads have concrete replies and resolved state; final bilingual docs pass.

## Idempotence and Recovery

Preserve existing commits; do not force push. If target cleanup cannot be proven,
report uncertainty and retain evidence instead of claiming successful creation.

## Artifacts and Notes

Replies: [page rollback](https://github.com/mahcialet/agent-env/pull/11#discussion_r3963707422), [heading anchors](https://github.com/mahcialet/agent-env/pull/11#discussion_r3963707551). Both resolve mutations returned true. Final independent popup/rollback17 cases plus boundary tests pass race×5 2.224s. Candidate ab71b64 CI: Verify34298060578/34298063439, Browser34298060667/34298063305, Release preview34298063300 all passed; results are reconciled.

This plan records reproductions, validation and thread outcomes. No developer-local
SDK paths or secrets enter durable files.

## Interfaces and Dependencies

No new dependency, CLI flag or runtime lifecycle owner. Markdown link extraction
must continue ignoring links written as code examples.
