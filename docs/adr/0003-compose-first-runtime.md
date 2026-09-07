---
status: accepted
owner: maintainers
last_verified: 2026-09-08
---

# Compose first runtime

[日本語](0003-compose-first-runtime.ja.md)

## Decision

Accepted: Compose v2 is the first runtime adapter, with explicit unique project identity, absolute config paths, selected service closure, normalized configuration digest, and observed resources. Reject a thin Compose wrapper as the whole product: immutable sources and lease reconciliation remain separate responsibilities. Defer generic processes and Android until the Compose MVP passes.

## Consequences

This decision requires tests and documented limitations. Changes require an ADR and synchronized implementation/checks.
