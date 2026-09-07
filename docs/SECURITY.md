---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# Security and trust

## MVP trust level

The MVP is intended for trusted internal repositories and controlled PRs.

It must not claim safe execution of arbitrary external pull requests.

## Manifest authority

A PR may modify `.agent-env.yaml`. Executing that modified manifest is equivalent to accepting new code-execution instructions.

For the MVP, support an explicit manifest path and record its source commit/digest. Prefer loading the manifest from the control checkout supplied by the user, not silently from an untrusted target ref.

A future trust mode may use:

```text
trusted base-branch manifest
+
limited PR-controlled overlay
```

## Compose checks

Normalize Compose configuration and evaluate host policy before starting. At minimum, detect:

- `privileged: true`;
- `network_mode: host`;
- Docker socket mounts;
- host root or broad host-directory mounts;
- unexpected devices;
- fixed `container_name`;
- static host ports that collide across leases.

## Secret handling

- Do not copy the complete parent environment into evidence.
- Do not serialize secret-bearing values into SQLite metadata.
- Redact configured key names and token patterns in logs.
- Do not inject host credentials into target containers by default.

---

## Host policy

Add a host policy file under the agent-env state/config area. A minimal policy schema may be:

```yaml
version: 1

lease:
  default_ttl: 4h
  max_ttl: 24h
  max_active: 8

compose:
  forbid_privileged: true
  forbid_host_network: true
  forbid_docker_socket: true
  forbid_container_name: true
  allow_absolute_binds: false

filesystem:
  allowed_external_roots: []

runtime:
  max_parallel_creates: 2
```

Repository manifests express requested mechanism. Host policy decides what is permitted on the machine.

The MVP may use built-in defaults and only partially expose policy configuration, but the policy evaluation must not be entangled with YAML parsing or Compose execution.

---
