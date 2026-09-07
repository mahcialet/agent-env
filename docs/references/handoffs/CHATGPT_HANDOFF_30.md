# CHATGPT_HANDOFF_30 — `agent-env` cross-platform MVP

## Status

- Handoff date: 2026-09-07
- Intended executor: Codex CLI / Codex coding agent
- Proposed repository: standalone repository `mahcialet/agent-env`
- Proposed implementation branch: `feat/agent-env-mvp`
- Primary target: Windows native, macOS, and Linux
- Initial vertical slice: Git worktree + Docker Compose + SQLite-backed environment leases + component/stack resolution
- Repository-development harness: required from the first milestone (`AGENTS.md`, `ARCHITECTURE.md`, indexed `docs/`, an active ExecPlan, cross-platform repository checks)
- Planned extension: Flutter + Android Emulator, browser/CDP observation, artifact promotion to a local OCI registry
- Go language baseline: Go 1.26; CI and release verification on supported Go 1.26.x and 1.27.x toolchains

---

## Codex instructions

1. Work in a **dedicated branch**. Create and switch to:

   ```text
   feat/agent-env-mvp
   ```

2. Treat this as a standalone CLI project. If the current repository is empty or newly created, initialize the Go module as:

   ```text
   github.com/mahcialet/agent-env
   ```

   If the current repository already has a different canonical module path, preserve that path instead of rewriting it.

3. Before implementing product code, establish the repository-native development harness described in this handoff. At minimum, create `AGENTS.md`, `ARCHITECTURE.md`, `docs/index.md`, `docs/PLANS.md`, `docs/design-docs/core-beliefs.md`, `docs/product-specs/agent-env-mvp.md`, and `docs/exec-plans/active/agent-env-mvp.md`. The active ExecPlan must be self-contained and updated throughout implementation.

4. Treat `CHATGPT_HANDOFF_30.md` as bootstrap input, not as the permanent encyclopedia. Preserve its provenance under `docs/references/handoffs/` if it is checked into the repository, then move durable knowledge into the appropriate design document, product specification, ADR, test, or executable check. Do not leave essential project knowledge only in an external chat.

5. Keep the root `AGENTS.md` short and navigational. Target roughly 80–120 lines and enforce a hard limit of 150 lines. It must point to authoritative documents and stable commands rather than duplicate this handoff.

6. Use `docs/PLANS.md` to define when and how ExecPlans are maintained. For this multi-milestone implementation, create and continuously update `docs/exec-plans/active/agent-env-mvp.md`; after all acceptance criteria pass, move it to `docs/exec-plans/completed/`.

7. Do not rewrite published history. Do not use force-push. Make small, meaningful commits and push each coherent milestone when a remote is already configured and credentials permit it.

8. Do not build the entire long-term vision in one undifferentiated change. Complete the MVP vertical slice first, with tests and documentation, then proceed to optional extensions only if the core acceptance criteria pass.

9. Do not silently replace a failed external integration with a fake success. Report unsupported prerequisites, failed commands, and partial cleanup explicitly.

10. Preserve evidence from failed allocation and cleanup attempts. A partially created environment must become `failed`, `degraded`, or `quarantined`; it must not disappear from the registry without an event trail.

11. The repository-specific startup contract is `.agent-env.yaml`. Do not encode repository-specific Compose paths, service names, or test commands in the Go source.

12. Implement native Windows support. Do not define “Windows support” as “run the Linux binary under WSL.” WSL may be documented later as a separate execution environment, but it is not the compatibility baseline for this MVP.

---

# Goal

Implement `agent-env`, a reusable CLI that materializes one or more Git repositories at pinned commits into an isolated, disposable, inspectable environment lease.

A repository or workspace describes how it should be started in `.agent-env.yaml`. `agent-env` handles the common control-plane responsibilities:

- resolve refs to immutable commit IDs;
- create isolated Git worktrees;
- resolve a requested startup group into the minimum required component dependency closure;
- start an isolated Docker Compose project with a unique project name;
- persist lease, source, component, runtime, command, and event records in SQLite;
- expose accurate `list` and `show` views by reconciling registry state with actual Git and Docker state;
- run named tests and collect their output as evidence;
- safely destroy, garbage-collect, or quarantine environments;
- provide stable human-readable and machine-readable output for agents;
- run natively on Windows, macOS, and Linux from one Go codebase;
- make the `agent-env` repository itself legible and operable by coding agents through a repository-native development harness;
- preserve design intent, progress, validation evidence, and recurring operational lessons in version-controlled artifacts rather than relying on chat history.

The repository-development harness and the product runtime manifest are distinct:

```text
Repository-development harness
  AGENTS.md + ARCHITECTURE.md + docs/ + repoctl + CI
  -> tells Codex how to understand, change, and verify agent-env itself

Target-repository runtime harness
  .agent-env.yaml
  -> tells agent-env how to materialize a target repository/workspace
```

The intended user experience is:

```text
agent-env plan   ./control-repo --stack api
agent-env create ./control-repo --stack api --ref refs/pull/3/head
agent-env list
agent-env test   <lease-id> api-smoke
agent-env logs   <lease-id>
agent-env destroy <lease-id>
```

The CLI must hide implementation details such as worktree locations, Compose file locations, Compose project names, transient ports, and resource cleanup order from the calling agent.

The long-term target is broader than Compose:

```text
repository/workspace definition
              |
              v
          agent-env
              |
      environment lease
       /      |       \
 worktrees  runtimes  evidence
             /   \
        Compose  Android Emulator
```

However, the first implementation must produce a reliable Compose-backed vertical slice rather than an incomplete collection of adapters.

---

# Core terminology

## Environment

The actual resources that make an application runnable: worktrees, Compose project, containers, networks, volumes, generated configuration, ports, logs, and later emulators or browsers.

## Lease

The control-plane record describing who owns an environment, what immutable source set it represents, why it exists, how long it may remain allocated, and what state its resources are in.

A lease is not merely a PID and not merely a Compose project name.

## Source set

The immutable tuple of repositories and resolved Git commits used by a lease.

Example:

```text
mobile-app@aaaa1111
backend-api@bbbb2222
shared-schema@cccc3333
```

The requested refs must also be recorded, but runtime identity is based on resolved commits.

## Component

A logical participant in the system, such as `api`, `dashboard`, `database`, or `mobile`. Components form a dependency graph and may map to one or more runtime-specific resources.

## Stack

A named startup group consisting of explicit root components. `agent-env` computes the transitive dependency closure.

Examples:

```text
api        -> api
Dashboard  -> dashboard + api
full       -> mobile + dashboard + api
```

Use the manifest key and CLI term `stack`. Do not use inheritance between stacks in the MVP.

## Profile

Reserve `profile` for *how* a stack is realized on a particular host or verification mode, not *which participants* are started. For example, a future profile might choose Android API level 35 versus 36, rootless Docker versus Docker Desktop, or local versus externally supplied infrastructure.

The MVP uses `stack` for startup groups and does not implement profile overlays. This avoids overloading one term with both component selection and runtime configuration.

## Capability

A semantic feature provided by a resolved environment, such as `api`, `web-ui`, `logs`, `browser-e2e`, or later `android-ui`. Capabilities are descriptive in the MVP; capability-based automatic stack selection is a later extension.

## Scenario

A repeatable verification workflow that requires capabilities and invokes tests or observations. Scenarios are out of scope for the MVP except for reserving terminology; named tests are sufficient initially.

---

# Decisions

## 1. Implement the CLI in Go

Use Go for the core implementation.

Reasons:

- straightforward native binaries for Windows, macOS, and Linux;
- `os/exec` permits direct argument-array execution without requiring a POSIX shell;
- good standard-library support for paths, JSON, contexts, HTTP readiness checks, hashing, and structured logging;
- a single repository can cross-build the CLI for all target operating systems;
- pure-Go SQLite permits `CGO_ENABLED=0` builds;
- adapters mostly orchestrate existing external tools rather than requiring an application runtime.

Use the oldest currently supported major release as the module language baseline:

```text
go 1.26.0
```

As of this handoff, Go 1.27 is the current major release and Go 1.26 remains supported. Test with the latest patch releases of both Go 1.26.x and Go 1.27.x. Use the latest supported toolchain for release builds, but avoid a `toolchain` directive that silently downloads a compiler in offline or restricted agent environments unless the repository later adopts that behavior deliberately.

Preferred dependencies:

- CLI: `github.com/spf13/cobra`
- YAML: `gopkg.in/yaml.v3`
- SQLite driver: `modernc.org/sqlite`
- sortable lease IDs: `github.com/oklog/ulid/v2`, or an equivalently small and well-tested implementation
- Windows system calls only where the standard library is insufficient: `golang.org/x/sys/windows`

Keep dependencies minimal. Do not introduce an ORM; use `database/sql`, explicit SQL migrations, and small repository methods.

## 2. No CGO in release builds

Release binaries must build with:

```text
CGO_ENABLED=0
```

The state database must use a CGo-free SQLite driver. This avoids requiring a C compiler on Windows and simplifies cross-compilation and release packaging.

## 3. Native cross-platform execution, not shell-script orchestration

The Go binary must invoke tools directly with argument arrays:

```text
exec.CommandContext(ctx, "git", "worktree", "add", ...)
exec.CommandContext(ctx, "docker", "compose", "-p", project, ...)
```

Do not implement core behavior with:

```text
sh -c
bash
zsh
cmd.exe /C <arbitrary concatenated string>
powershell -Command <arbitrary concatenated string>
Makefile-only lifecycle logic
```

