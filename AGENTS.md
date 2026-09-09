# Agent entry point

[日本語](AGENTS.ja.md)

## Read first

1. [Architecture](ARCHITECTURE.md): responsibilities and dependency directions.
2. [Documentation index](docs/index.md): authoritative repository knowledge.
3. [MVP specification](docs/product-specs/agent-env-mvp.md): original requirements; [product contracts](docs/product-specs/index.md) define current capabilities.
4. [Plan policy](docs/PLANS.md): active ExecPlan requirements and mandatory structure.
5. [Completed MVP ExecPlan](docs/exec-plans/completed/agent-env-mvp.md): historical delivered scope and evidence.

## Standard workflow

- Inspect current branch, working tree, and existing differences before editing.
- Each substantial change uses a dedicated branch and an active ExecPlan.
- The active ExecPlan is the authority for the expected branch and current work.
- Preserve user changes, existing assets, and the MIT license.
- Read the relevant product and design documents before changing behavior.
- Maintain the active ExecPlan at meaningful checkpoints.
- Use Plan IDs, dependency selection and Git provenance from the plan policy.
- Never auto-promote drafts, auto-run human validation, or merge with stale review.
- Resolve uncertain public behavior before implementing dependent changes.
- Make a demonstrable change, then verify the relevant behavior.
- Record discoveries, decisions, failed checks, and next actions in the plan.
- Keep tests and behavior changes in the same coherent slice.
- Commit and push coherent verified milestones when already authorized.
- Never rewrite published history, force-push, or reconfigure remotes implicitly.
- Inspect final differences and distinguish your work from other contributors.

## Stable commands

Use a supported Go toolchain on PATH. Commands are implemented by the Go
repository harness; they must work without Bash, Make, or PowerShell.

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go run ./tools/repoctl test-unit
go run ./tools/repoctl test-integration
go run ./tools/repoctl docs-check
go run ./tools/repoctl generated-check
go run ./tools/repoctl arch-check
go run ./tools/repoctl generate
```

See [quality](docs/QUALITY.md) for underlying commands and verification scope.
Integration tests explicitly require Docker; missing prerequisites are not passes.

## Non-negotiable invariants

- Support native Windows, macOS, and Linux; WSL is separate Linux execution.
- Release builds require no CGO, POSIX shell, symlinks, or mandatory daemon.
- Execute external tools with argument arrays and OS-native path handling.
- Resolve all source refs to immutable commits before runtime startup.
- Keep source tuples normalized; never reduce a lease to one global commit.
- Repository-specific startup belongs in the target manifest.
- Components own dependencies; stacks declare explicit roots.
- Every lease owns distinct worktrees and explicit Compose project identities.
- SQLite records desired state; reconcile external resources for observed state.
- Preserve failed allocations, cleanup evidence, and quarantine visibility.
- Never silently delete unexpected tracked changes or ambiguous resources.
- GC defaults to dry-run; force deletion remains explicit and auditable.
- Environment isolation is not a malicious-code sandbox.
- Keep secrets out of logs, database metadata, and committed artifacts.
- A passing cross-build does not prove native runtime behavior.
- Report unsupported prerequisites and unverified acceptance honestly.

## Where knowledge belongs

- [Product specifications](docs/product-specs/index.md): user-visible contracts.
- [Design documents](docs/design-docs/index.md): durable mechanisms and rationale.
- [ADRs](docs/adr/index.md): concrete choices and rejected alternatives.
- [Plans](docs/PLANS.md): current execution, recovery, and acceptance evidence.
- [Security](docs/SECURITY.md): trust and host policy.
- [Reliability](docs/RELIABILITY.md): reconciliation and conservative cleanup.
- [Portability](docs/PORTABILITY.md): OS and process-execution constraints.
- [Roadmap](docs/roadmap.md): deferred features and open decisions.
- [References](docs/references/index.md): historical inputs and provenance.

Generate schema truth from migrations using repoctl; do not hand-maintain it.
Validators and tests own enforceable invariants; documents explain their intent.
Do not claim a rule is enforced until its validator and negative fixture pass.

## Documentation maintenance

Keep this file navigational and at most 150 lines; target 80–120 lines.
Use local indexes for all durable design, product, and ADR documents.
Maintain status, owner, and last_verified metadata on durable documents/plans.
Promote recurring findings into tests or checks rather than enlarging this file.
Avoid nested instruction files unless a subtree has materially different rules.
The archived handoff is historical input, not the permanent project manual.
Finish acceptance and prove merge into base before archiving a plan as completed.
Update links when moving a completed plan and retain its retrospective.

Durable human-facing documentation is maintained in English and Japanese.
English `*.md` files are canonical; Japanese translations use corresponding
`*.ja.md` paths. Update both languages in the same coherent change whenever
adding or changing a durable document. Review translation meaning before
refreshing its source hash; a matching hash alone does not prove accuracy.
Translate full meaning for each language’s readers; sentence and section order
need not match. Follow the language policy’s independent review workflow for
substantial documentation restructuring.
Generated documentation and historical archives have explicit exceptions.
Documentation checks must detect missing or stale translations; run docs-check
before completion. See the [language policy](docs/design-docs/bilingual-documentation.md)
for scope, metadata, indexes, and the exception registry.
