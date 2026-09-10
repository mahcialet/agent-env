---
status: active
owner: maintainers
last_verified: 2026-09-10
---

# ExecPlan policy

[日本語](PLANS.ja.md)

Use an active ExecPlan to make substantial work executable, reviewable and
recoverable. The plan records current instructions and acceptance evidence;
completed plans retain the history of delivered work.

## Start substantial work

Each substantial change requires a dedicated branch and a version-controlled
ExecPlan under `docs/exec-plans/active/`. The active plan is the authority for the
expected branch and current work. State both explicitly.

Keep the plan self-contained. An implementer must be able to find the exact paths,
commands, results, remaining work, decisions and safe recovery steps without
reconstructing a conversation. The [completed MVP plan](exec-plans/completed/agent-env-mvp.md)
is a historical example of scope and evidence.

## Required structure

Every noncompleted plan and every completed plan using the lifecycle schema must contain the sections below as level-two Markdown headings.
English plans use the English names. Japanese plans may use either the Japanese
equivalents or the English names. Arbitrary labels do not satisfy the structure
check.

| English heading | Japanese heading |
| --- | --- |
| Purpose / Big Picture | 目的 / 全体像 |
| Progress | 進捗 |
| Surprises & Discoveries | 想定外の発見 |
| Decision Log | 判断の記録 |
| Outcomes & Retrospective | 成果と振り返り |
| Context and Orientation | 背景と構成 |
| Plan of Work | 作業計画 |
| Concrete Steps | 具体的な手順 |
| Validation and Acceptance | 検証と受け入れ |
| Idempotence and Recovery | 冪等性と復旧 |
| Artifacts and Notes | 成果物と注記 |
| Interfaces and Dependencies | インターフェースと依存 |

## Maintain evidence during execution

Durable plan metadata must contain `status`, `owner` and `last_verified`.
Update the plan at every meaningful checkpoint:

- Use dated checked/unchecked items in Progress. A checkbox records observed
  completion, never intention.
- Record exact commands and results. Distinguish local tests, cross-builds and
  actual native CI; preserve failed checks and unresolved platform gaps.
- Record decisions with date, author role and rationale. Promote durable
  decisions into ADRs.
- Keep remaining work and safe recovery steps current.

## Complete and archive

Move a plan from active to completed only after every acceptance requirement has
direct evidence, Outcomes & Retrospective is filled, and the delivered work is
merged into its configured base branch. Implementation-finished but unmerged
work remains active. Record the full merge commit SHA as `merge_commit` and verify
that it is an ancestor of the base before archiving. Update all links when
moving it. Preserve completed plans; do not delete them. Historical reference
archives are inputs, not operational authority.

Any substantial change that adds or modifies durable human-facing documentation
must include the corresponding Japanese translation before the ExecPlan can be
completed. This includes the living ExecPlan itself. Keep both plan files in the
same lifecycle directory and update both sets of links when moving them.
Only explicitly registered pre-migration historical plans are exempt.

Follow the [language policy](design-docs/bilingual-documentation.md), including its
separate English, Japanese and semantic parity reviews for substantial
restructuring. Run `repoctl docs-check` before completion. A passing freshness
hash does not replace review of the translated meaning.

## Lifecycle and migration

The directory under `docs/exec-plans/` must equal the plan's `status`:

| State | Meaning and transition |
| --- | --- |
| `draft` | Proposed work; explicit promotion after `promotion_criteria` are met makes it active. Never auto-promote or select it. |
| `active` | Eligible for execution after dependency and conflict checks; active does not mean currently running. |
| `paused` | A real blocker prevents progress. Record `pause_reason` and `resume_when`; explicitly resume when the condition is met. Do not use pause as a concurrency queue. |
| `completed` | Accepted work merged into base, with retrospective and both language files archived. |
| `abandoned` | Work intentionally discontinued, with `abandonment_reason`. It does not satisfy dependencies. |

