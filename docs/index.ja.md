---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/index.md
source_sha256: 2409394402983fa141748518bd4d4fbf7f1b743b0e13aebcee7d91f49204c7d6
---

# リポジトリの知識

[英語版（翻訳元）](index.md)

[アーキテクチャ](../ARCHITECTURE.ja.md)、[MVP仕様](product-specs/agent-env-mvp.ja.md)、[完了済みExecPlan](exec-plans/completed/agent-env-mvp.md)から読む。

提供済みの拡張：[Android Emulatorの契約](product-specs/android-emulator.ja.md)と、その[完了済みExecPlan](exec-plans/completed/android-emulator-lease.md)。

提供済みの拡張：[Flutter Androidの契約](product-specs/flutter-android-runtime.ja.md)と、その[再開したExecPlan](exec-plans/active/flutter-android-runtime.ja.md) / [English](exec-plans/active/flutter-android-runtime.md)。

- [製品仕様](product-specs/index.ja.md)：ユーザーに見える契約。
- [設計文書](design-docs/index.ja.md)：仕組みと責務境界。
- [ADR](adr/index.ja.md)：採用した代案と結果。
- [計画の規則](PLANS.ja.md)：更新を続ける作業記録と完了の証拠。
- [品質](QUALITY.ja.md)：コマンドと検証範囲。
- [信頼性](RELIABILITY.ja.md)：失敗と慎重な復旧。
- [セキュリティ](SECURITY.ja.md)：信頼とホスト方針。
- [移植性](PORTABILITY.ja.md)：ネイティブプラットフォームの要件。
- [ロードマップ](roadmap.ja.md)：延期した機能と未解決の判断。
- [参考資料](references/index.ja.md)：歴史的な出所。

設計・製品・ADR・計画文書にはstatus、owner、last_verifiedメタデータを付ける。永続文書はローカル索引から見つけられるようにする。アーカイブ資料は参考資料索引に従い、生成文書は、生成処理の実装後にmigrationから作る。鮮度の日付は文書を見直した日であり、計画した機能の実装を証明するものではない。

- [生成されたデータベースschema](generated/db-schema.md)：埋め込まれたmigrationから機械的に導出する。

- [言語の方針](design-docs/bilingual-documentation.ja.md)：英語版を内容の基準にする方針、日本語訳の保守、明示的な例外。
