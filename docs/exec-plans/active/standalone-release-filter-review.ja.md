---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/active/standalone-release-filter-review.md
source_sha256: 7e1c1c772772da1028249c90cc887cb4d46adf0047db779fadbf77ad039c4113
---

# リリースcheckoutのfilterを隔離する

[English](standalone-release-filter-review.md)

## Purpose / Big Picture

PR #7のrelease checkoutが外部のGit属性・filterで変換されないようにする。

## Progress

- [x] 未解決のfilter指摘を確認。
- [x] 2026-09-08: global/system fixtureが修正前に両方失敗し、設定隔離後に成功。
- [ ] 検証・push・返信・Resolveを行う。

## Surprises & Discoveries

smudgeがコンパイラ入力を変えても、対応するclean filterがGit statusから変化を隠せる。

## Decision Log

2026-09-08 / maintainers: release用Gitをglobal/system設定・属性から隔離する。リポジトリ内のGit識別と通常の実行ファイル検索は維持する。

## Outcomes & Retrospective

未完了。

## Context and Orientation

想定ブランチ: `feat/standalone-release-finalization`。AGENTS.md・ARCHITECTURE.md・docs/PLANS.mdに従う。対象はtools/repoctl/release_source.goとテスト、英日の配布設計。

## Plan of Work

使い捨てリポジトリで変換済みかつcleanのcheckoutを再現し、設定を隔離して厳密なsource検証を維持する。

## Concrete Steps

修正前後の対象回帰、repoctl check、repoctl raceを実行。commit/push後に返信してResolveする。

## Validation and Acceptance

global/systemのfixtureは通常checkoutを実際に変換し、release専用checkoutはcommit内容と一致すること。harness/sourceテストが成功し、threadをResolveすること。

## Idempotence and Recovery

子プロセス環境と引数だけを変え、ユーザーのGit設定や公開refsは変更しない。

## Artifacts and Notes

指摘: https://github.com/mahcialet/agent-env/pull/7#discussion_r3954872828 。Go 1.27.1でsource/filter回帰、`go run ./tools/repoctl check`、`go test -race ./tools/repoctl`、`AGENT_ENV_RELEASE_CANDIDATE=build go test ./tools/repoctl -run TestReleaseCandidate -count=1` が成功。最後の検査は6ターゲットの実成果物と18ケースを含む。独立した読み取りレビューでも重大な問題は確認されなかった。

## Interfaces and Dependencies

依存やreleaseコードのPOSIX shellを追加しない。テストのGit filterはGit経由でnative Go test helperを実行する。

