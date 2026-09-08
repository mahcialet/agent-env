---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/roadmap.md
source_sha256: efc12d1d41590f5aa6dacbce63eec28840029f149c746eca86470efcaad13728
---

[英語版（翻訳元）](roadmap.md)

# ロードマップと未解決の決定事項

実装済みの範囲は、固定したローカルGitソース、detached review worktree、隔離したComposeとAndroid Emulatorのruntime、Flutter Androidアプリのビルド・インストール・起動とバックエンドへのreverse設定、所有 Android UI の accessibility 観測・操作、名前付きargvテスト、証拠、移植可能なリポジトリharnessです。このロードマップは、延期した機能を利用可能なコマンドとして紹介するものではありません。

## 確定したMVPの選択

moduleは`github.com/mahcialet/agent-env`で、既存のMITライセンスを保持します。AGENTSは150行を上限とし、パッケージ境界には構造的な検査を設け、データベース文書は連番migrationから生成します。組み込みポリシーは固定公開ポートと、選択対象の外部・共有Composeリソースを拒否します。SQLite transactionと、更新可能でfencingを備える操作lockによりローカルプロセスを調整します。実行スナップショットには選択したサービス・リソースの依存閉包のみを含めます。ネイティブのプロセスツリー取消は現在のrunnerの一部であり、将来のホストプロセスアダプターではありません。

## 信頼とホストポリシー

設定可能なホストポリシーファイル、猶予・保持期間の設定、並列割り当ての制御は今後の作業です。信頼するbaseマニフェストと対象overlayのマージ、`--manifest-ref`、信頼できないforkの明示的な実行には、別途信頼設計が必要です。助言的なownerラベルは認証を提供しません。安全でないリポジトリを受け入れるためだけに組み込みポリシーを弱めないでください。

## ソースと書き込みworkflow

リモートmirror/cache管理、HTTPS/SSH認証、provider固有のPR省略記法、自動fetchは延期しています。書き込み可能な修正リースには、branch所有権、ソースごとの書き込み選択、復旧ルールが必要です。fork/checkpoint/reproduceと、稼働中stackの拡大・縮小には、明示的な識別情報と成果物のモデルが必要です。

## runtimeの拡張

iOS、browser/CDPの自動操作・snapshot、汎用の永続ホストプロセス、Podman固有対応、分散・複数ホストの調整は未実装です。Android Emulatorリースは専用AVD状態とローカルSDKプロセスを所有します。追加の実機CIにはアクセラレーションを利用できるrunnerが必要です。ブラウザーリソースには明示的な所有権とcleanupルールが必要です。

[Android UI observer](product-specs/android-ui-observer.ja.md) は、既存の所有 Emulator に対し、
上限付きの意味情報 snapshot、PNG、Unicode 置換、navigation、現在の PID の log を提供します。
独立した任意の platform companion を使い、対象アプリへの instrumentation 追加は不要です。
OCR、visual regression、より豊富な gesture、物理デバイス、remote Emulator host は引き続き延期しています。
observer の最終受け入れ確認と platform 別の証拠は[進行中の計画](exec-plans/active/android-ui-observer.ja.md)で管理します。

## 成果物とリリース

ローカルOCI registry、image promotion、image保持参照、厳密な成果物のreplayは延期しています。runtime検査は実際のコンテナーimageの識別情報を記録しますが、再現可能なimage promotionではありません。成果物の自動期限切れ、event圧縮、migration rollbackツール、生成CLI/JSON Schemaリファレンス、長期的なhandoff archive方針は未決です。

GitHub Releasesやpackage managerによるリリースパッケージ化は未決です。リリースはテストしたrevisionのネイティブ検証と統合検証の証拠を引用する必要があります。対応build targetはその証拠の代わりにはなりません。

## CIの拡張

Windows/macOSのネイティブDocker統合には、Dockerが使えるself-hosted runnerが必要になる場合があります。Android統合には適切なhardware accelerationが必要です。`last_verified`からの文書の経過日数は、現在CIが強制する鮮度の期限ではありません。metadataの妥当性と文書を見つけられることは検査で強制します。
