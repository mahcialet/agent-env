---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Compose provider design

[日本語](compose-providers.ja.md)

The [product contract](../product-specs/compose-providers.md) defines selection and
safety requirements. The [completed ExecPlan](../exec-plans/completed/compose-provider-podman.md)
records implementation decisions and acceptance evidence for the architecture and
normalization described below. Real Machine validation remains unavailable.

## Responsibility and persistence

`compose.Client` dispatches to private `dockerClient` and `podmanClient`
implementations. App retains orchestration, readiness and quarantine policy;
provider adapters own process argv and engine observations. Android and Flutter
remain separate runtime boundaries. No provider imports another runtime adapter.

`domain.ComposeProviderName` identifies `docker-compose` or `podman-compose`.
New Compose plans record the effective selection in `domain.Runtime.Provider`.
`EffectiveComposeProvider` maps an absent legacy value to Docker while preserving
unknown nonempty identities for rejection. Android has no Compose provider.
The optional manifest field remains absent in canonical JSON when omitted, so
introducing the default does not rewrite old manifest digests. The lease snapshot
pins provider identity independently of later edits to the manifest.

## Engine pinning and native execution

Docker retains its existing context and ownership checks. Podman records whether
the engine is local or remote, the resolved endpoint and the observed host
OS/architecture and storage roots. This non-secret fingerprint detects changes
in the recorded topology; it is not an immutable engine generation identifier.
An in-place reset recreating the same topology can evade that fingerprint, so
resource identity checks are also required before cleanup.

Remote operations use recorded endpoint parameters, not mutable connection names
or current defaults. Inherited routing variables must be scrubbed before restoring
the pinned route. Private key contents, passwords and sensitive environment values
must not enter registry metadata, logs or committed evidence. Re-observation that
cannot establish the recorded identity stops destructive operations.

podman-compose launches Podman child processes. A native bridge implemented by the
current agent-env executable supplies the pinned Podman global arguments to those
children, so the provider and direct engine observations use the same route.
Execution uses native argument arrays and the existing execx process boundary;
the bridge is not a generated shell script. Exact endpoint parsing and argv/env
behavior require native regression evidence, including paths with spaces and
non-ASCII characters.

## Common configuration and observed truth

Docker's normalized JSON and podman-compose's normalized YAML enter the same
host-policy model. Policy runs before effects, then the selected reachable
service/resource closure is recorded with a digest as canonical JSON. Before
podman-compose reparses a mutation snapshot, literal dollar signs are escaped in a
private copy so frozen values are not interpolated again. The same private copy
omits `published` when the recorded port is zero while preserving `host_ip`;
Podman then assigns a dynamic port with the intended loopback restriction. The
canonical configuration and digest remain unchanged. The accepted host range
is Podman 5.x with podman-compose >=1.6.0,<2.0.0. Real Linux rootless acceptance
passed with 5.4.2 / 1.6.0 and Docker coexistence; the earlier 1.3 provider was
rejected by that gate. Native Windows/macOS/Linux provider CI passed on 4a5de3d (run 34216579481), and real Machine
infrastructure is unavailable.

The first Compose file's parent must match `project_directory`, avoiding Podman's
different base-directory semantics. `env_file` and config/secret file references
are resolved to absolute regular files confined to that directory, including after
symlink resolution. Null or bare-key environment pass-through is rejected;
normalized values must be explicit rather than supplied from mutable ambient state.
Project `.env` files reject reserved `PODMAN_*`, `CONTAINER_*`,
`AGENT_ENV_PODMAN_*` and `COMPOSE_*` routing/behavior keys.

Pod creation is disabled. Reject `x-podman*` recursively, including resource-level
and deeply nested extensions. Only `bind`, `volume` and `tmpfs` mount types are
modeled; reject Podman's host-expanding `glob` and other types. `network_mode`
accepts omitted/empty, `bridge` and `none`; `host` passes to common policy for
rejection, while namespace paths, `pasta`, `slirp4netns` and other modes fail here.
Provider detached startup does not make provider
`--wait` the readiness authority: app performs bounded readiness from observations.

Use direct structured Podman inspection for live containers, networks, volumes,
health and published ports. Podman-native project/service identities, including
`io.podman.compose.*`, complement `io.agent-env.lease` and `io.agent-env.runtime`.
Docker-compatible labels alone are insufficient Podman ownership authority. If
portable network/volume agent-env labels are not proven, retain the documented
provider project-label plus recorded-ID proof rather than claiming stronger labels.
Never select resources by list order, `latest`, or generated name alone.

Direct Podman logs with timestamps may provide stable container/service attribution
without depending on Docker-only Compose flags. Dynamic mappings become endpoints
only after the appropriate host reachability contract is established. Podman
Machine mappings remain unavailable when host reachability is unproven; fake
inspection data cannot establish Machine forwarding behavior.

## Cleanup and evidence

Before down, `PrepareCleanup` observes resources attached to proven-owned
containers, including native anonymous volumes, and app persists the returned
`Runtime.cleanup_evidence` before invoking Down. This proof includes the exact
volume fingerprint, lease/runtime identity and proven container attachment.
Recovery can therefore observe residuals after the original containers disappear;
it never reconstructs authority from a matching volume name alone. After down,
inspect the actual engine again and match retained fingerprints. A residual
anonymous volume is eligible for direct removal only if its prior attachment was
proven and current engine observation rules out external/sibling references.
Compose-declared anonymous mounts are normalized into deterministic project-owned
named volumes; image-declared native anonymous volumes require this separate proof.
Ambiguous ownership, identity mismatch or incomplete observations preserve
quarantine and evidence. Never compensate through global prune.

Tests must prove two Podman leases survive independent lifecycles, Docker and
Podman coexist, and changing a default connection cannot redirect cleanup.
The same named/E2E fixture must run against both providers. Native platform tests
cover parsing, argv, path handling and engine pinning, while real Linux rootless
integration proves endpoints, sibling survival and cleanup. Real Machine testing
is separate and conditional on infrastructure. All tested versions and remaining
gaps belong in the ExecPlan; versions outside Podman 5.x and podman-compose
>=1.6.0,<2.0.0 are not accepted as the release acceptance environment.
