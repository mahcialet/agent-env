---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Multi-host control plane

[日本語](multi-host-control-plane.ja.md)

This document defines the required multi-host contract. Implementation and final
acceptance verification are in progress in the
[ExecPlan](../exec-plans/active/multi-host-control-plane.md). A requirement here
is not a claim that its integration or native acceptance test has passed.

## Modes and placement

Local mode remains the default: existing commands use local SQLite and providers
without a daemon. Explicit remote mode sends typed requests to one active
controller. A whole lease, including all selected sources, components and
runtimes, runs on exactly one assigned worker. No remote shell API is provided.

The controller schedules only enrolled, compatible, ONLINE, non-draining hosts
with the required capabilities and capacity. Capability names include `git`,
`compose.docker`, `compose.podman`, `android-emulator`, `flutter-android`,
`persistent-process` and `browser-cdp`. A requested host bypasses ranking only;
it cannot bypass enrollment, compatibility, capability or capacity checks.
Drain blocks new placements without migrating or destroying existing leases.

## Authentication and ownership

Production requests use HTTPS with mutual TLS and explicitly enrolled certificate
roles. A client certificate authorizes client operations; a worker certificate
authorizes its enrolled host and instance, not arbitrary host IDs in a request.
Workers initiate registration, heartbeats, polling and transfers outbound. They
require no inbound worker listener. Private keys remain filesystem inputs and
are never stored in controller or worker SQLite.

The controller owns global placement and requested lifetime. The worker owns
local process/container/device identity and cleanup evidence. The global lease
ID is the same canonical ULID used in the worker registry. Its controller ID,
host ID, persistent host-instance ID and nonzero assignment epoch are durable
before source materialization or runtime startup. Assignment epoch is fixed
across operations on that assignment; an operation ID identifies an individual
request. A new operation never advances the epoch implicitly.

Ordinary local mutation, reconciliation, test, UI/browser operations and GC cannot
alter a controller-managed lease. Local force is not an authority bypass.
Read-only access returns recorded management metadata; it does not silently
reconcile a lease without assignment authority. Managed lifetime expiry or a
controller outage alone never authorizes local cleanup.

### Initial operation dispatch policy

Workers dispatch one operation at a time; already-created leases continue running
concurrently. The controller rejects a second active operation on the same lease,
including destroy during an active remote test. Remote cancel-active and parallel
operation dispatch require a separate protocol and are not implemented. This
policy avoids racing operation identity and local fences while retaining concurrent
live leases. Inspect the current request with `operation <operation-id>` or wait
for its outcome before submitting another operation.

## Sources and evidence

Only committed Git inputs are transferred. Every source alias identifies an
exact commit and a verified SHA-256 bundle digest. The worker independently
verifies the transferred manifest, source set and selected stack before runtime
effects. Client absolute paths never become worker paths or CAS keys. Unsupported
shallow, LFS or submodule cases must be rejected explicitly; no implicit network
fetch fills missing source data. Remote `${env:NAME}` resolves on the worker;
client environment secrets are not forwarded implicitly.

The controller CAS accepts bounded, digest-verified objects: at most 1 GiB per
source object and 64 MiB per artifact. Caller paths never determine storage
locations. A local durable result precedes evidence upload. Failed transfers may
be retried by digest; they do not authorize replaying the operation that produced
the evidence. Repository content and artifacts are private administrative data;
encryption at rest is not claimed.

## Operations, uncertainty and recovery

Remote operations cover create, list/show, renew, reconcile, destroy, named tests,
logs/artifacts and typed Android UI and Browser/CDP operations on capable workers.
Existing local operation fences, stale snapshot checks and resource ownership
checks continue to apply. Endpoint-consuming operations execute on the worker:
`127.0.0.1` URLs in results are worker-local, not client-local tunnels.

Both sides journal operation identity and payload. A duplicate operation ID with
conflicting content is rejected. Repeated delivery of the same request returns
or reconciles the durable result rather than repeating a mutation. Connection
loss after dispatch means uncertainty, not proof of no effect. Upload recovery
retries result delivery only. Destroy reaches global RELEASED only after the
worker proves local cleanup/absence under the existing lifecycle rules.

Heartbeat expiry makes the host OFFLINE and lease observations stale/UNKNOWN.
It never proves resource absence or permits automatic reassignment of a live or
uncertain lease. The same worker instance reconnects to the existing assignment.
A different instance cannot adopt a friendly host name while assignments remain.
Host removal is refused while active, stale, uncertain or unreleased assignments
remain. Controller restart retains its identity, assignments and journals.

## Limits and acceptance

This slice assumes one trusted administrative domain and one active controller.
Cloning controller databases does not provide safe failover. HA, live migration,
split-host leases, transparent endpoint tunnels, secret distribution and implicit
break-glass recovery are outside this contract.

Acceptance requires actual TLS socket tests with two distinct worker state roots
and native controller/worker/client execution on Windows, macOS and Linux.
Same-host worker processes prove protocol isolation, not physical multi-host
behavior. Cross-builds do not prove native execution. Physical-machine or VM
coverage, unavailable prerequisites and remaining verification gaps must remain
explicit in the ExecPlan before completion.
