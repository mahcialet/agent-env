---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Finalize standalone release engineering and distribution

[日本語](standalone-release-finalization.ja.md)

This ExecPlan is a living document. Maintain it according to `docs/PLANS.md`.

Parent ExecPlan:

`docs/exec-plans/active/standalone-distribution.md`

This plan completes the remaining release-engineering milestones required by the
parent standalone-distribution plan. It does not supersede or replace the parent.

Expected branch: `feat/standalone-release-finalization`.

This plan is the execution authority for completing the archive-based standalone
release pipeline after the standalone-distribution foundation has landed.

Prerequisite foundation:

- `internal/buildinfo` exists and development `agent-env version` works;
- generic bundled-asset materialization exists with digest verification and
  tamper detection;
- `AGENT_ENV_HOME` remains the sole explicit state-root override for this release
  slice;
- bilingual standalone product/design documentation exists;
- repository docs/unit/vet checks pass at the foundation revision.

Before implementation, update the chosen base, record its exact revision below,
run the repository harness, and verify that the prerequisite foundation is present.

Preferred start: merge the standalone-distribution foundation first, then branch
from `master`. If a stacked PR is intentionally used instead, record the exact
base branch and base commit in this plan and do not misrepresent stacked evidence
as merged-master evidence.

Starting revision: `e37242a312c090c51430d519ea623e1bb41d2941`

Starting base branch: `feat/standalone-distribution`

## Purpose / Big Picture

After this work, maintainers can create one version tag and obtain a fully
validated GitHub Release containing standalone `agent-env` archives for the six
supported target tuples:

    windows/amd64
    windows/arm64
    darwin/amd64
    darwin/arm64
    linux/amd64
    linux/arm64

The release process has one mechanical authority: `repoctl`.

A successful release is accepted only when all of the following are proven:

1. release source identity is exact and immutable;
2. the version/tag contract is satisfied;
3. the working tree is clean;
4. all six archives are produced with normalized layout and metadata;
5. checksums and `release-manifest.json` match the actual bytes;
6. `release-check` rejects malformed, unsafe or mismatched artifacts;
7. same-source/toolchain repeat builds are compared for deterministic output;
8. extracted release binaries run outside the repository without Go;
9. native Windows/macOS/Linux smoke evidence is recorded separately from
   cross-build evidence;
10. GitHub Actions publishes only already validated artifacts from a
    maintainer-created tag;
11. durable English/Japanese documentation describes the delivered contract;
12. all acceptance evidence and retrospective are recorded before the plan is
    archived.

The release pipeline must not require Bash, PowerShell, external `tar`, `zip`,
`sha256sum`, CGO, or a separately installed release tool. Go standard libraries
and existing repository Go code are the packaging implementation.

## Scope

In scope:

- freeze the initial archive-based release contract;
- strict Git tag/version/source validation;
- `repoctl release-build`;
- `repoctl release-check`;
- six-target cross-platform build matrix;
- `CGO_ENABLED=0`;
- normalized zip/tar.gz generation using Go libraries;
- one versioned top-level directory per archive;
- tagged commit timestamp used as normalized archive mtime;
- stable archive order/modes/path handling/gzip metadata;
- `checksums.txt`;
- `release-manifest.json`;
- executable digest and build-info verification;
- bundled-asset metadata verification;
- archive path traversal/symlink rejection;
- source/temp/worktree path leakage checks;
- same-source/toolchain repeated build comparison;
- extracted-artifact smoke tests outside the source tree;
- native Windows/macOS/Linux smoke jobs;
- explicit evidence distinction for arm64 targets without native runners;
- maintainer-created tag triggered GitHub Release workflow;
- validation-before-publication behavior;
- bilingual architecture/portability/quality/security/roadmap updates;
- prerequisite matrix for Git/Docker/Android/Flutter/Java;
- final repository harness/race/cross-build/release validation;
- direct acceptance evidence;
- bilingual Outcomes & Retrospective and ExecPlan archival.

Out of scope:

- Homebrew/Winget/Scoop/Chocolatey/apt/rpm packaging;
- code signing, notarization or certificate acquisition;
- self-update;
- Git tag creation by CI;
- automatic installation of Git/Docker/Android/Flutter/Java;
- OCI image promotion;
- Android UI observer implementation;
- browser/CDP implementation;
- package-manager release metadata;
- malicious-code sandboxing;
- claiming full ecosystem reproducible builds beyond directly proven evidence.

## Release Contract

The following contract is the default decision for this plan. If implementation
evidence requires a change, update the Decision Log before changing code that
depends on it.

### Version authority

The release version authority is the Git tag.

Accepted tag syntax:

    v<MAJOR>.<MINOR>.<PATCH>

`repoctl release-build --version X.Y.Z` must require all of:

- `HEAD` resolves to the exact commit referenced by the release tag;
- the working tree and index are clean;
- the selected release tag matches `v<semver>`;
- requested version `X.Y.Z` equals the selected tag after removing the leading
  `v`;
- source commit identity is available.

If multiple matching version tags point to HEAD, fail unless an explicit,
deterministic tag-selection rule is documented. Prefer failure over guessing.

The plan does not require annotated tags specifically. Lightweight, annotated or
signed tags may be accepted if they resolve to the same exact commit and satisfy
the version contract. Signature verification is outside this slice.

### Target matrix

The release matrix is fixed:

| GOOS | GOARCH | Archive |
| --- | --- | --- |
| windows | amd64 | `.zip` |
| windows | arm64 | `.zip` |
| darwin | amd64 | `.tar.gz` |
| darwin | arm64 | `.tar.gz` |
| linux | amd64 | `.tar.gz` |
| linux | arm64 | `.tar.gz` |

No target may be silently omitted.

### Archive names

For version `X.Y.Z`:

    agent-env_vX.Y.Z_windows_amd64.zip
    agent-env_vX.Y.Z_windows_arm64.zip
    agent-env_vX.Y.Z_darwin_amd64.tar.gz
    agent-env_vX.Y.Z_darwin_arm64.tar.gz
    agent-env_vX.Y.Z_linux_amd64.tar.gz
    agent-env_vX.Y.Z_linux_arm64.tar.gz

### Archive layout

Every archive contains exactly one versioned top-level directory:

    agent-env_vX.Y.Z_<goos>_<goarch>/
      agent-env[.exe]
      LICENSE
      README.txt

No absolute archive paths, `..` segments, symlinks, hard links, device entries or
unexpected files are permitted.

### Normalized timestamps

Archive member mtimes use the tagged commit's commit timestamp. Do not use
wall-clock packaging time.

Zip/tar/gzip metadata that can inject current time, uid/gid, username, hostname or
local paths must be normalized or omitted.

### Release metadata

`checksums.txt` contains SHA-256 digests of the six archives using a documented
stable format.

`release-manifest.json` is a versioned schema and records at minimum:

- schema version;
- product name;
- release version;
- Git tag;
- source commit;
- normalized source timestamp;
- release Go toolchain version;
- each target tuple;
- archive filename and SHA-256;
- executable path within archive and SHA-256;
- executable build-info version/commit/platform identity;
- bundled asset name/version/digest/size metadata.

It must not contain host paths, temporary directories, worktree paths,
credentials or environment secrets.

## Progress

- [x] 2026-09-08: Created `feat/standalone-release-finalization` from `e37242a`; baseline `repoctl check` passed with Go 1.27.1.

