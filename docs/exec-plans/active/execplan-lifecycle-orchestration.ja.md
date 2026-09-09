---
status: active
plan_id: EP-OPS-001
plan_type: implementation
priority: 10
merge_policy: guarded
base_branch: master
branch: feat/execplan-lifecycle-orchestration
workstreams:
  - repoctl
  - documentation
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/exec-plans/active/execplan-lifecycle-orchestration.md
source_sha256: 4610c15f3792d535eb818961e5c081d7d6fdc3f9eb625c7f95e7aa6991838bd8
---

# ExecPlan lifecycle orchestrationと自動delivery gateを追加する

[English](execplan-lifecycle-orchestration.md)

このExecPlanはliving documentであり、`docs/PLANS.md`に従って更新する。

Bootstrap Plan ID: `EP-OPS-001`

想定ブランチ: `feat/execplan-lifecycle-orchestration`

開始revision: `8f17283dd35436ae5ef6ad43fe0d7a3377a594f2`（PR #13 merge済みmaster）。

現行repoはlegacy metadataから開始する。M1で新schemaを導入後、本Plan自身を`plan_id: EP-OPS-001`へself-migrateする。中間commitでもdocs-checkを壊さない。

## 目的 / 全体像

ExecPlanをliving documentから機械的に検証・選択・review・merge・dependency progression・human validationできるdelivery graphへ発展させる。

```text
DRAFT
  -> explicit promotion
ACTIVE
  -> runnable selection
  -> branch / implementation / tests / PR
  -> Codex/independent review
  -> merge gate
  -> auto merge
COMPLETED
  -> graph reevaluate
  -> next ACTIVE
  -> Human Validation
       -> explicit human kick
       -> PASS / FINDING / BLOCKED
```

implementation unitはExecPlan。Human Validation unitはcoherent product milestone。

M1〜M5を通常実装し、M6で完了済みmulti-hostの履歴を再構成する。M7で完成済み製品に対する新規Human Validation Planを作り、M8で履歴再現・人間検証・次の新規ExecPlanの実運用を組み合わせる。

## 進捗

- [x] 2026-09-09: base/branch/baseline。
- [x] 2026-09-09: M1 lifecycle policy/directories/schema。
- [x] 2026-09-09: M1 draft/paused/abandoned bilingual validator。
- [x] 2026-09-09: M1 self-migrate `EP-OPS-001`。
- [x] 2026-09-09: M2 stable Plan ID / parent / dependency / cycle。
- [x] 2026-09-09: M2 merged-default / stacked-exception。
- [x] 2026-09-09: M3 repoctl plans list/check/graph/ready。
- [x] 2026-09-09: M4 deterministic selection / concurrency=1 / workstream。
- [x] 2026-09-09: M4 branch / `ExecPlan:` trailer / PR provenance。
- [x] 2026-09-09: M5 automatic/guarded/manual merge policy。
- [x] 2026-09-09: M5 latest-HEAD review gate / CI / thread / blocker gate。
- [x] 2026-09-09: M5 safe auto-merge / guarded fallback。
- [ ] M1-M5 acceptance。
- [x] 2026-09-09: 完了済みmulti-hostの履歴を新modelへ対応付ける。
- [x] 2026-09-09: M6 historical orchestration replay。
- [x] 2026-09-09: M7 human-validation schema/preflight/evidence/human-kick。
- [ ] M7 multi-host Human Validation checkpoint。
- [ ] M8 履歴再現＋Human Validation＋次の新規ExecPlanのforward live dogfood。
- [ ] M8 FINDING/BLOCKED feedback。
- [x] 2026-09-09: 永続ドキュメントの英日更新。
- [ ] 最終repoctl/docs/translation/race/native検証。
- [ ] 振り返りを完成させ、英日Planをアーカイブ。

## 想定外の発見

現行repoctlのactive/completed前提、unknown frontmatter、path-based index、translation move/hash、Codex reviewed SHA、GitHub auto-merge/ruleset、commit trailer edge、stacked retarget、parent pause semantics、Human Validation evidence qualityを記録する。

