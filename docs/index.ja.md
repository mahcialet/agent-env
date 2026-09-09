---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/index.md
source_sha256: 6f3686a6b77234e5abb61c47eb9129952133f52e839af3409e24c7a4423120ab
---

# リポジトリの知識

[英語版（翻訳元）](index.md)

[アーキテクチャ](../ARCHITECTURE.ja.md)、[MVP仕様](product-specs/agent-env-mvp.ja.md)、[完了済みExecPlan](exec-plans/completed/agent-env-mvp.md)から読む。

提供済みの拡張：[Android Emulatorの契約](product-specs/android-emulator.ja.md)と、その[完了済みExecPlan](exec-plans/completed/android-emulator-lease.md)。

提供済みの拡張：[Flutter Androidの契約](product-specs/flutter-android-runtime.ja.md)と、その[完了済みExecPlan](exec-plans/completed/flutter-android-runtime.ja.md) / [English](exec-plans/completed/flutter-android-runtime.md)。

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

Browser/CDP: [製品契約](product-specs/browser-cdp-automation.ja.md)、[設計](design-docs/browser-cdp-automation.ja.md)、[完了の検証記録](exec-plans/completed/browser-cdp-automation.ja.md)。

複数 host の調整は、文書化した範囲で受け入れ検証が完了しています。[製品仕様](product-specs/multi-host-control-plane.ja.md)、
[設計](design-docs/multi-host-control-plane.ja.md)、[ADR 0006](adr/0006-single-authority-multi-host.ja.md)、
[ExecPlan](exec-plans/completed/multi-host-control-plane.ja.md) を参照してください。
[README の使用方法](../README.ja.md#明示的な-remote-モード) に登録と各 role の起動方法を記載しています。
[品質](QUALITY.ja.md#複数-host-の-native-検証) では、Windows・macOS・Linux の native TLS 実行の成功を記録し、
未実施の物理 host 検証と区別します。
