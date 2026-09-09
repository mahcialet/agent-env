---
status: completed
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/exec-plans/completed/repository-correctness-audit.md
source_sha256: 64778d19be0e69a415dd8d7f92203d18dd4c015a0dc433bda063253421125245
---

# Browser/CDP merge後にrepository全体のcorrectness invariantを監査する

[English](repository-correctness-audit.md)

このExecPlanはliving documentであり、`docs/PLANS.md`に従って更新する。

想定ブランチ: `audit/repository-correctness`

PR #10を必須前提とし、`master`へmergeされるまで開始しない。

開始時:
1. master fast-forward
2. post-PR#10 revision記録
3. audit branch作成
4. baseline harness/race/native
5. Phase A target freeze

監査対象revision: `031869c8b9073b8e23bc17fbc55243666a52f557`

## 目的 / 全体像

Browser/CDPまでmergeされたagent-env全体をfeature単位ではなく横断correctness
invariantでadversarial reviewする。新機能追加が目的ではない。

重点はlimit/truncation、false completeness、false ownership/absence、state transition、
effect後persistence、cancel/lock-loss、cleanup proof、stale action、concurrency/reuse、
redaction/truncation、path、native OS差。

代表例:

    limit=2048
    2047 complete
    2048 complete
    2049 truncated

`truncated=true`は実際に返されなかった有効item/frameがあることを意味する。
returned count == limitだけではtruncationではない。

## 対象範囲

freezeしたpost-PR#10 masterのconfig/domain/app/store、Git、Docker、merge済ならPodman、
Android、Flutter、Android UI、persistent process、Browser/CDP、execx、evidence、
paths、readiness/endpoints、reconcile/destroy/GC/quarantine、standalone release/
buildinfo/assets、durable docsのmechanical claimを対象にする。

さらに、completed ExecPlanとreview follow-up planをhistorical defect corpusとして
監査入力に含め、当時追加したregressionと「なぜ既存testが見逃したか」も再検証する。

iOS等の新機能、aesthetic refactor、performance-only workは対象外。

## 監査方式

### Phase A — Review only
product code変更禁止。findingのみ記録。

### Phase B — Disposition
全findingをACCEPT/REJECT/DEFER/DUPLICATEへ分類。
REJECTは証拠、DEFERはseverity/risk/reason/follow-up、DUPLICATEはcanonical finding。

### Phase C — Remediation
ACCEPTだけ:

    reproduce -> regression -> minimal fix -> focused verify -> full verify

completionはfinding zeroではなく、coverage完了、全disposition、全ACCEPT resolved、
unresolved Critical/High=0、DEFER follow-up明示、final verification pass。加えてACCEPT
findingのescape analysis / earliest preventable stageは100%記録し、recurring defect
classにはpreventive guardrailまたはbroader preventionが非現実的な理由を必須とする。

## Finding形式

    AUDIT-<AREA>-NNN

AREA: BOUNDARY/STATE/OWNERSHIP/PERSIST/CANCEL/CLEANUP/CONCURRENCY/STALE/
REDACTION/PATH/PORTABILITY/RELEASE/SECURITY/DOCS。

ID、Severity、Disposition、Invariant、Location、Trigger、Observed、Expected、Impact、
Coverage、Reproducer、Regression、Resolution、Verification、Relatedを記録。
fact/hypothesis/repairを区別。

## Severity

Critical:
unrelated resource破壊、wrong target action、clear secret leak、ambiguous ownershipで
destructive effect。

High:
incomplete cleanupをRELEASED、ambiguous identityをowned/absent、effect成功後identity
loss、stale wrong-target action、live可能process中source削除、cleanup redirect。

Medium:
exact-limit false truncation、valid action誤拒否、non-destructive state誤分類、
retry/timeout off-by-one。

Low:
correctness decisionに影響しないmetadata/diagnostic不整合。

## 主要invariant


### Historical review corpus

completed review/follow-up ExecPlanをdead historyとして扱わずaudit inputにする。

最低限:

- `docs/exec-plans/completed/*review*.md`
- 通常feature ExecPlan内のreview/finding/thread follow-up
- planに記録されたPR thread ID/link
- `Surprises & Discoveries`、failed approach、independent review、
  retrospectiveの「見逃し理由」

