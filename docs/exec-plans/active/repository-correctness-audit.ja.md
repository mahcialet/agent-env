---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/exec-plans/active/repository-correctness-audit.md
source_sha256: d66029def94d6b011c62605a9243f1b706ed0eb3c0070bcaed40effa21789ea1
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

- [x] 2026-09-09: PR #10のmergeを確認し、masterをfast-forward確認（既に最新）、audit/repository-correctnessを作成し、Phase A対象を031869c8b9073b8e23bc17fbc55243666a52f557に固定した。ユーザーが配置した未追跡の日英監査Planを保持して採用する。Phase Aで製品変更は行わない。


- [x] PR #10 merge/target revision
- [x] audit branch
- [ ] baseline checks
- [x] Phase A freeze
- [ ] matrix/finding path
- [ ] completed review/follow-up ExecPlan inventory
- [ ] historical finding -> current regression/entry-point mapping
- [ ] material historical findingごとにearliest preventable stage / detection opportunity / escape reason / guardrail記録
- [ ] historical regression replay / stale coverage確認
- [ ] historical escape reasonをS0-S9/categoryでaggregate
- [ ] boundary review
- [ ] state review
- [ ] persistence review
- [ ] ownership review
- [ ] cleanup review
- [ ] cancel/timeout/lock review
- [ ] concurrency review
- [ ] stale review
- [ ] redaction review
- [ ] path review
- [ ] portability review
- [ ] release review
- [ ] docs enforcement review
- [ ] product code変更無しでPhase A完了
- [ ] 全finding disposition
- [ ] Critical/High silent defer無し
- [ ] ACCEPT regression
- [ ] accepted Critical/High resolved
- [ ] accepted Medium/Low in-scope resolved
- [ ] recurring pattern harness昇格
- [ ] independent read-only re-review
- [ ] final harness/race/native/integration
- [ ] unresolved Critical=0 / High=0
- [ ] bilingual retrospective
- [ ] completed移動/link/hash

## 想定外の発見

- 2026-09-09: 固定版の全 repoctl、全 race、Docker integration、Linux Browser native race、実 Podman 共存（126.631秒）、実 Android 並行リースと手動終了（48.587秒）、隔離 clone での配布検証（6対象、2回生成した8ファイルの完全一致、Linux実行）が成功。固定 master の Verify 34290359477、Browser native 34290359439 は Windows/macOS/Linux で成功。Flutter/UI integration は実行中で、修正後の検証は未実施。
- 2026-09-09: 推測した CLI テスト2件はタグと名前が誤り「no tests to run」だったため検証件数に含めない。Podman は正しい指定で再実行した。Android fixture 作成は未導入のイメージ種別を指定して失敗後、導入済みイメージを使い専用の一時ホームに作成できた。
- 2026-09-09: Phase A の一時 overlay で AUDIT-BOUNDARY-001（Androidログの上限ちょうど）、AUDIT-DOCS-001（コード内の偽リンク先見出し）、AUDIT-REDACTION-001（Android UIの秘匿処理後上限）、High 候補 AUDIT-CLEANUP-001（readiness後の削除）を再現した。最後は Create 全体でプロセス群の終了未確認後に7回再試行し、source を削除して released となる（0.024秒）。Phase B まで採否は保留し、製品コードは変更していない。

- 2026-09-09: baseline repoctl checkはunit/vet成功後、提供された日本語Planにtranslation_ofとsource_sha256がないためdocs-checkで失敗した。日英内容を確認してPlan metadataのみ補い、製品修正とは扱わない。想定した.github/workflows/verify.ymlは存在せず、実際はci.ymlだった。


exact-limitをfalse truncatedにするpatternをBrowserだけでなくrepository全体で探す。
disproved hypothesisも削除せずconfirmed defectと区別する。

## 判断の記録

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

