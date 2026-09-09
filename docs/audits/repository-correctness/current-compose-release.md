---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Current Compose, assets and release correctness review

[日本語](current-compose-release.ja.md) · [Audit index](index.md) ·
[Historical corpus](history-compose-release.md) ·
[Execution authority](../../exec-plans/completed/repository-correctness-audit.md)

Phase A target: `031869c8b9073b8e23bc17fbc55243666a52f557`.
Phase A used review and temporary Go overlays only. Phase B accepted the three
findings below at checkpoint `56b9c2c`. Phase C implementation and validation are
recorded at the end of this report; no provider mutations or publication were
performed by this review slice.

## Phase A coverage matrix

| Invariant | Reviewed implementation and evidence | Result / limits |
| --- | --- | --- |
| Provider selection / identity | Compose dispatcher, Docker recorded context, Podman encoded executable/URL/fingerprint/environment, native bridge. | No fallback; Podman rechecks fingerprints before mutations. Docker pins context name, not daemon fingerprint: this review does not infer stronger protection. |
| Ownership before destruction | Container/named-resource inspection, declared-name lookup, anonymous-volume attachment/fingerprint/current-user proof, Down re-observation. | Missing container ID authorizes Down: finding below. Named-resource and inventory empty-ID guards already exist. Concurrent engine replacement between observations is not disproved by static review. |
| Readiness / protocol | Selected service presence, container running/health, retained-volume existence, Podman remote TCP-only checks. | Historical P04/P12 regressions pass. TCP success does not prove UDP/application readiness. Actual Machine forwarding is not claimed. |
| Partial cleanup / persistence | App cleanup-proof Save before Down, repeat Destroy after lost containers, scoped residual deletion, absence after rm/down. | Historical P10 app failure injection passes; no new confirmed defect in reviewed paths. Entire app saga is reviewed separately by the parent audit. |
| Inventory completeness | Available/recorded provider union and independent native labelled traversal; missing/stale Compose frontend. | Historical P01/P13 pass, including partial errors. Count-only inspection set matching is a follow-up hypothesis, not a reproduced finding here. |
| Policy / path relocation / ambient state | Raw Podman normalization, recursive extensions, allowed mount/network forms, file confinement, explicit environment, private escaped JSON. | Existing provider/normalization regressions pass. No native malicious-manifest experiment conducted. |
| Evidence / bounds | Native diagnostic redaction before 8 KiB tail; canonical snapshot secrets and inherited secrets; archive/file/asset readers. | Release reader growth bypass confirmed. Diagnostic long-message test remains historical coverage limitation; no diagnostic leak reproduced. |
| Assets / concurrency / path | Portable names, provenance, root/intermediate entry checks, concurrent mkdir, bounded opened reads, native publication implementation. | Existing Linux independent-process and corruption tests pass. Windows held-handle/long-path/winner tests require native baseline. Static symlink checks are not proof against arbitrary replacement races. |
| Release source | Strict canonical tag, HEAD equality, clean index/tree, private committed checkout, filter/config isolation, final identity comparison. | Source/filter/hidden-index regressions pass. Mid-build final-identity injection remains historical coverage limitation. No public refs modified. |
| Packaging / exact boundaries | Three members, exact eight-file set, canonical manifest, checksums, local/central ZIP, whole-second mtimes, member stream caps. | Existing archive mutation/range tests pass. Short-root path filtering and outer regularRead bound fail. Large 256 MiB exact-size allocations were not run merely to simulate the small-cap reproducer. |
| Publication / reproducibility | Independent stage, source validation, destination-local transfer, existing-output refusal, workflow build→smoke→publish, exact artifact name. | Preservation/workflow negative tests pass. Real repeat builds and native smoke are parent baseline work; no public GitHub Release is created. |

## AUDIT-RELEASE-001 — short absolute source roots bypass leakage detection

- Severity: Low. Disposition: ACCEPT (`56b9c2c`).
- Invariant: a known nonempty absolute checkout path in binary bytes must not be
  exempted solely because its root string is short.
- Location: `tools/repoctl/release.go:releaseBinaryContainsPathWithModules`,
  `if len(path) <= 3 { return false }`.
- Trigger: `releaseBinaryContainsPath([]byte("/a/private.go"), "/a")`, or root
  `/ab` and corresponding bytes. No module identity is present.
- Observed: both return false. The Go-overlay assertion fails for both roots.
- Expected: detect the known absolute prefix; any intentional root/volume-root
  exception should be structural and documented, not cover ordinary short paths.
- Impact: the path-leak guard has a blind spot for valid short checkout roots.
  This is a detector-level contract gap; no secret exposure or whole candidate
  validation bypass has been demonstrated.
- Existing coverage: `TestReleaseBinaryKnownModulePaths` uses `/tmp`, `/agent-env`
  and longer Windows roots; none exercise lengths 2 or 3. The real concatenated
  literal fixture also uses `/agent-env`.
- Reproducer: temporary `go test -overlay` test calling the two expressions above
  and requiring true; observed both failures on native Linux Go 1.27.1.
