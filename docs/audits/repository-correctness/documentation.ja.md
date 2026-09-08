---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/documentation.md
source_sha256: 92dcd78c07cf89c16ce9af68423d8359ce77cd9ba6d08a1871c6b0e7f8f38438
---

# 文書検査の過去指摘再検証と現行監査

[English](documentation.md)

監査対象は `031869c8b9073b8e23bc17fbc55243666a52f557`。Phase A の記録であり、製品コードは変更していない。
資料は `docs/exec-plans/completed/` の `bilingual-documentation.md` と `bilingual-documentation-review.md`。PR #3 の5件の Thread ID は後者に記録されている。

## 過去指摘の対応表

以下はすべて現在も適用される。検出段階は S8、最も早く現実的に防げた段階は S2。言語方針には既に要件が明記されていた。現行実装は `tools/repoctl/main.go` と `tools/repoctl/translations.go`。特記しないテストは `tools/repoctl/translation_review_test.go` にあり、補助関数だけでなく `docsCheck` 全体を通る。対象回帰テストは race 検査付きで成功した（2.296秒）。

| ID / 元の不具合 | 現行の回帰テスト | 以前の検出機会と見逃し | 防止策 / 再発 |
| --- | --- | --- | --- |
| HIST-DOC-01 / 英語版への表示リンクがなくても翻訳メタデータだけで通過 | `TestTranslationRequiresVisibleSourceLink` | S2 にリンク欠落の負例がなく、正常系 fixture も同じ欠落を含んだ。ORACLE_COUPLING、NEGATIVE_FIXTURE_GAP。S6 は不正文書を受け入れ、S7 は要件別に照合しなかった。 | 非表示・空・別文書・エスケープのリンクを個別変更する既存テスト。フラグメントの再発は下記。 |
| HIST-DOC-02 / ルートの日本語文書で基本メタデータ欠落を許可 | `TestRootTranslationsRequireCoreMetadata` | S2 の生成 fixture がメタデータを欠き、受け入れを期待した。ORACLE_COUPLING。S6 と S7 もルートと docs の適用範囲差を見逃した。 | フィールドごとの欠落・空・不正値。現行の翻訳範囲はルートと docs 全体を含む。 |
| HIST-DOC-03 / 新しい完了 Plan を翻訳例外に追加可能 | `TestCompletedPlanExceptionCannotExpandMigrationSet` | 移行時点という境界があるのに、S2 は新規例外の許可を期待した。ORACLE_COUPLING、NEGATIVE_FIXTURE_GAP。S6 はレジストリを信頼し、S7 は将来追加を見なかった。 | 固定4件の許可リストと新規 Plan の負例が存在する。別の例外判定元は確認されなかった。 |
| HIST-DOC-04 / 日本語 Plan の必須節欠落や任意見出しを許可 | `TestRequiredPlanSectionsMustBeProse`、`structure_review_test.go` の `TestReviewJapanesePlanRequiresEverySection` | S2 は各必須節を削除せず、任意の日本語見出しを許可した。ORACLE_COUPLING、NEGATIVE_FIXTURE_GAP。S6 は日本語構造を免除し、S7 は正常な文書だけに依存した。 | 節ごとの削除、英語見出し許可、本文抽出の既存テスト。別実装のフラグメント検査で AUDIT-DOCS-001 が再発。 |
| HIST-DOC-05 / 英日両索引から翻訳へのリンクを保証しない | `TestIndexRequiresNavigableJapaneseLinks`、`structure_review_test.go` の `TestReviewJapaneseDocumentsRequireBothLocalIndexes` | S2 の負例は重複リンクの片方だけを消したため、要件を満たしたままだった。ORACLE_COUPLING。S6、S7 は誤解を招く成功結果に依存した。 | 全体呼出しでの独立削除と非表示・参照リンクの負例。索引欠落の現行再発は未確認。 |
| HIST-DOC-06 / 文字列抽出がコード例・コメント・重複参照定義を受け入れた | `TestTranslationRequiresVisibleSourceLink`、`TestRequiredPlanSectionsMustBeProse` | S2 で小さな反例により表示内容との差を検証できた。NEGATIVE_FIXTURE_GAP、COMPOSITION_GAP。S6 の共有解析を全利用箇所へ適用せず、S7 も他の利用箇所を調べなかった。 | 本文・コメント・コード状態の解析と最初の参照定義を採用する既存実装。見出しフラグメントでは AUDIT-DOCS-001。 |
| HIST-DOC-07 / コメント除去順だけの修正でコード内コメントやコメント内フェンスを誤認 | `TestTranslationRequiresVisibleSourceLink` のコード・コメントを交互に置く正常例 | S2 は負例に加えて入れ子構文の正常例を必要とした。COMPOSITION_GAP。S7 は各除去処理を見て、合成した解析状態を見なかった。 | 状態を管理する既存解析と正常例は現在も成功。追加の順序不具合は未確認。 |

全体呼出しのテストは不正入力を拒否し、意味のあるエラーコードを検証している。今回は現行テストの実行と経路確認を行ったもので、過去の全実装を mutation test したわけではない。移行時の曖昧な説明や完了 Plan の `status: active` は歴史的事実として扱い、アーカイブを無断で書き換えない。

## AUDIT-DOCS-001

- 重要度: Low。採否: Phase B 待ち。
- 不変条件: 有効と判定した Markdown フラグメントは、表示される見出しを指す。
- 場所: `tools/repoctl/main.go` の `documentStructureCheck` 内フラグメント見出し走査。
- 条件: fixture 文書のコードフェンス内に `## Phantom` を追加し、別文書から `#phantom` へリンクする。翻訳を同期し、別の翻訳エラーで結果を隠さない。
- 観測: 表示される見出しが存在しなくても `docsCheck` 全体が nil を返す。
- 期待: DOC-004 でアンカー欠落を拒否する。
- 影響: 壊れた文書内リンクが検査を通る。ランタイムへの影響はない。
- 既存検証: 非表示リンクと必須見出しの負例はあるが、リンク先見出しは生の行を別途走査している。
- 再現: 一時 Go overlay の `TestAuditHiddenFragment` から `fixture`、`pairFixtureDocument`、`docsCheck` を使用。固定した製品コードに対し `go test -overlay <temporary-overlay> ./tools/repoctl -run TestAuditHiddenFragment -count=1` が失敗（0.004秒）。理由は「accepted link to fenced heading with no rendered anchor」。製品ファイルは変更していない。
- 原因仮説: 過去の表示本文に基づく修正がナビゲーションと必須節だけに適用され、リンク先フラグメントには適用されなかった。
- 修正案: リンク先見出しも同じ本文解釈で抽出し、重複アンカーの動作を保つ。コード・コメントの偽見出しと有効な重複見出しを検証する。
- 回帰テスト・解決・最終検証: 採否と Phase C 待ち。
- 関連: HIST-DOC-04、HIST-DOC-06。
- 見逃し分析: 検出 S9、最も早い防止段階 S2。既存の非表示見出し fixture を別の全体呼出し経路にも適用できた。S6 は生の行のままで、S7/S8 は参照元リンクと必須節に限定した。COMPOSITION_GAP、NEGATIVE_FIXTURE_GAP、REVIEW_CHECKLIST_GAP。
- 再発防止案: 表示見出しの共通抽出と全体呼出しのフラグメント負例。今後の検出は S2/S6 を想定。Phase A では未実装。
