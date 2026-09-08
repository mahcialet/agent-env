---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/active/standalone-distribution.md
source_sha256: bb599da3a0d90740385815ef9a17fcc331441050e191515fdf95c5ef3bd6b38c
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

## 子 ExecPlan

[standalone-release-finalization.ja.md](../completed/standalone-release-finalization.ja.md) は、本計画の完了した子 ExecPlan である。

残っている具体的なリリース実装・検証のうち、次を担当する。

- 厳密な Git tag のリリース検証
- `repoctl release-build`
- `repoctl release-check`
- 6 対象のアーカイブ生成
- 決定性・再現性の検証
- ネイティブ smoke test
- GitHub Release workflow
- 最終的なリリース文書と証拠

子計画の完了は必要条件であるが、それだけで親計画を完了してよいわけではない。
子計画を完了済みへ移動した後、親計画の受け入れ条件に照らして実装された動作と証拠を
整理し直し、「成果と振り返り」を記入する必要がある。

## 進捗

- [x] 2026-09-08: PR #7のmerge `16afc83` から親Planを再開。baselineの `go run ./tools/repoctl check` が成功。同梱一覧をCLIに明示し、JSON/表形式と状態未作成の回帰テストを追加。残りのauditと最終検証は継続中。

- [x] 2026-09-08: master `938e584`（PR #5を含む）から作業ブランチを作成。
- [x] 2026-09-08: baseline Go 1.27.1 raceが成功。初回checkは日本語Planの
  metadata欠落で失敗し、修復後のcheckが成功。
- [x] 2026-09-08: CLI/bootstrap・state root・repoctl・CI・cross-buildを確認。
- [x] 2026-09-08: 英日のproduct/design契約とindexを追加し、翻訳metadataを同期。
- [x] 2026-09-08: Git tagだけをversionの根拠とし、厳密な形式・clean tree・
  HEAD一致・要求version一致を方針とした。
- [x] 2026-09-08: buildinfoとversion出力に開発用default、commit、dirty、Go、
  platformを追加。任意providerは初期化しない。
- [x] 2026-09-08: 6ターゲットとtop-level directory付きarchive、commit mtimeを確定。
- [x] 2026-09-08: generic assetのcontent-addressed・digest検証・atomic生成、
  再利用、traversal/symlink/tamper拒否を実装した。
- [x] 2026-09-08: 子 `docs/exec-plans/completed/standalone-release-finalization.ja.md`
  を完了。release-build/check・packaging・native smoke・GitHub workflowを
  `641cb49`、preview 34190701402、Verify 34190701428で検証した。
- [x] 2026-09-08: 前提エラー、CLI一覧、asset stress、永続パスaudit、将来helper契約を実装・検証。local harness/race成功。最終revisionのnative CIは下記で記録する。
- [x] 2026-09-08: TestEmbeddedFixtureで17バイトと固定SHA-256を対象appなしで検証。asset race stressは10回成功。
- [x] 2026-09-08: 既存判断通りAGENT_ENV_HOMEのみをoverrideとし、--homeは追加しない。
- [x] 2026-09-08: design文書に全永続パス監査を記録。override、lifecycle、command evidence、helper stagingの回帰テストが成功。
- [x] 2026-09-08: 子でrelease-build/check、正規化archive/manifest/checksums、
  厳密なtag/version/clean guard、再現性比較を実装・検証した。
- [x] 2026-09-08: 子で3OSの展開済みnative smokeとrepoctlに委ねるtag workflowを検証。
- [x] 2026-09-08: 子でarchitecture/portability/quality/security/roadmapを英日更新。
- [x] 2026-09-08: Verify 34190701428で既存manifest/lease/workflowの回帰検査も成功。
- [x] 2026-09-08: 統合後の `go run ./tools/repoctl check` と `go test -race ./...` がLinux Go 1.27.1で成功。production変更の独立レビューに確認済み不具合なし。archival前に最終native CI/release証拠を確認する。
- [x] 2026-09-08: native/crossの範囲とリリース証拠を子から引き継いだ。
- [ ] 親の受け入れ証拠と振り返りを完成する。
- [ ] 親の英日Planをcompletedへ移動する。

チェックは観測済み完了を示す。子の完了で親の未検証条件を完了扱いにしない。

## 想定外の発見

- 2026-09-08: merge `16afc83` からの再開時に回帰テストで3件を発見。同時asset mkdirが正常な先行作成を拒否し5子プロセスが失敗、絶対パスoverrideでもHOMEが必要、UI helperの一時コピーがinstall成功時・失敗時ともOS一時領域に作成されていた。検証条件を弱めず修正。asset race stressは10反復、120子プロセス、10,800展開、300新規rootで成功。統合途中のテストはAssetInfo重複宣言と意図したstaging回帰で失敗したため、統合後に最終検証する。

- 2026-09-08: 提供された日本語の active plan に翻訳 metadata がなく、実装前の
  baseline docs-check が失敗した。正確な翻訳 metadata を追加して hash を同期した。
  検査は弱めていない。
- 2026-09-08: 既存の `AGENT_ENV_HOME` が、OS 固有の絶対パスによる状態保存先の
  優先順位を既に定義していた。別の `--home` flag は同じ方針を重複させるため、
  今回は既存の override を維持し、その判断を記録する。

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

- 判断: 製品の同梱一覧は明示的な空配列とし、将来の埋め込み利用は既存のDescribe/Materialize APIとテスト専用fixtureで示す。実際のhelper追加時はCLIとmanifestの由来情報を同時に更新する。破損は黙って修復せず拒否し、mkdir競合はロック追加ではなく先行作成の再検証で扱う。理由: 同一の不変バイト列は並行公開でき、packagingのために不要な製品helperを作るべきではない。日付/担当: 2026-09-08 / maintainers。

