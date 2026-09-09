---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Historical mobile correctness review corpus

[日本語](history-mobile.ja.md) · [Audit index](index.md) · [Execution authority](../../exec-plans/active/repository-correctness-audit.md)

Frozen target: `031869c8b9073b8e23bc17fbc55243666a52f557`. This Phase-A report changes no product/test files. It inventories material recorded findings, including implementation reviews and native test-oracle failures. Environment-only missing prerequisites, transient format/hash checks, and unproven historical scheduling causes are recorded separately rather than counted as product defects.

## Phase C resolution update

The descriptions below preserve the frozen Phase A evidence and original coverage gaps.
Phase B accepted all eight mobile findings. Their current disposition is **ACCEPT**;
the earlier “untriaged” and “no repair” statements describe Phase A only.
The candidate implements the following controls and permanent regressions:

| Finding | Delivered control | Regression evidence |
| --- | --- | --- |
| AUDIT-BOUNDARY-001 | Request 2001 device records, filter the time window, keep the newest 2000 with explicit actual omission | `TestUILogExactTailUsesOverflowProof`: 1999/2000/2001 and an out-of-window probe |
| AUDIT-REDACTION-001 | Recheck fields and actual escaped snapshot/result bytes after redaction; preserve node order and mark real omissions | `TestUIAuditUIRedactionBounds`, `TestUIFieldBoundaryAndSerializedObservationBoundary`: registered artifacts, retained identities, 4096 and 1 MiB minus/exact/plus boundaries |
| AUDIT-UI-001 | Distinguish automatic editable suppression from secret matches | `TestUIAuditEditableSnapshotRemainsActionable`: snapshot-to-set-text, callback dispatch/readback and two completed durable runs; existing error-redaction test now requires its callback |
| AUDIT-IDENTITY-001 | Assign exact verified helper backend; recover using recorded provenance independently of current host files | `TestUIAuditHelperBackendCarriesVerifiedDigest`: successful same-build input; `TestUIRecoveryUsesRecordedHelperWithAbsentOrReplacedHostFiles`: actual verified helper stop without installation |
| AUDIT-REDACTION-003 | Clear nested node hashes for window secrets; omit opaque after-action hashes whenever secret matchers exist | `TestUIAuditWindowSecretClearsDerivedNodeHashes`, `TestUIOpaqueAfterFingerprintIsNotPublishedWithConfiguredSecrets`: returned and persisted after-action evidence |
| AUDIT-PREREQUISITE-001 | Refuse absent/blank helper configuration before path normalization | `TestUIAuditUnsetHelperRefusesCurrentDirectory`: populated current directory cannot supply implicit opt-in |
| AUDIT-BOUNDARY-002 | Reject fractional lookback before provider execution and durable run creation | `TestUIAuditFractionalLogLookback` |
| AUDIT-LIFECYCLE-001 | Preserve certainty on typed preflight failure; retain uncertainty after actual dispatch failure | `TestUINativePreflightThroughAppDoesNotBlockCleanup`: failed durable run then successful release; `TestUINativeDispatchedFailureRemainsUnconfirmed`: Home/Back/tap/swipe |

Local validation: `go test -race ./internal/app ./internal/runtime/android ./internal/runtime/android/uihelper -run 'TestUI|TestVerify|TestLoad' -count=1` passed twice after the boundary/recovery fixtures were corrected (6.122s/1.353s/1.012s and 5.675s/1.340s/1.010s).
The first fixture revision incorrectly omitted the nine-byte JSON `log` property overhead;
the recovery fixture omitted the SDK response. Both failed honestly and were repaired
without relaxing production checks. An initially oversized envelope must omit at least
one node even if changing truncation flags alone would save two bytes.
An independent reviewer reproduced an additional opaque after-tree hash leak; the
conservative omission control closed that reproduction. The aggregate ExecPlan owns
native candidate and full harness evidence; these local results do not prove native
Windows/macOS behavior or every historical mutation replay.

## Sources and interpretation

- E: [Android feature](../../exec-plans/completed/android-emulator-lease.md), Surprises & Discoveries. No individual thread IDs recorded.
- R1: [Android review](../../exec-plans/completed/android-emulator-review.md), four PR2 findings. Recorded comment IDs: `3951260551`, `3951260553`, `3951260556`, `3951260560`.
- R2: [Android review2](../../exec-plans/completed/android-emulator-review-2.md), seven PR2 findings plus independently discovered CLI preflight and Linux procfs defects. Recorded comment IDs: `3952452797`, `3952452807`, `3952452813`, `3952452817`, `3952452823`, `3952452826`, `3952452828`.
- F: [Flutter feature](../../exec-plans/completed/flutter-android-runtime.md), independent review and real-provider findings.
- FR: [Flutter review](../../exec-plans/completed/flutter-android-review.md), all eight PR4 findings. No individual thread IDs recorded.
- U: [Android UI feature](../../exec-plans/completed/android-ui-observer.md), implementation review and native fixture findings. Later PR5 findings absent from the plan were fetched from the paginated GitHub comments API and inventoried below.

