---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# Quality and verification

## Unit tests

Test domain logic without external tools:

- stack dependency closure;
- cycle detection;
- manifest strict decoding;
- source-set digest stability;
- lease state transitions;
- project-name normalization;
- path resolution;
- policy diagnostics;
- GC eligibility;
- JSON output schemas;
- command argument handling.

## Fake command runner tests

All Git and Compose adapters must depend on an injected command runner. Use a fake runner to verify exact executable, argv, cwd, environment, timeouts, stdout/stderr handling, and error conversion.

Do not unit-test by asserting one giant shell command string.

## Fixture integration tests

Use temporary Git repositories and a small Compose fixture. Mark Docker-requiring tests with a build tag or explicit environment variable so ordinary unit tests remain reliable.

Required integration cases:

- two simultaneous Compose projects;
- selected service closure;
- failed startup rollback;
- project manually removed before reconcile;
- worktree path containing spaces;
- multiple repositories in one lease;
- dirty tracked worktree quarantine.

## Repository-harness tests

Test the development harness itself:

- `AGENTS.md` line-limit and referenced-path checks;
- Markdown internal-link resolution;
- local index completeness for design docs, product specs, and ADRs;
- active ExecPlan mandatory-section validation;
- document metadata parsing;
- generated schema drift detection;
- architecture import-boundary fixtures;
- stable diagnostic codes and nonzero exit statuses;
- `repoctl` argv handling on paths containing spaces and Unicode.

The tests must include intentionally broken fixtures so a passing checker is known to detect failure, rather than only testing the happy path.

## CI

Create a matrix for Windows, macOS, and Linux that runs:

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go build ./cmd/agent-env
```

`repoctl check` must visibly compose formatting verification, unit tests, `go vet`, documentation checks, generated-file checks, and architecture checks. Keep the underlying commands documented so the harness remains inspectable.

Run actual Docker Compose integration tests on Linux CI initially. Hosted macOS and Windows Docker availability may not be sufficient for identical integration coverage; compensate with adapter contract/fake-runner tests and document the limitation. Add self-hosted platform integration runners later if available.

Also add a cross-build job with `CGO_ENABLED=0` for the target OS/architecture matrix.

Race tests should run on at least Linux:

```text
go test -race ./...
```

---

## Harness commands

`go run ./tools/repoctl check` is the canonical cross-platform entry point. It composes formatting verification (`gofmt`), `go test ./...`, `go vet ./...`, docs-check, generated-check, and arch-check. `test-unit` runs unit tests; `test-integration` explicitly opts into Docker fixtures. Run doctor first for prerequisites. Missing prerequisites are failures or explicitly unverified coverage, never passing evidence. Generated schema comes from numbered SQL migrations; generated-check must detect drift. Broken-link, malformed-plan, unindexed-document, and forbidden-import fixtures must exercise checker failure diagnostics.
