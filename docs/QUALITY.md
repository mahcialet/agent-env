---
status: active
owner: maintainers
last_verified: 2026-09-10
---

# Quality and verification

[日本語](QUALITY.ja.md)

Use the repository harness for routine checks, then select the real-runtime
fixture for the behavior being changed. Commands below define how to verify;
recorded runs establish evidence only for their tested revision and environment.
Missing prerequisites and cross-build success do not count as native acceptance.

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

The schema document's source of truth is the numbered SQL set under [migrations](../migrations/001_initial.sql), not manually edited prose. Stable `AGENTENV-*` diagnostics identify the violated invariant and repair direction. Negative fixtures deliberately break links, indexes, metadata, plan sections, source formatting, schema generation, import boundaries, missing/orphan translations, incorrect source metadata, stale hashes, and language-specific links/indexes.

## Product tests

Unit and adapter tests cover strict manifest decoding, deterministic component/source identity, state transitions, TTL and GC eligibility, normalized Compose policy, path containment, command argv, streamed redaction, evidence failure paths, and lifecycle compensation. Fake runners assert executable, argv, working directory, environment controls, cancellation, output handling, and error conversion without constructing shell strings.

SQLite tests use real temporary databases, including Unicode paths, reopened/multiple connections, migration idempotency, foreign keys, atomic capacity/project reservation, normalized-row rollback, and durable lock behavior. Process tests exercise native argv and child-process cancellation; native platform execution is necessary to validate OS-specific implementations.

## Real Docker fixtures

The [integration suite](../internal/cli/integration_test.go) creates isolated temporary Git repositories and state homes and uses [the small Compose fixture](../testdata/compose/compose.yaml). Tests require both the `integration` build tag and explicit opt-in; the harness sets these automatically. Ordinary unit tests never start Docker containers.

The suite verifies these independent behaviors:

- Concurrent API/Dashboard leases use the selected closure, distinct projects and
  worktrees. Destroy preserves a sibling's container/network/volume IDs and an
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
selected service closure and reachable HTTP endpoints. Component logs, named
pass/fail tests, redaction and retained artifacts passed. Image-declared volumes
were natively anonymous and unlabelled, then absent after destroy. Destroy
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
English and Japanese reader reviews, followed by semantic parity review. Record
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

The tag workflow must gate publication on repository checks, static artifact
validation, repeat-build comparison and native smoke jobs. The publishing job
uploads the already checked candidate bytes without rebuilding. Workflow presence
alone is not evidence that a release or native test has succeeded.

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

The [process ExecPlan](exec-plans/completed/persistent-process-runtime.md) tracks direct
acceptance. Config tests cover strict process/Compose/Android field variants,
null/empty/merged YAML fields, named TCP ports, local readiness endpoint references,
portable directory names and unchanged legacy canonical snapshots. Adapter tests
cover confined executable/cwd resolution, interpolation, private state and launch
identity/prelaunch-redaction evidence, native identity recovery, bounded logs and
redaction after host secrets
change. Native primitive tests cover mismatched identity, termination retry and
root exit with live descendants; existing Android detached tests remain regressions.

Lifecycle acceptance additionally requires independent CLI survival/observation,
two concurrent leases, sibling/unrelated-process survival, persistence failure,
quarantine, TCP occupancy, source mutation, real HTTP readiness, process/Compose
coexistence and a browser-shaped state/CDP-like fixture. No browser semantics are
implemented by that fixture. Native Windows/macOS/Linux execution is required;
cross-compilation alone cannot complete this acceptance. Local adapter, race, integration and cross-build evidence, plus successful native
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
This regression evidence is separate from remote Browser operation acceptance.

## Multi-host native verification

```text
go test -tags=multihostintegration ./internal/cli -run '^TestMultiHostNativeCLI$' -count=1 -v -timeout=12m
```

This opt-in test needs Go and Git. It builds native agent-env and a committed process
fixture, generates short-lived test certificates with Go, and launches real
controller/client/two-worker processes over TLS with separate state roots. It uses
no shell script, Docker, SDK or browser. It verifies role/enrollment rejection,
whole-lease placement, drain, two simultaneous leases, distinct ports/worktrees,
local force refusal, controller outage/restart, worker restart retaining native
process identity, named tests invoked by relative executable path, retained logs,
registered artifact downloads with digest verification, renewal, client/worker
environment isolation, and independent cleanup. Unconfirmed cleanup retains fixture
state for investigation.

### Recorded role evidence and limits

Linux/amd64 passed in 21.406s. The initial run exposed a real plan-digest mismatch
when transport canonicalized JSON object order; semantic manifest canonicalization
and a permanent source roundtrip regression fixed it. The expanded fixture passed
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
| Forced invariant | Explicit barriers/state transitions reach the violating schedule or boundary and a discriminating oracle checks the invariant | Every scheduler interleaving or real OS behavior |
| Direct native/integration | Actual process, protocol or engine behavior on the named host with prerequisites satisfied | Other operating systems or absent infrastructure |
| Race/static/tooling | No reported issue in the exercised paths or the checked structural rule | All schedules, correct oracles or semantic translation parity |
| Stability/repetition | Observed reliability over the recorded repetitions and CPU settings | Deterministic reproduction or stronger proof through test count |
| Compilation/structural | Types, imports, generated artifacts or target compilation satisfy checks | Native execution or completion of runtime acceptance |

These classes describe different claims, not a ranking that permits substitution.
A test can supply more than one class, but its components must be identified.

### Fixture ownership and schedules

Inventory each helper's owner, resources, goroutines/callbacks, shared state,
completion signals, cleanup order, failure exits, consumers and assumptions.
Cleanup must stop admission, request cancellation or close transports as needed,
and join owned work before closing databases or removing temporary state. Closing
a listener does not close accepted connections; closing an HTTP server does not
join hijacked WebSocket callbacks. A cancellation signal is not a join.

Register failure-path cleanup before assertions. Do not discard cleanup failures
that mean native resources may remain. WaitGroup admission must be ordered before
Wait; a counter protected by an atomic still needs completion ordering. Avoid
joining from the worker being joined. Test timeouts bound a failed check; they do
not establish callback entry, completed cleanup, or a negative side effect.
Use channels/barriers or `testing/synctest` for schedules where appropriate; real
sockets/native processes need separately identified composition evidence.

### Oracles, findings and preventive controls

For every negative fixture, identify the earliest refusal that could accidentally
satisfy the assertion. Require the intended mutation/boundary to be reached, then
assert the specific error/state and include a valid neighboring control where
useful. An absent delayed file is not independent process-completion evidence.
Compare persisted state/effects independently rather than deriving the expected
answer through the same logic as the implementation under test.

Before repairs, record each finding's ID, violated invariant, earliest feasible
detection point, escape category, test versus production scope, sibling exposure,
ACCEPT/REJECT/DEFER disposition and rationale. Accepted repairs should demonstrate
fail-before/pass-after using the same oracle where practical; compilation failure
from a new test API does not count. Record unavailable native controls explicitly.

Prefer targeted regressions for stable lifecycle and boundary invariants. Existing
`check` and native/race CI execute these controls without adding a second policy
runner. Do not mechanically infer evidence quality from test names, prose, sleep
usage or pass counts: these need contextual review. Independent review examines
the diff, test reachability, finding dispositions and actual evidence; bilingual
reader/parity review remains separate from source-hash checks.

Run suites sharing fixed native resource pools serially unless isolation has been
verified. An isolated rerun after contention is diagnostic evidence and does not
erase the original failure.
