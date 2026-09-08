---
status: completed
owner: maintainers
last_verified: 2026-09-08
---

# Observe and interact with lease-owned Android user interfaces

[日本語](android-ui-observer.ja.md)

This ExecPlan is a living document. Maintain it according to `docs/PLANS.md`.

Expected branch: `feat/android-ui-observer`.

PR #4 (`feat: run Flutter Android applications on owned emulator leases`) is a hard prerequisite. Start this plan from `master` only after PR #4 is merged. At implementation start, record the exact post-merge `master` revision below before making implementation changes.

Starting revision: `f6167becefa81082c914e32769fc015cdb1865fc`

Do not reimplement or weaken the existing Git source, Compose, Android Emulator, Flutter build/install, shared-ADB, reverse-mapping, cleanup, fencing, quarantine, or application-provenance rules. This work consumes those identities.

## Purpose / Big Picture

After this work, a coding agent can inspect and operate the visible Android UI inside an existing environment lease without adding test hooks to the target application.

Given a lease containing a confirmed owned Android Emulator, `agent-env` can:

- capture a semantic/accessibility-oriented UI snapshot;
- render a compact agent-readable tree with ephemeral node references;
- retain the raw snapshot as evidence;
- capture a PNG screenshot;
- tap a semantic node or explicit coordinate;
- replace text in an editable semantic node;
- press Back or Home;
- perform an explicit swipe;
- collect bounded, package-scoped logcat evidence;
- record every observation and action against the exact lease/runtime/device identity;
- reject stale node references rather than silently tapping coordinates from an obsolete screen.

The target application remains a black box. Flutter is not a requirement for the observer: Flutter applications benefit because their standard widgets expose a Semantics/accessibility tree on Android, while Android system UI and native dialogs remain visible through the same Android accessibility surface.

A typical workflow is:

    agent-env create . --stack mobile
    agent-env ui snapshot <lease-id> --application mobile-app
    agent-env ui screenshot <lease-id> --application mobile-app
    agent-env ui tap <lease-id> --snapshot <snapshot-id> --node n7
    agent-env ui set-text <lease-id> --snapshot <snapshot-id> --node n3 --text "user@example.com"
    agent-env ui snapshot <lease-id> --application mobile-app
    agent-env ui logcat <lease-id> --application mobile-app --since 30s
    agent-env destroy <lease-id>

The exact CLI spelling may be refined before the product contract is published, but the capability and safety properties in this plan are normative.

This is an Android analogue of a browser agent receiving a DOM/accessibility snapshot, screenshot, navigation primitives and runtime diagnostics. It is an observation and interaction layer, not a new environment runtime.

## Scope

In scope:

- a new Android UI observation/interaction application-layer boundary;
- selection of an already lease-owned Android runtime, directly or through a selected application;
- semantic/accessibility-oriented window snapshots;
- a deterministic normalized snapshot schema;
- compact human/agent text rendering plus structured JSON output;
- raw snapshot retention for debugging;
- ephemeral, snapshot-scoped node references;
- stale-snapshot protection for semantic-node actions;
- PNG screenshots;
- semantic-node tap/click;
- explicit coordinate tap;
- editable-node text replacement with Unicode-capable behavior;
- Back and Home navigation;
- explicit swipe gestures;
- bounded waits/polling needed to make snapshot/action workflows deterministic;
- bounded package-scoped logcat collection;
- observation/action evidence, digests and timestamps;
- operation fencing so destroy cannot race an in-flight UI command;
- diagnostics on confirmed live Android resources even when the application is not foreground;
- Windows, macOS and Linux host portability;
- native fake/backend tests on supported hosts;
- real Flutter + Emulator observer validation when suitable prerequisites are available;
- English and Japanese product/design/ExecPlan documentation.

Out of scope:

- changing the target repository to add UI Automator, Espresso, Patrol or test hooks;
- requiring target-app instrumentation dependencies;
- Flutter widget-tree or Dart VM Service introspection;
- private Flutter engine APIs;
- OCR as the primary locator mechanism;
- computer-vision element detection;
- accessibility conformance scoring;
- visual-regression baselines or pixel-diff policy;
- arbitrary Android shell execution exposed as a UI command;
- continuous unbounded logcat streaming;
- video recording;
- multi-touch/pinch/zoom in the first vertical slice;
- physical Android devices;
- remote Emulator hosts;
- iOS Simulator;
- malicious-code isolation.

## Architectural Intent

### Observer, not runtime

Android Emulator ownership remains in the existing Android provider. Flutter application ownership remains in the application/workload lifecycle introduced by PR #4.

The observer may inspect and interact only after resolving a runtime identity from the durable lease and confirming that the live device still matches the owned Android resource. It never allocates an Emulator and never discovers a target by taking the first entry from `adb devices`.

Conceptually:

    Lease
      -> selected component/application
      -> owned android-emulator runtime
      -> confirmed serial/process/AVD identity
      -> AndroidUIProvider
           -> snapshot
           -> screenshot
           -> action
           -> bounded logcat

### Accessibility is the semantic source

The semantic snapshot is based on Android's accessibility/UI Automator-visible window tree. Do not claim it is the Flutter widget tree.

Flutter standard widgets normally expose accessibility semantics, and custom Flutter controls may require correct `Semantics` configuration. If a control is rendered visually but absent from the accessibility tree, the observer must show that limitation honestly. A screenshot is complementary evidence, not a substitute for inventing semantic nodes.

The normalized snapshot should preserve enough raw Android properties to avoid overstating inferred roles. At minimum consider:

- window/package identity;
- class name;
- resource ID when present;
- visible text;
- content description;
- bounds;
- clickable / long-clickable;
- enabled;
- focusable / focused;
- editable;
- password;
- scrollable;
- selected;
- checkable / checked;
- accessibility importance where observable.

A compact rendering may derive labels such as `button`, `textbox` or `text`, but structured output must retain the underlying evidence used for that derivation.

### Node references are ephemeral

