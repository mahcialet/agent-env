---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/exec-plans/active/reader-first-documentation-restructure.md
source_sha256: d2083cbf6038de9f37cc6bcf431594323f977a912fab72ad85dd65ecdfc2c5a6
---

# Durable documentation全体をreader-firstな英語・日本語へ再構成する

[English](reader-first-documentation-restructure.md)

このExecPlanはliving documentであり、`docs/PLANS.md`に従って更新する。

想定ブランチ: `docs/reader-first-documentation-restructure`

開始revision: `084da57de177c0a09bc3cb61ae99faff8bd79a94`

編集前にmasterをfast-forwardし、正確な開始revisionを記録して専用branchを作成し、
documentation baselineを実行する。

## 目的 / 全体像

対象は`roadmap`だけではない。

repository内のdurable human-facing documentation全体を、英語・日本語ともに
「正しいが圧縮されていて読み解く必要がある文章」から、読み手が目的に沿って
追える構造へ再構成する。

現状では、複数の文書で次の傾向がある。

- 関係の薄い複数機能を1文に詰め込む
- 実装済み・延期・対象外・未決事項が同じ段落に混ざる
- 詳細な実装証拠を入口文書でも繰り返す
- 英文で名詞句が積み重なり、仕様書のように構文解析しないと読みづらい
- 日本語が英文の節・語順を必要以上に保持し、意味は正しいが日本語として重い
- README、Architecture、roadmap、product spec、design doc、policy、ExecPlan間で
  同じ事実を異なる粒度で繰り返す
- どの文書が入口で、どれが詳細なauthorityなのか分かりづらい

目標の情報構造:

```text
README
  agent-envとは何か
  何ができるか
  最初にどう使うか
  次にどこを読むか

docs/index
  読者の目的・機能別navigation

ARCHITECTURE
  system map / boundary / authority / lifecycle

roadmap
  機能群ごとに
    implemented
    active
    deferred
    open decisions

policy
  QUALITY / RELIABILITY / SECURITY / PORTABILITY / PLANS

product spec
  user-visible contract

design docs / ADR
  why / how

active ExecPlan
  current work / acceptance

completed ExecPlan
  historical evidence
  原則として現在の文体へ書き換えない
```

情報を削って短くすることが目的ではない。
各文書が明確なreader questionへ答え、詳細を適切なauthorityへ置くことが目的。

## Reader-first編集モデル

英語はnormative canonical sourceのまま。

ただしcanonicalだからといって、読みにくい英語を維持し、それを日本語で同じ構造に
再現する必要はない。

各pairのworkflow:

```text
current facts
 -> document purpose分類
 -> duplicate/misplaced content特定
 -> Englishをreader-firstに再構成
 -> fact/link/contract verify
 -> complete meaningを日本語化
 -> 日本語としてreader-firstに再構成
 -> semantic parity review
 -> docs-check
```

日本語で必ず保持:

- normative meaning
- implemented/deferred/unsupported status
- safety limitation
- identifier/command
- evidence/link

保持不要:

- 英文と同じ文数
- clause順
- paragraph境界
- noun phrase構造
- 同一heading wording

英日pairはsemantic siblingsでありline-by-line mirrorではない。

## Documentation原則

### 1文書1primary reader question

READMEは「何で、どう始めるか」。
Architectureは「どう分割され、責務はどこか」。
Roadmapは「何があり、何を進め、何を延期しているか」。
Securityは「trust boundaryとsafety guaranteeは何か」。
Product specは「featureが何を約束するか」。
Design docは「どう実現するか」。
ExecPlanは「どう実装・検証するか」。

関連しているだけでcatch-allにしない。

### 実装順ではなくconceptでgroup

例:

```text
Sources/worktrees
Container runtimes
Android/mobile
Browser/UI
Persistent processes
Testing/evidence
Distribution
Multi-host
Security/trust
```

### statusを明示

必要な文書では:

- Implemented
- Active / in progress
- Deferred
- Unsupported / out of scope
- Open decision

を区別する。

### progressive disclosure

入口文書では要点とboundaryを説明し、詳細なcontract/evidenceへlinkする。
roadmapでprotocol-level evidenceを繰り返さない。

### hard constraintを削らない

reader-first編集でownership、cleanup proof、trust boundary、unsupported case、
platform limitation、MUST/NEVERを弱めない。

### noun pile/list sentenceを分解

独立したcapabilityを1文へ多数詰め込まない。
短い導入+bullets、複数文、table、subsectionを使う。

### concrete subject/verb

抽象名詞の連鎖より、誰が何をするかを明示する。

### evidenceはnarrativeではなくevidence

run ID/version/hashは重要だが、入口説明を妨げるなら詳細authorityへ置く。

## 対象範囲

### Tier 1 — entry/navigation

- README英日
- AGENTS英日
- ARCHITECTURE英日
- docs/index英日
- roadmap英日

最優先。

AGENTSはmap/operating guideとしての役割とline capを維持。

### Tier 2 — policy/quality

