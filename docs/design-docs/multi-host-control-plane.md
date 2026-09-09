---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Single-authority multi-host coordination

[日本語](multi-host-control-plane.ja.md)

The [product contract](../product-specs/multi-host-control-plane.md) defines
required behavior. The [ExecPlan](../exec-plans/completed/multi-host-control-plane.md)
records completed implementation and acceptance within the documented scope. This design does not
claim unexecuted protocol, integration or native checks have passed.

## Authorities and dependencies

A separate controller SQLite database owns controller identity, enrolled clients
and hosts, liveness, capabilities, capacity, global leases, assignments, operation
journals and blob references. The worker's existing registry owns source/worktree
identity, reservations, concrete runtimes, local operation fences, evidence and
cleanup. Controller scheduling does not import concrete runtime adapters. Worker
wiring reuses app and the existing providers; Android, Flutter and Browser/CDP
retain their existing independent responsibilities.

One same-state-root process lock protects the active controller. This is not
consensus: separately copied controller databases must not run concurrently.
Stable host ID is an operator label; persistent host-instance ID and enrolled
certificate bind the actual worker. A process incarnation identifies a connection
session without replacing the host instance or its assignments.

## Transport and request authority

Standard-library HTTP/TLS carries versioned typed JSON. Mutual TLS authenticates
explicitly enrolled client/worker certificate roles; payload identity must match
that enrollment. Incompatible protocol/product versions are unschedulable before
effects. Private keys are filesystem configuration, not database payloads.
Registration, heartbeat, long-poll, source download and result upload all originate
from the worker. No inbound worker RPC listener or generic shell endpoint is needed.

Every operation carries controller ID, global lease ID, host ID, host-instance ID,
assignment epoch and operation ID. Epoch identifies an assignment and remains
fixed for its operations. The worker uses that exact tuple as app authority; no
operation silently adopts a different controller or increments the assignment.
The same global ULID names the local lease and its resource/evidence paths.
Management metadata is saved at reservation before effects and cannot be removed,
added to an existing local lease or changed by ordinary registry Save.

Local app operations reject missing/mismatching authority before effects and
again under the existing operation fence. Destroy checks before cancellation of
running commands as well as before cleanup. Ordinary GC excludes managed leases,
including expired ones. Reads may expose cached metadata without reconciliation.

## Scheduling and uncertainty

Registration reports truthful semantic capabilities (`git`, `compose.docker`,
`compose.podman`, `android-emulator`, `flutter-android`, `persistent-process`,
`browser-cdp`) and simple capacity. Scheduling atomically reserves capacity for a
whole lease on one compatible ONLINE, enrolled, non-draining worker. Explicit
host selection retains every eligibility check; ranking is deterministic.

A missed heartbeat changes observation freshness to stale/UNKNOWN and the host
to OFFLINE. Placement remains durable. Only a pre-effect rejection with evidence
that no effects started may permit rescheduling. Dispatch ambiguity, upload
failure and worker silence never do. A different instance cannot adopt an assigned
host name. Reconnect and worker restart reconcile the same local resources;
controller restart reloads its original authority and journal. Drain blocks new
placements, and removal refuses hosts with unreleased assignments.

## Native execution and WSL boundaries

A worker owns processes in its native OS only. Windows computed execution
locations use the 240-UTF-16-unit compatibility envelope before effects. On WSL,
state homes on DrvFS or 9p mounts are rejected before directory/database creation;
filesystem/mount checks include canonical aliases and custom locations. This is
a conservative support boundary, not a claim of observed corruption. The guard
applies to CLI state roots, not arbitrary internal database openers or read-only
source locations. Windows likewise rejects WSL UNC state/execution locations,
including direct extended-UNC names and resolved aliases.

Direct PE executables are refused on non-Windows hosts, and Windows refuses the
`wsl.exe` entry point. Checks precede process start and detached output creation;
Git bundle commands use the same check. These guards prevent accidental direct
interop, not transitive execution by trusted scripts. Run separate native workers
for Windows and WSL; never share their state roots. See PORTABILITY for evidence
limits and path requirements.

