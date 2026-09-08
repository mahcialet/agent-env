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
- all writable agent-env state remains under an OS-native state root or the
  existing `AGENT_ENV_HOME` override;
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

## Child ExecPlans

[standalone-release-finalization.md](../completed/standalone-release-finalization.md) is a completed child execution plan of this plan.

It owns the remaining concrete release-engineering work:

- strict Git tag release validation
- `repoctl release-build`
- `repoctl release-check`
- six-target archive generation
- deterministic/reproducibility validation
- native smoke tests
- GitHub Release workflow
- final release documentation and evidence

Completion of the child plan is necessary but not by itself sufficient to
complete this parent plan. After the child is archived, this plan must reconcile
the delivered behavior and evidence against its own acceptance criteria and fill
its Outcomes & Retrospective.

## Progress

- [x] 2026-09-08: Resumed parent at PR #7 merge `16afc83`; baseline `go run ./tools/repoctl check` passed. Added explicit CLI bundled inventory and state-free JSON/table regression; remaining audits and final validation continue.

- [x] 2026-09-08: `master` at `938e584` includes PR #5; created
      `feat/standalone-distribution`.
- [x] 2026-09-08: Baseline `go test -race ./...` passed on Go 1.27.1 before
      implementation changes. Baseline `repoctl check` completed unit/vet but
      failed docs-check because the supplied Japanese plan lacked metadata;
      metadata was repaired before subsequent checks.
- [x] 2026-09-08: Inspected current CLI bootstrap, `AGENT_ENV_HOME` state-root,
      repoctl, CI and cross-build boundaries.
- [x] 2026-09-08: Added bilingual standalone product/design contracts and
      indexed them; docs-check metadata is synchronized.
- [x] 2026-09-08: Git tag is the sole release-version authority; enforce exact `v<semver>`, clean tree, `HEAD == tag commit`, and requested-version equality after removing `v`.
- [x] 2026-09-08: Added `internal/buildinfo` and expanded `agent-env version`
      table/JSON output with development-safe version, commit, dirty, Go and
      platform identity; no optional provider is initialized.
- [x] 2026-09-08: Fixed six-target matrix and top-level archive directory layout; archive mtimes use the tagged commit timestamp.
- [x] 2026-09-08: Added generic content-addressed, digest-verified atomic asset
      materialization with idempotence and traversal/symlink checks in
      `internal/assets`; tests cover reuse and tamper rejection.
- [x] 2026-09-08: Completed child `docs/exec-plans/completed/standalone-release-finalization.md`; release-build/check, deterministic packaging, native smoke and GitHub Release workflow delivered and validated at `641cb49`.
- [x] 2026-09-08: Completed capability prerequisite tests, CLI inventory, asset stress, persistent-path audit and future helper contract; local harness/race passed. Native final-revision CI is recorded below.
- [x] 2026-09-08: TestEmbeddedFixture verifies 17 embedded bytes and fixed SHA-256 without a target app; asset race stress passed ten times.
- [x] 2026-09-08: Retain AGENT_ENV_HOME as the sole explicit override; an additional --home flag is not justified (existing Decision Log).
- [x] 2026-09-08: Audited persistent paths in the design document; state override, lifecycle, command evidence and helper staging regression tests pass.
- [x] 2026-09-08: Implement `repoctl release-build`. Child evidence: `641cb49`, preview 34190701402, Verify 34190701428.
- [x] 2026-09-08: Implement `repoctl release-check`. Child evidence: `641cb49`, preview 34190701402, Verify 34190701428.
- [x] 2026-09-08: Generate normalized archives, release manifest and checksums. Child evidence: `641cb49`, preview 34190701402, Verify 34190701428.
- [x] 2026-09-08: Add exact-version/tag/clean-tree release guards. Child evidence: `641cb49`, preview 34190701402, Verify 34190701428.
- [x] 2026-09-08: Add repeated same-source deterministic build/package regression. Child evidence: `641cb49`, preview 34190701402, Verify 34190701428.
- [x] 2026-09-08: Add native extracted-artifact smoke tests on Windows/macOS/Linux. Child evidence: `641cb49`, preview 34190701402, Verify 34190701428.
- [x] 2026-09-08: Add a GitHub tag/release workflow that delegates mechanics to repoctl. Child evidence: `641cb49`, preview 34190701402, Verify 34190701428.
- [x] 2026-09-08: Update architecture/portability/quality/security/roadmap docs in both languages. Child evidence: `641cb49`, preview 34190701402, Verify 34190701428.
- [x] 2026-09-08: Verify existing manifests, leases and development workflows are unchanged. Child evidence: `641cb49`, preview 34190701402, Verify 34190701428.
- [x] 2026-09-08: Integrated `go run ./tools/repoctl check` and `go test -race ./...` passed on Linux Go 1.27.1. Independent review of production changes found no confirmed defect. Final native CI/release evidence passed at `a1013b5` (runs 34197046022 and 34197049188).
- [x] 2026-09-08: Record native-platform and release evidence honestly. Child evidence: `641cb49`, preview 34190701402, Verify 34190701428.
- [x] 2026-09-08: Completed acceptance evidence and retrospective.
- [x] 2026-09-08: Moved both ExecPlans to `docs/exec-plans/completed/`.

