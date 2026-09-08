---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/product-specs/manifest-v1.md
source_sha256: 0beec1c56873393b4258083b163c27c7c0f5d53bc6d7dc78407f9169708ad786
---

[English（翻訳元）](manifest-v1.md)

# Manifest v1

`.agent-env.yaml` は厳密な単一文書 YAML 契約です。未知の field、重複 key、未対応 version、必須 collection が空であること、不正な名前、依存関係 cycle、不在の source/runtime/component/stack への参照を拒否します。`agent-env validate <file-or-repository>` は Git や Compose provider を実行せずに契約を確認します。

## 完全な例

この例は、信頼済みのローカルリポジトリに `db`、`api`、`dashboard` サービスを持つ `compose.yaml` があることを前提にします。named test はそのリポジトリが Go project であると仮定します。endpoint 宣言はコンテナの target port に対する動的 loopback 公開の生成を要求します。元の Compose file に `ports` は不要です。

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

この named test を呼び出す前に、明示的に要求したホストの `TEST_TOKEN` 変数を設定してください。リポジトリで不要なら `env` の該当項目を削除します。`artifacts` が空でも stdout、stderr、run 記述子は保持されます。report を収集するにはリポジトリ相対の出力パスを追加します。

## Field

| 位置 | Field と動作 |
| --- | --- |
| Root | `version: 1` と空でない `sources`、`runtimes`、`components`、`stacks` が必須。`applications` と `tests` は任意 |
| `sources.<alias>` | ローカル `repository` が必須。`default_ref`（空なら Git HEAD）、`writable`（review mode では false のみ対応）は任意 |
| `runtimes.<name>` | `type` と `source` が必須。Compose は空でない `files` が必須で `project_directory` と `provider` は任意。Android は `type: android-emulator` と `avd` が必須で Compose field は不可。process は `working_directory` と argv の `command` が必須で、`env` と名前付きTCP `ports` は任意 |
| `applications.<name>` | `type: flutter-android`、`source`、Android の `runtime`、`build.command`、`build.artifact`、`package`、`activity`。`project_directory`、`build.timeout`、`reverse` は任意。[Flutter 契約](flutter-android-runtime.ja.md)を参照 |
| `components.<name>` | `runtime` が必須。Compose は空でない `compose_services` が必須。Android では Compose services/endpoints/readiness を省略し、同じ runtime の `application` を選択可能。process はCompose servicesを省略し`runtime_port` endpointを使う。`depends_on`、`provides`、適用可能な `readiness` は任意 |
| `stacks.<name>` | 空でない `roots` が必須。`description` は任意 |
| `tests.<name>` | `stack`、`source`、空でない argv `command` が必須。`working_directory`、文字列 map の `env`、`timeout`、`artifacts` は任意 |

alias と名前は英数字で始まり、英数字、dot、underscore、hyphen のみを含みます。source repository path はローカルです。リモート URL、provider 固有 PR 省略記法、自動 fetch、資格情報管理は未対応です。

相対 source repository は control repository を基準に解決します。

runtime file と project directory は、ランタイムに割り当てた source root を基準に解決します。file は `project_directory` 相対ではありません。

test/probe の working directory と artifact path は、宣言した source root を基準に解決します。

移植可能な manifest path は forward slash を使い、シンボリックリンク経由も含め root 外に出てはいけません。ローカル source repository の絶対パスは、一致するホストプラットフォーム上で許可します。

`--source alias=ref` は 1 つの source の default ref を上書きします。`--ref` は単一 source の manifest でのみ有効です。起動前に全 source を不変の commit に解決します。ソートした alias/repository/commit の組が source-set digest を決めます。canonical manifest とその SHA-256 digest は source 識別情報とは別に保持します。

選択コンポーネントは決定的な依存順序に従います。Compose のサービス依存関係の閉包も含めます。実行する正規化設定には選択サービスと、そこから到達できる network、volume、config、secret だけを含め、未選択のグローバルリソースが削除対象へ入り込むのを防ぎます。

