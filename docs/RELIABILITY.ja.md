---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/RELIABILITY.md
source_sha256: a38f93add1bb06c6fb356414ca20274e03ac89d115e2c7f3b1f301b7025e4cf0
---

[英語版（翻訳元）](RELIABILITY.md)

# 信頼性と復旧

## 割り当てとcleanup

割り当てでは、リソースを作る前に実行する内容を記録します。SQLite、Git、選択したruntimeの
ツールはそれぞれ独立しているため、処理全体を一つのトランザクションにはできません。
そこで各段階を記録し、後続の失敗時には完了した作用を逆順に取り消すsaga方式を使います。

実体化の前に、レジストリがリース、不変のソース識別情報、一意なworktreeパス、runtimeの
プロジェクトを予約します。イベントも外部操作より先に記録します。Composeの起動前には、
検証・正規化したスナップショットを、所有権ラベル、生成した動的loopback endpointの割り当て、
digestとともに保存します。このため、`up`が失敗してもcleanup対象の識別情報が残ります。

### 失敗時に残す記録

失敗時には、通常の要求取消後も有効なcleanup contextを使い、時間制限付きの逆順補償を実行します。完全に片付いた割り当て失敗は、失敗イベントと成果物を伴うreleasedとして記録に残ります。所有権が不確定、またはcleanupが不完全な場合は、復旧できるようにquarantinedの予約が見える状態を保ちます。再試行が成功したように見せるために証拠を消すことはありません。

### 削除の条件

cleanupでは、次の条件をすべて守ります。

- リソースを削除する前に、固定したソースの識別情報と追跡対象の変更を検査します。
- downの前に最終runtimeログを保存します。変更のないworktreeも、runtimeのcleanup後にのみ削除します。
- 明示的なforceでも追跡対象の差分証拠を保持します。識別情報が一致しなければ拒否します。

管理対象のreview worktree内にある追跡対象外ファイルは破棄可能です。
記録したリソースが既に存在しない場合、cleanupを繰り返しても結果は変わりません。

Compose cleanupはruntime snapshotに保存したproviderとengineを使います。
Podman down前に、appはnativeの匿名volume識別情報と接続証拠を含む
`Runtime.cleanup_evidence`を永続化します。container消失後に中断しても証拠は残ります。
復旧ではvolumeの識別情報と現在の参照を再確認し、外部・兄弟からの参照がないと証明できる
残存volumeだけを削除します。識別情報の変化、観測不能、不完全なcleanupはquarantineと
します。provider downの成功だけでは不在の証明とせず、実resourceを再検査します。

## レジストリと並行操作

SQLiteのレジストリは単一ホスト内のローカル状態です。未対応のネットワークファイルシステムに
置いたり、複数ホストの調整機構として使ったりしないでください。

永続化には、埋め込まれた連番migration、検証済みWAL、外部キー、接続のbusy timeoutを使います。
immediate transactionで容量とruntimeプロジェクト予約の一意性を守り、正規化した行と対応する
リーススナップショットを同時に更新します。

バージョン付きの`leases/<id>/environment.json`は補助的な診断情報です。ライフサイクルの変更を
永続化するときに更新しますが、書き込みに失敗しても状態の判断にはSQLiteの記録を使います。

### 操作の所有権

ライフサイクル変更、名前付きテスト、期限更新、reconcile、適用するGCは、永続的なリース操作lockを保持します。プロセスがクラッシュして更新が止まると、そのtokenは期限切れになります。操作contextは外部コマンドとレジストリ変更の両方へlockの所有権を引き継ぎます。更新失敗は作業を取り消し、transactional fencingが古い所有者の書き込みを拒否します。lockを失ったことを、以前のプロセスが後継所有者のリソースに対して補償を続ける許可にしてはいけません。

### 取消と完了の判定

移植可能な取消処理は、Unix process groupまたはWindows Job Objectを通じて管理対象コマンドの子孫を終了します。名前付きテストとprobeの実行を制限する仕組みであり、リポジトリコードが既に行った任意の副作用を元に戻すものではありません。外部割り当て後に取り消されたコマンドにも、観測に基づく復旧が必要です。

