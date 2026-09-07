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
- [x] 2026-09-07: Milestones 1–3 foundational packages and read-only validate/plan CLI implemented and targeted Linux tests pass; native Windows/macOS execution of this slice remains pending CI.
- [x] 2026-09-07: Milestones 4–7 implemented: owned Compose snapshots, create/destroy compensation, reconciliation, TTL/GC grace and running-run protection, named tests/evidence, process-tree cancellation, source provenance and endpoint descriptors. Final integrated Go 1.26.8/1.27.1 repoctl check and Linux race pass.
- [ ] Milestone 8: concurrency, rollback, dirty-source and multi-repository integration tests; actual native CI evidence; completion audit and plan relocation.

Current next action: finish review fixes, rerun full harness and Docker integration, push coherent adapter/lifecycle slices, and obtain native CI for the final implementation. Domain, paths, SQLite, strict manifest/stack and Git adapter units pass locally; native tests for this new slice remain pending. Bootstrap documents describe required contracts; their presence is not proof of product behavior. Acceptance below remains pending until direct evidence is recorded.

## Surprises & Discoveries

- 2026-09-07: adapter commit f306bed passed Linux and five cross-builds in CI 34121460804, but native macOS/Windows exposed missing-worktree registration alias mismatch. Comparison now checks existing ancestor filesystem identity, with a passing Linux parent-symlink regression; native rerun pending. Lifecycle, GC grace, cancellation, provenance, and final integration changes are being integrated before the final acceptance audit.

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
| 1 | A fixture repository with `api` and `dashboard` components can validate successfully. | Passed locally: strict config fixture tests and real CLI concurrent integration validate the API/dashboard manifest. |
| 2 | `plan --stack api` resolves only the API dependency closure. | Passed locally: stack TestClosure and real CLI plan assert API closure only. |
| 3 | `plan --stack dashboard` resolves API plus Dashboard in deterministic topological order. | Passed locally: stack TestClosure/TestDeterminismAndNoMutation and real dashboard lease assert ordered full closure. |
| 4 | Invalid cycles and unknown references fail with actionable diagnostics. | Passed locally: TestStrictManifest cycle/reference negatives and deterministic stack tests. |
| 5 | `create` records the exact requested ref and resolved commit before runtime startup. | Passed locally: lifecycle pinning tests and real CLI create/show inspect requested refs and resolved commits; explicit control origin is independent. |
| 6 | Two simultaneous leases from the same repository and commit receive distinct worktrees and Compose project names. | Passed locally: TestIntegrationConcurrentLeasesClosureAndEvidence starts two creates concurrently and compares worktree/project identities. |
| 7 | Destroying one lease does not alter the other lease’s containers, volumes, networks, or worktree. | Passed locally: same integration compares surviving sibling exact resource IDs/worktree and unrelated volume after destroy. |
| 8 | `list --output json` returns both leases with source, stack, component, desired, and observed state. | Passed locally: concurrent integration checks versioned list JSON and both lease models. |
| 9 | If a Compose project is manually stopped or removed, a subsequent list/reconcile marks the lease degraded rather than still ready. | Passed locally: integration manually removes Compose runtime then verifies list degradation; reconcile readiness regressions cover promotion guards. |
| 10 | A create failure after worktree creation triggers compensating cleanup; if cleanup fails, the lease remains visible as quarantined. | Passed locally: actual readiness-failure integration rolls back resources/worktrees; lifecycle fake rollback failure asserts quarantine. |
| 11 | A dirty tracked worktree is not silently deleted by GC. | Passed locally: multi-repository dirty GC integration retains tracked changes and quarantines; forced cleanup preserves diff. |
| 12 | `gc` without `--apply` deletes nothing. | Passed locally: integration compares source/runtime state before and after default GC preview; expiry/heartbeat/running-run guards have unit coverage. |
| 13 | A named test streams output, records exit code, and stores stdout/stderr evidence. | Passed locally: real named pass/fail test asserts streamed output, exit5, redacted logs/argv/artifacts; cancellation evidence-failure regression now preserves the running barrier and original report. |
| 14 | Multiple local repository sources are resolved and their commit tuple is visible in `show` and JSON output. | Passed locally: TestIntegrationMultiRepositoryPinsAndDirtyGC checks each requested ref and resolved commit in show/JSON. |
| 15 | The CLI compiles with `CGO_ENABLED=0` for at least:  windows/amd64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64  | Adapter revision b153331: all five CGO-disabled build jobs passed in CI 34122233326. Final lifecycle revision pending. |
| 16 | Unit tests pass on Windows, macOS, and Linux CI runners. | Adapter revision b153331: six native OS/Go jobs passed in CI 34122233326. Final lifecycle revision pending. |
| 17 | Paths containing spaces and Unicode are covered by tests. | Passed locally and adapter native CI: path, Git and execx tests use spaces/Unicode; real Linux integration uses spaced Unicode checkout/home. |
| 18 | No test requires Bash on Windows. | Adapter native Windows jobs pass without Bash; helper commands use test executables and native wrappers. Final product native rerun pending. |
| 19 | Command arguments containing spaces and quotes survive round-trip execution on each OS. | Passed on all native adapter jobs: TestNativeRoundTrip, Windows wrapper tests and repoctl TestCommandArgvRoundTrip. |
| 20 | `doctor` reports missing `git`, `docker`, or Compose v2 without a panic or misleading success. | Passed locally: structured CLI missing-prerequisite tests and Compose doctor missing docker/old plugin/daemon fixtures. |
| 21 | README includes an explicit statement that environment isolation is not a malicious-code sandbox. | Verified README and security guide explicitly state isolation is not a malicious-code sandbox. |
| 22 | `ARCHITECTURE.md` explains the lease/source/runtime/reconciliation boundaries without duplicating low-level implementation details. | Verified ARCHITECTURE.md maps lease/source/runtime/store/app observation boundaries; architecture checker passes. |
| 23 | `.agent-env.yaml` schema and examples are documented. | Verified manifest-v1 spec and testdata/manifests examples; strict parser validates fixtures. |
| 24 | Destructive commands document dry-run, force, and quarantine behavior. | Verified CLI/security/reliability docs explain destroy preview, explicit force, dirty protection and quarantine. |
| 25 | Deferred Android/browser/registry features are documented as roadmap items, not presented as implemented. | Verified roadmap clearly defers Android, browser pooling, remote providers and external registry. |
| 26 | `AGENTS.md` is no more than 150 lines, acts as a map, and all repository paths it references exist. | Passed repoctl docs-check and negative length/path fixtures; root AGENTS.md is a concise indexed map. |
| 27 | `docs/exec-plans/active/agent-env-mvp.md` contains the mandatory living-plan sections and accurately identifies current progress and next actions throughout implementation. | Verified living plan contains mandatory sections and records bootstrap, foundation, adapters, review findings and final gates. |
| 28 | All design documents, product specifications, and ADRs are discoverable through their local indexes; a deliberately unindexed file causes `repoctl docs-check` to fail with an actionable diagnostic. | Passed repoctl docs-check and negative unindexed/link/metadata/plan fixtures. |
| 29 | `go run ./tools/repoctl check` runs without Bash, Make, or PowerShell as a requirement on Windows, macOS, and Linux. | Adapter revision native repoctl check passed all six OS/Go jobs; final product native rerun pending. |
| 30 | `repoctl generated-check` detects a deliberate drift in `docs/generated/db-schema.md` after migrations exist. | Passed TestGeneratedDrift and generated-check; migration 002 schema is regenerated from SQL. |
| 31 | `repoctl arch-check` detects at least one fixture or synthetic forbidden dependency and explains the expected repair direction. | Passed TestArchitectureBoundaries negative forbidden imports with repair diagnostics and final arch-check. |
| 32 | The checked-in active ExecPlan plus repository documents are sufficient for a fresh Codex run to identify the branch, current milestone, required commands, acceptance behavior, and recovery path without consulting this chat. | Verified plan and indexed docs contain branch, commands, current evidence, remaining gates and recovery without chat dependency. |
| 33 | At completion, the ExecPlan is moved to `docs/exec-plans/completed/` with an outcomes/retrospective entry; historical handoff provenance remains under `docs/references/` if committed. | Pending: final native verification, retrospective and completed-plan relocation. |


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

