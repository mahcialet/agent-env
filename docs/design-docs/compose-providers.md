---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Compose provider design

[日本語](compose-providers.ja.md)

The [product contract](../product-specs/compose-providers.md) defines selection and
safety requirements. The [active ExecPlan](../exec-plans/active/compose-provider-podman.md)
owns implementation and acceptance. The architecture below is under implementation;
backend normalization details and real-provider evidence remain subject to that
plan's decisions and validation.

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
service/resource closure is recorded with a digest. Canonical JSON is the intended
shared recorded configuration; podman-compose 1.6.0 acceptance of the complete
representation must be demonstrated before claiming that backend contract is
validated. Provider-specific serialization changes require an explicit plan
decision and corresponding tests.

Pod creation is disabled. Reject `x-podman` behavior-changing extensions and
unmodeled provider-specific resource types; generic extensions must not provide a
route around common policy. Provider detached startup does not make provider
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

Before down, observe resources attached to proven-owned containers, including
anonymous volume IDs. After down, inspect the actual engine again. A residual
anonymous volume is eligible for direct removal only if its prior attachment was
proven and current engine observation rules out external/sibling references.
Ambiguous ownership, identity mismatch or incomplete observations preserve
quarantine and evidence. Never compensate through global prune.

Tests must prove two Podman leases survive independent lifecycles, Docker and
Podman coexist, and changing a default connection cannot redirect cleanup.
The same named/E2E fixture must run against both providers. Native platform tests
cover parsing, argv, path handling and engine pinning, while real Linux rootless
integration proves endpoints, sibling survival and cleanup. Real Machine testing
is separate and conditional on infrastructure. All tested versions and remaining
gaps belong in the ExecPlan; podman-compose below 1.6.0 is not accepted as the
release acceptance environment.
