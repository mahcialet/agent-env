---
status: completed
owner: maintainers
last_verified: 2026-09-08
---

# Materialize Flutter Android applications inside environment leases

This ExecPlan is a living document. Maintain it according to
`docs/PLANS.md`.

Expected branch: `feat/flutter-android-runtime`.

At implementation start, record the exact `master` revision below before making
changes. PR #2 (`Add isolated Android Emulator resources to leases`) is a hard
prerequisite and is already merged. Do not reimplement or weaken the Android
Emulator ownership, process, AVD, port, ADB-server, cleanup, or quarantine rules
delivered by that work.

Starting revision: `5e5c8b19b5880bff8e3dd97d0947ef47ac9ca066` (master and initial working HEAD).

## Purpose / Big Picture

After this work, `agent-env` can materialize a Flutter Android application from a
pinned Git source, install it into an already lease-owned Android Emulator,
connect it to selected Compose-backed API endpoints through lease-owned
`adb reverse` mappings, launch it, and record the exact build/install evidence.

A repository can describe stacks such as:

    api
    dashboard
    mobile
    full

where:

    api       = Web API only
    dashboard = Web API + Dashboard
    mobile    = Web API + Android Emulator + Flutter application
    full      = Web API + Dashboard + Android Emulator + Flutter application

The observable workflow is:

    agent-env plan . --stack mobile
    agent-env create . --stack mobile
    agent-env show <lease-id>
    agent-env test <lease-id> mobile-e2e
    agent-env doctor <lease-id>
    agent-env destroy <lease-id>

Two simultaneous `mobile` leases must use distinct source worktrees, Compose
projects, Android writable state, Emulator identities and host endpoint
allocations. Both applications may use the same device-side backend port because
each Emulator has an independent loopback namespace.

This plan does not add Android UI automation. UI hierarchy snapshots,
screenshots, taps, text input and exploratory agent interaction belong to the
separate `android-ui-observer` follow-up.

## Scope

In scope:

- Flutter SDK prerequisite discovery and diagnostics
- explicit Flutter Android application declarations in `.agent-env.yaml`
- build from the lease's pinned source worktree
- argv-only configurable build commands without shell interpretation
- APK path validation and SHA-256 evidence
- Android package/activity declaration and validation
- install into the lease-owned Emulator
- launch through the selected Android runtime
- `adb reverse` from device-local TCP ports to selected Compose component
  endpoints
- durable application/build/network identity in the lease registry
- `plan`, `create`, `show`, `list`, `doctor`, `reconcile`, `logs`, `test` and
  `destroy` behavior where relevant
- reverse-order compensation when build/install/network/launch fails
- named-test interpolation for the selected Android serial where required
- support for mixed Compose + Android + Flutter stacks
- concurrent lease isolation
- Windows, macOS and Linux path/argv/process behavior
- native fake-adapter coverage on all supported OSes
- real Flutter + Emulator integration evidence where suitable SDK and
  acceleration are available
- English and Japanese durable product/design documentation for this feature

Out of scope:

- UIAutomator or Accessibility hierarchy snapshots
- screenshots and screen recording
- exploratory tap/type/back/home commands
- logcat as a first-class observation stream
- iOS Simulator
- physical Android devices
- remote Android hosts
- automatic Android SDK, Flutter SDK or system-image installation
- accepting Android SDK licenses
- local OCI registries or APK artifact promotion/retention
- exact replay of historical APK artifacts after the lease is gone
- writable/fix worktrees
- malicious-code isolation
- generic application/plugin frameworks unrelated to this vertical slice

## Architectural intent

The existing `android-emulator` runtime remains a device/runtime resource.
Flutter must not be folded into `internal/runtime/android`, and the runtime type
must not be renamed to `flutter-android`.

The preferred model is a distinct application/workload declaration that binds:

- one pinned source;
- one existing `android-emulator` runtime;
- one Flutter project directory;
- one argv build command;
- one expected APK output;
- one Android package and launch activity;
- zero or more endpoint-to-device-port reverse bindings.

This plan may refine the exact manifest field names after inspecting current
config/domain/app interfaces, but it must preserve that ownership boundary.
If implementation requires changing this boundary, record the decision before
coding and promote it to an ADR.

An initial target manifest shape is:

```yaml
version: 1

sources:
  backend:
    repository: ../backend
    default_ref: HEAD

  mobile:
    repository: ../mobile
    default_ref: HEAD

runtimes:
  backend:
    type: compose
    source: backend
    project_directory: .
    files:
      - compose.yaml

  phone:
    type: android-emulator
    source: mobile
    avd: Pixel_API_35

applications:
  mobile-app:
    type: flutter-android
    source: mobile
    runtime: phone
    project_directory: .
    build:
      command:
        - flutter
        - build
        - apk
        - --debug
      artifact: build/app/outputs/flutter-apk/app-debug.apk
    package: com.example.app
    activity: .MainActivity
    reverse:
      - device_port: 8080
        endpoint: api.http

components:
  api:
    runtime: backend
    compose_services:
      - api
    endpoints:
      http:
        service: api
        target: 8080

  mobile:
    runtime: phone
    application: mobile-app
    depends_on:
      - api
    provides:
      - android-ui
      - mobile-app

stacks:
  api:
    roots:
      - api

  mobile:
    roots:
      - mobile

tests:
  mobile-e2e:
    stack: mobile
    source: mobile
    working_directory: .
    command:
      - flutter
      - test
      - integration_test
      - -d
      - ${android:phone:serial}
    timeout: 20m
    artifacts:
      - build/test-results
```

This example is a design target, not permission to bypass strict manifest
validation. Any final syntax must be documented and covered by tests that reject manifests violating its requirements.

## Progress

- [x] 2026-09-08: Confirmed master and working HEAD at
      `5e5c8b19b5880bff8e3dd97d0947ef47ac9ca066`; branch already
      `feat/flutter-android-runtime`, with only the supplied active plan untracked.
- [x] 2026-09-08: Inspected config/domain/app/runtime/store/CLI. Extend
      BuildPlan, source-materialization boundary, waitReady/probeReady, cleanup,
      Reconcile, named-test expansion, and concrete CLI wiring. SQLite already
      stores complete Lease JSON; additive application records need no SQL migration.
