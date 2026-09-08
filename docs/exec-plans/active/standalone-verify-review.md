---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Reject hidden caller index flags in release verification

[日本語](standalone-verify-review.ja.md)

Expected branch: `feat/standalone-distribution`. Starting revision: `385e27f`.
This plan owns the new PR 6 review finding; completed distribution plans remain historical evidence.

## Purpose / Big Picture

Make release-verify enforce the documented caller cleanliness policy before build effects.

## Progress

- [x] 2026-09-08: Read harness, confirm clean branch and one unresolved thread.
- [x] 2026-09-08: Reproduce the entry-point gap and share the cleanliness guard.
- [ ] 2026-09-08: Validate locally and in native CI, push, reply and resolve.
- [ ] 2026-09-08: Record outcomes and archive both languages.

## Surprises & Discoveries

All four new entry-point fixtures failed before the fix: execution reached LICENSE lookup in build preparation instead of rejecting index flags. After sharing the guard, the source and entry-point regression tests pass.

The previous regression tested releaseVersion only. release-verify validated the caller with porcelain and applied releaseVersion only to its clean private clone, missing caller index flags.

## Decision Log

- 2026-09-08 / maintainer: Extract the existing flag-aware cleanliness guard and reuse it at release-verify entry. Preserve private preview tags and caller index contents; exercise the command entry point rather than only the helper.

## Outcomes & Retrospective

Pending implementation and validation.

## Context and Orientation

`tools/repoctl/release_source.go` owns Git validation; `release_verify.go` owns private preview orchestration. The standalone product/design documents already require rejecting flagged entries even when unchanged.

## Plan of Work

Add both flags with changed/unchanged fixtures. Prove failure before the fix, extract the guard, and run repository checks. Keep runtime adapters untouched.

## Concrete Steps

Run `go test ./tools/repoctl -run TestReleaseVerifyRejectsHiddenIndexEntries -count=1`, `go run ./tools/repoctl check`, and `go test -race ./tools/repoctl`. Commit/push the fix, inspect native Verify and Release preview, then reply and resolve the thread.

## Validation and Acceptance

All four negative fixtures reject with the index-flags diagnostic before output or build logs, preserve caller bytes/index/refs, and need no public tag. Existing release tests, full harness, native CI and preview pass. Both translations pass docs-check.

## Idempotence and Recovery

Fixtures use temporary repositories only. Do not rewrite published history, mutate caller flags, merge, or publish tags/releases. Record failures and repair with additive commits.

## Artifacts and Notes

Local `go run ./tools/repoctl check` and `go test -race ./tools/repoctl` passed on 2026-09-08, including the four entry-point regressions. Native CI and preview remain pending.

Review: https://github.com/mahcialet/agent-env/pull/6#discussion_r3956143329 . Evidence belongs here and in the thread reply.

## Interfaces and Dependencies

Only repository harness internals change; use native Go and Git argv with no new dependencies or shell workflow.
