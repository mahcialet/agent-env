---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/active/standalone-release-finalization.md
source_sha256: 3685f4148514bf01e14d338fc38dfd191b941c94dbe9acaf01db25939516ead1
---

# スタンドアロン配布のリリース工程を完成させる

[English](standalone-release-finalization.md)

この ExecPlan は living document であり、`docs/PLANS.md` に従って更新する。

親ExecPlan:

`docs/exec-plans/active/standalone-distribution.ja.md`

本Planは親Planを置き換えるものではなく、親Planを完了するために残っている
release engineering作業を実装・検証する子ExecPlanである。

想定ブランチ: `feat/standalone-release-finalization`

本Planはstandalone-distribution基盤の後続として、archive-based standalone
release pipelineを完成させる実行authorityである。

前提基盤:

- `internal/buildinfo` とdevelopment版 `agent-env version`
- digest verification / tamper detection付きgeneric bundled asset
- 本sliceではstate-root overrideを `AGENT_ENV_HOME` のみに固定
- bilingual standalone product/design docs
- foundation revisionでdocs/unit/vetがpass

実装前に選択したbaseを更新し、正確なrevisionとbase branchを以下へ記録し、
前提基盤とbaseline harnessを確認する。

推奨はfoundationを先にmergeして`master`からbranchを切ること。意図的にstacked
PRにする場合はbase branch/base commitを明記し、stacked evidenceを
merged-master evidenceと表現しない。

開始 revision: `e37242a312c090c51430d519ea623e1bb41d2941`

開始 base branch: `feat/standalone-distribution`

## 目的 / 全体像

完了後、maintainerがversion tagを1つ作ると、以下6targetのstandalone
`agent-env` archiveを完全検証してGitHub Releaseへ公開できる。

    windows/amd64
    windows/arm64
    darwin/amd64
    darwin/arm64
    linux/amd64
    linux/arm64

release mechanicsの正は`repoctl`だけとする。

successful releaseとして受け入れるには、source identity、tag/version contract、
clean tree、6archive、checksum/manifest、release-check、same-source determinism、
source tree外のnative smoke、publication gate、英日doc、acceptance evidenceをすべて
直接証明する。

Bash/PowerShell/external tar/zip/sha256sum/CGO/separate release toolをrelease
pipeline要件にしない。packagingはGo標準libraryとrepository Go codeで実装する。

## 進捗

- [x] 2026-09-08: `e37242a` から作業ブランチを作成し、Go 1.27.1 の `repoctl check` が成功。

- [ ] exact starting base/revisionを記録しfoundation prerequisite確認
- [ ] baseline `repoctl check` / race suite
- [ ] tag/version/archive/release-manifest contractを英日docでfreeze
- [ ] strict Git release-source validation
- [ ] `repoctl release-build`
- [ ] 6target `CGO_ENABLED=0`
- [ ] normalized top-level directory付きzip/tar.gz
- [ ] tagged commit timestamp mtime
- [ ] deterministic `checksums.txt`
- [ ] versioned `release-manifest.json`
- [ ] `repoctl release-check`
- [ ] corruption/mismatch/traversal/symlink/path-leak negative fixture
- [ ] executable build info / bundled asset metadata static verify
- [ ] same-source/toolchain repeated binary/archive comparison
- [ ] spaces/non-ASCII state-root release binary test
- [ ] repository外/Go無しextracted artifact smoke
- [ ] Windows native smoke
- [ ] macOS native smoke
- [ ] Linux native smoke
- [ ] arm64 native/cross evidence区別
- [ ] tag-triggered GitHub Release workflow
- [ ] validation failureによるpublication gate実証
- [ ] README/architecture/portability/quality/security/roadmap英日更新
- [ ] capability prerequisite matrix
- [ ] final full harness
- [ ] final race suite
- [ ] final release-build/release-check
- [ ] final archive manual inspection
- [ ] acceptance evidence
- [ ] 英日Outcomes & Retrospective
- [ ] 英日ExecPlanをcompletedへ移動しlink/hash更新

checkboxは観測済み完了だけを表す。UTC date、revision、command/run、resultを記録する。

## 想定外の発見

- 2026-09-08: `0bf2d12` の完了記録は早すぎた。Git コマンドは root 引数を
  無視し、release-check はアーカイブを検査していなかった。入力の欠落と close
  エラーを見逃し、README.md を収録し、既存出力先を削除していた。既存の単体
  テストはリリース経路を検証していなかったため、直接検証するまで未完了とする。
