---
status: accepted
owner: maintainers
last_verified: 2026-09-08
---

# Repository-native harness

[日本語](0004-repository-native-harness.ja.md)

## Decision

Accepted: concise AGENTS.md, architecture map, indexed durable documents, living ExecPlans and Go repoctl form the development harness. A monolithic instruction file or chat-only progress cannot reliably support future agents. Enforce metadata, indexes, links, plan sections, generated drift and dependency boundaries through failure-tested validators. Root instructions have a 150-line hard maximum.

## Consequences

This decision requires tests and documented limitations. Changes require an ADR and synchronized implementation/checks.
