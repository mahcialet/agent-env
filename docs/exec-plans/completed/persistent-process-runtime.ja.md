---
status: completed
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/completed/persistent-process-runtime.md
source_sha256: d5e403ec13c36413c993c7ea24fef0ae96673b5ccec5ee9e94138329907db56d
---

# leaseが所有する汎用の常駐process runtimeを追加する

[English（翻訳元）](persistent-process-runtime.md)

このExecPlanは作業中に更新する文書であり、`docs/PLANS.md`に従って維持する。

作業branch: `feat/persistent-process-runtime`。

PR #8（`feat: add explicit Podman Compose provider with safe lease cleanup`）は
実装開始前に`master`へmerge済みである。本PlanはPodmanの挙動に依存しないが、
Compose providerの既存挙動が変わらないことも確認する。

推奨する開始方法:

- PR #8がmerge済みなら、その結果の`master`からbranchを作り、Compose providerの
  既存挙動が変わらないことも検証する。
- 未mergeなら現在の`master`からbranchを作る。利便性だけを理由に、未mergeの
  Podman branchの上へprocess runtimeの作業を積まない。

実際のbaseを以下に記録する。

開始時のbase branch: `master`
開始時のrevision: `01e3581`（PR #8 merge）

## Purpose / Big Picture

この作業により、`agent-env`は長期間動作するnative host processを通常のenvironment
runtimeとして所有できるようになる。

repositoryは開発用API server、local test daemon、application server、将来のChromium/
browser processなどを、Composeで包んだり`agent-env`へ製品固有のlifecycle codeを
追加したりせずに宣言できる。

確定したmanifest構文:

```yaml
runtimes:
  api:
    type: process
    source: app
    working_directory: .
    command:
      - ./bin/api-server
      - --listen
      - 127.0.0.1:${port:http}
    ports:
      http:
        protocol: tcp
    env:
      APP_ENV: test
      PORT: ${port:http}

components:
  api:
    runtime: api
    endpoints:
      http:
        runtime_port: http
    readiness:
      - type: http
        url: http://127.0.0.1:${endpoint:http}/health
```

上記runtime/endpoint構文は確定した。HTTP readinessでは、宣言したlocal endpointの
`${endpoint:http}`を数値の予約portへ展開する。この参照はaddress全体やcomponent名付き
endpointではない。architecture上の契約は満たす必要がある。

process runtimeは次を満たす:

- 固定した管理対象sourceから起動する。
- shellによる解釈を行わずargv配列を直接実行する。
- 起動元の`agent-env create` CLI processが終了しても存続する。
- stdout/stderrをlease所有のfileへ書く。
- 永続化可能なnative process treeの識別情報を持つ。
- lease所有のruntime状態directoryを受け取る。
- 動的に割り当てたloopback TCP portを受け取れる。
- readiness、show/list、logs、reconcile、destroy、GCに参加する。
- 所有processが予期せず終了するとDEGRADEDになる。
- この実装範囲では自動再起動しない。
- native識別情報の再検証後にのみ停止する。
- process treeの所有または停止が不確実ならquarantineにする。

主な後続の利用側はBrowser/CDPである:

```text
process runtime
    +-- Chromium process ownership
    +-- private runtime/profile directory
    +-- allocated loopback CDP port
    +-- future browser observer
          DOM/ARIA snapshot
          screenshot
          navigation
          console/network
```

Browser対応のためにprocess寿命、detached実行、logs、状態directory、endpoint割り当て、
cleanupを再実装する必要がないようにする。

## Scope

対象範囲:

- 独立した`type: process` runtime。
- 厳密なcommand/working-directory/env/portのmanifest契約。
- argvのみの直接実行。
- source相対でsource内に閉じた作業directory。
- source相対またはPATHで解決する実行ファイルとその証拠。
- lease専用のruntime directory。
- 最終的に定める`${runtime_dir}`、`${lease_id}`、名前付きportの補間。
- 動的に割り当てるloopback TCP port。
- 共通component endpoint model内のprocess runtime endpoint。
- app層のreadinessの再利用。
- 永続stdout/stderr log。
- process作用前の永続的な起動意図。
- 起動後の永続native process識別情報。
- 起動元CLI終了後のprocess存続。
- 後続の独立したCLIからの観測。
- native識別情報を条件とする停止。
- 保守的な強制停止への移行と、不確実なtreeのquarantine。
- createの補償とdestroy/reconcile/GCへの統合。
- 操作fenceとheartbeatの挙動。
- 異なるport/状態directoryを持つ並行process lease。
- processとComposeの共存。
- Windows/macOS/Linuxでの実際のnative process test。
- processをまたぐcrash/復旧test。
- 英日product/design/ExecPlan文書とarchitecture/roadmapの更新。

対象外:

- shell command型（`sh -c`、`cmd /c`、PowerShell）。
- PTY/TTYまたは対話stdin。
- terminal multiplexer。
- agent-env外で起動したprocessの取り込み。
- 自己daemon化するprocessの対応。
- 自動再起動/supervisor方針。
- replica。
- CPU/memory/cgroup/jobのresource制限。
- user切り替えや権限昇格。
- systemd/launchd/Windows Serviceの導入。
- remote/multi-host process実行。
- 最初の実装でのUDP。
- socket activation/継承するlisten socket。
- 固定host port。
- stdout解析によるport検出。
- Browser/CDP自体の意味・挙動。
- 信頼できないcodeのsecurity sandbox。

## Architectural Intent

### Native process primitive versus runtime policy

repositoryには既に`internal/execx`のnative detached-process primitiveがある。
次の安全性を保つ:

- 常駐processをrequest contextの寿命に結び付けない。
- 出力はCLI pipeではなくfileへ書く。
- native識別情報はPID単独より強い。
- process treeの観測は起動元CLIの終了後も可能。
- Windows Job/process-groupの生成識別情報を所有証明に使う。

lease方針を`execx`へ移さない。

目指す層分離:

```text
internal/execx
    native Start / Observe / identity-gated Terminate

internal/runtime/process
    executable resolution
    argv/env/interpolation
    runtime directory
    stdout/stderr
    dynamic port values
    inspect/terminate

internal/app
    intent persistence
    port reservation
    readiness
    compensation
    reconcile/quarantine
    cleanup ordering
```

