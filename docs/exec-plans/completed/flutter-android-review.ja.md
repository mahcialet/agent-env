---
status: completed
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/completed/flutter-android-review.md
source_sha256: 9a4e5447edd8e9ff66bceb3bda50556e21bc315694b493d6f24a10167fbf03f2
---

# PR 4 Flutter Androidレビュー対応

[English（翻訳元）](flutter-android-review.md)

予定ブランチは `feat/flutter-android-runtime`、開始revisionは `447aa99`。
本計画がレビュー対応を管理します。完了済みFlutter実装計画は過去の提供証拠として保持し、`docs/PLANS.md` に従います。

## 目的 / 全体像

PR 4の現在の8件のThreadを、同じ不具合を検出するテスト・公開文書の同期・各Threadへの返信と修正確認後のResolveまで対応します。

## 進捗

- [x] 2026-09-08: クリーンな作業ツリー、ブランチ、未解決Thread 8件を確認。
- [x] 2026-09-08: 選択APK出力衝突、省略可能なreverse観測、destroyプレビューを修正。
- [x] 2026-09-08: ホストFlutter doctor、既定plan表示、プロジェクトのsymlink、非ゼロADB診断を修正。
- [x] 2026-09-08: 既存公開文書の英日を同期。
- [x] 2026-09-08: 不具合の再発を検出するテスト、全harness、race、ネイティブCIを実行。
- [x] 2026-09-08: 全8件のThreadへ返信・Resolveし、本計画を完了済みへ移動。

## 想定外の発見

機能テストが成功しても既存の入口との統合漏れが残っていました。再現した失敗と予想外の結果を以下へ記録します。

## 判断の記録

- 2026-09-08 / 実装: 選択アプリのsource相対APK出力が衝突する場合、副作用前に拒否します。全ビルドを先に行う現ライフサイクルでは両出力を保持できません。レビューで提示された代案を採用し、新たなAPK保存・削除ライフサイクルの追加を避けます。未選択アプリは独立stackのplanを妨げません。
- 2026-09-08 / 実装: 元の完了計画を保存し、既存PRブランチで専用レビュー計画を使用します。app、CLI、adapter、文書の担当範囲を分け、同時編集の衝突を防ぎます。

## 成果と振り返り

8件すべてを `4677b89` で対応しました。各Threadに具体的な修正と同じ不具合を検出するテストを返信し、ネイティブCI成功後にResolveしました。最終GitHub照会はThread 8件、未解決0件でした。

既定の人間向けplan表示、ホストだけの前提診断、reverse省略、安全なプレビューなど、従来の機能テストが扱わなかった入口・任意経路を修正しました。同じ問題を検出するテストは元の正常系とは独立してこれらを検証します。選択APK出力の衝突は上書きが起こる前に明示的に拒否し、新しいAPK保存ライフサイクルやadapter責務を導入しません。従来の実SDK証拠は過去の検証として保持し、今回のレビュー変更はネイティブテスト・race・CIのCompose結合検証で確認しました。

## 背景と構成

appは処理順序とcleanup、Flutterは移植可能なビルド、Androidは汎用端末コマンドと識別を担当します。公開契約は `docs/product-specs/` にあり、READMEとroadmapが入口です。

## 作業計画

挙動ごとに不具合を検出して失敗するテストを追加し、所有権・証拠保持を弱めず修正した後、公開文書の英日を更新します。rootはapp・plan・previewと本計画、委譲先はCLI・adapter・公開文書を担当し、rootが統合とGitHub操作を行います。

## 具体的な手順

対応Go版で重点 `go test`、続いて `go run ./tools/repoctl check` と `go test -race ./...` を実行します。検証済み変更をcommit・pushし、ネイティブCIを確認、各Threadへ修正・テスト根拠を返信しResolveします。両計画をcompletedへ移動します。

## 検証と受け入れ

