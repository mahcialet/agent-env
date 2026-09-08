# agent-env

[日本語](README.ja.md)

Create disposable environment leases from pinned local Git commits and isolated Compose projects using Docker or Podman or private Android Emulators, and foreground native process runtimes. Select a stack, inspect its live state, run named tests with retained evidence, then clean up its resources. Multiple repositories and simultaneous leases are supported.

**Environment isolation is not a malicious-code sandbox.** Dockerfiles, Compose configuration, tests, and package scripts execute repository-controlled code. Use trusted or controlled repositories; arbitrary untrusted pull requests need a stronger outer boundary.

## Standalone archives

Download the archive matching your OS and architecture from GitHub Releases and
extract its versioned directory. Run `agent-env` (`agent-env.exe` on Windows)
directly or add that directory to PATH. `agent-env version --output json` reports
release version, source commit, Go build version and platform. The executable
needs no Go installation or checkout; Go is needed only to build from source.
Release availability and native verification are recorded in the
[release plan](docs/exec-plans/completed/standalone-release-finalization.md).

| Capability | External prerequisites |
| --- | --- |
| Version, help, core diagnostics | None; no Go, shell, Docker, SDK, Flutter or Java |
| Source resolution and managed worktrees | Git and a trusted local repository |
| Docker Compose leases (default) | Git, Docker daemon and Compose v2 plugin |
| Podman Compose leases | Git, Podman 5.x and standalone podman-compose >=1.6.0,<2.0.0; Linux rootless acceptance verified with 5.4.2 / 1.6.0 |
| Persistent process leases | Git and the declared native executable; no container daemon or SDK |
| Android Emulator leases | Git, Android SDK, Emulator, adb, installed system image/AVD template and host acceleration |
| Flutter Android applications | Android prerequisites plus Flutter and a compatible Java/Android build toolchain |
| Android UI observation | Android lease and separately built optional UI companion; SDK/JDK and Go are needed to build that companion |
| Release construction | Git and a supported Go toolchain; release CI pins Go 1.27.1 |

External capability tools and the optional UI companion are not bundled in the
archive. Python is not a core agent-env dependency. Missing optional prerequisites do not prevent version/help from running.
See the [distribution contract](docs/product-specs/standalone-distribution.md).

## Build and verify