Android Emulatorは専用runtimeのままにする。このPlanで`type: process`へ移行しない。
`execx`を変更する場合も、既存のAndroid process testをすべて維持する。

### Foreground lifecycle anchor

runtimeはforegroundで動く起点processを1つ所有する。起点は子processを作ってよいが、
runtimeの寿命中は生存することを期待する。

自己daemon化は非対応。

子孫が残ったまま起点が終了した場合、所有を証明できる十分なnative証拠だけを使う。
系譜が曖昧になったらDEGRADEDまたはQUARANTINEDを返し、破壊的作用を止める。
数値IDが過去の記録と一致するだけでPID/process groupをkillしない。

platform差は保守的なままでよい。Unixで系譜が不確実になる場合でもWindows Jobならtreeを
証明できることがある。実際にはない共通の証明を作り上げない。

### No automatic restart

予期しないprocess終了は観測状態をDEGRADEDへ変える。最初の実装では自動再起動しない。
復旧手順はinspect -> safe destroy -> recreateとする。

### Native termination

現在のdetached primitiveは観測できるが、汎用的な停止方針は公開していない。
必要最小限の、native識別情報を条件とする停止interfaceだけを追加する。

概念例:

```go
type DetachedProcess interface {
    Start(context.Context, Command, stdoutPath, stderrPath string) (ProcessIdentity, error)
    Observe(context.Context, ProcessIdentity) (ProcessObservation, error)
    Terminate(context.Context, ProcessIdentity, TerminateOptions) (TerminationResult, error)
}
```

実際の型は異なってよい。

Unixでは所有を証明できる間にgroup/treeへ穏当な停止を要求し、期限付きで待ち、証明が
有効な間だけ強制停止へ移行することを優先する。起点が消えて所有が曖昧ならquarantineにする。

Windowsでは既存Job/guardian識別情報を再利用し、PIDだけをkillせず所有Job/treeを停止する。
atomicな完了証拠を保つ。

platformが証明できない穏当な停止の意味を約束しない。

### Command and executable identity

`command`は常にargvとする。shell展開、pipe、redirect、command連結は行わない。

`working_directory`はsource相対で、選択worktree内に閉じる。

argv[0]はsource相対の実行ファイルかPATHで解決する単純なhost tool名にできる。
取得できる範囲で次を記録する:

- redaction後の要求argv。
- 解決済み実行ファイルpath。
- source/hostの由来。
- 読み取れる通常fileの場合は実行ファイルのSHA-256。
- source commit。
- 作業directory。

pathを記録しただけでPATH実行ファイルに再現性があると言わない。
Windows `.bat`/`.cmd`はdetached native runtime実行で引き続き非対応とする。

### Runtime-owned mutable state

各process runtimeにlease状態root配下の専用状態directoryを与える。概念例:

```text
<AGENT_ENV_HOME>/leases/<lease-id>/process-runtimes/<runtime>/
    state/
    stdout.log
    stderr.log
    owner.json
    redaction.json
    launch.json
```

`${runtime_dir}`のような明示的な補間を公開する。将来のChromiumは
`--user-data-dir=${runtime_dir}/profile`を使える。

tree全体の不在確認後にのみ可変状態を削除する。解放後も残すべきlogsや証拠は、既存の
evidence/artifact storageに置く。

### Dynamic loopback TCP ports

process runtimeは名前付きportを要求できる:

```yaml
ports:
  http: {protocol: tcp}
  metrics: {protocol: tcp}
```

初期契約:

- TCPのみ。
- loopbackのみ。
- agent-envがportを割り当てる。
- 固定host portを拒否する。
- 起動前に割り当てを永続化する。
- 並行leaseへ同じagent-env予約を渡さない。
- 外部占有時は保守的に失敗する。
- socketを保持する設計を実装しない限り、任意の外部processに対して競合なく排他できるとは
  主張しない。

`${port:http}`は数値portへ解決する。

### Endpoint model

既存のcomponent endpointはCompose向けである。Compose fieldを空にして別の意味へ
流用せず、明示的に拡張する。

推奨例:

```yaml
endpoints:
  http:
    runtime_port: http
```

既存のCompose endpoint構文は有効なまま保つ。Compose service targetかprocess runtime
portかを選ぶ厳密なendpoint variantにする。

Flutter reverse mappingを含むapp層のendpoint利用側には、同じprovider非依存の解決済み
endpoint表現を渡す。

### Readiness and logs

native processの起動だけではREADYとしない。

app層のreadinessを再利用する。readiness前に終了したprocessはcreateを失敗させる。
生きていてもreadyにならないprocessはreadinessの期限で失敗させ、保守的に停止する。

detached stdout/stderrは起動時からfileへ書く。`agent-env logs`はruntimeに帰属する出力を
memoryへ無制限にためずに公開する。

### Persistence ordering

起動前:

1. sourceを展開する。
2. runtime directoryの識別情報を確定する。
3. 名前付きportを予約する。
4. 起動意図を永続化する。
5. 出力pathを確定する。

起動後:

6. readinessが依存する前に、返されたnative識別情報を永続化する。
7. 起動証拠を確定する。
8. readinessへ進む。

process起動後の永続化に失敗した場合も、返された識別情報を保持し、所有を証明できる間だけ
補償する。

## Progress

