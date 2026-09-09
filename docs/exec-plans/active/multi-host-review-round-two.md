---
status: active
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

- [x] Read the four new threads and reconcile the prior completed fixes.
- [x] Route actual CLI controller serving through the shared expiry lifecycle.
- [x] Separate streaming blob deadlines from bounded metadata requests.
- [x] Serialize the canonical manifest once and retain worker verification.
- [x] Persist CAS directory publication with portable platform support.
- [ ] Validate, push, reply to and resolve all four threads; archive this plan.

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

Implementation and validation are in progress.

## Context and Orientation

CLI remote_services starts services; server.Run owns expiry and shutdown.
client owns HTTP transport, remotesource/worker own package validation, and
blobstore owns CAS file publication.

## Plan of Work

Repair each boundary with fail-before regression coverage and independent review.
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

PR #12 threads: PRRT_kwDOURHsR86gk8fD, PRRT_kwDOURHsR86gk8fG,
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
