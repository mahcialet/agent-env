---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Roadmap and unresolved decisions

[日本語](roadmap.ja.md)

The implemented scope is local pinned Git sources, detached review worktrees, isolated Compose and Android Emulator runtimes, Flutter Android application builds/install/launch and backend reverse mappings, owned Android UI accessibility observation/actions, named argv tests, evidence, and a portable repository harness. This roadmap does not advertise deferred features as available commands.

## Settled MVP choices

The module is `github.com/mahcialet/agent-env` and the existing MIT license is retained. AGENTS is capped at 150 lines, package boundaries have structural checks, and database documentation is generated from numbered migrations. Built-in policy rejects fixed published ports and selected external/shared Compose resources. SQLite transactions and renewable fenced operation locks coordinate local processes. Execution snapshots contain only the selected service/resource closure. Native process-tree cancellation belongs to the bounded runner; persistent lifetimes use a separate managed detached interface.

## Trust and host policy

A configurable host policy file, configurable grace/retention periods, and parallel-allocation controls remain future work. Trusted base-manifest plus target overlay merging, `--manifest-ref`, and explicit untrusted-fork execution require a separate trust design. Advisory owner labels do not provide authentication. Do not weaken built-in policy merely to accept an unsafe repository.

## Sources and writable workflows

Remote mirror/cache management, HTTPS/SSH authentication, provider-specific PR shorthand, and automatic fetching are deferred. Writable fix leases need branch ownership, per-source write selection, and recovery rules. Fork/checkpoint/reproduce and live stack expansion/shrink need an explicit identity and artifact model.

## Runtime extensions

iOS, browser/CDP automation and snapshots, and distributed/multi-host coordination are not implemented. Android Emulator leases own private AVD state and local SDK processes; additional real-device CI needs acceleration-capable runners. Browser observation and actions need a separate contract above generic process ownership.

The [persistent process runtime](product-specs/persistent-process-runtime.md) is
implemented with direct argv, private mutable state, named loopback TCP ports and
conservative native tree cleanup. Final native integration acceptance remains
in progress in the [active process plan](exec-plans/active/persistent-process-runtime.md).
Self-daemonization, automatic restart, interactive terminals, remote execution and
service installation remain outside this runtime contract.

The [Android UI observer](product-specs/android-ui-observer.md) adds bounded semantic
snapshots, PNGs, Unicode replacement, navigation and current-PID logs to existing
owned Emulators. Its separate optional platform companion requires no target-app
instrumentation. OCR, visual regression, richer gestures, physical devices and remote
Emulator hosts remain deferred. Final observer acceptance and platform evidence are
tracked in the [completed plan](exec-plans/completed/android-ui-observer.md).

Compose provider selection and the Podman adapter are implemented; evidence is in the
[provider plan](exec-plans/completed/compose-provider-podman.md). Docker remains the
default with no automatic fallback. Real Linux rootless acceptance passed with Podman 5.4.2 / podman-compose 1.6.0
and Docker coexistence. Native Windows/macOS/Linux provider CI passed on 4a5de3d (run 34216579481); real Machine
infrastructure is unavailable.
`podman compose` wrappers, Quadlet/Kubernetes, pod creation and arbitrary provider
executables remain outside this work.

## Artifacts and releases

Local OCI registries, image promotion, image-retention references, and exact-artifact replay are deferred. Runtime inspection records actual container image identity, but this is not reproducible image promotion. Automatic artifact expiry, event compaction, migration rollback tooling, generated CLI/JSON Schema references, and a longer-term handoff archive policy remain open.

The initial distribution uses GitHub Release archives under the [standalone contract](product-specs/standalone-distribution.md). Release engineering and direct native evidence are tracked in the [release plan](exec-plans/completed/standalone-release-finalization.md). Signing, notarization, package-manager recipes, self-update, SBOMs and attestations remain follow-up work. A release must cite the tested revision's native and integration evidence; supported build targets are not a substitute for that evidence.

## CI expansion

Native Docker integration on Windows and macOS may need self-hosted Docker-capable runners. Android integration would require suitable hardware acceleration. Documentation age beyond `last_verified` is not currently a CI freshness deadline; metadata validity and discoverability are enforced.