- PLANS
- QUALITY
- RELIABILITY
- SECURITY
- PORTABILITY
- bilingual documentation policy/design

rule/rationale/implementation/evidence/limitationを区別する。

### Tier 3 — current product specs

`docs/product-specs/`のcurrent pair全件。

各specで:

- 何を解決するか
- userが何を宣言/実行するか
- agent-envが何をownするか
- guarantee
- reject
- out of scope
- failure/recovery

が読みやすいことを確認。

### Tier 4 — design docs / ADR

`docs/design-docs/`, `docs/adr/`。

context/constraint/chosen architecture/authority/lifecycle/tradeoff/failure recoveryを
読みやすくする。

user-visible normative behaviorがdesignだけにある場合はproduct specへauthorityを戻す。

### Tier 5 — active ExecPlans

`docs/exec-plans/active/`。

historical evidenceを無闇に書き換えない。
Purpose/current state/milestone/acceptance/navigationの可読性を改善。
required section/decision/failure/evidenceは保持。

### Tier 6 — current audits/durable guidance

現在のquality guidanceとして読む必要がある`docs/audits/`を対象。
raw historical evidenceはstyle目的だけで書き換えない。

## 明示的な対象外

原則としてwholesale rewriteしない:

- completed ExecPlan
- historical review plan
- handoff/reference archive
- generated docs
- generated DB schema
- LICENSE
- third-party text
- source comment/CLI message

completed ExecPlanはhistorical evidence。
broken linkやmaterial metadata problem以外で現在風に直さない。

## 情報保持とtraceability

substantial doc編集前にcontent map:

```text
old claim
 -> keep
 -> move
 -> replace with authority link
 -> remove: stale/duplicate
```

nontrivial removalには理由が必要。

SECURITY/RELIABILITY/Architecture/product specs/active ExecPlanなどhigh-risk docは
final版とcontent mapを比較する。

## Fact verification

editorial changeだからstale factを温存するわけではない。

疑わしい記述はcurrent authorityを確認し、code/test/current contract/completed acceptance/
active planで検証して英日両方修正。

書きやすいから新機能を発明しない。
active/deferredをimplementedと書かない。

## Roadmap固有方針

roadmapはgrouped status documentへ。

候補:

```text
How to read
Current capability map
Sources/worktrees
Container runtimes
Android/mobile
Browser/UI
Host processes
Testing/evidence/harness
Distribution/releases
Multi-host/orchestration
Trust/writable workflows
CI/platform coverage
```

各groupでimplemented/active/deferred/openを必要に応じ明示。

## README固有方針

first contact優先:

```text
What it is
Core use cases
Key capabilities
Quick start
Lifecycle mental model
Safety/ownership
Runtime overview
Where to read more
Status/limitations
```

Architecture/roadmapのcopyにしない。

## Architecture固有方針

system map:

```text
At a glance
Core concepts
Authority/ownership
Package/component map
Create
Observe/reconcile
Destroy/cleanup
Persistence/evidence
Runtime/provider boundaries
Platform boundaries
Control-plane boundary
Detailed contract links
```

roadmap statusを混ぜすぎない。

## Policy文書

可能な範囲で:

```text
Purpose
Invariant/requirements
Enforcement
Evidence/checks
Limitations
Related docs
```

を意識。

## Product spec

必要に応じ:

```text
Purpose
User-visible model
Configuration/commands
Ownership/identity
Lifecycle
Observation/results
Failure/recovery
Security/safety
Platform
Out of scope
Related design/evidence
```

機械的template化はしない。

## 日本語編集ポリシー

bilingual documentation designへ明示追加:

- English canonicalは意味のauthority
- Japaneseはsemantic translationでstructural mirrorではない
- sentence/paragraph reorder可
- 日本語として明確なら主語省略可
- 安定した日本語技術語の方が明確なら不要な英語混在を避ける
- identifier/command/protocol/code語彙はprecision優先
- English noun pileを日本語noun chainへ移さない
- bullets/tableがparse costを下げるなら使う
- modal表現を直訳せずstatus/limitationを明示する

source_sha256はsource review acknowledgmentでありstructure同一性の証明ではない。

## 進捗

