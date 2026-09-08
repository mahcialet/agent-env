---
source_sha256: 2e17771726a8825507c5e1bd2cf917327d971e43a6ceee6aad2070237a3cb77c
translation_of: docs/exec-plans/active/browser-cdp-automation.md
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Lease所有Browser/CDP自動操作とsemantic snapshotを追加する

[English](browser-cdp-automation.md)

このExecPlanはliving documentであり、`docs/PLANS.md`に従って更新する。

想定ブランチ: `feat/browser-cdp-automation`

PR #9（persistent native process runtime）を必須前提とし、`master`へmergeされるまで
実装を開始しない。

開始revision: `3837c12c88d880c3082eb9715795ae035de8a181`

## 目的 / 全体像

完了後、Agentは通常ユーザーのbrowser profileへattachせず、lease内の隔離された
Chromium-family browserをCDPで観測・操作できる。

process lifecycleはPR #9の責務:

    process runtime
      -> native identity
      -> private runtime_dir
      -> loopback port
      -> logs/readiness/reconcile
      -> destroy/quarantine

Browser/CDPはその上に:

    page/target
    Accessibility semantic snapshot
    DOM/layout snapshot
    screenshot
    navigation
    stale-safe click/text/key/scroll
    waits
    console
    bounded network capture

を追加する。

暫定manifest:

```yaml
runtimes:
  browser-process:
    type: process
    source: app
    working_directory: .
    command:
      - chrome
      - --headless=new
      - --enable-automation
      - --user-data-dir=${runtime_dir}/profile
      - --remote-debugging-address=127.0.0.1
      - --remote-debugging-port=${port:cdp}
      - about:blank
    ports:
      cdp: {protocol: tcp}

browsers:
  web:
    type: chromium-cdp
    runtime: browser-process
    cdp_port: cdp
```

port名だけからbrowserをimplicit推測しない。

## 進捗

- [x] PR #9 merge / exact revision記録
- [x] branchを作成。
- [x] baselineの`go test -race ./...`成功（app 33.292秒）。
  `repoctl check`はunit/vet成功後、提示された日本語planの翻訳metadata欠落でdocs-check失敗。
  今回の文書更新でmetadataを補完。
- [x] final process runtime / Android UI observer調査
- [x] 英日Browser/CDP product/design docs
- [x] browser binding syntax
- [x] tested browser matrix
- [x] Go CDP/WebSocket implementation
- [x] capabilities/identity
- [x] private profile + exact CDP port validation
- [x] browser-level CDP connection
- [x] page/target model
- [x] Accessibility snapshot
- [x] bounded DOM/layout snapshot
- [x] screenshot
- [x] stale-safe click/text/key/scroll
- [x] navigation/waits
- [x] console diagnostics
- [x] bounded network capture
- [x] fencing/cross-lease/port-reuse tests
- [x] privacy/redaction
- [x] iframe/shadow DOM/multi-page
- [x] lease所有backendを含むLinux native fixtureを実行。
  Chrome 152.0.7977.64、CDP 1.3、amd64、sandbox有効。初回7.072秒、後続3反復が成功。
  最新raceも成功（package 8.489秒、native test 7.47秒）。Windows/macOSは下の未完了gateで追跡。
- [ ] macOS real integration
- [ ] Windows real integration
- [x] browser+lease backend E2E
- [x] 製品・設計・architecture・security/reliability・portability・quality・manifest/CLI・
  distribution・index・roadmapを英日更新し、今回の`repoctl docs-check`が成功。
  native CI後に最終受け入れ証拠を照合する。
- [ ] final harness/race/native/cross
- [ ] direct evidence/retrospective
- [ ] completed移動/link/hash

checkboxは観測済み完了のみ。browser/protocol version、native run、resultを記録する。

### 2026-09-08 push前の実装checkpoint

- `go run ./tools/repoctl check`成功。format、unit、vet、docs、generated、architectureを検査
  （app 8.352秒、CLI 5.357秒）。
- `go test -race ./...`成功（app 37.289秒、SQLite 15.069秒）。最新のapp browser対象raceも2.855秒で成功。
  fence喪失時の出力消去、identity field保持、後続の入力text redactionと改ざん拒否、degraded guard、destroy fencingを検証。
  拡充したCDP adapter testも成功。
