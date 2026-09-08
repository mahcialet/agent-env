---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: README.md
source_sha256: d017e68009f47cef7e3142e3f40d1b0e3ec4dd3838d5f3deff6af53aabf3ad8c
---

[英語版（翻訳元）](README.md)

# agent-env

固定したローカルGitコミットと、隔離されたDocker Composeプロジェクトまたは専用Android Emulatorから、使い捨ての環境リースを作成します。stackを選び、稼働状態を確認し、証拠を保持する名前付きテストを実行して、最後にリソースを片付けます。複数リポジトリと同時に存在する複数リースに対応します。

**環境の隔離は、悪意あるコードを封じ込めるsandboxではありません。** Dockerfile、Compose設定、テスト、パッケージスクリプトは、リポジトリが制御するコードを実行します。信頼できる、または管理下にあるリポジトリを使用してください。任意の信頼できないpull requestには、より強い外側の境界が必要です。

## スタンドアロンアーカイブ

GitHub Releases から OS と CPU に合うアーカイブを取得し、バージョン付き
ディレクトリを展開します。`agent-env`（Windows では `agent-env.exe`）を直接実行するか、
そのディレクトリを PATH に追加します。`agent-env version --output json` はリリース
バージョン、ソースコミット、ビルドに使った Go のバージョン、プラットフォームを表示します。
実行に Go や checkout は不要です。ソースからのビルドには Go が必要です。
リリースの公開状況とネイティブ検証の証拠は
[リリース計画](docs/exec-plans/completed/standalone-release-finalization.ja.md)で管理します。

| 機能 | 外部の前提条件 |
| --- | --- |
| version、help、基本診断 | なし。Go、シェル、Docker、SDK、Flutter、Java は不要 |
| ソース解決と管理対象 worktree | Git と信頼できるローカルリポジトリ |
| Compose リース | Git、Docker daemon、Compose plugin |
| Android Emulator リース | Git、Android SDK、Emulator、adb、インストール済み system image/AVD テンプレート、ホストのアクセラレーション |
| Flutter Android アプリ | Android の前提条件に加え、Flutter と互換性のある Java/Android ビルドツールチェーン |
| Android UI 観測 | Android リースと別途ビルドした任意の UI companion。そのビルドには SDK/JDK と Go が必要 |
| リリースの作成 | Git と対応する Go ツールチェーン。リリース CI は Go 1.27.1 に固定 |

機能ごとの外部ツールと任意の UI companion はアーカイブに同梱しません。
任意の前提ツールがなくても version/help は実行できます。
[配布仕様](docs/product-specs/standalone-distribution.ja.md)を参照してください。

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

対象の`.agent-env.yaml`に、sources、runtimes、任意のapplications、components、stacks、名前付きargvテストを宣言します。[マニフェストリファレンス](docs/product-specs/manifest-v1.ja.md)に完全な例があります。ルートにComposeファイルがちょうど1つある単純なリポジトリでは、`init`が既存ファイルを上書きせずに候補マニフェストを作成します。実行前に、選択されたサービスとホストポリシーを確認してください。

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

コンポーネントのendpointを宣言すると、ソースのComposeファイルを編集せずに、保存される実行設定へ動的なloopbackホスト公開設定を生成できます。Composeリソースはプロジェクト単位に限定され、mountはホストポリシーを満たす必要があります。固定コンテナー名、privileged mode、host networking、Docker socketのmount、安全でない外部bindは拒否されます。

名前付きテストはargv配列を使い、stdout、stderr、終了ステータス、宣言した成果物はcleanup後も保持されます。[CLI契約](docs/product-specs/cli-contract.ja.md)と[セキュリティポリシー](docs/SECURITY.ja.md)を参照してください。

`gc`は期限切れ候補をプレビューし、削除を要求するのは`gc --apply`だけです。追跡対象の変更、所有権の不確定、不完全なcleanupがあるリースはquarantinedになります。明示的な`destroy --force`は、追跡対象の編集を破棄する前に差分証拠を保持し、所有権の不一致を上書きすることはありません。

## Android Emulatorリース