automationを通すためgateを弱めない。

## 判断の記録

- 2026-09-09 / maintainers: lifecycleはdraft/active/paused/completed/abandoned。
- 2026-09-09 / maintainers: active複数可。activeはrunnableでありrunningではない。
- 2026-09-09 / maintainers: 初期implementation concurrencyは1。
- 2026-09-09 / maintainers: stable Plan IDをpath/titleから分離しbranch/commit/PRへ伝播。
- 2026-09-09 / maintainers: parentとdepends_onを分離。
- 2026-09-09 / maintainers: dependency defaultはbase branchへmerged+completed、stackedは明示例外。
- 2026-09-09 / maintainers: draft auto-promote禁止。
- 2026-09-09 / maintainers: pausedはpause_reason/resume_when必須でconcurrency queueにしない。
- 2026-09-09 / maintainers: normal Planはgate成熟後auto-merge、risk高いものguarded/manual。
- 2026-09-09 / maintainers: reviewはcurrent PR HEADのみ有効。
- 2026-09-09 / maintainers: completedはmerge後+archive。
- 2026-09-09 / maintainers: Human Validationは別plan type、auto-run禁止、checklist+evidence+dedicated plan+human kick。
- 2026-09-09 / maintainers: M1-M5後multi-hostでM6-M8をdogfood。

## 成果と振り返り

2026-09-09の実装時点では、M1〜M5のツール、M6の履歴再現、M7のHuman Validation基盤をDraft PR #14に実装した。完了済み記録を書き換えず、5状態のメタデータ、不変ID、依存関係による選択、Git上の帰属を検証できる。実環境の原子的な保護と機械判定できる受け入れ証拠が不足しているため、merge adapterは読み取り専用でguarded運用を維持する。自動mergeの実績はない。

本Planはactiveのままで、未完了である。multi-hostの環境準備、明示的なhuman kickとシナリオの証拠、次の実際に新しい通常ExecPlanによるforward live dogfood、最終native検証、baseへのmergeが残る。履歴再現や合成fixtureではこれらを代替しない。親Planの最終受け入れとアーカイブでは、それぞれの結果を照合する。

独立レビューでは、初期fixtureが通っていても検出できなかった祖先関係、ID保持、入力サイズ制限の不具合が見つかった。各失敗条件を切り分けた回帰テストを追加した。受け入れの直接証拠は、実装状況や実環境のレビュー・人間の観察についての推測と区別する。

## 背景と構成

読む:

- `docs/PLANS.md` / `.ja.md`
- bilingual documentation design/review
- repository correctness audit/review
- `tools/repoctl`
- AGENTS/ARCHITECTURE/QUALITY/roadmap英日
- GitHub Actionsとreview/check behavior
- 完了済みmulti-host-control-plane ExecPlanと実際のbranch・commit・PR・merge履歴

現行のdedicated branch + active plan + acceptance evidence + retrospective原則を維持しながら一般化する。

## 作業計画

### Milestone 1 — lifecycle contract

`draft/active/paused/completed/abandoned` directory、legal state、state-specific metadata、PLANS policy、repoctl validation、EN/JA negative fixture。本PlanをEP-OPS-001へself-migrate。

### Milestone 2 — stable identity/dependency

immutable Plan ID、parent/depends_on、missing/self/cycle reject、merged-default/stacked exception。

### Milestone 3 — repoctl plans

`list/check/graph/ready`をdeterministic/read-onlyで実装。

### Milestone 4 — execution selection/Git provenance

active+dependency satisfied+no blocker+no conflictをpriority/Plan ID順で1件選択。branch、commit trailer、PR Plan IDを検証。

### Milestone 5 — review/merge gate

automatic/guarded/manual。current HEAD review、CI/native/integration、blocking review/thread、acceptance、bilingual docs、base freshness、explicit blockerをgateにする。

free-form LGTMは認証に使わない。Codex reviewed-SHAがmachine-readableでなければguarded/manual fallback。

