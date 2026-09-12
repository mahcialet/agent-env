---
plan_id: EP-MHOST-001
plan_type: human-validation
status: paused
owner: maintainers
last_verified: 2026-09-09
parent: EP-OPS-001
priority: 100
base_branch: master
branch: validate/ep-mhost-001
merge_policy: manual
workstreams: [multi-host-validation]
conflicts: [multi-host-runtime]
execution_mode: human-kick
pause_reason: Awaiting provisioned multi-host environment and explicit human kick
resume_when: Maintainer provisions the required environment and explicitly authorizes a validation invocation
translation_of: docs/exec-plans/paused/multi-host-human-validation.md
source_sha256: 98b6d546348cb03f9ba694b71ac1b891555d7fc26f80b675e240a20d9623d171
---

# 提供済みのマルチホスト機能を検証する

[英語版（翻訳元）](multi-host-human-validation.md)

## 目的 / 全体像

人間が準備した環境で、提供済みのマルチホスト機能を検証します。`EP-OPS-001`の子計画として
新しい受け入れ証拠を取得します。[完了済みの実装](../completed/multi-host-control-plane.ja.md)はやり直しません。

## 進捗

- [x] 2026-09-09: 安定したシナリオIDと人間による明示的な開始条件を定義。
- [ ] ホスト、認証情報、runtime、承認済みの検証用workspaceを準備する。
- [ ] 明示的に事前確認を開始し、すべてのBLOCKED前提条件を解消する。
- [ ] 8シナリオを実行し、人が読む証拠と構造化された証拠を保持する。
- [ ] 指摘をレビュー計画で追跡し、影響するシナリオを再実行する。
- [ ] 受け入れ、マージ証拠、振り返りを照合してから完了済みへ移す。

## 想定外の発見

実環境への接続確認はまだ行っていません。マルチホストの実装はすでにマージされていたため、
実装を繰り返さず、提供済みの動作を検証します。

## 判断の記録

- 2026-09-09 / 実装担当: 環境準備と人間の明示的な許可までpausedとします。親子関係は実行依存を
  課しません。親の完了に依存すると、親が必要とする人による受け入れを進められなくなるためです。
- 2026-09-09 / 実装担当: 事前確認のREADYは実行ファイルの発見とTCP到達性だけを示します。
  認証、runtimeの健全性、シナリオのPASSは証明しません。

## 成果と振り返り

未完了です。人によるシナリオは未実行で、PASSもありません。実行後、実環境の対応範囲、
結果、証拠の使いやすさ、未解決の制約を記録します。

## 背景と構成

Controller、独立して登録した2つのworker環境、適切な範囲の有効な認証情報、承認済みのsource
リポジトリと検証manifestを準備します。該当workerにはComposeツール、Android SDKと使用可能な
AVD、常駐プロセスの検証用workload、対応browserを用意します。ローカルの事前確認には
`agent-env`、`git`、`adb`が必要ですが、worker上のruntimeのインストール状態までは検査しません。
使い捨ての検証リソースを使い、慎重な清掃の証拠を残します。認証情報、ホスト固有のパス、
機密情報を含む生ログは計画に書きません。

## 作業計画

人間が環境を準備・承認し、呼び出しごとに明示的に開始します。事前確認はREADY/BLOCKEDを返します。
人間がシナリオの操作を行い、意味のある観測と証拠への参照を添えてPASS/FINDING/BLOCKEDを記録します。
FINDINGには追跡中のdraft/activeレビュー計画IDが必要です。BLOCKEDならこのpaused計画を更新します。
コマンドがシナリオを自動実行したり状態を変えたりすることはありません。

## 具体的な手順

非公開の設定JSONを作り、`endpoints`で`controller`、`worker-a`、`worker-b`を承認したTCPの
`host:port`へ対応させます。認証情報はこの設定にも入れません。呼び出しごとに、既存の非公開
親ディレクトリの下へ新しい証拠ディレクトリを指定します。

```text
go run ./tools/repoctl plans human preflight --plan EP-MHOST-001 --kick --config <private-config.json> --evidence-dir <new-bundle-dir>
go run ./tools/repoctl plans human record --plan EP-MHOST-001 --kick --scenario EP-MHOST-001-01 --result PASS --evidence <observation.json> --evidence-dir <new-bundle-dir>
```

