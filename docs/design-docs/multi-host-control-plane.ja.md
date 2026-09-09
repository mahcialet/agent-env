---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/design-docs/multi-host-control-plane.md
source_sha256: f7c63aa45f7a8bbfff1e62dfb5cfea0638805a66abe9eec78b630829717d72e5
---

# 一つの管理主体による複数 host の調整

[English](multi-host-control-plane.md)

[製品仕様](../product-specs/multi-host-control-plane.ja.md) が必須の挙動を定めます。
[ExecPlan](../exec-plans/completed/multi-host-control-plane.ja.md) には文書化した範囲で完了した実装と受け入れ検証を記録しています。
この設計をもって、未実行の protocol、統合、native の検査に成功したとは扱いません。

## 管理主体と依存関係

専用の controller SQLite DB は、controller 識別情報、登録済み client/host、生存状況、capability、
容量、global lease、assignment、operation journal、blob 参照を管理します。
worker の既存 registry は source/worktree の識別情報、予約、具体的な runtime、local operation fence、
証拠、cleanup を管理します。controller の scheduling は具体的な runtime adapter を import しません。
worker の構成では app と既存 provider を再利用し、Android、Flutter、Browser/CDP の独立した責務を維持します。

同じ状態 root の process lock で有効な controller を一つに制限します。これは合意形成ではありません。
別々に複製した controller DB を同時稼働させてはいけません。stable host ID は運用者が指定する名前です。
永続 host-instance ID と登録済み証明書が実際の worker を結び付けます。
process incarnation は接続 session を区別しますが、host instance やその assignment を置き換えません。

## 通信と要求の権限

標準ライブラリの HTTP/TLS で、version を持つ型付き JSON を運びます。
相互 TLS で明示的に登録した client/worker 証明書の role を認証し、payload の識別情報も登録内容と一致させます。
protocol/product version に互換性がない場合は、作用が始まる前に配置対象から除外します。
秘密鍵はファイル設定であり、DB payload にはしません。
登録、heartbeat、long-poll、source download、result upload はすべて worker から開始します。
worker 側の受信用 RPC listener や汎用 shell endpoint は不要です。

各操作は controller ID、global lease ID、host ID、host-instance ID、assignment epoch、operation ID を
持ちます。epoch は assignment を識別し、各操作を通じて固定します。
worker はこの tuple をそのまま app の権限として使い、別 controller の採用や assignment の更新を暗黙に行いません。
同じ global ULID を local lease と resource/evidence path に使います。
管理 metadata は作用前の予約時に保存し、通常の registry Save では削除・変更できず、
既存 local lease に後から追加することもできません。

local app は作用前と既存 operation fence の取得後に、権限の欠落・不一致を拒否します。
destroy は cleanup に加え、実行中 command の cancellation 前にも検査します。
通常の GC は期限切れも含めて管理下の lease を除外します。読み取りでは reconcile せず記録済み metadata を
公開できます。

## Scheduling と不確実性

登録時に実態に合った semantic capability（`git`、`compose.docker`、`compose.podman`、
`android-emulator`、`flutter-android`、`persistent-process`、`browser-cdp`）と単純な容量を報告します。
scheduling は互換性のある ONLINE、登録済み、drain 中ではない worker 一つに lease 全体の容量を原子的に予約します。
host を明示指定しても適格性の全検査を維持し、順位付けは決定的にします。

heartbeat が途絶えた場合は観測を古い状態・UNKNOWN、host を OFFLINE に変えますが、配置は永続的に保持します。
作用前に拒否し、作用を開始していない証拠がある場合だけ、再配置を許可できます。
dispatch の曖昧さ、upload 失敗、worker からの無応答では許可しません。
別 instance は割当中の host 名を引き継げません。再接続と worker 再起動では同じ local resource を照合し、
controller 再起動では元の管理主体と journal を再読込します。drain は新規配置を止め、未解放 assignment のある
host の削除は拒否します。

## Native実行とWSLの境界

