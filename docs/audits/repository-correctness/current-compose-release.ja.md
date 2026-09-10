---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/current-compose-release.md
source_sha256: 9c6ba798b708ae0738a8034ad60a2b316ed0a59f3909e1c504c1c2fc99b9ea97
---

# 現在の Compose・asset・release の正確性レビュー

[English](current-compose-release.md) · [監査索引](index.ja.md) ·
[過去レビュー記録](history-compose-release.ja.md) ·
[作業の実行基準](../../exec-plans/completed/repository-correctness-audit.ja.md)

Phase A 対象は `031869c8b9073b8e23bc17fbc55243666a52f557`。
Phase A はレビューと一時 Go overlay のみを使用した。Phase B の checkpoint
`56b9c2c` で以下3件を採用した。Phase C の実装・検証は末尾に記録する。
この担当範囲では provider の変更操作・公開は行っていない。

## Phase A の確認範囲

| 不変条件 | 確認した実装と証拠 | 結果と限界 |
| --- | --- | --- |
| provider 選択・identity | Compose dispatcher、Docker 記録済み context、Podman の executable/URL/fingerprint/environment、native bridge。 | fallback はなく、Podman は作用前に fingerprint を再確認する。Docker は context 名を固定し、daemon fingerprint は固定しない。それ以上の保護は推定しない。 |
| 破壊操作前の所有権 | container/named resource の inspection、宣言名検索、匿名 volume の attachment/fingerprint/利用者、Down 後の再観測。 | container ID 欠落で Down を許可する指摘を確認。named resource/inventory には空 ID 拒否がある。観測間の engine 置換は静的確認だけで否定できない。 |
| readiness・protocol | 選択 service の存在、running/health、残存 volume の存在、Podman remote の TCP 限定検査。 | 過去 P04/P12 の、残存volumeの存在とTCP/UDPの扱いを確認するテストは成功。TCP 成功は UDP/application readiness の証明ではない。実 Machine forwarding は未主張。 |
| 部分 cleanup・永続化 | Down 前の app proof Save、container 消失後の Destroy 再試行、scope 内残存削除、rm/down 後の不在。 | 過去 P10 の失敗注入は成功。確認箇所に新しい欠陥は未確認。app saga 全体は親監査が別途確認する。 |
| inventory の完全性 | 利用可能・記録済み provider の集合、独立 native label traversal、Compose frontend 欠落・古いパス。 | P01/P13 は部分 error を含め成功。件数だけによる inspection 集合の一致判定は追加仮説であり、今回の再現済み指摘ではない。 |
| policy・パス移動・ambient state | raw Podman 正規化、再帰的拡張、許可 mount/network、ファイル範囲、明示 environment、private escaped JSON。 | providerの設定正規化とhostアクセス拒否を確認する既存テストは成功。悪意ある manifest を実 provider に渡す実験は行っていない。 |
| evidence・上限 | native 診断を redact 後に末尾8 KiBへ限定、canonical snapshot/inherited secret、archive/file/asset reader。 | release reader の成長による上限迂回を確認。診断長文は過去の coverage 制限として残り、診断漏洩は再現していない。 |
| asset・並行性・パス | portable 名、provenance、root/中間 entry、同時 mkdir、open 後の上限付き read、native publication。 | Linux の独立 process/破損テストは成功。Windows held-handle/long-path/winner は native baseline が必要。静的 symlink 検査では任意置換 race を証明できない。 |
| release source | canonical tag、HEAD 一致、clean index/tree、private commit checkout、filter/config 隔離、最終 identity 比較。 | sourceの固定、Git filterの隔離、hidden-index flagの拒否を確認するテストは成功。build 中の identity 変更注入は過去の coverage 制限。公開 ref は変更していない。 |
| packaging・正確な境界 | member3件、release file8件、canonical manifest、checksum、ZIP local/central、秒単位 mtime、member stream 上限。 | archive 変異・時刻範囲は成功。短い root の path filter と外側の regularRead 上限は失敗。小さい cap で再現できるため、模擬目的だけで256 MiB境界を allocation してはいない。 |
| publication・再現性 | 独立 stage、source 検証、出力先 transfer、既存出力拒否、workflow build→smoke→publish、artifact 名一致。 | 保存・workflow 拒否テストは成功。実 repeat build/native smoke は親の baseline 作業。公開 GitHub Release は作成しない。 |

