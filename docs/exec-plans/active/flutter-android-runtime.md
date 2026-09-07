---
status: active
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
validation. Any final syntax must be documented and covered by negative fixtures.

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
- [x] 2026-09-08: Add strict manifest parsing and negative fixtures.
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
- [ ] Run the complete repository harness and Go race checks.
- [ ] Record native Windows/macOS/Linux evidence distinctly from real SDK tests.
- [ ] Run real Flutter + Emulator integration where prerequisites are available.
- [ ] Complete acceptance evidence and retrospective.
- [ ] Move this plan to `docs/exec-plans/completed/` according to
      `docs/PLANS.md`.

A checked item means observed completion, not intention. Add the UTC date,
command/test/run identifier and relevant result when checking an item.

## Surprises & Discoveries

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

## Outcomes & Retrospective

Not completed.

At completion, summarize:

- delivered manifest/application contract;
- final lifecycle ordering;
- exact provenance retained for the installed APK;
- endpoint/reverse ownership model;
- test integration semantics;
- native platform evidence;
- real Flutter/Emulator evidence;
- known platform or Flutter-version gaps;
- follow-up work for Android UI observation and artifact promotion.

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
- `internal/stack` resolves deterministic component dependency closure.
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

resolve to the intended component closure.

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
8. Add strict manifest parsing and negative fixtures.
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
| F1 | Existing Compose-only and Android-only manifests remain valid and behaviorally unchanged. | Pending |
| F2 | Invalid Flutter application configuration fails strict manifest validation before side effects. | Pending |
| F3 | Planning a mobile stack reports Flutter/application requirements without build/runtime effects. | Pending |
| F4 | Missing Flutter executable/project prerequisites are reported as prerequisite failures without leaked runtime resources. | Pending |
| F5 | Build runs from the pinned source with argv-only execution and records source commit, Flutter version, logs and APK SHA-256. | Pending |
| F6 | The exact recorded APK is installed only on the selected lease-owned Emulator serial. | Pending |
| F7 | Package/activity launch succeeds or create compensates safely. | Pending |
| F8 | `adb reverse` binds a declared device TCP port to the actual selected Compose endpoint and records the mapping. | Pending |
| F9 | Two mobile leases may use the same device-side port while host endpoints and Emulator identities remain isolated. | Pending |
| F10 | Failed reverse/install/launch leaves no unowned effect and preserves required evidence. | Pending |
| F11 | Manual package removal or required reverse loss is observed as DEGRADED. | Pending |
| F12 | Reconcile never attaches to or kills an unproven external device. | Pending |
| F13 | Destroying one mobile lease leaves a sibling mobile lease READY and usable. | Pending |
| F14 | `${android:<runtime>:serial}` resolves only the lease-owned selected Android runtime and works without ambient device selection. | Pending |
| F15 | Named-test evidence distinguishes tests that rebuild/reinstall from the lifecycle-installed APK identity. | Pending |
| F16 | `api`, `dashboard`, `mobile` and `full` stacks resolve to the documented lightest component sets. | Pending |
| F17 | New durable Flutter Android product/design docs exist in both English and Japanese and are indexed. | Pending |
| F18 | Architecture/docs validators pass after any new application/workload dependency boundary is introduced. | Pending |
| F19 | Full repository harness and Go race checks pass on final implementation. | Pending |
| F20 | Native Windows/macOS/Linux portability evidence is recorded honestly and separately from real Flutter+Emulator execution. | Pending |
| F21 | At least one real Flutter+Emulator+backend integration run passes when suitable local/CI prerequisites are available, or the missing infrastructure is explicitly recorded without substituting fake evidence. | Pending |

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

Implementation milestone validation (2026-09-08):

- `go test ./internal/config`: pass, including negative shape/dependency fixtures.
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

## Unresolved issues to settle during Milestone 1

1. Final manifest noun: `applications` versus `workloads` or a narrower
   Flutter-specific field.
2. Whether package/activity must always be explicit or may be safely discovered
   from the built APK without introducing fragile extra tool dependencies.
3. Whether Flutter executable selection is PATH-only in this phase or supports an
   explicit host configuration path.
4. Exact SQLite normalization for build/application/reverse identity versus
   resource metadata JSON.
5. Whether named-test Android serial interpolation should be generalized into a
   broader runtime-property interpolation system now or remain intentionally
   Android-specific.
6. How much Flutter/Gradle cache metadata is useful evidence without treating
   mutable caches as reproducibility inputs.
7. Whether real integration can run in existing CI or remains an opt-in
   acceleration-capable runner/local fixture.

Resolve these explicitly in the Decision Log before the associated public
contract is considered stable.
