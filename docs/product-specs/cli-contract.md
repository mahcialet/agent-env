---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# CLI contract

[日本語](cli-contract.ja.md)

The following commands are implemented. Platform validation is recorded separately in the [implementation plan](../exec-plans/completed/agent-env-mvp.md).

```text
agent-env version
agent-env ui snapshot <lease-id> [--application <name>|--runtime <name>] [--all-windows]
agent-env ui screenshot <lease-id> [--application <name>|--runtime <name>]
agent-env ui tap <lease-id> --snapshot <snapshot-id> --node <ref>
agent-env ui set-text <lease-id> --snapshot <snapshot-id> --node <ref> --text <value>
agent-env ui tap-coordinate <lease-id> --runtime <name> --x <x> --y <y>
agent-env ui back <lease-id> [--runtime <name>]
agent-env ui home <lease-id> [--runtime <name>]
agent-env ui swipe <lease-id> --runtime <name> --x <x> --y <y> --to-x <x> --to-y <y> [--duration <duration>]
agent-env ui wait <lease-id> --application <name> --contains <text> [--timeout <duration>]
agent-env ui logcat <lease-id> --application <name> [--since <duration>]
agent-env ui recover <lease-id> --run <run-id>
agent-env init [repository]
agent-env validate [repository-or-manifest]
agent-env plan [repository] --stack <name> [--manifest <path>] [--ref <ref>] [--source alias=ref]
agent-env create [repository] --stack <name> [--manifest <path>] [--ref <ref>] [--source alias=ref] [--ttl <duration>] [--purpose <text>] [--mode review]
agent-env list [--cached] [--mine] [--state <state>]
agent-env show <lease-id>
agent-env capabilities <lease-id>
agent-env test <lease-id> <test-name>
agent-env logs <lease-id> [--component <name>] [--run <run-id>]
agent-env renew <lease-id> [--ttl <duration>]
agent-env destroy <lease-id> [--dry-run] [--force]
agent-env reconcile [lease-id]
agent-env gc [--apply]
agent-env doctor [repository|lease-id] [--runtime compose|process|android-emulator|flutter-android] [--provider docker-compose|podman-compose]
```

An omitted repository means the current directory. `init` requires one recognizable root Compose file, creates `.agent-env.yaml` exclusively, and reports that review is required; it does not start anything. `validate` checks schema and references without Docker or Podman. `plan` additionally resolves local Git commits and the deterministic component closure without creating state or worktrees. `--ref` requires a single source; use alias-specific `--source` overrides for multiple sources. Neither form fetches remote refs.

The manifest defaults to the supplied control checkout. `--manifest` on plan/create selects another trusted manifest explicitly; its digest and canonical snapshot are retained. Review mode creates detached, non-writable-by-contract source worktrees. There is no writable fix mode.

## Identity, state, and observation

`--owner <text>` is a global advisory owner selector. `AGENT_ENV_OWNER` supplies an explicit default. Otherwise a local user/host identity and unique suffix identify allocations; `list --mine` matches that local identity across invocations. Owner labels are filters, not authorization boundaries.

`list` and `show` inspect recorded sources, provider-pinned Compose projects and owned persistent process identities; `--cached` is the explicit registry-only list. `show` includes the lease, events, command runs, artifacts, and endpoint observations. `capabilities` reports capabilities declared by selected components, the observed state, and an endpoint address map. Declared endpoints generate dynamic loopback publishing in the saved runtime configuration. `reconcile <lease-id>` returns that observed lease. Full `reconcile` returns an object containing `leases` and `inventory`, including resources with no matching recorded owner; it never deletes them. Discovery examines installed provider executables and recorded provider/engine identities together, with provider-scoped resource IDs. An installed but unavailable engine returns partial inventory and a visible error.

Desired state is `active` or `released`. Observed state can be `requested`, `allocating`, `starting`, `ready`, `degraded`, `failed`, `releasing`, `released`, `quarantined`, or `unknown`. A stored ready row is not proof of live health. Missing Compose resources or an unexpectedly exited persistent process degrade an active lease; inspection failures remain visible as uncertainty. Expired or quarantined leases cannot become an unqualified ready result.

Runtime/container and bounded HTTP readiness can be re-observed. Arbitrary command probes run during create; list/show do not rerun repository commands. A healthy live observation therefore does not prove that a previously successful command probe would still pass.

