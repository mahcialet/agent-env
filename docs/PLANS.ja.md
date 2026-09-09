---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/PLANS.md
source_sha256: 29a8ed1aa863876d55af7561f640a5f3f3aba30b33c7b8c21ae72061684f8e30
---

# ExecPlanの規則

[英語版（翻訳元）](PLANS.md)

ExecPlanには、実質的な作業を実行・レビュー・復旧するための情報をまとめます。
作業中の計画には現在の指示と受け入れの証拠を記録し、完了した計画は実装の履歴として残します。

## 実質的な作業を始める

実質的な変更には、専用ブランチと`docs/exec-plans/active/`配下でバージョン管理する
ExecPlanが必要です。想定ブランチと現在の作業はactive ExecPlanの記述に従い、両方を計画に明記します。

計画は単体で理解できる内容にします。会話をたどり直さなくても、実装担当者が正確なパス、
コマンド、結果、残る作業、判断、安全な復旧手順を確認できるようにしてください。
[完了済みMVP計画](exec-plans/completed/agent-env-mvp.md)は、過去の作業範囲と証拠の記録例です。

## 必須の構成

すべてのactive planに、次の節をMarkdownのレベル2見出しで記載します。
英語版には英語の見出しを使います。日本語版には対応する日本語名または英語名を使えます。
任意の名前では構造検査を通過しません。

| 英語の見出し | 日本語の見出し |
| --- | --- |
| Purpose / Big Picture | 目的 / 全体像 |
| Progress | 進捗 |
| Surprises & Discoveries | 想定外の発見 |
| Decision Log | 判断の記録 |
| Outcomes & Retrospective | 成果と振り返り |
| Context and Orientation | 背景と構成 |
| Plan of Work | 作業計画 |
| Concrete Steps | 具体的な手順 |
| Validation and Acceptance | 検証と受け入れ |
| Idempotence and Recovery | 冪等性と復旧 |
| Artifacts and Notes | 成果物と注記 |
| Interfaces and Dependencies | インターフェースと依存 |

## 実行中に証拠を更新する

永続的な計画のメタデータには、`status`、`owner`、`last_verified`を必須とします。
意味のある区切りごとに、次の内容を更新します。

- 進捗には日付付きの完了・未完了項目を使います。チェックは確認済みの完了を表し、予定には付けません。
- 正確なコマンドと結果を記録します。ローカルテスト、クロスビルド、実際のネイティブCIを区別し、
  失敗した検証と未解決のプラットフォーム上の不足も残します。
- 判断には日付、担当者の役割、根拠を記します。継続して適用する決定はADRへ反映します。
- 残る作業と安全な復旧手順を現状に合わせます。

## 完了して履歴へ移す

すべての受け入れ要件に直接の証拠があり、「成果と振り返り」を記入してから、計画をactiveから
completedへ移します。移動時はすべてのリンクを更新します。完了済み計画は削除せず履歴として
保持してください。過去の参考資料アーカイブは入力資料であり、現在の運用上の判断基準にはしません。

人が読む永続文書を追加・変更する実質的な作業では、ExecPlanを完了する前に対応する日本語訳を
含めなければなりません。更新を続けるExecPlan自身も対象です。両言語の計画を同じ
active/completedディレクトリに置き、移動時は両方のリンクを更新します。
例外は、移行前の歴史的計画として明示的に登録されたものに限ります。

[言語の方針](design-docs/bilingual-documentation.ja.md)に従い、実質的な構成変更では
英語、日本語、意味の一致をそれぞれ別にレビューします。完了前に`repoctl docs-check`を実行します。
更新確認用hashの一致だけで、翻訳の意味を確認したことにはできません。