- [x] 2026-09-08: Write English and Japanese product contracts before public CLI behavior.
- [x] 2026-09-08: Write English and Japanese design documents before durable schema changes.
- [x] 2026-09-08: Decide and document the application/workload manifest shape.
- [x] 2026-09-08: Add strict manifest parsing and tests that reject invalid manifests.
- [x] 2026-09-08: Add Flutter SDK/project prerequisite diagnostics.
- [x] 2026-09-08: Implement build orchestration from the pinned source.
- [x] 2026-09-08: Persist APK digest and build identity/evidence.
- [x] 2026-09-08: Implement install and launch against the owned Emulator serial.
- [x] 2026-09-08: Implement endpoint resolution and lease-owned `adb reverse` bindings.
- [x] 2026-09-08: Integrate application lifecycle with create compensation.
- [x] 2026-09-08: Integrate application observation with show/doctor/reconcile.
- [x] 2026-09-08: Extend named-test interpolation with explicit Android runtime identity.
- [x] 2026-09-08: Add mixed `api`, `dashboard`, `mobile` and `full` stack fixtures.
- [x] 2026-09-08: Prove two concurrent mobile leases do not collide.
- [x] 2026-09-08: Add failure/recovery/quarantine coverage.
- [x] 2026-09-08: Final implementation Go 1.26.8 full harness passed all
      phases; Go 1.27.1 `go test -race ./...` passed. Latest-source CI passed at `81102f1` and validated docs at `d4d4289`.
- [x] 2026-09-08: Native CI `34166963420` passed all OS/toolchain jobs at
      `8975096`; real SDK evidence remains separate and Linux-only.
- [x] 2026-09-08: Run real Flutter + Emulator integration on Linux;
      `TestRealFlutterAndroidBackendLease` passed in 248.46s. Two-real-lease
      extension subsequently passed in 88.99s; native Windows/macOS SDK runs remain unverified.
- [x] 2026-09-08: Revalidated final acceptance after late CI failure;
      test-only correction `bb55393` passed both complete CI runs. Previous checkpoint: completed evidence and retrospective;
      independent read-only final review reported no concrete findings.
- [x] 2026-09-08: Re-archived both languages after correction CI passed. Previously moved according to
      `docs/PLANS.md`.

A checked item means observed completion, not intention. Add the UTC date,
command/test/run identifier and relevant result when checking an item.

## Surprises & Discoveries

- 2026-09-08: Reopened after archival commit `233192d`: PR CI
  `34169150614` failed on Ubuntu Go 1.27 in
  `TestCleanupStopsIndependentEffectsAfterOperationLockLoss` (3.04s). The test
  expected only `down:d-failing` but also observed `down:c-failing`. The cause
  is not established; lost-lock fixture/store synchronization is one hypothesis,
  not a finding. Push CI `34169148759` had 11 of 12 jobs passed at this record.
  Restore active execution authority before code investigation; preserve all
  prior successful evidence and the earlier archival history. Revalidate and
  re-archive only after the necessary correction is verified.


- 2026-09-08: The supplied plan has no Japanese sibling yet. Add and maintain
  that translation before committing the milestone. Existing Android reservations
  and operation fencing can be reused without changing their ownership rules.

Record at least:

- Flutter CLI behavior that differs by OS;
- Gradle/Java/Android SDK interaction that changes the design;
- unexpected writes to tracked source files;
- APK/package/activity discovery limitations;
- `adb install`, `adb reverse` or launch behavior that changes cleanup semantics;
- Compose endpoint-resolution limitations;
- Flutter integration tests that rebuild or replace the lifecycle-installed APK;
- test/CI environments that cannot provide real Emulator acceleration;
- any conflict with the existing Android ownership and ADB-server rules.

Do not convert an unexpected limitation into a silent test skip or weaker
ownership assertion.

- 2026-09-08: Unit and race tests cover simultaneous mobile leases with
  disjoint source/APK paths, digest evidence, Compose projects, Android identities,
  and reverse host ports. Failure fixtures cover build/install/launch,
  unconfirmed mapping ownership, package loss, reverse loss and forced cleanup.
- 2026-09-08: Independent review found that initial build quarantine alone did
  not prevent a later forced destroy from deleting a potentially live build's
  source. Persisting `build_unconfirmed` before execution now blocks later
  cleanup, including after restart. Recovery requires investigating process
  termination evidence; the CLI does not silently clear this guard.
- 2026-09-08: A reverse request is not ownership proof. If establishment was
  never confirmed and a mapping exists, cleanup retains it and quarantines.
  It may proceed once the mapping is absent. Durable-write errors halt cleanup.

- 2026-09-08: Final snapshot review found that launch intention alone could let
  reconcile mark an application READY after a crash before activity start.
  `launch_confirmed` now persists only after successful launch. Observation also
  requires confirmed build termination and matching executable/project directory.
  `TestMobileReconcileRejectsIncompleteApplicationSnapshot` covers launch,
  executable and directory inconsistencies. These guards were subsequently
  committed as `e2c23f8` and passed final local harness/race validation.

- 2026-09-08: The two-real-lease fixture built both APKs successfully, but both
  Emulator launches reported insufficient free space for their userdata
  partitions. The default temporary directory is memory-backed and the template
  requires 12 GiB free. Allow normal readiness failure and compensation to
  finish, then rerun with a process-local `TMPDIR` on a disk with ample space.
  No template resizing or weakened checks was used. The later disk-backed
  two-lease run passed in 88.99s after helper isolation; see final evidence.

- 2026-09-08: Final safety review found that clearing the process uncertainty
  guard after build exit, while required build-evidence writes failed, allowed
  later cleanup to delete the source/APK after store recovery. That approach was
  insufficient: execution termination and evidence persistence are independent.
  A durable `build_evidence_incomplete` flag is now set before build and cleared
  only after both required log artifacts and final lease state persist.
  `TestMobileIncompleteBuildEvidenceBlocksCleanup` injects `SaveArtifact` failure
  and checks APK retention. This guard subsequently passed the final local
  harness/race validation.

- 2026-09-08: The latest strict two-debug-lease real run brought both leases
  to READY; both Flutter HTTP markers and named device tests passed. First
  Destroy then quarantined a live process group after the Emulator leader
  exited. A cleanup retry failed while the sibling remained live; second-lease
  cleanup succeeded, and both process groups were empty after the sibling died.
  Emulator logs showed the same netsim Wi-Fi localhost port 34339. This suggests
  shared `netsimd` may have kept the first group live, but that cause is not yet
  proven by those logs alone. Investigation led to the private-helper isolation
  below. Normal recovery later succeeded, and the final two-lease run passed
  in 88.99s; ownership checks and quarantine were not weakened.

- 2026-09-08: After the sibling exited, normal first-lease Destroy succeeded
  at 22:52:12 UTC. Both leases were RELEASED; exact scoped Compose queries,
  worktrees, AVD state, ADB devices and process groups were empty. The original
  150.28s real debug attempt remains a failed integration, not a retroactive pass.
- 2026-09-08: A focused SDK 37.1.11 build 15917651 / netsimd 0.3.114 daemon
  probe confirmed lease-private `netsim.ini`, gRPC port 39107, HCI 0 configuration
  and libslirp enabled, then stopped the probe daemon. Public source used to
  understand discovery is not asserted to match this installed release exactly.
  The implementation now supplies aligned lease-private discovery directories
  and dynamic HCI configuration. The full local harness/race and two-lease
  real rerun subsequently passed; see final evidence.