A snapshot may render:

    Snapshot: ui-01K...
    Window: com.example.app

    [n1] text    "Sign in"
    [n2] textbox "Email"       bounds=[72,240][1008,344]
    [n3] textbox "Password"    bounds=[72,376][1008,480] password=true
    [n4] button  "Login"       bounds=[72,528][1008,640] enabled=true

`n4` is meaningful only within that snapshot. It is not a durable selector and must not be stored in a repository manifest.

A mutating action using `--snapshot <id> --node <ref>` must re-observe enough current UI state immediately before the action to prove that the referenced node still resolves unambiguously to the expected window/node fingerprint. If it does not, fail with a stable stale/ambiguous-snapshot diagnostic and perform no input.

Do not silently fall back to the old node's center coordinate.

Explicit coordinate actions are a separate intentional primitive and must be recorded as coordinate actions rather than semantic-node actions.

### No target-app test dependency

The product contract must not require modifications to the application under test.

During Milestone 1, select and document the concrete Android backend after a focused spike. Candidate mechanisms include:

1. a versioned `agent-env` companion instrumentation APK using AndroidX UI Automator/accessibility APIs, installed only on the owned private Emulator;
2. Android platform-provided UI hierarchy/screenshot/input primitives where they meet the same Unicode, structured-error, stale-reference and evidence requirements.

The public `agent-env ui` contract must not expose which backend is used.

A companion helper is preferred if platform shell primitives cannot provide reliable Unicode text replacement, structured errors and semantic-node actions. If a helper is selected:

- it is an `agent-env` artifact, not a target-repository dependency;
- its version and SHA-256 are recorded in UI evidence;
- installation targets only the confirmed owned Emulator serial;
- it does not own or stop the Emulator;
- its packaging/build strategy preserves Windows/macOS/Linux runtime use;
- unrelated `agent-env` commands do not silently require a host Android build toolchain.

Do not adopt a pre-release AndroidX dependency implicitly. Record the exact API surface/version and rationale in the Decision Log.

## Progress

- [x] 2026-09-08: PR #4 merged at 00:16:34 UTC; `git pull --ff-only` confirmed current master at the starting revision above; created `feat/android-ui-observer`.
- [x] 2026-09-08: Read repository harness, portability/security/reliability/quality policy and completed Android/Flutter evidence.
- [x] 2026-09-08: Inspected extension points: app `AcquireContext`/`SaveRun`/`recordRunArtifact`, Android `applicationADB`, CLI `serviceForStore`; retain adapter boundaries and cleanup barriers.
- [x] 2026-09-08: Owned API 35 Emulator spike: platform shell tree dump, standalone UiAutomation tree, Flutter semantic click, focused Unicode insertion and replacement verified. Selected platform companion; details below.
- [x] 2026-09-08: Wrote bilingual observer product contracts before public CLI behavior; `repoctl docs-check` passed after the initial translation.
- [x] 2026-09-08: Wrote bilingual observer design docs and indexes; retained existing adapter and SQL boundaries.
- [x] 2026-09-08: Defined version 1 domain snapshot/node/window types and deterministic compact text rendering in the product contract and `internal/domain/ui.go` / `internal/cli/ui.go`.
- [x] 2026-09-08: Defined `AGENTENV-UI-STALE`, `AGENTENV-UI-AMBIGUOUS` and `AGENTENV-UI-UNAVAILABLE` in the bilingual product contract; backend-status mapping regressions passed in the latest full Linux unit run.
- [x] 2026-09-08: Implement owned-runtime/application selection and identity proof. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Implement semantic snapshot capture and normalization. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Implement snapshot evidence persistence and compact rendering. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Implement screenshot capture and digest evidence. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Implement stale-safe semantic-node tap. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Implement explicit coordinate tap. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Implement Unicode-capable editable-node text replacement. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Implement Back, Home and swipe primitives. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Implement bounded wait/poll behavior. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Implement bounded package-scoped logcat capture. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Integrate operation lock/heartbeat behavior. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Add CLI JSON/text contracts and negative fixtures. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Add tests for multi-runtime/multi-application selection. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Add stale-snapshot and ambiguous-node regressions. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Add evidence redaction/size-bound tests. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Real run 6 proved cross-lease rejection, sibling Count0/UI and backend HTTP after the first destroy, and normal cleanup of both leases.
- [x] 2026-09-08: Run full repository harness and Go race validation. Linux unit verification passed in `repoctl check` at `867a862`; final real/native results are recorded below.
- [x] 2026-09-08: Native CI at `867a862` passed six OS/Go jobs; five cross-builds passed separately. These results do not substitute for real SDK runs.
- [x] 2026-09-08: `TestRealAndroidUIObserver` run 6 passed in 217.34s with the final dialog/Home/viewport fixture assertions.
- [x] 2026-09-08: Completed U1–U23 evidence and retrospective, distinguishing native CI from Linux SDK evidence.
- [x] 2026-09-08: Archived both language plans under `docs/exec-plans/completed/` and updated inbound links.

Implementation checkpoint (2026-09-08): documentation milestone `a717d59` and
implementation `867a862` are committed and pushed on `feat/android-ui-observer`.
The app/domain/Android adapter/companion/CLI implementation, recovery and focused
negative fixtures are complete at the Linux unit-test level. `go run
./tools/repoctl check` passed formatting, all unit tests, vet, documentation,
generated-output and architecture checks. `go run ./tools/repoctl doctor` passed
using Go 1.27.1; `go test -race ./...` passed. `go run ./tools/repoctl test-integration`
passed the existing real Docker/Compose integration suite; that suite is not SDK
evidence. `go test -tags flutterintegration ./internal/cli -run '^$'` previously
passed compilation of the observer fixture. The sixth real two-Emulator `TestRealAndroidUIObserver` attempt passed in 217.34s
on Linux amd64, Go 1.27.1, Flutter 3.47.2, API 35 and the third companion build.
The final fixture checks explicit dialog visibility before/after Back, no visible
application node after Home and scrolling within the current visible viewport.
Both leases cleaned up normally, with the sibling UI and backend HTTP surviving
the first destroy. CI run `34175369767` at `867a862` passed all 12 jobs (six native
OS/Go, five cross-builds and integration). All acceptance requirements have direct
evidence below. The post-archive full `repoctl check` passed formatting, unit tests, vet,
documentation, generated-output and architecture validation with exit code 0.

