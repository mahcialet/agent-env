---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# Add Android Emulator resources to environment leases

This ExecPlan is a living document. Maintain it according to
`docs/PLANS.md`.

Expected branch: `feat/android-emulator-lease`. This plan is the authority for
Android resource work on this branch. Starting revision: `7f1e42c`; the completed
MVP was already merged, and its behavior is regression scope, not new work.

## Purpose / Big Picture

After this work, `agent-env` can allocate one isolated Android Emulator
resource to an environment lease.

A user or coding agent can create an emulator-backed lease, inspect its
ADB serial and observed state, reconcile the lease against the actual
emulator process, and release or quarantine it safely.

The observable workflow is:

    agent-env create . --stack android-runtime
    agent-env show <lease-id>
    agent-env doctor <lease-id>
    agent-env destroy <lease-id>

Two simultaneous leases must never receive the same writable AVD state,
ADB serial, console port, or emulator slot.

This plan does not build, install, or launch a Flutter application.
Flutter integration and UI observation are separate follow-up plans.

## Scope

In scope:

- Android SDK and Emulator prerequisite discovery
- AVD template selection
- exclusive emulator slot allocation
- console and ADB port allocation
- emulator startup and boot readiness
- resource persistence in SQLite
- live inspection and reconciliation
- conservative cleanup and quarantine
- Windows, macOS, and Linux behavior
- fake-adapter and native integration coverage

Out of scope:

- Flutter build
- APK installation
- adb reverse
- UIAutomator interaction
- screenshots
- logcat collection
- hardware-device support
- remote emulator hosts
- malicious-code isolation

## Progress

- [x] 2026-09-07 UTC: Inspected domain/runtime/app/CLI/store and existing MVP.
      RuntimeProvider and allocation currently assume Compose; Reserve already
      uses immediate fenced transactions; execx bounded commands kill descendants.
- [x] 2026-09-07 UTC: Added indexed Android product contract and design document.
- [x] 2026-09-07 UTC: Selected private per-lease writable AVDs with durable port pairs.
- [ ] Implement Android SDK and Emulator discovery.
- [x] 2026-09-07 UTC: Durable slot/port reservation, migration 003, quarantine
      retention and immutable identity implemented; SQLite Go 1.26 tests and
      Go 1.27 race tests passed, including concurrent independent connections.
- [ ] Implement start, inspect, stop, and destroy operations.
- [x] 2026-09-07 UTC: Integrated Android create/destroy compensation and dispatch.
- [x] 2026-09-07 UTC: Added live missing/degraded and identity quarantine behavior.
- [x] 2026-09-07 UTC: App tests with real SQLite prove distinct concurrent AVDs,
      port pairs and serials, sibling survival, timeout compensation and forced
      cleanup refusal. Go 1.26 focused suite x20 and Go 1.27 race x10 passed.
- [ ] Verify supported native platforms or record explicit platform gaps.
- [ ] Complete acceptance evidence and retrospective.

A checked item means observed completion, not intention. Add the UTC date
and evidence when checking an item.

## Surprises & Discoveries

- 2026-09-07 UTC: Host has writable `/dev/kvm` but no Android SDK environment,
  adb/emulator/SDK manager executables, standard SDK directories or system images.
  Real Emulator tests are externally unavailable until these are provisioned;
  fake native tests and cross-builds will be reported separately.
- 2026-09-07 UTC: Existing OSRunner intentionally kills process descendants after
  completion. Reusing it to background an Emulator would violate both lifecycles;
  a separate injected execx detached-process boundary is necessary.
- 2026-09-07 UTC: Migration 002 already records command cancellation; Android
  reservation migration must be 003, preserving legacy databases.
- 2026-09-07 UTC: Independent review found cancellation could arrive after
  console identity inspection but before kill/deletion, and inherited ADB server
  settings could inspect a remote device with the same serial. Adapter now binds
  sockets to context, gates effects and explicitly selects the local ADB server;
  deterministic regressions pass. Review confirmed these findings resolved.
- 2026-09-07 UTC: Adapter-only tests missed app's required `lease` ownership
  metadata; integration review added it and removed the adapter's private boot
  wait so app's documented readiness deadline remains authoritative.
- 2026-09-07 UTC: Root PID disappearance cannot prove an Emulator launcher left
  no QEMU descendants. Detached tree observation is being strengthened before
  declaring cleanup acceptance. No parent-only proof will authorize deletion.
- 2026-09-07 UTC: Initial full harness encountered an unformatted in-progress
  adapter test file during parallel editing; no check was relaxed. Focused app,
  CLI, config and harness tests passed after integration; full rerun remains.

Record unexpected emulator, AVD, path, locking, process, or platform
behavior here. Include the failing command or test name and the resulting
design consequence.

## Decision Log

- Decision: Use private per-lease AVD configuration/state and transactionally
  reserve even console ports 5554..5682 plus adjacent ADB ports. Preserve all
  reservations during quarantine. Rationale: isolates writable state without
  requiring operators to pre-provision a finite pool; port scans cannot own slots.
  Date/Author: 2026-09-07 / implementation.
- Decision: Keep AndroidProvider separate from Compose and add a distinct execx
  detached API. Stop verifies AVD name and sends kill over the same console
  connection; durable native process birth identity detects recycled PIDs.
  Rationale: avoids coupling to Compose or Flutter and prevents serial/port ABA
  cleanup. Unknown launch identity stays quarantined. Date: 2026-09-07.
