---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# Portability

## General

- Core source must compile on `windows`, `darwin`, and `linux`.
- Do not rely on `/tmp`; use Go temporary and platform directory APIs.
- Do not parse command output affected by localization when a porcelain/JSON form exists.
- Do not assume newline is `\n`; scanners and log files must tolerate CRLF.
- Do not assume executable names end without `.exe`, `.cmd`, or `.bat`.
- Do not use Unix signals as the sole cancellation mechanism.
- Do not use Unix-domain sockets as a required control channel.
- Do not use symlinks as a required state-layout feature.
- Do not assume case-sensitive paths.
- Do not assume Docker Engine is local; respect the active Docker context, but record it in evidence.

## Windows

- Native Windows is supported with Git for Windows and Docker Desktop.
- Use `filepath` and normalized absolute paths; test repositories under paths containing spaces.
- Isolate process execution in `execx` with Windows-specific files guarded by build tags.
- Test `.cmd`/`.bat` invocation because tools such as Flutter or npm may resolve to wrappers.
- Do not mix Windows worktree paths with WSL-side Git or Docker paths in one lease. Detect and reject obviously mixed path styles where possible.
- Do not depend on POSIX file locking. Prefer SQLite transactions and, if a separate host lock is required, implement a platform abstraction.
- If generic managed process runtimes are added later, use Windows Job Objects to manage child process trees rather than attempting to emulate Unix process groups.

## macOS

- Support Intel and Apple Silicon builds where dependencies permit.
- Store durable state under Application Support, not `/tmp`.
- Docker Compose usually reaches Docker Desktop; record the Docker context and daemon information.
- Android Emulator availability and hardware acceleration are host concerns for the later adapter.

## Linux

- Support Docker Engine or compatible Docker CLI/Compose v2 setups.
- Rootless Docker should work where Compose itself works, but do not make rootless mode mandatory.
- Use XDG state/cache variables where present.

## WSL

WSL is not forbidden, but it is treated as Linux. A WSL lease should use WSL-side repositories, Git, Docker connectivity, and paths consistently. Controlling a Windows-host Android Emulator from WSL is deferred and must not be advertised as part of the MVP.

---

## Toolchains and evidence

The module baseline is Go 1.26.0 without an automatic toolchain directive. CI must test supported Go 1.26.x and 1.27.x on Windows, macOS, and Linux. Release builds use CGO_ENABLED=0 for windows/amd64, darwin/amd64, darwin/arm64, linux/amd64, and linux/arm64. Cross-compilation is not proof of native runtime behavior; native CI results must be recorded separately.
