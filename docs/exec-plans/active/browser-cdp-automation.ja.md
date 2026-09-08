---
source_sha256: 02c4f5c214cb79c33602a3d67c30cfe776b12c527fa75a39f35b261f3f80416f
translation_of: docs/exec-plans/active/browser-cdp-automation.md
status: active
owner: maintainers
last_verified: 2026-09-09
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

- [x] 2026-09-09: 最終統合repoctl check、全repository race、sandbox有効Linux native race（10.082秒）が成功した。6件のmutation callback検証を復旧後（race 10回、1.582秒）、独立相互レビューも成功した。redaction後のsnapshot回帰テスト成功（6.369秒）。2 MiB超の膨張、保存artifact 1 MiB上限、対象識別子保持、永続run完了を検証した。残るgateは新しい複数OS CI。

- [x] 2026-09-09: 第3回5件を修正し、修正前に失敗する回帰テストを追加した。closed shadow native race成功（11.031秒）、actions/focus race 3回成功（1.245秒）、AX snapshot/gone境界race 10回成功（15.425秒）、capture/network/transport race 10回成功（44.414秒）、最終console/capture race 10回成功（3.091秒）。この時点では統合harness/full raceと新しいnative CIは未完。


- [ ] 2026-09-09: PR #10の第3回レビューに対応する。redaction後のsemantic上限、closed shadow入力、capture購読期限、AX node境界、console引数省略を修正し、回帰テスト・harness・native CIの証拠を確認してから完了へ移す。

- [x] 2026-09-09: 最終実装cdcec91807a27b6215d2aeb0f6533ed8c96437cdで全検証が成功した。PR Verify 34252382308（12 jobs）、PR Browser native 34252379866（Linux 11.50秒、macOS 14.04秒、Windows 30.98秒）、Release preview 34252379587（buildと3 smoke jobs）、push Verify 34252373749、push Browser native 34252373761。第2回8件すべてに対応内容を返信済みで、archiveとともに最終CI確認とResolveを行う。


- [x] 2026-09-09: page前面化・document focus確認・前面化後の対象再検証を追加した。inactive documentと前面化時の対象変更を修正前に再現した。native fixtureは各入力前に別の前面tabを開く。最終CDP race成功（2.833秒）、Linux native race成功（10.791秒）、repoctl check成功、独立レビューで追加指摘なし。この時点ではmacOS/Windowsの受け入れはCI待ちだったが、上記の最終CI証拠で完了した。

- [x] 2026-09-09: 最終統合repoctl checkとfull raceが成功した。最終CDP race成功（2.509秒）、sandbox有効Linux native成功（8.491秒）。focus転送時の拒否、query/fragment条件、通常入力・shadow動作、frame境界を検証した。readback修正後の独立レビューも成功した。この時点ではcross-platform CIは未完だったが、上記の最終証拠で完了した。

- [x] 2026-09-09: 保存manifestの差し替え、redaction後の膨張、capture queueの無表示省略、不正DOM frame証拠、別tabによる拒否、focus転送、effect後のエラー、query/fragment URL waitを修正前に再現するテストを追加した。
- [x] 2026-09-09: レビュー8件の修正を実装した。app targeted race成功（2.656秒）、capture/transport isolated race 10回成功（4.600秒）、CDP race成功（2.439秒）、frame fixtureを強化したsandbox有効Linux native成功（8.633秒、Chrome 152.0.7977.64/CDP 1.3）。


- [x] 2026-09-09: PR #10の第2回レビュー8件に対応し、回帰テストとharness・3 OS CIの新しい証拠を確認してから再度完了へ移動する。

### PR #10 review対応（2026-09-09）

- [x] native process tree不在確認後のWindows共有違反cleanupを上限付きで検証し、汎用processの所有権と失敗時barrierを維持。

初回archive後にWindows native run 34236523326が失敗したため、
`feat/browser-cdp-automation`で本Planを再開した。PR起動の34240370827が変更なしで
成功しても修正済みとは扱わず、review修正と追加のWindows cleanup失敗の間はactiveを
維持した。`391288c`で最終gateがすべて成功し、9 Threadすべてへ返信・Resolveした後、
2026-09-09に英日Planを再archiveした。

