---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/product-specs/cli-contract.md
source_sha256: 5a8c2af67f25516a2498679f724abc8e87fae4b311879d43eb7dd1df2f46394b
---

[English（翻訳元）](cli-contract.md)

# CLI 契約

この文書は、実装済みコマンドの使い方、出力形式、終了コード、ローカルリースの復旧条件を定めます。
対象リポジトリの設定は [Manifest v1](manifest-v1.ja.md)、リモート構成と権限・期限・
操作の実行順序の違いは[複数ホストの仕様](multi-host-control-plane.ja.md)を参照してください。

初期の OS 別検証は [MVP 実装計画](../exec-plans/completed/agent-env-mvp.md)に記録しています。
後から追加した機能の仕様と検証記録には、以下の各機能から進めます。

## リースを操作するコマンド

```text
agent-env version
agent-env init [repository]
agent-env validate [repository-or-manifest]
agent-env plan [repository] --stack <name> [--manifest <path>] [--ref <ref>] [--source alias=ref]
agent-env create [repository] --stack <name> [--manifest <path>] [--ref <ref>] [--source alias=ref] [--ttl <duration>] [--purpose <text>] [--mode review]
agent-env list [--cached] [--mine] [--state <state>]
agent-env show <lease-id>
agent-env capabilities <lease-id>
agent-env test <lease-id> <test-name>
agent-env logs <lease-id> [--component <name>] [--run <run-id>]
agent-env renew <lease-id> [--ttl <duration>]
agent-env destroy <lease-id> [--dry-run] [--force]
agent-env reconcile [lease-id]
agent-env gc [--apply]
agent-env doctor [repository|lease-id] [--runtime compose|process|android-emulator|flutter-android] [--provider docker-compose|podman-compose]
```

### リポジトリとソースの選択

リポジトリ省略時は現在のディレクトリを使います。`init` は認識可能なルート Compose ファイルが 1 つあることを要求し、`.agent-env.yaml` を排他的に新規作成して review が必要と報告します。何も起動しません。

`validate` は Docker や Podman なしでスキーマと参照を確認します。`plan` はさらにローカル Git commit と決定的なコンポーネント閉包を解決し、状態やワークツリーは作成しません。

`--ref` はソースが 1 つの場合に限ります。複数ソースには alias ごとの `--source` override を使います。どちらもリモート ref を fetch しません。

マニフェストは既定で指定した control checkout から取得します。plan/create の `--manifest` は別の信頼済みマニフェストを明示的に選び、そのダイジェストと正規化したスナップショットを保持します。review mode は契約上書き込み不可の detached ソースワークツリーを作成します。書き込み可能な fix mode はありません。

## 識別情報、状態、観測

### 所有者ラベル

`--owner <text>` はグローバルな参考所有者 selector です。`AGENT_ENV_OWNER` が明示的な既定値を与えます。指定しない場合はローカル user/host の識別情報と一意の接尾辞で割り当てを識別し、`list --mine` は別の呼び出しでもそのローカル識別情報に一致させます。owner label は filter であり、認可境界ではありません。

### 実リソースの観測と一覧

`list`と`show`は記録したソース、プロバイダーを固定したCompose project、所有する常駐プロセスの識別情報を検査します。`--cached` はレジストリのみを読む明示的な list です。`show` はリース、event、コマンド run、成果物、接続先の観測を含みます。

`capabilities` は選択コンポーネントが宣言する capability、観測状態、接続先のアドレス map を報告します。宣言された接続先は、保存したランタイム設定に動的ループバック公開を生成します。

`reconcile <lease-id>` は観測したリースを返します。全体の `reconcile` は `leases` と `inventory` を含む object を返し、一致する所有者記録のないリソースも含めます。削除はしません。

### 記録する状態の意味

desired 状態は `active` または `released` です。observed 状態は `requested`、`allocating`、`starting`、`ready`、`degraded`、`failed`、`releasing`、`released`、`quarantined`、`unknown` です。保存済みの ready 行は実際の健全性の証明ではありません。Compose リソースの不在や常駐プロセスの予期しない終了はactive リースをdegradedにし、検査失敗は不確定として見える状態を保ちます。期限切れまたは隔離中のリースを、留保のない ready 結果にはできません。

