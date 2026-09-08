---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Add podman-compose as a Compose runtime provider

[日本語](compose-provider-podman.ja.md)

This ExecPlan is a living document. Maintain it according to `docs/PLANS.md`.

Expected branch: `feat/compose-provider-podman`.

Preferred start: merge the current standalone-distribution work first, then
branch from `master`. If intentionally stacked, record the exact base branch and
commit below and keep stacked evidence distinct from merged-master evidence.

Starting base branch: `master` (standalone PR 6 merged).
Starting revision: `f239fe5`

Upstream implementation:
- https://github.com/containers/podman-compose
- initial tested floor: `podman-compose >= 1.6.0`

Do not advertise untested legacy Podman combinations. Record the exact Podman
client/server and podman-compose versions in acceptance evidence.

## Purpose / Big Picture

After this work, the existing `type: compose` runtime supports either Docker
Compose or `podman-compose`.

Backward-compatible default:

```yaml
runtimes:
  backend:
    type: compose
    source: backend
    project_directory: .
    files: [compose.yaml]
```

Explicit providers:

```yaml
runtimes:
  backend:
    type: compose
    provider: docker-compose
    source: backend
    project_directory: .
    files: [compose.yaml]
```

```yaml
runtimes:
  backend:
    type: compose
    provider: podman-compose
    source: backend
    project_directory: .
    files: [compose.yaml]
```

Omitted provider means `docker-compose`. There is no automatic fallback between
Docker and Podman based on host executables.

Stacks/components remain provider-independent. Provider identity is pinned in the
lease execution snapshot and used by create, inspect, logs, reconcile and destroy.

The Podman provider must preserve the existing lease invariants: exact engine
identity, unique project/resource ownership, dynamic endpoints, sibling survival,
conservative cleanup and quarantine on uncertainty.

## Scope

In scope:

- `runtimes[].provider` with `docker-compose` and `podman-compose`;
- Docker default for existing manifests;
- provider in plan/snapshot/show/doctor;
- provider-neutral Compose adapter boundary;
- move current Docker behavior behind that boundary without regression;
- `podman-compose` Doctor/config/up/logs/down integration;
- direct Podman engine inspection for live resource truth;
- local rootless and remote/Podman Machine identity handling;
- pin exact Podman engine rather than mutable default connection;
- provider-neutral Compose normalization and host-policy evaluation;
- dynamic endpoint discovery;
- provider-native plus agent-env ownership labels;
- network/volume ownership strengthening where supported;
- residual-resource verification after down;
- safe handling of anonymous-volume differences;
- Docker+Podman coexistence;
- two concurrent Podman leases;
- Linux rootless real integration;
- Windows/macOS/Linux native fake/path/argv/identity tests;
- real Podman Machine validation where infrastructure is available;
- bilingual durable documentation.

Out of scope:

- bundling Podman or podman-compose;
- installing Python or provider dependencies;
- Docker Compose v1;
- `podman compose` wrapper as a distinct provider;
- arbitrary user-defined Compose executables;
- full `x-podman` extension support;
- Quadlet/Kubernetes/pod runtimes;
- global Podman prune;
- remote multi-host orchestration;
- OCI promotion/retention;
- malicious-code sandboxing.

## Progress

- [x] 2026-09-08: Recorded merged master f239fe5 and created `feat/compose-provider-podman`.
- [x] 2026-09-08: Run baseline repository harness and race suite.
- [x] 2026-09-08: Inspect current Compose/app/domain/store/CLI boundaries.
- [x] 2026-09-08: Write English/Japanese provider product/design docs.
- [x] 2026-09-08: Add provider field/default/strict validation.
- [x] 2026-09-08: Add provider identity to plan/snapshot/show/doctor.
- [x] 2026-09-08: Introduce provider-neutral Compose boundary.
- [x] 2026-09-08: Move current Docker implementation behind it.
- [x] 2026-09-08: Re-run all Docker tests and real Docker integration.
- [ ] Implement Podman Doctor/version checks.
- [ ] Define and persist local/remote Podman engine identity.
- [ ] Pin later Podman operations to the recorded engine.
- [ ] Normalize `podman-compose config` into common host-policy model.
- [ ] Decide canonical recorded config representation.
- [ ] Reject behavior-changing unmodeled Podman extensions.
- [ ] Implement Podman Up/Inspect/dynamic endpoints.
- [ ] Implement Podman Logs.
- [ ] Implement Down plus residual-resource reinspection.
- [ ] Add anonymous-volume cleanup regression.
- [ ] Strengthen common network/volume ownership labels if proven portable.
- [ ] Prove two simultaneous Podman leases are isolated.
- [ ] Prove Docker and Podman leases coexist.
- [ ] Prove default Podman connection changes cannot redirect cleanup.
- [ ] Run real Linux rootless Podman integration.
- [ ] Run the same named/E2E fixture with Docker and Podman.
- [ ] Add native Windows/macOS/Linux provider tests.
- [ ] Run real Podman Machine integration where available.
- [ ] Update bilingual architecture/portability/security/reliability/quality/roadmap.
- [ ] Update standalone prerequisite matrix.
- [ ] Run final harness/race/integration.
- [ ] Complete acceptance evidence and bilingual retrospective.
- [ ] Move both plans to `docs/exec-plans/completed/`.

