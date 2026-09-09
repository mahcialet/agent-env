---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/exec-plans/active/multi-host-control-plane.md
source_sha256: dcf432ee024d6fccb5aef29058bbe0939797b28e6507afd217f26dfe9ae63a4c
---

# Single-authority multi-host control planeを追加する

[English](multi-host-control-plane.md)

このExecPlanはliving documentであり、`docs/PLANS.md`に従って更新する。

想定ブランチ: `feat/multi-host-control-plane`

開始revision: `dc63308e53f68f8be99f7cbf59cafc78f7296b71`
（PR #11 repository correctness audit merge済みmaster）。

実装前にmasterをfast-forwardし、baseが変わっていれば正確なrevisionを記録し直して
baseline harness/race/integrationを実行する。

## 目的 / 全体像

既存local-first lease semanticsを維持したまま、複数machineへenvironment leaseを
placementできるcontrol planeを追加する。

local modeは今まで通りdaemon不要。

multi-host modeのみ:

```text
authenticated client
        |
        | HTTPS / mTLS
        v
single active control plane
  controller SQLite
  source/artifact CAS
        ^
        | outbound heartbeat / long-poll
   +----+----+
   |         |
worker A   worker B
local DB   local DB
local Git  local Git
local runtimes
```

初期placement unitは**Lease全体**。

```text
1 global lease
  -> exactly 1 worker
  -> selected source/component/runtimeを全て同じworkerで実行
```

APIをHost A、AndroidをHost Bのように1 lease内で分割しない。cross-host
networking、distributed cleanup、multi-authority resource graphは後続。

## 基本原則

### local modeをdaemon architectureへ変えない

controller未指定なら既存local SQLite/local Git/local runtimeだけを使う。
network/control process不要。

### one lease = one worker

Compose/process/Android/Flutter/Browser等のworker-local endpoint semanticsを維持する。

### OFFLINE != absent

worker heartbeat lossはcommunication lossしか証明しない。
container/process/Emulator/worktree/portのabsence proofにはならない。

assigned live/uncertain leaseをworker lossだけで別hostへauto reassignしない。
global observationはstale/UNKNOWNとして保持し、元workerのrelease proofを待つ。

### exactly-onceを仮定しない

network disconnect後にmutationをblind retryしない。
global operation ID、assignment epoch、worker durable journal、effect前intent persist、
reconnect recoveryを使う。

### raw remote shellを追加しない

remote APIはexisting typed agent-env operationのみ。manifestが宣言したprocess argvは
repository executionとして許容するが、control planeの任意shell endpointは作らない。

## 対象範囲

対象:

- same binaryのcontrol-plane serve / worker serve
- authenticated remote CLI
- controller専用SQLite
- persistent controller ID
- stable host ID + persistent host-instance ID
- worker enrollment
- mTLS
- versioned JSON/HTTP
- outbound heartbeat/long-poll
- host registry / ONLINE/OFFLINE/DRAINING
- capability/capacity reporting
- deterministic whole-lease scheduler
- explicit host selection
- assignment epoch
- controller/worker operation journal
- controller-managed local lease marker
- local GC/mutation protection
- immutable Git source transport
- content-addressed source/artifact store
- multiple source aliases
- result/artifact retry without effect replay
- stale/UNKNOWN global observation
- controller/worker restart recovery
- drain/undrain
- duplicate delivery/result-loss recovery
- two-worker real-socket integration
- Windows/macOS/Linux native protocol integration
- worker capabilityが満たすexisting Docker/Podman/Android/process/Browser利用
- 英日product/design/ADR/docs

対象外:

- active-active HA/consensus
- automatic controller failover
- live lease migration
- one lease split across workers
- overlay network/endpoint tunnel
- arbitrary SSH/inbound worker RPC
- hostile multi-tenant isolation
- automatic PKI/CA rotation
- implicit client secret forwarding
- Vault/KMS
- automatic remote Git fetch/auth
- unproven LFS/submodule transport
- cross-host writable fix lease
- autoscaling/CPU-memory bin packing
- service installation
- malicious-code sandbox

## authority model

### control plane

global lease ID、source identity/digest、stack、required capabilities、placement、
assignment epoch、global desired state、operation、host registry/liveness、CAS reference、
last acknowledged worker observation、global historyを所有する。

native PID/container/AVD/CDP cleanup proofは所有しない。

### worker local

existing worker-local DBがworktree、Compose/Podman、Android、Flutter、persistent
process、Browser/CDP、reservation、local fence、cleanup proofを所有する。

controllerはheartbeat lossからabsenceを推測しない。
workerはglobal reassignment authorityを推測しない。

### lease identity

controllerがglobal lease IDを生成し、可能ならworker local leaseも同じIDを使う。

workerは少なくとも:

```text
management_mode=controller
controller_id
host_id
host_instance_id
assignment_epoch
```

をdurableに保持する。

## host/controller identity

worker:

- operator-selected stable `host_id`
- state rootに保存するrandom `host_instance_id`
- enrolled cert identity/fingerprint
- worker process/session incarnation
- agent-env/protocol version
- OS/arch/capability

