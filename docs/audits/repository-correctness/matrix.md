---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Subsystem and invariant audit matrix

[日本語](matrix.ja.md) · [Audit index](index.md) · [Disposition ledger](findings.md)

This matrix synthesizes the bounded annexes against frozen Phase A revision
`031869c8b9073b8e23bc17fbc55243666a52f557`. It records coverage and finding
placement, not universal correctness or a substitute for candidate verification.
All 19 finding dispositions and final resolution status remain owned by the
ledger. Phase C mechanisms below are present in the working candidate; linked
annexes own fail-before/focused evidence, and the index/ExecPlan own final gates.
A historical or cached pass is not silently upgraded to fresh native evidence.

## Cell and source key

- `R(A)` etc.: bounded source/contract/regression review with limitations in
  the named annex. It does not mean every syscall schedule, numeric combination
  or native platform was exercised.
- `AUDIT-…`: reviewed cell with a confirmed finding at the frozen target.
  Repeating an ID across cells identifies one cross-boundary defect, not another
  finding; resolution is read from the ledger.
- `N-P`: no independent managed lifecycle/effect/cancellation responsibility
  in this pure schema/checker slice; its consumer is reviewed in the app/runtime row.
- `N-T`: no accessibility/page/node semantic-action or absence-wait surface.
  Other identity freshness is reviewed under identity/concurrency.
- `N-L`: no release-version/archive/publication responsibility. Being compiled
  into the executable does not itself create a release publisher.

A: [control plane](current-control-plane.md), including config/domain/store,
source/paths/evidence/readiness/reconcile/cleanup and its historical mappings.
B: [mobile](current-mobile.md) and [mobile history](history-mobile.md).
C: [process/browser/execx](current-process-browser.md) and
[history](history-process-browser.md), with the late
[CDP](supplemental-browser.md) and [CLI](supplemental-cli.md) supplements.
D: [Compose/release/assets](current-compose-release.md) and
[history](history-compose-release.md).
E: [documentation/harness](documentation.md).
All rows additionally inherit the relevant documented native/coverage limits;
a review mark cannot erase an annex's explicit missing oracle or unexecuted test.

## Matrix

The three panels use the same 20 subsystem rows and together cover all 14 core
invariant classes. Shared code is deliberately visible in multiple consumer rows.
For example, Docker/Podman share the inspected Compose container path, and the
readiness failure propagates through app state and later cleanup.

### Bounds, positive proof, transitions, persistence and identity

| Subsystem | Bounds | Positive proof | State transitions | Intent/effect/result persistence | Ownership/identity |
| --- | --- | --- | --- | --- | --- |
| Config/domain | R(A) | R(A) | R(A) | N-P | R(A) |
| App orchestration | R(A) | AUDIT-CLEANUP-001; AUDIT-CLI-001 | R(A) | AUDIT-CLEANUP-001 | R(A) |
| SQLite store | R(A) | R(A) | R(A) | R(A) | R(A) |
| Git sources/worktrees | R(A) | R(A) | R(A) | R(A) | R(A) |
| Docker Compose | R(D) | AUDIT-OWNERSHIP-001 | R(D) | R(D) | AUDIT-OWNERSHIP-001 |
| Podman Compose | R(D) | AUDIT-OWNERSHIP-001 | R(D) | R(D) | AUDIT-OWNERSHIP-001 |
| Android Emulator | R(B) | R(B) | R(B) | R(B) | R(B) |
| Flutter Android | R(B) | R(B) | R(B) | R(B) | R(B) |
| Android UI | AUDIT-BOUNDARY-001; AUDIT-BOUNDARY-002 | AUDIT-LIFECYCLE-001 | AUDIT-LIFECYCLE-001 | AUDIT-LIFECYCLE-001 | AUDIT-IDENTITY-001; AUDIT-PREREQUISITE-001 |
| Persistent process | R(C) | R(C) | R(C) | R(C) | R(C) |
| Browser/CDP | AUDIT-REDACTION-002; AUDIT-BOUNDARY-003 | AUDIT-STALE-001; AUDIT-STATE-001 | AUDIT-BOUNDARY-003 | R(C) | AUDIT-STALE-002 |
| execx | R(C) | R(C) | R(C) | R(C) | R(C) |
| Evidence/redaction | AUDIT-REDACTION-001; AUDIT-REDACTION-002 | R(A) | N-P | R(A) | R(A) |
| Paths/filesystems | R(A) | R(A) | N-P | R(A) | R(A) |
| Readiness/endpoints | R(A) | AUDIT-CLEANUP-001 | AUDIT-CLEANUP-001 | AUDIT-CLEANUP-001 | R(A) |
| Reconcile | R(A) | R(A) | R(A) | R(A) | R(A) |
| Destroy/GC/quarantine | R(A) | AUDIT-OWNERSHIP-001 | R(A) | R(A) | AUDIT-OWNERSHIP-001 |
| Release/buildinfo | AUDIT-RELEASE-001; AUDIT-RELEASE-002 | R(D) | R(D) | R(D) | R(D) |
| Embedded assets | R(D) | R(D) | R(D) | R(D) | R(D) |
| Docs/harness | R(E) | AUDIT-DOCS-001 | N-P | N-P | R(E) |

