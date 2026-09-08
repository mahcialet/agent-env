---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/product-specs/cli-contract.md
source_sha256: 7fa565d69b622b25c520c527a7ed3158661b2ccb733b33cbe6881d27926cd762
---

[English（翻訳元）](cli-contract.md)

# CLI 契約

次のコマンドを実装しています。プラットフォームの検証は[実装計画](../exec-plans/completed/agent-env-mvp.md)に別途記録します。

```text
agent-env version
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
agent-env doctor [repository|lease-id] [--runtime compose|android-emulator|flutter-android] [--provider docker-compose|podman-compose]
```

リポジトリ省略時は現在のディレクトリを使います。`init` は認識可能なルート Compose file が 1 つあることを要求し、`.agent-env.yaml` を排他的に新規作成して review が必要と報告します。何も起動しません。

`validate` は Docker や Podman なしで schema と参照を確認します。`plan` はさらにローカル Git commit と決定的なコンポーネント閉包を解決し、状態や worktree は作成しません。

`--ref` は source が 1 つの場合に限ります。複数 source には alias ごとの `--source` override を使います。どちらもリモート ref を fetch しません。

manifest は既定で指定した control checkout から取得します。plan/create の `--manifest` は別の信頼済み manifest を明示的に選び、その digest と canonical snapshot を保持します。review mode は契約上書き込み不可の detached source worktree を作成します。書き込み可能な fix mode はありません。

## 識別情報、状態、観測

`--owner <text>` はグローバルな参考所有者 selector です。`AGENT_ENV_OWNER` が明示的な既定値を与えます。指定しない場合はローカル user/host の識別情報と一意の接尾辞で割り当てを識別し、`list --mine` は別の呼び出しでもそのローカル識別情報に一致させます。owner label は filter であり、認可境界ではありません。

`list` と `show` は記録した source とproviderを固定したCompose project を検査します。`--cached` はレジストリのみを読む明示的な list です。`show` は lease、event、command run、artifact、endpoint の観測を含みます。

`capabilities` は選択コンポーネントが宣言する capability、観測状態、endpoint のアドレス map を報告します。宣言された endpoint は、保存したランタイム設定に動的 loopback 公開を生成します。

`reconcile <lease-id>` は観測した lease を返します。全体の `reconcile` は `leases` と `inventory` を含む object を返し、一致する所有者記録のないリソースも含めます。削除はしません。

desired state は `active` または `released` です。observed state は `requested`、`allocating`、`starting`、`ready`、`degraded`、`failed`、`releasing`、`released`、`quarantined`、`unknown` です。保存済みの ready 行は実際の健全性の証明ではありません。Compose リソースの不在は active lease を degraded にし、検査失敗は不確定として見える状態を保ちます。期限切れまたは隔離中の lease を、留保のない ready 結果にはできません。

ランタイム/コンテナと時間制限付き HTTP readiness は再観測できます。任意の command probe は create 中に実行し、list/show はリポジトリコマンドを再実行しません。したがって実リソースの健全な観測は、以前成功した command probe が現在も成功する証明にはなりません。

`logs --run` は記録した 1 つの command run の stdout/stderr を読みます。`logs --component` は、その Compose project を共有する全サービスを返すのではなく、コンポーネントが持つランタイム内サービスを選びます。filter がなければ、利用可能なランタイムのログと保持済みの削除証拠を含めます。

グローバルinventoryは、導入済みprovider実行ファイルと記録済みprovider/engine識別情報を
併せて探索し、resource IDをprovider単位に区別します。導入済みでもengineが利用不能なら、
部分的なinventoryと明示的なエラーを返します。

## 出力と終了 status

`--output table` が既定の人間向け表示です。一部の詳細コマンドは table の代わりにインデント付き object を表示します。全コマンドが次の envelope による `--output json` に対応します。

```json
{"schema_version":1,"data":{}}
```

各 lease は、永続化される lifecycle 変更時に更新する version 付きの診断 snapshot `leases/<id>/environment.json` も持ちます。この snapshot は診断用であり、lease の記録を管理するレジストリの代わりにはなりません。

