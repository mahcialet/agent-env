---
status: completed
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/completed/standalone-release-finalization.md
source_sha256: 5873102e5335acfc37a10266b9d2635dcb842923ee2bbb2bb82e3be001dd4868
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

- [x] 2026-09-08: 開始 base/revision と前提基盤を確認し、Go 1.27.1 の baseline
  check が成功。race は開始時に実行せず、今回の継続作業中に成功した。
- [x] 2026-09-08: 英日のタグ・バージョン・アーカイブ・schema 1 契約を整備。
- [x] 2026-09-08: 厳密な Git 検証、release-build/check、CGO=0 の6ターゲット、
  正規化した ZIP/tar.gz と commit timestamp を `d628098` の実配布候補で再検証。
- [x] 2026-09-08: checksums と manifest を生成し、全バイナリの識別情報と
  空の assets 配列を静的検証した。
- [x] 2026-09-08: アーカイブの不正入力テストと、実候補の17件の改ざん・保存
  テストが成功。2回のビルドで8ファイルのバイト一致も確認した。
- [x] 2026-09-08: Linux の展開済み version/help/list が、ソース外・空の PATH・
  日本語と空白を含む状態保存先で成功した。
- [x] 2026-09-08: Windows/macOS/Linux の native smoke job と tag workflow を追加。
  最終preview 34190701402で各OSの実行も成功した。
- [x] 2026-09-08: publication gate のテストで依存欠落・無条件公開・成果物名の
  不一致・再ビルド検証の欠落を拒否した。
- [x] 2026-09-08: README/architecture/portability/quality/security/roadmap を英日で
  更新し、前提ツールの比較表を追加。docs-check が成功した。
- [x] 2026-09-08: Windows/macOS/Linux の代表アーカイブを独立の Python
  zipfile/tarfile で確認した。詳細は成果物と注記に記録。
- [x] 2026-09-08: preview 34190096757 で linux/amd64・windows/amd64・darwin/arm64 のnativeが成功。他はcross/staticのみ。
- [x] 2026-09-08: Verify 34190096727 の12 job、preview 34190096757 の4 job、ローカル最終raceが成功。
- [x] 2026-09-08: R1–R29 の証拠を整理。英日版のarchivalも完了した。
- [x] 2026-09-08: 最終native/harness成功後に英日の成果と振り返りを完成した。
- [x] 2026-09-08: 両子Planをcompletedへ移動し、参照リンク・翻訳パス・hashを更新した。

チェックは観測済みの完了のみを示す。実装の存在と native 実行証拠を区別する。

## 想定外の発見

- 2026-09-08: 最初の native 成功後、ZIP のローカルヘッダーだけを改ざんすると、
  中央側の安全な名前が走査パス・絶対パスを隠し、検査を通ることを再現した。
  回帰テストは修正前に失敗。再圧縮せずにローカル・中央の名前、metadata、offset、
  descriptor を照合する修正を追加した。最終修正後のpreview 34190701402とVerify 34190701428も成功した。

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

2026-09-08 完了。最終実装 `641cb49` でローカル harness・実候補の負例・race、
Verify 34190701428（12 job）、Release preview 34190701402（buildと3つのnative
smoke）が成功した。

private clone の v0.1.0 候補で、厳密なtag/tree/version/commit検証、commit固定の
checkout、6つのCGO不要アーカイブ、metadata正規化、checksums、schema 1 manifest、
実行ファイルの静的識別を検証した。2回のビルドでarchive/executable/checksum/manifest
のバイトが一致した。nativeはLinux/amd64、Windows/amd64、macOS/arm64を実証し、
他の3組はcross-buildと静的検査のみとする。各native runnerでUnicodeの状態保存先と
空PATHでの実行が成功。同梱runtime assetsは現在空配列で、Android/Flutterツールを
暗黙に同梱・初期化しない。

