---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Restructure durable documentation for reader-first English and Japanese

[日本語](reader-first-documentation-restructure.ja.md)

This ExecPlan is a living document. Maintain it according to `docs/PLANS.md`.

Expected branch: `docs/reader-first-documentation-restructure`.

Starting revision: `084da57de177c0a09bc3cb61ae99faff8bd79a94`

Before editing, fast-forward `master`, record the exact starting revision, create
the dedicated branch, and run the documentation baseline.

## Purpose / Big Picture

Restructure the repository's durable human-facing documentation so it is easier
to understand, navigate, and maintain in both English and Japanese.

The problem is broader than `docs/roadmap.md`.

Many current documents are factually useful but read like compressed design
notes:

- several unrelated capabilities are packed into one sentence;
- implemented, deferred, unsupported, and undecided behavior are mixed in one
  paragraph;
- implementation evidence is repeated where a link would be enough;
- English noun phrases accumulate until the reader must parse the sentence like
  a specification;
- Japanese often preserves the English clause order too closely, producing a
  correct but unnatural translation;
- the same fact appears in README, Architecture, roadmap, product specs, design
  docs, policy docs, and ExecPlans with different levels of detail;
- readers cannot always tell which document is the entry point and which is the
  detailed authority.

This plan changes the information architecture and prose together.

The target documentation model is:

```text
README
  What is agent-env?
  What can I do with it?
  Quick start
  Where do I go next?

docs/index
  Navigation by reader goal and capability

ARCHITECTURE
  System map, boundaries, authorities, major data/lifecycle flows

roadmap
  Capability groups
    implemented
    active/in progress
    deferred
    open decisions

policy docs
  QUALITY / RELIABILITY / SECURITY / PORTABILITY / PLANS
  Cross-cutting repository rules and guarantees

product specs
  User-visible capability contracts

design docs / ADRs
  Why and how those contracts are implemented

active ExecPlans
  Current work and acceptance evidence

completed ExecPlans
  Historical implementation/review evidence; not rewritten as current prose
```

The goal is not to shorten documentation by deleting technical content.

The goal is to make every document answer a clear reader question and place
details at the right level.

## Reader-First Editing Model

English remains the canonical normative source.

However, "canonical" does not mean that English should remain compressed or that
Japanese should mirror English sentence structure.

The editorial workflow for every English/Japanese pair is:

```text
current source facts
    -> classify document purpose
    -> identify duplicate/misplaced content
    -> restructure English around reader questions
    -> verify facts/links/contracts
    -> translate complete meaning into Japanese
    -> restructure Japanese as natural technical Japanese
    -> semantic parity review
    -> docs-check
```

Japanese must preserve:

- normative meaning;
- implemented/deferred/unsupported status;
- safety limitations;
- identifiers and command names;
- links and evidence references.

Japanese does not need to preserve:

- English sentence count;
- clause order;
- paragraph boundaries;
- noun-phrase structure;
- heading wording when a natural Japanese heading communicates the same scope.

The English and Japanese documents are semantic siblings, not line-by-line
mirrors.

## Documentation Principles

Apply these principles consistently.

### One document, one primary reader question

A document may contain several sections, but its purpose must be clear from the
opening.

Examples:

- README: "What is this and how do I start?"
- Architecture: "How is the system divided and where do responsibilities live?"
- Roadmap: "What exists now, what is being worked on, and what is deferred?"
- Security: "What trust boundaries and safety guarantees exist?"
- Product spec: "What behavior does this capability promise?"
- Design doc: "How does the implementation satisfy that contract?"
- ExecPlan: "How will/was this substantial change delivered and verified?"

Do not make one document a catch-all because the information is related.

### Group by concept, not implementation chronology

Prefer capability groups such as:

```text
Sources and worktrees
Container runtimes
Android and mobile
Browser/UI automation
Persistent host processes
Testing and evidence
Distribution
Multi-host / orchestration
Security and trust
```

over paragraphs that follow the order features happened to be implemented.

### Separate status categories explicitly

Where status matters, distinguish:

- Implemented
- Active / in progress
- Deferred
- Unsupported / intentionally out of scope
- Open decision

Do not force readers to infer status from verbs such as "remains", "requires",
"tracked", or "would need".

### Prefer progressive disclosure

Keep the summary readable and link to the detailed authority.

For example, roadmap should say that Browser/CDP is implemented and summarize
its boundary, then link to the product contract/completed plan. It should not
repeat protocol-level acceptance details unless they materially affect roadmap
status.

### Preserve hard constraints

Reader-first editing must never soften or omit:

- ownership rules;
- cleanup proof requirements;
- trust boundaries;
- unsupported cases;
- platform evidence limitations;
- exact safety semantics;
- normative MUST/NEVER requirements.

Readability is not permission to weaken the contract.

### Reduce noun piles and list-like sentences

If a sentence enumerates several independent capabilities, convert it into:

- a short introductory sentence plus bullets;
- multiple sentences;
- a small table;
- separate subsections;

whichever best matches the document.

A sentence should not require the reader to maintain a large stack of modifiers
to find the subject and predicate.

### Prefer concrete subjects and verbs

Prefer:

```text
The worker verifies the source digest before it creates a worktree.
```

over:

```text
Pre-effect source-digest verification of transferred worker materialization is
required.
```

Use domain terminology where it improves precision, but do not turn every action
into a noun phrase.

### Treat evidence as evidence, not the main narrative

Exact CI run IDs, versions, hashes, and completed-plan references are important,
but do not interrupt an introductory explanation unless the reader needs them
there.

Keep durable evidence linked and discoverable.

## Scope

### Tier 1 — Repository entry points and navigation

Review and restructure:

- `README.md` / `README.ja.md`
- `AGENTS.md` / `AGENTS.ja.md`
- `ARCHITECTURE.md` / `ARCHITECTURE.ja.md`
- `docs/index.md` / `docs/index.ja.md`
- `docs/roadmap.md` / `docs/roadmap.ja.md`

These receive the highest priority because they shape how every other document
is discovered.

`AGENTS.md` remains constrained by its repository role and line cap. Reader-first
editing must keep it a map/operating guide, not turn it into a manual.

### Tier 2 — Cross-cutting policy and quality documents

Review and restructure:

- `docs/PLANS.md` / `.ja.md`
- `docs/QUALITY.md` / `.ja.md`
- `docs/RELIABILITY.md` / `.ja.md`
- `docs/SECURITY.md` / `.ja.md`
- `docs/PORTABILITY.md` / `.ja.md`
- bilingual documentation policy/design docs

Clarify the distinction between:

- policy;
- rationale;
- implementation detail;
- acceptance evidence.