A checked item means observed completion. Record date, exact command/run,
revision and outcome at each meaningful checkpoint.

## Surprises & Discoveries

- 2026-09-08: The first Windows fix (`611da29`, native run 34196522199) eliminated fixture/replace errors but still reproduced transient read sharing violations from competing publication calls. No-replace alone is insufficient; add bounded handling of Windows sharing/lock violations with deterministic held-handle regressions. Permanent permissions, missing files and content mismatches must still fail.

- 2026-09-08: Native Windows CI 34196175504 exposed two gaps hidden by Linux/macOS: Git autocrlf changed the embedded fixture from 17 to 18 bytes, and replacing an already published immutable file caused sharing/access-denied failures under stress. Preserve fixture bytes with a scoped -text attribute and publish without replacement on Windows; keep the same stress assertions. Native revalidation is required.

- 2026-09-08: Resumption at merge `16afc83` found three gaps using new regressions: concurrent asset mkdir rejected legitimate EEXIST winners (five child failures), an absolute state override still needed HOME, and UI helper staging used OS temporary storage on both install success/error. Fixed each without weakening checks. Asset race stress passed ten repetitions: 120 children, 10,800 materializations, 300 fresh roots. Intermediate combined tests saw duplicate AssetInfo while files were being integrated and the intentionally failing staging test; final validation must use the integrated tree.

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

- Decision: Reconcile the original open distribution choices with the completed child: strict numeric Git tags alone, no wall-clock identity, linker identity plus ReleaseRecord for trimmed release inspection, six CGO-disabled targets, versioned top-level directories, tagged-commit mtimes, AGENT_ENV_HOME and assets beneath that root, direct test-only go:embed, Go 1.27.1 pinned in release CI, and same-source/toolchain byte comparison for all eight files. No cross-process asset lock is needed after verified EEXIST handling. Signing/SBOM/attestations and extra native tuples are follow-up work. The first public version remains a maintainer publication choice, not a second version source or implementation blocker. Rationale: these are the implemented, directly tested child/parent contracts; no unresolved packaging mechanism remains. Date/Author: 2026-09-08 / maintainers.

- Decision: Keep the production asset inventory explicitly empty; describe future embedded consumers through the existing Describe/Materialize API and a test-only embedded fixture. Adding an actual helper must update CLI/manifest provenance together. Reject corruption rather than silently repairing it; revalidate mkdir race winners without adding locks. Rationale: identical immutable bytes can publish concurrently, and no production helper should be invented for a packaging task. Date/Author: 2026-09-08 / maintainers.

- Decision: Honor absolute AGENT_ENV_HOME before home discovery and stage APK installation in the owned runtime directory. Document external Git registration/tool caches separately from owned state. Rationale: headless installations need no default home, and crash leftovers belong to the lease while trusted external tools retain their existing responsibilities. Date/Author: 2026-09-08 / maintainers.

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

