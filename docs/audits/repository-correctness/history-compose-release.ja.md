---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/history-compose-release.md
source_sha256: e0a9505c06b0b35aebbe6787b4b5ae33618c4755b6d0897fd2dc64f7d3536846
---

# Compose と standalone release の過去レビュー記録

[English](history-compose-release.md) · [監査索引](index.ja.md) ·
[作業の実行基準](../../exec-plans/completed/repository-correctness-audit.ja.md)

検査対象の実装は `031869c8b9073b8e23bc17fbc55243666a52f557` に固定する。
本書は Phase A の証拠であり、修正の採否決定ではない。製品・テストのファイルは
変更していない。元の Plan に重大度ラベルは記録されていないため、以下の影響は
記録された不具合の結果を表し、元の重大度を推定で補うものではない。
見逃し理由は、元の記録が明示している場合を除き、今回の振り返りによる推論である。

## 出典と表の読み方

| 記号 | 過去の出典 | 指摘の所在 |
| --- | --- | --- |
| P | [Podman 機能](../../exec-plans/completed/compose-provider-podman.ja.md) | 独立レビューの番号付き7件、実環境での失敗、cleanup proof の受け入れ条件。 |
| PR | [Podman レビュー](../../exec-plans/completed/compose-provider-podman-review.ja.md) | PR 8 discussion `3957283152`（UDP）、`3957283162`（inventory）。 |
| D | [Distribution](../../exec-plans/completed/standalone-distribution.ja.md) | 作業再開時の回帰テストと Windows 実行の失敗。 |
| DR | [Distribution レビュー](../../exec-plans/completed/standalone-distribution-review.ja.md) | PR 6 の11 Thread。新規修正7件、対応済み4件。元の個別 ID 一覧は出典にない。 |
| F | [Release finalization](../../exec-plans/completed/standalone-release-finalization.ja.md) | `0bf2d12` の監査、独立レビュー、ZIP 変異、build-info の実験。 |
| RR | [Release 出力レビュー](../../exec-plans/completed/standalone-release-review.ja.md) | PR 7 `PRRT_kwDOURHsR86gHJ1z`、discussion `3954761466`。 |
| FR | [Release filter レビュー](../../exec-plans/completed/standalone-release-filter-review.ja.md) | PR 7 `PRRT_kwDOURHsR86gHb58`、discussion `3954872828`。 |
| VR | [Verify 入口レビュー](../../exec-plans/completed/standalone-verify-review.ja.md) | PR 6 discussion `3956143329`。 |

以下の項目は、この revision にも適用される。製品でまだ使用していない asset 基盤を
保護する項目も含む。製品の同梱 asset 一覧は空であり、CLI の利用経路を架空に補って
end-to-end 検証と扱わない。`HCR-*` は過去の記録 ID であり、新規 `AUDIT-*` 指摘ではない。

表内のパスはリポジトリルートからの相対パスである。テスト名は現在の回帰テストの
所在を示し、特記しない限り過去の修正時にも記録されたテストである。
`Yes/provider`、`Yes/app`、`Yes/API`、`Yes/command` は、その実際の入口に到達して
検証することを表す。必ずしも CLI ではない。`Yes/helper` は局所的な不変条件を検証
するが、そのテストは全体の入口を**通らない**。`Partial` または `No` は、過去の
不変条件全体を直接実証する回帰テストが不足することを表す。
現在の成功とコード確認は変異テストではない。今回、過去の欠陥を再導入してはいない。
再導入時に失敗するという評価は、見えている判定条件と現在の呼出経路に限定する。

`検出→最早` は実行 Plan の S0–S9 を使用する。別レビュアーによらない実装・再開時の
検査は S7、独立または PR レビューは S8、実際の provider・OS による失敗は S5 とした。
元の Plan が発見の手順をまとめている場合があるため、一部の分類は概算である。
見逃し分析には、利用できた具体的な早期検出機会と、それを逃した理由を記した。
明記した deferred を除き、防止策の状態は existing（既存）である。
この監査によって追加した防止策はまだない。

## Podman の対応関係