- 2026-09-08: 日本語 Plan の開始 revision と進捗を翻訳せず hash だけ更新した
  記録があった。本文も修正した。
- 2026-09-08: Go 1.27.1 で `go build -trimpath` を試すと、debug/buildinfo に
  linker flags が残らなかった。実行時と静的検証で共通の ReleaseRecord を読み、
  VCS・platform・CGO 情報も独立に確認する。

現時点ではなし。

複数version tag、local/CI build metadata差、archive nondeterminism、timestamp precision、
gzip header、Windows path/mode、native runner制約、absolute path leak、premature publish、
antivirus、asset metadata mismatch等を記録する。digest assertionを弱めて隠さない。

## 判断の記録

- 判断: 未追跡ファイルも dirty とし、ignore 対象の出力は除外する。HEAD 上の
  有効なリリース tag が複数なら、要求と一致するものがあっても拒否する。
  バージョンの数値には先頭のゼロを認めない。
  理由: 隠れた入力と曖昧なリリース識別を避ける。
  日付/担当: 2026-09-08 / maintainers.
- 判断: ZIP は UTC の拡張 Unix 秒を保存し、1980-01-01 から
  2106-02-07 06:28:15 UTC までを扱う。TAR は正規化した USTAR、gzip は
  timestamp・元ファイル名・comment なしとする。3つの通常ファイルだけを
  同じバージョン付きディレクトリ配下に収録し、ディレクトリエントリは作らない。
  理由: commit の秒を保ち、表現できないメタデータを拒否する。
  日付/担当: 2026-09-08 / maintainers.
- 判断: README.txt はハーネス内の英日テキストから生成し、LICENSE は release
  commit から取得する。ソースやローカルパスを収録しない。
  理由: 決定的な導入説明と既存の MIT license を配布する。
  日付/担当: 2026-09-08 / maintainers.
- 判断: trimpath が linker flags を省くため、CLI の識別情報を ReleaseRecord に
  格納し、debug/buildinfo も確認する。assets は空配列とする。任意の Android
  helper は現在埋め込まれていない。
  理由: 未検証の manifest だけから識別や同梱物を主張しない。
  日付/担当: 2026-09-08 / maintainers.
- 判断: release workflow は Go 1.27.1 に固定し、ローカル成果物は実際の
  toolchain を記録する。preview・再現性検証には private clone の v0.1.0 を使う。
  公開 tag は maintainer が作成し、公開には意図的な tag push が必要となる。
  理由: 公開リリースを暗黙に作らず、リリース条件を検証する。
  日付/担当: 2026-09-08 / maintainers.

- 判断: Git tagを唯一のrelease version authorityとする。
  理由: duplicate VERSION sourceを避け、Gitからrelease identityを証明。
  日付/担当: 2026-09-08 / maintainers.

- 判断: tagは`v<MAJOR>.<MINOR>.<PATCH>`、requested versionは`v`除去後と一致。
  理由: 初期contractとして単純でmachine-parseable。
  日付/担当: 2026-09-08 / maintainers.

- 判断: release buildは`HEAD == selected tag commit`かつclean tree/indexを要求。
  理由: version表示とsource bytes不一致を防止。
  日付/担当: 2026-09-08 / maintainers.

- 判断: 毎releaseでWindows/macOS/Linux × amd64/arm64の6targetをbuild。
  理由: release targetを暗黙変更しない。
  日付/担当: 2026-09-08 / maintainers.

- 判断: archiveはversion/target付きtop-level directoryを1つ持つ。
  理由: extractで散らからずself-identifying。
  日付/担当: 2026-09-08 / maintainers.

- 判断: archive mtimeはtagged commit timestamp由来。
  理由: wall-clock nondeterminism除去。
  日付/担当: 2026-09-08 / maintainers.

- 判断: release construction/static validationは`repoctl`、GitHub Actionsはorchestration/publishのみ。
  理由: local/CIで同一release implementation。
  日付/担当: 2026-09-08 / maintainers.

- 判断: native smokeとcross-build evidenceを別扱い。
  理由: foreign build成功はnative execution proofではない。
  日付/担当: 2026-09-08 / maintainers.

