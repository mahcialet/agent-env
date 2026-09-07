---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# Security and trust

Environment leases isolate names, worktrees, and lifecycle ownership. They are **not a malicious-code sandbox**. Repository Dockerfiles, Compose builds, tests, package hooks, and command probes can execute code with the privileges available to their tools. Use trusted repositories or a separately controlled outer sandbox for untrusted code.

## Manifest authority

Planning and creation read the control checkout's `.agent-env.yaml` by default. `--manifest` explicitly selects a trusted manifest path; the canonical snapshot and digest are saved with the lease. Source refs choose pinned runtime/test source content, not a silently substituted manifest from the target revision. Review changes to both the manifest and the code it executes. Trusted base/PR overlay merging and remote credential management are deferred.

Owner labels and `--mine` are advisory filters. Anyone with access to the local state directory and Docker daemon has the corresponding host authority. There is no distributed authentication or hostile multi-user isolation.

## Built-in host policy

The current CLI uses built-in defaults: TTL 4 hours, maximum TTL 24 hours, and 8 active reservations. Quarantined and incompletely cleaned leases retain reservations. A configurable host policy file and a separate maximum-parallel-create setting are not implemented.

Before startup, normalized Compose configuration is checked for privileged containers, host networking, fixed container names, fixed published host ports, Docker socket access, device passthrough, and unsafe mounts. Bind paths must stay within allocated source roots, including after symlink resolution. External networks/volumes, globally shared names on selected resources, and unsafe/custom volume drivers or driver options are rejected. These checks reduce accidental host access and collisions; they do not make Docker builds or repository commands trustworthy.

Only selected services and their reachable resource definitions enter the immutable execution snapshot. Ownership labels, a unique project, captured Docker context, and a configuration digest are retained. Execution and cleanup verify the saved configuration and observed resource identities. Unselected named resources cannot become collateral cleanup targets.

## Credentials and evidence

The process environment is not dumped into SQLite or evidence. Named tests may explicitly reference host values through `${env:NAME}`. Credential-like test environment keys must use an exact host-variable reference rather than a literal value in the manifest. Recognized inherited credential values appearing literally elsewhere in the canonical manifest are also rejected before reservation. The expanded environment remains execution input; recorded argv, streamed logs, and copied artifacts use configured and recognized inherited secret values for redaction.

This is value-based redaction of known credentials, not a universal secret detector. Credentials produced inside a tool, encoded or transformed values, and unrelated sensitive data may not be recognized. Review artifacts before sharing them. The known Hugging Face boolean control flag `HF_HUB_DISABLE_IMPLICIT_TOKEN` is not treated as a credential value.

Resolved credential-bearing Compose environment entries and recognized inherited credential values are rejected before saving the execution snapshot; execution configuration is never silently redacted into different behavior. Prefer container secret files, including absolute container-path `*_FILE` references. Do not place literal credentials in argv, URLs, labels, Dockerfiles, or arbitrary manifest fields: those are not a supported secret transport. Protect the local state directory and the target repository's own outputs.

## Cleanup boundaries

Cleanup validates pinned source and runtime ownership before deletion. Dirty tracked worktrees, resource identity mismatches, and uncertain cleanup quarantine the lease. `destroy --force` permits discarding tracked edits only after retaining a binary diff; it does not override ambiguous ownership. Untracked build/test output inside managed worktrees is disposable under ordinary cleanup.

`gc` is dry-run by default; `gc --apply` is explicit and excludes quarantined/in-progress leases. Orphan observations never authorize blanket Docker or Git cleanup. Recorded operation locks prevent cooperating agent-env processes from racing lifecycle operations, but do not prevent a user or unrelated process from directly changing Git, Docker, or the filesystem.