`type: android-emulator`、`source: app`、`avd: <installed-template>`を持つruntimeを宣言し、Compose servicesを持たないコンポーネントから参照します。SDKの前提条件は`doctor --runtime android-emulator`で、稼働状態は`doctor <lease-id>`で確認します。AndroidのみのstackにはDockerは不要です。各リースは専用の書き込み可能なAVD状態と予約済みのconsole/ADBポートペアを持ち、`show`でserialを確認できます。完全なマニフェストと復旧ルールは[Android契約](docs/product-specs/android-emulator.ja.md)を参照してください。

[Flutter Androidアプリケーション](docs/product-specs/flutter-android-runtime.ja.md)は、固定ソースからのAPKビルド、所有Emulatorへのインストール、バックエンドへのreverse設定、Activity起動に対応します。任意の `applications` を宣言し、コンポーネントから選択します。`doctor <repository> --runtime flutter-android` で設定済みFlutter実行ファイル、プロジェクト、Androidの前提条件を確認できます。互換性のあるFlutter・Java・Androidビルドツールチェーンが必要です。

## Android UI を観測する

[Android UI observer](docs/product-specs/android-ui-observer.ja.md) は、所有する Emulator の
accessibility snapshot と PNG を取得し、古い参照を拒否する意味情報に基づく tap、Unicode の
text 置換、現在の PID に範囲を限定した logcat 収集を行います。対象アプリに test 依存を追加せず、
Flutter semantics と native Android UI を扱えます。UI 観測で runtime を作成したり、manifest を変更したりしません。

インストール済み SDK/JDK を使って任意の companion を一度 build し、host の環境変数設定で
`AGENT_ENV_UI_HELPER` に生成先 directory を指定します。出力 directory は新規である必要があります。
例で指定する version は事前にインストールされている必要があり、builder は tool のインストールや
license 受諾を行いません。通常の Go build や関係のないコマンドに companion や JDK は不要です。

```text
go run ./tools/uihelper --sdk <sdk> --jdk <jdk> --platform android-35 --build-tools 36.0.0 --output <new-directory>
go run ./cmd/agent-env ui snapshot <lease-id> --application mobile-app
go run ./cmd/agent-env ui screenshot <lease-id> --application mobile-app
go run ./cmd/agent-env ui tap <lease-id> --snapshot <snapshot-id> --node n7
go run ./cmd/agent-env ui set-text <lease-id> --snapshot <snapshot-id> --node n3 --text <replacement>
go run ./cmd/agent-env ui logcat <lease-id> --application mobile-app --since 30s
```

意味情報に基づく参照は一つの snapshot に属します。対象が変化した場合や曖昧な場合は新しい snapshot が必要で、
入力を自動再実行することはありません。text 置換には、focus のある編集可能な node と読み戻しの一致確認が必要です。
保持する text 証拠では編集可能な値を伏せますが、PNG のピクセルには秘密情報が含まれ得ます。
helper run が中断した場合、cleanup 前に `ui recover <lease-id> --run <run-id>` が必要になる場合があります。
復旧では失敗または結果不確実という outcome を保持し、入力は再試行しません。
上限、状態の制約、navigation、wait、復旧の詳細は仕様を参照してください。

## 状態と制限

状態は対象リポジトリの外に保存されます。`AGENT_ENV_HOME`に絶対パスを指定すると、OS標準の保存先（LinuxのXDG state、macOSのApplication Support、WindowsのLOCALAPPDATA）を上書きできます。このhomeには`state.db`、管理対象worktree、正規化したruntime設定、リースの成果物、診断用の`leases/<id>/environment.json`記述子が入ります。リース状態の判断では、診断用記述子よりSQLiteの記録を優先します。既定のTTLは4時間、最大TTLは24時間、有効な予約数の上限は8です。quarantinedのリースは予約を保持します。ホストポリシー設定ファイルはまだ公開していません。

iOS、browser/CDPの自動操作、リモートGitキャッシュ、registry promotion、書き込み可能な修正リースは[ロードマップ項目](docs/roadmap.ja.md)です。

貢献者は[AGENTS.md](AGENTS.md)と[文書索引](docs/index.ja.md)から始めてください。既存の[MITライセンス](LICENSE)を適用します。
