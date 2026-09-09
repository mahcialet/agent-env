---
status: accepted
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/adr/0001-use-go.md
source_sha256: 63fc08016c1245b6098e45fb17e5a13aff5c7cc5f903b17298d02d5c5455327f
---

[英語版（翻訳元）](0001-use-go.md)

# Goを使用する

## 背景

ネイティブバイナリを配布し、引数配列で外部ツールを実行するため、リポジトリごとの
シェルによる制御よりも Go が適しています。

## 決定

module は `github.com/mahcialet/agent-env` とし、Go 言語の基準を 1.26.0 に置きます。
CI は対応する 1.26.x/1.27.x を使います。依存ライブラリは CGO なしで動作するものを
選び、自動 toolchain directive は追加しません。

## 帰結

cross-build だけではネイティブ環境の動作を証明できないため、各プラットフォームの
ネイティブ CI が引き続き必要です。この決定にはテストと制限事項の文書化が必要です。
変更時は ADR を用意し、実装と検査を合わせて更新します。
現在の検証対象は[品質ガイド](../QUALITY.ja.md)を参照してください。
