---
status: completed
owner: maintainers
last_verified: 2026-09-08
---

# Android Emulator PR review follow-up

## Purpose / Big Picture

Address PR #2 review findings without changing Android ownership or portability guarantees. Expected branch: `feat/android-emulator-lease`. This plan is the execution authority for the review follow-up; the completed Android implementation plan remains historical evidence.

## Progress

- [x] 2026-09-08: Reconciled clean branch at `90ae8e7` with four PR findings and completed implementation.
- [x] 2026-09-08: Fixed global inventory and runtime-specific destroy preview with regressions.
- [x] 2026-09-08: Separated readiness budgets with four mixed-runtime regressions.
- [x] 2026-09-08: Validated runnable SDK tools and acceleration before allocation with regressions.
- [x] 2026-09-08: Ran harness, Docker integration, race suite and native CI; replied to and resolved all four addressed threads. Real SDK smoke failures are recorded below, not counted as passes.

## Surprises & Discoveries

The global inventory optimization suppresses unrelated Compose orphans when only Android rows survive. SDK Doctor already performs checks absent from Validate.

## Decision Log

- 2026-09-08, implementation agent: Keep global discovery independent of registry runtime kinds; Android lifecycle still uses its existing separate provider.
- 2026-09-08, implementation agent: Use a focused follow-up plan rather than reopen historical completed acceptance.

## Outcomes & Retrospective

All four PR review findings are fixed and covered by regression tests. Changes are committed and pushed without history rewriting; all four threads have responses and are resolved. Local harness, Docker integration, race suite and all 24 CI checks on implementation commit `5d80521` passed.

Additional real SDK smoke testing reached two READY devices but twice failed first-device cleanup due to a rootless live process group whose ownership could not be established. The destruction/containment implementation is unchanged by this review follow-up. This remains an unresolved real SDK lifecycle limitation, not a successful end-to-end result or a reason to weaken quarantine. Windows/macOS real SDK execution was not tested.

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

Intermediate validation: `repoctl check` passed (Go 1.26.8, Linux). First `repoctl test-integration` exited 1 although the three real CLI Compose scenarios shown passed; tool output truncation omitted the failing detail. Rerunning with complete output retained at `/tmp/agent-env-review-integration.log` before classifying the failure.

Real Android run: both leases reached READY using SDK37.1.11/KVM12; first destroy quarantined after45s because the detached leader was absent but a live group remained uncertain. Test failed (91.14s) and retained `/tmp/agent-env Android integration 日本語 215162662`. Investigating read-only before any recovery; ownership checks remain unchanged.

Readiness evidence: Go 1.26.8 full app package passed; Go 1.27.1 mixed-readiness and existing timeout regressions with race detector repeated five times passed. Each runtime retains its own deadline; observations and poll sleeps use the earliest still-pending deadline so slow Android inspection cannot postpone a failing Compose budget. Satisfied runtimes continue ownership/health observation. The Android-first test initially exposed fixture ordering by component name, corrected without production scope changes.

Final code validation at `5d80521`: Go 1.26.8 `repoctl check` passed; Go 1.27.1 `go test -race ./...` passed. The earlier race run overlapped the Android-first fixture ordering correction and failed that assertion; the final run used the corrected fixture. Docker `repoctl test-integration` rerun passed with full retained log. All four PR threads received commit/test-specific replies and were resolved via GitHub API. Native CI pending.

Recovery evidence: after both recorded native groups and ports became absent, the original synthetic-source fixture was reconstructed in a temporary tagged test and standard `Service.Destroy(false, false)` retried. No direct SQL mutation, force, manual process kill, or ownership bypass. Both original rows independently re-read as desired/observed `released/released`, both markers `destroyed`, private AVDs removed and logs retained. Temporary recovery test removed. Repeating the real SDK acceptance on final code in isolation to separate repeatability from the earlier concurrent heavy test load; no cause for the transient live group is proven.

Independent review: separate agents found no concrete issues in inventory/preview and SDK preflight. SDK reviewer independently repeated prerequisite/Doctor/Validate tests three times (passed). Root inspected the combined readiness implementation and full validation; readiness author's own review is not described as independent.

The isolated real SDK rerun also failed the first Emulator destroy (81.35s) with the same rootless live-group uncertainty after both devices reached READY. Evidence: `/tmp/agent-env-review-android-final.log`, `/tmp/agent-env Android integration 日本語 380983221`. This is not a passing real SDK end-to-end acceptance; no timeout/ownership rule was relaxed. The four review changes do not modify detached-process identity or Android destruction. Investigating retained state to distinguish safe final cleanup from the failed sibling-lifetime scenario.

Native CI acceptance for implementation `5d80521`: push run 34157770668 and PR run 34157774457 each passed all 12 jobs (24/24 total). Native Windows/macOS/Linux across Go 1.26 and 1.27, five CGO-disabled cross-builds, and Linux race plus real Compose integration passed.

Second-run cleanup evidence: the fixture's deferred ordinary Destroy calls all succeeded (no cleanup errors), after which its root was removed by the fixture. Independent inspection found the root absent, no relevant SDK processes and ports 5554–5557 refused. No persisted second-run rows remain to independently reread. Shared crash/netsim helpers are a hypothesis only; the transient group member was not captured. The first failed run's retained DB/logs remain available. No additional recovery was necessary.
