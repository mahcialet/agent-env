---
status: active
owner: maintainers
last_verified: 2026-09-10
---

# Quality and verification

[日本語](QUALITY.ja.md)

To verify a change, start with the shared check commands, then select tests that
use the actual runtime for the behavior being changed. This document explains
how to run checks, what they examine, and what conclusions their results support.
Recorded runs establish evidence only for their tested revision and environment.
Missing prerequisites and cross-build success do not count as native acceptance.


These terms distinguish the test setup from the strength of its evidence.

| Term | Meaning in this document |
| --- | --- |
| Harness | The shared mechanism for running checks; this repository uses repoctl. |
| Fixture | Inputs, configuration, data, helper programs or resources prepared for a test. A fixture may use real resources or simulate their behavior. |
| Unit and adapter tests | Tests of individual operations and the code connecting to external tools. Fakes and mocks simulate external behavior, so their success is distinct from success in the real environment. |
| Native execution | Running the program on the operating system being verified. |
| Cross-build | Building an executable for another operating system or CPU; this does not establish that it ran there. |
| Smoke test | Starting a distribution candidate and checking basic behavior. |
| Lease, runtime and component | A lease is a managed environment; a runtime is an execution resource such as a process or Compose; a component is a functional unit declared in the manifest. |
| Manifest and worktree | A manifest declares environment configuration. A worktree is a source working directory managed by Git. |
| Digest and checksum | Values calculated from file contents to detect changes or mismatches. |
| Race test | A test that checks concurrent access to shared data on the paths actually executed. |
| Invariant | A condition that must hold as inputs or execution order change; an individual test does not necessarily prove every order. |

