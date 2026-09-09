---
status: active
plan_id: EP-OPS-001
plan_type: implementation
priority: 10
merge_policy: guarded
base_branch: master
branch: feat/execplan-lifecycle-orchestration
workstreams:
  - repoctl
  - documentation
owner: maintainers
last_verified: 2026-09-09
---

# Add ExecPlan lifecycle orchestration and automated delivery gates

[日本語](execplan-lifecycle-orchestration.ja.md)

This ExecPlan is a living document. Maintain it according to `docs/PLANS.md`.

Bootstrap Plan ID: `EP-OPS-001`

Expected branch: `feat/execplan-lifecycle-orchestration`

Starting revision: `8f17283dd35436ae5ef6ad43fe0d7a3377a594f2` (`master`, PR #13 merged).

This plan starts with the current legacy ExecPlan metadata. Milestone 1 introduces the new lifecycle schema, then this plan self-migrates to `plan_id: EP-OPS-001` without breaking intermediate docs checks.

## Purpose / Big Picture

Turn ExecPlans into a mechanically checked delivery graph:

```text
DRAFT
  -> explicit promotion
ACTIVE
  -> select runnable plan
  -> branch / implementation / tests / PR
  -> independent/Codex review
  -> merge gate
  -> automatic merge
COMPLETED
  -> dependency graph reevaluation
  -> next ACTIVE
  -> HUMAN VALIDATION checkpoint
       -> explicit human kick
       -> PASS / FINDING / BLOCKED
```

Implementation unit = ExecPlan. Human validation unit = coherent product milestone.

Deliver M1-M5 first. M6 reconstructs completed multi-host history without rewriting it; M7 creates a new human-validation Plan for the delivered product; M8 combines historical replay, explicit human validation and forward live dogfooding on the next new ExecPlan.

## Scope

In scope:

- statuses: `draft`, `active`, `paused`, `completed`, `abandoned`;
- matching directories under `docs/exec-plans/`;
- immutable `plan_id`;
- `plan_type`: implementation, review, human-validation;
- `parent` and `depends_on` as distinct relations;
- dependency cycle/missing-reference checks;
- merged-by-default dependency satisfaction; explicit stacked exception;
- `priority`, logical workstreams/conflicts;
- `pause_reason`, `resume_when`, draft promotion criteria;
- `repoctl plans list/check/graph/ready`;
- multiple active plans, initial implementation concurrency = 1;
- deterministic Plan-ID branch naming;
- `ExecPlan:` commit trailer;
- PR Plan-ID metadata;
- automatic/guarded/manual merge policy;
- latest-HEAD review gate;
- CI/native/integration/review-thread merge gates;
- post-merge completion/archive;
- child-merge graph reevaluation;
- parent finalization;
- human-validation plans, checklists, preflight, evidence and explicit human kick;
- PASS/FINDING/BLOCKED feedback into the plan graph;
- bilingual policy/check enforcement;
- historical multi-host replay and new Human Validation dogfooding.

Out of scope initially:

- starting coding agents from repoctl;
- scheduler daemon;
- multi-repository plan graphs;
- >1 implementation plan running concurrently by default;
- automatic draft promotion;
- automatic human-validation execution;
- stacked PRs by default;
- parsing free-form `LGTM` comments as authorization;
- bypassing GitHub rulesets/branch protection.

## Progress

- [x] 2026-09-09: Record exact base and create `feat/execplan-lifecycle-orchestration`.
- [x] 2026-09-09: Run baseline repoctl/docs/race checks.
- [x] 2026-09-09: M1: lifecycle policy, directories and metadata schema.
- [x] 2026-09-09: M1: draft/paused/abandoned bilingual validators and negative fixtures.
- [x] 2026-09-09: M1: self-migrate this plan to `EP-OPS-001`.
- [x] 2026-09-09: M2: stable Plan ID, parent/dependency graph, cycle checks.
- [x] 2026-09-09: M2: merged default and explicit stacked dependency semantics.
- [x] 2026-09-09: M3: `repoctl plans list/check/graph/ready`.
- [x] 2026-09-09: M4: deterministic selection, workstream conflicts and Git provenance.
- [x] 2026-09-09: M5: merge-policy schema, latest-HEAD review gate, CI/thread/blocker gates.
- [x] 2026-09-09: M5: automatic merge where machine-verifiable; guarded/manual fallback otherwise.
- [ ] Complete M1-M5 acceptance.
- [x] 2026-09-09: Map completed multi-host history into a derived lifecycle/dependency replay.
- [x] 2026-09-09: M6: historical orchestration replay of multi-host.
- [x] 2026-09-09: M7: human-validation plan/preflight/evidence framework.
- [ ] M7: first multi-host human-validation checkpoint.
- [ ] M8: historical replay + Human Validation + next new ExecPlan forward live dogfood.
- [ ] M8: feed human FINDING/BLOCKED results back into plans.
- [x] 2026-09-09: Update bilingual durable docs.
- [ ] Final repoctl/docs/translation/race/native checks.
- [ ] Complete retrospective and archive both plans.

## Surprises & Discoveries

Record current repoctl assumptions about active/completed-only paths, unknown frontmatter handling, path-based indexes, bilingual move/hash behavior, GitHub/Codex reviewed-SHA signals, auto-merge/ruleset behavior, commit-trailer edge cases, stacked PR retargeting, parent-pause semantics and human-validation evidence usability.

Do not weaken merge/review gates merely to make automation appear complete.

## Decision Log

- 2026-09-09 / maintainers: Use five states: draft, active, paused, completed, abandoned. Draft is not implementation-ready; paused requires a real blocker.
- 2026-09-09 / maintainers: Multiple plans may be active. Active means runnable, not currently executing.
- 2026-09-09 / maintainers: Initial implementation concurrency is one plan.
- 2026-09-09 / maintainers: Stable Plan ID is independent of file path/title and propagates to branch, commit trailer and PR.
- 2026-09-09 / maintainers: Parent hierarchy and execution dependency are separate.
- 2026-09-09 / maintainers: Dependency satisfaction defaults to completed work merged into the configured base; stacked work is explicit.
- 2026-09-09 / maintainers: Draft is never auto-promoted.
- 2026-09-09 / maintainers: Paused requires `pause_reason` and `resume_when` and is not a concurrency queue.
- 2026-09-09 / maintainers: Normal plans may auto-merge after proven gates; guarded/manual remains for higher-risk changes.
- 2026-09-09 / maintainers: Review authorization is valid only for current PR HEAD.
- 2026-09-09 / maintainers: Completed means merged into base and archived, not merely implementation-finished.
- 2026-09-09 / maintainers: Human validation is a distinct plan type, never auto-run, and consists of checklist + purpose-built evidence + dedicated plan + explicit human kick.
- 2026-09-09 / maintainers: Build M1-M5 first, then dogfood M6-M8 on multi-host.

## Outcomes & Retrospective

Implementation checkpoint (2026-09-09): M1-M5 tooling, M6 historical replay and
the M7 Human Validation framework are delivered in Draft PR #14. Five-state
metadata, immutable IDs, dependency selection and Git provenance are validated
without rewriting completed records. The merge adapter remains read-only and
guarded because live atomic enforcement and machine acceptance proof are missing.
No automatic merge has been demonstrated.

The Plan remains active and incomplete. Actual multi-host environment preparation,
explicit human kick and scenario evidence, the next genuinely new normal
ExecPlan's forward live dogfood, final native verification and merge into base
remain outstanding. Historical replay and synthetic fixtures do not replace these
observations. Final parent acceptance and archival must reconcile their results.

Independent review found ancestry, identity-preservation and bounded-input defects
that passing initial fixtures had missed. Regression tests now isolate those
failure modes. Keep direct acceptance evidence separate from implementation
status and from assumptions about live review or human observations.

## Context and Orientation

Read before implementation:

- `docs/PLANS.md` / `.ja.md`;
- bilingual-documentation design and completed review plans;
- repository-correctness audit/review plans;
- `tools/repoctl`;
- `AGENTS.md` / `.ja.md`;
- `ARCHITECTURE.md` / `.ja.md`;
- `docs/QUALITY.md` / `.ja.md`;
- `docs/roadmap.md` / `.ja.md`;
- GitHub Actions workflows and current review/check behavior;
- completed `multi-host-control-plane` ExecPlan and its actual branch/commit/PR/merge history.

Current policy already requires a dedicated branch, active ExecPlan, direct acceptance evidence and retrospective before completion. Preserve those principles while generalizing lifecycle and automation.

## Plan of Work

### Milestone 1 — Lifecycle contract and policy

Introduce:

```text
docs/exec-plans/
├── draft/
├── active/
├── paused/
├── completed/
└── abandoned/
```

Define legal states/transitions and state-specific mandatory metadata. Update PLANS policy, repoctl structural checks and EN/JA negative fixtures. Bootstrap-migrate this plan to `EP-OPS-001`.

### Milestone 2 — Stable identity and dependency graph

Add repository-unique immutable Plan IDs. Resolve `parent` and `depends_on` by ID. Reject duplicate/missing/self/cyclic references. Define merged-default dependency satisfaction and explicit stacked exception.

### Milestone 3 — `repoctl plans`

Implement deterministic read-only commands:

```text
repoctl plans list
repoctl plans check
repoctl plans graph
repoctl plans ready
```

`ready` distinguishes runnable active plans from paused/draft/completed/abandoned and execution conflicts.

### Milestone 4 — Execution selection and Git provenance

Initial selection:

```text
status=active
AND dependencies satisfied
AND no blocker
AND no workstream conflict
ORDER BY priority, plan_id
LIMIT 1
```

Define deterministic branch naming, `ExecPlan:` commit trailer and PR Plan-ID metadata. Prove provenance remains queryable after branch deletion.

### Milestone 5 — Review and merge gate

Support `automatic`, `guarded`, `manual`.

Automatic eligibility requires current-HEAD review, required checks/native/integration evidence, no blocking review/thread, acceptance complete except merge/archive, bilingual docs checks, base/ruleset freshness and no explicit blocker.

Never parse free-form review prose as authorization. If Codex review lacks a reliable machine-readable current-HEAD signal, retain a guarded/manual fallback.

### Milestone 6 — Historical orchestration replay

Use completed multi-host ExecPlan, branch, commits, PR and merge history as inputs.
Reconstruct and validate how the new lifecycle/dependency model would have
progressed. Derived parent/child nodes are replay data, not rewritten historical
Plans. Validate graph reevaluation after child merge and explicit parent-level
acceptance; completed children alone never complete a parent.

### Milestone 7 — Human Validation dogfood

Create a new `plan_type: human-validation` ExecPlan for the already delivered
multi-host product. Require `execution_mode: human-kick`, stable scenario IDs,
prerequisites, actions, expected observations, evidence and PASS criteria.
The human prepares the environment and explicitly kicks; only then may preflight
report READY/BLOCKED. Evidence has readable and structured forms. PASS/FINDING/
BLOCKED feeds back into tracked Plans rather than ad-hoc fixes.

### Milestone 8 — End-to-end acceptance

Combine historical replay, real Human Validation and forward live dogfooding on
the next new normal ExecPlan. Exercise selection, branch/commit/PR provenance,
current-HEAD review, merge, post-merge archival, graph reevaluation and next-plan
selection. Invocation may remain explicit/CLI-driven; no scheduler daemon is
required. Keep future-plan and human-kick evidence pending until actually observed.

## Concrete Steps

1. Confirm base and create branch.
2. Add bilingual active plans under current policy.
3. Run baseline checks.
4. Update lifecycle policy/directories/schema.
5. Add lifecycle/bilingual negative fixtures.
6. Self-migrate this plan to `EP-OPS-001`.
7. Add Plan-ID/dependency graph.
8. Add repoctl plans commands.
9. Add selection/workstream rules.
10. Add branch/commit/PR provenance.
11. Add merge policy/gates.
12. Prove post-review commit invalidates review gate.
13. Prove safe auto-merge or document guarded fallback.
14. Complete M1-M5 acceptance.
15. Reconstruct multi-host history without rewriting completed Plans.
16. Validate M6 historical parent/child progression.
17. Implement M7 human-validation/evidence/preflight.
18. Create first multi-host human-validation plan.
19. Run it only after explicit human kick.
20. Feed findings back into plan graph.
21. Combine replay, Human Validation and next new ExecPlan forward live dogfood.
22. Update durable docs and run final checks.
23. Complete retrospective/archive.

## Validation and Acceptance

| ID | Required behavior | Evidence |
| --- | --- | --- |
| E1 | Existing plans have a documented migration path. | PASS: PLANS lifecycle migration policy and frozen legacy completed allowlist; new Plans require schema. |
| E2 | Five lifecycle states have strict semantics. | PASS: strict five-state metadata parser with state-specific positive/negative fixtures. |
| E3 | status/path mismatch fails validation. | PASS: TestPlanMetadataStrictValidation rejects directory/status mismatch. |
| E4 | paused requires reason and resume condition. | PASS: paused metadata negatives require both pause_reason and resume_when. |
| E5 | draft is never selected or auto-promoted. | PASS: draft criteria required; planReadiness never selects or changes draft state. |
| E6 | multiple active plans are valid. | PASS: graph/readiness fixtures accept multiple active nodes and select at most one. |
| E7 | Plan IDs are unique and EN/JA synchronized. | PASS: TestLoadPlanGraphPairsAndReferences rejects duplicate IDs and EN/JA metadata drift. |
| E8 | Plan ID survives path/title/branch deletion. | PASS: identity history survives renaming and merge-source proof does not depend on retained branch refs; deletion regressions. |
| E9 | parent and depends_on are distinct and validated. | PASS: separate parent and dependency graphs; hierarchy alone adds no execution dependency. |
| E10 | missing/self/cyclic dependencies fail. | PASS: missing/self/cyclic-reference negative fixtures. |
| E11 | default dependency requires completed work merged into base. | PASS: completed status plus base ancestry, identity and unique merge-source trailer required; arbitrary draft-containing commits rejected. |
| E12 | stacked dependency is explicit. | PASS: explicit stacked enum; actual consumer-branch ancestry regression, declared base only before branch creation. |
| E13 | repoctl plans list/check/graph/ready are deterministic. | PASS: sorted graph/JSON output and deterministic read-only list/check/graph/ready commands. |
| E14 | runnable selection is deterministic; initial execution concurrency is one. | PASS: priority then ID ordering, one selection and Git-worktree execution-slot checks. |
| E15 | paused is never used solely as a concurrency queue. | PASS: readiness reports concurrency without mutating states; paused policy requires real blocker. |
| E16 | workstream conflicts affect selection without changing lifecycle state. | PASS: workstream/conflict reasons emitted by read-only readiness; lifecycle states retained. |
| E17 | branch/commit/PR carry consistent Plan-ID provenance. | PASS: deterministic branches, unique commit trailer and visible standalone PR Plan-ID validation. |
| E18 | applicable commits contain `ExecPlan:` trailer. | PASS: current delivery-range nonmerge commits require exactly one ExecPlan trailer; bootstrap commits carry EP-OPS-001. |
| E19 | latest-HEAD review is invalidated by later commits. | PASS: later-HEAD and stale-check/base negative tests in TestPlanMergeGateFailsClosed. |
| E20 | free-form review prose is not merge authorization. | PASS: gate consumes structured approvals only; prose is absent from authorization model. |
| E21 | CI/native/integration/review-thread state participates in merge gate. | PASS: required CI/docs/native/integration coverage, pinned app identity and unresolved-thread/review negatives. |
| E22 | automatic plans merge only when all machine gates pass. | MODEL PASS / LIVE GUARDED: complete trusted synthetic evidence can pass; live atomic protection and machine acceptance proof unavailable, so no automatic merge. |
| E23 | guarded/manual plans cannot auto-merge. | PASS: guarded/manual never authorize automatic merging. |
| E24 | implementation-complete but unmerged remains active. | PASS SO FAR: this unmerged implementation stays active; final completion is not claimed. |
| E25 | completed means merged into base and archived. | PASS: schema requires archived directory/full merge SHA; plans check verifies base reachability and Plan-specific merge provenance. |
| E26 | child merge triggers graph reevaluation. | PASS: replay invokes the real graph/readiness engine before and after verified child merge. |
| E27 | children do not auto-complete parent. | PASS: replay leaves parent blocked after implementation child merge; no automatic parent completion. |
| E28 | parent requires parent-level finalization. | PASS: hypothetical review merge only selects still-active parent reconciliation; parent acceptance remains separate. |
| E29 | human-validation plans never auto-run. | PASS: human nodes never auto-selected; no-kick returns before reads/probes/writes. |
| E30 | human-validation preflight reports READY or concrete BLOCKED prerequisites. | FRAMEWORK PASS: named executable/endpoint preflight produces READY or concrete BLOCKED; real environment pending. |
| E31 | human scenarios have stable IDs, expected observation and PASS criteria. | PASS: EP-MHOST-001 contract validates eight unique scenarios with actions, observations and PASS criteria. |
| E32 | validation evidence has human-readable and structured forms. | PASS: JSON and Markdown evidence, bounded same-snapshot digest, immutable bundle directories. |
| E33 | human FINDING returns to a tracked Plan, not ad-hoc fix. | FRAMEWORK PASS / REAL FEEDBACK PENDING: FINDING requires a tracked draft/active review Plan and records its ID; actual Plan updates remain operator work. |
| E34 | Completed multi-host history is reconstructed and validated under the new lifecycle/dependency model. | PASS: actual PR #12 Git/GitHub facts and real readiness-engine replay recorded below. |
| E35 | A new multi-host Human Validation checkpoint runs after explicit human kick; M8 also records next new ExecPlan forward live dogfood. | PENDING: explicit human kick/scenarios and next new normal ExecPlan forward live dogfood. |
| E36 | repository-correctness guardrails remain enforced. | PASS: full harness and original negative fixtures retained; fixture metadata migrated without relaxing assertions. |
| E37 | bilingual durable docs reflect final lifecycle/automation behavior. | PASS: reviewed EN/JA policy, Plan metadata and command limits; docs-check/hash validation. |
| E38 | final repoctl/docs/translation/race/native checks pass. | LOCAL PASS / NATIVE PENDING: full check, full race, final repoctl race and Windows/macOS cross-builds passed; native CI pending. |
| E39 | both language plans contain direct evidence and retrospective before archival. | PENDING FINALIZATION: bilingual implementation evidence recorded; final retrospective/archive waits for human/forward acceptance and base merge. |

## Idempotence and Recovery

Graph queries are read-only. EN/JA status moves must stay synchronized. Never auto-promote draft, invent pause reasons, auto-merge stale review, blindly retry uncertain merge, mark completed before base merge, or auto-run human validation.

After interruption, re-read repository plan graph, Git state and GitHub PR/check/review/merge state and derive the next action. Do not trust in-memory scheduler state.

## Artifacts and Notes

Potential derived outputs:

```text
repoctl plans list
repoctl plans graph
repoctl plans ready
```

Potential transition events later include plan_promoted/paused/resumed/selected, review_gate_satisfied, merge_gate_satisfied, plan_merged/completed, human_validation_kicked/passed/finding.

Do not advertise these as implemented before their milestones land.

## Interfaces and Dependencies

Primary implementation surface is expected in `tools/repoctl/`, with reusable plan parsing/model code extracted internally if needed.

Conceptual layers:

```text
plan parser/model
 -> lifecycle/dependency validator
 -> query/selection engine
 -> Git provenance checker
 -> merge-gate adapter
 -> human-validation contract
```

Keep GitHub transport behind a narrow adapter. The product runtime `agent-env` must not depend on ExecPlan orchestration.

M1-M4 should add no new production dependency. For M5 prefer existing GitHub/workflow capabilities rather than a large SDK unless justified.

## Unresolved Issues to Settle During Milestone 1

1. Exact Plan-ID grammar/allocation.
2. Whether migrated plans default `plan_type=implementation`.
3. Exact `depends_on` YAML shape and satisfaction enum.
4. Structured versus prose `resume_when`.
5. Draft promotion-criteria representation.
6. Priority ordering convention.
7. Workstream/conflict schema.
8. Abandoned dependency replacement semantics.
9. Historical completed-plan ID migration policy.
10. Commit-trailer enforcement boundaries.
11. Branch normalization/truncation.
12. PR Plan-ID field format.
13. Codex machine-readable reviewed-SHA signal.
14. GitHub ruleset versus repoctl merge-gate authority.
15. Auto-merge enabling mechanism.
16. Base freshness/merge-queue policy.
17. Safe merge failure reconciliation.
18. Required/optional child representation.
19. Parent active/paused semantics while children run.
20. Human-validation evidence/operator-note location.
21. Whether any preflight occurs before explicit human kick (default: no).
22. Whether validation bundle belongs to product CLI or repoctl/harness.
23. Human FINDING -> follow-up Plan creation/linking.
24. Future parallel execution running-state authority.

## Execution notes (2026-09-09)

The starting tree contained only this untracked English/Japanese Plan pair.
Multi-host and reader-first documentation are already merged (PRs #12 and #13).
The multi-host split described by the original plan is therefore stale; M1-M5
proceeds independently while the maintainer clarifies a historical replay plus
new human-validation checkpoint for M6-M8. No completed implementation is redone.

Baseline `repoctl check` passed unit tests and vet but failed docs-check because
the supplied Japanese Plan omitted translation_of/source_sha256. Add that metadata
after reviewing the pair, then rerun. The race baseline is running.
The work branch is the explicit bootstrap exception to future Plan-ID naming.

2026-09-09 / maintainer scope correction: M6 historical orchestration replay, M7 new Human Validation dogfood, M8 replay + Human Validation + next new ExecPlan forward live dogfood replace the stale multi-host re-split. Preserve original completed history. Baseline race passed; docs-check passed after supplied translation metadata repair.

### Initial schema and migration decisions

- This bootstrap Plan is `EP-OPS-001`, `merge_policy: guarded`, with the explicitly
  requested branch retained. Future branch names derive from type and lowercase ID.
- Dependencies use `plan_id` and `satisfaction: merged|stacked`; omitted satisfaction
  means merged. Readiness uses per-consumer Git ancestry, not status text alone.
- Priority is a required nonnegative integer, lower first; ties use Plan ID. Worktree
  branch observations reserve the single execution slot. Human validation is never
  selected without its separate explicit kick workflow.
- Completed legacy Plans are an exact frozen filename allowlist; all new Plans
  require structured metadata. Legacy language exceptions remain separate.
- The gate is fail-closed and read-only unless trusted current-base policy and atomic
  enforcement can be established. Existing GitHub prose comments are not approval.
  No current automatic merge eligibility is claimed for this guarded bootstrap.

### Implementation and review evidence (2026-09-09)

M1-M4 are implemented by `plans_model.go`, `plans.go` and the existing docs/translation
harness integration. New active/draft/paused/abandoned Plans and new completed
Plans are strict YAML; historical completed records stay unchanged. Positive test
fixtures now copy their full lifecycle metadata to Japanese; negative missing-section
fixtures retain valid schema so they still isolate section detection.

M5 delivers a pure current-HEAD merge decision model and a live read-only GitHub
adapter. Tests permit only complete trusted evidence and reject stale approval,
stale checks/base, self approval, blocking threads and unknown evidence. The live
adapter intentionally returns BLOCKED until trusted base policy, atomic ruleset
protection and machine acceptance proof are available. This is the documented
guarded fallback, not a claim that live automatic merge has been demonstrated.

M6 ran against actual PR #12: merge `084da57de177c0a09bc3cb61ae99faff8bd79a94`,
source head `cc55382e29db86794f832cbb5c0e3e6d775e722e`, base
`dc63308e53f68f8be99f7cbf59cafc78f7296b71`, branch `feat/multi-host-control-plane`,
23 commits, merged at `2026-09-09T11:10:50Z`. `gh pr view 12` independently matched
these Git facts. The replay uses the real graph/readiness engine for derived
implementation/review/parent nodes. Verified implementation merge unblocks review;
parent remains blocked. A hypothetical review merge selects parent reconciliation,
never marks historical review approved or parent complete.

M7 created paused `EP-MHOST-001` and eight stable scenarios. Human preflight and
record commands require explicit kick before reads/probes/writes. Synthetic tests
cover missing prerequisites, loopback READY, no inferred PASS, tracked FINDING,
immutable evidence directories, oversized/trailing JSON and same-snapshot hashing.
Actual human environment preparation, kick, scenario execution and tracked Plan
feedback remain pending. M8 also requires the next genuinely new normal ExecPlan;
no synthetic replay or this bootstrap is relabeled as that forward live evidence.

Independent review of root integration caught (and regression tests now cover):
stacked ancestry tested against a changed base rather than the consumer branch;
quoted YAML IDs bypassing completed-section checks; deletion of all lifecycle
pairs bypassing identity preservation. The root additionally tightened merge proof
so an arbitrary reachable draft-containing commit cannot satisfy a dependency.
Independent human-framework review caught a bounded-reader EOF hiding trailing JSON
followed by an unbounded hash reread; validation now parses and hashes one bounded
snapshot. Policy and Human Validation pairs passed independent English, standalone
Japanese and parity reviews. No checks were weakened to resolve fixture failures.

The full `repoctl check` and `go test -race ./...` passed after integration. Focused
new provenance/merge tests passed again after the final proof hardening. Native CI
runs the new `plans check` with full Git history; its remote results are pending.

### Delivery checkpoint

Draft PR #14 carries `ExecPlan: EP-OPS-001`; commits `b1f8c8e`, `c95268a` and
`c79d709` retain the same trailer. `plans provenance --plan EP-OPS-001 --pr-body`
passed. Live `plans gate --plan EP-OPS-001 --pr 14 --repo mahcialet/agent-env`
returned BLOCKED with observed HEAD/base and the concrete missing trusted-base
policy reason; no merge was attempted. The final model check also rejects a
stacked dependency that is abandoned even if a caller supplies stale proof.
A real-Git regression deletes the merged fixture branch and still verifies its
Plan-specific merge evidence. Both focused tests passed.

Delivery commits also include `9cca40b`. Its local full `repoctl check` and
`go test -race ./tools/repoctl -count=1` passed (race: 9.616s). The English and
Japanese Progress checkboxes now both record the delivered M3 commands; the
Japanese checkbox had remained unchecked despite the implementation evidence.

### PR #14 CI readiness fixture repair (2026-09-10)

PR Verify run 34356431558 at `03b656d` failed in `go test -race ./...`:
`TestLifecycleCreatePersistedIntentAndUniqueIsolation` returned
`context deadline exceeded` on its second Create. Docker integration was skipped.
The push run 34356426394 at the same HEAD passed. The shared fixture gave healthy
creation only 50ms for readiness, including SQLite persistence. Use the existing
5s convention for healthy fixtures and explicitly retain 50ms in
`TestLifecycleReadinessTimeoutRollsBack`; production timeouts and rollback/isolation
assertions are unchanged.

An attempted `errors.Is(context.DeadlineExceeded)` assertion failed during the
50-repeat race run because SQLite can report `sql: transaction has already been
committed or rolled back` when cancellation races persistence. Removed that new
assertion rather than imposing a new production error contract; the original
non-nil error and complete resource cleanup assertions remain. Full local harness
and full race checks passed during repair; final repeat evidence follows below.

Final focused validation: `go test -race ./internal/app -run
'^TestLifecycle(CreatePersistedIntentAndUniqueIsolation|ReadinessTimeoutRollsBack)$'
-count=50 -timeout=3m` passed (22.606s), exercising both success and rollback.
