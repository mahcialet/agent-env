---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Historical Compose and standalone release review corpus

[日本語](history-compose-release.ja.md) · [Audit index](index.md) ·
[Execution authority](../../exec-plans/active/repository-correctness-audit.md)

Frozen production revision: `031869c8b9073b8e23bc17fbc55243666a52f557`.
This is Phase A evidence, not a remediation disposition. No product/test files
were changed. Historical severity labels were not recorded in these plans;
impact below is the recorded failure's consequence, not an invented original
severity. Escape explanations are retrospective inferences unless the source
explicitly explains the missing test.

## Sources and conventions

| Key | Historical source | Finding provenance |
| --- | --- | --- |
| P | [Podman feature](../../exec-plans/completed/compose-provider-podman.md) | Seven numbered independent findings; failed native attempts; cleanup-proof acceptance. |
| PR | [Podman review](../../exec-plans/completed/compose-provider-podman-review.md) | PR 8 discussions `3957283152` (UDP), `3957283162` (inventory). |
| D | [Distribution](../../exec-plans/completed/standalone-distribution.md) | Resumption regressions and native Windows failures. |
| DR | [Distribution review](../../exec-plans/completed/standalone-distribution-review.md) | PR 6 eleven threads: seven new fixes, four already delivered. Individual original IDs are not enumerated in the source. |
| F | [Release finalization](../../exec-plans/completed/standalone-release-finalization.md) | Audit of `0bf2d12`, independent review, ZIP mutation, build-info probe. |
| RR | [Release output review](../../exec-plans/completed/standalone-release-review.md) | PR 7 `PRRT_kwDOURHsR86gHJ1z`, discussion `3954761466`. |
| FR | [Release filter review](../../exec-plans/completed/standalone-release-filter-review.md) | PR 7 `PRRT_kwDOURHsR86gHb58`, discussion `3954872828`. |
| VR | [Verify entry review](../../exec-plans/completed/standalone-verify-review.md) | PR 6 discussion `3956143329`. |

All findings below remain applicable to this revision, including the historical
controls protecting currently unused production asset infrastructure. The
production bundled-asset inventory is empty; no CLI consumer is invented to
claim end-to-end asset coverage. IDs `HCR-*` are historical inventory IDs, not
current `AUDIT-*` findings.

Paths in the mapping tables are relative to repository root. Test names are the
current regression locations; unless stated otherwise they are also the
regressions recorded when the historical issue was fixed. `Yes/provider`,
`Yes/app`, `Yes/API`, and `Yes/command` mean the assertion reaches that actual
entry point (not necessarily the CLI). `Yes/helper` means the local invariant is
proved but the full entry point is **not** covered by that test. `Partial` or `No`
means the complete historical invariant lacks a directly demonstrated regression.
A passing present test plus source inspection is not mutation testing: this pass
did not reintroduce historical defects. A claim that a regression would fail is
limited to its visible oracle and current call path.

`Detected→earliest` uses the execution plan's S0–S9 stages. Detection at S7 is
used for implementation/resumption checks without a separate reviewer, S8 for
independent or PR review, and S5 for native-provider/OS failures. The original
plans sometimes combine discovery steps; those classifications are approximate.
Each row's escape column identifies the concrete earlier opportunity and why it
was missed. “Existing” is the guard status unless explicitly marked deferred;
no guardrail has been added by this audit.

## Podman mappings

