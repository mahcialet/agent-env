---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/exec-plans/active/execplan-lifecycle-orchestration.md
source_sha256: aec33d37140e1b774cc9600f32791dc6c713c7b59dcf8f07ba166c160a8eaac0
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

M1〜M5を先に完成させ、その後multi-hostを子Plan化してM6〜M8をdogfoodする。

## 進捗

- [ ] base/branch/baseline。
- [ ] M1 lifecycle policy/directories/schema。
- [ ] M1 draft/paused/abandoned bilingual validator。
- [ ] M1 self-migrate `EP-OPS-001`。
- [ ] M2 stable Plan ID / parent / dependency / cycle。
- [ ] M2 merged-default / stacked-exception。
- [ ] M3 repoctl plans list/check/graph/ready。
- [ ] M4 deterministic selection / concurrency=1 / workstream。
- [ ] M4 branch / `ExecPlan:` trailer / PR provenance。
- [ ] M5 automatic/guarded/manual merge policy。
- [ ] M5 latest-HEAD review gate / CI / thread / blocker gate。
- [ ] M5 safe auto-merge / guarded fallback。
- [ ] M1-M5 acceptance。
- [ ] multi-host子Plan分割。
- [ ] M6 parent/child orchestration dogfood。
- [ ] M7 human-validation schema/preflight/evidence/human-kick。
- [ ] M7 multi-host Human Validation checkpoint。
- [ ] M8 merge->complete->reevaluate->next loop。
- [ ] M8 FINDING/BLOCKED feedback。
- [ ] bilingual docs/final checks/retrospective/archive。

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

未完了。

M1-M5 checkpointでschema/ID/dependency/repoctl/provenance/merge gate/Codex limitation/first auto-merge evidenceを記録。

最終的にmulti-host child graph、automatic transition/merge、paused blocker、draft promotion、parent finalization、HV結果/evidence、人間finding、残るmanual stepを記録する。

## 背景と構成

読む:

- `docs/PLANS.md` / `.ja.md`
- bilingual documentation design/review
- repository correctness audit/review
- `tools/repoctl`
- AGENTS/ARCHITECTURE/QUALITY/roadmap英日
- GitHub Actionsとreview/check behavior
- active multi-host-control-plane ExecPlan

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

### Milestone 6 — parent/child

multi-hostを子Plan化。child merge後graph reevaluate。childrenだけでparent auto-completeしない。

### Milestone 7 — Human Validation

`plan_type: human-validation`、`execution_mode: human-kick`、stable scenario ID、PASS/FINDING/BLOCKED。人間が環境を揃えてkick後、AgentがREADY/BLOCKED preflight。purpose-built summary+structured evidenceを用意。

### Milestone 8 — end-to-end

select->branch->implement->review->merge->archive->reevaluate->next->HVを接続。daemon不要、CLI-driven可。

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
15. multi-host reconcile/split。
16. M6 dogfood。
17. M7 Human Validation/evidence。
18. first multi-host checkpoint。
19. finding feedback。
20. M8 dogfood。
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
- E34 multi-host M6 dogfood。
- E35 multi-host M7 checkpoint。
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
