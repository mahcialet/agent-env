---
status: completed
owner: maintainers
last_verified: 2026-09-08
---

# Add a lease-owned generic persistent process runtime

[日本語](persistent-process-runtime.ja.md)

This ExecPlan is a living document. Maintain it according to `docs/PLANS.md`.

Expected branch: `feat/persistent-process-runtime`.

PR #8 (`feat: add explicit Podman Compose provider with safe lease cleanup`) is
merged into `master` before implementation started. This plan does not depend on
Podman behavior; Compose provider behavior remains in regression scope.

Preferred start:

- if PR #8 has merged, branch from the resulting `master` and include Compose
  provider behavior in regression scope;
- otherwise branch from current `master` and do not stack this process-runtime
  work on an unmerged Podman branch merely for convenience.

Record the actual base below.

Starting base branch: `master`
Starting revision: `01e3581` (PR #8 merge)

## Purpose / Big Picture

After this work, `agent-env` can own a long-lived native host process as a normal
environment runtime.

A repository can declare a process such as a development API server, local test
daemon, application server, or future Chromium/browser process without wrapping
it in Compose and without adding product-specific lifecycle code to `agent-env`.

Final manifest shape:

```yaml
runtimes:
  api:
    type: process
    source: app
    working_directory: .
    command:
      - ./bin/api-server
      - --listen
      - 127.0.0.1:${port:http}
    ports:
      http:
        protocol: tcp
    env:
      APP_ENV: test
      PORT: ${port:http}

components:
  api:
    runtime: api
    endpoints:
      http:
        runtime_port: http
    readiness:
      - type: http
        url: http://127.0.0.1:${endpoint:http}/health
```

The runtime/endpoint syntax above is finalized. HTTP readiness expands a declared
local `${endpoint:http}` to its numeric reserved port; this reference is not a
whole address or a component-qualified endpoint name. The architectural contract
is normative.

A process runtime:

- starts from a pinned managed source;
- executes an argv array directly with no shell interpretation;
- survives the `agent-env create` CLI process that launched it;
- writes stdout/stderr to lease-owned files;
- has durable native process-tree identity;
- receives a lease-owned runtime state directory;
- may receive dynamically allocated loopback TCP ports;
- participates in readiness, show/list, logs, reconcile, destroy and GC;
- becomes DEGRADED when its owned process exits unexpectedly;
- never automatically restarts in this slice;
- is terminated only after native identity is revalidated;
- quarantines when process-tree ownership or termination is uncertain.

The main follow-up consumer is Browser/CDP:

```text
process runtime
    +-- Chromium process ownership
    +-- private runtime/profile directory
    +-- allocated loopback CDP port
    +-- future browser observer
          DOM/ARIA snapshot
          screenshot
          navigation
          console/network
```

Browser support should not need to reinvent process lifetime, detached execution,
logs, state directories, endpoint allocation or cleanup.

## Scope

In scope:

- first-class `type: process` runtime;
- strict command/working-directory/env/port manifest contract;
- argv-only direct execution;
- source-relative confined working directory;
- source-relative or PATH-resolved executable support with evidence;
- private lease-owned runtime directory;
- `${runtime_dir}`, `${lease_id}` and named port interpolation as finally defined;
- dynamically allocated loopback TCP ports;
- process-runtime endpoints in the common component endpoint model;
- app-level readiness reuse;
- persistent stdout/stderr logs;
- durable launch intent before process effects;
- durable native process identity after launch;
- process survival after launching CLI exit;
- observation by later independent CLI invocations;
- native identity-gated termination;
- conservative escalation and uncertain-tree quarantine;
- create compensation and destroy/reconcile/GC integration;
- operation fencing and heartbeat behavior;
- concurrent process leases with distinct ports/state directories;
- process + Compose coexistence;
- Windows/macOS/Linux real native process tests;
- cross-process crash/recovery tests;
- bilingual product/design/ExecPlan docs and architecture/roadmap updates.

Out of scope:

- shell command types (`sh -c`, `cmd /c`, PowerShell);
- PTY/TTY or interactive stdin;
- terminal multiplexers;
- adopting processes started outside agent-env;
- self-daemonizing process support;
- automatic restart/supervisor policy;
- replicas;
- CPU/memory/cgroup/job resource limits;
- user switching or privilege escalation;
- systemd/launchd/Windows Service installation;
- remote/multi-host process execution;
- UDP in the first slice;
- socket activation / inherited listening sockets;
- fixed host ports;
- port discovery by parsing stdout;
- browser/CDP semantics themselves;
- security sandboxing of untrusted code.

## Architectural Intent

### Native process primitive versus runtime policy

The repository already has a native detached-process primitive in
`internal/execx` whose safety properties must be preserved:

- persistent processes are not request-context-owned;
- output is file-backed, not CLI-pipe-backed;
- native identity is stronger than PID alone;
- process-tree observation survives the launching CLI;
- Windows Job/process-group birth identity participates in ownership proof.

Do not move lease policy into `execx`.

Desired layering:

```text
internal/execx
    native Start / Observe / identity-gated Terminate

internal/runtime/process
    executable resolution
    argv/env/interpolation
    runtime directory
    stdout/stderr
    dynamic port values
    inspect/terminate

internal/app
    intent persistence
    port reservation
    readiness
    compensation
    reconcile/quarantine
    cleanup ordering
```

Android Emulator remains specialized. Do not migrate it to `type: process` in
this plan. Any `execx` changes must preserve all existing Android process tests.

### Foreground lifecycle anchor

The runtime owns one foreground lifecycle anchor. The root may spawn children,
but it is expected to remain alive for the runtime lifetime.

Self-daemonization is unsupported.

If the root exits while descendants remain, use only native evidence strong
enough to prove ownership. If lineage becomes ambiguous, report DEGRADED or
QUARANTINED and stop destructive effects. Never kill a PID/process group solely
because its numeric identifier matches a historical record.

Platform differences may remain conservative. A Windows Job may prove a tree in
a case where Unix lineage becomes uncertain; do not invent false common proof.

### No automatic restart

Unexpected process exit changes observed state to DEGRADED. The first slice does
not restart it automatically. Recovery is inspect -> safe destroy -> recreate.

### Native termination

The current detached primitive observes but does not expose generic stop policy.
Add only the minimum identity-gated native termination surface required.

Conceptual shape:

```go
type DetachedProcess interface {
    Start(context.Context, Command, stdoutPath, stderrPath string) (ProcessIdentity, error)
    Observe(context.Context, ProcessIdentity) (ProcessObservation, error)
    Terminate(context.Context, ProcessIdentity, TerminateOptions) (TerminationResult, error)
}
```

Exact types may differ.

Unix preference: request graceful group/tree termination while ownership is
proven, wait a bounded grace period, and escalate only while proof remains valid.
If the lifecycle anchor disappears and ownership becomes ambiguous, quarantine.

Windows preference: reuse existing Job/guardian identity and terminate the owned
Job/tree rather than PID-only kill. Preserve atomic completion evidence.

Do not promise graceful semantics the platform cannot prove.

### Command and executable identity

`command` is always argv. No shell expansion, pipes, redirects or chaining.

`working_directory` is source-relative and confined to the selected worktree.

argv[0] may be a source-relative executable or a bare PATH-resolved host tool.
Record, where available:

- requested argv after redaction;
- resolved executable path;
- source-vs-host origin;
- executable SHA-256 for a regular readable file;
- source commit;
- working directory.

Do not call a PATH executable reproducible merely because its path was recorded.
Windows `.bat`/`.cmd` remain unsupported for detached native runtime execution.

### Runtime-owned mutable state

Each process runtime gets a private state directory below the lease state root,
conceptually:

```text
<AGENT_ENV_HOME>/leases/<lease-id>/process-runtimes/<runtime>/
    state/
    stdout.log
    stderr.log
    owner.json
    redaction.json
    launch.json
```

Expose an explicit interpolation such as `${runtime_dir}`. Future Chromium may
use `--user-data-dir=${runtime_dir}/profile`.

Delete mutable state only after whole-process-tree absence is confirmed. Logs or
other evidence that must survive release belong in existing evidence/artifact
storage.

### Dynamic loopback TCP ports

A process runtime may request named ports:

```yaml
ports:
  http: {protocol: tcp}
  metrics: {protocol: tcp}
```

Initial contract:

- TCP only;
- loopback only;
- port allocated by agent-env;
- fixed host ports rejected;
- allocation persisted before launch;
- concurrent leases cannot receive the same agent-env reservation;
- external occupancy causes conservative failure;
- no claim of race-free exclusion against arbitrary external processes unless a
  held-socket design is actually implemented.

`${port:http}` resolves to the numeric port.

### Endpoint model

Existing component endpoints are Compose-oriented. Extend them explicitly rather
than overloading empty Compose fields.

Preferred direction:

```yaml
endpoints:
  http:
    runtime_port: http
```

Existing Compose endpoint syntax remains valid. A strict endpoint variant is
selected as either Compose service target or process runtime port.

App-level endpoint consumers, including Flutter reverse mappings, should receive
the same provider-neutral resolved endpoint representation.

### Readiness and logs

Starting a native process is not READY.

Reuse app-level readiness. A process that exits before readiness fails creation;
a live process that never becomes ready fails at the bounded readiness deadline
and is terminated conservatively.

Detached stdout/stderr are file-backed from launch. `agent-env logs` exposes
runtime-attributed output without unbounded in-memory buffering.

### Persistence ordering

Before launch:

1. source materialized;
2. runtime directory identity fixed;
3. named ports reserved;
4. launch intent persisted;
5. output paths fixed.

After launch:

6. returned native identity persisted before readiness depends on it;
7. launch evidence finalized;
8. readiness proceeds.

If persistence fails after process start, retain any returned identity and
compensate only while ownership remains proven.

## Progress

- [x] (2026-09-08) Record base branch/revision and create `feat/persistent-process-runtime`.
- [x] (2026-09-08) Run baseline repository harness and race suite.
- [x] (2026-09-08) Inspect native detached code and Android regression surface.
- [x] (2026-09-08) Inspect app/config/domain/store/endpoint/readiness/log/cleanup paths.
- [x] (2026-09-08) Write English/Japanese product and design docs.
- [x] (2026-09-08) Finalize process manifest/endpoint/interpolation contract.
- [x] (2026-09-08) Add identity-gated native termination primitive if required. (Native three-OS evidence below.)
- [x] (2026-09-08) Add native termination/identity negative regressions. (Native three-OS evidence below.)
- [x] (2026-09-08) Keep all Android detached/guardian tests passing. (Native three-OS evidence below.)
- [x] (2026-09-08) Implement strict process runtime config/domain types.
- [x] (2026-09-08) Implement executable and working-directory resolution.
- [x] (2026-09-08) Implement runtime-owned state directory.
- [x] (2026-09-08) Implement named dynamic loopback TCP port reservations.
- [x] (2026-09-08) Implement launch intent/evidence and Start.
- [x] (2026-09-08) Integrate readiness, endpoints, show and logs.
- [x] (2026-09-08) Implement Inspect/reconcile and unexpected-exit degradation.
- [x] (2026-09-08) Implement Destroy/termination/compensation/GC.
- [x] (2026-09-08) Add two-concurrent-process-lease isolation.
- [x] (2026-09-08) Add process+Compose coexistence fixture.
- [x] (2026-09-08) Add cross-process recovery through separate CLI invocations.
- [x] (2026-09-08) Add PID-reuse and root-exit-with-descendant regressions.
- [x] (2026-09-08) Prove no automatic restart occurs.
- [x] (2026-09-08) Add native Windows/macOS/Linux persistent-process integration.
- [x] (2026-09-08) Add real HTTP helper using allocated loopback endpoint.
- [x] (2026-09-08) Add browser-shaped CDP-like state-dir/port fixture.
- [x] (2026-09-08) Update bilingual architecture/portability/reliability/security/quality/roadmap.
- [x] (2026-09-08) Run final harness/race/native/cross-build verification.
- [x] (2026-09-08) Complete direct acceptance evidence and bilingual retrospective.
- [x] (2026-09-08) Move both plans to `docs/exec-plans/completed/`.


A checked item means observed completion. Record UTC date, revision, exact test or
workflow run and result.

### Final acceptance checkpoint (2026-09-08)

[Verify run 34226859965](https://github.com/mahcialet/agent-env/actions/runs/34226859965)
at `f58896050b11481b021bf1ad07701818d3c133eb` PASS: all 12 jobs succeeded.
The six native Windows/macOS/Ubuntu jobs (Go 1.26 and 1.27) ran `repoctl doctor`,
`repoctl check` and CLI build. This includes `TestPersistentProcessNativeCLI`,
managed-process and Android detached/guardian regressions; Windows also ran
`TestManagedWindowsCompletedJobIgnoresReusedHistoricalPID`. Integration ran
`go test -race ./...` and `repoctl test-integration`, both PASS. Five CI cross-build
jobs PASS; the separate local six-target CGO-disabled build evidence remains
compile-only. This result closes the native gates left open in earlier checkpoints.
The first failed run is retained below as historical evidence.

### Integration checkpoint (2026-09-08)

All results below concern the integrated working tree based on `01e3581`; final
commit/native CI success is not implied.

- The unchanged base was reconstructed in an isolated worktree after editing began:
  `go run ./tools/repoctl check` and `go test -race ./...` PASS. This is baseline
  evidence, not a claim that the complete baseline ran before implementation.
- Integrated `go run ./tools/repoctl check` PASS through all stages, including after
  the readiness-secret fix. `go test -race ./...` PASS (app 53.504s, CLI 7.749s,
  execx 8.255s, SQLite 20.320s). Final local check also PASS after the H22/H24 fixture additions (native CLI
  4.085s); targeted native CLI race PASS (5.592s).
- Linux `go test ./internal/cli -run '^TestPersistentProcessNativeCLI$' -count=1 -v`
  PASS (5.422s): independent built CLI calls, symlinked home, Japanese/spaced paths,
  distinct lease ports/profiles, real CDP-like HTTP, manual root exit/degraded/no
  restart, retained logs and repeated destroy. It kills only a creating CLI during
  readiness, then expires only the isolated test's operation lock to exercise
  normal recovery while the native process survives. Interrupted allocation stays
  degraded rather than being falsely completed by show.
- The same native test adds live tracked README modification: non-force destroy
  quarantines and preserves bytes plus both HTTP servers; force captures the
  tracked diff before release. A separate direct host helper outside all leases,
  with kernel-assigned port and private profile, preserves its exact PID/profile
  through all lease cleanup/refusal paths, then receives explicit fixture teardown.
- Real Docker `TestIntegrationPersistentProcessComposeCoexistence` PASS (55.965s).
  Full `go run ./tools/repoctl test-integration` PASS; Android hardware and Podman
  opt-in suites remain gated as designed in that command.
- Separate `TestPodmanIntegrationConcurrentLeasesAndEvidence` PASS (150.407s) with
  `AGENT_ENV_PODMAN_INTEGRATION=1` and `AGENT_ENV_PODMAN_DOCKER_COEXISTENCE=1`:
  rootless Podman plus Docker coexistence regression evidence.
- `go run ./tools/repoctl doctor` and `agent-env doctor --runtime process --output json`
  PASS; process diagnostics report native detached support without a Compose dependency.
- Config/domain, architecture negative fixtures and repeated docs-check PASS.
  Five confirmed review issues were fixed with targeted regressions: death during
  readiness, proven pre-spawn failure, canonical paths before reservation, endpoint
  alias precedence, and literal readiness credentials entering snapshots. Review
  of another implementer's lifecycle/backend was independent; an author's review
  of their own config/docs was not classified as independent review.
- Native Windows/macOS execution and final native CI remain pending. Cross-builds
  are additional evidence only. This plan remains active until those gates pass.

### First published native CI checkpoint (2026-09-08)

Implementation commit `fb0d22ea9eb64798c8f70c57c69e0166aa90ccc7` was pushed.
[Verify run 34225937603](https://github.com/mahcialet/agent-env/actions/runs/34225937603)
completed with 11 of 12 jobs successful: native macOS and Ubuntu Go 1.26/1.27,
Windows Go 1.27, all cross-build jobs and integration passed. Windows Go 1.26 job
`102059952028` FAILED. The workflow is not an overall success.

The failure was `TestPersistentProcessNativeCLI` line 329 while destroying the
second manually exited process: `detached job root identity reused or ambiguous`.
Inspection found historical root PID validation before exact named-Job completion
evidence, and managed observation still inspecting the root after whole-tree
absence was proven. A fix and native negative regression are in progress:
completed owned Job evidence must take precedence over a recycled historical PID
without weakening active Job identity checks. Preserve the failure as evidence;
no successful retry or fixed-commit native result is claimed.

Local `CGO_ENABLED=0` CLI cross-builds passed all six Windows/macOS/Linux amd64/arm64
targets. These are compile-only results. Native macOS now has direct CI evidence;
final acceptance still waits for the Windows fix and its native validation.
This plan remains active; final retrospective and archival are not complete.

## Surprises & Discoveries

- 2026-09-08: The supplied English plan did not have its required Japanese
  sibling. The initial docs-check failed with missing-translation/missing-link
  diagnostics; the full Japanese translation was added in the contract milestone.
- 2026-09-08: Existing HTTP readiness had no endpoint interpolation. The new
  process-only `${endpoint:localName}` resolves to a numeric reserved port; config
  validates the declared local endpoint before substituting a placeholder only
  for URL syntax checking. Existing non-process readiness behavior stays unchanged.
- 2026-09-08: Unix birth/group observation and group signaling are not atomic.
  Revalidation immediately precedes each signal, and no further destructive action
  is allowed once the foreground anchor or tree proof becomes uncertain. This is
  a native platform limit, not proof of race-free PID/group ownership.
- 2026-09-08: Portable directory names require more than the existing generic
  identifier pattern. Process runtime names also reject Windows reserved device
  names/trailing dots and case collisions.
- 2026-09-08: Config/domain and bilingual product/design contracts are validated
  locally; native primitive, app/store lifecycle, adapter and Linux integration
  now pass integrated validation. Native Windows/macOS acceptance and final CI remain pending.

Record process identity behavior after root exit, Unix/macOS process-group limits,
Windows Job/guardian termination, executable-resolution differences, dynamic-port
races, source mutation, readiness failure with children, self-daemonization, log
lifecycle, state-directory cleanup and browser-like multiprocess behavior.

Never convert uncertain process ownership into a false clean state to make tests
pass.

- A terminal reservation guard initially rejected valid repeated destroy and anomalous post-release observations. Terminal reservation evidence must persist independently of observed lifecycle state; `Desired=released` updates never reacquire ports.
- Logs based only on current host environment lose startup redaction after that environment changes. Persist non-plaintext fingerprints before launch, independently of the launch identity receipt, so receipt-write failure still permits safe diagnostic capture and identity-based compensation.


- 2026-09-08: An early fixed-range port allocator contradicted the dynamic contract
  and could repeatedly encounter the same occupied first port. It was replaced
  with OS `127.0.0.1:0` listeners held until SQLite commit. The external bind race
  after listener closure remains explicit; startup does not change saved identity.
- 2026-09-08: A successful probe could outlive its process and falsely produce
  READY. A final owned native observation now runs after probes/application startup.
  Typed `ErrProcessNotStarted` plus zero PID distinguishes proven pre-spawn failure
  and returns to prepared state; an unknown zero-identity result remains uncertain.
- 2026-09-08: Prepare canonicalized symlinked homes after immutable evidence paths
  were reserved, causing SQLite rejection. `CanonicalFuture` now resolves those
  paths before reservation without weakening immutability.
- 2026-09-08: Raw runtime port keys could overwrite a declared component alias
  pointing at another port. Declared aliases now retain precedence.
- 2026-09-08: Literal inherited secrets in process readiness could enter durable
  snapshots. Validation now rejects them before snapshot publication; explicit
  host references remain supported and their output redacted.

- 2026-09-08: Published CI exposed a Windows Go 1.26 root-PID reuse/ambiguity
  failure after Job completion despite Linux/macOS and Windows Go 1.27 success.
  Completed-tree proof must precede historical root lookup; active ownership checks
  must remain strict. The targeted platform fix/regression are in progress.

## Decision Log

- Decision: Introduce a first-class `type: process` runtime.
  Rationale: long-lived host processes should participate in the same lease
  lifecycle as Compose and Android resources.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Reuse `execx` detached native identity as the low-level primitive.
  Rationale: it already survives CLI exit and uses stronger-than-PID tree
  identity on supported native OSes.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Do not migrate Android Emulator to `type: process` in this plan.
  Rationale: Android has specialized AVD/ADB/console ownership and proven cleanup.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Commands are direct argv only.
  Rationale: preserve portability, auditability and the no-shell invariant.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Initial process contract requires a foreground lifecycle anchor and
  does not support self-daemonization.
  Rationale: later CLI invocations and cleanup must be able to prove ownership.
  Date/Author: 2026-09-08 / maintainers.

- Decision: No automatic restart in the first slice.
  Rationale: restart changes resource identity and is supervisor policy, not
  basic lease lifecycle.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Process ports are dynamically allocated loopback TCP ports; fixed
  host ports are not supported.
  Rationale: concurrent lease isolation and future Browser/CDP need stable local
  dynamic endpoints.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Every process runtime receives a private mutable runtime directory.
  Rationale: browsers/dev servers need writable state without polluting pinned
  worktrees.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Termination is conservative and identity-gated.
  Rationale: quarantining is safer than killing a recycled PID or ambiguous tree.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Durable docs and this ExecPlan are bilingual.
  Rationale: repository documentation policy.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Finalize `working_directory`, argv `command`, `env` references,
  `ports.<name>.protocol: tcp`, and endpoint `runtime_port`. Reject incompatible
  fields by presence, including YAML null/empty values and merge/alias inputs.
  Command/env accept only runtime_dir, lease_id, port, and host-env references;
  argv[0]/cwd remain literal. Source-path interpolation and a TCP probe are not
  added; HTTP/command readiness is reused.
  Rationale: explicit variants retain existing Compose canonical bytes and keep
  the generic process contract small and portable.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Shell command types and implicit shell execution remain unsupported,
  but do not introduce an executable-basename blacklist for explicitly selected
  native interpreters. Reject `.bat`/`.cmd` wrappers.
  Rationale: direct argv is the invariant; a blanket interpreter restriction is
  an additional public policy outside this plan.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Use the managed native process interface above the existing detached
  primitive; Unix uses birth plus foreground group proof, while Windows retains
  the exact named Job handle through revalidation and termination.
  Rationale: keep OS mechanics out of app policy and preserve Android's detached
  behavior. Windows Job termination does not promise console-style graceful stop.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Preserve executable source/host origin, absolute path and readable
  regular-file SHA-256 as evidence; reserve ports durably before launch and retain
  ports/private mutable state whenever cleanup is uncertain. Omit restart fields.
  Rationale: evidence is not host-tool reproducibility, SQLite reservations cannot
  prevent external bind races, and future Browser/CDP consumes this lifecycle
  without adding browser policy to the process adapter.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Allocate dynamic OS ports with listeners held through reservation
  commit, canonicalize evidence paths before reservation, and preserve declared
  aliases over diagnostic endpoint keys.
  Rationale: remove fixed-range assumptions without weakening immutable identity
  or common consumer endpoint intent.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Persist versioned secret fingerprints before Start in independent
  `redaction.json`; `launch.json` stores ownership and native identity only.
  Rationale: redaction survives changed host secrets and failed post-launch receipt
  writes without plaintext persistence. Missing proof fails closed; fingerprints
  and raw logs still require private storage.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Observe the exact named Windows Job before consulting its historical
  root PID; empty/missing Jobs require matching synced guardian completion proof.
  Active Jobs retain birth/membership checks, with a second census if they complete
  during PID observation. Never signal the historical PID after proven completion.
  Rationale: the first native CI exposed post-completion PID reuse/ambiguity;
  complete Job evidence permits safe repeated cleanup without weakening ownership.
  Date/Author: 2026-09-08 / maintainers.

## Outcomes & Retrospective

Completed on 2026-09-08. H1–H33 have direct acceptance evidence below; the
English and Japanese plans are archived together. Implementation is `fb0d22e`,
with the Windows completion correction in `f588960`; its full Verify run passed.

The delivered `type: process` contract uses direct argv, confined
`working_directory`, named TCP ports and `runtime_port` endpoints. Config/domain,
SQLite reservations, the process adapter, app orchestration and `execx.ManagedProcess`
retain separate responsibilities. Private mutable state, file-backed logs, executable
origin/digest evidence and prelaunch secret fingerprints support later independent
CLI observations and safe diagnostics without persisting plaintext credentials.
Persisted intent/receipts recover interrupted creation; unexpected death degrades
without restart, and uncertain cleanup retains ports, state and worktrees.

Native tests on three OSes demonstrate concurrent lease isolation, HTTP/CDP-like
readiness, creator interruption, retained logs, source-change protection and unrelated
process survival. Unix signals require fresh birth/group proof and retain an
observation-to-signal race; ambiguous descendants quarantine. Windows terminates the
exact Job and requires matching guardian proof for completion. Initial Windows CI
exposed historical-PID lookup after Job completion. A two-process regression now
simulates recycled historical identity and proves unrelated-process survival; it does
not force kernel PID reuse. Independent review and native CI caught issues that
Linux-only success and cross-compilation could not settle.

No acceptance blockers remain. External port-bind races and native ownership limits
remain documented constraints. Self-daemonization, PTY, automatic restart and browser
semantics remain outside scope. Browser/CDP should build observation above this
process lifecycle, private profile state and endpoint contract; Android remains separate.

## Context and Orientation

Read before implementation:

- `AGENTS.md` / `AGENTS.ja.md`
- `ARCHITECTURE.md` / `ARCHITECTURE.ja.md`
- `docs/PLANS.md` / `docs/PLANS.ja.md`
- `docs/PORTABILITY.md` / `.ja.md`
- `docs/RELIABILITY.md` / `.ja.md`
- `docs/SECURITY.md` / `.ja.md`
- `docs/QUALITY.md` / `.ja.md`
- `docs/roadmap.md` / `.ja.md`
- Android Emulator design and completed ExecPlans;
- Flutter Android docs;
- Android UI observer docs;
- standalone distribution docs;
- `internal/execx/detached*.go`;
- Android runtime adapter;
- `internal/app`, `internal/domain`, `internal/config`, `internal/stack`;
- `internal/paths`, `internal/store/sqlite`;
- readiness/probe, endpoint and logs/evidence implementations.

Current `execx.NativeDetached` already rejects request-owned writers, uses private
file output, has no persistent-command timeout, survives caller CLI exit, records
PID plus native StartID, observes process-tree identity rather than PID alone, and
rejects Windows `.bat`/`.cmd` wrappers. Preserve these constraints.

## Plan of Work

### Milestone 1 — Product contract and architecture

Create bilingual product/design docs and finalize `type: process`, working dir,
command, env, ports, endpoint reference and runtime-dir interpolation. Preserve
Compose/Android/Flutter compatibility and update architecture checks as needed.

### Milestone 2 — Native termination primitive

Add the minimum native identity-gated stop API. Test exact live identity,
already-gone process, mismatched/reused identity, root exit, surviving descendants,
cancellation, repeated termination and completion proof on native OSes. Keep all
Android detached-process tests green.

### Milestone 3 — Manifest, paths and executable resolution

Strictly validate process-only fields, argv, NUL, source-confined cwd and native
Windows executable restrictions. Define PATH/source-relative resolution and
persist redacted launch evidence.

### Milestone 4 — Runtime directory and ports

Create private runtime state after lease/source identity exists. Add
`${runtime_dir}` and named loopback TCP reservation/interpolation. Persist port
identity before launch and fail conservatively on external occupancy. Extend the
common endpoint model with an explicit process runtime-port variant.

### Milestone 5 — Launch and evidence

Creation ordering:

```text
source -> runtime dir -> ports -> launch intent -> logs -> Start
       -> identity persistence -> readiness -> READY
```

Retain nonzero identity returned with an error. Compensate persistence failures
only while identity is proven.

### Milestone 6 — Readiness, endpoints and logs

Use a native HTTP helper bound to `${port:http}`. READY only after normal app
readiness succeeds. Show exposes state/endpoints; logs expose file-backed output.

### Milestone 7 — Inspect, reconcile and degradation

Observe exact native identity. Gone process => DEGRADED. Mismatch/uncertain tree =>
conservative state/quarantine. No auto restart. Manual death must be visible in a
later independent CLI invocation and PID reuse must not restore READY.

### Milestone 8 — Destroy and recovery

Destroy acquires the fence, proves ownership, requests termination, waits,
escalates only while proof remains valid, confirms whole-tree absence, then
releases ports/runtime state and continues source cleanup. Uncertain tree absence
retains dependent resources and quarantines. Repeated destroy and GC use the same
proof.

### Milestone 9 — Cross-process/concurrency integration

Use a real native helper that listens, logs, optionally spawns children, terminates
cleanly and can simulate root-exit/child-survival. Prove create CLI exit survival,
later CLI observation, two-lease isolation, sibling survival, unrelated-process
safety, manual-death degradation and conservative child uncertainty.

### Milestone 10 — Browser-ready fixture

Without implementing browser semantics, add a browser-shaped helper using
`${runtime_dir}/profile` and `${port:cdp}` with a minimal CDP-like HTTP endpoint.
Prove private state, persistent ownership, endpoint, readiness, logs, later CLI
observation and safe destroy.

### Milestone 11 — Native CI and documentation

Require actual persistent-process execution on Windows, macOS and Linux. This
feature has no hardware acceleration prerequisite. Cross-build remains additional
evidence only. Update bilingual Architecture/Portability/Reliability/Security/
Quality/Roadmap and remove generic persistent host processes from the unimplemented
roadmap after completion.

## Concrete Steps

1. Record base and create branch.
2. Add bilingual active plans.
3. Baseline harness/race.
4. Inspect native detached/Android regression surface.
5. Write bilingual product/design docs.
6. Finalize process/endpoint/interpolation syntax.
7. Add identity-gated termination.
8. Re-run Android detached tests.
9. Add config/domain/process runtime.
10. Add runtime dir/interpolation.
11. Add dynamic TCP ports and endpoint mapping.
12. Implement launch/evidence/readiness/logs/show.
13. Implement inspect/reconcile.
14. Implement destroy/quarantine/GC.
15. Add native HTTP helper and two-lease tests.
16. Add manual death/PID reuse/descendant uncertainty tests.
17. Add browser-shaped fixture.
18. Run Windows/macOS/Linux native integration.
19. Update bilingual durable docs.
20. Run final harness/race/native/cross-build.
21. Record direct acceptance evidence.
22. Complete bilingual retrospective.
23. Move plans to completed and update links/hashes.

## Validation and Acceptance

| ID | Required behavior | Evidence |
| --- | --- | --- |
| H1 | Existing Compose/Android/Flutter/UI-observer behavior remains valid. | Local check/race/Docker integration and Podman+Docker PASS; all six native regression jobs PASS in Verify 34226859965. |
| H2 | `type: process` is strictly validated and rejects incompatible runtime fields. | TestProcessManifestContract; TestProcessManifestNegativeFixtures; TestProcessPresenceRejectsYAMLMergeAndAliases; TestProcessFieldsDoNotChangeLegacyCanonicalShape — PASS, Linux integrated working tree 2026-09-08. |
| H3 | Command is direct argv with confined cwd and no implicit shell. | TestStartInterpolatesWithoutSnapshotSecrets; TestPrepareRejectsUnownedRootAndSourceEscape; TestPrepareRejectsSymlinkCWDAndRuntimeRoot — PASS, Linux integrated working tree 2026-09-08. |
| H4 | Process remains alive after launching create CLI exits. | Linux TestPersistentProcessNativeCLI PASS: create exits before independent later CLI observations. |
| H5 | A later independent CLI inspects the exact process using durable native identity. | Linux native CLI PASS: persisted identity survives independent show and creator interruption. |
| H6 | PID reuse/identity mismatch never makes an unrelated process owned. | TestManagedTermination; TestRecoveryUsesReceiptAndChecksMismatch; Windows TestManagedWindowsCompletedJobIgnoresReusedHistoricalPID PASS in final native CI. Identity reuse is simulated, not forced kernel PID reuse. |
| H7 | stdout/stderr are file-backed and available through runtime logs. | TestProcessComponentLogsRouteAndRetainIsolatedArtifacts; TestLogsRequireDurableRedactionVersion; TestBoundedLogsDoNotExposeSecretAcrossBoundary — PASS, Linux integrated working tree 2026-09-08. |
| H8 | Each runtime has a private state directory under agent-env state root. | TestDestroyPreservesLogsAndReceipt; TestPersistentProcessNativeCLI — PASS, Linux integrated working tree 2026-09-08. |
| H9 | Runtime-dir interpolation works with spaces/non-ASCII native paths. | TestPersistentProcessNativeCLI PASS on all three native OSes in Verify 34226859965: spaced/Unicode paths; Unix symlink home. |
| H10 | Named loopback TCP ports are dynamically allocated and persisted before launch. | TestProcessConcurrentReservationPortsAreDisjoint; TestProcessLifecyclePersistedIntentMixedRoutingAndIsolation — PASS, Linux integrated working tree 2026-09-08. |
| H11 | Concurrent process leases never share agent-env port reservation or runtime directory. | TestProcessConcurrentReservationPortsAreDisjoint; TestPersistentProcessNativeCLI — PASS, Linux integrated working tree 2026-09-08. |
| H12 | External port occupancy fails safely without silently changing persisted runtime identity. | TestProcessDynamicReservationAvoidsExternallyBoundPort; TestReservedPortOccupationPreventsLaunch — PASS, Linux integrated working tree 2026-09-08. |
| H13 | Process endpoint integrates with common component endpoint model without breaking Compose consumers. | TestProcessReadinessUsesRecordedNumericEndpoint; TestProcessEndpointAliasWinsOverRawRuntimePort; TestIntegrationPersistentProcessComposeCoexistence — PASS, Linux integrated working tree 2026-09-08. |
| H14 | Real HTTP helper becomes READY through normal app readiness. | Linux native CLI real /json/version HTTP readiness PASS. |
| H15 | Process exit before readiness fails create and compensates conservatively. | TestProcessExitDuringSuccessfulProbeCannotBecomeReady; TestProcessIdentitySaveFailureAndPartialStartCompensate — PASS, Linux integrated working tree 2026-09-08. |
| H16 | Manual death is DEGRADED and no automatic restart occurs. | TestProcessCrashDegradesWithoutRestartAndReleasedEffectsQuarantine; TestPersistentProcessNativeCLI — PASS, Linux integrated working tree 2026-09-08. |
| H17 | Root exit with descendants is never false-clean without native proof. | TestManagedRootGoneDescendant and TestExitedRootWithDescendantsIsNotReady PASS in all six native jobs, Verify 34226859965. |
| H18 | Destroy revalidates native identity immediately before termination. | Linux TestManagedTermination mismatch/cancellation/live-root protection PASS. Native Unix observation-to-signal gap remains documented. |
| H19 | Whole owned tree absence is confirmed before releasing ports/runtime state. | TestProcessUnknownOwnershipQuarantinesAndRecovers; TestProcessSaveRejectsImmutableSnapshotChanges; TestPersistentProcessNativeCLI — PASS, Linux integrated working tree 2026-09-08. |
| H20 | Ambiguous termination quarantines and retains recovery evidence/resources. | TestProcessUnknownOwnershipQuarantinesAndRecovers; TestProcessFenceLossRetainsLaunchForLaterRecovery; TestMissingLaunchingReceiptIsUncertain — PASS, Linux integrated working tree 2026-09-08. |
| H21 | Repeated destroy cannot kill a later PID/process-tree user. | TestManagedTermination; TestProcessReleasedReservationsCannotResurrect; TestPersistentProcessNativeCLI — PASS, Linux integrated working tree 2026-09-08. |
| H22 | Destroying A preserves sibling B and unrelated host processes. | TestPersistentProcessNativeCLI (5.422s): sibling and separate direct host-helper PID/profile survive cleanup/refusal — PASS, Linux integrated working tree 2026-09-08. |
| H23 | Process and Compose runtimes can coexist in a declared stack. | TestIntegrationPersistentProcessComposeCoexistence (55.965s) — PASS, Linux integrated working tree 2026-09-08. |
| H24 | Source tracked-change protections remain intact while a process may reference its worktree. | TestPersistentProcessNativeCLI (5.422s): live tracked README refusal preserves bytes/process; force captures tracked-diff before release — PASS, Linux integrated working tree 2026-09-08. |
| H25 | No shell/Python/Node/systemd/launchd/Windows Service becomes a core requirement. | Integrated arch-check/build/tests and Go-built CLI/helper PASS; no new core runtime/daemon. |
| H26 | Real persistent-process integration passes natively on Windows. | Windows Go 1.26/1.27 native repoctl check, including TestPersistentProcessNativeCLI and completed-Job regression, PASS in Verify 34226859965 at f588960. |
| H27 | Real persistent-process integration passes natively on macOS. | macOS Go 1.26/1.27 native repoctl check, including TestPersistentProcessNativeCLI, PASS in Verify 34226859965 at f588960. |
| H28 | Real persistent-process integration passes natively on Linux. | TestPersistentProcessNativeCLI (5.422s) — PASS, Linux integrated working tree 2026-09-08. |
| H29 | Browser-shaped fixture proves state dir + CDP-like port + readiness + later observation + cleanup. | TestPersistentProcessNativeCLI: private profile, /json/version, readiness, independent show, cleanup — PASS, Linux integrated working tree 2026-09-08. |
| H30 | Existing Android detached/guardian acceptance still passes after execx changes. | Android detached/guardian regressions PASS in all six native repoctl check jobs at f588960, Verify 34226859965. |
| H31 | Bilingual durable docs describe the delivered contract. | Bilingual durable contracts, completion links and native evidence updated together; docs-check PASS. |
| H32 | Final harness/translation/race/native CI passes. | Local harness/race/integration PASS; Verify 34226859965 at f588960 PASS, all 12 jobs including six native jobs, race/integration and five cross-build jobs. |
| H33 | Both ExecPlans contain direct evidence and retrospective before archival. | Both plans contain the final checkpoint, H1–H33 evidence and completed retrospective, and are archived together with updated links. |

Code existence alone is not acceptance. Record successful commands, native jobs,
observed process identities and port assignments where appropriate.

## Idempotence and Recovery

Plan/validation are read-only. Create is a saga and persists intent before effects.
If launch may have occurred, retain native identity/evidence and inspect before
cleanup. Never infer absence from PID lookup alone and never automatically retry
an uncertain launch.

Destroy is observe -> prove ownership -> terminate -> confirm absence -> release.
Do not release worktree/runtime directory/ports while a process tree may still use
them. GC uses the same proof and remains dry-run by default. Manual process death
is observed as degradation, not auto-repaired.

## Artifacts and Notes

Windows correction checkpoint (2026-09-08): the exact same-session Job is now
queried before historical PID validation. Empty/missing Jobs require matching
synced guardian completion evidence; active Jobs retain birth and membership
checks. A second census handles completion during PID observation. No historical
PID is signaled after proven completion. The new native regression
`TestManagedWindowsCompletedJobIgnoresReusedHistoricalPID` uses two real processes
and retains the old Job handle to test both empty and missing Job cases, plus
missing/wrong proof and active identity mismatch rejection. Linux harness and
Windows amd64/arm64 test cross-compiles PASS; native CI is pending. Independent
review found no weakening of active ownership or Android guardian conditions.


Native primitive evidence (2026-09-08, working tree based on `01e3581`):

- `go test ./internal/execx`: PASS.
- `go test -race ./internal/execx`: PASS (8.027s).
- `go test ./internal/execx -run TestManaged -count=3`: PASS (3.583s).
- `go test ./internal/runtime/android`: PASS (0.552s).
- `GOOS=windows GOARCH=amd64 go test -c ./internal/execx` and
  `GOOS=darwin GOARCH=arm64 go test -c ./internal/execx`: PASS, with outputs in
  temporary locations. These prove compilation only; native execution is pending.
- Linux process census can change during sampling. Bounded retries cover only the
  transient census sentinel, never ownership errors.

Contract milestone evidence (2026-09-08, working tree based on `01e3581`):

- `go test ./internal/config ./internal/domain`: PASS, including invalid runtime
  fields, local readiness endpoint references, Windows directory names, malformed
  references, YAML merge/alias presence, and unchanged legacy canonical shape.
- First `go run ./tools/repoctl docs-check`: FAIL because the supplied active plan
  lacked a Japanese sibling; translated plan added, follow-up result recorded at
  the next checkpoint. `go run ./tools/repoctl docs-check` then PASS after full
  translation/checkpoint update. Baseline harness/race later passed on the isolated reconstructed base; see the
  integration checkpoint.


Retain enough evidence to reconstruct launch and termination:

- lease/runtime/source commit;
- redacted argv/env;
- cwd and resolved executable;
- executable digest where recorded;
- runtime directory;
- named ports;
- stdout/stderr paths;
- PID/start/tree identity;
- launch/readiness timestamps;
- termination result;
- degradation/uncertainty observations.

Runtime mutable state is not automatically promoted to evidence. Browser profiles,
caches and local databases may be sensitive and should normally be removed after
confirmed cleanup.

## Interfaces and Dependencies

Likely package:

    internal/runtime/process/

Potential config concepts:

```go
type ProcessRuntime struct {
    Source           string
    WorkingDirectory string
    Command          []string
    Env              map[string]string
    Ports            map[string]ProcessPort
}
```

Potential native observation/termination types may include `Alive`, `RootAlive`,
certainty and a bounded grace duration. Exact types follow architecture review.

OS-specific PID/Job/process-group operations must not leak into app policy.

The only runtime external prerequisite is the executable declared by the target
repository/host. No new language runtime or daemon becomes a core agent-env
requirement.

## Resolved Milestone 1 Questions

1. Final `working_directory` field name.
2. Process port declaration syntax.
3. Component endpoint field for a process runtime port.
4. Interpolation namespace and whether source-path interpolation is needed.
5. PATH executable policy versus source-relative preference.
6. Executable digest policy for host binaries/scripts.
7. Windows versus Unix graceful termination semantics.
8. Extend current DetachedProcess versus introduce a clearer managed-process interface.
9. Process detail exposed by show/list.
10. Dynamic-port allocation strategy and external-race statement.
11. Need for a TCP readiness probe beyond existing HTTP/command probes.
12. Mutable-state retention after failed/quarantined cleanup.
13. Unix root-exit-with-descendants quarantine policy.
14. Whether to reserve a future restart contract or omit it entirely.
15. Whether Browser/CDP later layers an observer on `type: process` or composes a specialized runtime internally.

All questions above are resolved by the Decision Log, delivered product/design
contract and H1–H33 evidence. Native termination and show/list were verified by
the final three-OS CLI and primitive tests.