tag workflowはharness/race、厳密なsourceのbuild、静的検査、同じtagの再生成比較、
全native smokeを公開の前提とする。previewでupload済み候補を実行し、gateの回帰
テストで迂回を拒否した。公開tagやGitHub Releaseは作成していない。実際の公開は
maintainerによる意図的なtag pushで行う。署名、notarization、package manager、
SBOM、attestationはこのPlanの範囲外として残る。

当初のharness成功はリリース完了の根拠にならなかった。配布物を検査するテストが
なかったためである。実際の負例で検査不足とZIPのlocal/central名の不一致を発見した。
別担当のレビューでignored source混入と最終source識別の未比較も発見し、現在は
防止処理と回帰証拠を備える。英日進捗の本文もhashとともに修正した。親の受け入れは
別途照合し、asset inventory・stress・より広いstate/prerequisite条件はactiveで残す。

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
| R1 | missing/malformed/ambiguous tagをbuild前に拒否 | source/usage の負例テストが成功。欠落・不正・曖昧な tag はビルド前に拒否。 |
| R2 | HEAD == tag commit必須 | HEAD 移動・tag の commit 解決・annotated tag のテストが成功。専用 snapshot はローカル変更を含めない。 |
| R3 | clean-tree/index policyを強制 | dirty/staged/untracked と専用 checkout の回帰テストが成功。 |
| R4 | requested X.Y.Z == selected vX.Y.Z minus v | 要求バージョン不一致・数値の先頭ゼロを拒否するテストが成功。 |
| R5 | 6targetすべてCGO=0 build | 641cb49 のローカル検証と preview 34190701402 で6ターゲットをビルドし、CGO=0 を静的検証。 |
| R6 | archive name contract一致 | 6つの契約通りの名前をローカル・preview・独立検査で確認。 |
| R7 | top-level dir + executable/LICENSE/READMEのみ | roundtrip・不正memberテストと独立検査で、同じディレクトリ配下の3通常ファイルを確認。 |
| R8 | commit timestamp mtime + normalized metadata | テストと独立検査で commit 秒、ZIP 拡張timestamp、mode、TAR/gzip 正規化を確認。 |
| R9 | checksumsが6archive bytes一致 | 全checksumを検証し、ローカルと preview 34190701402 で2組のバイト一致を確認。 |
| R10 | manifestがactual version/tag/source/toolchain/target/archive/executable/asset bytes一致 | 6バイナリを静的検査。実候補テストで manifest/target/digest/toolchain/架空asset の不一致を拒否。 |
| R11 | valid setをforeign binary実行無しでrelease-check pass | 全ターゲットの静的検査が成功。外国OSの実行ファイルを起動せず検証する。 |
| R12 | corruption/mismatch/traversal/symlink/missing/unexpected拒否 | 不正アーカイブのテストと実候補17 subtest が成功。hash再計算後の不正version/license/readmeも拒否。 |
| R13 | source/worktree/temp absolute path leak無し | private source pathを加えたバイナリはhashを再計算しても拒否。検査範囲は既知のパスと管理下のmetadataで、万能の秘密検出ではない。 |
| R14 | same-source/toolchain digest比較 | 641cb49・Go 1.27.1の2回のビルドで8ファイルが一致。preview 34190701402 も成功。実tag workflowもrepeatを要求。 |
| R15 | repository外/Go無しでextracted version/help | preview 34190701402 の3OSで、ソース外・空PATHの version/help/list が成功。 |
| R16 | AGENT_ENV_HOME spaces/non-ASCII + state leak無し | 同じnative smokeでUnicode/空白のAGENT_ENV_HOME、version/helpの無書込、配布先/作業先/default homeへの漏洩なしを確認。 |
| R17 | Windows native smoke | preview 34190701402 の Windows/amd64 native smoke が成功。 |
| R18 | macOS native smoke | preview 34190701402 の Darwin/arm64 native smoke が成功。 |
| R19 | Linux native smoke | preview 34190701402 とローカルの Linux/amd64 native smoke が成功。 |
| R20 | arm64 native/cross evidence正確区別 | Windows/arm64・Darwin/amd64・Linux/arm64 はcross-build/静的検査のみ。他の3ターゲットにはnative証拠あり。 |
| R21 | maintainer-created tag起点、Git ref mutation無し | maintainerのv* tag pushで起動し厳密なtagを検証。tag作成は使い捨てclone内のみで、callerのrefs保護をテスト。 |
| R22 | workflowがrepoctlへmechanics委譲 | 両workflowはbuild/check/repeat/smokeをrepoctlへ委ね、YAMLやシェルにpackaging処理を実装しない。 |
| R23 | validation/smoke failure時publish無し | publication gateと4つの迂回負例が成功。build/smoke依存とrepeatを必須にし、無条件公開や失敗無視を拒否。公開releaseは作成していない。 |
| R24 | publish bytesがvalidated bytesそのもの | preview native jobはupload済み候補をdownloadし、再ビルドしない。gateテストは同じartifact名を要求しpublishのrunを拒否。 |
| R25 | 英日durable docsがfinal contract/prerequisiteを説明 | 英日の文書・前提ツール比較表を更新し、docs-check成功。 |
| R26 | final repoctl/docs/translation/race pass | Verify 34190701428 の12 job（native harness、Linux race、Docker integration）が成功。ローカル最終raceも成功。 |
| R27 | Windows/macOS/Linux代表archive manual inspection | 641cb49 の代表3アーカイブを Python zipfile/tarfile で独立検査し、下記へ記録。 |
| R28 | 英日Outcomes/Retrospective/direct evidence完成 | 2026-09-08: 最終ローカル/native証拠と公開/platformの限界を上記に記録。 |
| R29 | 英日plan completed移動、links/hash同期 | 2026-09-08: 両子Planをarchivalし、親/README/quality/portability/roadmapリンクと翻訳metadataを更新。 |

