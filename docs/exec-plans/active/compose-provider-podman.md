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
- [x] 2026-09-08: Implement Doctor/version checks; `TestPodmanDoctorVersionFloorAndSnapshot` rejects 1.3.0 and 2.0.0 and accepts the 1.6.0 fixture.
- [x] 2026-09-08: Persist local/remote identity and pin native children; `TestPodmanIdentityPinsLocalAndRemote`, `TestPodmanBridgeNativeRoundTrip`, and `TestPodmanChangedEngineRefusesMutation` pass locally.
- [x] 2026-09-08: Normalize YAML into canonical JSON and common policy; `TestPodmanNormalizeComposeModel` and `TestPodmanRenderRejectsProviderSpecificHostAccess` pass. Real 1.6.0 lifecycle acceptance of recorded JSON subsequently passed (110.13 s).
- [x] 2026-09-08: Reject unmodeled extensions recursively and unresolved environment pass-through; focused Render and normalization regressions pass.
- [x] 2026-09-08: Implement detached Up, structured Inspect/endpoints, timestamped Logs, and Down/reinspection. Local harness/race and real Linux lifecycle acceptance pass.
- [x] 2026-09-08: Persist anonymous-volume proof before Down, retain it across interrupted cleanup, and halt before effects on write failure; app/backend regression tests pass.
- [x] 2026-09-08: Fix and independently recheck all seven review findings recorded below.
- [x] 2026-09-08: Add portable provider/path/argv/identity tests; local Linux execution passes. Final native Windows/macOS/Linux CI remains pending.
- [x] 2026-09-08: Update bilingual architecture, portability, security, reliability, quality, roadmap, and prerequisite documentation with implemented scope and remaining gaps.
- [x] 2026-09-08: Prove common network/volume ownership labels in the real Podman lifecycle, alongside the Docker evidence.
- [x] 2026-09-08: `TestPodmanIntegrationConcurrentLeasesAndEvidence` passed in 110.13 s with Docker coexistence enabled, Go 1.27.1, rootless Podman 5.4.2 and podman-compose 1.6.0. Both Podman leases reached READY; endpoints, named tests, sibling/foreign/Docker survival and complete cleanup passed.
- [x] 2026-09-08: Reconcile real canonical JSON, logs, actual anonymous-volume proof, redacted artifacts, and all-attached-volume absence with acceptance evidence below.
- [ ] 2026-09-08: Run final native Windows/macOS/Linux CI, and record real Podman Machine evidence where available (currently unavailable).
- [x] 2026-09-08: Full local check and full race passed again after the final port fix; real Docker and Podman integration passed. Built standalone help/version also succeeded with an empty PATH and no Python/Podman tools.
- [ ] 2026-09-08: Complete acceptance and bilingual retrospective, then move both plans to completed only when all requirements have direct evidence.

A checkbox means observed completion. Record UTC date, revision, command/test/run
and result.

## Surprises & Discoveries

- 2026-09-08: The first real 1.6.0 lifecycle attempt failed without useful native diagnostics because the shared command wrapper discarded provider stderr. Podman errors now retain native provider identity and shared-evidence-redacted stderr (at most 8 KiB); Up unwraps the adapter error rather than presenting a misleading Docker failure. The next attempt exposed Podman's rejection of `published: "0"`. These were failed approaches, not acceptance passes. SIGINT allowed orderly cleanup, and subsequent inspection found no remaining containers.
- 2026-09-08: Translate numeric/string published zero into an omitted published port only in the private Podman JSON copy. Keep host_ip, target, protocol, and the recorded canonical snapshot/digest unchanged. The subsequent real lifecycle passed in 110.13 s, including endpoint/coexistence/cleanup acceptance.
- 2026-09-08: The real fixture now retains its failed temporary directory and recovers lease identities from its private registry when create fails or does not return JSON. Cleanup of fixture images/foreign volumes follows verified lease cleanup; uncertain cleanup retains inputs and evidence. This prevents failed validation from erasing recovery evidence.

