---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/index.md
source_sha256: 67b823fcb2177e31b25536d8d24be793f70a1b702d9365907361b8ce6c0f7fbc
---

# 目的から文書を探す

[English](index.md)

知りたいことに合わせて、次の文書を選んでください。[README](../README.ja.md)では製品の役割と
最初のlease操作を説明します。現在の挙動は製品仕様、特定revisionでの検証結果は完了済みPlanや
監査記録で確認できます。

## 機能を使う

現行の契約は[製品仕様index](product-specs/index.ja.md)にまとめています。リポジトリの設定には
[manifest仕様](product-specs/manifest-v1.ja.md)、操作には[CLI仕様](product-specs/cli-contract.ja.md)を使います。

| 知りたいこと | 次に読む文書 |
| --- | --- |
| container engineを選びたい | [Compose provider](product-specs/compose-providers.ja.md) |
| native serverを継続稼働させたい | [常駐process lease](product-specs/persistent-process-runtime.ja.md) |
| EmulatorやFlutterアプリを動かしたい | [Android Emulator](product-specs/android-emulator.ja.md)、続いて[Flutterアプリ](product-specs/flutter-android-runtime.ja.md) |
| UIを観測・操作したい | [Android UI](product-specs/android-ui-observer.ja.md)または[Browser/CDP](product-specs/browser-cdp-automation.ja.md) |
| 実行ファイルを導入したい／releaseを作りたい | [standalone配布](product-specs/standalone-distribution.ja.md) |
| hostを登録し、remoteで実行したい | [multi-hostの準備と操作](product-specs/multi-host-control-plane.ja.md) |
| 当初のMVPの範囲を知りたい | [MVP仕様](product-specs/agent-env-mvp.ja.md) |

## リポジトリを変更する

まず[AGENTS.md](../AGENTS.ja.md)で作業手順を、[Architecture](../ARCHITECTURE.ja.md)で依存境界を
確認してください。[Planの方針](PLANS.ja.md)は、大きな変更をactive ExecPlanで進める方法と、
完了してarchiveできる条件を定めます。

| 判断したいこと | 方針・仕組みの説明 |
| --- | --- |
| どの検証で受け入れを判断するか | [品質](QUALITY.ja.md) |
| 失敗やcleanup中断からどう復旧するか | [信頼性](RELIABILITY.ja.md) |
| どのリポジトリと作用を信頼するか | [セキュリティ](SECURITY.ja.md) |
| 対応OSの前提条件は何か | [移植性](PORTABILITY.ja.md) |
| 日英の文書をどう維持するか | [言語方針](design-docs/bilingual-documentation.ja.md) |
| なぜその設計にしたか | [設計index](design-docs/index.ja.md)と[採用済みADR](adr/index.ja.md) |
| DB構造はどこで定義するか | migrationから作る[生成schema](generated/db-schema.md) |

[roadmap](roadmap.ja.md)は、実装済みの機能、現在の作業、延期した判断を区別します。
提案中の機能が利用可能なコマンドであるとは扱いません。

## 履歴と現在の規則を区別して証拠を探す

検証コマンド、検証範囲、native受け入れ証拠の入口は[品質方針](QUALITY.ja.md)です。
各機能の仕様と設計から、対応する完了済みPlanへ進めます。たとえば、次の記録があります。

- [MVP](exec-plans/completed/agent-env-mvp.md)、[Android](exec-plans/completed/android-emulator-lease.md)、
  [Flutter](exec-plans/completed/flutter-android-runtime.ja.md)のPlanは、local環境の基盤を実装した記録です。
- [Browser](exec-plans/completed/browser-cdp-automation.ja.md)と
  [multi-host](exec-plans/completed/multi-host-control-plane.ja.md)のPlanは、追加機能の受け入れを記録します。
  [multi-hostの品質証拠](QUALITY.ja.md)では、同一runner上のTLS検証と未検証の物理host環境を区別します。
- [正しさ監査のindex](audits/repository-correctness/index.ja.md)は、固定したbaseline、修正候補、指摘を
  分けて案内します。[履歴資料](references/index.ja.md)では、古い資料の位置付けとarchive例外を説明します。

設計、仕様、ADR、Planには`status`、`owner`、`last_verified`を付け、各indexから辿れるようにします。
検証日があるだけで、計画中の機能を実装済みと判断してはいけません。生成文書と過去の参考archiveは、
言語方針と履歴資料indexに明記した例外に従います。
