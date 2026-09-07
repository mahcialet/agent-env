---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Establish bilingual English/Japanese repository documentation

[日本語](bilingual-documentation.ja.md)

This ExecPlan is a living document. Maintain it according to
`docs/PLANS.md`.

Expected branch: `feat/bilingual-documentation`.

## Purpose / Big Picture

After this work, durable human-facing documentation in `agent-env` is
maintained in both English and Japanese.

English documents remain the canonical normative source. Japanese documents
are first-class maintained translations rather than optional reference copies.

A change to a durable English document must update its Japanese translation
in the same coherent change. Repository documentation checks detect missing
or stale translations.

The initial migration also provides Japanese documentation for the existing
MVP and the Android Emulator functionality delivered through PR #2.

## Scope

In scope:

* define the repository bilingual-documentation policy
* define English/Japanese file naming and linking conventions
* add Japanese translations for existing durable human-facing documentation
* add Japanese translations for Android Emulator product and design documentation
* update repository indexes to expose both languages
* update `AGENTS.md` so future coding agents maintain both languages
* update `docs/PLANS.md` so ExecPlan completion requires synchronized durable documentation
* extend `repoctl docs-check` to detect missing translation pairs
* extend documentation validation to detect stale Japanese translations
* document explicit exceptions
* verify Windows, macOS and Linux repository-harness behavior

Out of scope:

* translating generated database/schema output unless separately justified
* translating historical handoff/reference archives
* translating source-code comments
* localizing CLI output
* localizing error messages
* translating third-party quoted material

## Documentation Model

English paths are canonical:

```
README.md
ARCHITECTURE.md
docs/product-specs/android-emulator.md
```

Japanese translations use `.ja.md`:

```
README.ja.md
ARCHITECTURE.ja.md
docs/product-specs/android-emulator.ja.md
```

Japanese documents must identify their canonical English source.

If English and Japanese content disagree, the English document is normative
until the translation is corrected. Such disagreement is considered
documentation drift and must not be intentionally introduced.

## Progress

* [x] 2026-09-08: Inspected indexed documentation at merged PR #2 (`fddcc56`) on the prepared dedicated branch.
* [x] 2026-09-08: Define the bilingual documentation contract.
* [x] 2026-09-08: Update `AGENTS.md`.
* [x] 2026-09-08: Update `docs/PLANS.md`.
* [x] 2026-09-08: Update documentation indexes.
* [x] 2026-09-08: Add Japanese translation metadata/conventions.
* [x] 2026-09-08: Implement missing-pair validation.
* [x] 2026-09-08: Implement stale-translation validation.
* [x] 2026-09-08: Translate repository-level durable documentation.
* [x] 2026-09-08: Translate product specifications.
* [x] 2026-09-08: Translate design documents.
* [x] 2026-09-08: Translate ADRs where applicable.
* [x] 2026-09-08: Translate Android Emulator documentation delivered by PR #2.
* [x] 2026-09-08: Ran the complete repository harness successfully; rerun after final review fixes.
* [ ] Record acceptance evidence and retrospective.

## Surprises & Discoveries

The user-provided untracked plan and dedicated branch were already present. Its malformed front matter was repaired without replacing its scope. There are no existing Japanese sibling documents. Existing generated schema, archived handoff, and four pre-migration completed plans need explicit historical/generated exceptions.

Record documentation categories that cannot reasonably follow the normal
pairing rule, validator portability problems, or cases where translation
drift cannot be determined mechanically.

## Decision Log

* Decision: English durable documentation is canonical and Japanese
  documentation is a maintained first-class translation.
  Rationale: A single normative source prevents ambiguity while still making
  Japanese documentation a required repository artifact.
  Date/Author: 2026-09-08 / maintainers.

* Decision: Japanese sibling files use the `.ja.md` suffix.
  Rationale: The relationship is visible, tooling is simple, and translations
  remain adjacent to their canonical documents.
  Date/Author: 2026-09-08 / maintainers.

* Decision: Generated and historical reference documents may be explicitly
  exempted from translation requirements.
  Rationale: Mechanical output and archival provenance have different
  maintenance economics from durable human-facing knowledge.
  Date/Author: 2026-09-08 / maintainers.

## Outcomes & Retrospective

Not completed.

## Context and Orientation

The repository currently treats `AGENTS.md` as the agent entry point and
`docs/index.md` as the durable documentation index.

`docs/PLANS.md` governs substantial changes and ExecPlan completion.

`tools/repoctl` provides the cross-platform repository harness and already
owns documentation validation. Bilingual validation belongs there rather
than in a platform-specific shell script.

Read before implementation:

* `AGENTS.md`
* `ARCHITECTURE.md`
* `docs/index.md`
* `docs/PLANS.md`
* `docs/QUALITY.md`
* `docs/product-specs/index.md`
* `docs/design-docs/index.md`
* `docs/adr/index.md`
* completed MVP ExecPlan
* completed Android Emulator ExecPlan

## Plan of Work

### Milestone 1 — Define the language contract

Document:

* canonical language
* translation naming
* required document categories
* explicit exemptions
* translation drift semantics
* index/link behavior

Update `AGENTS.md` and `docs/PLANS.md`.

### Milestone 2 — Add mechanical validation

Extend `repoctl docs-check`.

