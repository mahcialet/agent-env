---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Android Emulator PR review follow-up

## Purpose / Big Picture

Address PR #2 review findings without changing Android ownership or portability guarantees. Expected branch: `feat/android-emulator-lease`. This plan is the execution authority for the review follow-up; the completed Android implementation plan remains historical evidence.

## Progress

- [x] 2026-09-08: Reconciled clean branch at `90ae8e7` with four PR findings and completed implementation.
- [x] 2026-09-08: Fixed global inventory and runtime-specific destroy preview with regressions.
- [ ] Separate readiness budgets with mixed-runtime regressions.
- [x] 2026-09-08: Validated runnable SDK tools and acceleration before allocation with regressions.
- [ ] Run harness, relevant integration and native CI; reply to and resolve addressed threads.

## Surprises & Discoveries

The global inventory optimization suppresses unrelated Compose orphans when only Android rows survive. SDK Doctor already performs checks absent from Validate.

## Decision Log

- 2026-09-08, implementation agent: Keep global discovery independent of registry runtime kinds; Android lifecycle still uses its existing separate provider.
- 2026-09-08, implementation agent: Use a focused follow-up plan rather than reopen historical completed acceptance.

## Outcomes & Retrospective

Pending implementation and validation.

## Context and Orientation

Findings concern `internal/app/reconciliation_inventory.go`, `destroy_preview.go`, `lifecycle.go`, and `internal/runtime/android/discovery.go`. Android adapter owns SDK validation; app owns readiness and allocation ordering.

## Plan of Work

Implement each finding with behavior-focused regression coverage, inspect combined changes, validate locally, commit and push without rewriting history, then reply and resolve the four review threads.

## Concrete Steps

Use supported Go 1.26.8 from `/tmp/agent-env-toolchains/go1.26.8/go/bin` on PATH. Run targeted app/Android tests, `go run ./tools/repoctl doctor`, `go run ./tools/repoctl check`, and `go run ./tools/repoctl test-integration`. Inspect PR #2 checks after push.

## Validation and Acceptance

Require Compose orphan visibility with Android-only rows, accurate effect-free Android dry runs, successful delayed Android readiness despite short satisfied Compose probes and failure of overdue Compose readiness, and prerequisite failure before reservation for unusable tools/acceleration. Record concrete results below as observed. Native CI remains distinct from real SDK execution.

## Idempotence and Recovery

Preserve quarantine/ownership checks and shared ADB lifetime. Tests use private fixtures. Failed validation is corrected without weakening checks. Retain evidence on uncertain real cleanup; never stop unrelated emulators. Normal commits and push only.

## Artifacts and Notes

PR review comments: 3951260551, 3951260553, 3951260556, 3951260560. Both new inventory/preview regressions failed against the original code (missing orphan and Compose-only diagnostic), then passed after fixes. `go test ./internal/app -run 'TestInventory|TestAndroid' -count=1` passed; `repoctl doctor` passed with Go 1.26.8 on Linux.

## Interfaces and Dependencies

No new dependencies or architecture edges. Keep SDK commands behind execx native argument arrays and Android separate from Compose and Flutter.

SDK evidence: new real app/SQLite regression reproduced missing emulator/acceleration preflight and late adb failure before the fix. Go 1.26.8 Android package and prerequisite/Doctor regressions repeated 20 times passed; Go 1.27.1 Android package race tests passed. Shared read-only 15-second native SDK probes keep Doctor and Validate aligned; no ADB startup is added to validation.
