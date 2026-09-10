---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/exec-plans/completed/multi-host-review-followup.md
source_sha256: 22a8131d72067a850efa837bf5881d6ca0c185e769b1a1bc5ab9e597fe2e0023
---

# PR 12のcontrol-planeレビューに対応する

[English](multi-host-review-followup.md)

## 目的 / 全体像

native runtimeの所有権と保守的なcleanup保証を維持し、PR 12の7件を修正する。
想定ブランチは`feat/multi-host-control-plane`。完了したmulti-host-control-plane実装への
レビュー修正は本Planに従う。ユーザーは実装・push・各threadへの返信とResolveを明示的に依頼した。

## 進捗

- [x] 2026-09-09: harnessを読み、PR headの作業ツリーがcleanであることと未解決7件を確認した。
- [x] 別CLI processから再試行してもremote createの暗黙ownerを安定させる。
- [x] controllerが管理するTTLを永続化し、期限切れのcleanupを保守的に予約する。
- [x] workerの広告capacityをlocal policyに照らして検証する。
- [x] 一時的なUI/Browser入力textをcontroller/worker journalへ保存しない。
- [x] active操作があるhostの削除と、削除後の新規操作を拒否する。
- [x] worker登録の恒久エラーを再試行し続けない。
- [x] 作用開始後の全create失敗で不確実性を保持し、安全な後続cleanupを可能にする。
- [x] 指摘された不具合を再検出するテスト・全harness/race・native CIを実行し、統合差分をレビューする。
- [x] 修正をpushし、対応済みthreadへ返信してResolveし、本Planを日英でarchiveする。

## 想定外の発見

一部のコメントは短いため、文面だけでなく実装と既存契約から具体的な失敗と、同じ不具合を検出するための条件を確認する。

修正前の不具合再現テストで、別processのowner変化、容量超過がTLS設定まで進む挙動、未完了操作を持つ
hostの削除、期限切れ処理の欠落、恒久的な登録拒否の再試行、曖昧なcreateのfailed判定、worker
journalへの平文入力受け入れを再現した。独立レビューでは過去の不正TTLによる起動失敗も再現した。
過去の記録の移行だけは元のcreate時刻と既定TTLを使い、不正な過去のrenewを無視する。
新規要求の不正TTLは引き続き拒否し、移行でcapacityを解放しない。

## 判断の記録

- controllerの配置/TTL管理と、workerのlocal resource/cleanup管理を分離する。
- 通常のlocal owner既定値は維持し、remote再試行の識別情報を安定させる。
- 永続的な操作queueではlocalの非永続text入力の契約を実現できない。
  対応する経路ができるまで、remote text入力を送信・保存前に拒否する。
  黙って伏せ字へ変えて別の値を実行しない。

- TTLには受理時刻を使い、cleanupのoperation記録を永続化する。起動時、1秒ごと、poll時に
  期限を調べ、終了時は監視処理の停止を待つ。
- 補償済みcreateのpayloadは維持するが、releasedのLocalState確認はdestroy専用とし、
  不確実なcreateのcapacityを保守的に保持する。
- worker/protocolとcontroller/CLIを独立レビューした。移行の指摘を修正し、元の再現テストが
  race有効で成功した。

## 成果と振り返り

7件すべてを`80d031d`で修正し、個別に返信してResolveした。localの全体harness、race、
Docker統合、2 workerのnative fixture、独立レビュー、4種類のCI workflowが成功した。
push Verify 34330840976も変更なしの失敗job再実行で成功した。初回のWindows失敗は後述のとおり
記録した。再発しなかったことだけでSQLiteの具体的な競合原因が確定したとは扱わない。
非永続の通信経路ができるまでremote text入力は明示的に非対応とする。本レビューPlanを完了し、archiveする。

## 背景と構成

指摘対象はCLIのremote request生成、controller SQLite state、worker受領/結果journal、
appの予約境界である。[完了した実装Plan](../completed/multi-host-control-plane.ja.md)、
product契約、ADR 0006/0007の責務境界と移植性要件を維持する。

## 作業計画

CLI、controller、workerの担当ファイルを分け、rootが統合する。可能な範囲で修正前に失敗する
同じ不具合を検出するテストを追加する。TTLと非対応のremote text入力について日英の契約を更新する。
統合後にrepository harnessと該当するnative/runtime検査を実行し、各threadへの返信に
具体的な修正と検証の証拠を対応付ける。

## 具体的な手順

repositoryのGo toolchainで`go run ./tools/repoctl check`、`go test -race ./...`、
対象package/native fixtureを実行する。履歴は書き換えず、既存PRブランチへ通常のcommit/pushを行う。
修正または根拠ある対応不要の判断を確認してから返信とResolveを行う。

## 検証と受け入れ

別processのcreate再試行で暗黙ownerを含むrequest識別が一致する。controllerの期限は再起動と
renew再試行で維持され、期限切れでもcleanup証明までcapacityを保持する。host削除でqueued/dispatched
操作を取り残さない。不正capacityはsetup前に失敗する。一時textは直接APIを含めて両journalへの
保存前に拒否する。恒久的な登録エラーは終了し、一時障害は再試行する。localで補償処理したcreateも
含め、作用開始後の失敗は信頼できるcleanup確認まで不確実性を保持する。
完了前に該当harness/CI結果とthreadの対応結果を記録する。

## 冪等性と復旧

永続replay/fenceと保守的な不在証明を維持する。過去の失敗や機密情報のリスクを隠すために
既存journalを書き換えない。保存stateが変わる場合は追加的なmigrationを行う。

## 成果物と注記

PR: https://github.com/mahcialet/agent-env/pull/12
初回レビュー対象head: `efb12a0`。

## インターフェースと依存

controllerをapp/provider実装から独立させる。共通protocol検証でCLI/controller/workerの
入口にて一時textを拒否できるようにする。広告するworker capacityはlocal app policyを上限とする。

検証記録（2026-09-09）: `repoctl check`と`go test -race ./...`が成功した。
移行・終了処理の最終変更後もstore/serverのraceが成功した（2.439秒/2.426秒）。
独立担当によるworker/protocolのraceも成功した（9.413秒/1.022秒）。
2 workerを使う`TestMultiHostNativeCLI`が成功し、実際のTLS経由のCLI操作を確認した。
JSONの重複・大文字小文字違いの入力検証とDB/WALのcanary検査で、拒否したtextがjournalに
残らないことを確認した。native CIとThreadの完了は、検証済みcommitのpush後に確認する。

Docker daemonを使うlocalの`repoctl test-integration`も成功した。`80d031d`を通常pushし、
7件すべてのレビューThreadへ修正と検証を個別返信してResolveした。native CIは確認中。

`80d031d`のPR Verify 34330844403、Multi-host native 34330844402、Browser native
34330844436、Release preview 34330844500は成功した。同時実行のpush Verify 34330840976では
既存Windows検証が失敗した。`TestExitAndTimeout/exit`は100msの期限付近で期待出力前に終了し、
`TestConcurrentColdOpen`は初期化の期限を超過した。今回の修正では対象のexecx/local SQLite実装と
テストは変更していない。失敗を成功扱いにせず記録し、期限・assertion・並列数を変更せずに
失敗jobを再実行した。再実行結果は確認中。

SQLiteの失敗はテスト専用の期限ではなく、本番の初期化上限10秒に達したもの。競合や負荷の
影響は考えられるが、具体的なlock保持元は未確定である。再発時の調査のため証拠を保持する。

最終確認: Verify 34330840976の再実行が成功した。レビューThreadは7件すべてResolve済み。
再実行のためにコードやテストの制約を緩める必要はなかった。