| ID / source | Invariant, historical defect and impact | Current production → regression; proof/full-entry status |
| --- | --- | --- |
| HCR-P01 / P finding 1 | Inventory must find orphan-only providers. Enumerating only surviving lease rows suppressed entire engines. | `internal/app/reconciliation_inventory.go` provider union → `internal/app/compose_provider_test.go:TestInventoryDiscoversOrphansOutsideRecordedProviders`; Yes/app, empty/Docker/Podman recorded sets still call both engines. |
| HCR-P02 / P finding 2 | Every provider execution extension must enter policy or be rejected. Root/service-only checks missed nested network/secret keys. | `internal/runtime/compose/podman_normalize.go:rejectPodmanExtensions` via Render → `podman_test.go:TestPodmanRejectsNestedExecutionExtensions` and `TestPodmanRenderRejectsProviderSpecificHostAccess`; Yes/provider for the network case; helper covers five nested locations. |
| HCR-P03 / P finding 3 | Provider-specific mounts/network modes cannot bypass common host policy. `glob`, `ns:` and other modes escaped Docker-shaped validation. | Same normalization through `podmanClient.Render` → `TestPodmanRenderRejectsProviderSpecificHostAccess`; Yes/provider, config-call counter proves reaching normalization and rejects thirteen unsafe forms before runtime effects. |
| HCR-P04 / P finding 4 | Residual resources imply existence. Anonymous-volume evidence was appended after `Exists`, allowing false absence. | `internal/runtime/compose/podman.go:Inspect` → `TestPodmanInspectRetainedAnonymousVolumeExists`; Yes/provider, empty engine project lists plus one retained volume assert both existence and exact resource. |
| HCR-P05 / P finding 5 | Relocating a snapshot must preserve confined env/secret/config file references. Relative paths broke under temporary directories. | `podman_normalize.go:validatedPodmanFile`, `podman.go` adapter directory guard → `TestPodmanRelativeFileReferencesSurviveSnapshotRelocation`, `TestPodmanRejectsInitialFileDirectoryMismatchBeforeProvider`; Yes/helper/adapter, not full native Up. Absolute readable paths plus outside/directory/missing negatives and zero provider calls are asserted. |
| HCR-P06 / P finding 6 | Native service labels must prove Podman attribution. Docker compatibility labels alone were trusted. | `podman_normalize.go:normalizePodmanInspection` through native adapter → `TestPodmanNativeServiceOwnership`; Yes/helper, missing/conflicting native service denied and native-only accepted. Provider inventory has additional conflicting-label coverage (P13). |
| HCR-P07 / P finding 7 | Rendered environment must not re-resolve ambient secrets at Up. Null-map and bare-list entries leaked late host dependence. | `normalizePodmanConfig` → `TestPodmanEnvironmentRequiresExplicitPassThroughValues` plus Render negatives; Yes/provider for refusal; helper also preserves explicit empty/literal values and checks secret-free errors. |
| HCR-P08 / P native failure | Provider failures must preserve useful redacted native diagnostics. Shared wrapper discarded stderr and misidentified failures as Docker. | `podman.go:podmanCommandError`, adapter → `TestPodmanProviderFailurePreservesRedactedNativeDiagnostic`; Yes/adapter for provider/exit/message/redaction. **Partial** for the documented 8 KiB diagnostic cap: historical plan explicitly states no long-diagnostic regression. |
| HCR-P09 / P native failure | Canonical dynamic-port intent must be translated without modifying retained configuration. Real podman-compose rejected published numeric/string zero. | `podman_normalize.go:escapePodmanSnapshot` through Up → `TestPodmanUpUsesDynamicPortWithoutChangingCanonicalSnapshot`; Yes/provider, checks received temporary bytes, numeric/string forms, scope fields, and original snapshot. Native CLI fixture supplies recorded real evidence. |
| HCR-P10 / P cleanup milestone | Persist anonymous attachment proof before Down; interrupted removal must retain proof after containers disappear. | `internal/app/lifecycle.go` cleanup evidence and Podman Down → `compose_provider_test.go:TestCleanupRetainsProofAfterContainerDisappears`, `TestCleanupProofWriteFailurePreventsDown`; Yes/app, two Destroy calls, quarantine/released states, saved proof and zero effects on Save failure. This is material preventive work, not one of the seven numbered defects. |
| HCR-P11 / P failed fixture | Failed real validation must retain recovery identity and must not remove inputs while lease cleanup is uncertain. | `internal/cli/podman_integration_test.go` fixture cleanup/recovery; **Partial**, real fixture contains recovery policy but this corpus found no separately injected fixture-failure regression. This concerns test infrastructure, not an observed current product defect. |
| HCR-P12 / PR UDP | TCP reachability cannot classify UDP endpoint existence/readiness. Remote UDP mappings disappeared after TCP dial failure. | `podman.go:Inspect` protocol filter → `podman_review_test.go:TestPodmanRemoteInspectPreservesUDPAndChecksTCP`; Yes/provider, actual local UDP/TCP sockets plus remote-shaped identity; UDP-only, mixed reachable and mixed unreachable cases. Real Podman Machine forwarding remains outside this evidence. |
| HCR-P13 / PR inventory | Engine-only Doctor and Inventory must both work without a Compose frontend. Shared Docker traversal still called Compose ls. | `podman.go:Inventory`, `inventory.go` labelled traversal → `podman_review_test.go:TestPodmanInventoryWithoutComposeDiscoversNativeOrphans`; Yes/provider dispatcher, Doctor then InventoryFor, missing/stale frontend, all three kinds, conflicts and partial failure. |

