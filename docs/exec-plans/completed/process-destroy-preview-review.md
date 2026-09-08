---
status: completed
owner: maintainers
last_verified: 2026-09-08
---

# Process destroy preview review

[日本語](process-destroy-preview-review.ja.md)

## Purpose / Big Picture

Correct PR #9 destroy previews for persistent processes on `feat/persistent-process-runtime`. This plan governs this review follow-up; the completed implementation plan remains historical evidence.

## Progress

- [x] (2026-09-08) Inspect the review, architecture and cleanup contracts.
- [x] (2026-09-08) Reproduce, fix and validate process preview behavior.
- [x] (2026-09-08) Push correction `45175dc`, reply to the review thread and resolve it.

## Surprises & Discoveries

Process inspection was routed correctly, but preview formatting fell through to Compose whenever a process existed. Exited processes also need a private-state cleanup preview. The initial targeted regression failed for both live and exited states with force off/on, reproducing the review before changing production code.

## Decision Log

2026-09-08, maintainers: add the process case after ownership validation and before the Compose fallback; keep inspection errors on the existing quarantine path. No adapter or native lifecycle changes.

## Outcomes & Retrospective

The process case now reports native identity validation, termination if running, retained logs, whole-tree absence and private-state removal. Live and exited previews failed before the fix; all six state/force cases pass after it. Existing Android preview regression also passes.


Correction `45175dc` is pushed; the sole review thread was replied to and resolved. No review findings remain open. A post-commit whitespace check found extra blank EOF lines in the new plan pair; this archive update removes them without rewriting history.

## Context and Orientation

`internal/app/destroy_preview.go` formats read-only `Destroy` previews. `internal/app/process_lifecycle_test.go` provides process/source/store fixtures.

## Plan of Work

Add a public Destroy regression covering live, exited and uncertain processes with force both off and on. Verify registry, events, artifacts, private state and provider counters remain unchanged. Then fix formatting.

## Concrete Steps

Run the targeted test before/after the fix, `repoctl check`, and targeted race tests. Commit/push only reviewed changes and reply with concrete evidence.

## Validation and Acceptance

The preview names the process runtime and private-state path and describes identity validation, termination, log retention and cleanup after whole-tree absence. No Compose text appears for process-only leases. Uncertain ownership remains quarantine. All preview variants are read-only. Evidence: `go test -race ./internal/app -run 'Test(ProcessDestroyPreviewReportsCleanupWithoutEffects|AndroidDestroyPreviewReportsPrivateCleanupWithoutEffects)' -count=1` PASS; `go run ./tools/repoctl check` PASS (unit, vet, docs, generated and architecture checks).

## Idempotence and Recovery

Repeat dry-run safely; do not change published history or actual cleanup semantics. Failed validation remains recorded until resolved.

## Artifacts and Notes

Review: https://github.com/mahcialet/agent-env/pull/9#discussion_r3958123366

## Interfaces and Dependencies

No new interfaces, dependencies, OS-specific behavior or shell invocation.
