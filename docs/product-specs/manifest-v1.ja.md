---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/product-specs/manifest-v1.md
source_sha256: fd152d1aa78e4d1b24f24898bdcfcc88c385e32a9e2c79d3e571b625eae56e14
---

[English（翻訳元）](manifest-v1.md)

# Manifest v1

`.agent-env.yaml` には、リースに必要なソース、ランタイムのリソース、コンポーネント間の依存関係を宣言します。
この文書は実装済みのバージョン 1 形式を定めます。まず Compose の例を読み、使いたい
ランタイムの形式へ進んでください。

ファイルは一つの YAML 文書でなければなりません。未知のフィールド、キーの重複、
非対応バージョン、空の必須集合、不正な名前、依存関係の循環、存在しない
source/runtime/component/stack への参照を拒否します。
`agent-env validate <file-or-repository>` は Git や Compose プロバイダーを実行せずに検証します。

## 完全な例

この例は、信頼済みのローカルリポジトリに `db`、`api`、`dashboard` サービスを持つ `compose.yaml` があることを前提にします。named テストはそのリポジトリが Go project であると仮定します。接続先宣言はコンテナの target ポートに対する動的ループバック公開の生成を要求します。元の Compose ファイルに `ports` は不要です。

```yaml
version: 1
sources:
  backend:
    repository: .
    default_ref: HEAD
runtimes:
  backend:
    type: compose
    source: backend
    project_directory: .
    files: [compose.yaml]
components:
  api:
    runtime: backend
    compose_services: [db, api]
    provides: [api, logs]
    readiness:
      - type: compose
        timeout: 2m
        interval: 1s
      - type: command
        source: backend
        working_directory: .
        command: [git, rev-parse, --verify, HEAD]
        timeout: 10s
        interval: 1s
    endpoints:
      http:
        service: api
        target: 8080
        protocol: tcp
  dashboard:
    runtime: backend
    compose_services: [dashboard]
    depends_on: [api]
    provides: [web-ui]
    endpoints:
      web:
        service: dashboard
        target: 8081
stacks:
  api:
    description: API and its database
    roots: [api]
  dashboard:
    description: API plus Dashboard
    roots: [dashboard]
tests:
  api-smoke:
    stack: api
    source: backend
    working_directory: .
    command: [go, test, ./...]
    env:
      TEST_TOKEN: "${env:TEST_TOKEN}"
      LEASE: "${lease_id}"
    timeout: 5m
    artifacts: []
```

この named テストを呼び出す前に、明示的に要求したホストの `TEST_TOKEN` 変数を設定してください。リポジトリで不要なら `env` の該当項目を削除します。`artifacts` が空でも stdout、stderr、run 記述子は保持されます。report を収集するにはリポジトリ相対の出力パスを追加します。

## Field

| 位置 | Field と動作 |
| --- | --- |
| Root | `version: 1` と空でない `sources`、`runtimes`、`components`、`stacks` が必須。`applications`、`browsers`、`tests` は任意 |
| `sources.<alias>` | ローカル `repository` が必須。`default_ref`（空なら Git HEAD）、`writable`（review mode では false のみ対応）は任意 |
| `runtimes.<name>` | `type` と `source` が必須。Compose は空でない `files` が必須で `project_directory` と `provider` は任意。Android は `type: android-emulator` と `avd` が必須で Compose フィールドは不可。プロセスは `working_directory` と引数配列の `command` が必須で、`env` と名前付きTCP `ports` は任意 |
| `applications.<name>` | `type: flutter-android`、`source`、Android の `runtime`、`build.command`、`build.artifact`、`package`、`activity`。`project_directory`、`build.timeout`、`reverse` は任意。[Flutter 契約](flutter-android-runtime.ja.md)を参照 |
| `browsers.<name>` | `type: chromium-cdp`、プロセスの `runtime`、名前付き TCP `cdp_port`。後述の明示的なブラウザー設定を参照 |
| `components.<name>` | `runtime` が必須。Compose は空でない `compose_services` が必須。Android では Compose services/endpoints/readiness を省略し、同じランタイムの `application` を選択可能。プロセスはCompose servicesを省略し`runtime_port` 接続先を使う。`depends_on`、`provides`、適用可能な `readiness` は任意 |
| `stacks.<name>` | 空でない `roots` が必須。`description` は任意 |
| `tests.<name>` | `stack`、`source`、空でない引数配列 `command` が必須。`working_directory`、文字列 map の `env`、`timeout`、`artifacts` は任意 |

### 名前とソースリポジトリ

alias と名前は英数字で始まり、英数字、dot、underscore、hyphen のみを含みます。ソースリポジトリパスはローカルです。リモート URL、プロバイダー固有 PR 省略記法、自動 fetch、資格情報管理は未対応です。

### パスの解決基準

相対ソースリポジトリは control リポジトリを基準に解決します。

ランタイムファイルと project ディレクトリは、ランタイムに割り当てたソースルートを基準に解決します。ファイルは `project_directory` 相対ではありません。