- [x] (2026-09-08) base branch/revisionを記録し、`feat/persistent-process-runtime`を作る。
- [x] (2026-09-08) baselineのrepository harnessとrace suiteを実行する。
- [x] (2026-09-08) native detached codeとAndroidの既存挙動の検証範囲を調べる。
- [x] (2026-09-08) app/config/domain/store/endpoint/readiness/log/cleanup経路を調べる。
- [x] (2026-09-08) 英日product/design文書を書く。
- [x] (2026-09-08) processのmanifest/endpoint/補間契約を確定する。
- [x] (2026-09-08) 必要に応じ、識別情報を条件とするnative停止primitiveを追加する。（3 OSのnative証拠は下記。）
- [x] (2026-09-08) 不正・不一致の識別情報を与えたときの観測・native停止を検証するテストを追加する。（3 OSのnative証拠は下記。）
- [x] (2026-09-08) Android detached/guardian testをすべて通し続ける。（3 OSのnative証拠は下記。）
- [x] (2026-09-08) 厳密なprocess runtimeのconfig/domain型を実装する。
- [x] (2026-09-08) 実行ファイルと作業directoryの解決を実装する。
- [x] (2026-09-08) runtime所有の状態directoryを実装する。
- [x] (2026-09-08) 名前付き動的loopback TCP port予約を実装する。
- [x] (2026-09-08) 起動意図/証拠とStartを実装する。
- [x] (2026-09-08) readiness、endpoint、show、logsを統合する。
- [x] (2026-09-08) Inspect/reconcileと予期しない終了時のdegraded化を実装する。
- [x] (2026-09-08) Destroy/停止/補償/GCを実装する。
- [x] (2026-09-08) 並行する2つのprocess leaseの分離を追加する。
- [x] (2026-09-08) processとComposeの共存fixtureを追加する。
- [x] (2026-09-08) 別々のCLI起動を通じたprocess間復旧を追加する。
- [x] (2026-09-08) PID再利用と、子孫が残る起点終了時の安全性を確認するテストを追加する。
- [x] (2026-09-08) 自動再起動しないことを証明する。
- [x] (2026-09-08) native Windows/macOS/Linuxの常駐process integrationを追加する。
- [x] (2026-09-08) 割り当てたloopback endpointを使う実HTTP helperを追加する。
- [x] (2026-09-08) Browserを模したCDP状状態directory/portのfixtureを追加する。
- [x] (2026-09-08) architecture/portability/reliability/security/quality/roadmapを英日更新する。
- [x] (2026-09-08) 最終harness/race/native/cross-build検証を実行する。
- [x] (2026-09-08) 直接の受け入れ証拠と英日retrospectiveを完成させる。
- [x] (2026-09-08) 両Planを`docs/exec-plans/completed/`へ移す。


check済みは観測した完了を意味する。UTC日付、revision、正確なtestまたはworkflow runと結果を記録する。

### 最終受け入れ記録（2026-09-08）

