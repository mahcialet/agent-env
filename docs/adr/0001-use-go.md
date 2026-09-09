---
status: accepted
owner: maintainers
last_verified: 2026-09-09
---

# Use Go

[日本語](0001-use-go.ja.md)

## Context

Native binaries and argument-array execution favor Go over per-repository shell
orchestration.

## Decision

Use module `github.com/mahcialet/agent-env`, with Go language baseline 1.26.0
and supported 1.26.x/1.27.x CI. Dependencies must work without CGO. Do not add an
automatic toolchain directive.

## Consequences

Cross-builds do not establish native behavior; native platform CI remains
necessary. This decision requires tests and documented limitations. Changes
require an ADR and synchronized implementation/checks. See the
[quality guide](../QUALITY.md) for the current validation matrix.
