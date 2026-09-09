---
status: completed
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
- [x] 2026-09-09: Push, reply and resolve the thread; archive this plan.

## Surprises & Discoveries

Lease count preflight is bounded, but Android capacity checked only negativity.

## Decision Log

2026-09-09, implementation: derive exported capacity from the SQLite allocator port bounds; reuse it at the CLI composition boundary without adding a controller-to-local-store dependency.

## Outcomes & Retrospective

Completed in `07c83b4`: advertised Android capacity cannot exceed the fixed
allocator pool, and invalid capacity fails before resources or registration.
The new regression reproduces the original defect and checks both boundaries;
existing allocator exhaustion/rollback coverage establishes the actual pool size.
Independent review found no blocker. Deriving the advertised maximum from the
allocation bounds prevents another independently maintained capacity value.

All 40 checks now pass: PR Verify34341921634, push Verify34341916259 attempt2,
Multi-host native34341916300/34341921496, Browser native34341916340/34341921586
and Release preview34341921491. The push Windows Go1.27 first attempt timed out
as recorded below; the unchanged failed-job rerun succeeded. The corresponding
app race regression also passed five local repetitions (4.060s). This does not
prove a root cause for the initial transient timeout. No timeout/assertion changes
were made. The final documentation-only archival commit is validated separately
with repoctl docs-check; native evidence applies to the code commit above.

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

Independent read-only review found no blocker. Commit `07c83b4` was pushed;
thread PRRT_kwDOURHsR86gmqDy received the correction/validation reply and was
resolved. PR Verify34341921634 passed all jobs, including Windows Go1.26/1.27.
Push Verify34341916259 attempt1 Windows Go1.27 failed the unchanged
TestMobileConcurrentNamedTestsUseOwnedSerialAndPersistWarning at applications_test.go:585,
while creating two fixture leases under its existing 10-second context (18.74s).
The Android allocator diff substitutes equal constants without changing allocation
behavior; the CLI guard does not run in this app fixture. No exact contention
cause is proved. Preserve the failed evidence and rerun only that failed job with
unchanged source and assertions; do not enlarge timeouts to obtain a pass.
