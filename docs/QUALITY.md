---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Quality and verification

[日本語](QUALITY.ja.md)

## Canonical checks

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go run ./tools/repoctl test-integration
```

`doctor` locates Go, gofmt, and Git. Docker is required only for the explicit integration command. `check` visibly composes formatting verification, `go test ./...`, `go vet ./...`, documentation validation, generated-file drift detection, and architecture checks. It requires no Bash, Make, or PowerShell. Underlying commands remain directly runnable.

| Harness command | Scope |
| --- | --- |
| `test-unit` | Ordinary Go tests; no Docker daemon required |
| `test-integration` | Checks Docker daemon/Compose and runs uncached tagged tests with explicit integration opt-in |
| `docs-check` | Metadata, links/headings, local indexes, AGENTS length/paths, mandatory active-plan sections, and missing/stale English/Japanese pairs |
| `generate` | Regenerates the database document from embedded SQL migration sources |
| `generated-check` | Fails when the generated schema differs from migrations |
| `arch-check` | Enforces documented import boundaries with actionable diagnostics |

The schema document's source of truth is the numbered SQL set under [migrations](../migrations/001_initial.sql), not manually edited prose. Stable `AGENTENV-*` diagnostics identify the violated invariant and repair direction. Negative fixtures deliberately break links, indexes, metadata, plan sections, source formatting, schema generation, import boundaries, missing/orphan translations, incorrect source metadata, stale hashes, and language-specific links/indexes.

## Product tests

Unit and adapter tests cover strict manifest decoding, deterministic component/source identity, state transitions, TTL and GC eligibility, normalized Compose policy, path containment, command argv, streamed redaction, evidence failure paths, and lifecycle compensation. Fake runners assert executable, argv, working directory, environment controls, cancellation, output handling, and error conversion without constructing shell strings.

SQLite tests use real temporary databases, including Unicode paths, reopened/multiple connections, migration idempotency, foreign keys, atomic capacity/project reservation, normalized-row rollback, and durable lock behavior. Process tests exercise native argv and child-process cancellation; native platform execution is necessary to validate OS-specific implementations.

## Real Docker fixtures

The [integration suite](../internal/cli/integration_test.go) creates isolated temporary Git repositories and state homes and uses [the small Compose fixture](../testdata/compose/compose.yaml). Tests require both the `integration` build tag and explicit opt-in; the harness sets these automatically. Ordinary unit tests never start Docker containers.

The suite verifies simultaneous API/Dashboard leases, selected closure, manifest-generated dynamic loopback HTTP without source port declarations, versioned environment descriptors, component-scoped live/retained logs, distinct project/worktree identities, and preservation of a sibling's container/network/volume IDs. It also verifies an unselected foreign volume survives cleanup, manually removed projects become degraded, multiple repositories respect source ref overrides, named tests retain redacted stdout/stderr/artifacts and nonzero exit status, readiness failure rolls back real resources, and dirty tracked worktrees survive GC until explicit force with diff evidence.

The final local Linux CLI integration run passed in 109.95 seconds, including generated endpoints, diagnostic descriptors, and component-scoped live and archived logs. This establishes real Docker behavior for that tested revision, not completion of later edits or every native platform. All fixture resources have unique tracked identities and lease-specific cleanup; the suite never runs a general Docker prune.

## CI and completion evidence

CI runs the harness and CLI build natively on Windows, macOS, and Linux for Go 1.26.x and 1.27.x. Linux runs `go test -race ./...` and explicit Docker integration. Separate cross-build jobs cover five targets with `CGO_ENABLED=0`.

The [completed implementation plan](exec-plans/completed/agent-env-mvp.md) records all 33 acceptance criteria and resolved independent review findings. CI 34124194139 on c641286 passed all 12 jobs: six native OS/Go checks, five CGO-disabled builds, and a Linux job running full race plus actual Docker integration. Local Go 1.26.8/1.27.1 checks and real Docker integration also passed. Docker integration on hosted macOS/Windows is not claimed.

## Translation verification

Durable English files are canonical and paired with Japanese `.ja.md` files.
The [language policy](design-docs/bilingual-documentation.md) defines metadata
and exact-path exceptions. `docs-check` compares the translation's source hash
with SHA-256 of the full English file after CRLF-to-LF normalization; a Windows
checkout therefore agrees with Linux/macOS. Checks are read-only and never
refresh translation metadata automatically. A matching hash detects which
source revision was acknowledged; it cannot prove that the translation is
accurate. Review the actual Japanese text before updating the hash.