| ID / 出典 | 不変条件、過去の欠陥、影響 | 現在の実装 → 回帰テスト、検証・全体入口の状態 |
| --- | --- | --- |
| HCR-P01 / P 指摘1 | inventory は orphan だけの provider も検出する。残存 lease 行だけの列挙で engine 全体が消えた。 | `internal/app/reconciliation_inventory.go` provider 集合 → `internal/app/compose_provider_test.go:TestInventoryDiscoversOrphansOutsideRecordedProviders`。Yes/app。記録なし・Docker・Podman の各状態で両 engine を呼ぶ。 |
| HCR-P02 / P 指摘2 | provider の実行拡張はすべて policy を通すか拒否する。root/service 限定の検査が network/secret の入れ子を逃した。 | `internal/runtime/compose/podman_normalize.go:rejectPodmanExtensions` を Render から呼ぶ → `podman_test.go:TestPodmanRejectsNestedExecutionExtensions` と `TestPodmanRenderRejectsProviderSpecificHostAccess`。network のケースは Yes/provider、helper は入れ子5か所を検証する。 |
| HCR-P03 / P 指摘3 | provider 固有の mount/network mode に共通 host policy を迂回させない。`glob`、`ns:` 等が Docker 型の検査を逃した。 | 同じ正規化を `podmanClient.Render` から実行 → `TestPodmanRenderRejectsProviderSpecificHostAccess`。Yes/provider。config 呼出回数で正規化への到達を確認し、不正な13形態を runtime の作用前に拒否する。 |
| HCR-P04 / P 指摘4 | 残存 resource があれば存在する。匿名 volume の証拠を `Exists` 算出後に追加して、不在と誤判定した。 | `internal/runtime/compose/podman.go:Inspect` → `TestPodmanInspectRetainedAnonymousVolumeExists`。Yes/provider。engine の project 一覧が空でも、残存 volume 1件から存在と具体的 resource を検証する。 |
| HCR-P05 / P 指摘5 | snapshot を移動しても範囲内の env/secret/config ファイル参照を保つ。相対パスが一時ディレクトリで壊れた。 | `podman_normalize.go:validatedPodmanFile` と `podman.go` のディレクトリ検査 → `TestPodmanRelativeFileReferencesSurviveSnapshotRelocation`、`TestPodmanRejectsInitialFileDirectoryMismatchBeforeProvider`。Yes/helper/adapter。実際の Up 全体ではない。絶対かつ読めるパス、範囲外・ディレクトリ・欠落の拒否、provider 呼出ゼロを検証する。 |
| HCR-P06 / P 指摘6 | native service label で Podman の帰属を証明する。Docker 互換 label だけを信用した。 | `podman_normalize.go:normalizePodmanInspection` を native adapter が利用 → `TestPodmanNativeServiceOwnership`。Yes/helper。native の欠落・競合を拒否し、native だけなら受理する。provider inventory にも競合 label テストがある（P13）。 |
| HCR-P07 / P 指摘7 | Render 済み environment を Up で host secret から再解決しない。null map と値なし list が後から host に依存した。 | `normalizePodmanConfig` → `TestPodmanEnvironmentRequiresExplicitPassThroughValues` と Render の拒否テスト。拒否は Yes/provider。helper は明示的空値・リテラルの保存と error の秘密非露出も検証する。 |
| HCR-P08 / P 実環境失敗 | provider 失敗は有用かつ redaction 済みの native 診断を残す。共通 wrapper が stderr を捨て、Docker 失敗と誤表示した。 | `podman.go:podmanCommandError`、adapter → `TestPodmanProviderFailurePreservesRedactedNativeDiagnostic`。provider・exit・message・redaction は Yes/adapter。8 KiB 診断上限は **Partial**。長い診断の回帰テストがないことを元の Plan も明記する。 |
| HCR-P09 / P 実環境失敗 | 保存済み設定を変更せず、動的 port の意図を provider 用に変換する。実際の podman-compose が数値・文字列の published zero を拒否した。 | `podman_normalize.go:escapePodmanSnapshot` を Up が利用 → `TestPodmanUpUsesDynamicPortWithoutChangingCanonicalSnapshot`。Yes/provider。一時ファイルとして受け取る bytes、数値・文字列、scope 項目、元の snapshot を検証する。native CLI fixture に過去の実環境証拠がある。 |
| HCR-P10 / P cleanup milestone | Down 前に匿名 attachment proof を永続化し、container 消滅後の中断でも proof を残す。 | `internal/app/lifecycle.go` の cleanup evidence と Podman Down → `compose_provider_test.go:TestCleanupRetainsProofAfterContainerDisappears`、`TestCleanupProofWriteFailurePreventsDown`。Yes/app。Destroy 2回、quarantine/released、保存済み proof、Save 失敗時の作用ゼロを検証する。重要な予防作業だが、番号付き7件の一つではない。 |
| HCR-P11 / P fixture 失敗 | 実環境検証の失敗時も復旧用 identity を残し、lease cleanup が不確実な間は入力を削除しない。 | `internal/cli/podman_integration_test.go` の fixture cleanup/recovery。**Partial**。実 fixture に復旧方針はあるが、fixture 自体の失敗を独立に注入する回帰テストは今回見つからなかった。テスト基盤の話であり、現在の製品欠陥を確認したわけではない。 |
| HCR-P12 / PR UDP | TCP 接続性で UDP endpoint の存在・readiness を判定しない。remote UDP mapping が TCP dial 失敗で消えた。 | `podman.go:Inspect` の protocol filter → `podman_review_test.go:TestPodmanRemoteInspectPreservesUDPAndChecksTCP`。Yes/provider。実際の local UDP/TCP socket と remote 型 identity を使い、UDP のみ・混在到達可・混在到達不可を検証する。実 Podman Machine forwarding の証拠ではない。 |
| HCR-P13 / PR inventory | engine-only Doctor と Inventory の両方が Compose frontend なしで動く。共有 Docker traversal が依然 Compose ls を呼んだ。 | `podman.go:Inventory`、`inventory.go` の label traversal → `podman_review_test.go:TestPodmanInventoryWithoutComposeDiscoversNativeOrphans`。Yes/provider dispatcher。Doctor 後に InventoryFor を実行し、frontend 欠落・古いパス、3種類、競合、部分失敗を検証する。 |