## Podman escape analysis

| IDs | Detected→earliest; escape reasons | Earlier detection opportunity, missing oracle, preventive control and recurrence |
| --- | --- | --- |
| P01 | S8→S4; COMPOSITION_GAP, NEGATIVE_FIXTURE_GAP | Engine tests knew only their engine; app fixtures with no surviving rows were missing. Existing app provider-union test is the earlier guard (S4). Search all app inventory kind/provider filters; P13 is a related composition recurrence, not the same root cause. |
| P02 | S8→S2; NEGATIVE_FIXTURE_GAP | The policy already rejected execution extensions, but fixtures only mirrored checked nesting. Existing recursive mutation matrix moves detection to S2; Render coverage prevents helper disconnection. Apply the same search to config/Android/browser nested payload validators. |
| P03 | S8→S3; ORACLE_COUPLING, COMPOSITION_GAP | A shared policy was specified, but Docker-normalized fixtures omitted raw Podman variants. Existing raw Render negatives are S3 prevention. Other adapter normalization boundaries require their own provider dialect fixtures; one universal syntax validator is not justified. |
| P04 | S8→S2; ORACLE_COUPLING | Existence was tested with visible containers, sharing the early-resource-set assumption. Existing retained-only Inspect fixture makes the final-set invariant explicit at S3; smallest realistic local test was S2. Search every observer that appends cleanup evidence after status derivation. |
| P05 | S8→S3; COMPOSITION_GAP | Normalized fixtures did not simulate changed snapshot base directory. Existing file-reference and adapter guards prevent local regression at S3; real relocation through Up remains a focused future enhancement, not a claimed test. Search process/browser/Android staging path relocation separately. |
| P06 | S8→S2; ORACLE_COUPLING, NEGATIVE_FIXTURE_GAP | Docker-compatible labels were convenient positive fixtures, omitting native-only/missing/conflicting combinations. Existing three-way helper matrix is S2 guard. Repeat the identity-label matrix for every destructive resource kind. |
| P07 | S8→S2; NEGATIVE_FIXTURE_GAP, COMPOSITION_GAP | Explicit environment fixtures hid the Render→Up ambient lookup window. Existing bare/null/empty/literal matrix and raw Render tests catch at S2/S3. Ambient-state recurrence includes release checkout filters (R17). |
| P08 | S5→S3; FAILURE_INJECTION_GAP | Provider command failure could have been injected before real execution; success-only adapter fixtures did not check usable diagnostics. Existing redacted-failure test catches at S3. Long-message cap guard remains **deferred** for boundary audit; do not label current cap disproved. |
| P09 | S5→S5; NATIVE_EVIDENCE_GAP, ORACLE_COUPLING | Docker port-zero semantics were assumed by fakes; actual supported podman-compose was the first reliable oracle. Existing real CLI coexistence fixture plus private-byte Up regression catches at S5/S3. Native evidence cannot be replaced by cross-builds. |
| P10 | S7→S4; FAILURE_INJECTION_GAP | A normal Down success cannot prove retry after destructive partial effects. Existing app failure injection proves Save-before-Down and retained proof at S4. Apply that effect/persistence schedule to all runtimes; broad helper adoption needs current recurrence evidence. |
| P11 | S5→S4; FAILURE_INJECTION_GAP | Success-only fixture cleanup assumed Create returned JSON. Registry fallback and retained directories are existing controls; independent failure-injection coverage is **deferred** to audit disposition. Check other native fixtures before interpreting a failed run's cleanup as safe. |
| P12 | S8→S3; NEGATIVE_FIXTURE_GAP, COMPOSITION_GAP | Remote observations lacked mixed protocol fixtures; TCP helper success did not validate UDP behavior. Existing full Inspect socket matrix catches at S3. Search protocol-specific readiness for UDP/TCP crossover, without claiming UDP dial proves readiness. |
| P13 | S8→S3; HELPER_ONLY, COMPOSITION_GAP | The original named Doctor regression passed while subsequent Inventory invoked a different dependency. Existing Doctor→InventoryFor test catches at S3. This explicitly recurs in releaseVersion→release-verify (R15/R18); preventive pattern is paired public-entry tests, not additional helper-only assertions. |