`host_id`だけをauthorityにしない。
active/uncertain assignmentがあるhost IDへdifferent instanceが来たらreject/quarantine。

controllerはpersistent random `controller_id`を持つ。
different controllerがactive controller-managed leaseをsilent adoptしない。

これはconsensusではなくcloned active controllerを安全にするものではない。

## protocol/auth

初期候補:

```text
HTTPS + mTLS + versioned JSON
```

Go標準`net/http`, `crypto/tls`, `crypto/x509`, `encoding/json`を優先。

workerはoutbound only。

```text
register
 -> capability
 -> heartbeat
 -> long-poll operation
 -> local journal
 -> execute/recover
 -> upload result
 -> ack
```

payloadはprotocol/controller/host/instance/global lease/epoch/operation ID/digestを持つ。
body limit/deadline/typed statusを明示する。

non-loopback production listenerはTLS + enrolled cert必須。
private keyはSQLiteに保存しない。
loopback insecure modeが必要ならtest限定。

automatic CA/rotationは後続。

## source transport

client local repository pathをworker pathとして使わない。

```text
client:
  manifest validate
  source alias -> exact commit
  Git bundle/object package
  SHA256
  controller CAS upload

controller:
  capability scheduling

worker:
  blob download
  digest verify
  git verify/import
  private mirror/worktree
  exact commit verify
  transferred manifest/plan verify
  existing local lifecycle
```

untracked/uncommitted fileはtransportしない。
remote Git credential不要の方式を初期defaultとする。

shallow/LFS/submoduleは暗黙network fetchせず明示検証/制限。

## CAS

controller state root内のprivate content-addressed store。

```text
sha256/<digest>
```

atomic write、stream digest/size verify、size limit、duplicate upload safety、reference
tracking、referenced blob非削除、private permission、caller pathをfilesystem authorityに
しないことを要求。

source/artifactはsecretを含み得る。encryption at restは未保証。

## scheduler

worker doctor/prerequisiteからsemantic capabilityをreport:

```text
git
compose.docker
compose.podman
android-emulator
flutter-android
persistent-process
browser-cdp
```

exact versionはevidence。schedulerはstable capability名を使う。

初期capacityはmax controller leases + truthful named slot（例: Android slots）程度。
CPU/memory bin packingはしない。

filter:

authenticated / ONLINE / non-draining / protocol compatible / capability /
constraint / capacity。

その後deterministic least-load + host-id tie break等。
placement/capacity reservationをcontroller transactionでatomicにする。

`--host`もsafety checkをbypassしない。

## capability derivation/preflight

clientがselected immutable stack closureからrequired capabilityをderive。
workerがtransferred source/manifest/planを独立verify。

workerがeffect前rejectし`effects_started=false`を証明した場合のみ別host再schedule可。
effect可能性があればauto reschedule禁止。

## assignment/operation fence

operationは:

```text
controller_id
global_lease_id
host_id
host_instance_id
assignment_epoch
operation_id
operation_type
```

を持ち、workerはeffect前persist。

wrong controller/host/instance、old epoch、conflicting duplicate、terminal state不整合をreject。

同じoperation IDのduplicate deliveryはdurable resultを返す/再構築し、mutationをblind
repeatしない。

## worker/controller journal

worker conceptual state:

```text
received
prepared
effect_started
result_pending
completed
uncertain
```

existing command/evidence tableを再利用できるならduplicate authorityを作らない。

restart/reconnect後はunfinished opをreportしactual local resourceをreconcileしてから
resumeする。

controller transport stateはlocal resource truthと分離:

```text
queued
assigned
dispatched
running
result_pending
completed
uncertain
```

## liveness / host loss

desired stateとobservation freshnessを分離。

heartbeat expiry時:

```text
desired=active
placement=linux-01
last_known=READY
observed=UNKNOWN
```

stored READYをcurrent healthyとして表示しない。
same worker instanceが戻ったらsame leaseをreconcileし、別workerへplaceしない。

## heartbeat/drain/remove

heartbeatはidentity/version/capability/capacity/controller-managed lease summaryを含む。

drainはnew assignment停止のみ。migration/destroy無し。

active/stale/uncertain/unreleased assignmentがあるhost forget/removeをreject。

## outage semantics

controller outage中workerはexisting workloadを維持する。
new remote mutation無し。controller heartbeat消失だけでauto GC/destroyしない。
journal/evidence保持、後でreconnect。

worker restartはexisting local DB/stateを再openしsame identityでcontrollerへ戻り、
resource/journalをreconcileする。

## local operator

controller-managed leaseを明示表示。

ordinary local destroy/renew/reconcile/GCで変更できないようにする。
read-only diagnosticsは可。

safe design無しにbreak-glass `--force`を追加しない。

## TTL

global lifetimeはcontroller authority。
controller outage/TTL expiryだけでworker destroyを許可しない。
renew/destroyはfenced remote operation。

## remote operation

create/list/show/renew/reconcile/destroy/named test/log/artifact/Android UI/Browser CDPを
assigned workerで実行。

raw shell無し。