`logs --run` reads one recorded command run's stdout/stderr. `logs --component` selects the component's services within a Compose runtime; a process component selects its runtime's file-backed stdout/stderr. Without a filter, logs include available runtime and retained cleanup evidence.

## Output and exit status

`--output table` is the default human view. Some detailed commands render an indented object rather than a table. Every command supports `--output json` using the envelope:

```json
{"schema_version":1,"data":{}}
```

Each lease also has a versioned `leases/<id>/environment.json` diagnostic snapshot refreshed during persisted lifecycle changes; it is not an alternative registry authority.

The data shape depends on the command: plan/create return an object, list returns a lease array, show returns a lease plus evidence collections, and GC returns `apply` plus `leases`. Named-test streaming output goes to stderr in JSON mode so stdout remains a single JSON document. A failed create, cleanup, or named test may still emit its recorded result; inspect both the process exit status and that result.

| Exit | Meaning |
| --- | --- |
| 0 | Successful requested operation |
| 2 | Invalid command, arguments, manifest, or other user input |
| 3 | Missing or unavailable prerequisite |
| 4 | Allocation or readiness failure |
| 5 | Named test failed, timed out, or was canceled |
| 6 | Cleanup could not finish safely, including quarantine |
| 7 | Internal, registry, or observation failure |

These are CLI exit codes; a named command's own exit code is recorded separately in its run record. `doctor` without a repository or explicit provider checks Docker by default. With a repository it checks all manifest-declared Compose providers; `--provider` selects one provider for diagnosis without overriding lease execution. Reports retain provider identity, versions and engine prerequisites. Podman requires Podman 5.x and standalone podman-compose >=1.6.0,<2.0.0; no fallback is attempted. `--provider` applies only to Compose prerequisite diagnosis, not another runtime or a lease ID. `--runtime android-emulator` checks local SDK tools and AVD templates instead of Docker. `--runtime flutter-android` checks the default Flutter executable and Android SDK/AVD prerequisites without Docker; with a repository, it checks configured executables, projects and Android prerequisites for all declared applications. See the [Flutter contract](flutter-android-runtime.md). A lease ID selects live lease diagnostics; a non-ready lease returns exit 3. See the [Android contract](android-emulator.md).

## Cleanup and renewal

Default TTL is 4 hours, maximum TTL 24 hours, and active reservation capacity 8. `renew --ttl` moves expiration relative to now and updates heartbeat; it does not restart missing resources. Quarantined, releasing, and released leases cannot be renewed.

`destroy --dry-run` returns the current cleanup candidate without deleting resources. Ordinary cleanup requests cancellation of exact active named-run IDs and requires confirmed termination and finalized evidence before acquiring the lease lock for removal. An unrelated active operation remains busy. Cleanup then checks source/runtime ownership, preserves per-service final logs, removes runtimes in reverse order, removes managed worktrees, and marks the lease released.

Dirty tracked files quarantine the lease. `destroy --force` explicitly permits their removal after retaining a binary Git diff as an artifact; it still requires matching pinned ownership and successful evidence capture. Untracked test/build output inside a managed review worktree may be removed during normal cleanup. Do not keep unrelated work in managed worktrees.

`gc` previews candidates without mutation. Default policy requires five minutes since expiration and one minute since the last heartbeat. Durable `running` command rows block preview and apply, even if their operation lock expired. `gc --apply` reacquires each lease lock and rechecks these conditions before ownership/source-cleanliness checks and cleanup. Quarantined and in-progress leases are excluded. There is no automatic artifact expiry or general host-resource prune.

## Named tests

A test must be present in the pinned manifest, and its required component stack must be contained in the lease. The lease must be ready and unexpired. Source identities and tracked cleanliness are checked before and after execution; a test that edits tracked files quarantines the review lease.

The command uses an argv array and an allocated source-relative working directory. A durable operation lock excludes simultaneous cleanup, and heartbeat is updated around execution. Each run retains its name, redacted argv, source, working directory, timestamps, exit code, status, stdout/stderr paths, JSON run descriptor, and declared artifacts. Logs stream while being captured. Evidence survives failed tests and request cancellation.

The registry receives a terminal status only after process-tree termination and required evidence finalization are confirmed. An unconfirmed-termination error or failed stream/artifact/descriptor finalization leaves the durable run `running`, so destroy and GC refuse cleanup. Ordinary nonzero exits, timeouts and confirmed cancellations can become terminal results after their evidence is retained. Local recovery descriptors do not authorize a stale lock holder to finalize registry rows.

