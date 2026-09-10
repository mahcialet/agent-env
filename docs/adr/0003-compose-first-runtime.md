---
status: accepted
owner: maintainers
last_verified: 2026-09-09
---

# Compose first runtime

[日本語](0003-compose-first-runtime.ja.md)

## Context

The initial MVP needed a first runtime without reducing the product to a thin
Compose wrapper. Immutable sources and lease reconciliation are separate
responsibilities.

## Decision

Use Compose v2 as the first runtime adapter. Record an explicit unique project
identity, absolute configuration paths, the selected services including all their
direct and indirect service dependencies, normalized configuration digest and observed resources.

At that stage, generic processes and Android were deferred until the Compose MVP
passed. That sequencing decision is historical: their current implementations
are described in the [design index](../design-docs/index.md).

## Consequences

Compose orchestration does not replace source management or reconciliation.
See the [Compose design](../design-docs/compose-runtime.md) for the common runtime
mechanism. This decision requires tests and documented limitations. Changes
require an ADR and synchronized implementation/checks.
