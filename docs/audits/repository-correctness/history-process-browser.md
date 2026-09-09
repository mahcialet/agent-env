---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Historical review corpus: process, browser and MVP

[日本語](history-process-browser.ja.md) · [Audit execution authority](../../exec-plans/completed/repository-correctness-audit.md)

Phase A target: `031869c8b9073b8e23bc17fbc55243666a52f557`.
This is a review-only historical annex, not disposition or remediation.
The 56 entries below separate 22 browser PR findings (9 + 8 + 5) from
implementation/native discoveries and 10 process / 15 grouped MVP entries.
No product/test changes or remote actions were made for this annex.

## Source inventory and interpretation

- P: [Persistent process plan](../../exec-plans/completed/persistent-process-runtime.md), Integration checkpoint, Surprises, first published native CI, Windows correction checkpoint and retrospective.
- D: [Process destroy preview review](../../exec-plans/completed/process-destroy-preview-review.md), entire plan.
- B1/B2/B3: [Browser plan](../../exec-plans/completed/browser-cdp-automation.md), first/second/third PR #10 follow-up, Surprises, Decision Log and retrospectives. BI denotes independent implementation/review follow-ups; BN denotes native/fixture follow-ups.
- M: [MVP plan](../../exec-plans/completed/agent-env-mvp.md), Surprises, independent review checkpoints, cancellation review, native/concurrency repair, renewal fixture and PR #1 follow-up.

Original severity is not recorded for most entries; do not invent one.
HB26 is explicitly P1, HB27 P2, and HM10/HM11 P1. Other rows state
impact rather than retrospectively assigning severity. Sources contain PR numbers,
commits and CI links but no individual review-thread ID inventory for these rows.
Consequently IDs here are stable corpus IDs, not fabricated original thread IDs.
The MVP groups several corrections without enough detail to split every original
comment; HM08 preserves that grouping. Routine formatting/translation setup
failures and prerequisite observations are not counted as separate product defects.

The stage column is detected/earliest realistically preventable. S8 is documented
independent/PR review, S5 native execution and S4 actual concurrent integration.
“implementation” means the source does not identify a finer detected stage;
it is not silently recast as S8. Earliest stages are this audit's assessment,
using the then-existing contract and interfaces, not a claim that such a test existed.

## Coverage semantics

Production/test paths below are relative to `internal/`, except explicit
`tools/`, `docs/`, `.github/` and `ARCHITECTURE.md`.
Unqualified subsequent test names share the preceding test file.

- F: inspected regression invokes the public package/app/store entry point with real local persistence/filesystem or injected effects. This proves the stated slice, not a whole CLI/provider execution.
- C: inspected CDP protocol-component entry point with explicit protocol fixtures; it does not execute JavaScript or a real browser by itself.
- H: helper-only regression; it still checks the local invariant but cannot prove composition/call ordering at the full public entry point.
- N: real native fixture exists, sometimes alongside deterministic component tests; current-run native evidence is owned by the audit baseline, not implied by source inspection.
- D: documentation/process claim, inspected prose/recorded CI; docs-check does not prove semantic truth or that every acceptance criterion has passed.
- U: exact original regression mapping is unresolved; nearby coverage is not counted as proof.

All rows remain applicable to the frozen target. Current tests still assert the
named local invariant except U; H/D explicitly do not prove the full behavioral
claim. Inspection confirmed production call sites remain live, but no blanket
claim is made that all 56 would fail under a restored historical mutation.
No mutation replay was performed in this annex. Tests were not weakened.
Current audit finding IDs: none assigned by this annex; the open coverage questions
below require further Phase A evidence before disposition.

## Historical findings

### HP01 — P

Successful readiness probe outlived the process and falsely authorized READY; reobserve owned process after probes.

- Detected/earliest: S8/S4; escape: `COMPOSITION_GAP`.
- Production: `app/process.go:confirmProcessesReady`.
- Current regression: `app/process_lifecycle_test.go:TestProcessExitDuringSuccessfulProbeCannotBecomeReady`; coverage F.

### HP02 — P

Proven pre-spawn failure became ambiguous cleanup; typed no-spawn plus zero PID permits prepared-state release, unknown zero identity does not.

- Detected/earliest: S8/S3; escape: `FAILURE_INJECTION_GAP`.
- Production: `app/process.go:startProcess; runtime/process/process.go:Start`.
- Current regression: `app/process_lifecycle_test.go:TestProcessKnownNoSpawnCompensatesPreparedState; runtime/process/process_test.go:TestProvenNoSpawnFailureCanReleaseWithoutQuarantine`; coverage F.