- Decision: Manifest runtime type is `android-emulator`, with `source` and `avd`;
  Android components omit compose_services. Doctor gains runtime selection and
  lease diagnostics while existing Compose defaults remain. Rationale: explicit
  opt-in preserves existing manifests. Date: 2026-09-07.

- Decision: Treat an emulator as a lease-owned resource, not as an
  independently managed utility.
  Rationale: Its serial, writable state, process, connectivity, and cleanup
  must participate in the same lifecycle as worktrees and Compose projects.
  Date/Author: 2026-09-07 / maintainers.

- Decision: Do not add Flutter behavior in this plan.
  Rationale: Emulator ownership and reconciliation must be independently
  testable before application-specific behavior is introduced.
  Date/Author: 2026-09-07 / maintainers.

## Outcomes & Retrospective

Not completed.

At completion, summarize delivered behavior, native-platform evidence,
known limitations, accepted tradeoffs, and follow-up plans.

## Context and Orientation

The existing environment lifecycle is orchestrated under `internal/app`.
Pure resource and lease types belong under `internal/domain`. External
process invocation uses `internal/execx`. SQLite persistence belongs under
`internal/store/sqlite`.

The Compose runtime under `internal/runtime/compose` is the nearest existing
adapter, but the Android adapter must not import Compose-specific packages.

Read before implementation:

- `AGENTS.md`
- `ARCHITECTURE.md`
- `docs/PLANS.md`
- `docs/PORTABILITY.md`
- `docs/RELIABILITY.md`
- `docs/SECURITY.md`
- relevant product, design, and ADR indexes

## Plan of Work

### Milestone 1 — Contract and resource model

Define the user-visible Android runtime contract and extend the resource
model without introducing Flutter-specific fields.

Observable result:

    agent-env plan . --stack android-runtime --output json

shows the Android resource requirement without starting an emulator.

### Milestone 2 — Discovery and allocation

Discover SDK paths and available AVD templates. Allocate emulator ports and
writable state atomically under the existing operation-lock model.

Observable result:

    agent-env doctor

reports emulator prerequisites accurately.

### Milestone 3 — Lifecycle

Start an emulator, wait for bounded boot readiness, persist stable external
identity, inspect it, and stop it.

Observable result:

    agent-env create . --stack android-runtime

returns a READY lease containing an observed ADB serial.

### Milestone 4 — Reconciliation and cleanup

Detect missing processes, reused ports, mismatched AVD identity, expired
leases, and failed cleanup.

Observable result:

manually terminating an emulator causes reconcile to mark the lease
DEGRADED rather than READY.

### Milestone 5 — Cross-platform and concurrency evidence

Run fake-runner coverage on every supported OS and real emulator integration
where suitable acceleration and SDK images are available.

Observable result:

two concurrent leases receive distinct resources and destroying one leaves
the other usable.

## Concrete Steps

1. Inspect the current resource and runtime interfaces.
2. Add product and design documentation before public CLI behavior.
3. Add domain types and SQLite migration.
4. Implement an injected Android command adapter.
5. Add deterministic allocation and operation fencing.
6. Integrate lifecycle compensation.
7. Add reconciliation and quarantine.
8. Add native-platform and real-emulator evidence.
9. Run repository harness checks.
10. Update acceptance evidence and move this plan to completed.

## Validation and Acceptance

| ID | Required behavior | Evidence |
| --- | --- | --- |
| A1 | Missing Android SDK is reported as a prerequisite failure. | Pending |
| A2 | Missing AVD template is reported without partial allocation. | Pending |
| A3 | A created lease records AVD, serial, port, and observed state. | Pending |
| A4 | Two leases never share writable AVD state or ports. | Pending |
| A5 | Emulator boot timeout triggers compensation. | Pending |
| A6 | Cleanup uncertainty results in quarantine. | Pending |
| A7 | Manual emulator termination is detected by reconcile. | Pending |
| A8 | Destroying one lease leaves a sibling lease unchanged. | Pending |
| A9 | Windows/macOS/Linux path and argv handling has native evidence. | Pending |
| A10 | Repository harness and race tests pass. | Pending |

## Idempotence and Recovery

Allocation must use fenced reservations rather than scanning ports and later
assuming ownership.

A failed create compensates only resources whose ownership is proven.
Ambiguous emulator identity or dirty writable state is quarantined rather
than force-deleted.

Re-running reconcile must be safe. Re-running destroy against a fully
released lease must not affect unrelated emulator processes.

## Artifacts and Notes

Store compact evidence references in this plan. Put large emulator logs,
test outputs, and screenshots in the configured artifact location rather
than embedding them into the plan.

Record:

- tested revision
- host OS
- Android Emulator version
- system image identifier
- acceleration mode
- command exit status
- CI or local evidence distinction

## Interfaces and Dependencies

Expected interfaces:

    AndroidProvider.Validate(...)
    AndroidProvider.Plan(...)
    AndroidProvider.Create(...)
    AndroidProvider.Inspect(...)
    AndroidProvider.Destroy(...)

Do not expose raw subprocess management to the application layer.
The adapter uses the existing cross-platform command runner and returns
stable domain observations.

Expected external tools:

- Android SDK command-line tools
- Android Emulator
- adb

No POSIX shell may be required.