をinventoryする。

各historical findingについて:

```text
source / historical ID
original invariant
original defect
original severity/impact（分かる場合）
detected stage
earliest preventable stage
pre-discovery detection opportunities
why each earlier opportunity failed
why tests/review missed it
missing guardrail
preventive control
added regression
current regression location
current production location
still applicable?
current test still proves it?
current test hits real/full entry point?
same-pattern recurrence locations
current AUDIT IDs
guardrail status
```

`Earliest preventable stage`は共通stageで分類する。

```text
S0  Product specification / acceptance criteria
S1  Design / invariant definition
S2  Implementation-local unit test
S3  Package/component integration test
S4  Cross-component integration test
S5  Native / real-provider integration
S6  Repository harness / static validator
S7  PR author self-review
S8  Independent/adversarial PR review
S9  Post-merge repository audit
```

hindsightで早いstageを選ばず、その時点で実際に存在した情報・test surfaceから
現実的に防止可能だった最早stageを記録する。

を記録する。

目的はfix済みコードを再議論することではない。

確認するのは:

1. 当時のregressionが現在も存在し、defect再導入時に本当に落ちるか。
2. refactorでreal entry pointが変わり、old regressionが形骸化していないか。
3. 同じdefect patternが他subsystemへ再発していないか。
4. local fixで終わったがrepository-wide invariantへ昇格すべきだったものがないか。
5. 当時明らかになったtest methodologyの弱点が他でも残っていないか。
6. 本来どのstageで最初に止められたはずか。
7. merge前にどんな具体的detect opportunityが存在し、なぜそれぞれ機能しなかったか。
8. 原因がspec/invariant不足、boundary不足、oracle coupling、helper-only、composition、
   native evidence、failure injection、concurrency、negative fixture、review processの
   どこにあるか。
9. 同じclassを次回S8 reviewより前に落とすpreventive guardrailは何か。

historical replayはcurrent correctness evidenceとdefect escape evidenceの両方を出す。

既知pattern:

- positive fixture/permissive testがimplementationと同じ誤った仮定を共有
- helper/Doctorだけをtestして次のfull entry pointを通していない
- global inventory/filterでunrelated resource kindを落とす
- runtimeごとのreadiness budgetを共有してしまう
- validationより前にstore creation/reservation等のside effectがある
- 1 runtime cleanup failureでindependent safe cleanupまで止める
- OS process observation boundary race
- TCP向けlogicをUDP等別protocolへ誤適用
- optional frontend/toolをlower-level inventoryが誤って必須化

historical finding自体がfix済みでも、same pattern recurrenceがあればcurrent audit
findingを新規発行する。

### Defect escape analysis

current ACCEPT findingと重要なhistorical findingすべてについて、単に「なぜcodeが
間違ったか」ではなく、

> なぜもっと早い段階で検出できたはずのdefectが、実際の発見stageまでescapeしたか

を分析する。

各findingで:

```text
detected stage
earliest preventable stage
detection opportunities before discovery
escape reason(s)
missing/weak oracle
missing composition/native/failure-injection coverage
review-process gap
preventive guardrail
guardrail implementation/evidence
expected future detection stage
```

を記録する。

escape reason category:

- `SPEC_GAP`
- `INVARIANT_GAP`
- `BOUNDARY_GAP`
- `ORACLE_COUPLING`
- `HELPER_ONLY`
- `COMPOSITION_GAP`
- `FAILURE_INJECTION_GAP`
- `NATIVE_EVIDENCE_GAP`
- `CONCURRENCY_GAP`
- `NEGATIVE_FIXTURE_GAP`
- `REVIEW_CHECKLIST_GAP`
- `HARNESS_GAP`

複数category可。final reviewerを責めるのではなく、最も早く改善可能な原因を優先する。

preventive control候補はproduct/design invariant、negative fixture、boundary helper、
full-entry-point integration、failure injection、native CI、repoctl/static check、
ownership/absence helper、PR author self-review/adversarial review checklist。

recurring classはlocal regressionだけで完了にせず、broader guardrailが非現実的なら
その理由を明示する。

