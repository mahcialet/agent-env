---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Supplemental Browser review: late PR #10 comments

[日本語](supplemental-browser.ja.md) · [Audit index](index.md) · [Ledger](findings.md)

This is part of the existing repository audit, not a separate PR task.
Final corpus reconciliation found four unresolved postmerge PR #10 comments absent
from the completed browser plan. This annex records three CDP findings; the
[fourth CLI result finding](supplemental-cli.md) is recorded separately.
All were reproduced against frozen revision
`031869c8b9073b8e23bc17fbc55243666a52f557` before the coordinator accepted
their repairs at the supplemental Phase B checkpoint. Original discovery was S8;
audit reproduction is S9. No thread reply/Resolve or remote change was performed
for this annex.

## Reproduction isolation and common evidence

Every tracked CDP Go source/test file was loaded from the frozen Git revision
into a temporary Go overlay. Additional tests used the actual public
`Client.Observe` or protocol-component `wait`/`act` entry points and
existing protocol fixtures; audit production files were not modified during
Phase A. All three regressions failed. Permanent tests also failed on the audit
candidate before these fixes, so earlier audit repairs did not resolve them.
Protocol input dispatch is proven by recorded mock calls, not a claim that a
real browser received a harmful click during this safe replay.

## AUDIT-BOUNDARY-003 — Page creation crosses the supported page limit

- Severity: Medium; disposition: ACCEPT.
- Source: [PR comment 3963154175](https://github.com/mahcialet/agent-env/pull/10#discussion_r3963154175).
- Invariant: a bounded resource-creation operation must refuse before effect when
  its result would exceed the supported resource count.
- Location: `internal/browser/cdp/client.go`, pages enumeration and page-create.
- Trigger: a valid owned browser already has exactly 128 page targets.
- Observed: `Client.Observe(page-create)` succeeds and dispatches
  Target.createTarget, yielding 129 pages. A subsequent explicit page-close
  fails at the existing enumeration limit and sends no close command.
  The adapter has made its own supported management path unusable.
- Expected: refuse creation before Target.createTarget at 128; creating from
  127 to 128 and closing an existing page at 128 remain supported.
- Reproducer: `TestSupplementPageLimitPreventsIrrecoverableGrowth`, full
  Observe with valid browser PID/flags/discovery and a stateful target census.
  Frozen output showed create error nil, close error page-limit exceeded,
  create calls 1, close calls 0, page count 129.
- Escape: earliest S3, COMPOSITION_GAP + BOUNDARY_GAP + NEGATIVE_FIXTURE_GAP.
  Enumeration's correct cap did not constrain the later creation effect.
  Checking over-limit listing alone did not exercise creation at the boundary.
- Phase C: one shared 128-page constant now governs enumeration and pre-create
  validation. No new target is dispatched at/above capacity.
- Guardrail: permanent 127/128/129 table asserts creation and close call counts
  and resulting census, so nearby success and unchanged over-limit refusal are
  both exercised. Future expected detection: S3.
- Limitation: externally supplied 129-page browsers still fail the pre-existing
  bound. This repair prevents this adapter from creating that state; it does not
  add unbounded target discovery or change recovery scope.

## AUDIT-STATE-001 — Ignored AX nodes satisfy accessibility wait predicates

- Severity: Medium; disposition: ACCEPT.
- Source: [PR comment 3963154186](https://github.com/mahcialet/agent-env/pull/10#discussion_r3963154186).
- Invariant: text/role and gone waits use accessible semantic nodes, not nodes
  Chromium marks ignored. Partial evidence still cannot prove absence.
- Location: `internal/browser/cdp/client.go:wait`, text/gone node scan.
- Trigger: snapshot contains only an ignored node matching both role and name.
- Observed: text wait falsely succeeds; gone wait times out because that ignored
  node is counted as a match. Snapshot correctly preserves the Ignored bit.
- Expected: ignore such nodes for both predicates; preserve existing truncated
  gone refusal and ordinary accessible matching.
- Reproducer: `TestSupplementIgnoredAXCannotSatisfyWait` calls wait with an
  ignored button named Needle and verifies AX retrieval was reached. Frozen text
  returned success; frozen gone returned deadline exceeded.
- Escape: earliest S3, COMPOSITION_GAP + INVARIANT_GAP + NEGATIVE_FIXTURE_GAP.
  Snapshot retained accurate state and actions refused ignored input, but the
  separate wait consumer did not apply that semantic eligibility rule.
- Phase C: skip Ignored nodes before role/name matching. The complete-snapshot
  condition for gone is unchanged.
- Guardrail: protocol regression covers both text false-positive and gone
  false-negative; existing visible-text and truncated-gone regressions remain.
  Future expected detection: S3. No DOM selector or arbitrary script fallback added.

## AUDIT-STALE-002 — Pressed-state changes do not invalidate action fingerprints

- Severity: Medium; disposition: ACCEPT.
- Source: [PR comment 3963154191](https://github.com/mahcialet/agent-env/pull/10#discussion_r3963154191).
- Invariant: a semantic action must refuse when meaningful target state changes
  after the requested snapshot, even if role/name/backend/bounds stay identical.
- Location: `internal/browser/cdp/snapshot.go`, state allowlist feeding the node
  fingerprint; `actions.go` consumes that fingerprint.
- Trigger: a toggle button's pressed state changes after snapshot and at the
  native ownership-verification callback immediately before input validation.
- Observed: pressed was absent from States/fingerprint; stale click input was
  dispatched and act returned performed=true with nil error.
- Expected: false/true/mixed pressed transitions invalidate the stored fingerprint
  before any input. This does not establish a wrong-browser or different-node attack.
- Reproducer: `TestSupplementPressedStateRefusesStaleInput`, copied from the
  live ownership-mutation regression with pressed as the changed property.
  It asserts mutation callback reached and counts input dispatch; frozen code fails.
- Escape: earliest S3, NEGATIVE_FIXTURE_GAP + ORACLE_COUPLING + COMPOSITION_GAP.
  Existing checked-state coverage did not establish that the separate pressed
  protocol property was retained. Fingerprint correctness depends on the producer's
  property set, not hashing implementation alone.
- Phase C: add pressed through the existing bounded `axState` conversion.
  Review confirmed checked, selected, expanded, readonly, required, focusable,
  focused and multiselectable were already retained. No free-form value,
  description or relationship text was added to fingerprints.
- Guardrail: the permanent table covers pressed boolean/mixed transitions and
  all eight existing allowed flags, with injection-reached and zero-input assertions.
  Future expected detection: S3. Existing arbitrary-state-text rejection remains.

## Phase C validation and remaining gates

Focused supplemental protocol regressions with race detection and three
repetitions: PASS 1.668s. Independent read-only review found no blockers, and
its five race repetitions passed in 2.039s. This includes page 127/128/129, ignored text/gone
predicates and ten state-transition cases. The coordinator additionally reported
the integrated CDP race suite PASS 8.446s and real sandbox-enabled Browser native
race PASS 10.157s. Final candidate
harness/native/CI acceptance remains owned by the coordinator and ledger;
these results do not by themselves archive the audit.

No broader generic checker was added: the existing public protocol fixtures now
exercise resource growth, semantic eligibility and state provenance separately.
Their controls complement the earlier exact-count, origin and post-redaction
tests rather than assuming those tests cover every downstream consumer.