### Cleanup, cancellation, concurrency, stale semantics and evidence

| Subsystem | Cleanup proof | Cancel/deadline/fence | Concurrency/reservations | Stale semantic targets | Redaction/bounded evidence |
| --- | --- | --- | --- | --- | --- |
| Config/domain | N-P | N-P | R(A) | N-T | R(A) |
| App orchestration | AUDIT-CLEANUP-001 | AUDIT-CLEANUP-001; AUDIT-CLI-001 | R(A) | R(A) | R(A) |
| SQLite store | R(A) | R(A) | R(A) | N-T | R(A) |
| Git sources/worktrees | R(A) | R(A) | R(A) | N-T | R(A) |
| Docker Compose | AUDIT-OWNERSHIP-001 | R(D) | R(D) | N-T | R(D) |
| Podman Compose | AUDIT-OWNERSHIP-001 | R(D) | R(D) | N-T | R(D) |
| Android Emulator | R(B) | R(B) | R(B) | N-T | R(B) |
| Flutter Android | R(B) | R(B) | R(B) | N-T | R(B) |
| Android UI | AUDIT-LIFECYCLE-001 | R(B) | R(B) | AUDIT-UI-001; AUDIT-IDENTITY-001 | AUDIT-REDACTION-001; AUDIT-REDACTION-003 |
| Persistent process | R(C) | R(C) | R(C) | N-T | R(C) |
| Browser/CDP | R(C) | R(C) | AUDIT-STALE-001; AUDIT-STALE-002 | AUDIT-STALE-001; AUDIT-STALE-002; AUDIT-STATE-001 | AUDIT-REDACTION-002; AUDIT-CLI-001 |
| execx | R(C) | R(C) | R(C) | N-T | R(C) |
| Evidence/redaction | R(A) | R(A) | R(A) | N-T | AUDIT-REDACTION-003 |
| Paths/filesystems | R(A) | R(A) | R(A) | N-T | R(A) |
| Readiness/endpoints | AUDIT-CLEANUP-001 | AUDIT-CLEANUP-001 | R(A) | N-T | R(A) |
| Reconcile | R(A) | R(A) | R(A) | N-T | R(A) |
| Destroy/GC/quarantine | AUDIT-CLEANUP-001 | AUDIT-CLEANUP-001 | R(A) | N-T | R(A) |
| Release/buildinfo | R(D) | R(D) | AUDIT-RELEASE-002 | N-T | AUDIT-RELEASE-001 |
| Embedded assets | R(D) | R(D) | R(D) | N-T | R(D) |
| Docs/harness | N-P | N-P | N-P | N-T | R(E) |

### Paths, native semantics, release and documentation

| Subsystem | Paths/filesystems | Native portability | Release correctness | Docs claims/enforcement |
| --- | --- | --- | --- | --- |
| Config/domain | R(A) | R(A) | N-L | R(A) |
| App orchestration | R(A) | R(A) | N-L | R(A) |
| SQLite store | R(A) | R(A) | N-L | R(A) |
| Git sources/worktrees | R(A) | R(A) | N-L | R(A) |
| Docker Compose | R(D) | R(D) | N-L | R(D) |
| Podman Compose | R(D) | R(D) | N-L | R(D) |
| Android Emulator | R(B) | R(B) | N-L | R(B) |
| Flutter Android | R(B) | R(B) | N-L | R(B) |
| Android UI | AUDIT-PREREQUISITE-001 | R(B) | N-L | R(B) |
| Persistent process | R(C) | R(C) | N-L | R(C) |
| Browser/CDP | R(C) | R(C) | N-L | R(C) |
| execx | R(C) | R(C) | N-L | R(C) |
| Evidence/redaction | R(A) | R(A) | N-L | R(A) |
| Paths/filesystems | R(A) | R(A) | N-L | R(A) |
| Readiness/endpoints | R(A) | R(A) | N-L | R(A) |
| Reconcile | R(A) | R(A) | N-L | R(A) |
| Destroy/GC/quarantine | R(A) | R(A) | N-L | R(A) |
| Release/buildinfo | AUDIT-RELEASE-001; AUDIT-RELEASE-002 | R(D) | AUDIT-RELEASE-001; AUDIT-RELEASE-002 | R(D) |
| Embedded assets | R(D) | R(D) | R(D) | R(D) |
| Docs/harness | R(E) | R(E) | R(E) | AUDIT-DOCS-001 |

