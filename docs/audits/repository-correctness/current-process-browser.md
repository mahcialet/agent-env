---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Current process, browser and executor correctness review

[日本語](current-process-browser.ja.md) · [Audit authority](../../exec-plans/completed/repository-correctness-audit.md) · [Historical corpus](history-process-browser.md)

Frozen target: `031869c8b9073b8e23bc17fbc55243666a52f557`.
Phase A reproductions used Go overlays outside the repository. The coordinator
accepted both findings at the Phase B checkpoint (`56b9c2c`); Phase C repairs and
tests checking the repairs are recorded below. This report completes the bounded source/reproducer pass
for process runtime, Browser/CDP and execx; it does not replace the global audit
matrix, native baseline or independent final review.

## Reviewed invariant matrix

| Area / invariant | Inspected entry points and evidence | Result / limitation |
| --- | --- | --- |
| Browser authority and isolation | CDP connect/discovery, exact loopback WebSocket, browser PID/version/flags, native callback, selected target sessions and page limits | Existing tests that reject browser adoption on identity mismatch remain live; source review found no new wrong-browser adoption path. Same-user malicious protocol forgery remains outside the contract. |
| Browser frame and stale-node proof | snapshot/approvedFrames/recheckFrames, frame-owner census, act final AX/node/hit checks, isolated-world focus, native closed-shadow fixture | Before/after origin/topology and stale-input guards retained. Load wait has the mixed-document gap below; later input still revalidates. |
| Browser exact limits/completeness | 2048 AX/DOM nodes, 32 frames, 128 pages, 512 targets/events, 1 MiB semantic JSON, 4 KiB fields, 64 KiB capture strings, 256 records | Tests of AX omission reporting cover 2047/2048/2049 and multiple/empty trailing frames. DOM loop tests omission on the next node, not equality alone. Post-redaction DOM inconsistency below. No claim that every numeric combination was executed here. |
| Capture duration and omitted evidence | capture context before subscribe/enable, per-event deadline checks, atomic unsubscribe pending/overflow, omitted console argument flags | Existing tests of deadlines, queue overflow and omitted console arguments pass. Helper-only deadline-return coverage limitation remains in historical HB18. |
| Browser mutation/evidence barriers | Observe confirmation classification, action_performed before input/focus, app durable run/provenance and typed redaction | Existing post-effect uncertainty and persistence tests retained. General app fence/finalization audit remains coordinator-owned. |
| Process launch identity and recovery | Prepare -> Start -> identity/receipt, typed no-spawn, receipt identity mismatch and recovered identity | Incomplete native identity remains uncertain; known no-spawn requires typed proof plus zero PID. Missing receipt with returned full identity still permits compensation; mismatched proof refuses it. |
| Process paths and cleanup | validateOwner/validatePaths, canonical reserved paths, Destroy -> native Terminate -> Observe -> state removal | Whole-tree absence is rechecked; path/symlink mismatches and unexpected files refuse. Windows retry only covers sharing violations with path proof each attempt. Generic filesystem TOCTOU review belongs to the coordinator; trusted-code limitation is preserved. |
| Process ports/readiness/logs | held OS dynamic reservations, prelaunch bind checks, final readiness observation, independent startup redaction fingerprints | Existing collision/readiness/dead-root/receipt-secret tests remain live. External bind race after releasing reservation listeners is documented, not silently solved. Command-readiness cleanup recurrence is coordinator finding AUDIT-CLEANUP-001. |
| Unix native ownership/cancellation | detached birth identity and group proof, revalidation before signals, bounded census retries, managed absence waits | No new managed-process false-absence finding confirmed. Atomic PID/group signaling is unavailable and documented. Ordinary Runner's successful signal does not itself census descendants; hypothesis remains unconfirmed below. |
| Windows native ownership/cancellation | suspended Job assignment, session/nonce/birth, completion marker before absence, historical PID precedence, exact Job handle termination and empty-Job wait | Historical identity repairs retained. Windows source/tests inspected; current Windows execution belongs to baseline CI, not this Linux review. |
| Cross-platform failure propagation | OSRunner ExitError unwrap, ErrProcessTreeUnconfirmed, ErrOutputIncomplete; named command and readiness consumers | Confirmed readiness caller loses typed uncertainty, independently supporting AUDIT-CLEANUP-001; do not duplicate that finding. |

## AUDIT-REDACTION-002 — DOM evidence escapes the adapter's final-size limit

- Severity: Medium (bounded-evidence inconsistency; specification qualification below).
- Disposition: ACCEPT.
- Invariant: evidence transformation must preserve the intended final artifact
  budget or explicitly disclose/reject its overflow.
- Location: `internal/app/browser.go`, DOM branch after `redactBrowserJSON`
  and before `save("browser-dom", ...)`; compare
  `internal/browser/cdp/snapshot.go:domSnapshot` and
  `internal/app/browser_redaction.go:boundRedactedBrowserSnapshot`.