- `TestBrowserNativeCLI`はLinux amd64、sandbox有効、Chrome 152.0.7977.64 / protocol 1.3で成功。
  初回・後続3反復・最新race（package 8.489秒、test 7.47秒）を実行。
  空入力での消去、browser名を入力してもidentityを壊さないこと、重複/置換node拒否時のbackend副作用不在、
  後続consoleのUnicode redaction、および下記fixture範囲を検証。
- `CGO_ENABLED=0`のCLI cross-buildはWindows/Darwin/Linux × amd64/arm64の6 targetで成功。
  これはコンパイルの証拠のみ。Windows/macOSのnative browser CIは未実行。
- `TestArchitectureBoundaries`へbrowser adapter依存の負例を追加し、arch-check成功。
  今回の証拠更新後にdocs-checkを再実行する。
- その後Dockerを使う完全な`repoctl test-integration`が成功。明示opt-in前提testは設計どおりskipするため、
  このrunでoptional native Android/Podman/browser gateまで検証済みとはしない。B25/B26/B33/B34は未完了。


### 最終local実装checkpoint（2026-09-08）

同一document内のURL digest、許可したAX状態、Windowsの混在区切りprofile path比較、
redaction済みlabelの照合を実装した。これらを含む`go run ./tools/repoctl check`は成功し、
architecture負例と翻訳検査も通った。最終実Linux native race fixture成功
（package 8.819秒 / test 7.81秒）、Chrome 152.0.7977.64 / CDP 1.3、sandbox有効。
以前のcheckpointで残っていたlocal追加修正の検証はこの結果で完了した。
Windows/macOSのbrowser実行と公開最終CIは未完了のため、Planはactiveを維持する。

## 想定外の発見

- 独立reviewのP1：lease fence喪失後、redaction前のprovider observationがerror resultに残り得た。
  この経路ではobservation出力を消去し、`TestBrowserLockLossDoesNotExposeObservation`でraw secretが出ないことを検証。
  fenceを失った所有者は状態を確定しない。
- 独立reviewのP2：構造化文字列を一律redactすると、入力がbrowser名`web`などに一致した場合に操作対象identityを壊した。
  authority fieldは保持し、人向け内容をredactするよう修正。
  `TestBrowserPriorTextRedactionPreservesAuthority`とnative入力で後続snapshot再利用・redaction証拠の改ざん拒否を検証。
- 統合途中のharnessはCDP transport編集中のformat-checkで失敗した。整形を修正して完全harnessが成功。
  check・test・portability要件は弱めていない。

- baselineのunit/vet・full raceは成功したが、最初のharnessはdocs-checkで失敗した。
  提示された日本語active planに`translation_of`と`source_sha256`がなかったためであり、
  baseline全体の成功とは扱わない。対応する英日planの意味を確認し、今回metadataを追加した。
- loopback discoveryだけではCDP listenerとnative lease rootを結び付けられない。
  process provider再検査、`SystemInfo.getProcessInfo`、`Browser.getBrowserCommandLine`、
  discovery、`Browser.getVersion`を組み合わせる。異なるbrowser PIDをforkするlauncherは非対応とし、
  browser実行ファイルを直接使う。
- DOM snapshotの文字列・attributeは入力node以外にもsecretを含み得る。
  DOMはstructure/layoutのみ、AXはeditable valueを除外し、入力textの永続redaction証拠はfingerprintだけを保持する。
- same-origin iframe・shadow観測には対応する。今回のiframe semantic入力は明示的に非対応とし、
  cross-origin/OOPIFを暗黙の観測・入力fallbackにしない。

CDP version差、AX/DOM差、OOPIF/iframe、shadow DOM、navigation target replacement、
stale race、console history、network late attach、profile lock、browser auto-update、
OS executable差、WebSocket teardown等を記録する。

dynamic page対応のためidentity/stale checkを弱めない。

## 判断の記録

- 2026-09-08 — 実装判断：`browsers.<name>`に`type: chromium-cdp`、`runtime`、`cdp_port`を定義。
  既存process runtimeの名前付きTCP portへ結び付け、1 runtimeにつきbindingは1つ。
  明示宣言で無関係なprocessへのbrowser機能付与を防ぐ。必須argvはそれぞれ1回だけ、
  `--headless=new`、`--enable-automation`、`--user-data-dir=${runtime_dir}/profile`、
  `--remote-debugging-address=127.0.0.1`、`--remote-debugging-port=${port:<cdp_port>}`とする。
  保護対象の別表記・重複・profile/debugging上書きを拒否し、adapterから追加しない。