- 判断: state-root overrideは本sliceで`AGENT_ENV_HOME`のみに固定。
  理由: foundation decisionを再設計せずreleaseで検証する。
  日付/担当: 2026-09-08 / maintainers.

- 判断: durable docs/ExecPlanは英日双方。
  理由: bilingual documentation policy。
  日付/担当: 2026-09-08 / maintainers.

## 成果と振り返り

未完了。

完了時に、end-to-end version、tag enforcement、6target artifact、archive normalization、
manifest schema、determinism、native smoke matrix、release workflow evidence、failure gate、
state-root/asset検証、doc更新、signing/package-manager follow-upを記録する。

## 背景と構成

実装前に読む:

- `AGENTS.md` / `.ja.md`
- `ARCHITECTURE.md` / `.ja.md`
- `docs/PLANS.md` / `.ja.md`
- standalone distribution英日product/design docs
- standalone-distribution foundation ExecPlan英日
- `PORTABILITY` / `QUALITY` / `SECURITY` / `roadmap`英日
- `internal/buildinfo`
- `internal/assets`
- state/path package
- CLI version
- `tools/repoctl`
- cross-build/native CI
- `.github/workflows/`

本Planではdevelopment build-info/version、generic bundled asset、tamper rejection、
bilingual standalone foundation、`AGENT_ENV_HOME`を再実装しない。release evidenceで
直接defectが出た場合のみ修正する。

## 作業計画

### Milestone 1 — Release contract固定

`v<MAJOR>.<MINOR>.<PATCH>`を正とし、release-buildはbuild effect前に
HEAD==tag、clean tree/index、tag syntax、requested version一致、source identityを検証。
no tag/malformed/mismatch/dirty/staged/untracked policy/ambiguous tagをnegative test。

### Milestone 2 — `repoctl release-build`

Goのみ。release identity first、commit timestamp、fixed matrix、CGO=0、stable build flag、
private staging、executable/LICENSE/README、6archive後にchecksum/manifest。Go標準の
zip/tar/gzip/SHA256/JSONを使用。

### Milestone 3 — Normalized archive / metadata

tar.gzはfixed order/path、uid/gid/name/mode/mtime/gzip headerをnormalize。zipもorder/path/
mode/timestampをnormalize。checksums/manifestはfinal bytesから生成。

### Milestone 4 — `repoctl release-check`

output set、6archive、checksums、manifest、version/tag/source、digest、path safety、member set、
symlink等禁止、executable digest/build info、asset metadata、license/readme、path leak、mtime/mode
をstatic verify。corruption/mismatch/traversal等negative fixture。

### Milestone 5 — Determinism / state root

同じcommit/tag/version/Goから2回release-build。executable/archive/checksum/manifest digest比較。
repository外・spaces/non-ASCII pathへextractし、`AGENT_ENV_HOME`も別pathにしてstate leak無しを検証。

### Milestone 6 — Native smoke

release archiveを入力にし、`go run`禁止。

    agent-env version --output json
    agent-env --help

必要ならcore doctor。Windows/macOS/Linux native。arm64はactual native runner時のみnative evidence。

### Milestone 7 — GitHub Release workflow

maintainer-pushed `v*` tag trigger。exact tag checkout、pinned Go、full harness、release identity、
release-build/check、native smoke待ち、validated bytesそのものをpublish。tag mutation、publish job rebuild、
validation前publishは禁止。

### Milestone 8 — Documentation

英日README/ARCHITECTURE/PORTABILITY/QUALITY/SECURITY/roadmap/standalone docs更新。
core/Git/Compose/Android/Flutterのcapability prerequisite matrix追加。

### Milestone 9 — Final evidence / archival

final repoctl check/race/release/determinism/native smoke/manual archive inspection/workflow IDsを記録。
全acceptanceをdirect evidence化し、英日Outcomes & Retrospective後にcompletedへ移動。

## 具体的な手順

1. foundation merge/確立
2. base branch/revision記録
3. `feat/standalone-release-finalization`
4. 英日active plan
5. baseline harness/race
6. contract freeze
7. strict Git validation
8. release-build
9. six-target build
10. normalized archive
11. checksum/manifest
12. release-check
13. negative fixtures
14. determinism
15. extracted state-root smoke
16. native smoke jobs
17. GitHub Release workflow
18. bilingual docs/matrix
19. final harness/race/release
20. representative manual archive inspection
21. acceptance/workflow evidence
22. bilingual retrospective
23. completed move/link/hash

