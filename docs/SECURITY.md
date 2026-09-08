---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Security and trust

[日本語](SECURITY.ja.md)

Environment leases isolate names, worktrees, and lifecycle ownership. They are **not a malicious-code sandbox**. Repository Dockerfiles, Compose builds, tests, package hooks, and command probes can execute code with the privileges available to their tools. Use trusted repositories or a separately controlled outer sandbox for untrusted code.

## Manifest authority

Planning and creation read the control checkout's `.agent-env.yaml` by default. `--manifest` explicitly selects a trusted manifest path; the canonical snapshot and digest are saved with the lease. Source refs choose pinned runtime/test source content, not a silently substituted manifest from the target revision. Review changes to both the manifest and the code it executes. Trusted base/PR overlay merging and remote credential management are deferred.

Owner labels and `--mine` are advisory filters. Anyone with access to the local state directory and selected container engine has the corresponding host authority. There is no distributed authentication or hostile multi-user isolation.

## Built-in host policy

The current CLI uses built-in defaults: TTL 4 hours, maximum TTL 24 hours, and 8 active reservations. Quarantined and incompletely cleaned leases retain reservations. A configurable host policy file and a separate maximum-parallel-create setting are not implemented.

Before startup, normalized Compose configuration is checked for privileged containers, host networking, fixed container names, fixed published host ports, Docker socket access, device passthrough, and unsafe mounts. Bind paths must stay within allocated source roots, including after symlink resolution. External networks/volumes, globally shared names on selected resources, and unsafe/custom volume drivers or driver options are rejected. These checks reduce accidental host access and collisions; they do not make Docker builds or repository commands trustworthy.

Only selected services and their reachable resource definitions enter the immutable execution snapshot. Ownership labels, a unique project, recorded provider and engine identity, and a configuration digest are retained. Execution and cleanup verify the saved configuration and observed resource identities. Unselected named resources cannot become collateral cleanup targets.

Podman applies the same policy to normalized YAML converted to canonical JSON.
Pod creation and `x-podman*` extensions at any nesting depth are rejected.
The supported mount types are `bind`, `volume` and `tmpfs`; Podman-specific `glob`
and other unmodeled types fail explicitly. Supported `network_mode` values are
omitted/empty, `bridge` and `none`; `host` is denied by common policy, while `ns:`,
`pasta`, `slirp4netns` and other unmodeled modes fail rather than widen host access. Project
`.env` files must not set reserved `PODMAN_*`, `CONTAINER_*`, `AGENT_ENV_PODMAN_*`
or `COMPOSE_*` keys. Inherited routing controls are scrubbed, and the native bridge
pins child calls to the recorded engine. Docker-compatible labels alone do not
prove Podman ownership: native project labels and exact resource identities remain
required. An engine topology fingerprint cannot detect an in-place reset that
recreates identical topology; resource ownership checks are still necessary.

## Credentials and evidence

The process environment is not dumped into SQLite or evidence. Named tests may explicitly reference host values through `${env:NAME}`. Credential-like test environment keys must use an exact host-variable reference rather than a literal value in the manifest. Recognized inherited credential values appearing literally elsewhere in the canonical manifest are also rejected before reservation. The expanded environment remains execution input; recorded argv, streamed logs, and copied artifacts use configured and recognized inherited secret values for redaction.

This is value-based redaction of known credentials, not a universal secret detector. Credentials produced inside a tool, encoded or transformed values, and unrelated sensitive data may not be recognized. Review artifacts before sharing them. The known Hugging Face boolean control flag `HF_HUB_DISABLE_IMPLICIT_TOKEN` is not treated as a credential value.

Resolved credential-bearing Compose environment entries and recognized inherited credential values are rejected before saving the execution snapshot; execution configuration is never silently redacted into different behavior. Prefer container secret files, including absolute container-path `*_FILE` references. Do not place literal credentials in argv, URLs, labels, Dockerfiles, or arbitrary manifest fields: those are not a supported secret transport. Protect the local state directory and the target repository's own outputs.

## Cleanup boundaries

Cleanup validates pinned source and runtime ownership before deletion. Dirty tracked worktrees, resource identity mismatches, and uncertain cleanup quarantine the lease. `destroy --force` permits discarding tracked edits only after retaining a binary diff; it does not override ambiguous ownership. Untracked build/test output inside managed worktrees is disposable under ordinary cleanup.