data の形はコマンドごとに異なります。plan/create は object、list は lease 配列、show は lease と証拠 collection、GC は `apply` と `leases` を返します。JSON mode では named test のストリーム出力を stderr に送り、stdout を単一 JSON 文書に保ちます。create、削除、named test が失敗しても記録済み結果を出力することがあります。プロセス終了 status と結果の両方を確認してください。

| Exit | 意味 |
| --- | --- |
| 0 | 要求された操作が成功 |
| 2 | コマンド、引数、manifest、その他ユーザー入力が不正 |
| 3 | 前提条件が欠けている、または使用不能 |
| 4 | 割り当てまたは readiness の失敗 |
| 5 | named test の失敗、timeout、キャンセル |
| 6 | 隔離を含め、安全に削除を完了できなかった |
| 7 | 内部、レジストリ、観測の失敗 |

これは CLI の終了 code です。named command 自身の終了 code は run 記録に別途保存します。`doctor`はrepositoryもproviderも明示しない場合、既定でDockerを検査します。repository指定時はmanifestで宣言した全Compose providerを検査します。`--provider`は診断対象を1つ選択し、leaseの実行設定は上書きしません。診断はprovider情報、バージョン、engineの前提条件を保持します。PodmanにはPodman 5.xと独立したpodman-compose >=1.6.0,<2.0.0が必要であり、fallbackしません。`--provider`はCompose前提条件の診断専用で、別runtimeやlease IDには適用できません。`--runtime android-emulator` は Docker の代わりにローカル SDK tool と AVD template を確認します。`--runtime flutter-android` は Docker や Podman なしで既定の Flutter 実行ファイルと Android SDK/AVD の前提条件を検査します。リポジトリを指定すると、宣言した全アプリの設定済み実行ファイル、プロジェクト、Android の前提条件を検査します。[Flutter 契約](flutter-android-runtime.ja.md)を参照してください。lease ID は実際の lease の診断を選び、ready でない lease は exit 3 を返します。[Android 契約](android-emulator.ja.md)を参照してください。

## 削除と更新

既定 TTL は 4 時間、最大 TTL は 24 時間、active な予約容量は 8 です。`renew --ttl` は現在からの相対値で期限を変更し、heartbeat を更新します。不在リソースの再起動はしません。quarantined、releasing、released の lease は更新できません。

`destroy --dry-run` はリソースを削除せず、現在の削除候補を返します。通常の削除は対象を厳密に特定した active named run ID にキャンセルを要求し、終了確認と証拠確定を待ってから削除用 lease ロックを取得します。無関係な active 操作は busy のままです。その後 source/runtime の所有権を確認し、サービスごとの最終ログを保持し、ランタイムを逆順に削除し、管理対象 worktree を削除して lease を released にします。

追跡対象ファイルに変更があると lease を隔離します。`destroy --force` は、binary Git diff を artifact として保持した後の削除を明示的に許可します。それでも固定された所有権の一致と証拠収集の成功が必要です。管理対象 review worktree 内の未追跡の test/build 出力は、通常の削除で除去されることがあります。管理対象 worktree に無関係な作業を置かないでください。

`gc` は変更せず候補を preview します。既定 policy は期限切れから 5 分、最後の heartbeat から 1 分の経過を要求します。永続化された `running` コマンド行がある lease は、操作ロックが期限切れでも preview と apply の対象から除外します。`gc --apply` は各 lease ロックを再取得し、所有権/source の変更確認と削除の前に、これらの条件を再確認します。隔離中と処理中の lease は除外します。artifact の自動期限切れや、一般的なホストリソース prune はありません。

## Named test

test は固定された manifest に存在し、要求する component stack が lease 内に含まれなければなりません。lease は ready で期限内である必要があります。実行前後に source 識別情報と追跡対象に変更がないことを確認し、追跡対象を書き換えた test は review lease を隔離します。

コマンドは argv 配列と、割り当てた source 相対の working directory を使います。永続化された操作ロックが同時削除を排除し、実行の前後に heartbeat を更新します。各 run は名前、伏せ字処理した argv、source、working directory、timestamp、exit code、status、stdout/stderr パス、JSON run 記述子、宣言 artifact を保持します。ログは保存と同時にストリーム出力します。test の失敗や要求のキャンセル後も証拠を保持します。

