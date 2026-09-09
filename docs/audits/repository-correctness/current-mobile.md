---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Current mobile correctness audit

[日本語](current-mobile.ja.md) · [Audit index](index.md) · [Historical mobile corpus](history-mobile.md) · [Execution authority](../../exec-plans/active/repository-correctness-audit.md)

Phase A, review only. Target `031869c8b9073b8e23bc17fbc55243666a52f557`; branch `audit/repository-correctness`. No product/test changes, commits, or remote actions. New tests run only through temporary Go overlays. Product contracts: [Android](../../product-specs/android-emulator.md), [Flutter](../../product-specs/flutter-android-runtime.md), [UI](../../product-specs/android-ui-observer.md). Baseline/native matrix belongs to the root audit and is not duplicated here.

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

## Reviewed invariant matrix

`No new finding` means the inspected source and scoped existing assertions agree; it is not exhaustive proof. All current findings remain untriaged until Phase B. Existing regression execution is recorded in the historical corpus.

| Area / production boundary | Invariant checked | Evidence and result |
| --- | --- | --- |
| Android `Validate` / app allocation | Tools, acceleration, ABI, shared ADB and path aliases fail before source/resource writes | `app_test.go` preflight fixtures plus CLI `create_store_test.go`; no new finding. Factory regression does not exercise Cobra dispatch. |
| Android `Inspect` / `Destroy` | Marker, lease, tuple, PID/Job, console token and live port must agree before kill/delete | `lifecycle.go` marker/process/console checks and stale-marker/reused-port tests; no new finding. Native Windows second-read race test gap M12 remains. |
| Android process/port cleanup | Root absence does not imply tree absence; authenticated kill waits for group and ports | `DestroyWaitsForPostKillTreeConfirmation`, cancellation tests and containment source; no new finding. Native OS race proof is separate. |
| Shared ADB | Inspect is noncreating; incompatible daemon is not replaced; startup outside lease containment | `adb.go` smart-socket preflight and matching negative fixtures; no new finding. |
| Mixed app lifecycle | Independent readiness deadlines, sibling cleanup on local failure, stop on lost global fence | `mixed_readiness_test.go`, `cleanup_siblings_test.go`, actual `waitReady`/`cleanup`; no new finding. |
| Flutter build paths | Explicit executable, project/ancestor symlink refusal, APK regularity, source retention on uncertain termination | `flutter.go` Validate/Build and app build guards; no new finding. External APK execution/real toolchain remains native evidence. |
| Application desired state | Confirmed install/reverse/launch differs from intent; executable and directory retained for reconcile | `applicationConsistent` and complete-field mutation tests; no new finding. |
| Application cleanup | Current reverse endpoint matches recorded ownership before removal; build/fence failures retain resources | `cleanupApplications`, reverse-preservation and build-barrier/preview tests; no new finding. |
| UI intent/recovery | Durable started intent and positive recovery classification precede helper-only stop; no replay; host errors stay blocked | `ui.go`, `ui_recovery.go`, `ui_recovery_test.go`; supplementary PR5 trace confirms AUDIT-LIFECYCLE-001 and AUDIT-IDENTITY-001. |
| UI stale input | Snapshot digest, lease/runtime/package/backend, unique fingerprint and completeness precede action | `loadUISnapshot`, Service selection, Java helper current-tree checks and app stale tests; supplementary PR5 trace confirms AUDIT-UI-001 and AUDIT-IDENTITY-001. Atomic concurrent UI changes are not claimed impossible. |
| UI Java producer | Incomplete hierarchy and action readback must have explicit evidence | `Observer.java` implementation present; producer assertion gaps M48/M49, not confirmed current defects. |
| UI log provider | Attribution/time window and true truncation, independently of cap equality | Current `AUDIT-BOUNDARY-001`; concrete adapter overlay fails exact 2000 short lines. |
| UI post-redaction publication | Final field and normalized-document limits apply after escaping/metadata/redaction | Current `AUDIT-REDACTION-001`; app entry overlay persists oversized untruncated evidence. |
| UI evidence persistence | Failed artifact/metadata writes keep mutation cleanup barrier; invalid completed capture can finish failed without a false running barrier | `TestUIEvidenceFailureKeepsCleanupBarrier`, `TestUIInvalidCompletedCaptureFinalizesFailed`; no new finding in the inspected cases. |

