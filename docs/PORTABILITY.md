---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Portability

The module targets Go 1.26.x and 1.27.x, native Windows, macOS, and Linux, with no CGO requirement. Git and the Docker Compose plugin are external runtime prerequisites. Cross-compilation proves build compatibility; it does not prove native process, path, SQLite, or Docker behavior.

## State and paths

| Platform | Default state directory |
| --- | --- |
| Linux | `$XDG_STATE_HOME/agent-env`, otherwise `~/.local/state/agent-env` |
| macOS | `~/Library/Application Support/agent-env` |
| Windows | `%LOCALAPPDATA%/agent-env`, otherwise the user's `AppData/Local/agent-env` |

`AGENT_ENV_HOME` overrides the entire directory and must be absolute. Durable SQLite state and evidence use native filesystem APIs. Managed worktrees use unique lease/source paths outside the target checkout. Tests include spaces, Unicode, Windows drive URIs, traversal, and symlink escapes.

Manifest paths within a source use forward slashes. Native absolute local repository paths are permitted on their matching platform. Windows drive paths on non-Windows hosts and mixed path styles are rejected where detectable. No symbolic links are required for the state layout, and SQLite operation locks replace application-level POSIX lock files.

## Native tools and cancellation

Commands use executable-plus-argv, explicit working directories, deadlines, and streamed output. Git inspection uses machine-readable output; Docker inspection uses JSON and recorded context identity. Newline handling tolerates CRLF. The Go repository harness invokes standard tools directly and requires no shell scripting language.

On Unix, managed command process groups provide cancellation of descendants. On Windows, command children are assigned to a Job Object before they run, with job termination used for cancellation and timeout. This supports bounded named tests and probes; it is not a generic persistent host-process runtime. Background programs deliberately escaping OS containment are outside the trusted-repository model. Failure to verify termination is surfaced as a typed unconfirmed-process-tree result; the app must retain a running registry record and refuse cleanup until reviewed recovery establishes completion.

Windows `.cmd` and `.bat` execution is isolated in the Windows adapter. Wrapper arguments are quoted through the platform path; tokens that cannot be represented safely are rejected instead of silently changing argv. Native executable argv tests cover spaces, quotes, trailing separators, and Unicode. Shell-sensitive wrapper behavior is a separately tested boundary.

## Docker contexts

Windows and macOS normally use Docker Desktop. Linux may use Docker Engine or rootless Docker where Compose works. The chosen Docker context is captured before allocation and used during observation, logs, and cleanup, even if the user's active context later changes. A reachable context alone does not guarantee that its daemon can access local worktree bind paths; remote-daemon path availability is a host prerequisite.

WSL is treated as Linux. Keep repositories, Git, Docker connectivity, and paths consistently on that side of the boundary. Mixed Windows/WSL leases and Windows-host Android Emulator control from WSL are not supported workflows.

## Verification coverage

CI defines native unit/harness jobs for Windows, macOS, and Linux on both supported Go minors. The release-build matrix sets `CGO_ENABLED=0` for windows/amd64, darwin/amd64, darwin/arm64, linux/amd64, and linux/arm64. Linux also runs race tests and explicit real Docker integration.

CI 34124194139 on c641286 passed all six native OS/Go jobs and all five CGO-disabled builds. Linux race and actual Compose integration passed in the same run; local real fixtures also passed, including concurrent projects, Unicode worktrees, multi-repository pins, named evidence, rollback and dirty cleanup. The [completed implementation plan](exec-plans/completed/agent-env-mvp.md) records the full evidence and native regression fixes. Actual Docker integration on Windows/macOS was not run and remains dependent on suitable runners.

## Android persistent processes

Android uses a separate detached-process API with native file-backed output. It survives the invoking CLI and its request context. Linux/macOS retain process-group identity and treat surviving descendants after root exit as uncertain lineage that prevents cleanup; Windows assigns the suspended process to a named Job before resuming it; a private helper retains its handle until the Job is empty and records that evidence. Missing helper evidence or a different logon session prevents cleanup. Birth identities reject reused processes, and uncertain observation prevents writable-state deletion. Persistent launch requires native executables, with no batch wrapper or shell dependency.

SDK discovery uses `ANDROID_HOME`, then `ANDROID_SDK_ROOT` (conflicting values fail), then platform defaults. Templates use `ANDROID_AVD_HOME`, `ANDROID_USER_HOME/avd`, or the user's `.android/avd`. Emulator architecture and usable host acceleration are prerequisites.

A compatible local ADB server on `127.0.0.1:5037` is a shared prerequisite. The adapter compares a direct read-only `host:version` response with the SDK client's protocol version before startup or boot inspection. It refuses incompatible or malformed servers. If absent, the SDK's `adb -L tcp:localhost:5037 start-server` runs through a separate detached launch before the Emulator, with retained startup diagnostics. Bounded boot inspection uses `-H 127.0.0.1 -P 5037 -s <reserved-serial>` and clears inherited server-routing variables; it does not start a missing server. The compatibility probe prevents the normal SDK client's version-mismatch replacement path. Shared ADB is outside lease cleanup.

Native unit CI and actual Emulator integration are distinct. Real SDK integration has been exercised on Linux; actual Windows/macOS SDK startup, acceleration and shared-server lifetime remain unverified. The Android ExecPlan records the tested revisions and remaining platform gaps.