- [x] browser報告のsecurity originでiframeを判定。継承・blob・opaque originとnative timingの回帰を検証。
- [x] URL waitに空でないsubstringを必須とし、roleを拒否。
- [x] 保存network文字列すべてに上限を適用し、DOM名の省略をtruncatedに反映。
- [x] 省略されたsnapshotから消失を断定しない。
- [x] captureに必要なeventだけを購読し、無関係eventで通常操作を切断しない。
- [x] 不確定runを含め、入力前にpage/snapshot/nodeの根拠を保存。textは開示しない。
- [x] architectureの英日statusを修正し、契約と判断の証拠を更新。
- [x] 回帰・race・harnessと実3 OS native CIを完了し、最新証拠を照合してからarchive。
- [x] PR #10の対応済みThreadすべてへ返信しResolve。

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
  最新raceも成功（package 8.489秒、native test 7.47秒）。Windows/macOSもその後成功し、下記とB25/B26に証拠を記録。
- [x] macOS real integration: `9b94b42`、run 34234714187、darwin/arm64、
  Chrome 152.0.7977.82 / CDP 1.3、native test 20.98秒で成功。
- [x] Windows real integration: `9b94b42`、run 34234714187、windows/amd64、
  Chrome 152.0.7977.82 / CDP 1.3、native test 25.54秒で成功。
- [x] browser+lease backend E2E
- [x] 製品・設計・architecture・security/reliability・portability・quality・manifest/CLI・
  distribution・index・roadmapを英日更新し、今回の`repoctl docs-check`が成功。
  native CI後に最終受け入れ証拠を照合する。
- [x] (2026-09-08) final harness/race/native/cross: `b48ab64`のVerify
  34235476057、Browser native 34235476126で成功。
- [x] (2026-09-08) B1–B34の直接証拠と英日retrospectiveを完成。
- [x] (2026-09-08) 英日Planをcompletedへ移動し、参照・翻訳hash・文書検査を更新。

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

- 2026-09-09: 独立レビューで、既存のstale target回帰fixtureに新たに必要な隔離execution contextがなく、mutation callback前の拒否だけでテストが通ることが判明した。context fixtureとcallback実行の必須assertionを追加し、意図した拒否経路を検証する。統合harnessとfull raceは成功し、snapshot検証もartifact実byte数、保持node識別子、永続runのpassed状態まで確認した。

- 2026-09-09: 短いsecretのredactionでprovider上限内の文字列がAX fieldとsemantic JSON上限を超え、大きな回帰ケースは従来のapp 2 MiB guardも超えた。2048 nodeちょうどの単一/複数frameや末尾空frameが誤ってtruncatedになっていた。native closed root clickはhost.shadowRootを参照できず修正前に失敗した。遅いdomain enableで20 ms captureが500 ms超になり、省略console引数も完全な証拠と表示された。既存bulk-eventテストはenable時間の除外を前提としていたため、event件数とassertionを維持してenable処理を含む時間予算へ更新した。


- 2026-09-09: 859ca74の新しいnative CIでfocus転送回帰テストがmacOSとWindowsのpush run 34251804805・PR run 34251809149の両方で失敗した。Linuxは成功した。key操作が不確定拒否でなく成功を返しており、tab focus/event dispatchを調査する。テストやsandbox条件を緩めず、受け入れは未完とする。

- 2026-09-09: 実ChromeでURL mockの不足が判明した。Page.Frame.urlにfragmentは含まれずurlFragmentで別返却される。native query waitは成功したがfragment waitはtimeoutした。frame decode・一時条件評価・document identityへurlFragmentを追加し、commit前にnative検証を再実行する。

- 2026-09-09: 統合docs-checkで再開した日本語版のtranslation_ofがcompleted/を参照していたため拒否された。メタデータを修正し、日英の内容を確認した。文書検証が未完の段階でfull raceは成功した。

- 2026-09-09: 独立レビューで、入力後のJavaScript readback例外やboolean欠落も不確定状態を保つ必要があると判明した。例外なし・明示booleanの検証を追加した。appのartifact回帰テストもrun.jsonの登録を必須確認し、保存検証が空振りしないようにした。