### Milestone 6 — Historical orchestration replay

完了済みmulti-hostのExecPlan、branch、commit、PR、merge履歴を入力に、新しいlifecycleと
依存modelならどう進行したかを再構成・検証する。派生した親子ノードは再現用データであり、
過去のPlanを分割し直さない。子のmerge後の再評価と親自身の受け入れ確認を検証し、
子の完了だけでは親を完了しない。

### Milestone 7 — Human Validation dogfood

完成済みmulti-hostに対する新規の`plan_type: human-validation` ExecPlanを作る。
`execution_mode: human-kick`、固定scenario ID、前提、操作、期待観測、証拠、PASS条件を必須とする。
人間が環境をそろえて明示的に開始した後だけ、preflightを行いREADY/BLOCKEDを報告する。
人が読める証拠と構造化した証拠を用意し、PASS/FINDING/BLOCKEDを追跡可能なPlanへ戻す。

### Milestone 8 — End-to-end acceptance

履歴再現、実際のHuman Validation、次の新規の通常ExecPlanによるforward live dogfoodを組み合わせる。
選択、branch・commit・PRの識別情報、最新HEADのreview、merge、merge後のarchive、依存再評価、
次のPlan選択を確認する。明示的なCLI呼び出しでよくdaemonは不要。
人間による開始や次のPlanでの証拠は、実際に観測するまで未完了として残す。

## 具体的な手順

1. base/branch。
2. bilingual active plans。
3. baseline。
4. lifecycle policy/schema。
5. negative fixtures。
6. self-migrate EP-OPS-001。
7. graph。
8. repoctl plans。
9. selection/workstream。
10. Git provenance。
11. merge policy/gate。
12. post-review commitでgate invalidation証明。
13. safe auto-merge/fallback。
14. M1-M5 acceptance。
15. 完了済みmulti-hostの履歴を変更せず再構成。
16. M6で過去の親子の進行を検証。
17. M7 Human Validation/evidence。
18. first multi-host checkpoint。
19. finding feedback。
20. 履歴再現・Human Validation・次の新規ExecPlanの実運用を統合。
21. durable docs/final checks。
22. retrospective/archive。

## 検証と受け入れ

- E1 existing plan migration path。
- E2 5 lifecycle semantics。
- E3 status/path mismatch reject。
- E4 paused reason/resume required。
- E5 draft non-runnable/non-auto-promote。
- E6 multiple active valid。
- E7 Plan ID unique + EN/JA sync。
- E8 ID survives move/title/branch deletion。
- E9 parent/dependency separate。
- E10 dependency missing/self/cycle reject。
- E11 default merged dependency。
- E12 stacked explicit。
- E13 deterministic repoctl plans commands。
- E14 deterministic selection / concurrency 1。
- E15 paused not concurrency queue。
- E16 workstream conflict。
- E17 branch/commit/PR provenance。
- E18 applicable commit has `ExecPlan:` trailer。
- E19 later commit invalidates review gate。
- E20 free-form review prose not authorization。
- E21 CI/native/integration/thread gate。
- E22 automatic plans merge only with all gates。
- E23 guarded/manual no auto-merge。
- E24 unmerged remains active。
- E25 completed means merged+archived。
- E26 child merge graph reevaluate。
- E27 children do not auto-complete parent。
- E28 parent-level finalization required。
- E29 Human Validation never auto-run。
- E30 preflight READY/BLOCKED。
- E31 stable scenario/expected/PASS。
- E32 human-readable + structured evidence。
- E33 FINDING returns to tracked Plan。
- E34 完了済みmulti-host履歴を新lifecycle・依存modelで再構成・検証。
- E35 新規multi-host Human Validationを明示開始後に実施し、M8では次の新規ExecPlanの実運用も記録。
- E36 correctness-audit guardrails preserved。
- E37 bilingual durable docs。
- E38 final checks。
- E39 evidence/retrospective/archive。

## 冪等性と復旧