### HP03 — P

Symlinked state home changed immutable paths after reservation; canonicalize before reservation.

- Detected/earliest: S8/S4; escape: `COMPOSITION_GAP`.
- Production: `app/lifecycle.go:CanonicalFuture before reservation`.
- Current regression: `cli/process_native_test.go:TestPersistentProcessNativeCLI`; coverage N.

### HP04 — P

Raw port key overwrote explicit endpoint alias; declaration wins.

- Detected/earliest: S8/S2; escape: `NEGATIVE_FIXTURE_GAP`.
- Production: `app/readiness.go:Endpoints`.
- Current regression: `app/plan_process_test.go:TestProcessEndpointAliasWinsOverRawRuntimePort`; coverage F.

### HP05 — P

Readiness literals could persist inherited secrets; reject before snapshot, allow explicit environment references.

- Detected/earliest: S8/S2; escape: `HELPER_ONLY,NEGATIVE_FIXTURE_GAP`.
- Production: `app/plan.go:BuildPlan -> validateProcessSecrets`.
- Current regression: `app/plan_process_test.go:TestProcessReadinessRejectsLiteralInheritedSecretBeforeSnapshot`; coverage H.

### HP06 — P

Completed owned Job was rejected on historical PID reuse; durable empty-Job proof precedes historical root lookup while active checks remain strict.

- Detected/earliest: S5/S5; escape: `NATIVE_EVIDENCE_GAP,CONCURRENCY_GAP`.
- Production: `execx/detached_windows.go; execx/detached.go`.
- Current regression: `execx/managed_windows_test.go:TestManagedWindowsCompletedJobIgnoresReusedHistoricalPID`; coverage N.

### HP07 — P

Terminal reservation guard rejected repeated destroy/post-release quarantine; terminal reservation history must remain separate from observed lifecycle and never reacquire ports.

- Detected/earliest: implementation/S3; escape: `COMPOSITION_GAP`.
- Production: `store/sqlite/process.go:saveProcesses`.
- Current regression: `store/sqlite/process_test.go:TestProcessReleasedReservationsCannotResurrect; TestProcessTerminalReservationKeepsPostReleaseQuarantineVisible`; coverage F.

### HP08 — P

Current host environment and launch receipt alone lost startup secret proof; persist independent fingerprints before launch and preserve them on receipt failure.

- Detected/earliest: implementation/S3; escape: `FAILURE_INJECTION_GAP`.
- Production: `runtime/process/process.go; runtime/process/redaction.go`.
- Current regression: `runtime/process/process_test.go:TestLogsRedactOriginalSecretsWhenEnvironmentChanges; TestReceiptFailureStillHasDurableLogRedactionForCompensation`; coverage F.

### HP09 — P

Fixed-range allocator could repeatedly select an occupied first port; hold OS-selected loopback listeners through reservation commit.

- Detected/earliest: implementation/S2; escape: `BOUNDARY_GAP,ORACLE_COUPLING`.
- Production: `store/sqlite/process.go:allocateProcesses`.
- Current regression: `store/sqlite/process_test.go:TestProcessDynamicReservationAvoidsExternallyBoundPort; TestProcessConcurrentReservationPortsAreDisjoint`; coverage F.

### HP10 — D

Correct process inspection fell through to Compose preview and omitted exited private-state cleanup; six state/force cases assert text and no side effects.

- Detected/earliest: S8/S4; escape: `HELPER_ONLY,COMPOSITION_GAP`.
- Production: `app/destroy_preview.go`.
- Current regression: `app/process_lifecycle_test.go:TestProcessDestroyPreviewReportsCleanupWithoutEffects`; coverage F.

### HB01 — B1

Plan completion preceded passing native acceptance; old green CI does not accept later repairs, and final gates must precede archival.

- Detected/earliest: S8/S7; escape: `REVIEW_CHECKLIST_GAP`.
- Production: `docs/exec-plans/completed/browser-cdp-automation.md`.
- Current regression: `repoctl docs-check plus recorded final native CI`; coverage D.

### HB02 — B1

URL equality misclassified opaque/inherited/blob frames. Native placeholder origins and omitted OOPIFs require browser-enforced origin and topology proof.

- Detected/earliest: S8+S5/S3; escape: `ORACLE_COUPLING,NATIVE_EVIDENCE_GAP`.
- Production: `browser/cdp/snapshot.go:frame proof`.
- Current regression: `browser/cdp/snapshot_test.go:TestFrameClassificationUsesSecurityOrigin; TestOutOfProcessFrameCannotBeSilentlyOmitted; cli/browser_native_test.go`; coverage N.