`f58896050b11481b021bf1ad07701818d3c133eb`の
[Verify run 34226859965](https://github.com/mahcialet/agent-env/actions/runs/34226859965)
は全12 job成功。Windows/macOS/Ubuntu × Go 1.26/1.27の6 native jobで
`repoctl doctor`、`repoctl check`、CLI buildを実行した。
`TestPersistentProcessNativeCLI`、managed-process、Android detached/guardianの既存動作を確認するテストを含み、
Windowsでは`TestManagedWindowsCompletedJobIgnoresReusedHistoricalPID`も実行した。
integrationの`go test -race ./...`と`repoctl test-integration`はいずれも成功。
CIの5 cross-build jobも成功した。別途localの6 target CGO無効buildの証拠はコンパイルのみを
示す。この結果により以前のcheckpointで残っていたnative gateを満たした。
初回失敗runは過去の証拠として下記に保持する。

### 統合時点の記録（2026-09-08）

以下は`01e3581`ベースの統合作業treeの結果であり、最終commit/native CIの成功を意味しない。

- 編集開始後に未変更baseを隔離worktreeへ再構成し、`go run ./tools/repoctl check`と
  `go test -race ./...`が成功した。baselineの証拠であり、実装前に全baselineを実行した
  という意味ではない。
- 統合版`go run ./tools/repoctl check`はreadiness secret修正後も全段階が成功。
  `go test -race ./...`も成功（app 53.504s、CLI 7.749s、execx 8.255s、SQLite 20.320s）。
  H22/H24 fixture追加後の最終local checkも成功（native CLI 4.085s）。
  native CLIの対象raceも成功（5.592s）。
- Linuxで`go test ./internal/cli -run '^TestPersistentProcessNativeCLI$' -count=1 -v`
  が成功（5.422s）。独立したbuild済みCLI起動、symlink home、日本語/空白path、異なるleaseの
  port/profile、実CDP状HTTP、手動起点終了/degraded化/再起動なし、保持log、destroy再実行を
  検証した。readiness中にcreate CLIだけをkillし、隔離test自身の操作lockだけを期限切れに
  して、native processが生存したまま通常復旧を検証する。中断割り当てはdegradedのままで、
  showが誤って完了扱いにはしない。
- 同じnative testは稼働中にtracked READMEを変更する。非force destroyはquarantineにし、
  file bytesと両HTTP serverを保つ。forceはtracked diffを記録してから解放する。
  全lease外の直接host helperにはkernel割り当てportと専用profileを与え、全leaseのcleanup/
  拒否経路を通じて正確なPID/profileの存続を確認した後、fixture自身が明示的に停止する。
- 実Dockerの`TestIntegrationPersistentProcessComposeCoexistence`成功（55.965s）。
  全`go run ./tools/repoctl test-integration`も成功。このcommandではAndroid hardwareと
  Podman opt-in suiteは設計どおりgateされたままである。
- 別途`AGENT_ENV_PODMAN_INTEGRATION=1`と`AGENT_ENV_PODMAN_DOCKER_COEXISTENCE=1`で
  `TestPodmanIntegrationConcurrentLeasesAndEvidence`が成功（150.407s）。
  rootless PodmanとDockerとの既存の共存動作を確認する証拠となる。
- `go run ./tools/repoctl doctor`と`agent-env doctor --runtime process --output json`が
  成功。process診断はComposeに依存せずnative detached対応を報告する。
- config/domainの検査、禁止された依存関係を検出するarchitectureテスト、繰り返したdocs-checkが成功。
  reviewで確定した5件に同じ不具合を検出するテストを追加して修正した。readiness中の終了、起動前と証明
  できる失敗、予約前のpath正規化、endpoint別名の優先、literal readiness認証情報のsnapshot
  混入である。他の実装担当のlifecycle/backendを評価したreviewは独立したものだが、
  config/docsの作者による自身の確認は独立reviewには数えない。
- native Windows/macOS実行と最終native CIは未完了。cross-buildは追加証拠のみであり、
  これらのgateが通るまで本Planはactiveのままとする。

### 最初の公開native CI checkpoint（2026-09-08）

実装commit `fb0d22ea9eb64798c8f70c57c69e0166aa90ccc7`をpushした。
[Verify run 34225937603](https://github.com/mahcialet/agent-env/actions/runs/34225937603)
は12 job中11成功で終了した。native macOSとUbuntuのGo 1.26/1.27、Windows Go 1.27、
全cross-build job、integrationが成功し、Windows Go 1.26 job `102059952028`が失敗した。
workflow全体の成功ではない。

失敗は`TestPersistentProcessNativeCLI`の329行目で、手動終了した2番目のprocessをdestroy
した際の`detached job root identity reused or ambiguous`だった。正確なnamed Jobの完了
証拠より先に過去の起点PIDを検証し、managed観測もtree全体の不在証明後に起点を検査していた。
修正と、PID再利用時の観測・停止を検証するnativeテストを実装中である。完了した所有Jobの証拠を再利用された過去のPIDより優先
しつつ、active Jobの識別情報検査を弱めてはならない。この失敗を証拠として保持する。
再試行成功や修正commitのnative成功はまだ主張しない。

localの`CGO_ENABLED=0` CLI cross-buildはWindows/macOS/Linuxのamd64/arm64、全6対象で
成功した。これはcompileのみの結果である。macOSにはnative CIの直接証拠が得られたが、
最終受け入れはWindows修正とそのnative検証を待つ。本Planはactiveのままとし、最終
retrospective/archiveは未完了である。

## Surprises & Discoveries

- 2026-09-08: 提供された英語Planには必須の日本語版がなかった。初回docs-checkは翻訳不足/
  link不足で失敗した。契約milestoneで全文の日本語訳を追加した。
- 2026-09-08: 既存HTTP readinessにはendpoint補間がなかった。process専用の
  `${endpoint:localName}`は数値予約portへ解決する。configは宣言済みlocal endpointを確認し、
  URL構文検査時だけ仮の数値へ置き換える。process以外の既存readiness挙動は変えない。
- 2026-09-08: Unixの生成/group観測とgroupへのsignal送信はatomicではない。各signalの
  直前に再検証し、foregroundの起点やtree証明が不確実になったら以後の破壊的操作を行わない。
  これはnative platformの制限であり、PID/group所有が競合なく証明できるという意味ではない。
- 2026-09-08: portableなdirectory名には、既存の汎用識別子pattern以上の条件が必要。
  process runtime名はWindows予約device名、末尾のdot、大文字小文字の衝突も拒否する。
- 2026-09-08: config/domainと英日product/design契約はlocal検証済み。native primitive、
  app/store lifecycle、adapter、Linux integrationは統合検証が成功した。native Windows/macOS受け入れと
  最終CIは未完了。

起点終了後のprocess識別挙動、Unix/macOSのprocess group制約、Windows Job/guardian停止、
実行ファイル解決の差、動的port競合、source変更、子を持つreadiness失敗、自己daemon化、
log寿命、状態directory cleanup、Browser状の複数process挙動を記録する。

testを通すために、不確実なprocess所有を誤ってclean状態にしない。

- 解放済み予約を復活させない検査が、正当な反復destroyや解放後の異常観測も拒否した。予約解放済みの記録を観測状態とは独立に保存し、`Desired=released`の更新ではportを再取得しない。
- 現在のhost環境だけでlogをredactすると、起動後に環境変数が変わった場合に起動時secretを除去できない。plaintextではないfingerprintを起動前にidentity receiptとは別に保存し、receipt保存失敗時も安全な診断収集とidentityに基づく補償を可能にする。


- 2026-09-08: 初期の固定範囲port割り当ては動的契約に反し、先頭portの占有に繰り返し遭遇
  し得た。SQLite commitまで保持するOSの`127.0.0.1:0` listenerへ置き換えた。
  listenerを閉じた後の外部bind競合は明記したままとし、起動時に保存識別情報を変更しない。
- 2026-09-08: 成功するprobeより先にprocessが終了しても誤ってREADYになれた。
  probe/application起動後に最後のnative所有観測を行うようにした。型付き
  `ErrProcessNotStarted`とPID 0で起動前の失敗を証明し、preparedへ戻す。
  証明のない識別情報0の結果は不確実なまま扱う。
- 2026-09-08: 不変の証拠pathを予約した後にPrepareがsymlink homeを正規化し、SQLiteが
  拒否した。不変条件を弱めず、予約前に`CanonicalFuture`でpathを解決するようにした。
- 2026-09-08: 生のruntime port keyが別portを指すcomponent別名を上書きできた。
  宣言した別名を優先するようにした。
- 2026-09-08: process readinessのliteral継承secretが永続snapshotへ入り得た。
  snapshot公開前に拒否するように検証を追加した。明示host参照は引き続き使え、出力を
  redactionする。

- 2026-09-08: 公開CIで、Linux/macOSとWindows Go 1.27が成功していても、Windows
  Go 1.26ではJob完了後に起点PIDの再利用/曖昧性で失敗した。過去の起点検索よりtree完了証明を
  優先する必要がある。active所有検査は厳密なまま保ち、対象platformの修正と同じ不具合を検出するテストを実装中である。

## Decision Log

- 決定: 独立した`type: process` runtimeを導入する。
  理由: 長期間動くhost processもCompose/Android resourceと同じlease lifecycleへ参加させる。
  日付/著者: 2026-09-08 / maintainers。

- 決定: 低水準primitiveとして`execx`のdetached native識別情報を再利用する。
  理由: 既にCLI終了後も存続し、対応native OSでPID単独より強いtree識別情報を使っている。
  日付/著者: 2026-09-08 / maintainers。

- 決定: このPlanではAndroid Emulatorを`type: process`へ移行しない。
  理由: AndroidにはAVD/ADB/console固有の所有管理と検証済みcleanupがある。
  日付/著者: 2026-09-08 / maintainers。

- 決定: commandはargvの直接実行のみとする。
  理由: portability、監査可能性、shell不要の不変条件を保つ。
  日付/著者: 2026-09-08 / maintainers。

- 決定: 初期process契約はforegroundの起点を必要とし、自己daemon化には対応しない。
  理由: 後続CLIとcleanupが所有を証明できる必要がある。
  日付/著者: 2026-09-08 / maintainers。

- 決定: 最初の実装では自動再起動しない。
  理由: 再起動はresource識別情報を変えるsupervisor方針であり、基本lease lifecycleではない。
  日付/著者: 2026-09-08 / maintainers。

- 決定: process portは動的割り当てのloopback TCPとし、固定host portは非対応。
  理由: 並行leaseの分離と将来のBrowser/CDPには安定したlocal動的endpointが必要。
  日付/著者: 2026-09-08 / maintainers。

- 決定: 各process runtimeへ専用の可変runtime directoryを与える。
  理由: Browser/dev serverには、固定worktreeを汚さない書き込み可能な状態が必要。
  日付/著者: 2026-09-08 / maintainers。

- 決定: 停止は保守的に行い、識別情報を条件とする。
  理由: 再利用PIDや曖昧なtreeをkillするよりquarantineが安全。
  日付/著者: 2026-09-08 / maintainers。

- 決定: 永続文書とこのExecPlanを英日で維持する。
  理由: repositoryの文書方針に従う。
  日付/著者: 2026-09-08 / maintainers。

- 決定: `working_directory`、argvの`command`、`env`参照、`ports.<name>.protocol: tcp`、
  endpointの`runtime_port`を確定する。非互換fieldはYAML null/空値、merge/aliasも含め、
  存在で拒否する。command/envの参照はruntime_dir、lease_id、port、host-envのみ。
  argv[0]/cwdはliteralとする。source path補間やTCP probeは追加せずHTTP/commandを再利用する。
  理由: 明示variantで既存Composeのcanonical bytesを保ち、汎用process契約を小さくportableに保つ。
  日付/著者: 2026-09-08 / maintainers。

- 決定: shell command型と暗黙のshell実行は引き続き非対応だが、明示的に選んだnative
  interpreterの実行ファイル名を一律拒否する規則は追加しない。`.bat`/`.cmd` wrapperは拒否する。
  理由: 不変条件はargv直接実行である。interpreterの一律制限は本Plan外の追加公開方針になる。
  日付/著者: 2026-09-08 / maintainers。

- 決定: 既存detached primitiveの上にmanaged native process interfaceを置く。
  Unixは生成識別情報とforeground group証明を使い、Windowsは正確なnamed Job handleを
  再検証から停止まで保持する。
  理由: OSの仕組みをapp方針から分離し、Androidのdetached挙動を保つ。Windows Job停止は
  console型の穏当な停止を約束しない。
  日付/著者: 2026-09-08 / maintainers。

- 決定: 実行ファイルのsource/host由来、絶対path、読み取れる通常fileのSHA-256を証拠として
  保持する。起動前にportを永続予約し、cleanupが不確実ならport/専用可変状態を保持する。
  restart fieldは設けない。
  理由: 証拠はhost toolの再現性ではなく、SQLite予約は外部bind競合を防げない。
  将来のBrowser/CDPはprocess adapterへBrowser方針を入れず、このlifecycleを利用する。
  日付/著者: 2026-09-08 / maintainers。

- 決定: 動的OS portのlistenerを予約commitまで保持し、予約前に証拠pathを正規化する。
  診断用endpoint keyより宣言済み別名を優先する。
  理由: 不変の識別情報や共通利用側へ渡すendpointの意図を弱めず、固定範囲の仮定をなくす。
  日付/著者: 2026-09-08 / maintainers。

- 決定: version付きsecret fingerprintを独立した`redaction.json`へStart前に保存し、
  `launch.json`には所有情報とnative識別情報だけを保存する。
  理由: 平文を永続化せず、host secret変更や起動後receipt保存失敗があってもredactionを
  継続する。証拠がなければ出力を拒否する。fingerprintと未加工logも専用storageで保護する。
  日付/著者: 2026-09-08 / maintainers。

- 決定: 過去の起点PID検索より先に正確なWindows named Jobを観測する。空/不在Jobには
  一致する同期済みguardian完了証明を要求する。active Jobの生成/所属検査は保ち、PID観測中に
  完了した場合は再調査する。完了証明後に過去PIDへsignalを送らない。
  理由: 初回native CIで完了後のPID再利用/曖昧性が判明した。Job完了証拠に基づけば、
  所有検査を弱めず安全にcleanupを繰り返せる。
  日付/著者: 2026-09-08 / maintainers。

## Outcomes & Retrospective

2026-09-08完了。H1〜H33の直接の受け入れ証拠を下記に記録し、英日Planを一緒に
archiveした。実装は`fb0d22e`、Windows完了判定の修正は`f588960`であり、後者の
Verify run全体が成功した。

提供した`type: process`契約はargv直接実行、範囲を制限した`working_directory`、名前付き
TCP port、`runtime_port` endpointを使う。config/domain、SQLite予約、process adapter、
app調整、`execx.ManagedProcess`の責務を分離した。専用可変状態、file出力log、実行ファイルの
由来/digest証拠、起動前secret fingerprintにより、平文認証情報を永続化せずに、後続の独立CLI
から観測と安全な診断を行える。永続化した意図/receiptで中断したcreateを復旧する。
予期しない終了は再起動せずdegradedとし、cleanupが不確実ならport、状態、worktreeを保持する。

3 OSのnative testで、並行lease分離、HTTP/CDP状readiness、作成CLIの中断、log保持、
source変更保護、無関係processの存続を検証した。Unixではsignal直前の生成/group証明を要求するが、
観測からsignalまでの競合は残り、子孫の所有が曖昧ならquarantineする。Windowsは正確なJobを停止し、
完了には一致するguardian証明を要求する。初回Windows CIでJob完了後の過去PID検索の問題が判明した。
2 processを使うテストで過去識別情報の再利用を模擬し、無関係processの存続を検証する。
kernelのPID再利用を強制したtestではない。独立レビューとnative CIにより、Linuxのみの成功や
cross-buildでは確定できなかった問題を検出できた。

受け入れの障害は残っていない。外部port bind競合とnative所有証明の限界は文書化した制約として残る。
自己daemon化、PTY、自動再起動、Browser自体の機能は対象外。Browser/CDPの観測はこのprocess
lifecycle、専用profile状態、endpoint契約の上に構築し、Androidは独立したまま扱う。

## Context and Orientation

実装前に読む:

- `AGENTS.md` / `AGENTS.ja.md`
- `ARCHITECTURE.md` / `ARCHITECTURE.ja.md`
- `docs/PLANS.md` / `docs/PLANS.ja.md`
- `docs/PORTABILITY.md` / `.ja.md`
- `docs/RELIABILITY.md` / `.ja.md`
- `docs/SECURITY.md` / `.ja.md`
- `docs/QUALITY.md` / `.ja.md`
- `docs/roadmap.md` / `.ja.md`
- Android Emulatorの設計とcompleted ExecPlan。
- Flutter Android文書。
- Android UI observer文書。
- standalone distribution文書。
- `internal/execx/detached*.go`。
- Android runtime adapter。
- `internal/app`、`internal/domain`、`internal/config`、`internal/stack`。
- `internal/paths`、`internal/store/sqlite`。
- readiness/probe、endpoint、logs/evidenceの実装。

現在の`execx.NativeDetached`は、request所有writerを既に拒否し、専用fileへ出力し、常駐
commandにtimeoutを設けず、呼出元CLI終了後も存続する。PIDとnative StartIDを記録し、PID単独
ではなくprocess treeの同一性を観測し、Windows `.bat`/`.cmd` wrapperを拒否する。これらを保つ。

## Plan of Work

### Milestone 1 — Product contract and architecture

英日product/design文書を作り、`type: process`、作業directory、command、env、port、
endpoint参照、runtime directory補間を確定する。Compose/Android/Flutter互換性を保ち、
必要に応じarchitecture checkを更新する。

### Milestone 2 — Native termination primitive

必要最小限のnative識別情報を条件とするstop APIを追加する。正確なlive識別情報、既に消えた
process、不一致/再利用識別情報、起点終了、子孫残存、取消し、停止の再実行、native OSの
完了証拠をtestする。Android detached-process testをすべて通し続ける。

### Milestone 3 — Manifest, paths and executable resolution

process専用field、argv、NUL、source内に閉じたcwd、native Windowsの実行ファイル制約を
厳密に検証する。PATH/source相対の解決を定め、redactionした起動証拠を永続化する。

### Milestone 4 — Runtime directory and ports

lease/source識別情報の確定後に専用runtime状態を作る。`${runtime_dir}`と名前付き
loopback TCPの予約/補間を追加する。起動前にport識別情報を保存し、外部占有では保守的に
失敗する。共通endpoint modelへ明示的なprocess runtime-port variantを追加する。

### Milestone 5 — Launch and evidence

createの順序:

```text
source -> runtime dir -> ports -> launch intent -> logs -> Start
       -> identity persistence -> readiness -> READY
```

errorとともに返された0以外の識別情報も保持する。永続化失敗の補償は、識別情報を証明できる間だけ行う。

### Milestone 6 — Readiness, endpoints and logs

`${port:http}`へbindするnative HTTP helperを使う。通常のapp readinessが成功して初めて
READYにする。showは状態/endpointを、logsはfile出力を公開する。

### Milestone 7 — Inspect, reconcile and degradation

正確なnative識別情報を観測する。process消失はDEGRADEDとする。不一致/不確実なtreeは
保守的な状態/quarantineにする。自動再起動はしない。手動終了が後続の独立CLIで見える必要が
あり、PID再利用でREADYへ戻してはならない。

### Milestone 8 — Destroy and recovery

destroyはfenceを取得し、所有を証明し、停止を要求して待つ。証明が有効な間だけ強制停止へ
進み、tree全体の不在を確認してからport/runtime状態を解放し、source cleanupへ進む。
tree不在が不確実なら依存resourceを保持してquarantineにする。destroyの再実行とGCも同じ証明を使う。

### Milestone 9 — Cross-process/concurrency integration

listen、log出力、任意の子process生成、正常終了、起点終了/子残存の再現ができる実native
helperを使う。create CLI終了後の存続、後続CLI観測、2 leaseの分離、兄弟leaseの存続、
無関係なprocessの安全、手動終了時degraded化、子の所有が不確かな場合の保守的挙動を証明する。

### Milestone 10 — Browser-ready fixture

Browser自体を実装せず、`${runtime_dir}/profile`と`${port:cdp}`、最小CDP状HTTP endpointを
使うBrowser状helperを追加する。専用状態、永続所有、endpoint、readiness、logs、後続CLI観測、
安全なdestroyを証明する。

### Milestone 11 — Native CI and documentation

Windows、macOS、Linuxで実際の常駐process実行を必須にする。この機能にhardware acceleration
前提はない。cross-buildは追加証拠にとどめる。英日Architecture/Portability/Reliability/
Security/Quality/Roadmapを更新し、完了後は汎用の常駐host processを未実装roadmapから外す。

## Concrete Steps

1. baseを記録してbranchを作る。
2. 英日のactive Planを追加する。
3. baseline harness/raceを実行する。
4. native detached/Androidの既存挙動の検証範囲を調べる。
5. 英日product/design文書を書く。
6. process/endpoint/補間構文を確定する。
7. 識別情報を条件とする停止を追加する。
8. Android detached testを再実行する。
9. config/domain/process runtimeを追加する。
10. runtime directory/補間を追加する。
11. 動的TCP portとendpoint mappingを追加する。
12. 起動/証拠/readiness/logs/showを実装する。
13. inspect/reconcileを実装する。
14. destroy/quarantine/GCを実装する。
15. native HTTP helperと2 lease testを追加する。
16. 手動終了/PID再利用/子孫の所有が不確かな場合のtestを追加する。
17. Browser状fixtureを追加する。
18. Windows/macOS/Linux native integrationを実行する。
19. 英日永続文書を更新する。
20. 最終harness/race/native/cross-buildを実行する。
21. 直接の受け入れ証拠を記録する。
22. 英日retrospectiveを完成させる。
23. Planをcompletedへ移し、link/hashを更新する。

## Validation and Acceptance

| ID | 必須の挙動 | 証拠 |
| --- | --- | --- |
| H1 | 既存Compose/Android/Flutter/UI-observerの挙動が有効なまま。 | local check/race/Docker integration、Podman+Docker成功。Verify 34226859965の既存動作を検証する6つのnative jobも成功。 |
| H2 | `type: process`を厳密に検証し、非互換のruntime fieldを拒否する。 | TestProcessManifestContract; TestProcessManifestNegativeFixtures; TestProcessPresenceRejectsYAMLMergeAndAliases; TestProcessFieldsDoNotChangeLegacyCanonicalShape — 2026-09-08、Linux統合作業treeで成功。 |
| H3 | commandは閉じたcwdを使うargv直接実行で、暗黙のshellを使わない。 | TestStartInterpolatesWithoutSnapshotSecrets; TestPrepareRejectsUnownedRootAndSourceEscape; TestPrepareRejectsSymlinkCWDAndRuntimeRoot — 2026-09-08、Linux統合作業treeで成功。 |
| H4 | 起動元create CLI終了後もprocessが生存する。 | Linux TestPersistentProcessNativeCLI成功。create終了後に後続の独立CLIで観測。 |
| H5 | 後続の独立CLIが永続native識別情報で正確なprocessを検査する。 | Linux native CLI成功。独立showとcreate側中断をまたいで永続識別情報を維持。 |
| H6 | PID再利用/識別情報不一致で無関係なprocessを所有扱いしない。 | 最終native CIでTestManagedTermination、TestRecoveryUsesReceiptAndChecksMismatch、WindowsのTestManagedWindowsCompletedJobIgnoresReusedHistoricalPID成功。識別情報再利用の模擬であり、kernel PID再利用の強制ではない。 |
| H7 | stdout/stderrはfileへ書き、runtime logsで取得できる。 | TestProcessComponentLogsRouteAndRetainIsolatedArtifacts; TestLogsRequireDurableRedactionVersion; TestBoundedLogsDoNotExposeSecretAcrossBoundary — 2026-09-08、Linux統合作業treeで成功。 |
| H8 | 各runtimeがagent-env状態root配下に専用状態directoryを持つ。 | TestDestroyPreservesLogsAndReceipt; TestPersistentProcessNativeCLI — 2026-09-08、Linux統合作業treeで成功。 |
| H9 | 空白/非ASCIIのnative pathでもruntime directory補間が動く。 | Verify 34226859965の3 native OSでTestPersistentProcessNativeCLI成功。空白/Unicode path、Unix symlink homeを検証。 |
| H10 | 名前付きloopback TCP portを動的に割り当て、起動前に永続化する。 | TestProcessConcurrentReservationPortsAreDisjoint; TestProcessLifecyclePersistedIntentMixedRoutingAndIsolation — 2026-09-08、Linux統合作業treeで成功。 |
| H11 | 並行process leaseはagent-env port予約やruntime directoryを共有しない。 | TestProcessConcurrentReservationPortsAreDisjoint; TestPersistentProcessNativeCLI — 2026-09-08、Linux統合作業treeで成功。 |
| H12 | 外部port占有では、保存済みruntime識別情報を黙って変えず安全に失敗する。 | TestProcessDynamicReservationAvoidsExternallyBoundPort; TestReservedPortOccupationPreventsLaunch — 2026-09-08、Linux統合作業treeで成功。 |
| H13 | process endpointがCompose利用側を壊さず共通component endpoint modelへ統合される。 | TestProcessReadinessUsesRecordedNumericEndpoint; TestProcessEndpointAliasWinsOverRawRuntimePort; TestIntegrationPersistentProcessComposeCoexistence — 2026-09-08、Linux統合作業treeで成功。 |
| H14 | 実HTTP helperが通常のapp readinessを通じてREADYになる。 | Linux native CLIの実/json/version HTTP readiness成功。 |
| H15 | readiness前のprocess終了はcreateを失敗させ、保守的に補償する。 | TestProcessExitDuringSuccessfulProbeCannotBecomeReady; TestProcessIdentitySaveFailureAndPartialStartCompensate — 2026-09-08、Linux統合作業treeで成功。 |
| H16 | 手動終了はDEGRADEDとなり、自動再起動しない。 | TestProcessCrashDegradesWithoutRestartAndReleasedEffectsQuarantine; TestPersistentProcessNativeCLI — 2026-09-08、Linux統合作業treeで成功。 |
| H17 | 子孫が残る起点終了を、native証拠なしに誤ってcleanにしない。 | Verify 34226859965の6 native jobでTestManagedRootGoneDescendantとTestExitedRootWithDescendantsIsNotReady成功。 |
| H18 | destroyは停止直前にnative識別情報を再検証する。 | Linux TestManagedTerminationの不一致/取消し/稼働起点保護が成功。Unix観測からsignalまでのgapは明記したまま。 |
| H19 | port/runtime状態の解放前に所有tree全体の不在を確認する。 | TestProcessUnknownOwnershipQuarantinesAndRecovers; TestProcessSaveRejectsImmutableSnapshotChanges; TestPersistentProcessNativeCLI — 2026-09-08、Linux統合作業treeで成功。 |
| H20 | 曖昧な停止はquarantineにし、復旧証拠/resourceを保持する。 | TestProcessUnknownOwnershipQuarantinesAndRecovers; TestProcessFenceLossRetainsLaunchForLaterRecovery; TestMissingLaunchingReceiptIsUncertain — 2026-09-08、Linux統合作業treeで成功。 |
| H21 | destroyの再実行で後続のPID/process tree利用者をkillしない。 | TestManagedTermination; TestProcessReleasedReservationsCannotResurrect; TestPersistentProcessNativeCLI — 2026-09-08、Linux統合作業treeで成功。 |
| H22 | Aのdestroyは兄弟Bと無関係なhost processを保つ。 | TestPersistentProcessNativeCLI（5.422s）成功。cleanup/拒否を通じ兄弟と別の直接host helperのPID/profileが存続。 |
| H23 | 宣言したstackでprocessとCompose runtimeが共存できる。 | TestIntegrationPersistentProcessComposeCoexistence (55.965s) — 2026-09-08、Linux統合作業treeで成功。 |
| H24 | processがworktreeを参照し得る間も、sourceのtracked変更保護を保つ。 | TestPersistentProcessNativeCLI（5.422s）成功。稼働中tracked READMEの変更を拒否時にbytes/processごと保持し、forceはtracked-diff記録後に解放。 |
| H25 | shell/Python/Node/systemd/launchd/Windows Serviceをcore要件にしない。 | 統合arch-check/build/testとGo製CLI/helper成功。新たなcore runtime/daemonは不要。 |
| H26 | 実常駐process integrationがnative Windowsで成功する。 | f588960のVerify 34226859965でWindows Go 1.26/1.27 native repoctl check成功。TestPersistentProcessNativeCLIと完了Jobの証拠を優先することを確認するテストを含む。 |
| H27 | 実常駐process integrationがnative macOSで成功する。 | f588960のVerify 34226859965でmacOS Go 1.26/1.27 native repoctl check成功。TestPersistentProcessNativeCLIを含む。 |
| H28 | 実常駐process integrationがnative Linuxで成功する。 | TestPersistentProcessNativeCLI (5.422s) — 2026-09-08、Linux統合作業treeで成功。 |
| H29 | Browser状fixtureが状態directory、CDP状port、readiness、後続観測、cleanupを証明する。 | TestPersistentProcessNativeCLI成功。専用profile、/json/version、readiness、独立show、cleanupを検証。 |
| H30 | execx変更後も既存Android detached/guardianの受け入れが成功する。 | f588960のVerify 34226859965で6 native repoctl check jobのAndroid detached/guardianの既存動作のテスト成功。 |
| H31 | 英日永続文書が提供した契約を説明する。 | 英日永続契約、完了先リンク、native証拠を一緒に更新。docs-check成功。 |
| H32 | 最終harness/翻訳/race/native CIが成功する。 | local harness/race/integration成功。f588960のVerify 34226859965は6 native、race/integration、5 cross-buildを含む全12 job成功。 |
| H33 | archive前に両ExecPlanが直接証拠とretrospectiveを持つ。 | 両Planに最終checkpoint、H1〜H33証拠、完了retrospectiveを記録し、リンクを更新して一緒にarchiveした。 |

codeの存在だけでは受け入れとしない。成功command、native job、必要に応じて観測したprocess
識別情報とport割り当てを記録する。

## Idempotence and Recovery

plan/検証は読み取り専用。createはsagaで、作用前に意図を永続化する。起動した可能性がある
場合はnative識別情報/証拠を保持し、cleanup前に検査する。PID検索だけで不在を推定せず、
不確実な起動を自動再試行しない。

destroyはobserve -> prove ownership -> terminate -> confirm absence -> releaseの順。
process treeが使う可能性がある間はworktree/runtime directory/portを解放しない。GCも
同じ証明を使い、既定はdry-runのまま。手動のprocess終了はdegradedとして観測し、自動修復しない。

## Artifacts and Notes

Windows修正時点の記録（2026-09-08）: 同一sessionの正確なJobを、過去のPIDより先に
確認する。空または存在しないJobの不在判定には、同期済みguardian完了証拠の一致が必要で、
活動中Jobのbirth・所属検査は維持する。PID観測中に完了した場合はJobを再観測する。
完了を証明した後は過去のPIDへsignalを送らない。同じPID再利用の問題を検出する新しいnativeテスト
`TestManagedWindowsCompletedJobIgnoresReusedHistoricalPID`では2つの実processと保持した
旧Job handleを使い、空のJob・消失したJob・証拠欠落/不一致・活動中identity不一致を確認する。
Linux harnessとWindows amd64/arm64テストcross-compileは成功した。native CIは未確認である。
独立レビューでは、活動中の所有権やAndroid guardianの条件が弱まっていないことを確認した。


native primitiveの証拠（2026-09-08、`01e3581`ベースの作業tree）:

- `go test ./internal/execx`: 成功。
- `go test -race ./internal/execx`: 成功（8.027s）。
- `go test ./internal/execx -run TestManaged -count=3`: 成功（3.583s）。
- `go test ./internal/runtime/android`: 成功（0.552s）。
- `GOOS=windows GOARCH=amd64 go test -c ./internal/execx`と
  `GOOS=darwin GOARCH=arm64 go test -c ./internal/execx`: temporary出力先を指定して成功。
  証明できるのはcompileのみであり、native実行は未検証。
- Linuxのprocess censusは取得中に変化することがある。期限付き再試行は一時的なcensus
  sentinelだけを対象とし、所有errorは再試行しない。

契約milestoneの証拠（2026-09-08、`01e3581`ベースの作業tree）:

- `go test ./internal/config ./internal/domain`: 成功。不正runtime field、local readiness
  endpoint参照、Windows directory名、不正補間、YAML merge/aliasの存在、既存canonical
  表現の維持を含む。
- 初回`go run ./tools/repoctl docs-check`: 提供active Planの日本語版不足で失敗。翻訳を
  追加しcheckpointを更新した後の`go run ./tools/repoctl docs-check`は成功。
  baseline全harness/raceは後に隔離した再構成baseで成功した。統合checkpointを参照。


起動と停止を再構成できるだけの証拠を保持する:

- lease/runtime/source commit。
- redactionしたargv/env。
- cwdと解決済み実行ファイル。
- 記録した場合の実行ファイルdigest。
- runtime directory。
- 名前付きport。
- stdout/stderr path。
- PID/start/tree識別情報。
- 起動/readiness timestamp。
- 停止結果。
- degraded/不確実性の観測。

runtime可変状態を自動で証拠へ昇格しない。Browser profile、cache、local databaseには
機密情報が含まれ得るため、通常はcleanup確認後に削除する。

## Interfaces and Dependencies

想定package:

    internal/runtime/process/

configの概念例:

```go
type ProcessRuntime struct {
    Source           string
    WorkingDirectory string
    Command          []string
    Env              map[string]string
    Ports            map[string]ProcessPort
}
```

native観測/停止の型には`Alive`、`RootAlive`、確実性、期限付き猶予時間などを含め得る。
正確な型はarchitecture reviewに従う。

OS固有のPID/Job/process-group操作をapp方針へ漏らさない。

runtimeの外部前提は、対象repository/hostが宣言する実行ファイルのみ。
新たな言語runtimeやdaemonをagent-envのcore要件にしない。

## Resolved Milestone 1 Questions

1. 最終的な`working_directory`のfield名。
2. process portの宣言構文。
3. process runtime port用のcomponent endpoint field。
4. 補間namespaceとsource path補間の必要性。
5. PATH実行ファイル方針とsource相対の優先。
6. host binary/scriptの実行ファイルdigest方針。
7. Windows/Unixの穏当な停止の意味。
8. 現在のDetachedProcess拡張か、より明確なmanaged-process interfaceの導入か。
9. show/listが公開するprocess詳細。
10. 動的port割り当て方式と外部競合の説明。
11. 既存HTTP/command probeに加えてTCP readiness probeが必要か。
12. 失敗/quarantine cleanup後の可変状態保持。
13. Unixで子孫が残る起点終了時のquarantine方針。
14. 将来のrestart契約を予約するか、完全に省くか。
15. 将来Browser/CDPが`type: process`の上にobserverを重ねるか、専用runtimeを内部で組み合わせるか。

上記の論点はDecision Log、提供したproduct/design契約、H1〜H33の証拠で解決済み。
最終3 OSのCLI/primitive testでnative停止とshow/listを検証した。
