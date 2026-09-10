---
translation_of: docs/exec-plans/active/test-architecture-evidence-governance.md
source_sha256: c463b3329b6e167a705586e2cd0ee0b709e39d04b23e87af42d08a123480c3ce
status: active
plan_id: EP-QUAL-001
plan_type: implementation
priority: 20
merge_policy: guarded
base_branch: master
branch: feat/ep-qual-001
workstreams:
  - testing
  - repoctl
  - documentation
owner: maintainers
last_verified: 2026-09-10
---

# テストアーキテクチャと検証証拠の全体監査

[English](test-architecture-evidence-governance.md)

このExecPlanはliving documentである。`docs/PLANS.md`に従って更新する。

Plan ID: `EP-QUAL-001`

想定ブランチ: `feat/ep-qual-001`

開始revision: PR #14の`master`へのmerge commit `cbd84ed1d2ef4456b8a95db215461bd3d726fe59`。`EP-OPS-001`のfeature branchや未mergeのstacked revisionから実装を開始しない。

## 目的 / 全体像

リポジトリのテストを、production codeを検証する手段としてだけでなく、それ自体が正しさを要求される一つのシステムとして監査する。

直接の契機はBrowser/CDPの回帰テストである。client側の待機がcancelで終了した後もmock server側のcallbackは実行を継続できたが、テスト本体はcallbackが所有する状態を同期なしで読み取っていた。

つまり、production側のhidden ordering assumptionを検出するために追加した回帰テスト自身が、別のhidden ordering assumptionを持っていた。

今回の監査では、単に、

    production behaviorにテストがあるか

だけを問わない。

次も監査対象とする。

    oracleは本当に正しいか
    意図したfailure scheduleを実際に通しているか
    fixture自身はconcurrency-safeか
    cleanupはcloseだけでなくcompletionを保証しているか
    意図した処理へ到達しなくてもPASSできないか
    報告された検証結果は実際には何を証明しているか

目標とする検証の流れは次である。

    production invariant
        -> 明示的なtest oracle
        -> 信頼できるfixture
        -> 必要なら決定論的failure schedule
        -> fail-before
        -> repair
        -> pass-after
        -> 証拠能力を正しく分類した補助検証

反復PASS、CI rerun、race detector、複数OSでのPASSは引き続き有用な観測結果である。ただし、回数を積み上げても、そのテストが強制していないinvariantの証明へ格上げしてはならない。

このPlanは、`EP-OPS-001`で導入する新ExecPlan lifecycleを、次の通常作業でforward live dogfoodする最初のPlanとしても利用する。

## Scope

対象:

- リポジトリ所有のGo testと共有test helper;
- concurrencyを持つfixture/helper;
- goroutine/process/socket/server/clientのtest lifecycle;
- cancellation、deadline、polling、cleanup;
- test goroutine間で共有するmutable state;
- timing、sleep、小さなtimeoutへの依存;
- mock/fakeの妥当性と実runtimeとの差;
- deterministic failure injection;
- positive/negative control;
- errorだけを見るassertion;
- callback/effectへ本当に到達したかというoracle;
- test oracleの独立性;
- boundary/composition coverage;
- fixtureのownershipとcleanup proof;
- race-sensitive test;
- `-count=N`、CI rerun、native反復実行の証拠能力;
- 証拠の分類と報告ルール;
- 必要な`AGENTS.md`、`QUALITY.md`、Plan policy等の改善;
- 誤検出が少なく安定して強制できるpreventive control;
- 監査で確認したtest architecture defectの修正;
- 監査と修正への独立review。

優先対象:

- `internal/browser/cdp`;
- `internal/execx`;
- application cancellation/readiness;
- controller/client/workerおよびmulti-host fixture;
- persistent process fixture;
- Android Emulator / Android UI helper;
- Compose integration fixture;
- release/filesystem mutation・failure injection test。

対象外:

- すべてのmockをreal integrationへ置き換えること;
- integration testを常にunit testより強い証拠とみなすこと;
- `time.Sleep`、deadline、polling、atomicの一律禁止;
- failureを消すためだけのtimeout延長;
- 全テストのproperty-based testing化;
- 具体的な不変条件で必要性を説明できない汎用model checkerの構築;
- 過去のcompleted ExecPlanを書き換えて新しい証拠分類へ合わせること;
- scheduler interleavingの完全探索を主張すること;
- 既存のproduction correctness、portability、native verification要件を弱めること。

## Progress

- [x] PR #14のmerge revision `cbd84ed1d2ef4456b8a95db215461bd3d726fe59`を記録し、`feat/ep-qual-001`を作成した。
- [x] 変更前baseline harness/full race/関連native結果を失敗も含め記録した（2026-09-10）。
- [x] 修正前に監査方法とevidence taxonomyを固定する。
- [x] test helperとconcurrency-bearing fixtureを棚卸しする。
- [x] Browser/CDP、`mockBrowser`、`eventBrowser`を直接監査する。
- [x] cancellation/deadline系testをリポジトリ横断で監査する。
- [x] process、worker/controller、Android、integration fixtureのlifecycleを監査する。
- [x] false-positive pathとunreached effectを含むoracle監査を行う。
- [x] 全findingへACCEPT / REJECT / DEFERと理由を記録する。
- [x] ACCEPT findingをfail-before付きで修正する。
- [x] concurrency/order findingへ可能な限りdeterministic scheduleを導入する。
- [x] 必要なhelperに明示的lifecycle synchronizationを追加する。
- [x] evidence classとevidence laundering防止ルールを定義する。
- [x] Agent指示とquality documentationを更新する。
- [x] 妥当な範囲だけmechanical preventive controlを追加する。
- [x] audit、disposition、修正を独立reviewする。
- [x] focused deterministic regressionを最終実行する。
- [x] full race/harnessの最終実行が成功した（2026-09-10）。
- [ ] 必要なWindows/macOS/Linux native validationを完了する。
- [x] English/Japanese独立reviewとsemantic parity reviewを完了する。
- [ ] merge後にretrospectiveを完成しarchiveする。






