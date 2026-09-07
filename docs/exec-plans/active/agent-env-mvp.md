---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# agent-env MVP execution plan

## Purpose / Big Picture

Deliver a usable native Windows/macOS/Linux CLI that reads an explicit target manifest, resolves minimal component stacks and pinned local Git source tuples, creates detached worktrees and distinct Compose projects, persists leases/resources/events/evidence in SQLite, reconciles real resources, runs named tests, and safely destroys or quarantines environments. Deliver the repository-native harness before substantive product code. Android/browser/registry extensions remain deferred until all MVP acceptance passes.

## Progress

- [x] 2026-09-07: inspected initial repository; dedicated branch is `feat/agent-env-mvp`; original working tree contained only local bootstrap inputs besides existing repository assets.
- [x] 2026-09-07: preserved existing MIT LICENSE; archived the supplied handoff with SHA-256 provenance.
- [x] 2026-09-07: established architecture, indexed durable docs, initial ADRs, and this self-contained plan.
- [x] 2026-09-07: initial `go run ./tools/repoctl docs-check` passed with Go 1.27.1 (exit 0); negative checker tests and complete harness verification remain pending.
- [x] 2026-09-07: Milestone 0 local bootstrap verified: repoctl check passes (format, unit including negative fixtures, vet, docs, generated, architecture); CLI version executes; CGO_ENABLED=0 builds pass for windows/amd64, darwin/amd64, darwin/arm64, linux/amd64, linux/arm64. Native CI is configured and its results remain pending.
- [ ] Milestones 1–3: domain/paths/SQLite, strict manifest/stack planner, native command runner and pinned multi-repository Git sources.
- [ ] Milestones 4–7: Compose adapter, create/destroy saga, reconciliation/TTL/GC, named tests and evidence.
- [ ] Milestone 8: concurrency, rollback, dirty-source and multi-repository integration tests; actual native CI evidence; completion audit and plan relocation.

Current next action: integrate Compose with create/destroy and evidence, then validate concurrent real-container lifecycle behavior. Domain, paths, SQLite, strict manifest/stack and Git adapter units pass locally; native tests for this new slice remain pending. Bootstrap documents describe required contracts; their presence is not proof of product behavior. Acceptance below remains pending until direct evidence is recorded.

## Surprises & Discoveries

- 2026-09-07: bootstrap commit 327ca02 pushed to origin/feat/agent-env-mvp. CI run 34117876714 passed Linux/macOS on both Go minors and all five cross-builds; both Windows checks failed because CRLF checkout bytes differed from gofmt LF output. Format validation now normalizes CRLF only, with a regression proving actual formatting drift still fails. Native rerun pending.
- 2026-09-07: Go 1.27.1 full repoctl check passed for domain/paths/policy/config/stack/Git/SQLite/read-only CLI foundation; SQLite targeted tests also passed Go 1.26.8 and race. Compose/lifecycle/evidence remain under implementation.

- 2026-09-07: the handoff called license selection unresolved, but actual LICENSE is MIT, copyright 2026 mahcialet. Preserve it.
- 2026-09-07: Go was absent from PATH. Coordinator verified official SHA-256 downloads for Go 1.27.1 and 1.26.8; select an explicit installed toolchain or install a supported version before running commands. Do not add a toolchain directive solely to hide the missing prerequisite.
- 2026-09-07: coordinator observed Docker 29.7.2 with Compose plugin 5.5.0 and a working daemon. Treat supported Compose v2-and-later plugin behavior as the integration contract; do not reject a newer major solely for not starting with 2.

## Decision Log

- 2026-09-07, maintainers: preserve existing canonical module github.com/mahcialet/agent-env and MIT license; repository evidence supersedes historical guesses.
- 2026-09-07, maintainers: retain Go baseline 1.26.0, pure-Go SQLite, Compose-first runtime, and repository-native harness; see initial indexed ADRs.
- 2026-09-07, implementation: app orchestrates injected interfaces, domain remains adapter-free, config YAML models remain separate, independent policy evaluates normalized runtime requests, root migrations package embeds SQL. CLI may wire concrete implementations while delegating behavior to app.

## Outcomes & Retrospective

In progress. Documentation bootstrap is established. Product implementation, full repository checks, native platform results, and final acceptance are not yet certified. Record actual delivered behavior, limitations and lessons here before moving this plan to completed.

## Context and Orientation

Work on `feat/agent-env-mvp` in the standalone agent-env repository. Inspect `git status --short --branch` and remotes at resume; preserve other work and never rewrite history. Entry point is root AGENTS.md; ARCHITECTURE.md defines boundaries. docs/index.md leads to product contracts, design, security, portability, ADRs and recovery. The archived handoff is provenance, not a required external context source.

Planned code responsibilities: cmd/agent-env starts the CLI; internal/cli parses/formats and wires dependencies; internal/app owns use cases; internal/domain owns pure types; internal/config strictly decodes YAML; internal/stack resolves closure; internal/paths finds OS state; internal/execx executes argv; internal/source/gitcli manages Git; internal/runtime/compose manages Compose; internal/store/sqlite persists normalized domain records; internal/policy evaluates permission/collision rules; internal/evidence preserves redacted artifacts; internal/reconcile compares desired/observed state. tools/repoctl validates this repository. Paths describe intended package ownership until created.

