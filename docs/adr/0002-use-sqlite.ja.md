---
status: accepted
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/adr/0002-use-sqlite.md
source_sha256: 5b319b9c7c5a8361da82e8b5eb94ae7423a42e2c20d44e3b94221a743e7c1c97
---

[英語版（翻訳元）](0002-use-sqlite.md)

# SQLiteを使用する

## 背景

lease の状態には、正規化したソース、リソース、コマンド実行、成果物、イベントの
関連が含まれます。SQLite はこれらを永続化します。ただし、外部への操作には補償処理が
必要であり、システムをまたぐトランザクションであるかのようには扱えません。

## 決定

`database/sql` と `modernc.org/sqlite` を使い、番号付きの SQL migration を明示的に
埋め込みます。SQLite が lease と上記の関連記録を管理します。外部キー、5000 ms 以上の
busy timeout、ローカルファイルシステム上で検証した WAL を有効にします。
JSON だけのレジストリと ORM は採用しません。

## 帰結

DB を commit しても、Git やランタイムへの操作を同じトランザクションにはできません。
saga と復旧のために保持する証拠は [lease の設計](../design-docs/lease-control-plane.ja.md)
で説明します。この決定にはテストと制限事項の文書化が必要です。
変更時は ADR を用意し、実装と検査を合わせて更新します。