- [ ] Record exact starting base branch/revision and verify foundation prerequisites.
- [ ] Run baseline `repoctl check` and documented race suite.
- [ ] Freeze tag/version/archive/release-manifest contract in bilingual durable docs.
- [ ] Implemented strict Git release-source validation in repoctl. Revalidate against acceptance; earlier completion was premature.
- [ ] Implemented `repoctl release-build` for the six target matrix. Revalidate against acceptance; earlier completion was premature.
- [ ] Release builds set `CGO_ENABLED=0` for all six targets. Revalidate against acceptance; earlier completion was premature.
- [ ] Implemented versioned top-level zip/tar.gz archives. Revalidate against acceptance; earlier completion was premature.
- [ ] Archive entries use the tagged commit timestamp. Revalidate against acceptance; earlier completion was premature.
- [ ] Generate deterministic `checksums.txt`.
- [ ] Generate versioned `release-manifest.json`.
- [ ] Implemented initial `repoctl release-check` tag/tree validation. Revalidate against acceptance; earlier completion was premature.
- [ ] Add corruption/mismatch/path-traversal/symlink/path-leak negative fixtures.
- [ ] Verify executable build info and bundled asset metadata statically.
- [ ] Add same-source/toolchain repeated binary/archive digest comparison.
- [ ] Add state-root release-binary tests with spaces and non-ASCII paths.
- [ ] Prove extracted release runs outside repository without Go.
- [ ] Add Windows native smoke job.
- [ ] Add macOS native smoke job.
- [ ] Add Linux native smoke job.
- [ ] Record arm64 native/cross-build evidence honestly per available runners.
- [ ] Add tag-triggered GitHub Release workflow using repoctl mechanics.
- [ ] Prove validation failure prevents publication.
- [ ] Update README/architecture/portability/quality/security/roadmap in English/Japanese.
- [ ] Add capability prerequisite matrix.
- [ ] Run final full harness.
- [ ] Run final race suite.
- [ ] Run final release-build/release-check.
- [ ] Inspect final archives manually and record the inspection.
- [ ] Populate direct acceptance evidence.
- [ ] Complete English/Japanese Outcomes & Retrospective.
- [ ] Move both ExecPlans to `docs/exec-plans/completed/` and update links/hashes.

A checked item means observed completion, not intent. Record UTC date, exact
revision, command/workflow run and outcome.

## Surprises & Discoveries

- 2026-09-08: Audit of `0bf2d12` found premature completion checkboxes:
  Git commands ignored the root argument; release-check inspected no artifacts;
  packaging silently omitted inputs, used README.md, ignored close errors and
  removed an existing output directory. Existing unit tests did not exercise
  release behavior. Completion is reset until direct release tests pass.
- 2026-09-08: Japanese plan hashes had been refreshed without translating the
  updated starting revision and progress. Corrected the actual text, not only hashes.
- 2026-09-08: A Go 1.27.1 probe using `go build -trimpath` showed that
  debug/buildinfo omits linker flags. Runtime and static verification now share
  a ReleaseRecord, with independent VCS/platform/CGO checks.

Preserve discoveries such as multiple version tags, build-info differences,
archive nondeterminism, timestamp precision differences, gzip header behavior,
Windows path/mode behavior, native-runner limitations, path leakage, premature
publication pressure, antivirus/quarantine behavior, or asset metadata mismatch.

Do not mask nondeterminism by weakening digest assertions.

## Decision Log

- Decision: Untracked files count as dirt; ignored outputs do not. Reject more
  than one canonical release tag on HEAD, even when one matches the request.
  Numeric version components have no leading zeros.
  Rationale: Prevent hidden inputs and ambiguous release identity.
  Date/Author: 2026-09-08 / maintainers.
- Decision: ZIP uses exact UTC extended Unix seconds; supported common timestamp
  range is 1980-01-01 through 2106-02-07 06:28:15 UTC. TAR uses normalized USTAR;
  gzip has no timestamp, original filename, or comment. Exactly three regular
  members share an implicit versioned top-level directory.
  Rationale: Preserve commit seconds while rejecting unrepresentable metadata.
  Date/Author: 2026-09-08 / maintainers.
- Decision: README.txt is generated from bilingual repository-harness text.
  LICENSE is read from the release commit; no source or local paths are shipped.
  Rationale: Deterministic install instructions and the existing MIT license.
  Date/Author: 2026-09-08 / maintainers.