- [x] 2026-09-09: start revision/branch
- [x] 2026-09-09: baseline docs-check
- [x] 2026-09-09: durable pair inventory
- [x] 2026-09-09: Tier/exceptions分類
- [x] 2026-09-09: 各doc purpose/reader question記録
- [x] 2026-09-09: duplicate/stale/misplaced content特定
- [x] 2026-09-09: Tier1/2/high-risk doc content map
- [x] 2026-09-09: bilingual policy更新
- [x] 2026-09-09: README
- [x] 2026-09-09: docs index
- [x] 2026-09-09: roadmap
- [x] 2026-09-09: Architecture
- [x] 2026-09-09: AGENTS
- [x] 2026-09-09: PLANS
- [x] 2026-09-09: QUALITY
- [x] 2026-09-09: RELIABILITY
- [x] 2026-09-09: SECURITY
- [x] 2026-09-09: PORTABILITY
- [x] 2026-09-09: current product specs全件
- [x] 2026-09-09: current design docs全件
- [x] 2026-09-09: current ADR全件
- [x] 2026-09-09: active ExecPlans
- [x] 2026-09-09: current audit docs
- [x] 2026-09-09: duplicate authority consolidation
- [x] 2026-09-09: status fact verification
- [x] 2026-09-09: English link review
- [x] 2026-09-09: Japanese semantic rewrite
- [x] 2026-09-09: source_sha refresh after review
- [x] 2026-09-09: docs-check
- [x] 2026-09-09: architecture/metadata/generated checks
- [x] 2026-09-09: independent English review
- [x] 2026-09-09: independent Japanese review
- [x] 2026-09-09: bilingual parity review
- [x] 2026-09-09: final repository harness
- [ ] retrospective
- [ ] completed移動

## 想定外の発見

- Browserのmanifest項目とprocess利用は実装済みだったが、英日とも将来機能としていた。
  `internal/config/browser.go`と既存のconfigテストで確認した。Android UIの既存コマンドも
  実装済みと明記し、MVPの除外事項は当初の範囲であると説明した。
- lease作成時は予約より先に計画を作る（`internal/app/lifecycle.go`の`BuildPlan`と
  `Store.Reserve`）。設計の概念例は実際のAPI宣言とは区別した。TTLは現在、組み込み方針で
  決まり、利用者が編集するhost方針ファイルは未実装である。
- CASのキーは小文字のdigestで、保存先は`<cas-root>/<digest>/data`だった。
  古い`sha256/<digest>`表記を`internal/blobstore/store.go`の`ValidDigest`、`Open`と照合した。
- reconcileはCompose以外も含む、記録済みのruntime providerを観測する。
  既存のAndroidとprocessの回帰テストもこの境界を裏付ける。
- 最終のローカルBrowser CI証拠は`440082b`の`34320519250`だった。
  `34316121411`は`53fe81a`の実行である。元の証拠は完了済みmulti-host Planに保持し、
  roadmapでは詳細を重複掲載せず、その記録へ案内した。
- 元のリポジトリが不変という保証は広すぎた。元checkoutのファイルは変更しないが、
  Git worktree登録はmetadataを変更する。Podmanの到達性検査もTCPに限定し、UDPは
  engineによる観測と区別した。
- 日本語レビューで実行中の操作という条件の脱落、名詞の連結、native Emulatorと実機の
  曖昧な区別、誤訳したコードフェンスの言語タグを検出し、hash確定前に修正した。
- 編集途中の検査では一時的な古いhash、消えたindexへの導線、翻訳例外の過去Planに対する
  存在しない日本語リンク、READMEの翻訳metadata不足、空白を検出した。
  検査や例外規則を変えずに修正した。

## 判断の記録

- 対象をroadmapからdurable current docs全体へ拡大。
  理由: 可読性問題がsystemic。
- English canonicalは維持するがEnglish自体をreader-first編集。
  理由: dense sourceを忠実翻訳しても問題が複製される。
- Japaneseはsemantic fidelityを保ちstructural fidelityを要求しない。
  理由: 自然な日本語と既存hash modelに整合。
- reader question/capabilityでgroup。
  理由: implementation chronologyより読み手に有用。
- statusをImplemented/Active/Deferred/Unsupported/Openに明示。
- entry docsではprogressive disclosure。
- completed/historical ExecPlanはwholesale rewriteしない。
- high-risk rewrite前にcontent-preservation map。
- readabilityのためcontractを弱めない。

すべて日付/担当: 2026-09-09 / maintainers.


### Milestone 1の判断（2026-09-09 / 実装担当）

後掲の12項目は、同じ順で次のように決定した。

1. 棚卸し、情報の移動計画、用語の判断は本Planに残す。別の監査ファイルは記録が重複するため作らない。
2. 監査本文は過去の証拠として13組とも保持し、現行の案内であるindexだけを整理する。
3. multi-hostは開始revisionにマージ済みで、並行する実装作業はない。専用の文書ブランチで進める。
4. 用語集は増やさない。厳密な識別子は契約、文章の選び方は日英方針に置き、管理先を増やさない。
5. 製品仕様には目的・設定・挙動・失敗・制限の構成を推奨する。見出しの完全な一致は強制しない。
6. 設計には背景・境界・仕組み・復旧・証拠を推奨し、ADRは判断・背景・結果を簡潔に保つ。
7. 編集規則はbilingual-documentationに集約する。QUALITYはそこへ案内し、検証方法を説明する。
8. 言語別の独立レビューと意味の一致の確認を使う。推測的な文章lintは追加せず、既存検査を維持する。
9. 入口での短い要約は許容し、詳細は現行の契約、過去の証拠は完了Planへ案内する。
   手順は移動先にそろえてから重複を削る。
10. 自然で定着した日本語の専門用語を選び、曖昧でなければ主語を省略できる。
    コマンド、protocol用語、識別子は厳密に保持する。
