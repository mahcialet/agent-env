---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# Compose runtime

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

Merge `compose_services` from the resolved component closure while preserving deterministic order and eliminating duplicates.

Do not assume component name equals Compose service name.

## Readiness

Support at least:

- Compose container state/health inspection;
- HTTP probe declared in the manifest;
- command probe declared as argv array.

Use bounded timeouts and include the failing probe in diagnostics.

## Ports

The MVP should strongly prefer one of:

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
