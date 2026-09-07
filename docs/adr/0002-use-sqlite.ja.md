---
status: accepted
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/adr/0002-use-sqlite.md
source_sha256: 8d6f32fd8509f82fceb28d0020b5ceaf6481c920b5a232fb33cc332df7bc66a4
---

[English（正本）](0002-use-sqlite.md)

# SQLiteを使用する

## 決定

採用：database/sqlとmodernc.org/sqlite、および明示的に埋め込む連番SQL migrationを使用します。SQLiteはリース、正規化したソースtuple、リソース、command run、成果物、eventを管理します。外部キー、5000 ms以上のbusy timeout、ローカルファイルシステム上で検証済みのWALを有効にします。JSONだけのレジストリとORMは採用しません。外部作用には補償が必要であり、架空のシステム横断transactionで扱いません。

## 帰結

この決定にはテストと制限事項の文書化が必要です。変更にはADRと、実装・検査の同期が必要です。