- 判断: 絶対パスのAGENT_ENV_HOMEをhome探索より先に採用し、APK install用コピーは所有runtime内に置く。Git登録情報と外部ツールのキャッシュは所有状態と区別する。理由: headless環境ではdefault homeが不要であり、crash残留物はlease内に保存する一方、信頼する外部ツールの既存責務は維持する。日付/担当: 2026-09-08 / maintainers。

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

- 判断: スタンドアロンの状態保存先 override は `AGENT_ENV_HOME` のままとし、
  今回は別の `--home` flag を追加しない。
  理由: 既存のパス解決が OS 固有の絶対パスを必須にし、lifecycle コードもその結果を使っている。
  CLI に優先順位を重複実装すると、状態保存先の仕様が二つできてしまうため。
  日付/担当: 2026-09-08 / maintainers.

## 成果と振り返り

2026-09-08 子との照合: `641cb49` でリリース工程を完了した。子の受け入れ証拠を
下記の S1/S2、S6–S10、S15–S24 に対応付けた。親はactiveのままとし、S3、
S4/S13のasset一覧、S11のcross-process stress、S14の広範な永続パスaudit、
S25の将来helper契約は直接的な証拠を引き続き必要とする。子の完了をこれらの
親の検証の代わりにしない。

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
| S1 | release展開後Go/source tree無しで`version`/help実行可能。 | 子R15・preview 34190701402で3OSの展開済みversion/help/listを空PATH・ソース外で確認。 |
| S2 | core startupにDocker/Android/Flutter/Java/Python/Node/shell不要。 | 同じnative smokeで任意の外部ツールなしにcore commandを実行。 |
| S3 | capability commandがmissing prerequisiteをlazy/正直に報告。 | Pending |
| S4 | `version --output json`がversion/commit/toolchain/platform/asset metadataを返す。 | buildinfo.CurrentとCLI JSONはversion/commit/dirty/Go/platformをprovider初期化なしで表示。asset一覧は未完了。 |
| S5 | development buildがrelease metadata無しでも正直なidentity。 | 開発defaultはdevel/unknownと表示し、単体テストが成功。 |
| S6 | tag/version/commit mismatchまたはdirty release inputを拒否。 | 子R1–R4の厳密なGit検証と専用commit checkoutの負例が成功。 |
| S7 | release buildが`CGO_ENABLED=0`かつshell packaging tool不要。 | 子R5で6バイナリのCGO=0とGoのみのarchive生成を確認。 |
| S8 | fixed matrixからdocumented archive setだけ生成。 | 子R6–R7で6つの名前とprefix配下の3通常memberを確認。 |
| S9 | checksum/manifestがarchive/executable bytesと一致。 | 子R9–R11で実bytesのchecksum/manifest/identityと不一致拒否を確認。 |
| S10 | archiveはsafe relative regular filesのみでsymlink/traversal無し。 | 子R12のtraversal/symlink/memberとZIP local/central名の回帰が成功。 |
| S11 | bundled assetがcontent-addressed/digest verified/atomic/concurrency-safe/idempotent。 | assetsのcontent-addressed再利用とatomic生成のテスト成功。cross-process stress証拠は未完了。 |
| S12 | corrupt materialized assetを検出しsilent trustしない。 | TestMaterializeIsContentAddressedAndIdempotentで改ざんbytesを拒否。 |
| S13 | capability initialize無しでasset metadata取得可能。 | Pending |
| S14 | persistent stateがstate-root precedenceへ従いtarget repoへ漏れない。 | Pending |
| S15 | explicit state rootがnative OS path/spaceで動き可能ならnon-ASCIIも検証。 | 子R16でWindows/amd64・macOS/arm64・Linux/amd64のUnicode/空白state rootを確認。 |
| S16 | same-source/toolchain repeated releaseのdigest比較とgap記録。 | 子R14で641cb49の同source/toolchainによる8ファイル一致を確認。 |
| S17 | extracted releaseをnative Windows/macOS/Linuxでsmoke test。 | preview 34190701402でWindows/amd64・macOS/arm64・Linux/amd64のnative成功。 |
| S18 | cross-build-onlyとnative evidenceを区別。 | Windows/arm64・macOS/amd64・Linux/arm64はcross-build/静的検査のみ。 |
| S19 | GitHub release workflowがmechanicsをrepoctlへ委譲しvalidation failure後publishしない。 | 子R22–R24でrepoctl処理とvalidation/repeat/native公開gateを検証。公開releaseは作成していない。 |
| S20 | maintainer-created tagを使いCIがGit historyを変更しない。 | 子R21でmaintainer tag起動。tag作成はprivate cloneのみで、caller refs保護をテスト。 |
| S21 | 既存manifest/lease/development command非回帰。 | Verify 34190701428の12 job（既存nativeとLinux Docker実integrationを含む）が成功。 |
| S22 | full harness/docs/translation/race pass。 | 子R26で641cb49のローカルraceとhosted harness成功。親の残作業には別途最終検査が必要。 |
| S23 | standalone docs/ExecPlan英日双方が存在しindex済み。 | 英日文書・indexがあり、子archival後のdocs-check成功。 |
| S24 | 完了後roadmapのarchive release undecided表現解消。 | 英日roadmapでarchiveによるGitHub Releaseを決定済みとした。 |
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