- Trigger: record set-text fingerprint for short text `qz`; observe 2048 DOM
  nodes with 126-byte legal custom-element names (`qz-` repeated 42 times).
  Provider DOM is 411,520 bytes, below the adapter's 1 MiB and 128-byte name caps.
- Observed behavior: full `Service.Browser(dom-snapshot)` persists a
  1,099,648-byte `browser-dom` artifact with `Truncated=false` and a passed
  durable run. No secret is leaked: the defect is final boundedness, not redaction
  failure. The generic save helper accepts up to 16 MiB.
- Expected behavior: reconcile the intended DOM final-output budget and enforce
  it after redaction; never imply the adapter's 1 MiB limit still holds when it does not.
- Impact: callers relying on adapter/capabilities snapshot-size limits receive
  larger complete-labeled artifacts. This does not authorize semantic input from DOM.
- Existing coverage: semantic post-redaction bounds and console/network persisted
  bounds are tested, but no corresponding final DOM-byte assertion exists.
- Reproducer: isolated Go overlay adds `TestAuditDOMPostRedactionBudget` to
  `internal/app/browser_review_test.go`, using existing `browserFixture`.
  Call snapshot, then set-text `qz`, inject confirmed DOM with the 2048 nodes
  described above, call dom-snapshot, require a registered DOM artifact and assert
  its encoded length is at most 1 MiB. Actual result: FAIL, 0.079s, with the exact
  input/output lengths above. Product files remain untouched.
- Tests checking the repair: `TestBrowserDOMBoundsAfterRedaction` and
  `TestBrowserDOMEncodedBoundary` now exercise persisted output and exact limits.
- Resolution: Phase C now bounds redacted DOM encoding to 1 MiB by retaining a
  whole-node prefix. It preserves retained identity fields and metadata, marks
  both artifact and observation truncated only after actual omission, and
  completes the read-only run. Metadata alone above the cap is rejected.
- Verification: deterministic full app reproduction, actual artifact read and
  returned passed run/truncation checks. No native Chrome reproduction was required
  to demonstrate the app transformation; the legal custom-name shape avoids
  relying on arbitrary unknown DOM fields.
- Related findings: historical HB16/HB20 (same transform-after-bound class).
- Escape analysis: detected S9; earliest S4, because app already owns both
  redaction and artifact publication. COMPOSITION_GAP + BOUNDARY_GAP:
  previous repairs measured final semantic/capture output but did not enumerate
  every persisted browser artifact kind. Missing oracle is final DOM byte length.
  Preventive control: a per-artifact final-encoding budget matrix, expected S4.
  Implementation/evidence: final DOM guard and public persisted-artifact regression
  now pass; the broader matrix remains coordinator-owned.

Qualification: the product text explicitly promises **semantic JSON** at most
1 MiB; it describes DOM as bounded but does not separately state a final DOM
number. The adapter rejects DOM JSON above 1 MiB, while capabilities reports
“1 MiB snapshots.” The reproduced mismatch is certain; whether the durable
contract should impose the same final DOM cap is a disposition decision.
Do not misquote the semantic-only sentence as an explicit DOM specification.

## AUDIT-STALE-001 — Load wait can return evidence from a different document

- Severity: Medium.
- Disposition: ACCEPT.
- Invariant: a successful wait's evidence must describe the document on which its
  successful predicate was established.
- Location: `internal/browser/cdp/client.go:wait`, `case "load"`.
- Trigger: navigation changes the document after `snapshot` completes its own
  before/after proof but before `Runtime.evaluate(document.readyState)`.
- Observed behavior: readyState `complete` from the new document causes success
  returning the old snapshot/document token. The load branch does not compare
  document identity after evaluating; the URL branch already does.
- Expected behavior: bind load-state proof to the snapshot document, or retry
  changed identity within the existing deadline.
- Impact: successful load wait reports stale semantic evidence. Subsequent input
  still has independent document validation, so this reproduction does not show
  stale input hitting a different node.
- Existing coverage: frame changes during snapshot and URL-wait identity checks
  are covered. No test checks document changes across the additional load-evaluation step.
- Reproducer: isolated Go overlay adds `TestAuditLoadWaitDoesNotMixDocuments`
  to `internal/browser/cdp/snapshot_test.go`. A protocol fixture reports root
  loader `old` during snapshot, changes to `new` when Runtime.evaluate runs
  and returns `complete`. It asserts the mutation was reached, then compares
  the returned snapshot token with a fresh frameDocument token. FAIL, 0.004s:
  returned `main:old:<same-URL-digest>`, current `main:new:<same-URL-digest>`.
- Tests checking the repair: `TestLoadWaitRechecksDocumentAfterPredicate` now exercises single
  and repeated navigation at the predicate boundary.
- Resolution: Phase C rechecks the root document identity after a complete load
  predicate and retries mismatches within the existing deadline, matching the URL
  wait strategy. Continuous navigation returns no snapshot on timeout.