- 2026-09-08 — 実装判断：`github.com/gorilla/websocket` v1.5.3を接続専用CDP transportに使い、
  browser launcher依存やNode/Pythonを追加しない。domainはbrowser値、appはfencing/証拠、
  `internal/browser`はprotocolを担当し、process lifecycleを持たない。native identity確認は既存process providerに委ねる。
- 2026-09-08 — 実装判断：直接起動できるheadless Chromiumとnative browser-root PID・command-line証明を必須とする。
  loopback discoveryだけで許可しない。native matrixはChrome for Testing 152.0.7977.82、Go 1.27、
  Windows/macOS/Linux。ローカルChrome 152.0.7977.64、protocol 1.3も検証するが、machine固有pathは文書へ書かない。
- 2026-09-08 — 実装判断：すべての操作でlease fenceを保持。degradedはidentityを証明できるread-only診断のみ。
  quarantine、未完了run、所有権不明を拒否する。変更操作は入力前にintentを保存し、切断・確定不明時には
  running cleanup barrierを残して、自動再実行しない。
- 2026-09-08 — 実装判断：stale検証はnative/browser/page identity、document loader、backend DOM/frame fingerprintを照合。
  same-origin iframe/shadowは観測するが、identity/actionを支えられないiframe入力・cross-origin/OOPIFは明示的に非対応。
  対象nodeに限定した内部固定JavaScript readbackは許可し、任意JavaScript・CDP passthroughは公開しない。
- 2026-09-08 — 実装判断：networkはheader/bodyを保存せず、URLのuserinfo/query値をredactする。
  console/networkは接続期間と容量に上限を設ける。DOMは文字列・attributeを除いたstructure/layout、AXはeditable/password値を除外。
  set-textは入力前に長さ・全体/接頭辞SHA fingerprintをartifact登録し、後続観測がそれを読み込みechoされた入力もredactする。
  証拠欠落・破損は安全側に倒して失敗し、検証CPUにも上限を設ける。証拠はprivateに保ち、暗号化とは扱わない。
  PNG pixelには秘密が残り得るため、自動redaction済みとはしない。
- 2026-09-08 — 実装判断：最小surfaceに`page-create`、`page-close`、`dom-snapshot`を追加する。
  既存command run/artifactを再利用し、browser lifecycle tableは追加しない。
  Playwright、download、外部接続、Android/browser共通抽象化は導入しない。

- 判断: Browser/CDPはPR #9 persistent process上へlayerし別process lifecycleを持たない。
  理由: process identity/state/port/log/cleanupはgeneric基盤の責務。
  日付/担当: 2026-09-08 / maintainers.

- 判断: initial scopeはChromium-family CDPのみ。
  理由: Accessibility/DOMSnapshot/Page/Input/Runtime/Log/Networkが必要primitiveを提供。
  日付/担当: 2026-09-08 / maintainers.

- 判断: private lease-owned user-data-dir必須。
  理由: personal browser stateをautomationから隔離しmodern Chrome security要件と整合。
  日付/担当: 2026-09-08 / maintainers.

- 判断: external existing browser attachmentはscope外。
  理由: CDP input前にprocess/profile/port ownership proofが必要。
  日付/担当: 2026-09-08 / maintainers.

- 判断: Accessibility treeをsemantic snapshotの正、DOMSnapshotをstructure/layout補完。
  理由: AX role/nameがagent actionに適しDOMはbounds/debugに有用。
  日付/担当: 2026-09-08 / maintainers.

- 判断: node handleはephemeral/snapshot-scoped。
  理由: navigation/dynamic updateでDOM/AX identityが変わる。
  日付/担当: 2026-09-08 / maintainers.

- 判断: stale/ambiguous action拒否、old coordinate fallback無し。
  理由: stale observationで別elementを操作しない。
  日付/担当: 2026-09-08 / maintainers.

- 判断: cleanupはprocess runtimeへ委譲し`Browser.close`をownership proofにしない。
  理由: native process tree/profile lifecycleはprocess runtime責務。
  日付/担当: 2026-09-08 / maintainers.

- 判断: CIはChrome for Testing推奨、bundle無し。
  理由: versionable evidenceとstandalone coreを両立。
  日付/担当: 2026-09-08 / maintainers.

