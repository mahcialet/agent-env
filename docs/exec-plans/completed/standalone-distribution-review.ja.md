---
status: completed
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/completed/standalone-distribution-review.md
source_sha256: 4fd27e208f3c2cc74d1332d7c69d97902d5c039ef1d88c04249db7b587b366e1
---

# PR 6のstandalone配布レビューへ対応する

[English](standalone-distribution-review.md)

想定ブランチ: `feat/standalone-distribution`。開始revision: `5aa0b59`。
本PlanはPR 6の残る指摘への修正と返信を担当します。完了済み親Planは履歴の証拠です。

## 目的 / 全体像

未解決11Threadについて修正または既存実装の直接証拠を確認し、返信・Resolveします。

## 進捗

- [x] 2026-09-08: harness、cleanなブランチ、全review Threadを確認。
- [x] 2026-09-08: asset名/サイズ、VCS fallback、releaseパス検査/index flagを同じ問題を検出するテスト付きで修正。
- [x] 2026-09-08: 日英のcompleted statusとstate.db auditを修正。日本語の進捗は英語と一致し、checksums/manifestはarchiveの兄弟、root symlinkと初回並行保存の安全性を確認するテストも成功と確認。
- [x] 2026-09-08: 23f19fdでlocal harness/race、Verify 34206038055、Release preview 34206043365が成功。全11Threadへ返信・Resolve。
- [x] 2026-09-08: 成果を記録し本Planの日英をcompletedへ移動。

## 想定外の発見

独立レビューで最初のtoken境界方式を却下。Goは文字列領域を連結するため、実際の絶対パスの直前が普通の英字になることがあります。確認済みの正確なmodule一致以外では元の漏洩検出を維持し、このパス漏洩を検出するテストを追加します。

修正前は25件の不正asset名を受理、32MiBキャッシュにサイズ拒否なし、実Gitのclean buildでVCS識別情報を欠落、4種類のflagged indexを受理、module末尾がcheckout rootに一致していました。新規テストで再現し、修正後は成功しました。

一部の未解決コメントはrelease実装のmerge前のものです。既存のmkdir stress、
root symlinkテスト、翻訳済み進捗、release-set配置の証拠を再利用します。
archival済み親のmetadataがactiveのまま、registry auditがCLIのstate.dbではなく
registry.sqliteとなっていました。

## 判断の記録

- 2026-09-08 / maintainers: 呼び出し元indexを書き換えずcleanを保証するため、変更がなくてもflag付きindexを拒否。キャッシュはopen前のサイズ確認と開いた後の制限付き読み取りを行い、Windowsの共有再試行を維持。linker値がunknownの場合だけ埋め込みVCSを使い、明示release識別情報を優先。全OSでWindows名制約を使い、パス漏洩検査は確認済みmoduleパスだけを除外します。

- 2026-09-08 / maintainers: 許可済みPRブランチで本active review Planに従って
  修正します。親の過去の受け入れ証拠を再開・上書きせず、古い指摘は現在の実装と
  照合してから返信します。

## 成果と振り返り

コードrevision `23f19fd` で完了。全11指摘へ返信しThreadをResolveしました。4件は既存修正、7件は新規の実装または文書修正です。同じ問題を検出するテスト、full harness/race、Windows/macOS/Linux native、release previewが成功。独立レビューで文字列領域のパス検査の退行を検出し、公開前に修正しました。公開tag/releaseと履歴の書き換えは行っていません。

## 背景と構成

`internal/assets` は汎用バイト列の保存、`internal/buildinfo` は識別情報、
`tools/repoctl/release*.go` は厳密なrelease生成・検証を担当します。
standaloneのproduct/design文書と完了済み親が契約を記述しています。

## 作業計画

移植可能な名前、過大なキャッシュ、埋め込みVCS fallback、パス漏洩検査の誤検知、
隠れたtracked変更を拒否するテストを実装します。metadataとstate.db auditを日英で
修正し、既存のThread修正を直接検証します。

## 具体的な手順

対応GoをPATHに置き `go run ./tools/repoctl check`、`go test -race ./...`、
対象packageテスト、candidate release検証を実行します。まとまった修正を
commit/pushし、native Verify/Release preview、証拠付き返信、Resolveへ進みます。

## 検証と受け入れ

各未解決Threadに実装または文書の証拠と返信があります。可能な範囲で新しく追加した不具合検出テストの
修正前失敗・修正後成功を確認し、既存テストを弱めません。コードrevisionの
Windows/macOS/Linux nativeとrelease preview成功、日英docs-check成功、
最終push後のclean treeを確認します。

## 冪等性と復旧

公開履歴を書き換えず、公開tag/releaseを変更しません。index flagは専用の一時Git
fixtureだけで操作し、ユーザーのrepositoryのflagを変更しません。
CIでOS固有の問題が出たら失敗を記録して修正を積みます。

## 成果物と注記

2026-09-08のコードrevision `23f19fd` でlocal `repoctl check`、全体の
`go test -race ./...`、最終release packageのraceが成功。
`release-verify --out dist/pr6-review-candidate` は6ターゲット各2回のbuildで
8ファイルのバイト一致とLinux native smokeに成功。
`AGENT_ENV_RELEASE_CANDIDATE=../../dist/pr6-review-candidate go test ./tools/repoctl -run TestReleaseCandidate -count=1 -v`
は全19ケース成功。実moduleパスの誤検知回避と実際のパス漏洩の拒否も含みます。
asset/buildinfoと最終releaseの独立レビューに確認済み残存不具合なし。
文書編集中のharnessは翻訳hash未更新だけで失敗し、実際の翻訳とhashを同期して
full harness再実行が成功。既存修正4Threadは証拠を確認して返信・Resolve済みで、
新規修正7件もnative CI成功後に返信・Resolve済み。Verify: https://github.com/mahcialet/agent-env/actions/runs/34206038055 。Release preview: https://github.com/mahcialet/agent-env/actions/runs/34206043365 。

PR: https://github.com/mahcialet/agent-env/pull/6 。証拠はここ及びThread返信に
記録し、追加のreportファイルを作りません。

## インターフェースと依存

既存package境界とnative Go workflowを維持し、実行時ツールを追加しません。