A checkbox means observed completion. Record UTC date, revision, command/test/run
and result.

## Surprises & Discoveries

- 2026-09-08: User installed Podman 5.4.2 and podman-compose 1.3.0; rootless info works. The installed compose version is below the required 1.6.0 floor. Requested an update or permission for an isolated verification install; do not lower the floor.
- 2026-09-08: Concurrent full-check/race execution hit an Android port ownership test once while the race suite and Docker integration passed. The same full check passed when rerun serially; no Android changes were made.

- 2026-09-08: Baseline Go tests and vet passed; full check stopped at missing translation_of/source_sha256 in the supplied Japanese plan. Review and add its metadata before re-running.
- 2026-09-08: Docker 29.7.2 is available; Podman and podman-compose are absent from PATH. Installation is outside this plan; requested an existing environment or explicit installation scope while proceeding with independent implementation.

Record provider differences including config normalization, labels, health state,
dynamic ports, rootless networking, anonymous volumes, Podman Machine forwarding,
Windows process behavior, path handling and any Compose option that behaves
differently from Docker.

Do not hide Podman differences behind Docker-oriented assumptions.

## Decision Log

- 2026-09-08: Preserve omitted-provider canonical manifest bytes/digest; resolve the legacy Docker default only in the domain snapshot. Inventory keys include provider, engine identity and project.
- 2026-09-08: Use a native agent-env child bridge for podman-compose: its --podman-args are appended after the command and cannot reliably pin global remote flags. The bridge prepends pinned flags and scrubs ambient routing; no shell wrapper.

- Decision: Keep runtime `type: compose`; represent implementation with provider.
  Rationale: components/stacks describe Compose workloads independently of engine.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Omitted provider means `docker-compose`.
  Rationale: existing manifests must remain behaviorally unchanged.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Never auto-fallback between Docker and Podman.
  Rationale: host-dependent selection breaks reproducibility and cleanup identity.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Initial Podman provider targets upstream podman-compose 1.6.0+.
  Rationale: use the current stable baseline rather than expanding the first slice
  to legacy behavior.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Podman and podman-compose remain optional host prerequisites.
  Rationale: preserve the standalone one-binary core and avoid bundling provider
  runtime/package dependencies.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Pin Podman engine identity, not only a mutable connection name.
  Rationale: changing the default or connection definition must not redirect
  destructive effects.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Common host policy applies before either provider creates resources;
  unmodeled behavior-changing Podman extensions are rejected.
  Rationale: provider choice must not bypass safety policy.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Do not trust `podman-compose down --volumes` as complete cleanup
  evidence; re-inspect actual resources.
  Rationale: current Podman Compose differs from Docker for some anonymous volume
  cleanup and agent-env already requires evidence-backed release.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Durable docs and this ExecPlan are bilingual.
  Rationale: repository documentation policy.
  Date/Author: 2026-09-08 / maintainers.

## Outcomes & Retrospective

Not completed.

At completion summarize final manifest syntax, provider architecture, Docker
non-regression, tested Podman versions, engine identity model, config format,
ownership labels, endpoint semantics, rootless results, cleanup differences,
Docker-vs-Podman E2E evidence, Podman Machine evidence/gaps and deferred features.

## Context and Orientation

Read before implementation:

- `AGENTS.md` / `AGENTS.ja.md`
- `ARCHITECTURE.md` / `ARCHITECTURE.ja.md`
- `docs/PLANS.md` / `docs/PLANS.ja.md`
- Compose/MVP product and design docs
- `docs/PORTABILITY.md` / `.ja.md`
- `docs/SECURITY.md` / `.ja.md`
- `docs/RELIABILITY.md` / `.ja.md`
- `docs/QUALITY.md` / `.ja.md`
- `docs/roadmap.md` / `.ja.md`
- standalone distribution docs after merge
- completed MVP ExecPlan
- Flutter Android endpoint consumer docs
- `internal/runtime/compose`
- `internal/app`
- `internal/domain`
- `internal/config`
- `internal/policy`
- `internal/execx`
- `internal/store/sqlite`
- existing Docker Compose integration fixtures

The current Compose adapter directly owns Docker context discovery, Docker
Compose v2 invocation, Docker Engine inspection, Docker labels and endpoint
parsing. First separate those mechanisms without changing Docker behavior.

Upstream references:
- https://github.com/containers/podman-compose
- https://docs.podman.io/en/latest/
- https://compose-spec.io/

## Plan of Work

### Milestone 1 — Contract and provider boundary

Create:

    docs/product-specs/compose-providers.md
    docs/product-specs/compose-providers.ja.md
    docs/design-docs/compose-providers.md
    docs/design-docs/compose-providers.ja.md

Add strict `provider` validation and provider identity to the durable snapshot.

Refactor Docker behind a provider boundary. Do not proceed to Podman effects
until existing Docker unit/race/real integration passes unchanged.

### Milestone 2 — Podman Doctor and engine identity

Inspect at least Linux rootless Podman, and Podman Machine where available:

    podman --version
    podman info --format json
    podman system connection list --format json
    podman-compose version

Define a non-secret engine fingerprint. Distinguish local versus remote/Machine.

For remote mode, record exact URI/identity data sufficient to avoid mutable
connection-name authority. Scrub inherited routing variables and explicitly
restore the recorded endpoint for later provider and direct Podman calls.

Before destructive effects, re-observe the engine fingerprint. Mismatch =>
quarantine, never redirected cleanup.

### Milestone 3 — Render normalization and policy

Docker normalizes with JSON; podman-compose emits normalized YAML.

Podman path:

1. run provider config normalization;
2. parse YAML;
3. convert to common normalized model;
4. apply the same host policy;
5. prune selected reachable service/resource closure;
6. serialize immutable recorded config and digest.

Prove whether podman-compose 1.6.0 reliably accepts the current canonical JSON
recorded config. If yes, prefer common JSON. Otherwise use common model plus a
deterministic provider-specific serialization.

Reject behavior-changing `x-podman` or provider-specific resource types unless
explicitly modeled by policy.

### Milestone 4 — Up, Inspect and endpoints

Start with explicit detached mode. Do not use provider `--wait` as readiness
authority; app keeps bounded readiness.

Use direct structured Podman commands for live truth where appropriate:

    podman ps --all --filter ... --format json
    podman inspect
    podman network ls/inspect
    podman volume ls/inspect
    podman port

Never use `latest` or list order.

Normalize observations to existing resource/readiness/endpoint model.

Prove dynamic loopback publishing with real Linux rootless Podman. On
macOS/Windows Podman Machine, return a host-local endpoint only after native
evidence proves that the reported mapping is reachable from the agent-env host.

### Milestone 5 — Ownership labels

Keep:

    io.agent-env.lease
    io.agent-env.runtime

Provider-native observations include Docker `com.docker.compose.*` and Podman
`io.podman.compose.*` plus current compatibility labels.

Do not make Docker-compatible labels the sole Podman authority.

Test whether agent-env labels can be stamped onto selected top-level networks and
named volumes in both providers. If supported, adopt them. Otherwise document the
provider-native project-label + recorded-ID proof.

No destructive action uses only a generated name.

### Milestone 6 — Logs and tests

Implement provider-specific log collection without assuming Docker-only flags.
For Podman, direct `podman logs --timestamps` is acceptable if it gives more
stable ownership and service attribution.

Run the same named test/evidence path used by Docker fixtures. Tests must not
depend on provider-specific container names.