### HB03 — B1

Role-only URL wait accepted an empty predicate; require substring and reject role before dependencies.

- Detected/earliest: S8/S2; escape: `NEGATIVE_FIXTURE_GAP`.
- Production: `app/browser.go:validation`.
- Current regression: `app/browser_test.go:TestBrowserURLWaitRequiresSubstring; cli/browser_test.go:TestBrowserRejectsMissingOrUnsafeCLIInputBeforeStore`; coverage F.

### HB04 — B1

Network metadata bypassed string budgets (1,230,500 versus 65,536 bytes); every retained string must count.

- Detected/earliest: S8/S2; escape: `BOUNDARY_GAP`.
- Production: `browser/cdp/capture.go:boundCaptureStrings`.
- Current regression: `browser/cdp/capture_test.go:TestNetworkCaptureBoundsEveryPersistedString; TestNetworkCaptureCountsAllStringsAgainstTotalBudget`; coverage C.

### HB05 — B1

Shortened DOM names falsely reported complete evidence; disclose actual omission.

- Detected/earliest: S8/S2; escape: `BOUNDARY_GAP`.
- Production: `browser/cdp/snapshot.go:domSnapshot`.
- Current regression: `browser/cdp/snapshot_test.go:TestDOMNameTruncationIsReported`; coverage C.

### HB06 — B1

Architecture called the implemented browser provider future work; paired prose must match delivered scope.

- Detected/earliest: S8/S7; escape: `REVIEW_CHECKLIST_GAP`.
- Production: `ARCHITECTURE.md:Android/browser paragraphs`.
- Current regression: `paired architecture inspection; docs-check`; coverage D.

### HB07 — B1

Truncated AX evidence falsely proved gone; incomplete evidence cannot prove absence.

- Detected/earliest: S8/S2; escape: `INVARIANT_GAP,NEGATIVE_FIXTURE_GAP`.
- Production: `browser/cdp/client.go:wait`.
- Current regression: `browser/cdp/snapshot_test.go:TestGoneCannotSucceedWithTruncatedSnapshot`; coverage C.

### HB08 — B1

Unrelated events exhausted the capture queue and disconnected ordinary commands; admit only active session/methods, retain overflow failure.

- Detected/earliest: S8/S3; escape: `COMPOSITION_GAP`.
- Production: `browser/cdp/transport.go:subscribe/read loop`.
- Current regression: `browser/cdp/transport_test.go:TestTransportIgnoresUnsubscribedEvents; TestTransportCaptureIgnoresOtherSessionsAndMethods; TestTransportSubscribedOverflowFailsClosed`; coverage C.

### HB09 — B1

Input intent omitted page/snapshot/node provenance; persist validated target identity before every semantic effect including uncertain outcomes.

- Detected/earliest: S8/S4; escape: `FAILURE_INJECTION_GAP`.
- Production: `app/browser.go:Browser run intent`.
- Current regression: `app/browser_test.go:TestBrowserSemanticProvenanceSurvivesUncertainInput`; coverage F.

### HB10 — BI

Frame changes during AX collection paired evidence with old identity; recheck topology/origin and discard changed evidence entirely.

- Detected/earliest: S8/S3; escape: `CONCURRENCY_GAP`.
- Production: `browser/cdp/snapshot.go:post-collection proof`.
- Current regression: `browser/cdp/snapshot_test.go:TestSnapshotDiscardsAXWhenFrameProofChanges; TestWaitRetriesFrameChangesButNeverPublishesPartialEvidence`; coverage C.

### HB11 — BN

Windows sharing violations blocked profile removal after proved tree absence; retry only that OS error with deadlines and fresh path proof.

- Detected/earliest: S5/S5; escape: `NATIVE_EVIDENCE_GAP`.
- Production: `runtime/process/cleanup.go; cleanup_windows.go`.
- Current regression: `runtime/process/cleanup_windows_test.go:TestWindowsStateRemovalRetriesHeldFile; cleanup_test.go:TestStateRemovalRevalidatesOwnerBeforeRetry`; coverage N.

### HB12 — B2

Post-input protocol/readback errors falsely confirmed the whole action; exceptions or missing boolean readback remain uncertain.

