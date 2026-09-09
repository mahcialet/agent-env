---
status: accepted
owner: maintainers
last_verified: 2026-09-09
---

# Bound native execution paths and Windows/WSL interop

[日本語](0007-native-execution-boundaries.ja.md)

## Context

Native Windows diagnostics showed that a 333-character working directory could
not start either a Go child or Git, including with an extended path prefix.
Git's long-path setting also does not apply to its early `-C` handling. General
file API long-path support therefore does not establish process-start support.
The user requested a safe compatibility envelope, including WSL2, instead of
implicitly crossing platform limits or adding persistent execution aliases.

WSL is Linux for process ownership. Linux process-group/identity cleanup does not
prove absence of Windows descendants started through interop. Shared state on
DrvFS/9p also lacks validation against this repository's durability assumptions;
no corruption or actual orphan was reproduced.

## Decision

Windows resolved execution directories are limited to 240 UTF-16 code units,
leaving headroom below MAX_PATH. Check derived source/worktree/runtime/test/probe
locations before reservation/materialization and recheck before process start.
Reject unsupported paths with actionable errors. Keep native Linux/macOS deep
path behavior. Do not require global Windows settings, 8.3 names or junctions.

### Separate Windows and WSL ownership

Use distinct native workers and state roots for Windows and WSL. Reject WSL state
on DrvFS/9p before directory/database creation, using kernel/filesystem/mount
observations and canonical existing ancestors. Windows rejects known WSL UNC
state and execution paths, including extended spellings and resolved aliases.
This policy does not newly reject arbitrary UNC shares or read-only source mounts.

### Reject direct cross-OS execution

Reject directly invoked PE binaries on non-Windows hosts and the `wsl.exe` entry
point on Windows before process start and detached output creation. Check format,
not an `.exe` suffix. Apply the same guard to source-bundle Git commands. Trusted
scripts remain responsible for staying within one OS; this is not a sandbox or
proof about all descendants, wrappers, renamed bridges or concurrently replaced
executables.

## Alternatives and consequences

Relying on OS long-path settings alone did not satisfy native validation. Temporary
junctions can leave invalid Git worktree pointers after removal. Persistent aliases
would require durable ownership, restart reconstruction and cleanup across source
and runtime layers, and would change the no-required-symbolic-links policy.

Users may need a shorter home/repository location. The previous unconditional
Windows deep-path success expectation becomes explicit rejection-before-effects
evidence by user agreement; non-Windows deep-path success remains tested. WSL
mount detection is conservatively tested with injected observations. Actual WSL2
mount/interop acceptance has not been executed and is not implied by those tests.
See [PORTABILITY](../PORTABILITY.md) and the
[ExecPlan](../exec-plans/completed/multi-host-control-plane.md) for evidence.