Additional local evidence (2026-09-08): the implementation owner exercised snapshot,
screenshot, explicit recovery and normal destroy on a single owned Emulator; these
operations succeeded. Reusing an older semantic snapshot after keyboard appearance
was safely refused as stale. This does not satisfy the full two-device observer
acceptance run. The latest harness pass completed unit tests and vet; a docs-check
failure during simultaneous English/Japanese edits was an intermediate freshness
check, not a final validation result. After synchronization, standalone
`go run ./tools/repoctl docs-check` passed on Linux / Go 1.27.1.

A checked item means observed completion, not intention. Include date, command/test identifier, result and relevant environment when checking an item.

## Surprises & Discoveries

- 2026-09-08 fifth real attempt: failed after 177.24s at the swipe assertion. Back correctly restored Count1. The visible ScrollView bounds were `[0,80,320,346]` because the IME resized the viewport, while the fixed screenshot-based swipe started at y=480, outside that viewport. The fixture now derives explicit coordinates from the unique currently visible scrollable node, inset by one eighth, and retains the assertion that scrolling exposes a previously unseen visible label. It also independently verifies the dialog is visible before Back and absent afterward, and that Home leaves no visible application nodes. The sixth attempt subsequently passed with all these fixture changes; its final evidence is recorded above.

- 2026-09-08 fourth real attempt: failed after 144.49s at the swipe assertion. Both pre-swipe and post-swipe accessibility trees were empty because Back had exited the application; the fixture had incorrectly assumed Back would dismiss the IME. Both leases were confirmed released after the attempt. The fixture now explicitly opens a dialog, waits for it, sends Back and waits for Count1 before swiping. This establishes a known Back target without weakening the swipe assertion. The fifth real attempt is recorded separately; final real acceptance was pending at that checkpoint and subsequently passed; see final evidence.
- 2026-09-08 evidence correction: the recorded test durations for attempts 1 and 2 are 104.97s and 127.31s respectively. Earlier 222.41s / 276.86s entries were inaccurate elapsed-time summaries and have been corrected throughout this plan; the failure causes and retained evidence are unchanged.

- 2026-09-08 third real attempt: fixture creation stopped after 342.98s during Flutter dependency resolution, before any lease allocation. A direct HTTPS probe to pub.dev timed out after 10s. Only the owned fixture's pub subprocess was deliberately terminated to finish this failed attempt; no leases had been allocated. No observer acceptance was exercised.
- 2026-09-08: added explicit fixture-only `AGENT_ENV_FLUTTER_OFFLINE_FIXTURE=1`, which appends official `flutter create --offline` and consumes an existing cache, failing if incomplete. Ordinary core build/runtime behavior and every acceptance check are unchanged. The fourth attempt used this option and the third helper build; its outcome is recorded separately.
- 2026-09-08 native evidence: head `867a862` passed all six Windows/macOS/Linux × supported-Go native CI jobs and five cross-build jobs. This is native fake/backend and build evidence, not real Android SDK evidence. The CI integration job also passed: run `34175369767` at `867a862` completed all 12 jobs successfully.

- 2026-09-08 second real two-Emulator attempt: `TestRealAndroidUIObserver` ran 127.31s and launched both applications, then rejected the fresh semantic tap as stale. Retained before/after evidence matched except for Android window IDs changing from 8 to 11 and the derived fingerprints. Window IDs are ephemeral across UiAutomation reconnections, so they cannot be durable semantic identity. The helper now uses semantic window metadata and has been rebuilt (third production build); the real regression retry was pending at that checkpoint and subsequently passed; see final evidence.
- 2026-09-08 follow-up checks: full unit tests and vet passed again. CLI coverage now checks inline log content, rejection of explicitly zero durations, missing lease exit 2 and corrupt registry exit 7. Standalone `go run ./tools/repoctl docs-check` passed after this translation synchronization; Go race and native/real validation were still pending at this checkpoint and subsequently passed; see final evidence.

- 2026-09-08 real integration attempt: `TestRealAndroidUIObserver` failed after 104.97s during Android activity startup (`am start -W` returned `Status: timeout`), before reaching observer assertions. Registry inspection and conservative cleanup confirmed both leases released. No readiness, ownership or timeout check was relaxed. The independent retry is recorded separately; this failed attempt is not observer acceptance evidence.
- 2026-09-08 independent app/recovery review: the final classification `SaveRun` could fail after registering the original result, leaving an unclassified run. Recovery previously rejected only explicitly forbidden classifications, allowing that missing classification to bypass host-process/evidence barriers. It now requires positively durable `termination-unconfirmed` eligibility plus verified original result evidence. `TestUIRecoverRefusesUnpersistedFailureClassification` covers the failed-write path. Focused recovery/fence/backend-diagnostic tests passed on Linux / Go 1.27.1; final acceptance was pending at that checkpoint and subsequently completed; see final evidence.
- 2026-09-08 independent CLI review: raw UI errors previously defaulted to exit 2 even for missing tools and registry failures. Typed error propagation and CLI mapping now distinguish prerequisite 3, invalid options/selection/missing lease/stale/ambiguous 2, and registry/observation 7 without exposing secrets through diagnostic errors.

- 2026-09-08: Baseline `go run ./tools/repoctl check` passed unit tests and vet, then failed AGENTENV-DOC-008 because the supplied Japanese plan lacked translation metadata. Added metadata; `repoctl docs-check` passed. No check was weakened.
- 2026-09-08: Owned API 35 Emulator + temporary Flutter fixture: shell `uiautomator dump` and a self-targeting platform UiAutomation APK both expose the Flutter accessibility surface. Semantic click changed Count 0 to Count 1. Focused ACTION_SET_TEXT set `日本語 🙂 café`, and a fresh node reference replaced it with `置換済み 🚀` (verified via subsequent accessibility text). No target-project instrumentation dependency was added.
- 2026-09-08 failed approach: an unfocused Flutter node returned true for ACTION_SET_TEXT without changing the value. Focusing changes keyboard windows and node ordinals; replaying a prior ordinal also returned success on the wrong node. Production must require advertised action/focus, semantic fingerprint matching and read-back equality. Successful dispatch alone is not acceptance evidence.