destroyは、対象とする実行中の名前付きrun IDを厳密に指定して取消を要求し、プロセス終了と最終証拠の確認を最大10秒待ってから、リースlockをcleanupへ引き継ぎます。古い要求が新しいrunを取り消すことはありません。無関係な実行中リース操作がある場合はbusyのままです。取消を確認できなければリソースを削除しません。有効な操作所有者がいないのに永続的な`running`記録がある場合、何が起きたかを明示的にレビューした復旧で確定するまで、forceを含めdestroyを禁止します。

runの終端ステータスは、単なるコマンド終了コードではなく完了判定の条件です。プロセスツリーの終了確認、取得したstreamのflush、run記述子と宣言された成果物の保持、最終証拠の永続化のいずれかに失敗すると、レジストリのrunを`running`のまま保持します。これにより、そのリースに対するdestroyとGCを意図的に止めます。

型で識別できる終了未確認の失敗は、通常のコマンド失敗や取消と区別できます。終了コード0も一般的な取消結果も、子孫プロセスの停止を証明しません。保持したローカルログと記述子は復旧証拠であり、新しい所有者のレジストリ状態を上書きする許可ではありません。

## 稼働状態の観測

list/showは、レジストリの意図を、登録済みGit worktree、固定コミット、Composeプロジェクトラベル、具体的なリソース識別情報、サービスの健全性、時間制限付きの読み取り専用HTTP probeと比較します。`list --cached`は観測を明示的に省略します。コマンドprobeは作成時に実行し、一覧表示の副作用として再実行しません。

リソースの欠落や不健全な状態は有効なリースをdegradedにし、検査できない場合は診断を伴うunknownを維持します。期限切れでも健全なruntimeは観測状態`ready`を保つことがありますが、reconcileは有効期限を報告します。永続化された未完了のコマンド行は、プロセス完了未確認のrunningとして報告し、heartbeatが古ければstale-runの診断を加えます。reconcileが終端runステータスを作り出すことはありません。解放済みリソースの再出現や、ソースの変更・不一致はリースをquarantinedにします。孤立リソースの発見は観測のみです。無関係なリソースや所有権が曖昧なリソースは削除せずに報告します。unknownのリソースに関する証拠は、空の成功結果に置き換えず保持します。

## TTLとgarbage collection

組み込みのTTLは4時間、最大24時間、容量は有効な予約8件です。期限更新は現在時刻を基準に有効期限を設定し直してheartbeatを更新しますが、欠落したruntime状態を修復しません。失敗またはquarantinedの割り当ては、cleanupがreleasedと確認されるまで容量を使い続けます。

`gc`は読み取り専用の候補プレビューです。既定ポリシーでは、有効期限から5分以上、最後のheartbeatから1分以上の経過を要求します。ステータス`running`の永続的なcommand-run行がある場合、古い操作lockが期限切れになっていても、そのリースをプレビューと適用の対象から除外します。`gc --apply`はリースlockを取得し、期限の猶予、heartbeatの猶予、ライフサイクル状態、running記録を再確認してから、リソース識別情報と追跡対象に変更がないことを確認します。プレビュー後の期限更新や新しいコマンドも尊重します。処理中およびquarantinedのリースは対象外です。

ポリシーモデルは`GCGrace`と`HeartbeatGrace`に対応しますが、CLIは現在、組み込みの5分・1分の値を使用します。成果物の自動期限切れポリシーやホストリソース全般のpruneはありません。成果物は運用者が明示的に管理するまで残ります。

## 復旧手順

1. `show`、`reconcile`、保持したログで、判明しているもの、欠落、変更、不確定なものを特定します。
2. 観測を再試行する前に、記録されたprovider/engineやその他の前提条件を復旧します。
3. 残したい追跡対象の変更は管理対象リースの外へ保存します。変更のないリソースには通常のdestroyを使います。追跡対象の編集を意図的に破棄し、差分証拠を保持できる場合にのみ明示的なforceを使います。
4. 再度reconcileを行い、リースのruntime/worktreeリソースが存在しないことを確認してからcleanup完了と判断します。