## Surprises & Discoveries

- baseline docs-checkで古いdraft metadataを検出し、開始時に現行schemaへ移行した。
- Windows guardianのpollは全観測errorを無視しており、runtime側へ伝播するsharing violationも隠せた。handle所有者は未特定。確認済み空Jobでのsharing競合だけを完了待ちとして扱う。
- 旧snapshot切り詰め負例は無関係な早期失敗注入でも成功した。強化したoracleは同じ注入を拒否し、期待した境界を未検証のまま数えない。

個別bugだけでなく、監査方法自体へ影響する発見を記録する。

特に次を残す。

- intended mutation/effectへ到達せずPASSするtest;
- implementationと同じ誤った前提を共有するoracle;
- resourceをcloseするが処理終了を待たないcleanup;
- assertionと並行実行されるcallback;
- data raceは消すが必要なorderingを作らないatomic;
- scheduling assumptionを隠すtimeout延長;
- real protocol/runtimeと重要な点で異なるmock;
- failure scheduleを強制していない大量反復;
- 証拠能力以上に強く報告されていたnative/CI結果;
- 同じ誤った前提を持つsibling fixture;
- static checker化すると誤検出の方が多くなるrule。

findingを十分記録する前に、その場で修正して消してはならない。

## Decision Log

- 2026-09-10 / maintainer承認済みの開始判断: 提供されたdraftをactiveへ移し、規定の`feat/ep-qual-001`を使用する。循環するEP-OPS-001の完了依存は外す。merge済み実装を基盤とし、未実施のHuman Validationは免除しない。

- 2026-09-10 / maintainers: testとfixture自身もcorrectness review対象とする。
- 2026-09-10 / maintainers: race detector PASSは観測したexecutionでraceが報告されなかった証拠であり、全scheduleを通した証明ではない。
- 2026-09-10 / maintainers: `-count=N`等の反復は、関連state/orderをtest自身が強制しない限りstability/flakiness evidenceとして扱う。
- 2026-09-10 / maintainers: 証拠は量によって格上げされない。weak evidenceを大量に積んでもdeterministic proofにはならない。
- 2026-09-10 / maintainers: concurrency defectではtiming probabilityより明示的happens-before/barrierを優先する。
- 2026-09-10 / maintainers: semantic requirementがcompletion/orderである場合、atomic化だけでは修正完了としない。
- 2026-09-10 / maintainers: timeout自体が破られたcontractでない限り、timeout延長をconcurrency repairとして認めない。
- 2026-09-10 / maintainers: findingはfailing lineだけで閉じず、violated invariant、escape、sibling exposure、preventive controlまで追う。
- 2026-09-10 / maintainers: completed planのhistorical evidenceは保持し、新しいevidence vocabularyを過去へ遡って書き換えない。
- 2026-09-10 / maintainers: `AGENTS.md`は簡潔に維持し、詳細なevidence semanticsはquality/design documentationとvalidatorへ置く。

## Outcomes & Retrospective

Q13までの実装は検証済みである。下記Q14のPR review対応は改めて最終検証する。merge/archiveの受け入れは未完了である。限定監査はbaselineの28 package directory・195 test fileを対象にQ01〜Q14を記録した。台帳の13項目を採用して修正し、未実証のstream/constructor失敗経路に関するQ09は理由を明記して保留した。所有fixtureの対照を加え、現在は199 test fileである。

旧oracleの誤成功を実証し、原因を区別する失敗段階とlifecycle完了を明示した。native証拠の限界も維持している。最初の修正ではnative CIがEOFのみを想定した移植性不備を検出し、race CIが一度だけ登録したhost情報の期限切れを検出した。deadline、controller TTL、移植性検査を緩めずに両方を修正した。独立reviewは新testの順序assertionにも2件の不足を指摘し、受け入れ前に修正した。local testとソースreviewは有用だがnative検証や強制故障対照の代用にはならなかった。

PR #15がcurrent-HEADで人間のreviewを受けmasterへmergeされるまでarchiveしない。信頼するbaseにgate policyがないためguarded/manual fallbackを維持し、自動mergeの許可とは解釈しない。

採用した台帳の12項目はtestのみの不備で、Q08はproductionの完了観測とtest oracleの両方に関わる。lifecycleと到達性の不足はCDPとapp/process helperに集中した。過去の反復成功はcallback完了やOS間のsocketの意味を証明しなかった。対象を絞った不変条件の回帰は実行可能な検査となるが、証拠分類、限定した同種箇所のreview、言語の意味はreviewで判断する。Q09、直接公開されないinline client readerのjoin、強制していない実行順序、未実行の環境は限界として明示し、網羅的な正しさとは主張しない。

この新PlanはEP-OPS-001の規定branch、安定Plan ID、commit/PR trailer、provenance、guarded gate確認を実装段階まで実運用した。信頼するpolicyがないためgateは適切にmanual fallbackを維持し、Human Validationも自動開始していない。forward merge/archiveの証拠はmaintainerがPR #15を完了するまで未完了である。

mergeまでは未完了とする。

最終retrospectiveでは最低限、次へ回答する。

- どのtest architecture defect classが見つかったか。
- どのsubsystemへ偏っていたか。
- production defectとtest-only defectはいくつあったか。
- false confidenceを作れるtestは存在したか。
- 過大評価されていたrepeat/CI evidenceはあったか。
- どのhelper contractを変更したか。
- どのruleを機械的に強制できたか。
- review disciplineとして残ったruleは何か。
- test assumption監査からproduction defectも見つかったか。
- probabilistic/unverifiedな領域は何が残ったか。
- `EP-OPS-001` lifecycleのforward live dogfoodとして成立したか。

単純なPASS回数を成果指標にしてはならない。

## Context and Orientation

実装前に読むもの:

- `AGENTS.md` / `AGENTS.ja.md`;
- `docs/PLANS.md` / `docs/PLANS.ja.md`;
- `docs/QUALITY.md` / `docs/QUALITY.ja.md`;
- repository-correctness audit matrix / findings / subsystem annex;
- completed Browser/CDP ExecPlan;
- completed persistent-process / multi-host ExecPlan;
- `internal/browser/cdp/*_test.go`;
- 他packageの共有test helper;
- `tools/repoctl`;
- GitHub Actions workflow。

既存repository correctness auditの分類:

    CONCURRENCY_GAP
    COMPOSITION_GAP
    FAILURE_INJECTION_GAP
    ORACLE_COUPLING
    NEGATIVE_FIXTURE_GAP
    BOUNDARY_GAP

は可能な限り再利用する。

必要性が確認できた場合だけ、例えば次を追加する。

    FIXTURE_LIFECYCLE_GAP
    NONDETERMINISTIC_ORACLE
    EVIDENCE_OVERCLAIM
    UNREACHED_EFFECT
    TIMING_PROXY
    CLEANUP_COMPLETION_GAP

## Plan of Work

### Milestone 0 — Baselineと監査基準固定

`EP-OPS-001` merge後に正確なbaseを記録し、専用branchを作成する。

変更前のrepoctl check、full race、関連native状態を残す。

既存failureを発見した場合、すぐrerunして消すのではなく最初のfailureを保存する。

修正前にaudit rubricとevidence taxonomyを記述し、後からgreen resultに都合よく評価基準を変えない。

### Milestone 1 — Evidence taxonomy

証拠を量ではなく「何を証明するか」で分類する。

最低限:

**Deterministic invariant evidence**

testが主張対象のstate/transition/order/boundary/failureを意図的に作る。

例:

    channel/barrierでorderingを固定
    fail-before -> repair -> pass-after
    exact boundary fixture
    persistence failure injection
    ownership mismatch injection

**Direct native/integration evidence**

実runtimeを明示した環境で観測した証拠。

platform/runtime semanticsには重要だが、rare scheduleのdeterministic proofとは限らない。

**Race/static/tooling evidence**

`go test -race`、`go vet`、architecture/docs validator等。

各toolが実際に検査するproperty以上の主張をしない。

**Stability evidence**

`-count=50`、CI rerun、複数CPU設定、native反復等。

flakiness発見には役立つが、強制していないscheduleを通した証明ではない。

**Compilation/structural evidence**

cross-build、generated comparison、schema validation等。

native実行証拠として扱わない。

authoritative guidanceへ次を同等の意味で明記する。

    Evidence must be classified by what it proves.
    Quantity does not upgrade evidence class.
    Repetition is not a substitute for deterministic reproduction.

concurrency bugについてstability evidenceを報告する場合は、「何を証明していないか」も記載する。

### Milestone 2 — Test helper / fixture棚卸し

次を起動・所有するhelperを探す。

- goroutine;
- HTTP/WebSocket server;
- client/read loop;
- subprocess/process tree;
- Docker/Podman resource;
- Android/ADB/Emulator process;
- controller/worker service;
- concurrent temporary DB;
- 読取中に意図的に変更するfile;
- timer/poll loop。

各helperについて最低限:

    owner
    起動resource
    concurrency
    completion signal
    cleanup
    cleanupが終了待ちをするか
    shared mutable state
    failure injection surface
    consumer tests
    hidden assumption

を記録する。

### Milestone 3 — Deterministic concurrency audit

最初にBrowser/CDPを処理する。

`mockBrowser`について、

    request受信
    callback開始
    callback終了
    response write
    client response受信またはconnection close
    handler終了
    server shutdown完了

のlifecycleを明示する。

client cancellationからserver callback completionを暗黙推論してはならない。

load-wait regressionではscheduler任せにせず、

    serverがevaluationへ入る
    -> testがentryを観測
    -> client cancel/deadline
    -> client operation return
    -> server workがまだactiveになり得ることを確認
    -> server workをrelease
    -> completionを待つ
    -> final assertion

という順序を決定論的に構築する。

channel、barrier、WaitGroup等、意図が読める同期手段を使う。

assertionがcompletionを必要とする場合、atomic化だけで終わらせない。

同じ`mockBrowser`を使うcancellation/deadline系testと`eventBrowser`を横断監査する。

その後、他fixtureにも同じlifecycle questionを適用する。

### Milestone 4 — Oracle / reachability audit

「別の理由でPASSできるtest」を探索する。

例えば:

- `err != nil`しか確認しない;
- intended callback/effect到達を確認しない;
- regression対象より前段で拒否される;
- fakeとproductionが同じ誤前提を共有する;
- positive/negative conditionがcoupleしている;
- boundary testが複数上限を同時に超える;
- cleanup完了前にassertする;
- cancellationとsubordinate work terminationを混同する。

必要な場所には、

    mutation callback executed
    destructive call remained uncalled
    expected request arrived
    intended boundary reached
    persistence step occurred

等のreachability oracleを追加する。

単にcounterを増やすのではなく、false-positive pathを区別する場合だけ追加する。

### Milestone 5 — Finding ledger

各findingへ:

    ID
    subsystem / fixture
    observed defect
    violated invariant
    earliest realistic detection stage
    escape reason
    production/test/both
    sibling exposure
    required evidence
    ACCEPT / REJECT / DEFER
    rationale
    preventive control

を記録する。

local fixだけでは完了せず、同じassumptionを持つsiblingを確認する。

### Milestone 6 — Repair

ACCEPT findingをcoherent sliceで修正する。

concurrency/order findingでは:

1. 可能ならdeterministic fail-beforeを作る。
2. lifecycle/order contractを修正する。
3. 同じreproducerでpass-afterを示す。
4. race detectorを実行する。
5. sibling testsを実行する。
6. repeatは補助stability evidenceとしてのみ使う。
7. 未強制scheduleを明記する。

testを見栄え良く統一するためだけの大規模refactorは行わない。

### Milestone 7 — Agent instruction / quality policy