11. 事実の照合と編集レビューを行ったペアだけ`last_verified`を更新する。
    native runtimeを新たに実行した証拠とは扱わない。
12. 現行仕様と設計へ案内する。古い入口が完了Planだったことだけを理由に新たな要約文書は作らず、
    過去の証拠を保持する。

## 成果と振り返り

未完了。

完了時にstart/final revision、reviewed/restructured/unchanged pairs、duplicate consolidation、
stale fact、authority move、reader-first policy、terminology、英語review、日本語review、
parity review、docs-check、remaining debtを記録。

future docs向けrecommended structureもまとめる。

## 背景と構成

README/AGENTS/ARCHITECTURE/docs index/roadmap/PLANS/QUALITY/RELIABILITY/SECURITY/
PORTABILITY/bilingual policy/current product/design/ADR/active ExecPlan/translation
exception/repoctl docs-checkを読む。

existing bilingual policyの「Japaneseはmaintained artifact」「hashはsemantic qualityを
検証しない」は維持する。

## 作業計画

### Milestone 1 — inventory/role map

全pairをpath/tier/audience/question/authority/duplicate/rewrite/riskでmatrix化。

generated/historical exception明示。

### Milestone 2 — editorial/bilingual policy

semantic-not-structural translationをpolicy化。

必要ならQUALITYにも小さくreader-first ruleを追加。
noiseの多いlinterは追加しない。

### Milestone 3 — entry points

README -> index -> roadmap -> Architecture -> AGENTS。

各pairごとにEnglish/fact/Japanese/hash/docs-checkまで完結。

### Milestone 4 — policy

PLANS/QUALITY/RELIABILITY/SECURITY/PORTABILITY。

requirement/rationale/enforcement/evidence/limitationを明確化。

### Milestone 5 — product specs

current spec全件。
user-visible guaranteeがdesign/completed planだけに埋もれていないか確認。

### Milestone 6 — design/ADR

architecture/decision/authority/lifecycle/failure/tradeoffを整理。
product contract duplicateはsummary+linkへ。

### Milestone 7 — active plans/audits

current operational readabilityのみ改善。
history/evidenceは保持。

### Milestone 8 — cross-doc dedup/authority

重要factごとにcurrent authorityを一つ特定。
entry docsでは意図的summary repetitionは許容。

### Milestone 9 — independent language reviews

English reader-first
Japanese reader-first
semantic parity

を別々に実施。

### Milestone 10 — final verification

`repoctl docs-check`, `repoctl check`。
hash/pair/link/history/status/safety constraint確認。

## 具体的な手順

1. master更新/revision
2. branch
3. 英日active plan
4. baseline
5. inventory
6. exception
7. bilingual policy
8. README
9. index
10. roadmap
11. Architecture
12. AGENTS
13. PLANS
14. QUALITY
15. RELIABILITY
16. SECURITY
17. PORTABILITY
18. product specs
19. design/ADR
20. active plans
21. audits
22. dedup/authority
23. fact/status/link verify
24. hashes
25. English review
26. Japanese review
27. parity review
28. final check
29. evidence/retrospective
30. completed

## 検証と受け入れ

