---
status: active
owner: maintainers
last_verified: 2026-09-08
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
confirmed before writable deletion. On Unix, surviving group members after root
exit prevent cleanup but do not prove lineage for a stop; observation reports
uncertainty until the group is absent. Windows uses a private helper per resource
to retain the named Job handle across CLI/root exit. It inherits explicit handles,
exits when all Job members stop and persists matching empty-Job evidence. A lost
Job name without that evidence, or observation from a different Windows logon
session, is uncertainty. Confirmed manual termination degrades;
uncertain ownership quarantines. Reconcile never adopts or restarts an Emulator.
A released runtime's old ports convey no continuing ownership.

## Shared ADB server lifetime

The local ADB server on port 5037 is a shared SDK prerequisite. Before launching
an Emulator, the adapter reads the SDK client's protocol version with `adb version`
and directly probes `127.0.0.1:5037` using the read-only `host:version` protocol.
An incompatible, malformed or unobservable existing server fails the prerequisite
without attempting replacement. A refused connection identifies an absent server;
when startup is needed, it invokes `adb -L tcp:localhost:5037 start-server` through
the detached-process API, separately from the Emulator's native containment.
A bounded readiness wait repeats the compatibility probe before Emulator launch.
Server startup has its own `adb-server.stdout.log`, `adb-server.stderr.log` and
`adb-server-start.json` beside the retained runtime evidence. Its identity is not
stored as the Emulator's process identity. Lease compensation, destroy and GC
never stop the shared server, including one started during a failed allocation.

Boot observation checks the shared protocol again before invoking
`adb -H 127.0.0.1 -P 5037 -s <reserved-serial> shell getprop sys.boot_completed`.
Inherited server-routing and serial variables are cleared. This client socket
form avoids auto-start when the server disappears. The direct compatibility
check is also necessary: [ADB's version-mismatch path can kill an existing server
even with `-H`](https://android.googlesource.com/platform/packages/modules/adb/+/9084198a2d4b0f6a0f174260fb42da33485b684d/client/adb_client.cpp#311).
A missing shared prerequisite degrades readiness; observation does not repair it.

This separation matters on Windows: [ADB daemon startup uses
`DETACHED_PROCESS`](https://android.googlesource.com/platform/packages/modules/adb/+/9084198a2d4b0f6a0f174260fb42da33485b684d/adb.cpp#938),
which detaches its console but does not escape inherited Job membership.
[Windows Job rules](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects)
would otherwise put an auto-started server inside a bounded command's terminating
Job, or leave it counted as a surviving member of an Emulator's Job.

## Portability and evidence

Native argv, explicit paths/environments and no shell/CGO are required. SDK and
image architecture compatibility remain host prerequisites; WSL is a Linux host.
Tests cover discovery, unsafe templates, console ownership, detached lifetime,
PID reuse, concurrent reservations, compensation, sibling isolation and quarantine.
Native CI and cross-builds are distinct from actual accelerated Emulator tests.
Actual SDK integration has been exercised on Linux; real Windows/macOS SDK,
acceleration and shared-server startup behavior remain unverified.
The [ExecPlan](../exec-plans/completed/android-emulator-lease.md) records evidence,
implementation decisions, unresolved prerequisites and platform gaps.