### Milestone 7 — Cleanup and residual resources

Before down, capture exact resources attached to proven-owned containers,
including anonymous volume identities.

After provider down, re-inspect actual engine state.

A residual anonymous volume may be removed directly only when:

- it was attached to a proven-owned container; and
- no external/sibling container currently references it.

Otherwise quarantine and retain evidence.

Never run:

    podman system prune
    podman volume prune
    podman network prune

for lease cleanup.

### Milestone 8 — Concurrency and coexistence

Prove:

- two Podman leases of the same source are isolated;
- destroying A preserves B;
- Docker and Podman leases coexist;
- project-name coincidence cannot cross ownership;
- changing default Podman connection after create does not redirect reconcile or
  destroy;
- changed recorded remote identity blocks cleanup.

Run the same application/E2E fixture against both providers and record real
behavioral differences.

### Milestone 9 — Native and real evidence

Required real evidence:

- Linux rootless Podman;
- podman-compose >= 1.6.0;
- two simultaneous leases;
- dynamic endpoint;
- named test;
- sibling survival;
- cleanup verification.

Native Windows/macOS/Linux CI covers parsing, argv/path handling, provider
selection, identity parsing and remote pinning logic without shell dependencies.

Run real Podman Machine integration where infrastructure exists. Do not call fake
tests or cross-builds real Machine validation.

### Milestone 10 — Documentation and completion

Update bilingual README, Architecture, Portability, Security, Reliability,
Quality, Roadmap, product/design indexes and standalone prerequisite matrix.

Document:

| Provider | External prerequisites | Initial tested floor |
| --- | --- | --- |
| Docker Compose | docker + Compose v2 | existing contract |
| Podman Compose | podman + podman-compose | podman-compose 1.6.0 |

Do not describe Python as a core agent-env dependency.

Run final repository harness, translation checks, race and integration suites,
then complete evidence/retrospective and archive both plans.

## Concrete Steps

1. Record base and create branch.
2. Add bilingual active plans.
3. Run baseline harness/race.
4. Write provider product/design docs.
5. Add manifest provider field/default.
6. Refactor Docker behind provider interface.
7. Revalidate Docker fully.
8. Implement Podman Doctor/identity pinning.
9. Implement config normalization/policy.
10. Implement Up/Inspect/endpoints.
11. Implement ownership improvements.
12. Implement logs.
13. Implement Down/residual verification.
14. Add anonymous-volume regression.
15. Add two-Podman isolation.
16. Add Docker+Podman coexistence.
17. Add default-connection-change regression.
18. Run Linux rootless integration.
19. Run same E2E on both providers.
20. Add native OS tests.
21. Run Podman Machine integration where possible.
22. Update bilingual durable docs.
23. Run final harness/race/integration.
24. Record direct acceptance evidence.
25. Complete bilingual retrospective.
26. Move plans to completed and update links/hashes.

## Validation and Acceptance

| ID | Required behavior | Evidence |
| --- | --- | --- |
| P1 | Existing provider-omitted Compose manifests still use Docker and pass current integration unchanged. | Pending |
| P2 | Explicit `docker-compose` equals the default. | Pending |
| P3 | Explicit `podman-compose` selects only Podman and never falls back. | Pending |
| P4 | Unknown/non-Compose provider configuration fails before effects. | Pending |
| P5 | Plan/snapshot/show diagnostics persist provider identity. | Pending |
| P6 | Docker context/cleanup safety is unchanged by refactor. | Pending |
| P7 | Podman Doctor records provider version, client/server versions, mode and non-secret engine identity. | Pending |
| P8 | Later Podman operations remain pinned to the recorded engine after default connection changes. | Pending |
| P9 | Engine identity mismatch blocks destructive cleanup and quarantines. | Pending |
| P10 | podman-compose config enters the same common host-policy model before effects. | Pending |
| P11 | Unmodeled Podman extensions cannot bypass common policy. | Pending |
| P12 | Podman starts only selected service closure in detached mode. | Pending |
| P13 | Structured Podman inspection reports owned resources/readiness. | Pending |
| P14 | Real Linux rootless dynamic endpoints work and fixed host ports remain rejected. | Pending |
| P15 | Logs retain timestamps and service/container attribution without unsupported Docker-only flags. | Pending |
| P16 | Ownership identity is verified before Podman destructive effects. | Pending |
| P17 | Two simultaneous Podman leases have distinct projects/resources/endpoints and both become READY. | Pending |
| P18 | Destroying Podman A preserves Podman B and external Podman resources. | Pending |
| P19 | Docker and Podman leases coexist without cross-provider observation/cleanup. | Pending |
| P20 | Anonymous-volume cleanup differences are detected; proven residuals are handled safely and ambiguous residuals quarantine. | Pending |
| P21 | No global Podman prune command is used. | Pending |
| P22 | Same named/E2E fixture succeeds on Docker and Podman, or differences are explicitly documented. | Pending |
| P23 | Real Linux rootless integration proves create/endpoint/test/sibling/cleanup. | Pending |
| P24 | Native Windows/macOS/Linux provider/path/argv/identity tests pass without shell dependency. | Pending |
| P25 | Podman Machine evidence is recorded where available; missing environments are not replaced by fake claims. | Pending |
| P26 | Podman remains optional; standalone core commands require neither Podman nor Python. | Pending |
| P27 | Bilingual durable docs describe final provider contract and prerequisites. | Pending |
| P28 | Final harness/translation/race suites pass. | Pending |
| P29 | Both plans contain direct evidence and retrospective before archival. | Pending |

