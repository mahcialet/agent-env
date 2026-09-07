---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Architecture

[日本語](ARCHITECTURE.ja.md)

The system materializes pinned local Git sources and a selected component closure into an environment lease with Compose and/or Android Emulator resources. The [MVP specification](docs/product-specs/agent-env-mvp.md) defines behavior; the [completed plan](docs/exec-plans/completed/agent-env-mvp.md) records delivered boundaries and verification evidence.

The CLI parses arguments and formats output, then delegates use cases to app. Domain types model leases, immutable source sets, components, resources and events without concrete adapters. Config strictly decodes the manifest; stack resolves deterministic dependency closure. App coordinates source and runtime interfaces, policy, readiness, evidence, and compensating cleanup.

SQLite owns desired state, reservations, ownership, source identity and event history. Git source providers resolve refs, create detached worktrees, inspect tracked changes and remove safe worktrees. Compose runtime adapters validate normalized configuration, create selected services, inspect resources, collect evidence and destroy by explicit project identity. Reconciliation compares registry intent against Git and Docker observations; it never equates a stored ready row with a live healthy environment.

## Dependency direction

- Domain must not import CLI, SQLite, Git or Compose adapters.
- App may use domain and interfaces; it must not depend on CLI formatting.
- Runtime adapters must not import CLI.
- Store implements persistence and must not own app orchestration policy.
- CLI delegates lifecycle behavior to app; concrete wiring belongs at the application boundary.

These boundaries must be checked by repoctl arch-check with negative fixtures; no boundary is claimed mechanically enforced before that validator passes. Update this map, its checker, and an ADR or plan decision together when changing the graph.

## Cross-cutting invariants

Use argument arrays and platform-native paths, with Windows wrapper handling isolated in execx. Release builds require no CGO or shell. State lives outside target repositories under OS-native state paths, overridden by AGENT_ENV_HOME. SQLite is local, with foreign keys, busy timeout and verified WAL.

Allocation is a saga across separate authorities. Save intent before effects, record results, and compensate in reverse order. Dirty tracked sources, ambiguous identities, or incomplete cleanup remain quarantined with events and artifacts. Lease isolation prevents accidental collisions; it is not a malicious-code sandbox.

The development harness is separate from the target manifest: [agent instructions](AGENTS.md), indexed docs, plans, repoctl and CI describe this repository; `.agent-env.yaml` describes target-repository startup.

Android Emulator resources extend Compose through a separate `app.AndroidProvider` and `internal/runtime/android` adapter. The adapter uses domain identities, app observations and `execx` native process boundaries; it never imports Compose. SQLite owns exclusive AVD/port reservations, and app owns compensation and readiness. Detached processes are distinct from bounded command process trees. See the [Android design](docs/design-docs/android-emulator.md) and [completed execution evidence](docs/exec-plans/completed/android-emulator-lease.md). Browser and Flutter integration remain separate work.