For PR2, the source records comment IDs as an unordered group without a per-finding mapping; this report does not invent one. Severity labels are absent from the plan-only sources; original PR5 P1/P2 labels are preserved below. Each row describes the actual impact instead of attributing a retrospective severity to the reviewer. All listed invariants remain applicable (`yes`), including fixture invariants; original defects are historical, not automatically new audit findings.

`Earliest/detected/profile` uses S0–S9 from the ExecPlan. Discovery stage follows the source (native CI is S5, independent review is S8); earliest prevention and escape profiles are this audit's evidence-based classification, not a claim about undocumented reviewer intent. E/M05 is earliest S3 because a real launcher-child fixture was possible before native CI; OS-specific Job behavior needs S5.

Regression names are the tests now covering the original repair (the historical plan may name only a category). Test validity below is source-level assertion inspection and current Linux execution, **not mutation-replay proof**. A future mutation campaign is still needed before claiming each original defect was reintroduced and caught. Codes:

| Code | Current test proves invariant? | Public/full entry point? |
| --- | --- | --- |
| A | yes at app boundary with injected providers and durable state assertions | yes: Service use case; concrete SDK not exercised |
| C | yes at CLI parsing/output boundary | yes: CLI command or its explicit host-diagnostic entry |
| F | yes for factory ordering | no: factory + Service, not Cobra command dispatch |
| D | yes at concrete adapter/native primitive boundary | no: app/CLI composition not exercised by this regression |
| W | yes by source inspection; Windows-only execution not rerun locally | no: native primitive, not complete app |
| N | yes by native test source and recorded historical pass; not rerun in this subtask | yes: actual real-provider entry path |
| S | yes separately for each side; composition only in historical native fixture | no: the pair alone cannot prove their interoperation |
| G | no direct producer regression located for the exact recorded omission | no; named test only covers consumer/adjacent behavior |
| H | yes for the fault injector | no: test infrastructure, not a product entry point |
| DOC | no automated semantic truth proof; docs-check checks structure/hash | no; human paired meaning review remains required |

Guardrail status is `existing` for scoped regressions above; broader prevention in the profile table is `deferred` to audit disposition. G rows explicitly have a missing direct guardrail. No preventive product control was added in Phase A.

## Escape profiles and earlier opportunities

Each row inherits its profile's opportunities, explanation for earlier escape, missing guardrail, proposed control, and recurrence search. Before the earliest listed stage, the source does not establish enough knowledge to assert a preventable miss. S7 could have requested the missing negative/composed/native fixture, but a green suite without that oracle offered no direct detection; no personal review failure is inferred. Where S8 discovered the problem it was a working detection layer, not a failed one.

| Profile | Opportunities and escape reasons | Missing guardrail / preventive control / expected future stage | Same-pattern candidates inspected or queued |
| --- | --- | --- | --- |
| I | Independent invalid-input/identity mutation existed at S2/S3; positive fixtures matched valid values. `NEGATIVE_FIXTURE_GAP`, sometimes `ORACLE_COUPLING`. | Requirement-specific invalid identity/path/environment fixtures; assert zero effects, S2/S3. | Android marker/ADB paths; application reverse; process receipts; browser targets. |
| C | Public entry points already existed at S3/S4, but helper/one-provider/JSON-only tests omitted the next dependency or optional branch. `HELPER_ONLY`, `COMPOSITION_GAP`. | Matrix of CLI/factory/app/adapter and absent/present optional features with observable no-effect assertions, S3/S4. | Inventory vs Doctor frontend requirements; runtime-specific preview; CLI store factories; Raw/Binary evidence. |
| P | Failure/cancel/restart windows were expressible with injected stores/providers before review. Success-only or initial-failure tests omitted later recovery. `FAILURE_INJECTION_GAP`, `COMPOSITION_GAP`, sometimes `INVARIANT_GAP`. | Durable-intent/effect/finalization matrix and repeat Destroy/Reconcile after store recovery; verify callback actually ran, S2/S4. | Application build guards, UI recovery, process start receipts, browser mutation barriers. |
| N | Real SDK/OS lifecycle semantics were not represented by fake identity or tool output; cross-build cannot settle them. `NATIVE_EVIDENCE_GAP`, `CONCURRENCY_GAP`. | Native lifecycle, parent-exits-child-lives, two simultaneously live leases and sibling cleanup, S5 (M05 S3 real process fixture). | Windows Job completion, Linux procfs races, private Android helper discovery, shared ADB containment. |
| T | Fixture assumptions about lock acquisition or visible device state were unverified; a failure could precede intended injection/action. `ORACLE_COUPLING`, `FAILURE_INJECTION_GAP`. | Assert injection row count/error and visible preconditions before action; native viewport-derived gestures, S3/S5. | Lock-loss injectors, browser mutation callbacks, UI Back/swipe fixture. |
| B | Producer traversal omissions were not independently exercised; consumers saw synthetic complete/truncated flags. `BOUNDARY_GAP`, `HELPER_ONLY`. | Producer limit−1/limit/limit+1 and missing-child fixtures; observe actual omission, S2. | Android UI helper traversal; Android logcat cap; browser AX/DOM caps. |
| D | Docs shape/hash checks cannot determine whether roadmap prose matches product behavior. `REVIEW_CHECKLIST_GAP`. | Final implementation-to-contract/README/roadmap semantic review, S7; avoid claiming hash equality proves meaning. | Product/design feature status, CLI tables, optional prerequisites. |