Implementation and local validation finished on 2026-09-08 at `a1013b5`.
Final native CI passed: Verify [34197046022](https://github.com/mahcialet/agent-env/actions/runs/34197046022) and Release preview [34197049188](https://github.com/mahcialet/agent-env/actions/runs/34197049188). The parent is complete and archived.

The completed child supplies strict Git-tag validation, isolated immutable build
sources, static artifact validation and GitHub Release publication gates. The
parent adds honest lazy prerequisite checks, explicit CLI inventory, deterministic
embedded fixture/concurrent materialization evidence and the persistent-path audit.

All Windows/macOS/Linux amd64/arm64 archives contain a versioned top-level directory
with the executable, LICENSE and README.txt. Git `v<major>.<minor>.<patch>` is the only
release-version authority: HEAD must equal the unique canonical tag, tracked/index/
untracked source must be clean, and requested version must equal the tag without v.
Go 1.27.1 and tagged-commit timestamps produce eight byte-identical candidate files
across two independent builds. Runtime linker identity and ReleaseRecord expose
version/commit/dirty/platform/build provenance without needing the source tree.

Production assets remain explicitly empty. The test-only go:embed fixture proves
the generic Describe/Materialize path without a target app or downloader. Windows
uses no-replace publication and verifies the winning bytes; Unix atomically renames
identical content. Inventory remains state-free. Absolute AGENT_ENV_HOME works
without home discovery, and owned temporary APKs now stay with the runtime. The
design audit explicitly excludes external-tool-owned Git registration/caches.

Native release smoke covers Linux/amd64, Windows/amd64 and macOS/arm64; the other
three tuples have cross-build/static evidence only. It checks version/help/list
outside source with empty PATH and Unicode/space paths. Release workflow publishes
the already-validated candidate after native gates, without rebuilding. No public
tag or Release was created by this work; preview v0.1.0 is private test input.

The key lesson is that Linux race testing alone missed Windows replacement-sharing
semantics and Git newline conversion. Native CI caught both; scoped byte attributes
and no-replace Windows publication fixed them without relaxing the stress test.
A separate reviewer found no confirmed production defect after the final revision.

Signing, notarization, package managers, SBOM/attestations and additional native
architectures remain explicitly future work. Android UI observation may consume
embedded bytes through this contract, but actual helper embedding requires its own
license/version/inventory change; no lifecycle responsibility moved into Flutter.

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
| S1 | Extracted release runs `version`/help without Go or repository files. | Child R15 and preview 34190701402: extracted version/help/list run with empty PATH outside source on all three native OS runners. |
| S2 | Core CLI startup does not require Docker, Android, Flutter, Java, Python, Node or shell. | The same native smoke runs core commands with no optional tools on PATH. |
| S3 | Capability-specific commands report missing prerequisites lazily and honestly. | 2026-09-08, `8eb92d5`: `TestStandaloneCommandsRequestGitOnlyWhenSourcesAreNeeded` (empty PATH) and existing Compose/Android/Flutter doctor/plan regressions pass: actionable missing tools, exit 3, no premature state or unrelated discovery. |
| S4 | `version --output json` reports documented version/commit/toolchain/platform/asset metadata. | 2026-09-08, `8eb92d5`: `TestVersionReportsBundledInventoryWithoutState` passes JSON `assets: []` and table inventory; buildinfo tests retain honest development/release identity. |
| S5 | Development builds have an honest identity without release metadata. | 2026-09-08: development defaults report `devel`/`unknown` identity through `agent-env version`; unit test passes. |
| S6 | Release builds reject mismatched tag/version/commit or dirty release input. | Child R1–R4: strict Git identity and negative fixtures, including private committed-source isolation. |
| S7 | Release builds use `CGO_ENABLED=0` and no shell packaging tools. | Child R5: all six binaries statically verify CGO_ENABLED=0; Go-only archive mechanics pass. |
| S8 | The fixed matrix produces exactly the documented archive set. | Child R6–R7: exact six archive names and three prefixed regular members. |
| S9 | Checksums and release manifest match exact archive/executable bytes. | Child R9–R11: real checksums/manifest/binary identity verified and mismatch fixtures rejected. |
| S10 | Archives contain only safe relative regular files; no symlink/traversal. | Child R12: traversal/symlink/member tests plus ZIP local/central-name regression pass. |
| S11 | Bundled asset materialization is content-addressed, digest-verified, atomic, concurrency-safe and idempotent. | 2026-09-08, `8eb92d5`: `TestEmbeddedFixture`, tamper/path tests and `go test -race ./internal/assets -count=10` pass: 120 children, 10,800 calls, 300 fresh roots, identical final bytes and no temporary residue. |
| S12 | Corrupted materialized asset content is detected and never silently trusted. | 2026-09-08: `TestMaterializeIsContentAddressedAndIdempotent` rejects tampered bytes. |
| S13 | Bundled asset metadata is available without capability initialization. | 2026-09-08, `8eb92d5`: `assets.Inventory` feeds buildinfo without I/O; CLI test uses empty PATH and verifies state stays absent. Test-only fixture is not shipped. |
| S14 | Persistent runtime state follows documented state-root precedence and does not leak into target repositories. | 2026-09-08, `8eb92d5`: Design persistent-path audit plus `TestResolveOverrideWithoutUserHome`, `TestLifecyclePersistentStateStaysUnderOverride`, `TestNamedCommandEvidenceStaysUnderStateRoot`, `TestUIHelperStagingStaysInOwnedRuntimeAndCleansUp` and existing Android private-environment tests pass. External-tool state is explicitly distinguished. |
| S15 | Explicit state root works on native Windows/macOS/Linux paths including spaces; non-ASCII is tested where practical. | Child R16: Unicode/space state-root smoke on Windows/amd64, macOS/arm64 and Linux/amd64. |
| S16 | Repeated same-source/toolchain release construction is compared for deterministic binary/archive output and any gap is documented. | Child R14: two same-source/toolchain builds at 641cb49 produce eight identical files. |
| S17 | Extracted release artifacts are smoke-tested on native Windows/macOS/Linux. | Preview 34190701402 passes native smoke on Windows/amd64, macOS/arm64 and Linux/amd64. |
| S18 | Cross-build-only architecture evidence is clearly separated from native execution. | Windows/arm64, macOS/amd64, Linux/arm64 remain cross-build/static-only evidence. |
| S19 | GitHub release workflow delegates artifact mechanics to repoctl and refuses publication after validation failure. | Child R22–R24: repoctl mechanics and tested validation/repeat/native publication gates; no public release was cut. |
| S20 | Release workflow starts from a maintainer-created tag and never rewrites Git history. | Child R21: maintainer tag trigger; only private-clone test tags created, caller refs protected by tests. |
| S21 | Existing manifests, leases and development commands continue to work unchanged. | Verify 34190701428 passes all 12 jobs, including existing native tests and actual Linux Docker integration. |
| S22 | Full harness, docs/translation checks and Go race suite pass at final revision. | Child R26: final code 641cb49 passes local race and all hosted harness jobs; parent-specific remaining work still needs its own final check. |
| S23 | Standalone product/design/ExecPlan docs exist in English and Japanese and are indexed. | Bilingual indexed docs exist and docs-check passes after child archival. |
| S24 | Roadmap no longer describes archive-based release packaging as undecided after completion. | Updated bilingual roadmap settles archive-based GitHub Release packaging. |
| S25 | The distribution contract is sufficient for `android-ui-observer` to consume a future embedded helper without a separate downloader design. | 2026-09-08, `8eb92d5`: Design documents the future go:embed → Describe → Materialize consumer contract; `TestEmbeddedFixture` proves it without SDK/target app/downloader. Current external helper remains unchanged; actual production embedding is follow-up work. |

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

- 2026-09-08: Final revision `a1013b5` passed local harness, full race, six-target release-verify (two builds/eight identical files/Linux native smoke), and all 18 candidate cases. Verify 34197046022 passed all native Go 1.26/1.27 jobs, cross-build and integration/race. Release preview 34197049188 passed candidate build and Linux/amd64, Windows/amd64, macOS/arm64 smoke. Windows held-handle, >280-character path, winner-preservation and process stress regressions now pass without relaxing assertions. No other native tuples are claimed.

2026-09-08 parent validation at `a1013b58a3e4e1e98b4d741237e193f8013eec65`,
Go 1.27.1, private preview `v0.1.0` (no public tag):
`go run ./tools/repoctl release-verify --out dist/parent-final-verified` passed
all six builds twice, eight byte-identical files and Linux/amd64 native smoke.
`AGENT_ENV_RELEASE_CANDIDATE=../../dist/parent-final-verified go test ./tools/repoctl -run TestReleaseCandidate -count=1 -v`
passed all 18 cases. Manual archive inspection found exactly three regular members
per versioned prefix: executable, LICENSE (1066 bytes), README.txt (666 bytes).
Production bundled assets: 0, 0 bytes. Local preview artifact evidence:

| Target | Archive bytes | Archive SHA-256 |
| --- | --- | --- |
| windows/amd64 | 5987107 | `467a0026c23a1f4c0c34986f7d8f86da009a7e819eb0919c83edb1c23cc9b635` |
| windows/arm64 | 5480058 | `c9635cc44b41ce2a4269a40d41b3a59d23ca1129d1e34e2c4a5d77374ff2967f` |
| darwin/amd64 | 5841440 | `3fe24eb6516977e7a8ad07ad3e04dfa087cdb440284195371de5ca1b176619f7` |
| darwin/arm64 | 5529771 | `e963f6700047f860795ffa849d87816b4f69c4f0ece3a74db654540cb67e9956` |
| linux/amd64 | 5776247 | `20d8936a9383479de523b60ca4740e32a2041198e71f2d67e6a3588aa4e4f3cc` |
| linux/arm64 | 5354850 | `cf759872d158328443398a4476e07a325eea68ff8a69507e4778f6ad0d5ccb23` |

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