worker-local endpointをclient-local `127.0.0.1`と偽って表示しない。
endpoint tunnelingは後続。

## result/artifact

local result persist -> blob upload -> controller verify -> result manifest -> ack。

upload failureはresult_pendingとしてuploadだけretry。
underlying test/UI/browser/lifecycle effectを再実行しない。

## security

controller/client/workerはone trusted administrative domain。
target repo trustはlocal modeと同等。multi-tenant sandboxではない。

`${env:NAME}`はworker environmentでresolve。
client env secretを自動forwardしない。

## 進捗

- [x] 2026-09-09: product/design文書とauthority ADR0006を両言語で追加。
- [x] 2026-09-09: worker host-instance IDとremote-operation journal、commit済みsource bundleとdigest検証を実装。
- [x] 2026-09-09: global lease IDと管理metadataをlocal Createに接続し、ローカル変更とGCから保護。

- [x] 2026-09-09: base `dc63308e53f68f8be99f7cbf59cafc78f7296b71`を確認し、`feat/multi-host-control-plane`を作成。
- [x] 2026-09-09: 残るnative runtime baseline検証。
- [x] 2026-09-09: 英日product/design + ADR
- [x] 2026-09-09: protocol/auth contract
- [x] 2026-09-09: controller/worker/global state model
- [x] 2026-09-09: controller SQLite/migrations/id
- [x] 2026-09-09: worker instance id/enrollment
- [x] 2026-09-09: mTLS HTTP
- [x] 2026-09-09: registration/heartbeat/long-poll
- [x] 2026-09-09: capability/capacity
- [x] 2026-09-09: hosts list/show/drain
- [x] 2026-09-09: scheduler
- [x] 2026-09-09: Git source package + CAS
- [x] 2026-09-09: worker source verify/materialize
- [x] 2026-09-09: manifest/plan identity verify
- [x] 2026-09-09: global lease ID local create
- [x] 2026-09-09: controller-managed marker/local GC protection
- [x] 2026-09-09: assignment epoch
- [x] 2026-09-09: controller/worker journals
- [x] 2026-09-09: dispatch/reconnect
- [x] 2026-09-09: remote create/list/show
- [x] 2026-09-09: renew/reconcile/destroy
- [x] 2026-09-09: test/log/artifact
- [x] 2026-09-09: artifact CAS/result manifest
- [x] 2026-09-09: Android UI remote
- [x] 2026-09-09: Browser remote
- [x] 2026-09-09: heartbeat stale/UNKNOWN
- [x] 2026-09-09: no auto reassignment
- [x] 2026-09-09: worker/controller restart recovery
- [x] 2026-09-09: same host-id different instance reject
- [x] 2026-09-09: drain/remove safety
- [x] 2026-09-09: auth/body/path/blob negative tests
- [x] 2026-09-09: duplicate delivery / effect-result loss regression
- [x] 2026-09-09: outage with live resource
- [x] 2026-09-09: two-worker real-socket integration
- [x] 2026-09-09: remote process/browser E2E
- [x] 2026-09-09: Docker/Podman remote integration where possible
- [x] 2026-09-09: Windows/macOS/Linux native protocol integration
- [x] 2026-09-09: separate machine/VM evidence where available
- [x] 2026-09-09: bilingual durable docs
- [ ] final harness/race/native/integration/release
- [ ] evidence/retrospective/completed

## 想定外の発見

- 2026-09-09: Verify 34317326990でもWindowsの深いsource lifecycleが失敗し、Gitが長い`-C`ディレクトリを拒否した。Git for Windowsの実装ではrepository config初期化前の`are_long_paths_enabled`がfalseとなるため、command-lineの`core.longpaths`では早期chdirを解決できない。native multi-host 34317327009とBrowser 34317326997は3 OSすべて成功した。Windowsの拡張prefix付きprocess cwdを調査し、任意の8.3名への依存やテストの深さ削減は導入しない。

- 2026-09-09: WindowsのGitは`-C`で作業ディレクトリを選び、source配置と回帰テストを維持したままCreateProcessの長いcwd制限を回避する。Browser CIでは`Target.closeTarget`後の一覧反映が非同期だったため、元のページだけになるまで最大5秒待つ検査に変更した。元のページの消失や未知のtargetは即失敗とする。Linuxの実Chromeを3回実行して成功（28.929s）。ローカル全harness/raceとworker UIテストが成功し、remote Browser/Docker/Podman E2Eも再成功した（33.142s）。native CIで再検証する。

- 2026-09-09: 相対cloneパスだけではVerify 34316762287のWindows検査は通らず、Git起動前にCreateProcessが260文字超の作業ディレクトリを拒否した。native multi-host 34316762301は引き続き3 OSすべて成功した。Browser native 34316762319ではmacOSのpage-close後の一覧検査が失敗し、source変更とは別に非同期のtarget反映を調査している。いずれも修正の検証が終わるまでは未解決として扱う。