Draft, active and paused work may be explicitly abandoned. Keep its evidence.
Moving or renaming a plan never changes its ID. A replacement for abandoned work
gets a new ID; update affected dependencies explicitly and record the decision.
Completed children never complete their parent automatically. The parent must
reconcile integration, its own acceptance criteria, documentation and retrospective.
A parent waiting for children stays active unless it has a recorded real blocker.

All noncompleted plans use the new schema; there are no implicit defaults for
`plan_type`. Historical completed plans on the harness's exact frozen legacy
allowlist retain their existing content and metadata. That lifecycle migration
exception is distinct from the four historical translation exceptions. New plans
cannot join either allowlist by merely omitting fields. Newly archived plans keep
the lifecycle schema and mandatory sections.

## Identity and metadata

Keep `status`, `owner` and `last_verified`. Lifecycle plans additionally declare:

| Field | Contract |
| --- | --- |
| `plan_id` | Repository-unique immutable ID matching `EP-[A-Z][A-Z0-9]*-[0-9]{3,}`; allocate an unused number in the chosen namespace. |
| `plan_type` | Explicit `implementation`, `review`, or `human-validation`. |
| `base_branch` | Explicit Git base branch; merge and dependency evidence is evaluated against it. |
| `priority` | Nonnegative integer; lower values are selected first. |
| `workstreams`, `conflicts` | Lists of logical workstream names; conflicts prevent simultaneous selection without changing status. |
| `blockers` | Optional list of explicit blockers; any entry prevents selection. |
| `branch` | Optional declaration of the deterministic branch below; only bootstrap `EP-OPS-001` keeps `feat/execplan-lifecycle-orchestration`. |
| `parent` | Optional parent Plan ID; hierarchy alone imposes no dependency. |
| `depends_on` | List of objects with `plan_id` and optional `satisfaction`: `merged` (default) or `stacked`. |
| `merge_policy` | `automatic`, `guarded`, or `manual`; missing machine-verifiable authorization never permits automatic merging. |

Draft `promotion_criteria` is a nonempty list of nonempty strings. Paused
`pause_reason` and `resume_when`, and abandoned `abandonment_reason`, are
nonempty prose strings.
Completed plans require a full hexadecimal commit SHA in `merge_commit`.
Human-validation plans require `execution_mode: human-kick`; they never auto-run.
Keep lifecycle values identical in English and Japanese; translate their
explanations in the body. ExecPlan metadata uses the defined YAML lists/objects;
ordinary document metadata retains its single-line scalar format.

Reject missing, duplicate, self-referential and cyclic graph references. A merged
dependency requires completed work and a merge commit reachable from the dependent
plan's configured base. `stacked` is an explicit exception, not the default:
prove the prerequisite branch's commit is included in the dependent branch and
recheck base/target relationships after merging or retargeting. Parent relationships
and dependency relationships are validated separately.

## Query, selection and provenance

The repository harness provides deterministic read-only graph queries:

```text
go run ./tools/repoctl plans list
go run ./tools/repoctl plans check
go run ./tools/repoctl plans graph
go run ./tools/repoctl plans ready
```

Select active work only when dependencies are satisfied, no blocker remains and
workstreams do not conflict. Order by `priority`, then `plan_id`; initial
implementation concurrency is one. Reevaluate from repository and Git state after
a child merge or interruption. Queries do not start agents or mutate lifecycle
state, and no scheduler daemon is required.

Use the lowercase Plan ID as branch suffix: implementation `feat/ep-ops-001`,
review `fix/ep-ops-001`, human validation `validate/ep-ops-001`. Do not truncate or
replace the ID with a title slug. The bootstrap plan `EP-OPS-001` explicitly keeps
`feat/execplan-lifecycle-orchestration`. Applicable implementation commits carry
an `ExecPlan: EP-OPS-001` trailer; the PR body has the same standalone line.
The ID remains queryable in commit/PR history after branch deletion. Historical
commits are not rewritten to add trailers; validate the current plan's delivery
range, excluding unrelated inherited history.

## Review and merge gates