- 判断: arbitrary raw CDP/unrestricted JS public escape hatch無し。
  理由: typed primitiveの方がsafety/evidenceをreview可能。
  日付/担当: 2026-09-08 / maintainers.

- 判断: durable docs/ExecPlanは英日。
  理由: repository policy。
  日付/担当: 2026-09-08 / maintainers.

- 2026-09-08 — 実装判断：browser省略時は割り当て済みbindingが1つ、page省略時はeligible pageが1つに限る。
  page一覧はtarget IDで整列し、page-closeには明示IDが必要。backend identityのないAX nodeは観測のみで入力を許可しない。
  shadow AXも同じ表現へ正規化する。protocol番号のallowlistではなく、必要メソッドの非対応を明示的に失敗させ、
  discoveryとlive versionを一致確認する。各CDP callは操作期限内で5秒、discoveryは64 KiB、WebSocket messageは8 MiB、
  pageは128、frameは32、AX/DOM nodeは2048、semantic JSONは1 MiBまで。
  console/networkは各256 record・文字列合計64 KiB・各文字列4096 byteまでとし、超過文字列全体を`[TRUNCATED]`にする。
  512 eventのtransport buffer超過はcapture失敗とする。consoleは接続前eventのreplayを無視する。
  部分的な証拠で暗黙に操作を許可しないため、上限と非対応を明示する。
  download・headfulは延期し、このsliceではAndroid/browser共通UI層を導入する根拠はない。

- 2026-09-08 — 実装調整：明示navigateはhash/history移動などでloaderが変わらない場合も以前のbrowser snapshotを無効にする。
  `TestBrowserNavigateInvalidatesSameDocumentSnapshot`を追加（最終再実行は未完了）。document tokenにもraw URLのdigestを含め、
  raw queryを保存せずsame-loaderのURL変化を検出する。node fingerprintには許可した非text AX state
  （`checked`、`selected`、`expanded`、`readonly`、`required`、`focusable`、`focused`、`multiselectable`）を含める。
  意味が変わったnode/documentをfreshと誤認しないための変更。
- 2026-09-08 — CLI調整：set-textの`--text`は明示必須とし、明示した空文字列は許可する。
  操作に必要なkey/URL/page/snapshot/node引数を必須とし、明示zero durationはstoreを開く前に拒否する。
  引数省略が意図しない空文字列への置換になることを防ぐ。これらの最終調整は受け入れ前の再検証が必要。

## 成果と振り返り

実装とlocal検証はまとまったcheckpointに達したが、最終受け入れ完了ではない。config/domain契約、
fence付きapp処理、CDP transport/操作、private証拠、英日文書を実装した。local完全harness・race・6 cross-build・
Linux real browser反復が成功し、独立reviewの指摘を修正して回帰testを追加した。
Docker integrationも成功した。Windows/macOS native、最新stale-state/CLI調整の再検証、最終受け入れ照合が済むまでarchiveしない。
baseline docs-check失敗とnative PID/command-line証明が必要だった発見は、後続成功で隠さず記録した。

未完了。

完了時にbinding、browser versions、CDP transport、ownership、profile isolation、
page model、AX/DOM normalization、stale semantics、input、console/network制約、
privacy、native evidence、Android/Browser共通UI abstraction昇格要否をまとめる。

## 背景と構成

読む:

- AGENTS/ARCHITECTURE/PLANS 英日
- PR #9 persistent-process product/design/completed plan
- Android UI observer docs
- PORTABILITY/SECURITY/RELIABILITY/QUALITY/roadmap 英日
- standalone docs
- internal/runtime/process
- internal/execx
- internal/app/domain/config
- endpoint/readiness/evidence/store

protocol reference:
CDP Accessibility/DOMSnapshot/Page/Target/Input/Runtime/Log/Network、
Chrome remote-debugging security guidance、Chrome for Testing。

repository docsをagent-env behaviorの正とする。

## 作業計画

### Milestone 1 — Browser binding

英日product/design作成。

top-level `browsers`を第一候補。

bindingはowned `process` runtime + named TCP CDP portをexplicit参照。

validate:
- runtime type process
- CDP port TCP
- persisted argvがexact port使用
- loopback address
- user-data-dirがruntime_dir内

browser-owned process lifecycleを追加しない。

