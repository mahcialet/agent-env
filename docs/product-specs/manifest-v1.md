---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Manifest v1

[日本語](manifest-v1.ja.md)

`.agent-env.yaml` is a strict, single-document YAML contract. Unknown fields, duplicate keys, unsupported versions, empty required collections, invalid names, dependency cycles, and references to missing sources/runtimes/components/stacks are rejected. `agent-env validate <file-or-repository>` checks the contract without running Git or a Compose provider.

## Complete example

This example expects `compose.yaml` with `db`, `api`, and `dashboard` services in the trusted local repository. The named test assumes that repository is a Go project. Endpoint declarations request generated dynamic loopback publishing for their container target ports; the source Compose file does not need a `ports` entry.

```yaml
version: 1
sources:
  backend:
    repository: .
    default_ref: HEAD
runtimes:
  backend:
    type: compose
    source: backend
    project_directory: .
    files: [compose.yaml]
components:
  api:
    runtime: backend
    compose_services: [db, api]
    provides: [api, logs]
    readiness:
      - type: compose
        timeout: 2m
        interval: 1s
      - type: command
        source: backend
        working_directory: .
        command: [git, rev-parse, --verify, HEAD]
        timeout: 10s
        interval: 1s
    endpoints:
      http:
        service: api
        target: 8080
        protocol: tcp
  dashboard:
    runtime: backend
    compose_services: [dashboard]
    depends_on: [api]
    provides: [web-ui]
    endpoints:
      web:
        service: dashboard
        target: 8081
stacks:
  api:
    description: API and its database
    roots: [api]
  dashboard:
    description: API plus Dashboard
    roots: [dashboard]
tests:
  api-smoke:
    stack: api
    source: backend
    working_directory: .
    command: [go, test, ./...]
    env:
      TEST_TOKEN: "${env:TEST_TOKEN}"
      LEASE: "${lease_id}"
    timeout: 5m
    artifacts: []
```

Set the explicitly requested host `TEST_TOKEN` variable before invoking this named test, or remove its `env` entry if the repository does not need it. An empty `artifacts` list means stdout, stderr, and the run descriptor are still retained; add repository-relative output paths to collect reports.

## Fields

| Location | Fields and behavior |
| --- | --- |
| Root | Required `version: 1`, nonempty `sources`, `runtimes`, `components`, and `stacks`; optional `applications` and `tests` |
| `sources.<alias>` | Required local `repository`; optional `default_ref` (Git HEAD when empty), `writable` (only false is supported in review mode) |
| `runtimes.<name>` | Required `type` and `source`; Compose requires nonempty `files` with optional `project_directory` and `provider`; Android requires `type: android-emulator` and `avd`, without Compose fields |
| `applications.<name>` | `type: flutter-android`, `source`, Android `runtime`, `build.command`, `build.artifact`, `package`, and `activity`; optional `project_directory`, `build.timeout`, and `reverse`. See the [Flutter contract](flutter-android-runtime.md) |
| `components.<name>` | Required `runtime`; Compose requires nonempty `compose_services`; Android omits Compose services/endpoints/readiness and may select an `application` using the same runtime. Optional `depends_on`, `provides` and applicable `readiness` |
| `stacks.<name>` | Required nonempty `roots`; optional `description` |
| `tests.<name>` | Required `stack`, `source`, nonempty argv `command`; optional `working_directory`, string-map `env`, `timeout`, and `artifacts` |

Aliases and names start with a letter or digit and contain only letters, digits, dots, underscores, or hyphens. Source repository paths are local; remote URLs, provider PR shorthand, automatic fetches, and credential management are unsupported.

Relative source repositories resolve from the control repository. Runtime files and project directories resolve from the runtime's allocated source root; files are not relative to `project_directory`. Test/probe working directories and artifact paths resolve from their declared source root. Portable manifest paths use forward slashes and must not escape that root, including through symlinks. Absolute local source repository paths are allowed on the matching host platform.

`--source alias=ref` overrides one source's default ref; `--ref` is only valid for a single-source manifest. Every source is resolved to an immutable commit before startup. The sorted alias/repository/commit tuple determines the source-set digest. The canonical manifest and its SHA-256 digest are retained separately from source identity.

Selected components follow deterministic dependency order. Compose's service dependency closure is also included. The executed normalized configuration contains only selected services and their reachable networks, volumes, configs, and secrets, preventing unselected global resources from entering cleanup.

Compose runtimes accept optional `provider: docker-compose` or
`provider: podman-compose`. Omission means Docker; explicit empty/null, unknown
values and provider fields on Android are errors. Plans and lease snapshots record
the effective provider without adding an omitted field to the canonical manifest.
Old snapshots without provider retain Docker semantics. The selected provider never
falls back to another engine. Podman requires Podman 5.x and standalone
podman-compose >=1.6.0,<2.0.0. Real Linux rootless acceptance passed with
5.4.2 / 1.6.0; final native CI remains pending and real Machine infrastructure is
unavailable. See the
[provider contract](compose-providers.md).

