---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Finding disposition and remediation ledger

[日本語](findings.ja.md) · [Audit index](index.md)

Phase B checkpoint recorded2026-09-09 after all bounded Phase A reports and182 historical rows were complete. The dispositions below are effective. The linked reports retain frozen pre-fix observations and reproductions. This ledger owns final disposition and resolution status.

## Disposition

The disposition for all19 reproduced findings (original15 plus four postmerge supplements) is ACCEPT: each is an in-scope violation or recurrence of an existing safety, identity, boundedness or navigation contract and can be repaired locally. Seven High findings are mandatory; none is silently deferred. Lower-severity symptoms share classes but have separate production callers/oracles, so they are not duplicates. No confirmed finding is REJECT/DEFER/DUPLICATE.

| ID | Severity | Disposition | Rationale and evidence | Resolution |
| --- | --- | --- | --- | --- |
| AUDIT-CLEANUP-001 | High | ACCEPT | [Unconfirmed readiness command permits retry and source cleanup](current-control-plane.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-OWNERSHIP-001 | High | ACCEPT | [Missing container identity permits destructive Down](current-compose-release.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-DOCS-001 | Low | ACCEPT | [Invisible target headings pass fragment validation](documentation.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-RELEASE-001 | Low | ACCEPT | [Short source roots bypass leak detection](current-compose-release.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-RELEASE-002 | Medium | ACCEPT | [Preflight-only read cap fails under file growth](current-compose-release.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-BOUNDARY-001 | Medium | ACCEPT | [Exactly2000 log records falsely report omission](history-mobile.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-REDACTION-001 | Medium | ACCEPT | [Mobile evidence expands beyond post-redaction budgets](current-mobile.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-REDACTION-002 | Medium | ACCEPT | [DOM publication bypasses adapter budget after redaction](current-process-browser.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-STALE-001 | Medium | ACCEPT | [Load wait mixes old snapshot with new document predicate](current-process-browser.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-UI-001 | High | ACCEPT | [Editable suppression prevents subsequent intended input](current-mobile.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-IDENTITY-001 | High | ACCEPT | [Helper provenance omits configured digest](current-mobile.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-REDACTION-003 | High | ACCEPT | [Window secret redaction leaves derived node fingerprints](current-mobile.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-PREREQUISITE-001 | High | ACCEPT | [Empty helper path implicitly adopts current directory](current-mobile.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-BOUNDARY-002 | Medium | ACCEPT | [Fractional Since is silently shortened](current-mobile.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-LIFECYCLE-001 | High | ACCEPT | [Preflight rejection falsely reports unconfirmed native effects](current-mobile.md) | Resolved; tests checking the repair and independent review passed. |
| AUDIT-BOUNDARY-003 | Medium | ACCEPT | [Page creation exceeds128 and blocks later close](supplemental-browser.md) | Resolved; reproduced against the frozen revision, then repair verification tests and independent review passed. |
| AUDIT-STATE-001 | Medium | ACCEPT | [Ignored AX nodes incorrectly satisfy wait predicates](supplemental-browser.md) | Resolved; reproduced against the frozen revision, then repair verification tests and independent review passed. |
| AUDIT-STALE-002 | Medium | ACCEPT | [Pressed-state changes do not invalidate input fingerprints](supplemental-browser.md) | Resolved; reproduced against the frozen revision, then repair verification tests and independent review passed. |
| AUDIT-CLI-001 | Medium | ACCEPT | [Mutation table prints pre-effect page inventory](supplemental-cli.md) | Resolved; reproduced against the frozen revision, then repair verification tests and independent review passed. |

## Decisions and remaining review risks

- DOM final output will preserve the adapter/capabilities1MiB snapshot budget after redaction, consistent with semantic and capture controls. The product's explicit semantic-only sentence is not misquoted as a DOM requirement. Refuse or bound overflow with truthful omission metadata; do not weaken the adapter limit.
- Readiness uses existing persistent command runs for intent/completion and later cleanup gating. A typed unconfirmed-tree/output result cannot be retried or reduced to plain diagnostic text. Unresolved rows require inspection; no automated claim of absence is invented.
- Mobile final PR #5 issues are independently reproduced at the frozen target, not assumed fixed based on prior conversational acknowledgements. Their severity records the observed contract breach without asserting wrong-target input or native destructive effects that were not demonstrated.
- Guardrail-only gaps without a reproduced current defect remain explicit follow-ups, not undispositioned findings: direct Windows PID-reread scheduling, Java producer traversal/fingerprint tests, release close/late-persistence failure injection, and broader native malicious-failure schedules. Risks are unexercised rare paths, not known current defects. Follow-up requires a dedicated ExecPlan if substantial; existing source/native tests remain intact. A synthetic universal OS/ADB fault framework would exceed local repair scope and provide no immediate native proof.
- Same-user hostile filesystem/provider forgery remains outside existing trust assumptions. No new trust relaxation is accepted.

## Verification and independent review

Implementation commits: readiness `dc358e1`, ownership/release `b477aa4`, fragments `b91074f`, Browser `499c5b2`, Android UI `6872286`. Their linked Phase C sections record failing reproductions against the frozen revision and focused tests checking the repaired behavior. Full candidate `repoctl check` and `go test -race ./...` pass (app45.555s); isolated process preview race×10 passes8.205s without changing the test. Docker integration passes (CLI170.949s), latest real UI139.632s, real Emulator44.613s, private-clone six-target release verification (two builds/eight identical files/Linux smoke), and Browser native CI34294068663 on all3OS pass. Verify34294068659 and Podman99.951s also pass. Independent readiness/docs, Browser and final mobile reviews pass (mobile bounds race×3 4.352s). Four late postmerge findings are also repaired and independently verified at final candidate f2ec634; their final evidence is below. Root separately inspected Compose/release production code and tests checking the repairs, with no blocker.

Supplemental implementation: CLI `ddf8ae4`, CDP `f2ec634`. Independent CLI review confirmed the operation-error success guard and affected target identity; supplemental CDP race×5 PASS2.039s. Final candidate full race, `repoctl check`, sandbox-enabled Browser native race10.157s and six-target reproducibility verification pass. Browser native34295144958 passes all3OS; Verify34295144985 also passes all native/cross-build/integration jobs. The initially incorrect Japanese supplement metadata was corrected and full docs-check passes.
