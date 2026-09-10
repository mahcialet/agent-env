---
status: completed
owner: maintainers
last_verified: 2026-09-09
---

# Multi-host review round two

This living ExecPlan owns the additional PR #12 review fixes on
`feat/multi-host-control-plane`. The completed review follow-up remains historical.

## Purpose / Big Picture

Make CLI controller expiry operational, permit context-bounded streaming blobs,
avoid duplicate manifest envelopes, and make acknowledged CAS publication durable.

## Progress

- [x] 2026-09-09: Read the four new threads and reconcile the prior completed fixes.
- [x] 2026-09-09: Route actual CLI controller serving through the shared expiry lifecycle.
- [x] 2026-09-09: Separate streaming blob deadlines from bounded metadata requests.
- [x] 2026-09-09: Serialize the canonical manifest once and retain worker verification.
- [x] 2026-09-09: Persist CAS directory publication with portable platform support.
- [x] 2026-09-09: Capture bounded source diffs for forced remote cleanup.
- [x] 2026-09-09: Reuse validated materialized packages before extraction.
- [x] 2026-09-09: Confirm released leases after explicit reconcile.
- [x] 2026-09-09: Fence incarnation replacement against active copied worker roots.
- [x] 2026-09-09: Preserve lease state after uncertain read-only logs/artifacts.
- [x] 2026-09-09: Bound Compose log capture before buffering remote results.
- [x] 2026-09-09: Bound Android display-log reads before allocation (new overlapping thread).
- [x] 2026-09-09: Make retained package publication durable before effects, including retries.
- [x] 2026-09-09: Preserve the running UI recovery barrier in uncertain results.
- [x] 2026-09-09: Separate successful action outcome from failed artifact publication.
- [x] 2026-09-09: Validate, push, reply to and resolve all additional threads; archive this plan.

## Surprises & Discoveries

The prior server sweep test exercised server.Run while CLI serve duplicated its
listener lifecycle. Actual CLI serving therefore bypassed the tested sweep.

Fail-before actual CLI test (5.021s) queued no cleanup; the fixed entry point
passed (1.032s). Client upload/download failed the scaled metadata deadline;
blocked upload also exposed missing request-body close on cancellation. Server
TLS regressions failed with deadline clearing disabled, then passed (race1.724s).
The near-limit manifest failed with old CLI/worker overlays (1.950s/0.331s), then
passed real mTLS create/GetOperation/poll and worker validation (race33.913s/3.147s).

An initial test fixture used the wrong controller home/log marker, then exposed
real missing sweep after correction. Its independent SQLite connection needed a
bounded busy timeout to coordinate with legitimate server transactions. Manifest
fixture CA names and capabilities were corrected; limits/auth checks were kept.
CAS->instance violated architecture rules, so Windows publication uses direct
native byte-range locking without changing the architecture rule.

## Decision Log

Use one controller serving lifecycle and test the actual CLI entry point.
Keep compact package hydration explicit; accept legacy full packages without
relaxing manifest mismatch checks. Stream operations honor caller cancellation.

CAS uses native durability barriers: Unix directory fsync; Windows write-through
moves plus cancellable bounded native publication locking. Duplicate/retry paths
also confirm durability before acknowledgement. Native extended/UNC conversion is
shared by move and lock creation. No physical power-loss proof is claimed.

Independent reviews covered CLI lifecycle, stream authorization, CAS locking and
publication; final native Windows behavior remains for CI.

## Outcomes & Retrospective

All 14 additional review threads owned by this plan were addressed, replied to
and resolved; together with the earlier follow-up, all 21 PR #12 threads are
resolved. Delivered changes restore cleanup/recovery authority, bound transport
and display data before allocation, preserve action outcomes, and durably publish
retained inputs. Production-entry-point and deterministic failure-order tests
caught gaps that wrapper-only and happy-path tests missed. Independent review
also exposed missing adapter forwarding and a legacy duplicate durability gap.

Implementation commits `3dba741`, `9707fed`, `90ea6c3` and `c056a76` are published
without history rewriting. Final code commit `c056a76` passed all 40 PR checks:
Verify34337028533/34337035407, Multi-host native34337028511/34337035381,
Browser native34337028471/34337035387 and Release preview34337035402. This includes
native Windows/macOS/Linux, Go1.26/1.27, Docker integration and release smoke.
Physical power-loss experiments and malicious indistinguishable journal/credential
copies remain outside these claims. The final archival change is documentation
only and is checked separately with repoctl docs-check.

## Context and Orientation

CLI remote_services starts services; server.Run owns expiry and shutdown.
client owns HTTP transport, remotesource/worker own package validation, and
blobstore owns CAS file publication.

## Plan of Work

Repair each boundary with tests detecting the same defects before the fix and independent review.
Maintain English/Japanese contracts and retain prior failed CI evidence.

## Concrete Steps

Run focused Go tests, repoctl check, full race and native multi-host fixtures.
Push tested commits normally and verify CI before recording completion.

## Validation and Acceptance

An idle offline expired lease is queued via the real CLI server. Blob transfers
outlive metadata request limits but stop on context cancellation. Near-limit
manifests fit create/poll envelopes and retain verification. CAS acknowledges only
after platform-appropriate durable publication, including retries.

## Idempotence and Recovery