graph queryはread-only。EN/JA moveは同期。draft auto-promote、pause reason invent、stale review auto-merge、uncertain merge blind retry、pre-merge completed、HV auto-run禁止。

restart時はplan graph、Git、GitHub PR/check/review/merge stateを再readしてnext actionをderiveする。

## 成果物と注記

`repoctl plans list/graph/ready`はderived output。

将来event候補: plan_promoted/paused/resumed/selected、review_gate/merge_gate、merged/completed、human_validation_kicked/passed/finding。

実装前にimplementedと宣伝しない。

## インターフェースと依存

主に`tools/repoctl/`。必要ならgeneric plan parser/modelをinternal packageへ分離。

```text
parser/model
 -> lifecycle/dependency
 -> query/selection
 -> Git provenance
 -> merge-gate adapter
 -> Human Validation contract
```

agent-env product runtimeから依存しない。

M1-M4はnew production dependency不要。M5もexisting GitHub/workflow capabilityを優先。

## 未解決事項

1 Plan ID grammar/allocation
2 migrated plan_type default
3 depends_on YAML/satisfaction enum
4 resume_when structure
5 promotion criteria
6 priority ordering
7 workstream schema
8 abandoned dependency replacement
9 historical completed Plan ID migration
10 trailer scope
11 branch normalization
12 PR Plan-ID format
13 Codex reviewed-SHA signal
14 GitHub ruleset vs repoctl gate authority
15 auto-merge enabling mechanism
16 base freshness/merge queue
17 merge failure recovery
18 required/optional child representation
19 parent active/paused semantics
20 HV evidence/operator notes location
21 human-kick前preflight可否（default no）
22 validation bundleをproduct/repoctlどちらに置くか
23 human FINDING -> follow-up Plan ID
24 future parallel execution running-state authority

## 実行記録（2026-09-09）

開始時の未追跡ファイルは本Planの日英ペアだけだった。multi-hostと文書再構成はPR #12、#13で
マージ済みである。元のmulti-host分割手順は現状に合わないため、履歴の再現と新規の人間検証を
使うM6〜M8の確認を依頼し、独立して進められるM1〜M5に着手する。完了済み実装はやり直さない。

baselineのrepoctl checkは単体テストとvetに成功したが、提供された日本語Planに
translation_of/source_sha256がないためdocs-checkが失敗した。内容を照合してmetadataを追加し、
再検査する。race baselineは実行中。作業ブランチは将来のPlan-ID命名に対する明示的な初回例外とする。

2026-09-09 / maintainerによる範囲の修正: M6は履歴再現、M7は新規Human Validation、M8は履歴再現＋人間検証＋次の新規ExecPlanの実運用とし、古いmulti-host再分割手順を置き換える。完了済み履歴を保持する。baseline raceは成功し、翻訳metadataの修正後のdocs-checkも成功した。

### 初期schemaと移行の判断

- 本Planは`EP-OPS-001`、`merge_policy: guarded`とし、指定された初回ブランチを維持する。
  今後のブランチ名はtypeと小文字のIDから決める。
- 依存は`plan_id`と`satisfaction: merged|stacked`で表し、省略時はmergedとする。
  選択可能性は状態の文章だけでなく、利用側のbaseに対するGitの祖先関係で判断する。
- priorityは必須の非負整数で、小さい順、同順位ならPlan ID順とする。
  worktreeのブランチを観測して実行枠を1つに制限し、人間検証は別途明示開始するまで選択しない。
- 過去の完了Planは固定したファイル名の許可リストで保持し、新規Planには構造化metadataを求める。
  過去の翻訳例外はこれと区別する。
- merge gateは現在のbaseに由来する信頼済み方針と原子的な保護を確認できなければ拒否し、読み取りだけにとどめる。
  GitHub上の自由文の賛成コメントを承認とは扱わない。本Planはguardedで、自動merge可能とは主張しない。

### 実装とレビューの証拠（2026-09-09）