観測JSONには空でない`observation`と、空でない`evidence_refs`リストを含めます。
そのファイルと参照先の成果物は非公開で保持します。証拠bundleには機密情報を含みうる本文を
複製せず、SHA-256を記録します。FINDINGでは両言語のレビュー計画を作成・追跡したうえで
`--follow-up-plan <review-plan-ID>`を追加します。記録したPASSは実行者の申告であり、
独立した自動検証ではありません。

## 検証と受け入れ

[構造化シナリオ契約](../validation/EP-MHOST-001.json)がID、操作、期待する観測、PASS条件を定義します。
すべて未実行です。

| シナリオ | 操作と期待する観測 |
| --- | --- |
| `EP-MHOST-001-01` | 1つのcontrollerに対して独立登録した2つのworkerで2エージェントを実行する。Leaseは選出先に留まり、識別情報や成果物が混ざらない。 |
| `EP-MHOST-001-02` | Tokenのない要求と承認workspace外のsourceパスを試す。Leaseを作らず、認証情報を漏らさず、両方を拒否する。 |
| `EP-MHOST-001-03` | 各workerでCompose leaseを作成・確認・破棄する。Projectの所有権が分離され、所有するリソースだけを清掃する。 |
| `EP-MHOST-001-04` | 使用可能なEmulator leaseを確保し、readyを確認して破棄する。AVDとportを排他予約し、清掃で不在を確認・記録する。 |
| `EP-MHOST-001-05` | 常駐プロセスleaseを起動し、ログを取得して破棄する。プロセスと子孫の識別・終了を確認し記録する。 |
| `EP-MHOST-001-06` | プロセス上のbrowser leaseでページ一覧、snapshot、対応操作を実行する。リモート転送後もbrowser識別とsnapshotの出所を維持する。 |
| `EP-MHOST-001-07` | 所有するlease endpointへTCP転送し、検証データを送ってleaseを閉じる。所有する接続先だけに届き、清掃後listenerが消える。 |
| `EP-MHOST-001-08` | Controller再起動、worker切断・再接続、同じoperation IDの再試行、秘匿処理済み証拠の確認を行う。Receiptで重複作用を防ぎ、曖昧なリソースを可視化し、公開証拠に秘密を残さない。 |

すべての期待する観測を裏付ける証拠を残して初めてPASSとします。予想外の動作はFINDING、
前提不足はBLOCKEDでありPASSではありません。Bundleには`evidence.json`と`evidence.md`が入ります。
結果と後続Plan IDを進捗へ記録します。コマンドのaction_required表示だけで更新を済ませません。

## 冪等性と復旧

証拠ディレクトリを再利用せず、既存bundleを上書きしません。書き込みが中断したら不完全な
bundleを残し、原因を解消して新しいディレクトリを使います。操作の再試行前にruntime状態を
読み直し、曖昧なリソースを強制清掃しません。以前の開始許可は次の呼び出しの許可にはなりません。

## 成果物と注記

明示的な承認がなければ、範囲を絞った秘匿処理済み成果物はリポジトリ外に保持します。
Bundleにはendpointアドレス、認証情報、生の観測、詳細なエラーを含めません。安定したシナリオID、
結果、時刻、観測digest、必要な後続対応を残します。実際のホスト値は永続文書に書きません。

## インターフェースと依存

ハーネスは事前確認と証拠形式を担当し、製品動作は引き続き`agent-env`が担当します。
親は`EP-OPS-001`ですが、親への実行依存はありません。グラフ選出は人による検証を自動実行しません。
完了には、この計画自身の受け入れ、振り返り、マージ証拠、両言語の移動が必要です。

Androidシナリオ`EP-MHOST-001-04`では、SDK・emulator・ADBを指定workerに用意します。
clientに必要なのは`agent-env`と`git`であり、ローカルAndroidツールは不要です。
client経由でEmulator leaseを作成し、worker/lease識別情報とworker側ADBの起動・準備完了を
記録した後、leaseを破棄してそのworker上での消失を確認します。worker側Androidツールが
不足する場合はBLOCKEDとします。clientローカルのADBやTCP preflightだけでは合格しません。