workerは自身と同じOSのprocessだけを管理します。Windowsでは、算出した実行先を作用開始前に
240 UTF-16単位の互換性範囲で検査します。WSLでは、DrvFSまたは9p mount上のstate homeを
ディレクトリ・DB作成前に拒否します。filesystem/mountの検査は、解決したaliasや独自mount先も
扱います。これは保守的な対応範囲であり、破損を観測したという主張ではありません。
検査対象はCLIのstate rootであり、内部の任意のDB open関数や読み取り専用source配置ではありません。
Windows側もWSL UNC上のstate/実行先を拒否し、直接の拡張UNC名と解決後のaliasを検査します。

Windows以外ではPE実行ファイルの直接起動を拒否し、Windowsでは`wsl.exe`の起動を拒否します。
process起動とdetached出力作成の前に検査し、Git bundleコマンドにも同じ検査を適用します。
意図しない直接interopを防ぐものであり、信頼するscriptの間接実行を制限するものではありません。
WindowsとWSLには個別のnative workerを置き、state rootを共有しません。
検証の限界とパスの要件はPORTABILITYを参照してください。

## Journal と配送順序

create では、app が lease を予約する直前に worker が `effect_started` を記録します。
その前の読み取り専用の検証や provider 診断が失敗した場合は、作用を開始していない証拠を永続化できます。
その後の destroy は、過去の全操作がこの境界を越えていないことを journal で証明できる場合だけ、
assignment を解放できます。予約を一度でも試行した後は、local 行がないエラーでも不確実な状態として扱います。

operation の受領と payload の識別情報を dispatch/作用の前に永続化します。
worker は準備、作用開始の可能性、local 結果、配送状態を記録します。
同一要求の再配送では journal を参照し、同じ ID で payload が異なる場合は失敗にします。
作用開始の可能性がある時点以降の crash は不確実な状態として照合を要求し、
create、test、UI/browser 入力を無条件に再実行しません。

artifact upload より先に local 結果を永続化し、配送の確認応答より先に central の結果と blob 参照を commit します。
未転送の upload や結果配送は永続識別情報を使って再試行し、artifact を作り直すために操作を再実行しません。
global RELEASED には worker による明確な cleanup/不在の証明が必要で、HTTP 成功や controller TTL の失効では
代用しません。controller の停止中も workload と journal を保持します。

### 初期の操作 dispatch 方針

worker は操作を一つずつ実行しますが、作成済み lease は並行して稼働します。
controller は同じ lease の二つ目の active 操作を、実行中の remote test に対する destroy も含めて拒否します。
remote の実行中操作の cancellation と操作の並行 dispatch は専用 protocol を必要とし、未実装です。
この方針により、live lease の並行稼働を維持しながら、operation の識別情報と local fence の競合を避けます。
別の操作を送信する前に `operation <operation-id>` で現在の要求を確認するか、結果を待ってください。

## Source と artifact の CAS

client は各 source alias を不変 commit に解決し、SHA-256 識別情報を持つ検証可能な Git bundle を用意します。
worker は local runtime の起動前に digest、commit、manifest、選択した依存範囲を独立に検証します。
client の絶対 path を CAS key や remote filesystem 操作の根拠にはしません。
未 commit ファイルは含めず、未対応 shallow/LFS/submodule は暗黙の network fetch で補わず明示的に拒否します。
worker の環境変数 placeholder は local で解決し、client の秘密値を暗黙に転送しません。

CAS key は `sha256/<digest>` です。非公開の一時ファイルに streaming し、size と hash を検査してから原子的に公開します。
同じ内容の同時 upload を安全に扱い、既存 object は検証後に限って再利用します。
参照を永続化し、参照中 object を回収しません。source object は一つあたり 1 GiB、artifact は一つあたり
64 MiB に制限します。転送再試行と mutation 再実行を分けます。
保存時の暗号化は保証せず、source と証拠は信頼された管理用データの境界内で扱います。

## 証拠と制限

型付き remote test/UI/browser の経路でも worker-local の lease fence、process/device 所有権、
古い参照の検査、秘密値の redaction、証拠の上限を維持します。返す loopback endpoint は worker を指し、
client への暗黙の tunnel はありません。HA、migration、host をまたぐ resource graph、remote shell、
秘密値配布、緊急時の管理引継ぎは別の設計課題です。