## AUDIT-RELEASE-001 — 短い絶対 source root で漏洩検出を迂回する

- 重大度: Low。採否: ACCEPT（`56b9c2c`）。
- 不変条件: binary bytes に既知の空でない絶対 checkout path がある場合、root の文字列が短いだけで除外しない。
- 場所: `tools/repoctl/release.go:releaseBinaryContainsPathWithModules` の `if len(path) <= 3 { return false }`。
- 条件: `releaseBinaryContainsPath([]byte("/a/private.go"), "/a")`、または `/ab` と対応 bytes。module identity はない。
- 観測: 両方 false を返す。Go overlay の assertion は両 root で失敗する。
- 期待: 既知の絶対 prefix を検出する。root/volume-root を意図的に除外するなら、その構造と説明に基づき、通常の短い path まで対象にしない。
- 影響: 正当な短い checkout root で path-leak guard に死角がある。detector の契約不足であり、secret 漏洩や candidate 検証全体の迂回は実証していない。
- 既存 coverage: `TestReleaseBinaryKnownModulePaths` は `/tmp`、`/agent-env`、長い Windows root を使い、長さ2・3はない。実連結 literal も `/agent-env` を使う。
- 再現: 一時 `go test -overlay` で上記2式が true であることを要求し、native Linux Go 1.27.1 で両失敗を観測した。
- 修正と確認テスト: Phase A 中は変更せず、Phase C の記録を末尾に示す。
- 検証: overlay `TestAuditShortReleaseRootLeak` の失敗。通常 test file は未変更。関連は過去 HCR-R13/R14 と現在の path/boundary 監査。
- 見逃し: 検出 S9、最早 S2。`BOUNDARY_GAP`、`NEGATIVE_FIXTURE_GAP`。early-return 境界の前後で path 長を試す機会があった。既存 fixture が長い directory に偏り、helper/実 binary ともこの分岐を通さなかった。
- 防止策: 構造的 root の例、隣接長の正常 root、実 path を検出し module path は誤検出しないことを確認するテストを組み合わせる。次回期待段階は S2。追加した実装・証拠は Phase C の記録に示す。

## AUDIT-RELEASE-002 — 事前サイズ検査で実際の read を制限できない

- 重大度: Medium。採否: ACCEPT（`56b9c2c`）。
- 不変条件: `regularRead(path, limit)` は limit 超過 bytes を返さず、サイズ検査後の無制限成長に応じて allocation しない。
- 場所: `tools/repoctl/release.go:regularRead` が `os.Lstat().Size()` 後に無制限の `os.ReadFile` を呼ぶ。`checkRelease`、`compareReleaseDirectories`、`copyVerifiedRelease` が利用する。`release_archive.go:writeArchive` も member 読込で同じ構造を繰り返す。
- 条件: Lstat と ReadFile の間に別 handle が regular file を伸ばす。限定したテストで一時 file を1 byte/64 KiB に交互変更し、`regularRead(path, 1)` を呼ぶ。caller/release データは書き換えない。
- 観測: 38回目に、limit 1 byte に対して8,193 bytes を error なしで返した。fake stat ではなく実 native filesystem の interleaving である。
- 期待: open した file の確認と上限付き stream、余分な data の拒否により、成長・置換に関係なく制限する。
- 影響: release candidate/manifest/checksum、copy/compare の cap が厳密な read 上限になっていない。後段の JSON/checksum が拒否しても過大 allocation の後になる。破損 release の受理を確認したわけではない。
- 既存 coverage: archive member 展開は LimitReader を使用する。過去 asset HCR-A08 は open 後サイズと bounded read を確認する。release fixture は静的な完成 file を変異させ、同時成長を扱わなかった。
- 再現: 一時 overlay `TestAuditRegularReadGrowthBound`。writer は自分の一時 file を一度 open し、`WriteAt(64 KiB, 0)` と `Truncate(1)` を繰り返す。reader は最大20,000回、`err == nil && len(bytes) > 1` で失敗する。終了時に writer を join する。interleaving が起きなければ skip とし、安全性の証明とはしない。
- 修正と確認テスト: Phase A 中は変更せず、Phase C の記録を末尾に示す。writer staging の再発箇所は source 確認のみで、別の失敗注入は行っていない。
- 検証: native Linux Go 1.27.1 の overlay が失敗。RELEASE-001 と合わせ package 0.008s。通常 repoctl race は別途8.768sで成功した。
- 関連: 過去 HCR-A08（asset の同種欠陥）、HCR-R03/R05（writer error/境界 coverage）、横断 file-reader 監査。
- 見逃し: 検出 S9、最早 S2。`CONCURRENCY_GAP`、`BOUNDARY_GAP`、`FAILURE_INJECTION_GAP`。事前検査後の変異、または bounded reader 自体を判定基準にする機会があった。asset の局所修正が release 基盤に残る同種構造を検出しなかった。
- 防止策: regular-file read の共通上限処理、または asset/release/evidence を横断する cap/growth fixture。期待段階 S2/S3。実装修正と、その動作を確認するテストの証拠は Phase C の記録に示す。

