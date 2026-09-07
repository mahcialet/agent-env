---
status: accepted
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/adr/0004-repository-native-harness.md
source_sha256: 2446df343a7f8eed77452400166f1f15ce0a6e57d47e59cd720e0637d2f22e28
---

[英語版（翻訳元）](0004-repository-native-harness.md)

# リポジトリ内のharness

## 決定

採用：簡潔なAGENTS.md、アーキテクチャ構成図、索引付きの恒常文書、更新を続けるExecPlan、Goのrepoctlを開発harnessとします。巨大な単一指示ファイルやチャットだけの進捗記録では、将来のエージェントを確実に支援できません。metadata、索引、link、計画のsection、生成物のずれ、依存境界を、違反時に失敗することをテストしたバリデーターで強制します。ルート指示の厳格な上限は150行です。

## 帰結

この決定にはテストと制限事項の文書化が必要です。変更にはADRと、実装・検査の同期が必要です。