Environment maps are not copied into run records. `${env:NAME}` explicitly reads a host variable; `${lease_id}` inserts the lease identity. Literal credential-like environment values are rejected before allocation. See [manifest details](manifest-v1.md) and [security limits](../SECURITY.md).

## Android UI observation

`ui` operates on durable owned Android runtime identities, never arbitrary serials.
Snapshots expose Android accessibility semantics, not Flutter widgets. The default
application scope is its package; runtime-only or `--all-windows` snapshots include
system windows. JSON uses the existing envelope; table output renders snapshot-local
node refs and artifact paths. The [observer contract](android-ui-observer.md) defines
selection, state restrictions, stable error codes, bounds and privacy.

Semantic tap/set-text require a registered snapshot and current unique fingerprint.
A stale or ambiguous target returns an error before input, without coordinate fallback.
Text replacement requires a focused editable node and successful read-back equality.
Raw and normalized observations, PNGs, scoped logs and operation evidence remain in
lease artifacts. PNG pixels cannot be redacted like text. UI commands share the lease
fence with cleanup; uncertain remote completion retains the running-command barrier.
`ui recover` only handles a registered interrupted helper operation, verifies its
identity and absence, and records recovery before clearing that run's barrier.
It neither retries input nor declares the interrupted operation successful.
Recovery requires positively persisted `termination-unconfirmed` eligibility and
verified original result evidence; unclassified crashes remain blocked.

UI errors follow the shared exit contract: missing prerequisites return 3; invalid
options, target selection, missing leases and stale/ambiguous references return 2;
registry and observation failures return 7. Typed error categories survive redaction
so secret-safe diagnostics do not collapse these distinctions.

Set `AGENT_ENV_UI_HELPER` to a verified companion build directory for semantic
commands. Build it explicitly with `go run ./tools/uihelper --sdk <sdk> --jdk <jdk>
--platform android-35 --build-tools 36.0.0 --output <new-directory>` using installed
tools; the directory must not already exist. Host environment configuration remains
the user's choice. No target-manifest change or automatic download is required.

## Deferred commands

Expand/shrink, writable forks, checkpoint/reproduce, and artifact promotion are not implemented. They do not return placeholder success; see the [roadmap](../roadmap.md).

## Cancellation and manifest provenance

`destroy` requests cancellation of the exact active named-run IDs and waits up to ten seconds for termination/evidence finalization before cleanup. An unconfirmed cancellation or an ownerless running record prevents cleanup, including with `--force`. An operation already unrelated to those named runs remains busy; destroy does not cancel a newer run that begins during the wait.

Plans and leases expose `manifest_path`, `manifest_commit`, and `manifest_modified`. The path identifies the actual selected file; the commit is its control checkout HEAD, independent of runtime source refs. A modified, untracked or ignored file is marked modified. Outside Git, the commit is empty and diagnostics explain that the canonical snapshot/digest provides provenance. `--manifest` does not change the control repository used to resolve relative source repository paths.

## Persistent process diagnostics and logs

`doctor --runtime process` selects process runtime diagnostics without requiring
Docker, Podman or the Android SDK. `doctor <lease-id>` observes the recorded native
identity; there is no adoption or automatic restart. Process-only lifecycle and
individual lease observation do not require a Compose engine. Full inventory still
reports installed/recorded Compose provider availability separately.

For process runtimes, `show` retains the process snapshot and observed endpoint
addresses. `${endpoint:localName}` in process readiness is a numeric port, while
`capabilities`/endpoint output uses the common `host:port` representation. `logs`
reads attributed stdout/stderr with bounded output and secret redaction based on
independent prelaunch `redaction.json`, not on successful `launch.json` creation. A later host environment change does not remove that redaction proof;
missing or mismatched proof fails closed after launch. Final process logs can be
retained as cleanup artifacts; raw private logs and mutable state are not an
unconditional artifact export. See the [process contract](persistent-process-runtime.md).

## Browser/CDP commands

`browser` provides capabilities, pages, page-create/page-close, navigate, snapshot,
dom-snapshot, screenshot, click, set-text, key, scroll, wait, console and network
operations on an explicit lease browser binding. `--browser` and `--page` select
exact targets; semantic input requires registered `--snapshot` and `--node`.
Commands return run, observation and artifact evidence using the common output
contract. See the [browser contract](browser-cdp-automation.md) for flags, bounds,
privacy, stale-reference checks and uncertain-input cleanup barriers.