### Tier 3 — Current product specifications

Review all current product specs under:

```text
docs/product-specs/
```

For each capability, make the contract easy to answer:

- What problem does this feature solve?
- What does the user declare or invoke?
- What does agent-env own?
- What does it guarantee?
- What does it reject?
- What is explicitly out of scope?
- Which failure/recovery states are visible?

Avoid turning product specs into code-structure descriptions.

### Tier 4 — Current design documents and ADRs

Review current design documents and ADRs under:

```text
docs/design-docs/
docs/adr/
```

Design documents should explain:

- context;
- constraints;
- chosen architecture;
- authority/ownership boundaries;
- data/lifecycle flow;
- alternatives/tradeoffs;
- failure and recovery behavior.

Move user-visible normative behavior back to product specs when it has drifted
into design-only prose.

ADRs remain concise decision records and should not be expanded into tutorials.

### Tier 5 — Active ExecPlans

Review active ExecPlans under:

```text
docs/exec-plans/active/
```

Do not rewrite their evidence/history gratuitously.

Improve only where current instructions are difficult to follow:

- Purpose / Big Picture;
- current state;
- milestone grouping;
- acceptance organization;
- cross-links;
- Japanese readability.

Preserve required headings, decision history, failed verification, and direct
acceptance evidence.

### Tier 6 — Current audits and durable guidance

Review current durable audit outputs under `docs/audits/` when they are meant to
remain understandable operational or quality guidance.

Historical raw evidence should not be polished merely for style.

## Explicit Non-Goals / Historical Exceptions

Do not wholesale rewrite:

- completed ExecPlans under `docs/exec-plans/completed/`;
- historical review plans;
- historical handoffs/reference archives;
- generated documentation under `docs/generated/`;
- generated database schema;
- LICENSE;
- third-party text;
- source comments and CLI messages unless another finding requires it.

Completed ExecPlans are historical records. Their wording is evidence of what
was planned, observed, reviewed, or fixed at the time.

They may be:

- linked more clearly from current docs;
- indexed better;
- corrected only for a material broken link/factual metadata issue;
- translated only where the existing bilingual policy explicitly requires or a
  separate migration is intentionally performed.

Do not silently modernize historical wording because current style has improved.

## Information Preservation / Traceability

A documentation rewrite can accidentally delete a limitation or change a
contract while making prose cleaner.

Before rewriting each substantial document, create a lightweight content map:

```text
old section / claim
    -> keep here
    -> move to another current document
    -> replace with link to authority
    -> remove because stale/duplicate
```

Any removal of nontrivial information requires a reason:

- stale;
- duplicate of a more authoritative document;
- implementation evidence moved behind a link;
- historical information moved to an explicit archive;
- no longer true because code/contract changed.

Do not remove a constraint merely because it seems too detailed.

For high-risk docs such as SECURITY, RELIABILITY, Architecture, product specs and
active ExecPlans, compare the final version against the pre-edit content map
before completion.

## Fact Verification

This is an editorial change, not a license to preserve stale facts.

When an existing statement is questionable:

1. identify its current authority;
2. verify against current product/design docs, code, tests, completed acceptance
   evidence, or current active plan;
3. correct both language versions;
4. record the discovered drift in this ExecPlan.

Do not invent a new capability because a rewritten sentence sounds cleaner.

Do not advertise active/deferred work as implemented.

## Roadmap-Specific Structure

The roadmap should become a grouped status document, not a long prose list.

Preferred shape:

```text
# Roadmap and open decisions

## How to read this roadmap

## Current capability map
  compact table or grouped summary

## Sources and worktrees
  Implemented
  Active
  Deferred / open

## Container runtimes
  ...

## Android and mobile
  ...

## Browser and UI automation
  ...

## Host processes
  ...

## Testing, evidence and harness
  ...

## Distribution and releases
  ...

## Multi-host and orchestration
  ...

## Trust, policy and writable workflows
  ...

## CI and platform coverage
  ...
```

The exact headings may change after content inventory.

The roadmap should not repeat low-level acceptance evidence already captured in
product specs or completed ExecPlans.

## README-Specific Structure

README should optimize for first contact.

Preferred reading sequence:

```text
What agent-env is
Core use cases
Key capabilities
Quick start / minimal example
Lifecycle / mental model
Safety/ownership model
Supported runtime overview
Where to read more
Project status / limitations
```

Do not make the README a copy of Architecture or the roadmap.

## Architecture-Specific Structure

Architecture should be scannable as a system map.

Prefer:

```text
System at a glance
Core domain concepts
Authority and ownership
Package/component map
Create lifecycle
Observe/reconcile lifecycle
Destroy/cleanup lifecycle
Persistence and evidence
Runtime/provider boundaries
Platform boundaries
Control-plane boundary (when implemented)
Where detailed contracts live
```

Avoid mixing roadmap status into architecture except where needed to mark a
boundary as intentionally absent.

## Policy-Document Structure

QUALITY, RELIABILITY, SECURITY, PORTABILITY and PLANS should use consistent
patterns where practical:

```text
Purpose
Invariants / requirements
How they are enforced
Evidence / checks
Known limitations
Related documents
```

Do not force identical headings when the subject does not fit, but make the
difference between "rule" and "background explanation" obvious.

## Product-Spec Structure

Prefer a common reader-oriented skeleton where applicable:

```text
Purpose
User-visible model
Configuration / commands
Ownership and identity
Lifecycle
Observation/results
Failure/recovery
Security/safety constraints
Platform behavior
Out of scope
Related design/evidence
```

Do not mechanically reformat a spec when a different structure reads better.

## Japanese Editorial Policy

Update `docs/design-docs/bilingual-documentation.md` and `.ja.md` to state
explicitly:

- English is canonical for normative meaning;
- Japanese is a maintained semantic translation, not a structural mirror;
- translators/editors may reorder sentences and paragraphs;
- natural Japanese subject omission is acceptable where unambiguous;
- avoid unnecessary katakana/English mixing when a stable Japanese technical
  term is clearer;
- preserve identifiers, command names, protocol names and code vocabulary where
  translation would reduce precision;
- do not reproduce dense English noun phrases as Japanese noun chains;
- use bullets/tables when they reduce parsing cost;
- translate status and limitations clearly rather than relying on English-style
  modal phrasing.

The freshness hash continues to prove that the English source was reviewed, not
that the Japanese structure is identical.

## Progress

- [x] 2026-09-09: Record exact starting revision and create
      `docs/reader-first-documentation-restructure`.