### Milestone 2 — CDP transport/capabilities

already-running browserへattachするGo CDP client。

browser WebSocket/target session/deadline/cancel/event demux。
Node/Python helper、browser launch ownership無し。

capabilitiesはread-onlyでproduct/version/protocolを報告。

### Milestone 3 — Identity/page

operation前:
1. lease/browser/runtime解決
2. process identity proof
3. recorded CDP port
4. loopback discovery
5. browser WS
6. product/protocol verify
7. exact page

old portを別processがreuseした場合attach禁止。

multiple pageはexplicit selection。

### Milestone 4 — AX/DOM snapshot

primaryはAccessibility。

compact `[n1] role "name"`。

structured evidenceはframe/role/name/value/backend DOM/layout/state。

DOMSnapshotはbounded補完。
node/byte limitとdeterministic truncation。

### Milestone 5 — Screenshot

PNG、identity/URL/title/dimensions/timestamp/digest。
invalid base64はfail。
pixel secret非redactionをdocument。

### Milestone 6 — Stale-safe input

click/set-text/key/scroll。

prior snapshot/node参照必須。
process/CDP/page/node再検証。
CDP Input優先。
Unicode real test。
coordinate fallback無し。
uncertain action auto retry無し。

### Milestone 7 — Navigation/wait

explicit URL。
load/URL/role-name-text/disappearance/stable conditionをbounded wait。
navigationでold snapshot invalid。
timeout後poll無し。

### Milestone 8 — Console/network

console bounded/attributed/redacted、history制約明記。

networkはduration/action-scoped。
URL/method/status/type/timing/failure。
Authorization/Cookie/Set-Cookie redaction。
response body default保存無し。

### Milestone 9 — Isolation/failure

cross lease/browser/page snapshot拒否、process death拒否、port reuse拒否、
destroy fencing、profile deletion proof、normal user profile未使用、auto restart無し。

DEGRADED read-only policyを実装前決定。
QUARANTINED input禁止。

### Milestone 10 — Native real integration

fixture:
heading/email/Unicode/password/button/DOM replacement/duplicate/iframe/shadow/
scroll/console/network/navigation。

Windows/macOS/LinuxでAX/DOM/screenshot/input/stale/console/network/cleanupまで証明。
CIは可能ならChrome for Testing、exact version記録。

### Milestone 11 — Browser + backend

agent-env提供endpointへnavigateしsemantic actionでbackend request。
特定Compose providerへcoupleしない。

### Milestone 12 — Docs/completion

README/Architecture/Portability/Security/Reliability/Quality/Roadmap/index/
standalone prerequisite matrix英日更新。

Browser/CDP external prerequisiteはcompatible Chromium-family browser。
bundleしない。

## 具体的な手順

1. PR #9 merge
2. master更新/revision
3. branch
4. 英日plan
5. baseline harness/race
6. process/UI調査
7. bilingual product/design
8. binding
9. CDP transport
10. capabilities/identity
11. page
12. AX
13. DOM
14. screenshot
15. input
16. navigation/wait
17. console
18. network
19. isolation/failure/privacy
20. fixture
21. Linux real
22. macOS real
23. Windows real
24. browser+backend E2E
25. docs
26. final harness/race/native/cross
27. evidence
28. retrospective
29. completed/link/hash

## 検証と受け入れ