- 2026-09-08 independent review: found a concrete adapter/app logcat channel mismatch (`Raw` versus `Binary`), missing backend-status diagnostic mapping, missing post-action fingerprint, and incomplete hierarchy traversal handling. Fixes and regression coverage belong to this implementation; final review/validation was pending at that checkpoint and subsequently completed; see final evidence. This demonstrates why an app fake alone cannot prove the concrete adapter/evidence boundary.
- 2026-09-08 recovery design finding: a local ADB timeout does not itself prove that remote instrumentation has stopped. Retaining only a running barrier without a supported recovery route could strand normal destroy. Add verified helper-only quiescence under a valid fence and explicit recovery of registered helper runs; never replay input or recover arbitrary commands.

Record at least:

- UI Automator/accessibility behavior that differs across Android API levels;
- Flutter semantics that are merged, absent or represented unexpectedly;
- system dialogs/windows that require cross-package observation;
- duplicate nodes that defeat a proposed semantic fingerprint;
- Unicode text-entry limitations;
- helper/instrumentation packaging constraints;
- screenshot encoding differences across hosts;
- logcat filtering behavior that makes package attribution ambiguous;
- cases where a command succeeded but output/evidence persistence is uncertain;
- real Emulator behavior that contradicts fake backend tests;
- any pressure to weaken existing Android identity or cleanup rules.

Failed experiments remain part of this plan when they materially affect the design.

## Decision Log

- 2026-09-08, implementation owner: editable/password node fields remain suppressed and the set-text payload is redacted from its own command evidence. Do not persist that input as a future secret matcher. Later noneditable application echoes or explicitly requested application logcat may contain the value; configured secret redaction still applies. Rationale: preserve direct-command secrecy without claiming retrospective knowledge of deliberately unretained input.

- 2026-09-08, implementation owner: window identity uses type, title, bounds, root package/class and active state; exclude ephemeral Android window ID, traversal ordinal and layer. Retain raw IDs as observation evidence and retain ambiguity rejection. Rationale: identical Flutter UI must remain actionable after a UiAutomation reconnect without treating duplicate semantic windows as unique.
- 2026-09-08, implementation owner: semantic actions carry `ExpectedBackend` from the registered snapshot. Reject backend provenance mismatches before semantic dispatch, so a snapshot from another helper build cannot authorize the current backend. Rationale: fingerprint rules belong to a specific verified helper implementation and must not silently change under an existing reference.

- 2026-09-08, implementation owner: recovery eligibility is a positive durable capability, not the absence of a forbidden classification. Require the recorded `termination-unconfirmed` classification and registered original result evidence before dispatching helper recovery. Rationale: a failed final classification write or crash must not turn host-process/evidence uncertainty into permission to clear the barrier. Unclassified interruptions remain blocked for explicit investigation.

- 2026-09-08, implementation owner: internal helper deadlines may use up to ten additional seconds for verified companion-only quiescence while the original fence remains valid. Caller cancellation or lock loss does not trigger automatic recovery. `ui recover LEASE --run RUN` acquires a new fence, validates the original registered runtime/serial and helper identity, proves process absence, and retains failed/uncertain recovery evidence before releasing only that run's barrier. Rationale: make incomplete remote operations recoverable without weakening ownership, hiding uncertainty, replaying input or touching native/test processes.

- 2026-09-08, implementation owner: choose an agent-env-owned self-targeting APK using stable platform UiAutomation (API >=26), with no AndroidX dependency. Shell text input cannot provide Unicode replacement. An explicit native Go build produces locally distributed APK+version/source/APK digests, keeping ordinary Go builds SDK/JDK independent. Reject a conflicting helper already on the template; never overwrite it.
- 2026-09-08, implementation owner: default application scope is its package; runtime-only/all-windows explicitly includes system UI. Refuse quarantined/expired/cleanup leases; allow read diagnostics on ready/degraded owned Android; mutations require ready. Fingerprints include semantic ancestry/window/bounds/state, exclude ordinals and editable values; truncated trees cannot authorize action. Fully redact entered/editable/password values; report readback only as equality. Limits/API floor/logcat PID attribution are recorded in the product/design contracts. No manifest or adapter-boundary change is needed.

- Decision: `android-ui-observer` is an observation/interaction capability over an existing owned Android runtime, not a new runtime type.
  Rationale: Device lifecycle and application lifecycle already have independent ownership models; UI observation must consume them without creating a third owner.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Use Android accessibility/UI Automator-visible state as the semantic black-box surface; screenshots are complementary.
  Rationale: This works across Flutter semantics, native Android UI and system dialogs without target-app hooks and mirrors the semantic-tree-plus-screenshot model used by browser agents.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Snapshot node references are ephemeral and snapshot-scoped.
  Rationale: Accessibility nodes and bounds change as the UI changes. Treating them as durable selectors would make agent actions silently unsafe.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Semantic-node actions must reject stale or ambiguous snapshots instead of falling back to stored coordinates.
  Rationale: The agent must know when its observation is obsolete rather than clicking a different control.
  Date/Author: 2026-09-08 / maintainers.

- Decision: The target application must not add instrumentation dependencies for `agent-env ui`.
  Rationale: The observer is intended as reusable black-box infrastructure for arbitrary repositories and system UI.
  Date/Author: 2026-09-08 / maintainers.

- Decision: UI evidence is retained under the lease artifact system and uses existing redaction/digest boundaries where applicable.
  Rationale: UI/debug evidence must participate in the existing audit/recovery model without creating an unmanaged artifact store.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Durable docs and this living ExecPlan are maintained in English and Japanese.
  Rationale: This is repository policy.
  Date/Author: 2026-09-08 / maintainers.

Final verification checkpoint (2026-09-08): targeted tagged CLI `TestUI` and
`repoctl doctor` passed. Final `go test -race ./...` passed (app 32.784s).
Final fixture changes were committed/pushed as `2001eac`. CI `34176592070` on that
head has ten successful jobs; integration and macOS Go 1.27 are still pending. The unchanged production code
already has full native/integration CI evidence at `867a862`. The post-archive
full `repoctl check` passed all formatting/unit/vet/docs/generated/architecture
checks with exit code 0.