ランタイム/コンテナと時間制限付き HTTP 起動確認は再観測できます。任意のコマンド probe は create 中に実行し、list/show はリポジトリコマンドを再実行しません。したがって実リソースの健全な観測は、以前成功したコマンド probe が現在も成功する証明にはなりません。

### ログの選択

`logs --run` は記録した 1 つのコマンド run の stdout/stderr を読みます。`logs --component`はCompose ランタイム内のコンポーネントのサービスを選びます。プロセスコンポーネントではランタイムのファイル出力stdout/stderrを選びます。filter がなければ、利用可能なランタイムのログと保持済みの削除証拠を含めます。

グローバルinventoryは、導入済みプロバイダー実行ファイルと記録済みprovider/engine識別情報を
併せて探索し、リソース IDをプロバイダー単位に区別します。導入済みでもエンジンが利用不能なら、
部分的なinventoryと明示的なエラーを返します。

## 出力と終了 status

`--output table` が既定の人間向け表示です。一部の詳細コマンドは table の代わりにインデント付き object を表示します。全コマンドが次の envelope による `--output json` に対応します。

```json
{"schema_version":1,"data":{}}
```

各リースは、永続化される lifecycle 変更時に更新するバージョン付きの診断スナップショット `leases/<id>/environment.json` も持ちます。このスナップショットは診断用であり、リースの記録を管理するレジストリの代わりにはなりません。

data の形はコマンドごとに異なります。plan/create は object、list はリース配列、show はリースと証拠 collection、GC は `apply` と `leases` を返します。JSON mode では named テストのストリーム出力を stderr に送り、stdout を単一 JSON 文書に保ちます。create、削除、named テストが失敗しても記録済み結果を出力することがあります。プロセス終了 status と結果の両方を確認してください。

| Exit | 意味 |
| --- | --- |
| 0 | 要求された操作が成功 |
| 2 | コマンド、引数、マニフェスト、その他ユーザー入力が不正 |
| 3 | 前提条件が欠けている、または使用不能 |
| 4 | 割り当てまたは起動確認の失敗 |
| 5 | named テストの失敗、タイムアウト、キャンセル |
| 6 | 隔離を含め、安全に削除を完了できなかった |
| 7 | 内部、レジストリ、観測の失敗 |

これは CLI の終了 code です。named コマンド自身の終了 code は run 記録に別途保存します。

## 前提条件の診断

`doctor`はリポジトリもプロバイダーも明示しない場合、既定でDockerを検査します。リポジトリ指定時はマニフェストで宣言した全Compose プロバイダーを検査します。`--provider`は診断対象を1つ選択し、リースの実行設定は上書きしません。診断はプロバイダー情報、バージョン、エンジンの前提条件を保持します。PodmanにはPodman 5.xと独立したpodman-compose >=1.6.0,<2.0.0が必要であり、fallbackしません。`--provider`はCompose前提条件の診断専用で、別ランタイムやリース IDには適用できません。

`--runtime android-emulator` は Docker の代わりにローカル SDK ツールと AVD テンプレートを確認します。

`--runtime flutter-android` は Docker や Podman なしで既定の Flutter 実行ファイルと Android SDK/AVD の前提条件を検査します。リポジトリを指定すると、宣言した全アプリの設定済み実行ファイル、プロジェクト、Android の前提条件を検査します。[Flutter 契約](flutter-android-runtime.ja.md)を参照してください。

リース ID は実際のリースの診断を選び、ready でないリースは exit 3 を返します。[Android 契約](android-emulator.ja.md)を参照してください。

## 削除と更新

### 有効期間と更新

既定 TTL は 4 時間、最大 TTL は 24 時間、active な予約容量は 8 です。`renew --ttl` は現在からの相対値で期限を変更し、heartbeat を更新します。不在リソースの再起動はしません。quarantined、releasing、released のリースは更新できません。

### 削除と force 指定

`destroy --dry-run` はリソースを削除せず、現在の削除候補を返します。通常の削除は対象を厳密に特定した active named run ID にキャンセルを要求し、終了確認と証拠確定を待ってから削除用リースロックを取得します。無関係な active 操作は busy のままです。その後 source/runtime の所有権を確認し、サービスごとの最終ログを保持し、ランタイムを逆順に削除し、管理対象ワークツリーを削除してリースを released にします。

