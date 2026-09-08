---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/active/standalone-distribution.md
source_sha256: 4d3a52c60467a03448709c7ce4128901a39e45172b0d50b3ca064db5d5a7e041
---

# agent-env をクロスプラットフォームのスタンドアロン配布物にする

[English](standalone-distribution.md)

この ExecPlan は living document であり、`docs/PLANS.md` に従って更新する。

想定ブランチ: `feat/standalone-distribution`

PR #4 (`feat: run Flutter Android applications on owned emulator leases`) の
マージ後に開始することを推奨する。本作業は後続の `android-ui-observer` 等が
利用する packaging、build provenance、runtime asset 境界を変更するため、
不安定なdistribution contractの上に後続機能を積まない。

実装開始時に `master` を更新し、正確なrevisionを以下へ記録し、専用branchを
作成し、既存repository harnessを実行してbaselineを記録する。

開始 revision: `938e584`（PR #5 merge後の`master`）

## 目的 / 全体像

この作業の完了後、`agent-env` をnative Windows / macOS / Linux向けの
standalone applicationとして配布できる。

standaloneを以下のように定義する。

- `agent-env` 自身の実行にGo installation不要
- CLI実行だけのためにPython、Node.js、Bash、POSIX shell、PowerShell、Make、
  repository checkout不要
- agent-env所有companion assetを初回利用時に別download/buildさせず同梱可能
- installはarchive展開・binary配置のみ
- executable自身がversion/source revision/platform/bundled asset provenanceを報告
- writable stateはOS-native state rootまたは既存の`AGENT_ENV_HOME`配下
- release archive生成・検証はrepositoryのGo harnessが担当
- 展開release artifactをnative Windows/macOS/Linuxでsmoke test

standaloneはoptional external toolすべての同梱を意味しない。

- Git sourceにはGit
- Compose stackにはDocker + Compose
- Android EmulatorにはAndroid SDK / Emulator / host acceleration
- Flutter buildにはFlutter / Java / Android build toolchain

が必要である。

APIだけ使うユーザーへAndroid/Flutterを要求しない。`agent-env version`、help、
core diagnosticだけでDocker/Android/Flutter/Javaをinitializeしない。

想定release:

    agent-env_vX.Y.Z_windows_amd64.zip
    agent-env_vX.Y.Z_windows_arm64.zip
    agent-env_vX.Y.Z_darwin_amd64.tar.gz
    agent-env_vX.Y.Z_darwin_arm64.tar.gz
    agent-env_vX.Y.Z_linux_amd64.tar.gz
    agent-env_vX.Y.Z_linux_arm64.tar.gz
    checksums.txt
    release-manifest.json

通常archiveはexecutable、MIT license、簡潔なstandalone install/readmeを含む。
将来Android UI companion APK等のagent-env所有helperが必要になった場合、原則Go
executableへembedし、必要時にcontent-addressedな検証済みlocationへ展開する。
別manual downloadを要求しない。

compatibility burdenが小さい今の段階では、長期standalone contractを単純にする
ための内部package/build layoutの大きな変更を許容する。ただしpublic behavior
変更は明示的にdocument/migrationする。

## 進捗

- [ ] PR #4 merge、`master`更新、開始revision記録、`feat/standalone-distribution`作成。
- [ ] baseline `repoctl check` とdocumented race suiteを記録。
- [ ] build info/state root/CI/release/cross-buildを調査。
- [ ] standalone product/design docsを英日作成。
- [ ] release version source/tag policy確定。
- [ ] central build-info APIと`agent-env version`定義。
- [ ] release target matrix / archive naming/layout確定。
- [ ] generic bundled asset metadata/materialization実装。
- [ ] target-app dependencyを作らないembedded asset test追加。
- [x] 2026-09-08: 既存の`AGENT_ENV_HOME`をstate-root overrideとして維持し、
      重複する`--home` flagは追加しない判断を記録。
