---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Reliability and recovery

[日本語](RELIABILITY.ja.md)

## Allocation and cleanup

Allocation is a saga across SQLite, Git, and selected runtime tools. The registry reserves the lease, immutable source identities, unique worktree paths, and runtime projects before materialization. Events precede external actions. A validated, ownership-labeled normalized Compose snapshot, generated dynamic loopback endpoint bindings, and digest are saved before startup, so even an unsuccessful `up` command has a recorded cleanup identity.

A failure triggers bounded reverse compensation using a cleanup context that survives ordinary request cancellation. A completely cleaned failed allocation remains recorded as released with failure events and artifacts. Uncertain ownership or incomplete cleanup leaves a quarantined reservation visible for recovery. Cleanup never erases evidence to make a retry appear successful.

Before deleting resources, cleanup inspects pinned source identity and tracked changes. Final runtime logs are saved before down; clean worktrees are removed only after runtime cleanup. Explicit force requires tracked-diff evidence and still refuses identity mismatches. Untracked files in a managed review worktree are disposable. Repeated cleanup is idempotent when the recorded resources are already absent.

Compose cleanup uses the provider and engine stored in the runtime snapshot.
Before Podman down, app persists `Runtime.cleanup_evidence`, including native
anonymous-volume identity and attachment proof. This survives interruption after
containers disappear. Recovery revalidates volume identity and current references;
only a proven residual with no external/sibling reference can be removed. Changed
identity, unavailable observation or incomplete cleanup quarantines the lease.
Provider down succeeding is not proof of absence; actual resources are reinspected.

## Registry and concurrent operations

SQLite uses embedded numbered migrations, verified WAL, foreign keys, and a busy timeout on connections. Immediate transactions enforce capacity and unique runtime project reservations. Normalized rows and the corresponding lease snapshot update together. A versioned `leases/<id>/environment.json` descriptor provides secondary diagnostic evidence and is refreshed during persisted lifecycle changes; SQLite remains authoritative if descriptor writing fails. The registry remains local to one host; do not place it on an unsupported network filesystem or treat it as multi-host coordination.

Lifecycle changes, named tests, renewal, reconciliation, and applied GC hold a durable lease operation lock. Its token expires after a crashed process stops renewing it. Operation contexts carry lock ownership to both external commands and registry mutations; renewal failure cancels work, and transactional fencing rejects stale writers. A lost lock must not become permission for the old process to continue compensation against a successor's resources.

Portable cancellation terminates managed command descendants through Unix process groups or Windows Job Objects. This bounds named tests and probes; it does not reverse arbitrary side effects already performed by repository code. A command canceled after an external allocation still requires observed recovery.

Destroy requests cancellation of the exact active named-run IDs, then waits up to ten seconds for confirmed process termination and final evidence before transferring the lease lock into cleanup. A newer run is not canceled by an older request. An unrelated active lease operation remains busy. If cancellation is unconfirmed, no resources are removed. A durable `running` record without an active operation owner blocks destroy, including force, until explicit reviewed recovery establishes what happened.

A terminal run status is a completion gate, not just the command's exit status. Failure to confirm process-tree termination, flush captured streams, preserve the run descriptor and declared artifacts, or persist final evidence leaves the registry run `running`. This deliberately blocks destroy and GC. Typed unconfirmed-termination failures remain distinguishable from ordinary command failure or cancellation; neither a zero exit code nor a generic canceled result proves that descendants stopped. Retained local logs and descriptors are recovery evidence, not permission to overwrite a newer owner's registry state.

## Live observation

List/show compare registry intent with registered Git worktrees, pinned commits, Compose project labels, concrete resource identities, service health, and bounded read-only HTTP probes. `list --cached` explicitly skips observation. Command probes execute during creation and are not rerun as a side effect of listing.

Missing or unhealthy resources degrade an active lease; unavailable inspection remains unknown with diagnostics. An expired healthy runtime may retain observed state `ready`, but reconciliation reports its expiration timestamp. A durable unfinished command row is reported as running with unverified process completion; an old heartbeat adds a stale-run diagnostic. Reconciliation does not invent a terminal run status. Released resources that reappear and dirty or mismatched sources quarantine the lease. Orphan discovery is observational: unrelated or ambiguously owned resources are reported without deletion. Evidence about unknown resources is preserved rather than replaced with an empty success result.

## TTL and garbage collection

The built-in TTL is 4 hours, maximum 24 hours, and capacity 8 active reservations. Renewal moves expiry relative to now and updates heartbeat; it does not repair missing runtime state. Failed or quarantined allocations retain capacity until cleanup is verified released.

`gc` is a read-only candidate preview. Default policy requires expiration at least five minutes ago and the last heartbeat at least one minute ago. A durable command-run row with status `running` blocks preview and apply even after its old operation lock expires. `gc --apply` acquires the lease lock and rechecks expiry grace, heartbeat grace, lifecycle state and running records before checking resource identity and tracked cleanliness. Renewal or a new command after preview is respected. In-progress and quarantined leases are excluded.

The policy model supports `GCGrace` and `HeartbeatGrace`; the CLI currently uses the built-in five-minute/one-minute values. There is no automatic artifact-expiry policy or general host-resource prune. Artifacts remain until explicitly managed by the operator.

## Recovery workflow

1. Use `show`, `reconcile`, and retained logs to identify what is known, missing, dirty, or ambiguous.
2. Restore the recorded provider/engine or other prerequisite before retrying observation.
3. Preserve wanted tracked changes outside the managed lease. Use ordinary destroy for clean resources; use explicit force only when discarding tracked edits is intended and diff evidence can be retained.
4. Reconcile again and require absence of the lease's runtime/worktree resources before considering cleanup complete.

