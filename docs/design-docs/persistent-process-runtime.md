---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Persistent process lifecycle design

[日本語](persistent-process-runtime.ja.md)

The [product contract](../product-specs/persistent-process-runtime.md) defines
manifest syntax. The [active ExecPlan](../exec-plans/active/persistent-process-runtime.md)
tracks implementation and validation. This mechanism keeps process management
independent of Compose, Android Emulator, Flutter, and future browser semantics.

## Responsibilities and persistence

Config validates the strict runtime and endpoint variants. Domain owns an additive
`Runtime.process` snapshot with unexpanded command/environment references, pinned
source commit, resolved executable evidence, paths, named ports, and native
PID/start identity. Omitted process fields preserve legacy JSON snapshots.
App coordinates sources, state, reservations, launch intent, readiness, evidence,
operation fences, heartbeat, compensation, and release. The process adapter owns
native effects through `execx`; it does not import other runtime adapters.

Before effects, app materializes the immutable source and fixes runtime directory,
reserved ports, and stdout/stderr paths. It persists launch intent before starting
the detached process, and returned native identity before readiness. A nonzero
identity returned with an error remains cleanup evidence. Never retry an uncertain
launch as if nothing happened. Expanded host credentials remain transient inputs,
not desired-state metadata.

The runtime root is `leases/<id>/process-runtimes/<runtime>/` and contains
`state/`, `stdout.log`, `stderr.log`, `owner.json`, `redaction.json`, and
`launch.json`. Only `state/`
is exposed as `${runtime_dir}`. Source-relative cwd and executable paths are
resolved using native paths and confinement checks after symlink resolution.
PATH tools record host origin, absolute resolution, and readable-file SHA-256;
this evidence does not promise immutable host software or close a replacement race.

## Observation and termination

The detached primitive proves native tree identity beyond PID. A later independent
CLI inspects that identity; a stored ready row alone is not observed health. The
foreground root is the lifecycle anchor. Unexpected exit degrades the lease and
never schedules an automatic restart. A process may continue using a worktree
until the entire owned tree is proven absent.

Termination revalidates exact identity before signaling and before escalation.
Unix groups can become ambiguous after root exit or surviving descendants;
unproven absence quarantines instead of killing a recycled group. Unix birth/group checks immediately precede signals but cannot make native group
signaling atomic with observation. Windows native Job/guardian state provides tree
ownership beyond a numeric PID; termination holds the exact Job handle and forcibly
ends the Job without promising graceful console signaling. Cancellation,
persistence errors, and incomplete termination preserve recovery evidence and
reservations. Only proven whole-tree absence permits mutable-state deletion, port
release, and ordinary tracked-change-protected source cleanup.

## Ports, endpoints, and future consumers

SQLite reservations serialize agent-env's named loopback TCP allocation. Native
availability checks detect external occupancy but cannot eliminate the race until
the target binds; socket activation and inherited listening sockets are excluded.
The process must bind the supplied loopback address/port. App maps `runtime_port`
into the common resolved endpoint representation, so HTTP readiness and Flutter
reverse consumers need no process-specific endpoint syntax after resolution.

Private mutable state is isolated per runtime and retained during quarantine.
Evidence storage retains only intentional diagnostics; browser profiles and local
databases are not automatically promoted. Future browser observation can consume
this process lifetime, state directory, logs, and CDP-like endpoint without adding
browser behavior to the process adapter. Native Windows/macOS/Linux integration,
crash recovery, sibling survival, and uncertain-root regressions are required;
cross-compilation is additional evidence only.

Private `launch.json` retains ownership and native identity. Independent
`redaction.json` is saved before native Start and retains ownership, format version,
and secret length/full-digest/prefix-digest fingerprints. This supports bounded
redaction after host secret variables change or disappear, without plaintext
secret persistence, even when post-launch receipt writing fails. Invalid/missing
redaction proof blocks log export after launch. Raw stdout/stderr still require
private storage; fingerprints are not encryption. Logs retained as cleanup
artifacts pass through the same bounded redaction path.