## Podman の見逃し分析

| ID | 検出→最早、見逃し分類 | 早期機会、足りなかった判定、防止策、再発先 |
| --- | --- | --- |
| P01 | S8→S4、COMPOSITION_GAP、NEGATIVE_FIXTURE_GAP | engine テストは自 engine だけを知り、残存行ゼロの app fixture がなかった。既存 provider-union テストが S4 の防止策。app inventory の kind/provider filter を調べる。P13 は関連する結合問題だが根本原因は同一ではない。 |
| P02 | S8→S2、NEGATIVE_FIXTURE_GAP | 実行拡張の拒否は既知だったが、fixture が既に検査する階層に偏った。既存の再帰的変異 matrix で S2、Render テストで helper の切断を防ぐ。config/Android/browser の入れ子 validator にも同じ観点を適用する。 |
| P03 | S8→S3、ORACLE_COUPLING、COMPOSITION_GAP | 共通 policy は指定済みでも、Docker 正規化済み fixture が raw Podman の違いを省いた。既存 raw Render の拒否が S3 の防止策。adapter ごとに dialect の fixture が必要で、万能な syntax validator は根拠不足。 |
| P04 | S8→S2、ORACLE_COUPLING | 見える container を持つ fixture が、途中の resource 集合を使う仮定を共有した。既存 retained-only Inspect は S3 で最終集合を検証する。最小の局所テストなら S2 で防げた。状態算出後に cleanup evidence を追加する observer を探す。 |
| P05 | S8→S3、COMPOSITION_GAP | 正規化 fixture に snapshot の基準ディレクトリ移動がなかった。既存ファイル参照・adapter 検査は S3 の防止策。実際の Up を通す移動テストは今後の補強であり、実施済みとはしない。他 runtime の staging 移動も別途確認する。 |
| P06 | S8→S2、ORACLE_COUPLING、NEGATIVE_FIXTURE_GAP | 便利な Docker 互換 label の正例が native のみ・欠落・競合を省いた。既存3分類 helper matrix が S2 の防止策。破壊操作対象の resource 種類ごとに identity label の組合せを確認する。 |
| P07 | S8→S2、NEGATIVE_FIXTURE_GAP、COMPOSITION_GAP | 値を明示した fixture が Render→Up の ambient lookup を隠した。既存 bare/null/empty/literal と raw Render で S2/S3 に検出する。ambient state の再発例に release filter（R17）がある。 |
| P08 | S5→S3、FAILURE_INJECTION_GAP | 実 provider 前でもコマンド失敗を注入できたが、成功 fixture に有用な診断の判定がなかった。既存 redaction 失敗テストが S3 の防止策。長文上限は境界監査まで **deferred**。現在の上限が破られたとはしない。 |
| P09 | S5→S5、NATIVE_EVIDENCE_GAP、ORACLE_COUPLING | fake が Docker の port-zero 意味論を仮定し、対応版 podman-compose の実行が最初の信頼できる判定手段だった。既存 native CLI 共存と private-byte Up テストが S5/S3 の防止策。cross-build で代用できない。 |
| P10 | S7→S4、FAILURE_INJECTION_GAP | 通常の Down 成功では、破壊作用の部分成功後に再試行できることを示せない。既存 app の失敗注入で Save-before-Down と proof 保存を S4 に検証する。他 runtime にも作用・永続化の順序を適用し、共通 helper 化は現在の再発証拠で判断する。 |
| P11 | S5→S4、FAILURE_INJECTION_GAP | 成功前提の fixture cleanup が Create の JSON を仮定した。registry fallback とディレクトリ保持は既存の防止策。独立失敗注入は監査の採否判断まで **deferred**。他 native fixture の失敗 cleanup も確認する。 |
| P12 | S8→S3、NEGATIVE_FIXTURE_GAP、COMPOSITION_GAP | remote 観測に protocol 混在 fixture がなく、TCP helper 成功では UDP を検証できなかった。既存 Inspect socket matrix が S3 の防止策。UDP/TCP の取り違えを探すが、UDP dial を readiness 証明とはしない。 |
| P13 | S8→S3、HELPER_ONLY、COMPOSITION_GAP | 元の Doctor 回帰は通ったが、その後の Inventory は別依存を呼んだ。既存 Doctor→InventoryFor が S3 の防止策。releaseVersion→release-verify（R15/R18）に明確な再発がある。helper の追加だけでなく関連する公開入口を対にして検証する。 |

