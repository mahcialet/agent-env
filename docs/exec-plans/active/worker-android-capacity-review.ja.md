---
translation_of: docs/exec-plans/active/worker-android-capacity-review.md
source_sha256: 5eb023155e1fc1d428b56fbfa378df3b376c4b5e4cd20a8e13d9efaa726daa87
status: active
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
- [ ] 2026-09-09: push・返信・Resolveを完了し、このPlanをarchiveする。

## Surprises & Discoveries

lease数の起動前検査には上限があるが、Android容量は負数しか検査していなかった。

## Decision Log

2026-09-09、実装担当: SQLite allocatorのポート範囲から公開容量を導出し、CLIの接続箇所で再利用する。controllerからlocal storeへの依存は追加しない。

## Outcomes & Retrospective

作業中。

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