## AUDIT-OWNERSHIP-001 — container identity 欠落で Down を許可する

- 重大度: High。採否: ACCEPT（`56b9c2c`）。
- 不変条件: 外部 container identity の欠落は不完全な観測であり、破壊的 Compose Down を許可しない。
- 場所: `internal/runtime/compose/compose.go:dockerClient.Inspect` の件数検査後の container loop。label を検証するが `v.ID` を検証しない。Down は戻り error に基づいて `compose down --volumes --remove-orphans` を実行する。
- 条件: 管理 runtime に lease/runtime/project と immutable JSON が記録され、ps が1 container を列挙する。inspect は所有 label が一致する running な選択 service を1件返すが `Id` を省く。Down 後の一覧は空。
- 観測: 公開 `Client.Down` 全体が破壊操作を dispatch し、nil を返した。一時 fake runner は呼出を記録するだけで engine を実行しない。
- 期待: container identity がないため Down の作用前に拒否する。
- 影響: 不完全な identity 観測が破壊操作の検査を通る。native engine がこの不正 record を実際に返したこと、無関係な資源の実削除、任意の foreign resource 取得は実証していない。
- 既存 coverage: `TestManagedResourcesRequireBothOwnershipLabels` は有効 ID を保ちながら label を変える。named resource と native inventory は空 identity を既に拒否するが、container 分岐には同等の検査がない。
- 再現: 一時 overlay `TestAuditMissingContainerIDAuthorizesDown`。既存 runtimeFixture を使用し、一時 canonical snapshot を書き、LeaseID を設定し、上記応答を返す。error あり・Down dispatch ゼロを要求したが `dispatched=true err=<nil>` を観測した（package 0.025s）。
- 修正と確認テスト: Phase A 中は変更せず、Phase C の記録を末尾に示す。
- 関連: 過去 HCR-P04/P06。同種検索で named resource/Inventory の既存検査と Podman 匿名 attachment の ID 厳密一致を確認した。件数だけ一致する異なる非空 ID 集合は、別の未確認仮説として残る。
- 見逃し: 検出 S9。最早は欠落 field に対し S2、Down 検査に対し S3。`NEGATIVE_FIXTURE_GAP`、`ORACLE_COUPLING`、`COMPOSITION_GAP`。所有 label を正常に保ちながら identity field を個別に省き、公開破壊入口が作用に到達しないことを試せた。既存 fixture は ID と正常 record を結び付けていた。
- 防止策: provider inspection 共通の missing/empty/duplicate/wrong-ID 応答 matrix と公開 Down の作用拒否 assertion。期待段階 S2/S3。採否決定後に追加した防止策は Phase C の記録に示す。

## 否定した仮説と確認範囲の限界

