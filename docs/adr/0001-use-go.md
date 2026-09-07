---
status: accepted
owner: maintainers
last_verified: 2026-09-07
---

# Use Go

## Decision

Accepted: use module github.com/mahcialet/agent-env, Go language baseline 1.26.0, and supported 1.26.x/1.27.x CI. Native binaries and argv execution favor Go over per-repository shell orchestration. Use CGo-free dependencies and no automatic toolchain directive. Native platform CI remains necessary beyond cross-builds.

## Consequences

This decision requires tests and documented limitations. Changes require an ADR and synchronized implementation/checks.