| ID | 必須動作 | 証拠 |
| --- | --- | --- |
| D1 | 全durable human-facing pairをinventoryしscope/exception分類 | 個別棚卸し、日本語の分類表、履歴・生成物28件の除外表を記録。現行46組と本Planを扱い、既存の翻訳例外6件を維持。 |
| D2 | in-scope doc全件にprimary purpose/reader question | 棚卸しに読者の質問と役割を記録。現行文書の冒頭と次に読む文書への案内を確認。 |
| D3 | Tier1をgrammarだけでなくinformation architecture review | 入口6組とAGENTSの構成を確認。README、Architecture、roadmap、indexを読者の質問に沿って整理。 |
| D4 | roadmapがcapability group/statusを明示 | roadmapで実装済み、先送り、未決定、証拠の制限を分離。 |
| D5 | READMEがfirst useとdeeper docsへのpathを提供 | READMEは導入、前提条件、最小のlease操作の後に機能別の仕様へ案内。 |
| D6 | Architectureがsystem/boundary map | Architectureにレイヤーの責務表、local lifecycle、runtime・application・control-planeの境界を記載。 |
| D7 | AGENTSがconcise operating guideを維持 | AGENTSは105行。現行契約と編集方針への案内を補い、既存の運用規則を維持。 |
| D8 | policy docsがrequirement/rationale/enforcement/limitationを区別 | 方針7組で要件・理由・検査・証拠を区別し、独立レビューを通過。 |
| D9 | current product spec全件contract clarity review | 製品仕様12組を確認。Browserのmanifest、Android UIの状態、MVPの当初の範囲を実装と照合。 |
| D10 | current design/ADR全件architecture/decision clarity review | 設計13組とADR8組を確認。API概念例、TTL、CAS、local GCの範囲を正し、採用済み判断を維持。 |
| D11 | active ExecPlanのrequired history/evidence保持 | 元のPlan要件とmaintainer判断を保持。日付付きで発見、対応、証拠を追記。 |
| D12 | completed/historical planをstyle目的でwholesale rewriteしない | 既存の完了Planの英語26件と対応する日本語版は開始revisionから変更なし。 |
| D13 | generated/reference exception保持 | 生成schema、元の引き継ぎ記録、例外registryに変更なし。引き継ぎ記録のSHA-256は後掲。 |
| D14 | high-risk rewriteにcontent map、normative情報loss無し | 編集前の移動計画を記録。READMEの手順を先に仕様へ移し、必須条件の保持をレビュー。 |
| D15 | implemented/deferred/unsupported/openをcurrent authorityでverify | Browser、config、lifecycle、blobstoreの実装と既存テストを確認し、roadmapの状態を明記。 |
| D16 | orientation docsでprogressive disclosure | README、index、roadmapから現行の機能契約と過去の受け入れ証拠へ案内。 |
| D17 | repeated current guidanceにcurrent authority | remote起動とAndroid helperの手順は仕様へ集約。編集規則は日英方針に置き、QUALITYから案内。 |
| D18 | English noun pile/mixed-status/list sentence review | 密な機能列挙と状態の混在を整理し、独立した英語レビューを通過。 |
| D19 | Japaneseをnatural technical Japaneseとしてreview | 名詞の連結とAndroid・processの密な段落を修正し、日本語の独立レビューを通過。 |
| D20 | Japanese reorder可、meaning完全保持 | 日本語の段落と表を独立して整理。別途、意味の一致を確認し、未解決の差異なし。 |
| D21 | bilingual policyにsemantic-not-structural translation明記 | 日英方針に意味の保持、言語別レビュー、意味の一致の確認、異なる構成の許容を明記。 |
| D22 | identifiers/commands/protocol precision保持 | 仕様のコードフェンスを開始時点と比較。コマンド、識別子、必須条件を保持し、タグの誤訳を修正。 |
| D23 | safety/ownership/cleanup/trust constraint非弱体化 | 所有権、cleanup、操作fence、TCPだけの検査を確認。元checkoutについての過大な保証を正確に限定。 |
| D24 | deferred/unsupportedをimplementedと誤表現しない | roadmapと契約で未対応・先送りの範囲とnative・実機の証拠の制限を維持。 |
| D25 | stale factを英日でcorrectしauthority記録 | 想定外の発見に実装パス、API名、元のCI証拠と日英の修正理由を記録。 |
| D26 | independent English reader-first review | 後掲の独立レビュー証拠に入口6、仕様12、方針7、設計13、ADR8組の担当分離を記録。 |
| D27 | independent Japanese reader-first review | 同じ独立担当が、日本語単独での読みやすさを別に確認し、指摘を報告。 |
| D28 | bilingual semantic parity review | 意味の一致を別途確認。実行中という条件、native環境の表現、方針の最終追記も再確認。 |
| D29 | affected source_sha256 fresh | 意味の確認と対象ペアのhash更新後、最終repoctl check内のdocs検査が成功。 |
| D30 | repoctl docs-check pass | 開始時と途中のdocs-checkが成功し、最終の全体checkでもdocs検査が成功。 |
| D31 | full repository documentation/harness checks pass | 2026-09-09の最終go run ./tools/repoctl checkは終了コード0。整形、単体テスト、vet、文書、生成物、構成の検査が成功。 |
| D32 | retrospectiveにchanged/unchanged/authority/stale/debt記録 | Pending |

green hashだけではsemantic acceptanceではない。

## 冪等性と復旧

documentation-onlyが原則。fact verificationでproduct defectを見つけても本Planでcodeを
都合よく変更しない。

coherent doc group単位でEnglish -> fact -> Japanese -> hash -> docs-check -> commit。

English再変更時はJapaneseをsemantic reviewしてからhash更新。
repository全体hashだけmechanical refresh禁止。

rewriteが悪化したpairは独立revert可能にする。

## 成果物と注記

working artifact候補:

```text
docs/audits/reader-first-documentation/
  inventory.md/.ja.md
  authority-map.md/.ja.md
  terminology.md/.ja.md
```

durableに残すかはMilestone1で決定。

## インターフェースと依存

production dependency無し。

Markdown/current code/tests/repoctl/harness/bilingual hash mechanismを使用。

external `reader-first-editor` skillは利用可能ならediting aidとして使えるが必須依存に
しない。repository fact/policy/bilingual reviewがauthority。

external translation serviceを必須にしない。

## Milestone 1の検討項目（判断の記録で解決済み）

1. reader-first audit working artifactをdurableに残すか
2. current audit reportの全文prose review範囲
3. active multi-host docsとの並行編集/rebase方法
4. glossaryを作るか
5. product spec common skeletonをformalizeするか
6. design doc common skeletonをformalizeするか
7. reader-first policyをbilingual docだけかQUALITYにも置くか
8. low-noise structural docs-checkを追加できるか
9. README/roadmap/Architecture intentional duplicationの許容量
10. Japanese termごとのtranslated/English preference
11. substantial editorial reviewでlast_verifiedを更新するか
12. completed planだけがcurrent authorityの箇所にcurrent summary docを作るか