- 「Podman UDP がまだ TCP dial する」: 現在の suffix filter と TCP/UDPを混在させてInspect全体を通すテストで否定した。mapping は application 応答ではない。
- 「Inventory は常に podman-compose が必要」: engine-only Doctor 後の InventoryFor を frontend 欠落・古いパス、全 native kind で検証して否定した。
- 「残存匿名 volume を不在扱いできる」: 過去の条件は最終 resource 集合での存在算出と 残存volumeだけがある状態を検査するテストで否定した。任意の同時外部削除・再生成まで証明したのではない。
- 「release-verify が caller の hidden index flag をまだ受理する」: build/output 前の executeArgs 4例で caller bytes/index/refs を保存したまま拒否するため否定した。
- 「private clone が active global/system clean/smudge filter を引き継ぐ」: 対照の通常 checkout は変換され、隔離 checkout は commit bytes と一致する fixture で否定した。
- 「ZIP central-safe/local-unsafe がまだ通る」: 現在の local-record parser と直接 local-header 変異で否定した。
- 「asset cache 事前検査も release regularRead と同じ」: 現在の asset reader は open 後 Stat、LimitReader、追加1 byte を検査するため否定した。RELEASE-002 は別 package の再発であり、未修正 asset bug ではない。
- Docker の可変 context routing、呼出間 file 置換、診断 secret source の不足、同件数で異なる非空 inspection ID は、契約・再現の追加確認が必要な限定仮説である。上の再現済み3件に混ぜない。

## 検証と次の工程

[過去レビュー記録](history-compose-release.ja.md) に成功した package/targeted race と
native/candidate の限界を記録している。一時 Go overlay 2個は Go の mapping 経由で
テストだけを追加し、Phase A 中は repository の製品ファイルを変更しなかった。
release overlay は native Go の一時 file、Compose overlay は injected runner のみを
使い、Docker/Podman の変更操作は実行しない。

Phase B の checkpoint で3件すべてを採用した。過去の coverage 制限と未確認仮説は、
これらの決定と引き続き区別する。

## Phase C の修正と検証

- **AUDIT-OWNERSHIP-001:** `dockerClient.Inspect` は観測の収集や Down に進む前に、欠落・空 container ID を拒否する。恒久的な `TestMissingContainerIdentityRefusesInspectionAndDown` は公開 Inspect/Down の両入口と、field 省略・明示的空の両方で、実際の inspection 到達、identity 診断、破壊 dispatch ゼロを要求する。修正前は4例すべて失敗し、Down の2例は dispatch 後に nil を返した。provider 入口の防止策（S3）である。
- **AUDIT-RELEASE-001:** 一律の長さ cutoff を、空 path/filesystem root の構造的な除外に置き換えた。通常 root `/a`、`/ab`、`/abc` では連結 literal も検出し、既知 module identity の除外を保つ。`TestReleaseBinaryShortCheckoutPaths` は修正前に `/a`、`/ab` で失敗し、修正後は3つの実 path/module の対で成功する（S2）。
- **AUDIT-RELEASE-002:** `regularRead` は open した handle の種類・サイズを確認し、cap と追加確認1 byte までに read を制限する。成長した file は部分 data を返さず error とする。archive staging も同じ bounded helper を使い、もう一つの無制限 member read を除去した。`TestReleaseReadBoundsGrowthAfterOpenedStat` は Stat 取得直後に wrapper が実 file を伸ばし、sleep・race・skip に依存しない。同等の無制限 reader を抽出した段階では cap 0/1/16 のすべてで65,536 bytes を返して失敗した。修正後は全件拒否し、実 read bytes が cap+1 以下であることも検証する。公開入口の `TestReleaseRegularReadExactBounds` は空・上限一致・1 byte 超過を確認する（S2/S3）。

修正後、native Linux Go 1.27.1 で検証した。

- path/read/archive と provider identity の targeted race を10回実行し、repoctl 2.945s、Compose 1.129s で成功した。
- 全 `TestRelease*` race は1.886sで成功した。opt-in の実 candidate はここでは skip し、release-build 証拠とは扱わない。
- Compose package 全体の race は1.185sで成功した。
- 両 package の Windows/amd64、CGO 無効 test binary の compile が成功した。compile の portability 証拠であり、native Windows 実行証拠ではない。

不確実な所有権や上限超過入力を許容するように公開動作を広げていない。
全体 harness/native/release 検証と独立レビューは親監査の統合工程で行う。
この担当範囲では commit しない。