- 2026-09-08: `TestLockLossFixtureWaitsForSQLiteWriter` reproduced a defect in
  the test's lock-loss injector: with a real SQLite writer held, the old helper
  failed `database is locked (5) (SQLITE_BUSY)` in 0.063s before replacing the
  token. The original test passed 50 repeats (30.267s), and the historical CI did
  not directly log that SQLite error. This is a reproduced fixture defect and a
  probable matching mechanism, not proof of the historical failure's exact cause.
  The helper now uses `SetMaxOpenConns(1)` and `PRAGMA busy_timeout=10000`, matching
  store policy; errors immediately call `t.Errorf`, and exactly one affected row
  is required. Production fencing and the original exact no-later-effects
  assertion are unchanged. The test of lock-loss injection while a real SQLite writer is held and existing cleanup
  lock-loss test passed `-race -count=30` (33.834s). The full Go 1.27 harness
  passed (app 5.794s), as did full `go test -race ./...` (app 25.304s).
  This test's 50ms wait schedules a bounded writer release, not a short
  success deadline. Fixed-source CI subsequently passed at `bb55393`, and both plans were re-archived.

## Decision Log

- Decision: Use `applications` and `component.application`; store additive
  application identity in the existing Lease JSON payload. Build all selected
  applications immediately after source materialization, before any runtime Up.
  Rationale: Keeps the resource boundary intact, preserves old rows/manifests,
  and avoids heavyweight allocation on build failure.
  Date/Author: 2026-09-08 / implementation.


- Decision: Keep Flutter application lifecycle separate from the existing
  `android-emulator` runtime provider.
  Rationale: PR #2 deliberately made the Emulator independently leaseable and
  Flutter-agnostic. Build/install/launch are application workload concerns and
  must not weaken device ownership or cleanup.
  Date/Author: 2026-09-08 / maintainers.

- Decision: A Flutter Android application must use a pinned managed source and
  an already selected Android runtime in the same lease.
  Rationale: Source provenance and device ownership must be explicit; an
  application must not attach to an arbitrary external Emulator.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Backend connectivity uses explicit lease-owned `adb reverse`
  mappings to resolved TCP endpoints rather than rebuilding the application with
  per-lease host ports.
  Rationale: Every Emulator can use the same stable device-local URL while host
  endpoint allocation remains isolated per Compose project.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Build commands remain argv arrays executed directly without shell
  evaluation.
  Rationale: Preserves Windows/macOS/Linux portability and the repository's
  existing execution-security boundary.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Preserve the SHA-256 digest and build evidence of the APK actually
  installed during lease creation, but do not add long-term APK promotion in
  this plan.
  Rationale: This plan needs auditability without prematurely introducing an
  artifact registry/retention subsystem.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Durable human-facing Flutter Android product and design
  documentation is delivered in English and Japanese.
  Rationale: Repository documentation is intended to be maintained bilingually
  going forward.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Reject an already-installed declared package before installing the
  APK, then require that package after install. This avoids attributing an
  unrelated APK digest to a package supplied by the template system image.
  Date/Author: 2026-09-08 / implementation following independent review.
- Decision: Reverse references must resolve to loopback TCP host endpoints;
  reject remote Docker endpoints instead of discarding their host address.
  Activity names exclude shell metacharacters, including `$` nested classes.
  Date/Author: 2026-09-08 / implementation.
- Decision: Repository Flutter doctor checks the current declared source
  checkout and every application, without allocating worktrees or requiring
  Docker. Create checks pinned projects and selected dependencies separately.
  Date/Author: 2026-09-08 / implementation.

- Decision: Persist launch success separately from launch intent, and validate
  execution identity against the recorded build when observing applications.
  Rationale: Package presence and an intended launch are insufficient evidence
  that the initial lifecycle completed after a crash; foreground state is still
  deliberately not required.
  Date/Author: 2026-09-08 / implementation following snapshot review.

- Decision: Persist evidence incompleteness independently of process uncertainty.
  Rationale: Successful process termination cannot authorize deletion when
  required build logs or lease finalization failed to persist. Reconcile must
  quarantine and normal/force cleanup must preserve source/APK even after the
  store recovers, until the evidence failure is investigated and recovered.
  Date/Author: 2026-09-08 / implementation following final safety review.

- Decision: Settle the remaining initial contract questions explicitly: require
  package/activity instead of APK inference; use the first build argv element
  for PATH or explicit-path Flutter selection and version discovery; keep serial
  interpolation Android-specific; exclude mutable cache metadata from evidence;
  keep real Flutter/Emulator validation opt-in on an accelerated host.
  Rationale: These choices preserve strict portable behavior, avoid extra APK
  tools or a premature generic framework, and distinguish build provenance and
  actual SDK execution from unsupported reproducibility/platform claims.
  Together with the existing applications/Lease-JSON decisions, this resolves
  all seven initial Milestone 1 questions.
  Date/Author: 2026-09-08 / implementation contract reconciliation.

- Decision: Isolate generic Android Emulator helper discovery with child-only
  `TMPDIR`/`TMP`/`TEMP`/`XDG_RUNTIME_DIR` under `AVDHome/emulator-data/Temp`, and
  Windows child `LOCALAPPDATA` under `AVDHome/emulator-data`. Set
  `NETSIM_INSTANCE=1`, `NETSIM_HCI_PORT=0` and `-netsim-args --no-web-ui`.
  Rationale: The Emulator client and daemon need matching private discovery
  locations; the default instance must agree, HCI must not claim a shared fixed
  port, and the auxiliary web UI must not bind default 8080. Keep radio/guest
  networking, shared ADB policy and process ownership/cleanup proof unchanged.
  A focused installed-SDK probe supports these settings. The subsequent 88.99s
  real two-lease run established complete sibling-preserving lifecycle behavior.
  Date/Author: 2026-09-08 / implementation following SDK investigation.

- Decision: Repair only the test's external lock-loss injector to wait for the
  real SQLite writer and prove that one token was replaced before asserting
  subsequent fencing behavior; report injection errors at their source.
  Rationale: An unconfigured raw connection can fail before actually injecting
  lock loss, causing a misleading later-effects failure. Match existing store
  busy handling without changing production fencing or weakening exact effect
  ordering assertions.
  Date/Author: 2026-09-08 / implementation following late-CI fixture reproduction.

## Outcomes & Retrospective

Re-completed after the late archival CI failure at `233192d`. The test-only
lock-loss injector correction `bb55393` passed full local harness/race and both
complete CI runs. Re-archived on 2026-09-08. The fixture defect was reproduced;
the exact historical CI SQLite error was not logged and remains unproven.
Production fencing and exact no-later-effects assertions were preserved.