将来のAgentがgreen-looking proxyへ最適化しないよう指示を改善する。

`AGENTS.md`には短い原則だけを入れる。

例えば:

    repair前にviolated invariantを特定する
    concurrencyではcancellationとcompletionを区別する
    repetitionをdeterministic proofとして報告しない
    timeout/atomicでordering defectを隠さない
    failed assumptionを共有するsiblingを監査する
    evidenceは何を証明するかで分類する

詳細なtaxonomyと例は`docs/QUALITY.md`または適切なdesign documentへ置く。

failure triage/reviewでは概念的に:

    observed failure
    -> root cause
    -> violated invariant
    -> earliest missed prevention
    -> sibling exposure
    -> deterministic reproducer
    -> repair
    -> regression
    -> preventive control
    -> evidence classification

まで要求する。

repeat countをdefault quality metricとして要求しない。

### Milestone 8 — Mechanical preventive control

false positiveが少なく安定して強制できるものだけ自動化する。

候補:

- helper-specific lifecycle assertion;
- regression callback到達確認;
- portableで信頼できる場合のleak detection;
- unsynchronized accessを困難にするtest helper API;
- concurrency-bearing helper packageへのfocused race test。

単なるgrep policyで「自動化した」ことにしない。

機械的強制に向かない重要ruleはreview obligationとして明示する。

### Milestone 9 — Independent review

read-only独立reviewで:

- audit completeness;
- finding disposition;
- fixture repair;
- deterministic regression;
- evidence taxonomy;
- Agent instruction;
- mechanical controls

を確認する。

reviewerは特に、

- intended pathを通さずPASSできるtest;
- race detectorを黙らせただけのsync;
- waitしないcleanup;
- evidence以上に強いclaim;
- Agentが表面的に満たせるreview rule;
- correctnessではなくcount/green statusへ最適化するcheck

を探す。

## Concrete Steps

kick時:

    git status --short
    git branch --show-current
    git rev-parse HEAD
    git rev-parse master

PR #14の実装が`master`へ入っていることを確認する。EP-OPS-001は未完了の受け入れ条件のためactiveを維持する。

branch:

    feat/ep-qual-001

paired plan:

    docs/exec-plans/active/test-architecture-evidence-governance.md
    docs/exec-plans/active/test-architecture-evidence-governance.ja.md

baseline:

    go run ./tools/repoctl doctor
    go run ./tools/repoctl check
    go test -race ./...

`EP-OPS-001`が導入した`repoctl plans`のcheck/provenanceも実行する。

focused auditでは可能な限り対象package/testを明示して`-race`を使う。

反復が有用な場合:

    go test -race ./internal/browser/cdp -run '<test>' -count=50

を実行してよいが、結果は、

    stability evidence only;
    relevant ordering is proven by <deterministic regression>

と分類する。

最終的に:

    go run ./tools/repoctl check
    go test -race ./...
    go run ./tools/repoctl plans check

と、変更したsubsystemに必要なnative/integration workflowを実行する。

workflow run ID、OS、toolchain/runtime version、結果を記録する。

## Validation and Acceptance

### Audit

- scope内のconcurrency-bearing test helperを棚卸し済み。
- Browser/CDP fixtureをlifecycle/oracleの両面で直接監査済み。
- cancellation/deadline testをリポジトリ横断で検索済み。
- process/controller/worker/Android/integration fixtureを監査済み。
- 全findingにdispositionと理由がある。
- ACCEPT findingはsibling exposureを確認済み。
- REJECT/DEFERも記録が残る。

### Regression

- triggering Browser/CDP testにunsynchronized callback-owned stateが残っていない。
- cancellation/server overlapをdeterministicにexerciseする。
- 必要なhelper cleanupが明示的completion semanticsを持つ。
- sibling `mockBrowser` testを監査済み。
- `eventBrowser`も同じ観点で監査済み。
- ACCEPT concurrency findingに可能な限りfail-before/pass-afterがある。
- race PASSはsupporting evidenceとして分類されている。

### Evidence governance

authoritative guidanceに最低限:

    Evidence must be classified by what it proves.
    Quantity does not upgrade evidence class.
    Repetition is not a substitute for deterministic reproduction.

を明記する。

さらに:

- deterministic invariant evidence;
- native/integration evidence;
- race/static/tooling evidence;
- stability/repetition evidence;
- compilation/structural evidence

を区別する。

cross-buildからnative behaviorを推論しない。

repeat countだけからunforced concurrency scheduleを推論しない。

### Agent behavior

Agent instructionは概念的に:

    failure
    -> root cause
    -> violated invariant
    -> earliest missed prevention
    -> sibling exposure
    -> deterministic reproducer
    -> repair
    -> regression
    -> preventive control
    -> evidence classification

を要求する。

次を抑止する。

- cosmetic timeout inflation;
- ordering未定義のままrace detectorだけ黙らせるatomic;
- CI rerun成功をroot cause消滅の証明として扱うこと;
- repeat countでclaimを格上げすること;
- failing lineだけ修正してsibling assumptionを見ないこと。

### Repository

- final `repoctl check` PASS;
- final full `go test -race ./...` PASS;
- 必要なintegration PASS;
- 必要なWindows/macOS/Linux native PASS;
- docs/translation check PASS;
- independent English review PASS;
- independent Japanese review PASS;
- semantic parity review PASS;
- technical reviewにunresolved blockerなし;
- final HEADに対するPR reviewが有効;
- Plan lifecycle/provenance gate PASS;
- `master`へmergeしてからarchive。

大量のrepeat PASSそのものはacceptance criterionにしない。

## Idempotence and Recovery

audit phaseはread-onlyで再実行可能とする。

finding inventoryを作る前にproduction/test codeを変更しない。isolated reproducer作成はfinding記録後に行う。

途中中断しても再開できるようfinding/dispositionをdurableに保存する。

repair後に新failureが出た場合:

1. failure evidenceを保存する。
2. 新invariantの発見かrepair不備か判断する。
3. ledgerを更新する。
4. green CIへ戻すためだけにregressionを弱めない。

unchanged codeのnative CI failureも最初のfailureを保持する。rerun成功はfailureを消さない。

`master`から遅れた場合は、履歴を書き換えない通常の手順で更新し、影響するbaselineの前提を再検証する。branch更新時にpublished historyをforce-pushしない。

開始前に`EP-OPS-001`のschemaが変更された場合、merged schemaへこのdraftを合わせてからactiveへpromoteする。

## Artifacts and Notes

想定成果物:

- English/Japanese ExecPlan;
- bounded test-architecture inventory;
- finding/disposition ledger;
- authoritative evidence taxonomy;
- concise Agent workflow rules;
- ACCEPT findingのdeterministic regression;
- 必要なfixture/helper修正;
- 妥当なmechanical checks;
- independent review record;
- final retrospective。

historical repeat/CI evidenceはそのまま保存する。過去のcompleted planを「きれいにする」ために書き換えない。

retrospectiveでは、過去に不必要なconfidenceを生んでいたevidence patternがあれば明示する。

## Interfaces and Dependencies

PR #14でmerge済みのEP-OPS-001実装を基盤とする。EP-OPS-001自身が本Planでのforward live dogfoodと別途Human Validationを必要とするため、完了への依存は設けない。bootstrapの完了を偽らず循環を避ける。

production runtimeへの新dependency追加は想定しない。

test-only synchronizationにはGo標準libraryのchannel、`sync.WaitGroup`、`sync.Once`、mutex、atomic等を必要に応じて利用できる。

選択基準:

- 本当にconcurrentなshared stateならatomic/mutex;
- ordering/completion relationが必要ならchannel/barrier/WaitGroup;
- memory safetyとlifecycle completionを混同しない。

repository checkerを追加する場合は既存`tools/repoctl` architectureへ置き、Windows/macOS/Linux nativeで動作し、Bash/Make/PowerShellを必須化しない。

durable human-facing documentationはEnglish/Japaneseを同一coherent changeで維持する。
## 開始時の監査記録（2026-09-10）

修正前に監査基準を固定した。証拠は、強制した不変条件の実行順序、native/integrationの直接観測、race・静的ツール、反復による安定性、コンパイル・構造検査に分ける。反復回数で証拠の種類は変わらない。負例では、無関係な早期拒否でも成功しないかを確認する。キャンセル通知は完了の証拠ではない。

PR #14のmerge `cbd84ed1d2ef4456b8a95db215461bd3d726fe59` でのbaseline: `repoctl check` の単体テストとvetは成功したが、提供Planの依存関係がmappingではなくscalarだったため拒否された。同時に実行した`go test -race ./...`はAndroid fixtureの5554ポート使用中で失敗し、単独のAndroid raceは2.545秒で成功した。suite同士の衝突は仮説であり、外部プロセスの不具合と断定しない。PR #14の最終Windows native run 34418194332/job 102687663661は、即時終了時にguardian完了証拠の読み取りがsharing violationで失敗した。merge済みであることはnative検証の成功を意味しない。

修正前の限定的inventoryとfinding ledger:

| ID / 所有者 | 資源・完了・共有状態・失敗経路 | 判断 / 不変条件・見逃し・最初の検出機会と予防 | 証拠 |
| --- | --- | --- | --- |
| Q01 CDP fixtures | 6種類のHTTP/WS fixtureがhijack handlerとclient readerを所有。mockBrowserのみhandlerをjoin。共有状態はatomic/mutex | ACCEPT: teardown前に所有処理をjoinする。同種helperのlifecycle確認で見逃した。admission/close/joinを共通化 | ソース確認。強制回帰は未実施 |
| Q02 CDP負例 | cancellation、overflow、AX切り詰め、frame変更で汎用error受理やeffect確認欠落 | ACCEPT: 意図した失敗段階への到達が必要。早期拒否でも成功する。実装前のoracle設計と到達・原因確認で予防 | ソース確認。故障対照は未実施 |
| Q03 CDP load wait | atomic評価counterは妥当だが実際のload cancellation重複を強制していない。既存50ms観測は時間依存 | ACCEPT: evaluation開始とcancelを強制し通知と完了を区別。実行順序・oracle設計で検出 | ソース確認 |
| Q04 app command fixtures | SQLite/temp資源に対し成功経路のみdoneを受信。早期Fatalではcancel/release/joinを飛ばせる。cancellation/UI/browser/readinessも対象 | ACCEPT: 失敗経路のoperation所有をcleanupへ持たせる。DB cleanup前の完了をhelper設計で保証 | ソース確認 |
| Q05 execx/process fixtures | native subprocessのstop/wait/Destroyのcleanupエラーを無視。detachedのfallback処理は個別確認が必要 | ACCEPT: 終了失敗は検証失敗とする。cleanup oracle確認と完了失敗注入 | ソース確認 |
| Q06 子孫marker | 遅延marker書き込みのエラーを無視。sleep後の不在は終了の独立した証拠にならない | ACCEPT: identity/完了の独立oracleと成功対照。時間依存proxyをoracle確認で検出 | ソース確認。production leakの証明ではない |
| Q07 Android servers | fakeVersionServerにlistener/accept loop/未joinのdeadlineなしhandler。fake emulatorは固定port | lifecycle調査をACCEPT。port衝突は再現まで別扱い。cleanup所有とsuite分離を確認 | ソース確認と同時baseline失敗 |
| Q08 Windows detached | 即時終了Inspectは証拠読み取りエラーを返すがnative helper testは全errorを再試行して隠す | producer/observer契約の調査をACCEPT。競合handleは未特定。production/oracleの区別に制御した証拠が必要 | native CI失敗とソース確認 |
| Q09 control-plane/worker | ServerはRunをjoinしstore capacityは呼び出しをjoin。stream producerとconstructor早期失敗は追加調査。workerは主に同期 | 限定監査を継続。一般的production不具合とは断定しない。並行実行の結果と強制順序を区別 | ソース確認 |