Do not infer that work stopped from a stale plan, lock record, or observation timeout. Inspect the actual process and resource identities. Never repair an ambiguous lease with blanket Docker/Podman cleanup or Git worktree pruning across unrelated repositories.

## Pinned manifest provenance

Plans and leases record the selected absolute, symlink-resolved `manifest_path`, the owning control checkout's HEAD as `manifest_commit`, and `manifest_modified`. The control commit is independent of runtime source `--ref` overrides. Dirty, untracked and ignored manifest files set modified true. A manifest outside a Git checkout has an empty commit, modified true, and a diagnostic; its stored canonical snapshot and digest are authoritative. Relative source repository paths remain relative to the supplied control repository even when the selected manifest is elsewhere.

## Android recovery

SQLite reserves private AVD identities and even/odd console/ADB port pairs before startup. An ownership marker records launch intent and native process birth identity outside disposable AVD state. Cleanup verifies AVD identity and sends kill on the same authenticated console connection, then requires process-tree and port absence before deleting private writable state. Logs and markers remain evidence. A missing launch identity, recycled resource, or uncertain descendant observation quarantines the lease and retains reservations; force cannot override it.

Reconciliation of an individual Android-only lease does not require Docker. Global Compose orphan inventory examines the union of installed provider executables and recorded provider/engine identities, even when only Android lease rows remain. An installed but unavailable engine produces a visible partial error, not an empty successful inventory; resource IDs are provider-scoped. Android inspection is scoped to recorded identities. Reconcile observes manual termination but never adopts or restarts an Emulator.

The shared local ADB server has a separate lifetime from each lease. Creation
establishes protocol compatibility at `127.0.0.1:5037` with a direct read-only
`host:version` probe before launching an Emulator; an absent server is started
separately through the detached-process API. A mismatched, malformed or
unobservable existing server causes a prerequisite failure without replacement.
Boot observation repeats the compatibility guard before running the SDK client;
it reports a missing shared prerequisite instead of starting or replacing it.

Startup diagnostics and identity remain in `adb-server.stdout.log`, `adb-server.stderr.log` and
`adb-server-start.json` beside the runtime evidence. A later allocation failure
does not authorize stopping this host service. Destroy and GC remove only owned
Emulator resources; they never issue global ADB server cleanup. Recovery must
restore the shared prerequisite without treating its process as a lease-owned
Emulator or discarding failed-start evidence.

## Persistent process recovery

Process creation fixes source commit, runtime paths and named TCP reservations
before launch intent is persisted. Returned native identity is saved before
readiness. A nonzero identity returned with an error is still cleanup evidence;
`launch.json` can recover the launch when the later registry save fails. Never
repeat an uncertain Start as if it had no effects. Later independent CLI calls
observe native identity and bounded HTTP health; root exit degrades the lease and
does not automatically restart it.

Destroy revalidates native ownership before signaling and again before escalation.
The foreground root is required on Unix when live descendants remain; ambiguous
group lineage quarantines instead of authorizing a numeric group kill. Windows
uses the exact owned Job. Whole-tree absence gates private-state deletion, port
release and ordinary tracked-change-protected worktree cleanup. An uncertain
launch, termination, output-evidence failure or lost fence retains recovery state
and reservations. Repeated destroy/GC never override that proof requirement.

Runtime files live under `leases/<id>/process-runtimes/<runtime>/`; mutable `state/`
is deleted after confirmed cleanup, while logs/launch identity remain diagnostic
evidence. Independent `redaction.json` persists versioned ownership and secret
fingerprints before native Start, allowing redacted compensation logs even when
writing `launch.json` fails. A later CLI does not depend on the current host secret
environment. A typed `ErrProcessNotStarted` plus zero PID permits recovery to
prepared state; an untyped zero-identity failure remains uncertain. See the
[process lifecycle design](design-docs/persistent-process-runtime.md).

## Browser operation recovery

All browser operations retain the lease fence through CDP work and evidence
finalization. Active unexpired ready leases permit input; degraded leases permit
only read-only diagnostics with proven native identity. Quarantine, dead or
ambiguous processes and unfinished command runs block operations. Port reuse,
changed browser/page/document/node identity and truncated snapshots never authorize
input. Mutations attempt once and are not automatically replayed after disconnect.
Unconfirmed completion or failed evidence finalization retains a running command
barrier; inspect the actual outcome and evidence before reviewed recovery.
Destroy/GC still requires generic process-tree absence before deleting profiles,
with no browser-specific cleanup or auto-restart. See the
[browser design](design-docs/browser-cdp-automation.md).

## Controller and worker recovery

The [control plane](design-docs/multi-host-control-plane.md) adds a separate global
SQLite authority; each worker retains its local resource registry and fences.
The assignment's controller/host/host-instance/epoch tuple remains fixed across
operations. Worker and controller journals retain payload identity, possible
effect start, local result and delivery state. Duplicate delivery may recover or
upload an existing result; it never blindly repeats an uncertain mutation. Local
results precede artifact uploads, and global RELEASED requires worker cleanup
proof. Loss of a heartbeat or controller connection makes observations stale or
UNKNOWN and never authorizes reassignment, cleanup or a change of host instance.

Workers initially dispatch operations serially; already-created leases continue
running concurrently. The controller rejects a second active operation on one
lease, including destroy during a remote test. Remote cancel-active is deferred
until a separate cancellation protocol can preserve operation identity and fences.
Wait for or inspect the current operation with `operation <operation-id>` rather
than assuming another destroy can interrupt it. Controller/worker restart must
reuse their original state roots; copying a controller DB is not safe failover.
Drain only blocks new placements. Local GC skips controller-managed expired
leases; force never overrides assignment ownership or uncertain cleanup.
