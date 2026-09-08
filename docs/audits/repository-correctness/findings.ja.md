---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/findings.md
source_sha256: 71811a2327b1a738f742658c47dfb4647fddd0aaa13a475cf1b21c484c4eadda
---

# 指摘の採否と修正記録

[English](findings.md) · [監査索引](index.ja.md)

2026-09-09、全担当の Phase A レポートと過去182行の記録が揃った後、Phase B の区切りを記録し、下記採否を確定した。参照先には固定した修正前の観測と再現を残す。この記録で最終採否と解決状況を管理する。

## 採否

再現した15件をすべて ACCEPT とする。既存の安全性・identity・上限・文書リンクに関する範囲内の契約違反または再発であり、局所的に修正できる。High7件は必須で、黙って先送りしない。低い重要度の症状は分類を共有しても実装経路と検証条件が異なるため重複扱いにしない。確認済み指摘の REJECT/DEFER/DUPLICATE はない。

| ID | 重要度 | 採否 | 理由と証拠 | 解決 |
| --- | --- | --- | --- | --- |
| AUDIT-CLEANUP-001 | High | ACCEPT | [readiness の終了未確認後に再試行・source削除を許可](current-control-plane.ja.md) | 回帰追加と修正待ち。 |
| AUDIT-OWNERSHIP-001 | High | ACCEPT | [container identity 欠落で破壊的 Down を許可](current-compose-release.ja.md) | 回帰追加と修正待ち。 |
| AUDIT-DOCS-001 | Low | ACCEPT | [非表示見出しでフラグメント検証を通過](documentation.ja.md) | 回帰追加と修正待ち。 |
| AUDIT-RELEASE-001 | Low | ACCEPT | [短い source root の漏洩検出を回避](current-compose-release.ja.md) | 回帰追加と修正待ち。 |
| AUDIT-RELEASE-002 | Medium | ACCEPT | [事前検査だけの読込上限をファイル成長で超過](current-compose-release.ja.md) | 回帰追加と修正待ち。 |
| AUDIT-BOUNDARY-001 | Medium | ACCEPT | [ログ2000行ちょうどで省略と誤判定](history-mobile.ja.md) | 回帰追加と修正待ち。 |
| AUDIT-REDACTION-001 | Medium | ACCEPT | [mobile 証拠が秘匿後の上限を超過](current-mobile.ja.md) | 回帰追加と修正待ち。 |
| AUDIT-REDACTION-002 | Medium | ACCEPT | [DOM 公開時の秘匿後に adapter 上限を超過](current-process-browser.ja.md) | 回帰追加と修正待ち。 |
| AUDIT-STALE-001 | Medium | ACCEPT | [load wait が旧 snapshot と新文書の判定を混合](current-process-browser.ja.md) | 回帰追加と修正待ち。 |
| AUDIT-UI-001 | High | ACCEPT | [編集可能値の抑制により後続の正当な入力が失敗](current-mobile.ja.md) | 回帰追加と修正待ち。 |
| AUDIT-IDENTITY-001 | High | ACCEPT | [helper 来歴から設定済み digest が欠落](current-mobile.ja.md) | 回帰追加と修正待ち。 |
| AUDIT-REDACTION-003 | High | ACCEPT | [window の secret 秘匿後に派生 fingerprint が残存](current-mobile.ja.md) | 回帰追加と修正待ち。 |
| AUDIT-PREREQUISITE-001 | High | ACCEPT | [空の helper パスでカレントディレクトリを暗黙採用](current-mobile.ja.md) | 回帰追加と修正待ち。 |
| AUDIT-BOUNDARY-002 | Medium | ACCEPT | [小数秒の Since を黙って短縮](current-mobile.ja.md) | 回帰追加と修正待ち。 |
| AUDIT-LIFECYCLE-001 | High | ACCEPT | [事前拒否を native 外部作用の未確認と誤報](current-mobile.ja.md) | 回帰追加と修正待ち。 |

## 判断と残るレビュー上のリスク

- DOM の最終出力も、semantic/capture と整合させて adapter/capabilities の1MiB snapshot 上限を秘匿後に保つ。製品文書の semantic 限定の文章を DOM の明示要件と誤引用しない。超過は拒否または実際の省略を示して制限し、adapter 上限は弱めない。
- readiness は既存 command run を使い intent・完了と後続削除を制御する。プロセス群・出力未確認の型を通常診断へ落としたり再試行したりしない。未解決行には調査が必要で、不存在を自動推定しない。
- PR #5 最終指摘は過去会話の対応済み表明に依存せず、固定対象で独立に再現した。観測された契約違反を分類し、実証していない別 target 入力や native 破壊を断定しない。
- 現行不具合を再現していない防止策不足は、未採否指摘ではなく明示的な後続候補とする。Windows PID 再読取の直接競合、Java producer の traversal/fingerprint、release close/後段永続化失敗注入、native の悪意ある失敗順序。リスクは稀な経路が直接未検証であることで、既知の現行不具合ではない。大きな後続作業には専用 ExecPlan が必要。既存コードと native 検証を保つ。汎用 OS/ADB 失敗注入基盤の新設は局所修正の範囲を超え、それ自体で native 証明にもならない。
- 同一ユーザーによる敵対的 filesystem/provider 偽装は既存の信頼範囲外。新たに信頼制約を緩めない。

## 検証と独立レビュー

未実施。各担当は解決扱いにする前に、修正前の失敗、対象検証結果、実装とテストの場所を本記録または参照先へ追記する。全 repoctl、race、integration、修正版 native CI、最終独立レビュー、Plan 受入照合が必要。基準版の成功で修正版の証拠を代用しない。