### Boundary
0/1/limit-1/limit/limit+1。Browser AX/DOM、Android UI、console/network/log/artifact、
redaction、retry、timeout、port、page/frame、SQLite retry、release limitを確認。
truncatedにはactual omitted evidence必須。

### Positive proof
READY/owned/healthy/complete/clean/absent/released/safe-to-delete/same-targetが
failure未検出だけで成立していないか監査。

### State transition
ALLOCATING->READY、READY->DEGRADED、*->QUARANTINED、
active->releasing->released、failed create compensationのrequired/forbidden evidenceと
persistence barrierを確認。

### Persistence
全effectをintent->effect->identity/result->later effectで監査。
worktree、Compose/Podman、Emulator、Flutter、adb reverse、Android UI、process、
Browser/CDP、release publication。

### Ownership
PID/Job/group、port、project/resource、Podman connection、serial/AVD、CDP port、
page/target/node、path/symlink reuse。nameだけをproofにしない。

### Cleanup
command success != absence proof。uncertain cleanupはevidence/state/reservation保持、
quarantine、dependency削除停止。

### Cancellation
effect前後、persist中、cleanup中、lock loss。lost lock後追加effect無し。
timeout != effect未実行proof。

### Concurrency
independent process/connectionでsame lease/sibling/source/SQLite/ports/assets/release/
cleanup-vs-actionを確認。

### Stale
Android/Browserはcurrent identity再検証+node再resolve+unique後input。
coordinate fallback禁止。truncated evidenceでgone評価禁止。

### Redaction
chunk/truncation/multibyte/redaction expansion/persisted capを確認。

### Path
symlink、..、case、Windows drive/UNC、space、Unicode、state/source、archive、
runtime/profile。

### Cross-platform
Windows/macOS/Linux process identity/tree/file/path/browser差をnative検証。

### Release
tag/version/source、clean tree、determinism、checksum/manifest、validated bytes publish、
archive safety、validation後rebuild無し。

### Docs claim
must/never/always/enforced/rejects/guaranteesのmechanical claimに実装+negative test。

## 進捗

- [x] (2026-09-09) PR #10のmergeを確認し、masterをfast-forward確認（既に最新）、audit/repository-correctnessを作成し、Phase A対象を031869c8b9073b8e23bc17fbc55243666a52f557に固定した。ユーザーが配置した未追跡の日英監査Planを保持して採用する。Phase Aで製品変更は行わない。


- [x] (2026-09-09) PR #10 merge/target revision
- [x] (2026-09-09) audit branch
- [x] (2026-09-09) baseline checks
- [x] (2026-09-09) Phase A freeze
- [x] (2026-09-09) matrix/finding path
- [x] (2026-09-09) completed review/follow-up ExecPlan inventory
- [x] (2026-09-09) historical finding -> current regression/entry-point mapping
- [x] (2026-09-09) material historical findingごとにearliest preventable stage / detection opportunity / escape reason / guardrail記録
- [x] (2026-09-09) historical regression replay / stale coverage確認
- [x] (2026-09-09) historical escape reasonをS0-S9/categoryでaggregate
- [x] (2026-09-09) boundary review
- [x] (2026-09-09) state review
- [x] (2026-09-09) persistence review
- [x] (2026-09-09) ownership review
- [x] (2026-09-09) cleanup review
- [x] (2026-09-09) cancel/timeout/lock review
- [x] (2026-09-09) concurrency review
- [x] (2026-09-09) stale review
- [x] (2026-09-09) redaction review
- [x] (2026-09-09) path review
- [x] (2026-09-09) portability review
- [x] (2026-09-09) release review
- [x] (2026-09-09) docs enforcement review
- [x] (2026-09-09) product code変更無しでPhase A完了
- [x] (2026-09-09) 全finding disposition
- [x] (2026-09-09) Critical/High silent defer無し
- [x] (2026-09-09) ACCEPT regression
- [x] (2026-09-09) accepted Critical/High resolved
- [x] (2026-09-09) accepted Medium/Low in-scope resolved
- [x] (2026-09-09) recurring pattern harness昇格
- [x] (2026-09-09) independent read-only re-review
- [x] (2026-09-09) final harness/race/native/integration
- [x] (2026-09-09) unresolved Critical=0 / High=0
- [x] (2026-09-09) bilingual retrospective
- [x] (2026-09-09) completed移動/link/hash