code/workflow存在だけではacceptanceではない。

## 冪等性と復旧

release identity validationはread-only。release-buildは明示staging/outputだけへwriteし、Git refs、target repo、
normal lease state/default state rootを変更しない。arbitrary output directoryを削除しない。partial setをcompleteに
見せるchecksum/manifestを残さない。release-checkはread-only。

same input mismatchは証拠記録前に上書きして隠さない。GitHub publicationはfinal side effectであり、source tagを
auto move/recreateしない。

## 成果物と注記

最終コード checkpoint: `641cb49d91972b40fff352c14945697d876dad6d`。

- `go run ./tools/repoctl release-verify --out dist/verified-641cb49` が成功。
  6ターゲットを2回生成し、8ファイルのバイト一致と Linux のnativeを確認。
- `AGENT_ENV_RELEASE_CANDIDATE=../../dist/verified-641cb49 go test ./tools/repoctl
  -run 'TestReleaseCandidate|TestReleaseArchive|TestReleasePublicationGate' -count=1`
  が成功。修正前に失敗した ZIP の local 名改ざんも拒否した。
- 最終ローカル `go test -race ./...` が成功。別担当によるアーカイブの独立
  レビューで追加の重大な不具合は見つからず、対象テストも成功した。
- source timestamp は 1788845253、toolchain は go1.27.1、private test tag は v0.1.0。
  manifest SHA-256 は `c82f7917ecbd08ce0ee719003a3a31c0f9c82e07e2266dab1e420e193a097830`。
- Windows/macOS/Linux amd64 を zipfile/tarfile で再確認。正しい3 member、
  正確な tag 対象秒（ZIP拡張fieldを含む）、0644/0755、TAR uid/gid 0 を確認した。
- 最終 hosted run 34190701402（Release preview）と34190701428（Verify）は
  全job成功。前回のnative証拠も以下へ保持する。