- [ ] 全persistent writable pathがstate-root contractへ従うことを検証。
- [ ] `repoctl release-build`実装。
- [ ] `repoctl release-check`実装。
- [ ] normalized archive/release manifest/checksum生成。
- [ ] exact version/tag/clean-tree release guard追加。
- [ ] same-source deterministic build/package regression追加。
- [ ] native Windows/macOS/Linux extracted-artifact smoke追加。
- [ ] mechanicsをrepoctlへ委譲するGitHub tag/release workflow追加。
- [ ] architecture/portability/quality/security/roadmapを英日更新。
- [ ] 既存manifest/lease/development workflowの非回帰確認。
- [ ] full harness/race validation。
- [ ] native platform/release evidenceを正直に記録。
- [ ] acceptance/retrospective完成。
- [ ] 英日planをcompletedへ移動。

checkboxは観測済み完了を表す。checkpointごとに日付、revision、command/run、
resultを記録する。

## 想定外の発見

現時点ではなし。

少なくとも以下を記録する。

- build modeによるGo/VCS metadataの予期せぬ差
- archive metadataによるnondeterminism
- cross-buildから見えないWindows executable/path behavior
- resolved home外へのstate漏れ
- optional providerのeager initialization
- embedded asset concurrency/corruption
- release logicをYAMLへ移したくなるGitHub Actions制約
- cross-build成功/native実行失敗target
- extracted artifactに対するantivirus/quarantine挙動
- helper肥大化によるone-binary方針への影響
- bundled assetによるlicense obligation

設計へ影響した失敗した試行は残す。

## 判断の記録

- 判断: standaloneを「agent-env自身にlanguage/runtime/manual helper downloadが
  不要」と定義し、「optional external tool全部をbundle」とはしない。
  理由: Git/Docker/Android/Flutter prerequisiteを維持しつつcoreを小さくするため。
  日付/担当: 2026-09-08 / maintainers.

- 判断: 通常runtime distributionは単一Go executable + license/readmeとし、
  future companionは原則embed/on-demand materializeする。
  理由: installを単純化しcompanion version/download driftを防ぐため。
  日付/担当: 2026-09-08 / maintainers.

- 判断: bundled assetはAndroid専用ではなくgeneric core infrastructureとする。
  理由: Android UI helper/browser helper等で同じprovenance modelを使うため。
  日付/担当: 2026-09-08 / maintainers.

- 判断: Git/Docker/Android SDK/Flutter/Javaはbundleしない。
  理由: 大容量かつ独立更新・license・platform constraintを持つため。
  日付/担当: 2026-09-08 / maintainers.

- 判断: release mechanicsは`repoctl`へ置き、CIはharnessを呼ぶだけにする。
  理由: cross-platform仕様を一つにしrepositoryをsystem of recordにするため。
  日付/担当: 2026-09-08 / maintainers.

- 判断: executableへwall-clock build timestampを埋め込まない。
  理由: version/commit/toolchain identityの方が有用でnondeterminismを減らすため。
  日付/担当: 2026-09-08 / maintainers.

- 判断: OS-native state defaultを維持し、binary隣へ暗黙にstateを書かない。
  理由: Program Files等read-only pathへinstallされ得るため。
  日付/担当: 2026-09-08 / maintainers.

- 判断: 初期releaseはGitHub Release archive + SHA-256 checksumとしpackage
  manager/signingはfollow-up。
  理由: artifact contract確立後にdistribution surfaceを増やすため。
  日付/担当: 2026-09-08 / maintainers.

- 判断: durable docs/living ExecPlanは英日双方で維持。
  理由: bilingual documentation policyに従うため。
  日付/担当: 2026-09-08 / maintainers.

## 成果と振り返り

未完了。

完了時にまとめる。

- release target matrix
- archive naming/layout
- version/tag policy
- binary/build provenance
- bundled asset design
- state-root behavior
- determinism/reproducibility evidenceと制約
- native smoke evidence
- GitHub Release workflow
- platform distribution gap
- signing/notarization/package manager/SBOM follow-up
- `android-ui-observer`への影響

