---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# ExecPlan policy

[日本語](PLANS.ja.md)

Each substantial change uses a dedicated branch and a version-controlled active ExecPlan under `docs/exec-plans/active/`. The active ExecPlan is the authority for the expected branch and current work; state both explicitly in the plan. Keep it self-contained and update it at every meaningful checkpoint. Include exact paths, commands, results, remaining work, decisions, and safe recovery steps. Evidence distinguishes tests run locally, cross-builds, and actual native CI. The [completed MVP plan](exec-plans/completed/agent-env-mvp.md) preserves historical scope and evidence.

Mandatory sections are: Purpose / Big Picture; Progress; Surprises & Discoveries; Decision Log; Outcomes & Retrospective; Context and Orientation; Plan of Work; Concrete Steps; Validation and Acceptance; Idempotence and Recovery; Artifacts and Notes; Interfaces and Dependencies.

Japanese active plans require the same sections, using the Japanese equivalents below or their English headings. Use level-two Markdown headings; arbitrary labels do not satisfy the structure check.

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

Durable plan metadata contains status, owner, and last_verified. Progress uses dated checked/unchecked items. A checkbox means observed completion, never intention. Preserve failed verification results and unresolved platform gaps. Record decisions with date, author role, and rationale; promote durable decisions into ADRs.

Move a plan from active to completed only after every acceptance requirement has direct evidence and outcomes/retrospective is filled. Update all links when moving it. Completed plans preserve history; do not delete them. Historical reference archives are inputs, not operational authority.

Any substantial change that adds or modifies durable human-facing documentation
must include the corresponding Japanese translation before the ExecPlan can be
completed. This includes the living ExecPlan itself. Keep both plan files in the
same active/completed directory and update both sets of links when moving them.
Only explicitly registered pre-migration historical plans are exempt. Follow the
[language policy](design-docs/bilingual-documentation.md) and verify `repoctl docs-check`.