- 2026-09-09: mode不正、TTL上限超過、process provider不在の3ケースで、local Reserve前のcreate失敗なのにworkerの作用開始を記録する問題を修正前に再現した。Reserve直前のapp callbackでjournalを更新するようにし、予約前の失敗では作用未開始の永続証拠を保持する。Reserveの曖昧な失敗は不在証明にせず、fence失敗はReserveを防ぐ回帰テストも追加した。app/worker全raceテスト成功（47.217s / 4.243s）。

- 2026-09-09: `53fe81a` の multi-host と Browser の native CI は3 OSすべてで成功した（34316121492 / 34316121411）。Verify 34316121380 では、Windows の300文字超のソースライフサイクル検査がまだ失敗した。`core.longpaths=true` だけでは、Git index-pack が絶対パスの `$GIT_DIR` を拒否する問題を解消できなかった。検証済みの専用展開ディレクトリからの相対パスで clone するよう修正し、深いパスのテストは維持した。ローカルの全 harness と source/worker の race テストは成功し、native の再検証を行う。

- 2026-09-09: 最初の実TLS createはJSON objectのkey順序変更によるpackage digest不一致で失敗した。型付きmanifestをcanonical化してhashを計算し、commit済みcontrol fileからの変換証明は独立して維持する。順序変更回帰テストとLinux native E2Eが成功。controllerのglobal stateも、想定した大文字値ではなく`released`を含む実際のdomainの小文字stateに対応させた。
- 2026-09-09: 独立レビューでpreflight診断の秘密情報漏出を修正前の4ケースで再現。継承secretのredactionとmetadata上限を適用した。controllerが受け付ける可読operation IDをexecutorが拒否する不整合も修正し、lease IDのULID要件は維持した。
- 2026-09-09: 外部作用前のcreate失敗ではlocal leaseがないためdestroyも失敗し、予約を解放できなかった。journalに作用開始前の境界を原子的に記録し、入力検証済みのnon-dry-run destroyだけが同じassignmentの証明を使えるようにした。local row不在やerror payloadを証明にはしない。再起動、dry-run、不正入力、証拠欠落、作用開始済みの回帰検証が成功。
- 2026-09-09: `0f05d09`の初回native macOS CIは、同一の一時directory祖先を示す`/var`と`/private/var`を異なるrepository identityとして比較し失敗。Linuxでは見えなかったOSのpath aliasであり、その後、信頼するrootの正規化で修正し、34314956327のnative検証が成功した。Verify 34314568570、Multi-host native 34314568601。

- 2026-09-09: 独立レビューでPrepare中のheartbeat切断後も外部作用を開始できる経路を発見。永続的なeffect-started遷移の直前に検査を追加し、再接続まで作用を開始しない回帰テストが成功。
- 2026-09-09: remote actionの初期テストはUI/Browserのflagが全action共通と誤って想定していた。実際のactionごとのflag定義に対応させ、local flagを変更せず修正した。初回全harnessは新規テストで失敗し、修正後の個別テストは成功。再実行はcontrollerの編集中に整形検査で失敗したため、安定した変更単位で全検査を再実行する。

- 2026-09-09: baselineのunit/vet、raceテストが成功。実Dockerを使用した`repoctl test-integration`も終了コード0。`repoctl check`は、提供された日本語Planに必須の見出しがなかったためdocs-checkで失敗。両言語に不足する節を追加した。この時点では残るnative runtime baselineは未実施で、その後成功した。

## 判断の記録

- 2026-09-09、実装担当: workerのoperation dispatchは初期仕様では直列とし、同じleaseの2つ目のactive operationを拒否する。長時間稼働するlease自体は同時に動く。remote destroyによるactive remote testのcancelは専用protocolの将来課題とし、assignment fenceを競合させない。
- 2026-09-09、検証担当: 別物理host/VMの環境は利用できない。実TLS fixtureは同一host上の独立したnative controller/client/2-worker processを使う。Windows/macOSのnative証拠はcross-buildではなくCIで取得する。

- 2026-09-09、実装担当: assignment epochはleaseの配置ごとに固定し、各commandをoperation IDで識別する。local SQLiteは管理metadataの削除、後付け、変更を、有効なlocal lockがあっても拒否する。
- 2026-09-09、実装担当: FULL synchronousのSQLiteでworker journalを別管理する。controllerとの恒久的な対応、host-instance ID、processごとのincarnationを保存する。結果upload/ackの再試行は実行開始済みreceiptをリセットしない。native SQLite lockで同じrootのworker二重起動を拒否し、crash後はOSがlockを解放する。
- 2026-09-09、実装担当: `--controller`と明示的な`--tls-ca`、`--tls-cert`、`--tls-key`、`control-plane serve/enroll`、`worker serve`を採用する。証明書は外部で用意し、leaf fingerprintをclient/worker roleで登録する。controller未指定時のlocal commandは従来の動作を保つ。

