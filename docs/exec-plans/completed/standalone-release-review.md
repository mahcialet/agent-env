---
status: completed
owner: maintainers
last_verified: 2026-09-08
---

# Correct release output staging after PR review

[日本語](standalone-release-review.ja.md)

## Purpose / Big Picture

Address PR #7 thread PRRT_kwDOURHsR86gHJ1z: allow a new release output inside a non-ignored worktree without weakening dirty-source guards.

## Progress

- [x] 2026-09-08: Inspected clean branch and the unresolved review thread.
- [x] 2026-09-08: Real six-target regression failed before the fix with the clean-tree diagnostic.
- [x] 2026-09-08: External construction and destination-local publication pass real-candidate, cleanup, source-guard, harness and repoctl race checks.
- [x] 2026-09-08: Pushed fix 6912ec6, replied in discussion_r3954818214 and confirmed the thread resolved.

## Surprises & Discoveries

Sibling staging becomes untracked content before final Git validation. Existing evidence used outside-tree or ignored output paths.

The first attempt reused an older candidate and correctly failed identity verification. The opt-in test now accepts `AGENT_ENV_RELEASE_CANDIDATE=build` to construct a fresh candidate from the current HEAD in an isolated clone.

## Decision Log

2026-09-08 / maintainers: Keep source cleanliness checks strict. Construct outside the worktree, then copy validated bytes into an owned destination sibling for same-filesystem publication after source validation.

## Outcomes & Retrospective

Completed on 2026-09-08. Commit `6912ec6` fixes non-ignored worktree output without excluding source changes from cleanliness checks. The real regression fails before and passes after; existing bytes, source guards and owned cleanup remain covered. Independent read-only review found no confirmed material regression. The single reviewed thread was answered and resolved. Both language versions are archived; hosted CI is separate from the successful local evidence recorded below.

## Context and Orientation

Expected branch: `feat/standalone-release-finalization`. Previous delivery: [release finalization](../completed/standalone-release-finalization.md). Relevant files: `tools/repoctl/release.go`, `tools/repoctl/release_e2e_test.go` and bilingual distribution design.

## Plan of Work

Add the real-output regression first, move construction staging, and preserve existing destinations and failed-build cleanup.

## Concrete Steps

Run the candidate regression before and after the fix, repoctl check and race. Commit and push only scoped changes; reply with evidence and resolve the reviewed thread.

## Validation and Acceptance

The unignored nested Unicode output must succeed and match existing candidate bytes. Existing dirty-source and output-preservation cases must continue to pass. All review threads must be answered and resolved.

## Idempotence and Recovery

Never remove caller output. Delete only owned temporary staging. Checks remain strict for unrelated tracked/index/untracked edits.

## Artifacts and Notes

Review: https://github.com/mahcialet/agent-env/pull/7#discussion_r3954761466 . Local Go 1.27.1: `AGENT_ENV_RELEASE_CANDIDATE=build go test ./tools/repoctl -run TestReleaseCandidate/nonignored_worktree_output -count=1 -v` failed before the fix (clean-tree diagnostic). Afterward, the complete real-candidate suite (18 subtests), publication cleanup/preservation and source guard tests passed. `go run ./tools/repoctl check` and `go test -race ./tools/repoctl` passed. Public refs were unchanged by tests.

## Interfaces and Dependencies

No new dependency, command or runtime behavior. Use native Go filesystem operations; permit temporary and output directories on different filesystems.