## Journal and delivery ordering

For create, the worker records `effect_started` immediately before the app reserves
the lease. Read-only validation and provider diagnostics before that boundary can
fail with durable no-effect evidence. A later destroy can release that assignment
only when the journal proves all prior operations never crossed the boundary.
After any reservation attempt, even an error without a local row is uncertain.

Persist operation receipt and payload identity before dispatch/effects. The worker
records preparation, possible effect start, local result and delivery state.
Duplicate identical delivery consults that journal; conflicting payloads with the
same ID fail. A crash after possible effect start is uncertain and requires
reconciliation, not blind replay of create, test or UI/browser input.

Persist the local result before uploading artifacts. Commit central result/blob
references before acknowledging delivery. Retry missing uploads or result delivery
by durable identity; never rerun an action to reconstruct an artifact. Global
RELEASED requires affirmative worker cleanup/absence proof, not HTTP success or
expired controller TTL. Controller outage leaves workloads and journals intact.

### Initial operation dispatch policy

Workers dispatch one operation at a time; already-created leases continue running
concurrently. The controller rejects a second active operation on the same lease,
including destroy during an active remote test. Remote cancel-active and parallel
operation dispatch require a separate protocol and are not implemented. This
policy avoids racing operation identity and local fences while retaining concurrent
live leases. Inspect the current request with `operation <operation-id>` or wait
for its outcome before submitting another operation.

## Source and artifact CAS

The client resolves every source alias to an immutable commit and prepares a
verifiable Git bundle with a SHA-256 identity. The worker validates the digest,
commit, manifest and selected closure independently before local runtime startup.
Client absolute paths are neither CAS keys nor remote filesystem authority.
Uncommitted files are excluded; unsupported shallow/LFS/submodule inputs fail
explicitly without implicit network fetch. Worker environment placeholders are
resolved locally; client secrets are not implicitly forwarded.

CAS keys are `sha256/<digest>`. Stream to private temporary files with size and
hash checks, then publish atomically. Concurrent identical uploads are safe and
existing objects are reused only after verification. Keep durable references and
never collect referenced objects. Bound each source object to 1 GiB and artifact
to 64 MiB. Transfer retry is separate from mutation replay. Encryption at rest is
not claimed; source and evidence retain the trusted administrative data boundary.

## Evidence and limits

Typed remote test/UI/browser paths retain worker-local lease fences, process and
device ownership, stale references, secret redaction and evidence bounds.
Returned loopback endpoints refer to the worker; there is no implicit client tunnel.
HA, migration, split-host resource graphs, remote shells, secret distribution and
break-glass adoption remain separate designs.

Acceptance must distinguish deterministic state-machine tests, actual TLS socket
integration with two worker state roots, native Windows/macOS/Linux role execution,
and physical-machine/VM multi-host evidence. A same-host test or cross-build cannot
substitute for those last two categories. The ExecPlan records exact results and
remaining evidence limits; implementation acceptance is complete within that scope.

## Durable lifetime and sensitive-input boundaries

The additive `lease_lifetimes` table stores controller deadlines and the automatic
cleanup operation ID. Existing leases are backfilled from durable create and
completed renew timestamps. Startup, the one-second server sweep and worker polls
queue expiry cleanup transactionally. Queued/dispatched work defers cleanup;
failed or uncertain automatic cleanup is not blindly retried as a new mutation.
Only successful renew resets the deadline and cleanup marker. Sweep database
errors stop the server visibly instead of silently disabling expiry enforcement.

The shared protocol guard examines UI/browser JSON tokens before persistence,
including duplicate and case-variant text fields. It rejects transient text and
`set-text`; it never redacts a payload and then executes altered input. Existing
journals are not rewritten. A non-persistent input protocol is future work.

After the create effect boundary, failures remain uncertain even if reservation
returned no lease. A compensated create may include a released local lease in its
result payload, but does not assert the destroy-only cleanup confirmation field.
An explicit destroy establishes authoritative release, including after recovery.