## Historical findings and current coverage

In the table, paths without a repository prefix are under `internal/`; production symbols identify the current owning entry/boundary. References to helper tests do not claim a full composed regression.

| ID / source | Original invariant, defect and impact | Earliest/detected/profile | Current regression location | Current production location | Coverage |
| --- | --- | --- | --- | --- | --- |
| M01 / E | Cancellation after console identity must prevent kill/deletion; late cancellation could still mutate. | S2/S8/P | `runtime/android/android_test.go:TestCancellationDuringIdentityHandshakeCannotKill` | `runtime/android/lifecycle.go:Destroy` | D |
| M02 / E | ADB must use the owned local server; inherited routing could inspect a remote device with the same serial. | S2/S8/I | `runtime/android/android_test.go:TestReadinessPinsLocalADBServer` | `runtime/android/adb.go:adbRoutingEnvironment` | D |
| M03 / E | Adapter resource must include app-required lease ownership metadata; adapter-only success did not compose. | S4/S8/C | `runtime/android/app_test.go:TestRealAdapterThroughAppPersistsOwnedResource` | `runtime/android/lifecycle.go:Create` | A |
| M04 / E | App readiness deadline must govern boot; private adapter boot wait bypassed it. | S4/S8/C | `runtime/android/android_test.go:TestBootingLaunchReturnsForAppReadinessAndCompensation` | `runtime/android/lifecycle.go:Create` | D |
| M05 / E | Root absence must not authorize descendant cleanup; launcher exit can leave QEMU alive. | S3/S8/N | `execx/detached_test.go:TestDetachedSurvivesLaunchingCLI` | `execx/detached.go:Observe` | D |
| M06 / E | Supported SDK data-directory seeds must work; userdata.img-only assumption rejected modern images. | S5/S5/N | `runtime/android/android_test.go:TestModernSystemImageSeedsPrivateDataWithoutFabricatedImage` | `runtime/android/discovery.go:Validate` | D |
| M07 / E | Stopped markers cannot authorize deletion of a reappeared process or occupied port. | S2/S8/I | `runtime/android/android_test.go:TestStoppedMarkerNeverAuthorizesCleanupOfReappearedResource` | `runtime/android/lifecycle.go:Inspect/Destroy` | D |
| M08 / E | A Windows Job in another logon session is uncertain, not absent. | S5/S8/N | `execx/detached_windows_test.go:TestDetachedObservationRejectsDifferentWindowsSession` | `execx/detached_windows.go:observeDetachedJob` | W |
| M09 / E | Windows Job name must survive launching CLI; final handle close lost reopenable identity while processes lived. | S5/S5/N | `execx/detached_test.go:TestDetachedSurvivesLaunchingCLI` | `execx/detached_windows.go:startDetached` | D |
| M10 / E | Explicit local ADB selection must permit separately owned startup; numeric -H disabled auto-start. | S5/S5/N | `runtime/android/adb_test.go:TestSharedADBStartsOutsideEmulatorContainment` | `runtime/android/adb.go:ensureADBServer` | D |
| M11 / E | Authenticated kill must wait for actual tree/port absence; 10s was shorter than Emulator37 graceful stop. | S5/S5/N | `runtime/android/android_test.go:TestDestroyWaitsForPostKillTreeConfirmation` | `runtime/android/lifecycle.go:Destroy` | D |
| M12 / E | Missing-Job proof must reject PID reuse on the second identity read. | S2/S8/I | `execx/detached_windows_test.go:TestDetachedMissingGuardianRequiresDurableEmptyEvidence` | `execx/detached_windows.go:observeDetachedJob` | G |
| M13 / E | Shared ADB daemon must not inherit Emulator containment; otherwise emulator cleanup can remain live forever. | S5/S8/N | `runtime/android/adb_test.go:TestSharedADBStartsOutsideEmulatorContainment` | `runtime/android/adb.go:ensureADBServer` | D |
| M14 / E | SDK clients must not replace incompatible shared ADB; probe protocol before operational commands. | S3/S8/I | `runtime/android/adb_test.go:TestIncompatibleOrMalformedSharedServerNeverInvokesOperationalADB` | `runtime/android/adb.go:compatibleADB` | D |
| M15 / E | Empty Job must include completed evidence publication; late guardian writes raced directory deletion. | S5/S5/N | `execx/detached_windows_test.go:TestDetachedEmptyJobWaitsForPublishedCompletion` | `execx/detached_windows.go:observeDetachedJob` | W |
| M16 / R1 | Global inventory must show Compose orphans even when registry contains only Android rows. | S4/S8/C | `app/reconciliation_inventory_test.go:TestInventoryFindsComposeOrphansWithOnlyAndroidLeases` | `app/reconciliation_inventory.go:Inventory` | A |
| M17 / R1 | Android destroy preview must describe private AVD cleanup without Compose-only diagnostics or effects. | S3/S8/C | `app/android_test.go:TestAndroidDestroyPreviewReportsPrivateCleanupWithoutEffects` | `app/destroy_preview.go:previewDestroy` | A |
| M18 / R1 | Each runtime owns its pending readiness budget; satisfied short Compose checks must not shorten Android or postpone overdue Compose. | S4/S8/C | `app/mixed_readiness_test.go:TestMixedReadinessSatisfiedComposeDoesNotShortenAndroidBoot;TestMixedReadinessPendingComposeBoundsEarlierAndroidInspection` | `app/lifecycle.go:waitReady` | A |
| M19 / R1 | Runnable SDK tools and acceleration must fail before allocation; Doctor checks were absent from Validate. | S4/S8/C | `runtime/android/app_test.go:TestSDKPrerequisitesFailBeforeAppAllocation` | `runtime/android/discovery.go:sdkPrerequisites/Validate` | A |
| M20 / R2 | Repository doctor must validate every selected AVD, not merely generic SDK health. | S3/S8/C | `cli/android_test.go:TestAndroidRepositoryDoctorValidatesEverySelectedTemplate` | `cli/lifecycle.go:doctorManifest` | C |
| M21 / R2 | Android logs remain readable/redacted after release and scoped to component. | S4/S8/C | `cli/logs_test.go:TestAndroidProcessLogsActiveReleasedAndComponentIsolation` | `cli/lifecycle.go:runtimeLogEntries` | C |
| M22 / R2 | Image ABI must match supported host; invalid architecture must precede reservations. | S2/S8/I | `runtime/android/app_test.go:TestImageArchitectureMatrix;TestArchitectureAndADBFailBeforeAppAllocation` | `runtime/android/discovery.go:validateImageArchitecture` | A |
| M23 / R2 | Shared ADB compatibility must be checked during preflight, before reservation. | S4/S8/C | `runtime/android/app_test.go:TestArchitectureAndADBFailBeforeAppAllocation` | `runtime/android/discovery.go:Validate` | A |
| M24 / R2 | State/output must not overlap immutable SDK/template/image inputs, including aliases and other runtimes. | S2/S8/I | `app/android_paths_test.go:TestAndroidInputOverlapResolvesAliasesAndAncestors;TestAndroidPathsCompareOtherRuntimeInputs` | `app/android_paths.go` | A |
| M25 / R2 | Portable runtime names must reject case-folding collisions before allocation. | S2/S8/I | `app/android_paths_test.go:TestAndroidCaseCollisionFailsBeforeAllocation` | `config/manifest.go` | A |
| M26 / R2 | Independent cleanup must continue after local failure while retaining sources/reservations; global cancellation/fence loss still stops effects. | S4/S8/P | `app/cleanup_siblings_test.go:TestCleanupContinuesIndependentRuntimesAndRetainsSourcesUntilRetry;TestCleanupContinuesAfterRuntimeLocalTimeout` | `app/lifecycle.go:cleanup` | A |
| M27 / R2 | CLI must not create registry before app input preflight; app-only immutability tests missed the factory write. | S4/S8/C | `cli/create_store_test.go:TestCreateServiceRejectsInputOverlapBeforeOpeningRegistry` | `cli/create_store.go:openCreateService` | F |
| M28 / R2 | A procfs read racing process exit can return ESRCH; vanished census entries cannot prove group absence. | S5/S5/N | `execx/detached_linux_test.go:TestDetachedProcStatReadAfterExitReportsVanishedEntry;TestDetachedProcDisappearanceDoesNotHideInspectionFailures` | `execx/detached_linux.go:detachedProcEntryGone` | D |
| M29 / F | Unconfirmed build termination must block later forced cleanup, not only initial quarantine. | S4/S8/P | `app/applications_test.go:TestMobileUnconfirmedBuildBlocksLaterForcedCleanup` | `app/applications.go:applicationCleanupBarrier` | A |
| M30 / F | Requested reverse mapping is not confirmed ownership; preserve an ambiguous existing mapping and block device cleanup. | S3/S8/I | `app/applications_test.go:TestMobileUnconfirmedReversePreservesMapping` | `app/applications.go:cleanupApplications` | A |
| M31 / F | Launch intent is not READY; require launch confirmation and executable/directory identity on reconcile. | S3/S8/P | `app/application_identity_test.go:TestMobileReconcileRejectsIncompleteApplicationSnapshot` | `app/applications.go:applicationConsistent` | A |
| M32 / F | Build termination and evidence durability are independent; failed artifact persistence must retain source/APK after store recovery. | S4/S8/P | `app/application_identity_test.go:TestMobileIncompleteBuildEvidenceBlocksCleanup` | `app/applications.go:buildApplications/applicationCleanupBarrier` | A |
| M33 / F | SDK helper discovery must be lease-private; two live emulators shared netsim discovery and sibling cleanup became uncertain. | S5/S5/N | `runtime/android/android_test.go:TestEmulatorEnvironmentUsesPrivateNativeDiscoveryPaths;cli/flutter_integration_test.go:TestRealFlutterAndroidBackendLease` | `runtime/android/lifecycle.go` | N |
| M34 / F | Lock-loss injection must really replace the token and assert write success; raw SQLite writer contention could bypass injection. | S3/S5/T | `app/lockloss_test.go:TestLockLossFixtureWaitsForSQLiteWriter` | `app/lockloss_test.go:replaceOwner` | H |
| M35 / FR | Selected APK outputs must not collide before effects; build-first otherwise overwrites a previous application. | S2/S8/I | `app/flutter_review_test.go:TestSelectedApplicationAPKOutputsCannotCollide` | `app/plan.go:BuildPlan` | A |
| M36 / FR | Public docs must agree with delivered Flutter support; stale roadmap/manifest/CLI statements mislead users. | S7/S8/D | `tools/repoctl:docs-check plus paired meaning review` | `README.md and docs/product-specs` | DOC |
| M37 / FR | Host Flutter doctor requires Android too, without installation or allocation. | S3/S8/C | `cli/flutter_test.go:TestFlutterHostDoctorRequiresBothToolchains;TestFlutterHostDoctorMissingPrerequisitesIsPure` | `cli/flutter.go:doctorFlutterHost` | C |
| M38 / FR | Default plan table must show selected application requirements; JSON-only tests missed default output. | S3/S8/C | `cli/flutter_test.go:TestFlutterPlanTableIncludesSelectedApplicationRequirements` | `cli/lifecycle.go:addLifecycle` | C |
| M39 / FR | No reverse mapping means no reverse/endpoint query; optional path must remain usable. | S3/S8/C | `app/flutter_review_test.go:TestApplicationWithoutReverseSkipsNetworkObservation` | `app/applications.go:observeApplication` | A |
| M40 / FR | Project/ancestor symlinks must be refused before and after build, even when their targets stay in tree. | S2/S8/I | `runtime/flutter/flutter_test.go:TestBuildRejectsInternalProjectSymlinks` | `runtime/flutter/flutter.go:Validate/Build` | D |
| M41 / FR | Dry-run must honor both durable build barriers, including force; preview cannot claim unsafe deletion. | S3/S8/C | `app/flutter_review_test.go:TestDestroyPreviewHonorsApplicationBuildBarriers` | `app/destroy_preview.go:previewDestroy` | A |
| M42 / FR | Nonzero ADB results must retain bounded/redacted diagnostics and error identity. | S2/S8/I | `runtime/android/application_test.go:TestApplicationExecutionFailureRetainsDiagnosticsAndErrorIdentity` | `runtime/android/application.go` | D |
| M43 / U | Window IDs change across instrumentation reconnect; semantic identity must use stable window metadata. | S5/S5/N | `cli/ui_integration_test.go:TestRealAndroidUIObserver` | `runtime/android/uihelper/source/Observer.java:tree` | N |
| M44 / U | Recovery needs durable positive eligibility; missing final classification must not clear a host/evidence barrier. | S4/S8/P | `app/ui_recovery_test.go:TestUIRecoverRefusesUnpersistedFailureClassification` | `app/ui_recovery.go:RecoverUI` | A |
| M45 / U | CLI UI exits must preserve prerequisite/input/registry distinctions; raw errors all defaulted to input exit2. | S4/S8/C | `cli/ui_test.go:TestUIExitClassificationPreservesCauses;TestUIRegistryOpenFailureReturnsObservationExit;TestUICorruptRegistryReturnsObservationExit` | `cli/ui.go` | C |
| M46 / U | Concrete logcat evidence must cross the adapter/app boundary; Raw versus Binary mismatch lost logs. | S4/S8/C | `runtime/android/ui_test.go:TestUILogPIDAttributionAndBounds;app/ui_test.go:TestUILogcatReturnsBoundedRedactedInlineEvidence` | `runtime/android/ui.go:ObserveUI;app/ui.go:UI` | S |
| M47 / U | Backend stale/ambiguous/unavailable status must remain a stable diagnostic. | S3/S8/C | `app/ui_test.go:TestUIBackendRefusalsRetainStableDiagnostics` | `app/ui.go:UI` | A |
| M48 / U | Mutating helper must report post-action fingerprint; initial implementation omitted it. | S3/S8/C | `cli/ui_integration_test.go:TestRealAndroidUIObserver (no explicit fingerprint assertion found)` | `runtime/android/uihelper/source/Observer.java:onStart` | G |
| M49 / U | Incomplete accessibility traversal must never be presented as complete evidence. | S2/S8/B | `app/ui_test.go:TestUIRejectsSnapshotTamperingAndAmbiguity (consumer only)` | `runtime/android/uihelper/source/Observer.java:node/tree` | G |
| M50 / U | ACTION_SET_TEXT success is not readback success; focusing can invalidate ordinals and hit a wrong node. | S5/S5/N | `cli/ui_integration_test.go:TestRealAndroidUIObserver` | `runtime/android/uihelper/source/Observer.java:onStart` | N |
| M51 / U | ADB timeout does not prove remote helper stopped; recovery must prove helper-only quiescence without replay. | S1/S8/P | `app/ui_test.go:TestUIHelperRecoveryAfterInternalTimeout;app/ui_recovery_test.go:TestUIRecoverFailureRetainsBarrier` | `app/ui.go:UI;app/ui_recovery.go:RecoverUI` | A |
| M52 / U | Back fixture must establish its target; assuming IME dismissal exited the application. | S5/S5/T | `cli/ui_integration_test.go:TestRealAndroidUIObserver` | `cli/ui_integration_test.go (fixture only)` | N |
| M53 / U | Swipe fixture must use actual viewport after IME resize; fixed screenshot coordinates missed the ScrollView. | S5/S5/T | `cli/ui_integration_test.go:TestRealAndroidUIObserver` | `cli/ui_integration_test.go (fixture only)` | N |

