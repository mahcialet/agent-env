---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/product-specs/multi-host-control-plane.md
source_sha256: 50997d90a830d15926627b2850516dd92ab8d0cb9df808bdc5b59dcab0d92541
---

# 複数 host の control plane

[English](multi-host-control-plane.md)

リモートモードでは、リース全体を登録済みのワーカー一つに配置します。この文書は、
実装済みの構成手順、認証、配置、操作結果が不確実な場合の扱いを定めます。
ローカルモードは既定のままで、コントローラーを必要としません。

検証した環境は末尾の受け入れ記録と[完了済み ExecPlan](../exec-plans/completed/multi-host-control-plane.ja.md)
で確認してください。要件の記載だけを、統合テストや各 OS での検証成功とは扱いません。

## リモート構成の導入手順

CA、コントローラーの DNS 名に有効なサーバー証明書、別々のクライアント証明書と
ワーカー証明書を事前に用意してください。コントローラー、各ワーカー、クライアントには、
ホストの環境変数設定でそれぞれ異なる絶対パスの `AGENT_ENV_HOME` を指定します。

### コントローラー側で登録して起動する

証明書の登録は、コントローラーの状態保存先を使うオフラインの管理コマンドです。
コントローラーを起動する前に、その保存先で実行してください。

```text
agent-env control-plane enroll --certificate client.pem --role client
agent-env control-plane enroll --certificate worker.pem --role worker --host-id build-a
agent-env --tls-ca ca.pem --tls-cert controller.pem --tls-key controller.key control-plane serve --listen 0.0.0.0:9443
```

### ワーカーを起動する

ワーカー専用の状態保存先と、使用するランタイムの前提環境を用意して実行します。

```text
agent-env --controller https://controller.example:9443 --tls-ca ca.pem --tls-cert worker.pem --tls-key worker.key worker serve --host-id build-a --max-leases 2
```

### クライアントから利用する

コントローラー URL、証明書、コミット済みのリポジトリを実際のものに置き換えます。

```text
agent-env --controller https://controller.example:9443 --tls-ca ca.pem --tls-cert client.pem --tls-key client.key hosts list
agent-env --controller https://controller.example:9443 --tls-ca ca.pem --tls-cert client.pem --tls-key client.key create ../trusted-repo --stack api --host build-a
agent-env --controller https://controller.example:9443 --tls-ca ca.pem --tls-cert client.pem --tls-key client.key artifact-download <digest> --destination evidence.json
```

list/show、renew、reconcile、destroy、名前付きテスト、対応する UI/browser 操作にも
同じ接続フラグを使います。成果物のダイジェストは登録済みのリモート証拠から取得します。
ダウンロード時にダイジェストを検証します。保存先には新しいファイルを指定してください。
`hosts drain <host-id>` は新規配置を止め、`hosts undrain <host-id>` は配置対象に戻します。
リモートの `set-text` は非対応です。テキスト入力にはローカルモードを使ってください。

## モードと配置

既定のローカルモードでは、既存コマンドがデーモンなしでローカル SQLite とプロバイダーを使います。
明示的なリモートモードは、一つの有効なコントローラーに型付き要求を送ります。
選択した全ソース、コンポーネント、ランタイムを含むリース全体を、一つの割当先ワーカーで実行します。
リモートシェルの API は提供しません。

コントローラーは、登録済みで互換性があり、ONLINE かつ drain 中ではなく、必要な capability と
容量を備えたホストのみを選びます。capability 名には `git`、`compose.docker`、`compose.podman`、
`android-emulator`、`flutter-android`、`persistent-process`、`browser-cdp` があります。
ホストの明示指定で省略するのは順位付けだけです。登録、互換性、capability、容量の検査は省略できません。
drain は新規配置を止めますが、既存リースを移動・破棄しません。

## 有効期間と予約容量

リモート create の既定の所有者はユーザーとホストに基づく安定した値とし、明示した`--owner`または
`AGENT_ENV_OWNER`を優先する。同じ operation IDを別プロセスから再試行してもリクエストの
識別情報を維持する。ワーカーの`--max-leases`は1以上、ローカルポリシーの上限8以下とする。

ワーカーの`--android-slots`は0〜65を受け付ける。これはローカルの割り当て処理の偶数コンソールポート
5554〜5682に対応する。範囲外の値はTLS初期化・状態作成・登録の前に拒否する。
0はAndroid容量なしとして広告する。

コントローラーはcreate受理時刻に基づく期限を永続化する（既定4時間、最大24時間）。renewが
成功すると、そのrenewの受理時刻と要求TTLから期限を設定する。再送や再起動によって期限を
さらに延ばさない。期限切れでは、実行中の操作がなければ通常の非force destroyを1件予約する。
オフラインのワーカー、不確実な結果、クリーンアップ失敗では、不在を証明するまで予約容量を保持する。
解放済みリースでもqueued/dispatched操作が残るホストは削除できず、削除したホストへの新規操作は拒否する。

