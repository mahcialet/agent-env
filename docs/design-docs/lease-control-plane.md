---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Lease control plane

[日本語](lease-control-plane.ja.md)

## Environment

The actual resources that make an application runnable: worktrees, Compose project, containers, networks, volumes, generated configuration, ports, logs, and later emulators or browsers.

## Lease

The control-plane record describing who owns an environment, what immutable source set it represents, why it exists, how long it may remain allocated, and what state its resources are in.

A lease is not merely a PID and not merely a Compose project name.

## Source set

The immutable tuple of repositories and resolved Git commits used by a lease.

Example:

```text
mobile-app@aaaa1111
backend-api@bbbb2222
shared-schema@cccc3333
```

The requested refs must also be recorded, but runtime identity is based on resolved commits.

## Component

A logical participant in the system, such as `api`, `dashboard`, `database`, or `mobile`. Components form a dependency graph and may map to one or more runtime-specific resources.

## Stack

A named startup group consisting of explicit root components. `agent-env` computes the transitive dependency closure.

Examples:

```text
api        -> api
Dashboard  -> dashboard + api
full       -> mobile + dashboard + api
```

Use the manifest key and CLI term `stack`. Do not use inheritance between stacks in the MVP.

## Profile

Reserve `profile` for *how* a stack is realized on a particular host or verification mode, not *which participants* are started. For example, a future profile might choose Android API level 35 versus 36, rootless Docker versus Docker Desktop, or local versus externally supplied infrastructure.

The MVP uses `stack` for startup groups and does not implement profile overlays. This avoids overloading one term with both component selection and runtime configuration.

## Capability

A semantic feature provided by a resolved environment, such as `api`, `web-ui`, `logs`, `browser-e2e`, or later `android-ui`. Capabilities are descriptive in the MVP; capability-based automatic stack selection is a later extension.

## Scenario

A repeatable verification workflow that requires capabilities and invokes tests or observations. Scenarios are out of scope for the MVP except for reserving terminology; named tests are sufficient initially.

---

## Persistence and lifecycle

Recommended Go domain types:

```text
Lease
LeaseState / DesiredState / ObservedState
LeaseOwner
SourceSpec
ResolvedSource
SourceSet
RuntimeSpec
Component
ResolvedComponentGraph
Stack
Capability
Resource
ResourceKind
CommandRun
TestRun
Artifact
Event
```

Do not let database row structs, YAML structs, and domain structs collapse into one shared type. Keep parsing, validation, domain behavior, and persistence boundaries explicit.

## Runtime adapter contract

The exact names may differ, but preserve the responsibilities:

```go
type Runtime interface {
    Type() string
    Validate(ctx context.Context, req ValidateRequest) ([]Diagnostic, error)
    Plan(ctx context.Context, req PlanRequest) (RuntimePlan, error)
    Create(ctx context.Context, req CreateRequest) ([]Resource, error)
    Inspect(ctx context.Context, req InspectRequest) (ObservedRuntime, error)
    Collect(ctx context.Context, req CollectRequest) ([]Artifact, error)
    Destroy(ctx context.Context, req DestroyRequest) error
}
```

Do not require all adapters to be long-running processes controlled by PID. Compose and Android have their own stable external identifiers.

## Source adapter contract

```go
type SourceProvider interface {
    Resolve(ctx context.Context, spec SourceSpec, requestedRef string) (ResolvedSource, error)
    Materialize(ctx context.Context, lease Lease, source ResolvedSource) (WorktreeResource, error)
    Inspect(ctx context.Context, resource WorktreeResource) (ObservedSource, error)
    Remove(ctx context.Context, resource WorktreeResource, force bool) error
}
```

The Git implementation should shell out to the installed Git CLI rather than reimplementing Git object/ref behavior with a library.

---

## `agent-env plan`

`plan` must not mutate Git, Docker, or SQLite lease state.

It should:

1. locate and parse `.agent-env.yaml`;
2. validate schema and references;
3. identify source repositories;
4. resolve requested refs to commits without creating worktrees;
5. resolve stack roots to component dependency closure;
6. identify required runtime operations;
7. validate host policy and prerequisites where possible;
8. show a deterministic plan and diagnostics.

Example human output:

```text
Repository: C:\src\control-repo
Stack:      dashboard

Sources:
  backend   requested=refs/pull/3/head   resolved=abc1234...

Components:
  1. api
  2. dashboard

Runtime:
  backend   compose
  files:    infra/compose.yaml, infra/compose.agent.yaml
  services: db, api, dashboard

Warnings:
  none
```

## `agent-env create`

Conceptual saga:

```text
reserve lease ID and resource names
  -> persist requested lease and event
  -> resolve and persist complete source set
  -> create all worktrees
  -> render and validate runtime plans
  -> create/start Compose project
  -> run readiness checks
  -> persist observed resources and evidence
  -> mark ready
```

On failure, compensate in reverse order. If compensation is incomplete, preserve the lease as quarantined.

Do not expose an environment as `ready` before all required components pass readiness.

## `agent-env destroy`

Conceptual saga:

```text
mark releasing
  -> prevent new commands
  -> collect final logs/config
  -> stop/down Compose project
  -> inspect tracked worktree changes
  -> remove safe worktrees
  -> retain artifacts by policy
  -> mark released
```

---

## State paths and ownership

AGENT_ENV_HOME overrides OS defaults. Linux uses XDG_STATE_HOME/agent-env or ~/.local/state/agent-env; macOS uses ~/Library/Application Support/agent-env; Windows uses LOCALAPPDATA/agent-env. Keep state.db and worktrees, repositories, leases, artifacts, logs, generated, locks beneath this state root. Independently disposable caches use OS cache locations. Local SQLite is required; NFS/SMB state databases are unsupported.

Desired state is active, stopped, or released. Observed state distinguishes allocating, starting, ready, degraded, stopped, failed, releasing, released, quarantined and unknown. Allocation/start failure becomes failed; missing/unhealthy resources become degraded; unsafe/incomplete release becomes quarantined. Persist timestamps in UTC/RFC3339, requested stack, resolved components, manifest and source-set digests, ownership, creation, heartbeat and expiration.

Ownership is advisory metadata, not authorization. Explicit --owner overrides AGENT_ENV_OWNER; otherwise derive descriptive local owner metadata with a generated unique token rather than PID alone. Lease commands refresh heartbeat. renew changes expiration; default TTL is four hours, configurable by host policy.

Persist normalized repositories, leases, lease_sources, lease_components, resources, events, command_runs and artifacts, with foreign keys and indexes for owner/state/expiration/repository/resource lookup. Reserve unique resource names transactionally. Keep YAML/domain/database models separate. Enable foreign_keys, busy_timeout at least 5000 milliseconds and verified WAL. Never serialize arbitrary secret-bearing environment maps.

A sorted alias/repository identity/resolved-commit tuple determines source-set digest. Store exact requested refs, resolved commits, checkout mode, writable policy and timestamps for every source before materialization. Review worktrees are detached and writable for generated output; detect staged and unstaged tracked edits before cleanup instead of treating filesystem permissions as review policy.

Command readiness attempts persist a running command record before execution. Only complete process-tree termination and evidence persistence allow a terminal record. Unconfirmed termination or output halts retries, quarantines creation, and leaves the record running so later destroy/GC also retains the source. Ordinary completed probe failures may retry. Durable cancellation requests stop readiness without starting another attempt.