以前のcorrectness auditは履歴として維持し、その証拠を書き換えない。M2/M5受け入れ前に残りのリポジトリ所有Goテスト/helper群を確認する。suiteが後で成功しただけでは修正済みとしない。

追加指摘の修正前判断: Q10（ACCEPT、Flutterの遅延path変更oracle）: `TestBuildRejectsSymlinkArtifactsAndLatePathEscape`は変更失敗やfake build未到達でも成功できる。変更成功と意図した拒否を確認する。最初の予防機会はoracle・到達性review（UNREACHED_EFFECT）。Q11（調査をACCEPT、Docker integration所有）: cleanup登録がcreateの返却JSONに依存し、割り当て後の出力欠落でlease cleanupを失い得る。既存Podmanのregistry fallbackを比較し、限定した回復対照を追加する。今回、daemon資源の実漏出は観測していない。

M2 inventoryはリポジトリ所有helper群単位のソース確認であり、全実行順序の探索ではない。対象はCDP HTTP/WS、app DB/operation、execx native subprocess、Android fake TCP/emulator、control-plane TLS/stream/DB、worker journal/CAS、repoctl Git/plan/release/docs、CLI Docker/Podman/multi-host/process/browser/Flutter/UI、Compose runner/port、Flutter runner/path変更、source Git/remotesource、instance lock、SQLite、blobstore、assets、evidence/config/stack/domain/policy/paths、buildinfo。いずれもtest所有の資源である。同期群は逐次callback/counterとtemp cleanupを使い、fakeの並行安全性は主張しない。SQLite capacityとblobstoreは並行呼び出しの結果を受信してjoinする。control-plane serverはcancelしてRunをjoinしてからDBを閉じる。stream失敗出口の完了は追加review対象。workerは主に同期。CLI nativeはprocess stop/Waitとlog closeを所有し、実engineは明示opt-inが必要。repoctl releaseはprivate clone/fileと同期buildを所有し、byte capはgrowthを強制して変更flagを確認する。instance lockはstdout barrierとprocess Waitを使い、共有可変callback状態は見つからなかった。source/CASや純粋な境界helperは同期filesystem変更を使う。platform固有testのnative証拠は実行したplatformに限る。

Q09判断: stream/constructor失敗経路の追加強化はDEFER。pipe close/contextが既に処理を制限し、限定reviewで具体的な誤成功は示せていない。constructor/stream失敗注入とproducer完了oracleで再検証してから不具合を主張する。sleepやerror-only assertionの一律禁止はREJECT。それだけでは不正oracleと判断できない。

### 修正の証拠（2026-09-10）

- Q01/Q03: CDPの受付・handler joinとreader終了通知を明示した。joinを除くfault-model overlayは`TestFixtureWorkJoinWaitsForCallback`で“join returned before callback completion”となる。`testing/synctest`で待機順序を確認する。これは旧挙動の再構成であり、無変更HEADでの実行ではない。実際のload waitはevaluation開始、cancel、release、完了を強制する。
- Q02: 同じ無関係な早期error注入を、旧truncated-gone oracleは0.004秒で通し、強化後は3ケースとも“AX truncation boundary not reached”で拒否する。通常の対象テストは成功。CDP full raceは8.842秒で成功。
- Q04: operation cleanup登録を除くoverlayは“resource cleanup preceded operation completion”で失敗。appの6呼び出しでassertion前にcancel/joinを登録し、早期終了時のcleanupを成功経路とは別に検査した。
- Q05: managed/native process fixtureの終了・完了失敗を報告する。全cleanup箇所への故障注入は主張しない。
- Q06: native leafがTCP接続を所有しready byteで開始、EOFでfixture接続寿命の終了を確認する。live echo、明示終了、native Waitが正常対照。親だけkillするoverlayはexecx normalとapp Destroyの両方で“descendant lifetime connection did not end”となる。対象leafの寿命を検証したのであって任意の子孫tree全体の消滅証明ではない。対象packageのLinux full raceはapp 47.223秒、execx 6.603秒で成功。
- Q07: Android TCP fixtureは受付と受付済み接続を閉じてhandlerをjoinする。listener closeだけのoverlayはpartial-request回帰で“cleanup returned with a partial-request handler still active”となる。通常の対象テストは成功、Android full raceは2.600秒で成功。固定portのsuite分離はharness制約として直列実行し、port所有検査を緩めない。
- Q08: Windows DELETE-handle回帰はsharing競合への到達を確認してから空Jobの完了待ちと解除後のabsenceを検査する。不正proofと他の読取失敗は拒否を維持。Windows amd64 cross-compileは成功したがnative実行とnative修正前失敗はCI待ち。

mechanical controlの判断: 上記の実行可能なlifecycle/到達性回帰を既存harness/native/race jobで動かす。証拠の種類を意味的に区別できない文章・sleep・回数scannerは追加しない。

残る範囲の限界: inline CDP Observe fixtureはserver handlerをjoinするが内部client reader完了を直接公開していない。connectionを保持するfixtureはreaderもjoinする。full harness/race、最終integration/native CI、独立technical/英日/parity review、mergeは未完了。

### review修正と残りの採用指摘

独立technical reviewで新しい修正にも2件の不足が見つかった。CDPのoperation worker自身に失敗時joinが必要であり、reader/serverのjoinだけでは足りない。またDockerの出力欠落testは非空runtime identityとcleanup前の実container観測が必要だった。両方修正した。browser mutation-fenceの開始待ちは無期限に待たず早期operation結果を拒否する。早期終了overlayは0.008秒で該当assertionを失敗させ、対象raceは1.246秒で成功した。