## Outcomes & Retrospective

Completed on 2026-09-08. The observer delivers versioned, bounded accessibility
snapshots with ephemeral node references, validated PNGs, stale-safe semantic taps,
focused Unicode text replacement with read-back, explicit coordinate taps,
Back/Home/swipe, bounded waits and attributed bounded logcat. A separately built,
self-targeting platform UiAutomation companion avoids target-app instrumentation
and AndroidX dependencies; verified version/source/APK provenance binds snapshots
to their backend. App orchestration, Android resource management and Flutter builds
remain separate. Existing registry runs/artifacts provide intent, conservative
cleanup barriers and narrow evidence-backed recovery without schema changes.

Actual Flutter tests established that platform action success does not guarantee
text replacement without focus/read-back, and Android window IDs are ephemeral
across instrumentation reconnects. Semantic window identity preserves valid refs
while unique matching rejects stale/ambiguous input. IME-resized viewports and Back
navigation require observable fixture state, not assumptions based on screen size
or keyboard presence. The final fixture strengthens these assertions. Five failed
attempts remain recorded below; tests and ownership checks were not weakened.

Real run 6 passed in 217.34s on Linux amd64 / Go 1.27.1 / Flutter 3.47.2 / API 35
with two concurrent leases, all requested observer actions, Unicode/privacy/PNG
checks, sibling isolation and normal cleanup. Full Linux harness, race and real
Docker checks passed, and CI `34175369767` at `867a862` passed all 12 jobs. Native
Windows/macOS fake/backend execution and cross-builds do not claim real SDK
execution there. API 26 is the supported floor, but real SDK evidence is API 35;
other API images remain unverified. Richer gestures, OCR, visual regression,
physical devices, remote hosts and browser/CDP symmetry remain future work.

Editable/password fields and direct set-text command evidence are suppressed;
future arbitrary app echoes/logcat and PNG pixels can still contain sensitive
content. Unclassified crash/host-process/evidence uncertainty remains a deliberate
cleanup barrier. The final post-archive harness rerun passed all checks with exit code 0;
no feature acceptance remains pending.

## Context and Orientation

Read before implementation:

- `AGENTS.md`
- `ARCHITECTURE.md` and `ARCHITECTURE.ja.md`
- `docs/PLANS.md` and `docs/PLANS.ja.md`
- `docs/PORTABILITY.md` / `.ja.md`
- `docs/RELIABILITY.md` / `.ja.md`
- `docs/SECURITY.md` / `.ja.md`
- `docs/QUALITY.md` / `.ja.md`
- `docs/roadmap.md` / `.ja.md`
- Android Emulator product/design docs and completed ExecPlans in both languages
- PR #4's final Flutter Android product/design docs in both languages
- PR #4's final completed Flutter Android ExecPlans in both languages

PR #4 establishes that Flutter applications are separate from their `android-emulator` runtime, records the installed APK digest/package/activity, uses exact owned serials, and treats UI automation as follow-up work.

Current architectural expectations after PR #4:

- config strictly decodes manifests;
- stack resolves deterministic component closure;
- app owns orchestration, fencing, readiness, compensation and evidence policy;
- Android adapter owns concrete Emulator/ADB effects but not Flutter builds;
- Flutter application lifecycle owns build/install/launch/reverse identity;
- SQLite persists the lease snapshot plus events/runs/artifacts;
- `execx` owns portable process boundaries;
- `evidence` owns redaction and digest helpers;
- CLI formatting remains separate from lifecycle/observer policy.

The observer should preferably require no new target-manifest section. It operates on selected lease resources. If a small optional policy is needed later, justify it explicitly rather than turning transient UI state into repository configuration.

## Plan of Work

### Milestone 1 — Backend spike and product contract

Using a real lease-owned Emulator from the merged PR #4 implementation, compare the smallest viable black-box mechanisms for:

- enumerating visible accessibility windows/nodes;
- preserving text/content descriptions/bounds/state;
- clicking a semantic element;
- replacing editable text including non-ASCII input;
- capturing screenshots;
- distinguishing structured command failure;
- operating across target app and Android system dialogs.

Do not begin with a target-app test dependency.

Record the selected backend in the Decision Log. If a companion APK is required, define its source location, build/release ownership, runtime discovery/extraction, version/digest evidence, installation lifecycle and compatibility policy.

Write:

    docs/product-specs/android-ui-observer.md
    docs/product-specs/android-ui-observer.ja.md
    docs/design-docs/android-ui-observer.md
    docs/design-docs/android-ui-observer.ja.md

Update both language indexes and architecture docs if a new adapter boundary is introduced.

### Milestone 2 — Snapshot schema and capture

Define a versioned normalized snapshot schema. Suggested shape:

```json
{
  "schema_version": 1,
  "snapshot_id": "ui-01K...",
  "lease_id": "...",
  "runtime": "phone",
  "serial": "emulator-5554",
  "captured_at": "...",
  "display": {"width": 1080, "height": 2400},
  "windows": [
    {
      "id": "w1",
      "package": "com.example.app",
      "active": true,
      "nodes": [
        {
          "ref": "n4",
          "class": "android.widget.Button",
          "resource_id": "",
          "text": "Login",
          "content_description": "",
          "bounds": [72, 528, 1008, 640],
          "clickable": true,
          "enabled": true
        }
      ]
    }
  ]
}
```

Requirements:

- snapshot ID is unique in the local artifact store;
- node refs are unique only within one snapshot;
- raw backend output is retained separately when useful;
- compact text output is deterministic;
- node/byte limits have an explicit truncation marker;
- semantic snapshots remain lightweight enough for frequent agent use;
- screenshot is not implicitly captured for every snapshot.

### Milestone 3 — Screenshot evidence

Implement PNG capture from the exact owned serial and record lease/runtime/serial identity, timestamp, dimensions and SHA-256. Reject incomplete image output and publish evidence atomically.

