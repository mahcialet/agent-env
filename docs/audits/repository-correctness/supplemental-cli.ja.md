---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/supplemental-cli.md
source_sha256: 6f408ad38533d2cb4c5416e6170f08df7615980d4c181b33657fedb29c26d44d
---

# Browser CLI結果の追加監査

[English](supplemental-cli.md) · [監査一覧](index.ja.md)

## AUDIT-CLI-001: 変更結果のtableが作用前の一覧を表示する

重要度Medium、採否ACCEPT。最終履歴照合で発見した [PR10指摘3963154182](https://github.com/mahcialet/agent-env/pull/10#discussion_r3963154182) が対象。固定031869c8の `internal/cli/browser.go` は全操作で `Observation.Pages` を表示する。adapterは作成・削除前の一覧を保存し、変更対象は別の `Observation.Page` に返す。

固定版のnative入口（Linux Chrome、sandbox有効）に一時テストoverlayで `TestBrowserNativeCLI` の検証を追加した。既定tableでpageを作り、独立したJSON一覧から新IDを取得し、tableで削除後、元の兄弟pageが残ることを確認した。tableの2検証は失敗し、作成では新IDがなく、削除では消したIDが一覧行として表示された。元のJSON検証は成功した。利用者向け出力の不具合であり、別対象へのlifecycle作用を実証したものではない。

元の検出はS8（merge後に投稿された独立PRレビュー）、監査再現はS9。最早防止段階S3（CLI結果表示のpackage integration）。作用とtable表示を組み合わせた検証が不足していた。分類COMPOSITION_GAP/NEGATIVE_FIXTURE_GAP/ORACLE_COUPLING: adapterから表示への結果の不一致、table出力で必要な表示があり、誤った表示がないことを確認するテストの不足、JSONだけに基づく期待値。コマンドやschemaの変更は不要。

## 修正と再発防止

page-create/page-closeの成功tableでは `Observation.Page` を `Page created`/`Page closed` と明示する。失敗時は既存診断経路を維持し、変更成功と表示しない。読取り用page一覧とJSON契約は維持する。恒久native CLIテストは実行ファイルを経由し、独立JSON一覧とのID一致・兄弟保持・実削除を確認する。追加した恒久テストも修正前に失敗した。初回編集では存在しないobservation statusを参照してcompile失敗し、実際のoperation errorで成功を判定するよう直した。最終native race証拠は一覧に記録する。
