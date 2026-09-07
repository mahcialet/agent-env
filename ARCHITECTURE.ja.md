---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: ARCHITECTURE.md
source_sha256: a237aa4ced0b7de068cdb6bc9fdd83993c417a0119d52a3f7289ddd761ca50e5
---

[English（正本）](ARCHITECTURE.md)

# アーキテクチャ

このシステムは、固定したローカルGitソースと選択したコンポーネントの依存閉包を、ComposeまたはAndroid Emulatorのリソースを持つ環境リースとして実体化します。[MVP仕様](docs/product-specs/agent-env-mvp.ja.md)が振る舞いを定義し、[完了済み計画](docs/exec-plans/completed/agent-env-mvp.md)が実装済みの責務境界と検証証拠を記録します。

CLIは引数を解析し、出力を整形して、ユースケースをappに委譲します。domainの型は、具体的なアダプターに依存せず、リース、不変のソース集合、コンポーネント、リソース、イベントをモデル化します。configはマニフェストを厳密にデコードし、stackは決定的な依存閉包を解決します。appはソースとruntimeのインターフェース、ポリシー、準備完了判定、証拠、補償cleanupを調整します。

SQLiteは期待状態、予約、所有権、ソースの識別情報、イベント履歴を管理します。Gitソースプロバイダーはrefを解決し、detached worktreeを作成し、追跡対象の変更を検査して、安全なworktreeを削除します。Compose runtimeアダプターは正規化した設定を検証し、選択したサービスを作成し、リソースを検査し、証拠を収集して、明示的なプロジェクト識別情報に基づき破棄します。reconcileはレジストリの意図とGit・Dockerの観測結果を比較します。保存されたreadyの行を、稼働中の健全な環境と同一視しません。

## 依存方向

- domainはCLI、SQLite、Git、Composeのアダプターをimportしてはいけません。
- appはdomainとインターフェースを利用できますが、CLIの出力整形に依存してはいけません。
- runtimeアダプターはCLIをimportしてはいけません。
- storeは永続化を実装し、appのオーケストレーション方針を担ってはいけません。
- CLIはライフサイクルの振る舞いをappに委譲します。具体的な依存の接続はアプリケーション境界に置きます。

これらの境界は、違反を検出する負例fixtureを伴うrepoctl arch-checkで検査する必要があります。そのバリデーターが通るまでは、境界が機械的に強制されていると主張しません。依存グラフを変更する際は、この構成図、チェッカー、ADRまたは計画の決定記録を一緒に更新します。

## 横断的な不変条件

引数配列とプラットフォーム固有のパスを使い、Windowsラッパーの処理はexecxに隔離します。リリースビルドはCGOもshellも必要としません。状態は対象リポジトリの外にあるOS標準の状態パスに保存し、AGENT_ENV_HOMEで上書きできます。SQLiteはローカルで使用し、外部キー、busy timeout、検証済みのWALを備えます。

割り当ては、権限を持つ別々の管理主体にまたがるsagaです。外部作用の前に意図を保存し、結果を記録し、逆順で補償します。追跡対象のソースが変更されている場合、識別情報が曖昧な場合、cleanupが不完全な場合は、イベントと成果物を伴うquarantined状態を維持します。リースの隔離は偶発的な衝突を防ぐものであり、悪意あるコードを封じ込めるsandboxではありません。

開発harnessと対象マニフェストは別物です。[エージェント向け指示](AGENTS.md)、索引付き文書、計画、repoctl、CIはこのリポジトリを説明し、`.agent-env.yaml`は対象リポジトリの起動方法を説明します。

Android Emulatorリソースは、独立した`app.AndroidProvider`と`internal/runtime/android`アダプターを通じてComposeに加わります。このアダプターはdomainの識別情報、appの観測結果、`execx`のネイティブプロセス境界を使い、Composeをimportしません。SQLiteはAVDとポートの排他的予約を担い、appは補償と準備完了判定を担います。detached processと実行時間を制限したコマンドのプロセスツリーは別の仕組みです。[Android設計](docs/design-docs/android-emulator.ja.md)と[完了済みの実行証拠](docs/exec-plans/completed/android-emulator-lease.md)を参照してください。ブラウザーとFlutterの統合は別の作業です。