## Current finding escape and preventive-control map

All 19 were reproduced at S9 and ACCEPTed at initial/supplemental Phase B.
The initial 15 were first discovered by this audit at S9; the extra four were
originally external S8 comments, then independently reproduced at S9. Earliest realistic stages
below agree with the individual reports: S2 = 9, S3 = 7, S4 = 3. Ownership's
missing-field oracle is S2; proving the public Down barrier is S3. No S0/S1
requirement absence is invented where an enforceable contract already existed.
The expected future stage is where the added test/check can fire, not a promise
that every variant of its class is now detected. Test names identify actual
candidate tests; their execution evidence lives in the owning annex.

### AUDIT-CLEANUP-001

- Detected/earliest: S9/S4; categories: `COMPOSITION_GAP,FAILURE_INJECTION_GAP,HELPER_ONLY,REVIEW_CHECKLIST_GAP`.
- Earlier opportunity and escape: Typed executor safety existed; readiness caller retried/stringified it. The new composed test checks later Destroy, not just the immediate error.
- Implemented control and verification test: `readinessCommand -> runWithCancellation + durable CommandRun; TestReadinessRetainsUnconfirmedCommand, TestReadinessOrdinaryFailureMayRetryAndRelease, TestReviewReadinessCancellationDoesNotRetry`.
- Expected future detection: S3/S4; candidate evidence/resolution: owning annex and ledger.

### AUDIT-OWNERSHIP-001

- Detected/earliest: S9/S2; categories: `NEGATIVE_FIXTURE_GAP,ORACLE_COUPLING,COMPOSITION_GAP`.
- Earlier opportunity and escape: Label-positive fixtures always supplied IDs; missing field must be independent of labels and the destructive entry point must remain uncalled.
- Implemented control and verification test: `dockerClient.Inspect shared by provider Down; TestMissingContainerIdentityRefusesInspectionAndDown`.
- Expected future detection: S2/S3; candidate evidence/resolution: owning annex and ledger.

### AUDIT-DOCS-001

- Detected/earliest: S9/S2; categories: `COMPOSITION_GAP,NEGATIVE_FIXTURE_GAP,REVIEW_CHECKLIST_GAP`.
- Earlier opportunity and escape: Existing prose filtering protected links/required headings but not the other target-heading consumer.
- Implemented control and verification test: `documentProse reused in fragment target scan; TestFragmentsRequireRenderedTargetHeadings`.
- Expected future detection: S2/S6; candidate evidence/resolution: owning annex and ledger.

### AUDIT-RELEASE-001

- Detected/earliest: S9/S2; categories: `BOUNDARY_GAP,NEGATIVE_FIXTURE_GAP`.
- Earlier opportunity and escape: All root examples were longer than an early length exemption; short valid roots expose it while true volume roots retain explicit policy.
- Implemented control and verification test: `structural root classification; TestReleaseBinaryShortCheckoutPaths`.
- Expected future detection: S2; candidate evidence/resolution: owning annex and ledger.

### AUDIT-RELEASE-002

- Detected/earliest: S9/S2; categories: `CONCURRENCY_GAP,BOUNDARY_GAP,FAILURE_INJECTION_GAP`.
- Earlier opportunity and escape: Stable oversized files missed growth after stat; the asset fix had not propagated to release readers/writer.
- Implemented control and verification test: `bounded opened-file reader shared by check/copy/compare and writer; TestReleaseReadBoundsGrowthAfterOpenedStat, TestReleaseRegularReadExactBounds`.
- Expected future detection: S2/S3; candidate evidence/resolution: owning annex and ledger.

### AUDIT-BOUNDARY-001