Q10: mutation前build errorのoverlayは旧Flutter oracleを通し、強化後は5ケースとも拒否した。通常対象testは成功、Flutter対象raceは1.017秒で成功。Q11: fixture cleanupは返却CLI JSONに依存せず専用SQLite registryを列挙し、完了を確認できなければ回復状態を保存する。global discovery/pruneで削除を認めない。daemon不要の対照は出力欠落・不正JSON・ID不一致・未完了状態・destroy失敗・registry破損を扱う（CLI対象race 1.282秒）。実Dockerの出力欠落回帰はcleanup前のcontainer存在と後の不在を確認する。native結果は追記待ち。新helper APIがなかったことだけで旧コードのDocker修正前失敗とは主張しない。

英日のpolicy追加部分は独立した読者・意味reviewを受けた。提供された日本語Planには汎用model checkerを対象外とする規則とmasterから遅れたときの復旧手順が欠けており、両方補った。M5の欠落指摘は、要求された3項目がreview時点から既に日本語schemaにあったことをreviewerが再確認し撤回した。

Q12（修正前にACCEPT）: Android UI companionのsymlink負例は空targetや不正metadataを使い、無関係なfile不在・source不一致でも成功できる。不変条件: 正常なcompanion artifactの成功を確認してからdirectory/file symlinkを分離し、意図した拒否を検査する。最初の予防機会は負例fixture/oracle設計。testのUNREACHED_EFFECTでありproductの回避を実証したものではない。

Q12の証拠: directory/fileのLstatをStatへ置換するoverlayを旧symlink負例は通す。正常artifactから始める強化後の対照は同じ故障（directory・metadata・APKのsymlink受理）を拒否する。通常uihelper testは成功。相対linkでfile targetを正常なcompanion root内へ置き、無関係なroot逸脱拒否を避ける。baseline corpusは28 package directoryの195 test fileであり、所有fixture fileを3つ追加した現在は198。UI helper/Android UIは同期provenance/payload/dispatch fixtureを使い、追加の負例監査でQ12を発見した。process runtimeのnative即時終了検査がQ08を露呈した直接のcallerである。

local検証checkpoint: Linux Go 1.27.1で`repoctl doctor`と`repoctl check`が成功。Docker/Composeを使う`repoctl test-integration`が成功。最後のcleanup前container oracle追加後、`AGENT_ENV_INTEGRATION=1 go test -tags=integration ./internal/cli -run '^TestIntegrationRegistryRecoversLostCreateResponse$' -count=1`が25.594秒で成功した。Q11の直接daemon証拠である。cacheなし全体raceとreview後最終harnessは実行中。Windows/macOS native CI、current-HEAD review gate、mergeは未完了。

最終local race: `go test -race ./... -count=1`は全package成功（app 47.616秒、CLI 41.073秒、CDP 8.257秒、execx 6.642秒、Android 2.451秒）。修正後の独立technical reviewではCDP worker所有、Docker資源の前後観測、Q12 symlink対照に追加指摘なし。読者・意味一致の再reviewも成功した。これらは独立agent reviewであり、GitHub current-HEAD承認やmaintainer受け入れではない。

review後の最終`repoctl check`はformat、単体test、vet、docs、generated、architectureの全検査に成功した。local実装・検証はcommitとnative CIへ進める状態である。Q09は上記の理由で調査を保留する。review gateとmergeの前に完了・archiveとはしない。

### native CIでの修正（Q06）

commit `d5f43cc8a3b160b6926acc0ad786c7a035d46ca8`、Verify push run 34420910331でWindows Go 1.26/1.27の新しい子孫oracleが失敗した。制御した正常終了を含め、process終了はEOFではなくWSAECONNRESETとなる。またexecxはrunner完了後にconnectionをacceptしていた。Windowsではprocess終了時に待ち行列のconnection/readiness byteが破棄され、寿命観測前のaccept/readが失敗し得る。Linux/macOS成功ではこのWindows protocolの仮定は証明できなかった。今回のQ06修正に入ったtest fixtureの移植性不備であり、新しいproduction process leakの証拠ではない。

修正前の方針: 終了操作・親の正常終了の前にstartup handshakeの確認を必須化する。接続寿命の終了はEOFまたは正確なremote resetだけで判定し、timeout・local close・無関係なerrorは引き続き拒否する。生存leafの故障対照とnative CIを再実行し、この失敗した方法を残す。同じfull runでWindows process-runtimeのQ08とCDP packageは成功した。Browser/Multi-host nativeとRelease previewはWindows/macOS/Linuxで成功し、Linux integrationも成功した。

PR #15で本Planを追跡する。commit後の`plans provenance --plan EP-QUAL-001 --pr-body <file>`と`plans check`は成功した。読み取り専用`plans gate --plan EP-QUAL-001 --pr 15 --repo mahcialet/agent-env`はguarded/manual fallbackとなった。信頼するbaseに`.github/execplan-gates.json`がないため、policyを勝手に追加したりmergeしたりせず、最終受け入れ・archiveにはmaintainerの通常reviewとmergeが必要である。

Q06方針の補足: 元の300msのCommand.Timeoutを維持する。最初の修正では不要に3秒へ拡張していた。startup後にtimerを開始するためだけにdeadlineの意味が変わる独自Contextは導入しない。Runと並行してstartupを観測・確認し、親の継続より前に成立させる。実command timeoutはnativeの時間挙動の証拠として残す。startup前にdeadlineが切れればtest失敗であり、開始済み処理のcancelを検証した成功とはしない。強制証拠とするのはcallback/barrierの順序だけである。

Q06修正を実装した。非同期runnerとaccept済みready/ackを親の継続より前に置き、型付きread reset（Windows WSAECONNRESET）またはEOFだけを接続終了とする。deadline/local-close/refusal/write-reset/bare-reset対照は引き続き拒否する。Linux対象raceはapp 1.261秒、execx 1.416秒で成功。親だけkillするoverlayは生存leafのtimeoutで引き続き失敗した（5.011秒/5.129秒）。PR Verify run 34420955724でも初回実装の同じWindows失敗を確認し、同runのLinux integrationは成功した。