## Readiness probes

A component's `readiness` is a list of these probe shapes:

| Type | Required fields | Optional fields |
| --- | --- | --- |
| `compose` | `type` | `timeout`, `interval` |
| `http` | `type`, credential-free HTTP(S) `url` | `timeout`, `interval` |
| `command` | `type`, `source`, argv `command` | `working_directory`, `timeout`, `interval` |

Durations must be positive Go duration strings such as `500ms`, `10s`, or `2m`. Selected Compose probe declarations bound aggregate runtime readiness using the earliest declared timeout and fastest interval together with the built-in 2-minute/1-second defaults. Fields belonging to another probe type are errors. Running containers must have healthy container healthchecks where defined, and every selected service must exist. HTTP probes require a successful response within bounded observation. Command probes execute in the pinned source during create and retain output evidence; ordinary list/show do not rerun repository commands.

HTTP URLs are literal, explicitly configured URLs. There is no automatic substitution of a dynamic endpoint into a probe URL. Use a Compose healthcheck for container-local HTTP readiness and use endpoint observations to discover the allocated host port.

## Endpoints

`components.<name>.endpoints` maps endpoint names to `service`, integer `target` (1–65535), and optional `protocol` (`tcp`, the default, or `udp`). The service must be selected directly by that component. Before startup, each declaration generates a `127.0.0.1` binding with host port `0` for that service/target/protocol in the normalized execution snapshot; source Compose files are unchanged. The selected engine chooses the actual host port. Remote Podman mappings require observed reachability from the agent-env host; unproven endpoints are withheld. This does not permit fixed host ports rejected by policy in the input configuration. Observation records allocated ports in resource metadata and exposes component-qualified endpoint names through `capabilities`, alongside runtime service/port/protocol observations. Missing or stopped bindings do not provide a live usable endpoint.

## Named command environment and artifacts

Commands are argument arrays, never a shell command string. Explicit `${env:NAME}`, `${lease_id}`, and `${android:<runtime>:serial}` substitutions are supported in named-test argv and environment values. Missing host variables, unknown expressions, and malformed substitutions fail before running the command. Android serials resolve only for selected runtimes with confirmed lease ownership. Shell expansion, pipelines, and general template evaluation are not provided.

Credential-like environment names (token, password, secret, key) must use an exact `${env:NAME}` reference rather than a literal sensitive value. Expanded configured credentials and recognized inherited credentials are redacted from captured output, argv records, and copied artifacts. The complete environment is not written to evidence. A recognized inherited credential appearing literally anywhere in the canonical manifest, including argv or a noncredential environment key, is rejected before reservation. Configure Compose credentials through secret files instead of resolved credential-bearing environment values in the persisted execution snapshot. Absolute container-path `*_FILE` references are supported.

Artifact entries are explicit source-relative file or directory paths, not glob patterns. Escaping symlinks and nonregular files are rejected. Captured artifacts survive environment cleanup; the original source outputs can disappear when its managed worktree is removed.

## Multiple repositories and extensions

Add another local entry under `sources` and reference its alias from a runtime, test, or command probe. All declared sources are pinned and materialized, even if only one runtime is selected. The [integration fixture](../../internal/cli/integration_test.go) verifies separate local repositories and alias-specific commit overrides.

[Android Emulator runtimes](android-emulator.md) select a local AVD template with `avd` and allocate private writable state independently of Flutter. Planning does not require the SDK or allocate a device. [Flutter applications](flutter-android-runtime.md) build APKs on the host and install them on those runtimes using optional `applications` and component `application` fields. Browser/CDP, writable fix leases, remote source caches, and arbitrary host-process runtimes remain [deferred](../roadmap.md); their proposed fields are not valid manifest v1 YAML.

## Manifest origin and readiness bounds

The control repository selects `.agent-env.yaml`, or an explicit `--manifest` file. Runtime source ref overrides never select another manifest. Plans and leases record its absolute resolved path, control checkout HEAD when available, and modified flag in addition to the canonical snapshot digest. Dirty, untracked or ignored files are marked modified; a non-Git origin has an empty commit and a diagnostic. Relative `sources.*.repository` paths use the supplied control repository, not the explicit manifest file's directory.

Compose readiness probe timeout and interval can tighten the aggregate readiness deadline and polling interval. The application's global readiness timeout remains an upper bound. HTTP and command probes also honor their configured positive durations within the operation's overall deadline. Cancellation stops observation and prevents cleanup under lost ownership.
