# agent-env

[日本語](README.ja.md)

agent-env creates disposable development and test environments from pinned Git
commits. Each environment is a **lease**: it owns isolated runtime resources,
retains test evidence, and can be inspected and cleaned up without editing files
in the original checkout. You can use multiple repositories and keep several leases
running at once.

Choose Compose services, private Android Emulators, or foreground native
processes. Flutter applications, Android UI observation, and browser automation
build on those runtimes. Local execution is the default; an optional controller
can place a whole lease on an enrolled worker.

**Environment isolation is not a malicious-code sandbox.** Dockerfiles, Compose
configuration, tests, and package scripts execute repository-controlled code.
Use trusted or controlled repositories. Arbitrary untrusted pull requests need a
stronger outer boundary; see the [security policy](docs/SECURITY.md).

## Standalone archives

Download the archive for your OS and architecture from GitHub Releases and
extract its versioned directory. Run `agent-env` (`agent-env.exe` on Windows)
directly, or add that directory to PATH.

```text
agent-env version --output json
agent-env --help
```

Version output includes the release version, source commit, Go build version,
and platform. The executable needs neither Go nor a repository checkout.
External runtime tools and the optional Android UI companion are not bundled.
Missing optional tools do not prevent version or help from running; Python is
not a core dependency.

The [distribution contract](docs/product-specs/standalone-distribution.md) defines
archive contents and release requirements. Release availability and native
verification are recorded in the [completed release plan](docs/exec-plans/completed/standalone-release-finalization.md).

## Choose a capability

| What you want to do | External prerequisites | Contract and next step |
| --- | --- | --- |
| Resolve sources and create managed worktrees | Git and a trusted local repository | [Manifest reference](docs/product-specs/manifest-v1.md) |
| Run container services | Git, Docker daemon and Compose v2; or Podman 5.x and standalone podman-compose >=1.6.0,<2.0.0 | [Compose providers](docs/product-specs/compose-providers.md) |
| Keep a native server running | Git and the declared native executable; no daemon or SDK | [Persistent processes](docs/product-specs/persistent-process-runtime.md) |
| Inspect or control a browser | A process lease and a directly executable compatible headless Chromium-family browser | [Browser/CDP automation](docs/product-specs/browser-cdp-automation.md) |
| Run an Android Emulator | Git, Android SDK, Emulator, adb, installed system image, stopped AVD template, and host acceleration | [Android Emulator leases](docs/product-specs/android-emulator.md) |
| Build and launch a Flutter Android app | Android prerequisites, Flutter, and a compatible Java/Android build toolchain | [Flutter applications](docs/product-specs/flutter-android-runtime.md) |
| Observe or interact with Android UI | An Android lease and optional UI companion; SDK/JDK and Go to build the companion | [UI setup, commands and recovery](docs/product-specs/android-ui-observer.md) |
| Run a lease on another host | Pre-provisioned TLS certificates; Git for source transfer and worker tools for the selected runtimes | [Controller/client/worker setup](docs/product-specs/multi-host-control-plane.md) |

Docker Compose is the default container provider. Selecting Podman is explicit,
pinned in the lease, and never falls back automatically. Process and Android-only
stacks need no Docker. Browser automation uses the declared process; it does not
launch a separate browser. UI observation uses an existing owned Emulator and
does not create a runtime or change the manifest.

These contracts distinguish implemented behavior, excluded features, and tested
environments. Consult [quality](docs/QUALITY.md) and [portability](docs/PORTABILITY.md)
for exact acceptance evidence, including environments that remain unverified.

## Use a trusted repository

Describe sources, runtimes, optional applications, components, stacks, and named
argv tests in `.agent-env.yaml`. The [manifest reference](docs/product-specs/manifest-v1.md)
has a complete example. If a repository has exactly one root Compose file, `init`
can produce a candidate manifest without overwriting an existing one. Review its
service selection and host policy before running it.

With the executable on PATH, replace the repository path, stack, test, and lease
ID in this sequence:

```text
agent-env doctor ../trusted-repo
agent-env validate ../trusted-repo
agent-env plan ../trusted-repo --stack api
agent-env create ../trusted-repo --stack api --ref HEAD
agent-env list --output json
agent-env show <lease-id>
agent-env capabilities <lease-id>
agent-env test <lease-id> api-smoke
agent-env destroy <lease-id> --dry-run
agent-env destroy <lease-id>
```

`plan` resolves commits without allocating resources. `plan` and `create` accept
`--manifest <path>` for an explicit trusted control manifest; runtime files still
come from pinned sources. Use `--source alias=ref` for multi-repository overrides.
Component endpoints provide dynamic loopback publishing without editing source
Compose files. The [CLI contract](docs/product-specs/cli-contract.md) describes
command behavior and recovery.

Named tests use argv arrays and retain stdout, stderr, exit status, and declared
artifacts after cleanup. Snapshot-scoped UI and browser actions require fresh,
unambiguous targets and never replay input automatically. PNG pixels can contain
secrets even when retained text is redacted. Remote UI/browser `set-text` is
rejected before durable submission; text input is available in local mode.

## State and limits

State lives outside target repositories. `AGENT_ENV_HOME` overrides the native
state root with an absolute path: Linux XDG state, macOS Application Support, or
Windows LOCALAPPDATA. The root holds SQLite `state.db`, managed worktrees,
normalized runtime configuration, artifacts, and the diagnostic
`leases/<id>/environment.json` descriptor. SQLite determines durable state;
the descriptor does not replace it.

Defaults are a 4-hour TTL, a 24-hour maximum TTL, and 8 active reservations.
Quarantined leases keep reservations. A configurable host policy file is not
exposed yet. Compose policy rejects fixed container names, privileged mode, host
networking, Docker socket mounts, and unsafe external binds; resources must be
project-scoped.

`gc` previews expired candidates; only `gc --apply` requests deletion. Tracked
changes, uncertain ownership, or incomplete cleanup quarantine a lease.
`destroy --force` retains tracked-diff evidence before discarding tracked edits
and never overrides an ownership mismatch.

In remote mode, endpoints remain on the worker; there is no client tunnel.
Workers dispatch serially while live leases run concurrently. A second active
operation on one lease, including destroy during a test, is rejected. Local
force/GC cannot bypass controller management. The [remote contract](docs/product-specs/multi-host-control-plane.md)
explains setup, source transfer, enrollment, artifacts, and supported operations.

## Build and verify

Contributors need Go 1.26.x or 1.27.x. The repository harness needs no Bash, Make,
PowerShell, or Docker for ordinary unit checks.

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go build ./cmd/agent-env
go run ./tools/repoctl test-integration
```

The last command explicitly runs real Docker fixtures on Linux. Native OS tests
and CGO-disabled cross-builds are separate coverage. From this checkout you can
also replace `agent-env` in the examples with `go run ./cmd/agent-env`.
Release construction needs Git and Go; release CI pins Go 1.27.1.

Start development with [AGENTS.md](AGENTS.md), find topic-specific guidance in the
[documentation index](docs/index.md), and check [roadmap decisions](docs/roadmap.md)
for deferred work such as iOS, remote Git caching, image promotion, and writable
fix leases. Licensed under the existing [MIT license](LICENSE).