Use Go 1.26.x or 1.27.x. Runtime operations need Git plus the selected runtime prerequisites: Docker with Compose v2 or Podman with podman-compose for containers, an installed Android SDK, Emulator, adb and stopped AVD template for Android, or the declared native executable for process runtimes. The repository harness itself needs no Bash, Make, PowerShell, or Docker for ordinary unit checks.

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go build ./cmd/agent-env
go run ./tools/repoctl test-integration
```

The last command explicitly runs real Docker fixtures on Linux. Native Windows/macOS/Linux unit CI and five CGO-disabled build targets are separate from Docker integration coverage. Completed acceptance and verification evidence are recorded in the [implementation plan](docs/exec-plans/completed/agent-env-mvp.md), [quality guide](docs/QUALITY.md), and [portability notes](docs/PORTABILITY.md).

## Use a trusted repository

Declare sources, runtimes, optional applications, components, stacks, and named argv tests in its `.agent-env.yaml`. The [manifest reference](docs/product-specs/manifest-v1.md) has a complete example. For a simple repository with exactly one root Compose file, `init` creates a candidate manifest without overwriting an existing file; review its service selection and host policy before running it.

The following commands run from this checkout; replace the repository path, stack, and named test with your own values:

```text
go run ./cmd/agent-env doctor ../trusted-repo
go run ./cmd/agent-env validate ../trusted-repo
go run ./cmd/agent-env plan ../trusted-repo --stack api
go run ./cmd/agent-env create ../trusted-repo --stack api --ref HEAD
go run ./cmd/agent-env list --output json
go run ./cmd/agent-env show <lease-id>
go run ./cmd/agent-env capabilities <lease-id>
go run ./cmd/agent-env test <lease-id> api-smoke
go run ./cmd/agent-env destroy <lease-id> --dry-run
go run ./cmd/agent-env destroy <lease-id>
```

Alternatively, build the executable and put it on PATH to use `agent-env` (`agent-env.exe` on Windows). `plan` resolves commits without allocating resources. `plan` and `create` accept `--manifest <path>` to select a trusted control manifest explicitly; runtime files still come from each pinned source. Multi-repository ref overrides use `--source alias=ref`.

Declare component endpoints to generate dynamic loopback host publishing in the saved execution configuration without editing source Compose files. Compose resources must be project-scoped and mounts must satisfy host policy. Fixed container names, privileged mode, host networking, Docker socket mounts, and unsafe external binds are rejected. Named tests use argv arrays, with stdout, stderr, exit status, and declared artifacts retained after cleanup. See the [CLI contract](docs/product-specs/cli-contract.md) and [security policy](docs/SECURITY.md).

`gc` previews expired candidates; only `gc --apply` requests deletion. Tracked changes, uncertain ownership, or incomplete cleanup quarantine a lease. Explicit `destroy --force` retains tracked-diff evidence before discarding tracked edits and never overrides an ownership mismatch.

Compose runtimes may set `provider: docker-compose` (the omitted-field default)
or `provider: podman-compose`. Selection is pinned in each lease; missing tools
never trigger fallback. `doctor --provider podman-compose` checks that provider,
while `doctor <repository>` checks the providers declared by its manifest.
Real Linux rootless acceptance passed with Podman 5.4.2 and podman-compose 1.6.0,
including concurrent leases and Docker coexistence. Native Windows/macOS/Linux provider CI passed on 4a5de3d
(run 34216579481); real Podman Machine infrastructure is unavailable. See the [provider contract](docs/product-specs/compose-providers.md).

## Persistent process leases

Declare `type: process` with a pinned `source`, `working_directory` and native argv
`command`. Optional named TCP ports and `${runtime_dir}` support local servers or
private profile state without Compose. Use `doctor --runtime process` for process
diagnostics. The foreground process survives create CLI exit and participates in
readiness, show/logs and conservative destroy; unexpected exit does not restart it.
See the [process contract](docs/product-specs/persistent-process-runtime.md).
Final native integration acceptance is tracked in its active ExecPlan.

## Android Emulator leases

Declare a runtime with `type: android-emulator`, `source: app`, and `avd: <installed-template>`, and reference it from a component without Compose services. Use `doctor --runtime android-emulator` to inspect SDK prerequisites and `doctor <lease-id>` for live state. Android-only stacks do not require Docker. Each lease gets private writable AVD state and a reserved console/ADB port pair; `show` exposes its serial. See the [Android contract](docs/product-specs/android-emulator.md) for a complete manifest and recovery rules.

[Flutter Android applications](docs/product-specs/flutter-android-runtime.md) add pinned-source APK builds, installation, backend reverse mappings, and activity launch on these owned Emulators. Declare optional `applications` and select one from a component. Use `doctor <repository> --runtime flutter-android` to check the configured Flutter executable, project and Android prerequisites; a compatible Flutter/Java/Android build toolchain is required.

## Observe Android UI

The [Android UI observer](docs/product-specs/android-ui-observer.md) captures
accessibility snapshots and PNGs, performs stale-safe semantic taps and Unicode
text replacement, and collects bounded current-PID logcat on owned Emulators.
It works with Flutter semantics and native Android UI without adding target-app
test dependencies. UI observation does not create a runtime or change the manifest.

Build the optional companion once using an installed SDK/JDK, then set
`AGENT_ENV_UI_HELPER` to the generated directory using your host environment settings.
The output directory must not already exist. These example versions must be installed;
the builder does not install tools or accept licenses. Ordinary Go builds and unrelated
commands do not need the companion or a JDK.

```text
go run ./tools/uihelper --sdk <sdk> --jdk <jdk> --platform android-35 --build-tools 36.0.0 --output <new-directory>
go run ./cmd/agent-env ui snapshot <lease-id> --application mobile-app
go run ./cmd/agent-env ui screenshot <lease-id> --application mobile-app
go run ./cmd/agent-env ui tap <lease-id> --snapshot <snapshot-id> --node n7
go run ./cmd/agent-env ui set-text <lease-id> --snapshot <snapshot-id> --node n3 --text <replacement>
go run ./cmd/agent-env ui logcat <lease-id> --application mobile-app --since 30s
```

A semantic reference belongs to one snapshot. A changed or ambiguous target requires
a fresh snapshot; input is never replayed automatically. Text replacement requires
a focused editable node and verified read-back. Editable values are redacted from
retained text evidence, but PNG pixels may contain secrets. An interrupted helper
run can require `ui recover <lease-id> --run <run-id>` before cleanup; recovery retains
an uncertain/failed outcome and never retries input. See the contract for limits,
state restrictions, navigation, waits and recovery.

## State and limits

State lives outside target repositories. Set `AGENT_ENV_HOME` to an absolute path to override the native defaults: Linux XDG state, macOS Application Support, or Windows LOCALAPPDATA. The home contains `state.db`, managed worktrees, normalized runtime configuration, lease artifacts, and a diagnostic `leases/<id>/environment.json` descriptor. SQLite remains authoritative. Defaults are a 4-hour TTL, a 24-hour maximum TTL, and 8 active reservations; quarantined leases retain reservations. A host policy configuration file is not exposed yet.

iOS and browser/CDP automation, remote Git caching, registry promotion, and writable fix leases are [roadmap items](docs/roadmap.md).

Contributors start at [AGENTS.md](AGENTS.md) and the [documentation index](docs/index.md). Licensed under the existing [MIT license](LICENSE).