| ID | 必須動作 | 証拠 |
| --- | --- | --- |
| B1 | existing Compose/Podman/Android/Flutter/UI/process非回帰 | local full race/harnessとDocker integration成功。optional前提を要するopt-in testは設計どおりskipし、追加のnative Android/Podman成功は主張しない。 |
| B2 | owned process runtime + named CDP port explicit binding | `TestBrowserManifestContract`、`TestBrowserManifestNegativeFixtures`、`TestBrowserRequiresProcessRuntime`。`TestEndpointBoundary`で別endpoint authority拒否。 |
| B3 | runtime-owned private profileのみ、default profile未使用 | `TestPrivateProfileFlags`とconfig負例。Linux `TestBrowserNativeCLI`で2 leaseのstate directory・PID・CDP port相違を確認。 |
| B4 | browser operation前process identity再検証 | `TestBrowserLifecycleGuards`でdead/uncertainをprovider呼出し前に拒否。`TestNodeChangesDuringOwnershipVerificationNeverInputs`で入力直前再検査。 |
| B5 | unrelated port reuse誤認無し | `TestBrowserPIDAndDiscoveryProof`で不一致PID/version/endpoint拒否、`TestEndpointBoundary`で別port拒否。制御したtransport負例であり、実kernel port再利用raceの再現とはしない。 |
| B6 | capabilities side effect無し | Linux `TestBrowserNativeCLI`でChrome 152.0.7977.64 / CDP 1.3を記録し、capabilities後もpageがabout:blankであることを確認。 |
| B7 | page deterministic、multiple時explicit | Linux `TestBrowserNativeCLI`で2つ目のpage作成/閉鎖、省略時のmulti-page snapshot拒否。adapterはpage IDで整列。 |
| B8 | AX snapshot deterministic versioned JSON/text | `TestBrowserSnapshotRegistrationAndSemanticInput`でsnapshot artifact登録。native fixtureでheading/button/textboxのroleとiframe/shadow nodeを確認。 |
| B9 | bounded DOM/layout evidence | `TestDOMSnapshotSuppressesSensitiveStrings`、`TestSnapshotOmitsOversizeSensitiveValue`。native DOM artifactのversion 1、空でないnode、2048以内を確認。 |
| B10 | valid PNG/digest/exact page identity | native fixtureでPNG復号、正の寸法、SHA256、lease/run/browser/page帰属を確認。`TestScreenshotMessageBoundAndDecodeFailure`で不正/過大messageを検証。 |
| B11 | fresh semantic click | native clickでbackend count 1とprocess logの相関を確認。node置換後の古いhandleではbackend副作用が増えない。 |
| B12 | Unicode set-text | native fixtureで日本語・emoji・accent・引用符/backslashをCDP/backendへ往復し、空入力消去とbrowser名に等しい入力も確認。 |
| B13 | key/scroll selected pageのみ | native Enterとscrollで期待page textを確認。独立した2つ目のlease/pageはabout:blank、backend count 0のまま。 |
| B14 | stale/ambiguous input前fail、coordinate fallback無し | `TestStaleAndAmbiguousNodeNeverInputs`、`TestNodeChangesDuringOwnershipVerificationNeverInputs`。native置換・navigation後に古いhandleを拒否。 |
| B15 | bounded navigation、old snapshot invalid | nativeで/nextへ移動後に以前のsnapshot入力を拒否し、期限付きURL waitが成功。 |
| B16 | wait timeout後hidden poll無し | `TestWaitUsesOperationDeadlineRatherThanCaptureDuration`。native不在text waitは250 ms timeout、外側5秒以内に終了を確認。 |
| B17 | console bounded/attributed/redacted/history明示 | `TestConsoleIgnoresHistoryAndOmitsOversizeValues`。native heartbeat・後続Unicode echo captureで出力と登録JSONのredactionを確認。 |
| B18 | network bounded/auth-cookie redaction | native network captureで/tick request IDとstatus 200を照合し、artifactに認証/header試験文字列がないことを確認。modelはheader/bodyを保存せず、上限を文書化。 |
| B19 | cross lease/browser/page reuse拒否 | native fixtureで別lease/pageのhandle拒否。appでsnapshot browser/page一致とdigest登録をprovider入力前に確認。 |
| B20 | destroyがmutating operationを追い越さない | `TestBrowserMutationFenceBlocksDestroy`、`TestBrowserUncertainMutationRetainsCleanupBarrier`、`TestBrowserEvidenceFailureRetainsBarrier`。 |
| B21 | process death後CDP拒否 | native fixtureで2つ目のbrowser rootをkillし、後続browser pages拒否、showがnon-readyで履歴PID不変を確認。 |
| B22 | process absence前profile削除無し | native fixtureで両lease destroy後のstate/profile directory不在を確認。generic `TestMissingLaunchingReceiptIsUncertain`とbrowser結果不明/証拠barrierで保守的cleanupを維持。 |
| B23 | process lifecycle重複実装無し | `TestArchitectureBoundaries`のbrowser依存負例とarch-check成功。native fixtureはCDP Browser.closeではなく通常destroyでcleanup。 |
| B24 | auto restart無し | `TestBrowserLifecycleGuards`で起動回数不変。native手動終了fixtureで履歴PID不変・readyに戻らないことを確認。 |
| B25 | Windows real headless pass | 未完了：Windows native browser CI未実行。6 target cross-buildはnative証拠ではない。 |
| B26 | macOS real headless pass | 未完了：macOS native browser CI未実行。 |
| B27 | Linux real headless pass | Linux amd64 `TestBrowserNativeCLI`、sandbox有効、Chrome 152.0.7977.64 / CDP 1.3。初回・3反復・最新race成功（package 8.489秒 / test 7.47秒）。 |
| B28 | real AX/DOM/screenshot/Unicode/click/stale/iframe/shadow/console/network/cleanup | Linux native fixtureで列挙機能を検証。same-origin iframe/shadowは観測のみで、iframe入力とcross-origin観測は明示的に非対応。Windows/macOS未完了。 |
| B29 | provider非依存lease backend E2E | native fixtureでrepository所有HTTP backendを別process runtimeとしてbuild。Unicode request/count/log相関と別lease backendの不変を確認。Compose provider不使用。 |
| B30 | Browser未使用時coreにbrowser不要 | browser integration tagなしでcore unit/raceと6 CGO-free CLI cross-build成功。browser前提は明示的browserintegration test/commandだけに適用。 |
| B31 | Node/Python/Playwright/Selenium/ChromeDriver runtime dependency無し | Go gorilla/websocket transportと直接native argvを使用。architecture check成功、helper runtime/browser同梱なし。 |
| B32 | 英日durable docs final behavior/privacy | 製品・設計・関連文書を英日更新し、正確なflag、native前提、privacy/上限を記載。意味確認後hash更新、docs-check成功。 |
| B33 | final harness/translation/race/native CI | 未完了：local完全harness/race・Docker integration・6 cross-build成功。Windows/macOS native CI未実行、最後のstale-state/CLI調整は再検証が必要。 |
| B34 | 英日ExecPlan evidence/retrospective後archive | 最終受け入れ未完了。今回の直接証拠・review修正・残るnative/integration gateを記録し、英日planをactiveに維持。 |

