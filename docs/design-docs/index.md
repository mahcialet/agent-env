---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Design documents

[日本語](index.ja.md)

These documents explain how the [product contracts](../product-specs/index.md)
are implemented. Start with [Architecture](../../ARCHITECTURE.md) for the system
map, or [ADRs](../adr/index.md) for accepted choices and alternatives.

## Lease model and cleanup

- [Core beliefs](core-beliefs.md) ([日本語](core-beliefs.ja.md)): Design tradeoffs and invariant priorities.
- [Local lease model](lease-control-plane.md) ([日本語](lease-control-plane.ja.md)): Sources, component closure, lifecycle and stored ownership.
- [Reconciliation and GC](reconciliation-and-gc.md) ([日本語](reconciliation-and-gc.ja.md)): Diagnostics, cleanup eligibility and unfinished-operation barriers.
- [Multi-host coordination](multi-host-control-plane.md) ([日本語](multi-host-control-plane.ja.md)): Controller placement, worker authority, journals and verified transfers.

## Runtime resources and applications

- [Common Compose lifecycle](compose-runtime.md) ([日本語](compose-runtime.ja.md)): Service selection, invocation, readiness, ports and inspection.
- [Docker and Podman providers](compose-providers.md) ([日本語](compose-providers.ja.md)): Engine pinning, normalization and ownership-based cleanup.
- [Persistent processes](persistent-process-runtime.md) ([日本語](persistent-process-runtime.ja.md)): Native process identity, retained state, termination and endpoints.
- [Android Emulator resources](android-emulator.md) ([日本語](android-emulator.ja.md)): Private AVDs, reservations, shared ADB and conservative recovery.
- [Flutter Android applications](flutter-android-runtime.md) ([日本語](flutter-android-runtime.ja.md)): Build ordering, installation, reverse mappings and compensation.

## Observation and control

- [Android UI observer](android-ui-observer.md) ([日本語](android-ui-observer.ja.md)): Snapshot identity, companion input and explicit interrupted-run recovery.
- [Browser/CDP automation](browser-cdp-automation.md) ([日本語](browser-cdp-automation.ja.md)): Connection proof, stale-node rejection, evidence and bounded capture.

## Distribution and documentation

- [Standalone distribution](standalone-distribution.md) ([日本語](standalone-distribution.ja.md)): Git-derived releases, deterministic archives, inspection and asset storage.
- [Bilingual documentation](bilingual-documentation.md) ([日本語](bilingual-documentation.ja.md)): Language ownership, translation review, metadata and exceptions.