- [x] 2026-09-09: Run baseline `go run ./tools/repoctl docs-check`.
- [x] 2026-09-09: Inventory all durable human-facing English/Japanese document pairs.
- [x] 2026-09-09: Classify each pair by Tier 1-6 or explicit historical/generated exception.
- [x] 2026-09-09: Record document purpose and primary reader question for every in-scope pair.
- [x] 2026-09-09: Identify duplicate/stale/misplaced content before rewriting.
- [x] 2026-09-09: Create content-preservation maps for Tier 1/2 and high-risk Tier 3/4 docs.
- [x] 2026-09-09: Update bilingual documentation policy with semantic-not-structural
      translation guidance.
- [x] 2026-09-09: Restructure README pair.
- [x] 2026-09-09: Restructure docs index pair.
- [x] 2026-09-09: Restructure roadmap pair.
- [x] 2026-09-09: Restructure Architecture pair.
- [x] 2026-09-09: Review/restructure AGENTS pair without exceeding its role/line constraints.
- [x] 2026-09-09: Restructure PLANS pair.
- [x] 2026-09-09: Restructure QUALITY pair.
- [x] 2026-09-09: Restructure RELIABILITY pair.
- [x] 2026-09-09: Restructure SECURITY pair.
- [x] 2026-09-09: Restructure PORTABILITY pair.
- [x] 2026-09-09: Review all current product-spec pairs.
- [x] 2026-09-09: Review all current design-doc pairs.
- [x] 2026-09-09: Review all current ADR pairs.
- [x] 2026-09-09: Review active ExecPlan pairs for current-reader clarity.
- [x] 2026-09-09: Review current durable audit docs where applicable.
- [x] 2026-09-09: Remove or consolidate duplicated current guidance only after preserving an
      authoritative source/link.
- [x] 2026-09-09: Verify all implemented/deferred/unsupported/open statuses against current
      repository authority.
- [x] 2026-09-09: Verify English internal links.
- [x] 2026-09-09: Update Japanese translations semantically, not line-by-line.
- [x] 2026-09-09: Refresh all affected Japanese `source_sha256` values only after translation
      review.
- [x] 2026-09-09: Run `repoctl docs-check`.
- [x] 2026-09-09: Run architecture/metadata/generated-output checks included by repository
      harness.
- [x] 2026-09-09: Perform independent English reader-first review.
- [x] 2026-09-09: Perform independent Japanese reader-first review.
- [x] 2026-09-09: Perform bilingual semantic parity review.
- [x] 2026-09-09: Run final repository harness.
- [ ] Complete Outcomes & Retrospective with changed-document inventory.
- [ ] Move both ExecPlans to completed and update links/hashes.

A checked item means observed completion, not intention.

## Surprises & Discoveries

- Browser manifest fields and process consumption were already implemented, while
  both languages still described them as deferred. `internal/config/browser.go`
  and existing config tests establish the current contract. Android UI commands
  were likewise implemented; MVP exclusions now explicitly describe initial scope.
- Lease creation builds its plan before reservation (`internal/app/lifecycle.go`:
  `BuildPlan` before `Store.Reserve`). Conceptual design snippets are not real API
  declarations. Local TTL currently uses built-in policy; an editable host policy
  file remains deferred.
- CAS uses a lowercase digest key and `<cas-root>/<digest>/data`, not the old
  `sha256/<digest>` sketch (`internal/blobstore/store.go`: `ValidDigest`, `Open`).
- Reconciliation covers all recorded runtime providers, not Compose alone;
  existing Android/process reconciliation regressions corroborate this boundary.
- The final recorded local Browser CI run is `34320519250` at `440082b`;
  `34316121411` belongs to `53fe81a`. The completed multi-host Plan preserves the
  original evidence. Roadmap now links there rather than duplicating run details.
- Review caught an overbroad promise that the original repository is unchanged:
  source files stay unchanged, but Git worktree registration changes metadata.
  Podman endpoint probing is TCP only; UDP remains engine-observed.
- Japanese review found omitted active-operation conditions, dense noun chains,
  an imprecise native-Emulator/physical-device distinction, and a translated
  fenced-code language tag. These were repaired before final hash review.
- Intermediate checks exposed temporary stale hashes during editing, a missing
  index route, nonexistent Japanese siblings for explicitly exempt historical
  Plans, missing README translation metadata and whitespace. Those failed states
  were repaired without changing checks or exception rules.

## Decision Log

- Decision: Expand the work from roadmap editing to all durable human-facing
  current documentation.
  Rationale: the readability issue is systemic and appears in entry points,
  policy docs and capability documentation, not only the roadmap.
  Date/Author: 2026-09-09 / maintainers.

- Decision: English remains canonical for normative meaning, but English itself
  receives reader-first editing before Japanese translation.
  Rationale: translating dense canonical prose faithfully reproduces the original
  readability problem.
  Date/Author: 2026-09-09 / maintainers.

- Decision: Japanese preserves semantics, not English sentence/paragraph
  structure.
  Rationale: line-by-line structural fidelity produces unnatural Japanese and is
  not required by the existing freshness-hash model.
  Date/Author: 2026-09-09 / maintainers.

- Decision: Group current documentation by reader question and capability rather
  than implementation chronology.
  Rationale: readers primarily need orientation and status, not the historical
  order in which features were added.
  Date/Author: 2026-09-09 / maintainers.

- Decision: Distinguish Implemented, Active, Deferred, Unsupported and Open
  decisions explicitly where status matters.
  Rationale: roadmap/current-state prose should not force readers to infer status.
  Date/Author: 2026-09-09 / maintainers.

- Decision: Use progressive disclosure and link to detailed authorities instead
  of repeating acceptance evidence in entry-point documents.
  Rationale: preserves evidence while reducing cognitive load and documentation
  drift.
  Date/Author: 2026-09-09 / maintainers.

- Decision: Completed ExecPlans and historical review plans are not wholesale
  rewritten.
  Rationale: they are historical evidence and their original wording is part of
  that record.
  Date/Author: 2026-09-09 / maintainers.

- Decision: Information-preservation maps are required before high-risk
  restructuring.
  Rationale: readability edits must not silently delete safety constraints,
  unsupported cases, or normative behavior.
  Date/Author: 2026-09-09 / maintainers.

- Decision: Reader-first editing may change document structure but must not weaken
  product/security/reliability contracts.
  Rationale: clarity and normative precision are both required.
  Date/Author: 2026-09-09 / maintainers.


### Milestone 1 dispositions (2026-09-09 / implementation)

The twelve questions below are settled as follows, in the same order:

1. Keep inventory, preservation maps and terminology decisions in this Plan;
   additional audit files would duplicate the execution record.
2. Preserve raw audit reports as historical evidence; review only their current
   navigation index. All 13 report pairs remain unchanged.