## Plan of Work

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

## Concrete Steps

1. Inspect branch, diff and applicable instructions; preserve existing assets. Use the dedicated branch.
2. Complete harness and Go module bootstrap. Stable commands below must be implemented as direct argv compositions without a shell prerequisite.
3. Add domain state transitions, platform paths and explicit embedded SQLite migrations with foreign keys, busy timeout and verified WAL. Generate schema from migrations; never hand-maintain it.
4. Implement manifest/stack contract and multi-source Git pinning before runtime allocation. Test exact argv, paths, graph diagnostics and stable source-set digest.
5. Implement Compose normalized config policy and selected service closure. Add durable resource reservation/journal and create/destroy compensation, then bounded readiness and reconcile.
6. Add lifecycle commands, dry-run GC, command-run/evidence capture, and JSON contracts. Every required CLI command must do real work or report failure.
7. Run negative harness tests, unit/fake-runner tests, explicit Docker fixtures, race tests, supported-toolchain tests and five-target CGo-free cross-builds. Obtain native Windows/macOS/Linux CI evidence separately.
8. Update acceptance evidence, review final diff, commit coherent verified slices, and push to the existing authorized remote/branch when credentials permit. Do not reconfigure remotes or rewrite history.
9. Audit all 33 criteria and broader command/schema/safety requirements. Only then finish retrospective, move this plan to completed, and update links.