| 指摘 | 受け入れ証拠 |
| --- | --- |
| APK衝突 | 選択出力の衝突をビルド・runtime副作用前に拒否し、別stackは有効。 |
| 公開文書 | README・roadmap・manifest・CLIの英日契約が提供済み挙動と一致しdocs-check成功。 |
| ホストdoctor | Flutterあり・Androidなしで正しく失敗し、SDKをインストールしない。 |
| 既定plan | tableに選択app・source・runtime・artifact・reverseを副作用なしで表示。 |
| 任意reverse | 宣言なしのappではreverse・endpoint問い合わせを行わない。 |
| project symlink | project自身・祖先の内部symlinkをビルド前後に拒否。 |
| destroyプレビュー | 両build guardがforceでも削除予定表示を止め、変更を起こさない。 |
| ADB失敗 | 非ゼロ終了の出力を秘匿・長さ制限して保持し、エラー識別を保つ。 |

## 冪等性と復旧

公開履歴を書き換えません。失敗証拠と所有権検査を保持し、SDK導入・ライセンス承認・一括cleanup・秘密やローカルパスの開示を行いません。修正後に再検証し、未対応Threadは未解決のまま残します。

## 成果物と注記

commit、コマンド、CI、Thread対応結果をここへ記録します。ローカルSDK/Flutterパスは実行時入力に限定し、文書へ記載しません。

## インターフェースと依存

新しいruntime依存・adapter間importを追加せず、引数配列とWindows/macOS/Linuxネイティブ動作、既存Lease JSONと資源所有権を維持します。

レビューチェックポイント（2026-09-08）:

- 初期実装で期待した挙動を確認するテストが失敗することを確認しました。選択出力衝突4ケース、不要なreverse観測、preview guard 2種類、ホストAndroid診断と既定table情報の欠落、内部symlink 6ケース、非ゼロADB出力破棄6ケースです。
- appの不具合を再検出する重点テストと既存build guardはGo 1.27.1 `-race -count=10` で成功（12.410秒）。CLIも `-race -count=1` で成功（1.940秒）。Flutter/AndroidはGo 1.26.8のテストとGo 1.27.1のrace 5反復が成功しました。appの独立した読み取りレビューで具体的な問題は見つかりませんでした。
- 最初の全harnessは単体テストとvetが成功し、公開文書編集中の古い翻訳ハッシュを正しく拒否しました。文書担当が意味確認とハッシュ同期を終え、docs-checkは成功しました。その後、commit前の整合した全harness再実行が成功しました。

最終ローカル検証（2026-09-08）:

- Go 1.27.1とGo 1.26.8の `go run ./tools/repoctl check` が全段階で成功しました。
- Go 1.27.1の `go test -race ./...` が成功しました（app 24.654秒、CLI 1.811秒、Android 2.110秒、Flutter 3.039秒）。
- 指摘の不具合を直接検出するテスト `TestSelectedApplicationAPKOutputsCannotCollide`、`TestApplicationWithoutReverseSkipsNetworkObservation`、`TestDestroyPreviewHonorsApplicationBuildBarriers`、`TestFlutterHostDoctorRequiresBothToolchains`、`TestFlutterHostDoctorMissingPrerequisitesIsPure`、`TestFlutterPlanTableIncludesSelectedApplicationRequirements`、`TestBuildRejectsInternalProjectSymlinks`、`TestApplicationExecutionFailureRetainsDiagnosticsAndErrorIdentity` はすべて成功しました。
- 公開文書6組の英日を同期し、ローカルFlutterパスが追加されていないことを確認しました。ネイティブCIとThreadへの回答も、その後以下のとおり完了しました。

最終完了証拠（2026-09-08）:

- 修正commit: `4677b89`。
- [push CI 34171724270](https://github.com/mahcialet/agent-env/actions/runs/34171724270) と [PR CI 34171727162](https://github.com/mahcialet/agent-env/actions/runs/34171727162) が両方とも全12ジョブ成功しました。Go 1.26/1.27のWindows/macOS/Linuxネイティブ検査、5対象のクロスビルド、Linuxのrace・実Compose結合検証を含みます。
- PR 4の全8件へ返信し、全Resolve操作が成功しました。新しいページ対応照会で `total=8`、`unresolved=[]`、次ページなしを確認しました。人間のレビュー・mergeは別です。
- 英日計画をcompletedへ同時に移動し、翻訳情報を同期しました。ローカルSDKパスや生成物はcommitしていません。