Delivered outcome: `applications` binds strict pinned Flutter builds to
independently owned Android runtimes. Builds precede expensive runtime creation;
install, exact package verification, loopback reverse and confirmed launch precede
READY. Existing Lease JSON retains source/build/APK/network identity and separate
process, evidence and launch confirmation. Named tests use observed owned serials
and explicitly distinguish possible test-APK rebuilding from creation provenance.
Two real debug leases now prove separate guest connectivity, private helper
lifetimes and sibling-preserving non-force cleanup. UI observation, long-term APK
promotion and real Windows/macOS SDK validation remain follow-up work.

Retrospective: green injected tests did not expose a real shared netsimd
lifetime coupling. Strict process-group cleanup correctly quarantined that case;
fixing helper discovery and port ownership preserved the safety check. Process
termination, required evidence persistence and launch success needed independent
durable confirmation to survive crashes and later forced cleanup. Tests now inject
unconfirmed process termination, evidence-persistence failure and launch failure
to exercise each distinction. Timing-based fake readiness failures
required deterministic triggers without removing wait-loop cancellation coverage.
Temporary-storage sizing is an infrastructure prerequisite, not a reason to resize
templates or weaken checks. Local final race/harness and latest-source native CI passed. Both language
versions are archived with the failure history and remaining real-SDK platform gaps.


Completed on 2026-09-08. Contracts (`4457dfd`), initial lifecycle (`8975096`),
recovery guards (`e2c23f8`), diagnostic/test fixes (`aee0982`) and final private
helper isolation/two-lease fixture (`81102f1`) were delivered through PR #4.
The validated bilingual documentation checkpoint is `d4d4289`. All F1–F21
requirements have direct evidence below, including the successful two-real-lease
Linux run and complete native CI. Human PR approval/merge is separate from this
implementation and plan completion. Real Windows/macOS Flutter/Emulator runs
remain an explicitly recorded platform gap, as permitted by F20/F21.

## Context and Orientation

Read these before implementation:

- `AGENTS.md`
- `ARCHITECTURE.md`
- `docs/PLANS.md`
- `docs/PORTABILITY.md`
- `docs/RELIABILITY.md`
- `docs/SECURITY.md`
- `docs/QUALITY.md`
- `docs/roadmap.md`
- `docs/product-specs/agent-env-mvp.md`
- `docs/product-specs/android-emulator.md`
- `docs/design-docs/android-emulator.md`
- `docs/exec-plans/completed/agent-env-mvp.md`
- `docs/exec-plans/completed/android-emulator-lease.md`

Current architecture:

- `internal/config` strictly decodes `.agent-env.yaml`.
- `internal/stack` selects root components and all their direct and indirect dependencies in a deterministic order.
- `internal/domain` contains lease/source/component/runtime/resource/event state
  without concrete adapters.
- `internal/app` owns orchestration, readiness, compensation, evidence and
  reconciliation policy.
- `internal/runtime/compose` owns Compose-specific effects.
- `internal/runtime/android` owns Android Emulator process/AVD/port/device
  effects and must remain Flutter-independent.
- `internal/execx` owns portable command/process boundaries.
- `internal/store/sqlite` owns desired state, resource identity, reservations,
  events, runs and artifacts.
- `internal/evidence` owns redaction and artifact digest helpers.
- `internal/paths` owns source-confined path resolution.
- CLI packages format/decode command input but do not own lifecycle policy.

The current named-test implementation only interpolates `${lease_id}` and
`${env:NAME}`. Flutter integration tests need a safe explicit way to address the
lease-owned Emulator. Prefer extending interpolation with a stable runtime-scoped
token such as `${android:phone:serial}` rather than relying on ambient
`ANDROID_SERIAL`, implicit `adb devices` selection, or shell substitution.

The current Android contract deliberately excludes Flutter build, APK install,
`adb reverse`, UI interaction, screenshot and logcat. This plan consumes that
stable boundary rather than modifying its ownership guarantees.

## Plan of Work

### Milestone 1 — Contract and application model

Inspect the current manifest, domain runtime/component shapes, execution
snapshot and SQLite schema.

Write:

    docs/product-specs/flutter-android-runtime.md
    docs/product-specs/flutter-android-runtime.ja.md
    docs/design-docs/flutter-android-runtime.md
    docs/design-docs/flutter-android-runtime.ja.md

Update English/Japanese indexes according to the repository's current
bilingual-documentation policy. If the global bilingual migration has not yet
merged, this plan must still provide both language versions for newly introduced
Flutter Android durable docs and must not duplicate unrelated translation work.

Decide whether the manifest field is named `applications`, `workloads`, or a
more constrained equivalent. The final design must maintain these invariants:

1. application != Android Emulator runtime;
2. application source is pinned and materialized by the lease;
3. application targets an explicitly selected lease-owned Emulator;
4. application configuration is strict and portable;
5. old manifests remain valid without opting into Flutter;
6. application identity is persisted and inspectable.

If this introduces a new architectural node/dependency direction, update
`ARCHITECTURE.md`, its Japanese sibling when present, `repoctl arch-check`, and
an ADR in the same coherent change.

Observable result:

    agent-env plan . --stack mobile --output json

shows the selected Flutter application, its source, target Android runtime,
build artifact path and reverse-binding requirements without building,
allocating ports, starting Docker, starting an Emulator, or changing the SDK.

### Milestone 2 — Flutter prerequisite discovery

Implement injected, testable Flutter CLI discovery.

Validate without installing or modifying toolchains:

- configured/selected Flutter executable is runnable;
- version information is parseable and retained as evidence;
- declared project directory exists inside the pinned source;
- Flutter project metadata required for Android build exists;
- declared APK output path is source-relative and confined;
- package/activity values are syntactically valid;
- target runtime is `android-emulator`;
- endpoint references resolve to selected dependency components;
- reverse bindings are TCP and device ports are valid/unique.

Do not run `flutter doctor` in a way that downloads SDK artifacts or modifies
the host as part of normal prerequisite checking.

Observable result:

    agent-env doctor <repository> --runtime flutter-android

or the final documented equivalent reports Flutter/project prerequisites
without allocating an environment.

### Milestone 3 — Build and APK provenance

Build after pinned worktrees exist but before expensive runtime allocation where
the orchestration can safely do so.

The build command:

- is an argv array;
- runs in the declared source/project directory;
- receives only documented environment;
- is bounded and cancellable through `execx`;
- cannot escape the pinned source through manifest paths;
- records stdout/stderr as evidence;
- fails create without starting unrelated heavyweight resources when possible.

After a successful build:

- verify the declared artifact exists;
- reject symlink/path escape;
- require a regular APK file;
- compute SHA-256;
- persist build command, source commit, Flutter version, artifact path and digest;
- distinguish build evidence from long-term promoted artifacts.