## 背景と構成

実装前に読む。

- `AGENTS.md` / `AGENTS.ja.md`
- `ARCHITECTURE.md` / `ARCHITECTURE.ja.md`
- `docs/PLANS.md` / `docs/PLANS.ja.md`
- `docs/PORTABILITY.md` / `docs/PORTABILITY.ja.md`
- `docs/QUALITY.md` / `docs/QUALITY.ja.md`
- `docs/SECURITY.md` / `docs/SECURITY.ja.md`
- `docs/RELIABILITY.md` / `docs/RELIABILITY.ja.md`
- `docs/roadmap.md` / `docs/roadmap.ja.md`
- completed MVP / Android Emulator plan
- PR #4 merge後のFlutter Android英日docs/plan
- `.github/workflows/`
- `tools/repoctl`
- state/path package
- CLI root construction
- no-CGO/cross-build tests

既存invariantはnative Windows/macOS/Linux、mandatory CGO無し、POSIX shell無し、
argv process、OS-native state root、`AGENT_ENV_HOME`、capability-specific
prerequisite、cross-build != native evidence。

roadmapではrelease packagingがundecided。本planはarchive-based standalone
releaseを確定し完了時roadmap更新。

companion APKを採用し得る`android-ui-observer`より先に完了するのが望ましい。
UI observerは独自downloader/extractorを作らずgeneric bundled assetを利用する。

## 作業計画

### Milestone 1 — Product contract / build audit

main/CLI bootstrap、`debug.ReadBuildInfo`/linker variable、state root、repoctl、
CGO/cross-build CI、workflow permission、build outputを調査。

英日作成:

    docs/product-specs/standalone-distribution.md
    docs/product-specs/standalone-distribution.ja.md
    docs/design-docs/standalone-distribution.md
    docs/design-docs/standalone-distribution.ja.md

release version source、tag format、target matrix、archive layout、`--home`要否、
Go toolchain pinningを決定し必要ならADR/index更新。

### Milestone 2 — Build info / version

build-info source of truthを1つにする。

最低限:

- release versionまたは`devel`
- source commit
- knowableなdirty state
- Go version
- GOOS/GOARCH
- bundled asset metadata

    agent-env version
    agent-env version --output json

release linker flag無しdevelopment buildも動く。`version`でoptional providerを
initializeしない。decorativeなwall-clock timestampは追加しない。

### Milestone 3 — Generic bundled asset

agent-env-owned immutable asset用generic facilityを追加。

要件:

- compile/release inputでtarget repo contentではない
- safe internal name
- exact SHA-256
- content-addressed path
- atomic write/rename
- reuse前digest verify
- symlink/path traversal無し
- concurrency-safe/idempotent
- capability initialize無しでmetadata取得
- capability利用時だけmaterialize

production helperが無ければtiny test fixtureで証明しuser archiveへ不要assetを入れない。

将来:

    assets.Materialize("android-ui-helper")

のように利用可能にする。

### Milestone 4 — State-root contract

SQLite、worktree、evidence、private AVD、bundled asset、logs、atomic temp file等をaudit。

`--home`追加なら:

    --home > AGENT_ENV_HOME > OS-native default

external effect前validate。relative pathはabsolute化。binary隣へ暗黙writeする
portable modeは追加しない。

### Milestone 5 — Cross-platform release builder

repoctlで:

    release-build --version 0.1.0 --out dist

matrix:

- windows/amd64
- windows/arm64
- darwin/amd64
- darwin/arm64
- linux/amd64
- linux/arm64

`CGO_ENABLED=0`、必要なら`-trimpath`、Go libraryによるzip/tar/gzip/checksum。
external shell tool無し。

release-manifestにversion/source commit/Go version/target/archive+binary digest/
bundled asset digest。host path/secret/temp path無し。