## Current evidence and unresolved coverage

On frozen target, Linux/Go 1.27.1 targeted historical regressions passed with `-race -count=1`: Android 2.461s, Flutter 1.037s, execx 1.300s, app 19.000s, CLI 2.313s. Selection included the named Android/SDK/ADB/Detached/MixedReadiness/Cleanup/CreateService/Flutter/Mobile/UI regressions and the selected APK, reverse, preview, symlink and fault-injector tests. The UI helper subpackage reported **no tests to run** under that regex; it is not counted as test execution. Windows-only tests and real SDK/UI tests were inspected, not executed in this subtask; baseline owner runs those separately. Existing source files were not modified.

M12 also lacks a located forced second-read PID-race regression: the named Windows test uses an absent PID to validate proof, and the generic PID-reuse test does not force that precise window.

M48/M49 are coverage gaps, not confirmed current producer defects. The Java helper computes `after_fingerprint` and marks inaccessible/missing hierarchy children truncated. The existing Go helper tests verify provenance/build inputs, not the Java traversal/action algorithm. `TestRealAndroidUIObserver` asserts complete normal trees and real effects but does not explicitly assert post-action fingerprint or force an inaccessible child/exact producer boundary. `TestUIRejectsSnapshotTamperingAndAmbiguity` proves that the app rejects a supplied truncated flag; it cannot prove the helper emits one. Proposed follow-up: producer-level fixtures (or a controlled Android instrumentation fixture) with explicit result fields and omission cases. Status: deferred coverage disposition; no current correctness finding ID assigned for these two gaps.