- Detected/earliest: S8/S3; escape: `FAILURE_INJECTION_GAP`.
- Production: `browser/cdp/client.go:Observe; actions.go:act`.
- Current regression: `browser/cdp/input_review_test.go:TestPostInsertErrorRemainsUncertain; app/browser_test.go:TestBrowserUncertainMutationRetainsCleanupBarrier`; coverage C+F.

### HB13 — B2

Focus handlers redirected input; inactive tabs retained activeElement without reliable event delivery. Activate, revalidate and prove exact/document focus after focus and select-all.

- Detected/earliest: S8+S5/S3; escape: `COMPOSITION_GAP,NATIVE_EVIDENCE_GAP`.
- Production: `browser/cdp/actions.go:act/verifyFocus`.
- Current regression: `browser/cdp/input_review_test.go:TestInputFocusRedirectRefusesKeyboardDispatch; TestSelectionFocusChangeRefusesTextInsertion; TestInactiveDocumentRefusesKeyboardDispatch; TestActivationMutationRefusesKeyboardDispatch; cli/browser_native_test.go`; coverage N.

### HB14 — B2

DOM snapshots lacked AX origin/topology safeguards; both paths require before/after proof and approved document frame IDs.

- Detected/earliest: S8/S3; escape: `COMPOSITION_GAP`.
- Production: `browser/cdp/snapshot.go:domSnapshot; snapshot.go`.
- Current regression: `browser/cdp/snapshot_test.go:TestDOMSnapshotOriginAndDocumentProof`; coverage C.

### HB15 — B2

An unrelated tab OOPIF blocked the selected page; correlate parentless targets with selected-session owners without adopting foreign content.

- Detected/earliest: S8/S3; escape: `NEGATIVE_FIXTURE_GAP`.
- Production: `browser/cdp/snapshot.go:iframe ownership`.
- Current regression: `browser/cdp/snapshot_test.go:TestUnparentedIframeTargetsArePageScoped; cli/browser_native_test.go`; coverage N.

### HB16 — B2

Redaction expanded bounded console/network strings beyond final limits; rebound persisted run.json, not only provider values.

- Detected/earliest: S8/S4; escape: `COMPOSITION_GAP,BOUNDARY_GAP`.
- Production: `app/browser_redaction.go:boundRedactedBrowserCapture`.
- Current regression: `app/browser_review_test.go:TestBrowserCaptureBoundsAfterRedaction`; coverage F.

### HB17 — B2

URL predicates used scrubbed data; mocks included fragment in Frame.url while Chrome uses urlFragment. Match transient raw URL, retain scrubbed evidence.

- Detected/earliest: S8+S5/S3; escape: `ORACLE_COUPLING`.
- Production: `browser/cdp/client.go:wait; snapshot.go:frame URL`.
- Current regression: `browser/cdp/input_review_test.go:TestURLWaitUsesRawURLButPersistsScrubbedEvidence; cli/browser_native_test.go`; coverage N.

### HB18 — B2

Deadline completion silently omitted queued events; atomically stop admission and mark pending omission, preserving overflow error.

- Detected/earliest: S8/S3; escape: `CONCURRENCY_GAP,BOUNDARY_GAP`.
- Production: `browser/cdp/capture.go:finishCapture; transport.go:subscribe`.
- Current regression: `browser/cdp/capture_test.go:TestCaptureDeadlineMarksPendingEventsTruncated`; coverage H.

### HB19 — B2

Stored browser manifest digest was unchecked; refuse changed declaration before inspection or effects.

- Detected/earliest: S8/S2; escape: `NEGATIVE_FIXTURE_GAP,HELPER_ONLY`.
- Production: `app/browser.go:Browser -> selectBrowser`.
- Current regression: `app/browser_review_test.go:TestBrowserRejectsChangedStoredManifest`; coverage H.

### HB20 — B3

Redacted semantic evidence exceeded field/1 MiB bounds or failed at an old 2 MiB guard. Bound final encoding, retain node identity, complete read-only run and refuse truncated input.

- Detected/earliest: S8/S4; escape: `COMPOSITION_GAP,BOUNDARY_GAP`.
- Production: `app/browser_redaction.go:boundRedactedBrowserSnapshot`.
- Current regression: `app/browser_semantic_bounds_test.go:TestBrowserSemanticBoundsAfterRedaction`; coverage F.

### HB21 — B3

AX exposed closed-root controls but host.shadowRoot traversal refused input; target-outward root traversal proves every host hit/focus and rejects overlays.