| ターゲット | アーカイブ SHA-256 | 実行ファイル SHA-256 |
| --- | --- | --- |
| windows/amd64 | `7e6903883dacca78756ec5f38542e5665605972324433f6fc2e1e4a56effa71a` | `779b8e44290c9465508e998b92c02e71f9ec9198ccb1e7a2ba1544857f0a25fe` |
| windows/arm64 | `c8b287df2a9f97a83f3d3f9380c54a5195f6716b16776df25ed81ae4d36c2d05` | `d9f3909083087fe6fdf08bcb6952c2af4f19ff4e7ae9080dce322458384a23e1` |
| darwin/amd64 | `3f7a715e6e6341849009905f28654fbbfc1ae566d2a3940d80f6e155a9621abf` | `c11d274bfa4f1115c3ea91cac820e5ddaddfbece282091b11db8610023bba683` |
| darwin/arm64 | `a336bde3e9aba609a48618e8e3226e85719f09d889e1763aa5040dc51f4be850` | `19634f3cf0f975b038a48e0347d4e863487088f64a5169f4ab42646673c21179` |
| linux/amd64 | `e281cb2840024a364e2b5d2933162a7b2c9f4e3c1331e8dd232e21be2577cd3b` | `5679e746800e6a0e5bd801ebd17e1562bb46dd35414e66571cef5c7ee831c3f7` |
| linux/arm64 | `732ca440fd9f7202c2f984770b68e1b545ddfb20fbe70af794a1fc56407310b6` | `3411d2c2fc75d0e99205ad94c10629b0d165370075235557f14655a3dffad5ff` |


2026-09-08、`d6280988441537418d70925174cfa58374efea9b` のローカル証拠:

- Go 1.27.1 で `go run ./tools/repoctl check` と `go test -race ./...` が
  実装中の checkpoint で成功。workflow gate の回帰テスト追加後も harness が成功。
- `go run ./tools/repoctl release-verify --out dist/verified-d628098` が成功。
  6ターゲットを2回ビルドし、8ファイルのバイト一致と Linux/amd64 の実行を確認。
- `AGENT_ENV_RELEASE_CANDIDATE=../../dist/verified-d628098 go test ./tools/repoctl
  -run TestReleaseCandidate -count=1 -v` の17 subtest が成功。環境変数のパスは
  Go test の package directory 基準で、CI でも同じ方法で渡す。
- private test tag は v0.1.0、source timestamp は 1788844585。
  manifest SHA-256 は `d69cfbcd2ad5e5251654b460a23cd5317ffd67790edc945a9ec369198093a41d`。
- Python zipfile/tarfile で Windows/macOS/Linux amd64 の代表アーカイブを確認。
  バージョン付きディレクトリ配下は LICENSE、README.txt、実行ファイルのみ。
  mode は 0644/0755、TAR の uid/gid は0、mtime は source 秒と一致。
  今回は奇数秒のため ZIP の DOS 秒は1秒切り下がるが、拡張 Unix timestamp は
  正確な source 秒を保持している。
- ソース外・空の PATH・日本語と空白を含むパスで version/help/list が成功し、
  list は override した状態保存先だけを作成した。
- 独立レビューで ignored source 混入、最後の source identity 未比較、tag 用
  repeat gate 欠落を発見し、候補 checkpoint 前に修正した。専用 checkout の
  回帰テストで隠れた変更を排除し、最後の識別比較と tag workflow の再ビルドを追加。
- Hosted run は Release preview 34190096757 と Verify 34190096727。両runとも全job成功。
  公開 tag/release は作成していない。公開は maintainer の意図的な操作で行う。

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

初期の10項目は判断の記録と証拠で解決した。private test v0.1.0、未追跡もdirty、
HEADの複数有効tagは拒否、ZIP拡張秒と扱える範囲、USTAR/gzipの正規化、生成する
英日README.txt、debug/buildinfoとReleaseRecord、CIのGo 1.27.1固定、native arm64は
macOSのみ、公開はmaintainerの意図的なtag操作まで行わず事前gateを検証する方針である。
