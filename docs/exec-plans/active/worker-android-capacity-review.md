---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Worker Android capacity review

This ExecPlan owns PR #12 capacity correction on `feat/multi-host-control-plane`.

## Purpose / Big Picture

Reject worker Android capacity above the allocator pool before registration or state creation.

## Progress

- [x] 2026-09-09: Confirm the new thread and existing allocator bounds.
- [x] 2026-09-09: Implement and validate boundary rejection.
- [ ] 2026-09-09: Push, reply and resolve the thread; archive this plan.

## Surprises & Discoveries

Lease count preflight is bounded, but Android capacity checked only negativity.

## Decision Log

2026-09-09, implementation: derive exported capacity from the SQLite allocator port bounds; reuse it at the CLI composition boundary without adding a controller-to-local-store dependency.

## Outcomes & Retrospective

In progress.

## Context and Orientation

internal/cli/remote_services.go advertises slots. internal/store/sqlite/android.go owns the 5554..5682 even console port pool.

## Plan of Work

Add boundary regressions, shared allocator capacity and bilingual flag documentation.

## Concrete Steps

Run focused fail-before tests, repoctl check and focused race tests; inspect native CI after push.

## Validation and Acceptance

Negative values and values above 65 fail before TLS/state setup; 0 and 65 reach normal TLS preflight. The allocator still exhausts after 65 reservations.

## Idempotence and Recovery

No state migration or allocator expansion; normal commits only.

## Artifacts and Notes

PR #12 thread PRRT_kwDOURHsR86gmqDy.

## Interfaces and Dependencies

CLI already imports SQLite; allocator port bounds remain unchanged. No new dependency or shell workflow.

Validation evidence: the pre-fix CLI boundary test failed for 66 and 1000
(0.005s). `go run ./tools/repoctl check` passed, including unit, vet, docs,
generated and architecture checks. CLI/SQLite race passed (46.073s/15.922s),
including the existing real 65-reservation exhaustion/rollback regression.
The first harness run caught a missing English-source link in the new Japanese
plan; the link was added and the harness passed without changing check rules.
