---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Compose runtime

[日本語](compose-runtime.ja.md)

This document explains the common Compose lifecycle: select services, render a
configuration, start an isolated project, and observe readiness and endpoints.
The [provider design](compose-providers.md) specifies Docker/Podman command
selection, configuration restrictions and cleanup proof. The invocation below
uses Docker to illustrate the common arguments.

## Invocation

Build arguments directly. A representative invocation is:

```text
docker compose
  -p <normalized-project-name>
  --project-directory <absolute-worktree-project-directory>
  -f <absolute-compose-file-1>
  -f <absolute-compose-file-2>
  up -d <selected-services...>
```

Run `docker compose config` before `up`; save its rendered output and hash.

## Service selection

Merge `compose_services` from the selected stack’s root components and all components reached by recursively following their dependencies. Preserve deterministic order and eliminate duplicates.

Do not assume component name equals Compose service name.

## Readiness

Support at least:

- Compose container state/health inspection;
- HTTP probe declared in the manifest;
- command probe declared as argv array.

Use bounded timeouts and include the failing probe in diagnostics.

## Ports

The Compose MVP's port-isolation design strongly prefers one of:

- no host publishing, with tests running inside Compose;
- dynamic host publishing configured in Compose;
- a generated Compose override for declared endpoint target ports.

Do not silently start a second lease when a fixed host port collision is known. Either generate an isolated override or fail before `up` with an actionable diagnostic.

Generated overrides belong under the lease’s generated directory and their content digest must be recorded.

## Inspection

Record:

- Compose project name;
- Docker context;
- selected services;
- container IDs;
- image references and image IDs/digests where available;
- networks and volumes attributable to the project;
- health/status;
- discovered endpoint mappings.

Identify project resources by explicit project name and Compose labels, not by guessed container-name formatting alone.

---
