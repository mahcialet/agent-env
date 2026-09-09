---
status: accepted
owner: maintainers
last_verified: 2026-09-09
---

# Use SQLite

[日本語](0002-use-sqlite.ja.md)

## Context

Lease state spans normalized sources, resources, command runs, artifacts and
events. SQLite records these durable relationships; external effects still need
compensation rather than fictitious cross-system transactions.

## Decision

Use `database/sql` with `modernc.org/sqlite` and explicitly embedded, numbered SQL
migrations. SQLite owns leases and the associated records above. Enable foreign
keys, a busy timeout of at least 5000 ms, and verified WAL on local filesystems.
Reject a JSON-only registry and an ORM.

## Consequences

Database commits cannot make Git or runtime effects transactional. The
[lease design](../design-docs/lease-control-plane.md) explains the saga and
retained recovery evidence. This decision requires tests and documented
limitations. Changes require an ADR and synchronized implementation/checks.