Do not claim a reproducible build merely because the APK digest was recorded.

Observable result:

`show --output json` exposes the build/install identity needed to answer:
"Which pinned source and which APK digest did this lease install?"

### Milestone 4 — Install, reverse bindings and launch

Once the selected Android runtime is READY:

1. install the recorded APK using the exact owned Emulator serial;
2. verify the declared package is installed on that serial;
3. resolve each declared dependency endpoint from the actual Compose lease
   observation;
4. create `adb reverse` mappings on that exact serial;
5. verify the installed mappings;
6. launch the declared activity;
7. record application READY only after the lifecycle conditions succeed.

Never select a device by "first `adb devices` entry". Never attach to an
external serial.

For example, if two leases use:

    device localhost:8080

they may map independently to:

    emulator-5554 localhost:8080 -> host 127.0.0.1:49173
    emulator-5556 localhost:8080 -> host 127.0.0.1:49218

The device URL remains stable while Compose host ports remain lease-specific.

`adb reverse` must use the existing Android adapter's selected compatible local
ADB-server policy. Do not introduce a second independent ADB daemon policy.

### Milestone 5 — Compensation, destroy and reconcile

Extend the create saga with application effects.

Preferred compensation ordering after application effects begin:

    launch/install/reverse failure
      -> remove owned reverse bindings when identity is confirmed
      -> retain build evidence
      -> clean independent runtime resources in existing reverse order
      -> clean sources only after runtime/application cleanup is confirmed

Because the Emulator uses private per-lease writable state, successful Emulator
destruction removes the installed application state. Normal destroy need not
invent a separate global uninstall phase if no application state survives outside
the private AVD.

Reconcile must inspect:

- target Android runtime still has confirmed identity;
- declared package remains installed;
- required reverse mappings still point to the recorded host endpoints;
- APK/build identity in the registry remains internally consistent.

Do not require the Flutter process/activity to remain foreground or running after
creation; users and tests may legitimately background or close the app.

If the package is manually uninstalled or a required reverse mapping disappears,
the active lease becomes DEGRADED. Identity ambiguity must quarantine rather than
repair or kill an unproven device.

### Milestone 6 — Named tests and Flutter integration tests

Extend test interpolation with explicit lease-runtime identity.

Target syntax:

    ${android:<runtime-name>:serial}

Example:

```yaml
tests:
  mobile-e2e:
    stack: mobile
    source: mobile
    working_directory: .
    command:
      - flutter
      - test
      - integration_test
      - -d
      - ${android:phone:serial}
```

The interpolation must:

- resolve from the pinned lease snapshot/observed owned runtime;
- fail on unknown or absent runtime;
- reject non-Android runtime references;
- never use ambient device selection;
- preserve existing `${lease_id}` and `${env:NAME}` behavior.

Important: a Flutter `integration_test` command may build/reinstall another test
application. If so, evidence must not claim it executed the exact APK recorded
during lifecycle creation. Record that distinction in product/design docs and
test output. Exact black-box interaction with the lifecycle-installed APK belongs
to the Android UI observer work unless a test mechanism proves otherwise.

### Milestone 7 — Stack fixtures and concurrency

Add realistic fixtures proving:

    api
    dashboard
    mobile
    full

select the intended root components and all their direct and indirect dependencies.

At minimum prove:

- `api` does not allocate Android/Flutter;
- `dashboard` does not allocate Android/Flutter unless explicitly required;
- `mobile` selects API + Emulator + Flutter app;
- `full` selects API + Dashboard + Emulator + Flutter app;
- two concurrent `mobile` leases have distinct worktrees, Compose projects,
  host endpoints, AVD state, serials and APK build records;
- both can use the same device-side reverse port;
- destroying one leaves the sibling READY and usable.

### Milestone 8 — Native and real integration evidence

Fake/injected adapter and path/argv tests must execute on supported native
Windows, macOS and Linux CI.

Run the full repository harness:

    go run ./tools/repoctl check

and the repository's documented Go race suite.

Add an opt-in real integration fixture that uses:

- real Git worktree materialization where appropriate;
- real Compose backend;
- real Flutter CLI/build;
- real Android Emulator lease;
- real APK install;
- real `adb reverse`;
- real activity launch;
- final cleanup.

The fixture must fail rather than silently skip after it has explicitly been
selected and prerequisites are missing.

If suitable real Windows/macOS Flutter+Emulator runners are unavailable, record
that fact as an unresolved native-execution gap. Cross-builds and fake native
tests must not be reported as real Emulator/Flutter validation.

## Concrete Steps

1. `git switch master && git pull --ff-only`.
2. Record the exact starting revision in this plan.
3. Create and switch to `feat/flutter-android-runtime`.
4. Read the repository harness and all context listed above.
5. Inspect config/domain/app/runtime/store schemas before changing them.
6. Create the bilingual product and design docs.
7. Record the final manifest/application-model decision; add/update ADR as needed.
8. Add strict manifest parsing and tests that reject invalid manifests.
9. Add persistence migration only after durable fields are defined.
10. Add Flutter command/discovery adapter behind injected interfaces.
11. Add application build and evidence recording.
12. Add install/launch/reverse behavior using the existing Android ownership model.
13. Add create compensation and destroy/reconcile integration.
14. Extend test interpolation with explicit Android serial identity.
15. Add stack-resolution and concurrent-lease tests.
16. Add real integration fixture.
17. Repeatedly run focused tests, full harness and race tests during integration.
18. Push coherent milestone commits without rewriting published history.
19. Record every acceptance item with direct evidence.
20. Fill Outcomes & Retrospective.
21. Move this plan to `docs/exec-plans/completed/` and update all links.

## Validation and Acceptance