- 2026-09-08: The installed podman-compose 1.3.0 was rejected before allocation (prerequisite exit 3); this was not a lifecycle pass. The user subsequently supplied podman-compose 1.6.0. The subsequent rootless Podman 5.4.2 real coexistence validation passed in 110.13 s.
- 2026-09-08: Independent review found seven defects; all were fixed and rechecked without weakening policy:
  1. Inventory depended on surviving provider rows and missed orphan-only providers. Discovery now unions recorded providers with available host engines; inventory-only Doctor does not require podman-compose.
  2. Only root/service `x-podman` keys were rejected. Recursive checks now reject network, nested service-network, and secret extensions too.
  3. Podman `glob` mounts and `ns:`/other provider-specific network modes bypassed the common bind/host-network checks. Render now rejects unmodeled forms before effects.
  4. Retained anonymous volumes were appended after `Exists` was calculated. Inspect now derives existence from the final resource set.
  5. Relative service `env_file` and secret/config paths broke when snapshots moved to temporary directories. File references are validated, confined, and made absolute; differing initial first-file/project directories are explicitly unsupported.
  6. Container service attribution trusted only Docker-compatible labels. Native Podman service labels are now mandatory and conflicting compatibility labels are rejected.
  7. Null-map/bare-list environment values were resolved again at Up time. Render now rejects unresolved pass-through while preserving explicit empty/literal values and keeping host secrets out of diagnostics.

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

- 2026-09-08 / maintainers: Preserve the common dynamic-port representation in durable JSON and adapt published zero only at the Podman execution boundary. Upstream 1.6.0 formats an omitted published value plus host_ip as `host_ip::target`; this retains loopback binding while requesting an engine-assigned port. Do not weaken fixed-port policy or rewrite the validated snapshot to accommodate provider syntax.
- 2026-09-08 / maintainers: Native provider diagnostics are necessary to distinguish configuration failures from engine failures. Use shared secret redaction before limiting stderr to its last 8 KiB, preserve error causes, and retain failed integration registries for recovery. Diagnostics and test cleanup must not discard the evidence needed to repair a failed allocation.

- 2026-09-08 / maintainers: Keep canonical recorded JSON and its digest. Podman receives a temporary dollar-escaped JSON copy, preventing second interpolation of literal values. Real 1.6.0 lifecycle acceptance subsequently passed. Restrict mounts to bind/volume/tmpfs and network modes to the modeled subset; reject recursive `x-podman` and unresolved environment pass-through rather than silently accepting provider-specific effects. File references must be regular files confined to project_directory and become absolute before snapshot relocation.
- 2026-09-08 / maintainers: App owns durable `Runtime.CleanupEvidence` before Down. Proof combines lease/runtime/project/engine scope, attachment container, and anonymous-volume fingerprint. Preserve proof on failure and retry, block Down on persistence failure, recheck current references and identity before residual removal, and clear proof only after verified cleanup. This prevents container disappearance from destroying the only cleanup authority.
- 2026-09-08 / maintainers: Global inventory unions registered provider identities with available host engines and uses an engine-only inventory Doctor. This finds orphan resources without turning optional podman-compose into a dependency of Docker-only lifecycle operations. Provider dispatch itself never falls back.
- 2026-09-08 / maintainers: Share existing Compose traversal through a native Podman command adapter in the same package. Require native project/service labels plus agent-env ownership and reject contradictory compatibility labels. Fingerprints bind endpoint and host/store topology, not an immutable engine generation; in-place resets with identical topology still require resource ownership checks.

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