Foundation evidence (2026-09-07): targeted Go 1.27.1 tests for domain, paths, policy, config, stack, execx, source/gitcli, store/sqlite, app, cli, repoctl all pass. Domain digest/state/GC guards, path symlink/traversal cases, strict manifest negatives, deterministic API/dashboard closure, JSON envelope, concurrent SQLite capacity/locking/rollback and actual Git worktree cleanup are covered. Commands create/destroy/reconcile/test are not yet connected; no full lifecycle acceptance claimed.

Native CI evidence: bootstrap CRLF fix d755acf passed all jobs (run 34118678913). Foundation 75c0b65 run 34118745919 exposed two Windows issues: negative YAML fixture substitutions assumed LF; paths.Within accepted slash-rooted paths because filepath.IsAbs on Windows requires a volume. Tests now normalize fixture line endings and assert every negative case really changes input; CRLF manifest digest equivalence is verified; source-relative path guard explicitly rejects leading slash on all OSes. Git/execx (including .cmd/.bat roundtrip) and SQLite tests passed native Windows on both Go minors in that run.

Implementation checkpoint (2026-09-07):

- Foundation 33ac8be passed every native/cross-build job in CI run 34119012663 (Windows/macOS/Linux, both Go minors). Later lifecycle changes have not yet been pushed or certified by native CI.
- Full actual Docker suite passed in 108.181 seconds: concurrent API/dashboard leases, HTTP, named pass/fail evidence, sibling and unrelated-volume preservation, multi-repository pinning, dirty GC preview/apply quarantine/forced diff, and readiness rollback. Tests live under internal/cli/integration_test.go and testdata/compose. They must rerun after current review changes.
- Independent review identified and implementation addressed: CRLF fixtures; slash-rooted Windows paths; unselected Compose cleanup resources (snapshot pruned); indirect symlink/volume-driver host mounts; initial readiness not completed before reconcile promotion; missing Git registrations; secret literals in pinned manifest; CLI owner/exit/manifest behavior. Exact-service archived logs and strict labels are being finalized.
- SQLite now supplies operation contexts, cancellation on renewal loss/watchdog, and transaction token fences. App preserves original ownership through ordinary-cancel compensation, avoids stale cleanup after loss, and preserves local command evidence without stale registry writes. Actual token-replacement tests passed on both Go minors and race.
- Native runner now kills ordinary descendant trees before returning: Unix process groups; Windows suspended Job assignment before resume. Linux late-write regression failed before fix and passes afterward. Deliberate Unix detachment remains outside trusted-code containment; native Windows/macOS final CI pending.
- Initial documentation and all 33 acceptance rows remain active. Final audit, native verification of current code, and completion-plan relocation are not yet done.