## Distribution と asset の対応関係

| ID / 出典 | 不変条件、過去の欠陥、影響 | 現在の実装 → 回帰テスト、検証・全体入口の状態 |
| --- | --- | --- |
| HCR-A01 / D、DR 対応済み | 同一 bytes の同時 materialize は正当な mkdir 勝者を許容する。EEXIST で子 process が失敗した。 | `internal/assets/assets.go:safeMkdirAll` を Materialize が利用 → `assets_test.go:TestMaterializeAcrossProcesses`。Yes/API。独立12 process、30 root、各 round 3回、正確な bytes と staging 残存なしを検証する。 |
| HCR-A02 / D | 絶対 state override は既定 HOME なしで動く。home 検出が早すぎた。 | `internal/paths/paths.go:Resolve` → `paths_test.go:TestResolveOverrideWithoutUserHome`。Yes/API。CLI の検証とはしない。 |
| HCR-A03 / D | UI helper install の staging は失敗時も所有 runtime state 内に置く。OS temp が state-root 契約から外れた。 | `internal/runtime/android/ui.go` helper install → `ui_test.go:TestUIHelperStagingStaysInOwnedRuntimeAndCleansUp`。Yes/provider。install 成功・失敗で runtime 内 APK と cleanup を観測する。 |
| HCR-A04 / D Windows | checkout の改行変換で embedded provenance を変えない。Windows autocrlf が17 bytes を18にした。 | `.gitattributes` の fixture `-text` → `assets_test.go:TestEmbeddedFixture`。コンパイル済み bytes/hash は Yes/API。checkout 変換の検証自体には native Windows が必要。 |
| HCR-A05 / D Windows | immutable publication は同時書込の勝者を保護する。Windows の置換で sharing/access failure が起きた。 | `internal/assets/publish_windows.go:publishAsset` → `publish_windows_test.go:TestPublishAssetPreservesWindowsWinner` と process stress。Yes/helper/API。native Windows 限定。 |
| HCR-A06 / D Windows | 一時的な read sharing/lock failure は上限内で再試行できるが、恒久的 error は失敗のままとする。no-replace 修正後も read が失敗した。 | `publish_windows.go:readAsset` → `TestReadAssetWindowsSharingConflict`、`TestReadAssetWindowsDoesNotRetryMissingFile`。Yes/helper。保持した native handle で再試行・解放・上限付き失敗を検証する。Linux では未再実行。 |
| HCR-A07 / DR | asset 論理名は全 host で portable とする。不正名25件が通った。 | `assets.go:portableName` を Describe が利用 → `assets_test.go:TestDescribePortableLogicalNames`。Yes/API。Windows 予約名・区切り・末尾形式を全 host で検証する。 |
| HCR-A08 / DR | cached asset は read 前と read 中に上限を設ける。大きな cache をサイズ拒否前に読んだ。 | `assets.go:readAssetOnce` を Materialize が利用 → `TestMaterializeRejectsOversizedCacheBeforeReading`、`TestReadAssetRejectsSizeMismatch`。Yes/API と helper。32 MiB cache と open 後のサイズ不一致。 |
| HCR-A09 / DR | linker identity 未設定なら実 embedded VCS を使い、明示値は優先する。clean Git build で commit identity が消えた。 | `internal/buildinfo/buildinfo.go:Current` → `buildinfo_test.go:TestCurrentVCSFallback`、`vcs_test.go:TestCurrentEmbeddedVCSIdentity`。Yes/API と実 build executable。clean/dirty identity を検証する。 |
| HCR-A10 / DR 対応済み | root/ancestor symlink に materialize を転送させない。古い Thread は既に検証済みだった。 | `assets.go:safeMkdirAll` → `TestMaterializeRejectsNonDirectoryAndSymlinkEntries`、`TestMaterializeRejectsSymlinkedAncestorAndPortableNames`。Yes/API。全パス位置と外部ディレクトリが空のままであることを検証。symlink を使えない環境は skip であり pass ではない。 |