## Canonical checks

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go run ./tools/repoctl test-integration
```

`doctor` locates Go, gofmt and Git. Docker is required only for the explicit
integration command. Underlying commands remain directly runnable; the harness
requires no Bash, Make or PowerShell.

`check` visibly runs formatting verification, `go test ./...`, `go vet ./...`,
documentation validation, generated-file drift detection and architecture checks.

| Harness command | Scope |
| --- | --- |
| `test-unit` | Ordinary Go tests; no Docker daemon required |
| `test-integration` | Checks Docker daemon/Compose and runs uncached tagged tests with explicit integration opt-in |
| `docs-check` | Metadata, links/headings, local indexes, AGENTS length/paths, mandatory active-plan sections, and missing/stale English/Japanese pairs |
| `generate` | Regenerates the database document from embedded SQL migration sources |
| `generated-check` | Fails when the generated schema differs from migrations |
| `arch-check` | Enforces documented import boundaries with actionable diagnostics |

The database schema document is generated from the numbered SQL files under
[migrations](../migrations/001_initial.sql), not from manually edited prose.
Stable `AGENTENV-*` diagnostic codes identify the violated condition and the
repair direction. Tests deliberately create the following invalid states to
check that the validators detect them:

- Broken links, indexes, metadata and required plan sections.
- Violations of source formatting, schema generation and allowed import boundaries.
- Missing or orphaned translations, incorrect source metadata, stale hashes and
  broken language-specific links or indexes.

## Product tests

Unit and adapter tests cover settings, lifecycle operations and compensation after failures:

- Strict manifest decoding, repeatable component/source identity and state transitions.
- Time to live (TTL), eligibility for garbage collection (GC), and policy applied
  to normalized Compose settings.
- Paths staying within their permitted scope, command argument arrays (argv),
  and masking secrets as output is streamed.
- Failure to save evidence and compensation after a lifecycle operation fails.
  Compensation cleans up effects already performed, such as creation.

Fake runners simulate external command execution. They check the executable,
argv, working directory, environment controls, cancellation, output handling and
error conversion without constructing shell command strings.

SQLite tests use real temporary databases to check:

- Unicode paths, reopening databases and multiple connections.
- Repeating migrations without changing the result, and foreign-key constraints.
- Capacity/project reservations completing atomically, and rollback of normalized rows.
- Durable locking behavior.

Process tests exercise OS argument passing and child-process cancellation.
OS-specific implementations need execution on that operating system.

## Real Docker fixtures

The [integration suite](../internal/cli/integration_test.go) creates isolated temporary Git repositories and state homes and uses [the small Compose fixture](../testdata/compose/compose.yaml). Tests require both the `integration` build tag and explicit opt-in; the harness sets these automatically. Ordinary unit tests never start Docker containers.

The suite verifies these independent behaviors:

- Concurrent API/Dashboard leases include the selected services and all their direct and indirect dependencies,
  with distinct projects and worktrees. Destroy preserves a sibling's container/network/volume IDs and an
  unselected foreign volume.
- Manifests generate dynamic loopback HTTP without source port declarations.
  Versioned environment descriptors and component-scoped live/retained logs
  remain available.
- Manually removed projects become degraded; multiple repositories respect
  source ref overrides.
- Named tests retain redacted stdout/stderr/artifacts and nonzero exit status.
  Readiness failure rolls back real resources.
- Dirty tracked worktrees survive GC until explicit force with diff evidence.

### Recorded Docker evidence

The final local Linux CLI integration run passed in 109.95 seconds, including generated endpoints, diagnostic descriptors, and component-scoped live and archived logs. This establishes real Docker behavior for that tested revision, not completion of later edits or every native platform. All fixture resources have unique tracked identities and lease-specific cleanup; the suite never runs a general Docker prune.

## Real Podman fixtures

Set `AGENT_ENV_PODMAN_INTEGRATION=1` using your platform’s native environment
configuration, then run:

```text
go test -tags=integration ./internal/cli -run TestPodmanIntegration -count=1 -v
```

Also set `AGENT_ENV_PODMAN_DOCKER_COEXISTENCE=1` to require Docker coexistence
and run the same named test fixture against both engines. The suite builds and
executes the actual CLI to exercise its native Podman child bridge. It requires
Linux rootless Podman 5.x and podman-compose >=1.6.0,<2.0.0; opting in with missing
or unsupported prerequisites fails rather than skips. Ordinary tests start neither
engine. The fixture checks concurrent leases, HTTP endpoints, redacted named-test
evidence, logs, sibling/foreign-resource survival and residual cleanup. It makes
lease-scoped engine changes and uses owned cleanup, never global prune.

### Recorded Podman evidence and limits

On 2026-09-08, `TestPodmanIntegrationConcurrentLeasesAndEvidence` passed in
110.13 seconds with both opt-ins enabled, using Linux rootless Podman 5.4.2 and
podman-compose 1.6.0 alongside Docker. Both Podman leases reached ready with the
selected services and all their direct and indirect service dependencies, with reachable HTTP endpoints. Component logs, named
pass/fail tests, redaction and retained artifacts passed. In this test, Podman created anonymous volumes from the image’s Dockerfile
`VOLUME` declaration. Volume inspection confirmed `Anonymous: true` and empty
`Labels`; the volumes were absent after destroy. Destroy
preserved the sibling lease, foreign volume and live Docker lease; the same named
tests also passed/failed as expected on Docker. Native Windows/macOS/Linux provider CI passed on 4a5de3d
(run 34216579481), and real Machine infrastructure is unavailable. Exact evidence remains
in the
[provider plan](exec-plans/completed/compose-provider-podman.md).

Run [34216579481](https://github.com/mahcialet/agent-env/actions/runs/34216579481)
on 4a5de3d passed all six native Windows/macOS/Linux jobs across Go 1.26/1.27,
all five CGO-disabled cross-builds, and the Linux race/Docker integration job.
All 12 jobs succeeded. The local Docker and Podman/Docker coexistence runs above
also passed.

## CI and completion evidence

CI runs the harness and CLI build natively on Windows, macOS, and Linux for Go 1.26.x and 1.27.x. Linux runs `go test -race ./...` and explicit Docker integration. Separate cross-build jobs cover five targets with `CGO_ENABLED=0`.

### Historical MVP evidence

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

For substantial restructuring, the language policy also requires independent
English and Japanese reader reviews, followed by a comparison to check that both
languages preserve the same meaning. Record
the findings in the active ExecPlan; passing mechanical checks alone does not
establish readability or preserve meaning.

## Release verification

Run release construction from the exact clean tagged source using an output
path that does not exist. Use separate output directories for repeat builds.

```text
go run ./tools/repoctl release-build --version X.Y.Z --out <new-directory>
go run ./tools/repoctl release-check --dir <directory> --version X.Y.Z
go run ./tools/repoctl release-repeat --dir <directory> --version X.Y.Z
go run ./tools/repoctl release-smoke --dir <directory> --version X.Y.Z
```

`release-check` statically inspects the six-archive set and Go executable build
information without executing foreign binaries. `release-smoke` executes only the
host tuple after extraction outside the checkout, with a restricted executable
search path and separate state home containing spaces/non-ASCII characters.
Compare executable and archive digests, checksums and manifest across repeated
same-source, same-toolchain builds; do not substitute successful compilation for
byte comparison. CI pins the release builder to Go 1.27.1. Native Windows/macOS/Linux
smoke results and arm64 coverage must be recorded explicitly in the
[release plan](exec-plans/completed/standalone-release-finalization.md).

### Publication gates

The tag workflow must satisfy all of these publication conditions.

| Item | Required check | Why it is required |
| --- | --- | --- |
| Repository checks | Pass the repository harness and race tests before publishing. | Packaging can succeed even when repository tests or invariants fail. |
| Static artifact validation | Validate archive contents, checksums and executable build identity with `release-check`. | Detect malformed or mismatched distribution files without executing foreign-target binaries. |
| Repeat-build comparison | Rebuild from the same commit and toolchain and compare the output bytes with `release-repeat`. | A successful build alone does not establish reproducibility. |
| Native smoke tests | Run the extracted candidate on Windows, macOS and Linux. | Detect startup failures or undeclared CLI dependencies that cross-compilation cannot reveal. |
| Publish the checked bytes | Upload the already validated candidate without rebuilding it in the publishing job. | Keep the distributed files identical to the files that passed validation. |

Workflow presence alone is not evidence that a release or native test has succeeded.

### Repeat builds and private previews

`release-repeat` validates the tag-specific candidate, rebuilds from the same
commit/toolchain and compares all eight output files byte for byte. For untagged
branch/PR validation, use a private preview:

```text
go run ./tools/repoctl release-verify --out <new-directory>
go run ./tools/repoctl release-preview-smoke --dir <directory>
```

`release-verify` requires clean source, creates `v0.1.0` only inside a private
clone, performs two builds, compares bytes and runs the local native smoke before
copying the candidate to a new output directory. `release-preview-smoke` verifies
that preview against the current commit in another private clone. Neither command
creates public refs or publishes. The preview workflow runs this comparison and
three native OS smoke jobs; the tag workflow independently requires repeat-build
and three native smoke gates before publishing. These are workflow requirements,
not a claim that any particular run has passed.

## Persistent process verification

The [process ExecPlan](exec-plans/completed/persistent-process-runtime.md) tracks direct acceptance evidence.

Configuration tests check the manifest before runtime operations.

| Item | What the tests check | Why it is checked |
| --- | --- | --- |
| Fields allowed for each runtime | Accept the fields defined for process, Compose or Android and reject fields belonging to a different runtime type. | Prevent irrelevant settings from being silently ignored. |
| YAML null, empty and merged values | Check field presence even when values are null or empty, or fields arrive through YAML merges. | Do not let alternate YAML forms bypass restrictions on allowed fields. |
| Named TCP ports and readiness references | When named ports are configured, require TCP; resolve readiness endpoint references only to endpoints declared on the same component. | Prevent readiness from probing an undefined port or another component’s endpoint. |
| Runtime names used as directory names | Reject process runtime names that are invalid directory names on supported operating systems, such as reserved names or trailing dots. | Keep a configuration that works on one OS from failing because of directory naming on another. |
| Existing canonical snapshots | Adding process fields must not change the normalized serialized form of existing runtime settings. | Preserve compatibility with previously stored configuration snapshots. |

Adapter tests check how validated settings become process operations and retained evidence.

| Item | What the tests check | Why it is checked |
| --- | --- | --- |
| Executable and working-directory resolution | Keep source-relative executable paths and working directories inside the source; resolve host executables selected through PATH to their concrete files. | Avoid unintentionally running a different file or using a directory outside the declared source. |
| Argument and environment expansion | Expand runtime-directory, lease, port and environment references for execution without replacing secret references in the saved snapshot with plaintext values. | Pass the intended values to the process without retaining plaintext secrets in configuration snapshots. |
| Private state and launch identity | Verify the runtime’s private state, ownership records and recorded identity of the launched process. | Keep state and process operations tied to the correct lease and runtime. |
| Redaction evidence before launch | Persist the information needed to mask secrets before starting the process. | Keep failure-recovery logs safe even if saving the later launch receipt fails. |
| Recovery of native identity | Recover process identity from the saved launch receipt and reject inconsistent identity evidence. | Let later operations identify the existing process rather than rely on a PID alone. |
| Bounded logs | Limit returned log data and mask secrets even where the output limit cuts across them. | Prevent excessive output and secret fragments leaking at the truncation boundary. |
| Secret changes on the host | Continue masking secrets used at launch after the corresponding host environment variables change or disappear. | Do not expose old log secrets merely because the current environment has changed. |

Native process tests check OS-specific identity and termination behavior.

| Item | What the tests check | Why it is checked |
| --- | --- | --- |
| Mismatched identity | Refuse termination with mismatched process identity and preserve the live process. | Prevent a reused or incorrect PID from authorizing termination of an unrelated process. |
| Repeated termination | A termination call can be made again after confirmed termination. | Allow cleanup retries without requiring the caller to know whether an earlier call already completed. |
| Root exit with live descendants | Do not report the whole process tree as absent when the root has exited but descendants remain; preserve OS-specific ownership checks. | A root PID disappearing must not hide surviving child processes or authorize unverified termination. |

Keep the existing Android detached-process tests, which check processes running
independently of their launcher, to detect changes that break existing behavior.

Lifecycle acceptance additionally requires the following checks.

| Item | Required check | Why it is required |
| --- | --- | --- |
| Survival across CLI invocations | After the creating CLI exits, the process remains alive and another CLI can observe its state and logs. | Verify that the persistent process does not depend on the creating CLI remaining alive. |
| Recovery after CLI interruption | Interrupt the CLI during readiness and use a later CLI to observe and clean up the same process. | Verify that CLI failure does not lose process ownership or recovery information. |
| Two live leases | Two leases coexist with distinct PIDs, ports and private state directories. | Detect unintended sharing of process identity, endpoints or mutable state. |
| Sibling and unrelated processes | After one lease is cleaned up, sibling and unrelated host processes still respond. | Verify that cleanup stays within the owned resource scope. |
| Persistence failure | Exercise compensation and identity/evidence retention when recording a started process fails; refuse termination if ownership evidence is uncertain. | Avoid losing a started process or stopping one without sufficient ownership evidence. |
| Quarantine | On an identity mismatch, refuse termination, quarantine the lease and retain its sources. | Keep uncertain ownership from authorizing destructive cleanup. |
| TCP occupancy | Refuse startup when another listener occupies the allocated port. | Detect port conflicts before starting a process against an unavailable endpoint. |
| Source edits | Normal cleanup refuses tracked edits before stopping the process; explicit force preserves the diff before cleanup. | Protect edits from silent deletion and retain evidence when force is requested. |
| Real HTTP readiness | Check an actual HTTP response at the allocated endpoint. | Do not equate a live process with an available service. |
| Process/Compose coexistence | Exercise both runtimes in a lease and preserve the sibling lease when cleaning up one lease. | Check that different runtime adapters do not interfere with each other’s lifecycle management. |
| Browser-shaped fixture | Run a process with private profile state and a CDP-like HTTP endpoint. This fixture implements no browser semantics. | Check the state, port and lifetime needs of a browser-shaped workload without treating this as browser feature validation. |
| Native Windows/macOS/Linux | Execute on all three operating systems; cross-compilation alone cannot complete acceptance. | Compilation cannot establish OS-specific process lifetime, termination or path behavior. |

Local adapter, race, integration and cross-build evidence, plus successful native
Windows/macOS/Linux CI at `f588960` (Verify 34226859965), are recorded in the completed plan.

## Browser/CDP verification

The [completed browser ExecPlan](exec-plans/completed/browser-cdp-automation.md) owns
acceptance evidence. `TestBrowserManifestContract`,
`TestBrowserManifestNegativeFixtures`, `TestBrowserRequiresProcessRuntime` and
`TestBrowserAbsentPreservesLegacyCanonicalShape` cover explicit bindings, exact
private-profile/debugging flags, YAML null/merge/alias variants and compatibility.
Config unit and race tests passed locally. Baseline `go test -race ./...` passed;
the initial harness reached docs-check after unit/vet success and failed because
the supplied Japanese plan lacked translation metadata. That failure is recorded
and corrected in the bilingual plan.

### Recorded browser evidence

Real headless Chrome for Testing 152.0.7977.82 / CDP 1.3 passed on all three
OSes with Go 1.27 at `391288c` (Browser native 34247636411). The fixture exercised
AX/DOM, screenshots, Unicode input and clearing, stale rejection, iframe/shadow
observation, bounded diagnostics, a lease-hosted backend, durable input redaction
and safe profile cleanup. The plan records native evidence and CI repair history
separately from mock tests and cross-builds.

The existing local Browser/CDP matrix passed again on Windows, macOS and Linux at
`440082b` ([run 34320519250](https://github.com/mahcialet/agent-env/actions/runs/34320519250)).
This evidence that existing behavior still works is separate from remote Browser
operation acceptance.

## Multi-host native verification

```text
go test -tags=multihostintegration ./internal/cli -run '^TestMultiHostNativeCLI$' -count=1 -v -timeout=12m
```

This opt-in test needs Go and Git. It builds native agent-env and a committed process
fixture, generates short-lived test certificates with Go, and launches real
controller/client/two-worker processes over TLS with separate state roots. It uses
no shell script, Docker, SDK or browser. The controller manages placement and
the workers execute operations.

The fixture checks the following behavior.

| Item | What the fixture checks | Why it is checked |
| --- | --- | --- |
| Roles and enrollment | Reject client operations with worker or unenrolled credentials and placement on unenrolled hosts. | A certificate alone must not grant another role’s authority or make a host eligible. |
| Whole-lease placement and drain | Place all runtimes of one lease on one worker and exclude draining workers from new placement. | Prevent resources in one lease from receiving conflicting assignments or ignoring drain. |
| Two simultaneous leases | Keep their worktrees and ports distinct and their endpoints reachable. | Detect workspace or endpoint collisions during concurrent use. |
| Local force refusal | Reject local forced destruction of a controller-managed lease. | Do not let local commands bypass controller ownership. |
| Controller outage and restart | Workloads remain available during the outage; restarting with the same state preserves controller/worker identity. | Do not treat a control-plane outage as workload termination or a new owner. |
| Worker restart | Keep the running process identity through worker restart and reconciliation. | Recover management of existing workloads without silently restarting them. |
| Named tests and environment isolation | Invoke committed tests through relative executable paths; resolve variables on the worker and do not forward client-only variables. | Keep execution tied to worker sources and configuration rather than the client’s environment. |
| Retained logs | Retrieve run output and the selected component’s runtime logs without mixing sibling output. | Keep remote diagnostic evidence accessible and attributable to the correct component. |
| Registered artifact download | Retrieve registered artifacts and verify their bytes and digests. | Check that transferred files match the registered evidence. |
| Renewal | Extend the lease expiry while retaining management and assignment identity. | Renew the existing lease without changing its managing controller or placement. |
| Independent cleanup | Clean up one lease without affecting its sibling; retain fixture state when cleanup cannot be confirmed. | Avoid collateral deletion and preserve investigation evidence after uncertain cleanup. |

### Recorded role evidence and limits

Linux/amd64 passed in 21.406s. The initial run exposed a real plan-digest mismatch
when transport canonicalized JSON object order; semantic manifest canonicalization
fixed it; a permanent test checks that meaning survives a source roundtrip
and detects recurrence. The expanded fixture passed
all Windows/macOS/Linux jobs at `440082b` in the
[native workflow run 34320519252](https://github.com/mahcialet/agent-env/actions/runs/34320519252).
Each runner used two worker roots on one host. This does not prove physical-machine/VM
multi-host behavior, and cross-builds do not prove native role execution.
Final acceptance is complete within the documented scope; exact results belong to the completed
[ExecPlan](exec-plans/completed/multi-host-control-plane.md).

### Remote runtime fixtures

Additional remote runtime fixtures use the same explicit build tag:

```text
go test -tags=multihostintegration ./internal/cli -run '^TestMultiHostRemoteBrowser$' -count=1 -v -timeout=12m
go test -tags=multihostintegration ./internal/cli -run '^TestMultiHostRemoteCompose$' -count=1 -v -timeout=12m
```

The Browser fixture requires a directly executable compatible `google-chrome` on
PATH and a usable browser sandbox. The Compose fixture requires both working Docker
Compose and Podman/podman-compose environments. Selecting either test with missing
prerequisites fails. They launch real controller/worker/runtime processes, retain
registered evidence, and perform lease-scoped cleanup; ordinary unit tests do not
start these external runtimes. These commands are not a claim of remote native
acceptance on every OS; consult the completed plan for the tested scope and results.

### Path and WSL verification limits

The Windows execution-path tests cover the 240 UTF-16 boundary, supplementary
characters, canonical paths, derived worktree/runtime directories and refusal
before reservation/materialization/output creation. Non-Windows deep source
lifecycle remains a success test. Direct-interop tests reject PE binaries through
absolute, relative, PATH and symlink lookup while preserving native .exe names.
WSL state tests inject kernel/filesystem/mount observations and check custom
mounts, aliases and missing future homes. These tests do not replace a real WSL2
mount/interop run; that environment has not been exercised.

## Evidence classes and test architecture

Evidence must be classified by what it proves. Quantity does not upgrade evidence
class. Repetition is not a substitute for deterministic reproduction. Record the
revision, command, environment, result and limits with each acceptance claim.

| Class | What it establishes | What it does not establish |
| --- | --- | --- |
| Forced invariant | Synchronization points or state transitions reproduce the problematic order or boundary; a decision rule (oracle) that distinguishes failure causes checks the condition | Every scheduler interleaving or real OS behavior |
| Direct native/integration | Actual process, protocol or engine behavior on the named host with prerequisites satisfied | Other operating systems or absent infrastructure |
| Race/static/tooling | No reported issue in the exercised paths or the checked structural rule | All schedules, correct oracles or semantic translation parity |
| Stability/repetition | Observed reliability over the recorded repetitions and CPU settings | Deterministic reproduction or stronger proof through test count |
| Compilation/structural | Types, imports, generated artifacts or target compilation satisfy checks | Native execution or completion of runtime acceptance |

These classes describe different claims, not a ranking that permits substitution.
A test can supply more than one class, but its components must be identified.

### Fixture ownership and schedules

For each test helper, record its owner, resources, concurrent work (goroutines or
callbacks), shared state, completion signals, cleanup order, failure exits,
callers and assumptions. Cleanup must stop admitting new work and request
cancellation or close connections as needed. Wait for the owned work to finish
before closing databases or removing temporary state. This wait for completion
is called a join.

A stop request or a closed connection alone does not establish completion:

- Closing a listener does not close connections it already accepted.
- Closing an HTTP server does not wait for WebSocket callbacks on connections
  taken out of HTTP management through hijacking.
- Cancellation requests termination; it is not a join that waits for termination.

Time-bounded prerequisites such as worker heartbeats, the periodic signals that
report liveness, must remain valid through
setup and the exercised operation. Successful registration is not continuing
availability; force expiry and refresh when validating such fixtures.

Register failure-path cleanup before checking test results with assertions.
Do not discard cleanup failures that could leave native resources behind.
Follow these rules when coordinating concurrent work and testing its order:

- Register work with a WaitGroup before calling Wait. Protecting a counter with
  atomic operations does not by itself establish completion order.
- Do not make a worker wait for its own completion.
- A timeout bounds the wait for a failed check. It does not establish callback
  entry, completed cleanup or the absence of a side effect.
- Acknowledging cancellation does not show that the cleanup goroutine has reached
  its wait for completion.
- In tests that use only channels to control execution order and check that cleanup
  has not completed, use `testing/synctest` to let work settle into a waiting state
  before making that assertion.

Use channels, synchronization points (barriers), or `testing/synctest` to control
execution order where appropriate. Identify evidence from actual combinations
of sockets or native processes separately from these controlled-order tests.

### Oracles, findings and preventive controls

An oracle is the rule used to decide whether a test result is correct. In tests
that check an expected rejection or error, identify the earliest refusal that
could satisfy the assertion before the intended operation is reached. Require
the intended mutation or boundary to be reached, then check the specific error
or state. Where useful, also try a nearby valid input and check that it is accepted.

Distinguish failure causes and completion evidence:

- Inspect the returned error itself; context state alone does not identify the
  operation’s failure cause.
- The absence of a file scheduled to appear later is not independent evidence
  that a process has finished.
- Compare persisted state and side effects independently, instead of calculating
  the expected answer with the same logic as the implementation being tested.

Before making a repair, record the finding’s ID, violated invariant, earliest
realistic detection point and category of reason it was missed. Also record
whether the problem is in tests or production code, exposure in similar locations,
and the ACCEPT/REJECT/DEFER decision with its rationale. Where practical, accepted
repairs should fail before the repair and pass afterward using the same decision
rule. Failure to compile because a new test API is absent from the old code is
not a reproduction of the defect. State explicitly when native before/after
controls could not be run.

Prefer targeted regression tests for stable lifecycle and boundary conditions.
A regression test checks that a change has not broken existing behavior; when
it covers a repaired defect, it also detects recurrence of that defect.
Run these through the existing `check` and native/race CI rather than adding a
second policy runner.

Do not judge evidence quality mechanically from test names, prose, sleep usage
or pass counts. Review those in context. Independent review checks the diff,
whether the test reaches its intended operation, finding decisions and actual
evidence. Separate English/Japanese reader reviews and a comparison of their
meaning remain distinct from source-hash checks.

Run suites sharing fixed sets of native resources, such as a port range, one at
a time until their isolation has been verified. An isolated rerun after contention is diagnostic evidence and does not
erase the original failure.