3. Multi-host was already merged into the starting revision; there is no parallel
   active implementation to edit or rebase. Use the dedicated documentation branch.
4. Do not add a glossary: existing contracts define precise identifiers and the
   bilingual policy defines prose choices without another source to maintain.
5. Recommend purpose/configuration/behavior/failure/limits for product contracts;
   do not enforce identical headings where the subject differs.
6. Recommend context/boundaries/mechanism/recovery/evidence for designs and retain
   concise decision/context/consequences for ADRs.
7. Keep the editorial rules in bilingual-documentation; QUALITY links to them
   and describes validation instead of owning a second policy.
8. Use independent human-style language and parity review. No speculative prose
   heuristics or new linter are needed; existing deterministic checks remain intact.
9. Allow short orientation summaries; link to one detailed current contract and
   to completed Plans for historical evidence. Move recipes before deleting copies.
10. Prefer natural established Japanese technical terms and omit unambiguous
    subjects; preserve commands, protocol terms and identifiers exactly.
11. Refresh `last_verified` only for pairs actually fact-checked and editorially
    reviewed. This is not evidence of a new native runtime execution.
12. Route readers through current specs/designs. No new summary document is needed
    merely because a completed Plan was the old entry point; preserve its evidence.

## Outcomes & Retrospective

Not completed.

At completion summarize:

- starting/final revision;
- document pairs reviewed;
- document pairs materially restructured;
- documents intentionally left unchanged;
- duplicated guidance consolidated;
- stale facts corrected;
- current authority moves;
- reader-first conventions added to policy;
- terminology decisions;
- English review findings;
- Japanese review findings;
- bilingual parity findings;
- docs-check/harness evidence;
- remaining documentation debt.

Also record which document structures became recommended templates for future
work.

## Context and Orientation

Read before editing:

- `README.md` / `.ja.md`
- `AGENTS.md` / `.ja.md`
- `ARCHITECTURE.md` / `.ja.md`
- `docs/index.md` / `.ja.md`
- `docs/roadmap.md` / `.ja.md`
- `docs/PLANS.md` / `.ja.md`
- `docs/QUALITY.md` / `.ja.md`
- `docs/RELIABILITY.md` / `.ja.md`
- `docs/SECURITY.md` / `.ja.md`
- `docs/PORTABILITY.md` / `.ja.md`
- `docs/design-docs/bilingual-documentation.md` / `.ja.md`
- current product/design indexes;
- all in-scope product specs;
- all in-scope design docs/ADRs;
- active ExecPlans;
- translation exception registry;
- repoctl docs-check implementation where behavior needs clarification.

The current bilingual policy already states that Japanese is a maintained
artifact and that the freshness hash detects source changes but does not validate
semantic translation quality. Preserve that mechanism while improving the
editorial policy.

## Plan of Work

### Milestone 1 — Inventory and document-role map

Enumerate every durable English/Japanese pair.

Create a working matrix:

```text
path
tier
primary reader
primary question
canonical authority for
duplicates / overlaps with
rewrite needed? yes/no
risk: low/medium/high
```

Mark generated/historical exceptions explicitly.

Do not start broad rewriting until the map shows where content should live.

### Milestone 2 — Editorial and bilingual policy

Update the bilingual documentation design to codify semantic-not-structural
translation.

If needed, add a small reader-first section to QUALITY or another appropriate
current policy document. Avoid creating redundant policy documents.

Define review heuristics such as:

- long noun-chain detection by reviewer, not necessarily a linter;
- status-category clarity;
- one paragraph/section purpose;
- explicit subject when ambiguity exists;
- progressive disclosure;
- no evidence dumping in orientation docs;
- no line-by-line Japanese mirroring requirement.

Do not add a noisy mechanical linter unless demonstrated examples show a useful
low-false-positive rule.

### Milestone 3 — Entry points

Rewrite Tier 1 in this order:

1. README
2. docs index
3. roadmap
4. Architecture
5. AGENTS

After each English edit:

- verify facts and links;
- update Japanese as natural prose;
- refresh hash;
- run focused docs-check if available.

The roadmap should be grouped by capability/status.
The README should orient a first-time user.
Architecture should be a map, not a changelog.
AGENTS should remain concise operating guidance.

### Milestone 4 — Cross-cutting policy docs

Rewrite PLANS, QUALITY, RELIABILITY, SECURITY and PORTABILITY.

Preserve normative terms.

Separate:

- requirement;
- rationale;
- enforcement;
- evidence;
- known limitation.

Cross-link instead of duplicating the same rule.

### Milestone 5 — Product specs

Review every current product spec.

Create consistent navigation and contract clarity without forcing identical
wording.

For each spec, verify that user-visible guarantees are not hidden only in design
docs or completed plans.

Move/link content carefully when a better authority exists.

### Milestone 6 — Design docs and ADRs

Review design docs/ADRs for:

- architecture clarity;
- decision context;
- ownership/authority;
- lifecycle diagrams/text;
- failure/recovery;
- tradeoffs.

Remove duplicated user-contract prose when the product spec is authoritative,
replacing it with a concise summary/link.

Do not rewrite ADR history into present-tense documentation.

### Milestone 7 — Active plans and audits

Review only current operational readability.

Preserve:

- required sections;
- historical decision entries;
- failed attempts;
- acceptance evidence;
- exact commands/results.

Improve navigation and prose where an implementer currently has to decode dense
paragraphs.

### Milestone 8 — Cross-document deduplication and authority review

Search repeated claims across README/Architecture/roadmap/policy/product/design.

For every important repeated fact, identify one current authority.

Other documents should:

- summarize it if needed;
- link to authority;
- avoid independently evolving detailed copies.

Do not over-deduplicate introductory text; a small amount of intentional
repetition is useful for orientation.

### Milestone 9 — Independent language reviews

Run three separate reviews:

1. English reader-first review
2. Japanese reader-first review
3. bilingual semantic parity review

The Japanese reviewer should evaluate the Japanese document as Japanese, not by
comparing sentence alignment.

The parity review separately checks that meaning/status/limitations did not
drift.

### Milestone 10 — Final verification

Run:

```text
go run ./tools/repoctl docs-check
go run ./tools/repoctl check
```

plus any repository-specific generated/index checks included by the harness.

Verify:

- no stale translation hash;
- no missing language pair;
- no broken internal link;
- no orphan Japanese doc;
- no accidental historical-plan rewrite;
- no unsupported/deferred feature advertised as implemented;
- no normative safety constraint lost.

Complete the retrospective before archival.

## Concrete Steps

