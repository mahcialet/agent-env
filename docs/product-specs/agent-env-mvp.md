---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# MVP specification

The Compose-backed CLI and repository harness are implemented. This document preserves the original required scope and acceptance criteria; completion of every platform and review gate is tracked from evidence in the [implementation plan](../exec-plans/active/agent-env-mvp.md). The [CLI contract](cli-contract.md) and [manifest reference](manifest-v1.md) describe current commands, fields, limits, and recovery behavior.

Implement `agent-env`, a reusable CLI that materializes one or more Git repositories at pinned commits into an isolated, disposable, inspectable environment lease.

A repository or workspace describes how it should be started in `.agent-env.yaml`. `agent-env` handles the common control-plane responsibilities:

- resolve refs to immutable commit IDs;
- create isolated Git worktrees;
- resolve a requested startup group into the minimum required component dependency closure;
- start an isolated Docker Compose project with a unique project name;
- persist lease, source, component, runtime, command, and event records in SQLite;
- expose accurate `list` and `show` views by reconciling registry state with actual Git and Docker state;
- run named tests and collect their output as evidence;
- safely destroy, garbage-collect, or quarantine environments;
- provide stable human-readable and machine-readable output for agents;
- run natively on Windows, macOS, and Linux from one Go codebase;
- make the `agent-env` repository itself legible and operable by coding agents through a repository-native development harness;
- preserve design intent, progress, validation evidence, and recurring operational lessons in version-controlled artifacts rather than relying on chat history.

The repository-development harness and the product runtime manifest are distinct:

```text
Repository-development harness
  AGENTS.md + ARCHITECTURE.md + docs/ + repoctl + CI
  -> tells Codex how to understand, change, and verify agent-env itself

Target-repository runtime harness
  .agent-env.yaml
  -> tells agent-env how to materialize a target repository/workspace
```

The implemented workflow is:

```text
agent-env plan   ./control-repo --stack api
agent-env create ./control-repo --stack api --ref refs/pull/3/head
agent-env list
agent-env test   <lease-id> api-smoke
agent-env logs   <lease-id>
agent-env destroy <lease-id>
```

The CLI must hide implementation details such as worktree locations, Compose file locations, Compose project names, transient ports, and resource cleanup order from the calling agent.

The long-term target is broader than Compose:

```text
repository/workspace definition
              |
              v
          agent-env
              |
      environment lease
       /      |       \
 worktrees  runtimes  evidence
             /   \
        Compose  Android Emulator
```

However, the first implementation must produce a reliable Compose-backed vertical slice rather than an incomplete collection of adapters.

---

## Scope

## Must implement

1. Repository-native harness bootstrap: concise `AGENTS.md`, `ARCHITECTURE.md`, indexed `docs/`, `docs/PLANS.md`, active ExecPlan, initial ADRs, and cross-platform `tools/repoctl`.
2. Go CLI skeleton and version command.
3. Platform state-path resolution with `AGENT_ENV_HOME` override.
4. SQLite database, migrations, repositories, leases, sources, components, resources, events, command runs, and artifacts.
5. `.agent-env.yaml` parsing with strict validation.
6. Component dependency resolution and stack planning.
7. Multiple local Git source repositories and immutable ref-to-commit resolution.
8. Detached worktree materialization for review leases.
9. Compose v2 runtime adapter with unique project names and selected services.
10. Compose config rendering, digesting, and basic policy diagnostics.
11. Create saga with compensation and event recording.
12. `list`, `show`, `renew`, `destroy`, `reconcile`, `gc`, and `doctor`.
13. Named test execution with log/evidence capture.
14. Human table output and stable JSON output.
15. Unit tests and fixture-driven integration tests.
16. Cross-platform build/test CI.
17. README, `ARCHITECTURE.md`, manifest documentation, security limitations, and Windows notes.
18. `repoctl docs-check`, `generated-check`, and an initial `arch-check` with stable diagnostic codes.
19. CI that invokes the repository harness and verifies the active plan, document indexes, generated schema, and architecture checks remain coherent.

## Should implement if the vertical slice is stable

