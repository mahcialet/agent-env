---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/PLANS.md
source_sha256: 0e28d9dbaa4aa8ff7f68d1230ce643114bd1b66c796182dfdd45dbb20c7476bb
---

# ExecPlanの規則

[正本の英語版](PLANS.md)

実質的な変更には専用ブランチと、`docs/exec-plans/active/`配下でバージョン管理するactive ExecPlanを使う。想定ブランチと現在の作業はactive ExecPlanを正とし、その両方を計画内に明記する。計画は単体で理解できる内容に保ち、意味のある区切りごとに更新する。正確なパス、コマンド、結果、残る作業、判断、安全な復旧手順を含める。証拠では、ローカルテスト、クロスビルド、実際のネイティブCIを区別する。[完了済みMVP計画](exec-plans/completed/agent-env-mvp.md)には過去の範囲と証拠を残す。

正本の必須セクションは、Purpose / Big Picture、Progress、Surprises & Discoveries、Decision Log、Outcomes & Retrospective、Context and Orientation、Plan of Work、Concrete Steps、Validation and Acceptance、Idempotence and Recovery、Artifacts and Notes、Interfaces and Dependenciesである。

永続的な計画のメタデータにはstatus、owner、last_verifiedを含める。Progressには日付付きの完了・未完了項目を使う。チェックは観測した完了を意味し、予定を意味しない。失敗した検証と未解決のプラットフォーム差を残す。判断には日付、担当者の役割、根拠を記し、持続的な決定はADRへ反映する。

すべての受け入れ要件に直接の証拠があり、成果と振り返りを記入してから、計画をactiveからcompletedへ移す。移動時はすべてのリンクを更新する。完了済み計画は履歴として保持し、削除しない。過去の参考資料アーカイブは入力であって、運用上の正本ではない。

人が読む永続文書を追加・変更する実質的な作業では、ExecPlanを完了する前に対応する日本語訳を含めなければならない。これは更新を続けるExecPlan自身も含む。両言語の計画を同じactive/completedディレクトリに置き、移動時は両方のリンクを更新する。例外は、移行前の歴史的計画として明示的に登録されたものに限る。[言語の方針](design-docs/bilingual-documentation.ja.md)に従い、`repoctl docs-check`で検証する。
