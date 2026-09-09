---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Historical corpus inventory and escape summary

[日本語](historical-corpus.ja.md) · [Audit index](index.md)

The inventory covers all 20 English completed plans present at the frozen target. Japanese siblings were treated as translations, not separate findings. Feature plans were searched for internal reviews and discoveries as well as explicit follow-up plans. The linked annexes identify individual material findings, source passages, current production entries, actual regression assertions and coverage limits; absent thread IDs are not invented. PR #5's24 external comments have been incorporated because its feature plan omits later review rounds. The corpus contains182 material rows (77 mobile,56 MVP/process/browser,42 Compose/release,7 documentation); grouped rows are not a count of individual PR comments.

## Complete plan inventory

| Completed plan | Material findings and current regression mapping | Applicability |
| --- | --- | --- |
| [android-emulator-lease](../../exec-plans/completed/android-emulator-lease.md) | [history-mobile](history-mobile.md) | Applicable; shipped behavior or active regression harness. |
| [android-emulator-review](../../exec-plans/completed/android-emulator-review.md) | [history-mobile](history-mobile.md) | Applicable; shipped behavior or active regression harness. |
| [android-emulator-review-2](../../exec-plans/completed/android-emulator-review-2.md) | [history-mobile](history-mobile.md) | Applicable; shipped behavior or active regression harness. |
| [flutter-android-runtime](../../exec-plans/completed/flutter-android-runtime.md) | [history-mobile](history-mobile.md) | Applicable; shipped behavior or active regression harness. |
| [flutter-android-review](../../exec-plans/completed/flutter-android-review.md) | [history-mobile](history-mobile.md) | Applicable; shipped behavior or active regression harness. |
| [android-ui-observer](../../exec-plans/completed/android-ui-observer.md) | [history-mobile](history-mobile.md) | Applicable; shipped behavior or active regression harness. |
| [agent-env-mvp](../../exec-plans/completed/agent-env-mvp.md) | [history-process-browser](history-process-browser.md) | Applicable; shipped behavior or active regression harness. |
| [persistent-process-runtime](../../exec-plans/completed/persistent-process-runtime.md) | [history-process-browser](history-process-browser.md) | Applicable; shipped behavior or active regression harness. |
| [process-destroy-preview-review](../../exec-plans/completed/process-destroy-preview-review.md) | [history-process-browser](history-process-browser.md) | Applicable; shipped behavior or active regression harness. |
| [browser-cdp-automation](../../exec-plans/completed/browser-cdp-automation.md) | [history-process-browser](history-process-browser.md) | Applicable; shipped behavior or active regression harness. |
| [compose-provider-podman](../../exec-plans/completed/compose-provider-podman.md) | [history-compose-release](history-compose-release.md) | Applicable; shipped behavior or active regression harness. |
| [compose-provider-podman-review](../../exec-plans/completed/compose-provider-podman-review.md) | [history-compose-release](history-compose-release.md) | Applicable; shipped behavior or active regression harness. |
| [standalone-distribution](../../exec-plans/completed/standalone-distribution.md) | [history-compose-release](history-compose-release.md) | Applicable; shipped behavior or active regression harness. |
| [standalone-distribution-review](../../exec-plans/completed/standalone-distribution-review.md) | [history-compose-release](history-compose-release.md) | Applicable; shipped behavior or active regression harness. |
| [standalone-release-finalization](../../exec-plans/completed/standalone-release-finalization.md) | [history-compose-release](history-compose-release.md) | Applicable; shipped behavior or active regression harness. |
| [standalone-release-review](../../exec-plans/completed/standalone-release-review.md) | [history-compose-release](history-compose-release.md) | Applicable; shipped behavior or active regression harness. |
| [standalone-release-filter-review](../../exec-plans/completed/standalone-release-filter-review.md) | [history-compose-release](history-compose-release.md) | Applicable; shipped behavior or active regression harness. |
| [standalone-verify-review](../../exec-plans/completed/standalone-verify-review.md) | [history-compose-release](history-compose-release.md) | Applicable; shipped behavior or active regression harness. |
| [bilingual-documentation](../../exec-plans/completed/bilingual-documentation.md) | [documentation](documentation.md) | Applicable; shipped behavior or active regression harness. |
| [bilingual-documentation-review](../../exec-plans/completed/bilingual-documentation-review.md) | [documentation](documentation.md) | Applicable; shipped behavior or active regression harness. |