- Detected/earliest: S8/S5; escape: `NATIVE_EVIDENCE_GAP,ORACLE_COUPLING`.
- Production: `browser/cdp/actions.go:nodeHit/verifyFocus`.
- Current regression: `cli/browser_native_test.go:TestBrowserNativeCLI /closed-shadow`; coverage N.

### HB22 — B3

Capture deadline began after enable, letting a 20 ms request wait over 500 ms; include subscription/enable and fail unfinished enable.

- Detected/earliest: S8/S3; escape: `BOUNDARY_GAP,COMPOSITION_GAP`.
- Production: `browser/cdp/capture.go:capture`.
- Current regression: `browser/cdp/capture_test.go:TestCaptureDurationIncludesDomainEnable`; coverage C.

### HB23 — B3

Exactly 2048 AX nodes were falsely truncated across single/multiple frames and empty trailing frame; actual omission is required.

- Detected/earliest: S8/S2; escape: `BOUNDARY_GAP`.
- Production: `browser/cdp/snapshot.go:snapshot`.
- Current regression: `browser/cdp/snapshot_test.go:TestSnapshotNodeLimitMarksOnlyOmittedNodes`; coverage C.

### HB24 — B3

Nonstring/malformed/null console arguments were silently dropped as complete; omit private details but disclose truncation.

- Detected/earliest: S8/S2; escape: `NEGATIVE_FIXTURE_GAP`.
- Production: `browser/cdp/capture.go:console decoding`.
- Current regression: `browser/cdp/capture_test.go:TestConsoleMarksOmittedArgumentsTruncated`; coverage C.

### HB25 — BI

Missing isolated context made stale-target regressions pass before mutation. Assert injection was reached, not merely failure/no effects.

- Detected/earliest: S8/S2; escape: `ORACLE_COUPLING,NEGATIVE_FIXTURE_GAP`.
- Production: `browser/cdp/actions.go:act`.
- Current regression: `browser/cdp/actions_test.go:TestNodeChangesDuringOwnershipVerificationNeverInputs`; coverage C.

### HB26 — BI

P1: lost-fence error exposed raw provider observation before redaction; clear output and forbid stale-owner finalization.

- Detected/earliest: S8/S4; escape: `FAILURE_INJECTION_GAP`.
- Production: `app/browser.go:lost-fence return`.
- Current regression: `app/browser_test.go:TestBrowserLockLossDoesNotExposeObservation`; coverage F.

### HB27 — BI

P2: redaction corrupted authority when entered text equaled browser name; redact content while preserving identity.

- Detected/earliest: S8/S4; escape: `ORACLE_COUPLING,COMPOSITION_GAP`.
- Production: `app/browser_redaction.go:structured redaction`.
- Current regression: `app/browser_test.go:TestBrowserPriorTextRedactionPreservesAuthority; cli/browser_native_test.go`; coverage N.

### HB28 — BN

macOS synthetic Meta+A did not reliably clear text; explicit selectAll editing command plus readback is required.

- Detected/earliest: S5/S5; escape: `NATIVE_EVIDENCE_GAP`.
- Production: `browser/cdp/actions.go:selectAll`.
- Current regression: `browser/cdp/actions_test.go:TestSelectAllUsesExplicitEditingCommandOnEveryPlatform; cli/browser_native_test.go`; coverage N.

### HB29 — BN

Windows privacy oracle confused legitimate Unicode identity paths with entered secret; decode JSON and match actual entered text including escapes.

- Detected/earliest: S5/S2; escape: `ORACLE_COUPLING`.
- Production: `cli/browser_native_test.go:privacy oracle`.
- Current regression: `cli/browser_native_test.go:TestBrowserEvidenceSecretDetection`; coverage H.

### HB30 — BN

Linux sandbox startup and Windows LPAC executable access failed; provision only pinned executable installation access, never global sandbox relaxation.

- Detected/earliest: S5/S5; escape: `NATIVE_EVIDENCE_GAP`.
- Production: `.github/workflows/browser.yml:AppArmor/LPAC ACL`.
- Current regression: `cli/browser_native_test.go:TestBrowserNativeCLI startup/sandbox checks`; coverage N.

### HB31 — BN

50 ms Create fixture expired inside SQLite before the browser assertion; separate setup budget and restore operation settings.

- Detected/earliest: S5/S3; escape: `ORACLE_COUPLING`.
- Production: `app/browser_test.go:browserFixture`.
- Current regression: `app/browser_test.go:TestBrowserLifecycleGuards`; coverage F.

### HM01 — M

Windows CRLF checkout failed formatting despite equivalent source; normalize only CRLF and still reject real drift.