受け入れでは、決定的な状態機械 test、二つの worker 状態 root を使う実際の TLS socket 統合、
Windows/macOS/Linux の native role 実行、物理マシン・VM の複数 host 証拠を区別します。
同じ host の test や cross-build で最後の二分類を代用できません。
ExecPlan が正確な結果と不足を記録します。文書化した範囲で実装の受け入れは完了しています。

## 期限の永続化と機密入力の境界

追加する`lease_lifetimes`テーブルにcontrollerの期限と自動cleanupのoperation IDを保存する。
既存leaseは永続化済みcreateと成功したrenewの時刻から補完する。起動時、1秒ごとのserver検査、
workerのpollで、期限切れcleanupをtransaction内で予約する。queued/dispatched操作があれば
cleanupを延期する。自動cleanupが失敗または不確実になっても、新規の変更操作として盲目的に
再試行しない。成功したrenewだけが期限とcleanup記録をリセットする。検査中のdatabaseエラーは
serverの終了として表面化させ、期限処理が黙って無効になることを防ぐ。

共通protocol検証は保存前にUI/browserのJSON tokenを調べ、重複や大文字小文字違いのtext項目も
検査する。一時的なtextと`set-text`を拒否し、伏せ字へ変更した入力を実行しない。
既存journalは書き換えない。非永続の入力protocolは今後の課題とする。

createの作用境界を越えた失敗は、予約処理がleaseを返さなかった場合も不確実として扱う。
補償処理したcreateの結果payloadには解放済みlocal leaseを含められるが、destroy専用の
cleanup確認フィールドでは解放を宣言しない。復旧後も含め、明示的なdestroyで正式に解放を確認する。

CLI controllerもserverテストと同じServer.Runを呼び、定期的な期限処理と終了待機を共有する。
実際のCLI入口を使う回帰テストで、独立したdatabase接続からoffline leaseを期限切れにし、
pollなしでcleanupを確認する。認可済みblob handlerはHTTP serverの固定read/write期限を解除し、
metadataと認可されていない要求では通信処理の上限を維持する。

CAS公開ではfile同期に加え、platformの名前空間の永続化処理を成功応答前に実行する。
Unixでは一時directory、公開directory、CAS rootとその親を同期し、Windowsではnativeの
write-through moveを使う。検証済みの重複blobでも、過去の失敗後に見えているだけの状態を
信用せず、公開の永続化処理を再実行する。Windowsでは公開を直列化し、重複時は同期済みの
同一dataを再公開する。同時readerはfileの識別情報の変化を保守的に拒否する場合がある。
テストは公開失敗を注入して再試行を検証する。物理的な電源断や、native APIを超える
filesystem/hardwareの保証を実証したものとは扱わない。

配信した操作の所有者を現在のworker登録とは別に永続化する。前のincarnationがonlineなら、
再試行可能な登録拒否を返す。offline後の交代でも、過去に配信済みの変更操作をpollで引き継がせない。
localの永続receiptによる復旧完了は受け付ける。incarnationを記録していない旧操作の所有者は
未確認とし、capacityを保持して再配信しない。移行時点の登録先を推測で所有者にはしない。

対話的なログ取得ではCompose/Podmanの上限付き表示interfaceと、集計前に制限するAndroid file読み込みを
使う。既存の完全なcleanupログの証拠収集は別経路とし、表示上限を理由に必須証拠を捨てない。
remote source diffではbufferを埋め込まず、io.Copyがbytes.Buffer.ReadFrom経由でWriteの上限を
迂回できないようにする。

保持remote packageはcleanupの判断に必要な情報であり、journalが作用境界を越える前に永続的に
公開しなければならない。package fileを同期してから、platformのnative方式で一時・公開directoryの
永続化を確認する。同一packageが既にあっても成功応答前に再確認し、破損・不一致の記録は拒否する。
このlocal保持rootはworkerの直列処理で管理する。native APIの検証は物理的な電源断試験ではない。