- initial placementはone lease = one worker。split-hostは後続。
- local mode daemon-free維持。
- single active controller。HA/consensusは後続。
- heartbeat lossでabsence/reassignmentしない。
- worker outbound connection。
- versioned JSON/HTTPS/mTLS + typed operation。
- controller/global authorityとworker/local resource authorityを分離。
- sourceはcommit + digest-verified Git object/bundle transport。
- source/artifactはcontent-addressed。
- uncertain mutationはblind retryしない。
- controller-managed leaseをlocal GC/mutationから保護。
- endpointはworker-localのまま。
- remote envはworker側resolve、client secret自動forward無し。
- durable docs/ExecPlan英日。

すべて日付/担当: 2026-09-09 / maintainers.

## 成果と振り返り

実装とローカルの受け入れ検証は完了し、最終native CIを実行中である。

永続的なcontroller管理主体、相互認証するclient/workerの役割、host identity、
capability/capacityによる配置と、lease全体を一つのworkerへ置く仕組みを実装した。
厳密なcommitのGit bundleとdigest検証付きCASでsourceと登録済み証拠を運ぶ。
CLIのlifecycle、named test、logs、artifacts、Android UI、Browser操作は、
workerの既存app境界を通る。通常のlocalコマンドはcontroller管理のleaseを引き継げない。

controllerは配置を、workerのstate/journalはlocal所有権と結果配送を保持し、
二つの永続化主体が個別に復旧する。heartbeat喪失時もUNKNOWNのassignmentとcapacityを保持し、
ack喪失時は証拠の配送を再試行する。作用が不確実な操作は無条件に再実行しない。
予約前のcreate失敗には作用未開始の永続証拠があるが、予約を試行した後は、
行がないという理由だけでcleanup完了とは判断しない。

local modeはdaemon不要のままである。endpointはworker-localであり、複数leaseを
稼働させられるが、各workerのremote操作は直列に実行する。実行中remote testのcancel、
HA、migration、split-host lease、endpoint tunnel、secret配送、LFS/submodule転送、
CASの自動GCは後続課題とする。nativeの2-worker検証はすべて、一台の物理host上で
役割ごとにprocessを分けたものであり、別の物理機やVM間ネットワークの証拠はない。

特に有効だった回帰検証は、JSON転送の正規化、予約前のjournal境界、管理情報の不変性、
Windowsの実行ファイル検索と長いパス、Browser target削除の非同期反映という境界を扱った。
クロスコンパイルだけではnative OSの挙動を検出できず、実processと修正前に失敗する
negative testが必要だった。独立レビューでcontroller/workerの復旧とcreateのfenceを確認した。
worker UI経路のテストを追加し、transportだけの検証を広く説明していた証拠の不足も補った。

## 背景と構成

読む:

- AGENTS/ARCHITECTURE/PLANS英日
- QUALITY/RELIABILITY/SECURITY/PORTABILITY/roadmap英日
- repository correctness audit/review follow-up
- source/worktree
- Compose/Podman
- Android/Flutter/UI
- persistent process
- Browser/CDP
- standalone distribution
- SQLite migrations/store
- operation fencing

現行architectureはSQLite/operation lockをlocal authorityとしている。本Planはその上に
global authorityを追加し、local SQLiteをdistributed DBとして再解釈しない。

## 作業計画

### Milestone 1 — contract/ADR
multi-host product/design/ADR、single controller、whole-lease placement、authority split、
outbound worker、no heartbeat failover、source/CASを確定。

### Milestone 2 — controller persistence
controller専用store/migration。controller_meta/clients/hosts/capabilities/global_leases/
assignments/operations/blobs/events等。persistent controller ID。

### Milestone 3 — worker identity/mTLS
host-instance ID、enrollment、protocol、registration、heartbeat、long-poll、blob transfer。

### Milestone 4 — scheduler
capability/capacity、ONLINE/DRAINING、deterministic placement、explicit host。

### Milestone 5 — source transport
multi-source Git package/CAS、digest/commit verify、absolute path non-leakage、
shallow/LFS/submodule explicit policy。

### Milestone 6 — global lease/local create
controller lease ID、assignment epoch、worker independent preflight、controller-managed lease。
pre-effect rejectだけreschedule可。

### Milestone 7 — journals/reconnect
dispatch loss、duplicate、effect後disconnect、upload failure、controller/worker restart、late
responseをfailure injection。

### Milestone 8 — host loss
OFFLINE -> UNKNOWN、auto reassign無し、same instance reconnect。

### Milestone 9 — lifecycle remote surface
show/list/renew/reconcile/destroy/test/log/artifact。global RELEASEDはworker cleanup proof後。

### Milestone 10 — artifact transport
local result first、digest upload、upload-only retry。

### Milestone 11 — Android UI/Browser
controller fence + worker local fence。existing stale/identity safety維持。

### Milestone 12 — local protection
controller-managed leaseをordinary local mutation/GCから保護。

### Milestone 13 — two-worker integration
real TLS socket、separate state/DB、placement/load-balance/drain/offline/reconnect/cleanup。

### Milestone 14 — native OS
Windows/macOS/Linuxでcontroller+worker+client processをnative実行。

### Milestone 15 — real multi-host
可能ならseparate machine/VM。same-host workerをphysical evidenceと呼ばない。