Repository commands in `.agent-env.yaml` must use argv arrays, not shell command strings:

```yaml
command: ["go", "test", "./..."]
```

A future explicit `shell` command type may exist, but it must be opt-in and platform-qualified. It is not part of the MVP.

On Windows, executable resolution must account for `.exe`, `.cmd`, and `.bat`. Do not assume that `exec.LookPath` plus `CreateProcess` behavior is sufficient for every batch-file tool. Isolate Windows command construction behind a platform runner, use proper Windows argument escaping, and add tests for paths and arguments containing spaces, quotes, Unicode, and trailing backslashes.

## 4. Use OS-native path handling

Use `path/filepath`; never construct filesystem paths by concatenating `/`.

Support:

- Windows drive letters;
- spaces in repository and state paths;
- Unicode paths;
- macOS and Linux paths;
- case-insensitive filesystems;
- long paths where the host toolchain supports them.

Do not require symlink creation for correctness. Do not rely on POSIX mode bits or `chmod` as a policy boundary.

## 5. State belongs outside target repositories

The durable state database, worktrees, logs, and artifacts must live under an application state root, not inside the source repository.

Support an explicit override:

```text
AGENT_ENV_HOME
```

Default locations:

- Linux: `$XDG_STATE_HOME/agent-env`, otherwise `~/.local/state/agent-env`
- macOS: `~/Library/Application Support/agent-env`
- Windows: `%LOCALAPPDATA%\agent-env`

Use subordinate directories:

```text
<AGENT_ENV_HOME>/
├── state.db
├── worktrees/
├── repositories/
├── leases/
├── artifacts/
├── logs/
├── generated/
└── locks/
```

Caches that may be deleted independently should use the platform cache directory rather than the state directory. In particular, future remote Git mirrors and BuildKit metadata may be cache resources.

## 6. SQLite is the control-plane authority

SQLite is the authority for desired lease state, ownership, immutable source identity, resource allocation records, command runs, and event history.

SQLite is not the authority for whether a container is currently alive. Observed state must be reconciled from external systems.

Configure at least:

```text
foreign_keys = ON
busy_timeout = 5000 ms or greater
journal_mode = WAL, after verifying it is supported at the local state path
```

The database must remain on a local filesystem. Do not claim support for putting the control database on an NFS/SMB share in the MVP.

Use transactions and uniqueness constraints for resource reservation. Operations spanning Git, Docker, and the filesystem are implemented as a saga with compensating cleanup, not as a fictional cross-system transaction.

## 7. The repository manifest is explicit authority

A repository or control repository declares its environment contract in:

```text
.agent-env.yaml
```

`agent-env init` may detect Compose, Flutter, or test layouts and generate a candidate manifest, but automatic discovery is never runtime authority.

The manifest must state:

- sources;
- runtime definitions;
- components and dependencies;
- stacks;
- named tests;
- readiness probes;
- endpoints and artifact locations where applicable.

Moving `compose.yaml` must require changing only the manifest, not the agent skill or Go source.

## 8. Separate components from stacks

Components own dependencies. Stacks contain only explicit root components.

Example:

```yaml
components:
  api:
    runtime: backend
    compose_services: [db, api]

  dashboard:
    runtime: backend
    compose_services: [dashboard]
    depends_on: [api]

stacks:
  api:
    roots: [api]

  dashboard:
    roots: [dashboard]
```

Resolving `dashboard` yields `api` and `dashboard` in topological order.

Reject:

- unknown components;
- dependency cycles;
- duplicated names;
- a component referencing an incompatible runtime;
- a stack with no roots.

Do not implement stack inheritance in the MVP.

## 9. Multiple repositories are first-class

A lease may have one or more sources. The database must not store one global `commit_sha` on the lease as the sole source identity.

For each source record:

- stable source alias;
- canonical repository identity;
- requested ref;
- resolved commit SHA;
- worktree path;
- checkout mode;
- writable policy;
- resolution timestamp.

Resolve all requested refs first, persist the complete source set, then materialize worktrees. Compute a deterministic source-set digest by sorting normalized source records by alias and hashing repository identity plus resolved commit.

The MVP must support multiple **local** Git repositories referenced by the manifest. Remote URL caching and provider-specific PR shorthand may be added later.

## 10. Git worktree is the source-isolation primitive

For review leases, create detached worktrees at the resolved commit.

Example behavior:

```text
git worktree add --detach <path> <commit>
```

Use machine-readable Git output where available, especially `git worktree list --porcelain -z` and `git status --porcelain=v2 -z`.

Do not make the entire review worktree physically read-only. Build systems need to create `.dart_tool`, Gradle outputs, generated files, test results, and other untracked artifacts.

Review policy is instead:

- record the initial commit and clean tracked state;
- allow untracked/generated outputs;
- detect staged or unstaged tracked-file changes before release;
- mark a review lease as policy-violating or quarantined if tracked source was modified unexpectedly.

A later fix mode may create a generated branch and allow selected sources to be writable. MVP source policy should model `writable`, even if review mode is the only fully supported workflow initially.

## 11. Docker Compose is the first runtime adapter

Use Docker Compose v2 through:

```text
docker compose
```

Do not invoke legacy `docker-compose` in the MVP.

Every lease gets a unique Compose project name with a conservative character set. The display lease ID and normalized Compose project name may differ.

Example:

```text
lease ID:       review-pr3-01k4a7...
Compose project: ae_review_pr3_01k4a7
```

Always pass the project name explicitly with `-p`.

Resolve Compose paths relative to the materialized source named by the runtime. Pass absolute `-f` paths and an explicit `--project-directory` rather than relying on Compose’s parent-directory discovery.

Before starting, render and validate the Compose configuration. Save the normalized rendered configuration and its SHA-256 digest as evidence.

The MVP must detect and fail or prominently warn about multi-instance hazards:

- explicit `container_name`;
- fixed host ports needed by selected components;
- external networks or volumes with globally shared mutable state;
- host networking;
- privileged containers;
- Docker socket mounts;
- absolute host bind mounts outside allowed roots.

The exact block/warn policy belongs to host policy. Default to blocking clearly dangerous settings and blocking known multi-instance collisions.

Do not require every service in a Compose file to start. Start only the services required by the resolved component closure, plus dependencies Compose itself determines.

## 12. Leases have desired and observed state

Do not overload one string with all lifecycle semantics.

Recommended model:

```text
desired_state: active | stopped | released
observed_state: allocating | starting | ready | degraded | stopped |
                failed | releasing | released | quarantined | unknown
```

At minimum, implement transitions equivalent to:

```text
requested
  -> allocating
  -> starting
  -> ready
  -> releasing
  -> released
```

Failure paths:

```text
allocating/starting -> failed
ready -> degraded
releasing -> quarantined, when cleanup is incomplete or unsafe
```

A lease includes:

- owner ID;
- purpose;
- mode;
- requested stack;
- resolved component set;
- created time;
- last heartbeat;
- expiration time;
- manifest digest;
- source-set digest.

Use UTC timestamps in storage and RFC 3339 formatting. Human output may display local time.

## 13. Ownership is advisory control-plane metadata

Support:

```text
AGENT_ENV_OWNER
```

and:

```text
--owner
```

If neither is supplied, derive a descriptive local owner value from host and user and add a generated invocation/session token. Do not rely on PID alone, because PIDs are reused and do not represent a Codex session.

The lease ownership model is not a security boundary. It prevents accidental conflicts and informs garbage collection.

Commands operating on a lease update its heartbeat. Provide explicit renewal:

```text
agent-env renew <lease-id> --ttl 4h
```

Default TTL: 4 hours. Make it configurable by host policy.

## 14. Reconciliation is mandatory

`agent-env list` must not merely print rows from SQLite.

It should perform a bounded lightweight inspection by default and combine:

- SQLite registry records;
- `git worktree list --porcelain -z`;
- `docker compose ls --format json` and project-specific inspection;
- recorded worktree paths;
- recorded resource identifiers.

Provide:

```text
--cached
```

for callers that explicitly want registry-only output.

Provide a full command:

```text
agent-env reconcile [<lease-id>]
```

Reconciliation must detect:

- registry record exists but worktree is missing;
- registry record exists but Compose project is missing;
- Compose project exists but registry record is missing;
- worktree exists but registry record is missing;
- selected services are unhealthy;
- lease is expired but resources still exist;
- release was recorded but resources remain.

Do not automatically delete orphans during ordinary listing.

## 15. Cleanup is conservative

`destroy` removes resources in reverse dependency order and records each step.

Conceptual order:

```text
stop active test commands
collect final evidence
stop/down Compose project
inspect Git worktrees for tracked changes
remove clean worktrees
release reservations
mark released
```

If the worktree has unexpected tracked modifications, cleanup fails, or resource identity is ambiguous, quarantine instead of force deleting.

Support:

```text
agent-env destroy <lease-id> --dry-run
agent-env destroy <lease-id>
agent-env destroy <lease-id> --force
```

`--force` must be explicit and must still emit an event and preserve available evidence.

Garbage collection defaults to planning only:

```text
agent-env gc
```

must behave as a dry run. Actual deletion requires:

```text
agent-env gc --apply
```

This is intentional for unattended agent safety.

## 16. Command and test execution produces evidence

Named tests are declared in the manifest and invoked through:

```text
agent-env test <lease-id> <test-name>
```

Each run records:

- lease ID;
- test name;
- argv after safe interpolation;
- source/cwd alias;
- start and finish time;
- exit code;
- stdout path;
- stderr path;
- status;
- relevant environment metadata, with secrets redacted.

Stream output to the caller while also writing complete logs to the lease artifact directory.

Do not store secret values in SQLite or logs. Implement redaction hooks and avoid dumping the complete process environment.

## 17. Stable machine-readable output is a product requirement

Support a global output mode or command-specific option:

```text
--output table
--output json
```

At minimum, `plan`, `create`, `list`, `show`, `doctor`, and `gc` must support JSON.

JSON must be versioned and stable enough for agent skills. Do not print human commentary to stdout in JSON mode; diagnostics go to stderr.

Use nonzero exit status for failures. Distinguish at least:

- invalid user input/manifest;
- missing prerequisite;
- allocation failure;
- test failure;
- cleanup/quarantine result;
- internal error.

Document exit codes before the first public release.

## 18. No daemon in the MVP

Use a CLI, SQLite, and filesystem/database coordination. Do not begin with a background daemon, local RPC protocol, or service manager.

Periodic cleanup may later be invoked by `systemd --user`, `launchd`, or Windows Task Scheduler, but the executable and command behavior must be identical.

The design must not make a future daemon impossible, but a daemon is not required to implement leases correctly.

## 19. Lease isolation is not a security sandbox

State this prominently in the README.

`agent-env` may execute Dockerfiles, Compose definitions, test commands, package-manager scripts, and source code from the target repositories. A unique worktree and Compose project prevent accidental collision; they do not protect the host from malicious code.

The MVP is for trusted or controlled repositories.

Add a host-policy abstraction and implement basic Compose validation, but do not claim that it makes arbitrary fork PRs safe. Untrusted-code execution requires a stronger outer boundary such as a disposable VM or dedicated runner/daemon with no credentials and restricted networking.

## 20. Image retention uses promotion, not “push every build”

Do not make a local registry mandatory in the MVP and do not push every intermediate build.

For the MVP:

- use normal local BuildKit/Docker caching;
- record normalized Compose config digest;
- record actual image IDs/digests observed for running services where available;
- retain command/test evidence;
- rebuild from pinned sources when necessary.

Future artifact promotion flow:

```text
intermediate builds -> local BuildKit cache -> discardable
verified runtime image -> explicit promotion -> local OCI registry
```

Only images referenced by a meaningful test/review run should be promoted. Registry retention must be reference-based, with a grace period and explicit pins.

## 21. Android Emulator is the first planned non-Compose extension

Do not tightly couple the lease core to Compose.

Define runtime and observation interfaces so that a later `flutter-android` adapter can allocate:

- a Flutter source worktree;
- an Android Emulator slot/serial;
- APK build and install;
- `adb reverse` bindings to a Compose backend;
- UI snapshots, screenshots, and logcat artifacts.

The first branch may include interfaces, schema fields, documentation, and fakes for this adapter, but Android lifecycle is not required for the Compose MVP acceptance criteria.


## 22. The repository-native development harness is part of the product delivery

Do not postpone `AGENTS.md`, architecture maps, plans, quality checks, and evidence conventions until after the CLI exists. The repository is the first environment in which the project must demonstrate that an agent can orient itself, implement a multi-milestone change, validate it, and leave enough context for another agent to resume.

This is not documentation polish. It is part of the implementation mechanism and a prerequisite for sustained agent work.

## 23. `AGENTS.md` is a map, not a manual

The root `AGENTS.md` should contain only:

- the initial reading order;
- the standard development workflow;
- stable repository commands;
- a compact set of non-negotiable invariants;
- pointers showing where product behavior, architecture, plans, security, portability, and generated truth live;
- the rule requiring an ExecPlan for complex or multi-hour work.

Do not copy the full handoff, CLI specification, architecture, or troubleshooting guide into `AGENTS.md`. Prefer progressive disclosure from a small stable entry point.

A nested `AGENTS.md` may be added later only where a subtree has materially different commands or constraints. Do not proliferate nested instruction files in the MVP.

## 24. Complex work uses version-controlled living ExecPlans

Create `docs/PLANS.md` as the repository policy for ExecPlans. This implementation must have one active plan at:

```text
docs/exec-plans/active/agent-env-mvp.md
```

The plan must be usable without this chat or the handoff. It must maintain at least:

- Purpose / Big Picture;
- Progress, with timestamps and current incomplete work;
- Surprises & Discoveries, with concise evidence;
- Decision Log, including rationale and date/author;
- Outcomes & Retrospective;
- Context and Orientation;
- Plan of Work;
- Concrete Steps;
- Validation and Acceptance;
- Idempotence and Recovery;
- Artifacts and Notes;
- Interfaces and Dependencies.

Update the plan at every meaningful stopping point and whenever implementation evidence changes a decision. At completion, move it to `docs/exec-plans/completed/` rather than deleting it.

## 25. Knowledge has an explicit owner and location

Use the following ownership model:

| Artifact | Authoritative question |
|---|---|
| `AGENTS.md` | Where should an agent start, and which stable workflow should it follow? |
| `ARCHITECTURE.md` | What are the major modules, boundaries, dependency directions, and invariants? |
| `docs/product-specs/` | What user-visible behavior and CLI contract must exist? |
| `docs/design-docs/` | Why is the system designed this way, and how do durable mechanisms fit together? |
| `docs/adr/` | Why was one concrete alternative selected or rejected at a point in time? |
| `docs/exec-plans/` | What work is happening now, what has been learned, and what remains? |
| `docs/generated/` | What mechanically derived truth currently follows from code or migrations? |
| tests / validators / CI | Which invariants are enforced rather than merely described? |
| `docs/references/` | Which external or historical inputs informed the repository? |

When a rule repeatedly causes review findings, promote it from prose into a test, validator, structural check, or repository command. Keep the explanatory document, but make the invariant executable where practical.

## 26. Provide a cross-platform repository command surface

Add a small Go-based repository tool at:

```text
tools/repoctl/
```

It is a development-harness command, not part of the public `agent-env` CLI. It must run with ordinary Go tooling on Windows, macOS, and Linux and must not require Bash, Make, Just, or PowerShell for core workflows.

Stable commands should include:

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go run ./tools/repoctl test-unit
go run ./tools/repoctl test-integration
go run ./tools/repoctl docs-check
go run ./tools/repoctl generated-check
go run ./tools/repoctl arch-check
```

`check` should compose the safe, deterministic checks appropriate on every platform. Docker-dependent integration tests may remain an explicit command and a Linux CI job initially. The tool must invoke child processes through argv arrays and provide actionable repair messages.

Do not allow `repoctl` to become a second opaque build system. Each command should be a thin, inspectable composition of documented standard tools and repository-specific checks.

## 27. Repository knowledge is mechanically checked

Implement lightweight repository-harness checks early. At minimum, verify:

- `AGENTS.md` remains at or below 150 lines and all referenced repository paths exist;
- all Markdown files under `docs/design-docs/`, `docs/product-specs/`, and `docs/adr/` are linked from their local index;
- internal Markdown links resolve;
- active ExecPlans contain the mandatory living-plan sections;
- required document metadata is present where configured;
- `docs/generated/db-schema.md` matches the embedded SQL migrations once migrations exist;
- architectural package-boundary checks pass once package boundaries exist;
- generated or checked-in examples validate against the current manifest schema.

Freshness metadata may initially emit warnings rather than fail CI, but broken links, missing required plan sections, generated-file drift, and architectural invariant violations should fail CI.

---

# Proposed CLI surface

## Required in the MVP

```text
agent-env version
agent-env init [repository]
agent-env validate [repository]
agent-env plan <repository> --stack <name> [--ref <ref>] [--source alias=ref]
agent-env create <repository> --stack <name> [--ref <ref>] [--source alias=ref]
agent-env list [--cached] [--mine] [--state <state>]
agent-env show <lease-id>
agent-env capabilities <lease-id>
agent-env test <lease-id> <test-name>
agent-env logs <lease-id> [--component <name>] [--run <run-id>]
agent-env renew <lease-id> [--ttl <duration>]
agent-env destroy <lease-id> [--dry-run] [--force]
agent-env reconcile [<lease-id>]
agent-env gc [--apply]
agent-env doctor [repository]
```

## Deferred but reserved

```text
agent-env expand <lease-id> --stack <name>
agent-env fork <lease-id> --mode fix --writable <source>
agent-env checkpoint <lease-id>
agent-env browser <lease-id> ...
agent-env ui <lease-id> ...
agent-env artifact promote <lease-id> <artifact>
agent-env reproduce <lease-id> --artifacts|--rebuild
```

Do not ship placeholder commands that return success while doing nothing. Deferred commands may be omitted or return a clear unsupported-feature error.

---

# Manifest design

## MVP example: API and Dashboard in one Compose runtime

```yaml
version: 1

sources:
  backend:
    repository: .
    default_ref: HEAD

runtimes:
  backend:
    type: compose
    source: backend
    project_directory: .
    files:
      - compose.yaml
      - compose.agent.yaml

components:
  api:
    runtime: backend
    compose_services:
      - db
      - api
    provides:
      - api
      - logs

  dashboard:
    runtime: backend
    compose_services:
      - dashboard
    depends_on:
      - api
    provides:
      - web-ui
      - browser-e2e

stacks:
  api:
    description: Web API only
    roots:
      - api

  dashboard:
    description: Web API and Dashboard
    roots:
      - dashboard

  full:
    description: Alias for the currently available complete web stack
    roots:
      - dashboard

