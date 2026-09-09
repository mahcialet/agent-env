---
status: accepted
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/adr/0003-compose-first-runtime.md
source_sha256: 12b738cfa4bcf6452430dc61375456ecc5ba2d8e34fefe062c91bfa7a2be5670
---

[英語版（翻訳元）](0003-compose-first-runtime.md)

# 最初のruntimeにComposeを採用する

## 背景

初期 MVP では、最初のランタイムが必要でした。ただし、製品全体を薄い Compose wrapper
にする案は採用しません。不変のソース管理と lease の reconcile は、別の責務です。

## 決定

最初のランタイムアダプターには Compose v2 を採用します。明示的で一意なプロジェクトの
識別情報、設定の絶対パス、選択サービスの依存閉包、正規化した設定の digest、
観測したリソースを記録します。

当時は Compose MVP の合格まで、汎用プロセスと Android の実装を延期しました。
これは実装順序についての過去の決定です。現在の実装は
[設計索引](../design-docs/index.ja.md)を参照してください。

## 帰結

Compose の制御は、ソース管理や reconcile の代わりにはなりません。
共通のランタイム機構は [Compose の設計](../design-docs/compose-runtime.ja.md)で説明します。
この決定にはテストと制限事項の文書化が必要です。
変更時は ADR を用意し、実装と検査を合わせて更新します。