`gc` is dry-run by default; `gc --apply` is explicit and excludes quarantined/in-progress leases. Orphan observations never authorize blanket Docker, Podman or Git cleanup. Recorded operation locks prevent cooperating agent-env processes from racing lifecycle operations, but do not prevent a user or unrelated process from directly changing Git, container engines, or the filesystem.

## Android host trust

Installed SDK tools, immutable system images and AVD templates are trusted host inputs. Templates contribute hardware configuration; each lease starts with fresh private writable state, without sharing template userdata or snapshots. The adapter rejects unsafe writable paths, symlinks and template locks. Console authentication remains enabled; its token is read locally and never stored in registry metadata. ADB targets the local server explicitly. Direct host modification can invalidate ownership and cause quarantine; these leases do not isolate hostile SDK tools or users.

The server on `127.0.0.1:5037` is shared host state. Before Emulator launch or an
ADB boot query, the adapter directly requests `host:version` and checks it against
the SDK client's protocol version. It rejects mismatches and malformed replies
without invoking a client that could kill the existing server on mismatch.
An absent server may be started separately through the SDK's detached startup
path, with diagnostics retained. Its process identity does not confer lease
ownership: compensation, destroy and GC never stop or replace the global server.
Inherited ADB routing and serial variables are cleared, and boot queries name the
reserved serial explicitly. These guards do not lock out concurrent host changes
or make a hostile local server trustworthy.

The ADB protocol check is an observation of the current shared server, not a host-wide lock. Direct external server replacement or SDK version changes between that observation and an SDK command can still race with the SDK's own version handling. Keep a compatible shared server stable while leases are active; cross-tool server replacement is outside agent-env coordination.

## Release integrity

Git tags identify release versions; checksums detect changed bytes but do not
provide signatures or independent provenance. Signing, notarization and attestation
remain outside the initial archive release. Obtain archives and their manifest
from the trusted release channel. Static validation is not a malicious-binary
sandbox, and native smoke executes the candidate executable.

Release construction rejects changed tracked/index files, non-ignored untracked
files, missing or ambiguous version tags, and mismatched HEAD/version. Ignored
build outputs do not make the tree dirty. Construction stages privately and refuses
an existing output directory. Validation rejects unexpected archive members,
unsafe paths and links, and mismatched executable or metadata digests. Manifest
fields must not contain host paths, temporary paths or credentials. Release commands never mutate caller or public Git refs; preview verification
creates its test tag only in a private clone. Maintainers control publication.
Builds use the exact commit in a private checkout to exclude ignored sources and
local edits hidden by index flags.

Runtime prerequisites remain trusted host tools and are not installed by the
release CLI. The optional Android UI helper is externally built, not embedded in
the current release. Generic asset digest checks detect corruption, not hostile
host modification beyond the existing local trust boundary.

## Persistent host process trust and logs

A process runtime executes the declared native executable with the host user's
privileges. Pinned source, isolated state and reserved loopback ports are ownership
and collision controls, not a sandbox. The command must itself bind loopback;
agent-env cannot prevent arbitrary network or filesystem effects from trusted
repository code. Cwd/source-relative executable confinement follows symlinks, and
PATH resolution/digest records evidence without freezing host tools.

Process environment values use the same explicit `${env:NAME}` secret-reference
rule as named tests. Expanded credentials remain transient launch input; durable
command/environment fields retain references. Native stdout/stderr files are
private raw output and may contain secrets emitted by the application. Keep the
state root, launch receipt and prelaunch redaction evidence private; do not publish them as ordinary artifacts.

CLI log reads are bounded and redact using launch-time secret fingerprints in
independent prelaunch `redaction.json`, including when host variables are later
removed or changed, or writing the post-launch identity receipt fails. Missing,
invalid or mismatched redaction evidence fails closed for an already launched
process. Fingerprints avoid storing plaintext values, but are not encryption and
still require private storage. Unknown application secrets and transformed secret
values are not guaranteed to be recognized. Mutable profiles/databases are not
automatically copied to evidence and are removed only after confirmed tree absence.
See the [process contract](product-specs/persistent-process-runtime.md).
