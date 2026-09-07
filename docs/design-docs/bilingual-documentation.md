---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Bilingual documentation

[日本語](bilingual-documentation.ja.md)

Durable human-facing documentation is maintained in English and Japanese.
English is the canonical normative source; a disagreement is documentation drift
to fix, not permission to leave Japanese stale. Japanese translations are
maintained artifacts, not optional summaries.

## Scope and naming

All repository-root Markdown and Markdown under `docs/` follow this policy.
Japanese siblings replace `.md` with `.ja.md`, for example `README.ja.md` or
`docs/product-specs/android-emulator.ja.md`. This includes indexes, agent guidance,
ADRs and new living/completed ExecPlans. Source comments, CLI messages, third-party
code and test fixtures outside this documentation scope are not localized.

English and Japanese changes belong in the same coherent change. Translate the
full meaning, limitations and evidence; preserve commands, identifiers, diagnostic
codes and paths. English indexes expose both language versions. Japanese indexes
link to Japanese siblings; links to exempt historical/generated documents remain
English. Each Japanese document visibly links back to its canonical source.

## Translation metadata and review

Every Japanese document requires valid status, nonempty owner and last_verified
(YYYY-MM-DD), including root translations whose English source has no front matter.
When the English source has these fields, their values must match. Japanese front
matter also adds:

```yaml
translation_of: docs/design-docs/bilingual-documentation.md
source_sha256: <64 lowercase hexadecimal characters>
```

Metadata fields use single-line plain strings or balanced quoted strings; nested
or multiline YAML is outside the harness metadata format.

`translation_of` is the exact repository-relative, forward-slash English path.
`source_sha256` is SHA-256 of the complete English file, including front matter,
with CRLF line endings normalized to LF. Other bytes, including a final newline,
are significant. This is deterministic across native Windows, macOS and Linux.

Read the English diff, update the Japanese meaning and links, review both, then
record the new source digest. Do not refresh the digest without reviewing the
translation. The checker detects an unacknowledged source change, not semantic
translation quality or a dishonest acknowledgment. Normal `docs-check` never
writes files or calls an external translation provider.

## Explicit exceptions

[The exception registry](../translation-exceptions.json) lists exact paths and
nonempty reasons. Allowed categories are generated output under `docs/generated/`,
reference archives under `docs/references/handoffs/`, and the pre-migration
completed ExecPlans listed there. Directory-wide or implicit exemptions are not
allowed. Indexes remain bilingual even when their targets are archival.

The initial exceptions are the generated database schema, the original archived
handoff, and the four completed MVP/Android implementation and review plans that
predate this migration. The checker fixes that completed-plan allowlist to `agent-env-mvp.md`,
`android-emulator-lease.md`, `android-emulator-review.md` and
`android-emulator-review-2.md` under `docs/exec-plans/completed/`; registry entries
cannot exempt additional completed plans. Their English evidence remains intact. New plans require
both languages through completion; an old plan materially revised into current
guidance should lose its exemption and gain a translation.

## Checks and completion

`go run ./tools/repoctl docs-check` validates required pairs, canonical source
metadata, freshness hashes, visible source backlinks, both language indexes,
required Japanese plan sections and local links. Missing
pairs and stale translations are reported as errors. An orphan Japanese document
or one pointing to a different canonical path also fails. Existing link, metadata,
architecture and generated-output checks remain in force.

The active ExecPlan cannot be completed until affected Japanese documents are
updated and these checks pass. Move both plan files together, repair links and
refresh the translated plan's source metadata after the final English edits.