### AUDIT-BOUNDARY-001 — exact Android log tail is marked truncated

- Severity: Medium under the audit's exact-limit definition. Disposition: untriaged; Phase B has not accepted or rejected it.
- Invariant: `truncated=true` requires actual omitted evidence; an exact-size complete response is not itself proof.
- Location: `internal/runtime/android/ui.go:357`, `boundedUILog`, reached by `Adapter.ObserveUI` logcat at line 251; request at line 243 uses device `logcat -t 2000`.
- Trigger/reproducer: in the existing `applicationFixture`, set PID response `1234`, device time `1000`, and the exact logcat command response to 2000 repetitions of `999.000 1234 1234 I Tag: current\n`. Call `ObserveUI` with package `com.example.app`, `SinceSeconds:30`. The same setup with 1999 repetitions is the control.
- Observed: the temporary Go-overlay regression failed at 2000, returning all 66,000 bytes and `Truncated=true`; 1999 returned complete. Test package failed in 0.014s. No byte or locally retained-line omission occurred. This was an actual concrete adapter invocation with an injected runner, not a native-device claim.
- Expected/impact: a complete exact-size capture should remain complete. Current output falsely signals omission. The device tail cap may actually omit earlier lines when more than 2000 exist, but the command returns no overflow bit. The helper's unconditional `raw line count >=2000` conflates that possible upstream omission with observed omission. Merely removing the comparison would create false completeness for real overflow; upstream extra-item/proof design belongs to disposition/remediation.
- Existing coverage: `TestUILogPIDAttributionAndBounds` exercises two lines and a 2001-line, over-byte-budget response. It does not cover a short complete 1999/2000/2001 response; the current consumer tests likewise rely on supplied flags.
- Regression/resolution: temporary overlay only; no product/test change, no repair yet. Historical relation: M49's producer completeness gap and Browser exact-limit class. Same-pattern search candidates: Java `node/tree` limit checks, UI text/log artifact caps and browser evidence producers; no additional defect is implied by their presence.
- Escape analysis: detected S9; earliest realistic S2. `BOUNDARY_GAP` and `ORACLE_COUPLING`: the existing test combines count overflow with byte overflow and does not independently test exact count. S3/S4 fixtures use short logs; prior S5 UI native fixture does not generate an exact 2000-line tail. S6 runs those tests but has no semantic cap validator. S7/S8 opportunities were an independent exact-limit oracle; the historical plan does not record one. Proposed preventive control: shared boundary-case convention plus provider overflow proof, expected S2/S3; status deferred until disposition.