## 想定外の発見

- 2026-09-09 完了時点: 下記の過去時点で継続中だった検証をすべて照合した。最終19修正、独立レビュー、全harness/race、native CI34295144985/34295144958、最終release検証は成功。受入38行の証拠を記入し両言語Planを完了へ移した。以前の記録はその時点の観測を残したもので、現在の残件ではない。

- 2026-09-09 最終照合: 候補6872286のVerify34294068659とBrowser native34294068663は全成功。ローカル実integrationとrelease検証も成功。履歴資料の最終更新で完了PlanにないPR10のmerge後追加指摘3963154175/4182/4186/4191を発見した。page-create128上限、変更結果のtable表示、ignored AX wait、pressed-state fingerprintを追加のreview-onlyで再現中。採否・修正と新候補検証まで完了移動しない。

- 2026-09-09 時点: Phase A/Bは完了（文書のみのcommit `56b9c2c`）。採用15件の修正と回帰テストを実装した。集約matrix、最終独立レビュー、安定した修正版の検証は継続中。固定版をディスク上の一時領域で再検証しFlutter73.54秒成功、UI75.11秒で最初のtapが失敗しAUDIT-UI-001を確認した。最初の修正版はFlutter107.45秒/UI146.19秒、Linux Browser race10.178秒成功。UIは最後の出力上限修正後に再実行する。途中のharness/race/integrationは、新しいmobile fixtureのサイズとAPI応答の修正中に失敗した。並行integration負荷中のprocess preview deadline失敗は単独で再確認する。


- 2026-09-09: 固定版の全 repoctl、全 race、Docker integration、Linux Browser native race、実 Podman 共存（126.631秒）、実 Android 並行リースと手動終了（48.587秒）、隔離 clone での配布検証（6対象、2回生成した8ファイルの完全一致、Linux実行）が成功。固定 master の Verify 34290359477、Browser native 34290359439 は Windows/macOS/Linux で成功。Flutter/UI integration は実行中で、修正後の検証は未実施。
- 2026-09-09: 推測した CLI テスト2件はタグと名前が誤り「no tests to run」だったため検証件数に含めない。Podman は正しい指定で再実行した。Android fixture 作成は未導入のイメージ種別を指定して失敗後、導入済みイメージを使い専用の一時ホームに作成できた。
- 2026-09-09: Phase A の一時 overlay で AUDIT-BOUNDARY-001（Androidログの上限ちょうど）、AUDIT-DOCS-001（コード内の偽リンク先見出し）、AUDIT-REDACTION-001（Android UIの秘匿処理後上限）、High 候補 AUDIT-CLEANUP-001（readiness後の削除）を再現した。最後は Create 全体でプロセス群の終了未確認後に7回再試行し、source を削除して released となる（0.024秒）。Phase B まで採否は保留し、製品コードは変更していない。

- 2026-09-09: baseline repoctl checkはunit/vet成功後、提供された日本語Planにtranslation_ofとsource_sha256がないためdocs-checkで失敗した。日英内容を確認してPlan metadataのみ補い、製品修正とは扱わない。想定した.github/workflows/verify.ymlは存在せず、実際はci.ymlだった。


exact-limitをfalse truncatedにするpatternをBrowserだけでなくrepository全体で探す。
disproved hypothesisも削除せずconfirmed defectと区別する。

## 判断の記録

- 2026-09-09 追加Phase B: merge後4指摘を固定031869c8で再現した。AUDIT-BOUNDARY-003（128page作成超過）、AUDIT-STATE-001（ignored AX wait）、AUDIT-STALE-002（pressed fingerprint）、AUDIT-CLI-001（変更結果table）をすべてMediumとしてACCEPT。最初の15件も採用を維持し計19件。CDP負例3件は0.061秒で失敗。native CLIは作成/削除両表示の検証で失敗し、作用と兄弟保持は独立に確認した。詳細は追加監査別紙。本監査の範囲照合であり別PRタスクではない。