## 実行時の棚卸しと情報の移動計画（2026-09-09）

開始revisionは`084da57de177c0a09bc3cb61ae99faff8bd79a94`。開始時の未追跡ファイルは
ユーザーが用意した日英の本Planだけで、実装差分はなかった。`master`は最新で、
`docs/reader-first-documentation-restructure`へ切り替えた。baselineのdocs-checkと
check全体が成功した。配下固有のAGENTSはない。reader-first-editor Skillはこのセッションで
利用できないため、本Planの編集手順を適用する。

以下は英語文書と対応する日本語版の組を単位とする。現行の文書群は全文を確認する。
完了済みExecPlanと生成文書は履歴・生成物として書き換えない。監査の生の証拠も履歴として保持し、
indexだけを現行の案内として整理する。検証日だけで内容が最新だと証明されたとは扱わない。

編集前の移動計画:

- README: 密な機能紹介と追記されたruntime手順を、製品の役割、信頼境界、導入、前提条件、
  共通lease操作、機能別の案内、状態とcleanup、開発者向け案内へ整理する。詳細コマンドと
  受け入れ証拠は機能の仕様・品質文書・完了Planに残す。移動先にない手順は移してから元を削る。
- ARCHITECTURE: local処理、依存規則、共通不変条件、runtime・application・観測・remote・releaseの
  境界を分ける。package名、所有権記録、安全条件を維持し、MVPだけが全挙動を定義するという
  古い読み取り方を現行仕様への案内で修正する。
- docs/index: ファイル列挙を読者の質問別の案内へ変える。従来の入口を残し、現行規則・契約・
  仕組み・判断・履歴証拠を区別する。roadmapは実装済み範囲と未決定事項を分け、機能ごとの
  未検証環境と全除外事項を残す。
- 運用規則: 要件・理由・検査・証拠を分ける。PLANSは実行と完了条件を整理する。
  日英方針には意味の一致、英語からの執筆、言語別レビューと意味確認を明記し、見出し・段落の
  一致は義務にしない。
- 製品仕様: 目的と実装状態、設定、挙動、失敗とprivacy、制約と証拠、次の参照先へ整理する。
  数値・resource・安全条件と既存anchorを保持する。browserのmanifestとprocess利用、UIコマンドの
  実装済み状態をコードで確認し、古い記述を修正する。MVPの除外は当初の範囲として明示する。
- 設計・ADR: 境界、identityとlifecycle、不変条件と復旧、検証と次の参照先を分ける。
  概念的なAPI例を実interfaceと区別し、ADRの過去の判断・accepted状態と未検証環境を保持する。

棚卸しの全pair、Tier、役割、編集対象・維持理由は英語版の直前の表に対応する。
この日本語版でも実行結果の対象文書を以下に列挙し、対象の省略を防ぐ。
- `README.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `AGENTS.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `ARCHITECTURE.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/PLANS.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/PORTABILITY.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/QUALITY.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/RELIABILITY.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/SECURITY.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/adr/0001-use-go.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/adr/0002-use-sqlite.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/adr/0003-compose-first-runtime.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/adr/0004-repository-native-harness.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/adr/0005-separate-flutter-applications.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/adr/0006-single-authority-multi-host.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/adr/0007-native-execution-boundaries.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/adr/index.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/audits/repository-correctness/current-compose-release.ja.md`: 監査の履歴証拠を保持。
- `docs/audits/repository-correctness/current-control-plane.ja.md`: 監査の履歴証拠を保持。
- `docs/audits/repository-correctness/current-mobile.ja.md`: 監査の履歴証拠を保持。
- `docs/audits/repository-correctness/current-process-browser.ja.md`: 監査の履歴証拠を保持。
- `docs/audits/repository-correctness/documentation.ja.md`: 監査の履歴証拠を保持。
- `docs/audits/repository-correctness/findings.ja.md`: 監査の履歴証拠を保持。
- `docs/audits/repository-correctness/historical-corpus.ja.md`: 監査の履歴証拠を保持。
- `docs/audits/repository-correctness/history-compose-release.ja.md`: 監査の履歴証拠を保持。
- `docs/audits/repository-correctness/history-mobile.ja.md`: 監査の履歴証拠を保持。
- `docs/audits/repository-correctness/history-process-browser.ja.md`: 監査の履歴証拠を保持。
- `docs/audits/repository-correctness/index.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/audits/repository-correctness/matrix.ja.md`: 監査の履歴証拠を保持。
- `docs/audits/repository-correctness/supplemental-browser.ja.md`: 監査の履歴証拠を保持。
- `docs/audits/repository-correctness/supplemental-cli.ja.md`: 監査の履歴証拠を保持。
- `docs/design-docs/android-emulator.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/design-docs/android-ui-observer.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/design-docs/bilingual-documentation.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/design-docs/browser-cdp-automation.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/design-docs/compose-providers.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/design-docs/compose-runtime.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/design-docs/core-beliefs.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/design-docs/flutter-android-runtime.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/design-docs/index.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/design-docs/lease-control-plane.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/design-docs/multi-host-control-plane.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/design-docs/persistent-process-runtime.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/design-docs/reconciliation-and-gc.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/design-docs/standalone-distribution.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/index.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/product-specs/agent-env-mvp.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/product-specs/android-emulator.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/product-specs/android-ui-observer.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/product-specs/browser-cdp-automation.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/product-specs/cli-contract.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/product-specs/compose-providers.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/product-specs/flutter-android-runtime.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/product-specs/index.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/product-specs/manifest-v1.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/product-specs/multi-host-control-plane.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/product-specs/persistent-process-runtime.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/product-specs/standalone-distribution.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/references/index.ja.md`: 上記の文書群別の計画で全文を整理・検証。
- `docs/roadmap.ja.md`: 上記の文書群別の計画で全文を整理・検証。

