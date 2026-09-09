---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Single-authority multi-host coordination

[日本語](multi-host-control-plane.ja.md)

The [product contract](../product-specs/multi-host-control-plane.md) defines
required behavior. The [ExecPlan](../exec-plans/active/multi-host-control-plane.md)
tracks implementation and acceptance still in progress. This design does not
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

## Journal and delivery ordering

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
remaining gaps; implementation verification is currently pending.
