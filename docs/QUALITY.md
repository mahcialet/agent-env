---
status: active
owner: maintainers
last_verified: 2026-09-09
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

The tag workflow must gate publication on repository checks, static artifact
validation, repeat-build comparison and native smoke jobs. The publishing job
uploads the already checked candidate bytes without rebuilding. Workflow presence
alone is not evidence that a release or native test has succeeded.

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

Real headless Chrome for Testing 152.0.7977.82 / CDP 1.3 passed on all three
OSes with Go 1.27 at `391288c` (Browser native 34247636411). The fixture exercised
AX/DOM, screenshots, Unicode input and clearing, stale rejection, iframe/shadow
observation, bounded diagnostics, a lease-hosted backend, durable input redaction
and safe profile cleanup. The plan records native evidence and CI repair history
separately from mock tests and cross-builds.

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
process identity, and independent cleanup. Unconfirmed cleanup retains fixture
state for investigation.

Linux/amd64 passed in 21.406s. The initial run exposed a real plan-digest mismatch
when transport canonicalized JSON object order; semantic manifest canonicalization
and a permanent source roundtrip regression fixed it. The
[native workflow](../.github/workflows/multi-host.yml) defines Windows/macOS/Linux
jobs; Windows/macOS pass evidence is still pending. Two workers on one physical
host do not prove physical-machine/VM multi-host behavior, and cross-builds do not
prove native role execution. Exact evolving results belong to the
[ExecPlan](exec-plans/active/multi-host-control-plane.md).
