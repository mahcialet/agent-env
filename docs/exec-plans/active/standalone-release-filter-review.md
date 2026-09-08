---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Isolate release checkout filters

[日本語](standalone-release-filter-review.ja.md)

## Purpose / Big Picture

Keep PR #7 release checkout bytes independent of ambient Git attributes and filters.

## Progress

- [x] Inspected the unresolved filter review.
- [x] 2026-09-08: Global/system fixtures both failed before the fix and passed after configuration isolation.
- [ ] Validate, push, reply and resolve.

## Surprises & Discoveries

A smudge filter can change compiler input while a matching clean filter hides the change from Git status.

## Decision Log

2026-09-08 / maintainers: Isolate release Git commands from global/system config and attributes; retain repository-scoped Git identity and normal executable lookup.

## Outcomes & Retrospective

Not completed.

## Context and Orientation

Expected branch: `feat/standalone-release-finalization`. Follow AGENTS.md, ARCHITECTURE.md and docs/PLANS.md. Scope: tools/repoctl/release_source.go and its tests; bilingual release design.

## Plan of Work

Reproduce filtered-but-clean checkout in disposable repositories, isolate configuration, and preserve strict source validation.

## Concrete Steps

Run targeted regression before/after fix, repoctl check and repoctl race. Commit/push, then reply and resolve the thread.

## Validation and Acceptance

Global and system filter fixtures must actively transform an ordinary checkout yet leave private release bytes identical to committed content. Harness and source tests must pass; review thread must be resolved.

## Idempotence and Recovery

Change only child-process environment and arguments; never modify user Git configuration or public refs.

## Artifacts and Notes

Review: https://github.com/mahcialet/agent-env/pull/7#discussion_r3954872828 . Go 1.27.1: targeted source/filter regressions, `go run ./tools/repoctl check`, `go test -race ./tools/repoctl` and `AGENT_ENV_RELEASE_CANDIDATE=build go test ./tools/repoctl -run TestReleaseCandidate -count=1` passed. The latter covers all six real target artifacts and 18 candidate cases. Independent read-only review found no confirmed material defects.

## Interfaces and Dependencies

No new dependencies or POSIX shell in release code. Test Git filters invoke a native Go test helper through Git itself.

