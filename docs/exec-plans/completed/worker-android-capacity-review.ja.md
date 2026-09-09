---
translation_of: docs/exec-plans/completed/worker-android-capacity-review.md
source_sha256: f3fe9fb88fa12826f7255c27653a6a93de39456861491a949bf4a5ba87dc2ffb
status: completed
owner: maintainers
last_verified: 2026-09-09
---

# Worker Android容量レビュー

[英語版](worker-android-capacity-review.md)

本ExecPlanは`feat/multi-host-control-plane`でPR #12の容量修正を担当する。

## Purpose / Big Picture

workerのAndroid容量がallocatorの枠数を超える場合、登録や状態作成の前に拒否する。

## Progress

- [x] 2026-09-09: 新規Threadと既存allocatorの範囲を確認した。
- [x] 2026-09-09: 境界値の拒否を実装・検証する。
- [x] 2026-09-09: push・返信・Resolveを完了し、このPlanをarchiveする。

## Surprises & Discoveries

lease数の起動前検査には上限があるが、Android容量は負数しか検査していなかった。

## Decision Log

2026-09-09、実装担当: SQLite allocatorのポート範囲から公開容量を導出し、CLIの接続箇所で再利用する。controllerからlocal storeへの依存は追加しない。

## Outcomes & Retrospective

`07c83b4`で完了した。広告するAndroid容量はallocatorの固定枠数を超えられず、
不正な値はリソース作成・登録の前に拒否する。追加テストは元の不具合と両境界を検証し、
既存の枯渇・rollbackテストで実際の枠数も確認した。独立レビューで問題はなかった。
広告する上限を割当範囲から導出し、別の容量値を独立に管理しないようにした。

検査40件はすべて成功状態になった。PR Verify34341921634、push Verify34341916259のattempt2、
Multi-host native34341916300/34341921496、Browser native34341916340/34341921586、
Release preview34341921491である。push側Windows Go1.27の初回タイムアウトは後述のとおり
記録し、未変更の失敗job再実行が成功した。対象appのraceも手元で5回成功した（4.060秒）。
初回の一過性タイムアウトの根本原因を立証したとは主張しない。timeoutや検証条件は変更していない。
最後のarchive commitは文書のみで、repoctl docs-checkで別途検証する。
nativeの証拠は上記のコードcommitに対するものである。

## Context and Orientation

internal/cli/remote_services.goが枠数を広告し、internal/store/sqlite/android.goが5554〜5682の偶数console portを管理する。

## Plan of Work

境界値の回帰テスト、allocator容量の共有、日英のflag説明を追加する。

## Concrete Steps

修正前の対象テスト、repoctl check、対象raceテストを実行し、push後にnative CIを確認する。

## Validation and Acceptance

負数と65超はTLS・状態初期化前に拒否し、0と65は通常のTLS事前検査へ進む。allocatorは引き続き65予約で枯渇する。

## Idempotence and Recovery

状態移行やallocator拡張は行わず、通常のcommitで進める。

## Artifacts and Notes

PR #12のThread PRRT_kwDOURHsR86gmqDy。

## Interfaces and Dependencies

CLIは既にSQLiteを参照している。allocatorのポート範囲は維持し、新規依存やshell workflowは追加しない。

検証証拠: 修正前のCLI境界値テストは66と1000で失敗した（0.005秒）。
`go run ./tools/repoctl check`が成功し、unit・vet・文書・生成物・architecture検査を通過した。
CLI/SQLiteのraceも成功した（46.073秒/15.922秒）。既存の実65予約による枯渇・rollbackの
回帰テストも含む。最初のharnessは新規日本語Planの英語版リンク不足を検出した。
リンクを追加し、検査規則を変更せずharnessが成功した。

独立した読み取り専用レビューで問題はなかった。`07c83b4`をpushし、
Thread PRRT_kwDOURHsR86gmqDyに修正・検証内容を返信してResolveした。
PR Verify34341921634はWindows Go1.26/1.27を含む全jobが成功した。
push Verify34341916259のattempt1ではWindows Go1.27の既存テスト
TestMobileConcurrentNamedTestsUseOwnedSerialAndPersistWarningがapplications_test.go:585で
失敗した。既存の10秒context内で2つのfixture leaseを作成中だった（18.74秒）。
Android allocatorの差分は同値の定数への置換であり、割当動作は変えていない。
CLIの検査はこのapp fixtureでは実行されない。競合の具体的な原因は立証していない。
失敗証拠を残し、ソースと検証条件を変えず失敗jobだけを再実行する。成功させるための
timeout延長は行わない。