## 認証と所有権

本番の通信では HTTPS と相互 TLS を使い、証明書の role を明示的に登録します。
クライアント証明書はクライアント操作を許可します。ワーカー証明書は登録したホストと instance を許可し、
要求に任意のホスト ID を指定しても権限は得られません。ワーカーから登録、heartbeat、poll、転送を
開始するため、ワーカー側の受信用の待ち受け は不要です。秘密鍵はファイル入力として扱い、
コントローラーやワーカーの SQLite に保存しません。

コントローラーは 全体の配置と要求された有効期間を管理し、ワーカーはローカルの
process/container/device の識別情報とクリーンアップ証拠を管理します。全体で使うリース ID は
ワーカーのレジストリ でも使う同一の正規形式の ULID です。コントローラー ID、ホスト ID、永続
host-instance ID、ゼロ以外の assignment epoch を、ソースの展開 やランタイム起動より先に
永続化します。一つの assignment の epoch は各操作を通じて固定し、個々の要求には operation ID を使います。
新しい操作によって epoch を暗黙に進めません。

通常のローカルの変更操作、照合処理、テスト、UI/browser 操作、GC は、コントローラー管理下のリースを
変更できません。ローカル force でも権限検査を回避できません。読み取りでは記録済みの管理メタデータを返し、
assignment の権限なしに暗黙の reconcile を実行しません。管理下の有効期間切れやコントローラーの停止だけでは、
ローカルクリーンアップを許可しません。

assignment は、リースと host-instance ID・epoch の組を結び付ける割り当てです。
operation は、その割り当てに対する個々の型付き要求を指します。ワーカーの incarnation は
稼働中の登録セッションを識別するもので、永続的な host-instance ID とは別です。

### 初期の操作 dispatch 方針

ワーカーは操作を一つずつ実行しますが、作成済みリースは並行して稼働します。
コントローラーは同じリースの二つ目の active 操作を、実行中のリモートテストに対する destroy も含めて拒否します。
リモートの実行中操作の cancellation と操作の並行 dispatch は専用プロトコルを必要とし、未実装です。
この方針により、live リースの並行稼働を維持しながら、operation の識別情報とローカル fence の競合を避けます。
別の操作を送信する前に `operation <operation-id>` で現在の要求を確認するか、結果を待ってください。

## Source と証拠

転送する Git 入力は commit 済みのものに限ります。各ソース alias は正確な commit と検証済み
SHA-256 bundle ダイジェストを指定します。ワーカーはランタイムに作用する前に、転送されたマニフェスト、
ソース set、選択スタックを独立に検証します。クライアントの絶対パスをワーカーのパスや CAS キーにしません。
未対応の shallow、LFS、submodule は明示的に拒否し、足りないソースを暗黙のネットワーク fetch で補いません。
リモートの `${env:NAME}` はワーカーで解決し、クライアントの環境変数の秘密値を暗黙に転送しません。

コントローラー CAS はダイジェストを検証して上限を設けた object を受け付けます。ソース object は一つあたり
1 GiB、成果物は一つあたり 64 MiB が上限です。呼出側のパスを保存先の決定に使いません。
ローカルの結果を永続化してから証拠を upload します。転送失敗はダイジェスト単位で再試行できますが、
証拠を作った操作の再実行は許可しません。リポジトリ内容と成果物は管理用の非公開データとして扱い、
保存時の暗号化は保証しません。

### 転送と応答の上限

blobのupload/downloadはメタデータ要求の固定タイムアウトを使わず、呼び出し元contextの期限とcancelに
従う。メタデータ、接続開始、TLS/headerの上限は維持する。4 MiB上限のマニフェストはcreate要求に
1回だけ含める。省略形式のパッケージはワーカーの検証前に補完し、従来形式では重複マニフェストの
一致検証を維持する。

リモートワーカーのリース応答では、重複するマニフェストとプロセスのcommand/env宣言を省略する。
ランタイムの識別情報・パス・ポート・状態・ダイジェストは維持し、設定の宣言はcreate envelope/source CASと
ローカルリースに保持する。

## 操作、不確実性、復旧

リモート操作の対象は create、list/show、renew、reconcile、destroy、名前付きテスト、logs/artifacts、
対応ワーカー上の型付き Android UI および Browser/CDP 操作です（リモート `set-text`は除外。後述のリクエスト規則を参照）。既存のローカル operation fence、
古いスナップショットの検査、リソース所有権の検査を維持します。接続先を使う操作はワーカーで実行します。
結果の `127.0.0.1` URL は worker-local であり、client-local の tunnel ではありません。

