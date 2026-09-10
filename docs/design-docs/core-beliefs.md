---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Core beliefs

[日本語](core-beliefs.ja.md)

These principles guide design tradeoffs. The [architecture](../../ARCHITECTURE.md)
maps components; the [lease design](lease-control-plane.md) explains their lifecycle.

Immutable source sets make an environment reproducible. A lease records every requested ref and resolved commit before runtime startup; the lease is never reduced to one commit column.

SQLite owns desired state, reservations, and evidence. Git and runtime adapters observe external resource state. Reconciliation joins those facts without pretending external effects are transactional.

Failures are data: preserve failed allocations, compensations, and quarantine events. Cleanup prioritizes preserving tracked edits and resource identity over optimistic deletion.

Components own dependencies; stacks name roots. Selecting only those roots and all their direct and indirect dependencies avoids starting costly irrelevant services. Repository manifests are explicit startup authority.

Native Windows, macOS, and Linux behavior is a product requirement. Use argument arrays, OS-native paths, and CGo-free releases; no implicit shell or mandatory daemon.

Repository knowledge is an indexed system of record. Promote recurring lessons into executable checks. Keep instructions navigational, plans current, and claims tied to actual test evidence.

Leases prevent accidental collisions. They do not sandbox malicious repository code.
