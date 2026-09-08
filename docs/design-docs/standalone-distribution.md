---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Standalone distribution design

[日本語](standalone-distribution.ja.md)

The [product contract](../product-specs/standalone-distribution.md) defines
the user-visible archive and prerequisite behavior. Build identity is exposed
through `internal/buildinfo`; generic immutable bytes are handled by
`internal/assets`. Release mechanics belong to repoctl so local and CI builds
share one argument-array, no-shell implementation.

The normal state root remains owned by `internal/paths`. Release output is an
explicit caller directory and is never confused with runtime state. Future
embedded companions can describe bytes once, materialize them by digest, and
reuse verified existing content without Android-specific code in the asset core.

## Packaging and inspection

Release construction validates immutable Git identity before build effects,
uses `CGO_ENABLED=0` and trimmed paths, and stages all six targets in private
temporary storage outside the source worktree. After final source validation,
it copies the verified bytes into an owned sibling of the requested new output
directory and renames that complete set into place. This permits non-ignored
worktree output and different temporary/output filesystems without mistaking
construction files for source changes. A complete set includes checksums
and schema-1 manifest derived from actual archive/executable bytes. Git is the
only version authority; build tooling records the actual Go runtime version.
Release CI pins Go 1.27.1. Determinism is qualified to the same source and toolchain.

Go tar/zip/gzip writers normalize member order, relative paths, modes, ownership
and timestamps to the tagged commit. Gzip metadata contains no host identity.
ZIP's UTC extended timestamp retains seconds that its DOS field cannot represent.
Supported commit timestamps run from 1980-01-01 UTC through 2106-02-07
06:28:15 UTC; timestamps outside that common archive range are rejected.
The deterministic bilingual README.txt comes from the harness, not the host or
wall clock. No runtime companion is currently embedded; generic asset metadata
is consequently empty, while the external UI helper keeps its existing boundary.

Static inspection uses Go `debug/buildinfo` to inspect all target executables,
checks VCS identity, CGO and platform settings, and verifies a linker-supplied
`ReleaseRecord` marker also read by runtime build identity. With `-trimpath`, Go
omits linker flags from `debug/buildinfo`; the marker supplies version/commit
values for static comparison. Inspection also checks declared and actual digests
and rejects unsafe or unexpected archive entries. Path-leak coverage is bounded
to controlled metadata and known repository/temporary path prefixes, not a
universal search for every possible host path. It does not execute foreign binaries. Native smoke is a separate
operation on the extracted host tuple with restricted PATH, an external working
directory and isolated state. Workflow orchestration passes the same candidate
bytes through validation and smoke gates to publication without rebuilding.

Build input is a private checkout of the resolved commit, rather than the caller's
worktree. This excludes ignored Go sources and edits hidden by assume-unchanged or
skip-worktree flags. Publication uses private sibling staging only after source checks and refuses
existing destinations. Published non-ignored output counts as ordinary untracked
content in subsequent release commands; ignored or external output keeps the
source tree clean for those commands. The checksum list is ordered by archive filename. Preview verification
uses a private `v0.1.0` tag without modifying caller or public refs; production
repeat verification retains the requested real tag identity. CI passes release
tags through `AGENT_ENV_RELEASE_TAG` and `--tag-env`, not shell interpolation.

Release Git commands disable global/system Git configuration and external
attributes in their child-process environment. The private clone uses an empty
Git template. This prevents ambient smudge/process filters from changing compiler
inputs while clean filters conceal those changes. User Git configuration is never
modified; no release command relies on a globally configured checkout filter.

## Inventory and future embedded consumers

`assets.Inventory()` is the capability-independent production inventory consumed
by `buildinfo.Current()`. It returns an explicit empty list today; the fixture is
embedded only in test binaries. Release native smoke requires the CLI inventory
to match the current empty manifest inventory. Adding a real companion must update
both inventories and their validation in the same change; the release checker
currently rejects nonempty manifests rather than claiming unverified provenance.

A future helper integrates by embedding trusted build-time bytes with `go:embed`,
using `assets.Describe(name, version, bytes)` to compute exact metadata, and calling
`assets.Materialize(resolvedStateRoot, info, bytes)` only when selected. The caller
owns capability identity, license review and expected-version checks; assets owns
only immutable byte verification/publication. The returned regular file has mode
0600 (APK input is not a host executable). No downloader or target-app build is
part of this contract. The existing external UI helper path is unchanged until a
separate feature explicitly replaces it. Tests exercise embedded bytes with no
Android SDK, Flutter, target application or runtime initialization.

Materializers can race to create directories: an EEXIST winner is re-inspected
and accepted only as a nonsymlink directory. Each writer publishes identical
verified bytes from a unique temporary file in the destination directory; another
writer's completed regular file may be reused only after content verification.
Existing corruption is rejected, not silently repaired. The trusted state root
must not have hostile concurrent filesystem mutation; this is not a sandbox.

## Persistent path audit

| Owned data | Location below resolved state root |
| --- | --- |
| Registry, WAL and shared-memory sidecars | `registry.sqlite*` |
| Pinned source worktrees and build outputs | `worktrees/<lease>/<source>/` |
| Environment descriptor, Compose configuration | `leases/<lease>/` |
| Command logs, results, copied artifacts, UI recovery evidence | `leases/<lease>/artifacts/` |
| Android ownership marker, private AVD, emulator/ADB logs and startup identity | `leases/<lease>/android/<runtime>/` |
| Emulator host data and temporary directory | Runtime's private `emulator-data/` and `Temp/` |
| Verified helper install copy | Temporary APK in the owned runtime directory, removed on success/error |
| Immutable bundle cache | `assets/<name>/<sha256>/<name>` |

Atomic evidence/marker writes stage beside their destination. Windows detached
process completion proofs likewise stay beside the owned stdout file. Tests use
real SQLite and native command children with fake lifecycle providers; native
release smoke separately proves core state-root behavior on three operating systems.

Git intentionally registers linked worktrees in the source repository's Git
metadata. Docker resources, shared ADB service/keys and SDK/Flutter/Gradle caches
are external-tool state governed by their existing contracts. Target-defined
commands run in owned worktrees and can have other trusted-code side effects.
Developer release/helper builders use explicit output paths, outside the runtime
state contract. The audit does not claim containment of arbitrary external tools.