## Stage and category aggregation

A finding can have multiple categories; profile counts are not independent defect totals. S0–S9 means specification, design, unit, package, composition, native/provider, harness, author review, independent review, audit. The appendix rows own individual earliest/detected values and concrete opportunities; this aggregate does not overwrite them.

| Earliest stage | Recurring escape pattern | Concrete prevention and current evidence |
| --- | --- | --- |
| S0/S1 | Recovery/uncertainty contracts omitted or effects treated as atomic; SPEC_GAP, INVARIANT_GAP. | Clarify persistent barriers and positive ownership/absence proof before effects. Existing application/build/process/UI barriers; readiness composition recurrence AUDIT-CLEANUP-001. |
| S2 | Positive fixture mirrors implementation; exact equality absent; ORACLE_COUPLING, BOUNDARY_GAP, NEGATIVE_FIXTURE_GAP. | Independent limit−1/limit/limit+1 and field-removal mutations. Current Android log exact-cap and short release root findings show where existing tests stop short. |
| S3/S4 | Helper passes but later layer replaces flags, loses typed errors or expands data; HELPER_ONLY, COMPOSITION_GAP, FAILURE_INJECTION_GAP. | Full Service.Create/UI/Browser/Down or repoctl entry tests with independent durable-result/no-effect assertions. Current readiness, mobile/DOM redaction and missing container identity reproductions use these paths. |
| S5 | Host/provider behavior differs from fakes; NATIVE_EVIDENCE_GAP, CONCURRENCY_GAP. | Independent process/connection and Windows Job/native provider runs. Frozen native CI and Linux real-provider baseline are recorded separately from candidate proof; Flutter tmpfs disk failure remains a failure. |
| S6 | A green harness cannot discover an omitted oracle; HARNESS_GAP. | Guardrails must enter exercised test/check paths, not merely documentation. Translation historical failures and current hidden-anchor recurrence demonstrate this limit. |
| S7/S8 | Review fixes one caller but omits sibling consumers; REVIEW_CHECKLIST_GAP. | Search every caller of the repaired contract, then test composition at the effect/evidence boundary. Historical repeated PR rounds motivated this audit; individual reviewer intent is not inferred. |
| S9 | Current audit detects surviving/recurrent variants. | Record repro before remediation and independent re-review after repair. No claim that audit eliminates all future defects. |

All listed categories are represented in the annex classifications except where a source lacks evidence for a specific pre-discovery assertion. Existing direct-regression gaps are distinguished from product defects: Windows second PID read, Java helper producer completeness/fingerprint, selected process readiness and browser selection tests, late release persistence failures. They do not silently become accepted correctness findings. Phase B must record risks and follow-up for any broader controls left outside this repair; no automatic change to global instructions is proposed.

## Late external-comment supplement

The initial inventory above contained 182 numbered rows. Final GitHub
reconciliation discovered four postmerge PR #10 comments absent from the completed
plan. Adding these explicit external-source rows brings the reconciled corpus to
**186**; prior per-plan rows and historical severity are unchanged.
Detailed defect/stage/control evidence is in the [Browser supplement](supplemental-browser.md)
and [CLI supplement](supplemental-cli.md). All four reproduced at frozen
031869c8 and were ACCEPTed within this audit, not a separate PR task.

| Corpus row | Original comment | Audit finding |
| --- | --- | --- |
| EXT-PR10-01 | 3963154175 | AUDIT-BOUNDARY-003 |
| EXT-PR10-02 | 3963154182 | AUDIT-CLI-001 |
| EXT-PR10-03 | 3963154186 | AUDIT-STATE-001 |
| EXT-PR10-04 | 3963154191 | AUDIT-STALE-002 |

These four are additional historical review defects, not a claim that every
unrecorded external comment has always been represented in the archived plans.
The external-thread reconciliation exposed a corpus-coverage gap; link-based
supplementation preserves provenance instead of rewriting the completed plan.