| ID | Required behavior | Evidence |
| --- | --- | --- |
| F1 | Existing Compose-only and Android-only manifests remain valid and behaviorally unchanged. | PASS at `8975096`, Go 1.26.8 full check: `TestStrictManifest`, `TestLifecycleCreatePersistedIntentAndUniqueIsolation`, `TestAndroidConcurrentLeasesAndSiblingCleanup`. |
| F2 | Invalid Flutter application configuration fails strict manifest validation before side effects. | PASS same check: `TestFlutterManifestContract`, `TestFlutterApplicationCollisionsAndDependencyClosure`, `TestApplicationEndpointReferenceWithDots`. |
| F3 | Planning a mobile stack reports Flutter/application requirements without build/runtime effects. | PASS same check: `TestFlutterPlanJSONIsPureAndIncludesApplication`, `TestMobilePlanStackClosure`. |
| F4 | Missing Flutter executable/project prerequisites are reported as prerequisite failures without leaked runtime resources. | PASS same check: `TestMobilePrerequisiteNoReservation`, `TestFlutterDoctorMissingExecutableDoesNotRequireDockerOrAllocateState`, `TestInvalidPrerequisitesDoNotRunCommands`. |
| F5 | Build runs from the pinned source with argv-only execution and records source commit, Flutter version, logs and APK SHA-256. | PASS same check: `TestBuildArgvDirectoryAndEvidence`, `TestMobileEvidenceDoesNotKeepRawBuildOutputInLease`, `TestNativeBuildPreservesArgvAndProjectDirectory`; real Linux run below also passed. |
| F6 | The exact recorded APK is installed only on the selected lease-owned Emulator serial. | PASS same check: `TestApplicationOperationsUseOwnedSerialAndLocalServer`, `TestAPKInstallRequiresExplicitSuccess`, `TestMobileConcurrentLeasesAndObservation`; real Linux install passed. |
| F7 | Package/activity launch succeeds or create compensates safely. | PASS same check: `TestActivityLaunchReportsExitZeroFailures`, `TestMobileFailureCompensation`; real Linux launch passed. |
| F8 | `adb reverse` binds a declared device TCP port to the actual selected Compose endpoint and records the mapping. | PASS same check: `TestApplicationOperationsUseOwnedSerialAndLocalServer`, `TestReverseEndpointRejectsRemoteAndMalformed`; real Dart HTTP request reached the Compose backend. |
| F9 | Two mobile leases may use the same device-side port while host endpoints and Emulator identities remain isolated. | PASS injected concurrency/race and real two-debug-lease run (88.99s): isolated devices/backends/reverse ports; normal first Destroy preserved sibling READY and a fresh guest HTTP request. Exact evidence below. |
| F10 | Failed reverse/install/launch leaves no unowned effect and preserves required evidence. | PASS same check: `TestMobileFailureCompensation`, `TestMobileUnconfirmedReversePreservesMapping`, `TestMobileUnconfirmedBuildBlocksLaterForcedCleanup`, `TestReverseCleanupRequiresExactMapping`. |
| F11 | Manual package removal or required reverse loss is observed as DEGRADED. | PASS same check: `TestMobileConcurrentLeasesAndObservation` explicitly removes package and mapping, each yielding DEGRADED. |
| F12 | Reconcile never attaches to or kills an unproven external device. | PASS same check: `TestMobileUnknownIdentityAndCleanupPreservation`, `TestApplicationOperationsRefuseOwnershipOrServerMismatch`, `TestApplicationChecksServerAgainAfterObservation`. |
| F13 | Destroying one mobile lease leaves a sibling mobile lease READY and usable. | PASS injected concurrency/race and real two-debug-lease run (88.99s): isolated devices/backends/reverse ports; normal first Destroy preserved sibling READY and a fresh guest HTTP request. Exact evidence below. |
| F14 | `${android:<runtime>:serial}` resolves only the lease-owned selected Android runtime and works without ambient device selection. | PASS same check and repeated focused run: `TestAndroidSerialInterpolation`, `TestMobileConcurrentNamedTestsUseOwnedSerialAndPersistWarning`. |
| F15 | Named-test evidence distinguishes tests that rebuild/reinstall from the lifecycle-installed APK identity. | PASS same check and repeated focused run: `TestMobileConcurrentNamedTestsUseOwnedSerialAndPersistWarning`, `TestMobileFailedNamedTestRetainsEvidenceAndReadyLease`. |
| F16 | `api`, `dashboard`, `mobile` and `full` stacks resolve to the documented lightest component sets. | PASS same check: `TestMobilePlanStackClosure` covers all four named stacks. |
| F17 | New durable Flutter Android product/design docs exist in both English and Japanese and are indexed. | PASS `docs-check` within Go 1.26.8 full check at `8975096`; English/Japanese product/design docs and both indexes present. |
| F18 | Architecture/docs validators pass after any new application/workload dependency boundary is introduced. | PASS `arch-check` and `docs-check` in same full check; `TestArchitectureBoundaries` includes tests that explicitly introduce and detect forbidden Flutter cross-adapter dependencies. |
| F19 | Full repository harness and Go race checks pass on final implementation. | PASS final local implementation: Go 1.26.8 full harness all phases and Go 1.27.1 full `go test -race ./...`. Latest implementation CI passed at `81102f1`; validated documentation CI passed at `d4d4289`. Earlier failures retained below. |
| F20 | Native Windows/macOS/Linux portability evidence is recorded honestly and separately from real Flutter+Emulator execution. | PASS recorded separately: Linux real SDK run; native six OS/Go and five cross-build plus integration CI succeeded at final implementation `81102f1` and documentation `d4d4289`. Real Windows/macOS SDK runs remain unverified. |
| F21 | At least one real Flutter+Emulator+backend integration run passes when suitable local/CI prerequisites are available, or the missing infrastructure is explicitly recorded without substituting fake evidence. | PASS `TestRealFlutterAndroidBackendLease`: initial one-lease run 248.46s and latest two-debug-lease run 88.99s, including fresh sibling guest HTTP and both normal cleanups. Toolchain/source/APK evidence below. |

Acceptance requires direct evidence for every item. A test name without a
recorded successful run is not evidence.

## Idempotence and Recovery

Manifest planning and prerequisite checks are read-only.

Build failure may leave untracked Flutter/Gradle output inside the disposable
worktree. Cleanup must still obey existing tracked-dirty and source-ownership
rules. Never weaken dirty-source quarantine to delete build output.

Every external effect after build must have a durable identity before later
effects depend on it.

Application cleanup may remove only:

- reverse mappings proven to belong to the recorded Emulator serial and lease;
- application evidence under the lease artifact/state directory;
- private application state indirectly removed by confirmed destruction of the
  lease-owned private AVD.

Do not globally:

- run `adb kill-server`;
- delete shared Android SDK/Flutter caches;
- delete user AVD templates;
- uninstall packages from unrelated devices;
- clear an arbitrary serial;
- remove ports/resources based only on current availability.

Repeated destroy/reconcile must be safe. A reused host port, Emulator serial,
PID, AVD name or package on another device must not be mistaken for the original
lease-owned resource.

On ambiguous Android identity, retain reservations/evidence and quarantine.
`--force` must not bypass ownership proof.

## Artifacts and Notes

Re-completion evidence (2026-09-08):