- Decision: ReleaseRecord carries CLI identity because trimpath omits linker
  flags; static verification also checks debug/buildinfo. Assets are currently
  an explicit empty array: the optional Android helper is not embedded.
  Rationale: Do not claim shipped assets or identity from unverified manifest data.
  Date/Author: 2026-09-08 / maintainers.
- Decision: Pin Go 1.27.1 in release workflows; local artifacts record the actual
  Go toolchain. Use private-clone v0.1.0 for preview/repeat evidence. Public tags
  remain maintainer-created; publication requires an intentional tag push.
  Rationale: Exercise release guards without creating a public release implicitly.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Git tag is the sole release version authority.
  Rationale: Avoids a duplicate VERSION source and proves release identity from Git.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Release tags use `v<MAJOR>.<MINOR>.<PATCH>` and requested version is
  the same semantic version without `v`.
  Rationale: Conventional and machine-parseable.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Release build requires HEAD == selected tag commit and a clean tree/index.
  Rationale: Prevents claimed-version/source mismatch.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Build all six Windows/macOS/Linux amd64/arm64 targets for every release.
  Rationale: The release contract must not vary implicitly.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Every archive has a versioned target-specific top-level directory.
  Rationale: Extraction does not scatter files and remains self-identifying.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Archive mtimes derive from the tagged commit timestamp.
  Rationale: Avoids wall-clock packaging nondeterminism.
  Date/Author: 2026-09-08 / maintainers.

- Decision: `repoctl` owns release construction/static validation; GitHub Actions
  only orchestrates and publishes.
  Rationale: Local and CI release logic remain identical.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Native smoke evidence and cross-build evidence are separate.
  Rationale: Foreign build success is not native execution proof.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Keep `AGENT_ENV_HOME` as the only explicit state-root override for
  this slice.
  Rationale: Foundation work already selected this contract.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Durable docs and this ExecPlan are maintained in English and Japanese.
  Rationale: Repository bilingual-documentation policy.
  Date/Author: 2026-09-08 / maintainers.

## Outcomes & Retrospective

Not completed.

At completion summarize the first end-to-end version, final tag enforcement,
artifact list, normalization, manifest schema, determinism results, native smoke
matrix, release workflow evidence, failure gates, state-root/asset verification,
documentation updates and follow-up signing/package-manager work.

## Context and Orientation

Read before implementation:

- `AGENTS.md` / `AGENTS.ja.md`
- `ARCHITECTURE.md` / `ARCHITECTURE.ja.md`
- `docs/PLANS.md` / `docs/PLANS.ja.md`
- standalone distribution product/design docs in both languages
- standalone-distribution foundation ExecPlan and Japanese sibling
- `docs/PORTABILITY.md` / `.ja.md`
- `docs/QUALITY.md` / `.ja.md`
- `docs/SECURITY.md` / `.ja.md`
- `docs/roadmap.md` / `.ja.md`
- `internal/buildinfo`
- `internal/assets`
- state/path resolution packages
- CLI version command
- `tools/repoctl`
- existing cross-build/native CI workflows
- `.github/workflows/`

Foundation behavior considered delivered: development build-info/version,
generic bundled-asset verification/materialization, tamper rejection, bilingual
standalone docs foundation and `AGENT_ENV_HOME` state-root choice.

## Plan of Work

### Milestone 1 — Freeze release contract

Update English/Japanese product/design docs with the Release Contract above.
Implement strict version-tag parsing and source-tag selection before archive
construction. Negative cases: no tag, malformed tag, version mismatch, HEAD
mismatch, dirty/staged state, untracked policy violation, ambiguous tags and
unavailable Git identity.

### Milestone 2 — `repoctl release-build`

Implement entirely in Go. Validate release identity first, resolve commit
timestamp, use fixed matrix, set `CGO_ENABLED=0`, use stable build flags, stage
privately, package executable/LICENSE/README, and avoid complete-looking partial
outputs. Use Go archive/zip, archive/tar, compress/gzip, crypto/sha256 and
encoding/json rather than shell tools.

### Milestone 3 — Normalized archives and release metadata