| ID | 検出→最早、見逃し分類 | 早期機会、防止策、再発先 |
| --- | --- | --- |
| A01 | S7→S4、CONCURRENCY_GAP | 単一 process・冪等性検査では最初の同時生成がなかった。既存独立 process stress は S4 の防止策だが、確率的 scheduling で全 race の証明ではない。他の mkdir/cache も探す。 |
| A02 | S7→S2、NEGATIVE_FIXTURE_GAP | 絶対 override のテストにも HOME があった。既存 no-home fixture が S2 の防止策。CLI 初期化の遅延 prerequisite 順序にも同種の問題がある。 |
| A03 | S7→S3、COMPOSITION_GAP | 成功判定は subprocess の APK パスや error cleanup を観測しなかった。既存 runner 観測が S3 の防止策。他の staged helper にも個別に所有パス確認を適用する。 |
| A04 | S5→S5、NATIVE_EVIDENCE_GAP | Linux bytes/cross-build は Windows checkout 設定を実行しない。固定 hash と native CI が S5 の防止策。限定 attribute で無関係な改行変換変更を避ける。 |
| A05 | S5→S5、NATIVE_EVIDENCE_GAP、CONCURRENCY_GAP | Unix rename が native 置換制約を隠した。既存 native winner と同一 stress が S5 の防止策。他 publication の filesystem 意味論を同一視しない。 |
| A06 | S5→S5、NATIVE_EVIDENCE_GAP、FAILURE_INJECTION_GAP | 最初の native 修正は publication を扱い、競合者が保持する read を扱わなかった。既存 held-handle は S5 の防止策で、permission/missing は即時失敗を保つ。後の process cleanup にも sharing guard が必要になったが、全操作に同じ retry 方針を推定しない。 |
| A07 | S8→S2、NEGATIVE_FIXTURE_GAP | portability 要件に対して traversal 中心の名前検査しかなかった。既存 cross-host 不正名 matrix が S2 の防止策。他の論理名にも Windows alias/区切りを確認する。 |
| A08 | S8→S2、BOUNDARY_GAP | 小さい破損 fixture では拒否前の無制限 allocation が見えなかった。既存の大きな cache/open 後不一致が S2 の防止策。他の reader にも事前・stream 中の上限を確認する。 |
| A09 | S8→S3、HELPER_ONLY、COMPOSITION_GAP | 仮の build field では実 Go VCS embedding を証明できなかった。実 build/run が S3 の防止策で、field 単体は明示 override を守る。ReleaseRecord/static 検査には別の実 artifact 判定が必要。 |
| A10 | S8→S2、NEGATIVE_FIXTURE_GAP | 新たに再現した欠陥ではなく、古い指摘である。既存 path-position matrix は S2/S3 の防止策。元の初回検出段階は不明。静的 symlink fixture は TOCTOU 安全性を証明しないため、破壊操作の置換 race は別に調べる。 |

## Release の対応関係