- Regression / resolution: Phase A made no changes; see the Phase C record below.
- Verification: failing overlay `TestAuditShortReleaseRootLeak`; no normal test
  files changed. Related: historical HCR-R13/R14, current path/boundary audit.
- Escape: detected S9; earliest S2. `BOUNDARY_GAP`, `NEGATIVE_FIXTURE_GAP`.
  Earlier opportunity: requirement-specific path lengths around the early-return
  boundary. Existing fixtures shared only longer directory examples, so neither
  helper nor real-binary test exercised this branch.
- Preventive guardrail: structural root cases plus length-adjacent valid roots and
  genuine/module-path positive/negative pairs. Expected future stage S2.
  Guard implementation/evidence: see the Phase C record below.

## AUDIT-RELEASE-002 — preflight file size does not bound the actual read

- Severity: Medium. Disposition: ACCEPT (`56b9c2c`).
- Invariant: `regularRead(path, limit)` cannot return more than `limit` bytes or
  allocate based on unbounded growth after its size check.
- Location: `tools/repoctl/release.go:regularRead` checks `os.Lstat().Size()` then
  calls unbounded `os.ReadFile`. `checkRelease`, `compareReleaseDirectories` and
  `copyVerifiedRelease` use this helper. `release_archive.go:writeArchive` repeats
  the same preflight/unbounded-read pattern for source members.
- Trigger: another handle grows a regular file between Lstat and ReadFile. A
  bounded test alternates the file between one byte and 64 KiB while calling
  `regularRead(path, 1)`; it never writes caller or release data.
- Observed: a successful call returned 8,193 bytes with a one-byte limit on
  attempt 38. This is an actual native filesystem interleaving, not a fake stat.
- Expected: opened-file verification plus bounded streaming (and rejection if
  extra data exists), independent of concurrent growth or replacement.
- Impact: release candidate/manifest/checksum and copy/compare caps are not hard
  read bounds. Later JSON/checksum checks may still reject content, but only
  after excessive allocation. No accepted corrupt release is claimed.
- Existing coverage: archive member decompression already uses LimitReader;
  historical asset HCR-A08 checks opened-file size and bounded reads. Release
  fixtures mutate complete stable files, so they do not cover concurrent growth.
- Reproducer: isolated Go-overlay `TestAuditRegularReadGrowthBound`; writer opens
  its temporary file once, repeatedly `WriteAt(64 KiB, 0)` then `Truncate(1)`;
  reader loops up to 20,000 calls, fails on `err == nil && len(bytes) > 1`.
  Writer is joined at exit. If no interleaving occurs, the fixture explicitly
  skips rather than treating the run as safety proof.
- Regression / resolution: Phase A made no changes; see the Phase C record below. The writer
  staging recurrence is source-confirmed but not separately fault-injected.
- Verification: failing overlay on native Linux Go 1.27.1, 0.008s package run
  including RELEASE-001. Existing repoctl race suite passed 8.768s separately.
- Related: historical HCR-A08 (same defect class in assets), HCR-R03/R05 (writer
  error/boundary coverage); broader file-reader audit.
- Escape: detected S9; earliest S2. `CONCURRENCY_GAP`, `BOUNDARY_GAP`,
  `FAILURE_INJECTION_GAP`. Earlier opportunities: mutate after preflight or use a
  bounded reader oracle rather than only stable oversized-file rejection.
  The asset repair was local, leaving the same pattern in release infrastructure.
- Preventive guardrail: shared bounded regular-file read semantics or a reusable
  cap/growth fixture across asset/release/evidence readers. Expected S2/S3;
  implementation and regression evidence: see the Phase C record below.

## AUDIT-OWNERSHIP-001 — missing container identity authorizes Down

- Severity: High. Disposition: ACCEPT (`56b9c2c`).
- Invariant: missing external container identity is incomplete observation and
  cannot authorize destructive Compose Down.
- Location: `internal/runtime/compose/compose.go:dockerClient.Inspect`, container
  loop after the count check; it verifies labels but not `v.ID`. Down trusts the
  returned error status before issuing `compose down --volumes --remove-orphans`.
- Trigger: managed runtime has recorded lease/runtime/project and immutable JSON;
  ps lists one container; inspection returns one running selected-service record
  with matching ownership labels but omits `Id`; post-Down list is empty.
- Observed: the full public `Client.Down` dispatches destructive Down and returns
  nil. The temporary fake runner records the dispatch; it executes no engine.
- Expected: refuse before any Down effect because container identity is absent.
- Impact: an incomplete identity observation passes the destruction barrier.
  No native engine producing this malformed record, actual unrelated deletion,
  or arbitrary foreign resource takeover has been demonstrated.
- Existing coverage: `TestManagedResourcesRequireBothOwnershipLabels` varies
  labels while keeping IDs valid. Named-resource inspection and native inventory
  already reject empty identities; the container branch lacks the equivalent.
- Reproducer: isolated overlay `TestAuditMissingContainerIDAuthorizesDown` uses
  existing runtimeFixture, writes a temporary canonical snapshot, sets LeaseID,
  and returns the described runner responses. Assertion requires error and zero
  Down dispatch. Observed `dispatched=true err=<nil>` (0.025s package run).
