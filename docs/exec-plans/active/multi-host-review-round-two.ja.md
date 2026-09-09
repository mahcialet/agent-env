---
translation_of: docs/exec-plans/active/multi-host-review-round-two.md
source_sha256: 118a19f10a429667f8696dd35b2c08eadaeba898cfa6135c9bf4e66b0efc03af
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
- [x] remote force cleanupで上限付きsource diffを保存する。
- [x] 検証済みpackageを再展開前に再利用する。
- [x] 明示reconcileで解放済みleaseを確認する。
- [x] 稼働中のworker root複製によるincarnationの奪取を防ぐ。
- [x] 読み取り専用logs/artifactsの不確実な結果でlease状態を変えない。
- [x] remote結果の構築前にComposeログのcapture量を制限する。
- [ ] 検証・push・追加Threadすべてへの返信とResolveを終え、このPlanをarchiveする。

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

PR #12 Thread: PRRT_kwDOURHsR86gk8fC、PRRT_kwDOURHsR86gk8fG、
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

`3dba741`のpush後に追加6件が入り、本active Planで引き続き対応する。
PRRT_kwDOURHsR86glT_l、PRRT_kwDOURHsR86glT_s、PRRT_kwDOURHsR86glT_y、
PRRT_kwDOURHsR86glT_2、PRRT_kwDOURHsR86glT_7、PRRT_kwDOURHsR86glUAB。
対象はcleanup証拠、package再利用、reconcileの証明、workerの交代制御、読み取り専用操作の
状態保持、Composeログの上限である。

追加の発見: 実17MiBのGit diffで、埋め込みbytes.BufferのReadFromが旧buffer上限を迂回した。
埋め込みをやめて上限を実効化し、remotesource全raceが成功した（5.549秒）。Composeの共通上限は
必須cleanup証拠にも影響し、多量ログのserviceを削除できなくしたため、最終的には上限付きの
表示経路を既存のcleanup契約から分ける。Androidの表示ログも集計後ではなく読み込み前に制限する。

`3dba741`のMulti-host native・Browser native・Release previewは全platformで成功した。
push Verify34333854477では、serverの正常EOFがclientのcancel確認より先になるfixture競合が出た。
cancel確認までhandlerを未完了に保ち、厳密なエラー判定を維持した。race5回の反復が成功した
（16.236秒）。PR Verify34333859610では、未変更のTestLifecycleCreatePersistedIntentAndUniqueIsolationが
lifecycle_test.go229でcontext deadline exceededとなった。証拠を保持し、次commitのCIで確認する。

追加6件の検証: store/serverのraceが成功した（2.241秒/2.969秒）。online交代拒否、offline後の
再配信防止、所有者不明の移行、読み取り専用状態保持を含む。workerの補償済みcreateから
reconcile/controllerの解放確認が成功し、appのraceも成功した（52.942秒）。offline判定30秒を含む
native CLI再起動fixtureが成功した（46.249秒）。実Docker/Podmanのremote Compose fixtureも成功した
（7.34秒/21.61秒）。これにより本番CLIの上限付き表示adapterを確認した。独立レビューで欠落した
adapterの転送methodを検出し、push前に修正した。

削除後のcacheへの懸念は再現しなかった。削除対象はlease worktreeであり、bare cache repositoryは
残る。保存した回帰テストで、削除後に空CASを使うlogs/artifact/reconcile/再destroyを確認した。
検証条件は緩めていない。source diffと再利用は修正前の失敗も確認済み。実17MiB diffは実効化した
capture上限で安全に失敗し、cleanupの安全性を維持する。
