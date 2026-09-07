---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Android review second follow-up

## Purpose / Big Picture

Resolve seven new PR #2 findings on dedicated branch `feat/android-emulator-lease`. This plan is the execution authority for the second review follow-up. Prior completed plans remain historical evidence.

## Progress

- [x] 2026-09-08: Inspected clean HEAD `6c92486` and seven unresolved threads.
- [ ] Validate repository-selected AVDs in doctor and expose retained Android process logs.
- [x] 2026-09-08: Rejected incompatible image architectures and shared ADB prerequisites before reservation.
- [x] 2026-09-08: Rejected overlap with immutable Android inputs and portable case-folding runtime collisions.
- [ ] Continue independent reverse runtime cleanup after a failure, retaining sources and quarantine.
- [ ] Validate harness, regression/race tests and native CI; commit/push; reply and resolve threads.

## Surprises & Discoveries

Prior read-only SDK tool checks did not validate architecture or shared ADB compatibility. Previous real SDK smoke runs failed conservative rootless-group cleanup; this historical limitation remains unproven in cause and must not be hidden by this work.

## Decision Log

- 2026-09-08, implementation agent: Split work along existing CLI, Android adapter and app lifecycle boundaries; retain ownership checks and source cleanup barriers.

## Outcomes & Retrospective

Pending.

## Context and Orientation

Seven comments concern `internal/cli/lifecycle.go`, `internal/runtime/android/discovery.go`, `internal/app/lifecycle.go`, and `internal/store/sqlite/android.go`. Expected behavior is defined by Android product/design docs and repository portability/security constraints.

## Plan of Work

Add focused regressions and fixes for each finding, inspect integrated changes, run native tests/harness, push coherent commits and address each thread with evidence.

## Concrete Steps

Use Go at `/tmp/agent-env-toolchains/go1.26.8/go/bin` and `/tmp/agent-env-toolchains/go1.27.1/go/bin`. Run relevant package tests, `go run ./tools/repoctl doctor`, `check`, `test-integration`, and race suite. Inspect PR #2 checks after push.

## Validation and Acceptance

Doctor fails for invalid selected templates; active/released Android logs are readable/redacted and component-isolated; invalid architectures and ADB servers fail before allocation; immutable input overlap and portable case collisions fail before allocation; cleanup preserves uncertain runtimes but releases independently owned siblings without removing sources on incomplete cleanup. All addressed threads receive replies and resolution only after fixes and validation.

## Idempotence and Recovery

No force or ownership weakening. Preserve user assets, shared ADB server and evidence. No history rewriting. Retain any failed real SDK fixture until normal cleanup confirms ownership and termination.

## Artifacts and Notes

Comments: 3952452797, 3952452807, 3952452813, 3952452817, 3952452823, 3952452826, 3952452828. Evidence pending.

## Interfaces and Dependencies

Preserve CLI/app/adapter/store boundaries, native argument arrays and no shell/CGO core dependency. No Flutter behavior.

Validation checkpoint: Android adapter full unit package, new architecture/prerequisite tests repeated ten times, and Go 1.27.1 Android race tests passed. Nine real app/SQLite rejection cases verify zero reservations/materialization/process starts; architecture matrix covers 24 combinations. Real supplied SDK metadata matches required x86_64 ABI (read-only check only). App/config path overlap and portable name regressions passed five repetitions. Name collision regression failed against original config validation. Path comparisons include other runtimes' inputs, existing symlink aliases and future paths without creating them.
