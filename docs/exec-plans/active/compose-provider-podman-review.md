---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Podman provider PR review fixes

[日本語](compose-provider-podman-review.ja.md)

## Purpose / Big Picture

Address PR #8's UDP reachability and optional Compose inventory findings on
`feat/compose-provider-podman`, starting from `4800a1f`. This plan is the execution
authority for this review follow-up; the completed provider plan retains its
original implementation evidence.

## Progress

- [x] 2026-09-08: Read both unresolved threads and reconcile the implementation.
- [x] 2026-09-08: Reproduced both findings, fixed protocol filtering/native inventory,
  and passed full provider regression tests, repository check, and full race tests.
- [ ] Validate shared Docker traversal, Podman behavior, harness and native CI.
- [ ] Push fixes, reply to each thread, resolve addressed threads, and archive this plan.

## Surprises & Discoveries

The engine-only InventoryDoctor test stopped before Inventory itself, whose shared
Docker traversal invoked Compose ls. Remote endpoint inspection also discarded UDP
mappings because every mapping used a TCP connection attempt. Latest baseline CI
34217217740 succeeded after rerunning the Windows lock-loss test failure.

## Decision Log

- 2026-09-08, implementation: Limit TCP reachability checks to TCP endpoint keys.
  UDP dial success cannot establish application readiness; retain engine-observed
  UDP mappings and existing service/readiness checks without claiming UDP probing.
- 2026-09-08, implementation: Share native labelled-resource traversal separately
  from Docker Compose project listing. Podman inventory must neither execute nor
  require a Compose frontend, including when the recorded executable has moved. Bare project-label presence filters must also map
  to the Podman native label so project-only orphans are discoverable.

## Outcomes & Retrospective

Pending implementation and validation.

## Context and Orientation

`internal/runtime/compose/podman.go` owns remote endpoint checks and Podman
inventory delegation. `inventory.go` shares labelled container/network/volume
traversal. `podman_review_test.go` will exercise full provider entry points.

## Plan of Work

Reproduce both defects with runner-backed provider tests and native local sockets.
Keep provider identity, ownership checks, and Docker project discovery unchanged.
Update English and Japanese provider contracts with the clarified protocol and
inventory behavior. Validate, push, and respond to both original review threads.

## Concrete Steps

Run `go test ./internal/runtime/compose`, `go run ./tools/repoctl check`,
`go test -race ./...`, and `go run ./tools/repoctl test-integration`. Run the gated
Podman/Docker coexistence suite when shared traversal changes are complete.
Inspect final native CI and record exact results here.

## Validation and Acceptance

- Remote UDP mappings remain present; reachable TCP remains usable and unreachable
  TCP still marks the observation unready.
- InventoryDoctorFor followed by InventoryFor observes native labelled resources
  without Compose, including a stale Compose executable path.
- Native identity pinning, error handling, and Docker inventory behavior retain
  their existing tests; complete harness and relevant integration pass.
- Each original review thread has a concrete fix/evidence reply and is resolved.

## Idempotence and Recovery

Work only on the existing feature branch. Preserve published history. Tests use
isolated fixtures; cleanup only test-owned resources. Do not resolve a thread
until its fix is pushed and validated.

## Artifacts and Notes

Review comments: discussion_r3957283152 (UDP), discussion_r3957283162 (inventory).
Before the fix, `TestPodmanRemoteInspectPreservesUDPAndChecksTCP` lost UDP mappings
and `TestPodmanInventoryWithoutComposeDiscoversNativeOrphans` failed with missing
or stale Compose executables. Both now pass, including reachable/unreachable TCP,
all three native resource kinds, conflicting labels, and partial inventory failure.
An independent agent reviewed the production diff and ran provider race tests.
`go run ./tools/repoctl check` and `go test -race ./...` passed. Native Podman/Docker
coexistence and existing Docker integration are running; CI/push/thread actions
remain pending.

## Interfaces and Dependencies

No new dependencies or public fields. Native Go subprocess/socket APIs preserve
Windows, macOS, and Linux portability. Android/Flutter boundaries remain intact.
