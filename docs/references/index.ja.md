---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/references/index.md
source_sha256: 40b07bcdf6642c6a3ffa2e073da6db88c2cf0cf097804f98a8c8ba9bef7b8243
---

# 初期資料と外部参照

[English](index.md)

リポジトリの初期入力を辿るときや、利用ツールの公式資料を読むときの入口です。
現在の操作は[文書index](../index.ja.md)とactive ExecPlanに従ってください。
archive内の提案は、現行の製品仕様や設計に優先しません。

## 保存した初期handoff

[初期handoffのarchive](handoffs/CHATGPT_HANDOFF_30.md)は、2026-09-07に
`.codex/CHATGPT_HANDOFF_30.md`として提供されたものです。SHA-256は
`3f72a24b1767009cd66a0664ee62ba3359d9518599fb27d3d80fa26c7bfa864a`です。
後の実装証拠によって置き換わった提案も含め、当時の入力を保持しています。
資料内のlicense未決定事項は、維持している既存の[MIT license](../../LICENSE)で解消されています。

archiveしたhandoffは、durable文書のmetadataとlocal index検査の対象外です。
このページから辿れる状態を保ちます。翻訳例外は[言語方針](../design-docs/bilingual-documentation.ja.md)
を参照してください。archive自体のバイト列は編集しません。

## 実装で参照する公式資料

- [Go support policy](https://go.dev/doc/devel/release): サポートするreleaseの範囲。
- [Git worktrees](https://git-scm.com/docs/git-worktree): worktreeの仕組み。
- [Compose CLI](https://docs.docker.com/reference/cli/docker/compose/): Docker Composeコマンド。
- [SQLite driver](https://pkg.go.dev/modernc.org/sqlite): Goの永続化driver。