tests:
  api-smoke:
    stack: api
    source: backend
    working_directory: .
    command:
      - go
      - test
      - ./...

  dashboard-e2e:
    stack: dashboard
    source: backend
    working_directory: dashboard
    command:
      - npm
      - run
      - test:e2e
```

## Target extension example: mobile + API + Dashboard

This illustrates the intended schema direction. Do not require the Android adapter for the MVP.

```yaml
version: 1

sources:
  mobile:
    repository: ../mobile-app
    default_ref: main

  backend:
    repository: ../backend
    default_ref: main

runtimes:
  backend:
    type: compose
    source: backend
    project_directory: .
    files:
      - infra/compose.yaml
      - infra/compose.agent.yaml

  android:
    type: flutter-android
    source: mobile
    project_directory: .
    avd_pool: pixel-api-35
    package: com.example.app
    build:
      command: ["flutter", "build", "apk", "--debug"]
      artifact: build/app/outputs/flutter-apk/app-debug.apk

components:
  api:
    runtime: backend
    compose_services: [db, api]
    provides: [api]

  dashboard:
    runtime: backend
    compose_services: [dashboard]
    depends_on: [api]
    provides: [web-ui, browser-e2e]

  mobile:
    runtime: android
    depends_on: [api]
    provides: [android-ui, mobile-e2e]

stacks:
  api:
    roots: [api]

  dashboard:
    roots: [dashboard]

  mobile:
    roots: [mobile]

  full:
    roots: [dashboard, mobile]
```

## Manifest rules

- Unknown fields should be rejected by default to catch misspellings.
- Include `version` and reject unsupported major schema versions.
- Resolve repository-relative paths relative to the manifest’s control repository, then materialize equivalent paths under the corresponding source worktree.
- Do not allow path traversal outside declared source roots unless host policy explicitly permits it.
- Environment/template interpolation must be small, explicit, and typed. Do not implement general shell expansion.
- Save the exact manifest bytes or canonicalized manifest plus SHA-256 digest for each lease.

---

# Domain model

Recommended Go domain types:

```text
Lease
LeaseState / DesiredState / ObservedState
LeaseOwner
SourceSpec
ResolvedSource
SourceSet
RuntimeSpec
Component
ResolvedComponentGraph
Stack
Capability
Resource
ResourceKind
CommandRun
TestRun
Artifact
Event
```

Do not let database row structs, YAML structs, and domain structs collapse into one shared type. Keep parsing, validation, domain behavior, and persistence boundaries explicit.

## Runtime adapter contract

The exact names may differ, but preserve the responsibilities:

```go
type Runtime interface {
    Type() string
    Validate(ctx context.Context, req ValidateRequest) ([]Diagnostic, error)
    Plan(ctx context.Context, req PlanRequest) (RuntimePlan, error)
    Create(ctx context.Context, req CreateRequest) ([]Resource, error)
    Inspect(ctx context.Context, req InspectRequest) (ObservedRuntime, error)
    Collect(ctx context.Context, req CollectRequest) ([]Artifact, error)
    Destroy(ctx context.Context, req DestroyRequest) error
}
```

Do not require all adapters to be long-running processes controlled by PID. Compose and Android have their own stable external identifiers.

## Source adapter contract

```go
type SourceProvider interface {
    Resolve(ctx context.Context, spec SourceSpec, requestedRef string) (ResolvedSource, error)
    Materialize(ctx context.Context, lease Lease, source ResolvedSource) (WorktreeResource, error)
    Inspect(ctx context.Context, resource WorktreeResource) (ObservedSource, error)
    Remove(ctx context.Context, resource WorktreeResource, force bool) error
}
```

The Git implementation should shell out to the installed Git CLI rather than reimplementing Git object/ref behavior with a library.

---

# SQLite schema draft

Use embedded, numbered SQL migrations. The exact schema may be refined, but preserve the normalization and relationships below.

```sql
CREATE TABLE schema_migrations (
    version       INTEGER PRIMARY KEY,
    applied_at    TEXT NOT NULL
);

CREATE TABLE repositories (
    id                INTEGER PRIMARY KEY,
    canonical_id      TEXT NOT NULL UNIQUE,
    display_name      TEXT NOT NULL,
    local_git_dir     TEXT,
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL
);

CREATE TABLE leases (
    id                    TEXT PRIMARY KEY,
    purpose               TEXT NOT NULL,
    mode                  TEXT NOT NULL,
    owner_id              TEXT NOT NULL,
    requested_stack       TEXT NOT NULL,
    desired_state         TEXT NOT NULL,
    observed_state        TEXT NOT NULL,
    manifest_path         TEXT NOT NULL,
    manifest_digest       TEXT NOT NULL,
    source_set_digest     TEXT NOT NULL,
    created_at            TEXT NOT NULL,
    heartbeat_at          TEXT NOT NULL,
    expires_at            TEXT NOT NULL,
    released_at           TEXT,
    last_error            TEXT
);

CREATE TABLE lease_sources (
    lease_id              TEXT NOT NULL,
    source_alias          TEXT NOT NULL,
    repository_id         INTEGER NOT NULL,
    requested_ref         TEXT NOT NULL,
    resolved_commit       TEXT NOT NULL,
    worktree_path         TEXT NOT NULL,
    checkout_mode         TEXT NOT NULL,
    writable              INTEGER NOT NULL DEFAULT 0,
    resolved_at           TEXT NOT NULL,
    PRIMARY KEY (lease_id, source_alias),
    UNIQUE (worktree_path),
    FOREIGN KEY (lease_id) REFERENCES leases(id),
    FOREIGN KEY (repository_id) REFERENCES repositories(id)
);

CREATE TABLE lease_components (
    lease_id              TEXT NOT NULL,
    component_name        TEXT NOT NULL,
    runtime_name          TEXT NOT NULL,
    resolution_order      INTEGER NOT NULL,
    observed_state        TEXT NOT NULL,
    PRIMARY KEY (lease_id, component_name),
    FOREIGN KEY (lease_id) REFERENCES leases(id)
);

CREATE TABLE resources (
    id                    INTEGER PRIMARY KEY,
    lease_id              TEXT NOT NULL,
    kind                  TEXT NOT NULL,
    logical_name          TEXT NOT NULL,
    external_id           TEXT,
    desired_state         TEXT NOT NULL,
    observed_state        TEXT NOT NULL,
    metadata_json         TEXT NOT NULL,
    created_at            TEXT NOT NULL,
    updated_at            TEXT NOT NULL,
    UNIQUE (lease_id, kind, logical_name),
    FOREIGN KEY (lease_id) REFERENCES leases(id)
);

CREATE TABLE events (
    sequence              INTEGER PRIMARY KEY AUTOINCREMENT,
    lease_id              TEXT,
    occurred_at           TEXT NOT NULL,
    event_type            TEXT NOT NULL,
    payload_json          TEXT NOT NULL,
    FOREIGN KEY (lease_id) REFERENCES leases(id)
);

CREATE TABLE command_runs (
    id                    TEXT PRIMARY KEY,
    lease_id              TEXT NOT NULL,
    run_type              TEXT NOT NULL,
    logical_name          TEXT,
    source_alias          TEXT,
    working_directory     TEXT NOT NULL,
    argv_json             TEXT NOT NULL,
    started_at            TEXT NOT NULL,
    finished_at           TEXT,
    exit_code             INTEGER,
    status                TEXT NOT NULL,
    stdout_path           TEXT,
    stderr_path           TEXT,
    FOREIGN KEY (lease_id) REFERENCES leases(id)
);

CREATE TABLE artifacts (
    id                    TEXT PRIMARY KEY,
    lease_id              TEXT NOT NULL,
    command_run_id        TEXT,
    kind                  TEXT NOT NULL,
    path                  TEXT NOT NULL,
    sha256                TEXT,
    retention_class       TEXT NOT NULL,
    created_at            TEXT NOT NULL,
    FOREIGN KEY (lease_id) REFERENCES leases(id),
    FOREIGN KEY (command_run_id) REFERENCES command_runs(id)
);
```

Add indexes for common list/reconcile/GC queries, including owner, observed state, expiration, repository, and active resources.

Do not store arbitrary secret-bearing environment maps in `metadata_json`.

---

# Proposed repository layout

```text
agent-env/
├── AGENTS.md                         # short map and stable workflow
├── ARCHITECTURE.md                   # durable code and dependency map
├── README.md                         # user-facing introduction and quick start
├── CHATGPT_HANDOFF_30.md             # optional bootstrap input; archive after import
├── LICENSE                           # only after license is decided
├── go.mod
├── go.sum
├── .github/
│   └── workflows/
│       ├── ci.yml
│       ├── docs.yml
│       ├── integration-compose.yml
│       └── release.yml               # may be deferred
├── cmd/
│   └── agent-env/
│       └── main.go
├── internal/
│   ├── app/                          # use cases / orchestration
│   ├── cli/
│   ├── config/                       # manifest and host policy
│   ├── domain/
│   ├── evidence/
│   ├── execx/                        # cross-platform command runner
│   ├── paths/                        # platform state/cache locations
│   ├── reconcile/
│   ├── stack/                        # dependency closure / planning
│   ├── source/
│   │   └── gitcli/
│   ├── runtime/
│   │   ├── registry.go
│   │   └── compose/
│   └── store/
│       └── sqlite/
├── migrations/
│   └── 0001_initial.sql
├── tools/
│   └── repoctl/
│       └── main.go                   # cross-platform repository harness commands
├── docs/
│   ├── index.md
│   ├── PLANS.md                      # ExecPlan policy and required structure
│   ├── QUALITY.md
│   ├── RELIABILITY.md
│   ├── SECURITY.md
│   ├── PORTABILITY.md
│   ├── design-docs/
│   │   ├── index.md
│   │   ├── core-beliefs.md
│   │   ├── lease-control-plane.md
│   │   ├── manifest-components-and-stacks.md
│   │   ├── reconciliation-and-gc.md
│   │   └── cross-platform-execution.md
│   ├── product-specs/
│   │   ├── index.md
│   │   ├── agent-env-mvp.md
│   │   ├── cli-contract.md
│   │   └── manifest-v1.md
│   ├── adr/
│   │   ├── index.md
│   │   ├── 0001-use-go.md
│   │   ├── 0002-use-sqlite.md
│   │   ├── 0003-compose-first-runtime.md
│   │   └── 0004-repository-native-harness.md
│   ├── exec-plans/
│   │   ├── active/
│   │   │   └── agent-env-mvp.md
│   │   ├── completed/
│   │   └── tech-debt-tracker.md
│   ├── generated/
│   │   └── db-schema.md
│   └── references/
│       ├── index.md
│       ├── harness-engineering.md
│       └── handoffs/
│           └── CHATGPT_HANDOFF_30.md
└── testdata/
    ├── compose-basic/
    ├── compose-dashboard/
    ├── multi-repo/
    ├── invalid-cycle/
    └── path with spaces/
