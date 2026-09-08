---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/active/process-destroy-preview-review.md
source_sha256: dd95404b8e946983bb5a70cf1c042b52484ecf14dedab0ffc1db656ab6351520
---

# Process destroyプレビューのレビュー対応

[English](process-destroy-preview-review.md)

## Purpose / Big Picture

`feat/persistent-process-runtime`でPR #9の常駐processのdestroyプレビューを修正する。本Planが今回のレビュー対応を管理し、完了済み実装Planは過去の証拠として保持する。

## Progress

- [x] (2026-09-08) レビュー、architecture、cleanup契約を確認。
- [x] (2026-09-08) processプレビューを再現・修正・検証。
- [ ] 修正をpushし、レビューThreadへ返信してResolve。

## Surprises & Discoveries

processの観測経路は正しかったが、稼働中processのプレビューがComposeの説明に流れていた。終了済みprocessにも専用状態のcleanup説明が必要。最初の対象回帰testではforce有無の両方で稼働中/終了済みが失敗し、production code変更前に指摘を再現した。

## Decision Log

2026-09-08、maintainers: 所有確認後、Compose分岐の前にprocess分岐を追加。観測エラーは既存のquarantine経路を保つ。adapterやnative lifecycleは変更しない。

## Outcomes & Retrospective

process専用分岐でnative識別情報確認、稼働中なら停止、log保持、tree全体の不在確認、専用状態削除を説明する。修正前に稼働中と終了済みのプレビューが失敗し、修正後は状態/forceの全6ケースが成功した。既存Androidプレビューの回帰も成功。

## Context and Orientation

`internal/app/destroy_preview.go`が読み取り専用の`Destroy`プレビューを整形する。`internal/app/process_lifecycle_test.go`にprocess/source/store fixtureがある。

## Plan of Work

公開Destroy経路の回帰を追加し、稼働中・終了済み・所有不明をforce有無の両方で検証する。registry、event、artifact、専用状態、provider呼出回数が変わらないことを確認し、説明を修正する。

## Concrete Steps

修正前後の対象test、`repoctl check`、対象race testを実行する。確認した差分のみcommit/pushし、具体的証拠を返信する。

## Validation and Acceptance

runtime名と専用状態path、識別情報確認・停止・log保持・tree全体の不在確認後のcleanupを表示する。processのみのleaseにCompose説明を出さない。所有不明はquarantineとし、全プレビューが読み取り専用であることを検証する。証拠: `go test -race ./internal/app -run 'Test(ProcessDestroyPreviewReportsCleanupWithoutEffects|AndroidDestroyPreviewReportsPrivateCleanupWithoutEffects)' -count=1`成功。`go run ./tools/repoctl check`成功（unit、vet、docs、generated、architecture検査）。

## Idempotence and Recovery

dry-runは安全に繰り返せる。公開履歴と実cleanupの意味を変更しない。検証失敗を解消まで記録する。

## Artifacts and Notes

レビュー: https://github.com/mahcialet/agent-env/pull/9#discussion_r3958123366

## Interfaces and Dependencies

新しいinterface、依存、OS固有動作、shell呼出は追加しない。