- Detected/earliest: S5/S2; escape: `NATIVE_EVIDENCE_GAP,BOUNDARY_GAP`.
- Production: `tools/repoctl format check`.
- Current regression: `tools/repoctl/main_test.go:TestFormattingCRLFAndActualDrift`; coverage H.

### HM02 — M

Missing worktree aliases mismatched Git registrations; compare existing ancestor identity without dropping registration proof.

- Detected/earliest: S5/S3; escape: `NATIVE_EVIDENCE_GAP`.
- Production: `source/gitcli/git.go:registration identity`.
- Current regression: `source/gitcli/missing_test.go:TestMissingWorktreeThroughParentAlias; TestMissingWorktreeRegistrationIsObservedAndRemoved`; coverage F.

### HM03 — M

Slash-rooted Windows paths violated relative-path assumptions; reject rooted/escape paths explicitly.

- Detected/earliest: S8/S2; escape: `NEGATIVE_FIXTURE_GAP`.
- Production: `paths/paths.go:source confinement`.
- Current regression: `paths/paths_test.go:TestSourceConfinement`; coverage H.

### HM04 — M

Unselected Compose resources entered cleanup; prune snapshots and prove sibling/unrelated volume survival in actual destroy.

- Detected/earliest: S8/S4; escape: `COMPOSITION_GAP`.
- Production: `runtime/compose/compose.go:selected snapshot`.
- Current regression: `runtime/compose/compose_test.go:TestRenderSelectedClosureAndPolicy; cli/integration_test.go:TestIntegrationConcurrentLeasesClosureAndEvidence`; coverage F.

### HM05 — M

Symlink/volume-driver indirection bypassed host-mount policy; inspect indirection before permitting effects.

- Detected/earliest: S8/S2; escape: `NEGATIVE_FIXTURE_GAP`.
- Production: `policy/policy.go`.
- Current regression: `policy/escape_test.go:TestBindSymlinkAndVolumeDriverCannotEscapePolicy`; coverage H.

### HM06 — M

Reconcile could promote leases without completed initial readiness. Nearby test mainly covers missing resources/released leftovers; exact original regression remains untraced.

- Detected/earliest: S8/S3; escape: `INVARIANT_GAP`.
- Production: `app/lifecycle.go:Reconcile`.
- Current regression: `app/lifecycle_test.go:TestLifecycleReconcileMissingAndReleasedLeftovers (nearby, not exact original mapping)`; coverage U.

### HM07 — M

Literal secrets needed rejection but static schema names falsely matched credentials; inspect actual data values/dynamic keys instead of serialized schema text.

- Detected/earliest: S8+S5/S2; escape: `ORACLE_COUPLING,NEGATIVE_FIXTURE_GAP`.
- Production: `app/plan.go; evidence/structured.go`.
- Current regression: `app/lifecycle_test.go:TestLifecycleRejectsSecretSnapshots; app/manifest_credentials_test.go:TestInheritedCredentialDoesNotMatchManifestSchema; evidence/structured_test.go`; coverage F.

### HM08 — M

Owner/defaults, manifest provenance, exit behavior and scoped logs required full CLI checks; aggregate logs cannot claim component isolation. Source groups these corrections without individual thread IDs.

- Detected/earliest: S8/S4; escape: `HELPER_ONLY,NEGATIVE_FIXTURE_GAP`.
- Production: `cli/root.go; app/lifecycle.go; source/gitcli/origin.go`.
- Current regression: `cli/diagnostics_test.go; cli/source_origin_test.go:TestPlanManifestOriginIndependentOfRuntimeRef; cli/logs_test.go:TestArchivedComponentLogsThroughCLI; TestLegacyAggregateCannotPretendComponentIsolation`; coverage F.

### HM09 — M

Command completion left descendants able to write later; reap owned tree before returning or removing sources.

- Detected/earliest: S3/S3; escape: `COMPOSITION_GAP`.
- Production: `execx/process_unix.go; process_windows.go`.
- Current regression: `execx/process_tree_test.go:TestRunnerReapsOrdinaryDescendants; app/cancellation_test.go:TestDestroyCancelsActualCommandTreeBeforeSourceCleanup`; coverage N.

### HM10 — M

P1: terminal run preceded successful artifacts, letting force destroy delete evidence; retain running barrier on every finalization failure.

- Detected/earliest: S8/S4; escape: `FAILURE_INJECTION_GAP`.
- Production: `app/commands.go:finalization`.
- Current regression: `app/cancellation_test.go:TestDestroyRetainsSourceWhenCancellationFinalizationFails`; coverage F.

