---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/index.md
source_sha256: eea3c0043503bc21c336fcdbd3878a08875f5be0023a96df7b80c1d5af531f4c
---

# リポジトリ正確性監査

[English](index.md)

完了した監査の指摘と証拠を探すためのindexです。これは履歴の報告であり、現在のrevisionを
再検証したという意味ではありません。現在の挙動は[製品仕様](../../product-specs/index.ja.md)、
検証コマンドは[品質方針](../../QUALITY.ja.md)を参照してください。

実行指示は[完了した監査Plan](../../exec-plans/completed/repository-correctness-audit.ja.md)にあります。
Phase Aの固定対象はPR #10 merge後のmaster、`031869c8b9073b8e23bc17fbc55243666a52f557`です。
監査時のbranchは`audit/repository-correctness`でした。

Phase A/Bは完了しています。当初採用した15件は実装・回帰・native受け入れを通過しました。
その後merge後の4件を再現・採用し、計19件を修正して独立レビューしました。
過去の回帰テスト名が記録されていても、すべての過去の不具合版をmutation replayしたという意味ではありません。

## レポート

採否と修正状況は指摘一覧、subsystemの検証範囲はmatrixから確認できます。
ほかのレポートは、その根拠となる観測結果と履歴資料です。各レポートには日本語版があります。

| 知りたいこと | レポート |
| --- | --- |
| 何を採用し、修正したか | [指摘の採否と修正](findings.ja.md) |
| どの不変条件を何で守るか | [subsystem × invariant matrix](matrix.ja.md) |
| どの履歴資料を調べたか | [全履歴資料と見逃しの要約](historical-corpus.ja.md)、[文書履歴とfragment](documentation.ja.md) |
| control planeをどう調べたか | [control-plane監査](current-control-plane.ja.md) |
| Android・Flutter・UIをどう調べたか | [mobile履歴](history-mobile.ja.md)、[mobile現行監査](current-mobile.ja.md) |
| processとbrowserをどう調べたか | [process/browser/MVP履歴](history-process-browser.ja.md)、[process/browser現行監査](current-process-browser.ja.md) |
| Composeとreleaseをどう調べたか | [Compose/release履歴](history-compose-release.ja.md)、[Compose/release現行監査](current-compose-release.ja.md) |
| PR #10の終盤に何が追加されたか | [Browser追補](supplemental-browser.ja.md)、[CLI結果追補](supplemental-cli.ja.md) |

## 証拠の読み分け

以下の基準検証は固定した開始時の製品、修正版の検証は初期修正、最終受け入れは最後の追補revisionの
結果です。revisionと、失敗・未実行の項目を分けて読んでください。数値は当時の結果を保持したもので、
今回の文書編集で再測定した値ではありません。

## 基準検証

| 固定した製品コードの検査 | 結果 |
| --- | --- |
| `repoctl check` | 提供された日本語監査 Plan のメタデータ欠落修正後に成功。最初の文書エラーは Plan に残した。 |
| `go test -race ./...` | 全 package で成功。 |
| `repoctl test-integration` | 実 Docker とタグ付き検証に成功。process/Compose 共存も含む。 |
| Linux Browser native と race | sandbox 有効で成功、10.130秒。 |
| 固定 master の native CI | Verify34290359477 と Browser native34290359439 が Windows/macOS/Linux で成功。修正後の候補版 CI は別途必要。 |
| 実 Podman 並行リースと Docker 共存 | 明示した integration flag と導入済み podman-compose で成功、126.631秒。 |
| 実 Android Emulator 並行リース | 専用の一時 template home で独立リースと手動終了後の照合に成功、48.587秒。 |
| Flutter Android backend lease | 342.51秒で失敗。両 build は完了したが runtime readiness が期限切れ。両 Emulator ログで tmpfs 空き不足（必要12GiB、空き約10GiB）を確認。成功扱いにしない。UI observer も同様に332.31秒で失敗。テスト上限を変えず、固定した隔離cloneとディスク上の一時保存先で再実行した。Flutter73.54秒成功、UI75.11秒で最初のtapが失敗し、AUDIT-UI-001を独立に確認した。 |
| 隔離 clone の `release-verify` | 6対象を2回生成し8ファイルが完全一致。Linux 展開後 smoke も成功。tag は検証用の隔離 clone だけに作成。 |