レジストリへ終了 status を記録するのは、プロセスツリー終了と必要な証拠確定を確認した後だけです。終了未確認エラーや stream/artifact/descriptor 確定の失敗では、永続化した run は `running` のままなので、destroy と GC は削除を拒否します。通常の非ゼロ終了、timeout、確認済みキャンセルは、証拠保持後に終了結果にできます。ローカルの復旧記述子は、古いロック保持者にレジストリ行の確定を許可しません。

環境 map は run 記録にコピーしません。`${env:NAME}` はホスト変数を明示的に読み、`${lease_id}` は lease 識別子を挿入します。資格情報らしい環境値の literal は割り当て前に拒否します。[manifest 詳細](manifest-v1.ja.md)と[セキュリティ上の限界](../SECURITY.ja.md)を参照してください。

## Android UI の観測

`ui` は、永続化された所有 Android runtime の識別情報を使い、任意の serial は受け付けません。
snapshot が公開するのは Android accessibility の意味情報であり、Flutter widget ではありません。
application 指定時は既定でその package を対象とします。runtime だけを指定した場合や
`--all-windows` を指定した場合は system window も含めます。JSON は既存の envelope を使い、
table 出力では snapshot 内だけで有効な node 参照と artifact path を表示します。
対象選択、状態の制約、stable error code、上限、プライバシーは[observer 仕様](android-ui-observer.ja.md)で定義します。

意味情報に基づく tap/set-text には、登録済み snapshot と、現在の一意な fingerprint が必要です。
対象が古い場合や曖昧な場合は入力前に error を返し、座標へ fallback しません。
text 置換には、focus のある編集可能な node と、読み戻しの一致確認が必要です。
生および正規化した観測結果、PNG、対象を絞った log、操作証拠を lease artifact に保持します。
PNG のピクセルは text のように秘密値を伏せられません。UI コマンドは cleanup と同じ lease fence を使い、
remote 完了を確認できない場合は実行中コマンドの barrier を保持します。
`ui recover` は登録済みの中断された helper 操作だけを扱い、その同一性と不在を確認し、
復旧を記録してからその run の barrier を解除します。入力を再試行せず、中断された操作を成功とも扱いません。
復旧には、明示的に永続化された `termination-unconfirmed` の分類と、検証済みの元の結果証拠が必要です。
分類がない crash は引き続き拒否します。

UI error は共通の終了コード仕様に従います。前提条件の不足は 3、無効な option・対象選択・
存在しない lease・stale/ambiguous な参照は 2、registry と観測の障害は 7 を返します。
秘密値を伏せた後も error の型による分類を保ち、これらを区別できるようにします。

意味情報を使うコマンドでは、`AGENT_ENV_UI_HELPER` に検証済み companion build directory を指定します。
インストール済み tool を使い、`go run ./tools/uihelper --sdk <sdk> --jdk <jdk>
--platform android-35 --build-tools 36.0.0 --output <new-directory>` で明示的に build します。
出力 directory は新規である必要があります。host 環境変数の設定方法は利用者が選択します。
対象 manifest の変更や自動 download は不要です。

## 延期されたコマンド

Expand/shrink、書き込み可能な fork、checkpoint/reproduce、browser 観測、artifact promotion は未実装です。placeholder の成功は返しません。[ロードマップ](../roadmap.ja.md)を参照してください。

## キャンセルと manifest の由来

`destroy` は対象を厳密に特定した active named run ID にキャンセルを要求し、削除前に最大 10 秒、終了/証拠確定を待ちます。キャンセル未確認や所有者のない running 記録は `--force` を含め削除を阻止します。すでにそれらの named run と無関係な操作は busy のままです。待機中に始まった新しい run を destroy はキャンセルしません。

plan と lease は `manifest_path`、`manifest_commit`、`manifest_modified` を公開します。path は実際に選択した file、commit は runtime source ref とは独立した control checkout の HEAD です。変更済み、未追跡、ignored の file は modified と記録します。Git 外では commit は空で、canonical snapshot/digest が由来を示すことを診断で説明します。`--manifest` は相対 source repository path の解決に使う control repository を変更しません。
