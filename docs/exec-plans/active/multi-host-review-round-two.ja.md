---
translation_of: docs/exec-plans/active/multi-host-review-round-two.md
source_sha256: 86e38585910b5f0ca299a7b53128248c50dba7db38b2443daefb7e39044e3b1f
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Multi-host レビュー第2回

[英語版](multi-host-review-round-two.md)

本ExecPlanは`feat/multi-host-control-plane`でPR #12の追加指摘に対応するための実行計画である。
完了済みのレビュー追補は過去の記録として保持する。

## 目的 / 全体像

CLI controllerで期限処理を動かし、blob転送をcontextの期限で制御する。
manifestの二重送信をなくし、応答済みCAS公開の永続性を確保する。

## 進捗

- [x] 追加4件のThreadを読み、完了済み修正との関係を確認した。
- [x] CLI controllerの起動を共通の期限処理へ接続する。
- [x] blob streamの期限をmetadata要求の上限から分離する。
- [x] manifestを1回だけ送信し、workerの検証を維持する。
- [x] 移植性を維持してCAS directory公開を永続化する。
- [ ] 検証・push・4件への返信とResolveを終え、このPlanをarchiveする。

## 想定外の発見

前回の期限監視テストはserver.Runを検証していたが、CLI serveには独自のlistener管理があり、
実際のCLI起動は検証済みの監視処理を通っていなかった。

修正前の実CLIテストは5.021秒でcleanup未予約を再現し、修正後は1.032秒で成功した。
clientのupload/downloadは短縮したmetadata期限で失敗し、uploadのcancelでは本文のclose不足も
見つかった。serverのTLSテストは期限解除を無効化すると失敗し、修正後はrace1.724秒で成功した。
上限付近のmanifestは旧CLI/worker overlayで失敗し（1.950秒/0.331秒）、修正後は実mTLSの
create・取得・pollとworker検証がrace33.913秒/3.147秒で成功した。

CLIテストは当初home/logの指定を誤り、修正してから本来の期限検査欠落を再現した。
独立SQLite接続にはserverの正常なtransactionと調整するため、上限付きのbusy timeoutを設定した。
manifest fixtureのCA名とcapabilityも修正した。サイズ上限・認証検証は維持した。
CASからinstanceへの依存はarchitecture検査で拒否されたため、規則を変更せずWindowsのnative
byte-range lockで公開を管理する方式へ変えた。

## 判断の記録

controllerの起動経路を1つにまとめ、実際のCLI入口をテストする。
省略したpackageのmanifestは明示的に補完する。従来の完全なpackageも受け付けるが、
manifestの不一致検証は緩めない。streamは呼び出し元のcancelに従う。

CASはUnixのdirectory fsync、Windowsのwrite-through moveとcancel可能・上限付きnative lockを
使う。重複・再試行でも成功応答前に永続化を確認する。moveとlockでextended/UNC変換を共有する。
物理的な電源断への保証を実証したとは扱わない。

CLIの起動経路、stream認可、CAS lockと公開を独立レビューした。Windowsのnative動作はCIで確認する。

## 成果と振り返り

実装・検証中。

## 背景と構成

CLIのremote_servicesがserviceを起動し、server.Runが期限検査と終了を管理する。
clientがHTTP通信、remotesource/workerがpackage検証、blobstoreがCAS公開を担う。

## 作業計画

各境界を修正前に失敗する回帰テストと独立レビューで補強する。
日英の契約を更新し、過去のCI失敗の記録を保持する。

## 具体的な手順

対象Goテスト、repoctl check、全体race、native multi-host fixtureを実行する。
検証済みcommitを通常pushし、CIを確認してから完了を記録する。

## 検証と受け入れ

実際のCLI serverで、offlineかつ操作待ちのない期限切れleaseにcleanupを予約する。
blob転送はmetadata要求の上限を超えて継続でき、contextのcancelでは停止する。
上限付近のmanifestがcreate/pollに収まり、検証は維持される。
CASは再試行も含め、platformに適した永続化を終えてから成功応答する。

## 冪等性と復旧

解放証明を緩めず、履歴を書き換えない。公開失敗の記録を残し、同期前の既存公開物を
再試行で成功扱いにしない。

## 成果物と注記

PR #12 Thread: PRRT_kwDOURHsR86gk8fD、PRRT_kwDOURHsR86gk8fG、
PRRT_kwDOURHsR86gk8fL、PRRT_kwDOURHsR86gk8fQ。

## インターフェースと依存

controllerをhost runtimeから独立させ、shell依存を追加しない。

検証済み: 全体の`go test -race ./...`と2 workerのnative CLI fixture（21.073秒）が成功した。
最終harness・Docker統合・push後のnative CIは確認待ち。

結果の境界を独立レビューし、上限付近のmanifestで完了済みのworker応答がfailedへ変わる問題を
再現した（0.119秒）。remote応答だけをcopyして重複manifestとprocessのcommand/env宣言を省略し、
localの保存状態と4 MiBの結果上限を維持する。

最終local検証: repoctl checkとDocker統合が成功した。最終変更packageのraceも成功した
（worker8.806秒/CLI43.366秒、server/blobstoreも成功）。大きな応答のテストで完了状態・元の入力・
合計8MiBのoperation envelope上限を維持することを確認した。省略形式のdownload補完を含め、
native multi-host CLIも成功した。
