---
status: accepted
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/adr/0003-compose-first-runtime.md
source_sha256: 9ae3652e1b214d66a92c55d38a196d8d9edbc97af4e8f172b0b2312d41987163
---

[English（正本）](0003-compose-first-runtime.md)

# 最初のruntimeにComposeを採用する

## 決定

採用：最初のruntimeアダプターはCompose v2とし、明示的で一意なプロジェクト識別情報、絶対設定パス、選択サービスの依存閉包、正規化設定digest、観測リソースを備えます。製品全体を薄いCompose wrapperにする案は採用しません。不変のソースとリースのreconcileは別の責務です。汎用プロセスとAndroidは、Compose MVPが合格するまで延期します。

## 帰結

この決定にはテストと制限事項の文書化が必要です。変更にはADRと、実装・検査の同期が必要です。
