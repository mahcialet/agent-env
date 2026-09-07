---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Bilingual documentation review fixes

[日本語](bilingual-documentation-review.ja.md)

## Purpose / Big Picture

Address all five PR #3 review threads on `feat/bilingual-documentation`. This active plan is the authority for this follow-up work. Preserve the existing bilingual policy and portable Go harness; close enforcement gaps without changing runtime behavior.

## Progress

- [x] 2026-09-08: Confirmed clean branch at `41545f1` and read all five unresolved review threads.
- [x] 2026-09-08: Reproduced all five gaps with failing regression fixtures, implemented the checks and passed the full repoctl test suite and harness.
- [x] 2026-09-08: Full harness, Go 1.26/1.27 repoctl tests, race tests repeated three times and diff checks passed; committed and pushed `b94a94d` and `cc3b915`.
- [x] 2026-09-08: Replied to and resolved all five review threads; recorded outcomes and archived both plan languages.

## Surprises & Discoveries

The existing tests explicitly accepted arbitrary Japanese plan headings and any completed-plan exception. Positive fixtures also omitted root translation metadata and backlinks. These assumptions must change to match the documented policy; negative fixtures must remain independent of fixture regeneration.

Baseline evidence: the source-link, metadata and completed-plan tests failed because invalid inputs returned nil; canonical-index omission and all twelve missing Japanese sections likewise returned nil. After repair all these tests passed. One existing index negative mutated only one of two fixture links and therefore left a valid link; changing it to remove both restored the intended negative case without changing production acceptance.

Independent review found that textual link/heading extraction could accept code examples, escaped syntax, comments, unused reference definitions or a later duplicate reference definition. Added rendered-prose filtering shared by navigation and required-section checks, first-definition reference resolution, and regression cases. The prior implementation missed the policy gaps because positive fixtures and permissive tests shared its assumptions; future acceptance must map each requirement to an independently mutated negative fixture, not only to a passing repository snapshot.

A follow-up review caught a parsing-order regression after the main fix: removing comments before code spans misread literal comment markers, while removing comments after fences misread fences inside comments. Replaced ordering-only fixes with explicit comment/code state and retained both regressions as positive fixtures. The initial fix commit is `b94a94d`; threads remain open until the follow-up is verified and pushed.

## Decision Log

- 2026-09-08 / maintainers: Keep the PR branch for this review follow-up and use a new active bilingual plan; the delivered migration plan remains historical.
- 2026-09-08 / maintainers: Validate visible source links, both index languages, an immutable four-plan migration allowlist, Japanese plan-section equivalents, and required Japanese metadata. Reuse existing Go link/metadata checks and keep English section compatibility for translated plans.

## Outcomes & Retrospective

Completed all five requested enforcement fixes. Independent review additionally exercised thirteen known rendering/structure cases in an isolated copy and confirmed their resolution. Regression fixtures now distinguish visible navigation and actual plan sections from examples and comments. Fixed historical exceptions and valid translation metadata remain portable Go checks. Existing unrelated English documentation ambiguities remain outside this follow-up.

The original omission was a requirements-to-tests gap: fixture setup and permissive tests encoded implementation assumptions rather than proving each documented obligation. Preserve the failing-before/passing-after evidence and requirement-specific negative cases when extending these checks. Full local validation passed; remote CI for the new commits is tracked on PR #3 and is not claimed as completed by this record.

## Context and Orientation

`tools/repoctl/main.go` owns document structure, links and core metadata checks. `tools/repoctl/translations.go` owns bilingual pairs, hashes and exact exceptions. Tests under `tools/repoctl/` exercise isolated repository fixtures. `docs/design-docs/bilingual-documentation.md` and `docs/PLANS.md` define the contracts.

## Plan of Work

Add regression fixtures before fixing each check. Update positive fixtures to satisfy the policy without hiding regressions. Document accepted Japanese headings and retain fixed historical paths. Review source/hash synchronization after documentation changes.

## Concrete Steps

Run `go test ./tools/repoctl` for reproduction and repair, then `go run ./tools/repoctl doctor`, `go run ./tools/repoctl check`, and `go test -race ./tools/repoctl`. Run `git diff --check`, inspect staged changes, commit and push. Reply to each PR review thread with the change and evidence, then resolve it.

## Validation and Acceptance

All five formerly accepted invalid cases must fail with useful diagnostics; valid bilingual documents, reference links, CRLF checkouts and historical exceptions must still pass. Full harness and focused race tests must pass. Every addressed thread must have a reply and resolved status. Update both plan files before moving to completed.

## Idempotence and Recovery

Checks never rewrite translations. Preserve published commits and use ordinary follow-up commits; no force push. If verification fails, retain the failure evidence here and fix only the relevant cause.

## Artifacts and Notes

PR #3 review threads: `PRRT_kwDOURHsR86gCEZX`, `PRRT_kwDOURHsR86gCEZY`, `PRRT_kwDOURHsR86gCEZZ`, `PRRT_kwDOURHsR86gCEZd`, `PRRT_kwDOURHsR86gCEZe`.

## Interfaces and Dependencies

No new dependencies. Core workflow remains Go with native Windows, macOS and Linux support and no POSIX-shell requirement. Android and Flutter behavior is outside this change.
