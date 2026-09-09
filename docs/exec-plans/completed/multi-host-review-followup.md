---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Address PR 12 control-plane review

[日本語](multi-host-review-followup.ja.md)

## Purpose / Big Picture

Repair the seven PR 12 findings without changing the native runtime ownership or
conservative cleanup guarantees. Expected branch: `feat/multi-host-control-plane`.
This follow-up governs review fixes to the completed multi-host-control-plane plan.
The user explicitly requests implementation, push, replies and Resolve on every thread.

## Progress

- [x] 2026-09-09: Read the harness, verify clean PR head and retrieve all seven unresolved threads.
- [x] Stabilize implicit remote create owner across fresh-process retries.
- [x] Persist controller-owned TTL and schedule conservative expiry cleanup.
- [x] Validate advertised worker capacity against local policy.
- [x] Prevent transient UI/Browser input text from entering controller/worker journals.
- [x] Fence host removal against active operations and refuse submissions after removal.
- [x] Stop retrying permanent worker registration rejections.
- [x] Preserve uncertainty for every post-boundary create failure and allow safe follow-up cleanup.
- [x] Run focused regressions, full harness/race and native CI; review integrated changes.
- [x] Push fixes, reply to and resolve each addressed thread, archive this bilingual plan.

## Surprises & Discoveries

Several review comments are terse; implementation and existing contracts determine
the precise failure and required regression, not the comment wording alone.

Fail-before regressions reproduced fresh-process owner changes, excessive worker
capacity reaching TLS setup, host removal with active operations, missing expiry,
permanent registration retries, ambiguous create reported as failed, and plaintext
accepted into worker journals. Independent review also reproduced startup failure
when legacy operations contained invalid TTL. Historical-only migration now uses
the original create timestamp and default TTL, ignoring invalid historical renews;
new submissions still reject invalid TTL. It never frees capacity during migration.

## Decision Log

- Keep controller placement/TTL authority separate from worker local resource and cleanup authority.
- Preserve ordinary local owner defaults while making remote retry identity stable.
- A durable operation queue cannot provide the local non-persistent text-input contract.
  Reject remote text input before submission/persistence until such a channel exists;
  never silently redact and execute a different value.

- Use acceptance timestamps for TTL, with a persisted cleanup-operation marker;
  sweep expiry at startup, every second and on poll. Join the sweeper at shutdown.
- Preserve compensated create payloads but reserve released LocalState confirmation
  for destroy, so uncertain create remains conservatively accounted for.
- Independent worker/protocol and controller/CLI reviews completed. The migration
  finding was repaired and the original reproducer passed with race enabled.

## Outcomes & Retrospective

All seven findings are repaired in `80d031d`, with individual replies and resolved
threads. Full local harness, race, Docker integration, two-worker native fixture,
independent reviews, and all four native/preview CI workflows passed. Push Verify
34330840976 also passed its unchanged failed-job rerun. The initial Windows
failures remain documented below; their non-recurrence does not establish an
exact SQLite contention cause. Remote text input remains explicitly unsupported
until a non-persistent transport exists. This review plan is complete and archived.

## Context and Orientation

Review threads refer to CLI remote request construction, controller SQLite state,
worker receipt/result journals and the app reservation boundary. The completed
[parent implementation plan](../completed/multi-host-control-plane.md), product
contract and ADRs 0006/0007 retain architectural and portability requirements.

## Plan of Work

Use disjoint CLI, controller and worker implementation slices, with integration
owned by the root reviewer. Add fail-before regressions where practical. Update
bilingual durable contracts for TTL and unsupported remote text input. Integrate,
run the repository harness and applicable native/runtime checks, then associate
each thread reply with its concrete fix and validation evidence.

## Concrete Steps

Use the repository Go toolchain and `go run ./tools/repoctl check`, `go test -race
./...`, and focused package/native fixtures. Do not rewrite history. Use normal
commits/pushes on the existing PR branch. Reply and Resolve only after the issue is
fixed or a reasoned non-action is supported by evidence.

## Validation and Acceptance

Fresh-process create retries produce identical implicit-owner request identity.
Controller deadlines survive restart and renew retries, expired leases retain
capacity until cleanup proof, and removal cannot strand queued/dispatched work.
Invalid worker capacity fails before setup. Transient text is rejected before
both persistent journals, including direct API requests. Permanent registration
errors terminate; temporary failures retry. Post-boundary create errors remain
uncertain until authoritative cleanup, including compensated local creates.
All applicable harness/CI results and thread outcomes are recorded before completion.

## Idempotence and Recovery

Preserve durable replay/fencing and conservative absence proofs. Existing journals
are not rewritten to hide historical failures or sensitive-data risks. Review fixes
must use additive migrations where stored state changes.

## Artifacts and Notes

PR: https://github.com/mahcialet/agent-env/pull/12
Initial reviewed head: `efb12a0`.

## Interfaces and Dependencies

Keep controller components independent of app/provider implementations. Shared
protocol validation may reject transient text at CLI/controller/worker entry
points. Local app policy remains the limit for advertised worker capacity.

Validation evidence (2026-09-09): `repoctl check` passed; `go test -race ./...`
passed. After the final migration/shutdown changes, store/server race passed
(2.439s/2.426s). Worker/protocol independent race passed (9.413s/1.022s).
The two-worker `TestMultiHostNativeCLI` passed, covering real TLS CLI operations.
Raw JSON duplicate/case-variant input regressions and DB/WAL canary checks prove
rejected text does not reach durable journals. Native CI and thread closure remain
pending until the tested commits are pushed.

`repoctl test-integration` also passed locally with the Docker daemon. Commit
`80d031d` was pushed normally; all seven review threads received individual fix
and validation replies and were resolved. Native CI is still being checked.

PR Verify 34330844403, Multi-host native 34330844402, Browser native 34330844436
and Release preview 34330844500 passed on `80d031d`. The parallel push Verify
34330840976 failed existing Windows checks: `TestExitAndTimeout/exit` was killed
around its 100ms timeout before emitting its expected output, and
`TestConcurrentColdOpen` exceeded initialization's bounded deadline. The affected
execx/local SQLite implementations and tests are unchanged in this review fix.
These failures are recorded, not treated as passes; failed jobs were rerun without
changing timeout/assertion/concurrency settings. Their rerun result remains pending.

The SQLite failure reached the production 10-second initialization bound, not a
test-only deadline. Contention/load sensitivity is plausible, but the precise
lock holder is unconfirmed. Preserve this evidence for investigation if it recurs.

Final verification: Verify 34330840976 rerun completed successfully. All seven
review threads were resolved; no code/test relaxation was required for the rerun.
