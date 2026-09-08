---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: ARCHITECTURE.md
source_sha256: ae62af9dd6876136c509b87974c17c1560946b1fe3ce98b8efeb5ff8858d803f
---

[英語版（翻訳元）](ARCHITECTURE.md)

# アーキテクチャ

このシステムは、固定したローカルGitソースと選択したコンポーネントの依存閉包を、ComposeまたはAndroid Emulatorのリソースを持つ環境リースとして実体化します。[MVP仕様](docs/product-specs/agent-env-mvp.ja.md)が振る舞いを定義し、[完了済み計画](docs/exec-plans/completed/agent-env-mvp.md)が実装済みの責務境界と検証証拠を記録します。

CLIは引数を解析し、出力を整形して、ユースケースをappに委譲します。domainの型は、具体的なアダプターに依存せず、リース、不変のソース集合、コンポーネント、リソース、イベントをモデル化します。configはマニフェストを厳密にデコードし、stackは決定的な依存閉包を解決します。appはソースとruntimeのインターフェース、ポリシー、準備完了判定、証拠、補償cleanupを調整します。

SQLiteは期待状態、予約、所有権、ソースの識別情報、イベント履歴を管理します。Gitソースプロバイダーはrefを解決し、detached worktreeを作成し、追跡対象の変更を検査して、安全なworktreeを削除します。Compose runtimeアダプターは正規化した設定を検証し、選択したサービスを作成し、リソースを検査し、証拠を収集して、明示的なプロジェクト識別情報に基づき破棄します。reconcileはレジストリの意図とGitと記録済みCompose providerの観測結果を比較します。保存されたreadyの行を、稼働中の健全な環境と同一視しません。

`internal/runtime/compose.Client`は同じpackage境界内の非公開Docker/Podman clientへ
処理を振り分けます。両者は共通policyとcanonical JSON snapshotを使います。Podmanの
子processは、記録済みengine引数を持って現在のagent-envバイナリのnative bridgeへ入り、
coreにshell wrapperやPython依存を追加しません。domainのruntime snapshotはprovider
情報と`cleanup_evidence`を保持します。appはdown前の所有証拠を永続化するため、中断後の
cleanupで、削除済みcontainerの接続を再構成する必要がありません。
[provider設計](docs/design-docs/compose-providers.ja.md)を参照してください。
実providerの受け入れは完了済みの実装証拠を参照してください。実機のPodman Machine環境は
利用できませんでした。

## 依存方向

- domainはCLI、SQLite、Git、Composeのアダプターをimportしてはいけません。
- appはdomainとインターフェースを利用できますが、CLIの出力整形に依存してはいけません。
- runtimeアダプターはCLIや他のruntimeアダプターをimportしてはいけません。
- storeは永続化を実装し、appのオーケストレーション方針を担ってはいけません。
- CLIはライフサイクルの振る舞いをappに委譲します。具体的な依存の接続はアプリケーション境界に置きます。

これらの境界は、違反を検出する負例fixtureを伴うrepoctl arch-checkで検査する必要があります。そのバリデーターが通るまでは、境界が機械的に強制されていると主張しません。依存グラフを変更する際は、この構成図、チェッカー、ADRまたは計画の決定記録を一緒に更新します。

## 横断的な不変条件

引数配列とプラットフォーム固有のパスを使い、Windowsラッパーの処理はexecxに隔離します。リリースビルドはCGOもshellも必要としません。状態は対象リポジトリの外にあるOS標準の状態パスに保存し、AGENT_ENV_HOMEで上書きできます。SQLiteはローカルで使用し、外部キー、busy timeout、検証済みのWALを備えます。

割り当ては、SQLite、Git、runtimeのそれぞれが管理する状態にまたがるsagaです。外部作用の前に意図を保存し、結果を記録し、逆順で補償します。追跡対象のソースが変更されている場合、識別情報が曖昧な場合、cleanupが不完全な場合は、イベントと成果物を伴うquarantined状態を維持します。リースの隔離は偶発的な衝突を防ぐものであり、悪意あるコードを封じ込めるsandboxではありません。

開発harnessと対象マニフェストは別物です。[エージェント向け指示](AGENTS.md)、索引付き文書、計画、repoctl、CIはこのリポジトリを説明し、`.agent-env.yaml`は対象リポジトリの起動方法を説明します。

Android Emulatorリソースは、独立した`app.AndroidProvider`と`internal/runtime/android`アダプターを通じて、Composeに加えて利用できます。このアダプターはdomainの識別情報、appの観測結果、`execx`のネイティブプロセス境界を使い、Composeをimportしません。SQLiteはAVDとポートの排他的予約を担い、appは補償と準備完了判定を担います。detached processと実行時間を制限したコマンドのプロセスツリーは別の仕組みです。[Android設計](docs/design-docs/android-emulator.ja.md)と[完了済みの実行証拠](docs/exec-plans/completed/android-emulator-lease.md)を参照してください。ブラウザー統合は別の作業です。

Flutterのビルドには `app.FlutterProvider` と独立した
`internal/runtime/flutter` アダプターを使います。Androidのパッケージ確認、インストール、
reverse、起動は `app.AndroidApplicationProvider` を通じ、既存のAndroidアダプターが実装します。
両アダプターは互いやComposeをimportせず、appが順序と補償を担います。
アプリ・ビルド・reverseの追加記録は既存のLease JSON保存を利用します。
[Flutter設計](docs/design-docs/flutter-android-runtime.ja.md)と
[ADR 0005](docs/adr/0005-separate-flutter-applications.ja.md)を参照してください。

Android UI の観測には `app.AndroidUIProvider` を使います。既存の Android adapter と、
`internal/runtime/android/uihelper` にある任意の自己対象 companion がこれを実装します。
app は対象選択、古い参照の扱い、operation fencing、復旧、artifact 公開を担います。
domain は直列化できる UI 値を定義し、adapter は accessibility/ADB の副作用と helper の同一性を担います。
操作意図と cleanup barrier には既存の `CommandRun` row を使い、SQL migration や対象 manifest の
section は追加しません。`tools/uihelper` が native tool の argv を使って companion を明示的に build します。
runtime adapter 同士の import はありません。[observer 設計](docs/design-docs/android-ui-observer.ja.md)を参照してください。

## スタンドアロンリリースの責務境界

`internal/buildinfo` は実行ファイルの識別情報を公開し、`internal/assets` は digest を
検証する汎用のファイル配置を担当します。Android や Flutter の lifecycle は扱いません。
現在の CLI は runtime companion の資産を埋め込みません。Android UI helper は
明示的に別途ビルドする外部入力です。`tools/repoctl` は Git リリース条件の検証、
CGO 無効のクロスビルド、アーカイブの正規化、checksum、manifest 検証、
展開した実行ファイルのネイティブ smoke test を担当します。リリースのメタデータは
lease/domain のモデルに入れません。GitHub Actions はこれらのコマンドを呼び出し、
検証済みのバイト列を公開します。別のパッケージ生成処理は持ちません。
[配布設計](docs/design-docs/standalone-distribution.ja.md)を参照してください。
