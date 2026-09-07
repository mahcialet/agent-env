# agent-env

A CLI under development for disposable environment leases built from pinned local Git repositories and isolated Docker Compose projects. The [active implementation plan](docs/exec-plans/active/agent-env-mvp.md) records current availability and verification; the commands below describe the MVP target until that plan confirms completion.

**Environment isolation is not a malicious-code sandbox.** Repository Dockerfiles, Compose configuration, tests, and package scripts can execute code with host access. Use trusted or controlled repositories; arbitrary untrusted PRs require a stronger outer boundary.

## Build and verify

Use Go 1.26.x or 1.27.x, Git, and Docker with Compose v2. Native Windows, macOS, and Linux are required targets. Docker integration coverage is tracked separately from cross-compilation.

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go build ./cmd/agent-env
go run ./tools/repoctl test-integration
```

The Go harness needs no Bash, Make, or PowerShell. See [quality](docs/QUALITY.md) and [portability](docs/PORTABILITY.md) for checks and limitations.

## Intended workflow

Declare sources, runtimes, components, stacks and named argv tests in the target repository's `.agent-env.yaml`; see [manifest schema and examples](docs/product-specs/manifest-v1.md).

```text
agent-env validate ./control-repo
agent-env plan ./control-repo --stack api
agent-env create ./control-repo --stack api --ref HEAD
agent-env list --output json
agent-env show <lease-id>
agent-env test <lease-id> api-smoke
agent-env destroy <lease-id> --dry-run
agent-env destroy <lease-id>
```

`gc` is dry-run by default; `gc --apply` requests cleanup. Dirty tracked worktrees or uncertain cleanup are quarantined, not silently erased. `destroy --force` is explicit and must retain events and available evidence. [CLI contract](docs/product-specs/cli-contract.md) and [reliability](docs/RELIABILITY.md) describe the full target behavior.

State is outside target repositories. AGENT_ENV_HOME overrides Linux XDG state, macOS Application Support, or Windows LOCALAPPDATA defaults. The registry is local SQLite; worktrees and artifacts are retained according to lifecycle and evidence policy.

Android Emulator/Flutter, browser/CDP, remote Git caching, and local registry promotion are [roadmap items](docs/roadmap.md), not implemented MVP features.

Contributors start at [AGENTS.md](AGENTS.md) and [documentation index](docs/index.md). Licensed under the existing [MIT license](LICENSE).

Implemented foundation commands are `version`, `validate [repository]`, and `plan [repository] --stack <name>`, with `--output json` returning `{ "schema_version": 1, "data": ... }`. Planning resolves local Git commits without allocating a lease. Lifecycle integration remains in progress in the active plan.