### HM11 — M

P1: cancellation acknowledgement hid unconfirmed tree termination/incomplete output; typed uncertainty must block terminal publication and cleanup.

- Detected/earliest: S8/S4; escape: `FAILURE_INJECTION_GAP`.
- Production: `app/commands.go; execx/runner.go`.
- Current regression: `app/cancellation_test.go:TestDestroyRetainsSourceWhenCancellationFinalizationFails; execx/runner_test.go:TestOutputWriteFailureIsIncompleteEvidence`; coverage F.

### HM12 — M

Concurrent cold SQLite initialization returned BUSY; retry bounded initialization contention only, not operational/invariant failures.

- Detected/earliest: S4/S3; escape: `CONCURRENCY_GAP`.
- Production: `store/sqlite/store.go:initializeWithRetry`.
- Current regression: `store/sqlite/initialization_test.go:TestConcurrentColdOpen; TestConcurrentColdOpenProcesses; TestInitializationRetriesOnlyBoundedBusyContention; TestInitializationDoesNotRetryInvariantFailure`; coverage F.

### HM13 — M

Native filepath.Rel separators entered portable artifact API; convert at boundary and verify retained artifacts natively.

- Detected/earliest: S5/S2; escape: `NATIVE_EVIDENCE_GAP`.
- Production: `app/commands.go:artifact relative paths`.
- Current regression: `app/commands_test.go:TestNamedCommandPersistsRedactedEvidence; app/state_root_test.go:TestNamedCommandEvidenceStaysUnderStateRoot`; coverage F.

### HM14 — M

Fixed sleep crossed conservative renewal watchdog in a success test; observe durable renewal from another connection instead of assuming sleep proves ownership.

- Detected/earliest: S5/S3; escape: `ORACLE_COUPLING,CONCURRENCY_GAP`.
- Production: `store/sqlite/store_test.go:renewal fixture`.
- Current regression: `store/sqlite/store_test.go:TestOperationLockAcrossConnectionsAndExpiry`; coverage F.

### HM15 — M

macOS transient zombie groups returned EPERM after Wait; retry only bounded Darwin EPERM and still require success/ESRCH, retaining barriers for persistent failure.

- Detected/earliest: S5/S5; escape: `NATIVE_EVIDENCE_GAP,CONCURRENCY_GAP`.
- Production: `execx/process_unix.go:terminateProcessGroup`.
- Current regression: `execx/process_unix_test.go:TestTerminateProcessGroupWaitsForZombieReaping; TestTerminateProcessGroupPreservesFailures; app/cancellation_test.go`; coverage N.

## Escape analysis shared by each classified row

Each row inherits the corresponding concrete detection opportunity and explanation
below; this avoids treating “missing tests” as an explanation. These are
retrospective hypotheses unless the source explicitly records the failed fixture.