- 2026-09-09 / 監査実装担当、Phase B: 20件の完了Planと182行の履歴を含むPhase Aを製品変更なしで完了した。再現した15件（High7、Medium6、Low2）をすべてACCEPTとし、DEFER/REJECT/DUPLICATEはない。Phase Cは対象契約と回帰テストの修正に限定する。固定版の実Flutter成功後、実UIの最初のtapでAUDIT-UI-001を再現した。最終検証は未完了。

- 2026-09-09 / 監査実装担当: 恒久レポートはdocs/audits/repository-correctness/に日英index・matrix・findings・historical corpus・escape summaryと範囲を区切ったsubsystem付録として置き、完了後も保持する。Phase Aの再現は隔離一時checkout/Go overlayとlease所有fixtureのみで、製品ファイルや共有資源を変更しない。新規指摘はdisposition時点まで未分類とし、範囲内Medium/Lowは原則ACCEPT、未解決Critical/HighはDEFERしない。public issueやglobal policy変更は含めない。利用可能な既存実integrationとnative CIを必須とし、利用できない基盤は成功と数えず記録する。共通予防策は再発パターンを確認してから採用する。


- PR #10 merge後masterをfreezeして監査。理由: Browser/CDPでlimit/action logicが増える。
- Phase A review-only。理由: reviewer independenceとfinding可視性。
- ACCEPTは可能な限りfix前regression。理由: defect証明と再発防止。
- completionはfinding zero不要。理由: coverage/disposition/resolutionが品質。
- unresolved Critical/High zero必須。理由: core safety guarantee。
- truncatedはactual omitted evidenceを意味する。理由: limitちょうどはcomplete。
- recurring patternはshared check/helperへ昇格。理由: harness quality改善。
- durable audit docs/planは英日。理由: repository policy。

すべて日付/担当: 2026-09-09 / maintainers.

## 成果と振り返り

2026-09-09に完了。固定対象 `031869c8b9073b8e23bc17fbc55243666a52f557`、
最終製品revision `f2ec634baa00af5221217dbfcd5c0aef93c624c9`、branchは
`audit/repository-correctness`。文書のみのPhase A commit `56b9c2c` の後、
意味のまとまりごとに7件の修正commitを作成した。英日の
[報告一覧](../../audits/repository-correctness/index.ja.md)、
[台帳](../../audits/repository-correctness/findings.ja.md)、
[matrix](../../audits/repository-correctness/matrix.ja.md)、
[履歴資料](../../audits/repository-correctness/historical-corpus.ja.md) に詳細証拠と限界を保持する。

20領域×14不変条件、監査開始前の完了Plan20件を確認し、履歴186行
（当初182行＋外部追加4件）を記録した。採用19、解決19、Critical0・High7・Medium10・Low2。
REJECT0・DEFER0・DUPLICATE0。未解決Critical/High/Medium/Lowはすべて0。
現行不具合を実証していない検証不足の後続候補は台帳に明記し、
採用指摘を処置せず残したものとは区別する。

| 検出に関する集計 | 件数 |
| --- | --- |
| 元の独立PRレビューS8 | 4 |
| 元のrepository監査S9 | 15 |
| 固定版の監査再現S9 | 19 |
| 現実的な最早防止S2 | 9 |
| 現実的な最早防止S3 | 7 |
| 現実的な最早防止S4 | 3 |

見逃し分類は重複する。以下は現行19件の分類別所属数で、追加の不具合数ではない。
過去の段階別分析は履歴資料に保持する。

| 見逃し理由 | 指摘数 |
| --- | --- |
| COMPOSITION_GAP | 14 |
| NEGATIVE_FIXTURE_GAP | 10 |
| ORACLE_COUPLING | 9 |
| BOUNDARY_GAP | 7 |
| FAILURE_INJECTION_GAP | 4 |
| INVARIANT_GAP | 3 |
| CONCURRENCY_GAP | 2 |
| REVIEW_CHECKLIST_GAP | 2 |
| HELPER_ONLY | 1 |