| ID / 出典 | 不変条件、過去の欠陥、影響 | 現在の実装 → 回帰テスト、検証・全体入口の状態 |
| --- | --- | --- |
| HCR-R01 / F 監査 | Git 問合せは選択 root を使う。root 無視で別 repository を検証・build し得た。 | `tools/repoctl/release_source.go:gitOut` → `release_source_test.go:TestReleaseSourceValidTags`、`TestReleaseGitIgnoresAmbientRepositoryRouting`。Yes/helper。cwd と異なる一時 repo、汚染した routing を使用。command も同じ helper を通る。 |
| HCR-R02 / F 監査 | release-check は実 artifact を検査する。元は何も検査せず成功した。 | `release.go:checkRelease` → `release_e2e_test.go:TestReleaseCandidate` の identity/digest/tampering。Yes/component。実6 target candidate を使うが **opt-in で今回未再実行**。usage テストだけでは受け入れ証拠にならない。 |
| HCR-R03 / F 監査 | 必須入力欠落時は packaging を失敗させる。元は入力を黙って省いた。 | `release_archive.go:writeArchive/readReleaseArchive` → `release_archive_test.go:TestReleaseArchiveRejectsUnsafeMembers/missing` と round-trip。**Partial**。reader は欠落を拒否するが、writer の欠落入力を直接試す回帰は見つからなかった。実装は staging 件数と全必須 file を検査している。 |
| HCR-R04 / F 監査 | 正しい LICENSE/README.txt/executable を収録する。元は README.md を使った。 | `release.go:releaseReadme`、archive names → `TestReleaseArchiveRoundTripDeterministic`、candidate の再 hash した `license`/`readme` 改変。Yes/helper/component。独立した期待内容と意味検査。candidate は opt-in。 |
| HCR-R05 / F 監査 | archive close error を返して失敗出力を受理しない。元は close error を無視した。 | `release_archive.go:writeArchive` は deferred file/ZIP/gzip/TAR close error を結合する。**直接 close failure を注入する回帰は見つからない**。成功 round-trip では証明できない。防止策の状態は失敗注入を **deferred**。現在の失敗を確認したわけではない。 |
| HCR-R06 / F 監査 | build 成功のために caller の既存出力を削除しない。元は既存 directory を消した。 | `release.go:buildRelease/publishReleaseDirectory` → `release_publish_test.go:TestReleasePublicationPreservesOutputAndCleansTransfer`、candidate `existing_output_preserved`。Yes/helper/component。元 bytes と転送失敗時の出力不在を検証する。 |
| HCR-R07 / F 日英 | translation hash を意味の翻訳の代わりにしない。日本語 hash 更新済みでも revision/progress が英語だけ変わった。 | 日英 completed Plan と docs-check。**Partial**。hash は bytes の古さを検知できても誤訳を検知しない。人間の対訳確認が既存の防止策。親監査の日英 corpus と参照し、製品指摘に重複計上しない。 |
| HCR-R08 / F 実験 | static identity は trimpath 後も残り、独立した VCS/target/CGO と一致する。Go buildinfo が linker flags を省いた。 | `buildinfo.go:ReleaseRecord/Current`、`release.go:inspectReleaseBinary` → `TestCurrentReleaseRecord`、`TestInvalidReleaseRecordPreservesDevelopmentIdentity`、candidate `build_identity`/`version_record`。Yes/API/component。stripped binary の結合は実 candidate が必要。 |
| HCR-R09 / F 独立 | compiler 入力は commit の bytes とし、ignored/hidden な caller 変更を入れない。最初は可変 caller tree を build した。 | `release_source.go:privateReleaseSource` → `release_source_test.go:TestPrivateReleaseSourceUsesOnlyCommittedFiles`。Yes/helper。ignored file 不在、hidden tracked edit 除外、caller 保存を検証。candidate build も同じ helper を呼ぶ。 |
| HCR-R10 / F 独立 | publication 時も最初の source identity と一致する。最終検証の戻り identity を捨てた。 | `release.go:buildRelease` が2回の check/publication 前に tag/time/commit を比較。**Partial**。validator の拒否テストはあるが、build 途中で identity を変える専用回帰は見つからない。interleaving 注入は **deferred**。比較の実装自体は確認した。 |
| HCR-R11 / F 独立 | tag publication は同一 source の repeat と native smoke に依存し、検証済み bytes を使う。tag 固有の repeat gate がなかった。 | `.github/workflows/release.yml` → `release_workflow_test.go:TestReleasePublicationGate`。Yes/static workflow 判定。実際に YAML を4形態へ変異する。graph/command の存在を検査し、任意 shell/action の意味までは証明しない。native workflow 証拠は別扱い。 |
| HCR-R12 / F ZIP 変異 | 安全な central name で traversal/absolute local name を隠せない。元は central directory を信用した。 | `release_archive.go` local-header 検査 → `TestReleaseArchiveRejectsMismatchedZIPLocalNames`。Yes/helper。正常 archive の local header だけを再圧縮せず変える。reader は実 checkRelease 経路である。 |
| HCR-R13 / DR | checkout suffix と一致する module path は host-path leak ではない。単純 byte search が正常 artifact を拒否した。 | `release.go:releaseBinaryContainsPathWithModules` → `release_path_test.go:TestReleaseBinaryKnownModulePaths`、candidate `module_path_is_not_checkout_leak`。Yes/helper/component。実 candidate 内の存在と本物の path 追加を検証する。 |
| HCR-R14 / DR 不採用案 | Go string pool 内の連結した実 absolute path も検出する。token 境界による除外案が文字直後の path を通した。 | 同 detector → `TestReleaseBinaryDetectsConcatenatedPathLiteral` と table の連結例。Yes/helper、実 build binary を使用。独立レビューで退けた案であり、公開済み回帰ではない。 |
| HCR-R15 / DR | assume-unchanged/skip-worktree は未変更でも cleanliness 検証時に拒否する。porcelain が変更を隠した。 | `release_source.go:releaseSourceClean/releaseVersion` → `TestReleaseSourceRejectsHiddenIndexEntries`。Yes/helper。両 flag × 変更・未変更。command の漏れが R18 で再発した。 |
| HCR-R16 / RR | 正当な非 ignored worktree 出力で、最終 source check 前に tree を汚さない。sibling staging が偽の拒否を起こした。 | `release.go:buildRelease` は外部 staging 後に出力先 filesystem へ publish → candidate `nonignored_worktree_output`。Yes/component。入れ子・日本語・非 ignored 出力を byte 比較。opt-in で今回未再実行。 |
| HCR-R17 / FR | private checkout の compiler bytes は ambient filter/attribute に左右されない。smudge の変化を対応 clean が隠した。 | `release_source.go:releaseGitEnv/privateReleaseSource` → `TestPrivateReleaseSourceIgnoresAmbientCheckoutFilters`。Yes/helper integration。global/system filter が通常 checkout を実際に変え、隔離 clone は commit bytes と一致することを検証。 |
| HCR-R18 / VR | release-verify は build/output 前に caller cleanliness を検証する。caller は porcelain、flag 対応 releaseVersion は clean clone だけに実行した。 | `release_verify.go:executeReleaseVerify` → `release_verify_test.go:TestReleaseVerifyRejectsHiddenIndexEntries`。executeArgs による Yes/command。4例で診断、出力/build log ゼロ、destination 不在、caller index/bytes/refs 保存を検証。 |
| HCR-R19 / DR metadata | archive 済み状態と durable DB path は実装に一致する。親 status が active、説明が state.db でなく registry.sqlite だった。 | completed distribution metadata、design の path audit、`internal/cli/lifecycle.go` の state.db。**Partial**。docs の機械検査と source/文章の直接比較で、path 名の意味 validator はない。一般的な意味のずれは日英 corpus で扱う。 |