編集前の移動計画を基準に、担当からの詳細と独立レビューの採否を後続の実行記録に残す。
各現行文書の次の参照先は実在する相対リンクで示す。


### 実行記録と判断

2026-09-09: 入口文書の移動計画を実装した。remoteの起動とUI companionの準備は、
製品仕様に詳細が揃ったことを確認してからREADMEの重複を取り除いた。監査indexの案内を整理し、
履歴の証拠表とレポート本文は維持した。初期handoffのarchiveはバイト列を維持する既存例外であり、
存在しない日本語archiveを翻訳不足とは扱わない。MVP・Androidの古い完了Planにも既存の明示例外が
あるため、日本語の案内から英語の記録へリンクする。完了Planや生成物の編集対象外という分類で
既存の翻訳例外を拡大しない。

事実の修正: browser manifest対応とprocess利用、Android UIコマンドは実装済み。
MVPだけを現在の全範囲と読める文も修正した。QUALITYの最終local Browser runは`440082b`の
`34320519250`である。古い`34316121411`は`53fe81a`での実行であり、完了multi-host Planに記録がある。
現在の主張に合わせて実装や履歴証拠のバイト列を変更していない。

独立レビューは執筆者と分ける。規則担当が入口と製品文書、製品担当が規則、設計担当が設計完了後に
入口の案内と意味の保持を確認し、統合担当が設計・ADRを確認する。各報告は英語の使いやすさ、
日本語だけでの読みやすさ、日英の意味一致を分ける。自己確認を独立レビューには数えない。

2026-09-09の中間検証: 現行の文書群の構成整理後、`go run ./tools/repoctl check`が再び成功した
（unit/vet/文書/生成物/architecture）。入口の独立レビューで、元リポジトリを変更しないという
過剰な約束（Git worktreeの登録はmetadataを更新する）、日本語の実行中条件の抜け、現行仕様の
誤った言語リンク、native Emulatorと物理deviceの曖昧さ、operation fenceと証拠の文言を修正した。
reconcileの構成説明はComposeだけでなく記録済みの各runtimeにした。既存のapp Android/process
reconcile回帰テストで確認できる挙動である。製品レビューでは、既存のPodman endpoint要約をTCPに
限定し、UDPは観測だけでTCP/UDP probeをしない条件を維持した。日本語の誤字とcode fenceの言語tagも直した。

途中の検査では、明示例外のある古い日本語Planへのリンク、編集中のhash、一時的に失ったlocal index経路、
行末空白、余分な末尾空行を検出した。編集内容を修正し、validatorや不正fixtureの検証条件は緩めていない。

### 棚卸しの分類と読者の目的

以下は前掲の個別パスに適用する分類です。英語版の表を読まなくても、対象・役割・
編集の優先度を確認できます。各行は対応する日本語版も含みます。

| 対象 | Tier | 読者の質問・文書の役割 | 優先度と扱い |
| --- | --- | --- | --- |
| README、AGENTS、ARCHITECTURE、docs/index、docs/roadmap | 1 | どこから始めるか、何が実装済みか、どこに責務があるか。現行の入口と構成案内 | 高。情報の順序と機能別の案内を整理 |
| PLANS、QUALITY、RELIABILITY、SECURITY、PORTABILITY、design-docs/bilingual-documentation | 2 | どの規則を守り、どう検証するか。現行方針 | 高。要件・理由・検査・制限を分ける |
| product-specsの本文 | 3 | 各機能は何を保証するか。現行の利用契約 | 高。設定・挙動・失敗・制限を確認 |
| product-specs/index | 3 | どの仕様を読むか。現行の案内 | 中。仕様への導線を維持 |
| design-docsの本文（bilingual-documentationを除く） | 4 | なぜこの構成で、どのように契約を実現するか。現行設計 | 中〜高。責務・仕組み・復旧・証拠を保持 |
| adrの本文 | 4 | なぜその判断を採用したか。採用済みの判断記録 | 中〜高。判断と理由を変えず、必要な案内だけを補う |
| design-docs/index、adr/index | 4 | 詳細設計や判断の記録はどこか。現行の案内 | 中。リンク先を維持 |
| 本Planの日英ペア | 5 | 今回の変更をどう実行・検証したか。完了までの実行基準 | 高。要件・判断・失敗・証拠を保持して完了時に移動 |
| audits/repository-correctness/index、references/index | 6 | 監査証拠や過去の参照資料はどこか。現行の案内 | 中。案内だけを整理 |
| audits/repository-correctnessの各報告本文 | 6 | 監査時点で何が観測されたか。過去の証拠 | 文体変更の対象外。証拠を保持 |