| Category | Opportunity before discovery / why it escaped | Earlier preventive control and status |
| --- | --- | --- |
| BOUNDARY_GAP | Numeric caps and durations already existed, but ordinary/overflow fixtures omitted exact capacity, metadata totals, setup time or post-transform bytes. | Existing row regressions exercise those boundaries. Shared 0/1/limit-1/limit/limit+1 and final-encoded-output matrices remain a Phase A follow-up; local fixes are not a repository-wide guardrail. |
| NEGATIVE_FIXTURE_GAP | Public validation/ownership requirements allowed independent invalid inputs, but happy-path fixtures did not mutate each requirement. | Existing independent invalid-input fixtures; expand across entry points where H/U remains. Expected detection S2 or S3. |
| ORACLE_COUPLING | Mocks or expected output shared implementation assumptions: URL shape, empty selection, broad Unicode secret match, sleeping ownership, early-refusal success. | Existing native protocol fixtures and injection-reached/artifact-found assertions. General guardrail: a negative test must demonstrate entry into the intended branch, plus a nearby positive case. Expected S2–S5 by interface. |
| COMPOSITION_GAP | Success at one layer did not prove readiness after process exit, alias precedence, effect ordering, or budgets after redaction. Those interfaces existed before review. | Existing full app/store/effect fixtures and persisted-artifact checks. A shared effect-stage/final-encoding matrix is pending disposition; helper tests alone are insufficient. Expected S3/S4. |
| FAILURE_INJECTION_GAP | Persist intent/effect/result stages were documented, but store write, receipt, lock-loss, post-input and output-finalization failures were not injected at each transition. | Existing row-specific injections and durable running-barrier assertions. Cross-runtime failure-stage matrix is still required by the audit. Expected S3/S4. |
| NATIVE_EVIDENCE_GAP | Cross-builds/fakes cannot model Job completion/PID reuse, Darwin zombies, Chrome realms/accelerators, sandbox ACLs or native separators. | Existing Windows/macOS/Linux CI and real process/browser fixtures. Native prerequisites are explicit; unavailable platforms must remain unverified. Expected S5 where OS semantics are essential. |
| CONCURRENCY_GAP | One connection/one sequential process hid cold initialization, queue completion, renewal and identity-observation windows. | Existing separate SQLite connections/processes, deterministic mutation points and native identity negatives. Broader cross-runtime race replay remains pending. Expected S3–S5. |
| HELPER_ONLY | Correct inspection/validation helper was assumed to guarantee following formatting, persistence or routing; public caller was not the oracle. | HP10 now uses Destroy. HP05/HB19 remain helper-only for their specific negative invariant. Add a public-call injection assertion if disposition accepts that coverage work. |
| INVARIANT_GAP | “No target seen”/“resources running” lacked explicit positive proof of complete absence/initial readiness. | Existing gone/truncated regression; trace HM06 before claiming closure. Positive-proof decision tables remain a Phase A action. |
| REVIEW_CHECKLIST_GAP | Working documentation and CI existed, but stale prose/old green revisions were not reconciled with final acceptance. | Existing paired docs/hash checks do not prove semantic truth; final-revision acceptance review remains manual. A fully automatic CI-to-plan truth checker is impractical without a declared evidence model, so no such guard is claimed. |

## Current coverage limits and same-pattern search handoff

The following are confirmed coverage limitations or investigation leads, not
confirmed new product defects:

1. HP05 invokes `validateProcessSecrets` directly. `BuildPlan` currently calls it
   before digest/source resolution, and the neighboring `TestPlanProcessPinsSourceWithoutRuntimeEffects`
   tests full planning with runtime environment literals, but that is not the same
   readiness-literal injection. A full readiness-negative plan/create test could
   detect future caller bypass.
2. HB19 invokes `selectBrowser` directly. `Browser` currently calls it before
   provider inspection/run intent, but the regression does not mutate persisted
   manifest through the actual app entry point and assert zero provider calls.
3. HB18 directly tests `finishCapture` and subscription state. The live capture
   loop currently delegates deadline completion to it; the helper regression alone
   would not detect a future alternate return path. Separate enable-deadline and
   subscribed-overflow protocol tests cover adjacent behavior, not that exact race.
4. HM06's initial-readiness historical regression was not established by the
   bounded search. `TestLifecycleReconcileMissingAndReleasedLeftovers` is not an
   adequate substitute without checking its setup. Do not mark this mapping complete.
5. Recurrence targets found by scoped search: cancellation/evidence barriers in
   `app/commands.go`, `app/browser.go`, `app/ui.go`; incomplete snapshot/gone
   logic in browser and Android UI; post-redaction bounds in browser, process
   logs and shared evidence; process/Android destroy previews; source/Android/process
   canonical paths; independent-resource cleanup and mixed readiness. These are
   search destinations, not claims of current defects or completed broad review.
6. HB25 now asserts its ownership mutation callback ran for all six variants.
   HB16/HB20 assert registered artifacts actually exist and inspect their bytes;
   HB20 also verifies persisted passed run status and refusal of truncated input.
   These are concrete guards against the earlier vacuous-success methodology.

Guardrail status for all rows: existing local regression/process control, with
broader preventive controls explicitly deferred to audit disposition. This annex
added no new test guardrails. No current finding is ACCEPT/REJECT/DEFER here.

## Validation evidence and limits

On the frozen target, selected historical tests ran with `go test -race` and
`-count=1`: browser/cdp PASS 7.293s; runtime/process PASS 1.092s;
store/sqlite PASS 10.883s; app PASS 11.129s; execx PASS 1.115s.
The selector covered browser, capture, frame/DOM/gone/stale/focus/transport,
process readiness/no-spawn/preview/alias/readiness-secret and reservation guards,
receipt/redaction, cancellation-finalization, cold initialization and
process-group termination tests. Not every test named above was selected;
in particular build-tagged Windows/native CLI/Docker tests were source-inspected,
not executed by this replay. Windows/macOS evidence in historical plans is
historical, not current audit native evidence. The coordinator owns full baseline
and native/integration reruns. No original-defect mutation replays were performed.