`automatic` permits merging only after every machine-verifiable gate passes.
`guarded` and `manual` require explicit human authorization and cannot auto-merge.
Resolve applicable merge policy from the trusted base, not unreviewed PR changes.
GitHub branch protection and rulesets remain authoritative constraints.

Require reliable structured approval tied to the current PR HEAD, all required
CI and applicable native/integration evidence, passing bilingual documentation
checks, no blocking review or unresolved review thread, no explicit blocker, and
acceptance complete except merge/archive. Verify base and ruleset freshness again
immediately before merging. A later commit invalidates previous approval.
Free-form `LGTM` and other review prose are never authorization. If a review
provider cannot expose a reliable current-HEAD approval signal, retain a guarded
or manual fallback and record the limitation; do not weaken the gate.

On uncertain merge failure, reread remote PR, merge, base and check state before
retrying. Never blindly repeat a merge or archive merely because a request was
sent. Automatic merge evidence and remaining manual steps belong in the active
plan. Future parent orchestration and human preflight/evidence commands must not
be advertised as delivered before their milestones pass. Human validation needs
an explicit human kick before machine preflight or scenario execution.

### Command workflow and current limits

Run `git fetch origin` before decisions that depend on remote progress. `plans`
commands inspect local refs; a missing local `master` can use `origin/master`.
The native Verify jobs fetch full history and run `plans check` separately from
the portable documentation checks.

```text
go run ./tools/repoctl plans list
go run ./tools/repoctl plans check
go run ./tools/repoctl plans graph
go run ./tools/repoctl plans ready
go run ./tools/repoctl plans provenance --plan EP-OPS-001 --pr-body <file>
go run ./tools/repoctl plans gate --plan EP-OPS-001 --pr <number> --repo <owner/repository>
go run ./tools/repoctl plans replay --plan <completed-plan-path> --merge <full-sha> --pr <number>
```

`check` additionally preserves existing IDs against `master` even if all current
Plans are removed or retargeted. A merge proof must carry the Plan's unique commit
trailer in the merge source head (or squash result), not merely contain a draft
Plan somewhere in its tree. `ready` uses the actual consumer branch for stacked
ancestry; before branch creation it uses the explicitly declared starting base.
These commands do not create branches, commit, push, merge or archive automatically.

`gate` is a read-only fail-closed adapter. It loads optional version-1
`.github/execplan-gates.json` policy from the PR base commit, with a `policies`
map keyed by Plan ID; the entries follow `PlanMergePolicy` in
`tools/repoctl/plans_gate.go`. No trusted reviewer/check policy is provisioned by
this bootstrap. Atomic ruleset enforcement and machine-readable acceptance proof
are not yet available to the live adapter, so it reports BLOCKED and requires
the guarded/manual workflow even if the pure decision model could allow complete
synthetic evidence. It never arms deferred auto-merge. Replay outputs Git facts
and clearly labeled projections; the optional PR number is verified separately
against GitHub, and merge history alone never proves review approval.

Human-validation commands and the scenario contract are described in the
[multi-host checkpoint](exec-plans/paused/multi-host-human-validation.md).
`preflight` requires explicit `--kick`, named local executables/endpoints and an
unused evidence directory. `record` stores an operator's observation, not an
inferred runtime PASS; FINDING requires a tracked draft/active review Plan.
BLOCKED records a required paused-Plan update. A human must still perform the
scenario actions, review evidence and reconcile results into the Plan.

## Test-architecture acceptance evidence

Substantial test or correctness work must classify acceptance evidence using
[the quality evidence rules](QUALITY.md#evidence-classes-and-test-architecture).
Record the invariant, intended failure stage, fixture completion contract and
finding disposition before repairs. Preserve failed approaches and distinguish
forced fail-before/pass-after controls, native observations, tooling checks and
repetition. Counts cannot upgrade evidence class. Explain unavailable controls
and deferred siblings; do not complete a Plan with unfulfilled acceptance or
substitute compilation for native execution.