Windows向け修正後の独立technical reviewに追加不具合の指摘はなかった。EOFのみとしていたcommentをremote reset対応へ合わせた。Linux全体のcacheなしraceが再度成功した（`go test -race ./... -count=1`: app 48.064秒、execx 3.926秒、CLI 42.778秒）。対象2 test packageのWindows amd64 cross-compileも成功したが、これはコンパイル証拠に限る。

次のnative CI実行前に、修正後の`repoctl check`は全段階で成功した。

Q13（修正前に調査をACCEPT）: e558891のVerify push run34421905630 integration job102698956780で`TestRemoteCreateNearManifestLimitRoundtrip`が62.02秒後、controller capacity/no online compatible hostで失敗した。新たに観測したCLI fixture失敗であり、manifest転送の回帰を証明したものではない。高コストbundle準備とhost登録の鮮度の関係、同種fixtureを調査する。最初の予防機会は前提資源の有効期間契約とstale hostを強制した対照。TTL延長や再実行で失敗を隠さない。

Q13のソース証拠: 登録後、flags.create内で約4MiBのBuildとblob uploadを再実行する。Onlineはlast_seenから30秒で失効するがfixtureはheartbeatを送らない。PR integration job102698969153/run34421909744でも61.95秒後にcapacity失敗を再現した。CIの正確なheartbeat経過時間は未記録である。controller TTLではなく、所有したheartbeatとoffline/refreshの強制対照でliveness契約を直す。実multi-host fixtureは既にWorker.Runでheartbeatを送る。scheduler直接testはstale hostを意図的に検証しているため変更しない。

Q13実装の証拠: last_seenを強制的に古くしofflineを独立確認した旧fixtureは、同じcapacity拒否で失敗した（race24.57秒）。所有したheartbeatを持つ版は同じstale host条件、全near-limit転送、helper testに成功した（race30.490秒）。独立reviewは停止順序とerror保持を確認したが、cancel通知後のdefault selectによるjoin assertionはQ01/A5と同じ順序の曖昧さがあるとして拒否した。強制証拠に数える前にsynctestの待機状態確認とcancel-only故障対照へ置き換える。Q06のWindows Go1.26/1.27 native jobはPR run34421909744で両方成功した。

Q13のjoin oracleをsynctestの待機状態確認で修正した。cancel-only stopのoverlayは“stop returned before callback completion”で失敗し（0.009秒）、正しい実装のheartbeat対象raceは成功した（1.014秒）。最初の不正overlayはunused variableのcompile errorであり証拠に数えなかった。heartbeat実装を含む全体cacheなしraceは成功した（app50.671秒、CLI43.425秒）。その後の待機状態確認だけのtest修正には上記対象raceを実行した。所有heartbeat helper追加後のcorpusは199 test fileとなる。

Q13の最終独立technical/意味一致の再reviewと修正後`repoctl check`は成功した。heartbeat変更はnative/integration CIへ進める状態であり、先のintegration失敗2件は証拠として残す。

### 最終実装の受け入れ証拠

実装commit `79d4d60498f7a2fdec7441a056f59c59c588f5b6`:

- [Verify PR run 34422872234](https://github.com/mahcialet/agent-env/actions/runs/34422872234)は成功した。Go1.26/1.27でのWindows/macOS/Linux native、全体race、Docker integration、全cross-build targetを含む。
- [Browser native 34422872236](https://github.com/mahcialet/agent-env/actions/runs/34422872236)と[Multi-host native 34422872219](https://github.com/mahcialet/agent-env/actions/runs/34422872219)は3 OSすべて成功した。
- [Release preview 34422872253](https://github.com/mahcialet/agent-env/actions/runs/34422872253)はbuildと3 OSのnative smokeが成功した。
- 先行runの失敗は上記に残す。受け入れの根拠は反復回数ではなく、各修正で記録した対照とその証拠の種類である。

この時点では実装SHAに続く変更は証拠・文書の整合のみだった。その後のQ14のPR review修正には新たな検証が必要である。merge前にcurrent-HEAD CIと人間のreviewを再確認する。PR #15はそのreviewへ進める状態であり、通常のmerge/archive手順まで本Planはactiveを維持する。

Q14（修正前にACCEPT）: CI成功後、PR #15の自動reviewでtestの不足4点が見つかった。CDPのcancel済みcallと連続変更waitは返却errorでなくctx.Errを見ており、cancel後なら無関係な失敗も通せた（UNREACHED_CAUSE）。Android partial-requestとapp早期終了の対照は同期close/t.Runの終了後にしか救済が動かず、connection close/cancelだけが欠落してjoinが残ると救済前にdeadlockし得た（FAILURE_PATH_OWNERSHIP）。いずれもtestのみの指摘であり、返却errorのoracleを強化し、壊れたhelperの各対照に独立して所有する時間上限付き解放経路を設ける。各mutationの結果を記録し、green CIだけでreview済みとはしない。

Q14の対照: callback/frame変更へ到達した後に同じ無関係な返却errorを注入すると、旧CDPの4 testは成功し（0.514秒）、強化後は4つとも失敗する（0.515秒）。通常のCDP対象6 testはrace付きで成功した（1.563秒）。appでcancelのみ除去しjoinを残すmutationは独立watchdogで失敗し（5.013秒）、cleanup全体の除去も順序検査で失敗する（0.010秒）。Androidでconnection closeを除去してWaitを残すmutationは明示した救済assertionで失敗し（5.017秒）、通常の対象raceは成功した（1.012秒）。独立reviewと対象raceでapp/Android/CDPの所有とerror判別を確認した。最初の不正CDP overlayはcompile失敗のため除外し、修正したruntime mutationのみ証拠に数えた。

Q14の最終独立ソース・英日reviewに残る指摘はなかった。全体cacheなしraceは成功し（app47.446秒、CLI41.503秒、CDP8.138秒、Android2.429秒）、最終`repoctl check`も全段階で成功した。この対応commitのnative CIは別途確認し、先行CIの受け入れで最新revisionの検証を代用しない。