古い計画、lock記録、観測timeoutから作業の停止を推測しないでください。実際のプロセスとリソースの識別情報を確認します。無関係なリポジトリを巻き込む一括Docker/Podman cleanupやGit worktree pruneで、不確定なリースを修復してはいけません。

## 固定マニフェストの由来

計画とリースは、選択したマニフェストのsymlink解決済み絶対パスを`manifest_path`に、それを所有する制御用checkoutのHEADを`manifest_commit`に記録し、`manifest_modified`も保持します。制御用コミットはruntimeソースの`--ref`上書きとは独立です。変更済み、追跡対象外、ignoredのマニフェストはmodifiedをtrueにします。Git checkout外のマニフェストは空のcommit、modified true、診断を持ち、そのマニフェストの内容は、保存した正規化スナップショットとdigestに基づいて判断します。選択したマニフェストが別の場所にあっても、相対ソースリポジトリパスは指定した制御用リポジトリを基準にします。

## Androidの復旧

SQLiteは起動前に専用AVDの識別情報と、偶数・奇数のconsole/ADBポートペアを予約します。所有権markerは、起動の意図とネイティブプロセスの生成時識別情報を、破棄可能なAVD状態の外に記録します。cleanupはAVDの識別情報を検証し、同じ認証済みconsole接続でkillを送り、プロセスツリーとポートの不在を確認してから専用の書き込み可能状態を削除します。ログとmarkerは証拠として残ります。起動識別情報の欠落、リソースの再利用、子孫の観測が不確定な場合は、リースをquarantinedにして予約を保持します。forceでもこの制限を上書きできません。

個別のAndroidのみのリースのreconcileにはDockerは不要です。グローバルなCompose孤立resourceのinventoryは、Androidリースだけが登録されている場合も、導入済みprovider実行ファイルと記録済みprovider/engine識別情報の和集合を探索します。導入済みでもengineが利用不能なら、空の成功結果ではなく部分的なエラーを明示します。resource IDはprovider単位に区別します。Androidの検査は記録された識別情報に限定します。reconcileは手動終了を観測しますが、Emulatorを取り込んだり再起動したりしません。

### 共有ADBの寿命

共有ローカルADBサーバーの寿命は各リースとは別です。作成時にはEmulatorを起動する前に、`127.0.0.1:5037`へ直接送る読み取り専用の`host:version` probeでプロトコル互換性を確立します。サーバーがなければdetached-process APIを通じて別途起動します。既存サーバーのバージョン不一致、不正な応答、観測不能は、置き換えを行わず前提条件の失敗とします。bootの観測ではSDKクライアントを実行する前に互換性確認を繰り返します。共有前提条件が欠けていれば、起動や置き換えをせず報告します。

起動診断と識別情報は、runtimeの証拠と同じ場所の`adb-server.stdout.log`、`adb-server.stderr.log`、`adb-server-start.json`に残ります。その後に割り当てが失敗しても、このホストサービスを停止する許可にはなりません。destroyとGCは所有するEmulatorリソースのみを削除し、グローバルADBサーバーのcleanupは決して行いません。復旧では、このプロセスをリース所有のEmulatorとして扱ったり、起動失敗の証拠を捨てたりせず、共有前提条件を復元する必要があります。

## 常駐processの復旧

processの作成では、ソースのcommit、runtimeのパス、名前付きTCP予約を確定し、起動意図を
永続化してから起動します。返されたネイティブ識別情報はreadinessの確認前に保存します。
エラーとともに返された識別情報も、0でなければcleanupの証拠です。後続のレジストリ保存に
失敗した場合は、`launch.json`から起動の情報を復旧できます。

作用の有無が不明なStartを、何も起きなかったものとして繰り返してはいけません。
後から別のCLIを実行すると、ネイティブ識別情報と時間制限付きHTTP検査で稼働状態を観測します。
起点プロセスが終了したリースはdegradedになり、自動再起動はしません。

### 停止と削除の条件

