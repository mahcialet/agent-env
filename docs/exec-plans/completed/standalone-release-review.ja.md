---
status: completed
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/completed/standalone-release-review.md
source_sha256: 645f537c5ccf88dc2aed484dd9b6c50c58d3fb9b5795ca06d7ee1b656f4750dc
---

# PRレビュー後のリリース出力stagingを修正する

[English](standalone-release-review.md)

## Purpose / Big Picture

PR #7 の thread PRRT_kwDOURHsR86gHJ1z に対応する。dirty source の検査を弱めず、非ignoreの作業ツリー内にも新規出力できるようにする。

## Progress

- [x] 2026-09-08: cleanなブランチと未解決threadを確認。
- [x] 2026-09-08: 6ターゲットの成果物を作業ツリー内へ出力するテストが修正前にclean-tree診断で失敗した。
- [x] 2026-09-08: ツリー外の生成と出力先隣での配置を実装し、実候補・cleanup・source guard・harness・repoctl raceが成功。
- [x] 2026-09-08: 修正6912ec6をpushし、discussion_r3954818214へ返信してResolveを確認。

## Surprises & Discoveries

出力先の隣に作ったstagingが最終Git検査で未追跡変更になる。既存検証はツリー外またはignoreされた出力先のみだった。

最初は古い候補を再利用したため、識別情報の検査で正しく失敗した。opt-inテストに `AGENT_ENV_RELEASE_CANDIDATE=build` を加え、現在のHEADから隔離clone内で新規候補を作れるようにした。

## Decision Log

2026-09-08 / maintainers: sourceのclean検査を維持する。ツリー外で生成・検証し、source検査後に検証済みbytesを出力先の隣の専用領域へコピーして同一filesystem内で配置する。

## Outcomes & Retrospective

2026-09-08完了。commit `6912ec6` で、source変更をclean検査から除外せず、非ignoreの作業ツリー内へ出力できるようにした。実際に作業ツリー内へ出力するテストは修正前に失敗し、修正後に成功。既存bytes、source guard、所有する一時領域のcleanupも検証した。独立した読み取りレビューで重大な問題は確認されなかった。対象1threadへ返信してResolve済み。英日版をarchivalし、hosted CIと下記の成功済みローカル証拠を区別する。

## Context and Orientation

想定ブランチ: `feat/standalone-release-finalization`。前回の成果は[リリース完了Plan](../completed/standalone-release-finalization.ja.md)。対象は `tools/repoctl/release.go`、`tools/repoctl/release_e2e_test.go`、英日の配布設計。

## Plan of Work

作業ツリー内への実出力を確認するテストを先に追加し、生成stagingを移す。既存出力の保護と失敗時のcleanupを維持する。

## Concrete Steps

修正前後に候補の出力先に関する不具合を再検出するテスト、repoctl check、raceを実行。対象変更のみcommit/pushし、証拠付きで返信してthreadをResolveする。

## Validation and Acceptance

非ignore・日本語を含む階層下への出力が成功し、既存候補とbytesが一致すること。dirty sourceと既存出力保護も成功すること。全threadへ返信しResolveすること。

## Idempotence and Recovery

callerの出力は削除しない。専用の一時stagingだけを削除する。無関係なtracked/index/untracked変更の検査を維持する。

## Artifacts and Notes

レビュー: https://github.com/mahcialet/agent-env/pull/7#discussion_r3954761466 。ローカルGo 1.27.1で `AGENT_ENV_RELEASE_CANDIDATE=build go test ./tools/repoctl -run TestReleaseCandidate/nonignored_worktree_output -count=1 -v` が修正前にclean-tree診断で失敗した。修正後は実候補18 subtest、配置cleanup/保護、source guardが成功。`go run ./tools/repoctl check` と `go test -race ./tools/repoctl` も成功し、テストは公開refsを変更していない。

## Interfaces and Dependencies

依存・コマンド・runtime動作は追加しない。Goのnative filesystem操作を用い、一時領域と出力先が別filesystemでも扱う。

