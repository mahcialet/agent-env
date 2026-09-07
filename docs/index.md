---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# Repository knowledge

Start with [architecture](../ARCHITECTURE.md), [MVP specification](product-specs/agent-env-mvp.md), and the [completed ExecPlan](exec-plans/completed/agent-env-mvp.md).

Current extension: [Android Emulator contract](product-specs/android-emulator.md) and its [active ExecPlan](exec-plans/active/android-emulator-lease.md).

- [Product specifications](product-specs/index.md): user-visible contracts.
- [Design documents](design-docs/index.md): mechanisms and boundaries.
- [ADRs](adr/index.md): accepted alternatives and consequences.
- [Plan policy](PLANS.md): living work and completion evidence.
- [Quality](QUALITY.md): commands and verification scope.
- [Reliability](RELIABILITY.md): failures and conservative recovery.
- [Security](SECURITY.md): trust and host policy.
- [Portability](PORTABILITY.md): native platform requirements.
- [Roadmap](roadmap.md): deferred features and unresolved decisions.
- [References](references/index.md): historical provenance.

Design, product, ADR and plan documents carry status, owner and last_verified metadata. Their local indexes make all durable documents discoverable. Archive material is governed by the references index; generated truth is produced from migrations when implemented. Freshness dates describe document review, not proof that planned features are implemented.

- [Generated database schema](generated/db-schema.md): mechanically derived from embedded migrations.