1. Fast-forward master; record starting revision.
2. Create `docs/reader-first-documentation-restructure`.
3. Add English/Japanese active ExecPlans.
4. Run baseline docs-check/repoctl check.
5. Build document inventory/role matrix.
6. Mark historical/generated exclusions.
7. Update bilingual editorial policy.
8. Rewrite README pair.
9. Rewrite docs index pair.
10. Rewrite roadmap pair.
11. Rewrite Architecture pair.
12. Rewrite AGENTS pair.
13. Rewrite PLANS pair.
14. Rewrite QUALITY pair.
15. Rewrite RELIABILITY pair.
16. Rewrite SECURITY pair.
17. Rewrite PORTABILITY pair.
18. Review/restructure all current product-spec pairs.
19. Review/restructure all current design-doc/ADR pairs.
20. Review active ExecPlan pairs.
21. Review current durable audit docs.
22. Perform cross-document duplication/authority pass.
23. Verify all facts/status/links.
24. Refresh Japanese hashes after semantic review.
25. Run independent English review.
26. Run independent Japanese review.
27. Run semantic parity review.
28. Run final docs-check/repoctl check.
29. Record evidence and retrospective.
30. Move plans to completed and update links/hashes.

## Validation and Acceptance

| ID | Required behavior | Evidence |
| --- | --- | --- |
| D1 | Every durable human-facing Markdown pair is inventoried and classified as in-scope or explicit exception. | Per-path inventory, Japanese family classification and 28 historical/generated exclusions below; 46 current pairs plus this Plan; six existing language exceptions unchanged. |
| D2 | Every in-scope document has a clear primary purpose/reader question. | Inventory reader-question/role columns; current document openings identify purpose and next reads. |
| D3 | Tier 1 entry documents are materially reviewed for information architecture, not only grammar. | Six root/navigation pairs and AGENTS reviewed structurally; README, Architecture, roadmap and indexes reorganized by reader question. |
| D4 | Roadmap groups capabilities coherently and distinguishes Implemented/Active/Deferred/Unsupported/Open status where relevant. | Roadmap separates implemented capabilities, deferred work, open decisions and evidence limits. |
| D5 | README gives a first-time reader a direct path from product purpose to first use and deeper docs. | README installation, prerequisites and minimal lease workflow precede capability-specific contract links. |
| D6 | Architecture is a system/boundary map rather than a feature-history list. | Architecture layer table, local lifecycle and runtime/application/control-plane boundaries. |
| D7 | AGENTS remains concise operating guidance and does not absorb detailed manuals. | AGENTS remains 105 lines; current-contract route and editorial-policy link added; operational rules retained. |
| D8 | PLANS/QUALITY/RELIABILITY/SECURITY/PORTABILITY clearly distinguish requirements, rationale, enforcement and limitations. | Seven policy pairs separate requirements, rationale, enforcement and evidence; independent policy review passed. |
| D9 | Every current product spec is reviewed for user-visible contract clarity. | Twelve product pairs reviewed; manifest Browser fields, Android UI status and MVP historical scope reconciled with code. |
| D10 | Every current design doc/ADR is reviewed for architecture/decision clarity without duplicating product authority unnecessarily. | Thirteen design and eight ADR pairs reviewed; API sketches, TTL policy, CAS layout and local GC scope corrected; accepted decisions retained. |
| D11 | Active ExecPlans retain all required sections, decisions, failures and acceptance evidence. | Original Plan requirements and maintainer decisions retained; dated findings, dispositions and evidence added. |
| D12 | Completed ExecPlans/historical review plans are not wholesale rewritten for style. | All 26 pre-existing completed English Plans and their existing translations unchanged against starting revision. |
| D13 | Generated/reference/archive exceptions remain intact unless a separate justified migration is recorded. | Generated schema, original handoff and exception registry unchanged; handoff SHA-256 recorded below. |
| D14 | High-risk rewrites have a content-preservation map and no unaccounted loss of normative information. | Before-edit preservation maps above; recipes moved to product contracts before README removal; review checked normative constraints. |
| D15 | Implemented/deferred/unsupported/open status claims are verified against current authority. | Current Browser/config/lifecycle/blobstore sources and existing tests checked; roadmap status categories explicit. |
| D16 | Orientation docs use progressive disclosure rather than repeating detailed acceptance evidence. | README/indexes/roadmap link to current capability contracts and historical acceptance evidence. |
| D17 | Repeated current guidance has one identifiable authority with appropriate summaries/links elsewhere. | Remote setup and Android helper recipes live in product specs; language policy owns editorial rules; QUALITY links there. |
| D18 | English prose is reviewed for noun piles, mixed-status paragraphs and list-like sentences. | Independent English review passed after grouping dense capability lists and separating statuses. |
| D19 | Japanese prose is reviewed as natural technical Japanese rather than English structural mirroring. | Independent Japanese review passed after repairing noun chains and dense Android/process prose. |
| D20 | Japanese may reorder sentences/paragraphs while preserving complete normative meaning. | Japanese paragraphs and tables reorganized independently; separate parity review found no remaining drift. |
| D21 | Bilingual policy explicitly documents semantic-not-structural translation. | Bilingual-documentation policy explicitly requires semantic fidelity and independent language/parity review, permits different structure. |
| D22 | Identifiers/commands/protocol names remain precise across both languages. | Product fenced bodies compared with baseline; commands, identifiers and policy requirements retained; accidental fence-tag translation repaired. |
| D23 | Safety/ownership/cleanup/trust limitations are not weakened by reader-first editing. | Ownership, cleanup, operation fences and TCP-only probing reviewed; overbroad source-checkout guarantee narrowed accurately. |
| D24 | No deferred/unsupported capability is accidentally advertised as available. | Roadmap and contracts retain unsupported/deferred scope and native/physical-host evidence limits. |
| D25 | Stale facts discovered during editing are corrected in both languages with authority recorded in the plan. | Surprises section records exact source paths, API names and original CI evidence for bilingual stale-fact repairs. |
| D26 | A separate English reader-first review is completed. | Final independent review evidence below maps reviewers to six root, twelve product, seven policy, thirteen design and eight ADR pairs. |
| D27 | A separate Japanese reader-first review is completed. | Same independent roles separately read Japanese as standalone documentation, then reported readability findings. |
| D28 | A bilingual semantic parity review is completed after both language-specific reviews. | Separate parity review passed; missing active-operation qualification, native-device wording and final policy additions rechecked. |
| D29 | All affected Japanese `source_sha256` values match reviewed English sources. | Final repoctl check docs stage passed after semantic review and affected-pair hash refresh. |
| D30 | `repoctl docs-check` passes. | Baseline and repeated repoctl docs-check passed; final full check also passed its docs-check stage. |
| D31 | Full repository harness/documentation checks pass. | 2026-09-09 final go run ./tools/repoctl check exit 0: format, unit tests, vet, docs, generated output and architecture checks. |
| D32 | Outcomes & Retrospective records changed/unchanged documents, authority moves, stale facts and remaining debt before archival. | Pending |

