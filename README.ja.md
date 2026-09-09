---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: README.md
source_sha256: b76b06723bbcc9532cb0508b05c5b92098c7872ce7d94f985c6fd6e6c302cea3
---

[英語版（翻訳元）](README.md)

# agent-env

固定したローカルGitコミットと、DockerまたはPodmanを使う隔離されたComposeプロジェクト、専用Android Emulator、foregroundのnative process runtimeから、使い捨ての環境リースを作成します。stackを選び、稼働状態を確認し、証拠を保持する名前付きテストを実行して、最後にリソースを片付けます。複数リポジトリと同時に存在する複数リースに対応します。

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
| Docker Compose リース（既定） | Git、Docker daemon、Compose v2 plugin |
| Podman Compose リース | Git、Podman 5.x、独立したpodman-compose >=1.6.0,<2.0.0。5.4.2 / 1.6.0でLinux rootless受け入れを検証済み |
| 常駐processリース | Gitと宣言したnative実行ファイル。container daemonやSDKは不要 |
| Browser/CDP自動操作 | process leaseと、直接起動できる互換headless Chromium系browser。同梱しない |
| Android Emulator リース | Git、Android SDK、Emulator、adb、インストール済み system image/AVD テンプレート、ホストのアクセラレーション |
| Flutter Android アプリ | Android の前提条件に加え、Flutter と互換性のある Java/Android ビルドツールチェーン |
| Android UI 観測 | Android リースと別途ビルドした任意の UI companion。そのビルドには SDK/JDK と Go が必要 |
| 複数 host の controller/client/worker role | 事前に用意した TLS 証明書、source 転送用 Git、選択 runtime 用の worker ツール |
| リリースの作成 | Git と対応する Go ツールチェーン。リリース CI は Go 1.27.1 に固定 |

機能ごとの外部ツールと任意の UI companion はアーカイブに同梱しません。Pythonはagent-env coreの依存関係ではありません。
任意の前提ツールがなくても version/help は実行できます。
[配布仕様](docs/product-specs/standalone-distribution.ja.md)を参照してください。

## ビルドと検証

Go 1.26.xまたは1.27.xを使用します。runtime操作にはGitに加え、選択したruntimeの前提条件が必要です。コンテナーならCompose v2を備えたDocker、またはpodman-composeを備えたPodman、Androidならインストール済みのAndroid SDK、Emulator、adb、停止したAVDテンプレートを、process runtimeなら宣言したnative実行ファイルを用意します。リポジトリharnessの通常の単体検査にはBash、Make、PowerShell、Dockerは不要です。

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

Compose runtimeには`provider: docker-compose`（省略時の既定値）または
`provider: podman-compose`を指定できます。選択はleaseごとに固定し、ツールがなくても
fallbackしません。`doctor --provider podman-compose`はそのproviderを検査し、
`doctor <repository>`はmanifestで宣言したproviderを検査します。Podman 5.4.2と
podman-compose 1.6.0で、並行leaseとDocker共存を含む実Linux rootless受け入れが
成功しました。Windows/macOS/Linuxのnative provider CIは4a5de3d（run 34216579481）で成功で、実機のPodman Machine環境はありません。[provider契約](docs/product-specs/compose-providers.ja.md)を
参照してください。

## 常駐processリース

`type: process`に固定`source`、`working_directory`、native argvの`command`を宣言します。
任意の名前付きTCP portと`${runtime_dir}`を使うと、Composeなしでlocal serverや専用profile
状態を管理できます。process診断には`doctor --runtime process`を使います。
foreground processはcreate CLI終了後も存続し、readiness、show/logs、保守的destroyに
参加します。予期しない終了でも再起動しません。
[process契約](docs/product-specs/persistent-process-runtime.ja.md)を参照してください。
native integration受け入れはWindows・macOS・Linuxで成功しました。
[完了済みExecPlan](docs/exec-plans/completed/persistent-process-runtime.ja.md)を参照してください。

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

