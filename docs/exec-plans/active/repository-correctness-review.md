---
status: active
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
- [ ] Reproduce both defects with meaningful regressions, then repair them.
- [ ] Independently review changes and run harness, race and relevant native CI.
- [ ] Push verified fixes, reply to both threads and resolve them.
- [ ] Record outcomes and archive this bilingual plan.

## Surprises & Discoveries

- 2026-09-09 independent review: initial popup-race repair passed full harness/race and Linux Browser9.312s, but a malformed census item with the created ID and missing type was filtered out and falsely proved absence. A private negative failed0.024s. Require complete target identity/type and compare exact absence against all targets, not only page-filtered results. Revalidate this repair before push. Markdown independent race×5 passes2.663s; existing visible-source-link filtering remains unchanged. Command lock-probe race×10 passes4.390s with all original assertions.

- 2026-09-09: inline-code anchor regression failed four fixtures before repair; full repoctl package race passes8.915s after splitting block/inline filtering. An early full harness attempt overlapped unfinished CDP test formatting and stopped at format-check; rerun after files stabilize. PR Verify34296197727 Windows Go1.26 failed only the final fresh-lock release probe in TestNamedCommandFailuresRetainEvidence/sleep, after timed-out command/run/artifact assertions passed. The probe used1s TTL while the real command uses2min. Same-head push CI passed. Adjust only probe TTL to1min to avoid making lock availability verification depend on subsecond DB/scheduler latency; keep every assertion and production fence unchanged. Recheck native CI.

The current inline-span sanitizer replaces code contents with `code`, rather than
simply deleting them; the review correctly identifies a rendered-anchor mismatch.

## Decision Log

- 2026-09-09, implementation coordinator: accept comments3963647055 and3963647059.
  Retain the existing CDP lifecycle ownership boundary and hidden-heading checks.
  Separate block filtering from inline-text handling so link/provenance extraction
  retains its existing safety behavior.

## Outcomes & Retrospective

Pending implementation and acceptance.

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

This plan records reproductions, validation and thread outcomes. No developer-local
SDK paths or secrets enter durable files.

## Interfaces and Dependencies

No new dependency, CLI flag or runtime lifecycle owner. Markdown link extraction
must continue ignoring links written as code examples.
