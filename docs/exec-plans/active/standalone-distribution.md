---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Package agent-env as a standalone cross-platform distribution

[日本語](standalone-distribution.ja.md)

This ExecPlan is a living document. Maintain it according to `docs/PLANS.md`.

Expected branch: `feat/standalone-distribution`.

PR #4 (`feat: run Flutter Android applications on owned emulator leases`) should
be merged before this plan begins. This work changes packaging, build provenance,
runtime-asset boundaries and repository release infrastructure used by subsequent
features such as `android-ui-observer`; avoid implementing those features on top
of an unstable distribution contract.

At implementation start, update `master`, record the exact revision below, create
the dedicated branch, and run the existing repository harness before changing
code.

Starting revision: `938e584` (`master`, PR #5 merged)

## Purpose / Big Picture

After this work, `agent-env` is distributed as a real standalone application for
native Windows, macOS and Linux.

"Standalone" means:

- running `agent-env` itself requires no Go installation;
- no Python, Node.js, Bash, POSIX shell, PowerShell, Make, or repository checkout
  is required merely to run the CLI;
- agent-env-owned companion assets can be shipped inside the distribution rather
  than downloaded or built by the user at first use;
- installation is unpack/copy-and-run;
- the executable reports its version, source revision, platform and bundled-asset
  provenance;
- all writable agent-env state remains under an OS-native state root or an
  explicit `--home` / `AGENT_ENV_HOME` override;
- release archives are created and validated by the repository's Go harness;
- extracted release artifacts are smoke-tested on native Windows, macOS and
  Linux runners.

Standalone does **not** mean bundling every optional external tool. Capability
providers retain their own prerequisites:

- Git source materialization requires Git;
- Compose stacks require Docker with Compose support;
- Android Emulator stacks require Android SDK/Emulator and host acceleration;
- Flutter application builds require Flutter/Java/Android build tooling.

An API-only user must not be required to install Android or Flutter. Running
`agent-env version`, help, or core diagnostics must not initialize Docker,
Android, Flutter or Java.

The intended release shape is:

    agent-env_vX.Y.Z_windows_amd64.zip
    agent-env_vX.Y.Z_windows_arm64.zip
    agent-env_vX.Y.Z_darwin_amd64.tar.gz
    agent-env_vX.Y.Z_darwin_arm64.tar.gz
    agent-env_vX.Y.Z_linux_amd64.tar.gz
    agent-env_vX.Y.Z_linux_arm64.tar.gz
    checksums.txt
    release-manifest.json

A normal archive contains the executable, MIT license and concise standalone
install/readme text. Future agent-env-owned helper artifacts such as an Android
UI companion APK should normally be embedded into the executable and
materialized on demand into a content-addressed verified location under the
agent-env state/cache root. They must not require a separate manual download.

This work intentionally happens while compatibility burden is still small.
Breaking internal package/build layout changes are acceptable when they simplify
the long-term standalone contract, but public behavior changes must be explicit,
documented and migrated deliberately.

## Progress

- [x] 2026-09-08: `master` at `938e584` includes PR #5; created
      `feat/standalone-distribution`.
- [x] 2026-09-08: Baseline `repoctl check` and `go test -race ./...` started on
      Go 1.27.1 before implementation changes; results are recorded below.
- [x] 2026-09-08: Inspected current CLI bootstrap, `AGENT_ENV_HOME` state-root,
      repoctl, CI and cross-build boundaries.
- [x] 2026-09-08: Added bilingual standalone product/design contracts and
      indexed them; docs-check metadata is synchronized.
- [ ] Decide release version source/tag policy and record it.
- [x] 2026-09-08: Added `internal/buildinfo` and expanded `agent-env version`
      table/JSON output with development-safe version, commit, dirty, Go and
      platform identity; no optional provider is initialized.
- [ ] Define the release target matrix and archive naming/layout.
- [x] 2026-09-08: Added generic content-addressed, digest-verified atomic asset
      materialization with idempotence and traversal/symlink checks in
      `internal/assets`; tests cover reuse and tamper rejection.
- [ ] Add deterministic embedded-asset tests without a target-app dependency.
- [ ] Add/refine an explicit `--home` state-root override if justified.
- [ ] Verify every persistent writable path follows the resolved state-root contract.
- [ ] Implement `repoctl release-build`.
- [ ] Implement `repoctl release-check`.
- [ ] Generate normalized archives, release manifest and checksums.
- [ ] Add exact-version/tag/clean-tree release guards.
- [ ] Add repeated same-source deterministic build/package regression.
- [ ] Add native extracted-artifact smoke tests on Windows/macOS/Linux.
- [ ] Add a GitHub tag/release workflow that delegates mechanics to repoctl.
- [ ] Update architecture/portability/quality/security/roadmap docs in both languages.
- [ ] Verify existing manifests, leases and development workflows are unchanged.
- [ ] Run the full repository harness and final race validation.
- [ ] Record native-platform and release evidence honestly.
- [ ] Complete acceptance evidence and retrospective.
- [ ] Move both ExecPlans to `docs/exec-plans/completed/`.

A checked item means observed completion. Record date, exact command/run,
revision and outcome at each meaningful checkpoint.

## Surprises & Discoveries

- 2026-09-08: The supplied Japanese active plan lacked translation metadata,
  so the baseline docs-check failed before implementation. Added exact
  translation metadata and synchronized the hash; no checker was weakened.
- 2026-09-08: Existing `AGENT_ENV_HOME` already provides absolute OS-native
  state-root precedence. Adding a second `--home` flag would duplicate policy,
  so this slice keeps the established override and records that decision.

Record at least:

- unexpected differences in Go/VCS build metadata across build modes;
- archive metadata that prevents deterministic output;
- Windows executable/path behavior not visible from cross-builds;
- state written outside the resolved home;
- optional-provider initialization that accidentally becomes eager;
- embedded asset concurrency/corruption behavior;
- GitHub Actions limitations that pressure release logic into YAML;
- targets that cross-build but fail native execution;
- antivirus/quarantine behavior affecting extracted artifacts;
- helper assets large enough to materially challenge the one-binary decision;
- third-party license obligations introduced by bundled assets.

Preserve failed approaches when they influence the final design.

## Decision Log

- Decision: Define standalone as "agent-env itself needs no language/runtime or
  manually downloaded helper", not "bundle every optional external tool".
  Rationale: Keeps the distribution small and portable while preserving
  capability-specific Git/Docker/Android/Flutter prerequisites.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Keep the normal runtime distribution as one Go executable plus
  license/readme archive files; future agent-env-owned companion artifacts should
  normally be embedded and materialized on demand.
  Rationale: Simplifies installation and prevents companion version/download drift.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Bundled assets are generic core infrastructure, not Android-specific.
  Rationale: Android UI helpers, browser helpers and future immutable companions
  should use one provenance/materialization model.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Do not bundle Git, Docker, Android SDK, Flutter or Java.
  Rationale: They are large independently managed capabilities with separate
  licensing/update/platform constraints.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Release mechanics live in `repoctl`; CI calls the harness instead of
  duplicating build/archive logic in YAML.
  Rationale: Maintains one cross-platform executable specification and keeps the
  repository as the operational system of record.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Do not embed wall-clock build timestamps in the executable.
  Rationale: Stable version/commit/toolchain identity is more useful and avoids
  needless nondeterminism.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Keep OS-native state defaults and use an explicit home override rather
  than writing beside the executable.
  Rationale: Installed executables may live in read-only locations such as
  Program Files or system package directories.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Initial distribution is GitHub Release archives plus SHA-256
  checksums; package-manager recipes and signing are follow-up work.
  Rationale: Establish the artifact contract before multiplying downstream
  packaging surfaces.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Durable docs and this living ExecPlan are maintained in English and
  Japanese.
  Rationale: Repository documentation policy requires bilingual durable knowledge.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Keep `AGENT_ENV_HOME` as the standalone state override and do not add
  a second `--home` flag in this slice. Rationale: the existing path resolver
  already enforces absolute OS-native roots and lifecycle code consumes it;
  duplicating precedence in CLI would create two state contracts. Date/Author:
  2026-09-08 / maintainers.

## Outcomes & Retrospective

Not completed.

At completion summarize:

- final release target matrix;
- artifact/archive naming and layout;
- version/tag policy;
- binary/build provenance model;
- bundled-asset design;
- state-root behavior;
- deterministic/reproducibility evidence and limitations;
- native smoke-test evidence;
- GitHub Release workflow behavior;
- platform distribution gaps;
- follow-up work for signing/notarization/package managers/SBOM;
- impact on the upcoming `android-ui-observer` plan.

## Context and Orientation

Read before implementation:

- `AGENTS.md` / `AGENTS.ja.md`
- `ARCHITECTURE.md` / `ARCHITECTURE.ja.md`
- `docs/PLANS.md` / `docs/PLANS.ja.md`
- `docs/PORTABILITY.md` / `docs/PORTABILITY.ja.md`
- `docs/QUALITY.md` / `docs/QUALITY.ja.md`
- `docs/SECURITY.md` / `docs/SECURITY.ja.md`
- `docs/RELIABILITY.md` / `docs/RELIABILITY.ja.md`
- `docs/roadmap.md` / `docs/roadmap.ja.md`
- completed MVP / Android Emulator ExecPlans;
- PR #4's final Flutter Android product/design docs and completed plans;
- `.github/workflows/`;
- `tools/repoctl`;
- state/path packages;
- CLI root construction;
- current no-CGO/cross-build tests.

Current invariants already require native Windows/macOS/Linux, no mandatory
CGO, no POSIX shell dependency, argv-based external process invocation,
OS-native state roots, `AGENT_ENV_HOME`, capability-specific prerequisites, and
honest separation of cross-build evidence from native runtime evidence.

The roadmap currently says release packaging is undecided. This plan resolves
that item for archive-based standalone releases and must update the roadmap when
complete.

This plan should complete before `android-ui-observer` implementation if that
feature chooses a companion APK. The UI observer can then consume the generic
bundled-asset facility rather than inventing a separate downloader/extractor.

## Plan of Work

### Milestone 1 — Product contract and current build audit

Inspect current:

- main package/CLI bootstrap;
- `debug.ReadBuildInfo` / linker variables;
- `AGENT_ENV_HOME` / OS-native path resolution;
- repoctl command architecture;
- CGO-disabled/cross-build CI;
- GitHub workflow permissions;
- `.gitignore` and build outputs;
- generated docs/schema checks.

Write:

    docs/product-specs/standalone-distribution.md
    docs/product-specs/standalone-distribution.ja.md
    docs/design-docs/standalone-distribution.md
    docs/design-docs/standalone-distribution.ja.md

Decide:

- release version source;
- tag format (`vX.Y.Z` unless evidence supports another format);
- exact target matrix;
- archive layout/naming;
- whether `--home` is added;
- release Go toolchain pinning policy.

Update indexes and any ADR required by durable architectural choices.

### Milestone 2 — Build info and version command

Create one build-info source of truth.

At minimum report:

- semantic release version or `devel`;
- source commit SHA when known;
- dirty state when knowable;
- Go toolchain version;
- GOOS/GOARCH;
- bundled asset metadata.

Target:

    agent-env version
    agent-env version --output json

Development builds must work without release linker flags. Release builds must
validate exact version/commit. `version` must not initialize Docker, Android,
Flutter or other optional providers.

Do not add a wall-clock build timestamp solely for display.

### Milestone 3 — Generic bundled immutable assets

Add a small generic facility for agent-env-owned immutable runtime assets.

Conceptual metadata:

```go
type AssetInfo struct {
    Name    string
    Version string
    SHA256  string
    Size    int64
}
```

Requirements:

- compile-time/release input, not target-repository content;
- safe internal logical name;
- exact SHA-256 metadata;
- content-addressed materialization path;
- atomic write/rename;
- verify existing content before reuse;
- no symlink or path traversal;
- concurrency-safe and idempotent;
- asset metadata visible through build info without initializing its capability;
- materialize only when a capability actually needs it.

If no production helper exists, prove the mechanism with a tiny test fixture and
do not ship unexplained junk in release archives.

The facility should later support something equivalent to:

    assets.Materialize("android-ui-helper")

without Android-specific code in the asset core.

### Milestone 4 — Explicit state-root contract

Audit every persistent writable path, including:

- SQLite registry;
- worktrees;
- lease evidence/artifacts;
- Android private AVD state;
- bundled-asset materialization;
- command logs;
- atomic-write temporary files.

If `--home` is added, use precedence:

    --home
      > AGENT_ENV_HOME
      > OS-native default

Validate it before external effects. Relative `--home` becomes an absolute path
before persistence. Do not add implicit "write beside executable" portable mode.

Tests should point home to a temporary path and prove persistent agent-env state
does not leak into the target repository or default home.

### Milestone 5 — Cross-platform release builder

Implement release packaging in repoctl, for example:

    go run ./tools/repoctl release-build --version 0.1.0 --out dist

Initial matrix:

- windows/amd64
- windows/arm64
- darwin/amd64
- darwin/arm64
- linux/amd64
- linux/arm64

Use `CGO_ENABLED=0`, `-trimpath` where appropriate, stable version inputs and Go
libraries for zip/tar/gzip/checksum creation. Do not invoke external tar/zip or
shell utilities.

Generate:

- six target archives;
- `checksums.txt`;
- `release-manifest.json`.

The release manifest records version, source commit, expected Go version, target,
archive filename/digest, executable name/digest and bundled asset digests. Do not
record secrets, host paths or temporary paths.

Normalize file order, archive paths, file modes, timestamps and gzip metadata.
The output directory must be explicit and safely replaceable without deleting an
arbitrary caller directory.

### Milestone 6 — Release validator

Implement `repoctl release-check`.

Verify:

- exact expected file set;
- archive names/version/targets;
- checksum correctness;
- release-manifest correctness;
- executable name/mode;
- build info matches target/version/commit where inspectable;
- license/readme presence;
- no absolute/`..` archive paths;
- no symlinks;
- no local source/worktree path leakage;
- expected bundled-asset metadata.

Add negative fixtures for corruption, checksum mismatch and traversal.

### Milestone 7 — Determinism and no-source-tree smoke

Build at least one target twice from identical source/toolchain inputs and compare
binary/archive digests. Investigate differences and record any limit; do not claim
full reproducible builds until directly proven.

Extract release artifacts outside the repository and run:

    agent-env version --output json
    agent-env --help
    agent-env doctor

or a documented narrower core doctor command if generic doctor intentionally
checks optional capabilities.

Smoke tests must not use `go run` or repository files after extraction.

### Milestone 8 — Native distribution CI

Run extracted artifacts on native Windows, macOS and Linux CI. At minimum native
amd64 where hosted runners exist. arm64 cross-build/archive checks remain distinct
from native execution unless arm64 runners are actually used.

Where practical, test explicit state paths containing spaces and non-ASCII text.

### Milestone 9 — GitHub Release workflow

Add a tag-triggered workflow only after repoctl builder/validator are stable.

Flow:

1. checkout exact maintainer-created tag;
2. verify tag/version/source conditions;
3. run repository checks;
4. call repoctl release-build;
5. call repoctl release-check;
6. run/await native smoke jobs;
7. publish only the already-validated files.

Use least-required permissions. CI must not create, move or rewrite Git tags. A
failed validation must not publish a successful release artifact.

### Milestone 10 — Documentation and migration

Update both languages for:

- README installation;
- architecture;
- portability;
- quality;
- security if bundled-asset trust changes;
- roadmap;
- product/design indexes.

Document installation as archive extraction + PATH + `version`/`doctor`. Do not
document package-manager commands until implemented.

Document a capability prerequisite matrix so standalone is not misrepresented as
including Docker, Android or Flutter.

## Concrete Steps

1. Wait for PR #4 to merge.
2. `git switch master && git pull --ff-only`.
3. Record starting revision.
4. Create `feat/standalone-distribution`.
5. Add both active ExecPlans.
6. Run baseline repository harness and race suite.
7. Audit build-info/state/CI/repoctl paths.
8. Create bilingual product/design docs and decisions.
9. Implement build-info core and `agent-env version`.
10. Implement generic bundled-asset support and tests.
11. Finalize explicit state-root override/precedence.
12. Implement `repoctl release-build`.
13. Implement `repoctl release-check`.
14. Add deterministic-build regression.
15. Add native extracted-release smoke jobs.
16. Add tag-triggered GitHub Release workflow.
17. Update bilingual docs and roadmap.
18. Run full harness/race/cross-build/native smoke validation.
19. Inspect final archives manually once and record evidence.
20. Complete acceptance and retrospective in both languages.
21. Move both plans to completed and update links.

## Validation and Acceptance

| ID | Required behavior | Evidence |
| --- | --- | --- |
| S1 | Extracted release runs `version`/help without Go or repository files. | Pending |
| S2 | Core CLI startup does not require Docker, Android, Flutter, Java, Python, Node or shell. | Pending |
| S3 | Capability-specific commands report missing prerequisites lazily and honestly. | Pending |
| S4 | `version --output json` reports documented version/commit/toolchain/platform/asset metadata. | 2026-09-08: `internal/buildinfo.Current` and CLI JSON output report version, commit, dirty marker, Go version and GOOS/GOARCH without optional-provider initialization; asset list integration remains pending. |
| S5 | Development builds have an honest identity without release metadata. | 2026-09-08: development defaults report `devel`/`unknown` identity through `agent-env version`; unit test passes. |
| S6 | Release builds reject mismatched tag/version/commit or dirty release input. | Pending |
| S7 | Release builds use `CGO_ENABLED=0` and no shell packaging tools. | Pending |
| S8 | The fixed matrix produces exactly the documented archive set. | Pending |
| S9 | Checksums and release manifest match exact archive/executable bytes. | Pending |
| S10 | Archives contain only safe relative regular files; no symlink/traversal. | Pending |
| S11 | Bundled asset materialization is content-addressed, digest-verified, atomic, concurrency-safe and idempotent. | 2026-09-08: `internal/assets` tests pass content-addressed reuse and atomic creation; cross-process stress evidence remains pending. |
| S12 | Corrupted materialized asset content is detected and never silently trusted. | 2026-09-08: `TestMaterializeIsContentAddressedAndIdempotent` rejects tampered bytes. |
| S13 | Bundled asset metadata is available without capability initialization. | Pending |
| S14 | Persistent runtime state follows documented state-root precedence and does not leak into target repositories. | Pending |
| S15 | Explicit state root works on native Windows/macOS/Linux paths including spaces; non-ASCII is tested where practical. | Pending |
| S16 | Repeated same-source/toolchain release construction is compared for deterministic binary/archive output and any gap is documented. | Pending |
| S17 | Extracted release artifacts are smoke-tested on native Windows/macOS/Linux. | Pending |
| S18 | Cross-build-only architecture evidence is clearly separated from native execution. | Pending |
| S19 | GitHub release workflow delegates artifact mechanics to repoctl and refuses publication after validation failure. | Pending |
| S20 | Release workflow starts from a maintainer-created tag and never rewrites Git history. | Pending |
| S21 | Existing manifests, leases and development commands continue to work unchanged. | Pending |
| S22 | Full harness, docs/translation checks and Go race suite pass at final revision. | Pending |
| S23 | Standalone product/design/ExecPlan docs exist in English and Japanese and are indexed. | Pending |
| S24 | Roadmap no longer describes archive-based release packaging as undecided after completion. | Pending |
| S25 | The distribution contract is sufficient for `android-ui-observer` to consume a future embedded helper without a separate downloader design. | Pending |

Every acceptance item requires direct evidence. File/workflow existence alone is
not evidence until corresponding validation passes.

## Idempotence and Recovery

Release construction writes only to an explicit development output directory. It
must not mutate Git refs, target repositories, leases or the normal agent-env
state database.

`release-build` validates its output path before replacement, may replace only
its own known output tree, and must leave failed output clearly incomplete rather
than validated.

`release-check` is read-only.

Bundled-asset materialization writes only under the configured agent-env
asset/cache root, verifies existing bytes before reuse, uses per-content paths,
tolerates concurrent materialization, and never follows/creates a
caller-controlled symlink. Corrupted owned content may be quarantined or safely
re-materialized from embedded trusted bytes; record the chosen policy.

Version/release validation never creates, moves, deletes or force-updates Git
tags. CI starts from maintainer-created tags.

Do not add automatic network repair for missing embedded assets. Standalone
assets come from the installed executable.

## Artifacts and Notes

Development release output should be ignored by Git, for example:

    dist/
      agent-env_v0.1.0_windows_amd64.zip
      agent-env_v0.1.0_windows_arm64.zip
      agent-env_v0.1.0_darwin_amd64.tar.gz
      agent-env_v0.1.0_darwin_arm64.tar.gz
      agent-env_v0.1.0_linux_amd64.tar.gz
      agent-env_v0.1.0_linux_arm64.tar.gz
      checksums.txt
      release-manifest.json

Record in plan evidence:

- source commit;
- version/tag;
- Go version;
- repoctl release command;
- archive/executable digests;
- repeated-build comparison;
- native smoke run IDs;
- final GitHub Actions run IDs;
- architectures lacking native execution;
- archive size per target;
- bundled asset count/size.

Do not commit release binaries/archives to the repository.

If a future bundled APK includes third-party code, record license/notice
requirements before release.

## Interfaces and Dependencies

Likely areas:

    internal/buildinfo/
    internal/assets/
    internal/paths/
    internal/cli/
    tools/repoctl/

Exact paths may differ after inspection.

Conceptual data:

```go
type BuildInfo struct {
    Version   string
    Commit    string
    Dirty     bool
    GoVersion string
    GOOS      string
    GOARCH    string
    Assets    []AssetInfo
}
```

Do not collapse release-tool structs, CLI JSON structs and runtime domain structs
into one type if their responsibilities differ.

External development/release dependencies are Go, Git, and GitHub Actions for
hosted orchestration. Runtime external dependencies remain capability-specific.

No Bash, POSIX shell, PowerShell, external tar/zip/checksum utility, CGO,
mandatory daemon or separate helper downloader may be required by standalone
release mechanics.

Unresolved issues to settle during Milestone 1:

1. Exact first public version/tag (`0.1.0` is illustrative).
2. Whether release version comes exclusively from Git tags or needs another
   source; default preference is tag + build metadata without duplicate authority.
3. Whether `debug.ReadBuildInfo` is enough for development identity or release
   linker overrides are required.
4. Flat archive versus one versioned top-level directory.
5. Whether global `--home` is added now or `AGENT_ENV_HOME` is sufficient.
6. Bundled assets under state root versus separate OS-native cache root.
7. Cross-process locking strategy for asset materialization.
8. Direct `go:embed` versus generated metadata around embedded files.
9. Release Go patch-version pin/upgrade policy.
10. Whether bit-for-bit deterministic output is achievable for all six targets.
11. Whether GitHub artifact attestations/SBOM are low-cost enough to include now.
12. Availability of native arm64 smoke infrastructure by OS.

Resolve these in the Decision Log before the corresponding release contract is
considered stable.