共通していたのは局所的な正しい検査を利用先まで適用しない問題だった。
readinessはexecutorの確実性を失い、UI/DOMは最終redactionより前に上限を検査し、
page列挙の上限は作成を制限せず、state生成は古さの判断に必要なfieldを落としていた。
既存の永続CommandRun、表示対象proseの共通抽出、開いたfileの制限読込を使い、
厳密な境界・artifact検査とprotocol mutation fixtureで実際の利用先を確認した。
入力判断を安全に支えられない秘密値由来fingerprintは削除し、通常の編集可能snapshotは
入力に使えるように維持した。

独立レビューではreadiness通常キャンセルの再試行、作用後の不透明fingerprint、
fixture到達性・byte調整の弱さを発見し完了前に直した。重いnative integration中に
process previewのdeadline失敗が1回あったが、変更しない単独race付き10回と
その後の全native CIは成功した。tmpfsのemulator容量不足は実出力から診断し、
timeoutや検査を弱めずディスク上の一時領域で再実行した。最終履歴照合では元の完了Browser
Planに記載されていない実不具合4件を追加した。完了資料だけでなく外部follow-upも照合する必要がある。

最終 `repoctl check`、docs/翻訳検査、全race、独立レビューは成功。
Verify34295144985とBrowser native34295144958は最終製品revisionの
Windows/macOS/Linuxで成功し、nativeとcross-buildの証拠を区別した。
実Docker、Podman共存、Android、Flutter/UI、Browser成功は一覧に記録。
最終隔離cloneのrelease検証は6対象を2回生成して8ファイル一致とLinux native smokeを確認した。
その後は監査・製品文書の完了処理のみで、製品dependencyや責務境界は変えず、coreにPOSIX shellも追加していない。

今後のPRレビューには、実装から独立した上限期待値、注入到達、利用先全体・永続出力の検査、
拒否と許可の対、意味が異なるOSのnative証拠を推奨する。本監査の提案・実装済みtestであり、
未承認の全体規則にはしない。限界としてWindowsの2回目PID読取りへの直接注入、
Java traversal producer、release close/遅延永続化の一部注入は網羅し尽くしていない。
同一userの敵対的host/ADBやすべてのnative動作順序の証明も範囲外。既知の採用不具合は残らない。
merge・公開tag・release・無関係なrepository操作は完了作業に含めない。

## 背景と構成

AGENTS/ARCHITECTURE/PLANS/QUALITY/RELIABILITY/SECURITY/PORTABILITY/roadmap英日、
current product/design indexes、major completed ExecPlans、repoctl、CIを読む。

最低review package:
config/domain/app/store/source/Compose/Podman/Android/Flutter/Android UI/process/
Browser/CDP/execx/evidence/paths/release/buildinfo/assets。

## 作業計画

1. target freeze/baseline
2. invariant x subsystem matrix
3. historical review corpus inventory/replay + defect escape analysis
4. boundary audit
5. state/persistence/cleanup audit
6. identity/stale/concurrency audit
7. redaction/path/portability audit
8. Phase A report/disposition
9. severity順 remediation
10. recurring invariant promotion
11. independent read-only re-review
12. final verification/archive

## 具体的な手順

1. PR #10 merge
2. target revision
3. audit branch
4. 英日plan
5. baseline
6. matrix
7. historical review corpus
8. Phase A残り全review
9. finding set
10. disposition
11. regression/fix
12. harness昇格
13. re-review
14. final matrix
15. Critical/High zero
16. retrospective
17. completed/link/hash

## 検証と受け入れ