### Milestone 16 — docs/completion
README/Architecture/Portability/Security/Reliability/Quality/Roadmap/standalone/index英日。
PR #11のescaped-defect guardrailをprotocol/state boundary testへ適用。

## 具体的な手順

対応するGo toolchainをPATHに設定する。意味のある変更単位で`go run ./tools/repoctl check`、`go test -race ./...`、`go run ./tools/repoctl test-integration`を実行する。journal、transport、source、processの個別テストを追加し、その後native OSの証拠を収集する。実測結果を進捗と検証と受け入れに記録する。

## 検証と受け入れ

| ID | 必須動作 | 証拠 |
| --- | --- | --- |
| M1 | local mode daemon-free/non-regression | localの全harness/raceとDocker、Podman、Android、Flutter/UI、Browser統合が成功。最終native harnessは未完了。 |
| M2 | persistent controller identity + single authority | controller永続化・再起動、native process lock/crash解放、実TLSでのidentity保持を検証。 |
| M3 | host ID + host-instance identityでreplacement誤adopt防止 | host-instance拒否テストとnative worker再起動時のidentity保持が成功。 |
| M4 | production mTLS/auth、unenrolled拒否 | 実TLSで未登録証明書とworkerによるclient操作を拒否。role/host/blob ACLの負例が成功。 |
| M5 | worker outbound only | native fixtureはoutbound接続のworkerを2process使用し、worker管理用listenerを持たない。 |
| M6 | protocol mismatchをeffect前reject | 非互換inventory、upgrade、配置/poll拒否とworkerの実行前停止テストが成功。 |
| M7 | controller/globalとworker/local authority分離 | controller DB、worker journal、local state DBを分離。local管理metadataの不変性を検証。 |
| M8 | whole lease exactly one worker | native 2-worker fixtureでlease内の2つのprocess runtimeを同一workerへ配置。 |
| M9 | scheduler capability/capacity/host state検証 | 容量の原子的な予約、Android slot、ONLINE/capability、決定的schedulerのテストが成功。 |
| M10 | explicit hostでもsafety bypass無し | 未登録hostとdraining hostへの明示要求を実TLSで拒否。 |
| M11 | source aliasはcommit+bundle digest、client absolute path非使用 | commit済み複数alias bundleと固定commitのworktree lifecycleを検証。pathはsource aliasへ正規化。 |
| M12 | corrupt/wrong source effect前fail | 不正digest/bundle、shallow/LFS/submodule、CASのsize/path負例を作用開始前に拒否。 |
| M13 | worker source/manifest/stack独立verify | commit済みcontrol fileの変換証明、整合した偽造metadataの拒否、JSON順序変更を検証。 |
| M14 | global lease ID/epoch effect前persist | global IDと管理tupleがsource materialization前に予約されることをテスト。 |
| M15 | duplicate operationでmutation再実行無し | journalの競合/重複検査と実TLSの同一ID再試行で、証明fileへの追記は1回。 |
| M16 | effect後result lossをjournalからrecover | 結果喪失時はlocal状態を不確実として回復し、作用を再実行しない。結果再送も検証。 |
| M17 | artifact failureはuploadのみretry | upload/ack喪失後は再実行せずpublicationだけを再試行するテストが成功。 |
| M18 | heartbeat expiryはUNKNOWN、absence扱い無し | offline時はUNKNOWNとlast-known stateを表示し、予約容量を維持。 |
| M19 | offline lease auto reassignment無し | offline時の再配置拒否と保守的なhost削除テストが成功。 |
| M20 | same worker reconnectでsame lease reconcile | native worker再起動でinstance、lease、workload PID/start identityを保持。 |
| M21 | same host-id/different instance active時reject | 別instanceによる既存assignmentの引き継ぎを拒否するテストが成功。 |
| M22 | controller restart duplicate effect無し | store再openでoperation ID/result/epochを保持し、実controller再起動でも配置を保持。 |
| M23 | worker restart existing resource/journal recover | journalの再起動/結果喪失と、nativeの稼働processを保持した再起動が成功。 |
| M24 | controller outageでworker auto GC無し | controller outageでもendpointが稼働し、managed leaseをGC対象から除外。 |
| M25 | local mutation/GCでcontroller-managed lease変更不可 | managed mutation/GC/UI/Browser/cancel fenceとSQLite Saveの不変性テストが成功。 |
| M26 | remote destroyはworker cleanup proof後のみglobal RELEASED | controllerの解放証拠負例と、作用前journal証明のrestart/dry-run/不正入力テストが成功。 |
| M27 | worker local endpointをclient localと偽らない | worker応答はendpoint_scope=worker-localを表示し、native fixtureでも所有情報を検査。 |
| M28 | test/log/artifact remote、raw shell無し | 実TLSのnamed test、冪等再試行、run/live log、artifact digest取得・上書き拒否がLinux/macOSで成功。Windows相対実行pathは修正検証中。 |
| M29 | Android UI既存stale/device/fence維持 | Worker AppExecutorから実appのUI経路をfake Android providerで検証し、stale・digest改変snapshot、device変更、runtime不一致、managed assignmentのfenceを確認した。local実Android UI baselineも成功。 |
| M30 | Browser既存process/page/snapshot/focus/stale維持 | 実remote Chrome/CDP snapshot/pagesとartifact取得が成功。既存Browserのstale/focus/process検査も維持。 |
| M31 | two worker concurrent lease isolation | 実2-workerで別worktree/portを使用し、片方を破棄しても他方が応答。 |
| M32 | drainはnew placement停止のみ | 実TLS drain/undrainと安全性テストが成功し、移動・破棄を行わない。 |
| M33 | active/stale/uncertain host remove拒否 | active/stale/uncertain assignmentがあるhostの削除を拒否し、解放証明後のみ許可。 |
| M34 | CAS content-addressed/atomic/digest/concurrent safe | CASの同時重複writer、digest/size、原子的directory公開、破損負例が成功。 |
| M35 | caller pathをCAS authorityにしない | digestだけをCAS pathの入力とし、登録artifactの所有/path/symlink検査が成功。 |
| M36 | client secret implicit forwarding無し | 実TLSでclient専用token不在とworker env解決を確認。helper負例も成功（Linux18.394s）。 |
| M37 | Windows native protocol integration | CI 34314956327の初期lifecycleはWindowsで成功。拡張named testと長いpathは修正検証中。 |
| M38 | macOS native protocol integration | native multi-host CI 34314956327と拡張fixtureの34315479224がmacOSで成功。 |
| M39 | Linux native protocol integration | native Linux CI、拡張local TLS fixture、実remote runtime統合が成功。 |
| M40 | real socket two-worker scheduling/outage/recovery/cleanup | 実TLSのcontroller/client/2-worker fixtureで配置、outage、再起動、復旧、cleanupを検証。 |
| M41 | physical/VM evidenceをhonestに区別 | 同一物理host上の複数role processを使用。別machine/VMの証拠はなく、主張しない。 |
| M42 | existing local runtime integration非回帰 | 実local Docker、Podman共存、Android Emulator、Flutter/Android UI、Browser統合が成功。 |
| M43 | HA/live migration/split lease/tunnelをimplementedと宣伝しない | product/design/READMEはHA、移動、split-host lease、tunnel、remote cancel-activeを将来課題と明記。 |
| M44 | 英日docs authority/trust/failure/recovery | product/design/ADR、architecture、README、運用文書を日英で更新しdocs/translation検査が成功。 |
| M45 | final repoctl/docs/race/native/integration/release | local harness/raceと実6target release candidate build/check/native smoke/改変負例が成功。最終native CIは未完了。 |
| M46 | 英日ExecPlan evidence/retrospective後archive | 最終native証拠、振り返りの照合、両言語archiveが未完了。 |

