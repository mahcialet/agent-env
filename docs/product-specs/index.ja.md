---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/product-specs/index.md
source_sha256: c6d2b4f1fb3b86a3b4b2337e90da788d2cf2443a1d6eccc4318ae4c25f0310bf
---

# 製品仕様

[English](index.md)

利用できる機能、受け付ける入力、失敗時の扱いは、ここから各仕様へ進んで確認できます。
初期 MVP の記録と明記した文書を除き、現在の仕様を扱います。実装の仕組みは
[設計の索引](../design-docs/index.ja.md)、検証記録は各仕様が参照する ExecPlan と
[品質方針](../QUALITY.ja.md)を参照してください。

## リースを定義して操作する

| 確認したいこと | 仕様 |
| --- | --- |
| コマンド、出力形式、終了コード | [CLI](cli-contract.ja.md) / [English](cli-contract.md) |
| ソース、コンポーネント、スタックの宣言方法 | [Manifest v1](manifest-v1.ja.md) / [English](manifest-v1.md) |
| 最初の一連の機能に求めた要件 | [初期 MVP](agent-env-mvp.ja.md) / [English](agent-env-mvp.md) |

## ランタイムとアプリケーションを選ぶ

| 使いたい機能 | 仕様 |
| --- | --- |
| Docker または Podman のコンテナー | [Compose プロバイダー](compose-providers.ja.md) / [English](compose-providers.md) |
| 専用の状態とポートを使うネイティブの常駐サービス | [常駐プロセス](persistent-process-runtime.ja.md) / [English](persistent-process-runtime.md) |
| 専用の Android Emulator | [Android Emulator](android-emulator.ja.md) / [English](android-emulator.md) |
| Flutter Android APK のビルド、インストール、起動 | [Flutter アプリケーション](flutter-android-runtime.ja.md) / [English](flutter-android-runtime.md) |

## 画面を観測して操作する

- [Android UI](android-ui-observer.ja.md) / [English](android-ui-observer.md)：所有する Emulator の画面取得、入力、秘密情報の扱い、復旧。
- [Browser/CDP](browser-cdp-automation.ja.md) / [English](browser-cdp-automation.md)：ブラウザーの明示設定、ページ操作、古い参照の拒否、証拠の上限。

## 配布物を使う・複数ホストで運用する

- [スタンドアロン配布](standalone-distribution.ja.md) / [English](standalone-distribution.md)：前提条件、アーカイブ構成、リリース検証。
- [複数ホストのコントロールプレーン](multi-host-control-plane.ja.md) / [English](multi-host-control-plane.md)：リモート構成、配置、認証、結果が不確実な操作の扱い。

延期された機能や非対応の範囲は[ロードマップ](../roadmap.ja.md)を参照してください。