| ID | 必須動作 | 証拠 |
| --- | --- | --- |
| A1 | post-PR#10 masterをfreeze | 固定対象 `031869c8`、専用branch、Phase A文書commit `56b9c2c`。 |
| A2 | Phase A product code変更無し | `git diff 031869c8 56b9c2c` は監査文書のみ。製品修正は `dc358e1` から。 |
| A3 | subsystem x invariant matrix complete | `docs/audits/repository-correctness/matrix.ja.md`: 20領域×14不変条件。 |
| A4 | completenessに影響するlimitをexact-boundary review | matrixの上限列とmobile/process-browser/Compose-release別紙。AX/node/frame/log/byte/fieldを確認。 |
| A5 | accepted boundary classにlimit-1/limit/limit+1 regression | UI4096文字±1、1000node±1、JSON1MiB±1、ログ2000行±1、DOM JSON1MiB±1、release読込上限前後、Browser page127/128/129。 |
| A6 | count==limitだけでtruncatedにしない | 余分な1行と実node省略を根拠に判定。ちょうど上限なら完全。BOUNDARY-001、REDACTION-001/002。 |
| A7 | READY/owned/clean/absent/released positive proof audit | 制御処理の段階・肯定的根拠表とprovider別紙。CLEANUP-001、OWNERSHIP-001、LIFECYCLE-001を修正。 |
| A8 | major external effectのintent/effect/identity ordering audit | 制御処理のintent/effect/result確認、provider受領情報・永続化テスト、readiness試行の永続化回帰。 |
| A9 | effect成功+persistence failureでidentity silent loss無し | 既存process起動受領情報、application/UI/browser実行中行と保存失敗検証、新規readiness安全性テスト。 |
| A10 | cleanupはabsence proof監査 | Destroy/GCの実行中行による隔離、ID欠落の公開Down拒否、実リースの兄弟を保つ削除検証。 |
| A11 | PID/port/project/connection/serial/page/node reuse監査 | provider identity再利用表。source登録、process birth/Job、Compose engine/project、AVD serial、helper digest、Browser target/node。 |
| A12 | lock loss/cancel後unintended effect無し | readinessの型付きキャンセル回帰、既存lock-loss/command-tree/UI/Browser fence、修正版race成功。 |
| A13 | cross-process claimはindependent process/connection test | SQLite独立connection/process予約、execx native helper、実Docker/Android/UI並行リース。Podman99.951秒成功。 |
| A14 | Android/Browser stale action exact target再検証 | UIの編集可能値正常系とsecret/stale拒否、helper build一致と復旧、CDP述語評価後の文書切替回帰。 |
| A15 | truncated evidenceでgone/absence誤判定無し | 既存Android wait部分/打切りとBrowser AX/frame不存在の負例を維持。mobile/Browser表で確認。 |
| A16 | redaction/truncation multibyte/chunk/limit boundary | evidenceのchunk跨ぎ・複数byte文字と最終UI/DOM保存byte検査。秘密値由来hashを削除。 |
| A17 | path symlink/traversal/case/space/Unicode/Windows | source/paths/assets/release表とOS固有pathテスト。隔離release smokeのstate rootはUnicode。 |
| A18 | native Windows/macOS/Linux semantics分離検証 | Verify34295144985 native Go1.26/1.27×3OS成功、Browser34295144958実3OS成功。最終f2ec634。 |
| A19 | release source/determinism/validated-byte publication audit | cleanな最終f2ec634の隔離cloneでrelease-verify成功。6対象2回生成、8ファイル一致、Linux smoke、release回帰suite成功。 |
| A20 | durable mechanical claimにimplementation/negative test | docsCheck全体のfragment負例/正常例。UI/DOM上限とreadiness永続化の両言語説明を修正。docs-check成功。 |
| A21 | completed review/follow-up ExecPlanを全inventoryまたはN/A明記 | historical-corpusで完了英語Plan20件とPR5外部24指摘を記録。 |
| A22 | historical regressionをcurrent test/entry pointへmapしstale/weakened coverage確認 | 186行に現行実装・回帰経路と検証限界を対応。過去版の再実行とコード確認を区別。 |
| A23 | historical defect patternを他subsystemへrecurrence search | 過去・現行別紙の再発探索で19件を確認。不成立の仮説も記録。 |
| A24 | 過去の「なぜtestが見逃したか」をcurrent audit fixture/methodへ反映 | 公開経路callback、永続run/artifact検査、独立した負例/正常例、実JSON境界を追加。 |
| A25 | 重要なhistorical finding全件にdetected stageとearliest realistic preventable stage | 過去別紙の各重要行で検出・最早防止段階を分類。historical-corpusで集約。 |
| A26 | historical finding全件にpre-discovery detection opportunityと失敗理由 | 過去別紙に以前の検出機会・弱い検証条件・防止策を記録。レビュー担当者の意図は推測しない。 |
| A27 | current ACCEPT finding全件にescape reasonとearliest preventable stage | matrixはACCEPT19/19件を対応。監査再現S9、元S9=15件/S8=4件。最早S2=9件/S3=7件/S4=3件。 |
| A28 | recurring defect class全件にpreventive guardrail evidenceまたは非実施理由 | matrixの再発分類表に9種の防止策・再利用判断と証拠、汎用防止の限界を明記。 |
| A29 | 全finding disposition/rationale | findingsでACCEPT19、REJECT0、DEFER0、DUPLICATE0。各契約違反の採用根拠。 |
| A30 | ACCEPT finding regression evidence | Phase C別紙に固定版失敗overlayと恒久回帰を記録。独立レビューでreadiness通常キャンセルを発見し修正。 |
| A31 | accepted Critical/High resolved | 採用High7件をすべて修正し独立レビュー済み。Critical指摘なし。 |
| A32 | unresolved Critical=0 | 0件。Criticalの採用・先送りなし。 |
| A33 | unresolved High=0 | 0件。High7/7件を解決。 |
| A34 | DEFERにrisk/reason/follow-up | DEFERなし。現行不具合ではない検証不足の後続候補と理由はfindingsに明記。 |
| A35 | recurring defectをreusable harnessへ昇格 | documentProse再利用、readiness CommandRun/runWithCancellation、releaseの開いたfileの制限読込、UI raw/normalized/result共通上限、領域別境界検査。 |
| A36 | independent final read-only re-review | 実装者とは別の担当がreadiness/docs、Compose/release、mobile、Browser、CLIを確認。最終追加CDPはrace付き5回2.039秒成功、残る不具合なし。 |
| A37 | final repoctl/docs/race/native/integration pass | 最終repoctl/check/docs/race成功。Verify34295144985とBrowser34295144958成功。実Docker/Podman/Android/Flutter/UI/release証拠と適用範囲は監査一覧。 |
| A38 | 英日plan stats/evidence/retrospective後archive | 英日成果節に19指摘、履歴186行、段階・分類集計、具体策、最終CIと限界を記録。docs-checkと移動linkを検証。 |