### Milestone 4 — Stale-safe actions

For a semantic node action:

1. acquire the lease operation fence;
2. prove the target Android runtime still matches durable ownership;
3. load the referenced snapshot evidence;
4. validate snapshot lease/runtime identity;
5. re-observe current accessibility state;
6. resolve the old node fingerprint uniquely;
7. on stale/ambiguous state, return a stable error and perform no input;
8. otherwise perform the semantic action;
9. record action evidence and resulting state fingerprint.

Start with semantic `tap` and `set-text`.

`set-text` must support Unicode-capable replacement. Do not document `adb shell input text` as equivalent if the selected mechanism cannot faithfully represent the supplied string.

Coordinate tap is explicit and distinct from semantic tap.

### Milestone 5 — Navigation, swipe and waits

Add Back, Home, one-pointer swipe and bounded wait for a semantic predicate or stable-state condition. Waits always have explicit deadlines. Do not add arbitrary shell execution under `agent-env ui`.

### Milestone 6 — Logcat observation

Add bounded diagnostics scoped to a selected application/package where possible.

Requirements:

- exact owned serial;
- explicit since/duration/line/byte bound;
- package/PID attribution recorded;
- broader device scope labeled honestly when package scoping cannot be proven;
- persisted/output text goes through secret redaction;
- no unbounded background process in MVP;
- no global ADB-server lifecycle effect.

Logcat content alone does not change lease readiness.

### Milestone 7 — State policy and concurrency

Read-only snapshot/screenshot/logcat should be available for a confirmed live owned Android resource even when the lease is DEGRADED, because this is useful for diagnosis.

Define and test the exact rule for QUARANTINED leases before implementation. No read or write command may bypass ambiguous device identity.

Mutating UI actions require active, unexpired lease; confirmed owned Android identity; operation fence; and no cleanup in progress.

Prove:

- two concurrent leases with the same package/device-local ports remain independently observable;
- a snapshot from lease A is rejected when supplied to lease B;
- destroying lease A cannot race an action on A past the operation fence;
- no command on A selects or affects B's serial;
- an external/user Emulator is never selected implicitly.

### Milestone 8 — Agent-facing ergonomics

Provide concise text output suitable for coding agents while keeping JSON for structured clients.

Target output:

    $ agent-env ui snapshot lease-a --application mobile-app

    Snapshot ui-01K...
    Runtime phone (emulator-5554)
    Window com.example.app

    [n1] text    "Sign in"
    [n2] textbox "Email"       enabled
    [n3] textbox "Password"    enabled password
    [n4] button  "Login"       enabled

A normal UI-debug loop must not require ADB serials, AVD paths, helper package names, instrumentation runner names or artifact directories.

If exactly one eligible target exists, omission may resolve deterministically. Multiple eligible targets require explicit application/runtime selection rather than guessing.

### Milestone 9 — Real integration evidence

Add a real opt-in fixture on top of the merged Flutter integration support. The minimal Flutter UI should contain labeled text, ASCII and Unicode editable fields, a button causing a deterministic state change, and a scrollable element.

Prove with a real Emulator:

1. create reaches READY;
2. snapshot finds expected semantic nodes;
3. screenshot is a valid PNG;
4. set-text handles ASCII and non-ASCII test data;
5. stale snapshot action is rejected after a deliberate UI change;
6. fresh snapshot + tap changes application state;
7. Back/Home behave as documented;
8. bounded logcat evidence is captured;
9. sibling lease remains unaffected;
10. normal destroy cleans both leases.

Run fake/native portability tests on Windows/macOS/Linux separately. Do not call cross-builds or fake adapters real Android UI validation.

## Concrete Steps

1. Wait for PR #4 to merge.
2. `git switch master && git pull --ff-only`.
3. Record the exact starting revision above.
4. Create `feat/android-ui-observer`.
5. Add this English plan and its Japanese sibling under `docs/exec-plans/active/`.
6. Run the repository harness before implementation and record the baseline.
7. Inspect the merged Flutter/application domain and concrete Android adapter.
8. Execute the UI backend spike on a real lease-owned Emulator.
9. Record the backend decision before creating public CLI contracts.
10. Add bilingual product/design docs and indexes.
11. Add app/domain observer interfaces before concrete adapter wiring.
12. Implement snapshot capture/normalization/evidence.
13. Implement screenshot evidence.
14. Implement stale-safe semantic tap and Unicode text replacement.
15. Implement coordinate tap, navigation, swipe and bounded wait.
16. Implement bounded package-scoped logcat.
17. Wire CLI text/JSON output without moving policy into CLI.
18. Add fake backend, negative, fencing and cross-lease isolation tests.
19. Add the real Flutter/Emulator observer integration fixture.
20. Repeatedly run focused tests, `repoctl check`, race tests and native CI.
21. Record acceptance evidence honestly by platform/evidence type.
22. Fill Outcomes & Retrospective in both plan languages.
23. Move both plans to `docs/exec-plans/completed/` and update links.

## Validation and Acceptance

