---
status: accepted
owner: maintainers
last_verified: 2026-09-08
---

# Use SQLite

[日本語](0002-use-sqlite.ja.md)

## Decision

Accepted: use database/sql with modernc.org/sqlite and explicit embedded numbered SQL migrations. SQLite owns leases, normalized source tuples, resources, command runs, artifacts, and events. Enable foreign keys, busy timeout at least 5000 ms, and verified WAL on local filesystems. JSON-only registry and ORM are rejected. External effects require compensation, not fictitious cross-system transactions.

## Consequences

This decision requires tests and documented limitations. Changes require an ADR and synchronized implementation/checks.