```

The exact number of design documents may be reduced if two topics are genuinely inseparable, but do not collapse the whole knowledge base into `AGENTS.md` or one enormous design file. Every directory above needs a real purpose; do not generate empty placeholder documents merely to match the tree.

Do not require a root `.agent-env.yaml` for the `agent-env` repository during the Compose MVP. The repository-development harness describes how to develop this CLI, while `.agent-env.yaml` is the product contract consumed from target repositories. Dogfooding may later use a fixture or example workspace once the relevant runtime behavior exists.

# Repository-native agent harness

## Design principle

The repository must provide a small stable entry point and progressively disclose deeper context. An agent should be able to start from `AGENTS.md`, discover the active plan and authoritative design/product documents, run one stable check command, and understand how to prove a change works.

The harness must not assume access to the original chat. Anything required to resume implementation belongs in the repository.

## Required `AGENTS.md` outline

Keep the file concise. A suitable outline is:

```text
# Agent entry point

## Read first
- ARCHITECTURE.md
- docs/index.md
- relevant product specification
- relevant active ExecPlan

## Standard workflow
- inspect current branch and working tree
- use/update an ExecPlan for complex work
- make a demonstrable incremental change
- run repoctl check and relevant integration tests
- record discoveries and decisions
- commit coherent slices without rewriting history

## Stable commands
- go run ./tools/repoctl doctor
- go run ./tools/repoctl check
- go run ./tools/repoctl test-integration

## Non-negotiable invariants
- native Windows/macOS/Linux support
- no required POSIX shell or CGO release build
- exact commit pinning for leases
- conservative cleanup and visible quarantine
- lease isolation is not a security sandbox