Canonical commands (Go on PATH):

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go run ./tools/repoctl test-unit
go run ./tools/repoctl test-integration
go run ./tools/repoctl docs-check
go run ./tools/repoctl generated-check
go run ./tools/repoctl arch-check
go run ./tools/repoctl generate
go build ./cmd/agent-env
go test -race ./...
```

`check` visibly composes gofmt verification, go test, go vet, docs, generated and architecture checks. Integration is explicit and uses disposable Git/Docker fixtures. For cross-builds set CGO_ENABLED=0 and each GOOS/GOARCH via process environment (native shell syntax differs); compile windows/amd64, darwin/amd64, darwin/arm64, linux/amd64, linux/arm64. Do not interpret cross-build success as native test execution.

## Validation and Acceptance

All items initially pending. Replace status with verified only after recording a concrete test/result, native CI job, inspected file or command output that proves the entire requirement. Product requirements beyond these numbered criteria remain mandatory in indexed specs.

| ID | Required behavior | Status / evidence |
| --- | --- | --- |
| 1 | A fixture repository with `api` and `dashboard` components can validate successfully. | Pending — no complete verification recorded. |
| 2 | `plan --stack api` resolves only the API dependency closure. | Pending — no complete verification recorded. |
| 3 | `plan --stack dashboard` resolves API plus Dashboard in deterministic topological order. | Pending — no complete verification recorded. |
| 4 | Invalid cycles and unknown references fail with actionable diagnostics. | Pending — no complete verification recorded. |
| 5 | `create` records the exact requested ref and resolved commit before runtime startup. | Pending — no complete verification recorded. |
| 6 | Two simultaneous leases from the same repository and commit receive distinct worktrees and Compose project names. | Pending — no complete verification recorded. |
| 7 | Destroying one lease does not alter the other lease’s containers, volumes, networks, or worktree. | Pending — no complete verification recorded. |
| 8 | `list --output json` returns both leases with source, stack, component, desired, and observed state. | Pending — no complete verification recorded. |
| 9 | If a Compose project is manually stopped or removed, a subsequent list/reconcile marks the lease degraded rather than still ready. | Pending — no complete verification recorded. |
| 10 | A create failure after worktree creation triggers compensating cleanup; if cleanup fails, the lease remains visible as quarantined. | Pending — no complete verification recorded. |
| 11 | A dirty tracked worktree is not silently deleted by GC. | Pending — no complete verification recorded. |
| 12 | `gc` without `--apply` deletes nothing. | Pending — no complete verification recorded. |
| 13 | A named test streams output, records exit code, and stores stdout/stderr evidence. | Pending — no complete verification recorded. |
| 14 | Multiple local repository sources are resolved and their commit tuple is visible in `show` and JSON output. | Pending — no complete verification recorded. |
| 15 | The CLI compiles with `CGO_ENABLED=0` for at least:  windows/amd64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64  | Pending — no complete verification recorded. |
| 16 | Unit tests pass on Windows, macOS, and Linux CI runners. | Pending — no complete verification recorded. |
| 17 | Paths containing spaces and Unicode are covered by tests. | Pending — no complete verification recorded. |
| 18 | No test requires Bash on Windows. | Pending — no complete verification recorded. |
| 19 | Command arguments containing spaces and quotes survive round-trip execution on each OS. | Pending — no complete verification recorded. |
| 20 | `doctor` reports missing `git`, `docker`, or Compose v2 without a panic or misleading success. | Pending — no complete verification recorded. |
| 21 | README includes an explicit statement that environment isolation is not a malicious-code sandbox. | Pending — no complete verification recorded. |
| 22 | `ARCHITECTURE.md` explains the lease/source/runtime/reconciliation boundaries without duplicating low-level implementation details. | Pending — no complete verification recorded. |
| 23 | `.agent-env.yaml` schema and examples are documented. | Pending — no complete verification recorded. |
| 24 | Destructive commands document dry-run, force, and quarantine behavior. | Pending — no complete verification recorded. |
| 25 | Deferred Android/browser/registry features are documented as roadmap items, not presented as implemented. | Pending — no complete verification recorded. |
| 26 | `AGENTS.md` is no more than 150 lines, acts as a map, and all repository paths it references exist. | Pending — no complete verification recorded. |
| 27 | `docs/exec-plans/active/agent-env-mvp.md` contains the mandatory living-plan sections and accurately identifies current progress and next actions throughout implementation. | Pending — no complete verification recorded. |
| 28 | All design documents, product specifications, and ADRs are discoverable through their local indexes; a deliberately unindexed file causes `repoctl docs-check` to fail with an actionable diagnostic. | Pending — no complete verification recorded. |
| 29 | `go run ./tools/repoctl check` runs without Bash, Make, or PowerShell as a requirement on Windows, macOS, and Linux. | Pending — no complete verification recorded. |
| 30 | `repoctl generated-check` detects a deliberate drift in `docs/generated/db-schema.md` after migrations exist. | Pending — no complete verification recorded. |
| 31 | `repoctl arch-check` detects at least one fixture or synthetic forbidden dependency and explains the expected repair direction. | Pending — no complete verification recorded. |
| 32 | The checked-in active ExecPlan plus repository documents are sufficient for a fresh Codex run to identify the branch, current milestone, required commands, acceptance behavior, and recovery path without consulting this chat. | Pending — no complete verification recorded. |
| 33 | At completion, the ExecPlan is moved to `docs/exec-plans/completed/` with an outcomes/retrospective entry; historical handoff provenance remains under `docs/references/` if committed. | Pending — no complete verification recorded. |


Failure fixtures must prove docs-check detects broken links, missing plan sections and unindexed docs with repairs; generated-check detects schema drift; arch-check detects forbidden imports. Integration must prove isolated simultaneous leases and destruction, selected closure, pinning, rollback/quarantine, manual runtime removal, dirty tracked protection, evidence output, and paths with spaces. Native tests cover Windows wrappers and quote/Unicode/backslash round trips without Bash. doctor missing prerequisites must fail accurately. Record unsuccessful or unavailable checks explicitly.

## Idempotence and Recovery

Read current files and external state before repeating work. Never clean the worktree by deleting or stashing others' changes. Run tests with temporary AGENT_ENV_HOME and disposable repositories/projects, not user environments. Inspect live process handles before restart; timeouts are not proof of process death.

A failed create must retain registry intent/events and compensate in reverse order. A missing daemon, dirty tracked worktree, failed removal or uncertain identity means failure/quarantine with retained evidence. Use show/reconcile and logs to inspect before retry. GC without apply never deletes; force remains explicit. SQLite rows coordinate reservations; do not pretend SQL rolls back Docker/Git effects. Stop only dependent work when prerequisites or ownership are unresolved.

Harness failures should include invariant, file and repair direction. Fix the actual source document, migration or package import; do not weaken validators to produce green output. Schema regeneration is deterministic through repoctl generate; generated-check must fail drift without silently editing files.

## Artifacts and Notes

Provenance: docs/references/index.md and archived original handoff. Authoritative behavior: docs/product-specs/index.md. Durable rationale: docs/design-docs/index.md and docs/adr/index.md. Commands/evidence scope: docs/QUALITY.md. Current required harness documents are established. Initial docs-check passed on Linux with Go 1.27.1, exit 0. No product, native cross-platform, integration, or negative-validator verification is claimed by that result. Coordinator records subsequent command outputs and commit/CI evidence here.

## Interfaces and Dependencies

Go baseline 1.26.0, supported 1.26.x/1.27.x CI, CGo-free releases. Preferred libraries: cobra, yaml.v3, modernc.org/sqlite, ULID; no ORM. Git and docker compose are installed external tools accessed through an injected runner with executable, argv, cwd, environment, timeout, stream and capture semantics.

Source provider responsibilities are resolve/materialize/inspect/remove. Runtime responsibilities are validate/plan/create/inspect/collect/destroy, using stable external IDs rather than PID assumptions. SQLite repositories store leases, normalized sources/components/resources, events, command runs and artifacts. Reconciliation consumes observed adapters plus desired registry state. Domain/app interfaces prevent CLI formatting or concrete persistence from entering pure domain behavior.

Local bootstrap evidence (2026-09-07): Go 1.27.1 `repoctl check` exit 0, Go 1.26.8 initial `go test ./...` exit 0, official module verification successful. Five CGO-free CLI cross-builds exit 0. This verifies the bootstrap only; native CI and product acceptance remain pending.