destroyはsignalの送信前と強制停止へ移る前に、ネイティブプロセスの所有権を再検証します。
Unixでは子孫が残っている場合、foregroundの起点プロセスも必要です。グループの系譜が曖昧なら、
数値のグループIDへのkillを許可せずquarantineにします。Windowsでは所有するJobを厳密に指定します。

ツリー全体の不在を確認してから専用状態とポートを解放し、通常どおり追跡対象の変更を保護した
worktreeのcleanupへ進みます。起動・停止が不確実な場合、出力証拠の保存に失敗した場合、または
fenceを失った場合は、復旧用の状態と予約を保持します。destroyやGCを再実行しても、この証明要件は変わりません。

### 復旧に使うファイル

runtimeのファイルは`leases/<id>/process-runtimes/<runtime>/`配下に置きます。
書き換え可能な`state/`はcleanup確認後に削除し、ログと起動識別情報は診断証拠として残します。

独立した`redaction.json`は、バージョン付き所有情報と秘密値のfingerprintをネイティブStart前に
保存します。このため`launch.json`の保存に失敗しても、補償時のログを伏字化できます。
後続のCLIは、現在のホスト環境に秘密値が残っていることに依存しません。

型で識別できる`ErrProcessNotStarted`とPID 0がそろう場合はpreparedへ戻せます。
その型による証明のない識別情報0の失敗は、引き続き不確実なものとして扱います。
[processのライフサイクル設計](design-docs/persistent-process-runtime.ja.md)を参照してください。

## browser操作の復旧

すべてのbrowser操作で、CDP処理から証拠の確定までリースのfenceを保持します。
操作を許可する条件は次のとおりです。

- activeで期限内、かつreadyのリースでは入力できます。
- degradedのリースでは、ネイティブプロセスの識別情報を証明できる場合に限り、読み取り専用の診断ができます。
- quarantine、停止済みまたは識別情報が曖昧なプロセス、未完了のcommand runがある場合は操作を拒否します。

ポートの再利用、browser・page・document・nodeの識別情報の変化、切り詰めたsnapshotは、入力を
許可する根拠にはなりません。変更操作は一度だけ試み、切断後に自動再実行しません。

完了を確認できない場合や証拠の確定に失敗した場合は、commandをrunningのまま残して後続処理を
止めます。実際の結果と証拠を確認し、レビューを経て復旧してください。
destroyとGCは汎用processツリーの不在を確認してからprofileを削除します。
browser独自のcleanupや自動再起動は行いません。
[browser設計](design-docs/browser-cdp-automation.ja.md)を参照してください。

## Controller と worker の復旧

[control plane](design-docs/multi-host-control-plane.ja.md) は global 管理用の専用 SQLite を追加し、
各 worker は local resource registry と fence を維持します。
assignment の controller/host/host-instance/epoch tuple は各操作を通じて固定します。
worker と controller の journal に payload の識別情報、作用開始の可能性、local 結果、配送状態を保持します。
重複配送では既存結果の復旧・upload はできますが、不確実な mutation を無条件に繰り返しません。
artifact upload より先に local 結果を保存し、global RELEASED には worker の cleanup 証明を必要とします。
heartbeat や controller 接続の消失では観測を古い状態・UNKNOWN とし、再配置、cleanup、host instance の変更を
許可する根拠にはしません。

### 並行実行と未対応の復旧操作

初期実装の worker は操作を直列に実行しますが、作成済み lease は並行して稼働します。
controller は同じ lease の二つ目の active 操作を、remote test 中の destroy も含めて拒否します。
remote の実行中操作の cancellation は、operation の識別情報と fence を維持できる専用 protocol を設計するまで対象外です。
別の destroy で中断できると想定せず、現在の操作を待つか `operation <operation-id>` で確認してください。
controller/worker の再起動では元の状態 root を再利用します。controller DB の複製は安全な failover ではありません。
drain は新規配置のみを止めます。local GC は期限切れでも controller 管理下の lease を除外し、
force でも assignment の所有権や不確実な cleanup を無視できません。