A green translation hash alone is not semantic acceptance.

## Idempotence and Recovery

This work is documentation-only unless fact verification exposes a separate
product defect.

Do not change production code merely to make documentation easier to describe.

Edit in coherent document groups.

After each group:

1. finish English;
2. verify facts/links;
3. finish Japanese;
4. refresh source hash;
5. run docs-check;
6. commit.

If later editing changes the English source again, re-review the Japanese meaning
before refreshing the hash.

Do not refresh hashes mechanically across the repository without semantic review.

If a rewrite proves worse or loses an important distinction, revert that
document pair independently rather than forcing the entire documentation series
forward.

## Artifacts and Notes

Recommended working artifacts:

```text
docs/audits/reader-first-documentation/
    inventory.md
    inventory.ja.md
    authority-map.md
    authority-map.ja.md
    terminology.md
    terminology.ja.md
```

Whether these remain durable should be decided during Milestone 1. If they are
temporary implementation aids, keep the durable conclusions in the completed
ExecPlan instead.

Useful inventory fields:

```text
path
purpose
audience
status
authority
duplicates
reader-first findings
translation findings
action
verification
```

Track material information moves explicitly.

## Interfaces and Dependencies

No production dependency is required.

Use:

- repository Markdown files;
- current code/tests only for fact verification;
- `repoctl docs-check`;
- repository harness;
- existing bilingual metadata/hash mechanism.

The external `reader-first-editor` skill may be used as an editing aid if
available, but this plan does not depend on it. Its output must still satisfy
repository facts, policy, and bilingual review.

Do not add an external translation service requirement.

## Milestone 1 questions (resolved in Decision Log)

1. Exact durable scope of `docs/audits/reader-first-documentation/`.
2. Whether current audit reports belong in the full prose review or only
   navigation/summary review.
3. Whether active multi-host work should be edited concurrently or rebased into
   the new documentation structure after implementation stabilizes.
4. Whether a short terminology/glossary document is useful or would create
   another authority to maintain.
5. Whether product specs should adopt a recommended common skeleton formally.
6. Whether design docs should adopt a recommended common skeleton formally.
7. Whether reader-first guidance belongs only in bilingual-documentation design
   or also in QUALITY.md.
8. Whether docs-check should ever add low-noise structural heuristics, or human
   review remains the correct mechanism.
9. How much intentional summary duplication is acceptable in README/roadmap/
   Architecture before it becomes drift risk.
10. Whether Japanese technical vocabulary should prefer translated terms or
    established English identifiers on a term-by-term basis.
11. Whether `last_verified` should be refreshed only for fact verification or
    for substantial editorial review as well.
12. Whether completed plans currently linked as the only authority need new
    current summary documents rather than edits to historical evidence.


## Execution inventory and preservation maps (2026-09-09)

Start revision: `084da57de177c0a09bc3cb61ae99faff8bd79a94`. The tree contained only
the user's untracked English/Japanese Plan pair; no implementation differences.
`master` fast-forward check found no update; work switched to
`docs/reader-first-documentation-restructure`. Baseline `docs-check` and full
`check` passed. No nested AGENTS files were found. The reader-first-editor skill
is not available in this session; this Plan supplies the editorial procedure.

Each inventory row denotes the English document and its corresponding Japanese
pair. Current document families are reviewed in full, not inferred from filename
coverage. Completed ExecPlans and generated references are historical/generated
editorial exclusions and are not rewritten; existing translation exceptions are unchanged. Audit raw evidence remains historical; only its
index is a current navigation surface. Verification timestamps are not freshness
proof by themselves.

Before editing, the agreed preservation maps are:

- README: dense capability introduction and appended runtime recipes -> identity,
  trust warning, installation, prerequisite map, common lease workflow, capability
  routes, state/cleanup limits, contribution route. Specialized commands and exact
  acceptance evidence remain in linked capability contracts/quality/completed Plans;
  move missing setup recipes there before removing their README copies.
- ARCHITECTURE: introductory local orchestration, invariants containing Android/
  Flutter/UI details, then appended process/browser/remote sections -> purpose,
  common local flow, dependency rules, cross-cutting invariants, clearly separated
  runtime/application/observation/remote/release boundaries. Preserve package names,
  ownership records and safety requirements; current contracts replace the obsolete
  implication that MVP alone defines all behavior.
- docs/index: filename sections -> reader questions with clear current-policy,
  current-contract, mechanism, decision and historical-evidence routes. Every
  existing entry remains reachable. Roadmap: mixed implemented/deferred prose ->
  implemented scope first, deferred decisions grouped by boundary, explicit evidence
  limitations alongside capability entries; retain all excluded features.
- Policy: separate rule, rationale, enforcement and evidence without weakening any
  requirement. PLANS separates execution and completion gates. Bilingual policy
  explicitly permits different paragraph/headings structure but requires semantic
  parity, English-first writing and separate language/parity reviews.
- Product contracts: purpose/status -> configuration/selection -> behavior ->
  failure/privacy -> limitations/evidence -> next read. Preserve old anchors and
  every numerical/resource/safety contract. Correct verified stale statements:
  browser manifest fields are implemented, browser consumes process today, Android
  UI commands are implemented; MVP exclusions describe original scope only.
- Design/ADR: purpose and boundaries -> identity/lifecycle -> invariants/recovery ->
  validation/next read. Conceptual API examples are distinguished from concrete
  interfaces; accepted ADR decisions/status remain historical decisions. Preserve
  mechanism and proof limits, including unverified native infrastructure.

