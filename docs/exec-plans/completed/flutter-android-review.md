---
status: completed
owner: maintainers
last_verified: 2026-09-08
---

# Address PR 4 Flutter Android review

[日本語](flutter-android-review.ja.md)

Expected branch: `feat/flutter-android-runtime`. Starting revision: `447aa99`.
This plan governs review follow-up; the completed Flutter implementation plan
remains historical delivery evidence. Follow `docs/PLANS.md`.

## Purpose / Big Picture

Resolve all eight current PR 4 threads with regression evidence, synchronized
public documentation, per-thread replies and resolution after validated fixes.

## Progress

- [x] 2026-09-08: Confirmed clean working tree, branch and eight unresolved threads.
- [x] 2026-09-08: Correct colliding selected APK output paths, optional reverse observation and destroy previews.
- [x] 2026-09-08: Correct host Flutter doctor, default plan rendering, project symlinks and nonzero ADB diagnostics.
- [x] 2026-09-08: Synchronize existing English/Japanese public documentation.
- [x] 2026-09-08: Run regression tests, repository harness, race checks and native CI.
- [x] 2026-09-08: Reply to and resolve all eight threads, then archive this plan.

## Surprises & Discoveries

Review found integration gaps across existing entry points despite passing
feature tests. Record reproduced failures and unexpected results below.

## Decision Log

- 2026-09-08 / implementation: Reject selected applications whose source-relative
  APK output paths collide before effects. The current build-first lifecycle
  cannot preserve both outputs; explicit rejection is the reviewer's supported
  alternative to introducing another APK storage and cleanup lifecycle.
  Unselected applications must not prevent independent stacks from planning.
- 2026-09-08 / implementation: Keep the original completed plan intact and use
  this dedicated review plan on the existing PR branch. Separate app, CLI,
  adapters and documentation ownership avoids concurrent edits.

## Outcomes & Retrospective

All eight review requests are addressed in `4677b89`. Each thread received a
specific fix and regression-test reply, then was resolved after native CI passed.
The final GitHub query found eight threads and zero unresolved threads.

The fixes cover previously untested entry points and optional paths: default
human-readable planning, host-only prerequisites, absent reverse bindings and
safety previews. Regression tests now assert these independently of the original
happy-path feature tests. Selected colliding APK outputs are explicitly rejected
before effects rather than allowing one build to overwrite another. No new APK
storage lifecycle or adapter ownership boundary was introduced. Existing real
SDK evidence remains historical; this review was validated with native tests,
race checks and the CI Compose integration suite.

## Context and Orientation

App owns workload ordering and cleanup; Flutter owns portable build execution;
Android owns generic device commands and identity. Public contracts live in
`docs/product-specs/`, with README and roadmap as entry points.

## Plan of Work

Add failing regressions for each behavior, implement focused fixes without
weakening ownership or evidence guards, then update public documents in both
languages. Root owns app/planning/preview and this plan; delegated work owns CLI,
adapters and public docs. Root integrates and performs GitHub operations.

## Concrete Steps

Run focused `go test` commands with supported Go versions, then
`go run ./tools/repoctl check` and `go test -race ./...`. Commit and push coherent
verified changes, inspect native CI, reply to every thread with exact fix/test
references, resolve addressed threads, and move both plans to completed.

## Validation and Acceptance

| Review | Acceptance evidence |
| --- | --- |
| APK collision | Selected colliding outputs rejected before build/runtime effects; separate stacks remain valid. |
| Public docs | README, roadmap, manifest and CLI English/Japanese contracts agree with delivered behavior; docs-check passes. |
| Host doctor | Flutter-present/Android-missing fails honestly; neither toolchain is installed. |
| Default plan | Table output reports selected app/source/runtime/artifact/reverse without effects. |
| Optional reverse | No reverse or endpoint queries for an app with no declared mappings. |
| Project symlinks | In-tree project and ancestor symlinks rejected before and after build. |
| Destroy preview | Both durable build guards block removal previews, even with force, without mutations. |
| ADB failures | Nonzero command output remains bounded/redacted and preserves error identity. |

## Idempotence and Recovery

Do not rewrite published history. Preserve failed evidence and ownership checks;
no SDK installation, license acceptance, global cleanup or secret/path disclosure.
Re-run checks after corrections; unresolved threads remain visible until fixed.

## Artifacts and Notes

Record commits, commands, CI runs and thread results here. Local SDK/Flutter paths
are execution inputs and must not enter documentation.

## Interfaces and Dependencies

No new runtime dependency or cross-adapter import. Keep argument arrays and
native Windows/macOS/Linux behavior, existing lease JSON and resource ownership.

Review checkpoint (2026-09-08):

- All behavioral regressions failed against the initial implementation: four
  selected output collisions, unwanted reverse inspection, two unsafe preview
  guards, missing host Android diagnostics/default table data, six internal
  symlink cases and six discarded nonzero ADB command results.
- App targeted regressions plus existing build-guard tests passed with Go 1.27.1
  `-race -count=10` (12.410s). CLI `-race -count=1` passed (1.940s).
  Flutter/Android adapter tests passed on Go 1.26.8, and Go 1.27.1 race repeated
  five times. Independent read-only app review found no concrete issue.
- Initial full harness passed unit tests and vet, then correctly rejected stale
  translation hashes while the public documentation pairs were being edited.
  The documentation owner completed meaning review/hash sync and docs-check
  passed. The coherent full-harness rerun subsequently passed before commit.

Final local validation (2026-09-08):

- Go 1.27.1 and Go 1.26.8 `go run ./tools/repoctl check` passed all phases.
- Go 1.27.1 `go test -race ./...` passed (app 24.654s; CLI 1.811s;
  Android 2.110s; Flutter 3.039s).
- Direct review regressions: `TestSelectedApplicationAPKOutputsCannotCollide`,
  `TestApplicationWithoutReverseSkipsNetworkObservation`,
  `TestDestroyPreviewHonorsApplicationBuildBarriers`,
  `TestFlutterHostDoctorRequiresBothToolchains`,
  `TestFlutterHostDoctorMissingPrerequisitesIsPure`,
  `TestFlutterPlanTableIncludesSelectedApplicationRequirements`,
  `TestBuildRejectsInternalProjectSymlinks`, and
  `TestApplicationExecutionFailureRetainsDiagnosticsAndErrorIdentity` all passed.
- Public docs cover six synchronized English/Japanese pairs. No local Flutter
  path was added. Native CI and review-thread responses subsequently completed as recorded below.

Final completion evidence (2026-09-08):

- Fix commit: `4677b89`.
- [Push CI 34171724270](https://github.com/mahcialet/agent-env/actions/runs/34171724270)
  and [PR CI 34171727162](https://github.com/mahcialet/agent-env/actions/runs/34171727162)
  both passed all 12 jobs: native Windows/macOS/Linux on Go 1.26/1.27,
  five cross-build targets, and Linux race/real Compose integration.
- Replies were posted to all eight threads in PR 4; all eight resolve mutations
  succeeded. A fresh paginated-query check reported `total=8`, `unresolved=[]`,
  and no further page. Human review/merge remains separate.
- English/Japanese plan moved together to completed with translation metadata
  synchronized. No local SDK path or generated artifact was committed.