- 2026-09-09: 第2回PR reviewで、effect後の確認状態、focus証明、DOM origin範囲、別tabのtopology、redaction後のサイズ制限、raw URL条件、capture queueのtruncation、保存manifest digestの不足が判明した。Planを再開し、以前のCI成功は今回の修正の受け入れ証拠として扱わない。最初のlocal harnessは新規回帰テストのformatで停止したため、整形後に再検証する。

- 2026-09-09 — `3d3fce5`のpush Browser native 34246852839は3 OSすべて成功したが、
  PR Browser native 34246856039のWindowsは最後のprofile削除で共有違反に失敗した
  （`Cache_Data/sqldb0`）。全browser操作と新しいsandboxアクセスlog検査は成功していた。
  process Destroyはnative tree不在と所有pathを確認済みで、RemoveAllを1回だけ呼んでいた。
  logからfilesystemを保持していた主体は分からず、特定processやkernel要因と断定しない。
  別native runの成功があってもcleanup受け入れ失敗としてPlanをactiveに保つ。

- 2026-09-09 — 継承originの実証後、sandbox付き`srcdoc` iframeが`Page.getFrameTree`に
  現れず、選択pageのparent IDを持つiframe targetとしてのみ存在することが分かった。
  rootだけのsnapshotを返すと、完全な観測であると誤って示してしまう。target一覧で関連OOPIF
  と所属を証明できないiframe targetを拒否し、その内容は調べない。
- 別の実navigationでは、child URLが空でorigin不明の状態を確認した。確定したopaque frame
  と異なり、waitの既存期限内で再観測できる。独立reviewでは、AX取得前だけのorigin検証では
  navigation後の文書に古いframe識別情報を付ける競合も見つかった。取得後のframe/tree/origin
  とtargetの整合確認で変化を検出したら、全証拠を破棄する。決定的な回帰testでこの変化を模擬する。

- 2026-09-09 — PR #10回帰testで修正前の失敗を確認した。URL待機はroleだけでも成功し、
  semantic入力はpage/snapshot/nodeの根拠を保存していなかった。AX上限で対象nodeが落ちても
  `gone`が成功し、DOM名の省略はtruncationに反映されなかった。network metadataにより
  65,536-byte予算に対して1,230,500 bytesの文字列を保持できた。未購読や別sessionのeventも
  512-event queueを埋め、通常commandを切断していた。
- CDP security originを厳密に判定すると、実browserのprotocol上の挙動が判明した。
  同一originを継承する`about:blank`でも、load後かつloader IDありで`securityOrigin="://"`
  を返す。新たなnative 3回反復はすべて失敗し、protocol mock成功だけでは不十分だった。
  継承frameにはURLによる例外許可ではなく、browserの同一origin制約に基づく追加確認が必要。
- Windowsの記録済み失敗は、両leaseでCfT実行ファイルへのLPACアクセス拒否と、続く
  network-service crashを含む。Chromiumのsandbox文書はinstallerまたは手動のACL設定を
  要求し、公式test helperはSID S-1-15-2-2へインストール先のread/executeを許可している。
  失敗runにはframe originを保存していないため、その後の観測拒否の正確な原因は未確定であり、
  新しいnative検証で確認する。

- Verify 34234714195のLinux integration/race jobは、
  `TestBrowserLifecycleGuards/dead`のfixture作成時に
  `sql: transaction has already been committed or rolled back`で失敗した。
  browserのassertion前の失敗で、Linux Chrome起動とは別に調査する。
  native Windows/macOSの成功だけで最終harnessの受け入れ成功とはしない。

