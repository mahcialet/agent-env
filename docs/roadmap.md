---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Roadmap and unresolved decisions

[日本語](roadmap.ja.md)

Use this roadmap to distinguish delivered capabilities from decisions that need
new work. Current behavior belongs in the [product contracts](product-specs/index.md).
An active ExecPlan governs ongoing implementation; follow [plan policy](PLANS.md)
to start or resume work. A deferred item here is not an available command or an
approved implementation design.

## Settled MVP choices

The module is `github.com/mahcialet/agent-env`; the existing MIT license is
retained. The delivered foundations are pinned local Git sources, detached review
worktrees, selection of components together with all their direct and indirect dependencies, named argv tests, retained evidence, and
a portable repository harness.

AGENTS is capped at 150 lines. Structural checks enforce package boundaries, and
numbered migrations generate database documentation. SQLite transactions and
renewable fenced operation locks coordinate local processes. Execution snapshots
contain only the selected services, including their direct and indirect service dependencies,
and the resources referenced by those services. Built-in policy
rejects fixed published ports and selected external/shared Compose resources.

Bounded command process-tree cancellation and persistent managed lifetimes use
separate native interfaces. The following groups build on these choices; they do
not replace local ownership or cleanup checks.

## Trust and host policy

**Implemented:** built-in host policy applies to trusted or controlled
repositories. Advisory owner labels do not authenticate a user. Do not weaken
policy just to admit an unsafe repository; see [security](SECURITY.md).

**Open decisions:** configurable host policy files, grace/retention periods, and
parallel-allocation controls remain future work. Trusted base-manifest plus target
overlay merging, `--manifest-ref`, and explicit untrusted-fork execution need a
separate trust design.

## Sources and writable workflows

**Implemented:** sources are pinned local commits. Remote mode transfers committed
Git bundles; it does not provide general remote Git authentication or fetching.

**Deferred:** mirror/cache management, HTTPS/SSH authentication, provider-specific
PR shorthand, and automatic fetching. Writable fix leases require branch
ownership, per-source write selection, and recovery rules. Fork/checkpoint/reproduce
and live stack expansion/shrink need an explicit identity and artifact model.

## Runtime extensions

### Compose providers

**Implemented:** [Docker and Podman selection](product-specs/compose-providers.md)
is pinned per lease. Docker remains the default, with no automatic fallback.

**Evidence limits:** the [completed provider plan](exec-plans/completed/compose-provider-podman.md)
records real Linux rootless acceptance and Docker coexistence, plus native
Windows/macOS/Linux provider CI. Real Podman Machine infrastructure was unavailable.
Use that Plan for the tested versions, revisions, and run identifiers.

**Deferred:** `podman compose` wrappers, Quadlet/Kubernetes, pod creation, and
arbitrary provider executables remain outside this work.

### Persistent processes and browsers

**Implemented:** the [process runtime](product-specs/persistent-process-runtime.md)
provides direct argv, private mutable state, named loopback TCP ports, and
conservative native tree cleanup. [Browser/CDP](product-specs/browser-cdp-automation.md)
adds observation and actions through a separate interface above process ownership.
Both have native acceptance on Windows, macOS, and Linux, recorded in the
[process](exec-plans/completed/persistent-process-runtime.md) and
[browser](exec-plans/completed/browser-cdp-automation.md) completed Plans.

**Deferred:** process self-daemonization, automatic restart, interactive terminals,
direct remote process attachment, and service installation. Browser external
attachment, headful mode, downloads, Firefox/BiDi, Safari/WebKit, and a shared
Android/browser UI abstraction are outside the current slice.

### Android and Flutter

**Implemented:** [Android leases](product-specs/android-emulator.md) own private
AVD state and local SDK processes. [Flutter applications](product-specs/flutter-android-runtime.md)
build, install, launch, and create backend reverse mappings on those Emulators.
The [UI observer](product-specs/android-ui-observer.md) adds bounded semantic
snapshots, PNGs, Unicode replacement, navigation, and current-PID logs. Its
optional platform companion requires no target-app instrumentation.

**Evidence limits:** observer acceptance is recorded in its
[completed Plan](exec-plans/completed/android-ui-observer.md); additional native Emulator
CI needs acceleration-capable runners. Android integration likewise requires
suitable hardware acceleration.

**Deferred:** iOS, OCR, visual regression, richer gestures, physical devices, and
unmanaged remote Emulator attachment.

## Multi-host acceptance and deferred extensions

**Implemented:** the [single-controller contract](product-specs/multi-host-control-plane.md)
covers explicit whole-lease placement, committed source transfer, and typed worker
operations. Dispatch is serial while live leases run concurrently. A second
active operation on the same lease is rejected, including destroy during a test.

**Evidence limits:** the [completed Plan](exec-plans/completed/multi-host-control-plane.md)
records real-TLS native acceptance on Windows, macOS, and Linux, including the
tested revision and run identifier. Each runner uses two worker roots; the fixture covers named
tests, logs, artifact downloads, renewal, and environment isolation. Physical
multi-host/VM evidence remains pending. Completed acceptance applies to this
stated scope, not every deployment topology.

**Deferred:** remote cancel-active, concurrent operation dispatch, controller
HA/consensus, live migration, split-host leases, transparent endpoint tunnels,
secret provisioning, and break-glass host adoption require separate designs.

## Artifacts and releases

**Implemented:** initial distribution uses GitHub Release archives under the
[standalone contract](product-specs/standalone-distribution.md). The
[release Plan](exec-plans/completed/standalone-release-finalization.md) records
release engineering and direct native evidence. Runtime inspection records actual
container image identity; that does not provide reproducible image promotion.

**Deferred:** local OCI registries, image promotion, image-retention references,
exact-artifact replay, signing, notarization, package-manager recipes, self-update,
SBOMs, and attestations.

**Open decisions:** automatic artifact expiry, event compaction, migration rollback
tooling, generated CLI/JSON Schema references, and a longer-term handoff archive
policy. A release must cite the tested revision's native and integration evidence;
supported build targets do not replace that evidence.

## CI expansion

Native Docker integration on Windows and macOS may need self-hosted Docker-capable
runners. These infrastructure requirements are separate from cross-build support;
[quality](QUALITY.md) and [portability](PORTABILITY.md) describe current coverage.

Documentation metadata and discoverability are enforced. Age beyond `last_verified`
is not currently a CI freshness deadline. Use the [language policy](design-docs/bilingual-documentation.md)
for semantic review and translation checks; do not infer factual freshness from a
valid metadata field alone.
