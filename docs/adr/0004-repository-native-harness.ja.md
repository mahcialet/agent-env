---
status: accepted
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/adr/0004-repository-native-harness.md
source_sha256: 20d82ab2524225521144d6e427d50e8b6afec48adac875f1b1d52785262a5f62
---

[英語版（翻訳元）](0004-repository-native-harness.md)

# リポジトリ内のharness

## 背景

巨大な単一の指示ファイルや、チャットだけの進捗記録では、将来のエージェントを確実に
支援できません。リポジトリの知識と作業状況は、見つけやすい場所で維持する必要があります。

## 決定

簡潔な `AGENTS.md`、アーキテクチャの構成図、索引付きの恒常文書、更新を続ける
ExecPlan、Go の repoctl を開発ハーネスとします。ルート指示の上限は厳格に 150 行とします。

metadata、索引、リンク、計画の section、生成物のずれ、依存境界を検査します。
バリデーターには違反を拒否するテストを用意します。

## 帰結

この決定にはテストと制限事項の文書化が必要です。
変更時は ADR を用意し、実装と検査を合わせて更新します。
作業記録は[計画の方針](../PLANS.ja.md)、ハーネスのコマンドは
[品質ガイド](../QUALITY.ja.md)を参照してください。