Code existence alone is not acceptance. Record exact provider versions and
successful commands/tests/workflow runs.

## Idempotence and Recovery

Plan and Doctor are read-only.

Provider selection is immutable inside a lease.

Persist provider/engine identity before resource effects depend on it.

A failed or uncertain Podman command is not automatically retried when its effect
may already have occurred. Re-observe the exact recorded engine first.

Never switch provider during recovery.

Repeated destroy re-inspects actual engine state and removes only proven-owned
residuals. Ambiguous resources quarantine.

Do not solve Podman cleanup differences by weakening Docker assertions or common
host policy.

## Artifacts and Notes

2026-09-08 provider-boundary evidence: `go run ./tools/repoctl check`, `go test -race ./...`, and `go run ./tools/repoctl test-integration` all passed. The integration run exercised the existing real Docker lifecycle before enabling Podman effects. New tests cover mixed-provider Doctor-before-reservation, immutable snapshot cleanup, unknown-provider no-runner behavior, provider-scoped inventory collisions, and manifest-aware CLI Doctor without fallback.

Record:

- provider;
- podman-compose version;
- Podman client/server version;
- local/remote mode;
- non-secret engine fingerprint;
- connection name/URI where appropriate, excluding credentials;
- rootless state;
- project/config digest/services;
- container/network/volume IDs;
- native and agent-env labels;
- endpoints;
- cleanup residuals;
- test run IDs.

Never persist private key contents, passwords or sensitive environment values.

## Interfaces and Dependencies

Likely shape:

```go
type ComposeProviderName string

const (
    ComposeProviderDocker ComposeProviderName = "docker-compose"
    ComposeProviderPodman ComposeProviderName = "podman-compose"
)
```

Potential layout:

    internal/runtime/compose/
    internal/runtime/compose/docker/
    internal/runtime/compose/podman/

Exact package/type design may change after inspection.

Docker prerequisites:
- docker;
- Compose v2;
- selected Docker engine/context.

Podman prerequisites:
- podman;
- podman-compose >= tested floor;
- local engine or explicit reachable Podman service;
- Podman Machine where required by host OS.

No new core Python, shell, CGO or mandatory-daemon dependency is introduced.

## Unresolved Issues to Settle During Milestone 1

1. Final key name `provider` versus `compose_provider` (default preference:
   `provider`).
2. Stable Podman engine fingerprint fields.
3. Exact env/argument mechanism that pins podman-compose children to a recorded
   remote endpoint.
4. Whether podman-compose 1.6.0 reliably accepts canonical recorded JSON.
5. Whether to add agent-env labels to top-level networks/volumes.
6. Policy for harmless generic `x-*` versus behavior-changing `x-podman`.
7. Podman health-state mapping.
8. Podman Machine host-loopback endpoint semantics.
9. `podman-compose logs` versus direct `podman logs`.
10. Ownership proof for residual anonymous volumes.
11. Common project-name subset.
12. Future profile override interaction with provider.

Resolve these in Decision Log before dependent behavior is declared stable.