Normalize file order, paths, modes, uid/gid names, mtimes and gzip/zip metadata.
Generate checksums and manifest from actual final bytes. Document unavoidable zip
precision normalization.

### Milestone 4 — `repoctl release-check`

Statically validate file set, names, checksums, manifest, source identity,
archive paths/types, executable digests/build info, asset metadata,
license/readme and path leakage. Add corruption, checksum, manifest, traversal,
symlink, target/version/commit and asset-mismatch negative fixtures.

### Milestone 5 — Determinism and state-root verification

Run release construction twice from identical commit/tag/version/toolchain and
compare executable/archive/checksum/manifest digests separately. Extract native
archives outside repository into spaces/non-ASCII paths, set `AGENT_ENV_HOME` to
a separate such path, and prove state does not leak beside executable, target repo
or default state root.

### Milestone 6 — Extracted artifact smoke tests

Smoke release archives, not `go run`. Run `agent-env version --output json` and
`agent-env --help`; run core doctor if it does not eagerly require optional
providers. Validate version/commit/platform/asset metadata and no source-tree/Go
dependency. Windows/macOS/Linux native; arm64 only native where actually run.

### Milestone 7 — GitHub Release workflow

Trigger on maintainer-pushed `v*` tags but validate exact syntax in repoctl.
Checkout exact tag, install pinned Go, run harness, release-build/check, pass
candidate bytes to native smoke jobs, wait for gates, then publish those exact
validated bytes. Never create/move tags, force-push, rebuild in publish job or
publish before validation.

### Milestone 8 — Documentation completion

Update English/Japanese README, architecture, portability, quality, security,
roadmap and standalone docs. Add a capability prerequisite matrix distinguishing
core, Git, Compose, Android and Flutter requirements. Do not imply those tools are
bundled.

### Milestone 9 — Final evidence and archival

Run final repoctl check, race, release checks, determinism comparison and native
smoke. Manually inspect one Windows zip, one macOS tar.gz and one Linux tar.gz.
Record workflow IDs and native/cross evidence. Complete all acceptance evidence
and bilingual retrospective, then move both plans to completed and update links.

## Concrete Steps

1. Establish/merge standalone foundation prerequisite.
2. Record base branch/revision.
3. Create `feat/standalone-release-finalization`.
4. Add English/Japanese active ExecPlans.
5. Run baseline harness/race.
6. Freeze contract.
7. Implement strict Git validation.
8. Implement release-build.
9. Add six-target builds.
10. Add normalized archives.
11. Add checksum/manifest.
12. Implement release-check.
13. Add negative fixtures.
14. Add determinism tests.
15. Add extracted state-root smoke.
16. Add native smoke jobs.
17. Add GitHub Release workflow.
18. Update bilingual docs/prerequisite matrix.
19. Run final harness/race/release validation.
20. Manually inspect representative archives.
21. Record acceptance/workflow evidence.
22. Fill bilingual retrospective.
23. Move to completed and update links/hashes.

## Validation and Acceptance

