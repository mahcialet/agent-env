---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Architecture decisions

[日本語](index.ja.md)

ADRs record accepted choices, their rationale, alternatives and consequences.
Historical sequencing does not define today's feature availability; consult the
[design documents](../design-docs/index.md) for current mechanisms.

## Foundation decisions

- [0001 — Use Go](0001-use-go.md) ([日本語](0001-use-go.ja.md)): Native binaries, language baseline and CGO-free execution.
- [0002 — Use SQLite](0002-use-sqlite.md) ([日本語](0002-use-sqlite.ja.md)): Durable local state, migrations and external-effect compensation.
- [0003 — Compose first](0003-compose-first-runtime.md) ([日本語](0003-compose-first-runtime.ja.md)): Historical MVP sequencing and separation from source/reconciliation work.
- [0004 — Repository-native harness](0004-repository-native-harness.md) ([日本語](0004-repository-native-harness.ja.md)): Navigable instructions, living plans and failure-tested validators.

## Application and host boundaries

- [0005 — Separate Flutter applications](0005-separate-flutter-applications.md) ([日本語](0005-separate-flutter-applications.ja.md)): Independent builds and Android resource ownership.
- [0006 — Single controller authority](0006-single-authority-multi-host.md) ([日本語](0006-single-authority-multi-host.ja.md)): Whole-lease assignments, durable uncertainty and worker-local cleanup.
- [0007 — Native execution boundaries](0007-native-execution-boundaries.md) ([日本語](0007-native-execution-boundaries.ja.md)): Windows path limits and separate Windows/WSL process ownership.