PR 6 の残りの対応済み2件は、日本語 milestone と checksums/manifest が release set の
隣接 file であることに関するものだった。前者は R07、後者は R02/R04 と
`releaseChecksums/checkRelease` が扱う。出典から追加の現在欠陥は確認できない。
以上で DR の11件すべてを含み、古い指摘を新規修正として数え直していない。

## Release の見逃し分析

| ID | 検出→最早、見逃し分類 | 早期機会、防止策、再発先 |
| --- | --- | --- |
| R01 | S7→S2、HELPER_ONLY、NEGATIVE_FIXTURE_GAP | cwd と異なる一時 root は用意できたが、実 Git release behavior のテストがなかった。既存 source fixture/routing 汚染が S2/S3 の防止策。source/worktree の Git routing にも同じ問いが必要。 |
| R02 | S7→S3、HARNESS_GAP、NEGATIVE_FIXTURE_GAP | corrupt candidate の拒否なしで command 成功を受け入れとした。既存 real-candidate 変異 matrix は有効化時に S3/S4 の防止策。通常 unit 成功を candidate 証拠にしない。 |
| R03–R04 | S7→S2、NEGATIVE_FIXTURE_GAP、ORACLE_COUPLING | archive の構成・内容は指定済みで missing/wrong fixture を作れたが、release behavior 自体を試していなかった。既存 reader/semantic candidate が S2/S3 の防止策。writer 欠落入力は **deferred**。reader の検証で代用しない。 |
| R05 | S7→S2、FAILURE_INJECTION_GAP | close error は局所的な戻り値契約だが、成功 archive だけでは検出しなかった。error 結合は製品の防止策であり回帰証拠ではない。失敗 writer/close fixture は **deferred**。round-trip が error の取りこぼしを検知するとはしない。 |
| R06 | S7→S2、NEGATIVE_FIXTURE_GAP | sentinel bytes のある既存出力は当初から作れた。既存 publication/candidate 保存テストが S2/S3 の防止策。同じ no-clobber は A05 の native concurrency にも現れる。 |
| R07 | S7→S7、REVIEW_CHECKLIST_GAP | hash 検査が持つ情報は byte 不一致までで、意味は分からない。出典も hash だけの更新を記録する。対訳の人間による確認が現実的な早期防止策。汎用的な意味翻訳 validator が実用的とは主張しない。 |
| R08 | S7→S3、COMPOSITION_GAP | identity field の link 成功は実 trimmed executable の static inspection を証明しなかった。実 build/probe/candidate が S3 の防止策。field 単体だけでは不十分。実 VCS の結合漏れ A09 と関連する。 |
| R09 | S8→S3、NEGATIVE_FIXTURE_GAP | clean Git status は compiler 入力 identity の証明ではなく、ignored/flagged edit の fixture は作れた。既存 private-clone bytes が S3 の防止策。R17 の再発が、clean status だけを判定手段にできないことを示す。 |
| R10 | S8→S3、FAILURE_INJECTION_GAP | 初回・最終観測はあったが、その間の identity 変更テストがなかった。現在の比較は確認済み、専用 interleaving は **deferred**。作用後の検証結果を捨てる箇所を横断確認する。 |
| R11 | S8→S6、HARNESS_GAP、REVIEW_CHECKLIST_GAP | 必須 graph edge/repeat command はレビュー前に機械検査できた。既存の変異付き workflow テストが S6 の防止策。文字の存在は実行証明ではなく、preview/tag を別々に確認する。 |
| R12 | S8→S2、ORACLE_COUPLING、NEGATIVE_FIXTURE_GAP | archive library で作る fixture は local/central を一致させ、parser の仮定を共有した。独立した local-byte 変異が S2 の防止策。ZIP 固有 checklist より、二重表現への検証方法として一般化する。 |
| R13 | S8→S2、NEGATIVE_FIXTURE_GAP | module suffix と衝突する短い checkout basename がなかった。既存 collision/実 path matrix が S2、実 candidate が S3 の防止策。 |
| R14 | S8→S3、ORACLE_COUPLING | text fixture の token 仮定に対して Go は文字列を境界なしに連結する。実 Go binary の拒否が S3 の防止策。この不採用案を merge 後の見逃しとはしない。 |
| R15 | S8→S2、NEGATIVE_FIXTURE_GAP | Git に hidden-index flag があるのに porcelain を判定手段とした。既存4例 helper が S2 の防止策。R18 が、元の検証は実 command まで届かなかったことを示す。 |
| R16 | S8→S3、COMPOSITION_GAP | 以前の build 証拠は tree 外/ignored 出力だけで、汎用 output 契約に必要な非 ignored 例がなかった。実入れ子出力回帰が S3 の防止策。cleanliness の例外は追加していない。 |
| R17 | S8→S3、NEGATIVE_FIXTURE_GAP、COMPOSITION_GAP | private clone の fixture に status を clean に保てる active filter がなかった。通常 checkout を実際に変える対照が強い bytes 判定を作り、S3 で検出する。P07 の environment 解決にも ambient 依存がある。 |
| R18 | S8→S3、HELPER_ONLY、COMPOSITION_GAP | R15 は releaseVersion を試したが、orchestration がそれを別 repository にだけ適用した。executeArgs の拒否が順序と caller 保存を S3 で検証する。P13 Doctor/Inventory と同様に、共通検査を作る際は関連公開入口を対にする。 |
| R19 | S8→S7、REVIEW_CHECKLIST_GAP | metadata/path の文章を完了状態・CLI source と直接比較できた。docs-check は構文を検査するが意味までは検査せず、source/文章の対照が防止策。広い意味検証は docs 監査へ引き継ぐ。 |