## Distribution and asset mappings

| ID / source | Invariant, historical defect and impact | Current production → regression; proof/full-entry status |
| --- | --- | --- |
| HCR-A01 / D; DR already fixed | Concurrent identical materialization must tolerate legitimate mkdir winners. EEXIST caused child failures. | `internal/assets/assets.go:safeMkdirAll` through Materialize → `assets_test.go:TestMaterializeAcrossProcesses`; Yes/API, twelve independent processes, thirty roots, three writes per round, exact bytes and no leftover staging. |
| HCR-A02 / D | Absolute state override must work without default HOME. Home discovery ran too early. | `internal/paths/paths.go:Resolve` → `paths_test.go:TestResolveOverrideWithoutUserHome`; Yes/API, no CLI claim. |
| HCR-A03 / D | UI helper installation staging belongs under owned runtime state, including error leftovers. OS temp escaped the state-root contract. | `internal/runtime/android/ui.go` helper install → `ui_test.go:TestUIHelperStagingStaysInOwnedRuntimeAndCleansUp`; Yes/provider, install success/error observe runtime-local APK and cleanup. |
| HCR-A04 / D Windows | Embedded provenance must not vary with checkout line conversion. Native Windows autocrlf changed 17 bytes to 18. | `.gitattributes` fixture `-text` → `assets_test.go:TestEmbeddedFixture`; Yes/API for compiled bytes/hash, native Windows needed to test checkout conversion itself. |
| HCR-A05 / D Windows | Immutable publication must preserve a concurrent winner. Windows replace triggered sharing/access failures. | `internal/assets/publish_windows.go:publishAsset` → `publish_windows_test.go:TestPublishAssetPreservesWindowsWinner` plus process stress; Yes/helper/API, native Windows only. |
| HCR-A06 / D Windows | Transient Windows read sharing/lock violations may retry within a bound; permanent errors must remain failures. First no-replace repair still failed reads. | `publish_windows.go:readAsset` → `TestReadAssetWindowsSharingConflict`, `TestReadAssetWindowsDoesNotRetryMissingFile`; Yes/helper, held native handle proves retry/release and bounded failure; native-only, not replayed on Linux. |
| HCR-A07 / DR | Logical asset names must be portable on all hosts. Twenty-five invalid names passed. | `assets.go:portableName` through Describe → `assets_test.go:TestDescribePortableLogicalNames`; Yes/API, Windows reserved names/separators/trailing forms tested on every host. |
| HCR-A08 / DR | Cached assets must be bounded before/during read. An oversized cached file was read before size rejection. | `assets.go:readAssetOnce` through Materialize → `TestMaterializeRejectsOversizedCacheBeforeReading`, `TestReadAssetRejectsSizeMismatch`; Yes/API plus helper, 32 MiB cache and opened-read mismatch cases. |
| HCR-A09 / DR | Unset linker identity must fall back to actual embedded VCS while explicit identity stays authoritative. Clean Git builds lost commit identity. | `internal/buildinfo/buildinfo.go:Current` → `buildinfo_test.go:TestCurrentVCSFallback`, `vcs_test.go:TestCurrentEmbeddedVCSIdentity`; Yes/API and actual built executable, clean/dirty identity. |
| HCR-A10 / DR already fixed | State-root/ancestor symlinks must not redirect materialization. A stale thread was already covered. | `assets.go:safeMkdirAll` → `TestMaterializeRejectsNonDirectoryAndSymlinkEntries`, `TestMaterializeRejectsSymlinkedAncestorAndPortableNames`; Yes/API, all path positions, outside directory stays empty. Symlink-unavailable platforms explicitly skip, not pass. |