追跡対象ファイルに変更があるとリースを隔離します。`destroy --force` は、binary Git diff を成果物として保持した後の削除を明示的に許可します。それでも固定された所有権の一致と証拠収集の成功が必要です。管理対象 review ワークツリー内の未追跡の test/build 出力は、通常の削除で除去されることがあります。管理対象ワークツリーに無関係な作業を置かないでください。

### GC の対象と実行条件

`gc` は変更せず候補を preview します。既定 policy は期限切れから 5 分、最後の heartbeat から 1 分の経過を要求します。永続化された `running` コマンド行があるリースは、操作ロックが期限切れでも preview と apply の対象から除外します。`gc --apply` は各リースロックを再取得し、所有権/source の変更確認と削除の前に、これらの条件を再確認します。隔離中と処理中のリースは除外します。成果物の自動期限切れや、一般的なホストリソース一括削除はありません。

## Named test

### 実行できる条件

テストは固定されたマニフェストに存在し、要求するコンポーネントスタックがリース内に含まれなければなりません。リースは ready で期限内である必要があります。実行前後にソース識別情報と追跡対象に変更がないことを確認し、追跡対象を書き換えたテストは review リースを隔離します。

コマンドは引数配列と、割り当てたソース相対の working ディレクトリを使います。永続化された操作ロックが同時削除を排除し、実行の前後に heartbeat を更新します。各 run は名前、伏せ字処理した引数配列、ソース、working ディレクトリ、timestamp、exit code、status、stdout/stderr パス、JSON run 記述子、宣言成果物を保持します。ログは保存と同時にストリーム出力します。テストの失敗や要求のキャンセル後も証拠を保持します。

### 結果の確定と削除を阻止する条件

レジストリへ終了 status を記録するのは、プロセスツリー終了と必要な証拠確定を確認した後だけです。終了未確認エラーや stream/artifact/descriptor 確定の失敗では、永続化した run は `running` のままなので、destroy と GC は削除を拒否します。通常の非ゼロ終了、タイムアウト、確認済みキャンセルは、証拠保持後に終了結果にできます。ローカルの復旧記述子は、古いロック保持者にレジストリ行の確定を許可しません。

環境 map は run 記録にコピーしません。`${env:NAME}` はホスト変数を明示的に読み、`${lease_id}` はリース識別子を挿入します。資格情報らしい環境値のリテラルは割り当て前に拒否します。[manifest 詳細](manifest-v1.ja.md)と[セキュリティ上の限界](../SECURITY.ja.md)を参照してください。

## Android UI の観測

```text
agent-env ui snapshot <lease-id> [--application <name>|--runtime <name>] [--all-windows]
agent-env ui screenshot <lease-id> [--application <name>|--runtime <name>]
agent-env ui tap <lease-id> --snapshot <snapshot-id> --node <ref>
agent-env ui set-text <lease-id> --snapshot <snapshot-id> --node <ref> --text <value>
agent-env ui tap-coordinate <lease-id> --runtime <name> --x <x> --y <y>
agent-env ui back <lease-id> [--runtime <name>]
agent-env ui home <lease-id> [--runtime <name>]
agent-env ui swipe <lease-id> --runtime <name> --x <x> --y <y> --to-x <x> --to-y <y> [--duration <duration>]
agent-env ui wait <lease-id> --application <name> --contains <text> [--timeout <duration>]
agent-env ui logcat <lease-id> --application <name> [--since <duration>]
agent-env ui recover <lease-id> --run <run-id>
```

`ui` は、永続化された所有 Android ランタイムの識別情報を使い、任意のシリアルは受け付けません。
スナップショットが公開するのは Android accessibility の意味情報であり、Flutter widget ではありません。
application 指定時は既定でそのパッケージを対象とします。ランタイムだけを指定した場合や
`--all-windows` を指定した場合は system ウィンドウも含めます。JSON は既存の envelope を使い、
table 出力ではスナップショット内だけで有効なノード参照と成果物パスを表示します。
対象選択、状態の制約、stable エラー code、上限、プライバシーは[observer 仕様](android-ui-observer.ja.md)で定義します。