- Detected/earliest: S9/S2; categories: `BOUNDARY_GAP,ORACLE_COUPLING`.
- Earlier opportunity and escape: Prior count-overflow fixture also exceeded byte limit. Exact count needs independent overflow evidence, not >= alone.
- Implemented control and verification test: `2001-record provider probe -> boundedUILog; TestUILogExactTailUsesOverflowProof`.
- Expected future detection: S2/S3; candidate evidence/resolution: owning annex and ledger.

### AUDIT-REDACTION-001

- Detected/earliest: S9/S2; categories: `BOUNDARY_GAP,COMPOSITION_GAP,ORACLE_COUPLING`.
- Earlier opportunity and escape: Producer caps and redaction passed separately; final encoded bytes and expanded fields lacked a composed oracle.
- Implemented control and verification test: `boundUIEvidence reused for raw/normalized/result envelopes; TestUIAuditUIRedactionBounds, TestUIFieldBoundaryAndSerializedObservationBoundary`.
- Expected future detection: S2/S4; candidate evidence/resolution: owning annex and ledger.

### AUDIT-REDACTION-002

- Detected/earliest: S9/S4; categories: `COMPOSITION_GAP,BOUNDARY_GAP`.
- Earlier opportunity and escape: Semantic/capture final caps omitted the sibling DOM artifact kind. Read actual published bytes and retained identities.
- Implemented control and verification test: `boundBrowserDOM before artifact publication; TestBrowserDOMBoundsAfterRedaction, TestBrowserDOMEncodedBoundary`.
- Expected future detection: S4; candidate evidence/resolution: owning annex and ledger.

### AUDIT-STALE-001

- Detected/earliest: S9/S3; categories: `CONCURRENCY_GAP,COMPOSITION_GAP`.
- Earlier opportunity and escape: Snapshot consistency was assumed to cover the later predicate call. Injection must occur between those two stages.
- Implemented control and verification test: `load predicate post-evaluation frameDocument check; TestLoadWaitRechecksDocumentAfterPredicate`.
- Expected future detection: S3; candidate evidence/resolution: owning annex and ledger.

### AUDIT-UI-001

- Detected/earliest: S9/S3; categories: `ORACLE_COUPLING,FAILURE_INJECTION_GAP,COMPOSITION_GAP`.
- Earlier opportunity and escape: Error-only assertion passed on premature stale refusal without reaching input callback. Composed positive asserts callback and durable success.
- Implemented control and verification test: `editable suppression distinct from secret-derived hash invalidation; TestUIAuditEditableSnapshotRemainsActionable`.
- Expected future detection: S3/S4; candidate evidence/resolution: owning annex and ledger.

### AUDIT-IDENTITY-001

- Detected/earliest: S9/S3; categories: `ORACLE_COUPLING,INVARIANT_GAP,COMPOSITION_GAP`.
- Earlier opportunity and escape: An arbitrary different backend is refused by both correct digest and wrong generic string; exact same-build positive was missing.
- Implemented control and verification test: `exact verified backend + recorded-identity recovery; TestUIAuditHelperBackendCarriesVerifiedDigest, TestUIRecoveryUsesRecordedHelperWithAbsentOrReplacedHostFiles`.
- Expected future detection: S3/S4; candidate evidence/resolution: owning annex and ledger.

### AUDIT-REDACTION-003

- Detected/earliest: S9/S2; categories: `INVARIANT_GAP,NEGATIVE_FIXTURE_GAP,ORACLE_COUPLING`.
- Earlier opportunity and escape: Plaintext searches did not inspect identifiers derived from secret-bearing window ancestry.
- Implemented control and verification test: `window-secret invalidation clears dependent node hashes; TestUIAuditWindowSecretClearsDerivedNodeHashes`.
- Expected future detection: S2/S4; candidate evidence/resolution: owning annex and ledger.

### AUDIT-PREREQUISITE-001

- Detected/earliest: S9/S2; categories: `NEGATIVE_FIXTURE_GAP,COMPOSITION_GAP`.
- Earlier opportunity and escape: Every valid/invalid-provenance fixture passed an explicit directory; absent opt-in with valid cwd files was distinct.
- Implemented control and verification test: `uihelper.Load explicit nonblank directory before Abs; TestUIAuditUnsetHelperRefusesCurrentDirectory`.
- Expected future detection: S2/S3; candidate evidence/resolution: owning annex and ledger.

### AUDIT-BOUNDARY-002

- Detected/earliest: S9/S2; categories: `BOUNDARY_GAP,NEGATIVE_FIXTURE_GAP`.
- Earlier opportunity and escape: Integral duration examples never reached lossy integer conversion; reject unsupported fractions rather than silently shortening.
- Implemented control and verification test: `whole-second UI lookback validation before store/provider; TestUIAuditFractionalLogLookback`.
- Expected future detection: S2; candidate evidence/resolution: owning annex and ledger.