未完了。target revision、review area、severity/disposition統計、resolved/rejected/deferred、
root-cause pattern、新regression/check、cross-platform finding、promoted invariant、
remaining risk、future PR checklistを記録する。さらにdetected stage、earliest preventable
stage、escape reason category、追加したpreventive controlを集計し、今後どのstageへ
検出を前倒しできたかを示す。

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
| A1 | post-PR#10 masterをfreeze | Pending |
| A2 | Phase A product code変更無し | Pending |
| A3 | subsystem x invariant matrix complete | Pending |
| A4 | completenessに影響するlimitをexact-boundary review | Pending |
| A5 | accepted boundary classにlimit-1/limit/limit+1 regression | Pending |
| A6 | count==limitだけでtruncatedにしない | Pending |
| A7 | READY/owned/clean/absent/released positive proof audit | Pending |
| A8 | major external effectのintent/effect/identity ordering audit | Pending |
| A9 | effect成功+persistence failureでidentity silent loss無し | Pending |
| A10 | cleanupはabsence proof監査 | Pending |
| A11 | PID/port/project/connection/serial/page/node reuse監査 | Pending |
| A12 | lock loss/cancel後unintended effect無し | Pending |
| A13 | cross-process claimはindependent process/connection test | Pending |
| A14 | Android/Browser stale action exact target再検証 | Pending |
| A15 | truncated evidenceでgone/absence誤判定無し | Pending |
| A16 | redaction/truncation multibyte/chunk/limit boundary | Pending |
| A17 | path symlink/traversal/case/space/Unicode/Windows | Pending |
| A18 | native Windows/macOS/Linux semantics分離検証 | Pending |
| A19 | release source/determinism/validated-byte publication audit | Pending |
| A20 | durable mechanical claimにimplementation/negative test | Pending |
| A21 | completed review/follow-up ExecPlanを全inventoryまたはN/A明記 | Pending |
| A22 | historical regressionをcurrent test/entry pointへmapしstale/weakened coverage確認 | Pending |
| A23 | historical defect patternを他subsystemへrecurrence search | Pending |
| A24 | 過去の「なぜtestが見逃したか」をcurrent audit fixture/methodへ反映 | Pending |
| A25 | 重要なhistorical finding全件にdetected stageとearliest realistic preventable stage | Pending |
| A26 | historical finding全件にpre-discovery detection opportunityと失敗理由 | Pending |
| A27 | current ACCEPT finding全件にescape reasonとearliest preventable stage | Pending |
| A28 | recurring defect class全件にpreventive guardrail evidenceまたは非実施理由 | Pending |
| A29 | 全finding disposition/rationale | Pending |
| A30 | ACCEPT finding regression evidence | Pending |
| A31 | accepted Critical/High resolved | Pending |
| A32 | unresolved Critical=0 | Pending |
| A33 | unresolved High=0 | Pending |
| A34 | DEFERにrisk/reason/follow-up | Pending |
| A35 | recurring defectをreusable harnessへ昇格 | Pending |
| A36 | independent final read-only re-review | Pending |
| A37 | final repoctl/docs/race/native/integration pass | Pending |
| A38 | 英日plan stats/evidence/retrospective後archive | Pending |

existing CI greenだけではaudit passではない。

## 冪等性と復旧

Phase Aはobservational。finding/reportをversion管理。remediationはsmall coherent commit。
新failureが出てもinvariantを弱めない。invalid findingはREJECTへ変更し証拠を残す。

## 成果物と注記

候補:

    docs/audits/repository-correctness/
      findings.md
      findings.ja.md
      matrix.md
      matrix.ja.md
      historical-review-corpus.md
      historical-review-corpus.ja.md
      defect-escape-summary.md
      defect-escape-summary.ja.md

durableにするなら英日/indexed。stable ID、secret禁止。
final summaryはtarget revision、areas、finding統計、新regression/check、final run IDs。

## インターフェースと依存

audit用production dependency追加無し。existing Go tests/repoctl/native CI/fake provider/
real integration/SQLite fault injection/process/browser/android fixtureを使用。
shellよりportable Go harnessを優先。

## 未解決事項

1. finding/matrix/historical-review corpus durable path
2. completed後finding/corpus file保持方法
3. Phase A generated/untracked fixture policy
4. Medium defer policy
5. Low remediation policy
6. infrastructure unavailable時のfinal integration minimum
7. exact-limit ruleをQUALITY invariantへ即昇格するか
8. repoctlでsuspicious limit comparison検出可能か
9. DEFERをGitHub issueへmirrorするか
10. major feature merge後の小規模correctness audit定例化
11. S0-S9/escape reasonをfuture review follow-up ExecPlanで必須化するか
12. preventive guardrailをQUALITY.mdまたは専用review/testing policyへ記録するか
