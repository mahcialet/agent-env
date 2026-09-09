---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Add a single-authority multi-host control plane

[日本語](multi-host-control-plane.ja.md)

This ExecPlan is a living document. Maintain it according to `docs/PLANS.md`.

Expected branch: `feat/multi-host-control-plane`.

Starting revision: `dc63308e53f68f8be99f7cbf59cafc78f7296b71`
(`master`, PR #11 repository correctness audit merged).

Before implementation, fast-forward `master`, record a newer exact base revision
if HEAD moved, create the dedicated branch, and run the complete repository
baseline.

## Purpose / Big Picture

After this work, `agent-env` can place an entire environment lease onto one of
multiple remote worker hosts while preserving the existing local-first lease
model.

Local mode remains the default and requires no daemon:

```text
agent-env create .
  -> local SQLite
  -> local Git/worktrees
  -> local runtime adapters
```

Multi-host mode is explicit:

```text
authenticated CLI
      |
      | HTTPS / mTLS
      v
+-----------------------------+
| single active control plane |
| controller SQLite           |
| source/artifact CAS         |
+-----------------------------+
      ^                 ^
      | outbound        | outbound
      | heartbeat /     | heartbeat /
      | long-poll       | long-poll
+-----+------+     +----+-------+
| worker A   |     | worker B   |
| local DB   |     | local DB   |
| local Git  |     | local Git  |
| runtimes   |     | runtimes   |
+------------+     +------------+
```

The initial placement unit is the **whole lease**:

```text
one global lease
  -> exactly one worker host
  -> all selected sources/components/runtimes run there
```

This slice does not split one lease across hosts. API-on-host-A plus
Android-on-host-B requires cross-host networking and distributed cleanup, which
remain follow-up work.

Target CLI shape:

```text
agent-env control-plane serve ...
agent-env worker serve --controller https://control.example:7443 --host-id linux-01 ...
agent-env --controller https://control.example:7443 hosts list
agent-env --controller https://control.example:7443 create . --stack mobile
agent-env --controller https://control.example:7443 show <lease-id>
agent-env --controller https://control.example:7443 test <lease-id> e2e
agent-env --controller https://control.example:7443 logs <lease-id>
agent-env --controller https://control.example:7443 destroy <lease-id>
```

Exact spelling may change in Milestone 1. The authority and failure semantics
below are normative.

## Core Product Invariants

### Local mode stays daemon-free

With no controller configured, current local behavior remains unchanged. No
network service or worker process is required.

### One lease equals one worker in this slice

Compose endpoints, process loopback ports, ADB/Emulator, Flutter reverse
mappings and Browser/CDP remain worker-local. Remote typed operations execute on
the worker.

### OFFLINE does not mean absent

A missed heartbeat proves communication loss only. It does not prove that local
containers, processes, Emulators, worktrees or ports are gone.

An assigned live or uncertain lease is therefore never automatically reassigned
onto another worker after host loss.

### Exactly-once effects are not assumed

Use durable operation IDs, assignment epochs and worker journals. A network
disconnect after dispatch is not proof that a mutation did not occur.

### Typed operations only

Do not add a generic remote shell API. Remote control dispatches existing typed
agent-env operations. Manifest-declared process argv remains allowed because it
is repository execution, not an arbitrary control-plane shell endpoint.

## Scope

In scope:

- single active control-plane mode in the existing binary;
- worker mode in the same standalone binary;
- authenticated remote CLI mode;
- controller-specific SQLite store and migrations;
- persistent controller identity;
- stable worker host identity and persistent host-instance identity;
- worker enrollment and certificate identity;
- HTTPS/mTLS transport;
- versioned JSON protocol;
- outbound worker heartbeat and operation long-poll;
- host registry and ONLINE/OFFLINE/DRAINING states;
- capability and explicit capacity reporting;
- deterministic whole-lease scheduler;
- explicit host override;
- assignment epoch/generation fencing;
- controller and worker operation journals;
- controller-managed worker-local lease metadata;
- protection from ordinary local GC/mutation;
- immutable source transfer from local Git commits;
- digest-verified source/content-addressed storage;
- multiple source aliases;
- central artifact/evidence CAS;
- retryable result/artifact upload without replaying effects;
- stale/UNKNOWN global observation after heartbeat expiry;
- controller restart recovery;
- worker restart/reconnect recovery;
- drain/undrain and safe host removal rules;
- duplicate delivery and result-loss recovery tests;
- two-worker real-socket integration;
- native Windows/macOS/Linux protocol integration;
- remote Docker/Podman/Android/process/Browser capabilities when a worker
  truthfully advertises them;
- bilingual product/design/ADR/ExecPlan docs.

Out of scope:

- active-active controller HA or consensus;
- automatic controller failover;
- live lease migration;
- one lease split across workers;
- overlay networking / endpoint tunneling;
- arbitrary SSH or inbound worker RPC;
- hostile multi-tenant isolation;
- automatic PKI/CA issuance or rotation;
- implicit client-secret forwarding;
- Vault/KMS integration;
- automatic remote Git fetch/authentication;
- unproven Git LFS/submodule transport;
- cross-host writable/fix leases;
- autoscaling or CPU/memory bin packing;
- service installation;
- malicious-code sandboxing.

## Authority Model

### Control-plane authority

Controller SQLite owns:

- global lease ID;
- requested immutable source identities;
- source bundle digests;
- requested stack/components;
- required capabilities;
- selected worker placement;
- assignment epoch;
- global desired state;
- operation IDs/transport state;
- host registry/liveness;
- CAS references;
- last acknowledged worker observation;
- global history.

It does **not** own native PID/container/AVD/CDP absence proof.

### Worker local authority

The existing worker-local agent-env DB continues to own concrete local truth:

- worktrees/mirrors;
- Compose/Podman resources;
- Android AVD/process/port reservations;
- Flutter application identity;
- persistent process identity;
- Browser/CDP process/snapshot evidence;
- local operation fences;
- local cleanup proof;
- local evidence;
- local port reservations.

A controller may not infer local absence from heartbeat loss. A worker may not
infer global reassignment authority.

### Global/local lease identity

Preferred design: controller generates one globally unique lease ID and the
worker uses the same ID for its local materialized lease.

Worker persists additive remote-management metadata:

```text
management_mode=controller
controller_id
host_id
host_instance_id
assignment_epoch
```

Avoid a second opaque worker lease ID unless implementation evidence requires it.

## Host and Controller Identity

Worker identity contains:

- stable operator-selected `host_id`;
- persistent random `host_instance_id` under worker state root;
- enrolled certificate identity/fingerprint;
- current worker-process incarnation;
- agent-env/protocol version;
- OS/arch/capability snapshot.

`host_id` alone is never ownership proof.

A different instance reusing an existing host ID while active/uncertain
assignments remain is rejected or quarantined.

Controller owns a persistent random `controller_id`. Controller-managed leases
record it. A different controller identity cannot silently adopt them.

This is not consensus and does not make two cloned active controller databases
safe.

## Transport and Authentication

Preferred initial transport:

```text
HTTPS + mutual TLS + versioned JSON
```

Prefer Go standard-library `net/http`, `crypto/tls`, `crypto/x509`, and
`encoding/json` unless evidence requires a different protocol stack.

Workers connect outbound only.

Worker loop:

```text
register/authenticate
  -> capability snapshot
  -> heartbeat
  -> long-poll operation
  -> persist receipt/journal
  -> execute or recover
  -> upload result/evidence
  -> acknowledge
  -> repeat
```

Protocol payloads carry:

```text
protocol_version
controller_id
host_id
host_instance_id
global_lease_id
assignment_epoch
operation_id
operation_type
content digests
```

Production non-loopback listeners require TLS and enrolled client certificates.
Certificate private keys remain filesystem inputs, never SQLite values.

A loopback-only insecure mode may exist solely for deterministic tests if it is
strictly gated.

## Source Transport

Workers cannot use client-local repository paths. Preserve committed Git identity
without introducing remote Git credentials.

Preferred flow:

```text
client
  -> validate manifest
  -> resolve each source alias to exact commit
  -> create verifiable Git bundle/object package
  -> SHA-256 bundle
  -> upload missing blobs to controller CAS
  -> submit create request with alias/commit/digest/plan requirements

controller
  -> schedule worker
  -> dispatch immutable assignment

worker
  -> download blobs
  -> verify SHA-256
  -> verify/import Git object package
  -> create private mirror/worktree
  -> verify exact requested commit
  -> read/validate transferred manifest
  -> verify source/plan identity
  -> run existing local lifecycle
```

Client absolute paths never become worker paths or CAS keys.

Untracked/uncommitted files are not transported. Shallow/LFS/submodule cases must
be explicitly validated or rejected; never silently fetch from the network.

## Controller Content-Addressed Store

Controller private CAS stores source bundles and selected central artifacts.

Key form:

```text
sha256/<digest>
```

Requirements:

- atomic temp write + rename;
- streaming digest/size verification;
- bounded object size;
- concurrent duplicate upload safety;
- existing verified blob reuse;
- SQLite references;
- no deletion while referenced;
- private filesystem permissions;
- caller paths never used as filesystem authority.

Repository content and artifacts may contain secrets. Encryption at rest is not
claimed in this slice.

## Capability Inventory and Scheduling

Workers report stable semantic capabilities such as:

```text
git
compose.docker
compose.podman
android-emulator
flutter-android
persistent-process
browser-cdp
```

Exact versions remain diagnostic evidence; scheduler logic uses stable capability
names plus explicit version constraints only where the product contract requires
them.

Initial capacity is deliberately simple:

- max concurrent controller-managed leases;
- optional truthful named slots, e.g. Android Emulator slots;
- operator labels.

No dynamic CPU/memory bin packing claim.

Scheduler filters:

1. enrolled/authenticated host;
2. ONLINE and not DRAINING/QUARANTINED;
3. compatible protocol/product;
4. all required capabilities present;
5. explicit constraints satisfied;
6. capacity available.

Then choose deterministically, e.g. least assigned capacity then stable host ID.
Capacity reservation and placement are one controller transaction.

Explicit `--host` bypasses ranking only, never safety validation.

## Capability Derivation and Preflight

Client derives required capabilities from the immutable selected stack closure.
Worker independently revalidates the transferred manifest/source/plan before
runtime effects.

If worker rejects preflight and proves `effects_started=false`, controller may
release the placement and try another eligible host.

If effects may have started, automatic rescheduling is forbidden.

## Assignment Epoch and Operation Fencing

Every assignment owns a monotonically increasing `assignment_epoch`.

Every remote operation carries exact controller/lease/host/instance/epoch/op ID.
Worker persists that receipt before effects.

Reject:

- wrong controller;
- wrong host/instance;
- stale epoch;
- duplicate operation with conflicting payload;
- operation incompatible with local terminal state.

Duplicate delivery of the same operation ID returns/reconstructs the durable
result; it does not blindly repeat a mutation.

## Worker Operation Journal

Conceptual states:

```text
received
prepared
effect_started
result_pending
completed
uncertain
```

Reuse existing command/evidence tables where that preserves one local lifecycle
authority; do not create redundant truth unnecessarily.

On restart/reconnect, unfinished operations are reported and reconciled against
actual local resources before any retry.

## Controller Operation State

Controller transport state is separate from local resource truth:

```text
queued
assigned
dispatched
running
result_pending
completed
uncertain
```

Worker disconnect does not cause duplicate mutation dispatch.

## Global Liveness and Host Loss

Desired state and observation freshness are separate.

Example:

```text
desired=active
placement=linux-01
last_known=READY
heartbeat=expired

reported observed=UNKNOWN
```

Stored READY is never presented as current health after freshness expires.

When the same worker instance returns, reconcile the same local lease. Do not
place the same lease elsewhere.

## Heartbeat, Drain and Host Removal

Heartbeat includes identity, worker incarnation, versions, capability digest,
capacity usage, and controller-managed lease summary.

`drain` prevents new assignments only. It does not migrate or destroy existing
leases.

Host removal/forget is rejected while active, stale, uncertain or unreleased
assignments reference it.

## Controller and Worker Outage Semantics

During controller outage workers:

- keep existing workloads running;
- accept no new remote mutations;
- do not auto-GC controller-managed leases merely because controller heartbeats
  disappeared;
- retain journals/evidence;
- reconnect later.

Worker restart:

1. reopen existing local DB/state;
2. present same host/instance/controller identity;
3. report local remote-managed leases and unfinished operations;
4. reconcile actual local resources;
5. resume result upload;
6. accept mutations only after fencing succeeds.

## Local Operator Behavior on Workers

Controller-managed leases are visibly marked.

Ordinary local mutating commands and local GC must not silently alter them.
Read-only diagnostics may be allowed.

Do not add an undocumented local `--force` bypass. Break-glass recovery after
irrecoverable controller loss is a separate explicit design problem.

## TTL / Renewal

Controller owns global requested lease lifetime. Controller outage or global TTL
expiry alone does not authorize a worker to destroy resources.

Renew/destroy is dispatched through normal fenced operations. Local GC must skip
controller-managed leases.

## Remote Operation Surface

At completion remote mode supports existing meaningful typed operations:

- create;
- list/show;
- renew;
- reconcile;
- destroy;
- named tests;
- logs/artifacts/evidence;
- Android UI operations on Android-capable hosts;
- Browser/CDP operations on browser-capable hosts.

All endpoint-consuming operations execute on the assigned worker so worker-local
loopback semantics remain valid.

Do not present `127.0.0.1:<worker-port>` as a client-local address. Transparent
endpoint tunneling is future work.

## Results and Artifacts

Local worker evidence is the first durable execution record.

Central result flow:

```text
persist local result
  -> upload blobs by digest
  -> controller verifies bytes
  -> upload/commit result manifest
  -> controller acknowledges operation completion
```

If upload fails after effect success, retry upload only. Do not replay a test,
UI action, browser action or lifecycle mutation merely to regenerate evidence.

## Version Compatibility

Introduce explicit protocol versioning.

Initial conservative policy may require exact protocol version and exact product
version if that materially reduces ambiguity. Incompatible workers remain visible
but unschedulable before effects.

No best-effort schema guessing.

## Security Model

Initial multi-host mode is one trusted administrative domain, not hostile
multi-tenancy.

Target repositories are trusted to run on scheduled workers just as in local
mode. This is not a malicious-code sandbox.

Protect controller DB/CAS, worker state, TLS keys, source blobs and artifacts.

`${env:NAME}` resolves on the worker host. Client environment secrets are not
automatically forwarded.

A future secret-provider design may add explicit secret delivery.

## Progress

- [x] 2026-09-09: Confirmed base `dc63308e53f68f8be99f7cbf59cafc78f7296b71`; created `feat/multi-host-control-plane`.
- [x] 2026-09-09: Run baseline repoctl/docs/race/current integrations.
- [x] 2026-09-09: Write bilingual product/design docs and authority ADR.
- [x] 2026-09-09: Define protocol/version/auth contract.
- [x] 2026-09-09: Add controller store/migrations and persistent controller ID.
- [x] 2026-09-09: Add persistent worker host-instance ID.
- [x] 2026-09-09: Add enrollment/certificate-role validation.
- [x] 2026-09-09: Implement mTLS HTTP foundation.
- [x] 2026-09-09: Add registration/heartbeat/long-poll.
- [x] 2026-09-09: Add capability/capacity snapshot.
- [x] 2026-09-09: Add hosts list/show/drain/undrain.
- [x] 2026-09-09: Implement deterministic scheduler/atomic capacity reservation.
- [x] 2026-09-09: Implement source bundle creation/digests.
- [x] 2026-09-09: Implement controller source CAS.
- [x] 2026-09-09: Implement worker source download/verify/materialize.
- [x] 2026-09-09: Verify manifest/source/stack identity before effects.
- [x] 2026-09-09: Add global lease ID to worker-local create path.
- [x] 2026-09-09: Mark worker leases controller-managed.
- [x] 2026-09-09: Protect controller-managed leases from local mutation/GC.
- [x] 2026-09-09: Implement assignment epoch fencing.
- [x] 2026-09-09: Implement controller operation journal.
- [x] 2026-09-09: Implement worker remote-operation journal.
- [x] 2026-09-09: Implement outbound operation dispatch/reconnect.
- [x] 2026-09-09: Implement remote create/list/show.
- [x] 2026-09-09: Implement renew/reconcile/destroy.
- [x] 2026-09-09: Implement named tests/logs/artifacts.
- [x] 2026-09-09: Implement central artifact/result CAS.
- [x] 2026-09-09: Route Android UI remote operations.
- [x] 2026-09-09: Route Browser/CDP remote operations.
- [x] 2026-09-09: Implement stale/UNKNOWN observation on heartbeat expiry.
- [x] 2026-09-09: Prove no automatic reassignment after worker loss.
- [x] 2026-09-09: Implement controller restart recovery.
- [x] 2026-09-09: Implement worker restart recovery.
- [x] 2026-09-09: Reject same host ID/different instance adoption.
- [x] 2026-09-09: Add drain/remove safety tests.
- [x] 2026-09-09: Add mTLS/auth/body/path/blob negative tests.
- [x] 2026-09-09: Add duplicate-delivery regression.
- [x] 2026-09-09: Add disconnect-after-effect-before-result recovery.
- [x] 2026-09-09: Add controller-outage/live-worker regression.
- [x] 2026-09-09: Add worker-outage/surviving-process regression.
- [x] 2026-09-09: Add two-worker real-socket scheduling/isolation integration.
- [x] 2026-09-09: Add remote process-runtime E2E.
- [ ] Add remote Browser/CDP E2E.
- [ ] Add remote Docker/Podman integration where available.
- [x] 2026-09-09: Add Android-capability scheduling fixture.
- [ ] Run native Windows controller/worker/client integration.
- [ ] Run native macOS controller/worker/client integration.
- [x] 2026-09-09: Run native Linux controller/worker/client integration.
- [x] 2026-09-09: Record physical/VM multi-host evidence separately if available.
- [x] 2026-09-09: Update bilingual architecture/portability/security/reliability/quality/roadmap.
- [x] 2026-09-09: Update standalone docs for controller/worker modes.
- [ ] Run final harness/race/native/integration/release checks.
- [ ] Complete acceptance evidence and bilingual retrospective.
- [ ] Move both plans to `docs/exec-plans/completed/`.

## Surprises & Discoveries

- 2026-09-09: The first real TLS create failed because JSON object key reordering changed the package digest. Source hashes now canonicalize the typed manifest while independently proving the committed control-file transformation; reordered-JSON regression and native Linux E2E passed. Controller global-state mapping also had to use actual lowercase domain states, including `released`, rather than assumed uppercase wire values.
- 2026-09-09: Independent review reproduced preflight diagnostic secret leakage in four fail-before cases. Preflight errors now use inherited-secret redaction and bounded remote metadata. A readable operation ID was accepted by the controller but rejected by the executor; the executor now applies the same operation identity contract while retaining ULID lease identities.
- 2026-09-09: A pre-effect create failure could retain capacity permanently because no local lease existed for destroy. The journal now atomically records its pre-effect boundary. A validated non-dry-run destroy may use exact-assignment journal proof, never a missing local row or an error payload. Restart, dry-run, invalid request, missing evidence and effect-started regressions passed.
- 2026-09-09: First native macOS CI on `0f05d09` failed because the provider compared `/var` and `/private/var` spellings of the same temporary ancestor as distinct repository identities. Linux success did not expose this OS path alias; repair and new native validation are pending. Runs: Verify 34314568570, Multi-host native 34314568601.

- 2026-09-09: Independent runner review found that heartbeat loss during Prepare could still begin effects. Added a check immediately before the durable effect-started transition; a focused regression confirms no effect before reconnect.
- 2026-09-09: Initial remote action tests assumed flags were shared by all UI/Browser actions; the existing CLI registers them per action. Corrected the mapping and tests without changing local flags. The first full harness attempt failed those new tests; focused corrected tests passed. A subsequent harness attempt hit formatting while the controller implementation was being edited; the complete harness must be rerun on a stable milestone.

- 2026-09-09: Baseline unit/vet and race tests passed; `repoctl test-integration` exited 0 with real Docker. Baseline `repoctl check` failed docs-check because the supplied Japanese plan lacked mandatory section headings. Added the missing sections in both languages; remaining native runtime baselines are pending.

Record at minimum:

- local APIs that assume CLI-generated lease IDs;
- local GC assumptions that all local leases are locally managed;
- source logic tied to absolute client repository paths;
- worker-local endpoint display assumptions;
- operations that cannot recover after result delivery loss;
- controller SQLite contention under heartbeat/operation load;
- worker restart while containers/native processes survive;
- Git bundle behavior for merges/shallow/LFS/submodules;
- certificate/path behavior on Windows/macOS;
- clock-skew assumptions;
- HTTP disconnect immediately after mutation;
- duplicate registration / same host ID replacement;
- stale capability snapshots;
- large Android/Browser evidence uploads.

Preserve failed approaches that affect authority/recovery design.

## Decision Log

- 2026-09-09, implementation: Initially serialize worker operation dispatch and reject a second active operation on the same lease. Long-lived leases still run concurrently. Remote destroy does not cancel an active remote test; a separate cancellation protocol is deferred rather than racing an assignment fence.
- 2026-09-09, validation: Physical/VM multi-machine evidence is unavailable. The real TLS fixture uses independent native controller/client/two-worker processes on one host. Record this limit explicitly; native Windows/macOS evidence comes from CI, not cross-builds.

- 2026-09-09, implementation: Keep the assignment epoch fixed for a lease placement; distinct operation IDs fence individual commands. Local SQLite rejects removal, adoption, or alteration of management metadata, including writes under a valid local operation lock.
- 2026-09-09, implementation: Persist a separate worker journal with FULL-synchronous SQLite, a permanent controller binding, a host-instance identity, and a new process incarnation. Result upload and acknowledgement retries never reset effect-started receipts. A native SQLite process lock excludes another worker using the same root and is released by the OS after a crash.
- 2026-09-09, implementation: Use `--controller`, explicit `--tls-ca`, `--tls-cert`, and `--tls-key`, plus `control-plane serve/enroll` and `worker serve`. Provision certificates outside agent-env and enroll leaf fingerprints by client/worker role. Local commands remain unchanged when no controller is selected.

- Decision: initial placement is one whole lease on one worker. Rationale:
  preserve worker-local endpoint and cleanup semantics before distributed runtime
  composition. Date/Author: 2026-09-09 / maintainers.
- Decision: local mode remains daemon-free. Rationale: multi-host is additive,
  not a rewrite of local-first behavior. Date/Author: 2026-09-09 / maintainers.
- Decision: one active controller only; HA/consensus deferred. Rationale: establish
  safe authority/fencing first. Date/Author: 2026-09-09 / maintainers.
- Decision: heartbeat loss never proves absence and never triggers automatic
  reassignment. Rationale: old worker resources may still be live. Date/Author:
  2026-09-09 / maintainers.
- Decision: workers initiate outbound connections. Rationale: avoid inbound worker
  exposure and support common NAT/firewall layouts. Date/Author: 2026-09-09 /
  maintainers.
- Decision: versioned JSON over HTTPS/mTLS is the preferred initial protocol.
  Rationale: typed operations and standard-library implementation preserve
  portability. Date/Author: 2026-09-09 / maintainers.
- Decision: controller/global persistence and worker/local resource persistence
  remain separate authorities. Rationale: local DB/provider evidence remains the
  only concrete cleanup proof. Date/Author: 2026-09-09 / maintainers.
- Decision: source transport uses committed Git identity plus digest-verified
  object/bundle transport. Rationale: support local committed repositories without
  remote Git credentials. Date/Author: 2026-09-09 / maintainers.
- Decision: source/artifact transport is content-addressed. Rationale: retry and
  deduplication rely on exact bytes. Date/Author: 2026-09-09 / maintainers.
- Decision: uncertain mutations are never blindly replayed. Rationale: transport
  loss is not proof of no effect. Date/Author: 2026-09-09 / maintainers.
- Decision: controller-managed worker leases are protected from ordinary local
  mutation/GC. Rationale: local operators must not bypass global assignment
  authority accidentally. Date/Author: 2026-09-09 / maintainers.
- Decision: remote endpoints remain worker-local. Rationale: endpoint tunnels are
  a separate networking problem. Date/Author: 2026-09-09 / maintainers.
- Decision: remote `${env:NAME}` resolves on worker; client secrets are not
  implicitly forwarded. Rationale: avoid an accidental secret-distribution
  system. Date/Author: 2026-09-09 / maintainers.
- Decision: durable docs and this ExecPlan are bilingual. Rationale: repository
  documentation policy. Date/Author: 2026-09-09 / maintainers.

## Outcomes & Retrospective

Not completed.

At completion summarize the final CLI, authority split, protocol/auth model,
source transport, CAS, capability/scheduler model, assignment/operation fencing,
outage behavior, evidence delivery, remote operation coverage, native OS evidence,
real multi-host evidence, limitations, and next steps for split-host leases,
endpoint tunneling, HA and secrets.

## Context and Orientation

Read before implementation:

- `AGENTS.md` / `AGENTS.ja.md`
- `ARCHITECTURE.md` / `ARCHITECTURE.ja.md`
- `docs/PLANS.md` / `docs/PLANS.ja.md`
- `docs/QUALITY.md` / `.ja.md`
- `docs/RELIABILITY.md` / `.ja.md`
- `docs/SECURITY.md` / `.ja.md`
- `docs/PORTABILITY.md` / `.ja.md`
- `docs/roadmap.md` / `.ja.md`
- completed repository correctness audit/review follow-up;
- current source/worktree, Compose/Podman, Android/Flutter/UI,
  persistent-process, Browser/CDP and standalone designs;
- SQLite migrations/store and operation fencing.

Current architecture explicitly describes SQLite and operation locks as local.
This plan adds a higher global authority; it must not reinterpret local SQLite as
some distributed database.

## Plan of Work

### Milestone 1 — Product contract, ADR and authority map

Create bilingual product/design docs and an ADR defining single controller,
whole-lease placement, global/local authority split, outbound workers, no
heartbeat-based failover, and source/CAS transport.

### Milestone 2 — Controller persistence and identity

Add a separate controller store with numbered migrations. Conceptual entities:

```text
controller_meta
clients
hosts
host_sessions
host_capabilities
global_leases
lease_sources
assignments
operations
operation_results
blobs
blob_references
events
```

Add persistent controller ID and same-state-root single-process protection. Do
not claim safety for two separately copied active controller databases.

### Milestone 3 — Worker identity, enrollment and transport

Add persistent host-instance identity, enrollment, mTLS, protocol negotiation,
registration, heartbeat, long-poll, source download and artifact upload.

### Milestone 4 — Capability inventory and scheduler

Convert read-only worker doctor/prerequisite results into stable capabilities and
simple capacity. Implement ONLINE/DRAINING and deterministic placement with
atomic reservation.

### Milestone 5 — Immutable remote source packaging

Create and verify source bundles for exact commits; support multiple repos; reject
corruption/wrong commits and prevent client path leakage. Explicitly handle or
reject shallow/LFS/submodule cases.

### Milestone 6 — Global placement and worker-local create

Controller generates lease ID, chooses worker, persists assignment epoch, and
worker independently verifies all immutable inputs before creating a
controller-managed local lease.

Only pre-effect rejection may be rescheduled automatically.

### Milestone 7 — Durable operation journals

Cover dispatch loss, duplicate receive, disconnect after intent, effect success
before result, upload failure, controller restart, worker restart and late
responses. Mutation replay is forbidden when effect state is uncertain.

### Milestone 8 — Host loss and observation freshness

Heartbeat expiry -> OFFLINE and global UNKNOWN/stale; assignment remains. Same
worker reconnect reconciles same lease. Different instance with same host ID is
rejected while assignments remain.

### Milestone 9 — Remote lifecycle surface

Route list/show/renew/reconcile/destroy/test/log/artifact through the assigned
worker. Controller RELEASED requires worker-local existing cleanup proof, not HTTP
success alone.

### Milestone 10 — Artifact/evidence transport

Persist local result first, then upload digest-addressed artifacts and a result
manifest. Retry upload only.

### Milestone 11 — Android UI and Browser/CDP

Route typed UI/browser operations through both controller assignment fencing and
existing worker-local fences/stale checks. Disconnect after input remains
uncertain until journal reconciliation.

### Milestone 12 — Local worker protections

Ordinary local mutation and GC reject/exclude controller-managed leases. No
undocumented local force bypass.

### Milestone 13 — Two-worker real-socket integration

Run controller, worker A, worker B and client over actual TLS sockets with
separate state roots/DBs. Prove capability selection, load balancing, drain,
offline->UNKNOWN, no reassignment, reconnect, destroy and duplicate delivery.

### Milestone 14 — Native OS evidence

Run real controller/worker/client processes on Windows/macOS/Linux. Same-host
native role tests are native protocol evidence, not physical multi-host evidence.

### Milestone 15 — Real multi-host evidence

Where available, run separate machines/VMs and record host/placement/outage/
reconnect/cleanup honestly. If unavailable, retain the limitation.

### Milestone 16 — Documentation and completion

Update README, Architecture, Portability, Security, Reliability, Quality,
Roadmap, standalone docs and indexes in both languages. Preserve PR #11 escaped-
defect guardrails in the new protocol/state-machine tests.

## Concrete Steps

Use the supported Go toolchain on PATH. Run `go run ./tools/repoctl check`, `go test -race ./...`, and `go run ./tools/repoctl test-integration` at coherent milestones. Add focused journal, transport, source, and process tests before collecting native OS evidence. Record actual results under Progress and Validation and Acceptance.

## Validation and Acceptance

| ID | Required behavior | Evidence |
| --- | --- | --- |
| M1 | Local mode remains daemon-free and behaviorally compatible. | Local full harness/race and existing Docker, Podman, Android, Flutter/UI and Browser integrations passed; final native harness pending. |
| M2 | Multi-host uses one persistent controller identity and one active controller authority. | Controller persistence/restart and native process-lock/crash-release tests passed; real TLS controller restart retained identity. |
| M3 | Host ID + persistent host-instance identity prevents friendly-name adoption. | Controller host-instance rejection tests and native worker restart retained the original identity. |
| M4 | Production transport uses mTLS and rejects unenrolled clients/workers. | Real TLS tests reject unenrolled certificates and worker-as-client calls; role/host/blob ACL negatives passed. |
| M5 | Worker connects outbound; no inbound worker listener is required. | Native fixture runs two outbound worker processes; only controller and workload listeners are needed. |
| M6 | Protocol is versioned and incompatible workers are unschedulable before effects. | Compatibility inventory, upgrade, scheduling/poll denial tests and worker no-effect registration test passed. |
| M7 | Controller/global and worker/local persistence authorities remain distinct. | Separate controller DB, worker journal and local state DB; immutable local management tests passed. |
| M8 | One global lease is placed on exactly one worker. | Native two-worker fixture places both process runtimes of each lease on exactly one worker. |
| M9 | Scheduler requires ONLINE/non-draining/capable/capacity-available host. | Atomic capacity, Android slot, online/capability and deterministic scheduler tests passed. |
| M10 | Explicit host selection cannot bypass safety checks. | Native missing-host/draining-host requests rejected; scheduler negative tests passed. |
| M11 | Source aliases use exact commit + verified bundle/blob digest, not client paths. | Committed multi-alias bundle roundtrip and exact worktree lifecycle passed; paths normalized to source aliases. |
| M12 | Corrupt/wrong source blobs fail before runtime effects. | Wrong digest, corrupt bundle, shallow/LFS/submodule and CAS size/path negative tests passed before effects. |
| M13 | Worker independently verifies source/manifest/stack identity. | Committed control-file normalization proof, self-consistent tamper rejection and reordered-JSON tests passed. |
| M14 | Lease ID and assignment epoch are durable before worker runtime effects. | Managed Create tests prove global ID/authority are reserved before source materialization. |
| M15 | Duplicate operation delivery cannot repeat a mutating effect. | Journal conflict/duplicate tests plus native same-ID named-test retry produce one proof append. |
| M16 | Effect success + result loss recovers from worker journal without replay. | Lost-result receipt recovers uncertain local state without replay; persisted-result retry tests passed. |
| M17 | Artifact upload failure retries upload only. | Runner upload/ack-loss regression retries publication without another execution. |
| M18 | Heartbeat expiry yields stale/UNKNOWN, never resource absence. | Controller offline views report UNKNOWN with last-known state; capacity remains reserved. |
| M19 | Offline live/uncertain lease is never auto-reassigned. | Offline/no-reassignment and conservative host-removal tests passed. |
| M20 | Same worker reconnect reconciles original local lease. | Native worker restart preserves instance, lease, workload PID and process start identity. |
| M21 | Same host ID with different instance is rejected while assignments remain. | Controller foreign-instance registration cannot adopt retained assignments; tests passed. |
| M22 | Controller restart preserves assignment/operation recovery. | Store reopen retains operation IDs/results/epochs; native controller restart retains placement. |
| M23 | Worker restart recovers local resources and unfinished operations. | Worker journal restart/lost-effect tests and native surviving-process restart passed. |
| M24 | Controller outage does not trigger worker auto-GC. | Native controller outage preserves live endpoints; managed GC tests exclude worker leases. |
| M25 | Ordinary local mutation/GC cannot alter controller-managed leases. | Managed mutation/GC/UI/Browser/cancel fencing and immutable SQLite Save regressions passed. |
| M26 | Global RELEASED requires worker-local cleanup/absence proof. | Controller release-proof negatives and journal pre-effect destroy proof (restart/dry-run/invalid input) passed. |
| M27 | Worker-local endpoints are not mislabeled as client-local. | Worker responses mark endpoint_scope=worker-local; native tests inspect worker-owned endpoint metadata. |
| M28 | Test/log/artifact remote operations work without raw shell. | Extended native TLS named-test, idempotency, run/live logs, artifact digest download and no-overwrite passed on Linux/macOS; Windows relative lookup repair under validation. |
| M29 | Android UI remote actions preserve existing local stale/device/fence checks. | Typed remote UI option tests invoke existing app UI boundary; managed stale/device/fence regressions and real local Android UI integration passed. |
| M30 | Browser remote actions preserve existing process/page/snapshot/focus/stale checks. | Real remote Chrome/CDP snapshot/pages and artifact download passed; existing Browser stale/focus/process tests and native integration passed. |
| M31 | Two workers can run isolated concurrent leases; destroying one preserves the other. | Real two-worker fixture proves distinct worktrees/ports and that destroying one lease leaves the other responding. |
| M32 | Drain prevents new placements without migration/destruction. | Real TLS drain/undrain and unit safety tests passed without migration/destruction. |
| M33 | Removing host with active/stale/uncertain assignments is rejected. | Store remove tests reject active/stale/uncertain assignments; confirmed release enables removal. |
| M34 | CAS is atomic, content-addressed, digest-verified and duplicate-upload safe. | CAS concurrent duplicate writers, digest/size validation, atomic directory publication and corruption negatives passed. |
| M35 | Caller paths never become CAS filesystem authority. | CAS digest-only path validation and registered-artifact ownership/path/symlink tests passed. |
| M36 | Client secrets are not implicitly forwarded; remote env resolves on worker. | Native TLS fixture verifies client-only token absence and worker env resolution; negative helper controls passed (18.394s Linux). |
| M37 | Native controller/worker/client integration passes on Windows. | Initial two-worker lifecycle passed in CI 34314956327; expanded named-test and long-path Windows checks remain pending fixes. |
| M38 | Native controller/worker/client integration passes on macOS. | Native multi-host CI 34314956327 and expanded fixture on 34315479224 passed macOS. |
| M39 | Native controller/worker/client integration passes on Linux. | Native multi-host Linux CI passed; extended local TLS fixture and real remote runtime tests passed. |
| M40 | Real-socket two-worker integration proves scheduling/outage/reconnect/recovery/cleanup. | Native real TLS controller/client/two-worker fixture covers placement, outage, restart, recovery and cleanup. |
| M41 | Physical/VM evidence is distinguished honestly from same-host worker tests. | All multi-role fixtures use one physical host; no separate-machine/VM evidence available or claimed. |
| M42 | Existing Docker/Podman/Android/process/Browser local integrations remain non-regressed. | Real local Docker, Podman coexistence, Android Emulator, Flutter/Android UI and Browser integrations passed. |
| M43 | HA/live migration/split-host leases/tunnels are not advertised as implemented. | Product/design/README explicitly defer HA, migration, split-host leases, tunnels and remote cancel-active. |
| M44 | Bilingual durable docs describe authority/trust/failure/recovery boundaries. | Paired product/design/ADR, architecture, README and operational docs passed docs/translation checks. |
| M45 | Final repoctl/docs/translation/race/native/integration/release verification passes. | Local harness/race and real six-target release candidate build/check/native smoke/negative tests passed; final native CI pending. |
| M46 | Both ExecPlans contain direct evidence and retrospective before archival. | Pending final native evidence, retrospective reconciliation and bilingual archive. |

## Idempotence and Recovery

Reads are retryable. Mutations are operation-ID/epoch fenced and are not assumed
idempotent.

On transport uncertainty:

```text
do not issue a new mutation
  -> inspect/reconcile worker journal
  -> recover local effect/result identity
  -> resume result delivery
```

Source/artifact transfers are retryable by digest. Host OFFLINE is never a
cleanup event.

## Artifacts and Notes

Additional acceptance evidence: real remote Browser/Docker/Podman fixture passed in 35.659s; expanded two-worker named-test/log/artifact/renew fixture passed in 18.907s, and client/worker environment isolation in 18.394s. `AGENT_ENV_RELEASE_CANDIDATE=build go test ./tools/repoctl -run '^TestReleaseCandidate$' -count=1 -v -timeout=20m` passed on `b1a7df6`: six real target archives, verification, native smoke and negative tamper tests used an isolated private source/tag fixture; no public tag or release was created. Native multi-host CI 34314956327 passed all three OSes on `d993965`. The expanded fixture passed Linux/macOS on 34315479224; Windows revealed relative executable lookup before child cwd. Full Windows harness additionally exposed a >300-character Git path case. Both failures remain recorded until their targeted native checks pass.

2026-09-09 integrated milestone: commits `6d9dd7a` (managed local authority) and `0f05d09` (controller/worker/source/CLI) pushed to `origin/feat/multi-host-control-plane`. Full `repoctl check` and `go test -race ./...` passed locally. `TestMultiHostNativeCLI` passed in 21.406s with real TLS and built binaries, including placement, role denial, drain, isolation, outage, controller/worker restart and cleanup. Existing Browser native/secret tests, Podman coexistence integration (107.695s), real Android Emulator integration, real Flutter/Android UI integration and Docker `repoctl test-integration` all passed. The first Android invocation failed for missing explicit template configuration; rerunning with the installed stopped template passed. No host-specific prerequisite paths are stored here. Native CI is running; macOS revealed the path alias defect above. Additional remote runtime E2E evidence is still being collected.

2026-09-09 milestone evidence: `go test -race ./internal/worker` passed (1.073s), including receipt replay, failed upload/ack retry, lost-result uncertainty and pre-effect heartbeat fencing. `go test -race ./internal/instance` passed (1.035s), including two native processes and crash-release. Managed app/local-store/domain race tests passed (50.167s/16.235s/1.030s); an overlay using the pre-fix Store.Save reproduced six management overwrite failures. Source/CAS repeated focused race tests passed (5.380s/1.011s). Architecture boundary fixtures passed (0.028s). Native Windows/macOS and actual two-worker TLS acceptance remain pending.

Suggested controller state:

```text
<AGENT_ENV_HOME>/control-plane/
  controller.db
  controller-id
  blobs/sha256/
  logs/
```

Worker keeps existing state plus host-instance/controller-binding/remote-journal
metadata.

Central evidence records controller/protocol/global lease/host instance/epoch/
operation/source digest/worker version/local observed state/artifact digest/ack.

Never persist TLS private keys in SQLite.

## Interfaces and Dependencies

Potential packages:

```text
internal/controlplane/
internal/controlplane/store/
internal/controlplane/protocol/
internal/worker/
internal/remotesource/
internal/blobstore/
```

Control-plane scheduling code must not import concrete runtime adapters. Worker
wiring reuses existing app/runtime providers.

Prefer standard library HTTP/TLS/JSON. Do not add gRPC/protobuf unless real
requirements show the simpler protocol is insufficient.

Multi-host mode introduces long-running controller/worker processes; local mode
still has no daemon requirement.

## Unresolved Issues to Settle During Milestone 1

1. CLI naming: `control-plane` vs `controller`; global `--controller` vs remote/fleet subcommand.
2. Exact protocol/product version compatibility policy.
3. Worker/client certificate enrollment UX and rotation follow-up.
4. Client versus worker certificate authorization mapping.
5. Long-poll protocol details.
6. Cross-platform controller single-instance lock.
7. Git bundle format/shallow repository behavior.
8. Git LFS/submodule policy.
9. Source/artifact max sizes and resumable upload policy.
10. CAS retention/GC policy.
11. Final capability names/version constraints.
12. Capacity model beyond max leases/Android slots.
13. Scheduler tie-break/operator-label UX.
14. Client versus worker final authority for normalized plan digest.
15. Controller-managed local lease metadata storage shape.
16. One generic operation envelope versus typed protocol messages.
17. DEGRADED read-only Android/Browser remote policy.
18. Exact global stale/unknown state presentation.
19. Break-glass local recovery after permanent controller loss.
20. Controller DB/CAS backup/restore guidance.
21. Whether physical two-host evidence is mandatory for completion.
22. Future endpoint tunnel topology.
23. Whether split-host leases need a new global resource graph rather than extending assignment.
24. How PR #11 escaped-defect guardrails are systematically applied to protocol/state boundary tests.
