---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/findings.md
source_sha256: 6dbc8c0343dff8402bd5ed68abddee1d415176e52b0fbec3df2e326a3885286d
---

# 指摘の採否と修正記録

[English](findings.md) · [監査索引](index.ja.md)

2026-09-09、全担当の Phase A レポートと過去182行の記録が揃った後、Phase B の区切りを記録し、下記採否を確定した。参照先には固定した修正前の観測と再現を残す。この記録で最終採否と解決状況を管理する。

## 採否

最終照合でmerge後の4件を追加採用した。現行合計は19件（High7、Medium10、Low2）。最初の15件の判断も維持する。

再現した19件（当初15件とmerge後の追加4件）をすべて ACCEPT とする。既存の安全性・identity・上限・文書リンクに関する範囲内の契約違反または再発であり、局所的に修正できる。High7件は必須で、黙って先送りしない。低い重要度の症状は分類を共有しても実装経路と検証条件が異なるため重複扱いにしない。確認済み指摘の REJECT/DEFER/DUPLICATE はない。

| ID | 重要度 | 採否 | 理由と証拠 | 解決 |
| --- | --- | --- | --- | --- |
| AUDIT-CLEANUP-001 | High | ACCEPT | [readiness の終了未確認後に再試行・source削除を許可](current-control-plane.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-OWNERSHIP-001 | High | ACCEPT | [container identity 欠落で破壊的 Down を許可](current-compose-release.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-DOCS-001 | Low | ACCEPT | [非表示見出しでフラグメント検証を通過](documentation.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-RELEASE-001 | Low | ACCEPT | [短い source root の漏洩検出を回避](current-compose-release.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-RELEASE-002 | Medium | ACCEPT | [事前検査だけの読込上限をファイル成長で超過](current-compose-release.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-BOUNDARY-001 | Medium | ACCEPT | [ログ2000行ちょうどで省略と誤判定](history-mobile.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-REDACTION-001 | Medium | ACCEPT | [mobile 証拠が秘匿後の上限を超過](current-mobile.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-REDACTION-002 | Medium | ACCEPT | [DOM 公開時の秘匿後に adapter 上限を超過](current-process-browser.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-STALE-001 | Medium | ACCEPT | [load wait が旧 snapshot と新文書の判定を混合](current-process-browser.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-UI-001 | High | ACCEPT | [編集可能値の抑制により後続の正当な入力が失敗](current-mobile.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-IDENTITY-001 | High | ACCEPT | [helper 来歴から設定済み digest が欠落](current-mobile.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-REDACTION-003 | High | ACCEPT | [window の secret 秘匿後に派生 fingerprint が残存](current-mobile.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-PREREQUISITE-001 | High | ACCEPT | [空の helper パスでカレントディレクトリを暗黙採用](current-mobile.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-BOUNDARY-002 | Medium | ACCEPT | [小数秒の Since を黙って短縮](current-mobile.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-LIFECYCLE-001 | High | ACCEPT | [事前拒否を native 外部作用の未確認と誤報](current-mobile.ja.md) | 解決済み。修正を確認するテストと独立レビューが成功。 |
| AUDIT-BOUNDARY-003 | Medium | ACCEPT | [page作成が128を超え後のcloseを妨げる](supplemental-browser.ja.md) | 解決済み。固定版で不具合を再現し、修正を確認するテストと独立レビューが成功。 |
| AUDIT-STATE-001 | Medium | ACCEPT | [ignored AXがwait条件を誤って成立させる](supplemental-browser.ja.md) | 解決済み。固定版で不具合を再現し、修正を確認するテストと独立レビューが成功。 |
| AUDIT-STALE-002 | Medium | ACCEPT | [pressed変更で入力fingerprintが無効にならない](supplemental-browser.ja.md) | 解決済み。固定版で不具合を再現し、修正を確認するテストと独立レビューが成功。 |
| AUDIT-CLI-001 | Medium | ACCEPT | [変更結果tableが作用前page一覧を表示する](supplemental-cli.ja.md) | 解決済み。固定版で不具合を再現し、修正を確認するテストと独立レビューが成功。 |

## 判断と残るレビュー上のリスク

- DOM の最終出力も、semantic/capture と整合させて adapter/capabilities の1MiB snapshot 上限を秘匿後に保つ。製品文書の semantic 限定の文章を DOM の明示要件と誤引用しない。超過は拒否または実際の省略を示して制限し、adapter 上限は弱めない。
- readiness は既存 command run を使い intent・完了と後続削除を制御する。プロセス群・出力未確認の型を通常診断へ落としたり再試行したりしない。未解決行には調査が必要で、不存在を自動推定しない。
- PR #5 最終指摘は過去会話の対応済み表明に依存せず、固定対象で独立に再現した。観測された契約違反を分類し、実証していない別 target 入力や native 破壊を断定しない。
- 現行不具合を再現していない防止策不足は、未採否指摘ではなく明示的な後続候補とする。Windows PID 再読取の直接競合、Java producer の traversal/fingerprint、release close/後段永続化失敗注入、native の悪意ある失敗順序。リスクは稀な経路が直接未検証であることで、既知の現行不具合ではない。大きな後続作業には専用 ExecPlan が必要。既存コードと native 検証を保つ。汎用 OS/ADB 失敗注入基盤の新設は局所修正の範囲を超え、それ自体で native 証明にもならない。
- 同一ユーザーによる敵対的 filesystem/provider 偽装は既存の信頼範囲外。新たに信頼制約を緩めない。

## 検証と独立レビュー

実装commit: readiness `dc358e1`、ownership/release `b477aa4`、fragment `b91074f`、Browser `499c5b2`、Android UI `6872286`。参照先のPhase C節に固定版で不具合を検出したテストの失敗と、修正後に同じ条件を確認したテストを記録した。修正版の全 `repoctl check` と `go test -race ./...` は成功（app45.555秒）。process previewはテストを変更せず単独race付き10回で8.205秒成功。Docker integration（CLI170.949秒）、最新の実UI139.632秒、実Emulator44.613秒、隔離cloneの6対象release検証（2回生成・8ファイル一致・Linux smoke）、Browser native CI34294068663の3OSも成功。Verify34294068659とPodman99.951秒も成功。readiness/docs、Browser、最終mobileの独立レビューも成功（mobile bounds race付き3回4.352秒）。merge後の4件も最終候補f2ec634で修正・独立検証を完了し、下記に証拠を記録した。rootは別担当のCompose/releaseの実装と、修正を確認するテストを照合し、残る問題はなかった。

追加実装: CLI `ddf8ae4`、CDP `f2ec634`。独立CLIレビューでoperation errorによる成功判定と対象identityを確認。追加CDPはrace付き5回2.039秒成功。最終候補の全race、`repoctl check`、sandbox有効Browser native race10.157秒、6対象再現性検証は成功。Browser native34295144958は3OS成功、Verify34295144985もnative/cross-build/integrationの全jobが成功した。追加日本語文書の初期metadata誤りを修正し、全docs-checkも成功した。