## 冪等性と復旧

readはretry可能。
mutationはoperation ID/epochでfenceし、transport uncertainty時はnew mutationを発行せず
worker journal query/reconcile/result resume。

source/artifact transferはdigestでretry。
host OFFLINEはcleanup eventではない。

## 成果物と注記

最終実装のチェックポイント`e46f807`で、ローカルの`repoctl check`と`go test -race ./...`が成功した。実release-candidate fixtureは20.385sで成功し、6種類すべてのarchiveを生成・検証してnative smokeと改変拒否の検査を行った。更新後のremote Browser/Docker/Podman E2Eは33.142sで成功した。native multi-host 34317327009とBrowser 34317326997はWindows、macOS、Linuxすべてで成功した。archive前にVerify 34317326990を確認している。

追加証拠: 実remote Browser/Docker/Podmanが35.659s、拡張2-worker named-test/log/artifact/renewが18.907s、client/worker環境分離が18.394sで成功。`b1a7df6`で`AGENT_ENV_RELEASE_CANDIDATE=build go test ./tools/repoctl -run '^TestReleaseCandidate$' -count=1 -v -timeout=20m`が成功し、隔離したprivate source/tag fixtureを使って実6target archive、検査、native smoke、改変負例を検証した。公開tag/releaseは作成していない。`d993965`のnative multi-host CI 34314956327は3OSすべて成功。拡張fixtureの34315479224はLinux/macOSで成功し、Windowsでは子processのcwd適用前の相対実行path検索が失敗した。Windows全harnessでは300文字を超えるGit pathの追加ケースも失敗した。どちらも対応するnative検証が成功するまで失敗記録を維持する。

2026-09-09の統合milestone: `6d9dd7a`（local管理境界）と`0f05d09`（controller/worker/source/CLI）を`origin/feat/multi-host-control-plane`へpush。localの全`repoctl check`と`go test -race ./...`が成功。`TestMultiHostNativeCLI`は実TLSとbuild済みbinaryで21.406sで成功し、配置、role拒否、drain、隔離、outage、controller/worker再起動、cleanupを検証した。既存Browser native/secret、Podman共存（107.695s）、実Android Emulator、実Flutter/Android UI、Docker `repoctl test-integration`も成功。Androidの初回はtemplate指定不足で失敗し、導入済みの停止中templateを明示して再実行した。host固有の前提pathは記録しない。native CIは進行中で、macOSでは上記path alias不具合を検出した。追加remote runtime E2Eの証拠を収集中。

