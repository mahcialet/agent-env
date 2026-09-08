---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Current Compose, assets and release correctness review

[日本語](current-compose-release.ja.md) · [Audit index](index.md) ·
[Historical corpus](history-compose-release.md) ·
[Execution authority](../../exec-plans/active/repository-correctness-audit.md)

Phase A target: `031869c8b9073b8e23bc17fbc55243666a52f557`.
Review and temporary Go overlays only; no production/test changes, commits,
provider mutations or publication. Findings await Phase B disposition.

## Coverage matrix

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

- Severity: Low. Disposition: not yet assigned (Phase A).
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
- Regression / resolution: not added / not changed during Phase A.
- Verification: failing overlay `TestAuditShortReleaseRootLeak`; no normal test
  files changed. Related: historical HCR-R13/R14, current path/boundary audit.
- Escape: detected S9; earliest S2. `BOUNDARY_GAP`, `NEGATIVE_FIXTURE_GAP`.
  Earlier opportunity: requirement-specific path lengths around the early-return
  boundary. Existing fixtures shared only longer directory examples, so neither
  helper nor real-binary test exercised this branch.
- Preventive guardrail: structural root cases plus length-adjacent valid roots and
  genuine/module-path positive/negative pairs. Expected future stage S2.
  Guard implementation/evidence: deferred until disposition.

## AUDIT-RELEASE-002 — preflight file size does not bound the actual read

- Severity: Medium. Disposition: not yet assigned (Phase A).
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
- Regression / resolution: not added / not changed during Phase A. The writer
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
  current implementation and audit-added guard: none pending disposition.

## AUDIT-OWNERSHIP-001 — missing container identity authorizes Down

- Severity: High. Disposition: not yet assigned (Phase A).
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
- Regression / resolution: not added / not changed during Phase A.
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
  future detection S2/S3. No guard added before disposition.

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
overlays added tests only through Go's overlay mapping; the repository remains
free of product changes. The release overlay uses only native Go temporary files,
and the Compose overlay uses an injected runner: no Docker/Podman mutation.

Phase B must decide each confirmed finding and historical coverage limitations
separately. Keep observed defects,
missing tests and unconfirmed hypotheses distinct. After disposition, a regression
must fail before the minimal fix; current failing overlays provide that baseline
without crossing the review-only boundary.
