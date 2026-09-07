# Agent entry point

## Read first

1. [Architecture](ARCHITECTURE.md): responsibilities and dependency directions.
2. [Documentation index](docs/index.md): authoritative repository knowledge.
3. [MVP specification](docs/product-specs/agent-env-mvp.md): required behavior.
4. [Active ExecPlan](docs/exec-plans/active/agent-env-mvp.md): current work and evidence.
5. [Plan policy](docs/PLANS.md): mandatory living-plan structure.

## Standard workflow

- Inspect current branch, working tree, and existing differences before editing.
- Use `feat/agent-env-mvp` for this implementation.
- Preserve user changes, existing assets, and the MIT license.
- Read the relevant product and design documents before changing behavior.
- Use and maintain an ExecPlan for complex or multi-hour work.
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
Finish every acceptance requirement before moving the active plan to completed.
Update links when moving a completed plan and retain its retrospective.