Not completed. Provider implementation and local regression validation are delivered; real Linux lifecycle has passed and final native CI remains pending. Real Podman Machine infrastructure is unavailable. Keep this plan active until acceptance evidence is reconciled.

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
| P1 | Existing provider-omitted Compose manifests still use Docker and pass current integration unchanged. | Baseline `aee3a3d`: real Docker integration and Verify `34213899668` passed. |
| P2 | Explicit `docker-compose` equals the default. | Provider plan/config compatibility tests passed locally. |
| P3 | Explicit `podman-compose` selects only Podman and never falls back. | Explicit dispatcher and no-fallback CLI/app tests passed locally. |
| P4 | Unknown/non-Compose provider configuration fails before effects. | Strict manifest negative fixtures passed locally. |
| P5 | Plan/snapshot/show diagnostics persist provider identity. | Plan/domain snapshot and create-before-reservation tests passed locally. |
| P6 | Docker context/cleanup safety is unchanged by refactor. | Real Docker integration rerun passed after dispatch/cleanup changes; final full local check/race passed. |
| P7 | Podman Doctor records provider version, client/server versions, mode and non-secret engine identity. | Doctor fixture and real 1.6.0/Podman 5.4.2 rootless lifecycle passed. |
| P8 | Later Podman operations remain pinned to the recorded engine after default connection changes. | Local/remote identity and native bridge tests passed locally; final OS CI pending. |
| P9 | Engine identity mismatch blocks destructive cleanup and quarantines. | Changed-engine mutation rejection and cleanup quarantine/retry fixtures passed locally. |
| P10 | podman-compose config enters the same common host-policy model before effects. | Normalize/Render policy regressions and real canonical JSON lifecycle passed. |
| P11 | Unmodeled Podman extensions cannot bypass common policy. | Recursive extension and provider-specific host-access Render negative fixtures passed. |
| P12 | Podman starts only selected service closure in detached mode. | Real selected-closure and detached lifecycle fixture passed (110.13 s). |
| P13 | Structured Podman inspection reports owned resources/readiness. | Native label/health and retained-only regressions plus real READY/resource observations passed. |
| P14 | Real Linux rootless dynamic endpoints work and fixed host ports remain rejected. | Real dynamic loopback HTTP endpoints passed; fixed-port policy negatives passed. |
| P15 | Logs retain timestamps and service/container attribution without unsupported Docker-only flags. | Real logs and named-test artifact/redaction assertions passed. |
| P16 | Ownership identity is verified before Podman destructive effects. | Native project/service and conflicting-label fixtures plus real owned destruction passed. |
| P17 | Two simultaneous Podman leases have distinct projects/resources/endpoints and both become READY. | Real two-lease concurrent READY/distinct resource/endpoint fixture passed (110.13 s). |
| P18 | Destroying Podman A preserves Podman B and external Podman resources. | Real sibling and foreign resource survival assertions passed after lease destruction. |
| P19 | Docker and Podman leases coexist without cross-provider observation/cleanup. | Provider-scoped inventory tests and real Docker/Podman coexistence passed. |
| P20 | Anonymous-volume cleanup differences are detected; proven residuals are handled safely and ambiguous residuals quarantine. | Proof/replacement/retry/write-failure fixtures passed; real actual anonymous proof and every attached volume absent after cleanup passed. |
| P21 | No global Podman prune command is used. | Backend review and real scoped cleanup use no global prune commands. |
| P22 | Same named/E2E fixture succeeds on Docker and Podman, or differences are explicitly documented. | Same named pass/fail and evidence/artifact assertions succeeded with Docker coexistence enabled. |
| P23 | Real Linux rootless integration proves create/endpoint/test/sibling/cleanup. | Real Linux rootless lifecycle passed in 110.13 s; exact versions/command below. |
| P24 | Native Windows/macOS/Linux provider/path/argv/identity tests pass without shell dependency. | Portable native test sources added and passed on local Linux; final Windows/macOS/Linux CI pending. |
| P25 | Podman Machine evidence is recorded where available; missing environments are not replaced by fake claims. | Real Machine infrastructure unavailable; no real Machine claim. |
| P26 | Podman remains optional; standalone core commands require neither Podman nor Python. | Built standalone help/version passed with empty PATH and no Python/Podman tools. |
| P27 | Bilingual durable docs describe final provider contract and prerequisites. | Bilingual final scope/prerequisite/evidence updates and docs-check passed. |
| P28 | Final harness/translation/race suites pass. | Final local full check/race and real Docker/Podman integration passed; final native CI remains pending under P24. |
| P29 | Both plans contain direct evidence and retrospective before archival. | Direct bilingual evidence recorded; retrospective finalization/archive await final native CI. |

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

