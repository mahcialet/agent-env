---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Finding disposition and remediation ledger

[日本語](findings.ja.md) · [Audit index](index.md)

Phase B checkpoint recorded2026-09-09 after all bounded Phase A reports and182 historical rows were complete. The dispositions below are effective. The linked reports retain frozen pre-fix observations and reproductions. This ledger owns final disposition and resolution status.

## Disposition

The disposition for all15 reproduced findings is ACCEPT: each is an in-scope violation or recurrence of an existing safety, identity, boundedness or navigation contract and can be repaired locally. Seven High findings are mandatory; none is silently deferred. Lower-severity symptoms share classes but have separate production callers/oracles, so they are not duplicates. No confirmed finding is REJECT/DEFER/DUPLICATE.

| ID | Severity | Disposition | Rationale and evidence | Resolution |
| --- | --- | --- | --- | --- |
| AUDIT-CLEANUP-001 | High | ACCEPT | [Unconfirmed readiness command permits retry and source cleanup](current-control-plane.md) | Pending regression and remediation. |
| AUDIT-OWNERSHIP-001 | High | ACCEPT | [Missing container identity permits destructive Down](current-compose-release.md) | Pending regression and remediation. |
| AUDIT-DOCS-001 | Low | ACCEPT | [Invisible target headings pass fragment validation](documentation.md) | Pending regression and remediation. |
| AUDIT-RELEASE-001 | Low | ACCEPT | [Short source roots bypass leak detection](current-compose-release.md) | Pending regression and remediation. |
| AUDIT-RELEASE-002 | Medium | ACCEPT | [Preflight-only read cap fails under file growth](current-compose-release.md) | Pending regression and remediation. |
| AUDIT-BOUNDARY-001 | Medium | ACCEPT | [Exactly2000 log records falsely report omission](history-mobile.md) | Pending regression and remediation. |
| AUDIT-REDACTION-001 | Medium | ACCEPT | [Mobile evidence expands beyond post-redaction budgets](current-mobile.md) | Pending regression and remediation. |
| AUDIT-REDACTION-002 | Medium | ACCEPT | [DOM publication bypasses adapter budget after redaction](current-process-browser.md) | Pending regression and remediation. |
| AUDIT-STALE-001 | Medium | ACCEPT | [Load wait mixes old snapshot with new document predicate](current-process-browser.md) | Pending regression and remediation. |
| AUDIT-UI-001 | High | ACCEPT | [Editable suppression prevents subsequent intended input](current-mobile.md) | Pending regression and remediation. |
| AUDIT-IDENTITY-001 | High | ACCEPT | [Helper provenance omits configured digest](current-mobile.md) | Pending regression and remediation. |
| AUDIT-REDACTION-003 | High | ACCEPT | [Window secret redaction leaves derived node fingerprints](current-mobile.md) | Pending regression and remediation. |
| AUDIT-PREREQUISITE-001 | High | ACCEPT | [Empty helper path implicitly adopts current directory](current-mobile.md) | Pending regression and remediation. |
| AUDIT-BOUNDARY-002 | Medium | ACCEPT | [Fractional Since is silently shortened](current-mobile.md) | Pending regression and remediation. |
| AUDIT-LIFECYCLE-001 | High | ACCEPT | [Preflight rejection falsely reports unconfirmed native effects](current-mobile.md) | Pending regression and remediation. |

## Decisions and remaining review risks

- DOM final output will preserve the adapter/capabilities1MiB snapshot budget after redaction, consistent with semantic and capture controls. The product's explicit semantic-only sentence is not misquoted as a DOM requirement. Refuse or bound overflow with truthful omission metadata; do not weaken the adapter limit.
- Readiness uses existing persistent command runs for intent/completion and later cleanup gating. A typed unconfirmed-tree/output result cannot be retried or reduced to plain diagnostic text. Unresolved rows require inspection; no automated claim of absence is invented.
- Mobile final PR #5 issues are independently reproduced at the frozen target, not assumed fixed based on prior conversational acknowledgements. Their severity records the observed contract breach without asserting wrong-target input or native destructive effects that were not demonstrated.
- Guardrail-only gaps without a reproduced current defect remain explicit follow-ups, not undispositioned findings: direct Windows PID-reread scheduling, Java producer traversal/fingerprint tests, release close/late-persistence failure injection, and broader native malicious-failure schedules. Risks are unexercised rare paths, not known current defects. Follow-up requires a dedicated ExecPlan if substantial; existing source/native tests remain intact. A synthetic universal OS/ADB fault framework would exceed local repair scope and provide no immediate native proof.
- Same-user hostile filesystem/provider forgery remains outside existing trust assumptions. No new trust relaxation is accepted.

## Verification and independent review

Pending. Each owner must record the failing pre-fix regression, focused result and code/test location here or in the linked report before resolution. Full repoctl, race, integration, native candidate CI, final independent review and plan acceptance reconciliation remain required. Baseline successes do not replace candidate evidence.
