---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/product-specs/multi-host-control-plane.md
source_sha256: fc215a84880491230c75909a720107f643c904e94a4eae351abc7a99d15c21c3
---

# 複数 host の control plane

[English](multi-host-control-plane.md)

この文書は複数 host に対応するための必須仕様を定めます。文書化した範囲での実装と最終受け入れの証拠は、
[完了ExecPlan](../exec-plans/completed/multi-host-control-plane.ja.md) に記録しています。
ここに要件があるだけで、統合 test や native の受け入れ検証に成功したとは扱いません。

## モードと配置

既定の local モードでは、既存コマンドが daemon なしで local SQLite と provider を使います。
明示的な remote モードは、一つの有効な controller に型付き要求を送ります。
選択した全 source、component、runtime を含む lease 全体を、一つの割当先 worker で実行します。
remote shell API は提供しません。

controller は、登録済みで互換性があり、ONLINE かつ drain 中ではなく、必要な capability と
容量を備えた host のみを選びます。capability 名には `git`、`compose.docker`、`compose.podman`、
`android-emulator`、`flutter-android`、`persistent-process`、`browser-cdp` があります。
host の明示指定で省略するのは順位付けだけです。登録、互換性、capability、容量の検査は省略できません。
drain は新規配置を止めますが、既存 lease を移動・破棄しません。

## 認証と所有権

本番の通信では HTTPS と相互 TLS を使い、証明書の role を明示的に登録します。
client 証明書は client 操作を許可します。worker 証明書は登録した host と instance を許可し、
要求に任意の host ID を指定しても権限は得られません。worker から登録、heartbeat、poll、転送を
開始するため、worker 側の受信用 listener は不要です。秘密鍵はファイル入力として扱い、
controller や worker の SQLite に保存しません。

controller は global の配置と要求された有効期間を管理し、worker は local の
process/container/device の識別情報と cleanup 証拠を管理します。global lease ID は
worker registry でも使う同一の canonical ULID です。controller ID、host ID、永続
host-instance ID、ゼロ以外の assignment epoch を、source の materialization や runtime 起動より先に
永続化します。一つの assignment の epoch は各操作を通じて固定し、個々の要求には operation ID を使います。
新しい操作によって epoch を暗黙に進めません。

通常の local mutation、reconciliation、test、UI/browser 操作、GC は、controller 管理下の lease を
変更できません。local force でも権限検査を回避できません。読み取りでは記録済みの管理 metadata を返し、
assignment の権限なしに暗黙の reconcile を実行しません。管理下の有効期間切れや controller の停止だけでは、
local cleanup を許可しません。

### 初期の操作 dispatch 方針

worker は操作を一つずつ実行しますが、作成済み lease は並行して稼働します。
controller は同じ lease の二つ目の active 操作を、実行中の remote test に対する destroy も含めて拒否します。
remote の実行中操作の cancellation と操作の並行 dispatch は専用 protocol を必要とし、未実装です。
この方針により、live lease の並行稼働を維持しながら、operation の識別情報と local fence の競合を避けます。
別の操作を送信する前に `operation <operation-id>` で現在の要求を確認するか、結果を待ってください。

## Source と証拠

転送する Git 入力は commit 済みのものに限ります。各 source alias は正確な commit と検証済み
SHA-256 bundle digest を指定します。worker は runtime に作用する前に、転送された manifest、
source set、選択 stack を独立に検証します。client の絶対 path を worker の path や CAS key にしません。
未対応の shallow、LFS、submodule は明示的に拒否し、足りない source を暗黙の network fetch で補いません。
remote の `${env:NAME}` は worker で解決し、client の環境変数の秘密値を暗黙に転送しません。

controller CAS は digest を検証して上限を設けた object を受け付けます。source object は一つあたり
1 GiB、artifact は一つあたり 64 MiB が上限です。呼出側の path を保存先の決定に使いません。
local の結果を永続化してから証拠を upload します。転送失敗は digest 単位で再試行できますが、
証拠を作った操作の再実行は許可しません。repository 内容と artifact は管理用の非公開データとして扱い、
保存時の暗号化は保証しません。

## 操作、不確実性、復旧

remote 操作の対象は create、list/show、renew、reconcile、destroy、名前付き test、logs/artifacts、
対応 worker 上の型付き Android UI および Browser/CDP 操作です（remote `set-text`は除外。後述のリクエスト規則を参照）。既存の local operation fence、
古い snapshot の検査、resource 所有権の検査を維持します。endpoint を使う操作は worker で実行します。
結果の `127.0.0.1` URL は worker-local であり、client-local の tunnel ではありません。