file order/path/mode/timestamp/gzip metadataをnormalize。

### Milestone 6 — Release validator

`repoctl release-check`でexpected files、name/version/target、checksum、manifest、
executable、build info、license/readme、absolute/`..`/symlink無し、local path leak無し、
asset metadataを検証。corruption/traversal negative fixture追加。

### Milestone 7 — Determinism / no-source smoke

同一source/toolchainから少なくとも1targetを2回buildしbinary/archive digest比較。
差異を調査し、証明前にfull reproducible buildをclaimしない。

repository外へextractして:

    agent-env version --output json
    agent-env --help
    agent-env doctor

またはnarrow core doctorを実行。`go run`/source tree使用禁止。

### Milestone 8 — Native distribution CI

Windows/macOS/Linux native runnerでextracted artifactを実行。
amd64を最低限native。arm64 runner無しの場合cross-build/archive validationをnativeと
混同しない。可能ならspace/non-ASCII state pathも検証。

### Milestone 9 — GitHub Release workflow

tag-triggered flow:

1. maintainer-created exact tag checkout
2. tag/version/source検証
3. repository checks
4. repoctl release-build
5. repoctl release-check
6. native smoke
7. validated filesのみpublish

least-required permission。CIはtagを作成/移動しない。validation failure後publishしない。

### Milestone 10 — Documentation / migration

README install、architecture、portability、quality、security必要箇所、roadmap、indexesを
英日更新。installはarchive展開+PATH+version/doctor。未対応package manager commandは
書かない。standaloneがDocker/Android/Flutter同梱ではないことをmatrix化。

## 具体的な手順

1. PR #4 merge待ち。
2. `git switch master && git pull --ff-only`。
3. 開始revision記録。
4. `feat/standalone-distribution`作成。
5. 英日active plan追加。
6. baseline harness/race。
7. build-info/state/CI/repoctl audit。
8. bilingual product/design docs。
9. build-info core + `version`。
10. generic bundled asset。
11. state-root override/precedence確定。
12. `repoctl release-build`。
13. `repoctl release-check`。
14. deterministic build regression。
15. native extracted-release smoke。
16. GitHub tag/release workflow。
17. bilingual docs/roadmap更新。
18. full harness/race/cross-build/native smoke。
19. final archive manual inspectionとevidence。
20. acceptance/retrospective英日完成。
21. completedへ移動。

## 検証と受け入れ

| ID | 必須動作 | 証拠 |
| --- | --- | --- |
| S1 | release展開後Go/source tree無しで`version`/help実行可能。 | Pending |
| S2 | core startupにDocker/Android/Flutter/Java/Python/Node/shell不要。 | Pending |
| S3 | capability commandがmissing prerequisiteをlazy/正直に報告。 | Pending |
| S4 | `version --output json`がversion/commit/toolchain/platform/asset metadataを返す。 | Pending |
| S5 | development buildがrelease metadata無しでも正直なidentity。 | Pending |
| S6 | tag/version/commit mismatchまたはdirty release inputを拒否。 | Pending |
| S7 | release buildが`CGO_ENABLED=0`かつshell packaging tool不要。 | Pending |
| S8 | fixed matrixからdocumented archive setだけ生成。 | Pending |
| S9 | checksum/manifestがarchive/executable bytesと一致。 | Pending |
| S10 | archiveはsafe relative regular filesのみでsymlink/traversal無し。 | Pending |
| S11 | bundled assetがcontent-addressed/digest verified/atomic/concurrency-safe/idempotent。 | Pending |
| S12 | corrupt materialized assetを検出しsilent trustしない。 | Pending |
| S13 | capability initialize無しでasset metadata取得可能。 | Pending |
| S14 | persistent stateがstate-root precedenceへ従いtarget repoへ漏れない。 | Pending |
| S15 | explicit state rootがnative OS path/spaceで動き可能ならnon-ASCIIも検証。 | Pending |
| S16 | same-source/toolchain repeated releaseのdigest比較とgap記録。 | Pending |
| S17 | extracted releaseをnative Windows/macOS/Linuxでsmoke test。 | Pending |
| S18 | cross-build-onlyとnative evidenceを区別。 | Pending |
| S19 | GitHub release workflowがmechanicsをrepoctlへ委譲しvalidation failure後publishしない。 | Pending |
| S20 | maintainer-created tagを使いCIがGit historyを変更しない。 | Pending |
| S21 | 既存manifest/lease/development command非回帰。 | Pending |
| S22 | full harness/docs/translation/race pass。 | Pending |
| S23 | standalone docs/ExecPlan英日双方が存在しindex済み。 | Pending |
| S24 | 完了後roadmapのarchive release undecided表現解消。 | Pending |
| S25 | `android-ui-observer`がseparate downloader無しでfuture embedded helper利用可能。 | Pending |

