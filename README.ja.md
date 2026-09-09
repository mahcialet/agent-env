---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: README.md
source_sha256: e0a6b0816c263a90f41ba0ea8544e6bcfccc5122ddfaa4b17ef228d32528d2b7
---

# agent-env

[English](README.md)

agent-envは、commitを固定したGitソースから、開発やテスト用の使い捨て環境を作成します。
環境の管理単位を**lease**と呼びます。leaseは独立したruntimeリソースを所有し、テストの証拠を
保持します。元のcheckout内のfileを編集せずに、状態の確認と後片付けを行えます。
複数のリポジトリを使う場合や、複数のleaseを同時に動かす場合にも対応しています。

runtimeにはComposeサービス、専用Android Emulator、フォアグラウンドのnative processを
選べます。Flutterアプリ、Android UIの観測、ブラウザ自動操作は、それらのruntime上で動作します。
既定はローカル実行です。任意のcontrollerを使い、登録済みworkerにlease全体を配置することもできます。

**環境の分離は、悪意あるコードを閉じ込めるsandboxではありません。** Dockerfile、Compose設定、
テスト、package scriptは対象リポジトリが指定したコードを実行します。信頼できる、または管理下に
あるリポジトリを使ってください。任意の未信頼PRを実行するには、外側により強い分離境界が必要です。
詳しくは[セキュリティ方針](docs/SECURITY.ja.md)を参照してください。

## 配布archiveから使う

GitHub ReleasesからOSとarchitectureに合うarchiveを取得し、バージョン名付きディレクトリを
展開します。中の`agent-env`（Windowsでは`agent-env.exe`）を直接実行するか、そのディレクトリを
PATHへ追加してください。

```text
agent-env version --output json
agent-env --help
```

version出力ではrelease version、ソースcommit、ビルド時のGo version、platformを確認できます。
実行にGoやリポジトリのcheckoutは不要です。外部runtimeツールと任意のAndroid UI companionは
同梱していません。任意のツールがなくてもversionとhelpは使えます。Pythonもコアの依存ではありません。

archiveの内容とrelease条件は[配布仕様](docs/product-specs/standalone-distribution.ja.md)、
releaseの提供状況とnative検証の記録は[完了済みrelease Plan](docs/exec-plans/completed/standalone-release-finalization.ja.md)
で確認できます。

## 使いたい機能を選ぶ

| 目的 | 必要な外部ツール・環境 | 詳細と次の手順 |
| --- | --- | --- |
| ソースを解決し、管理対象worktreeを作る | Gitと信頼できるローカルリポジトリ | [manifest仕様](docs/product-specs/manifest-v1.ja.md) |
| コンテナサービスを動かす | Git、Docker daemonとCompose v2。またはPodman 5.xとstandalone podman-compose >=1.6.0,<2.0.0 | [Compose provider](docs/product-specs/compose-providers.ja.md) |
| native serverを継続して動かす | Gitと宣言したnative実行ファイル。daemonやSDKは不要 | [永続process](docs/product-specs/persistent-process-runtime.ja.md) |
| ブラウザを観測・操作する | process leaseと、直接実行できる対応headless Chromium系ブラウザ | [Browser/CDP操作](docs/product-specs/browser-cdp-automation.ja.md) |
| Android Emulatorを動かす | Git、Android SDK、Emulator、adb、導入済みsystem image、停止中のAVD template、ホストの仮想化支援 | [Android Emulator lease](docs/product-specs/android-emulator.ja.md) |
| Flutter Androidアプリをビルド・起動する | Androidの前提条件、Flutter、互換性のあるJava/Android build toolchain | [Flutterアプリ](docs/product-specs/flutter-android-runtime.ja.md) |
| Android UIを観測・操作する | Android leaseと任意のUI companion。companionのビルドにはSDK/JDKとGo | [UIの準備・コマンド・復旧](docs/product-specs/android-ui-observer.ja.md) |
| 別ホストでleaseを動かす | 事前に用意したTLS証明書、ソース転送用Git、選択したruntimeに必要なworker側のツール | [controller・client・workerの準備](docs/product-specs/multi-host-control-plane.ja.md) |

コンテナの既定providerはDocker Composeです。Podmanは明示的に選び、選択結果をleaseに固定します。
自動fallbackは行いません。processやAndroidだけのstackにDockerは不要です。ブラウザ操作は宣言済み
processを使い、別のブラウザを起動しません。UI観測も既存の所有Emulatorを使うため、runtimeを新規作成
したりmanifestを変更したりしません。

各仕様では、実装済みの挙動、対象外の機能、検証済み環境を区別しています。未検証環境を含む
受け入れ証拠の詳細は[品質方針](docs/QUALITY.ja.md)と[移植性](docs/PORTABILITY.ja.md)を参照してください。

