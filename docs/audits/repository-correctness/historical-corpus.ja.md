---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/historical-corpus.md
source_sha256: afd5b94b91c9db629ec79dbbed6e3a1330ff4d24d4eb46205eea32e30b4041cc
---

# 過去指摘の全体索引と見逃し分析

[English](historical-corpus.md) · [監査索引](index.ja.md)

固定対象に存在する英語の完了 Plan 全20件を確認した。日本語版は翻訳として扱い、別の指摘件数には数えない。後続レビュー Plan に加えて feature Plan 内の独立レビューと発見も調べた。各付録には重要な指摘、元の記録、現行実装入口、回帰テストの実際の assertion と限界を示す。元資料にない Thread ID は作らない。PR #5 の feature Plan には後続レビューがないため、外部コメント24件を補完した。全182行（mobile77、MVP/process/browser56、Compose/release42、文書7）を記録した。まとめた行数は個別 PR コメントの件数とは異なる。

## 全 Plan 一覧

| 完了 Plan | 重要な指摘と現行回帰テストの対応 | 適用 |
| --- | --- | --- |
| [android-emulator-lease](../../exec-plans/completed/android-emulator-lease.md) | [history-mobile](history-mobile.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [android-emulator-review](../../exec-plans/completed/android-emulator-review.md) | [history-mobile](history-mobile.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [android-emulator-review-2](../../exec-plans/completed/android-emulator-review-2.md) | [history-mobile](history-mobile.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [flutter-android-runtime](../../exec-plans/completed/flutter-android-runtime.ja.md) | [history-mobile](history-mobile.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [flutter-android-review](../../exec-plans/completed/flutter-android-review.ja.md) | [history-mobile](history-mobile.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [android-ui-observer](../../exec-plans/completed/android-ui-observer.ja.md) | [history-mobile](history-mobile.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [agent-env-mvp](../../exec-plans/completed/agent-env-mvp.md) | [history-process-browser](history-process-browser.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [persistent-process-runtime](../../exec-plans/completed/persistent-process-runtime.ja.md) | [history-process-browser](history-process-browser.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [process-destroy-preview-review](../../exec-plans/completed/process-destroy-preview-review.ja.md) | [history-process-browser](history-process-browser.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [browser-cdp-automation](../../exec-plans/completed/browser-cdp-automation.ja.md) | [history-process-browser](history-process-browser.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [compose-provider-podman](../../exec-plans/completed/compose-provider-podman.ja.md) | [history-compose-release](history-compose-release.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [compose-provider-podman-review](../../exec-plans/completed/compose-provider-podman-review.ja.md) | [history-compose-release](history-compose-release.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [standalone-distribution](../../exec-plans/completed/standalone-distribution.ja.md) | [history-compose-release](history-compose-release.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [standalone-distribution-review](../../exec-plans/completed/standalone-distribution-review.ja.md) | [history-compose-release](history-compose-release.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [standalone-release-finalization](../../exec-plans/completed/standalone-release-finalization.ja.md) | [history-compose-release](history-compose-release.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [standalone-release-review](../../exec-plans/completed/standalone-release-review.ja.md) | [history-compose-release](history-compose-release.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [standalone-release-filter-review](../../exec-plans/completed/standalone-release-filter-review.ja.md) | [history-compose-release](history-compose-release.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [standalone-verify-review](../../exec-plans/completed/standalone-verify-review.ja.md) | [history-compose-release](history-compose-release.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [bilingual-documentation](../../exec-plans/completed/bilingual-documentation.ja.md) | [documentation](documentation.ja.md) | 適用。現行製品の動作または有効な検証手段。 |
| [bilingual-documentation-review](../../exec-plans/completed/bilingual-documentation-review.ja.md) | [documentation](documentation.ja.md) | 適用。現行製品の動作または有効な検証手段。 |

## 検出段階と分類の集約

1件が複数分類を持つため、分類別件数を独立の不具合件数として足さない。S0–S9 は仕様、設計、unit、package、合成、native/provider、harness、作成者レビュー、独立レビュー、監査。指摘ごとの最早防止段階・検出段階・具体的機会は付録の行を参照し、この集約で上書きしない。

| 最早段階 | 繰り返す見逃し | 具体的な防止策と現行証拠 |
| --- | --- | --- |
| S0/S1 | 回復・不確実性の契約不足や外部作用を不可分とみなす。SPEC_GAP、INVARIANT_GAP。 | 外部作用前に永続的禁止記録と所有・不存在証明を決める。application/build/process/UI には既存の保護があるが、readiness で AUDIT-CLEANUP-001 が再発。 |
| S2 | 正常 fixture が実装をなぞり、上限等値がない。ORACLE_COUPLING、BOUNDARY_GAP、NEGATIVE_FIXTURE_GAP。 | limit−1/limit/limit+1 とフィールド削除を独立に検証。Android log の上限ちょうどと release の短い root が既存検証の不足を示す。 |
| S3/S4 | 補助関数の成功後に別層がフラグを置換し、型を失い、データを膨張させる。HELPER_ONLY、COMPOSITION_GAP、FAILURE_INJECTION_GAP。 | Service.Create/UI/Browser/Down または repoctl 全体を通し、永続結果・外部作用なしを独立に検証。readiness、mobile/DOM 秘匿、container identity の再現はこれらの入口を通る。 |
| S5 | host/provider が fake と異なる。NATIVE_EVIDENCE_GAP、CONCURRENCY_GAP。 | 独立プロセス・接続、Windows Job、実 provider 検証。固定版 native CI と Linux 基準検証は候補版証明と区別。Flutter の tmpfs 空き不足は失敗として保持。 |
| S6 | 検証条件がなければ harness 成功でも発見できない。HARNESS_GAP。 | 防止策を文書だけでなく実行される test/check 経路に置く。翻訳の過去不具合と非表示アンカーの再発がこの限界を示す。 |
| S7/S8 | 修正した呼出し元だけを見て別の利用箇所を逃す。REVIEW_CHECKLIST_GAP。 | 契約の全呼出し元を検索し、作用・証拠の境界で合成検証する。繰り返すレビュー指摘が本監査の理由であり、個人の意図は推測しない。 |
| S9 | 監査で残存・再発を発見。 | 修正前に再現を残し、修正後に独立再レビュー。将来の不具合全廃を保証しない。 |

各分類は付録の記録に基づき、資料が足りない検出機会を断定しない。Windows の2回目 PID 読取、Java producer の完全性・fingerprint、process readiness と browser 選択の一部、release の後段永続化失敗には直接回帰の不足があり、製品不具合と区別する。無条件に ACCEPT 指摘へ数えない。広い防止策を今回の修正外に残す場合は Phase B でリスクと後続作業を記録する。共通指示の自動変更は提案しない。

## 遅れて確認した外部commentの追加

上の初回一覧は番号付き182行だった。最終GitHub照合で完了Planにないmerge後PR #10
commentを4件発見し、外部参照元を明示する以下の行を加えて合計**186**件とする。
既存Plan別行と過去重要度は変更しない。詳細な不具合・段階・予防策は
[Browser追加資料](supplemental-browser.ja.md)と[CLI追加資料](supplemental-cli.ja.md)を参照。
4件全て固定031869c8で再現し、別PR作業ではなく本監査内でACCEPTとした。

| 資料内行 | 元comment | 監査指摘 |
| --- | --- | --- |
| EXT-PR10-01 | 3963154175 | AUDIT-BOUNDARY-003 |
| EXT-PR10-02 | 3963154182 | AUDIT-CLI-001 |
| EXT-PR10-03 | 3963154186 | AUDIT-STATE-001 |
| EXT-PR10-04 | 3963154191 | AUDIT-STALE-002 |

追加の過去指摘4件であり、未記録外部commentが従来から全て完了Planに含まれていたとはしない。
thread照合で資料coverage欠落が判明したため、完了Planを書き換えず参照付き追加で由来を保つ。