## Historical limitations retained

Android review plans correctly preserve failed two-emulator cleanup with rootless live groups, rather than count readiness as full acceptance. Flutter's later private netsim discovery and successful two-lease run supersede the missing acceptance, but logs alone did not prove every older shutdown failure shared that cause. No historical cause is retrospectively invented.

Missing SDK, offline package retrieval, insufficient temporary-disk space, and the initial 50ms unrelated fixture readiness budget were environment/test prerequisites, not independent product repairs in this corpus. The recorded SQLite injection defect is reproducible; its exact connection to the earlier CI failure remains a hypothesis. Native Windows/macOS SDK execution remains distinct from native Go tests. The missing PR5 inventory was completed with the external comments API.

No new audit control, product repair, commit, or remote write was performed by this historical review. The bounded broader mobile boundary/state/identity review is recorded in [current mobile](current-mobile.md), including `AUDIT-REDACTION-001`; final disposition belongs to the audit owner.

## Supplementary PR5 external review corpus

Fetched `repos/mahcialet/agent-env/pulls/5/comments` with pagination and distinguished all 24 substantive reviewer findings from author replies. M54–M77 preserve every finding, comment ID and original severity. Each ID resolves at `https://github.com/mahcialet/agent-env/pull/5#discussion_r<ID>`. Repair replies are not implementation proof: frozen source and tests were checked separately. All invariants remain applicable (yes). G means no direct regression for the original defect was located, not automatically that a fix is absent.