## AUDIT-REDACTION-001 — Android UI publication exceeds its post-redaction bounds

Severity: **Medium** (deterministic published boundary violation). Disposition: **untriaged**. This finding concerns size/completeness; the reproducer does not expose the configured secret and is not labeled a secret leak.

### Invariant, location, trigger, and impact

[UI contract](../../product-specs/android-ui-observer.md) requires at most 4096 characters per field and 1 MiB normalized evidence. `internal/app/ui.go:434` sanitizes the observation; lines 458–460 replace secret matches in text/description/hint without reapplying field limits. The snapshot publication block at line 350 marshals the post-redaction snapshot and rejects only above `2<<20`, so evidence between 1 MiB and 2 MiB is registered as successful, complete evidence. The helper's pre-redaction budget cannot guarantee a post-redaction budget.

Use configured secret `qz` and ordinary display text containing repeated `qz/`. Replacement expands two characters to `[REDACTED]`; the original text remains below the provider's budget. No huge single string or invalid provider response is needed.

The existing `uiFixture` is used with its normal Service/store/runtime setup and a provider returning successful snapshots. The overlay invokes `Service.UI(snapshot)` and reads the actual registered `ui-snapshot` artifact. Two independent cases:

| Case | Provider response | Observed publication | Expected |
| --- | --- | --- | --- |
| field | One node, text/description each `qz/` repeated 500 times; JSON 3612 bytes | Text becomes 5500 characters; successful snapshot | Final fields at most 4096, and any shortening honestly marked |
| aggregate | 550 nodes, each text/description `qz/` repeated 100 times; complete provider JSON 584693 bytes, below the Java 700000-byte guard | Registered snapshot 1429604 bytes; `Tree.Truncated=false` | Normalized artifact at most 1048576 bytes with truthful truncation, or explicit bounded refusal |

The temporary test failed as expected in 0.038s. Node count is below 1000; original fields below 4096; initial JSON is below the provider budget. Thus neither case depends on accepting already-invalid provider data. The fixture includes ordinary stable node refs/fingerprints; it invokes the public app use case, but not real Java/ADB. The root's native run is separate evidence.

### Reproduction and existing coverage

Temporary overlay test: `TestAuditUIRedactionBounds`, package `internal/app`. Reproduce without editing tracked sources by mapping an added package test file with Go `-overlay`, setting `AUDIT_PRIVATE_SECRET=qz`, using `uiFixture`, and returning the node counts/strings above. Assert rune count after `Service.UI`; read every registered `ui-snapshot` artifact and assert raw byte length at most `1<<20`. Local overlay path was communicated to the audit owner, not made a durable machine-specific repository dependency.

Existing `TestUIEvidenceRedactsEnteredAndEditableText` checks secret replacement/suppression, `TestUIEditableDescriptionAndHintRedacted` checks all editable fields, and `TestUIInvalidCompletedCaptureFinalizesFailed` checks a clearly oversized input. None independently combine a valid producer budget with replacement expansion and final serialized-byte measurement. The existing helper limits likewise run before redaction. No production repair or permanent regression has been added in Phase A.

### Escape and same-pattern analysis

Detected S9; earliest realistic S2. Categories: `BOUNDARY_GAP`, `COMPOSITION_GAP`, `ORACLE_COUPLING`. S2 could have varied replacement ratio as an independent boundary dimension. S3 checked redaction and bounds independently; S4 accepted a synthetic provider result without measuring the final persisted bytes after transformation. S5 native normal text does not reach the expansion boundary; S6 has no semantic field/artifact size validator. S7/S8 had an opportunity to compare 1 MiB contract with the app's 2 MiB guard and to require one post-transformation oracle. There is no evidence those checks were performed for this path.

Historical relation: M46's adapter/app evidence composition and M49's completeness producer gap. Cross-domain relation: Browser PR10 post-redaction bounds. Preventive candidates, pending disposition: a shared final-publication field/serialized-byte checker, limit−1/exact/+1 cases after secret expansion and JSON escaping, real artifact byte assertions, and retained node-identity/truncation assertions. Expected detection S2/S4. Do not silently raise the documented limit or drop evidence without a truncation signal to make a test pass.

## Other finding and review limitations

