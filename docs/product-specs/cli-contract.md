---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# CLI contract

Status: required commands; the active plan records availability. Deferred commands must never return fake success.

## Required in the MVP

```text
agent-env version
agent-env init [repository]
agent-env validate [repository]
agent-env plan <repository> --stack <name> [--ref <ref>] [--source alias=ref]
agent-env create <repository> --stack <name> [--ref <ref>] [--source alias=ref]
agent-env list [--cached] [--mine] [--state <state>]
agent-env show <lease-id>
agent-env capabilities <lease-id>
agent-env test <lease-id> <test-name>
agent-env logs <lease-id> [--component <name>] [--run <run-id>]
agent-env renew <lease-id> [--ttl <duration>]
agent-env destroy <lease-id> [--dry-run] [--force]
agent-env reconcile [<lease-id>]
agent-env gc [--apply]
agent-env doctor [repository]
```

## Deferred but reserved

```text
agent-env expand <lease-id> --stack <name>
agent-env fork <lease-id> --mode fix --writable <source>
agent-env checkpoint <lease-id>
agent-env browser <lease-id> ...
agent-env ui <lease-id> ...
agent-env artifact promote <lease-id> <artifact>
agent-env reproduce <lease-id> --artifacts|--rebuild
```

Do not ship placeholder commands that return success while doing nothing. Deferred commands may be omitted or return a clear unsupported-feature error.

---

## Output and failures

Support a global output mode or command-specific option:

```text
--output table
--output json
```

At minimum, `plan`, `create`, `list`, `show`, `doctor`, and `gc` must support JSON.

JSON must be versioned and stable enough for agent skills. Do not print human commentary to stdout in JSON mode; diagnostics go to stderr.

Use nonzero exit status for failures. Distinguish at least:

- invalid user input/manifest;
- missing prerequisite;
- allocation failure;
- test failure;
- cleanup/quarantine result;
- internal error.

Document exit codes before the first public release.

## Destructive operations

`destroy` removes resources in reverse dependency order and records each step.

Conceptual order:

```text
stop active test commands
collect final evidence
stop/down Compose project
inspect Git worktrees for tracked changes
remove clean worktrees
release reservations
mark released
```

If the worktree has unexpected tracked modifications, cleanup fails, or resource identity is ambiguous, quarantine instead of force deleting.

Support:

```text
agent-env destroy <lease-id> --dry-run
agent-env destroy <lease-id>
agent-env destroy <lease-id> --force
```

`--force` must be explicit and must still emit an event and preserve available evidence.

Garbage collection defaults to planning only:

```text
agent-env gc
```

must behave as a dry run. Actual deletion requires:

```text
agent-env gc --apply
```

This is intentional for unattended agent safety.

## Named tests

Named tests are declared in the manifest and invoked through:

```text
agent-env test <lease-id> <test-name>
```

Each run records:

- lease ID;
- test name;
- argv after safe interpolation;
- source/cwd alias;
- start and finish time;
- exit code;
- stdout path;
- stderr path;
- status;
- relevant environment metadata, with secrets redacted.

Stream output to the caller while also writing complete logs to the lease artifact directory.

Do not store secret values in SQLite or logs. Implement redaction hooks and avoid dumping the complete process environment.
