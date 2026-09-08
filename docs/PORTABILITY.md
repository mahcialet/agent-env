---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Portability

[日本語](PORTABILITY.ja.md)

The module targets Go 1.26.x and 1.27.x, native Windows, macOS, and Linux, with no CGO requirement. Git and the selected Compose provider’s tools are external runtime prerequisites. Cross-compilation proves build compatibility; it does not prove native process, path, SQLite, or Docker behavior.

## State and paths

| Platform | Default state directory |
| --- | --- |
| Linux | `$XDG_STATE_HOME/agent-env`, otherwise `~/.local/state/agent-env` |
| macOS | `~/Library/Application Support/agent-env` |
| Windows | `%LOCALAPPDATA%/agent-env`, otherwise the user's `AppData/Local/agent-env` |

`AGENT_ENV_HOME` overrides the entire directory and must be absolute. Durable SQLite state and evidence use native filesystem APIs. Managed worktrees use unique lease/source paths outside the target checkout. Tests include spaces, Unicode, Windows drive URIs, traversal, and symlink escapes.

Manifest paths within a source use forward slashes. Native absolute local repository paths are permitted on their matching platform. Windows drive paths on non-Windows hosts and mixed path styles are rejected where detectable. No symbolic links are required for the state layout, and SQLite operation locks replace application-level POSIX lock files.

## Native tools and cancellation

Commands use executable-plus-argv, explicit working directories, deadlines, and streamed output. Git inspection uses machine-readable output; Compose engine inspection uses structured output and recorded provider/engine identity. Newline handling tolerates CRLF. The Go repository harness invokes standard tools directly and requires no shell scripting language.

On Unix, managed command process groups provide cancellation of descendants. On Windows, command children are assigned to a Job Object before they run, with job termination used for cancellation and timeout. This supports bounded named tests and probes; it is not a generic persistent host-process runtime. Background programs deliberately escaping OS containment are outside the trusted-repository model. Failure to verify termination is surfaced as a typed unconfirmed-process-tree result; the app must retain a running registry record and refuse cleanup until reviewed recovery establishes completion.

Windows `.cmd` and `.bat` execution is isolated in the Windows adapter. Wrapper arguments are quoted through the platform path; tokens that cannot be represented safely are rejected instead of silently changing argv. Native executable argv tests cover spaces, quotes, trailing separators, and Unicode. Shell-sensitive wrapper behavior is a separately tested boundary.

## Docker contexts

Windows and macOS normally use Docker Desktop. Linux may use Docker Engine or rootless Docker where Compose works. The chosen Docker context is captured before allocation and used during observation, logs, and cleanup, even if the user's active context later changes. A reachable context alone does not guarantee that its daemon can access local worktree bind paths; remote-daemon path availability is a host prerequisite.

WSL is treated as Linux. Keep repositories, Git, Docker connectivity, and paths consistently on that side of the boundary. Mixed Windows/WSL leases and Windows-host Android Emulator control from WSL are not supported workflows.

## Podman prerequisites and evidence limits

The Podman provider requires Podman 5.x and standalone podman-compose
>=1.6.0,<2.0.0. Python belongs to the provider’s host installation, not agent-env
core. The current executable acts as the native child bridge; no generated shell
script is required. Local Linux uses a pinned local engine; remote/Machine calls
retain the resolved endpoint rather than a mutable connection name. Remote bind
paths must be accessible to that engine. Reported remote loopback endpoints require
successful host-side reachability checks.

Real Linux rootless integration passed with Podman 5.4.2 and podman-compose 1.6.0,
including Docker coexistence, dynamic endpoints and anonymous-volume cleanup.
Native Windows/macOS/Linux provider CI passed on 4a5de3d (run 34216579481); no real Podman Machine environment is
available. Cross-builds or fake connection tests do not replace Machine evidence. Exact evidence belongs to the
[provider plan](exec-plans/completed/compose-provider-podman.md).

## Verification coverage

CI defines native unit/harness jobs for Windows, macOS, and Linux on both supported Go minors. The release-build matrix sets `CGO_ENABLED=0` for windows/amd64, darwin/amd64, darwin/arm64, linux/amd64, and linux/arm64. Linux also runs race tests and explicit real Docker integration.

CI 34124194139 on c641286 passed all six native OS/Go jobs and all five CGO-disabled builds. Linux race and actual Compose integration passed in the same run; local real fixtures also passed, including concurrent projects, Unicode worktrees, multi-repository pins, named evidence, rollback and dirty cleanup. The [completed implementation plan](exec-plans/completed/agent-env-mvp.md) records the full evidence and native regression fixes. Actual Docker integration on Windows/macOS was not run and remains dependent on suitable runners.

## Android persistent processes

Android uses a separate detached-process API with native file-backed output. It survives the invoking CLI and its request context. Linux/macOS retain process-group identity and treat surviving descendants after root exit as uncertain lineage that prevents cleanup; Windows assigns the suspended process to a named Job before resuming it; a private helper retains its handle until the Job is empty and records that evidence. Missing helper evidence or a different logon session prevents cleanup. Birth identities reject reused processes, and uncertain observation prevents writable-state deletion. Persistent launch requires native executables, with no batch wrapper or shell dependency.

SDK discovery uses `ANDROID_HOME`, then `ANDROID_SDK_ROOT` (conflicting values fail), then platform defaults. Templates use `ANDROID_AVD_HOME`, `ANDROID_USER_HOME/avd`, or the user's `.android/avd`. Emulator architecture and usable host acceleration are prerequisites.

A compatible local ADB server on `127.0.0.1:5037` is a shared prerequisite. The adapter compares a direct read-only `host:version` response with the SDK client's protocol version before startup or boot inspection. It refuses incompatible or malformed servers. If absent, the SDK's `adb -L tcp:localhost:5037 start-server` runs through a separate detached launch before the Emulator, with retained startup diagnostics. Bounded boot inspection uses `-H 127.0.0.1 -P 5037 -s <reserved-serial>` and clears inherited server-routing variables; it does not start a missing server. The compatibility probe prevents the normal SDK client's version-mismatch replacement path. Shared ADB is outside lease cleanup.

Native unit CI and actual Emulator integration are distinct. Real SDK integration has been exercised on Linux; actual Windows/macOS SDK startup, acceleration and shared-server lifetime remain unverified. The Android ExecPlan records the tested revisions and remaining platform gaps.

## Standalone release targets

The archive contract adds windows/arm64 to the historical five-target build set:
Windows, macOS (`darwin`) and Linux each have amd64 and arm64 archives. Windows
uses ZIP; macOS/Linux use tar.gz. Every target is built with `CGO_ENABLED=0`.
Packaging uses Go libraries and needs no external shell, tar, zip or checksum tool.
Archives have one versioned directory and use the tagged commit timestamp, not
packaging time; ZIP stores a UTC extended timestamp in addition to its coarser DOS
field. State remains under the native state root or absolute `AGENT_ENV_HOME`,
including paths containing spaces and non-ASCII characters.

A successful six-target cross-build does not establish native operation on all six
tuples. Record each actual native smoke runner separately, including arm64; do not
infer execution coverage from a produced archive. The
[release plan](exec-plans/completed/standalone-release-finalization.md) tracks current
evidence. Extracted CLI execution needs no Go; selected external capabilities keep
the prerequisites in the [README](../README.md).
