---
status: active
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

Deliver M1-M5 first, then split the multi-host control-plane plan into child plans and dogfood M6-M8 on that work.

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
- multi-host dogfooding.

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
- [ ] Run baseline repoctl/docs/race checks.
- [ ] M1: lifecycle policy, directories and metadata schema.
- [ ] M1: draft/paused/abandoned bilingual validators and negative fixtures.
- [ ] M1: self-migrate this plan to `EP-OPS-001`.
- [ ] M2: stable Plan ID, parent/dependency graph, cycle checks.
- [ ] M2: merged default and explicit stacked dependency semantics.
- [ ] M3: `repoctl plans list/check/graph/ready`.
- [ ] M4: deterministic selection, workstream conflicts and Git provenance.
- [ ] M5: merge-policy schema, latest-HEAD review gate, CI/thread/blocker gates.
- [ ] M5: automatic merge where machine-verifiable; guarded/manual fallback otherwise.
- [ ] Complete M1-M5 acceptance.
- [ ] Split multi-host into child ExecPlans using the new model.
- [ ] M6: parent/child orchestration and multi-host dogfood.
- [ ] M7: human-validation plan/preflight/evidence framework.
- [ ] M7: first multi-host human-validation checkpoint.
- [ ] M8: merge -> completion -> graph reevaluation -> next-plan loop.
- [ ] M8: feed human FINDING/BLOCKED results back into plans.
- [ ] Update bilingual durable docs.
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

Not completed.

At M1-M5 checkpoint record final schema, migration impact, Plan-ID grammar, dependency semantics, repoctl commands, Git provenance, merge-gate implementation, Codex/latest-HEAD limitations and first automatic-merge evidence.

At final completion record multi-host child graph, automatic transitions/merges, blockers represented as paused, draft promotions, parent finalization, human-validation results, validation evidence usefulness, human findings returned to plans and remaining manual steps.

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
- active `multi-host-control-plane` ExecPlan.

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

### Milestone 6 — Parent/child orchestration

Split multi-host into child plans. After each merged child, reevaluate graph and expose newly runnable plans. Completed children do not auto-complete parent; parent must reconcile integration/acceptance/docs/retrospective.

### Milestone 7 — Human validation

Add `plan_type: human-validation`, `execution_mode: human-kick`, stable scenario IDs and PASS/FINDING/BLOCKED.

Every human-validation plan contains prerequisites, actions, expected observations, evidence/logs, PASS criteria and follow-up rules. Human prepares the environment and explicitly kicks. Agent then performs machine-checkable preflight and reports READY/BLOCKED.

Provide purpose-built human-readable + structured evidence instead of unbounded debug logs.

### Milestone 8 — End-to-end orchestration

Connect plan selection -> branch -> implementation -> review -> merge gate -> merge -> archive -> dependency reevaluation -> next plan -> human checkpoint.

Invocation may remain explicit/CLI-driven; no scheduler daemon is required.

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
15. Reconcile and split multi-host plan.
16. Dogfood M6 child progression.
17. Implement M7 human-validation/evidence/preflight.
18. Create first multi-host human-validation plan.
19. Run it only after explicit human kick.
20. Feed findings back into plan graph.
21. Dogfood M8 end-to-end.
22. Update durable docs and run final checks.
23. Complete retrospective/archive.

## Validation and Acceptance

| ID | Required behavior | Evidence |
| --- | --- | --- |
| E1 | Existing plans have a documented migration path. | Pending |
| E2 | Five lifecycle states have strict semantics. | Pending |
| E3 | status/path mismatch fails validation. | Pending |
| E4 | paused requires reason and resume condition. | Pending |
| E5 | draft is never selected or auto-promoted. | Pending |
| E6 | multiple active plans are valid. | Pending |
| E7 | Plan IDs are unique and EN/JA synchronized. | Pending |
| E8 | Plan ID survives path/title/branch deletion. | Pending |
| E9 | parent and depends_on are distinct and validated. | Pending |
| E10 | missing/self/cyclic dependencies fail. | Pending |
| E11 | default dependency requires completed work merged into base. | Pending |
| E12 | stacked dependency is explicit. | Pending |
| E13 | repoctl plans list/check/graph/ready are deterministic. | Pending |
| E14 | runnable selection is deterministic; initial execution concurrency is one. | Pending |
| E15 | paused is never used solely as a concurrency queue. | Pending |
| E16 | workstream conflicts affect selection without changing lifecycle state. | Pending |
| E17 | branch/commit/PR carry consistent Plan-ID provenance. | Pending |
| E18 | applicable commits contain `ExecPlan:` trailer. | Pending |
| E19 | latest-HEAD review is invalidated by later commits. | Pending |
| E20 | free-form review prose is not merge authorization. | Pending |
| E21 | CI/native/integration/review-thread state participates in merge gate. | Pending |
| E22 | automatic plans merge only when all machine gates pass. | Pending |
| E23 | guarded/manual plans cannot auto-merge. | Pending |
| E24 | implementation-complete but unmerged remains active. | Pending |
| E25 | completed means merged into base and archived. | Pending |
| E26 | child merge triggers graph reevaluation. | Pending |
| E27 | children do not auto-complete parent. | Pending |
| E28 | parent requires parent-level finalization. | Pending |
| E29 | human-validation plans never auto-run. | Pending |
| E30 | human-validation preflight reports READY or concrete BLOCKED prerequisites. | Pending |
| E31 | human scenarios have stable IDs, expected observation and PASS criteria. | Pending |
| E32 | validation evidence has human-readable and structured forms. | Pending |
| E33 | human FINDING returns to a tracked Plan, not ad-hoc fix. | Pending |
| E34 | multi-host successfully dogfoods parent/child orchestration. | Pending |
| E35 | multi-host successfully dogfoods first human-validation checkpoint. | Pending |
| E36 | repository-correctness guardrails remain enforced. | Pending |
| E37 | bilingual durable docs reflect final lifecycle/automation behavior. | Pending |
| E38 | final repoctl/docs/translation/race/native checks pass. | Pending |
| E39 | both language plans contain direct evidence and retrospective before archival. | Pending |

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