MVP仕様だけは当初の要件範囲も保持します。他の製品仕様にMVP限定という条件はありません。
各indexは案内であり、機能の契約や採用済み判断そのものではありません。

### 履歴・生成物の個別棚卸し

文体変更の対象外であることと、翻訳の例外であることは別です。例外registryは変更しません。
次の28件の英語文書と、既存の日本語版を保持します。

| 英語パス | 翻訳の扱い | 文体変更の対象外とする理由 |
| --- | --- | --- |
| `docs/exec-plans/completed/agent-env-mvp.md` | 既存の明示的な例外 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/android-emulator-lease.md` | 既存の明示的な例外 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/android-emulator-review-2.md` | 既存の明示的な例外 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/android-emulator-review.md` | 既存の明示的な例外 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/android-ui-observer.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/bilingual-documentation-review.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/bilingual-documentation.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/browser-cdp-automation.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/compose-provider-podman-review.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/compose-provider-podman.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/flutter-android-review.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/flutter-android-runtime.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/multi-host-control-plane.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/multi-host-review-followup.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/multi-host-review-round-two.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/persistent-process-runtime.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/process-destroy-preview-review.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/repository-correctness-audit.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/repository-correctness-review.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/standalone-distribution-review.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/standalone-distribution.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/standalone-release-filter-review.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/standalone-release-finalization.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/standalone-release-review.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/standalone-verify-review.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/exec-plans/completed/worker-android-capacity-review.md` | 日英ペアを保持 | 完了時点の実行証拠 |
| `docs/generated/db-schema.md` | 既存の明示的な例外 | 生成schema |
| `docs/references/handoffs/CHATGPT_HANDOFF_30.md` | 既存の明示的な例外 | 元の引き継ぎ記録 |

現行文書は46組を編集し、本Planを含めて47組を扱いました。前掲の監査報告本文と、この表の対象は変更していません。

### 最終の独立レビュー証拠（2026-09-09）

現行46組について、英語の読みやすさ、日本語単独での自然さ、日英の意味の一致を別々に確認した。
方針文書の執筆担当が入口6組と製品仕様12組を、製品仕様の執筆担当が方針7組を、
全体編集担当が別担当の執筆した設計13組とADR8組をレビューした。自己点検を独立レビューとは
数えていない。主語の省略と定着した日本語の用語に関する方針の最終追記も、別担当が日英と
意味の一致を確認し、問題がないと判断した。

製品仕様のコードフェンスを開始revisionと比較し、意図したCLIブロックの分割・移動を除いて
内容を保持した。方針の要件・コマンド・識別子を維持し、AGENTSは105行に収まる。
設計の最終レビューではTTLの設定可能性とlocal/remote GCの範囲を正し、概念上のAPI例を区別し、
Androidとprocessの密な日本語段落を整理した。具体的な未解決の指摘はない。
Planの棚卸し分類も執筆とは別の担当が確認した。

### 検証記録（2026-09-09）

- 開始時と途中の`go run ./tools/repoctl docs-check`は成功。
- 開始時、途中、最終の`go run ./tools/repoctl check`は終了コード0で成功。
  最終検査はformat-check、`go test ./...`、`go vet ./...`、docs-check、generated-check、
  arch-checkを含む。最終の単体テストは有効なGoのキャッシュを利用しており、
  native runtimeや実機を新たに実行した証拠ではない。
- `git diff --check`は成功。`git diff --name-only -- ':!*.md'`は空。
- 元の引き継ぎ記録の変更前後のSHA-256は同一:
  `3f72a24b1767009cd66a0664ee62ba3359d9518599fb27d3d80fa26c7bfa864a`。
- ソース、テスト、翻訳例外、生成物、過去の報告本文は変更していない。
  Windows、macOS、native環境の検証範囲が増えたとは扱わない。

Milestone commit `b733c73`に日英の編集方針、方針文書7組、本実行記録を含めた。
次のcommitは入口と製品仕様をまとめ、READMEの手順を同じ変更内で移す。

Milestone commit `d8b8b99`に入口・案内6組と製品仕様12組を含め、READMEの手順も同時に移した。
設計・ADRのcommitでは採用済み判断を保持し、仕組みと境界を明確にする。
