---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Address PR 6 standalone distribution review

[日本語](standalone-distribution-review.ja.md)

Expected branch: `feat/standalone-distribution`. Starting revision: `5aa0b59`.
This plan owns the remaining PR 6 review responses and fixes; the completed
parent remains historical evidence.

## Purpose / Big Picture

Close all eleven unresolved review threads with verified fixes or direct evidence
of previously delivered behavior, then reply and resolve each thread.

## Progress

- [x] 2026-09-08: Read harness, inspected clean branch and all review threads.
- [x] 2026-09-08: Fixed asset names/size, VCS fallback, release path matching/index flags with regression tests.
- [x] 2026-09-08: Corrected bilingual completed status and state.db audit; current JP milestones match English, archive checksums/manifest are release-set siblings, root symlink and first-write race regressions pass.
- [ ] Run local harness/race and native CI, reply/resolve all threads.
- [ ] Record outcomes and archive this bilingual plan.

## Surprises & Discoveries

Independent review rejected the first path-token-boundary approach: Go concatenates string-pool bytes, so a genuine absolute path may follow an ordinary letter. Preserve raw leakage detection outside exact known module occurrences and add this regression.

Before fixes: 25 invalid asset-name subcases passed incorrectly; a 32MiB cached file lacked size rejection; an actual clean Git build lost VCS identity; four flagged-index cases were accepted; module suffixes matched checkout roots. New tests reproduced each and pass after fixes.

Several unresolved comments predate merged release work. Reuse existing mkdir
stress, root symlink tests, translated progress and release-set layout evidence.
Archived parent metadata incorrectly remains active; registry audit says
registry.sqlite instead of the CLI's state.db.

## Decision Log

- 2026-09-08 / maintainers: Reject flagged index entries even when unchanged to keep cleanliness enforceable without rewriting the caller index. Reject cached sizes before open and bound the opened read; retain Windows sharing retries. Use embedded VCS only where linker fields are unknown, preserving explicit release identity. Apply Windows name restrictions on every host and verified module-path exclusions to path-leak checks.

- 2026-09-08 / maintainers: Keep fixes on the authorized PR branch with this active
  review plan. Do not reopen or overwrite historical parent acceptance evidence.
  Verify stale comments against current code before replying.

## Outcomes & Retrospective

In progress; no completion claimed before validation and thread operations.

## Context and Orientation

`internal/assets` owns generic byte materialization; `internal/buildinfo` owns
identity; `tools/repoctl/release*.go` owns strict release construction/validation.
Standalone product/design docs and the completed parent describe contracts.

## Plan of Work

Implement bounded regressions for portable names, oversized cached bytes, embedded
VCS fallback, path-leak false positives and hidden tracked edits. Correct metadata
and state.db audit in both languages. Validate existing thread fixes directly.

## Concrete Steps

Use supported Go on PATH: `go run ./tools/repoctl check`, `go test -race ./...`,
focused package tests and candidate release validation. Commit/push coherent fixes;
inspect native Verify/Release preview, reply using concrete evidence, then resolve.

## Validation and Acceptance

Each unresolved thread has code/document evidence and a reply. New regressions
fail before and pass after where practical. Existing tests remain strict. Native
Windows/macOS/Linux and release preview must succeed for the code revision.
Bilingual docs-check passes and working tree is clean after final push.

## Idempotence and Recovery

Never rewrite published history or change public tags/releases. Private temporary
Git fixtures own their index flags; do not modify flags in the user's repository.
Preserve failures and fix forward if CI reveals platform-specific behavior.

## Artifacts and Notes

PR: https://github.com/mahcialet/agent-env/pull/6 . Evidence is recorded here and
in thread replies, not in extra report files.

## Interfaces and Dependencies

Retain current package boundaries and native Go workflows; add no runtime tools.