| ID | Required behavior | Evidence |
| --- | --- | --- |
| R1 | `release-build` rejects missing/malformed/ambiguous tags before build effects. | Pending |
| R2 | `HEAD == tag commit` is mandatory. | Pending |
| R3 | Documented clean-tree/index policy is enforced. | Pending |
| R4 | Requested `X.Y.Z` equals selected `vX.Y.Z` without `v`. | Pending |
| R5 | All six targets build with `CGO_ENABLED=0`. | Pending |
| R6 | Archive names exactly match the contract. | Pending |
| R7 | Each archive contains one top-level dir with executable/LICENSE/README.txt only. | Pending |
| R8 | Member mtime derives from tagged commit timestamp and metadata is normalized. | Pending |
| R9 | `checksums.txt` matches all six final archives. | Pending |
| R10 | `release-manifest.json` matches actual version/tag/source/toolchain/target/archive/executable/asset bytes. | Pending |
| R11 | `release-check` accepts valid final set without executing foreign binaries. | Pending |
| R12 | `release-check` rejects corruption, mismatch, traversal, symlink, missing/unexpected members. | Pending |
| R13 | No source/worktree/temp absolute path leaks in covered release metadata. | Pending |
| R14 | Same-source/toolchain repeated executable/archive digests are compared and nondeterminism is resolved or qualified. | Pending |
| R15 | Extracted native binary runs version/help outside repository without Go/source tree. | Pending |
| R16 | `AGENT_ENV_HOME` works with spaces/non-ASCII and state does not leak beside executable/target/default home. | Pending |
| R17 | Windows native smoke passes. | Pending |
| R18 | macOS native smoke passes. | Pending |
| R19 | Linux native smoke passes. | Pending |
| R20 | arm64 is labeled native only where an actual native runner executed it. | Pending |
| R21 | Release workflow starts from maintainer-created valid tag and never mutates Git refs/history. | Pending |
| R22 | Workflow delegates artifact mechanics to repoctl. | Pending |
| R23 | Validation/smoke failure prevents successful publication. | Pending |
| R24 | Publication bytes are the exact validated bytes, not rebuilt copies. | Pending |
| R25 | English/Japanese durable docs describe final release contract and capability prerequisites. | Pending |
| R26 | Final repoctl check/docs/translation/race pass. | Pending |
| R27 | Representative Windows/macOS/Linux archives have recorded manual inspection. | Pending |
| R28 | Bilingual Outcomes/Retrospective and direct evidence are complete. | Pending |
| R29 | Both plans are moved to completed with links/hash synchronized. | Pending |

No item is accepted solely because code or a workflow exists.

## Idempotence and Recovery

Release identity validation is read-only. `release-build` writes only to explicit
staging/output and must not mutate Git refs, target repositories, normal lease
state or default state root. Do not delete arbitrary output directories. Failed
builds must not leave checksum/manifest that make a partial set look complete.
`release-check` is read-only.

If repeated builds differ, preserve useful evidence before replacing outputs.
GitHub publication is the final side effect; preferably validate everything before
creating the public release. Never auto-move/recreate the source tag.

## Artifacts and Notes

Expected output:

    dist/
      agent-env_vX.Y.Z_windows_amd64.zip
      agent-env_vX.Y.Z_windows_arm64.zip
      agent-env_vX.Y.Z_darwin_amd64.tar.gz
      agent-env_vX.Y.Z_darwin_arm64.tar.gz
      agent-env_vX.Y.Z_linux_amd64.tar.gz
      agent-env_vX.Y.Z_linux_arm64.tar.gz
      checksums.txt
      release-manifest.json

Record base/start revision, test tag/version, source timestamp, Go toolchain,
archive/executable/manifest digests, repeat-build result, native smoke run IDs,
final release workflow run and native/cross evidence matrix. Do not commit release
archives to Git.

## Interfaces and Dependencies

Expected harness surface:

    repoctl release-build --version X.Y.Z --out <dir>
    repoctl release-check --dir <dir> --version X.Y.Z

Responsibilities: Git release identity helper, build helper, deterministic archive
helper, manifest/checksum helper, static release-check helper, CI orchestration.
Do not collapse release manifest structs into runtime lease/domain types.

External release-construction tools permitted: Go toolchain and Git. Packaging and
checksums require no external archive/checksum utility. GitHub CLI is not a local
release-build dependency.

## Unresolved Issues to Settle Before Milestone 2

1. Exact first release/test version used for end-to-end evidence.
2. Clean-tree policy for untracked files.
3. Multiple version tags on HEAD behavior.
4. Zip timestamp range/precision normalization.
5. Tar/gzip metadata normalization details.
6. README.txt authority: maintained file or deterministic generated source.
7. Static foreign executable build-info inspection strategy.
8. Release Go patch-version authority.
9. Native runner availability for arm64 targets.
10. Whether this plan proves an actual public GitHub Release or stops at validated pre-publication gating until maintainers intentionally cut the first release.

Resolve each in Decision Log before dependent public behavior is stable.