意味情報に基づく tap/set-text には、登録済みスナップショットと、現在の一意なフィンガープリントが必要です。
対象が古い場合や曖昧な場合は入力前にエラーを返し、座標へ fallback しません。
テキスト置換には、focus のある編集可能なノードと、読み戻しの一致確認が必要です。
生および正規化した観測結果、PNG、対象を絞ったログ、操作証拠をリース成果物に保持します。
PNG のピクセルはテキストのように秘密値を伏せられません。UI コマンドはクリーンアップと同じリース fence を使い、
リモート完了を確認できない場合は実行中コマンドの barrier を保持します。
`ui recover` は登録済みの中断された補助ツール操作だけを扱い、その同一性と不在を確認し、
復旧を記録してからその run の barrier を解除します。入力を再試行せず、中断された操作を成功とも扱いません。
復旧には、明示的に永続化された `termination-unconfirmed` の分類と、検証済みの元の結果証拠が必要です。
分類がない crash は引き続き拒否します。

UI エラーは共通の終了コード仕様に従います。前提条件の不足は 3、無効な option・対象選択・
存在しないリース・stale/ambiguous な参照は 2、registry と観測の障害は 7 を返します。
秘密値を伏せた後もエラーの型による分類を保ち、これらを区別できるようにします。

意味情報を使うコマンドでは、`AGENT_ENV_UI_HELPER` に検証済み companion ビルドディレクトリを指定します。
インストール済みツールを使い、`go run ./tools/uihelper --sdk <sdk> --jdk <jdk>
--platform android-35 --build-tools 36.0.0 --output <new-directory>` で明示的にビルドします。
出力ディレクトリは新規である必要があります。ホスト環境変数の設定方法は利用者が選択します。
対象マニフェストの変更や自動 download は不要です。

## キャンセルと manifest の由来

`destroy` は対象を厳密に特定した active named run ID にキャンセルを要求し、削除前に最大 10 秒、終了/証拠確定を待ちます。キャンセル未確認や所有者のない running 記録は `--force` を含め削除を阻止します。すでにそれらの named run と無関係な操作は busy のままです。待機中に始まった新しい run を destroy はキャンセルしません。

### マニフェストの由来

plan とリースは `manifest_path`、`manifest_commit`、`manifest_modified` を公開します。パスは実際に選択したファイル、commit はランタイムソース ref とは独立した control checkout の HEAD です。変更済み、未追跡、ignored のファイルは modified と記録します。Git 外では commit は空で、正規化したスナップショットとそのダイジェストが由来を示すことを診断で説明します。`--manifest` は相対ソースリポジトリパスの解決に使う control リポジトリを変更しません。

## 常駐processの診断とlogs

`doctor --runtime process`はDocker、Podman、Android SDKを要求せずプロセスランタイム診断を
選びます。`doctor <lease-id>`は記録したネイティブ識別情報を観測します。外部プロセスの取り込みや
自動再起動はしません。プロセスのみのlifecycleと個別リース観測にCompose エンジンは不要です。
全体inventoryでは導入済み/記録済みCompose プロバイダーの可用性を別途報告します。

プロセスランタイムの`show`はプロセススナップショットと観測接続先 addressを保持します。
プロセス起動確認内の`${endpoint:localName}`は数値ポートで、`capabilities`/endpoint出力は
共通の`host:port`表現です。`logs`は帰属するstdout/stderrを上限付きで読み、独立した起動前
`redaction.json`で秘密情報を伏せ字処理します。`launch.json`の作成成功には依存しません。ホスト環境を後で変えてもその証拠は失われません。
起動後に証拠が欠落・不一致なら出力を拒否します。最終プロセスログはクリーンアップ成果物として
保持できますが、専用の未加工ログや可変状態を無条件で成果物へexportするものではありません。
[process契約](persistent-process-runtime.ja.md)を参照してください。

## Browser/CDPコマンド

`browser`はリースの明示的関連付けに対し、capabilities、pages、page-create/page-close、navigate、
スナップショット、dom-snapshot、screenshot、click、set-text、キー、scroll、wait、コンソール、ネットワークを提供します。
`--browser`と`--page`で正確な対象を選び、意味情報による入力には登録済み`--snapshot`と`--node`が必要です。
共通の出力契約でrun、observation、成果物の証拠を返します。フラグ・上限・privacy・古い参照の検証・
入力不明時のクリーンアップ barrierは[browser契約](browser-cdp-automation.ja.md)を参照してください。

## 延期されたコマンド

Expand/shrink、書き込み可能な fork、checkpoint/reproduce、成果物 promotion は未実装です。placeholder の成功は返しません。[ロードマップ](../roadmap.ja.md)を参照してください。
