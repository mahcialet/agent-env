---
status: accepted
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/adr/0001-use-go.md
source_sha256: 7e0bd50093ae31e6c17f5ea1037627f0f505bc56a233107ac3a03aa9974e486a
---

[English（正本）](0001-use-go.md)

# Goを使用する

## 決定

採用：module github.com/mahcialet/agent-env、Go言語の基準1.26.0、対応する1.26.x/1.27.xのCIを使用します。ネイティブbinaryとargv実行の点から、リポジトリごとのshellオーケストレーションよりGoを選びます。CGo不要の依存を使用し、自動toolchain directiveは使いません。cross-buildに加えてネイティブプラットフォームCIが引き続き必要です。

## 帰結

この決定にはテストと制限事項の文書化が必要です。変更にはADRと、実装・検査の同期が必要です。