### AUDIT-LIFECYCLE-001

- Detected/earliest: S9/S4; categories: `FAILURE_INJECTION_GAP,COMPOSITION_GAP,ORACLE_COUPLING`.
- Earlier opportunity and escape: Error/zero-input checks discarded the returned confirmation bit; public run/cleanup status must distinguish preflight and launched uncertainty.
- Implemented control and verification test: `effect-aware ADB invocation preserves preflight certainty; TestUIAuditNativePreflightRemainsConfirmed, TestUINativePreflightThroughAppDoesNotBlockCleanup, TestUINativeDispatchedFailureRemainsUnconfirmed`.
- Expected future detection: S3/S4; candidate evidence/resolution: owning annex and ledger.


### AUDIT-BOUNDARY-003

- Detected/earliest: historical S8, audit S9/S3; categories:
  `COMPOSITION_GAP,BOUNDARY_GAP,NEGATIVE_FIXTURE_GAP`.
- Earlier opportunity and escape: listing at capacity was tested, but creating
  one more page after the same census was not. The producer crossed its consumer's limit.
- Implemented control and verification test: shared page cap before creation;
  `TestSupplementPageLimitPreventsIrrecoverableGrowth` covers 127/128/129,
  effect counts and subsequent close.
- Expected future detection: S3; [supplement](supplemental-browser.md) records
  frozen/candidate failures and focused plus independent race evidence.

### AUDIT-CLI-001

- Detected/earliest: historical S8, audit S9/S3; categories:
  `COMPOSITION_GAP,NEGATIVE_FIXTURE_GAP,ORACLE_COUPLING`.
- Earlier opportunity and escape: the adapter returned the created ID/closed
  result correctly while ordinary text output discarded it. Tests of JSON could
  not prove the human-facing command result.
- Implemented control and verification test: `internal/cli/browser.go` inline RunE
  operation branches; `TestBrowserNativeCLI/page_mutation_table_identifies_affected_page`
  verifies the actual created target and subsequent absent closed target.
- Expected future detection: S3/S5; [CLI supplement](supplemental-cli.md)
  records 10.500s native race and fail-before public text assertions.

### AUDIT-STATE-001

- Detected/earliest: historical S8, audit S9/S3; categories:
  `COMPOSITION_GAP,INVARIANT_GAP,NEGATIVE_FIXTURE_GAP`.
- Earlier opportunity and escape: snapshot preserved Ignored and input rejected
  it, but the separate wait consumer counted it as accessible.
- Implemented control and verification test: ignored-node predicate filter;
  `TestSupplementIgnoredAXCannotSatisfyWait` verifies both text/gone polarity
  and AX-query reachability; truncated-gone guard retained.
- Expected future detection: S3; supplemental evidence is linked above.

### AUDIT-STALE-002

- Detected/earliest: historical S8, audit S9/S3; categories:
  `NEGATIVE_FIXTURE_GAP,ORACLE_COUPLING,COMPOSITION_GAP`.
- Earlier opportunity and escape: Tests rejecting stale input after checked-state changes did not prove the
  distinct pressed field was represented in the fingerprint.
- Implemented control and verification test: pressed enters existing bounded axState;
  `TestSupplementPressedStateRefusesStaleInput` covers boolean/mixed and all
  eight previously retained flags, asserting mutation reached and zero input.
- Expected future detection: S3; no arbitrary state text is admitted.

## Recurring classes: reusable controls and bounded exceptions