[AUDIT-BOUNDARY-001](history-mobile.md) records the Android device-tail/helper ambiguity and its overlay evidence. The ambiguity is explicitly retained; a local `>=` edit alone is not a validated fix.

The inspected Java field limiter truncates a value before app redaction; secret-prefix handling there deserves further security review, but no additional current finding is asserted without a concrete end-to-end producer reproduction. Scope did not include malformed APK execution, hostile accessibility services, malicious-code sandboxing, or proving every OS syscall interleaving. All such omissions remain limitations, not passes.

## Current reproductions of the final PR5 findings

Completing the 24-item external corpus established that all six final PR5 findings remain in the frozen target. All are untriaged; no product repair or permanent regression has been added. Temporary overlay tests failed in app0.016s, Android0.007s, helper0.003s. The retained overlay was passed to the audit owner.

### AUDIT-UI-001

Severity: **High**. Source: [PR5 comment 3954080090](https://github.com/mahcialet/agent-env/pull/5#discussion_r3954080090), historical M72. Location: `internal/app/ui.go:461`. Reproducer: `TestAuditEditableSnapshotRemainsActionable`.

Automatically suppressed editable fields must preserve safe semantic fingerprints. A helper returns an editable node with empty text/description/hint and a valid stable fingerprint. The sanitizer changes these empty values to `[REDACTED]`, treats that change as a configured-secret match and clears every fingerprint. The public Service snapshot succeeds, but a subsequent set-text on its retained node fails `AGENTENV-UI-STALE: missing or ambiguous snapshot node` before invoking the provider. The overlay reproduced both the empty stored fingerprint and rejected app call. The audit owner additionally ran the unchanged real UI fixture from a clean frozen clone on a volume with adequate disk space: both leases reached READY, then the first tap failed with the same missing/ambiguous snapshot-node error (75.11s). The real Flutter fixture passed separately (73.54s). This is native corroboration of the snapshot action failure, not a passing UI baseline.

Existing `TestUIEvidenceRedactsEnteredAndEditableText` is vacuous for its intended error-redaction path: it installs a callback checking entered text, but does not assert that callback ran. The stale rejection is also a non-secret error, satisfying the test. Detected S9, earliest S3; `ORACLE_COUPLING`, `FAILURE_INJECTION_GAP`, `COMPOSITION_GAP`. Preventive control: snapshot-to-set-text composed success with callback count, plus separate real configured-secret fingerprint refusal. Proposed regression must distinguish suppression from secret-derived identity, not restore hashes indiscriminately.

### AUDIT-IDENTITY-001

Severity: **High**. Source: [PR5 comment 3954080094](https://github.com/mahcialet/agent-env/pull/5#discussion_r3954080094), historical M73. Location: `internal/runtime/android/ui.go:72`. Reproducer: `TestAuditHelperBackendCarriesVerifiedDigest`.

Semantic snapshot/action/recovery authority must bind the verified source/APK digests. ObserveUI initializes Backend to `android-shell-v1`, then assigns helper-specific identity only if Backend is empty. This condition cannot hold on the normal helper branch. An overlay with a valid verified helper and its exact expected digest observed generic `android-shell-v1`; the normal backend cannot distinguish helper builds. Recovery fallback requires `uiautomation-v1:source=...:apk=...`, so evidence from this branch cannot support reconstruction when the configured helper directory disappears.

Existing `TestUISemanticActionRejectsChangedBackendBeforeDeviceInput` uses an arbitrary different string and therefore passes with either a correct digest or the wrong generic backend. Detected S9, earliest S3; `ORACLE_COUPLING`, `INVARIANT_GAP`, `COMPOSITION_GAP`. Preventive controls: assert exact verified backend for snapshot, acceptance of same build, refusal of a different build, and recovery with host files absent using recorded identity. No fallback to unverified identity is acceptable.

### AUDIT-REDACTION-003

Severity: **High**. Source: [PR5 comment 3954080097](https://github.com/mahcialet/agent-env/pull/5#discussion_r3954080097), historical M74. Location: `internal/app/ui.go:447`. Reproducer: `TestAuditWindowSecretClearsDerivedNodeHashes`.

Secret-derived identifiers must not remain offline verification oracles. A window title containing a configured secret is redacted and its Key cleared, but only nodeSecret triggers clearing node Fingerprints. The Java producer includes the window key in root ancestry before hashing node identity. An app-entry overlay confirmed that the window key is removed while the node fingerprint remains in a registered snapshot. Source inspection establishes the nested dependency on secret-bearing window data; the overlay uses a representative hash rather than claiming native password cracking.

Existing redaction tests search for plaintext and do not check derived identifiers; no direct nested-window-hash regression was located. Detected S9, earliest S2; `INVARIANT_GAP`, `NEGATIVE_FIXTURE_GAP`, `ORACLE_COUPLING`. Preventive control: construct a real producer-equivalent fingerprint from window metadata, redact a configured title/root secret and verify all derived keys/hashes are absent from raw, normalized, result and run evidence. Preserve actionable fingerprints only for metadata proven unrelated to a secret.

### AUDIT-PREREQUISITE-001

Severity: **High**. Source: [PR5 comment 3954080101](https://github.com/mahcialet/agent-env/pull/5#discussion_r3954080101), historical M75. Location: `internal/runtime/android/uihelper/helper.go:56`. Reproducer: `TestAuditUnsetHelperRefusesCurrentDirectory`.

Companion installation requires an explicit configured directory. Load("") calls filepath.Abs, which resolves the current working directory. An isolated temporary working directory containing internally consistent observer metadata/APK was accepted with an empty argument. ObserveUI passes os.Getenv directly to Load, so an unset opt-in can select and install cwd files. The overlay exercises the real loader, not an actual device install; source traces the accepted metadata into verified-byte installation.

Existing provenance fixtures always pass an explicit directory; absence is not a mismatched digest. Detected S9, earliest S2; `NEGATIVE_FIXTURE_GAP`, `COMPOSITION_GAP`. Preventive control: loader empty/whitespace/path matrix plus app/adapter missing-environment test from a populated cwd, asserting no install. Reject absence before normalization and keep the recorded-identity-only recovery exception explicit.

### AUDIT-BOUNDARY-002

Severity: **Medium**. Source: [PR5 comment 3954080103](https://github.com/mahcialet/agent-env/pull/5#discussion_r3954080103), historical M76. Location: `internal/app/ui.go:190`. Reproducer: `TestAuditFractionalLogLookback`.

Accepted lookback duration must not silently exclude requested records. Service.UI accepts Since=1900ms but sets SinceSeconds by integer division; the concrete provider request becomes 1 second. The app-entry overlay inspected that request and failed. Logs between 1 and 1.9 seconds old are then excluded although requested. This is separate from exact record-count truncation.

Existing log fixtures use integral seconds. Detected S9, earliest S2; `BOUNDARY_GAP`, `NEGATIVE_FIXTURE_GAP`. Preventive control: duration−epsilon/exact/+epsilon around seconds and maximum duration, inspecting the actual provider request and cutoff behavior. Disposition must choose upward rounding or reject unsupported fractions consistently with the existing public contract; silently shortening is not valid.

### AUDIT-LIFECYCLE-001

Severity: **High**. Source: [PR5 comment 3954080105](https://github.com/mahcialet/agent-env/pull/5#discussion_r3954080105), historical M77/M65. Location: `internal/runtime/android/ui.go:211`. Reproducer: `TestAuditNativePreflightRemainsConfirmed`.

Failed ownership/protocol preflight before native input must not imply uncertain dispatch. The native branch sets Confirmed=false before the combined preflight/input call; recognition of adbPreflightError prevents another false assignment but never restores the previous one. An adapter overlay changed the owned console name: it observed ErrResourceIdentity, zero runner commands and Confirmed=false. The app classifies this as termination-unconfirmed and retains a running cleanup barrier; RecoverUI excludes native Back/Home/tap/swipe. The adapter observation is executed evidence; the resulting app barrier is a source trace, not yet an additional composed replay.

Existing TestUIOwnershipFailureCannotDispatchInput discards the observation and asserts only error identity and zero input. The prior repair recognized a typed error without inspecting the final returned state. Detected S9, earliest S4; `FAILURE_INJECTION_GAP`, `COMPOSITION_GAP`, `ORACLE_COUPLING`. Preventive control: injected preflight failure through the app, durable failed run with no running barrier and successful later safe cleanup; separately retain uncertain status after actual input launch. A blanket Confirmed=true on all errors would weaken safety.