## 信頼できるリポジトリで使う

`.agent-env.yaml`にsources、runtimes、任意のapplications、components、stacks、名前付きのargvテストを
記述します。[manifest仕様](docs/product-specs/manifest-v1.ja.md)に完全な例があります。
リポジトリ直下にCompose fileが1つだけある場合、`init`は既存fileを上書きせずmanifest候補を作れます。
実行前に、選択されたサービスとホストのポリシーを確認してください。

実行ファイルをPATHに置いたら、次のリポジトリ、stack、テスト、lease IDを自分の値に置き換えます。

```text
agent-env doctor ../trusted-repo
agent-env validate ../trusted-repo
agent-env plan ../trusted-repo --stack api
agent-env create ../trusted-repo --stack api --ref HEAD
agent-env list --output json
agent-env show <lease-id>
agent-env capabilities <lease-id>
agent-env test <lease-id> api-smoke
agent-env destroy <lease-id> --dry-run
agent-env destroy <lease-id>
```

`plan`はcommitを解決しますが、リソースは確保しません。`plan`と`create`の`--manifest <path>`で
信頼できる制御用manifestを明示できます。runtimeのfileは引き続き固定したソースから取得します。
複数リポジトリのrefを上書きする場合は`--source alias=ref`を使います。componentのendpointにより、
元のCompose fileを編集せずloopbackへ動的に公開できます。コマンドの挙動と復旧は
[CLI仕様](docs/product-specs/cli-contract.ja.md)を参照してください。

名前付きテストはargv配列で実行し、cleanup後もstdout、stderr、終了status、宣言したartifactを保持します。
UI・ブラウザのsnapshotに基づく操作には、最新で一意に特定できる対象が必要です。入力を自動再実行することは
ありません。保持するテキストを秘匿化しても、PNGには秘密情報が写り得ます。remoteのUI/browser `set-text`は
永続化して送信する前に拒否されます。テキスト入力にはlocal modeを使ってください。

## 状態の保存と制約

状態は対象リポジトリの外へ保存します。既定の保存先はLinuxのXDG state、macOSのApplication Support、
WindowsのLOCALAPPDATAです。`AGENT_ENV_HOME`に絶対パスを設定して変更できます。保存先にはSQLiteの
`state.db`、管理対象worktree、正規化したruntime設定、artifact、診断用の
`leases/<id>/environment.json`があります。永続状態の判断基準はSQLiteであり、診断用descriptorは
その代わりにはなりません。

既定TTLは4時間、最大TTLは24時間、同時に保持する予約の上限は8です。隔離中のleaseも予約を保持します。
ホストポリシーを設定するfileはまだ公開していません。Composeのリソースはproject内に限定する必要があります。
固定container名、privileged mode、host networking、Docker socket mount、安全でない外部bindは拒否します。

`gc`は期限切れ候補のpreviewです。削除を要求するには`gc --apply`を明示します。追跡fileの変更、
不明な所有権、未完了のcleanupがあるleaseは隔離します。`destroy --force`で追跡fileの変更を破棄する場合も、
先にdiffの証拠を保持します。所有権の不一致を無視することはできません。

remote modeのendpointはworker上にあり、clientへのtunnelはありません。workerは操作を順番に処理しますが、
作成済みleaseは並行稼働できます。同じleaseで操作が実行中の場合は、テスト中のdestroyも含めて別の操作を拒否します。
localのforceやGCでcontroller管理を迂回することはできません。準備、ソース転送、登録、artifact、対応操作は
[remote仕様](docs/product-specs/multi-host-control-plane.ja.md)を参照してください。

## ソースからのビルドと検証

開発にはGo 1.26.xまたは1.27.xを使います。通常のunit検査にBash、Make、PowerShell、Dockerは不要です。

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go build ./cmd/agent-env
go run ./tools/repoctl test-integration
```

最後のコマンドは、Linuxで実Docker fixtureを明示的に実行します。native OSテストやCGO無効のcross-buildは
別の検証です。このcheckoutからなら、使用例の`agent-env`を`go run ./cmd/agent-env`に置き換えても実行できます。
releaseの作成にはGitとGoが必要です。release CIはGo 1.27.1に固定しています。

開発の入口は[AGENTS.md](AGENTS.ja.md)です。目的別の資料は[文書index](docs/index.ja.md)、
iOS、remote Git cache、image promotion、変更を残すfix leaseなどの未実装事項は
[roadmap](docs/roadmap.ja.md)にあります。ライセンスは既存の[MIT license](LICENSE)です。