iOS、リモートGitキャッシュ、registry promotion、書き込み可能な修正リースは[ロードマップ項目](docs/roadmap.ja.md)です。

貢献者は[AGENTS.md](AGENTS.md)と[文書索引](docs/index.ja.md)から始めてください。既存の[MITライセンス](LICENSE)を適用します。

## Browser/CDP自動操作

`browsers.<name>`からprocess runtimeと名前付きTCP CDP portを明示的に参照します。
manifestにheadless、automation、loopback debugging、専用profileの正確なflagを宣言し、
browser層は別processを起動しません。`browser pages`、`snapshot`、`screenshot`、`navigate`と、
snapshotに限定したsemantic入力を使えます。console/network captureには上限があり、
profileとartifactはprivateです。PNG pixelの自動redactionはしません。
完全なmanifestとコマンドは[browser契約](docs/product-specs/browser-cdp-automation.ja.md)、
native受け入れ状況は[完了plan](docs/exec-plans/completed/browser-cdp-automation.ja.md)を参照してください。
Chromeは外部の前提ツールであり、同梱しません。

## 明示的な remote モード

任意の [複数 host control plane](docs/product-specs/multi-host-control-plane.ja.md) は、
lease 全体を登録済み worker 一つに配置します。既定の local モードに controller は不要です。
CA、controller の DNS 名に有効な server 証明書、別々の client/worker 証明書を事前に用意します。
controller、各 worker、client ごとに、host の環境変数設定で別々の絶対 path の `AGENT_ENV_HOME` を
指定してください。enrollment は controller の状態 root に対する offline の管理コマンドです。
controller 起動前にその状態 root で実行します。

```text
agent-env control-plane enroll --certificate client.pem --role client
agent-env control-plane enroll --certificate worker.pem --role worker --host-id build-a
agent-env --tls-ca ca.pem --tls-cert controller.pem --tls-key controller.key control-plane serve --listen 0.0.0.0:9443
```

worker では専用の状態 root と runtime の前提環境を用意して実行します。

```text
agent-env --controller https://controller.example:9443 --tls-ca ca.pem --tls-cert worker.pem --tls-key worker.key worker serve --host-id build-a --max-leases 2
```

client では controller URL、証明書、commit 済み repository を自分のものに置き換えます。

```text
agent-env --controller https://controller.example:9443 --tls-ca ca.pem --tls-cert client.pem --tls-key client.key hosts list
agent-env --controller https://controller.example:9443 --tls-ca ca.pem --tls-cert client.pem --tls-key client.key create ../trusted-repo --stack api --host build-a
agent-env --controller https://controller.example:9443 --tls-ca ca.pem --tls-cert client.pem --tls-key client.key artifact-download <digest> --destination evidence.json
```

list/show、renew、reconcile、destroy、名前付き test、対応する UI/browser 操作にも同じ接続 flag を使います。
artifact digest は登録済みの remote 証拠から取得します。download 時に digest を検証し、保存先には新しいファイルを
指定します。`hosts drain <host-id>` は新規配置を止め、`hosts undrain <host-id>` は配置対象に戻します。
commit 済み source は検証済み Git bundle で転送し、client の path や暗黙の環境変数の秘密値を worker 入力にしません。
loopback endpoint は worker を指し、client への tunnel はありません。

worker は操作を直列に実行しますが、作成済み lease は並行して稼働できます。
同じ lease に対する二つ目の active 操作は、remote test 中の destroy も含めて拒否します。
remote の実行中操作の cancellation は未実装です。local force/GC で controller の管理を回避できません。
実 TLS の受け入れ検証は、各 runner の二つの worker root を使い、`440082b` の Windows・macOS・Linux で
成功しました（run 34320519252）。配置・再起動に加え、名前付き test、log、artifact download、
期限更新、環境変数の分離を検証しています。物理的な複数 host・VM の検証は未実施であり、
合意した範囲の受け入れ検証は完了しました。[品質](docs/QUALITY.ja.md) と
[完了Plan](docs/exec-plans/completed/multi-host-control-plane.ja.md) を参照してください。