2026-09-09の検証証拠: `go test -race ./internal/worker`成功（1.073s）。receipt再送、upload/ack失敗、結果喪失時の不確実状態、作用直前のheartbeat検査を含む。`go test -race ./internal/instance`成功（1.035s）。nativeの別process排他とcrash後の解放を含む。管理境界のapp/local-store/domain raceは50.167s/16.235s/1.030sで成功。修正前のStore.Saveをoverlayで使用し、6件の不正上書きを再現した。source/CASの反復raceは5.380s/1.011sで成功。architecture境界fixtureは0.028sで成功。この時点ではnative Windows/macOSと実TLSの2-worker受け入れは未完了で、その後の成功を上に記録した。

実装したcontroller state（controller IDはSQLiteに保存）:

```text
<AGENT_ENV_HOME>/control-plane/
  controller.db
  controller.lock
  blobs/<sha256>/data
```

workerはexisting state + host-instance/controller binding/journal。

central evidenceはcontroller/protocol/global lease/host instance/epoch/operation/source
 digest/worker version/local observed state/artifact digest/ackを記録。

TLS private keyをSQLiteへ保存しない。

## インターフェースと依存

候補package:

```text
internal/controlplane/
internal/controlplane/store/
internal/controlplane/protocol/
internal/worker/
internal/remotesource/
internal/blobstore/
```

control-plane schedulerからconcrete runtime adapterをimportしない。
worker wiringがexisting app/runtimeを再利用。

標準library HTTP/TLS/JSONを優先し、実要件不足の証拠がない限りgRPC/protobufを追加しない。

multi-host modeはlong-running controller/worker processを導入するが、local modeはdaemon不要。

## 初期論点の決定

初期の設計論点は以下のとおり決定した（2026-09-09、実装担当）。

1. **CLI:** `control-plane serve/enroll`、`worker serve`、明示的なglobal `--controller`とTLS file flagを採用。
2. **Compatibility:** protocol 1とproduct version完全一致を要求。非互換inventoryを表示し、配置/pollとlocal作用を拒否。
3. **Certificates:** 外部で用意した証明書のleaf fingerprintをlocalで登録。発行とrotationは将来課題。
4. **Roles:** clientとhostに対応するworkerを別登録し、endpointアクセス前にroleを認証。
5. **Polling:** workerが最大30秒のlong-pollと独立heartbeatを開始。dispatchは直列。
6. **Process lock:** service稼働中は別SQLiteのnative exclusive lockを保持し、終了時はOSが解放。
7. **Git bundles:** exact commitに固定した完全なlocal bundleを転送し、shallow repositoryを拒否。
8. **LFS/submodules:** 外部contentをfetchせず、Git LFS pointerとsubmoduleを拒否。
9. **Transfer bounds:** source blobは1 GiB、artifactは64 MiBを上限とする。digest単位で全体を再送し、部分resumeは将来課題。
10. **CAS retention:** immutable CAS objectを保持し、自動CAS GCは将来課題。DBとCASを一体で保護・backupする。
11. **Capabilities:** git、compose.docker、compose.podman、android-emulator、flutter-android、persistent-process、browser-cdpを使用し、worker preflightも必須。
12. **Capacity:** 最大lease数とAndroid slotを原子的に予約。CPU/memory schedulerは提供しない。
13. **Selection:** 安定したhost-ID順で最初の適格hostを選ぶ。明示選択でも同じ検査を行い、labelと負荷分散は将来課題。
14. **Plan authority:** workerがcanonical package/manifest/source-set digestと、commit済みcontrol fileから許される変換を検証。
15. **Local metadata:** materialization前にlocal lease JSONへ不変のremote_managementを保存し、SQLite Saveで後付け・tuple変更を拒否。
16. **Operations:** version付きidentity envelopeと列挙された型付きpayload、再利用可能なoperation IDを使用し、raw-shell endpointを追加しない。
17. **Degraded access:** 既存app UI/Browserのreadiness、stale/device/process/page/focus、復旧policyを再利用し、remoteで検査を迂回しない。
18. **Offline views:** offlineではUNKNOWNとlast-known stateを表示し、不在を推定したり予約を解放したりしない。
19. **Permanent loss:** localの緊急引き継ぎは未実装。元のcontroller authorityを復旧し、通常force/GCではremote leaseを引き継げない。
20. **Backup:** service停止中の整合したDB/CAS backupを使い、元のidentityを維持する。rootのcopyを第2の有効authorityにしない。
21. **Machine evidence:** 別物理host/VM証拠は任意で、この環境では取得不能。同一host上のnative role検証と明確に区別。
22. **Tunnels:** endpoint tunnelと将来のtopologyは対象外。現在のendpoint範囲はworker-local。
23. **Split-host leases:** 分散resource graphは別設計が必要。今回の実装は1 leaseを1 workerに保持。
24. **Regression guardrails:** 修正前の失敗再現、独立レビュー、実TLS/binary検証、障害境界テスト、native OS CIを使い、失敗した試行と証拠を保持。
