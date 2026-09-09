---
status: accepted
owner: maintainers
last_verified: 2026-09-09
---

# Repository-native harness

[日本語](0004-repository-native-harness.ja.md)

## Context

A monolithic instruction file or chat-only progress cannot reliably support
future agents. Repository knowledge and execution state need discoverable,
maintained homes.

## Decision

Use concise `AGENTS.md`, an architecture map, indexed durable documents, living
ExecPlans and Go repoctl as the development harness. Root instructions have a
150-line hard maximum.

Enforce metadata, indexes, links, plan sections, generated drift and dependency
boundaries through validators tested to reject violations.

## Consequences

This decision requires tests and documented limitations. Changes require an ADR
and synchronized implementation/checks. The [plan policy](../PLANS.md) governs
execution records; the [quality guide](../QUALITY.md) lists harness commands.