Never relax release proof or rewrite history. Preserve failed publication evidence
and ensure retries cannot acknowledge an unsynced winner.

## Artifacts and Notes

PR #12 threads: PRRT_kwDOURHsR86gk8fC, PRRT_kwDOURHsR86gk8fG,
PRRT_kwDOURHsR86gk8fL, PRRT_kwDOURHsR86gk8fQ.

## Interfaces and Dependencies

Keep the controller independent of host runtimes; add no shell dependency.

Validation so far: full `go test -race ./...` passed; native two-worker CLI fixture
passed (21.073s). Final harness/Docker integration and pushed native CI are pending.

Independent result-boundary review reproduced a near-limit manifest causing an
otherwise completed worker response to become failed (0.119s). Remote responses
therefore copy and omit repeated manifest and process command/env declarations;
local stored state stays intact and the 4 MiB result bound remains enforced.

Final local evidence: repoctl check and Docker integration passed. Final changed
packages race passed (worker8.806s/CLI43.366s; server/blobstore passed). Large
response tests preserve completed status, original input and the combined8MiB
operation envelope. Native multi-host CLI passed with compact download hydration.

After `3dba741` was pushed, six additional review threads arrived and remain in
this active plan: PRRT_kwDOURHsR86glT_l, PRRT_kwDOURHsR86glT_s,
PRRT_kwDOURHsR86glT_y, PRRT_kwDOURHsR86glT_2, PRRT_kwDOURHsR86glT_7,
PRRT_kwDOURHsR86glUAB. They cover cleanup evidence, package reuse, reconcile proof,
worker incarnation fencing, read-only operation state and bounded Compose logs.

Additional discoveries: real17MiB Git diff bypassed the old buffer limit through
promoted bytes.Buffer.ReadFrom; replacing the embedded buffer made the bound real.
Remotesource full race then passed (5.549s). The initial shared Compose cap also
affected required cleanup evidence and could strand noisy services; the final
scope separates bounded display from the existing cleanup capture contract.
Android display logs must be limited before reading, not after aggregation.

CI on `3dba741`: platform Multi-host native, Browser native and Release preview
passed. Push Verify34333854477 exposed a new cancellation-fixture race (graceful
server EOF could win before the client observed cancellation). Keep the handler
unfinished until cancellation is observed, retaining exact error assertions;
repeated race5 passed (16.236s). PR Verify34333859610 failed the unchanged
TestLifecycleCreatePersistedIntentAndUniqueIsolation at lifecycle_test.go229 with
context deadline exceeded; retain this evidence and verify the next commit's CI.

Final six-fix evidence: store/server race passed (2.241s/2.969s), including online
handoff refusal, offline no-redelivery, unknown-owner migration and read-only state
preservation. Worker compensated-create/reconcile/controller proof passed; app
race passed (52.942s). Native CLI restart fixture passed (46.249s) with the30-second
offline threshold. Actual remote Docker and Podman compose fixture passed
(7.34s/21.61s), verifying the production CLI bounded-display adapter. Independent
review caught that missing adapter forwarding and it was repaired before push.

A proposed post-destroy cache concern was disproved: removal deletes lease
worktrees, not bare cache repositories. The retained test checks post-destroy
cache reuse through logs/artifact/reconcile/repeated destroy using an empty CAS without
relaxing validation. Source diff and reuse have fail-before coverage; a17MiB real
diff fails safely at the enforced capture bound, retaining cleanup safety.

After `9707fed` was pushed, four further threads arrived: PRRT_kwDOURHsR86glp9w,
PRRT_kwDOURHsR86glp93, PRRT_kwDOURHsR86glp9_, PRRT_kwDOURHsR86glp-C. Android
bounded reads are already included in that commit. This plan also owns retained
package publication durability and truthful UI/action outcomes. Retention methods
were moved to package_retention.go to keep their durability implementation isolated.

2026-09-09 final additional-fix evidence: `go run ./tools/repoctl check` and
`go test -race ./...` passed (worker15.948s, CLI44.877s); actual native two-worker
CLI restart fixture passed (46.995s). Tests detecting UI recovery/evidence defects failed
against the old executor and pass after correction. Independent review found
that a legacy duplicate package requires synchronizing its winning file inode,
not merely the discarded staged copy; the new failure-injection regression and
full worker race passed (10.774s). No remaining independent-review blockers.

Decision (2026-09-09, implementation/review): preserve terminal action state and
report automatic staging failure separately as evidence unavailable; a separate
artifact operation retries available evidence without repeating the action.
Retained packages use native publication barriers before effects, including legacy
duplicates. Physical power-loss behavior is not claimed as experimentally proved.
All 40 PR checks on `9707fed` passed, including native Windows/macOS/Linux.

CI follow-up: `90ea6c3` push Verify34336686257 macOS Go1.26 failed upload-deadline
with io.ErrClosedPipe. Unlike the earlier graceful-EOF fixture race, explicit body
Close can beat transport cancellation bookkeeping in production. A deterministic
RoundTripper reproduces the lost context error before the fix (0.002s). Join the
caller context error with the transport error on unsuccessful upload; retain both
causes rather than weakening cancellation assertions or retrying CI unchanged.

Cancellation correction validation: full repoctl check passed; client race suite
passed five repetitions (16.897s). Independent read-only review found no blocker.