- Test-only correction `bb55393`: [push CI 34169765493](https://github.com/mahcialet/agent-env/actions/runs/34169765493)
  and [PR CI 34169767777](https://github.com/mahcialet/agent-env/actions/runs/34169767777)
  both succeeded in all 12 jobs. Together with the local full harness/race and
  independent review below, this closes the reopening caused by `233192d`.
- Both plans are completed again with corrected app harness duration 5.794s,
  preserved failure history and explicit uncertainty about the historical CI's
  exact SQLite error. No production fencing behavior or original exact effect
  assertion was changed. All acceptance requirements remain satisfied.


- Final self-review also applies `SetMaxOpenConns(1)` and
  `PRAGMA busy_timeout=10000` to the held-writer test connection before `Begin`,
  preventing contention setup itself from racing registry renewal. Production
  and exact assertions are unchanged. The prior full harness/race validates
  the injector repair; targeted 30-repeat race validation of this small setup
  adjustment passed (33.561s).


Reopened-plan correction checkpoint (2026-09-08):

- Archival commit `233192d` produced both outcomes: push CI `34169148759`
  succeeded; PR CI `34169150614` failed the lock-loss test. Preserve both results.
- The reproduced fixture correction passed targeted `-race -count=30` (33.834s),
  complete Go 1.27 harness (app 5.794s), and full Go 1.27
  `go test -race ./...` (app 25.304s).
- A separate read-only reviewer inspected the actual test-only diff and found
  no concrete issue: production fencing and exact assertions were unchanged,
  this test’s goroutine joins/resource cleanup were safe, and the 50ms delay
  releases a held writer rather than imposing a short success deadline.
- The historical CI did not log the exact SQLite failure; the held-writer test
  reproduces the fixture defect, not the historical error message. Only
  fixed-source CI and re-archival were then completed at validated `bb55393`.


Final completion evidence (2026-09-08):

- Final implementation `81102f1`: [push CI 34168707969](https://github.com/mahcialet/agent-env/actions/runs/34168707969)
  and [PR CI 34168710704](https://github.com/mahcialet/agent-env/actions/runs/34168710704)
  both passed. Validated documentation checkpoint `d4d4289`:
  [push CI 34168751870](https://github.com/mahcialet/agent-env/actions/runs/34168751870)
  and [PR CI 34168753637](https://github.com/mahcialet/agent-env/actions/runs/34168753637)
  both passed all 12 jobs: six native Windows/macOS/Linux × Go 1.26/1.27, five
  cross-build targets and integration.
- A separate read-only reviewer, not the production namespace change's author,
  inspected `81102f1`: lifecycle/layout/ownership markers and full cleanup path,
  detached environment including Windows case folding/Job checks, unit mappings,
  and docs. No concrete findings were reported. This independent review and
  native CI do not replace actual Windows/macOS Flutter/Emulator execution.
- All acceptance items are satisfied with the scope-qualified evidence below.
  Both plans moved to completed and navigation was updated. Past failed checks,
  failed real attempts and real-SDK platform gaps remain preserved.


- Final implementation and two-lease fixture were committed as `81102f1` and
  pushed for latest-source CI. The local harness/race and successful real run
  below validate this implementation; that commit's CI subsequently passed.


- Final implementation local validation passed: Go 1.27.1
  `go test -race ./...` (app 22.990s, Android 2.145s, CLI 1.769s, SQLite 11.985s)
  and Go 1.26.8 full harness, all phases (Android 0.322s, CLI 0.228s,
  SQLite 5.810s). Latest implementation CI passed at `81102f1`; validated documentation CI passed at `d4d4289`.


Successful two-real-lease evidence (2026-09-08):

- `TestRealFlutterAndroidBackendLease` passed in 88.99s with two simultaneous
  debug APK leases on Linux amd64: Go 1.26.8, Flutter 3.47.2, Dart 3.13.2,
  JBR Java 25, Gradle 9.3.1, NDK 28.2.13676358. Generated pinned source:
  `58f38ffb6598bd08d890cf742119c74654311457`.
- `emulator-5554` used backend host port 32832 and APK SHA-256
  `adfc18293649617d5c969697811ce194974a396671813533560e021238564f62`;
  `emulator-5556` used backend host port 32833 and APK SHA-256
  `8bc3a8b1c27005b5097a63afb258a3491dba9e8a1aad2bff36d458e2cf835bef`.
  Both used device port 8080; isolated netsim listeners were 43277 and 41769.
- Both reached READY, made Flutter HTTP requests to their own backends and passed
  the named scoped ADB device test. After normal first Destroy, a fresh sibling
  guest HTTP request increased the observed baseline, proving surviving guest
  connectivity rather than merely reusing an old request or host-only check.
  Both normal Destroy operations passed and the fixture root was removed.
- At 22:59:38 UTC, ADB and both exactly scoped Compose projects were empty.
  This successful run supersedes the open two-lease acceptance gap but does not
  erase the historical failed attempts recorded below. Latest-source full race
  and Go 1.26.8 full harness subsequently passed; latest implementation and
  validated documentation CI then passed, satisfying the archival gate.


- Push CI `34168074785` and PR CI `34168077225` at `aee0982` passed all
  jobs (six native OS/Go jobs, five cross-build targets and integration).
  They predate the generic Android helper-isolation change and cannot validate it.
- The complete local Go 1.27 harness passed all phases after helper isolation.
  The subsequent two-lease real rerun passed in 88.99s. Installed-SDK
  evidence remains Linux-only; native Windows/macOS fake/process CI and
  cross-builds do not prove those platforms' actual Flutter/Emulator behavior.


- Diagnostic/test/plan checkpoint `aee0982` was pushed. The product description
  of the two-lease fixture was then an uncommitted update awaiting success;
  the subsequent 88.99s run now validates that behavior.
  The latest real run proved both strict debug startups, Flutter HTTP markers
  and named device tests, but did not pass final sibling-preserving cleanup;
  see the process-group discovery above. Do not count this as a passing complete
  two-lease integration.


- PR CI `34167569456` at `e2c23f8` passed all jobs; push CI `34167566940`
  at the same commit failed the timing-sensitive case recorded below. Retain
  both results; a successful sibling run does not invalidate the failure.
- Focused Android diagnostics/readiness tests passed `-race -count=30` (1.502s).
  Independent review requested explicit wait-loop cancellation coverage; a third
  `canceled_while_waiting` case cancels after the first post-start unready probe
  and returns `(false, nil)` so the select cancellation branch executes. Its
  repeated validation passed: all three readiness cases plus diagnostics,
  `-race -count=30`, 1.759s; `arch-check` also passed. The next two-debug-lease
  real run later passed in 88.99s; see successful evidence above.


Latest validation checkpoint (2026-09-08):

- The full Go 1.27 harness passed sequentially after real Emulator cleanup.
  Uncertainty, evidence and launch guards were committed/pushed as `e2c23f8`.
- Push CI `34167566940` failed on Ubuntu Go 1.27 in existing
  `TestSharedADBReadinessFailureNeverLaunchesEmulator/unavailable`: its 100ms
  deadline expired before fake shared-server startup (`starts=[]`). This was
  not an unexpected Emulator launch. A deterministic test repair cancels at
  `Start` and injects unavailable `ProbeADB` errors only after `Start`, retaining
  the exact startup and no-Emulator assertions. Repeated race validation is
  subsequently passed (three cases, `-race -count=30`, 1.759s); neither this
  repair nor later diagnostics inherit prior CI success.
- Install/launch failure diagnostics now retain bounded, redacted stdout/stderr;
  launch still requires explicit `Status: ok`. The prior two-real-lease disk
  attempt had an unknown activity status. Its cleanup retry was confirmed
  RELEASED. Later helper isolation produced the successful 88.99s two-lease run.


Evidence checkpoint (2026-09-08, after `8975096`):

- `go run ./tools/repoctl check` with Go 1.26.8 passed at `8975096`.
  Go 1.27.1 `go test -race ./...` passed earlier; subsequent focused race/recovery
  checks also passed. These runs establish the implementation checkpoint, not
  a claim that future fixture extensions are already validated.
- A later Go 1.27.1 full check overlapped the real Emulator integration and
  failed existing `TestRealAdapterThroughAppPersistsOwnedResource`: its reserved
  port 5554 was occupied by that integration. Tests were not weakened. A
  sequential Go 1.27 full harness rerun after real cleanup subsequently passed.
- [Native CI run `34166963420`](https://github.com/mahcialet/agent-env/actions/runs/34166963420)
  passed at `8975096`: all six native OS × Go 1.26/1.27 jobs, five OS/architecture
  cross-build targets, and integration (race plus real Compose) passed.
  Native Windows/macOS real Flutter/Emulator execution remains unverified.
- Local Go 1.27.1 `go run ./tools/repoctl test-integration` subsequently exited 0.
  A later Reconcile guard, subsequently committed in `e2c23f8`, preserves uncertainty quarantine
  independently of desired state; focused test preserving quarantine after an unconfirmed build
  `TestMobileUnconfirmedBuildRemainsQuarantinedOnObservation` passed. Native CI
  at `8975096` does not establish verification of that later change.
- `go test -tags=flutterintegration -run TestRealFlutterAndroidBackendLease -v ./internal/cli -timeout=40m`
  passed on Linux amd64 in 248.46s: Go 1.26.8, Flutter 3.47.2, Dart 3.13.2,
  JBR Java 25, Gradle 9.3.1, NDK 28.2.13676358. Generated fixture source commit:
  `329768ca35b018608a31bb77b04655448f16d289`; installed APK SHA-256:
  `285597924dfe685f75573f0c6abfefe0bfabe1e4eb0e465dd497362b29d6cb88`.
  Build, install, package verification, reverse and launch passed. The Dart
  application's HTTP request reached the nginx backend; Show reported READY;
  non-force Destroy passed. Extending the fixture to two simultaneously active
  real leases subsequently passed in 88.99s, with exact evidence above.


Implementation milestone validation (2026-09-08):

- `go test ./internal/config`: pass, including tests that reject invalid configuration shapes and dependencies.
- `go test ./internal/app`: pass; mobile concurrency, compensation, degraded
  observations, owned-serial interpolation and durable uncertainty guards.
- `go test -race ./...` with Go 1.27.1: pass on Linux.
- `go test ./internal/runtime/android -count=10`: pass; Android generic argv,
  identity, shared-server, package, activity and reverse fixtures.
- `go test ./internal/app -run 'TestMobile(ConcurrentNamed|FailedNamed|Unconfirmed)' -count=10`:
  pass. Named tests use their own lease serial and retain APK provenance notes.
- Full harness initially failed on a concurrently unformatted new file, then
  on intentionally stale active-plan translation while this living document
  was being updated. Neither check was weakened; coherent checkpoint rerun
  is required before commit.
- Real Flutter fixture is opt-in under `flutterintegration`. An initial local
  run was interrupted by the implementation agent while clarifying SDK
  dependency setup; its failure is not product acceptance evidence. The
  unchanged fixture is being rerun with normal Flutter/Gradle dependencies.



For successful or failed lifecycle operations, retain compact evidence sufficient
to reconstruct what was attempted:

- lease ID;
- source alias and pinned commit;
- manifest digest;
- Flutter executable/version;
- build argv after secret redaction;
- build working directory;
- build stdout/stderr;
- APK relative path;
- APK SHA-256;
- Android runtime name and serial;
- package and activity;
- endpoint identity;
- device port and resolved host endpoint;
- install/reverse/launch result;
- relevant timestamps;
- cleanup/compensation result.

Do not store inherited secrets in SQLite, logs, argv snapshots or artifact
metadata. Use existing evidence redaction.

Large Gradle/Flutter caches are not evidence and must not be copied into the
lease artifact store.

## Interfaces and Dependencies

Expected new or extended interfaces should preserve app/domain separation.

A likely shape is conceptually:

    FlutterProvider.Validate(...)
    FlutterProvider.Build(...)

    AndroidApplicationProvider.Install(...)
    AndroidApplicationProvider.Inspect(...)
    AndroidApplicationProvider.ConfigureReverse(...)
    AndroidApplicationProvider.Launch(...)
    AndroidApplicationProvider.Cleanup(...)

The final interface names may differ after inspection. Do not create a provider
that owns or independently starts/stops the Emulator; that remains
`app.AndroidProvider`.

The application layer owns ordering and compensation across:

    source
      -> Flutter build
      -> Compose/Android runtime creation
      -> APK install
      -> endpoint resolution
      -> adb reverse
      -> activity launch
      -> application observation

The runtime adapter layer performs only concrete external operations requested
by app policy.

Expected external tools:

- `flutter`
- Flutter's compatible Dart/Gradle/Java/Android build toolchain
- Android `adb`
- Android Emulator through the existing `android-emulator` provider
- Docker Compose only when the selected stack requires Compose components

No POSIX shell, Bash, Make, PowerShell, symlink-based contract, CGO requirement,
implicit first-device selection, or fixed host-published port may be introduced
into the core workflow.

## Resolved Milestone 1 questions

All seven initial questions are resolved in the Decision Log and implemented
contracts:

1. `applications` with `component.application`; no generic workload framework.
2. Package and activity are always explicit, strictly validated identifiers.
   No APK inspection tool dependency or inferred activity is introduced.
3. `build.command[0]` selects Flutter on PATH or by explicit host executable path;
   the same executable supplies version evidence.
4. Additive application/build/reverse records live in existing Lease JSON and
   persist atomically with lease state; no SQL migration or separate table.
5. `${android:<runtime>:serial}` remains Android-specific; no broader runtime
   interpolation framework is introduced.
6. Retain Flutter version, pinned source, redacted build argv/logs and APK digest;
   do not retain mutable Flutter/Gradle cache metadata or claim reproducibility.
7. Real Flutter/Emulator/backend validation remains an explicit
   `flutterintegration` fixture on a suitable accelerated local/runner host.
   Existing CI covers native fake adapters and real Compose separately; it does
   not establish real Flutter/Emulator execution on Windows or macOS.
