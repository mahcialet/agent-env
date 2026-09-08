---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/roadmap.md
source_sha256: 057a53825e83a8fab933cad1ceacb8019ba6cd3fa90f08b41d5d5fc1646e7a49
---

[英語版（翻訳元）](roadmap.md)

# ロードマップと未解決の決定事項

実装済みの範囲は、固定したローカルGitソース、detached review worktree、隔離したComposeとAndroid Emulatorのruntime、Flutter Androidアプリのビルド・インストール・起動とバックエンドへのreverse設定、所有 Android UI の accessibility 観測・操作、名前付きargvテスト、証拠、移植可能なリポジトリharnessです。このロードマップは、延期した機能を利用可能なコマンドとして紹介するものではありません。

## 確定したMVPの選択

moduleは`github.com/mahcialet/agent-env`で、既存のMITライセンスを保持します。AGENTSは150行を上限とし、パッケージ境界には構造的な検査を設け、データベース文書は連番migrationから生成します。組み込みポリシーは固定公開ポートと、選択対象の外部・共有Composeリソースを拒否します。SQLite transactionと、更新可能でfencingを備える操作lockによりローカルプロセスを調整します。実行スナップショットには選択したサービス・リソースの依存閉包のみを含めます。native process treeの取消は時間制限付きrunnerが担い、常駐processの寿命には独立したmanaged detached interfaceを使います。

## 信頼とホストポリシー

設定可能なホストポリシーファイル、猶予・保持期間の設定、並列割り当ての制御は今後の作業です。信頼するbaseマニフェストと対象overlayのマージ、`--manifest-ref`、信頼できないforkの明示的な実行には、別途信頼設計が必要です。助言的なownerラベルは認証を提供しません。安全でないリポジトリを受け入れるためだけに組み込みポリシーを弱めないでください。

## ソースと書き込みworkflow

リモートmirror/cache管理、HTTPS/SSH認証、provider固有のPR省略記法、自動fetchは延期しています。書き込み可能な修正リースには、branch所有権、ソースごとの書き込み選択、復旧ルールが必要です。fork/checkpoint/reproduceと、稼働中stackの拡大・縮小には、明示的な識別情報と成果物のモデルが必要です。

## runtimeの拡張

iOS、browser/CDPの自動操作・snapshot、分散・複数ホストの調整は未実装です。Android Emulatorリースは専用AVD状態とローカルSDKプロセスを所有します。追加の実機CIにはアクセラレーションを利用できるrunnerが必要です。Browser観測と操作には汎用process所有の上に別の契約が必要です。

[常駐process runtime](product-specs/persistent-process-runtime.ja.md)は、argv直接実行、
専用可変状態、名前付きloopback TCP port、保守的なnative tree cleanupを実装しています。
最終native integration受け入れは3 OSで成功し、証拠を[完了process plan](exec-plans/completed/persistent-process-runtime.ja.md)
に記録しました。自己daemon化、自動再起動、対話terminal、remote実行、service導入は
このruntime契約の対象外です。

[Android UI observer](product-specs/android-ui-observer.ja.md) は、既存の所有 Emulator に対し、
上限付きの意味情報 snapshot、PNG、Unicode 置換、navigation、現在の PID の log を提供します。
独立した任意の platform companion を使い、対象アプリへの instrumentation 追加は不要です。
OCR、visual regression、より豊富な gesture、物理デバイス、remote Emulator host は引き続き延期しています。
observer の最終受け入れ確認と platform 別の証拠は[完了した計画](exec-plans/completed/android-ui-observer.ja.md)で管理します。

Compose provider選択とPodman adapterは
実装済みであり、証拠は[provider plan](exec-plans/completed/compose-provider-podman.ja.md)に
記録しています。
既定はDockerのままで、自動fallbackはありません。Podman 5.4.2 / podman-compose
1.6.0で、Docker共存を含む実Linux rootless受け入れが成功しました。Windows/macOS/Linuxのnative provider CIは
4a5de3d（run 34216579481）で成功であり、実機のMachine環境はありません。`podman compose` wrapper、
Quadlet/Kubernetes、pod作成、任意のprovider実行ファイルは今回の対象外です。

## 成果物とリリース

ローカルOCI registry、image promotion、image保持参照、厳密な成果物のreplayは延期しています。runtime検査は実際のコンテナーimageの識別情報を記録しますが、再現可能なimage promotionではありません。成果物の自動期限切れ、event圧縮、migration rollbackツール、生成CLI/JSON Schemaリファレンス、長期的なhandoff archive方針は未決です。

初回の配布は[スタンドアロン仕様](product-specs/standalone-distribution.ja.md)に従う GitHub Release アーカイブです。リリース実装と直接のネイティブ検証の証拠は[リリース計画](exec-plans/completed/standalone-release-finalization.ja.md)で管理します。署名、notarization、package manager 向け定義、自己更新、SBOM、attestation は後続作業です。リリースはテストしたrevisionのネイティブ検証と統合検証の証拠を引用する必要があります。対応build targetはその証拠の代わりにはなりません。

## CIの拡張

Windows/macOSのネイティブDocker統合には、Dockerが使えるself-hosted runnerが必要になる場合があります。Android統合には適切なhardware accelerationが必要です。`last_verified`からの文書の経過日数は、現在CIが強制する鮮度の期限ではありません。metadataの妥当性と文書を見つけられることは検査で強制します。
