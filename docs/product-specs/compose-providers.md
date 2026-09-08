---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Compose providers

[日本語](compose-providers.ja.md)

This is the provider contract delivered by the
[completed ExecPlan](../exec-plans/completed/compose-provider-podman.md). Real Linux rootless acceptance has passed with Podman 5.4.2 and podman-compose
1.6.0, including Docker coexistence. Native Windows/macOS/Linux provider CI passed on 4a5de3d (run 34216579481);
real Podman Machine infrastructure is unavailable.

## Selection and prerequisites

A Compose runtime may select exactly one provider:

```yaml
runtimes:
  backend:
    type: compose
    provider: podman-compose
    source: backend
    project_directory: .
    files: [compose.yaml]
```

Omitting `provider` selects `docker-compose`, with the same runtime behavior as
explicit `provider: docker-compose`. The other accepted value is `podman-compose`.
Unknown values, explicit empty/null values and provider fields on non-Compose
runtimes fail manifest validation before runtime effects. Components and stacks
remain independent of the provider. The selected provider is recorded in the plan
and lease execution snapshot and reported by show and runtime diagnostics.

There is no executable-based automatic fallback. A Podman runtime never becomes a
Docker runtime because Podman is unavailable. Existing snapshots without provider
identity retain Docker behavior. Later inspect, logs, reconcile and destroy use
the stored selection, even if the target manifest changes.

| Provider | External prerequisites | Version contract |
| --- | --- | --- |
| `docker-compose` | Docker client, Compose v2 and reachable selected engine | Existing Docker contract |
| `podman-compose` | Podman 5.x client/engine and standalone podman-compose | podman-compose >=1.6.0,<2.0.0; exact tested client/server versions belong in acceptance evidence |

Podman, podman-compose and their installation dependencies are host prerequisites,
not bundled assets or dependencies of the standalone core. An older installation,
including podman-compose 1.3, does not satisfy this contract. The `podman compose`
wrapper is not a separate provider. Doctor reports missing prerequisites rather
than installing tools or switching engines. Version/help and unrelated capabilities
continue to work without Podman or Python.

## Identity, endpoints and cleanup

Each lease owns an explicit Compose project on a recorded engine. Subsequent
operations must retain that engine even when a default connection changes.
Re-observation that cannot establish the recorded identity blocks destructive
cleanup and preserves quarantine evidence. The fingerprint distinguishes supported
local/remote configurations using endpoint, host OS/architecture and storage roots;
it cannot prove that an engine was not reset in place with identical topology.
Resource ownership checks remain necessary.

Both providers apply the existing host policy before creating resources. Only the
selected service closure starts. Fixed host ports remain rejected; dynamic
endpoints require observed mappings. The common snapshot retains published port
`0`; Podman receives a private copy with that field omitted and `host_ip` preserved,
requesting an engine-assigned port without widening the loopback binding. Podman Machine host-loopback mappings must
not be advertised as usable without evidence of reachability from the agent-env
host. Unsupported or unproven mappings fail closed. Native fake tests and
cross-builds do not establish real Machine support.

Live container, network and volume observations must prove ownership using
provider-native identities together with agent-env ownership evidence. A matching
generated name alone is insufficient. Logs preserve timestamps and service/container
attribution. Destroy re-inspects resources after provider down and preserves sibling
leases and unrelated resources.

Anonymous volumes require explicit cleanup evidence. Before down, app saves
`Runtime.cleanup_evidence` with the exact native volume fingerprint and proven
container attachment. Recovery retains that proof after containers disappear,
rechecks current identity and references, and removes a residual volume only when
no external or sibling container currently references it. Uncertainty requires
quarantine. No lease cleanup uses global Podman prune commands.

## Scope and acceptance

Pod creation is disabled and `x-podman*` extensions are rejected recursively at
any nesting depth. Mount types are limited to `bind`, `volume` and `tmpfs`;
`glob` and other unmodeled types fail explicitly. `network_mode` may be omitted,
empty, `bridge` or `none`; common policy denies `host`, and other explicit modes
such as `ns:`, `pasta` and `slirp4netns` are unsupported.

For Podman, the first Compose file must share its parent directory with
`project_directory`. `env_file`, config and secret file references must resolve to
regular files confined to that directory after symlink resolution; normalized
references use absolute paths. Environment values must be frozen explicitly in
normalized configuration: unresolved bare-key or null pass-through entries are
rejected. These restrictions fail before effects instead of silently widening
host access or rereading later ambient environment values.

Arbitrary provider executables, Docker
Compose v1, Quadlet/Kubernetes, OCI retention and hostile-code sandboxing are out
of scope.

Acceptance requires unchanged real Docker integration, Linux rootless Podman with
Podman 5.x and podman-compose >=1.6.0,<2.0.0, two concurrent Podman leases, reachable dynamic
endpoints, the same named/E2E fixture, sibling survival, verified cleanup and
Docker/Podman coexistence. Native Windows/macOS/Linux tests cover selection,
parsing, paths, argv and identity pinning. Real Podman Machine evidence is recorded
separately where infrastructure exists. See the ExecPlan for exact versions,
commands, outcomes and unresolved gaps, and the
[design](../design-docs/compose-providers.md) for implementation boundaries.