| IDs | Detected→earliest; escape reasons | Earlier opportunity, preventive control, recurrence |
| --- | --- | --- |
| A01 | S7→S4; CONCURRENCY_GAP | Single-process/idempotence checks omitted simultaneous first creation. Existing independent-process stress catches S4; scheduling is probabilistic, not proof of every race. Search other mkdir/materialization caches. |
| A02 | S7→S2; NEGATIVE_FIXTURE_GAP | Tests set a home even when specifying an absolute override. Existing no-home fixture catches S2. Lazy prerequisite order repeats at CLI initialization boundaries. |
| A03 | S7→S3; COMPOSITION_GAP | Helper success checks did not observe the subprocess APK path or error cleanup. Existing runner-observed success/error fixture catches S3. Apply path-ownership checks to each other staged helper independently. |
| A04 | S5→S5; NATIVE_EVIDENCE_GAP | Linux bytes/cross-builds did not exercise Windows checkout configuration. Existing fixed hash plus native CI is S5 guard; scoped attribute avoids unrelated conversion changes. |
| A05 | S5→S5; NATIVE_EVIDENCE_GAP, CONCURRENCY_GAP | Unix rename semantics masked native replacement restrictions. Existing native winner test and same process stress catch S5. Search release/process publication without assuming identical filesystem semantics. |
| A06 | S5→S5; NATIVE_EVIDENCE_GAP, FAILURE_INJECTION_GAP | Initial native repair covered publishing, not opening a file held by a competitor. Existing held-handle test is S5 guard; unrelated permission/missing-file errors remain immediate. Process cleanup later required its own sharing-violation guard; do not infer the same retry policy for all operations. |
| A07 | S8→S2; NEGATIVE_FIXTURE_GAP | Only traversal-oriented name checks existed despite stated portability. Existing cross-host invalid-name matrix catches S2; inspect other logical names for Windows aliases and separators. |
| A08 | S8→S2; BOUNDARY_GAP | Small corruption fixtures did not expose unbounded allocation before rejection. Existing sparse/large-cache and opened-read mismatch checks catch S2. Search other evidence/file readers for preflight-plus-stream bounds. |
| A09 | S8→S3; HELPER_ONLY, COMPOSITION_GAP | Simulated build fields did not prove real Go VCS embedding. Existing build/run fixture catches S3; fake field tests separately protect explicit overrides. ReleaseRecord/static inspection needs a distinct real artifact oracle. |
| A10 | S8→S2; NEGATIVE_FIXTURE_GAP | The review is stale, not a newly reproduced defect. Existing path-position matrix gives S2/S3 prevention; exact historical first detection stage is not recorded. Search destructive path checks separately for replacement races; these static fixtures do not prove TOCTOU safety. |

## Release mappings

