---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/adr/index.md
source_sha256: 7ee6d05113240aa889907f3f1c480538a3785097edbe3cb31022af1a3941f80c
---

# アーキテクチャの判断記録

[English](index.md)

ADR は採用した判断、その理由、代案、影響を記録します。過去の実装順序は
現在の機能の有無を示すものではありません。現在の仕組みは
[設計文書](../design-docs/index.ja.md)を参照してください。

## 基盤の判断

- [0001 — Go を使用する](0001-use-go.ja.md): ネイティブバイナリ、言語の基準、CGO を使わない実行。
- [0002 — SQLite を使用する](0002-use-sqlite.ja.md): ローカル状態の永続化、migration、外部操作の補償。
- [0003 — 最初の runtime に Compose を採用する](0003-compose-first-runtime.ja.md): MVP 当時の実装順序と、ソース管理・reconcile からの分離。
- [0004 — リポジトリ内のハーネス](0004-repository-native-harness.ja.md): 案内に徹する指示、更新を続ける計画、違反の拒否をテストした検査。

## アプリと host の責務境界

- [0005 — Flutter アプリを分離する](0005-separate-flutter-applications.ja.md): ビルドの独立性と Android リソースの所有権。
- [0006 — Controller の管理主体を一つにする](0006-single-authority-multi-host.ja.md): lease 全体の割り当て、不確実な結果の保持、worker ごとの削除管理。
- [0007 — Native 実行の範囲を定める](0007-native-execution-boundaries.ja.md): Windows のパス制限と、Windows/WSL のプロセス所有権の分離。
