---
status: completed
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/completed/standalone-release-filter-review.md
source_sha256: e4490ed34fc039718fd4fa573c204783280d69f686f178996ed9b60e7f0f3787
---

# リリースcheckoutのfilterを隔離する

[English](standalone-release-filter-review.md)

## Purpose / Big Picture

PR #7のrelease checkoutが外部のGit属性・filterで変換されないようにする。

## Progress

- [x] 未解決のfilter指摘を確認。
- [x] 2026-09-08: global/system fixtureが修正前に両方失敗し、設定隔離後に成功。
- [x] 2026-09-08: 検証して3010931をpush。discussion_r3954932030へ返信し、thread PRRT_kwDOURHsR86gHb58のResolveを確認。

## Surprises & Discoveries

smudgeがコンパイラ入力を変えても、対応するclean filterがGit statusから変化を隠せる。

## Decision Log

2026-09-08 / maintainers: release用Gitをglobal/system設定・属性から隔離する。リポジトリ内のGit識別と通常の実行ファイル検索は維持する。

## Outcomes & Retrospective

2026-09-08完了。release checkoutはglobal/system filter設定・外部属性・templateを継承しない。両filterによるrelease checkoutの改変を防ぐテスト、source guard、実候補、harness、repoctl raceが成功。指摘に返信してResolve済み。隔離処理とテストはユーザーのGit設定や公開refsを変更していない。hosted CIは下記の成功済みローカル証拠と区別する。

## Context and Orientation

想定ブランチ: `feat/standalone-release-finalization`。AGENTS.md・ARCHITECTURE.md・docs/PLANS.mdに従う。対象はtools/repoctl/release_source.goとテスト、英日の配布設計。

## Plan of Work

使い捨てリポジトリで変換済みかつcleanのcheckoutを再現し、設定を隔離して厳密なsource検証を維持する。

## Concrete Steps

修正前後のcheckoutがfilterで改変されないことを確認するテスト、repoctl check、repoctl raceを実行。commit/push後に返信してResolveする。

## Validation and Acceptance

global/systemのfixtureは通常checkoutを実際に変換し、release専用checkoutはcommit内容と一致すること。harness/sourceテストが成功し、threadをResolveすること。

## Idempotence and Recovery

子プロセス環境と引数だけを変え、ユーザーのGit設定や公開refsは変更しない。

## Artifacts and Notes

指摘: https://github.com/mahcialet/agent-env/pull/7#discussion_r3954872828 。Go 1.27.1でsource/filterの検証漏れを再検出するテスト、`go run ./tools/repoctl check`、`go test -race ./tools/repoctl`、`AGENT_ENV_RELEASE_CANDIDATE=build go test ./tools/repoctl -run TestReleaseCandidate -count=1` が成功。最後の検査は6ターゲットの実成果物と18ケースを含む。独立した読み取りレビューでも重大な問題は確認されなかった。

## Interfaces and Dependencies

依存やreleaseコードのPOSIX shellを追加しない。テストのGit filterはGit経由でnative Go test helperを実行する。