### 2026-09-08 implementation checkpoint

- 2026-09-08 follow-up: Full real Docker integration passed again after provider dispatch and durable-cleanup changes; full local `repoctl check` also passed. The later real Podman success is recorded below; final native CI remains separate and pending.
- Independent follow-up passed `go test ./internal/runtime/compose -run 'TestPodman(ProviderFailurePreservesRedactedNativeDiagnostic|UpUsesDynamicPortWithoutChangingCanonicalSnapshot)$' -count=1`. The tests check redacted native stderr and both numeric/string zero conversion with unchanged canonical bytes and preserved host_ip/target/protocol. The implementation's 8 KiB bound is code-reviewed; this focused test does not separately exercise a long diagnostic.

- Baseline provider-boundary commit: `aee3a3d`; native Verify run `34213899668` succeeded. This is baseline evidence, not final Podman backend CI.
- Current uncommitted implementation: local `go run ./tools/repoctl check` and `go test -race ./...` passed. Focused backend race and app cleanup tests also passed. Full check and full race passed again after the final port fix.
- Independent recheck passed `go test ./internal/runtime/compose -run 'TestPodman(RejectsNestedExecutionExtensions|InspectRetainedAnonymousVolumeExists|RenderRejectsProviderSpecificHostAccess|NativeServiceOwnership|RelativeFileReferencesSurviveSnapshotRelocation|RejectsInitialFileDirectoryMismatchBeforeProvider)$' -count=1` and `go test ./internal/runtime/compose -run 'TestPodman(EnvironmentRequiresExplicitPassThroughValues|RenderRejectsProviderSpecificHostAccess)$' -count=1`.
- App regressions: `TestCleanupRetainsProofAfterContainerDisappears`, `TestCleanupProofWriteFailurePreventsDown`, and `TestInventoryDiscoversOrphansOutsideRecordedProviders` passed. These prove persisted proof before effects, safe retry without containers, fail-closed writes, and orphan discovery beyond surviving registry rows.
- Real Linux acceptance: `go test -tags=integration ./internal/cli -run '^TestPodmanIntegrationConcurrentLeasesAndEvidence$' -count=1 -v` passed in 110.13 s with `AGENT_ENV_PODMAN_INTEGRATION=1` and `AGENT_ENV_PODMAN_DOCKER_COEXISTENCE=1`. Environment: Go 1.27.1, native Linux, rootless Podman client/server 5.4.2, podman-compose 1.6.0. The fixture proved selected closure, two simultaneous READY leases, reachable dynamic loopback endpoints, timestamped logs, actual anonymous-volume proof, successful and failed named tests, artifacts/redaction, absence of every attached volume after cleanup, and sibling/foreign/Docker survival. It exercises the actual canonical JSON execution path. No lease IDs or local executable paths are needed as durable evidence.
- Final local `go run ./tools/repoctl check` and `go test -race ./...` passed after the port fix. Built standalone `help` and `version` succeeded with PATH empty, proving these operations need neither Python nor Podman. Final native Windows/macOS/Linux CI is pending; real Podman Machine remains unavailable.

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

Implemented domain shape:

```go
type ComposeProviderName string

const (
    ComposeProviderDocker ComposeProviderName = "docker-compose"
    ComposeProviderPodman ComposeProviderName = "podman-compose"
)
```

Implemented layout keeps `dockerClient`, `podmanClient`, provider dispatch, normalization, native bridge, and anonymous cleanup in `internal/runtime/compose`. App owns provider selection and durable cleanup orchestration; CLI wires installed-engine discovery. No runtime adapter imports another runtime package.

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

## Original design questions and remaining acceptance

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

The decisions above settle selection, fingerprint, bridge, normalization subset, logs, and durable anonymous proof. Real Linux JSON/label/endpoint behavior is now validated; final native CI remains pending and profile overrides remain future scope. Retain these original questions to show what still needs acceptance rather than reopening settled decisions.