| Pair | Tier / reader question | Role and status | Priority / treatment |
| --- | --- | --- | --- |
| `README.md` — agent-env | 1 / where do I start? | entry or system map; current | high; restructure and preserve routes/invariants |
| `AGENTS.md` — Agent entry point | 1 / where do I start? | entry or system map; current | high; restructure and preserve routes/invariants |
| `ARCHITECTURE.md` — Architecture | 1 / where do I start? | entry or system map; current | high; restructure and preserve routes/invariants |
| `docs/PLANS.md` — ExecPlan policy | 2 / which rule applies? | policy; current | high; rule / rationale / enforcement / evidence |
| `docs/PORTABILITY.md` — Portability | 2 / which rule applies? | policy; current | high; rule / rationale / enforcement / evidence |
| `docs/QUALITY.md` — Quality and verification | 2 / which rule applies? | policy; current | high; rule / rationale / enforcement / evidence |
| `docs/RELIABILITY.md` — Reliability and recovery | 2 / which rule applies? | policy; current | high; rule / rationale / enforcement / evidence |
| `docs/SECURITY.md` — Security and trust | 2 / which rule applies? | policy; current | high; rule / rationale / enforcement / evidence |
| `docs/adr/0001-use-go.md` — Use Go | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/adr/0002-use-sqlite.md` — Use SQLite | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/adr/0003-compose-first-runtime.md` — Compose first runtime | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/adr/0004-repository-native-harness.md` — Repository-native harness | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/adr/0005-separate-flutter-applications.md` — Separate Flutter applications from Android resources | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/adr/0006-single-authority-multi-host.md` — One controller authority and whole-lease worker assignments | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/adr/0007-native-execution-boundaries.md` — Bound native execution paths and Windows/WSL interop | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/adr/index.md` — Architecture decisions | 4 / which detailed document should I read? | navigation; current | medium; reader routes, preserve destinations |
| `docs/audits/repository-correctness/current-compose-release.md` — Current Compose, assets and release correctness review | 6 / what audit evidence exists? | historical audit evidence | excluded from prose rewriting; preserve evidence |
| `docs/audits/repository-correctness/current-control-plane.md` — Control-plane current audit | 6 / what audit evidence exists? | historical audit evidence | excluded from prose rewriting; preserve evidence |
| `docs/audits/repository-correctness/current-mobile.md` — Current mobile correctness audit | 6 / what audit evidence exists? | historical audit evidence | excluded from prose rewriting; preserve evidence |
| `docs/audits/repository-correctness/current-process-browser.md` — Current process, browser and executor correctness review | 6 / what audit evidence exists? | historical audit evidence | excluded from prose rewriting; preserve evidence |
| `docs/audits/repository-correctness/documentation.md` — Documentation historical replay and current audit | 6 / what audit evidence exists? | historical audit evidence | excluded from prose rewriting; preserve evidence |
| `docs/audits/repository-correctness/findings.md` — Finding disposition and remediation ledger | 6 / what audit evidence exists? | historical audit evidence | excluded from prose rewriting; preserve evidence |
| `docs/audits/repository-correctness/historical-corpus.md` — Historical corpus inventory and escape summary | 6 / what audit evidence exists? | historical audit evidence | excluded from prose rewriting; preserve evidence |
| `docs/audits/repository-correctness/history-compose-release.md` — Historical Compose and standalone release review corpus | 6 / what audit evidence exists? | historical audit evidence | excluded from prose rewriting; preserve evidence |
| `docs/audits/repository-correctness/history-mobile.md` — Historical mobile correctness review corpus | 6 / what audit evidence exists? | historical audit evidence | excluded from prose rewriting; preserve evidence |
| `docs/audits/repository-correctness/history-process-browser.md` — Historical review corpus: process, browser and MVP | 6 / what audit evidence exists? | historical audit evidence | excluded from prose rewriting; preserve evidence |
| `docs/audits/repository-correctness/index.md` — Repository correctness audit | 6 / what audit evidence exists? | navigation; current | medium; navigation only |
| `docs/audits/repository-correctness/matrix.md` — Subsystem and invariant audit matrix | 6 / what audit evidence exists? | historical audit evidence | excluded from prose rewriting; preserve evidence |
| `docs/audits/repository-correctness/supplemental-browser.md` — Supplemental Browser review: late PR #10 comments | 6 / what audit evidence exists? | historical audit evidence | excluded from prose rewriting; preserve evidence |
| `docs/audits/repository-correctness/supplemental-cli.md` — Supplemental Browser CLI result audit | 6 / what audit evidence exists? | historical audit evidence | excluded from prose rewriting; preserve evidence |
| `docs/design-docs/android-emulator.md` — Android Emulator resource design | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/design-docs/android-ui-observer.md` — Android UI observer design | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/design-docs/bilingual-documentation.md` — Bilingual documentation | 2 / how do we maintain both languages? | editorial and translation policy; current | high; requirements, reviews and checks |
| `docs/design-docs/browser-cdp-automation.md` — Browser/CDP design | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/design-docs/compose-providers.md` — Compose provider design | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/design-docs/compose-runtime.md` — Compose runtime | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/design-docs/core-beliefs.md` — Core beliefs | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/design-docs/flutter-android-runtime.md` — Flutter Android lifecycle design | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/design-docs/index.md` — Design documents | 4 / which detailed document should I read? | navigation; current | medium; reader routes, preserve destinations |
| `docs/design-docs/lease-control-plane.md` — Lease control plane | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/design-docs/multi-host-control-plane.md` — Single-authority multi-host coordination | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/design-docs/persistent-process-runtime.md` — Persistent process lifecycle design | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/design-docs/reconciliation-and-gc.md` — Reconciliation and garbage collection | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/design-docs/standalone-distribution.md` — Standalone distribution design | 4 / why and how? | design or accepted decision | medium-high; boundary / mechanism / recovery / evidence |
| `docs/index.md` — Repository knowledge | 1 / where do I start? | entry or system map; current | high; restructure and preserve routes/invariants |
| `docs/product-specs/agent-env-mvp.md` — MVP specification | 3 / what behavior can I rely on? | contract; current (MVP original scope retained) | high; configuration / behavior / failure / limits |
| `docs/product-specs/android-emulator.md` — Android Emulator leases | 3 / what behavior can I rely on? | contract; current | high; configuration / behavior / failure / limits |
| `docs/product-specs/android-ui-observer.md` — Android UI observer | 3 / what behavior can I rely on? | contract; current | high; configuration / behavior / failure / limits |
| `docs/product-specs/browser-cdp-automation.md` — Browser/CDP automation | 3 / what behavior can I rely on? | contract; current | high; configuration / behavior / failure / limits |
| `docs/product-specs/cli-contract.md` — CLI contract | 3 / what behavior can I rely on? | contract; current | high; configuration / behavior / failure / limits |
| `docs/product-specs/compose-providers.md` — Compose providers | 3 / what behavior can I rely on? | contract; current | high; configuration / behavior / failure / limits |
| `docs/product-specs/flutter-android-runtime.md` — Flutter Android applications | 3 / what behavior can I rely on? | contract; current | high; configuration / behavior / failure / limits |
| `docs/product-specs/index.md` — Product specifications | 3 / which detailed document should I read? | navigation; current | medium; reader routes, preserve destinations |
| `docs/product-specs/manifest-v1.md` — Manifest v1 | 3 / what behavior can I rely on? | contract; current | high; configuration / behavior / failure / limits |
| `docs/product-specs/multi-host-control-plane.md` — Multi-host control plane | 3 / what behavior can I rely on? | contract; current | high; configuration / behavior / failure / limits |
| `docs/product-specs/persistent-process-runtime.md` — Persistent process runtimes | 3 / what behavior can I rely on? | contract; current | high; configuration / behavior / failure / limits |
| `docs/product-specs/standalone-distribution.md` — Standalone distribution | 3 / what behavior can I rely on? | contract; current | high; configuration / behavior / failure / limits |
| `docs/references/index.md` — References | 6 / where is historical reference material? | navigation; current | medium; current/deferred/evidence distinction |
| `docs/roadmap.md` — Roadmap and unresolved decisions | 1 / what exists and what is deferred? | capability status and open decisions; current | high; capability groups and evidence limits |
| `docs/exec-plans/active/reader-first-documentation-restructure.md` | 5 / how is this change delivered and verified? | execution authority; active until archival | high; preserve requirements, decisions, failures and evidence |

