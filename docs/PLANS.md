---
status: active
owner: maintainers
last_verified: 2026-09-09
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

Every active plan must contain the sections below as level-two Markdown headings.
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
direct evidence and Outcomes & Retrospective is filled. Update all links when
moving it. Preserve completed plans; do not delete them. Historical reference
archives are inputs, not operational authority.

Any substantial change that adds or modifies durable human-facing documentation
must include the corresponding Japanese translation before the ExecPlan can be
completed. This includes the living ExecPlan itself. Keep both plan files in the
same active/completed directory and update both sets of links when moving them.
Only explicitly registered pre-migration historical plans are exempt.

Follow the [language policy](design-docs/bilingual-documentation.md), including its
separate English, Japanese and semantic parity reviews for substantial
restructuring. Run `repoctl docs-check` before completion. A passing freshness
hash does not replace review of the translated meaning.