## Where knowledge belongs
- product specs / design docs / ADRs / plans / generated docs / tests
```

Do not add transient task details, large code examples, complete CLI documentation, or copied external articles to this file.

## Required ExecPlan behavior

Before substantive implementation, Codex must convert this handoff into a self-contained active plan. The plan must describe exact repository-relative files, expected commands, observable behavior, rollback/retry paths, and milestone-level validation. It must be updated as evidence appears; it is not a one-time design proposal.

At a minimum, the active plan should identify:

- the current repository state and branch;
- which harness documents exist and which are still missing;
- the next independently verifiable milestone;
- tests already run and their outputs;
- decisions changed from this handoff and why;
- unresolved platform-specific risks;
- exact next actions at the current stopping point.

## Documentation metadata and indexing

For durable design, product, ADR, and plan documents, use a minimal front matter or equivalent machine-readable header containing:

```text
status: draft | active | accepted | superseded | completed
owner: maintainers or a named repository role
last_verified: YYYY-MM-DD
```

Design and product directories require `index.md`. A document not linked from the relevant index is considered undiscoverable and should fail `docs-check`, except for deliberately generated or archived material governed by a documented rule.

## Generated truth

Once SQLite migrations exist, generate `docs/generated/db-schema.md` from the actual migration set or schema introspection. CI must verify that regeneration produces no diff. Do not manually maintain a second schema description that can silently diverge.

The same rule may later apply to CLI reference output, manifest JSON Schema, or exit-code tables. Generate only artifacts for which the generation path is deterministic and checked.

## Architectural checks

As package boundaries stabilize, encode dependency rules in `repoctl arch-check` rather than relying only on prose. Initial rules should prevent at least:

```text
internal/domain       -> must not import cli, sqlite, gitcli, or compose adapters
internal/app          -> may depend on domain and interfaces, not concrete CLI formatting
internal/runtime/*    -> must not import internal/cli
internal/store/*      -> must not own orchestration policy
internal/cli          -> delegates to app use cases rather than implementing lifecycle logic
```

The exact graph may change based on implementation evidence, but every change to the permitted dependency direction must update `ARCHITECTURE.md`, its structural check, and an ADR or ExecPlan decision log.

## Agent-oriented diagnostics

Repository validators should not merely report that a rule failed. Include the violated invariant, the relevant file/package, and the expected repair direction. Examples:

```text
AGENTENV-DOC-001: docs/design-docs/foo.md is not linked from docs/design-docs/index.md.
Repair: add an indexed description or move the file to an archival directory.

AGENTENV-ARCH-002: internal/domain imports internal/runtime/compose.
Repair: move the external runtime dependency behind an interface owned by domain/app.
```

Stable diagnostic codes are useful for agents, tests, and future documentation.

## Knowledge promotion loop

Use this maintenance path:

```text
one-off discovery
  -> active ExecPlan discovery/decision log
  -> durable design doc or ADR when generally relevant
  -> test/validator/repoctl check when recurrence is preventable
  -> AGENTS.md pointer only when every agent must know where to find it
```

Do not respond to every discovery by enlarging `AGENTS.md`.

## Handoff ingestion

If `CHATGPT_HANDOFF_30.md` is checked into the repository, treat it as historical input. After the active plan and durable documents contain the necessary knowledge:

1. copy or move it to `docs/references/handoffs/CHATGPT_HANDOFF_30.md`;
2. add a short header stating which ExecPlan and documents supersede it as operational authority;
3. retain it for provenance unless the repository owner explicitly requests deletion;
4. never require future agents to reread the entire handoff for ordinary work.


---

# Cross-platform constraints

## General

- Core source must compile on `windows`, `darwin`, and `linux`.
- Do not rely on `/tmp`; use Go temporary and platform directory APIs.
- Do not parse command output affected by localization when a porcelain/JSON form exists.
- Do not assume newline is `\n`; scanners and log files must tolerate CRLF.
- Do not assume executable names end without `.exe`, `.cmd`, or `.bat`.
- Do not use Unix signals as the sole cancellation mechanism.
- Do not use Unix-domain sockets as a required control channel.
- Do not use symlinks as a required state-layout feature.
- Do not assume case-sensitive paths.
- Do not assume Docker Engine is local; respect the active Docker context, but record it in evidence.

## Windows

- Native Windows is supported with Git for Windows and Docker Desktop.
- Use `filepath` and normalized absolute paths; test repositories under paths containing spaces.
- Isolate process execution in `execx` with Windows-specific files guarded by build tags.
- Test `.cmd`/`.bat` invocation because tools such as Flutter or npm may resolve to wrappers.
- Do not mix Windows worktree paths with WSL-side Git or Docker paths in one lease. Detect and reject obviously mixed path styles where possible.
- Do not depend on POSIX file locking. Prefer SQLite transactions and, if a separate host lock is required, implement a platform abstraction.
- If generic managed process runtimes are added later, use Windows Job Objects to manage child process trees rather than attempting to emulate Unix process groups.

## macOS

- Support Intel and Apple Silicon builds where dependencies permit.
- Store durable state under Application Support, not `/tmp`.
- Docker Compose usually reaches Docker Desktop; record the Docker context and daemon information.
- Android Emulator availability and hardware acceleration are host concerns for the later adapter.

## Linux

- Support Docker Engine or compatible Docker CLI/Compose v2 setups.
- Rootless Docker should work where Compose itself works, but do not make rootless mode mandatory.
- Use XDG state/cache variables where present.

## WSL

WSL is not forbidden, but it is treated as Linux. A WSL lease should use WSL-side repositories, Git, Docker connectivity, and paths consistently. Controlling a Windows-host Android Emulator from WSL is deferred and must not be advertised as part of the MVP.

---

# Host policy

Add a host policy file under the agent-env state/config area. A minimal policy schema may be:

```yaml
version: 1

lease:
  default_ttl: 4h
  max_ttl: 24h
  max_active: 8

compose:
  forbid_privileged: true
  forbid_host_network: true
  forbid_docker_socket: true
  forbid_container_name: true
  allow_absolute_binds: false

filesystem:
  allowed_external_roots: []

runtime:
  max_parallel_creates: 2
```

Repository manifests express requested mechanism. Host policy decides what is permitted on the machine.

The MVP may use built-in defaults and only partially expose policy configuration, but the policy evaluation must not be entangled with YAML parsing or Compose execution.

---

# Planning and allocation workflow

## `agent-env plan`

`plan` must not mutate Git, Docker, or SQLite lease state.

It should:

1. locate and parse `.agent-env.yaml`;
2. validate schema and references;
3. identify source repositories;
4. resolve requested refs to commits without creating worktrees;
5. resolve stack roots to component dependency closure;
6. identify required runtime operations;
7. validate host policy and prerequisites where possible;
8. show a deterministic plan and diagnostics.

Example human output:

```text
Repository: C:\src\control-repo
Stack:      dashboard

Sources:
  backend   requested=refs/pull/3/head   resolved=abc1234...

Components:
  1. api
  2. dashboard

Runtime:
  backend   compose
  files:    infra/compose.yaml, infra/compose.agent.yaml
  services: db, api, dashboard

Warnings:
  none
```

## `agent-env create`

Conceptual saga:

```text
reserve lease ID and resource names
  -> persist requested lease and event
  -> resolve and persist complete source set
  -> create all worktrees
  -> render and validate runtime plans
  -> create/start Compose project
  -> run readiness checks
  -> persist observed resources and evidence
  -> mark ready
```

On failure, compensate in reverse order. If compensation is incomplete, preserve the lease as quarantined.

Do not expose an environment as `ready` before all required components pass readiness.

## `agent-env destroy`

Conceptual saga:

```text
mark releasing
  -> prevent new commands
  -> collect final logs/config
  -> stop/down Compose project
  -> inspect tracked worktree changes
  -> remove safe worktrees
  -> retain artifacts by policy
  -> mark released
```

---

# Compose adapter details

## Invocation

Build arguments directly. A representative invocation is:

```text
docker compose
  -p <normalized-project-name>
  --project-directory <absolute-worktree-project-directory>
  -f <absolute-compose-file-1>
  -f <absolute-compose-file-2>
  up -d <selected-services...>
```

Run `docker compose config` before `up`; save its rendered output and hash.

## Service selection

Merge `compose_services` from the resolved component closure while preserving deterministic order and eliminating duplicates.

Do not assume component name equals Compose service name.

## Readiness

Support at least:

- Compose container state/health inspection;
- HTTP probe declared in the manifest;
- command probe declared as argv array.

Use bounded timeouts and include the failing probe in diagnostics.

## Ports

The MVP should strongly prefer one of:

- no host publishing, with tests running inside Compose;
- dynamic host publishing configured in Compose;
- a generated Compose override for declared endpoint target ports.

Do not silently start a second lease when a fixed host port collision is known. Either generate an isolated override or fail before `up` with an actionable diagnostic.

Generated overrides belong under the lease’s generated directory and their content digest must be recorded.

## Inspection

Record:

- Compose project name;
- Docker context;
- selected services;
- container IDs;
- image references and image IDs/digests where available;
- networks and volumes attributable to the project;
- health/status;
- discovered endpoint mappings.

Identify project resources by explicit project name and Compose labels, not by guessed container-name formatting alone.

---

# Reconciliation and garbage collection

## Reconciliation rules

The reconciler compares desired records with observed resources and produces diagnostics/events rather than directly mutating everything it finds.

Examples:

```text
lease ready + project absent        -> degraded
lease active + worktree absent      -> degraded
lease released + project present    -> cleanup_failed/quarantined
no lease + project with agent label -> orphaned external resource
expired active lease                -> stale candidate
```

Where possible, generated resources should carry an `agent-env` label in addition to Compose’s project labels.

## GC rules

GC candidates:

- lease expired beyond grace period;
- no active command run;
- no recent heartbeat;
- resource identities match registry records;
- worktrees have no unexpected tracked modifications.

Default `agent-env gc` prints the proposed actions. `--apply` performs them.

Artifacts have independent retention from active runtime resources. Do not delete evidence merely because containers and worktrees were released.

---

# Security and trust model

## MVP trust level

The MVP is intended for trusted internal repositories and controlled PRs.

It must not claim safe execution of arbitrary external pull requests.

## Manifest authority

A PR may modify `.agent-env.yaml`. Executing that modified manifest is equivalent to accepting new code-execution instructions.

For the MVP, support an explicit manifest path and record its source commit/digest. Prefer loading the manifest from the control checkout supplied by the user, not silently from an untrusted target ref.

A future trust mode may use:

```text
trusted base-branch manifest
+
limited PR-controlled overlay
```

## Compose checks

Normalize Compose configuration and evaluate host policy before starting. At minimum, detect:

- `privileged: true`;
- `network_mode: host`;
- Docker socket mounts;
- host root or broad host-directory mounts;
- unexpected devices;
- fixed `container_name`;
- static host ports that collide across leases.

## Secret handling

- Do not copy the complete parent environment into evidence.
- Do not serialize secret-bearing values into SQLite metadata.
- Redact configured key names and token patterns in logs.
- Do not inject host credentials into target containers by default.

---

# Attempted approaches and conclusions

These are design approaches considered during the discussion. Preserve the conclusions unless implementation evidence justifies an ADR changing them.

## 1. Per-repository lifecycle shell scripts

Considered:

```text
./tools/review-pr 3
```

Conclusion: useful as a repository hook, but insufficient as the shared control plane. It duplicates lease tracking, cleanup, list/reconcile behavior, and cross-platform concerns. `agent-env` should call repository-declared commands, not require every repository to reinvent allocation.

## 2. A thin Docker Compose wrapper

Considered: expose only `docker compose -p <name> up/down`.

Conclusion: too narrow. Source isolation, multiple repositories, ownership, TTL, evidence, stack resolution, and accurate environment listing require a first-class lease model.

## 3. Compose-file auto-discovery as authority

Considered: search parent directories or `find` for `compose.yaml`.

Conclusion: rejected. Monorepositories and examples often contain multiple Compose files. Discovery is acceptable only for `agent-env init`; `.agent-env.yaml` is authoritative.

## 4. One maximal environment for every task

Considered: always start mobile + API + Dashboard.

Conclusion: rejected. Android Emulator and UI services are expensive and often unnecessary. Use components plus named stacks and select the smallest stack satisfying the task.

## 5. Stack inheritance

Considered: `full extends mobile, dashboard`, with nested profile inheritance.

Conclusion: deferred/rejected for the MVP. Deep inheritance obscures the resolved environment. Components own dependencies; stacks are explicit root sets.

## 6. One commit column per lease

Considered: store `leases.commit_sha`.

Conclusion: rejected. A lease may combine multiple repositories. Use `lease_sources` and a source-set digest.

## 7. JSON files as the only registry

Considered: one `metadata.json` per environment.

Conclusion: rejected as the authority for concurrent agents. Keep human-readable environment snapshots, but use SQLite transactions and uniqueness constraints for coordination.

## 8. Push every image build to a local registry

Considered: push every intermediate Compose build for perfect history.

Conclusion: rejected due to unbounded storage growth. Record source/config/runtime identities in the MVP; later promote only artifacts referenced by meaningful test/review evidence.

## 9. Make review worktrees filesystem read-only

Considered: `chmod` or equivalent to enforce review-only behavior.

Conclusion: rejected. Build systems need writable generated/output directories and Windows does not share POSIX permission semantics. Enforce review policy by tracked-file diff/status checks.

## 10. Put lifecycle logic in an agent skill

Considered: teach each agent skill how to manage worktrees, Compose names, ports, and cleanup.

Conclusion: rejected. Skills should express policy—what to verify—while `agent-env` implements the mechanism safely and consistently.

## 11. Implement a daemon first

Considered: long-running local service with RPC and background heartbeats.

Conclusion: rejected for the MVP. CLI + SQLite + reconciliation is sufficient and substantially easier to deploy across three operating systems.

## 12. Implement generic host processes before Compose

Considered: support arbitrary long-running process runtimes first.

Conclusion: deferred. Cross-platform process-tree termination is materially harder, especially on Windows. Compose has stable external resource identifiers and lifecycle commands, making it the better first adapter.


## 13. One large `AGENTS.md`

Considered: put the full architecture, CLI contract, all platform guidance, and the implementation plan in one root instruction file.

Conclusion: rejected. It consumes context, gives every rule equal apparent priority, becomes stale, and is difficult to index or mechanically validate. Use a short map plus indexed authoritative documents.

## 14. Keep implementation state only in chat or the handoff

Considered: let Codex repeatedly consult `CHATGPT_HANDOFF_30.md` and conversation context rather than maintaining a repository plan.

Conclusion: rejected. A future agent run may not have this chat. Convert the handoff into a self-contained active ExecPlan and durable repository knowledge before deep implementation.

## 15. Documentation without executable checks

Considered: describe package boundaries, generated schema, documentation ownership, and cross-platform rules only in Markdown.

Conclusion: insufficient. Use documentation for intent and context, then promote enforceable invariants into tests, `repoctl`, or CI. Checks should produce agent-oriented remediation messages.

## 16. Make/Just/Bash as the sole repository workflow

Considered: use a Makefile, shell scripts, or a third-party task runner as the primary harness interface.

Conclusion: rejected for the cross-platform baseline. A small Go `repoctl` command provides one inspectable interface across Windows, macOS, and Linux. Convenience wrappers may be added later, but they must delegate to the canonical cross-platform command.

---

# Constraints

## Product constraints

- The tool must be useful to both humans and agents.
- All destructive operations need dry-run or explicit application semantics.
- `list` must reflect actual state, not just intended state.
- A failed cleanup must remain visible.
- A lease must pin immutable source commits.
- Multiple leases of the same repository and commit must be able to coexist.
- Multiple repositories must be representable in one lease.
- Repository-specific details must remain in the manifest.
- The smallest useful stack should be selectable.
- Machine-readable output is mandatory.
- The repository must remain understandable and operable without access to this chat.
- `AGENTS.md` must remain a concise map; durable detail belongs in indexed documents or executable checks.
- This implementation must maintain a self-contained active ExecPlan until completion.
- Repository-development commands must have a native cross-platform path.

## Engineering constraints

- Native Windows, macOS, and Linux builds.
- No required POSIX shell.
- No CGO in release builds.
- No mandatory daemon.
- No ORM.
- No reliance on unstable human-oriented command output where porcelain/JSON output exists.
- No automatic force deletion of dirty worktrees.
- No silent use of fixed global Compose names.
- No claim that leases sandbox malicious code.
- No critical development workflow that exists only in Bash, Make, PowerShell, or a platform-specific script.
- No manually maintained generated schema document without drift detection.
- No architectural rule that is described as enforced unless a test or validator actually enforces it.

## Git workflow constraints

- Work on `feat/agent-env-mvp`.
- Commit by coherent feature slice.
- Do not rewrite public history.
- Do not combine mechanical formatting with unrelated behavior changes when avoidable.
- Add tests in the same commit as behavior.
- Push continuously if the remote is already available; do not create or reconfigure a remote without explicit authority.
- Keep `docs/exec-plans/active/agent-env-mvp.md` synchronized with actual progress, discoveries, validation evidence, and next work.
- When a durable decision is finalized, record it in an ADR or design document rather than leaving it only in a commit message or review comment.

---

# MVP scope

## Must implement

1. Repository-native harness bootstrap: concise `AGENTS.md`, `ARCHITECTURE.md`, indexed `docs/`, `docs/PLANS.md`, active ExecPlan, initial ADRs, and cross-platform `tools/repoctl`.
2. Go CLI skeleton and version command.
3. Platform state-path resolution with `AGENT_ENV_HOME` override.
4. SQLite database, migrations, repositories, leases, sources, components, resources, events, command runs, and artifacts.
5. `.agent-env.yaml` parsing with strict validation.
6. Component dependency resolution and stack planning.
7. Multiple local Git source repositories and immutable ref-to-commit resolution.
8. Detached worktree materialization for review leases.
9. Compose v2 runtime adapter with unique project names and selected services.
10. Compose config rendering, digesting, and basic policy diagnostics.
11. Create saga with compensation and event recording.
12. `list`, `show`, `renew`, `destroy`, `reconcile`, `gc`, and `doctor`.
13. Named test execution with log/evidence capture.
14. Human table output and stable JSON output.
15. Unit tests and fixture-driven integration tests.
16. Cross-platform build/test CI.
17. README, `ARCHITECTURE.md`, manifest documentation, security limitations, and Windows notes.
18. `repoctl docs-check`, `generated-check`, and an initial `arch-check` with stable diagnostic codes.
19. CI that invokes the repository harness and verifies the active plan, document indexes, generated schema, and architecture checks remain coherent.

## Should implement if the vertical slice is stable

- `agent-env init` candidate manifest generation for simple Compose repositories.
- Generated Compose override for dynamic loopback endpoint publishing.
- `--mine` filtering by owner.
- lightweight Compose service health/readiness checks.
- environment descriptor JSON under each lease directory.
- recording actual image IDs/digests used by running services.

## Explicitly out of scope for the first MVP

- Android Emulator lifecycle;
- Flutter APK build/install;
- browser/CDP control;
- UI snapshot commands;
- remote Git credential management;
- GitHub/GitLab provider-specific PR shorthand;
- untrusted fork sandboxing;
- local OCI registry management;
- automatic image promotion;
- environment checkpoint/clone;
- live stack expansion/shrink;
- generic long-running host-process adapter;
- distributed/multi-host lease coordination;
- GUI/TUI.

---

# Acceptance criteria

## Core behavior

1. A fixture repository with `api` and `dashboard` components can validate successfully.
2. `plan --stack api` resolves only the API dependency closure.
3. `plan --stack dashboard` resolves API plus Dashboard in deterministic topological order.
4. Invalid cycles and unknown references fail with actionable diagnostics.
5. `create` records the exact requested ref and resolved commit before runtime startup.
6. Two simultaneous leases from the same repository and commit receive distinct worktrees and Compose project names.
7. Destroying one lease does not alter the other lease’s containers, volumes, networks, or worktree.
8. `list --output json` returns both leases with source, stack, component, desired, and observed state.
9. If a Compose project is manually stopped or removed, a subsequent list/reconcile marks the lease degraded rather than still ready.
10. A create failure after worktree creation triggers compensating cleanup; if cleanup fails, the lease remains visible as quarantined.
11. A dirty tracked worktree is not silently deleted by GC.
12. `gc` without `--apply` deletes nothing.
13. A named test streams output, records exit code, and stores stdout/stderr evidence.
14. Multiple local repository sources are resolved and their commit tuple is visible in `show` and JSON output.

## Cross-platform behavior

15. The CLI compiles with `CGO_ENABLED=0` for at least:

    ```text
    windows/amd64
    darwin/amd64
    darwin/arm64
    linux/amd64
    linux/arm64
    ```

16. Unit tests pass on Windows, macOS, and Linux CI runners.
17. Paths containing spaces and Unicode are covered by tests.
18. No test requires Bash on Windows.
19. Command arguments containing spaces and quotes survive round-trip execution on each OS.
20. `doctor` reports missing `git`, `docker`, or Compose v2 without a panic or misleading success.

## Documentation and safety

21. README includes an explicit statement that environment isolation is not a malicious-code sandbox.
22. `ARCHITECTURE.md` explains the lease/source/runtime/reconciliation boundaries without duplicating low-level implementation details.
23. `.agent-env.yaml` schema and examples are documented.
24. Destructive commands document dry-run, force, and quarantine behavior.
25. Deferred Android/browser/registry features are documented as roadmap items, not presented as implemented.
26. `AGENTS.md` is no more than 150 lines, acts as a map, and all repository paths it references exist.
27. `docs/exec-plans/active/agent-env-mvp.md` contains the mandatory living-plan sections and accurately identifies current progress and next actions throughout implementation.
28. All design documents, product specifications, and ADRs are discoverable through their local indexes; a deliberately unindexed file causes `repoctl docs-check` to fail with an actionable diagnostic.
29. `go run ./tools/repoctl check` runs without Bash, Make, or PowerShell as a requirement on Windows, macOS, and Linux.
30. `repoctl generated-check` detects a deliberate drift in `docs/generated/db-schema.md` after migrations exist.
31. `repoctl arch-check` detects at least one fixture or synthetic forbidden dependency and explains the expected repair direction.
32. The checked-in active ExecPlan plus repository documents are sufficient for a fresh Codex run to identify the branch, current milestone, required commands, acceptance behavior, and recovery path without consulting this chat.
33. At completion, the ExecPlan is moved to `docs/exec-plans/completed/` with an outcomes/retrospective entry; historical handoff provenance remains under `docs/references/` if committed.

---

# Testing strategy

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

# Suggested implementation sequence

## Milestone 0 — Repository harness and bootstrap

Before implementing lease behavior, make the repository itself usable by a fresh coding agent.

- inspect the current repository and create branch `feat/agent-env-mvp`;
- preserve/import this handoff under `docs/references/handoffs/` if it is committed;
- create concise `AGENTS.md` and top-level `ARCHITECTURE.md`;
- create `docs/index.md`, `docs/PLANS.md`, indexed design/product/ADR directories, and the initial core-beliefs document;
- create and begin maintaining `docs/exec-plans/active/agent-env-mvp.md`;
- record initial ADRs for Go, SQLite, Compose-first runtime, and repository-native harness;
- add the cross-platform `tools/repoctl` skeleton with `doctor`, `check`, and documentation validation;
- initialize the Go module and root/version command;
- add CI that invokes `repoctl` on Windows, macOS, and Linux;
- verify `CGO_ENABLED=0` cross-builds;
- prove that a deliberately broken doc link or malformed active plan causes the harness check to fail with a repair message.

Observable result: a fresh agent can read `AGENTS.md`, locate the active plan, run one documented check command, and understand the product and architecture boundaries before product implementation begins.

Suggested commits:

```text
docs: establish repository-native agent harness
chore: bootstrap cross-platform agent-env CLI
```

## Milestone 1 — Domain, paths, and SQLite

- implement platform state paths and `AGENT_ENV_HOME`;
- add domain types and state transition validation;
- add embedded migrations and SQLite store;
- add repository, lease, source, resource, event, and command-run repositories;
- enable foreign keys and busy timeout;
- add store tests.

Suggested commits:

```text
feat: add cross-platform state directory resolution
feat: add SQLite lease registry and migrations
```

## Milestone 2 — Manifest and stack planner

- strict YAML decode;
- schema validation;
- component dependency graph;
- deterministic topological closure;
- stack planning;
- `validate` and `plan`, including JSON output;
- fixture tests.

Suggested commits:

```text
feat: define repository environment manifest
feat: resolve component dependencies for stacks
```

## Milestone 3 — Cross-platform command runner and Git source provider

- `execx` abstraction;
- stdout/stderr streaming and capture;
- Windows argument/batch handling tests;
- Git prerequisite checks;
- canonical repository identity;
- ref resolution;
- detached worktree creation/inspection/removal;
- multiple local sources;
- source-set digest.

Suggested commits:

```text
feat: add cross-platform external command runner
feat: materialize pinned Git source worktrees
```

## Milestone 4 — Compose runtime

- Compose v2 prerequisite detection;
- project-name normalization and reservation;
- absolute Compose path resolution;
- selected service planning;
- normalized config render/hash;
- basic safety and multi-instance diagnostics;
- up/inspect/logs/down operations;
- runtime resource persistence.

Suggested commits:

```text
feat: add isolated Docker Compose runtime
feat: inspect and record Compose runtime resources
```

## Milestone 5 — Create/destroy saga

- lease reservation;
- source materialization;
- runtime creation;
- readiness;
- compensation on failure;
- quarantine on unsafe/incomplete cleanup;
- `create`, `show`, `destroy`, and JSON descriptor.

Suggested commit:

```text
feat: orchestrate environment lease lifecycle
```

## Milestone 6 — Listing, reconciliation, TTL, and GC

- accurate list with lightweight inspection;
- cached list mode;
- full reconcile;
- renew/heartbeat;
- expiration;
- `gc` dry-run by default and `--apply`;
- orphan/degraded/quarantine diagnostics.

Suggested commits:

```text
feat: reconcile registered leases with actual resources
feat: add conservative lease garbage collection
```

## Milestone 7 — Named tests and evidence

- test lookup and stack compatibility check;
- command execution within source worktree;
- streaming plus persistent logs;
- command-run and artifact records;
- logs command;
- redaction basics.

Suggested commit:

```text
feat: run named tests and retain execution evidence
```

## Milestone 8 — Integration tests and documentation

- two-lease Compose fixture;
- rollback/quarantine cases;
- multi-repo fixture;
- path-with-spaces tests;
- complete README, manifest, lifecycle, security, and platform docs;
- final cross-platform build matrix.

Suggested commits:

```text
test: cover concurrent isolated Compose leases
docs: document manifests lifecycle and trust boundaries
```

---

# Unresolved issues

Do not block the MVP on all of these. Record decisions as ADRs when they become concrete.

## Repository harness

- Confirm the final hard limit for root `AGENTS.md`; the current decision is 150 lines with an 80–120 line target.
- Decide whether document freshness past `last_verified` is warning-only or CI-blocking after the initial MVP.
- Decide whether nested `AGENTS.md` files are ever needed; default is no for the MVP.
- Refine the first enforced package-dependency graph after the package layout exists.
- Decide whether generated CLI reference and manifest JSON Schema should join the generated DB schema in the MVP or immediately after it.
- Decide whether historical handoffs remain in the main branch indefinitely or move to a separate archival policy later.

## Repository and release metadata

- Confirm the final repository/module path; current assumption is `github.com/mahcialet/agent-env`.
- License has not been selected.
- Release packaging method is undecided: GitHub Releases, package managers, or both.

## Manifest trust

- Decide how base-branch/trusted manifests are selected for PR reviews.
- Define the exact overlay fields an untrusted target ref may change.
- Decide whether `--manifest-ref` is needed.

## Remote sources

- Authentication and credential passthrough for HTTPS/SSH remotes.
- Bare mirror/cache lifecycle.
- Generic ref syntax versus GitHub/GitLab PR shorthand.
- Behavior when a requested ref is force-updated during planning.

## Windows command execution

- Final handling for `.cmd` and `.bat` wrappers with arbitrary argv.
- Whether to use `golang.org/x/sys/windows` argument helpers directly.
- Future process-tree management with Job Objects for non-Compose runtimes.

## Compose isolation

- Exact policy for fixed host ports: hard failure versus generated override.
- Support and policy for external networks/volumes.
- Whether to support Podman Compose later through a separate adapter.
- Minimum supported Docker Compose v2 version.
- How much of `docker compose config` normalized output is stable enough to persist as structured evidence.

## State and concurrency

- Whether a separate cross-process host lock is needed beyond SQLite transactions and uniqueness constraints.
- Recovery semantics if the CLI is killed between an external side effect and the following database event.
- Event compaction and retention.
- Schema migration rollback policy.

## Review and fix workflows

- Branch naming and ownership for fix leases.
- Per-source writable selection in multi-repo environments.
- Whether a review lease can be forked into a fix lease while preserving a reproduction checkpoint.

## Android and browser extensions

- Emulator slot pool versus per-lease AVD data directories.
- Cross-platform Android SDK discovery.
- Windows-host Emulator control versus WSL clients.
- Flutter `.bat` execution details.
- UIAutomator/Flutter semantics snapshot format.
- CDP/browser adapter ownership and whether browser resources run inside Compose or on the host.

## Artifact reproducibility

- Local OCI registry implementation and configuration.
- Promotion policy for the final image used by a test run.
- Reference-based image retention and Registry garbage collection.
- Recording base-image digests and toolchain identity.
- Rebuild comparison versus exact-artifact replay.

## CI infrastructure

- Native Docker Compose integration coverage on macOS and Windows may require self-hosted runners.
- Android Emulator integration will require hardware acceleration and platform-specific runners.

---

# Next actions

Codex should execute the following now:

1. Inspect the current repository, branch, remotes, and working tree without rewriting history or changing remote configuration.
2. Create and switch to `feat/agent-env-mvp`.
3. Establish the repository-native harness before substantive product code: `AGENTS.md`, `ARCHITECTURE.md`, indexed `docs/`, `docs/PLANS.md`, initial ADRs, and `tools/repoctl`.
4. Convert this handoff into the self-contained living plan at `docs/exec-plans/active/agent-env-mvp.md`; record actual starting state, commands, acceptance behavior, progress, discoveries, and decisions.
5. Archive this handoff under `docs/references/handoffs/` if it is added to the repository, and mark the active plan plus durable docs as operational authority.
6. Initialize or verify the standalone Go module with language baseline Go 1.26 and CI coverage for current supported Go 1.26.x and 1.27.x.
7. Make `go run ./tools/repoctl check` pass on the initial harness, including a test proving broken documentation is detected.
8. Implement Milestones 1 through 3 and update the ExecPlan after each independently verifiable slice.
9. Complete the Compose-backed create/list/show/destroy vertical slice; do not hide missing prerequisites or failed compensation.
10. Add reconciliation, dry-run GC, named tests, and evidence retention.
11. Add fixture-based concurrency tests demonstrating two simultaneous leases and multi-repository commit pinning.
12. Verify native cross-builds with `CGO_ENABLED=0` for Windows, macOS, and Linux, and run the repository harness on each available CI platform.
13. Promote any recurring implementation/review rule into a validator, structural test, or `repoctl` check rather than enlarging `AGENTS.md`.
14. When all acceptance criteria pass, complete the ExecPlan retrospective and move it from `active/` to `completed/`.
15. Produce a final implementation report containing:

    - files and architecture added;
    - commands implemented;
    - tests executed and their results;
    - cross-platform builds verified;
    - known limitations;
    - unresolved issues moved to ADRs or roadmap;
    - commit list;
    - whether changes were pushed and to which branch.

Do not start Android Emulator, CDP/browser, remote Git cache, or local registry implementation until the Compose MVP acceptance criteria pass.

---

# Definition of done for this handoff

This handoff is complete when the branch contains a usable CLI that can:

```text
1. orient a fresh agent through a concise AGENTS.md, indexed repository knowledge, and a completed self-contained ExecPlan;
2. run a cross-platform repository check command that validates code, docs, generated truth, and architecture rules;
3. read an explicit repository manifest;
4. resolve a small stack into its required components;
5. pin one or more local repositories to exact commits;
6. create isolated worktrees;
7. start a uniquely named Compose project;
8. record the lease and actual resources in SQLite;
9. list and reconcile active environments;
10. run a named test with retained evidence;
11. conservatively destroy or quarantine the lease;
12. compile natively for Windows, macOS, and Linux without CGO.
```

The architecture must leave a clear adapter path for Flutter + Android Emulator and browser/CDP support, but those extensions are not required to call the MVP complete.

---

# Official references

- OpenAI Harness Engineering, especially repository knowledge as a system of record: <https://openai.com/ja-JP/index/harness-engineering/>
- OpenAI Codex ExecPlan guidance: <https://developers.openai.com/cookbook/articles/codex_exec_plans>
- matklad on using `ARCHITECTURE.md` as a concise code map: <https://matklad.github.io/2021/02/06/ARCHITECTURE.md.html>
- Go release history and support policy: <https://go.dev/doc/devel/release>
- Docker Compose project-name isolation: <https://docs.docker.com/compose/how-tos/project-name/>
- Docker Compose CLI reference: <https://docs.docker.com/reference/cli/docker/compose/>
- Docker Compose multiple-file behavior: <https://docs.docker.com/compose/how-tos/multiple-compose-files/>
- Git worktree documentation and porcelain output: <https://git-scm.com/docs/git-worktree>
- Git status porcelain format: <https://git-scm.com/docs/git-status>
- Go operating-system-aware path APIs: <https://pkg.go.dev/os>
- Go path handling: <https://pkg.go.dev/path/filepath>
- Go external process execution: <https://pkg.go.dev/os/exec>
- CGo-free SQLite driver: <https://pkg.go.dev/modernc.org/sqlite>
- Go Windows system interfaces and argument escaping: <https://pkg.go.dev/golang.org/x/sys/windows>
- Android Emulator command-line reference for the planned adapter: <https://developer.android.com/studio/run/emulator-commandline>