Final integration checkpoint (2026-09-07): b153331 native path-alias repair passed all 12 jobs in CI 34122233326, including Windows/macOS/Linux on both Go minors and five CGO-disabled targets. Full lifecycle tree passes repoctl check on Go 1.26.8 and 1.27.1 and go test -race ./.... Actual Docker rerun and final lifecycle native CI are pending. Cross-process destroy cancellation is covered using two SQLite connections and an actual process tree; cleanup asserts final evidence before source removal. Manifest provenance tests use clean/dirty/untracked/ignored control files and linked worktrees; runtime refs remain independent.

Final local Docker evidence: repoctl test-integration passed; CLI suite 110.663 seconds. All three actual Docker scenarios passed; fixture lease containers, volumes, networks and unrelated-fixture volumes were absent afterward by label-filtered inventory. Independent cancellation review found evidence-finalization errors could permit deletion after a terminal run was saved; repair and regression are in progress before committing lifecycle.

Independent lifecycle review checkpoint: exact-run cancellation identity, lock acquisition races, stale-running refusal and SQLite fencing were reviewed with no additional findings. Two P1s were accepted: terminal run publication preceded successful artifact finalization, and terminal cancellation could hide unconfirmed process-tree termination. Both require a durable cleanup gate and failure regressions; they are not counted complete until fixes and re-review pass.

Cancellation review resolution: both P1 findings were fixed and independently re-reviewed with no remaining findings. Terminal run persistence now occurs only after source/lease/event and every evidence write succeed. Typed ErrProcessTreeUnconfirmed and ErrOutputIncomplete keep cleanup blocked on ambiguous termination or incomplete captured output. Concurrent force-destroy regressions retain the running record and original report on all three failure classes. Focused app/execx tests and Windows execx crosscompile pass; final harness and Docker rerun are underway.