M1〜M4は`plans_model.go`、`plans.go`と既存の文書・翻訳harnessへの統合で実装した。
新規の5状態のPlanには厳密なYAMLを求め、過去の完了記録は維持する。正常系テストの日本語版には
lifecycle metadata全体を引き継ぎ、見出し不足の異常系はschemaを有効に保って、見出し検出を単独で試す。

M5は最新HEADのmerge判定modelと、読み取り専用のGitHub adapterを用意した。テストでは完全な
信頼済み証拠だけを許可し、古い承認・check・base、自己承認、blocking thread、不明な証拠を拒否する。
実際のadapterはbase由来の信頼済み方針、原子的なruleset保護、機械可読の受け入れ証拠がそろうまで
BLOCKEDを返す。文書化したguardedへの移行であり、実際の自動mergeを実証したとは扱わない。

M6は実際のPR #12を使った。mergeは`084da57de177c0a09bc3cb61ae99faff8bd79a94`、
元のheadは`cc55382e29db86794f832cbb5c0e3e6d775e722e`、baseは
`dc63308e53f68f8be99f7cbf59cafc78f7296b71`、branchは`feat/multi-host-control-plane`、
commitは23件、merge時刻は`2026-09-09T11:10:50Z`で、`gh pr view 12`とも一致した。
replayは実際のgraph・選択engineで実装・review・親の派生ノードを検証する。実装のmergeで
reviewは選択可能になるが親は待機する。仮のreview merge後も親の受け入れ確認を選択するだけで、
過去のreview承認や親の完了を捏造しない。

M7はpausedの`EP-MHOST-001`と固定IDの8scenarioを作った。人間検証のpreflightとrecordは
読み取り・接続・書き込みの前に明示開始を要求する。模擬テストは前提不足、loopbackのREADY、
PASSを推測しないこと、追跡付きFINDING、証拠ディレクトリの不変性、過大・後続JSON、同じsnapshotのhashを確認した。
実際の環境準備、開始、scenarioの実施、追跡Planへの反映は未完了。M8は次の新しい通常ExecPlanも
必要とし、模擬再現や今回の初回実装をその実運用の証拠とは数えない。

独立レビューでは、stackedの確認先が利用branchでなく変更可能なbaseだったこと、引用符付きYAMLのIDで
完了Planの見出し検査を回避できること、全ペア削除でID保持検査を回避できることを発見し、回帰テスト付きで修正した。
全体担当はさらに、draftを含むだけの無関係なcommitが依存のmerge証拠にならないよう厳密化した。
人間検証frameworkの独立レビューでは、上限付きreaderが後続JSONを隠し、その後の上限なし再読み込みで
hashを計算していた問題を修正し、同じ上限付きsnapshotを解析・hash化するようにした。
方針と人間検証Planは英語、日本語単独での理解、意味の一致の独立レビューを通過した。
fixtureの失敗を直すために検査を弱めてはいない。

統合後の全体`repoctl check`と`go test -race ./...`は成功した。最後のmerge証拠の厳密化後も
provenance・mergeの関連テストは成功した。native CIは全Git履歴で新しい`plans check`を実行する。
remoteでの結果は未確認である。

### 受け入れ証拠の現在地（2026-09-09）