| ID / source | Invariant, historical defect and impact | Current production → regression; proof/full-entry status |
| --- | --- | --- |
| HCR-R01 / F audit | Git source queries must use the selected root. Ignoring root could validate/build the wrong repository. | `tools/repoctl/release_source.go:gitOut` → `release_source_test.go:TestReleaseSourceValidTags`, `TestReleaseGitIgnoresAmbientRepositoryRouting`; Yes/helper with disposable repositories distinct from cwd and poisoned routing. Command uses same helper. |
| HCR-R02 / F audit | release-check must inspect real artifacts, not merely return success. Earlier implementation inspected none. | `release.go:checkRelease` → `release_e2e_test.go:TestReleaseCandidate` identity/digest/tampering cases; Yes/component, real six-target candidate, **opt-in and not replayed here**. Command usage tests alone are not acceptance. |
| HCR-R03 / F audit | Packaging must fail if required inputs are missing. Earlier code silently omitted inputs. | `release_archive.go:writeArchive/readReleaseArchive` → `release_archive_test.go:TestReleaseArchiveRejectsUnsafeMembers/missing`, round-trip; **Partial**: reader rejects missing members, but no dedicated missing-input writer regression was found. Writer currently checks exact staging membership and every required file. |
| HCR-R04 / F audit | Archive must contain correct LICENSE/README.txt/executable. Earlier packaging used README.md. | `release.go:releaseReadme`, archive names → `TestReleaseArchiveRoundTripDeterministic`, candidate `license`/`readme` rehashed tampering; Yes/helper/component, independent expected contents and semantic checks. Candidate cases opt-in. |
| HCR-R05 / F audit | Archive close errors must propagate so failed output cannot be accepted. Earlier code ignored close errors. | `release_archive.go:writeArchive` joins deferred file/ZIP/gzip/TAR close errors; **No direct injected close-error regression located**. Successful round-trip does not prove this historical failure mode. Guard status: **deferred** failure-injection coverage; no current failure asserted. |
| HCR-R06 / F audit | Caller output must never be deleted to make a build succeed. Earlier code removed existing output. | `release.go:buildRelease/publishReleaseDirectory` → `release_publish_test.go:TestReleasePublicationPreservesOutputAndCleansTransfer`, candidate `existing_output_preserved`; Yes/helper/component, exact old bytes and failed-transfer absence. |
| HCR-R07 / F bilingual | Translation hashes must not stand in for translated meaning. Progress/revision changed only in English despite refreshed Japanese hash. | English/Japanese completed plans plus repository docs-check; **Partial**: hashes mechanically catch byte staleness, not semantic mistranslation. Human paired review is the existing control. Cross-reference the parent audit's bilingual corpus; do not double-count as a new product finding. |
| HCR-R08 / F probe | Static release identity must survive trimpath while independently matching VCS/target/CGO. Go omitted linker flags from buildinfo. | `buildinfo.go:ReleaseRecord/Current`, `release.go:inspectReleaseBinary` → `TestCurrentReleaseRecord`, `TestInvalidReleaseRecordPreservesDevelopmentIdentity`, candidate `build_identity`/`version_record`; Yes/API/component. Real candidate is required for stripped-binary composition. |
| HCR-R09 / F independent | Compiler input must be committed bytes, not ignored or hidden caller edits. Initial build consumed mutable caller tree. | `release_source.go:privateReleaseSource` → `release_source_test.go:TestPrivateReleaseSourceUsesOnlyCommittedFiles`; Yes/helper, ignored file absent, hidden tracked edit excluded, caller preserved. Full candidate build calls this same helper. |
| HCR-R10 / F independent | Source identity must still equal the initial identity at publication. Earlier final validation's returned identity was discarded. | `release.go:buildRelease` final tag/time/commit comparisons before two checks/publication; **Partial**: source validators have negatives but no dedicated during-build identity mutation regression located. Guard status: **deferred** interleaving injection; source inspection confirms comparison exists. |
| HCR-R11 / F independent | Tag publication must depend on same-source repeat validation and native smoke, using validated bytes. Earlier tag-specific repeat gate absent. | `.github/workflows/release.yml` → `release_workflow_test.go:TestReleasePublicationGate`; Yes/static workflow oracle, four active YAML mutations. It checks graph/command presence, not arbitrary shell/action semantics; native workflow evidence remains separate. |
| HCR-R12 / F ZIP mutation | Safe central ZIP names cannot conceal traversal/absolute local names. Initial checker trusted the central directory. | `release_archive.go` local-header validation → `TestReleaseArchiveRejectsMismatchedZIPLocalNames`; Yes/helper, mutation changes local headers of a valid archive without recompression; reader is the real checkRelease path. |
| HCR-R13 / DR | Module paths matching checkout suffixes are not host-path leakage. Initial byte search falsely rejected valid artifacts. | `release.go:releaseBinaryContainsPathWithModules` → `release_path_test.go:TestReleaseBinaryKnownModulePaths`, candidate `module_path_is_not_checkout_leak`; Yes/helper/component, real candidate presence assertion and appended genuine-path negative. |
| HCR-R14 / DR failed repair | Genuine absolute path leakage must still be detected within concatenated Go string pools. Proposed token-boundary exclusion admitted paths following letters. | Same detector → `TestReleaseBinaryDetectsConcatenatedPathLiteral` plus table concatenation case; Yes/helper with real built binary. This was an independently rejected approach, not a published regression. |
| HCR-R15 / DR | Cleanliness must reject assume-unchanged/skip-worktree even if unchanged. Porcelain hid caller edits. | `release_source.go:releaseSourceClean/releaseVersion` → `TestReleaseSourceRejectsHiddenIndexEntries`; Yes/helper, both flags × changed/unchanged. Full command gap recurred as R18. |
| HCR-R16 / RR | Valid new non-ignored worktree output must not dirty the source before its final check. Sibling construction staging caused false rejection. | `release.go:buildRelease` external staging then destination-local publish → candidate `nonignored_worktree_output`; Yes/component, nested Unicode non-ignored output compared byte-for-byte; opt-in, not replayed here. |
| HCR-R17 / FR | Private checkout compiler bytes must ignore ambient filters/attributes. Smudge changed bytes while matching clean hid drift. | `release_source.go:releaseGitEnv/privateReleaseSource` → `TestPrivateReleaseSourceIgnoresAmbientCheckoutFilters`; Yes/helper integration, global/system filters demonstrably transform ordinary checkout yet isolated clone equals committed bytes. |
| HCR-R18 / VR | release-verify must enforce caller cleanliness before any build/output. It checked porcelain on caller but flag-aware releaseVersion only on clean clone. | `release_verify.go:executeReleaseVerify` → `release_verify_test.go:TestReleaseVerifyRejectsHiddenIndexEntries`; Yes/command via executeArgs, all four cases assert diagnostic, zero output/build logs, absent destination and unchanged caller index/bytes/refs. |
| HCR-R19 / DR metadata | Archived parent status and durable database path must match delivered state. Parent remained active; audit said registry.sqlite rather than state.db. | Completed distribution metadata and design path audit; `internal/cli/lifecycle.go` state.db construction; **Partial** mechanical docs checks plus direct source/prose comparison, no semantic path-name validator. Bilingual corpus owns general semantic drift analysis. |