Compose runtimeは任意の`provider: docker-compose`または`provider: podman-compose`を
受け付けます。省略時はDockerです。明示的な空文字列・null、未知の値、Android/processのprovider
fieldはエラーです。planとlease snapshotには実効providerを記録し、省略したfieldを
canonical manifestへ追加しません。providerを持たない旧snapshotはDockerとして扱います。
選択したproviderから他engineへのfallbackはありません。PodmanにはPodman 5.xと独立した
podman-compose >=1.6.0,<2.0.0が必要です。5.4.2 / 1.6.0で実Linux rootless受け入れが
成功しました。Windows/macOS/Linuxのnative CIは4a5de3d（run 34216579481）で成功しました。
実機のMachine環境はありません。
[provider契約](compose-providers.ja.md)を参照してください。

## Readiness probe

コンポーネントの `readiness` は次の形式の probe のリストです。

| Type | 必須 field | 任意 field |
| --- | --- | --- |
| `compose` | `type` | `timeout`、`interval` |
| `http` | `type`、資格情報を含まない HTTP(S) `url` | `timeout`、`interval` |
| `command` | `type`、`source`、argv `command` | `working_directory`、`timeout`、`interval` |

期間は `500ms`、`10s`、`2m` などの正の Go duration 文字列にします。選択した Compose probe の宣言は、最短の宣言 timeout と最速の interval を組み込み既定値の 2 分/1 秒と組み合わせ、ランタイム全体の readiness に上限を与えます。別の probe type に属する field はエラーです。稼働コンテナに container healthcheck が定義されていれば healthy である必要があり、選択した全サービスが存在しなければなりません。HTTP probe は時間制限付き観測内の成功応答を要求します。command probe は create 中に固定 source 内で実行し、出力証拠を保持します。通常の list/show はリポジトリコマンドを再実行しません。

ComposeのHTTP URLは引き続き明示設定したliteral URLです。コンテナ内readinessにはCompose healthcheckを使い、host portはendpoint観測で取得します。process HTTP readinessでは同じcomponentの宣言済みendpointを`${endpoint:localName}`で参照でき、数値の予約portへ展開します。例は`http://127.0.0.1:${endpoint:http}/health`です。host:port全体への展開ではありません。未知の参照やlocal範囲を外れた参照は拒否します。process command readinessの引数ではendpointとprocess引数の参照を使えます。

## Endpoint

Composeでは`components.<name>.endpoints` は endpoint 名を `service`、整数 `target`（1〜65535）、任意の `protocol`（既定の `tcp` または `udp`）へ対応付けます。service はそのコンポーネントが直接選択している必要があります。起動前に各宣言から、その service/target/protocol に対する host port `0` の `127.0.0.1` binding を正規化実行 snapshot に生成します。元の Compose file は変更しません。実際のホストポートは選択したengineが決めます。remote Podmanのmappingはagent-envホストからの到達性確認が必要であり、未確認のendpointは公開しません。これは、入力設定で policy が拒否する固定ホストポートを許可するものではありません。観測は割り当てポートを resource metadata に記録し、runtime の service/port/protocol 観測とともに、コンポーネント名で修飾した endpoint 名を `capabilities` で公開します。不在または停止した binding は、実際に使える endpoint を提供しません。

## Named command の環境と artifact

コマンドは引数配列で、shell command string にはしません。named test の argv と環境値では、明示的な `${env:NAME}`、`${lease_id}`、`${android:<runtime>:serial}` の置換に対応します。ホスト変数の欠如、未知の式、不正な置換はコマンド実行前に失敗します。Android serial は選択済みでリースの所有権を確認できた runtime のみ解決します。shell 展開、pipeline、一般的な template 評価は提供しません。

資格情報らしい環境名（token、password、secret、key）は、機密値の literal ではなく、正確な `${env:NAME}` 参照を使わなければなりません。

展開した設定資格情報と認識した継承資格情報は、保存出力、argv 記録、コピーした artifact から伏せ字にします。完全な環境は証拠に書きません。

認識した継承資格情報が、argv や資格情報でない環境 key を含め canonical manifest のどこかに literal として現れた場合は、予約前に拒否します。