test/probe の working ディレクトリと成果物パスは、宣言したソースルートを基準に解決します。

移植可能なマニフェストパスは forward slash を使い、シンボリックリンク経由も含めルート外に出てはいけません。ローカルソースリポジトリの絶対パスは、一致するホストプラットフォーム上で許可します。

### ソースの固定とマニフェストの識別

`--source alias=ref` は 1 つのソースの default ref を上書きします。`--ref` は単一ソースのマニフェストでのみ有効です。起動前に全ソースを不変の commit に解決します。ソートした alias/repository/commit の組が source-set ダイジェストを決めます。正規化したマニフェストとその SHA-256 ダイジェストはソース識別情報とは別に保持します。

### 選択するコンポーネントとサービス

選択コンポーネントは決定的な依存順序に従います。Compose のサービス依存関係の閉包も含めます。実行する正規化設定には選択サービスと、そこから到達できるネットワーク、ボリューム、config、秘密情報だけを含め、未選択のグローバルリソースが削除対象へ入り込むのを防ぎます。

### Compose プロバイダーの選択

Compose ランタイムは任意の`provider: docker-compose`または`provider: podman-compose`を
受け付けます。省略時はDockerです。明示的な空文字列・null、未知の値、Android/processのプロバイダー
フィールドはエラーです。planとリーススナップショットには実効プロバイダーを記録し、省略したフィールドを
正規化したマニフェストへ追加しません。プロバイダーを持たない旧スナップショットはDockerとして扱います。
選択したプロバイダーから他エンジンへのfallbackはありません。PodmanにはPodman 5.xと独立した
podman-compose >=1.6.0,<2.0.0が必要です。対応構成、実環境での検証結果、
Machine 環境が未検証である点は [Compose プロバイダー仕様](compose-providers.ja.md)で確認できます。

## 常駐processのvariant

プロセスランタイムには`type: process`、`source`、リテラルの`working_directory`、引数配列の
`command`が必要です。`env`と`ports`は任意です。各ポートには`protocol: tcp`の明示が必須で、
固定ホストポートとUDPは拒否します。プロセス専用フィールドはCompose/Androidで無効です。
`provider`、`project_directory`、`files`、`avd`はnull/空でもプロセスで無効となります。
プロセス名は大文字小文字の衝突、Windows device名、末尾dotも拒否します。

コマンドの引数/環境変数値には`${runtime_dir}`、`${lease_id}`、`${port:name}`、`${env:NAME}`
を使えます。未知の参照はエラーです。実行ファイル名とcwdの補間は禁止です。
プロセスコンポーネントは`compose_services`を省略し、宣言したランタイムポートを
`endpoints.<name>.runtime_port`で参照します。このvariantではCompose用の
service/target/protocol フィールドを禁止します。共通の観測接続先 mapは引き続き`host:port`を
持ち、数値の起動確認参照とは区別します。フォアグラウンドの寿命、専用可変状態、実行ファイル証拠、
保守的なクリーンアップは[process契約全文](persistent-process-runtime.ja.md)を参照してください。

## 明示的browser binding

任意の`browsers.<name>`に`type: chromium-cdp`、`process` ランタイム、名前付きTCP `cdp_port`を
宣言します。名前の大文字・小文字衝突は禁止し、1 ランタイムにつき関連付けは1つです。
対象リポジトリはheadless、automation、ループバック debugging、`${runtime_dir}/profile`の
正確なフラグをネイティブ引数配列に1回ずつ宣言します。空または null の関連付けと、保護対象スイッチの別表記・重複は
検証で拒否します。ポート名からブラウザーを推測しません。
完全な[browser manifest契約](browser-cdp-automation.ja.md)を参照してください。

## Readiness probe

コンポーネントの `readiness` は次の形式の probe のリストです。

| Type | 必須フィールド | 任意フィールド |
| --- | --- | --- |
| `compose` | `type` | `timeout`、`interval` |
| `http` | `type`、資格情報を含まない HTTP(S) `url` | `timeout`、`interval` |
| `command` | `type`、`source`、引数配列 `command` | `working_directory`、`timeout`、`interval` |

期間は `500ms`、`10s`、`2m` などの正の Go duration 文字列にします。選択した Compose probe の宣言は、最短の宣言タイムアウトと最速の interval を組み込み既定値の 2 分/1 秒と組み合わせ、ランタイム全体の起動確認に上限を与えます。別の probe type に属するフィールドはエラーです。稼働コンテナにコンテナー healthcheck が定義されていれば healthy である必要があり、選択した全サービスが存在しなければなりません。HTTP probe は時間制限付き観測内の成功応答を要求します。コマンド probe は create 中に固定ソース内で実行し、出力証拠を保持します。通常の list/show はリポジトリコマンドを再実行しません。

