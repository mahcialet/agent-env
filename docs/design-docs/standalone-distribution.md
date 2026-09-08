---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Standalone distribution design

[日本語](standalone-distribution.ja.md)

The [product contract](../product-specs/standalone-distribution.md) defines
the user-visible archive and prerequisite behavior. Build identity is exposed
through `internal/buildinfo`; generic immutable bytes are handled by
`internal/assets`. Release mechanics belong to repoctl so local and CI builds
share one argument-array, no-shell implementation.

The normal state root remains owned by `internal/paths`. Release output is an
explicit caller directory and is never confused with runtime state. Future
embedded companions can describe bytes once, materialize them by digest, and
reuse verified existing content without Android-specific code in the asset core.