The other two already-delivered PR 6 comments concerned Japanese milestone
translation and checksums/manifest placement as release-set siblings. R07 covers
the translation control; R02/R04 and `releaseChecksums/checkRelease` cover the
layout contract. The source does not establish an additional current defect.
Thus all eleven DR comment topics are represented without recounting stale
comments as new fixes.

## Release escape analysis

| IDs | Detected→earliest; escape reasons | Earlier opportunity, preventive control and recurrence |
| --- | --- | --- |
| R01 | S7→S2; HELPER_ONLY, NEGATIVE_FIXTURE_GAP | A disposable root distinct from cwd was already feasible; tests did not exercise real Git release behavior. Existing source fixture/routing poison catches S2/S3. Source/worktree Git routing needs the same review question. |
| R02 | S7→S3; HARNESS_GAP, NEGATIVE_FIXTURE_GAP | A successful command exit was treated as acceptance without corrupt candidate tests. Existing real-candidate semantic mutation matrix catches S3/S4 when enabled; routine unit success must not be reported as candidate evidence. |
| R03–R04 | S7→S2; NEGATIVE_FIXTURE_GAP, ORACLE_COUPLING | Stated archive membership/content allowed direct missing/wrong-file fixtures; old tests did not exercise release behavior. Existing reader/semantic candidate guards catch S2/S3. Writer missing-input negative remains **deferred**; reader coverage alone is not equivalent. |
| R05 | S7→S2; FAILURE_INJECTION_GAP | Close errors were a local error-return contract, but success-only archive tests missed them. Existing error joins are production guard, not regression evidence. A failing writer/close fixture is **deferred**; avoid claiming round-trip catches dropped error propagation. |
| R06 | S7→S2; NEGATIVE_FIXTURE_GAP | Existing destination with sentinel bytes was an available negative fixture. Existing publication/candidate preservation tests catch S2/S3. Same no-clobber property recurs in A05 under native concurrency. |
| R07 | S7→S7; REVIEW_CHECKLIST_GAP | Hash checks had enough data to detect byte mismatch, not meaning; source explicitly records hash-only refresh. Existing paired human review is the realistic earlier semantic control. A generic semantic translation validator is not claimed practical. |
| R08 | S7→S3; COMPOSITION_GAP | Linking identity fields did not prove static inspection of a real trimmed executable. Existing real build/probe/candidate checks catch S3; field-unit tests alone do not. Related actual-VCS composition escape is A09. |
| R09 | S8→S3; NEGATIVE_FIXTURE_GAP | Clean Git status did not establish compiler input identity; ignored and flagged edits were constructible before review. Existing private-clone byte oracle catches S3. Ambient filter recurrence R17 shows clean status must never be the only source-byte oracle. |
| R10 | S8→S3; FAILURE_INJECTION_GAP | Initial/final observations existed but tests did not mutate identity between them. Current comparisons are inspected; dedicated interleaving guard **deferred**. Review every effect followed by a discarded verification result. |
| R11 | S8→S6; HARNESS_GAP, REVIEW_CHECKLIST_GAP | YAML could have been mechanically checked for required graph edges/repeat command before review. Existing active-mutation workflow test catches S6. Text presence is not execution proof; preview and tag workflows require distinct checks. |
| R12 | S8→S2; ORACLE_COUPLING, NEGATIVE_FIXTURE_GAP | Archive-library-generated fixtures kept local/central headers consistent, matching the parser assumption. Existing independent local-byte mutation catches S2; generalize methodology to duplicate representations, not a ZIP-only checklist. |
| R13 | S8→S2; NEGATIVE_FIXTURE_GAP | Short checkout basenames colliding with module suffixes were absent. Existing collision/genuine-path matrix catches S2, actual candidate composition catches S3. |
| R14 | S8→S3; ORACLE_COUPLING | Token assumptions looked plausible in text fixtures but Go packs adjacent strings without boundaries. Existing actual Go-binary negative catches S3. No post-merge escape is attributed to this rejected repair. |
| R15 | S8→S2; NEGATIVE_FIXTURE_GAP | Porcelain was the oracle despite Git's explicit hidden-index flags. Existing four-case helper matrix catches S2. R18 demonstrates this control originally stopped short of the actual command. |
| R16 | S8→S3; COMPOSITION_GAP | Previous build evidence used outside-tree/ignored destinations; the documented generic output path required a non-ignored case. Existing real nested-output regression catches S3; no cleanliness exception was added. |
| R17 | S8→S3; NEGATIVE_FIXTURE_GAP, COMPOSITION_GAP | Private clone tests omitted active clean/smudge filters that could preserve clean status. Existing active-transform control checkout makes a strong independent byte oracle at S3. Similar ambient dependency: P07 environment resolution. |
| R18 | S8→S3; HELPER_ONLY, COMPOSITION_GAP | Previous R15 helper tested releaseVersion, while orchestration applied it only to another repository. Existing executeArgs negatives assert ordering and preserved caller state at S3. Recurs with P13 Doctor versus Inventory; pair related public entry points when centralizing guards. |
| R19 | S8→S7; REVIEW_CHECKLIST_GAP | Metadata and path prose could be compared directly to completed state/CLI source. Existing docs check catches metadata syntax but not these meanings; paired source/prose review is the control. Defer broader semantic claim validation to the audit's docs work. |