Compose の資格情報は、永続化する実行 snapshot の資格情報入り解決済み環境値ではなく、secret file で設定してください。コンテナ絶対パスの `*_FILE` 参照に対応します。

artifact 項目は明示的な source 相対の file/directory path で、glob pattern ではありません。範囲外に出るシンボリックリンクと通常 file 以外を拒否します。収集 artifact は環境削除後も残ります。元の source 出力は管理対象 worktree の削除時に消えることがあります。

## 複数リポジトリと拡張

`sources` に別のローカル項目を加え、runtime、test、command probe からその alias を参照します。選択 runtime が 1 つでも、宣言した全 source を固定し materialize します。[統合 fixture](../../internal/cli/integration_test.go)が、別々のローカルリポジトリと alias ごとの commit override を検証します。

[Android Emulator ランタイム](android-emulator.ja.md)は `avd` でローカル AVD テンプレートを選び、Flutter と独立して専用の書き込み状態を割り当てます。plan に SDK は不要で、device を割り当てません。[Flutter アプリケーション](flutter-android-runtime.ja.md)は、任意の `applications` とコンポーネントの `application` フィールドで、ホスト上で APK をビルドし、それらの runtime にインストールします。[常駐process runtime](persistent-process-runtime.ja.md)は`type: process`を使い、固定sourceからforegroundのnative commandを実行します。browser/CDP、書き込み可能なfix lease、remote source cacheは[延期対象](../roadmap.ja.md)です。それらの提案fieldは有効なmanifest v1 YAMLではありません。

## Manifest の由来と readiness の上限

control repository は `.agent-env.yaml` または明示的な `--manifest` file を選びます。runtime source ref の override によって別の manifest を選ぶことはありません。plan と lease は、canonical snapshot digest に加え、解決した絶対パス、取得できれば control checkout HEAD、modified flag を記録します。dirty、未追跡、ignored の file は modified とし、Git 外なら commit は空で診断を付けます。相対 `sources.*.repository` path は、明示 manifest file の directory ではなく指定した control repository を基準にします。

Compose readiness probe の timeout と interval は、全体 readiness の期限と polling interval を厳しくできます。アプリケーションのグローバル readiness timeout は上限のままです。HTTP と command probe も、操作全体の期限内で設定した正の期間を守ります。キャンセルは観測を止め、所有権喪失状態での削除を防ぎます。

## 常駐processのvariant

process runtimeには`type: process`、`source`、literalの`working_directory`、argvの
`command`が必要です。`env`と`ports`は任意です。各portには`protocol: tcp`の明示が必須で、
固定host portとUDPは拒否します。process専用fieldはCompose/Androidで無効です。
`provider`、`project_directory`、`files`、`avd`はnull/空でもprocessで無効となります。
process名は大文字小文字の衝突、Windows device名、末尾dotも拒否します。

commandの引数/環境変数値には`${runtime_dir}`、`${lease_id}`、`${port:name}`、`${env:NAME}`
を使えます。未知の参照はerrorです。実行ファイル名とcwdの補間は禁止です。
process componentは`compose_services`を省略し、宣言したruntime portを
`endpoints.<name>.runtime_port`で参照します。このvariantではCompose用の
service/target/protocol fieldを禁止します。共通の観測endpoint mapは引き続き`host:port`を
持ち、数値のreadiness参照とは区別します。foregroundの寿命、専用可変状態、実行ファイル証拠、
保守的なcleanupは[process契約全文](persistent-process-runtime.ja.md)を参照してください。

## 明示的browser binding

任意の`browsers.<name>`に`type: chromium-cdp`、`process` runtime、名前付きTCP `cdp_port`を
宣言します。名前の大文字・小文字衝突は禁止し、1 runtimeにつきbindingは1つです。
対象リポジトリはheadless、automation、loopback debugging、`${runtime_dir}/profile`の
正確なflagをnative argvに1回ずつ宣言します。空/null bindingと、保護対象switchの別表記・重複は
検証で拒否します。port名からbrowserを推測しません。
完全な[browser manifest契約](browser-cdp-automation.ja.md)を参照してください。
