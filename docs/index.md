---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Repository knowledge

[日本語](index.ja.md)

Start with [architecture](../ARCHITECTURE.md) / [日本語](../ARCHITECTURE.ja.md), [MVP specification](product-specs/agent-env-mvp.md) / [日本語](product-specs/agent-env-mvp.ja.md), and the [completed ExecPlan](exec-plans/completed/agent-env-mvp.md).

Delivered extension: [Android Emulator contract](product-specs/android-emulator.md) / [日本語](product-specs/android-emulator.ja.md) and its [completed ExecPlan](exec-plans/completed/android-emulator-lease.md).

Delivered extension: [Flutter Android contract](product-specs/flutter-android-runtime.md) / [日本語](product-specs/flutter-android-runtime.ja.md) and its [completed ExecPlan](exec-plans/completed/flutter-android-runtime.md) / [日本語](exec-plans/completed/flutter-android-runtime.ja.md).

- [Product specifications](product-specs/index.md) / [日本語](product-specs/index.ja.md): user-visible contracts.
- [Design documents](design-docs/index.md) / [日本語](design-docs/index.ja.md): mechanisms and boundaries.
- [ADRs](adr/index.md) / [日本語](adr/index.ja.md): accepted alternatives and consequences.
- [Plan policy](PLANS.md) / [日本語](PLANS.ja.md): living work and completion evidence.
- [Quality](QUALITY.md) / [日本語](QUALITY.ja.md): commands and verification scope.
- [Reliability](RELIABILITY.md) / [日本語](RELIABILITY.ja.md): failures and conservative recovery.
- [Security](SECURITY.md) / [日本語](SECURITY.ja.md): trust and host policy.
- [Portability](PORTABILITY.md) / [日本語](PORTABILITY.ja.md): native platform requirements.
- [Roadmap](roadmap.md) / [日本語](roadmap.ja.md): deferred features and unresolved decisions.
- [References](references/index.md) / [日本語](references/index.ja.md): historical provenance.

Design, product, ADR and plan documents carry status, owner and last_verified metadata. Their local indexes make all durable documents discoverable. Archive material is governed by the references index; generated truth is produced from migrations when implemented. Freshness dates describe document review, not proof that planned features are implemented.

- [Generated database schema](generated/db-schema.md): mechanically derived from embedded migrations.

- [Language policy](design-docs/bilingual-documentation.md) / [日本語](design-docs/bilingual-documentation.ja.md): canonical English, maintained Japanese translations, and explicit exceptions.

Browser/CDP: [product contract](product-specs/browser-cdp-automation.md) / [日本語](product-specs/browser-cdp-automation.ja.md), [design](design-docs/browser-cdp-automation.md) / [日本語](design-docs/browser-cdp-automation.ja.md), and [completed execution evidence](exec-plans/completed/browser-cdp-automation.md) / [日本語](exec-plans/completed/browser-cdp-automation.ja.md).

Multi-host coordination acceptance is complete within the documented scope: [product contract](product-specs/multi-host-control-plane.md),
[design](design-docs/multi-host-control-plane.md), [ADR 0006](adr/0006-single-authority-multi-host.md),
and [ExecPlan](exec-plans/completed/multi-host-control-plane.md).
[README usage](../README.md#explicit-remote-mode) describes enrollment and role startup;
[quality](QUALITY.md#multi-host-native-verification) records successful native
Windows/macOS/Linux TLS execution and distinguishes it from pending physical-host
coverage.
