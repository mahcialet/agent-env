---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/design-docs/index.md
source_sha256: abc942cd7a72b17390a589811162b59d0d744d632c3024e16c60529d442ad525
---

# 設計文書

[English](index.md)

各文書は、[製品仕様](../product-specs/index.ja.md)を実現する仕組みを説明します。
全体の責務分担は[アーキテクチャ](../../ARCHITECTURE.ja.md)、採用した選択と代案は
[ADR](../adr/index.ja.md)を参照してください。

## Lease のモデルと削除

- [基本原則](core-beliefs.ja.md): 設計上の選択と、守るべき条件の優先順位。
- [Local lease のモデル](lease-control-plane.ja.md): ソース、依存範囲、ライフサイクル、所有情報の保存。
- [Reconciliation と GC](reconciliation-and-gc.ja.md): 診断、削除対象の判定、未完了操作による削除の制御。
- [複数 host の調整](multi-host-control-plane.ja.md): controller の配置管理、worker の権限、操作記録、検証済みデータの転送。

## ランタイムのリソースとアプリ

- [Compose 共通のライフサイクル](compose-runtime.ja.md): サービス選択、呼び出し、readiness、ポート、観測。
- [Docker と Podman の provider](compose-providers.ja.md): 接続先の固定、設定の正規化、所有権に基づく削除。
- [常駐プロセス](persistent-process-runtime.ja.md): OS 上のプロセス識別、状態の保持、停止、endpoint。
- [Android Emulator のリソース](android-emulator.ja.md): 専用 AVD、予約、共有 ADB、保守的な復旧。
- [Flutter Android アプリ](flutter-android-runtime.ja.md): ビルドの順序、インストール、reverse 設定、補償処理。

## 観測と操作

- [Android UI observer](android-ui-observer.ja.md): snapshot の識別、companion による入力、中断した実行の明示的な復旧。
- [Browser/CDP 自動操作](browser-cdp-automation.ja.md): 接続の根拠、古い node の拒否、証拠、時間と量を制限した収集。

## 配布と文書管理

- [スタンドアロン配布](standalone-distribution.ja.md): Git に基づくリリース、決定的なアーカイブ、検査、アセット保存。
- [英日ドキュメント](bilingual-documentation.ja.md): 言語ごとの役割、翻訳レビュー、metadata、例外。
