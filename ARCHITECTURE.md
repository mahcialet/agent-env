---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Architecture

[日本語](ARCHITECTURE.ja.md)

This map explains which layer owns a lease operation and where to make a change.
agent-env resolves pinned Git sources and a selected component closure into owned
Compose, Android Emulator, or persistent-process resources. Applications and UI
observation use those resources through separate interfaces. Explicit remote mode
adds placement and transport around worker-local lifecycle management.

The [product specifications](docs/product-specs/index.md) define current behavior;
[design documents](docs/design-docs/index.md) explain individual mechanisms.
The [original MVP plan](docs/exec-plans/completed/agent-env-mvp.md) records the
initial implementation, not the full current feature set.

## Local operation flow

| Layer | Responsibility |
| --- | --- |
| CLI | Parse arguments, format output, and delegate use cases to app. |
| Domain | Model leases, immutable source sets, components, resources, and events without concrete adapters. |
| Config | Strictly decode the manifest. |
| Stack | Resolve a deterministic dependency closure. |
| App | Coordinate source/runtime interfaces, policy, readiness, evidence, and compensating cleanup. |


SQLite owns desired state, reservations, ownership, source identity and event history.

Git source providers resolve refs, create detached worktrees, inspect tracked changes and remove safe worktrees.

Compose runtime adapters validate normalized configuration, create selected services, inspect resources, collect evidence and destroy by explicit project identity.

Reconciliation compares registry intent against Git and each recorded runtime provider’s observations; it never equates a stored ready row with a live healthy environment.

## Dependency direction

- Domain must not import CLI, SQLite, Git or Compose adapters.
- App may use domain and interfaces; it must not depend on CLI formatting.
- Runtime adapters must not import CLI or other runtime adapters.
- Store implements persistence and must not own app orchestration policy.
- CLI delegates lifecycle behavior to app; concrete wiring belongs at the application boundary.

These boundaries must be checked by repoctl arch-check with negative fixtures; no boundary is claimed mechanically enforced before that validator passes. Update this map, its checker, and an ADR or plan decision together when changing the graph.

## Cross-cutting invariants

Use argument arrays and platform-native paths, with Windows wrapper handling isolated in execx. Release builds require no CGO or shell. State lives outside target repositories under OS-native state paths, overridden by AGENT_ENV_HOME. SQLite is local, with foreign keys, busy timeout and verified WAL.

Allocation is a saga across separate authorities. Save intent before effects, record results, and compensate in reverse order. Dirty tracked sources, ambiguous identities, or incomplete cleanup remain quarantined with events and artifacts. Lease isolation prevents accidental collisions; it is not a malicious-code sandbox.

The development harness is separate from the target manifest: [agent instructions](AGENTS.md), indexed docs, plans, repoctl and CI describe this repository; `.agent-env.yaml` describes target-repository startup.

## Compose services

`internal/runtime/compose.Client` dispatches to private Docker and Podman clients
within the same package boundary. Both use the common policy and canonical JSON
snapshot. Podman children enter a native bridge in the current agent-env binary
with recorded engine arguments; no shell wrapper or Python dependency enters the
core. Domain runtime snapshots retain provider identity and `cleanup_evidence`;
app persists pre-down ownership evidence so interrupted cleanup can resume without
reconstructing deleted container attachments. See the
[provider design](docs/design-docs/compose-providers.md) and its completed
implementation evidence; real Podman Machine infrastructure was unavailable.

## Android Emulator lifecycle

Android Emulator resources use a separate `app.AndroidProvider` and `internal/runtime/android` adapter. The adapter uses domain identities, app observations and `execx` native process boundaries; it never imports Compose. SQLite owns exclusive AVD/port reservations, and app owns compensation and readiness. Detached processes are distinct from bounded command process trees. See the [Android design](docs/design-docs/android-emulator.md) and [completed execution evidence](docs/exec-plans/completed/android-emulator-lease.md). Browser/CDP automation is implemented through its own provider and adapter, described below; it does not share Android lifecycle or UI behavior.

## Flutter application lifecycle

Flutter builds use `app.FlutterProvider` and the independent
`internal/runtime/flutter` adapter. Android package/install/reverse/launch effects
use `app.AndroidApplicationProvider`, implemented by the existing Android adapter.
Neither adapter imports the other or Compose; app owns their ordering and
compensation. Additive application/build/reverse records use the existing Lease
JSON persistence. See the [Flutter design](docs/design-docs/flutter-android-runtime.md)
and [ADR 0005](docs/adr/0005-separate-flutter-applications.md).