existing CI greenだけではaudit passではない。

## 冪等性と復旧

Phase Aはobservational。finding/reportをversion管理。remediationはsmall coherent commit。
新failureが出てもinvariantを弱めない。invalid findingはREJECTへ変更し証拠を残す。

## 成果物と注記

監査別紙と最終証拠へリンクする恒久的な入口:

    docs/audits/repository-correctness/
      findings.md
      findings.ja.md
      matrix.md
      matrix.ja.md
      historical-corpus.md
      historical-corpus.ja.md
      index.md
      index.ja.md

durableにするなら英日/indexed。stable ID、secret禁止。
final summaryはtarget revision、areas、finding統計、新regression/check、final run IDs。

## インターフェースと依存

audit用production dependency追加無し。existing Go tests/repoctl/native CI/fake provider/
real integration/SQLite fault injection/process/browser/android fixtureを使用。
shellよりportable Go harnessを優先。

## 未解決事項

本監査では開始時の12論点をすべて解決した。

1. 文書は `docs/audits/repository-correctness/` に置き、index/findings/matrix/historical-corpusを英日で提供する。
2. 完了後も別紙を保持し、完了Planから詳細証拠へ参照する。
3. 固定版の再現は隔離した一時overlay/cloneを使い、恒久的に意味のある回帰だけを製品履歴へ加える。
4. Medium10件はすべて採用・修正し、先送りしない。
5. Low2件も採用・修正する。
6. 導入済みの必須integrationはDocker、Podman共存、Android、Flutter/UI、native Browser、release検証を実行した。native OS検証にはCIを使い、cross-buildだけで代替しない。
7. 未依頼のQUALITY規則変更はしない。既存契約と対象回帰で厳密な上限を守る。
8. 構文だけのlimit比較検査は導入しない。省略は領域ごとの有効性・encoding・skipに依存するため、matrixに正常例/負例の検証条件を記録する。
9. DEFERがないため、そのGitHub issueも作らない。
10. 小規模な反復監査は提案にとどめ、新しい必須workflowにしない。
11. S0–S9/escape分類は本監査で使い、将来Planへの必須化は別の方針決定とする。
12. 具体策は既存harness/testと監査matrixに置き、新しい一般方針文書は作らない。