The per-family maps above are the before-edit contract. Detailed contributor maps
and independent review dispositions are recorded in the execution notes below as
work completes. The next read from each current document must be an actual
relative link, not a promise that content exists elsewhere.


### Execution notes and dispositions

2026-09-09: Tier 1 root maps are implemented. Specialized remote startup and UI
companion recipes were confirmed at the product-contract destinations before
README copies were removed. Audit navigation was rewritten while all historical
evidence tables and report bodies stayed intact. The bootstrap archive remains
byte-identical and exempt; its nonexistent Japanese archive is not a missing
translation. The original MVP/Android completed Plans likewise keep their existing
explicit exceptions, so Japanese navigation links to those English records.

Confirmed fact corrections: current browser manifest support and process consumer,
implemented Android UI commands, obsolete MVP-only scope wording. QUALITY's final
local Browser run is `34320519250` at `440082b`; the old `34316121411` belonged to
`53fe81a`, as documented in the completed multi-host Plan. No source behavior or
historical evidence bytes were changed to fit current claims.

Independent reviewers are separated from authors: policy author reviews root and
product text; product author reviews policy; design author reviews root navigation
and semantic preservation after finishing design. Root reviews design/ADR work.
Each report separates English usability, standalone Japanese readability, and
bilingual semantic parity; self-review is not counted as independent review.

2026-09-09 intermediate validation: full `go run ./tools/repoctl check` passed
again after all current families were structurally edited (unit/vet/docs/generated/
architecture). The independent root review repaired an overbroad source-repository
promise (Git worktree registration changes metadata), Japanese missing active-operation
conditions, an incorrect current-spec language link, native Emulator/physical-device
ambiguity, and operation-fence/evidence wording. Architectural reconciliation now
names all recorded runtime providers rather than only Compose; app Android/process
reconcile regressions corroborate this existing behavior. Product review corrected
a baseline Podman endpoint summary to TCP, preserving UDP observation without a
TCP/UDP probe, and repaired a Japanese typo and a fenced-code language tag.

Intermediate checks caught missing links to explicitly exempt historical Japanese
Plans, temporary hashes while sibling edits were in progress, one removed local
index route, trailing spaces, and extra EOF blank lines. These are editorial repair
items; neither validators nor their negative fixtures were weakened.

### Historical and generated inventory exclusions

Editorial exclusion does not imply a translation exception. The registry is unchanged.
The following 28 English documents are preserved; existing Japanese siblings are also preserved.

| English path | Translation status | Reason for editorial exclusion |
| --- | --- | --- |
| `docs/exec-plans/completed/agent-env-mvp.md` | existing explicit exception | historical execution evidence |
| `docs/exec-plans/completed/android-emulator-lease.md` | existing explicit exception | historical execution evidence |
| `docs/exec-plans/completed/android-emulator-review-2.md` | existing explicit exception | historical execution evidence |
| `docs/exec-plans/completed/android-emulator-review.md` | existing explicit exception | historical execution evidence |
| `docs/exec-plans/completed/android-ui-observer.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/bilingual-documentation-review.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/bilingual-documentation.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/browser-cdp-automation.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/compose-provider-podman-review.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/compose-provider-podman.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/flutter-android-review.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/flutter-android-runtime.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/multi-host-control-plane.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/multi-host-review-followup.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/multi-host-review-round-two.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/persistent-process-runtime.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/process-destroy-preview-review.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/repository-correctness-audit.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/repository-correctness-review.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/standalone-distribution-review.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/standalone-distribution.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/standalone-release-filter-review.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/standalone-release-finalization.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/standalone-release-review.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/standalone-verify-review.md` | English/Japanese pair | historical execution evidence |
| `docs/exec-plans/completed/worker-android-capacity-review.md` | English/Japanese pair | historical execution evidence |
| `docs/generated/db-schema.md` | existing explicit exception | generated schema |
| `docs/references/handoffs/CHATGPT_HANDOFF_30.md` | existing explicit exception | original historical handoff |

The current corpus contains 46 edited pairs, plus this execution pair (47). The raw audit reports above and exclusions in this table are unchanged.

### Final independent review evidence (2026-09-09)

All 46 current pairs passed separate English usability, standalone Japanese
readability and bilingual parity reviews. The policy author reviewed six root/
orientation pairs and twelve product pairs; the product author reviewed seven
policy pairs; the root editor reviewed thirteen design and eight ADR pairs
written by the design editor. These are reviewer roles, not self-review claims.
The final bilingual-policy addition on subject omission and established Japanese
terms was independently re-read in both languages and passed parity review.

Product code-fence preservation was checked against the starting revision; only
intentional CLI-block splitting/reordering changed placement. Policy requirements,
commands and identifiers were retained; AGENTS remains 105 lines. The final design
review repaired misleading TTL configurability and local/remote GC scope, clarified
conceptual APIs and improved dense Japanese Android/process paragraphs. No concrete
review finding remains open. Plan inventory classification was also reviewed
separately from the document authorship.

### Validation record (2026-09-09)

- Baseline and intermediate `go run ./tools/repoctl docs-check`: PASS.
- Baseline, intermediate and final `go run ./tools/repoctl check`: PASS, exit 0.
  Final stages: format-check, `go test ./...`, `go vet ./...`, docs-check,
  generated-check, arch-check. The final unit invocation reused valid Go test cache;
  this is not a new native-runtime or physical-host run.
- `git diff --check`: PASS. `git diff --name-only -- ':!*.md'`: empty.
- Original handoff SHA-256 before/after:
  `3f72a24b1767009cd66a0664ee62ba3359d9518599fb27d3d80fa26c7bfa864a`.
- No source, tests, translation exceptions, generated output or historical report
  bodies changed. No new Windows/macOS/native infrastructure coverage is claimed.