| ID | Required behavior | Evidence |
| --- | --- | --- |
| U1 | Existing Compose, Android and Flutter manifests/leases work unchanged without using UI commands. | 2026-09-08: `repoctl check` at `867a862` passed existing unit regressions; `repoctl test-integration` passed real Docker/Compose. Both real Flutter leases reached application launch in attempt 2; observer end-to-end passed in real run 6. |
| U2 | UI commands resolve only a selected confirmed lease-owned Android runtime; no first-device fallback exists. | 2026-09-08 Linux PASS: `TestUIRejectsInvalidStateAndSelectionBeforeDevice`, `TestUIDegradedDiagnosticsAndApplicationScope`, `TestUIOwnershipFailureCannotDispatchInput` in full `repoctl check`. |
| U3 | A semantic snapshot returns deterministic structured JSON and compact text with lease/runtime/snapshot identity. | 2026-09-08 Linux PASS: `TestUISnapshotAndSemanticScope`, protocol tests and actual single-Emulator snapshot smoke; compact rendering implemented. The full observer workflow passed in real run 6. |
| U4 | Raw/normalized snapshot evidence is bounded, atomically published and auditable. | 2026-09-08 Linux PASS: snapshot tampering/symlink, protocol bounds, evidence failure and atomic-write regressions in `repoctl check`; actual single-Emulator snapshot artifacts retained. |
| U5 | Flutter semantic labels/text/editable controls appear through the documented Android accessibility surface in real integration. | 2026-09-08: owned API 35 spike exposed Flutter labels, text and editable controls; attempt 2 captured rich-fixture Count0/Increment0 nodes before failing on ephemeral window-ID matching. |
| U6 | Screenshot capture produces a valid PNG artifact with digest and exact device identity. | 2026-09-08 Linux PASS: `TestUIPNGValidation`, `TestUIInvalidCompletedCaptureFinalizesFailed`; actual single-owned-Emulator PNG/artifact smoke succeeded. |
| U7 | Semantic tap using a fresh snapshot acts on the intended node. | 2026-09-08 real run 6 PASS (217.34s): fresh semantic tap changed Count0 to Count1; stale reuse caused no extra increment. |
| U8 | A stale or ambiguous node reference fails before input and never falls back to an old coordinate. | 2026-09-08 Linux PASS: `TestUIRejectsSnapshotTamperingAndAmbiguity`, `TestUIBackendRefusalsRetainStableDiagnostics`, backend-provenance regression; actual older-snapshot smoke refused stale input. |
| U9 | Explicit coordinate tap is available and recorded distinctly from semantic tap. | 2026-09-08 real run 6 PASS: explicit coordinate tap and bounded-viewport swipe exercised; coordinate-range negative tests also pass. |
| U10 | Editable-node text replacement handles defined Unicode test cases without shell-escaping corruption. | 2026-09-08 real run 6 PASS: ASCII and Japanese/emoji/Greek replacements confirmed read-back equality; editable/password evidence stayed redacted. |
| U11 | Back, Home and swipe affect only the selected owned serial. | 2026-09-08 real run 6 PASS: dialog visible before Back and absent afterward with Count1 restored; Home left no visible app nodes; swipe exposed a previously unseen visible label. |
| U12 | Wait/poll operations are bounded and report timeout without hidden continued effects. | 2026-09-08 real run 6 PASS: bounded missing-predicate wait failed as expected; both normal cleanups passed. Unit timeout/recovery/fence regressions also pass. |
| U13 | Bounded logcat records package/PID scoping or labels broader scope honestly and redacts configured secrets. | 2026-09-08 Linux PASS: `TestUILogPIDAttributionAndBounds`, `TestUILogcatReturnsBoundedRedactedInlineEvidence`, nonfinite-timestamp and CLI inline-log tests. Numeric-PID history limits documented. |
| U14 | UI operations participate in the lease operation fence; destroy cannot race past an in-flight mutating action. | 2026-09-08 Linux PASS: `TestUIOperationFencePreventsDestroyRace` plus recovery/host-unconfirmed barrier tests; `go test -race ./...` passed. |
| U15 | A snapshot/node reference from one lease cannot be used against another lease. | 2026-09-08 Linux PASS: `TestUISnapshotAndSemanticScope` cross-lease refusal; real attempt 2 rejected cross-lease reference before the later fresh-tap failure. |
| U16 | Two concurrent mobile leases remain independently observable and operable; destroying one leaves the sibling usable. | 2026-09-08 real run 6 PASS: two isolated leases; after first destroy sibling snapshot retained Count0 and backend HTTP stayed alive; both released normally. |
| U17 | Ambiguous Android identity never permits observation/action through force or fallback selection. | 2026-09-08 Linux PASS: `TestUIOwnershipFailureCannotDispatchInput`, state/selection tests and existing Android ownership suite; no force or serial override exposed. |
| U18 | The target Flutter/application repository requires no UI-test dependency or source modification for observer use. | 2026-09-08: standalone companion build and API 35 Flutter spike required no target-app instrumentation dependency. Real fixture uses standard Flutter widgets; no observer manifest section exists. |
| U19 | If a companion helper is used, its artifact/version/digest and lifecycle are recorded and target-project instrumentation is not required. | 2026-09-08 Linux PASS: helper provenance/source/version/digest and backend-change tests. Three explicit companion builds and single-Emulator install/observation/recovery smoke recorded. |
| U20 | Observer docs and ExecPlans exist in English and Japanese and pass translation checks. | 2026-09-08 PASS: bilingual product/design/index/README/CLI/architecture/plan updates; full `repoctl check` includes passing docs-check. Living-plan update rechecked separately. |
| U21 | Architecture/docs/schema checks and the full repository harness pass on final implementation. | 2026-09-08 PASS at `867a862`: `repoctl doctor` and full `repoctl check` including formatting, unit tests, vet, docs, generated and architecture validation; real Docker integration also passed. |
| U22 | Go race validation and native Windows/macOS/Linux fake/backend tests pass. | 2026-09-08 PASS: Linux `go test -race ./...`; all six Windows/macOS/Linux × supported-Go native CI jobs at `867a862`. Five cross-builds passed separately and are not SDK evidence. |
| U23 | At least one real Flutter + Emulator observer run exercises snapshot, screenshot, Unicode input, stale rejection, action, logcat and cleanup, or a concrete external infrastructure blocker is recorded without substituting fake evidence. | 2026-09-08 real run 6 PASS, 217.34s, Linux amd64 / Go 1.27.1 / Flutter 3.47.2 / API 35 / companion build 3. All requested real CLI observer checks and both normal cleanups passed; attempts 1–5 remain recorded failures. |

Every accepted item requires direct recorded evidence. Test names alone are not evidence until a successful run is recorded.

## Idempotence and Recovery

Read-only snapshot/screenshot/logcat operations never change desired lease state. They may update heartbeat and append evidence/events under the operation fence.

A failed observation may leave an incomplete temporary artifact, but it must not publish a terminal successful artifact row. Recovery may remove only clearly owned incomplete files under the lease artifact directory.

A mutating UI action is not generally idempotent. Therefore:

- persist action intent and target identity before performing input where the existing event/run model can do so safely;
- record completion separately;
- never retry a tap automatically after an uncertain result;
- report uncertainty instead of replaying;
- require a fresh snapshot before a caller decides whether to retry.

Snapshot-scoped node refs are never reused after a stale failure.

If companion-helper installation is uncertain, do not uninstall a package from an unproven device. The private AVD normally removes helper/application state when confirmed Emulator destruction succeeds.

No UI recovery path may:

- run `adb kill-server`;
- select a device by list order;
- delete or modify user AVD templates;
- change another lease's package or reverse mappings;
- clear global logcat as a cleanup shortcut;
- weaken existing Android quarantine barriers;
- use `--force` to bypass device ownership proof.

## Artifacts and Notes

Final real evidence (2026-09-08): `go test -tags flutterintegration ./internal/cli -run
'^TestRealAndroidUIObserver$' -count=1 -v -timeout=35m` PASS, 217.34s.
Lease IDs: `01M1Z9KEH18ZTC8AHQSQ06MHT9` and `01M1Z9KEJE3EZDVT8Y5RBXT9MS`;
fixture source: `be35f3179692baf0c3c160715d7c6f5987889de9`. Linux amd64,
Go 1.27.1, Flutter 3.47.2, Android API 35, third verified companion build, explicit
cached fixture creation. The final test exercises actual CLI actions, Unicode
read-back, stale/cross-lease refusal, privacy and PNG/digests, viewport swipe,
Back/Home, current-PID logs and bounded wait. After the first destroy the sibling
still showed Count0 and served fresh backend HTTP; both normal cleanups passed.


Suggested per-operation artifact grouping:

    <agent-env-home>/leases/<lease-id>/artifacts/<ui-run-id>/
      run.json
      snapshot.json
      snapshot.txt
      raw-hierarchy.xml        # when backend exposes it
      screenshot.png           # screenshot operations only
      logcat.txt               # logcat operations only

The exact layout may follow existing artifact conventions instead of introducing a new hierarchy.

Record at minimum:

- UI run/observation ID;
- lease ID;
- selected component/application when used;
- Android runtime name;
- serial;
- package/window identity where applicable;
- backend/helper name/version/digest;
- operation;
- referenced snapshot ID/node ref for semantic actions;
- node fingerprint used for stale validation;
- coordinates for coordinate actions;
- capture/start/finish timestamps;
- output/truncation status;
- file SHA-256 for persisted artifacts;
- action result and uncertainty state.

Do not persist passwords or text-entry payloads in clear text by default. Decide and document whether payloads are fully redacted, length-only or explicitly opt-in retained.

Screenshots and raw UI snapshots may themselves contain secrets. Text redaction cannot remove secrets rendered into PNG pixels. Document this privacy property explicitly.

## Interfaces and Dependencies

Expected conceptual app boundary:

```go
type AndroidUIProvider interface {
    Capabilities(ctx context.Context, target AndroidUITarget) (...)
    Snapshot(ctx context.Context, target AndroidUITarget, opts SnapshotOptions) (...)
    Screenshot(ctx context.Context, target AndroidUITarget, opts ScreenshotOptions) (...)
    Act(ctx context.Context, target AndroidUITarget, action UIAction) (...)
    Logcat(ctx context.Context, target AndroidUITarget, opts LogcatOptions) (...)
}
```

Exact names and types are not fixed. Keep:

- domain data independent of CLI;
- app responsible for lease/runtime selection, operation fencing, stale policy, evidence policy and cross-provider orchestration;
- concrete Android/UI backend responsible for device commands and parsing;
- store responsible for persistence, not UI policy;
- CLI responsible for parsing and rendering.

Potential dependencies:

- existing Android SDK `adb`;
- Android accessibility/UI Automator surface;
- optional `agent-env` companion instrumentation artifact if selected;
- existing Emulator and Flutter runtime support from previous plans.

The target application must not need AndroidX UI Automator or an instrumentation runner solely for `agent-env ui`.

No Bash, POSIX shell, PowerShell, Make, implicit first-device selection, CGO or fixed host port may become a core requirement.

Milestone 1 decisions (settled; validation evidence is in the acceptance table):

1. Backend: self-targeting agent-env companion for semantic snapshots/actions; exact-owned-serial platform commands for PNG, navigation, coordinates and logcat.
2. API: stable platform `UiAutomation`/`AccessibilityNodeInfo`; no AndroidX or prerelease dependency. API 35 has real spike evidence; broader API validation is not claimed.
3. Packaging: embedded companion source, explicit native Go SDK/JDK build into a new directory, local APK/metadata distribution via `AGENT_ENV_UI_HELPER`. Version/source/APK digests are verified; ordinary Go builds remain SDK/JDK independent.
4. Fingerprint: semantic window type/title/bounds/root package/class/active state plus semantic ancestry, node class/package/resource ID/noneditable label/text/bounds/action/state flags. Exclude ephemeral window IDs/ordinals/layers and editable/password values. Require unique matches and matching backend provenance.
5. Scope: application snapshots default to the recorded package; runtime-only and explicit `--all-windows` include accessible system windows.
6. Privacy: fully redact entered/editable/password text and associated editable description/hint. Redact configured secrets in ordinary labels/logs. PNG pixels cannot be text-redacted; arbitrary displayed text may still be sensitive.
7. Logs: `ui logcat`, current numeric PID only, device-time lower bound, no silent device-wide fallback, no global clearing. Report PID reuse/history limitations and bounded inline/artifact output.
8. State: read diagnostics allow active unexpired ready/degraded leases with proven owned live Android. Mutations require ready. Quarantined/expired/released/cleanup leases refuse normal UI commands; narrowly eligible explicit helper recovery uses its separately documented fence/evidence policy.
9. Limits: 1000 nodes, depth 64, 4096 characters per snapshot field, 1 MiB backend response, explicit truncation. Partial trees cannot authorize semantic mutation; PNG and log bounds are defined in the product contract.
10. Compatibility: API 26 floor; native argv/filesystem support for Windows/macOS/Linux. Only recorded actual SDK/native executions count as platform evidence.

The Decision Log and bilingual product/design contracts record the rationale and recovery details.