| Recurring class / cross-subsystem search | Reusable or concrete preventive mechanism and evidence | Why no broader universal checker is claimed |
| --- | --- | --- |
| Bounds/completeness: Browser AX/DOM, Android tree/log, console/network, release files | Exact-boundary tables and actual final-encoding assertions; boundUIEvidence runs for raw/normalized/result; DOM whole-node prefix preserves IDs; log requests one overflow record; bounded release reader verifies opened bytes. Named regressions above enter normal package/harness execution. | Bytes, Unicode characters, records, nodes, caller envelopes and device-tail omission have different units and authority semantics. A single generic “truncate” helper would hide those distinctions. Existing common rules are exercised through explicit domain oracles. |
| Transform-after-bound / privacy: mobile, Browser, process logs, shared evidence | Full Service UI/Browser tests inspect retained artifacts after redaction, assert artifact existence/size/secret absence/identity and durable completion. Existing shared Redactor handles cross-chunk secrets; new dependent-window-hash test covers derived identity. | A text replacement engine cannot know which hashes are secret-derived or which fields grant action authority. Screenshots/private profiles are explicitly not pixel-redacted. No false blanket secrecy claim is added. |
| Positive identity before effects: Compose, process/Jobs, Android, Browser, Git | The missing-container-ID refusal test reaches public Down; exact helper same/different-build tests; existing tests rejecting mismatched or invalid OS birth/Job/session, frame/node and source-registration evidence retained. | Provider-specific proof differs (engine labels/IDs, Job handles, AVD identity, Chromium origin, Git worktree registration). A name-only common ownership abstraction would weaken the contract; cross-provider tests share proof questions, not interchangeable identity formats. |
| Persist/effect/cleanup and cancellation: named tests, readiness, UI, Browser, releases | Readiness now reuses CommandRun plus runWithCancellation; unsafe attempts remain running for existing Destroy/GC gates. UI preflight versus dispatched failure tests cover both sides; release publication already preserves validated bytes. | A generic transaction cannot atomically commit external OS/device/engine effects. Explicit durable intent/identity/finalization barriers and injected failures are necessary. Windows second-PID scheduling and late release close/persist injections remain identified follow-ups, not invisible passes. |
| Stale/temporal proof: Browser snapshot/load/URL, Android semantic input | Post-predicate document recheck plus tests preventing stale evidence after single or continuous navigation; retained existing frame mutation, stale fingerprint and no-coordinate-fallback controls. | There is no universal atomic observation across CDP/ADB/OS calls. Each multi-call operation must identify its own token and retry/uncertainty boundary. This audit adds the missing load boundary, not an impossible global atomicity guarantee. |
| Ambient prerequisite/path adoption: helper, Git, assets, release | Helper rejects absent/blank opt-in before Abs; existing provenance/symlink checks and asset opened-read controls; short-root release path cases and actual opened-file growth guard. | Host filesystems have platform-specific aliases, handle and case semantics. Native tests remain necessary; static checks alone cannot prove safety against arbitrary same-user hostile replacement outside the contract. |
| Weak/vacuous oracle: historical reviews across all adapters | Tests now require intended callbacks/effects to be reached, registered artifacts to exist, durable status to match and a nearby positive to succeed. Existing helper-only limitations remain listed in the historical annexes. | A generic test cannot decide whether another test's semantic injection was meaningful. Static coverage percentage is not proof of the intended branch; explicit injection counters and independent outputs provide the practical guard. |
| Documentation claims/navigation | Reuse documentProse for fragment targets as for required sections/links; full docsCheck tests rejecting hidden heading targets plus valid duplicate-heading cases; existing tests rejecting bilingual hash mismatches and invalid indexes retained. | Mechanical links/hashes are testable; actual translation meaning and native acceptance truth still need human/independent evidence review. No automatic prose-to-implementation theorem checker is asserted. |
| Native/concurrent behavior | Existing independent SQLite connections/processes, process/Job/native provider fixtures and candidate CI continue to provide non-mock evidence. File growth regression uses deterministic growth after opened stat. | Linux mocks/cross-builds cannot certify Windows/macOS/ADB/Machine behavior. Where no stable local fault hook exists, the ledger explicitly records the follow-up risk instead of adding a synthetic framework that pretends to be native proof. |

These mechanisms satisfy the concrete promotion decision: reuse existing
production/test entry points where the contract is shared, and add independent
domain-specific negative/positive oracles where it is not. No new global AGENTS
rule, organization policy, dependency or broad harness framework was required.
Historical tests reject changed manifests during browser selection and secret
literals in readiness only through helpers. These public-entry coverage gaps and
other remaining work, such as Java producer traversal, remain visible in the
source annexes and ledger; they are not rewritten as completed tests.

## Acceptance use and verification limits

- A3: the complete 20 × 14 matrix is above, with explicit N/A reasons and finding IDs.
- A27: every accepted finding has detected/earliest stages, escape categories,
  a concrete earlier opportunity and an implemented-control mapping.
- A28/A35: each recurring class above names reuse/regression evidence or states
  why broader automatic prevention would be misleading or disproportionate.
  This documents control scope; it does not mark final tests/independent review passed.
- The ledger records resolved counts; the completed plan and index record final
  acceptance, including successful native/integration verification and independent
  review. The matrix alone is not a substitute for that execution evidence.