Validation must fail when:

* a required English document has no Japanese sibling
* a Japanese document has no canonical English source
* translation metadata points to the wrong source
* the canonical English source changed after the recorded translation state
* indexes contain broken language links

Validation must work without Bash, Make, PowerShell, symlinks, or CGO.

### Milestone 3 — Translate existing durable documentation

Create Japanese siblings for current durable documentation, including the
MVP and Android Emulator product/design contracts.

Preserve technical terminology where translation would reduce precision.
Commands, identifiers, schema fields, filenames, diagnostic codes and API
names remain unchanged.

### Milestone 4 — Integrate indexing and navigation

English documents link to Japanese siblings where useful.
Japanese documents link back to canonical English sources.

Indexes expose both language versions without duplicating authority.

### Milestone 5 — Acceptance and regression

Run:

```
go run ./tools/repoctl docs-check
go run ./tools/repoctl check
```

Add negative fixtures proving that a missing or stale Japanese translation
fails validation.

Verify the repository harness on native Windows, macOS and Linux CI.

## Concrete Steps

Use supported Go 1.26.8/1.27.1 from `/tmp/agent-env-toolchains/` on PATH.
Run `go test ./tools/repoctl`, `go run ./tools/repoctl docs-check` and
`go run ./tools/repoctl check`. Review translations before recording hashes.
Commit and push coherent verified changes; verify native CI before completion.

## Validation and Acceptance

A1. Every required durable English document has a Japanese sibling.

A2. Every Japanese sibling identifies an existing canonical English document.

A3. Modifying an English document without synchronizing Japanese causes
`repoctl docs-check` to fail.

A4. Updating both documents causes the same check to pass.

A5. Android Emulator product and design documentation is available in both
English and Japanese.

A6. Repository-level architecture, reliability, security, portability and
quality documentation is available in both languages.

A7. `AGENTS.md` explicitly requires bilingual durable documentation.

A8. `docs/PLANS.md` requires bilingual documentation completion for relevant
ExecPlans.

A9. Generated and historical-reference exemptions are explicit and tested.

A10. Full repository harness and native CI pass.

## Idempotence and Recovery

Translation checks must be deterministic and read-only.

Automatic metadata updates, if introduced, must not silently rewrite human
documentation during normal checks.

A failed migration may leave untranslated documents, but the validator must
report them individually and must never delete or overwrite either language.

## Artifacts and Notes

Record:

* migrated document count
* intentionally exempt document count
* stale-translation negative fixture
* missing-translation negative fixture
* native CI evidence

## Interfaces and Dependencies

Expected repository-harness surface:

```
repoctl docs-check
```

The implementation may introduce internal documentation metadata helpers but
must not require an external translation service or network access.

Translation itself may be produced by coding agents, but repository
correctness must not depend on a specific LLM or translation provider.

Implementation checkpoint (2026-09-08): translation freshness uses SHA-256 of the complete canonical English file with CRLF normalized to LF. A hash match records source revision, not proof of semantic translation quality; human/agent review remains required. Explicit exact-path exemptions live in `docs/translation-exceptions.json`; new completed plans retain Japanese counterparts. Normal checks are read-only.

Discovery (2026-09-08): `docs/RELIABILITY.md` still described the pre-review Compose inventory guard. Corrected that one sentence to match merged PR #2 and translated the corrected behavior; no runtime behavior changed. Initial translation check correctly reported missing Japanese files/placeholders and a last_verified mismatch; these are migration work, not weakened validation.

Translation checkpoint (2026-09-08): all 29 required documents have full Japanese siblings; six exact exceptions preserve generated/history material. Independent translation review corrected four precision issues: ExecPlan authority is not operation permission, Job exit/persistence wording must not imply an unsupported order, renewal resets expiry rather than always extending it, and the no-Docker statement applies specifically to individual Android-only leases. Commands/inline identifiers and safety/verification caveats were cross-checked.

Harness checkpoint: repoctl unit tests and Go 1.26 race repetitions passed (initial ten repetitions, then three after duplicate-JSON-key regressions); Go 1.27 unit tests passed. CGO-disabled Windows/amd64 and Darwin/arm64 cross-builds passed; these do not establish native CI behavior. Tests cover missing/stale pairs, synchronized edits, CRLF, orphan/provenance errors, metadata consistency, broken Japanese links/indexes, translated plans, restricted exceptions and duplicate JSON keys.

Independent harness review (2026-09-08) found three enforceability gaps: case-insensitive JSON struct field matching could bypass exact duplicate-key rejection; unbalanced metadata quotes were accepted; the shared source enumerator skipped hidden/vendor directories inside docs despite the policy covering them. Accepted all three findings for fixes and regressions before final verification. No exception was added to hide them.

Final local acceptance (2026-09-08): all three harness review findings were fixed with actual docsCheck regressions. `repoctl doctor`, `docs-check`, and full `check` passed with Go 1.26.8. Final repoctl race tests repeated three times and Go 1.27.1 unit tests passed; Windows/amd64 and Darwin/arm64 CGO-disabled cross-builds passed. The initial migration contains 29 required EN/JA pairs and six documented exact-path exceptions. Generated files, historical handoff and pre-migration completed plans are unchanged. Native CI remains to be verified after push.