## Present evidence, limits and follow-up

2026-09-09, native Linux, Go 1.27.1, frozen revision:

- `go test -race ./internal/assets ./internal/buildinfo ./internal/runtime/compose ./tools/repoctl -count=1` passed: 2.079s, 2.146s, 1.306s, 8.768s respectively. This includes actual subprocess materialization and real Git/filter fixtures. `TestReleaseCandidate` skips without its opt-in variable; it is **not** counted as replayed.
- Focused race run of P01/P10/A02/A03 and standalone lazy-prerequisite regressions passed in app 1.381s, paths 1.029s, Android 1.047s and CLI 1.048s. Selected test names: `TestCleanupRetainsProofAfterContainerDisappears`, `TestCleanupProofWriteFailurePreventsDown`, `TestInventoryDiscoversOrphansOutsideRecordedProviders`, `TestResolveOverrideWithoutUserHome`, `TestUIHelperStagingStaysInOwnedRuntimeAndCleansUp`, `TestStandaloneCommandsRequestGitOnlyWhenSourcesAreNeeded`.
- Windows held-handle tests, Windows checkout conversion, native Podman/Docker coexistence and real six-target candidate/native smoke were **not** replayed by this bounded historical pass. Their original plans retain exact historical evidence; new audit baseline evidence belongs to the parent report.

No current production defect is confirmed by this historical replay. Coverage
limitations requiring disposition are P08 long diagnostic bounds, P11 failing
fixture recovery, R03 writer missing input, R05 close-error injection and R10
mid-build identity change. These are suspects about guard strength, not proof
that the existing implementation violates its contract; no `AUDIT-*` IDs are
assigned here. Full-entry gaps explicitly marked above should guide the later
cross-component audit rather than be hidden behind green helper tests.

Bounded recurrence searches inspected app inventory dispatch, Podman/shared Docker
traversal, asset publication/read paths, source validation callers, release
orchestration and workflow graph tests. Confirmed historical recurrence pairs:
P13/R18 (helper versus next entry), R09/R17 (clean status versus compiler bytes),
A05/A06 (native publication fix versus subsequent read), P01/P13 (inventory
composition), and A09/R08 (simulated identity versus actual binary). Broader
current runtime/portability/cleanup review remains a separate Phase A step.
