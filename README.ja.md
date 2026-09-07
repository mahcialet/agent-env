---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: README.md
source_sha256: 9c90ad8c71412b0c3523e2a8218895c7ed0d4285f4ec1ccc1d947993712a0fe7
---

[English（正本）](README.md)

# agent-env

固定したローカルGitコミットと、隔離されたDocker Composeプロジェクトまたは専用Android Emulatorから、使い捨ての環境リースを作成します。stackを選び、稼働状態を確認し、証拠を保持する名前付きテストを実行して、最後にリソースを片付けます。複数リポジトリと同時に存在する複数リースに対応します。

**環境の隔離は、悪意あるコードを封じ込めるsandboxではありません。** Dockerfile、Compose設定、テスト、パッケージスクリプトは、リポジトリが制御するコードを実行します。信頼できる、または管理下にあるリポジトリを使用してください。任意の信頼できないpull requestには、より強い外側の境界が必要です。

## ビルドと検証

Go 1.26.xまたは1.27.xを使用します。runtime操作にはGitに加え、選択したruntimeの前提条件が必要です。コンテナーならComposeを備えたDocker、Androidならインストール済みのAndroid SDK、Emulator、adb、停止したAVDテンプレートを用意します。リポジトリharnessの通常の単体検査にはBash、Make、PowerShell、Dockerは不要です。

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go build ./cmd/agent-env
go run ./tools/repoctl test-integration
```

最後のコマンドは、Linux上で実際のDocker fixtureを明示的に実行します。Windows/macOS/Linuxのネイティブ単体CIと、CGOを無効にした5つのビルド対象は、Docker統合の検証範囲とは別です。完了した受け入れ確認と検証証拠は、[実装計画](docs/exec-plans/completed/agent-env-mvp.md)、[品質ガイド](docs/QUALITY.ja.md)、[移植性の注意事項](docs/PORTABILITY.ja.md)に記録しています。

## 信頼できるリポジトリを使う

対象の`.agent-env.yaml`に、sources、runtimes、components、stacks、名前付きargvテストを宣言します。[マニフェストリファレンス](docs/product-specs/manifest-v1.ja.md)に完全な例があります。ルートにComposeファイルがちょうど1つある単純なリポジトリでは、`init`が既存ファイルを上書きせずに候補マニフェストを作成します。実行前に、選択されたサービスとホストポリシーを確認してください。

次のコマンドは、このcheckoutから実行します。リポジトリパス、stack、名前付きテストは自分の値に置き換えてください。

```text
go run ./cmd/agent-env doctor ../trusted-repo
go run ./cmd/agent-env validate ../trusted-repo
go run ./cmd/agent-env plan ../trusted-repo --stack api
go run ./cmd/agent-env create ../trusted-repo --stack api --ref HEAD
go run ./cmd/agent-env list --output json
go run ./cmd/agent-env show <lease-id>
go run ./cmd/agent-env capabilities <lease-id>
go run ./cmd/agent-env test <lease-id> api-smoke
go run ./cmd/agent-env destroy <lease-id> --dry-run
go run ./cmd/agent-env destroy <lease-id>
```

実行ファイルをビルドしてPATHに追加すれば、`agent-env`（Windowsでは`agent-env.exe`）として使用できます。`plan`はリソースを割り当てずにコミットを解決します。`plan`と`create`は`--manifest <path>`で信頼する制御用マニフェストを明示的に選択できます。runtimeのファイルは引き続き各固定ソースから取得します。複数リポジトリのref上書きには`--source alias=ref`を使います。

コンポーネントのendpointを宣言すると、ソースのComposeファイルを編集せずに、保存される実行設定へ動的なloopbackホスト公開設定を生成できます。Composeリソースはプロジェクト単位に限定され、mountはホストポリシーを満たす必要があります。固定コンテナー名、privileged mode、host networking、Docker socketのmount、安全でない外部bindは拒否されます。名前付きテストはargv配列を使い、stdout、stderr、終了ステータス、宣言した成果物はcleanup後も保持されます。[CLI契約](docs/product-specs/cli-contract.ja.md)と[セキュリティポリシー](docs/SECURITY.ja.md)を参照してください。

`gc`は期限切れ候補をプレビューし、削除を要求するのは`gc --apply`だけです。追跡対象の変更、所有権の不確定、不完全なcleanupがあるリースはquarantinedになります。明示的な`destroy --force`は、追跡対象の編集を破棄する前に差分証拠を保持し、所有権の不一致を上書きすることはありません。

## Android Emulatorリース

`type: android-emulator`、`source: app`、`avd: <installed-template>`を持つruntimeを宣言し、Compose servicesを持たないコンポーネントから参照します。SDKの前提条件は`doctor --runtime android-emulator`で、稼働状態は`doctor <lease-id>`で確認します。AndroidのみのstackにはDockerは不要です。各リースは専用の書き込み可能なAVD状態と予約済みのconsole/ADBポートペアを持ち、`show`でserialを確認できます。完全なマニフェストと復旧ルールは[Android契約](docs/product-specs/android-emulator.ja.md)を参照してください。FlutterビルドとAPKインストールは別の作業です。

## 状態と制限

状態は対象リポジトリの外に保存されます。`AGENT_ENV_HOME`に絶対パスを指定すると、OS標準の保存先（LinuxのXDG state、macOSのApplication Support、WindowsのLOCALAPPDATA）を上書きできます。このhomeには`state.db`、管理対象worktree、正規化したruntime設定、リースの成果物、診断用の`leases/<id>/environment.json`記述子が入ります。正本はSQLiteです。既定のTTLは4時間、最大TTLは24時間、有効な予約数は8です。quarantinedのリースは予約を保持します。ホストポリシー設定ファイルはまだ公開していません。

Flutter、browser/CDP、リモートGitキャッシュ、registry promotion、書き込み可能な修正リースは[ロードマップ項目](docs/roadmap.ja.md)です。

貢献者は[AGENTS.md](AGENTS.md)と[文書索引](docs/index.ja.md)から始めてください。既存の[MITライセンス](LICENSE)を適用します。