ComposeのHTTP URLは引き続き明示設定したリテラル URLです。コンテナ内起動確認にはCompose healthcheckを使い、ホストポートは接続先観測で取得します。プロセス HTTP 起動確認では同じコンポーネントの宣言済み接続先を`${endpoint:localName}`で参照でき、数値の予約ポートへ展開します。例は`http://127.0.0.1:${endpoint:http}/health`です。host:port全体への展開ではありません。未知の参照やローカル範囲を外れた参照は拒否します。プロセスコマンド起動確認の引数では接続先とプロセス引数の参照を使えます。

## Endpoint

Composeでは`components.<name>.endpoints` は接続先名を `service`、整数 `target`（1〜65535）、任意の `protocol`（既定の `tcp` または `udp`）へ対応付けます。サービスはそのコンポーネントが直接選択している必要があります。起動前に各宣言から、その service/target/protocol に対するホストポート `0` の `127.0.0.1` 関連付けを正規化実行スナップショットに生成します。元の Compose ファイルは変更しません。実際のホストポートは選択したエンジンが決めます。リモート Podman の TCP ポートの対応付けには agent-env ホストからの到達性確認が必要であり、未確認の接続先は公開しません。これは、入力設定で policy が拒否する固定ホストポートを許可するものではありません。観測は割り当てポートをリソースメタデータに記録し、ランタイムの service/port/protocol 観測とともに、コンポーネント名で修飾した接続先名を `capabilities` で公開します。不在または停止した関連付けは、実際に使える接続先を提供しません。

## Named command の環境と artifact

### 値の置換規則

コマンドは引数配列で、shell コマンド string にはしません。named テストの引数配列と環境値では、明示的な `${env:NAME}`、`${lease_id}`、`${android:<runtime>:serial}` の置換に対応します。ホスト変数の欠如、未知の式、不正な置換はコマンド実行前に失敗します。Android シリアルは選択済みでリースの所有権を確認できたランタイムのみ解決します。shell 展開、pipeline、一般的なテンプレート評価は提供しません。

### 秘密情報の扱い

資格情報らしい環境名（token、password、秘密情報、キー）は、機密値のリテラルではなく、正確な `${env:NAME}` 参照を使わなければなりません。

展開した設定資格情報と認識した継承資格情報は、保存出力、引数配列記録、コピーした成果物から伏せ字にします。完全な環境は証拠に書きません。

認識した継承資格情報が、引数配列や資格情報でない環境キーを含め正規化したマニフェストのどこかにリテラルとして現れた場合は、予約前に拒否します。

Compose の資格情報は、永続化する実行スナップショットの資格情報入り解決済み環境値ではなく、秘密情報ファイルで設定してください。コンテナ絶対パスの `*_FILE` 参照に対応します。

### 成果物のパスと保持

成果物項目は明示的なソース相対の file/directory パスで、glob pattern ではありません。範囲外に出るシンボリックリンクと通常ファイル以外を拒否します。収集成果物は環境削除後も残ります。元のソース出力は管理対象ワークツリーの削除時に消えることがあります。

## 複数リポジトリと拡張

`sources` に別のローカル項目を加え、ランタイム、テスト、コマンド probe からその alias を参照します。選択ランタイムが 1 つでも、宣言した全ソースを固定し materialize します。[統合 fixture](../../internal/cli/integration_test.go)が、別々のローカルリポジトリと alias ごとの commit override を検証します。

[Android Emulator ランタイム](android-emulator.ja.md)は `avd` でローカル AVD テンプレートを選び、Flutter と独立して専用の書き込み状態を割り当てます。plan に SDK は不要で、device を割り当てません。[Flutter アプリケーション](flutter-android-runtime.ja.md)は、任意の `applications` とコンポーネントの `application` フィールドで、ホスト上で APK をビルドし、それらのランタイムにインストールします。[常駐process runtime](persistent-process-runtime.ja.md)は`type: process`を使い、固定ソースからフォアグラウンドのネイティブコマンドを実行します。[Browser/CDP](browser-cdp-automation.ja.md) は、実装済みの明示的な `browsers` 設定を使います。
書き込み可能な fix リースとリモートソース cache は[延期対象](../roadmap.ja.md)です。
それらの提案フィールドは有効なマニフェスト v1 YAML ではありません。

## Manifest の由来と readiness の上限

control リポジトリは `.agent-env.yaml` または明示的な `--manifest` ファイルを選びます。ランタイムソース ref の override によって別のマニフェストを選ぶことはありません。plan とリースは、正規化したスナップショットダイジェストに加え、解決した絶対パス、取得できれば control checkout HEAD、modified フラグを記録します。dirty、未追跡、ignored のファイルは modified とし、Git 外なら commit は空で診断を付けます。相対 `sources.*.repository` パスは、明示マニフェストファイルのディレクトリではなく指定した control リポジトリを基準にします。

Compose 起動確認 probe のタイムアウトと interval は、全体起動確認の期限とポーリング interval を厳しくできます。アプリケーションのグローバル起動確認タイムアウトは上限のままです。HTTP とコマンド probe も、操作全体の期限内で設定した正の期間を守ります。キャンセルは観測を止め、所有権喪失状態での削除を防ぎます。
