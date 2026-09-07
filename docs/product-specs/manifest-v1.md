---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# Manifest v1

This is the required contract; executable validation and fixtures must be checked against it as implementation proceeds. Readiness uses bounded container health, HTTP, and argv command probes. Exact endpoint/probe field shapes must be documented alongside their implementation.

## MVP example: API and Dashboard in one Compose runtime

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
    files:
      - compose.yaml
      - compose.agent.yaml

components:
  api:
    runtime: backend
    compose_services:
      - db
      - api
    provides:
      - api
      - logs

  dashboard:
    runtime: backend
    compose_services:
      - dashboard
    depends_on:
      - api
    provides:
      - web-ui
      - browser-e2e

stacks:
  api:
    description: Web API only
    roots:
      - api

  dashboard:
    description: Web API and Dashboard
    roots:
      - dashboard

  full:
    description: Alias for the currently available complete web stack
    roots:
      - dashboard

tests:
  api-smoke:
    stack: api
    source: backend
    working_directory: .
    command:
      - go
      - test
      - ./...

  dashboard-e2e:
    stack: dashboard
    source: backend
    working_directory: dashboard
    command:
      - npm
      - run
      - test:e2e
```

## Target extension example: mobile + API + Dashboard

This illustrates the intended schema direction. Do not require the Android adapter for the MVP.

```yaml
version: 1

sources:
  mobile:
    repository: ../mobile-app
    default_ref: main

  backend:
    repository: ../backend
    default_ref: main

runtimes:
  backend:
    type: compose
    source: backend
    project_directory: .
    files:
      - infra/compose.yaml
      - infra/compose.agent.yaml

  android:
    type: flutter-android
    source: mobile
    project_directory: .
    avd_pool: pixel-api-35
    package: com.example.app
    build:
      command: ["flutter", "build", "apk", "--debug"]
      artifact: build/app/outputs/flutter-apk/app-debug.apk

components:
  api:
    runtime: backend
    compose_services: [db, api]
    provides: [api]

  dashboard:
    runtime: backend
    compose_services: [dashboard]
    depends_on: [api]
    provides: [web-ui, browser-e2e]

  mobile:
    runtime: android
    depends_on: [api]
    provides: [android-ui, mobile-e2e]

stacks:
  api:
    roots: [api]

  dashboard:
    roots: [dashboard]

  mobile:
    roots: [mobile]

  full:
    roots: [dashboard, mobile]
```

## Manifest rules

- Unknown fields should be rejected by default to catch misspellings.
- Include `version` and reject unsupported major schema versions.
- Resolve repository-relative paths relative to the manifest’s control repository, then materialize equivalent paths under the corresponding source worktree.
- Do not allow path traversal outside declared source roots unless host policy explicitly permits it.
- Environment/template interpolation must be small, explicit, and typed. Do not implement general shell expansion.
- Save the exact manifest bytes or canonicalized manifest plus SHA-256 digest for each lease.

---

## Readiness and endpoint fields

Components may declare `readiness` as a list of probes. A probe has `type: compose`, `type: http` with `url`, or `type: command` with `source`, `working_directory`, and argv-array `command`. Optional `timeout` and `interval` are positive Go duration strings. Components may declare `endpoints` as a map of endpoint name to `service`, integer `target` port, and optional `protocol` (`tcp` or `udp`). Runtime execution of these fields must be verified before lifecycle completion. Tests accept argv-array `command`, optional string-map `env`, positive `timeout`, and source-relative `artifacts` paths. Unknown YAML keys and duplicate aliases are errors.
