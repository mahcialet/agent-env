---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Documentation historical replay and current audit

[日本語](documentation.ja.md)

Frozen target: `031869c8b9073b8e23bc17fbc55243666a52f557`. Phase A only; no production changes.
Sources: completed `bilingual-documentation.md` and `bilingual-documentation-review.md` under `docs/exec-plans/completed/`.
The five PR #3 thread IDs are recorded verbatim in the latter plan.

## Historical corpus

All rows remain applicable. Detected stage is S8; earliest realistic prevention is S2, because the language policy already specified each obligation. Current locations are `tools/repoctl/main.go` and `tools/repoctl/translations.go`. The named tests are in `tools/repoctl/translation_review_test.go` unless indicated otherwise. Tests invoke the full `docsCheck` entry point; they are not helper-only. The selected tests checking rejection of invalid documents and acceptance of valid documents passed with race detection (2.296s).

| ID / original defect | Test checking the repaired behavior | Earlier opportunity and escape | Guardrail / recurrence |
| --- | --- | --- | --- |
| HIST-DOC-01 / source metadata accepted without visible English navigation | `TestTranslationRequiresVisibleSourceLink` | S2 tests rejecting missing backlinks were absent; positive fixtures shared the omission. ORACLE_COUPLING, NEGATIVE_FIXTURE_GAP. S6 therefore accepted invalid documents; S7 did not independently compare every requirement. | Existing separately mutated hidden, empty, wrong and escaped links; inspect fragment validation below. |
| HIST-DOC-02 / root Japanese documents accepted without core metadata | `TestRootTranslationsRequireCoreMetadata` | S2 fixture generation omitted metadata and tests expected acceptance. ORACLE_COUPLING. S6 encoded the same scope assumption; S7 missed root-vs-docs scope. | Existing per-field missing/blank/invalid mutations. Current translation scope includes root and all docs. |
| HIST-DOC-03 / new completed plans could expand translation exceptions | `TestCompletedPlanExceptionCannotExpandMigrationSet` | S2 had a permissive new-exception oracle despite a historical migration boundary. ORACLE_COUPLING, NEGATIVE_FIXTURE_GAP. S6 trusted the registry; S7 missed future additions. | Existing fixed four-plan allowlist plus a test rejecting a new plan added to the translation exceptions. Search confirmed no second exception authority. |
| HIST-DOC-04 / Japanese plans accepted arbitrary or missing required sections | `TestRequiredPlanSectionsMustBeProse`; `TestReviewJapanesePlanRequiresEverySection` in `structure_review_test.go` | S2 accepted arbitrary Japanese headings instead of deleting each required section. ORACLE_COUPLING, NEGATIVE_FIXTURE_GAP. S6 exempted Japanese structure; S7 relied on valid repository files. | Existing per-section deletion, English-equivalent positives, prose filtering. Fragment branch remains separately implemented: AUDIT-DOCS-001. |
| HIST-DOC-05 / canonical and Japanese indexes did not both expose translations | `TestIndexRequiresNavigableJapaneseLinks`; `TestReviewJapaneseDocumentsRequireBothLocalIndexes` in `structure_review_test.go` | S2 omission test removed only one duplicate valid link; fixture still met the requirement. ORACLE_COUPLING. S6 and S7 therefore had misleading green evidence. | Existing full-entry tests independently delete links or substitute hidden links/unused reference definitions and require rejection of documents that no longer meet index requirements; no current missing-index recurrence confirmed. |
| HIST-DOC-06 / textual extraction accepted examples, comments and duplicate reference definitions | `TestTranslationRequiresVisibleSourceLink`, `TestRequiredPlanSectionsMustBeProse` | S2 could compare rendered navigation with raw Markdown using small adversarial fixtures. NEGATIVE_FIXTURE_GAP, COMPOSITION_GAP. S6 parser was shared but not consistently applied; S7 did not examine other consumers. | Existing prose/comment/code-state filtering and first-reference resolution. Same assumption survives in heading-fragment lookup: AUDIT-DOCS-001. |
| HIST-DOC-07 / ordering-only comment filtering misread code containing comments and comments containing fences | Positive code/comment interleavings in `TestTranslationRequiresVisibleSourceLink` | S2 needed valid nested-syntax examples in addition to invalid cases. COMPOSITION_GAP. S7 reviewed individual removals rather than combined parsing state. | Existing state-aware parsing; selected positive regressions still pass. No additional parsing-order defect confirmed. |

The full-entry tests still fail invalid inputs and retain meaningful expected error codes. This replay executed current regressions and inspected their call paths; it did not mutation-test every historical implementation revision. Existing migration prose ambiguities and completed-plan `status: active` values are historical facts, not authorization to rewrite archives.

## AUDIT-DOCS-001

- Severity: Low. Disposition: pending Phase B.
- Invariant: a validated Markdown fragment must name a rendered heading.
- Location: `tools/repoctl/main.go`, `documentStructureCheck` fragment-heading loop.
- Trigger: append a fenced `## Phantom` example to a fixture document, then link to its `#phantom` fragment from another document; synchronize fixture translations so translation errors cannot mask the result.
- Observed fact: full `docsCheck` returns nil although no rendered heading exists.
- Expected: reject the missing anchor with DOC-004.
- Impact: broken durable navigation can pass the documented repository check. No runtime effect.
- Existing coverage: hidden navigation and required headings are tested, but target-fragment headings use a separate raw-line scan.
- Reproducer: temporary Go overlay `TestAuditHiddenFragment`, invoking `fixture`, `pairFixtureDocument`, and `docsCheck`. Against frozen production, `go test -overlay <temporary-overlay> ./tools/repoctl -run TestAuditHiddenFragment -count=1` FAILS (0.004s): “accepted link to fenced heading with no rendered anchor”. No product file was modified.
- Root-cause hypothesis: historical rendered-prose repair was applied to navigation and required sections but not target-fragment lookup.
- Proposed repair: use the same prose interpretation for target headings, preserving duplicate-anchor behavior; cover fenced/commented fake headings and valid real duplicates.
- Repair verification tests / resolution / final verification: pending disposition and Phase C.
- Related findings: HIST-DOC-04, HIST-DOC-06.
- Escape analysis: detected S9; earliest prevention S2. Existing S2 hidden-heading fixtures could have been applied to the other full-entry consumer. S6 still used raw target lines; S7/S8 scoped their check to backlinks/mandatory sections. COMPOSITION_GAP, NEGATIVE_FIXTURE_GAP, REVIEW_CHECKLIST_GAP.
- Proposed preventive guardrail: shared rendered-heading extraction with full-entry tests rejecting fragment links to hidden headings. Expected future detection S2/S6; not implemented during Phase A.

## Phase C resolution and independent review

AUDIT-DOCS-001 is ACCEPT and repaired. Target fragment headings now use `documentProse`, preserving actual duplicate anchors. `TestFragmentsRequireRenderedTargetHeadings` rejects fenced, commented and indented fake targets, then separately adds real and duplicate headings and requires acceptance. Before repair, the implementation accepted links to fenced/commented fake headings, so the tests requiring refusal failed; the indented variant already rejected the invalid target and remains a retained guard. Focused race passed1.071s; independent reviewer repeated the suite checking refusal of invalid links and acceptance of valid links 3 times with race (1.215s) and found no blocker. Full candidate harness remains separate.
