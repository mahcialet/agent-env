---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# Android Emulator resource design

The [product contract](../product-specs/android-emulator.md) adds a separate
`app.AndroidProvider`, implemented by `runtime/android`. App owns saga ordering,
locks, readiness and persistence. Domain holds pure identities, SQLite holds
exclusive reservations, and `execx` owns native detached process operations.
Android never imports Compose or Flutter. Bounded command trees retain their
existing cancellation behavior; persistent Emulator processes use a distinct API.

## Private AVDs and reservations

Each runtime receives a lease-derived unique AVD name and paths under
`leases/<id>/android/<runtime>/avd/`. The template supplies read-only configuration
and immutable SDK images. Its userdata, snapshots and locks are never shared as
writable state. Unsafe writable paths, symlinks and live templates fail validation.

One immediate SQLite transaction reserves lease/worktree identities, AVD identity
and a console/ADB pair. Console ports are even values from 5554 through 5682;
the following odd port belongs to ADB. Unique constraints arbitrate concurrent
CLI processes. Port probes only detect external conflicts, not ownership, and
never select an unrecorded fallback. Quarantine retains ports and capacity;
only verified `released` state releases reservations.

## Saga and conservative recovery

App persists launch intent before effects. The adapter writes an ownership marker
outside disposable AVD state before launch and records PID plus process birth
identity after launch. A crash before durable launch identity is ambiguous and
requires quarantine. Persistent stdout/stderr are local process diagnostics, not
logcat. Registry and environment descriptors retain reservation and observation.

Stop checks the lease-derived AVD name on the same authenticated console
connection used for `kill`, preventing a check/stop race on a recycled port.
Native birth identity detects PID reuse. Termination of the entire native process group or Windows Job and port absence must be
confirmed before writable deletion. Confirmed manual termination degrades;
uncertain ownership quarantines. Reconcile never adopts or restarts an Emulator.
A released runtime's old ports convey no continuing ownership.

## Portability and evidence

Native argv, explicit paths/environments and no shell/CGO are required. SDK and
image architecture compatibility remain host prerequisites; WSL is a Linux host.
Tests cover discovery, unsafe templates, console ownership, detached lifetime,
PID reuse, concurrent reservations, compensation, sibling isolation and quarantine.
Native CI and cross-builds are distinct from actual accelerated Emulator tests.
The [ExecPlan](../exec-plans/active/android-emulator-lease.md) records evidence,
implementation decisions, unresolved prerequisites and platform gaps.
