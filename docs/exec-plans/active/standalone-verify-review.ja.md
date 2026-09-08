---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/active/standalone-verify-review.md
source_sha256: e8db1ae59302ac95761b80a54aa17f3bb06bc16697e7295db71aa808ce6e3823
---

# release検証で呼び出し元の隠れたindexフラグを拒否する

[English](standalone-verify-review.md)

想定ブランチ: `feat/standalone-distribution`。開始revision: `385e27f`。
本PlanはPR 6の新規指摘を担当します。完了済み配布Planは過去の証拠として保持します。

## 目的 / 全体像

release-verifyでbuild開始前に、文書に定めた呼び出し元のclean条件を検査します。

## 進捗

- [x] 2026-09-08: harness、cleanなブランチ、未解決1Threadを確認。
- [x] 2026-09-08: コマンド入口の検査漏れを再現し、clean検査を共通化。
- [ ] 2026-09-08: localとnative CIで検証し、push・返信・Resolve。
- [ ] 2026-09-08: 成果を記録し日英をcompletedへ移動。

## 想定外の発見

新規の入口テストは修正前に4ケースすべて失敗しました。indexフラグを拒否せず、build準備のLICENSE取得へ進んでいました。検査の共通化後はsourceとコマンド入口の回帰テストが成功しました。

前回の回帰テストはreleaseVersionだけを対象にしていました。release-verifyは呼び出し元をporcelainで検査し、releaseVersionをcleanなprivate cloneだけに適用するため、呼び出し元のindexフラグを見落としていました。

## 判断の記録

- 2026-09-08 / maintainer: 既存のフラグ対応clean検査を関数へ抽出し、release-verify入口でも使用します。private preview tagと呼び出し元indexを維持し、helperだけでなくコマンド入口を検証します。

## 成果と振り返り

実装と検証は未完了です。

## 背景と構成

`tools/repoctl/release_source.go`はGit検証、`release_verify.go`はprivate previewの処理を担当します。standaloneのproduct/design文書は、変更がない場合もフラグ付きentryを拒否する契約を既に定めています。

## 作業計画

両フラグについて変更あり・なしのfixtureを追加します。修正前の失敗を確認して検査を抽出し、repository checkを実行します。runtime adapterは変更しません。

## 具体的な手順

`go test ./tools/repoctl -run TestReleaseVerifyRejectsHiddenIndexEntries -count=1`、`go run ./tools/repoctl check`、`go test -race ./tools/repoctl`を実行します。修正をcommit/pushし、native VerifyとRelease previewを確認してThreadへ返信・Resolveします。

## 検証と受け入れ

4つの不正fixtureすべてが出力先やbuildログを作る前にindexフラグの診断で拒否され、呼び出し元のファイル内容・index・refを保持し、公開tagを要求しないことを確認します。既存releaseテスト、full harness、native CI、previewと日英docs-checkが成功することを条件とします。

## 冪等性と復旧

fixtureは一時repositoryだけを使います。公開履歴の書き換え、呼び出し元フラグの変更、merge、tag/release公開は行いません。失敗を記録し追加commitで修正します。

## 成果物と注記

2026-09-08にlocalの`go run ./tools/repoctl check`と`go test -race ./tools/repoctl`が成功しました。入口の回帰4ケースも含みます。native CIとpreviewは未確認です。

指摘: https://github.com/mahcialet/agent-env/pull/6#discussion_r3956143329 。証拠は本PlanとThread返信に残します。

## インターフェースと依存

repository harness内部のみの変更です。native GoとGit argvを使い、新たな依存やshell workflowを追加しません。
