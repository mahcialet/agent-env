# agent-env

Create disposable environment leases from pinned local Git commits and isolated Docker Compose projects. Select a stack, inspect its live state, run named tests with retained evidence, then clean up its resources. Multiple repositories and simultaneous leases are supported.

**Environment isolation is not a malicious-code sandbox.** Dockerfiles, Compose configuration, tests, and package scripts execute repository-controlled code. Use trusted or controlled repositories; arbitrary untrusted pull requests need a stronger outer boundary.

## Build and verify

Use Go 1.26.x or 1.27.x. Runtime operations need Git and Docker with the Compose v2-or-later plugin and a reachable daemon. The repository harness itself needs no Bash, Make, PowerShell, or Docker for ordinary unit checks.

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go build ./cmd/agent-env
go run ./tools/repoctl test-integration
```

The last command explicitly runs real Docker fixtures on Linux. Native Windows/macOS/Linux unit CI and five CGO-disabled build targets are separate from Docker integration coverage. Current verification and remaining gates are recorded in the [implementation plan](docs/exec-plans/active/agent-env-mvp.md), [quality guide](docs/QUALITY.md), and [portability notes](docs/PORTABILITY.md).

## Use a trusted repository

Declare sources, Compose runtimes, components, stacks, and named argv tests in its `.agent-env.yaml`. The [manifest reference](docs/product-specs/manifest-v1.md) has a complete example. For a simple repository with exactly one root Compose file, `init` creates a candidate manifest without overwriting an existing file; review its service selection and host policy before running it.

The following commands run from this checkout; replace the repository path, stack, and named test with your own values:

```text
go run ./cmd/agent-env doctor ../trusted-repo
go run ./cmd/agent-env validate ../trusted-repo
go run ./cmd/agent-env plan ../trusted-repo --stack api
go run ./cmd/agent-env create ../trusted-repo --stack api --ref HEAD
go run ./cmd/agent-env list --output json
go run ./cmd/agent-env show <lease-id>
go run ./cmd/agent-env capabilities <lease-id>
go run ./cmd/agent-env test <lease-id> api-smoke
go run ./cmd/agent-env destroy <lease-id> --dry-run
go run ./cmd/agent-env destroy <lease-id>
```

Alternatively, build the executable and put it on PATH to use `agent-env` (`agent-env.exe` on Windows). `plan` resolves commits without allocating resources. `plan` and `create` accept `--manifest <path>` to select a trusted control manifest explicitly; runtime files still come from each pinned source. Multi-repository ref overrides use `--source alias=ref`.

Declare component endpoints to generate dynamic loopback host publishing in the saved execution configuration without editing source Compose files. Compose resources must be project-scoped and mounts must satisfy host policy. Fixed container names, privileged mode, host networking, Docker socket mounts, and unsafe external binds are rejected. Named tests use argv arrays, with stdout, stderr, exit status, and declared artifacts retained after cleanup. See the [CLI contract](docs/product-specs/cli-contract.md) and [security policy](docs/SECURITY.md).

`gc` previews expired candidates; only `gc --apply` requests deletion. Tracked changes, uncertain ownership, or incomplete cleanup quarantine a lease. Explicit `destroy --force` retains tracked-diff evidence before discarding tracked edits and never overrides an ownership mismatch.

## State and limits

State lives outside target repositories. Set `AGENT_ENV_HOME` to an absolute path to override the native defaults: Linux XDG state, macOS Application Support, or Windows LOCALAPPDATA. The home contains `state.db`, managed worktrees, normalized runtime configuration, lease artifacts, and a diagnostic `leases/<id>/environment.json` descriptor. SQLite remains authoritative. Defaults are a 4-hour TTL, a 24-hour maximum TTL, and 8 active reservations; quarantined leases retain reservations. A host policy configuration file is not exposed yet.

Android/Flutter, browser/CDP, remote Git caching, registry promotion, and writable fix leases are [roadmap items](docs/roadmap.md).

Contributors start at [AGENTS.md](AGENTS.md) and the [documentation index](docs/index.md). Licensed under the existing [MIT license](LICENSE).