CLI のタグと名前を推測した2回は `no tests to run` であり、検証実績に含めない。正しい Podman と process integration は上表に記載した。Android template は未導入のイメージ種別で最初に失敗し、その後導入済みイメージで成功した。開発者固有の SDK/Flutter パスは記載しない。

## 修正版の検証

製品候補: `687228645a4088547818ec07e428e5734110e18c`。この候補は当初15修正を検証した。後のBrowser/CLI追加修正は新候補で検証する。

| 検査 | 結果 |
| --- | --- |
| `repoctl check` / `go test -race ./...` | 成功。全候補版suite、app race45.555秒。process preview単独race付き10回はテスト変更なしで8.205秒成功。 |
| `repoctl test-integration` | 成功。実Docker/Composeとprocess共存、CLI170.949秒。opt-in Podmanは別実行を下記に記録。 |
| Real Flutter / latest UI | Flutter107.45秒成功。最新UI139.632秒成功。2件のFlutter/APK/Compose/Emulator並行リースとstale/privacy/input/cleanupを検証。 |
| Real Android Emulator | 44.613秒成功。独立リースと片側手動終了後の照合。 |
| Real Podman / Docker coexistence | 99.951秒成功。両opt-in flagで `TestPodmanIntegrationConcurrentLeasesAndEvidence` を実行。最初の完全一致名指定はテストに一致せず、検証に含めない。 |
| Linux Browser native race | 10.178秒成功。sandbox有効。 |
| Browser native CI34294068663 | 6872286でWindows/macOS/Linux成功。 |
| Private-clone release-verify | 6872286で成功。6対象2回生成、8ファイル一致。source外・空PATH・Unicode state rootのLinux smoke成功。公開tag/releaseなし。 |

Verify34294068659: Go1.26/1.27のnative OS6job、cross-build5job、integrationが成功。最終mobile独立レビューも成功（bounds race付き3回4.352秒、Android/helper1.280秒/1.010秒）。追加Browser対応は下記の最終候補検証まで成功した。未実行項目を成功に含めない。

## 最終受入

最終製品revision: `f2ec634baa00af5221217dbfcd5c0aef93c624c9`。
全 `repoctl check` と `go test -race ./...` は成功。追加protocol回帰も
race付き3回1.564秒、独立実行5回2.039秒成功。
実Browser native raceは新しい既定table表示の検査を含め10.157秒成功。
[Verify34295144985](https://github.com/mahcialet/agent-env/actions/runs/34295144985)
と [Browser native34295144958](https://github.com/mahcialet/agent-env/actions/runs/34295144958)
は同一revisionで両方成功し、native Go/OS6job、cross-build5job、Docker integration、
実Browser3OSを含む。同revisionの隔離cloneでrelease-verifyを再実行し、
6対象2回生成、8ファイル一致、native Linux smokeも成功した。
既に記録した実Android/Flutter/UI/Podmanの証拠も適用できる。その後の製品変更は
Browser/CDPとBrowser CLIに限定される。cross-buildだけでnative成功とはしない。

採用19件をすべて解決（High7、Medium10、Low2）。Critical0、REJECT/DEFER/DUPLICATE0。
履歴186件、matrix20×14。全件を独立レビューした。matrixには具体策と
未実行の稀な動作順序を明記し、限定した証拠を不具合の完全不存在の証明とはしない。
完了文書commitで移動・link・翻訳を最終検査する。merge・公開tag・releaseは行っていない。

PR #11追加対応は [レビューExecPlan](../../exec-plans/completed/repository-correctness-review.ja.md) に記録する。上記の元の監査証拠は保持する。