双方が operation の識別情報と payload を journal に記録します。同じ operation ID で内容が異なる要求は
拒否します。同じ要求が再配送された場合は、mutation を繰り返さず永続化した結果を返すか照合します。
dispatch 後の切断は不確実性を意味し、作用がなかった証明にはなりません。upload の復旧では結果の配送だけを
再試行します。destroy は、ワーカーが既存 lifecycle の規則に従ってローカルクリーンアップと不在を証明して初めて、
全体で RELEASED になります。

heartbeat の失効ではホストを OFFLINE、リースの観測を古い状態・UNKNOWN として扱います。
リソース不在の証明や、生存中・不確実なリースの自動再配置の許可にはしません。
同じワーカー instance が既存 assignment に再接続します。assignment が残る間、別 instance が
同じホスト名を引き継ぐことはできません。active、古い観測、不確実、未解放の assignment が残るホストの
削除は拒否します。コントローラーの再起動でも識別情報、assignment、journal を保持します。

非永続の入力経路ができるまで、リモート Android UIとBrowserの`set-text`は非対応とする。
直接プロトコルへ送信した場合も含め、CLI・コントローラー・ワーカーはjournalへの保存前にテキスト入力を
拒否する。ローカルの`set-text`は引き続き利用できる。

### 操作結果と証拠取得の分離

読み取り専用logs/artifactの結果は、復旧が不確実でもリースの状態を変更しない。
成功したreconcileでは、既に解放済みのローカルリースを確認できる。Compose表示ログの取得量と
保存済みログの合計に上限を設け、超過時はワーカーのメモリーを無制限に使わず、出力が不完全なことを
明示するエラーを返す。検証済みの保持ソースパッケージはCAS展開前に再利用する。
変更済みソースのforce クリーンアップでは、所有するワークツリーを削除する前に上限付きpatchを記録する。

確定した操作の結果と、自動的な証拠公開の結果は別に扱う。リモート応答は`evidence_status`を返し、
公開できない場合は上限付きの`evidence_error`を付ける。証拠の公開準備の失敗によって、成功したtest/UI/
ブラウザー操作を失敗した変更操作へ変えない。公開経路を復旧したら別の成果物操作で証拠を取得し、
証拠の再取得だけを理由に元の操作を繰り返さない。中断されたUI復旧では、実行中のrunと不確実性を保持する。

### ワーカーの再起動と登録拒否

恒久的な登録拒否ではワーカーを終了し、
一時的な通信障害、レート制限、サーバー障害は再試行する。

前の登録がオンラインの間は、別のワーカー incarnationへの交代を認めない。通常の再起動でも前の
heartbeatが期限切れになるまで待つ。この登録拒否は再試行できる。前のincarnationへ配信済みの
操作は新しいincarnationへpollで再配信しない。元の永続journalを持つワーカーは復旧・結果報告が
できるが、journalを失った場合は予約容量を保持したまま明示的な復旧が必要となる。
処理中のjournalと認証情報の両方を悪意をもって複製した場合を見分ける保証ではない。

## 制限と受け入れ

この段階では、信頼された一つの管理組織と一つの有効なコントローラーを前提にします。
コントローラー DB の複製は安全な フェイルオーバー を提供しません。HA、稼働中リースの移行、ホストをまたぐリース分割、
利用者から透過的に使える接続先トンネル、秘密値配布、暗黙の緊急復旧は対象外です。

受け入れには、別々の状態ルートを持つ二つのワーカーを使った実際の TLS socket テストと、
Windows、macOS、Linux のネイティブ controller/worker/client 実行が必要です。
同じホストのワーカープロセスで証明できるのはプロトコルの分離であり、物理的な複数ホストの挙動ではありません。
クロスビルドもネイティブ実行の証明にはなりません。物理マシン・VM での検証範囲、利用できない前提環境、
残る検証不足は、完了前に ExecPlan へ明記し続けます。

実 TLS の受け入れ検証は、各ランナーに二つのワーカー状態保存先を設け、`440082b` の
Windows・macOS・Linux で成功しました（run 34320519252）。配置と再起動に加え、名前付きテスト、
ログ、成果物のダウンロード、期限更新、環境変数の分離を検証しています。物理的に分かれた
ホストや VM の検証は未実施ですが、合意した範囲の受け入れは完了しています。
[品質方針](../QUALITY.ja.md)と[完了済み ExecPlan](../exec-plans/completed/multi-host-control-plane.ja.md)に
検証記録があります。責務分担は[設計](../design-docs/multi-host-control-plane.ja.md)を参照してください。