- `agent-env init` candidate manifest generation for simple Compose repositories.
- Generated Compose override for dynamic loopback endpoint publishing.
- `--mine` filtering by owner.
- lightweight Compose service health/readiness checks.
- environment descriptor JSON under each lease directory.
- recording actual image IDs/digests used by running services.

## Explicitly out of scope for the first MVP

- Android Emulator lifecycle;
- Flutter APK build/install;
- browser/CDP control;
- UI snapshot commands;
- remote Git credential management;
- GitHub/GitLab provider-specific PR shorthand;
- untrusted fork sandboxing;
- local OCI registry management;
- automatic image promotion;
- environment checkpoint/clone;
- live stack expansion/shrink;
- generic long-running host-process adapter;
- distributed/multi-host lease coordination;
- GUI/TUI.

---

## Acceptance criteria

## Core behavior

1. A fixture repository with `api` and `dashboard` components can validate successfully.
2. `plan --stack api` resolves only the API dependency closure.
3. `plan --stack dashboard` resolves API plus Dashboard in deterministic topological order.
4. Invalid cycles and unknown references fail with actionable diagnostics.
5. `create` records the exact requested ref and resolved commit before runtime startup.
6. Two simultaneous leases from the same repository and commit receive distinct worktrees and Compose project names.
7. Destroying one lease does not alter the other lease’s containers, volumes, networks, or worktree.
8. `list --output json` returns both leases with source, stack, component, desired, and observed state.
9. If a Compose project is manually stopped or removed, a subsequent list/reconcile marks the lease degraded rather than still ready.
10. A create failure after worktree creation triggers compensating cleanup; if cleanup fails, the lease remains visible as quarantined.
11. A dirty tracked worktree is not silently deleted by GC.
12. `gc` without `--apply` deletes nothing.
13. A named test streams output, records exit code, and stores stdout/stderr evidence.
14. Multiple local repository sources are resolved and their commit tuple is visible in `show` and JSON output.

## Cross-platform behavior

15. The CLI compiles with `CGO_ENABLED=0` for at least:

    ```text
    windows/amd64
    darwin/amd64
    darwin/arm64
    linux/amd64
    linux/arm64
    ```

16. Unit tests pass on Windows, macOS, and Linux CI runners.
17. Paths containing spaces and Unicode are covered by tests.
18. No test requires Bash on Windows.
19. Command arguments containing spaces and quotes survive round-trip execution on each OS.
20. `doctor` reports missing `git`, `docker`, or Compose v2 without a panic or misleading success.

## Documentation and safety

21. README includes an explicit statement that environment isolation is not a malicious-code sandbox.
22. `ARCHITECTURE.md` explains the lease/source/runtime/reconciliation boundaries without duplicating low-level implementation details.
23. `.agent-env.yaml` schema and examples are documented.
24. Destructive commands document dry-run, force, and quarantine behavior.
25. Deferred Android/browser/registry features are documented as roadmap items, not presented as implemented.
26. `AGENTS.md` is no more than 150 lines, acts as a map, and all repository paths it references exist.
27. `docs/exec-plans/active/agent-env-mvp.md` contains the mandatory living-plan sections and accurately identifies current progress and next actions throughout implementation.
28. All design documents, product specifications, and ADRs are discoverable through their local indexes; a deliberately unindexed file causes `repoctl docs-check` to fail with an actionable diagnostic.
29. `go run ./tools/repoctl check` runs without Bash, Make, or PowerShell as a requirement on Windows, macOS, and Linux.
30. `repoctl generated-check` detects a deliberate drift in `docs/generated/db-schema.md` after migrations exist.
31. `repoctl arch-check` detects at least one fixture or synthetic forbidden dependency and explains the expected repair direction.
32. The checked-in active ExecPlan plus repository documents are sufficient for a fresh Codex run to identify the branch, current milestone, required commands, acceptance behavior, and recovery path without consulting this chat.
33. At completion, the ExecPlan is moved to `docs/exec-plans/completed/` with an outcomes/retrospective entry; historical handoff provenance remains under `docs/references/` if committed.

---