## Android UI observation

Android UI observation uses `app.AndroidUIProvider`, implemented by the same Android
adapter and its optional self-targeting companion in `internal/runtime/android/uihelper`.
App owns selection, stale-reference policy, operation fencing, recovery and artifact
publication. Domain owns serializable UI values; the adapter owns accessibility/ADB
effects and helper identity. Existing `CommandRun` rows provide intent and cleanup
barriers; no SQL migration or target-manifest section is added. `tools/uihelper`
builds the companion explicitly using native tool argv. No runtime adapter imports
another runtime. See the [observer design](docs/design-docs/android-ui-observer.md).

## Persistent host processes

`app.PersistentProcessProvider` and `internal/runtime/process` implement generic
foreground process lifecycle separately from Compose, Android and Flutter. Domain
stores additive immutable launch references and native identity; app persists
intent/identity, reserves ports, resolves common endpoints, coordinates readiness
and compensates under the operation fence. SQLite owns cross-lease TCP reservations.
`execx.ManagedProcess` owns native Start/Observe/Terminate mechanics. Runtime
adapters do not import one another, and no browser behavior enters this boundary.

Each runtime keeps `state/`, `stdout.log`, `stderr.log`, `owner.json`,
`redaction.json` and `launch.json` below
`leases/<id>/process-runtimes/<runtime>/`. Only the private `state/` path is exposed
as `${runtime_dir}`. The launch receipt recovers identity after a registry-save
failure. Independent `redaction.json` stores versioned ownership and secret
fingerprints before native launch, so receipt-write failure does not prevent
bounded diagnostic redaction during compensation. Cleanup
must prove whole-tree absence before deleting mutable state or releasing ports and
worktrees. See the [process design](docs/design-docs/persistent-process-runtime.md)
and its [completed execution evidence](docs/exec-plans/completed/persistent-process-runtime.md);
native Windows/macOS/Linux acceptance passed.

## Browser/CDP observation and control

`app.BrowserProvider` and `internal/browser/cdp` add typed CDP operations above the
persistent-process contract. Config owns explicit browser bindings; domain owns
page/snapshot/identity values. App holds the lease fence, rechecks the process
provider, validates registered snapshot provenance and persists run/artifact
results. The browser adapter owns discovery, WebSocket target sessions, CDP
identity and stale-node checks. It does not import concrete runtime adapters or
own process/profile/port lifecycle. CLI provides concrete wiring. The transport
uses gorilla/websocket without a Node/Python helper. See the
[browser design](docs/design-docs/browser-cdp-automation.md) and
[completed implementation evidence](docs/exec-plans/completed/browser-cdp-automation.md).

## Multi-host control plane

`internal/controlplane` owns global placement and transport persistence, without
importing app, worker, local store, source materialization, or concrete runtime
adapters. Its protocol package contains only serializable wire values.
`internal/worker` coordinates durable remote receipts and app interfaces; CLI
wires concrete providers. `internal/remotesource` verifies committed Git bundles
and normalized plans separately from controller scheduling. `internal/blobstore`
and `internal/instance` are independent storage primitives. Repoctl checks these
import boundaries with positive and negative fixtures.

The assignment epoch stays fixed for a lease's placement. Operation IDs fence
individual commands. Worker-local management metadata is immutable; controller
transport state never replaces local cleanup evidence. Implementation and native
acceptance evidence are recorded in the [completed multi-host control-plane
ExecPlan](docs/exec-plans/completed/multi-host-control-plane.md), including the
documented support boundaries and unverified environments.

## Standalone release boundary

`internal/buildinfo` exposes executable identity; `internal/assets` owns generic
digest-verified materialization, without Android or Flutter lifecycle behavior.
The current CLI embeds no runtime companion assets. The Android UI helper remains
an explicitly built external input. `tools/repoctl` owns Git release validation,
CGO-disabled cross-builds, archive normalization, checksums, manifest validation
and extracted native smoke tests. Release metadata does not belong in lease/domain
models. GitHub Actions orchestrates these commands and publishes validated bytes;
it does not implement a second packaging algorithm. See the
[distribution design](docs/design-docs/standalone-distribution.md).

## Changing a boundary

Read [AGENTS.md](AGENTS.md) and [plan policy](docs/PLANS.md) before implementation.
The [reliability policy](docs/RELIABILITY.md) defines recovery guarantees; the
[quality guide](docs/QUALITY.md) identifies which checks and native evidence
validate them. A successful build alone does not prove runtime acceptance.
