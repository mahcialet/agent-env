---
status: accepted
owner: maintainers
last_verified: 2026-09-09
---

# One controller authority and whole-lease worker assignments

[日本語](0006-single-authority-multi-host.ja.md)

## Context

The local registry already owns conservative resource lifecycle, operation fences
and evidence. Remote placement needs global scheduling and transport recovery
without treating missed heartbeats as resource absence or weakening those local
rules. Local mode must remain daemon-free. The
[ExecPlan](../exec-plans/completed/multi-host-control-plane.md) records this
contract and the completed acceptance evidence within its documented scope.

## Decision

Use a separate controller SQLite authority and outbound workers over enrolled-role
mutual TLS with versioned typed JSON. Schedule one whole lease onto one worker.
Reuse its global canonical ULID locally. Persist controller/host/host-instance and
assignment epoch before local effects. Keep epoch fixed for all operations on an
assignment, and make the metadata immutable across normal registry saves.
Ordinary local mutation and GC cannot bypass controller management, even with
force or expiry. Worker operations carry the exact assignment tuple and retain
the existing app/local operation fences.

### Delivery and cleanup authority

Persist operation identity and payload on both sides. Duplicate delivery recovers
a journaled result; uncertainty after possible effect start requires reconciliation
and never authorizes blind mutation replay. Results become durable locally before
artifact delivery. Global RELEASED requires worker cleanup/absence proof.
Heartbeat loss marks OFFLINE/stale/UNKNOWN without reassigning live or uncertain
resources. Friendly host names cannot transfer ownership to a different instance.

### Verified transfer and local endpoints

Transfer committed, independently verifiable Git bundles and evidence through a
bounded SHA-256 CAS, with 1 GiB source-object and 64 MiB artifact limits. Caller
paths never become storage authority. Keep worker-local endpoint semantics and
worker-local environment resolution explicit. See the
[design](../design-docs/multi-host-control-plane.md) and
[product contract](../product-specs/multi-host-control-plane.md).

## Alternatives and consequences

Extending local SQLite into a distributed database would blur cleanup authority.
Heartbeat-based failover can duplicate live resources. Per-operation epoch
increments would confuse assignment identity with delivery identity. Retrying
mutations after lost responses can duplicate input and test effects. Splitting a
lease across workers requires networking and distributed cleanup beyond this slice.

A single controller is simpler but is not HA or consensus; independently copied
active controller databases are unsafe. Persistent journals and digest-addressed
transfers add storage and recovery work, and stale leases can require explicit
reconciliation rather than automatic replacement. These costs preserve uncertainty
and avoid pretending to provide exactly-once external effects.

Native Windows/macOS/Linux role tests and real TLS two-worker integration are
required evidence. Same-host role processes and cross-builds do not establish
physical/VM multi-host operation. The ExecPlan must retain those distinctions and
all unverified prerequisites; this ADR accepts the design, not a completion claim.