| 項目 | 状態と証拠 |
| --- | --- |
| E1 | 成功。PLANSの移行方針と過去の完了Planの固定allowlist。新規Planにはschemaを必須化。 |
| E2 | 成功。5状態の厳密なmetadata解析と、状態別の正常系・異常系fixture。 |
| E3 | 成功。TestPlanMetadataStrictValidationでdirectory/statusの不一致を拒否。 |
| E4 | 成功。pausedではpause_reasonとresume_whenの両方を必須とする異常系を検査。 |
| E5 | 成功。draftの昇格条件を必須化し、readinessはdraftを選択・変更しない。 |
| E6 | 成功。複数activeのgraphを受け入れ、選択は最大1件とするfixture。 |
| E7 | 成功。TestLoadPlanGraphPairsAndReferencesでID重複と日英metadataの差を拒否。 |
| E8 | 成功。改名しても履歴のIDを保持し、merge証拠はbranch参照の保存に依存しない。削除回帰を検査。 |
| E9 | 成功。親子と実行依存は別graph。親子関係だけでは実行依存を追加しない。 |
| E10 | 成功。欠落・自己参照・cycleの異常系fixture。 |
| E11 | 成功。completed、baseの祖先関係、ID、一意のmerge元trailerを要求し、draftを含むだけのcommitを拒否。 |
| E12 | 成功。明示stackedと実際の利用branchの祖先関係を検査。branch作成前だけ宣言baseを使う。 |
| E13 | 成功。整列したgraph/JSONと、決定的で読み取り専用のlist/check/graph/ready。 |
| E14 | 成功。priority、ID順で1件を選び、Git worktreeから実行枠を確認。 |
| E15 | 成功。readinessは状態を変更せず同時実行制限を報告。pausedには実際のblockerを求める方針。 |
| E16 | 成功。readinessがworkstream/競合理由を出力し、状態は維持。 |
| E17 | 成功。ブランチ命名、一意のcommit trailer、PR本文の表示される単独Plan-IDを検査。 |
| E18 | 成功。現在のdelivery範囲の非merge commitにExecPlan trailer各1件を要求。本PlanのcommitはEP-OPS-001付き。 |
| E19 | 成功。TestPlanMergeGateFailsClosedで追加HEAD、古いcheck/baseを拒否。 |
| E20 | 成功。構造化した承認だけを使い、自由文は承認modelに含めない。 |
| E21 | 成功。必須CI/docs/native/integration、app識別、未解決thread/reviewの異常系を検査。 |
| E22 | modelは成功。完全な模擬証拠は許可できるが、liveの原子的保護と機械的な受け入れ証拠は未確立なのでguardedとし、自動mergeしない。 |
| E23 | 成功。guarded/manualは自動mergeを許可しない。 |
| E24 | 現時点で順守。未mergeの本実装はactiveに保ち、完了とは宣言しない。 |
| E25 | 成功。archive先と完全なmerge SHAをschemaで要求し、plans checkでbaseへの到達とPlan固有のmerge provenanceを検査。 |
| E26 | 成功。replayが検証済み子mergeの前後で実際のgraph/readiness engineを使う。 |
| E27 | 成功。実装の子がmergeされても親は待機し、自動完了しない。 |
| E28 | 成功。仮のreview merge後もactiveの親の受け入れ確認を選択するだけで、親の受け入れは別途必要。 |
| E29 | 成功。人間検証を自動選択せず、kickがなければ読み取り・接続・書き込み前に拒否。 |
| E30 | frameworkは成功。名前付き実行ファイルとendpointからREADYまたは具体的なBLOCKEDを出力。実環境は未確認。 |
| E31 | 成功。EP-MHOST-001の8scenarioについて一意のID、操作、観測、PASS条件を検査。 |
| E32 | 成功。JSONとMarkdown、上限付きの同じsnapshotのdigest、証拠ディレクトリの不変性。 |
| E33 | frameworkは成功。FINDINGには追跡中のdraft/active review Planを求めてIDを記録。実際のPlanへの反映は未完了で操作担当が行う。 |
| E34 | 成功。PR #12のGit/GitHub事実と実際のreadiness engineによるreplayを記録。 |
| E35 | 未完了。人間による明示開始・scenario実施と、次の新規通常ExecPlanによるforward live dogfoodが必要。 |
| E36 | 成功。全体harnessと従来の異常系fixtureを保持。fixtureのmetadataを移行し、assertionは弱めていない。 |
| E37 | 成功。日英の方針・Plan metadata・コマンド制限をレビューし、docs-check/hashを検査。 |
| E38 | localは成功。全体check/race、最終repoctl race、Windows/macOS向けcross-buildに成功。native CIは未確認。 |
| E39 | 最終化は未完了。日英の実装証拠を記録し、最終retrospective/archiveは人間検証・forward受け入れ・base merge後。 |