- Regression / resolution: Phase A made no changes; see the Phase C record below.
- Related: historical HCR-P04/P06; same-pattern search found existing guards in
  named resource/Inventory and strict ID equality in Podman anonymous attachment.
  Count-only matching of nonempty sets remains a separate unconfirmed hypothesis.
- Escape: detected S9; earliest S2/S3 (S2 for the missing field, S3 for the Down
  barrier). `NEGATIVE_FIXTURE_GAP`, `ORACLE_COUPLING`, `COMPOSITION_GAP`.
  Earlier opportunity: independently omit each identity field while retaining
  valid ownership labels, then assert the full destructive entry point never
  reaches its effect. Existing identity fixtures coupled IDs to valid records.
- Preventive guardrail: missing/empty/duplicate/wrong-ID response matrices shared
  by provider inspection plus public Down negative effect assertions. Expected
  future detection S2/S3. The Phase C record below documents the guard added after disposition.

## Rejected and bounded hypotheses

- “Podman UDP still dials TCP”: disproved by current suffix filter and full mixed
  protocol regression; a mapping is not an application response.
- “Inventory always requires podman-compose”: disproved by engine-only Doctor
  followed by InventoryFor with absent/stale frontend and all native kinds.
- “Retained anonymous volume can be reported absent”: historical scenario is
  disproved by final-resource existence derivation and retained-only regression;
  this is not a proof against arbitrary concurrent external deletion/recreation.
- “release-verify still admits hidden caller index flags”: disproved by four
  executeArgs regressions before build/output, preserving caller bytes/index/refs.
- “Private clone inherits active global/system clean/smudge filters”: disproved
  by active-transform control fixtures and isolated checkout byte equality.
- “ZIP central-safe/local-unsafe names still pass”: disproved by the current
  local-record parser and direct local-header mutation fixture.
- “Asset cache preflight is the same as release regularRead”: disproved for the
  current asset reader, which uses opened-file Stat, LimitReader and an extra-byte
  check. RELEASE-002 is a recurrence in another package, not an unfixed asset bug.
- Docker mutable context routing, cross-call file replacement, missing diagnostic
  secret sources, and same-count wrong nonempty inspection IDs remain bounded
  hypotheses requiring focused contract/reproduction work. They are not folded
  into the three confirmed observations above.

## Validation and next step

The [historical corpus](history-compose-release.md) records all successful
package/focused race runs and native/candidate limitations. Two temporary Go
overlays added tests only through Go's overlay mapping; Phase A left the repository
free of product changes. The release overlay uses only native Go temporary files,
and the Compose overlay uses an injected runner: no Docker/Podman mutation.

The Phase B checkpoint accepted all three findings. The historical coverage
limitations and unconfirmed hypotheses remain distinct from those decisions.

## Phase C resolution and validation

- **AUDIT-OWNERSHIP-001:** `dockerClient.Inspect` now rejects a missing/empty
  container ID before collecting observations or reaching Down. Permanent
  `TestMissingContainerIdentityRefusesInspectionAndDown` exercises both public
  Inspect/Down entries and both omitted/explicit-empty fields, requiring actual
  inspection, the identity diagnostic, and zero destructive dispatch. All four
  cases failed against the pre-fix implementation; Down dispatched and returned
  nil in both destructive cases. This is a provider-level prevention guard (S3).
- **AUDIT-RELEASE-001:** the blanket length cutoff is replaced with a structural
  empty-path/filesystem-root exclusion. Normal roots `/a`, `/ab`, `/abc` now retain
  leak detection, including concatenated literals, while known module identities
  remain excluded. `TestReleaseBinaryShortCheckoutPaths` failed for `/a` and
  `/ab` before the fix and now passes all three real/module pairs (S2).
- **AUDIT-RELEASE-002:** `regularRead` opens the file, verifies the opened handle's
  type/size, then reads at most the configured cap plus one probe byte. Growing
  files return an error without exposing partial data. Archive staging now uses
  that same bounded helper, removing the second unbounded member read.
  `TestReleaseReadBoundsGrowthAfterOpenedStat` uses a real file whose wrapper grows
  it immediately after capturing Stat; there are no sleeps, races or skip paths.
  With the equivalent extracted unbounded reader it failed at caps 0/1/16,
  returning 65,536 bytes each time. The fixed reader rejects every case and the
  test asserts actual consumed bytes never exceed cap+1. Public
  `TestReleaseRegularReadExactBounds` covers empty/exact/one-over inputs (S2/S3).

Native Linux Go 1.27.1 validation after these changes:

- Focused path/read/archive and provider identity race tests, ten repetitions:
  repoctl 2.945s and Compose 1.129s, PASS.
- All `TestRelease*` race tests: PASS, 1.886s; opt-in real candidate test remains
  skipped here and is not claimed as release-build evidence.
- Full Compose race package: PASS, 1.185s.
- Windows/amd64 CGO-disabled test binaries for both packages compile successfully;
  this is compile portability evidence, not native Windows runtime evidence.

No public behavior was broadened to admit uncertain ownership or oversized
input. Broader harness/native/release validation and independent review belong
to the parent audit's integration phase. No commit is made by this slice.
