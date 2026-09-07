---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# ExecPlan policy

Use a version-controlled ExecPlan for complex, multi-hour, or multi-milestone work. Keep the [MVP plan](exec-plans/active/agent-env-mvp.md) self-contained and update it at every meaningful checkpoint. Include exact paths, commands, results, remaining work, decisions, and safe recovery steps. Evidence distinguishes tests run locally, cross-builds, and actual native CI.

Mandatory sections are: Purpose / Big Picture; Progress; Surprises & Discoveries; Decision Log; Outcomes & Retrospective; Context and Orientation; Plan of Work; Concrete Steps; Validation and Acceptance; Idempotence and Recovery; Artifacts and Notes; Interfaces and Dependencies.

Durable plan metadata contains status, owner, and last_verified. Progress uses dated checked/unchecked items. A checkbox means observed completion, never intention. Preserve failed verification results and unresolved platform gaps. Record decisions with date, author role, and rationale; promote durable decisions into ADRs.

Move a plan from active to completed only after every acceptance requirement has direct evidence and outcomes/retrospective is filled. Update all links when moving it. Completed plans preserve history; do not delete them. Historical reference archives are inputs, not operational authority.