- Verification: deterministic protocol-component reproduction. Native timing
  reproduction not run; unlike a wrong-protocol mock, the test uses documented
  loader changes and ordinary readyState results, not invented CDP fields.
- Related findings: historical HB10/HB17 (identity checks around multi-call observation).
- Escape analysis: detected S9; earliest S3. CONCURRENCY_GAP + COMPOSITION_GAP:
  snapshot's own proof was assumed to cover the later predicate call.
  Existing snapshot mutation tests stop before that extra step. Preventive control:
  enumerate every multi-call wait predicate and inject navigation between observation
  and predicate; expect S3. The new test preventing stale evidence after navigation passes with the product repair.

## Related executor evidence for AUDIT-CLEANUP-001

This review independently confirms the typed executor contract behind the
coordinator's readiness reproducer. Unix `runProcessTree` joins
`ErrProcessTreeUnconfirmed` when group termination fails after Wait.
Windows joins it on Job termination failure, incomplete empty-Job verification
and abort cleanup failure. `OSRunner` wraps these in an unwrap-capable ExitError.
`runCaptured` joins `ErrOutputIncomplete` on output pump/drain failure.
These are native failure modes, not arbitrary injected error strings.

`app/readiness.go:runProbe` stores these as `last`, retries, and eventually
formats the error with `%v`, unlike named-command finalization's typed running
barrier. The coordinator reproduced retries, source removal and RELEASED at the
full Create entry point. This is the same missing caller-level safety contract as
historical MVP HM10/HM11; keep the root finding canonical.

## Rejected or unconfirmed hypotheses and review limits

- **Exact-limit DOM false truncation:** not reproduced from source control flow.
  The cap test runs only when there is another node to omit; an empty following
  document does not itself set truncation. Name shortening can legitimately mark
  truncation and stop further document output. No exact-limit DOM defect claimed.
- **Managed process successful terminate permits unchecked removal:** rejected by
  source trace. `Client.Destroy` calls native Observe after Terminate and refuses
  alive/error before path-validated state removal.
- **Closed-shadow proof trusts page overrides:** rejected by existing native
  fixture and isolated-world resolution. Target-outward traversal checks every
  enclosing root/host, and tests reject input to targets obscured by overlays for all four input operations.
- **Ordinary Unix Runner signal success equals whole-tree absence:** source shows
  post-Wait SIGKILL success/ESRCH accepted without a process-group census, unlike
  Windows empty-Job checks and managed-process waits. Existing late-write tests
  check after one second. A concrete harmful remaining-process window was not
  reproduced; no new confirmed finding/severity is assigned here. This remains
  an explicit investigation limitation, not a claim of proven safe absence.
- **All historical regressions cover public composition:** rejected. HP05/HB19
  are helper-only; HB18 is helper-level completion coverage; HM06 still needs
  exact historical test mapping. Reconcile currently does check the durable
  `lease_ready` event before promotion, so missing mapping is not proof the
  original product defect survives.
- Windows/macOS execution, Docker/Podman/Android integration and global
  persistence/fence/path invariants are coordinator-owned. This source pass
  does not substitute cross-build or historical green CI for those current gates.

## Local verification

Selected historical race replay passed with current execution results recorded in
the historical annex. A subsequent full race invocation for process, execx and
CDP returned PASS from Go's cache; it is a cache reuse, not a new execution.
The two overlay reproductions failed as expected and changed no repository
product/test file. No remediation occurred during Phase A; Phase C changes are recorded below.
No commit, push or thread operation was performed by this reviewer.

## Phase C regression evidence

This section records permanent tests that recheck the repaired behavior and their execution results.

Permanent tests of saved DOM size and load-wait document consistency were first run against the unchanged product code and
failed: the 2048-node DOM case persisted 1,099,648 bytes, while load wait returned
stale success for both single and continuously changing documents.

- `TestBrowserDOMBoundsAfterRedaction` uses the public app entry point, actual
  persisted artifacts and durable run lookup. A 1000-node positive case stays
  complete; 2048 nodes require truthful truncation. It asserts no secret remains
  and every retained index/backend ID stays intact.
- `TestBrowserDOMEncodedBoundary` checks final JSON at 1 MiB minus one, exactly
  1 MiB, and 1 MiB plus one. The oversized case must omit a node, not merely turn
  `false` into the one-byte-shorter `true` and falsely label complete data partial.
- `TestLoadWaitRechecksDocumentAfterPredicate` asserts navigation injection was
  reached. One change requires a second predicate evaluation and returns only the
  new document's evidence; continuous change must return an error and nil snapshot.
- Focused CDP load-wait tests with race detection, three repetitions: PASS 1.962s.
  Focused tests of DOM saved by the app with race detection, three repetitions: PASS 7.938s.
  Earlier attempts encountered concurrent compile errors in other assigned files;
  those attempts are not counted as passes.
