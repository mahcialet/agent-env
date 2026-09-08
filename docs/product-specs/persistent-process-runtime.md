---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Persistent process runtimes

[日本語](persistent-process-runtime.ja.md)

A `process` runtime owns a foreground native host process for a lease. It uses a
pinned source, direct argv execution, private mutable state, file-backed output,
and durable native process-tree identity. It does not require Compose or a daemon.
The [completed ExecPlan](../exec-plans/completed/persistent-process-runtime.md) tracks
implementation and native acceptance; this document specifies the contract.

## Manifest

```yaml
runtimes:
  api:
    type: process
    source: app
    working_directory: .
    command: [./bin/server, '--listen=127.0.0.1:${port:http}', '--state=${runtime_dir}']
    ports:
      http: {protocol: tcp}
    env:
      API_TOKEN: ${env:API_TOKEN}
components:
  api:
    runtime: api
    endpoints:
      http: {runtime_port: http}
    readiness:
      - type: http
        url: http://127.0.0.1:${endpoint:http}/health
```

`working_directory` is required, source-relative, and confined to the selected
worktree after symlink resolution. Executables are source-relative paths or bare
PATH tool names. No interpolation is allowed in either the executable name or
working directory. Windows `.bat`/`.cmd` wrappers are unsupported. Command arrays
execute directly without an implicit shell; there is no shell command type.

Command arguments and environment values accept only `${runtime_dir}`,
`${lease_id}`, `${port:name}`, and `${env:NAME}`. `runtime_dir` identifies the
runtime's private mutable state directory. Unknown references fail validation;
missing host environment inputs fail before launch. Credential-like environment
keys must contain one exact host environment reference; literal credentials must
not enter durable manifests. Expanded credentials must not enter launch metadata.
Applications must avoid emitting secrets into their private raw stdout/stderr files.

Named ports require explicit `protocol: tcp`. Fixed host ports, UDP, and discovery
by parsing logs are unsupported. Each allocated loopback port is reserved before
launch. Reservations exclude other agent-env allocations, but do not prevent an
arbitrary external process from binding during the allocation-to-launch interval.
The target command must bind loopback using the supplied port; process execution
is not a network or malicious-code sandbox.

Process endpoints use only `runtime_port`, referring to a declared named port.
Compose endpoint fields (`service`, `target`, `protocol`) and `compose_services`
are rejected even when null or empty. Process runtimes reject Compose `provider`,
`project_directory`, `files`, and Android `avd` by field presence. Conversely,
process-only fields are rejected on other runtime types. Runtime names must be
path-safe and cannot collide by case. Existing Compose manifests retain their
canonical representation when the new fields are omitted.

HTTP readiness may use `${endpoint:localName}` for a declared endpoint on the
same component. It expands to the numeric reserved port, so a URL can use
`http://127.0.0.1:${endpoint:http}/health`. It is not a whole host:port address.
Command readiness arguments also accept these endpoint references and the process
argument references above; executable interpolation remains forbidden.

Runtime files reside in `leases/<id>/process-runtimes/<runtime>/`: `state/`,
`stdout.log`, `stderr.log`, ownership marker `owner.json`, private `launch.json`,
and independent prelaunch `redaction.json`. CLI logs are bounded and redacted
using secret fingerprints saved before launch, even after host environment
changes. Missing or invalid redaction proof blocks output after launch. Raw files
can contain application secrets; identity/redaction evidence and raw logs remain
private inputs. Redaction does not depend on the post-launch identity receipt
being written successfully.

## Lifetime and cleanup

Starting the process is not READY: existing HTTP/command readiness must succeed.
Later CLI invocations observe the same recorded native identity, and logs expose
runtime-attributed file output. An unexpected exit degrades the lease; there is
no automatic restart. A foreground lifecycle anchor must remain alive until
termination. Self-daemonization, adoption of external processes, interactive
stdin, PTY, remote execution, and service installation are outside this contract.

Destroy revalidates native identity, requests bounded termination, and confirms
whole-tree absence before releasing ports, mutable state, or source worktrees.
Uncertain ownership retains resources and quarantines the lease. Unix root exit
can leave tree ownership uncertain even when a PID is absent. Windows uses native
Job/guardian ownership; cross-build success alone is not native runtime evidence.
Repeated destroy and GC use the same ownership checks. Mutable state can contain
sensitive profiles or databases; it is not automatically retained as an artifact.

Executable path, source/host origin, and readable-file digest provide launch
evidence. They do not make a mutable PATH tool reproducible. Browser/CDP behavior
belongs in a future consumer above this generic lifecycle. See the
[design](../design-docs/persistent-process-runtime.md).