M70's repair reply describes a 1 MiB check after raw remarshal; typed result/snapshot publication is a separate path and AUDIT-REDACTION-001 reproduces its remaining violation. M72–M77 remain reproducible in the frozen target via overlays and are added to the current audit. M65/M77 are the initial and repeated incomplete-fix findings for one invariant.

| ID / comment / original severity | Invariant and impact | Earliest/detected/profile | Current test / limitation | Production | Scope |
| --- | --- | --- | --- | --- | --- |
| M54 / 3953621034 / P1 | Large valid log capture must reach bounded tail instead of permanent host-process cleanup barrier. | S3/S8/B | `runtime/android/ui_test.go:TestUILogPIDAttributionAndBounds` | `runtime/android/ui.go:ObserveUI(logcat)` | G |
| M55 / 3953621041 / P2 | Invalid/unknown/cross-lease snapshot references must exit2, preserving real storage exit7. | S4/S8/C | `cli/ui_test.go:TestUIExitClassificationPreservesCauses (typed synthetic errors)` | `app/ui.go:loadUISnapshot` | G |
| M56 / 3953621046 / P2 | Helper installation effect and identity must be recorded, not a generic may-install note. | S4/S8/C | `runtime/android/ui_test.go:TestUIHelperStagingStaysInOwnedRuntimeAndCleansUp (staging only)` | `runtime/android/ui.go:ObserveUI;app/ui.go:UI` | G |
| M57 / 3953621054 / P2 | Full-display screenshot/native inputs must not claim application package isolation. | S3/S8/C | `app/ui_test.go:TestUIDegradedDiagnosticsAndApplicationScope (request scope only)` | `app/ui.go:UI run notes` | G |
| M58 / 3953753802 / P2 | Parsed helper response and later polls must retain earlier HelperInstalled=true. | S3/S8/C | `no direct installation-marker/poll regression located` | `runtime/android/ui.go:ObserveUI;app/ui.go:UI` | G |
| M59 / 3953753807 / P2 | Explicit runtime/snapshot selector mismatch must remain an input error after loading. | S4/S8/C | `app/ui_test.go:TestUIRejectsInvalidStateAndSelectionBeforeDevice (no exact typed post-load assertion)` | `app/ui.go:UI` | G |
| M60 / 3953753811 / P2 | Wait sleep deadline must publish timeout/unsatisfied status, not last successful poll as ok. | S3/S8/P | `app/ui_test.go:TestUIWaitIsBoundedAndDoesNotInput (run failure only)` | `app/ui.go:UI polling` | G |
| M61 / 3953753815 / P2 | Semantic window title/bounds/root package/class must be inspectable, not only opaque key. | S3/S8/C | `cli/ui_integration_test.go:TestRealAndroidUIObserver (no explicit window-field assertion)` | `domain/ui.go:UIWindow;runtime/android/uihelper/source/Observer.java` | G |
| M62 / 3953753816 / P2 | Install verified helper bytes, not a later-opened replaceable configured path. | S2/S8/I | `runtime/android/ui_test.go:TestUIHelperStagingStaysInOwnedRuntimeAndCleansUp` | `runtime/android/ui.go:ObserveUI install staging` | D |
| M63 / 3953839026 / P1 | Redact window metadata before raw/normalized/result persistence. | S2/S8/I | `app/ui_test.go:TestUIEvidenceRedactsEnteredAndEditableText (no window secret fixture)` | `app/ui.go:sanitizeUIObservation` | G |
| M64 / 3953839032 / P2 | Invalid/stale recover run references must exit2, not storage exit7. | S4/S8/C | `app/ui_recovery_test.go:TestUIRecoverRefusesUnsafeIntentAndEvidence (no CLI exact classification)` | `app/ui_recovery.go:RecoverUI` | G |
| M65 / 3953839035 / P1 | Native preflight failure before input must not create unrecoverable uncertainty. | S4/S8/P | `runtime/android/ui_test.go:TestUIOwnershipFailureCannotDispatchInput (does not assert Confirmed)` | `runtime/android/ui.go:ObserveUI native branch` | G |
| M66 / 3953922442 / P1 | Configured-secret derived node/window hashes must not remain as offline verification oracles. | S2/S8/I | `app/ui_test.go:TestUIEvidenceRedactsEnteredAndEditableText (plaintext only)` | `app/ui.go:sanitizeUIObservation` | G |
| M67 / 3953922444 / P2 | Wait requires application scope; unrelated/system window must not satisfy predicate. | S3/S8/C | `cli/ui_test.go:TestUIFlagValidationAndExitCodes` | `app/ui.go:UI option validation;cli/ui.go` | C |
| M68 / 3953922447 / P2 | Persist redacted wait predicate/timeout/application as intent. | S3/S8/C | `app/ui_test.go:TestUIWaitIsBoundedAndDoesNotInput (no argv assertion)` | `app/ui.go:UI run argv` | G |
| M69 / 3953922449 / P2 | Missing SDK adb must retain prerequisite exit3 after safe-error sanitization. | S4/S8/C | `app/ui_test.go:TestUIPrerequisiteErrorSurvivesPrivacySanitization (synthetic provider marker)` | `runtime/android/ui.go:uiSafeError` | G |
| M70 / 3953922453 / P2 | Reapply raw/result response size after sanitization expands strings and omitted fields. | S2/S8/B | `app/ui_test.go:TestUIEvidenceRedactsEnteredAndEditableText (no final size assertion)` | `app/ui.go:sanitizeUIObservation/UI artifacts` | G |
| M71 / 3953922458 / P2 | Recover uses recorded helper identity even if mutable host helper path changes or disappears. | S4/S8/I | `app/ui_recovery_test.go:TestUIRecoverInterruptedHelperPreservesFailedOutcome (no real digest reconstruction)` | `app/ui_recovery.go:RecoverUI;runtime/android/ui.go:quiesce` | G |
| M72 / 3954080090 / P1 | Automatic editable suppression must retain safe actionable fingerprints. | S3/S8/C | `app/ui_test.go:TestUIEvidenceRedactsEnteredAndEditableText (callback not asserted)` | `app/ui.go:sanitizeUIObservation` | G |
| M73 / 3954080094 / P1 | Every helper branch must replace generic shell backend with verified source/APK digest. | S3/S8/I | `runtime/android/ui_test.go:TestUISemanticActionRejectsChangedBackendBeforeDeviceInput (different-string only)` | `runtime/android/ui.go:ObserveUI helper branch` | G |
| M74 / 3954080097 / P1 | Window-secret derived node fingerprints must be cleared together with window key. | S2/S8/I | `no direct nested window-hash regression located` | `app/ui.go:sanitizeUIObservation` | G |
| M75 / 3954080101 / P2 | Unset helper opt-in cannot resolve current working directory and install its APK. | S2/S8/I | `runtime/android/uihelper/helper_test.go:TestVerifyRejectsMismatchedProvenance (explicit directory only)` | `runtime/android/uihelper/helper.go:Load` | G |
| M76 / 3954080103 / P2 | Fractional log lookback must not lose valid requested records through integer truncation. | S2/S8/B | `runtime/android/ui_test.go:TestUILogPIDAttributionAndBounds (integral seconds only)` | `app/ui.go:UI request SinceSeconds` | G |
| M77 / 3954080105 / P1 | Typed native preflight protection must undo/prevent the already-false Confirmed state. | S4/S8/P | `runtime/android/ui_test.go:TestUIOwnershipFailureCannotDispatchInput (ignores observation)` | `runtime/android/ui.go:ObserveUI native branch` | G |