双方が operation の識別情報と payload を journal に記録します。同じ operation ID で内容が異なる要求は
拒否します。同じ要求が再配送された場合は、mutation を繰り返さず永続化した結果を返すか照合します。
dispatch 後の切断は不確実性を意味し、作用がなかった証明にはなりません。upload の復旧では結果の配送だけを
再試行します。destroy は、worker が既存 lifecycle の規則に従って local cleanup と不在を証明して初めて、
global RELEASED になります。

heartbeat の失効では host を OFFLINE、lease の観測を古い状態・UNKNOWN として扱います。
resource 不在の証明や、生存中・不確実な lease の自動再配置の許可にはしません。
同じ worker instance が既存 assignment に再接続します。assignment が残る間、別 instance が
同じ host 名を引き継ぐことはできません。active、古い観測、不確実、未解放の assignment が残る host の
削除は拒否します。controller の再起動でも識別情報、assignment、journal を保持します。

## 制限と受け入れ

この段階では、信頼された一つの管理組織と一つの有効な controller を前提にします。
controller DB の複製は安全な failover を提供しません。HA、live migration、host をまたぐ lease 分割、
透過的 endpoint tunnel、秘密値配布、暗黙の緊急復旧は対象外です。

受け入れには、別々の状態 root を持つ二つの worker を使った実際の TLS socket test と、
Windows、macOS、Linux の native controller/worker/client 実行が必要です。
同じ host の worker process で証明できるのは protocol の分離であり、物理的な複数 host の挙動ではありません。
cross-build も native 実行の証明にはなりません。物理マシン・VM での検証範囲、利用できない前提環境、
残る検証不足は、完了前に ExecPlan へ明記し続けます。

## レビューで補強した期限とリクエストの規則

remote createの既定ownerはユーザーとhostに基づく安定した値とし、明示した`--owner`または
`AGENT_ENV_OWNER`を優先する。同じoperation IDを別processから再試行してもリクエストの
識別情報を維持する。workerの`--max-leases`は1以上、local policyの上限8以下とする。

controllerはcreate受理時刻に基づく期限を永続化する（既定4時間、最大24時間）。renewが
成功すると、そのrenewの受理時刻と要求TTLから期限を設定する。再送や再起動によって期限を
さらに延ばさない。期限切れでは、実行中の操作がなければ通常の非force destroyを1件予約する。
offline worker、不確実な結果、cleanup失敗では、不在を証明するまでcapacityを保持する。
解放済みleaseでもqueued/dispatched操作が残るhostは削除できず、削除したhostへの新規操作は拒否する。

非永続の入力経路ができるまで、remote Android UIとBrowserの`set-text`は非対応とする。
直接protocolへ送信した場合も含め、CLI・controller・workerはjournalへの保存前にtext入力を
拒否する。localの`set-text`は引き続き利用できる。恒久的な登録拒否ではworkerを終了し、
一時的な通信障害、レート制限、server障害は再試行する。

blobのupload/downloadはmetadata要求の固定timeoutを使わず、呼び出し元contextの期限とcancelに
従う。metadata、接続開始、TLS/headerの上限は維持する。4 MiB上限のmanifestはcreate要求に
1回だけ含める。省略形式のpackageはworkerの検証前に補完し、従来形式では重複manifestの
一致検証を維持する。

remote workerのlease応答では、重複するmanifestとprocessのcommand/env宣言を省略する。
runtimeの識別情報・path・port・状態・digestは維持し、設定の宣言はcreate envelope/source CASと
local leaseに保持する。

前の登録がonlineの間は、別のworker incarnationへの交代を認めない。通常の再起動でも前の
heartbeatが期限切れになるまで待つ。この登録拒否は再試行できる。前のincarnationへ配信済みの
操作は新しいincarnationへpollで再配信しない。元の永続journalを持つworkerは復旧・結果報告が
できるが、journalを失った場合はcapacityを保持したまま明示的な復旧が必要となる。
処理中のjournalと認証情報の両方を悪意をもって複製した場合を見分ける保証ではない。

読み取り専用logs/artifactの結果は、復旧が不確実でもleaseの状態を変更しない。
成功したreconcileでは、既に解放済みのlocal leaseを確認できる。Compose表示ログのcapture量と
保存済みログの合計に上限を設け、超過時はworkerのmemoryを無制限に使わず、出力が不完全なことを
明示するエラーを返す。検証済みの保持source packageはCAS展開前に再利用する。
変更済みsourceのforce cleanupでは、所有するworktreeを削除する前に上限付きpatchを記録する。