すべてdirect evidence必須。workflow/file存在だけではacceptance evidenceではない。

## 冪等性と復旧

release constructionは明示output dirだけへwriteしGit refs/target repo/lease/通常SQLiteを
変更しない。

`release-build`はoutput pathをvalidateし自身の既知outputだけ置換する。failureを
validated output扱いしない。`release-check`はread-only。

bundled asset materializationはconfigured asset/cache root内だけへwriteし、existing bytesを
digest verifyしてreuse、per-content path、concurrent same bytesに耐え、caller-controlled
symlinkをfollow/createしない。corrupt owned contentはquarantineまたはembedded trusted
bytesから安全にrematerializeしてよい。採用方式を記録する。

repoctlはtag作成/移動/削除/force updateしない。missing embedded assetをnetworkから
自動downloadしない。

## 成果物と注記

development output例:

    dist/
      agent-env_v0.1.0_windows_amd64.zip
      agent-env_v0.1.0_windows_arm64.zip
      agent-env_v0.1.0_darwin_amd64.tar.gz
      agent-env_v0.1.0_darwin_arm64.tar.gz
      agent-env_v0.1.0_linux_amd64.tar.gz
      agent-env_v0.1.0_linux_arm64.tar.gz
      checksums.txt
      release-manifest.json

plan evidence:

- source commit
- tag/version
- Go version
- repoctl release command
- archive/executable digest
- repeated build comparison
- native smoke run ID
- GitHub Actions run ID
- native未実行architecture
- archive size
- bundled asset数/size

release binary/archiveはcommitしない。future bundled APKにthird-party codeが入る場合は
license notice requirementをrelease前に記録。

## インターフェースと依存

想定領域:

    internal/buildinfo/
    internal/assets/
    internal/paths/
    internal/cli/
    tools/repoctl/

責務に応じ実調査後変更可。

release-tool struct、CLI JSON、runtime domain typeを便利さだけで1型へ統合しない。

development/release dependencyはGo/Git/GitHub Actions orchestration。runtime dependencyは
capability-specific。

standalone release mechanicsへBash/POSIX shell/PowerShell/external tar/zip/checksum/CGO/
mandatory daemon/separate helper downloaderを導入しない。

Milestone 1で解決する未確定事項:

1. 最初のpublic version/tag（`0.1.0`は例）
2. version authorityをGit tagのみとするか追加source fileが必要か
3. development identityに`debug.ReadBuildInfo`で十分かrelease linker overrideが必要か
4. archive内layoutをflatかversioned top directoryにするか
5. `--home`を今追加するか`AGENT_ENV_HOME`だけで十分か
6. assetをstate rootか別OS-native cache rootに置くか
7. asset concurrent materialization lock strategy
8. direct `go:embed`かgenerated metadataか
9. release Go patch version pin/upgrade policy
10. bit-for-bit determinismを6target全てに要求可能か
11. GitHub artifact attestation/SBOMを今入れるかfollow-upか
12. OS別arm64 native smoke infrastructure有無

public release contract固定前にDecision Logで解決する。
