---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Manifest v1

[日本語](manifest-v1.ja.md)

Use `.agent-env.yaml` to declare the sources, runtime resources and component
dependencies needed by a lease. This reference defines the implemented version 1
format. Start with the Compose example, then consult the runtime variants below.

The file must be one strict YAML document. Validation rejects unknown fields,
duplicate keys, unsupported versions, empty required collections, invalid names,
dependency cycles and references to missing sources/runtimes/components/stacks.
`agent-env validate <file-or-repository>` checks these rules without running Git
or a Compose provider.

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
| Root | Required `version: 1`, nonempty `sources`, `runtimes`, `components`, and `stacks`; optional `applications`, `browsers` and `tests` |
| `sources.<alias>` | Required local `repository`; optional `default_ref` (Git HEAD when empty), `writable` (only false is supported in review mode) |
| `runtimes.<name>` | Required `type` and `source`; Compose requires nonempty `files` with optional `project_directory` and `provider`; Android requires `type: android-emulator` and `avd`, without Compose fields; process requires `working_directory` and argv `command`, with optional `env` and named TCP `ports` |
| `applications.<name>` | `type: flutter-android`, `source`, Android `runtime`, `build.command`, `build.artifact`, `package`, and `activity`; optional `project_directory`, `build.timeout`, and `reverse`. See the [Flutter contract](flutter-android-runtime.md) |
| `browsers.<name>` | `type: chromium-cdp`, a process `runtime` and its named TCP `cdp_port`; see [explicit browser bindings](#explicit-browser-bindings) |
| `components.<name>` | Required `runtime`; Compose requires nonempty `compose_services`; Android omits Compose services/endpoints/readiness and may select an `application` using the same runtime. Process omits Compose services and uses `runtime_port` endpoints. Optional `depends_on`, `provides` and applicable `readiness` |
| `stacks.<name>` | Required nonempty `roots`; optional `description` |
| `tests.<name>` | Required `stack`, `source`, nonempty argv `command`; optional `working_directory`, string-map `env`, `timeout`, and `artifacts` |

### Names and source repositories

Aliases and names start with a letter or digit and contain only letters, digits, dots, underscores, or hyphens. Source repository paths are local; remote URLs, provider PR shorthand, automatic fetches, and credential management are unsupported.

### Path resolution

Relative source repositories resolve from the control repository. Runtime files and project directories resolve from the runtime's allocated source root; files are not relative to `project_directory`. Test/probe working directories and artifact paths resolve from their declared source root. Portable manifest paths use forward slashes and must not escape that root, including through symlinks. Absolute local source repository paths are allowed on the matching host platform.

### Immutable source and manifest identity

`--source alias=ref` overrides one source's default ref; `--ref` is only valid for a single-source manifest. Every source is resolved to an immutable commit before startup. The sorted alias/repository/commit tuple determines the source-set digest. The canonical manifest and its SHA-256 digest are retained separately from source identity.

### Selected component and service closure

Selected components follow deterministic dependency order. Compose service selection also includes every direct and indirect dependency of the selected services. The executed normalized configuration contains only those services and the networks, volumes, configs, and secrets they reference, preventing unselected global resources from entering cleanup.

### Compose provider selection

Compose runtimes accept optional `provider: docker-compose` or
`provider: podman-compose`. Omission means Docker; explicit empty/null, unknown
values and provider fields on Android/process are errors. Plans and lease snapshots record
the effective provider without adding an omitted field to the canonical manifest.
Old snapshots without provider retain Docker semantics. The selected provider never
falls back to another engine. Podman requires Podman 5.x and standalone
podman-compose >=1.6.0,<2.0.0. Supported configurations, real acceptance results
and the unverified Machine environment are documented in the
[provider contract](compose-providers.md).

## Persistent process variant

A process runtime requires `type: process`, `source`, literal `working_directory`
and argv `command`; `env` and `ports` are optional. Each port requires explicit
`protocol: tcp`; fixed host ports and UDP are rejected. Process-only fields are
invalid on Compose/Android, and `provider`, `project_directory`, `files`, `avd`
are invalid on process even if null/empty. Process names additionally reject
case collisions, Windows device names and trailing dots.

Command arguments/environment values accept `${runtime_dir}`, `${lease_id}`,
`${port:name}`, `${env:NAME}`; unknown references are errors. Executable and cwd
interpolation is forbidden. Process components omit `compose_services` and use
`endpoints.<name>.runtime_port` referring to a declared runtime port; Compose
service/target/protocol fields are forbidden in this variant. Common observed
endpoint maps still contain `host:port`, distinct from numeric readiness references.
See the [complete process contract](persistent-process-runtime.md) for foreground
lifetime, private mutable state, executable evidence and conservative cleanup.

## Explicit browser bindings

Optional `browsers.<name>` declares `type: chromium-cdp`, a `process` runtime and
its named TCP `cdp_port`. Names are case-safe and each runtime has at most one
binding. The repository must declare exact singleton headless, automation,
loopback debugging and `${runtime_dir}/profile` flags in native argv. Empty/null
bindings and alternate/duplicate protected switches fail validation. No browser
is inferred from a port name. See the complete
[browser manifest contract](browser-cdp-automation.md).

## Readiness probes

A component's `readiness` is a list of these probe shapes:

| Type | Required fields | Optional fields |
| --- | --- | --- |
| `compose` | `type` | `timeout`, `interval` |
| `http` | `type`, credential-free HTTP(S) `url` | `timeout`, `interval` |
| `command` | `type`, `source`, argv `command` | `working_directory`, `timeout`, `interval` |

Durations must be positive Go duration strings such as `500ms`, `10s`, or `2m`. Selected Compose probe declarations bound aggregate runtime readiness using the earliest declared timeout and fastest interval together with the built-in 2-minute/1-second defaults. Fields belonging to another probe type are errors. Running containers must have healthy container healthchecks where defined, and every selected service must exist. HTTP probes require a successful response within bounded observation. Command probes execute in the pinned source during create and retain output evidence; ordinary list/show do not rerun repository commands.

Compose HTTP URLs remain literal, explicitly configured URLs. Use a Compose healthcheck for container-local readiness and endpoint observations to discover its host port. Process HTTP readiness additionally accepts `${endpoint:localName}` for a declared endpoint on the same component; it expands to the numeric reserved port, for example `http://127.0.0.1:${endpoint:http}/health`. This is not a whole host:port address. Unknown/local-scope-mismatched references are rejected; process command readiness arguments can use endpoint and process argument references.

## Endpoints

For Compose, `components.<name>.endpoints` maps endpoint names to `service`, integer `target` (1–65535), and optional `protocol` (`tcp`, the default, or `udp`). The service must be selected directly by that component. Before startup, each declaration generates a `127.0.0.1` binding with host port `0` for that service/target/protocol in the normalized execution snapshot; source Compose files are unchanged. The selected engine chooses the actual host port. Remote Podman TCP mappings require observed reachability from the agent-env host; unproven endpoints are withheld. This does not permit fixed host ports rejected by policy in the input configuration. Observation records allocated ports in resource metadata and exposes component-qualified endpoint names through `capabilities`, alongside runtime service/port/protocol observations. Missing or stopped bindings do not provide a live usable endpoint.

## Named command environment and artifacts

### Substitution rules

Commands are argument arrays, never a shell command string. Explicit `${env:NAME}`, `${lease_id}`, and `${android:<runtime>:serial}` substitutions are supported in named-test argv and environment values. Missing host variables, unknown expressions, and malformed substitutions fail before running the command. Android serials resolve only for selected runtimes with confirmed lease ownership. Shell expansion, pipelines, and general template evaluation are not provided.

### Secret handling

Credential-like environment names (token, password, secret, key) must use an exact `${env:NAME}` reference rather than a literal sensitive value. Expanded configured credentials and recognized inherited credentials are redacted from captured output, argv records, and copied artifacts. The complete environment is not written to evidence. A recognized inherited credential appearing literally anywhere in the canonical manifest, including argv or a noncredential environment key, is rejected before reservation. Configure Compose credentials through secret files instead of resolved credential-bearing environment values in the persisted execution snapshot. Absolute container-path `*_FILE` references are supported.

### Artifact paths and retention

Artifact entries are explicit source-relative file or directory paths, not glob patterns. Escaping symlinks and nonregular files are rejected. Captured artifacts survive environment cleanup; the original source outputs can disappear when its managed worktree is removed.

## Multiple repositories and extensions

Add another local entry under `sources` and reference its alias from a runtime, test, or command probe. All declared sources are pinned and materialized, even if only one runtime is selected. The [integration fixture](../../internal/cli/integration_test.go) verifies separate local repositories and alias-specific commit overrides.

[Android Emulator runtimes](android-emulator.md) select a local AVD template with `avd` and allocate private writable state independently of Flutter. Planning does not require the SDK or allocate a device. [Flutter applications](flutter-android-runtime.md) build APKs on the host and install them on those runtimes using optional `applications` and component `application` fields. [Persistent process runtimes](persistent-process-runtime.md) use `type: process` with a pinned source and foreground native command. [Browser/CDP](browser-cdp-automation.md) uses the implemented explicit `browsers` mapping.
Writable fix leases and remote source caches remain [deferred](../roadmap.md);
their proposed fields are not valid manifest v1 YAML.

## Manifest origin and readiness bounds

The control repository selects `.agent-env.yaml`, or an explicit `--manifest` file. Runtime source ref overrides never select another manifest. Plans and leases record its absolute resolved path, control checkout HEAD when available, and modified flag in addition to the canonical snapshot digest. Dirty, untracked or ignored files are marked modified; a non-Git origin has an empty commit and a diagnostic. Relative `sources.*.repository` paths use the supplied control repository, not the explicit manifest file's directory.

Compose readiness probe timeout and interval can tighten the aggregate readiness deadline and polling interval. The application's global readiness timeout remains an upper bound. HTTP and command probes also honor their configured positive durations within the operation's overall deadline. Cancellation stops observation and prevents cleanup under lost ownership.
