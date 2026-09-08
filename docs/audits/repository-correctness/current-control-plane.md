---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Control-plane current audit

[日本語](current-control-plane.ja.md)

Phase A at `031869c8b9073b8e23bc17fbc55243666a52f557`; source inspection and isolated overlay reproductions only. Full baseline unit/race/real-provider results are complementary evidence, not proof of every invariant.

## Subsystem and invariant review

| Subsystem | Reviewed invariants / production entry | Evidence and result |
| --- | --- | --- |
| Manifest and plan | Strict decode, portable relative paths, duration and argv validity, canonical digest, selected stack closure, provider snapshots; `config.Parse/Validate`, `app.BuildPlan` | Existing negative config/application/browser/process fixtures; all config tests included in baseline race. Unknown fields, aliases, case-colliding runtime names, source traversal, duplicate APK outputs rejected. No new defect confirmed. |
| Domain and policy | Immutable source tuples, temporal GC bounds, explicit quarantine; `SourceDigest`, `GCEligibleWithGrace`, policy TTL | Domain GC boundary tests cover equality and zero/negative input. `Transition` helper is not a universal enforcement boundary: orchestration also assigns states explicitly, so those assignments were reviewed with effects below. |
| SQLite registry | Transactional lease/source/process/Android reservation persistence, immutable prepared identity, rollback, lease-scoped rows; `Create/Save`, reservation helpers | SQL transaction and normalized-row checks; prior nonzero process identity cannot be removed/reassigned; released reservations cannot resurrect. Baseline race includes independent-connection/process tests. No database-only inference of provider absence. |
| Locks and cancellation | Private lock capability, renewal failure cancellation, token and expiry checks before writes, destroy/run coordination; `AcquireContext`, `execBound`, `runWithCancellation` | Existing stolen-lock and subprocess tests reach production boundaries. `context.WithoutCancel` retains capability and cannot bypass token checks. Readiness caller loses the process-tree sentinel: AUDIT-CLEANUP-001. |
| Create and readiness | Persist intent before source/runtime effects, pin rendered config before up, readiness requires Exists AND Ready, per-pending-runtime budget | `Create`, `allocate`, `waitReady`, `probeReady`; mixed-runtime real/fake baseline coverage. Command readiness differs from the protected named-test path: AUDIT-CLEANUP-001. |
| Cleanup / reconcile / GC | Source identity and tracked diff before effects, independent sibling cleanup, persist failure stops saga, provider absence before release | `cleanup`, `Reconcile`, `GC`, `previewDestroy`, `Inventory`; existing sibling/fence/foreign inventory tests. Stored running named tests protect sources; readiness has no equivalent persistent barrier. Reconcile does not rerun repository readiness commands. |
| Git and paths | Detached pinned registered root, repository filesystem identity, symlink escape, missing worktree registration, force only after tracked diff | `gitcli.Inspect/Remove`, `paths.Within`, state-root tests. Removed-source postinspection occurs in orchestration. User-controlled repository contents are not arbitrary private state trust; TOCTOU under malicious concurrent replacement remains outside demonstrated guarantees. No new path defect reproduced. |
| Commands and evidence | Named test intent row before start, terminal row after complete evidence and tree termination, redaction across chunks, regular-file artifacts, symlink containment | `Service.Test`, `collectTestArtifact`, `evidence.Redactor/AtomicWrite`. Existing stream-error and cancellation barriers apply to named tests; command readiness does not share them. Redactor chooses longest match and never rescans replacement; contextual caps belong to callers (mobile recurrence separately recorded). |
| Endpoints and health | Protocol-preserving endpoint keys, current provider inspection, process port expansion, HTTP status and timeout, no command health replay | `Endpoints`, `probeHealth`, endpoint-config tests. TCP and UDP keys remain distinct; Docker/Podman baseline covers provider boundary. No new confirmed endpoint defect. |
| Documentation | Visible rendered anchors, bilingual metadata and indexes, historical scope exclusions | Full-entry historical regression replay; AUDIT-DOCS-001 in [documentation audit](documentation.md). Broader portability/native claims checked against CI evidence; candidate native runs remain pending. |

This table records bounded review, not exhaustive enumeration of every possible schedule or malicious filesystem race. Mobile, process/browser and release/provider annexes cover their own adapter internals.

## AUDIT-CLEANUP-001

- Severity: High. Disposition: pending Phase B.
- Invariant: an unconfirmed command process tree must stop retry/effects and retain its source and durable cleanup barrier.
- Location: `internal/app/readiness.go` (`runProbe`), allocation error handling in `internal/app/lifecycle.go`.
- Trigger: configured command readiness returns `execx.ErrProcessTreeUnconfirmed`; it can originate from real Unix group termination failure or Windows Job termination/empty-job failure. `ErrOutputIncomplete` follows the same lossy caller path.
- Observed fact: full `Service.Create` retries the unsafe command seven times, removes the source and reports `released`; the returned error is only context deadline exceeded after the expired probe artifact write. Other timings stringify the last cause with `%v`, also losing `errors.Is` classification.
- Expected: no retry after incomplete cleanup/evidence; preserve typed safety classification, quarantine and retain the worktree. Later explicit destroy/GC must not bypass the unresolved command barrier.
- Impact: a potentially live child can lose its working tree; a later successful retry can hide earlier uncertainty. This is a safety defect, not only a diagnostic omission.
- Existing coverage: MVP named-test incomplete-tree/stream tests and allocation typed-sentinel guard exist, but there was no command-readiness full-create failure injection. Process readiness-secret regression only validates configuration.
- Reproducer: temporary Go overlay `TestAuditReadinessUnconfirmedTree`, using the normal lifecycle fixture and a Runner that returns the production typed sentinel. `go test -overlay <temporary-overlay> ./internal/app -run TestAuditReadinessUnconfirmedTree -count=1 -v` fails (0.024s): `state=released sources=0 calls=7`, operations end in `remove:self`. No product files changed.
- Independent corroboration: executor reviewer traced OSRunner's ExitError wrapping to real Unix and Windows error paths; the injected result is part of the production contract.
- Root cause: retry treats unsafe cleanup as an ordinary failed probe; error redaction/timeout handling drops classification and no durable command record protects subsequent cleanup.
- Proposed repair: preserve safety sentinels while redacting diagnostics, halt retry immediately, and persist a readiness command intent/terminal barrier using existing run records so later destroy also remains safe. Avoid inventing an automatic absence proof.
- Regression / resolution / final verification: pending Phase B/C. Cover both unsafe sentinels, secret-safe errors, no retries, source retention and later destroy; retain successful/ordinary-failure readiness behavior.
- Related: MVP incomplete tree/output capture historical fixes (HM10/HM11 in the process/browser historical annex); helper-only readiness coverage gap.
- Escape: detected S9; earliest realistic prevention S4. The typed Runner contract and Create quarantine branch already existed, so a full-create failure injection was feasible. S2/S3 tested executor and named-test barriers separately; S6 ran those without a caller composition test; S7/S8 did not trace every consumer of the safety sentinel. COMPOSITION_GAP, FAILURE_INJECTION_GAP, HELPER_ONLY, REVIEW_CHECKLIST_GAP.
- Preventive control: shared safety classification and table-driven full-create negative scenarios, plus persistent command cleanup gating; future detection S3/S4 before S6. Pending implementation; no global policy changes.