## 冪等性と復旧

read-only observationはdesired stateを変えない。

mutating browser operationはnon-idempotent。1回attemptし、uncertain時auto replayしない。
fresh snapshot後にretry判断。

CDP disconnectはinput未実行proofではない。

old port listenerやnormal user browserへrecover attachしない。

profileはprivate mutable runtime state。
destroy/GCはprocess runtime ownership proofへ委譲。

## 成果物と注記

例:

    leases/<id>/artifacts/<browser-run-id>/
      run.json
      snapshot.json
      snapshot.txt
      dom-snapshot.json
      screenshot.png
      redaction.json  (set-text前のfingerprint証拠)

record:
lease/browser/process/CDP port/product/version/protocol/page/URL/title/snapshot/node/
operation/time/truncation/digest/result/uncertainty。

password/secret inputをclear metadataへ保存しない。
DOM/console/URL/screenshotはsecretを含み得る。

## インターフェースと依存

想定:

    internal/browser/
    internal/browser/cdp/

BrowserProviderはCapabilities/Pages/Snapshot/Screenshot/Act/Navigate/Console/
CaptureNetwork等。

process runtimeはprocess/profile dir/port lifecycle。
appはfencing/stale/evidence。
CDP adapterはprotocol。
storeはpersistence、CLIはparse/render。

Browser利用時のみcompatible Chromium-family executableが外部prerequisite。

Node/Python/Playwright/Selenium/ChromeDriver/shell/CGO/daemonをcore requirementにしない。

Milestone 1の初期論点は判断の記録で解決済み。受け入れ時の照合用に保持する:

1. binding location/name
2. required Chrome flagsをrepo宣言のみかcontrolled appendか
3. tested browser matrix
4. Go CDP/WebSocket library
5. protocol compatibility
6. process/profile/CDP identity proof
7. default page selection
8. AX stale fingerprint
9. OOPIF
10. shadow DOM
11. unrestricted JS無しUnicode text
12. console history
13. network limits
14. download handling
15. headless/headful contract
16. Android/Browser shared UI abstraction時期

現行app portは`BrowserProvider.Observe(context.Context, domain.Runtime, domain.BrowserBinding, domain.BrowserRequest, func(context.Context) error) (domain.BrowserObservation, error)`、adapterは`internal/browser/cdp`。console/network配列は`run.json`に保存し、個別collection fileは作らない。set-text前にはfingerprintのみの`redaction.json`を登録する。
