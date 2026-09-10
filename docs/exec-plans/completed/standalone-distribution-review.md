---
status: completed
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
- [x] 2026-09-08: Fixed asset names/size, VCS fallback, release path matching/index flags with tests detecting the same defects.
- [x] 2026-09-08: Corrected bilingual completed status and state.db audit; current JP milestones match English, archive checksums/manifest are release-set siblings, root symlink and tests of first-write race safety pass.
- [x] 2026-09-08: Local harness/race and Verify 34206038055 / Release preview 34206043365 passed at 23f19fd; replied to and resolved all eleven threads.
- [x] 2026-09-08: Recorded outcomes and archived this bilingual plan.

## Surprises & Discoveries

Independent review rejected the first path-token-boundary approach: Go concatenates string-pool bytes, so a genuine absolute path may follow an ordinary letter. Preserve raw leakage detection outside exact known module occurrences and add a test detecting this path leak.

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

Completed at code revision `23f19fd`. All eleven reviewed findings have replies and resolved threads. Four were already fixed; seven received new code or documentation changes. Tests detecting the same defects, full harness/race, native Windows/macOS/Linux and release preview passed. Independent review caught and prevented a string-pool path-check regression before publication. No public tags/releases or history rewrites were performed.

## Context and Orientation

`internal/assets` owns generic byte materialization; `internal/buildinfo` owns
identity; `tools/repoctl/release*.go` owns strict release construction/validation.
Standalone product/design docs and the completed parent describe contracts.

## Plan of Work

Implement bounded tests detecting defects in portable names, oversized cached bytes, embedded
VCS fallback, path-leak false positives and hidden tracked edits. Correct metadata
and state.db audit in both languages. Validate existing thread fixes directly.

## Concrete Steps

Use supported Go on PATH: `go run ./tools/repoctl check`, `go test -race ./...`,
focused package tests and candidate release validation. Commit/push coherent fixes;
inspect native Verify/Release preview, reply using concrete evidence, then resolve.

## Validation and Acceptance

Each unresolved thread has code/document evidence and a reply. New defect-detection tests
fail before and pass after where practical. Existing tests remain strict. Native
Windows/macOS/Linux and release preview must succeed for the code revision.
Bilingual docs-check passes and working tree is clean after final push.

## Idempotence and Recovery

Never rewrite published history or change public tags/releases. Private temporary
Git fixtures own their index flags; do not modify flags in the user's repository.
Preserve failures and fix forward if CI reveals platform-specific behavior.

## Artifacts and Notes

2026-09-08 code revision `23f19fd`: local `repoctl check`, full `go test -race ./...`,
then final release-package race passed. `release-verify --out dist/pr6-review-candidate`
produced eight byte-identical files across two six-target builds and passed Linux
native smoke. `AGENT_ENV_RELEASE_CANDIDATE=../../dist/pr6-review-candidate go test ./tools/repoctl -run TestReleaseCandidate -count=1 -v`
passed all 19 cases, including real module-path collision and preserved genuine
path-leak negatives. Independent asset/buildinfo and final release reviews found
no confirmed remaining defects. A harness attempt during document editing failed
only stale translation hashes; both actual translations and hashes were updated,
and full harness passed afterward. Four previously fixed threads were replied to
and resolved after rechecking their evidence; all seven new fixes also passed native CI and received replies/resolution. Verify: https://github.com/mahcialet/agent-env/actions/runs/34206038055 . Release preview: https://github.com/mahcialet/agent-env/actions/runs/34206043365 .

PR: https://github.com/mahcialet/agent-env/pull/6 . Evidence is recorded here and
in thread replies, not in extra report files.

## Interfaces and Dependencies

Retain current package boundaries and native Go workflows; add no runtime tools.