- 2026-09-08 — Native CI [34234714187](https://github.com/mahcialet/agent-env/actions/runs/34234714187)
  で`9b94b42`修正後のmacOS・Windows受け入れが成功した。Linuxの起動診断では
  Chromeのfatal `No usable sandbox!`が確認でき、展開したChrome for Testingに対する
  UbuntuのAppArmor user namespace制限に該当した。最初のreadiness errorだけでは
  このhost前提条件が分からなかった。
- Verify [34233867022](https://github.com/mahcialet/agent-env/actions/runs/34233867022)
  で`506ed32`の失敗したmacOS readiness jobだけを再実行し、成功した。
  この再実行ではcode・timeout・assertionを変更していない。CI修正後の全local raceも
  成功した（CLI 4.798秒）。`9b94b42`の独立レビューで追加不具合は見つからなかった。
  最終revisionでのCIは引き続き必要。

- 2026-09-08 — 最初の公開CI（`506ed3286e66fe0602c3d68189cbca5a13164dc6`）では
  3 OSともnative browser受け入れに失敗した
  （[run 34233867023](https://github.com/mahcialet/agent-env/actions/runs/34233867023)）。
  macOSはChrome 152.0.7977.82 / CDP 1.3まで到達したが、空文字列への置換の一致検証に失敗。
  Windowsも同versionまで到達したが、保存console証拠のprivacy検査に失敗。
  Linuxはbrowser HTTP readinessに失敗した。原因を調査中であり、platformの成功証拠ではない。
- 最初のVerifyでも、macOS / Go 1.26.7の既存`TestHTTPReadinessAndObservedHealth`が失敗した。
  成功応答を確認する段階で既存の20 ms probe timeoutを超えた。readiness実装とtestは
  開始revisionから変更していない。local raceで30回反復成功したが、macOSでの成功証拠ではない。
  失敗履歴を保持し、最終CIの成功を確認する。

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

- 2026-09-09: redactionとapp metadata設定後にsemantic snapshotを再制限する。長いAX name/valueを置換し、node数とJSON encoding 1 MiB以内の最大node prefixを維持する。参照/fingerprintを保持してtruncatedを明示し、縮小証拠による入力を拒否する。成功したread-only観測でrunning barrierを残さない。closed shadowは隔離worldのbackend nodeから最大128 rootを外向きに辿り、各hostでoverlayを検証してhit/focusを証明する。capture時間は購読/domain enableを含み、期限までにenableが終わらなければ失敗する。console object詳細は保存せず省略をtruncatedとする。AX node上限は実際の省略nodeがある場合だけtruncatedとする。


- 2026-09-09: keyboard/textのfocus前に対象pageを前面化し、page handlerが動く可能性があるためsnapshot identity・node・hit検証を再実行する。隔離worldでdocument.hasFocus()と対象のfocus一致を必須とし、前面化後の失敗は不確定状態を保つ。非active pageという仮説に対応するが、native focus転送のassertionやsandboxは緩めない。

- 2026-09-09: protocol errorだけでは複数段階の操作全体を確認済みとしない。effect前の拒否と検証済み結果だけを確認済みにでき、focusもeffectとして扱う。focusとselectAllの後に隔離worldのnative active-element getterでfocusを確認し、不正readbackは不確定状態を保つ。
- 2026-09-09: AXとDOMで取得前後のframe検証を共用し、DOM文書frame IDを承認済みtopologyに限定する。親情報のないiframe targetは選択sessionのDOM frame ownerと照合し、対象pageの拒否条件を緩めず別tabを独立させる。
- 2026-09-09: redaction後の保存captureにも文字列・合計上限を再適用する。期限時点の未処理captureは購読解除と同時にtruncatedとする。browser検査前に保存manifest digestを検証し、URL条件はraw URLで一時評価しつつ証拠URLは秘匿化する。


- 2026-09-09 — 既存のnative不在証明後に限り、汎用process state cleanup内でWindowsの
  一時的な共有違反を扱う。そのOS errorだけを最大2秒・呼出し側cancel期限内で再試行し、
  毎回所有pathを再検証する。他のerrorと持続する共有違反は失敗のまま、既存の保持・隔離動作を
  維持する。Chromeのlifecycle処理追加、error無視、停止猶予の延長、既存の所有権・不在検査より
  前のresource解放は行わない。

- 2026-09-09 — frame判定はCDP SecurityOriginを優先する。Chromeがopaqueの代用値を返す
  継承`about:blank`/`about:srcdoc`は、検証済みの親のisolated worldでnative
  `contentDocument` getterを使いアクセスを証明する。`grantUniveralAccess: false`
  （CDPのparameter表記）を指定し、page realmのgetter上書きでは許可できない。
  512-target上限の一覧で選択pageのOOPIFと所属不明iframeを拒否し、外部targetを接続・
  採用しない。取得後もframeを再確認し、識別情報が変わった・得られない場合に部分証拠を返さない。
  未commitのoriginや取得中の文書変化という一時的なerrorだけを、waitの既存期限内で再試行する。

- 2026-09-09 — semantic入力前のCommandRunへpage ID・参照元snapshot run ID・node参照を
  保存する。不確定な結果とrun.jsonでも保持するが、set-text内容は保存しない。回帰providerは
  入力中にstoreを検査し、click/set-text/key/scrollの成功・不確定の両方で同じ根拠を検証する。
  URL待機は空でないsubstringを必須とし、roleのみ・role併用の要求を接続前に拒否する。
- capture queueはdomain有効化前に、1操作のsessionと対象eventだけを購読し、全returnで
  購読を解除する。無関係な通知は容量を使わず、購読対象があふれたら引き続き拒否する。
  console/networkのmetadataを含む全文字列で1文字列4 KiB・合計64 KiBを共有し、
  全体を置換して機密prefixを残さない。不完全なAX観測から消失を証明しない。
- WindowsのCfTインストール先だけに、制限付きapplication-package SIDのread/executeを
  許可する。[Chromium sandbox手順](https://chromium.googlesource.com/chromium/src/+/main/docs/design/sandbox.md)
  と`testing/scripts/common.py`の`set_lpac_acls`に従う。sandboxを無効化せず、user/profileや
  無関係な親pathへは許可しない。runnerだけの変更であり、local LinuxではWindows ACLを
  検証できないため、native CIで確認する。

- 2026-09-08 — browser fixtureが継承する50 ms readiness期限を、検査対象の操作に
  限定する。fixtureの`Create`時だけ5秒を許可し、browserのassertion前に元の設定へ戻す。
  `waitReady`はreadiness contextでleaseを保存するため、期限切れによるSQLiteの非同期
  rollbackが`ErrTxDone`として現れる経路がある。CIはprocessをdeadにする前のfixture
  作成中に失敗した。このため準備用期限による失敗が考えられるが、正確なscheduler timingは
  取得できていない。変更前のlocal race 10回反復は成功（20.980秒）し、local再現は
  できなかった。製品のreadiness/lockや、timeout・death・contention・cancellation・
  fence lossの専用assertionは変更しない。

- 2026-09-08 — 使い捨てのUbuntu CI runnerで、固定versionのdownload済みChrome
  実行ファイルのpathだけに適用するAppArmor profileを用意する。Chromiumが文書化した
  user namespace利用許可に従い、host全体の制限を保ったままChrome sandboxを使えるようにする。
  全体のsysctl制限緩和や`--no-sandbox`は採用しない。host準備はUbuntu workflow内に
  限定し、core Goの動作とWindows/macOS実行は変更しない。

- 2026-09-08 — 非公開の全選択keydownでCDPの明示的な`selectAll`編集commandを送る。
  native Inputと入力後の一時的な一致検証を維持する。macOSではplatform shortcutだけでは
  空でないtextを確実に全選択できなかった。Linux/macOS/Windowsのkeydownとkeyupの
  protocol上の動作を回帰testで検証する。
- Windowsのnative process birth proofには意図的にUnicode guardian pathが含まれる。
  広い`日本語`部分文字列検査は、その信頼されたpathを入力textと誤認した。
  固有の入力prefixを使用し、decodeしたJSON文字列から実際の入力Unicode文字列
  （引用符とbackslashを含む）を検査する。escape済みsecretを検出し、正当なproof pathは
  許容する回帰testを追加した。実装側のredactionと受け入れ要件は変更しない。
- native test失敗時、lease cleanup前にprocess診断を取得してLinuxのChrome起動の証拠を残す。
  readiness timeoutだけを根拠にsandboxを無効化したりhost policyを変えたりしない。

CI修正checkpoint: adapter race test成功（1.415秒）。全選択修正後のLinux実browser
受け入れは成功（8.762秒）。強化したprivacy検査と失敗時診断も成功（8.243秒）。
macOS/WindowsおよびLinuxのnative CIは未完了。既存のmacOS readiness test失敗は、
同じ公開commitで再実行中。timeoutやreadiness assertionは緩めていない。

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

第3回レビューの時点（2026-09-09）: 新規5件のため本Planを再開した。以下の過去の完了・native結果を今回の受け入れ証拠として扱わず、新しい統合検証とnative CIの成功を必要とする。


第2回レビュー完了（2026-09-09）: 8件すべてを859ca74とcdcec91で修正した。effect後のエラーで不確定状態を保持し、前面化・選択後のdocument/target focusを証明する。DOM origin/topologyと別tab境界を検証し、redaction後もcapture上限を維持してqueue省略を明示する。query/fragmentを一時URL条件へ含めつつ保存証拠を秘匿化し、保存manifest digestを検証する。cdcec91の新しいCIは上記全gateで成功した。859ca74のmacOS/Windows native focus失敗は過去の失敗として残し、成功には数えない。page前面化と再検証により強化nativeシナリオは3 OSすべて成功したが、OS/browserのevent配送機構自体は計測しておらず、当初の原因説明は仮説として扱う。回帰テスト・独立レビュー・実ブラウザの複数OS検証を組み合わせた。URL mockとLinuxだけのnative証拠では不十分だった。


以下は過去の第1回レビュー完了記録であり、今回の第2回レビュー完了は上記に記録した。

2026-09-09、PR #10 review対応を完了した。`3d3fce5`でoriginの証明、wait条件、
省略・byte予算、event購読、永続的な入力対象の根拠を修正し、`391288c`でnative不在証明後の
Windows共有違反cleanupを上限付きにした。所有権・sandbox・privacyとprocess/CDPの責務境界は
維持した。回帰testで修正前の不具合を再現し、実browser testではmockで見つからなかった
継承originの代用値と省略されたOOPIFを確認した。独立reviewで取得後の識別情報検査を追加し、
navigation中の部分証拠を古いorigin/loader付きで公開しないようにした。

最終revision `391288c351dec3e41febcfec15d912c65905a3f0`で
[PR Verify 34247636419](https://github.com/mahcialet/agent-env/actions/runs/34247636419)の全12 job、
[PR Browser native 34247636411](https://github.com/mahcialet/agent-env/actions/runs/34247636411)の全3 OS、
[Release preview 34247636491](https://github.com/mahcialet/agent-env/actions/runs/34247636491)の
buildと3 OS native smokeが成功した。pushのVerify/nativeも成功。9 Threadすべてへ
具体的な根拠を返信しResolveした。今回の最終archiveは文書だけの変更で、Windowsの失敗runと
試行した失敗案は以下に履歴として残す。続く部分は初回milestoneの振り返りである。

2026-09-08、`feat/browser-cdp-automation`で完了した。既存の常駐process runtimeの上に、
明示的なChromium-CDP bindingを実装した。processの所有・寿命管理はruntimeに残し、
adapterは上限付きCDP通信、page、AX/DOM観測、PNG screenshot、型を限定した入力、
console/network取得を担う。操作前にnative PID・command line・profileの一致とleaseの
operation fenceを検証する。登録snapshotはlease・browser・page・document・nodeに結び付け、
古い参照や曖昧な参照では過去の座標を再利用せず、操作を再送しない。set-textはUnicodeと
明示的な空文字消去に対応し、一時的な読戻しで一致を確認する。保存したfingerprintにより、
後の別CLI processでも入力textをredactする。

最終実装・test revision `b48ab643a3e01029d880122b3c7c6830d82ed325`で
[Verify 34235476057](https://github.com/mahcialet/agent-env/actions/runs/34235476057)の
全12 job（全race・integrationを含む）と、
[Browser native 34235476126](https://github.com/mahcialet/agent-env/actions/runs/34235476126)の
3 OSすべてが成功した。Chrome 152.0.7977.82 / CDP 1.3をLinux/amd64、Windows/amd64、
macOS/arm64で検証した。local harness・全race・実Docker integration・6 target build・
sandbox有効のLinux browser反復も成功した。各受け入れ行と日付付きcheckpointに、
実行した検証と証拠の限界を記録した。

独立reviewでは、fence loss時の早期returnで未redactの値を返す問題と、汎用JSON redactionが
操作の根拠となる識別情報まで壊す問題を発見し、修正と回帰testを追加した。native CIでは、
macOSの全選択、WindowsのUnicode proof pathによる誤検出、Ubuntuのsandbox前提条件が
明らかになった。明示CDP編集command、decode後のsecret検査、対象を限定したrunner設定で
解決した。browser fixtureの準備期限も、検査対象の操作期限から分離した。protocol mockと
local成功だけではnative受け入れを代替できないことが分かった。以前の失敗と、変更なしで
成功した既存readiness testの再実行は履歴に残す。製品のtimeout・fence・sandbox・privacy
要件は緩めていない。

制限は明記した。same-origin iframeとshadowの観測には対応するが、iframe入力と
cross-origin/OOPIF観測には対応しない。外部browser接続、公開script/CDP実行、download、
browser自動再起動、Android/browser共通UI層は追加していない。network header/bodyと
DOMの編集可能な値は保存しないが、screenshot・未知のpage text・専用profileには機密が
含まれ得る。この機能は暗号化やbrowser sandboxの境界を提供しない。英日Planをarchiveし、
現在の文書からの参照も更新した。

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
| B25 | Native Windows | 成功：`391288c`、PR Browser native 34247636411、windows/amd64、Chrome 152.0.7977.82 / CDP 1.3、実CLI fixture 26.76秒。sandboxアクセスguardと通常cleanupも成功。 |
| B26 | Native macOS | 成功：`391288c`、PR Browser native 34247636411、darwin/arm64、Chrome 152.0.7977.82 / CDP 1.3、実CLI fixture 9.60秒。 |
| B27 | Linux real headless pass | 成功：local sandbox有効Chrome 152.0.7977.64 / CDP 1.3、最終race native package 8.834秒。`391288c`のPR native 34247636411、Chrome 152.0.7977.82 / CDP 1.3、linux/amd64 test 8.86秒。 |
| B28 | 実機fixture機能 | `391288c`の3 OS native fixtureですべて成功（PR Browser native 34247636411）。継承blank/srcdoc/blob観測とopaque/OOPIF拒否を含む。iframe入力とcross-origin観測は引き続き非対応。 |
| B29 | provider非依存lease backend E2E | native fixtureでrepository所有HTTP backendを別process runtimeとしてbuild。Unicode request/count/log相関と別lease backendの不変を確認。Compose provider不使用。 |
| B30 | Browser未使用時coreにbrowser不要 | browser integration tagなしでcore unit/raceと6 CGO-free CLI cross-build成功。browser前提は明示的browserintegration test/commandだけに適用。 |
| B31 | Node/Python/Playwright/Selenium/ChromeDriver runtime dependency無し | Go gorilla/websocket transportと直接native argvを使用。architecture check成功、helper runtime/browser同梱なし。 |
| B32 | 英日durable docs final behavior/privacy | 製品・設計・関連文書を英日更新し、正確なflag、native前提、privacy/上限を記載。意味確認後hash更新、docs-check成功。 |
| B33 | 最終検査 | 成功：`391288c`のPR Verify 34247636419（12 job、全race/integration）、PR Browser native 34247636411（3 OS）、Release preview 34247636491（build/native smoke）、対応するpush CI、local harnessと英日文書review。 |
| B34 | 生きたplanと証拠 | 成功：PR #10 review gateとB1–B34を照合。9 Threadすべてへ返信・Resolveし、失敗履歴を保持。英日成果を更新し、最終受け入れ後に両Planを再archive。 |

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

2026-09-08 CI checkpoint: `a37f11f`で3 OSすべての実browser jobが成功
（[Browser native 34235169459](https://github.com/mahcialet/agent-env/actions/runs/34235169459)）。
Ubuntuの限定したAppArmor許可を含め、sandboxとhost全体の制限は有効。
browser fixture準備時だけreadiness期限を変更した状態で、全`TestBrowser*`の
local race 10回反復も成功（20.222秒）。全Verifyは未完了。

最終native証拠: `b48ab643a3e01029d880122b3c7c6830d82ed325`の
[Browser native 34235476126](https://github.com/mahcialet/agent-env/actions/runs/34235476126)で
3 jobすべて成功。Chrome 152.0.7977.82 / CDP 1.3、Go 1.27。
Linux/amd64のnative fixtureは8.16秒、Windows/amd64は29.97秒、macOS/arm64は19.62秒。
JSON privacy検査の回帰testも3 OSで成功した。この結果は以前のnative未完了checkpointを
更新するもので、失敗履歴は消さない。

最終完了記録（2026-09-08）: `b48ab64`のVerify 34235476057は全12 jobで成功した。
Browser native 34235476126と合わせ、以前の日付付きcheckpointに残っていた受け入れgateは
すべて完了した。この検証済みrevisionからの最終変更は、英日文書の照合とPlanのarchiveのみ。
archiveの区切りではruntimeやtestの動作を変更していない。

2026-09-09 review checkpoint: appのURL/provenance対象raceは成功（2.056秒）。
app・CDP全raceも成功（36.887秒 / 1.735秒）。capture/transportの回帰はrace 10回反復
成功（4.392秒）。app/provenance・transport/capture・限定したWindows ACL準備の
独立reviewで追加不具合は見つからず、文書検査も成功した。originの証明とnative受け入れは
引き続き実装・検証中であり、Planをactiveに維持する。

2026-09-09、最後の取得競合guard追加前の統合checkpoint: 全`repoctl check`と
`go test -race ./...`は成功。sandbox有効のLinux Chrome 152.0.7977.64 / CDP 1.3で、
強化したnative raceを3回反復して成功（25.160秒）。実fixtureは継承blank/srcdoc、
同一origin blob、page realm getterの悪意ある上書きがあってもopaque OOPIFを拒否する動作を
検証した。最後の競合guardとWindows LPAC準備は、受け入れgate完了前に改めて検証する。

2026-09-09、review修正の最終local検証: 取得後の整合guardを含むCDP全raceは成功
（2.331秒）、sandbox有効のLinux実native fixtureも成功（8.457秒）。独立reviewで
取得競合の指摘解消を確認し、追加不具合は見つからなかった。統合後の最終全harness/raceも成功。
Windows fixtureは以前の実行ファイルsandboxアクセス拒否ログも検出・拒否する。
完了には新しい複数OS CIとThread返信が引き続き必要。

2026-09-09 cleanup checkpoint: process race成功（1.964秒）、cleanup/Destroyの
回帰race 10回反復成功（2.916秒）、Windows amd64/arm64 test binaryのcrosscompile成功。
Windows native testはdelete sharingなしで実fileを保持し、RemoveAllの共有違反を確認して
から解放し、cleanup成功を検証する。独立reviewで新たな不具合は見つからなかった。2秒の
予算は再試行の開始を制限し、実行中の同期filesystem呼出しを中断しない。最終local harnessと
強化したLinux native検証は成功。Windows実行と新しい全CIは未完了。

2026-09-09、最終native/release証拠（`391288c351dec3e41febcfec15d912c65905a3f0`）:
[push Browser native 34247632201](https://github.com/mahcialet/agent-env/actions/runs/34247632201)と
[PR Browser native 34247636411](https://github.com/mahcialet/agent-env/actions/runs/34247636411)が
ともに3 OSすべて成功。Windowsの通常profile削除とsandboxアクセスguardも成功した。
PRのChrome 152.0.7977.82 / CDP 1.3でのnative時間はLinux/amd64 8.86秒、
Windows/amd64 26.76秒、macOS/arm64 9.60秒。
[Release preview 34247636491](https://github.com/mahcialet/agent-env/actions/runs/34247636491)も
buildとnative smokeが成功。8 Threadは返信・Resolve済みで、Planのgateだけを最終Verifyの
成功確認まで開いている。

PR #10最終完了（2026-09-09）: review進捗と受け入れgateはすべて完了した。
`391288c`の最終Verify 34247636419とpush Verify 34247632059は成功し、上記nativeと
releaseも成功した。9 ThreadすべてResolve済み。この記録で以前の未完了checkpointを更新する。
英日archiveの区切りではsource/test codeを変更していない。