## 今回の証拠、限界、引き継ぎ

2026-09-09、native Linux、Go 1.27.1、固定 revision で実行した。

- `go test -race ./internal/assets ./internal/buildinfo ./internal/runtime/compose ./tools/repoctl -count=1` は順に 2.079s、2.146s、1.306s、8.768s で成功した。実 subprocess materialization と実 Git/filter fixture を含む。`TestReleaseCandidate` は opt-in 変数未指定で skip し、**再実行済みには数えない**。
- P01/P10/A02/A03 と standalone 遅延 prerequisite の targeted race は app 1.381s、paths 1.029s、Android 1.047s、CLI 1.048s で成功した。対象は `TestCleanupRetainsProofAfterContainerDisappears`、`TestCleanupProofWriteFailurePreventsDown`、`TestInventoryDiscoversOrphansOutsideRecordedProviders`、`TestResolveOverrideWithoutUserHome`、`TestUIHelperStagingStaysInOwnedRuntimeAndCleansUp`、`TestStandaloneCommandsRequestGitOnlyWhenSourcesAreNeeded`。
- Windows held-handle、Windows checkout 変換、実 Podman/Docker 共存、実6 target candidate/native smoke は、この限定した過去検証では**再実行していない**。元の Plan に過去の具体的証拠があり、今回の baseline は親の報告に記録する。

この過去検証から現在の製品欠陥は確認していない。防止策の強さについて採否判断が
必要なのは、P08 の長文診断、P11 の fixture 失敗復旧、R03 の writer 欠落入力、
R05 の close error 注入、R10 の build 中 identity 変更である。
実装の契約違反を証明したのではなく、回帰の強さに関する懸念なので、ここでは
`AUDIT-*` ID を付けない。表に明記した全体入口の不足は後の結合監査に使い、
helper の成功で隠さない。

限定した再発検索では、app inventory dispatch、Podman/shared Docker traversal、
asset publication/read、source validation の呼出元、release orchestration、
workflow graph テストを確認した。確認した過去の再発対は P13/R18（helper と次の入口）、
R09/R17（clean status と compiler bytes）、A05/A06（native publication 修正と後続 read）、
P01/P13（inventory 結合）、A09/R08（仮 identity と実 binary）である。
より広い現在の runtime/portability/cleanup 確認は Phase A の別工程として扱う。