## 検証と受け入れ

| ID | 必須動作 | 証拠 |
| --- | --- | --- |
| R1 | missing/malformed/ambiguous tagをbuild前に拒否 | Pending |
| R2 | HEAD == tag commit必須 | Pending |
| R3 | clean-tree/index policyを強制 | Pending |
| R4 | requested X.Y.Z == selected vX.Y.Z minus v | Pending |
| R5 | 6targetすべてCGO=0 build | Pending |
| R6 | archive name contract一致 | Pending |
| R7 | top-level dir + executable/LICENSE/READMEのみ | Pending |
| R8 | commit timestamp mtime + normalized metadata | Pending |
| R9 | checksumsが6archive bytes一致 | Pending |
| R10 | manifestがactual version/tag/source/toolchain/target/archive/executable/asset bytes一致 | Pending |
| R11 | valid setをforeign binary実行無しでrelease-check pass | Pending |
| R12 | corruption/mismatch/traversal/symlink/missing/unexpected拒否 | Pending |
| R13 | source/worktree/temp absolute path leak無し | Pending |
| R14 | same-source/toolchain digest比較 | Pending |
| R15 | repository外/Go無しでextracted version/help | Pending |
| R16 | AGENT_ENV_HOME spaces/non-ASCII + state leak無し | Pending |
| R17 | Windows native smoke | Pending |
| R18 | macOS native smoke | Pending |
| R19 | Linux native smoke | Pending |
| R20 | arm64 native/cross evidence正確区別 | Pending |
| R21 | maintainer-created tag起点、Git ref mutation無し | Pending |
| R22 | workflowがrepoctlへmechanics委譲 | Pending |
| R23 | validation/smoke failure時publish無し | Pending |
| R24 | publish bytesがvalidated bytesそのもの | Pending |
| R25 | 英日durable docsがfinal contract/prerequisiteを説明 | Pending |
| R26 | final repoctl/docs/translation/race pass | Pending |
| R27 | Windows/macOS/Linux代表archive manual inspection | Pending |
| R28 | 英日Outcomes/Retrospective/direct evidence完成 | Pending |
| R29 | 英日plan completed移動、links/hash同期 | Pending |

code/workflow存在だけではacceptanceではない。

## 冪等性と復旧

release identity validationはread-only。release-buildは明示staging/outputだけへwriteし、Git refs、target repo、
normal lease state/default state rootを変更しない。arbitrary output directoryを削除しない。partial setをcompleteに
見せるchecksum/manifestを残さない。release-checkはread-only。

same input mismatchは証拠記録前に上書きして隠さない。GitHub publicationはfinal side effectであり、source tagを
auto move/recreateしない。

## 成果物と注記

    dist/
      agent-env_vX.Y.Z_windows_amd64.zip
      agent-env_vX.Y.Z_windows_arm64.zip
      agent-env_vX.Y.Z_darwin_amd64.tar.gz
      agent-env_vX.Y.Z_darwin_arm64.tar.gz
      agent-env_vX.Y.Z_linux_amd64.tar.gz
      agent-env_vX.Y.Z_linux_arm64.tar.gz
      checksums.txt
      release-manifest.json

planへbase/start revision、test tag/version、source timestamp、Go toolchain、archive/executable/manifest digest、
repeat-build result、native smoke run IDs、final workflow run、native/cross matrixを記録。archiveはGit commitしない。

## インターフェースと依存

想定surface:

    repoctl release-build --version X.Y.Z --out <dir>
    repoctl release-check --dir <dir> --version X.Y.Z

責務: Git identity、target build、deterministic archive、checksum/manifest、static release-check、CI orchestration。
runtime lease/domain型へrelease manifest型を安易に統合しない。

release construction external toolはGo + Gitのみ。archive/checksum external utility不要。GitHub CLIはlocal
release-build dependencyではない。

Milestone 2前に解決する未確定事項:

1. end-to-end evidence用version
2. untracked file clean-tree policy
3. HEAD複数version tagの扱い
4. zip timestamp range/precision
5. tar/gzip metadata normalization
6. README.txt authority
7. foreign executable buildinfo static inspection
8. release Go patch version authority
9. arm64 native runner availability
10. actual public releaseまで本Planで実証するかpre-publication gateまでにするか

public behavior固定前にDecision Logへ記録する。