### deliveryの確認点

Draft PR #14には`ExecPlan: EP-OPS-001`があり、`b1f8c8e`、`c95268a`、`c79d709`のcommitも
同じtrailerを保持する。`plans provenance --plan EP-OPS-001 --pr-body`は成功した。
実際の`plans gate --plan EP-OPS-001 --pr 14 --repo mahcialet/agent-env`は観測したHEAD/baseと
base側の信頼済み方針不足という理由を添えてBLOCKEDを返し、mergeは行っていない。
最終のmodel確認では、呼び出し元が古い証拠を渡してもabandonedのstacked依存を拒否する。
実Gitの回帰テストでmerge済みの模擬branchを削除してもPlan固有のmerge証拠を確認できた。
両方の関連テストが成功した。

実装commitには`9cca40b`も含む。この時点のローカルの全`repoctl check`と
`go test -race ./tools/repoctl -count=1`は成功した（race: 9.616秒）。
英日両方の進捗欄で、実装済みM3コマンドの完了を記録した。
日本語のチェック欄は、実装の証拠があるにもかかわらず未チェックのまま残っていた。

### PR #14 CIのreadiness fixture修正（2026-09-10）

`03b656d`のPR Verify run 34356431558では、`go test -race ./...`中に
`TestLifecycleCreatePersistedIntentAndUniqueIsolation`の2件目のCreateが
`context deadline exceeded`で失敗し、Docker integrationはスキップされた。
同じHEADのpush run 34356426394は成功した。共通fixtureは正常な作成でも、
SQLiteへの保存を含むreadiness処理に50msしか割り当てていなかった。
正常系fixtureには既存の慣例である5秒を使い、
`TestLifecycleReadinessTimeoutRollsBack`では50msを明示して維持する。
製品の期限と、rollback・分離に関するassertionは変更しない。

`errors.Is(context.DeadlineExceeded)`のassertion追加も試したが、
race付き50回反復で失敗した。キャンセルと永続化が競合すると、SQLiteが
`sql: transaction has already been committed or rolled back`を返す場合がある。
製品のエラー契約を新たに変更せず、この追加assertionを取り下げた。
元のエラー発生と全資源解放のassertionは維持している。
修正過程の全ローカルharnessと全race検査は成功した。最終反復結果は以下に追記する。

最終の対象検証：`go test -race ./internal/app -run
'^TestLifecycle(CreatePersistedIntentAndUniqueIsolation|ReadinessTimeoutRollsBack)$'
-count=50 -timeout=3m`は成功（22.606秒）。正常系とrollbackを両方検証した。

### stacked履歴の帰属と依存証拠の保持（2026-09-10）

レビューで、readinessは継承したstacked commitを認める一方、provenanceは
baseからHEADまでの全範囲をconsumerに帰属させる不整合が見つかった。
また、完了を検証できてもstacked readinessが依存branchの存続を要求していた。
両方を修正した。activeな依存の履歴を再帰的に検証し、検証したtipの履歴を
consumerの検査範囲から除外する。consumer自身のcommitが少なくとも1件必要な
条件と、正確なtrailerの検査は維持する。completedな依存は、削除されたbranchの
代わりに検証済みの不変なmerge証拠を使う。親が2つのmergeではsource headを、
squashでは配信されたsquash commitだけを証拠とし、consumerが実際にそのcommitを
含むことを要求する。任意の過去のtipを推測で採用しない。

検証：隔離Git回帰テストで、コマンド入口を通した正常なstacked provenance、
依存側・consumer側の誤ったtrailer、consumer commitの欠落、祖先関係の欠落、
完了後のbranch削除、不正なmerge証拠を検証した。
`go test -race ./tools/repoctl -count=1`は成功（12.162秒）、全`repoctl check`も
成功した。読み取り専用の独立レビューに指摘はなかった。以前の個別fixtureが
見落とした、選択と帰属検査を組み合わせた挙動を今回のテストで検証する。
