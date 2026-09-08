---
status: completed
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/completed/compose-provider-podman.md
source_sha256: d4788803ec50db5feb9cdd614205fa8f18102503d77b075282e2733186131bb7
---

# Compose runtime provider として podman-compose を追加する

[English](compose-provider-podman.md)

この ExecPlan は living document であり、`docs/PLANS.md` に従って更新する。

想定ブランチ: `feat/compose-provider-podman`

standalone-distribution作業を先にmergeして`master`から開始するのを推奨する。
stacked PRの場合はbase branch/commitを以下へ記録し、merged-master evidenceと
混同しない。

開始base branch: `master`（standalone PR 6のmerge後）。
開始revision: `f239fe5`

対象upstream:
- https://github.com/containers/podman-compose
- 初期tested floor: `podman-compose >= 1.6.0`

未検証の旧Podman構成を対応済みと記載しない。実際のclient/serverと
podman-composeのバージョンを受け入れ証拠へ記録する。

## 目的 / 全体像

既存の`type: compose` runtimeをDocker Composeまたは`podman-compose`で
materializeできるようにする。

provider省略は既存互換Docker:

```yaml
runtimes:
  backend:
    type: compose
    source: backend
    project_directory: .
    files: [compose.yaml]
```

Podman:

```yaml
runtimes:
  backend:
    type: compose
    provider: podman-compose
    source: backend
    project_directory: .
    files: [compose.yaml]
```

Docker明示:

```yaml
runtimes:
  backend:
    type: compose
    provider: docker-compose
    source: backend
    project_directory: .
    files: [compose.yaml]
```

providerはlease snapshotの一部でimmutable。host上のexecutable有無による
Docker/Podman auto fallbackは禁止する。

stack/componentはprovider非依存を維持し、providerはCompose workloadを
「どう実現するか」だけを表す。

Podmanでもengine identity、resource ownership、dynamic endpoint、sibling
survival、conservative cleanup、uncertainty quarantineを維持する。

## 進捗

- [x] 2026-09-08: merge済みmaster f239fe5を記録し`feat/compose-provider-podman`を作成。
- [x] 2026-09-08: baseline harness/race
- [x] 2026-09-08: current Compose/app/domain/store/CLI調査
- [x] 2026-09-08: 英日product/design docs
- [x] 2026-09-08: provider field/default/strict validation
- [x] 2026-09-08: plan/snapshot/show/doctorへprovider
- [x] 2026-09-08: provider-neutral Compose boundary
- [x] 2026-09-08: Docker implementation移行
- [x] 2026-09-08: Docker real integration非回帰
- [x] 2026-09-08: Doctorとバージョン検査を実装。`TestPodmanDoctorVersionFloorAndSnapshot`で1.3.0・2.0.0の拒否と1.6.0 fixtureの受け入れを確認。
- [x] 2026-09-08: local/remote識別情報を保存しnative childの接続先を固定。`TestPodmanIdentityPinsLocalAndRemote`、`TestPodmanBridgeNativeRoundTrip`、`TestPodmanChangedEngineRefusesMutation`がローカルで成功。
- [x] 2026-09-08: YAMLをcanonical JSONと共通policyモデルへ変換。`TestPodmanNormalizeComposeModel`と`TestPodmanRenderRejectsProviderSpecificHostAccess`が成功。実1.6.0 lifecycleによる保存JSONの受け入れも、その後110.13秒で成功。
- [x] 2026-09-08: 未対応拡張の再帰的拒否と、値が未確定の環境変数引き継ぎの拒否を実装。Renderと正規化の回帰テストが成功。
- [x] 2026-09-08: detached Up、構造化Inspectとendpoint、時刻付きLogs、Downと再検査を実装。ローカルharness/raceと実Linux lifecycleの受け入れが成功。
- [x] 2026-09-08: Down前のanonymous volume証拠保存、中断後の保持、保存失敗時の副作用停止を実装。app/backendの回帰テストが成功。
- [x] 2026-09-08: 以下の独立レビュー7件を修正し、別担当が再確認。
- [x] 2026-09-08: 各OSで実行可能なprovider/path/argv/identityテストを追加。ローカルLinuxで成功し、`4a5de3d`のVerify `34216579481`で最終Windows/macOS/Linux・Go 1.26/1.27のnative 6 jobがすべて成功。
- [x] 2026-09-08: 英日architecture、portability、security、reliability、quality、roadmap、前提条件文書へ実装範囲と未検証範囲を反映。
- [x] 2026-09-08: network/volumeの共通所有権labelを、Dockerの証拠に加えて実Podman lifecycleでも検証。
- [x] 2026-09-08: Docker共存を有効にした`TestPodmanIntegrationConcurrentLeasesAndEvidence`が110.13秒で成功。Go 1.27.1、rootless Podman 5.4.2、podman-compose 1.6.0を使用。Podmanの2leaseがREADYになり、endpoint、named test、他lease・外部リソース・Dockerの維持、完全cleanupが成功。
- [x] 2026-09-08: 実canonical JSON、logs、実anonymous volume証拠、秘密値を伏せたartifact、接続されていた全volumeの消失を以下の受け入れ証拠と照合。
- [x] 2026-09-08: `4a5de3d`のVerify `34216579481`でWindows/macOS/Linux・Go 1.26/1.27のnative 6 jobとcross-build 5 jobがすべて成功。remote integrationのraceとDocker integrationも成功し、12 jobのworkflow全体が成功で完了した。ローカルの実Docker integrationは成功済み。実Podman Machine環境は利用できず、検証済みとは主張しない。
- [x] 2026-09-08: 最終port修正後にローカルfull checkとfull raceが再度成功し、実Docker/Podman integrationも成功。ビルド済みstandaloneのhelp/versionは、PATHを空にしてPython/Podmanを利用できない環境でも成功。
- [x] 2026-09-08: 全受け入れ証拠を照合し、英日振り返りを完成させ、両Planをcompletedへ移した。native job、ローカル実integration、workflow全体の成功が必要な証拠を満たす。
checkboxは観測済み完了のみ。UTC date、revision、command/test/run、resultを記録する。

## 想定外の発見

- 2026-09-08: 最初の実1.6.0 lifecycle実行は、共通command wrapperがproviderのstderrを落としていたため、native診断を十分に得られず失敗した。Podmanのエラーはprovider名を維持し、共通evidence処理で秘密値を伏せたstderrを最大8 KiBまで保持する。Upはadapterのエラーを取り出し、誤解を招くDockerエラーとして表示しない。次の実行でPodmanが`published: "0"`を拒否することが判明した。いずれも失敗した試行であり、受け入れ成功ではない。SIGINTにより順序どおりcleanupでき、その後の検査でcontainerが残っていないことを確認した。
- 2026-09-08: 数値・文字列のpublishedゼロは、Podman専用の一時JSONでのみpublished省略へ変換する。host_ip、target、protocol、保存済みcanonical snapshotとdigestは維持する。その後の実lifecycleが110.13秒で成功し、endpoint・共存・cleanupの受け入れも確認した。
- 2026-09-08: 実fixtureは失敗時の一時ディレクトリを保持し、createが失敗したりJSONを返さなかった場合も専用registryからlease識別情報を回収する。fixtureのimage・外部volumeのcleanupはlease cleanup確認後に行い、不確実な場合は入力と証拠を保持する。検証失敗で復旧証拠まで失うことを防ぐ。

- 2026-09-08: 導入済みpodman-compose 1.3.0は予約前に拒否された（前提条件エラー、exit 3）。これはlifecycle成功ではない。その後ユーザーがpodman-compose 1.6.0を提供した。その後rootless Podman 5.4.2での実共存テストが110.13秒で成功した。
- 2026-09-08: 独立レビューで7件を発見し、policyを弱めず修正・再確認した。
  1. inventoryが残存provider行に依存し、孤立リソースだけのproviderを見落とした。登録済みproviderと利用可能なhost engineの和集合を調べ、inventory専用Doctorはpodman-composeを要求しない。
  2. root/service直下の`x-podman`だけを拒否していた。network、service内network、secretの拡張も再帰的に拒否する。
  3. Podmanの`glob` mountや`ns:`等の固有network modeが共通のbind/host network検査を回避できた。Renderが未対応形式を副作用前に拒否する。
  4. `Exists`算出後に残存anonymous volumeを追加していた。Inspectは最終リソース集合から存在を判定する。
  5. snapshotを一時ディレクトリへ移すとserviceの相対`env_file`、secret/config pathが壊れた。参照ファイルを検証・範囲制限して絶対pathへ変換し、最初のComposeファイルとproject directoryの不一致は明示的に未対応とする。
  6. service識別をDocker互換labelだけに依存していた。Podman native service labelを必須とし、互換labelとの矛盾を拒否する。
  7. null map/bare listの環境変数がUp時に再解決されていた。Renderで値未確定の引き継ぎを拒否し、明示的な空文字・literal値を維持する。hostの秘密値を診断へ出さない。

- 2026-09-08: ユーザーがPodman 5.4.2とpodman-compose 1.3.0を導入し、rootless infoは動作しました。composeは必須の1.6.0に届かないため、更新または隔離した検証環境への導入許可を問い合わせています。下限は変更しません。
- 2026-09-08: full checkとraceの並行実行中にAndroidのport所有権テストが一度失敗しました。raceとDocker integrationは成功しました。同じfull checkを単独で再実行した結果は成功しました。Androidには変更していません。

- 2026-09-08: baselineのGoテストとvetは成功しました。full checkは提供された日本語Planのtranslation_of/source_sha256不足で停止しました。翻訳を確認してmetadataを補い再実行します。
- 2026-09-08: Docker 29.7.2は利用できますが、Podmanとpodman-composeはPATH上にありません。導入はPlanの対象外のため、既存環境か導入の追加許可を問い合わせ、独立した実装を進めています。

config normalization、Compose option差、`x-podman`、label、health、dynamic port、
rootless networking、anonymous volume、Podman Machine forwarding、Windows process、
path/space/non-ASCII、same Compose fileのDocker/Podman behavior差を記録する。

Docker前提で差を隠さない。

## 判断の記録

- 2026-09-08 / maintainers: 永続JSONの共通dynamic port表現を維持し、Podman実行境界でのみpublishedゼロを変換する。upstream 1.6.0はpublished省略とhost_ipを`host_ip::target`へ変換するため、loopbackへのbindを保ちながらengine割り当てportを要求できる。provider構文に合わせるためfixed port policyを弱めたり、検証済みsnapshotを書き換えたりしない。
- 2026-09-08 / maintainers: 設定エラーとengineエラーを切り分けるためnative provider診断を残す。共通の秘密値マスク後にstderrを末尾8 KiBへ制限し、元のエラーを保持する。失敗したintegrationのregistryは復旧用に残す。診断とtest cleanupが、失敗した割り当ての修復に必要な証拠を消さないようにする。

- 2026-09-08 / maintainers: canonical JSONとdigestを保存し、Podmanにはドル記号をescapeした一時JSONを渡す。literal値の二重展開を防ぐためであり、実1.6.0のlifecycleによる受け入れもその後成功した。mountはbind/volume/tmpfs、network modeはモデル化した範囲に限定する。provider固有の副作用を黙って受け入れず、再帰的な`x-podman`と値未確定の環境変数引き継ぎを拒否する。参照ファイルはproject_directory内の通常ファイルに限定し、snapshot移動前に絶対pathへ変換する。
- 2026-09-08 / maintainers: appがDown前に`Runtime.CleanupEvidence`を永続化する。証拠はlease/runtime/project/engineの範囲、接続元container、anonymous volume fingerprintを組み合わせる。失敗・再試行で保持し、保存失敗時はDownを止める。残存volume削除前に現行の参照と識別を再検査し、cleanup確認後にだけ証拠を消す。container消失によるcleanup根拠の喪失を防ぐ。
- 2026-09-08 / maintainers: global inventoryは登録済みprovider識別情報と利用可能なhost engineの和集合を調べ、engine専用inventory Doctorを使用する。孤立リソースを発見しつつ、任意のpodman-composeをDockerだけのlifecycleの依存にしない。providerの実行先選択にfallbackはない。
- 2026-09-08 / maintainers: 同じpackage内のnative Podman command adapterを通じて既存Composeのリソース走査を共有する。native project/service labelとagent-env所有権を要求し、互換labelの矛盾を拒否する。fingerprintはendpointとhost/store構成を固定するもので、engine世代を不変に識別するものではない。同じ構成での再作成にもリソース所有権検査が必要となる。

- 2026-09-08: provider省略時のcanonical manifest内容とdigestを保ち、domain snapshotでのみDockerの既定値を確定します。inventoryのキーにはprovider、engine識別情報、projectを含めます。
- 2026-09-08: podman-composeの--podman-argsはsubcommand後に追加され、global remote flagを確実に固定できないため、native agent-env child bridgeを使用します。固定flagを先頭へ付け、環境の接続先指定を除去します。shell wrapperは使用しません。

- 判断: `type: compose`を維持しprovider fieldを追加。
  理由: stack/componentをengine implementationから独立させる。
  日付/担当: 2026-09-08 / maintainers.

- 判断: provider省略は`docker-compose`。
  理由: existing manifest非回帰。
  日付/担当: 2026-09-08 / maintainers.

- 判断: Docker/Podman auto fallback禁止。
  理由: reproducibility/cleanup identityを壊すため。
  日付/担当: 2026-09-08 / maintainers.

- 判断: initial Podman providerはpodman-compose 1.6.0+。
  理由: current stable baselineから開始しlegacy差でscopeを拡大しない。
  日付/担当: 2026-09-08 / maintainers.

- 判断: Podman/podman-composeはoptional host prerequisiteでbundleしない。
  理由: standalone one-binary coreを維持する。
  日付/担当: 2026-09-08 / maintainers.

- 判断: mutable connection nameではなくengine identityをpin。
  理由: default/connection変更でcleanup先を変えない。
  日付/担当: 2026-09-08 / maintainers.

- 判断: common host policyをprovider作成前に適用しunmodeled Podman extensionを拒否。
  理由: providerをpolicy bypassにしない。
  日付/担当: 2026-09-08 / maintainers.

- 判断: `podman-compose down --volumes`だけでcleanup完了としない。
  理由: anonymous volume等でDockerとの差があるためactual residualを再検証する。
  日付/担当: 2026-09-08 / maintainers.

- 判断: durable docs/ExecPlanは英日。
  理由: repository policy。
  日付/担当: 2026-09-08 / maintainers.

## 成果と振り返り

2026-09-08に完了。Compose runtimeは`docker-compose`または`podman-compose`を
明示的に選択でき、省略時と旧snapshotではDockerの動作を維持する。
provider識別情報を副作用前に保存し、別providerへのfallbackは行わない。
native child bridgeがPodmanの接続先を保存済みlocal/remote経路へ固定し、
engine構成の再確認とnative所有権labelでcleanupを保護する。

実Linux fixtureはGo 1.27.1、rootless Podman 5.4.2、podman-compose 1.6.0、
Docker共存有効の条件で110.13秒で成功した。同時READY lease、選択closure、
endpoint疎通、named pass/fail test、秘密値を伏せたartifact/log、他leaseと
外部リソースの維持、接続されていた全volumeのcleanupを確認した。
ローカルfull check/raceと実Docker integrationも成功した。`4a5de3d`のVerify
`34216579481`でnative OS/toolchain 6 jobとcross-build 5 jobがすべて成功し、
remote raceとDocker integrationも成功した。12 jobのworkflow全体が成功で完了した。

実行失敗により、fakeテストでは見えていなかったprovider差を発見した。
native stderrの欠落がportゼロ構文の拒否を隠していたため、診断は秘密値を伏せ、
サイズを制限したnative証拠を保持する。publishedゼロは実行用の一時copyでのみ
省略し、hostへのbindとcanonical snapshot/digestを維持する。anonymous volumeの
接続証拠はcontainer削除後も必要となるため、appがDown前に保存し失敗後も保持する。
cleanupは生成名だけを根拠にせず、global pruneも行わない。

対応範囲では、未モデル化のPodman拡張・mount/network mode、値未確定の環境変数
引き継ぎ、曖昧な参照pathを明示的に拒否する。providerツールは任意依存であり、
ビルド済みhelp/versionは空PATHでも成功した。fingerprintはendpointとhost/store
構成を識別するもので、不変のengine世代ではない。同じ構成で再作成された場合も
リソース所有権検査が必要となる。nativeテストはprocess/path/identity動作の証拠で
あり、実Machineネットワークの証拠ではない。Podman Machine環境は利用できず、
その実検証は行っていない。profile overrideとより広いprovider形式は将来範囲に残す。

独立レビュー7件を修正・再確認してから実受け入れを行った。Docker互換fieldや
config parseの成功だけから同じ動作を推測せず、native providerの意味と復旧証拠を
検証する必要がある。失敗したfixtureのディレクトリ・専用registryは復旧用に保持し、
成功が確認されたfixtureは他リソースを乱さず自分のリソースを削除する。

## 背景と構成

実装前に読む:

- `AGENTS.md` / `.ja.md`
- `ARCHITECTURE.md` / `.ja.md`
- `docs/PLANS.md` / `.ja.md`
- MVP/Compose英日product/design docs
- `PORTABILITY` / `SECURITY` / `RELIABILITY` / `QUALITY` / `roadmap`英日
- standalone distribution docs merge後
- completed MVP ExecPlan
- Flutter Android endpoint docs
- `internal/runtime/compose`
- `internal/app`
- `internal/domain`
- `internal/config`
- `internal/policy`
- `internal/execx`
- `internal/store/sqlite`
- Docker integration fixtures

現行Compose adapterはDocker context、Docker Compose v2、Docker Engine inspection、
Docker labels、endpoint parsingを直接所有している。まずDocker behaviorを変えず
mechanismをprovider boundaryへ分離する。

upstream:
- https://github.com/containers/podman-compose
- https://docs.podman.io/en/latest/
- https://compose-spec.io/

## 作業計画

### Milestone 1 — Contract / provider boundary

英日追加:

    docs/product-specs/compose-providers.md
    docs/product-specs/compose-providers.ja.md
    docs/design-docs/compose-providers.md
    docs/design-docs/compose-providers.ja.md

`provider: docker-compose|podman-compose`をstrict validation。
providerをplan/durable snapshotへ含める。

Docker-specific codeをprovider interfaceへ移動し、Docker unit/race/real
integrationがassertion弱体化無しでpassするまでPodman effect実装へ進まない。

### Milestone 2 — Podman Doctor / engine identity

Linux rootlessを最低限、可能ならPodman Machineで:

    podman --version
    podman info --format json
    podman system connection list --format json
    podman-compose version

を記録しnon-secret engine fingerprintを決める。

local/remote/Machineを区別。remoteではmutable connection nameだけをauthorityにせず、
exact endpoint/identityをlater child podman commandへ固定する。

destructive effect前にfingerprint再確認。mismatchはquarantine。

### Milestone 3 — Render normalization / policy

`podman-compose config` YAMLをparseしcommon normalized modelへ変換、existing
host policy適用、selected closure prune、immutable config/digest保存。

podman-compose 1.6.0がcanonical JSONを安定acceptするかreal testする。
可能なら両provider common JSON、不可ならcommon model + deterministic
provider-specific serialization。

behavior-changing `x-podman`やcommon policy未対応resource typeはreject。

### Milestone 4 — Up / Inspect / endpoint

explicit detached。provider `--wait`をreadiness authorityにしない。

direct structured Podman inspection:

    podman ps --all --filter ... --format json
    podman inspect
    podman network ls/inspect
    podman volume ls/inspect
    podman port

`latest`/list order禁止。

real Linux rootlessでdynamic loopback endpoint証明。
Podman Machineはagent-env hostからreachableとnativeに証明できるまでlocal endpoint
同等とadvertiseしない。

### Milestone 5 — Ownership labels

common:

    io.agent-env.lease
    io.agent-env.runtime

Docker native `com.docker.compose.*`、Podman native `io.podman.compose.*`等は
provider observation。

Docker compatibility labelだけをPodman sole authorityにしない。

network/named volumeへagent-env labelを付けられるか両providerで検証。
不可ならnative project label + recorded ID proofをdocument。

nameのみでdestroyしない。

### Milestone 6 — Logs / tests

Docker-only flagを仮定せずlogs実装。必要ならdirect
`podman logs --timestamps`。

service/container attribution維持。

Docker fixtureと同じnamed test/evidence pathを使い、provider-specific container
nameをtestへ漏らさない。

### Milestone 7 — Cleanup / residual

down前にowned containerのresource、anonymous volume IDをcapture。

down後actual state再inspect。

residual anonymous volumeはowned container由来かつexternal/sibling containerから
未参照を証明した時だけremove。それ以外quarantine。

禁止:

    podman system prune
    podman volume prune
    podman network prune

### Milestone 8 — Concurrency / coexistence

- same source two Podman lease
- A destroyでB維持
- Docker+Podman coexist
- project-name coincidenceでcross ownership無し
- default connection変更でredirect無し
- remote identity変更でcleanup block

same application/E2E fixtureを両providerで実行し差を正直に記録。

### Milestone 9 — Native / real evidence

real required:

- Linux rootless
- podman-compose >=1.6.0
- two leases
- dynamic endpoint
- named test
- sibling survival
- cleanup

native Windows/macOS/Linux CIはparse/argv/path/provider selection/identity/pinningを
shell無しで検証。

possibleならreal Podman Machine。fake/cross-buildをreal validationと呼ばない。

### Milestone 10 — Docs / completion

英日README/Architecture/Portability/Security/Reliability/Quality/Roadmap/index、
standalone prerequisite matrix更新。

| Provider | External prerequisite | Initial floor |
| --- | --- | --- |
| Docker Compose | docker + Compose v2 | existing |
| Podman Compose | podman + podman-compose | 1.6.0 |

Pythonをcore dependencyとは記載しない。

final harness/translation/race/integration、evidence/retrospective後completedへ。

## 具体的な手順

1. base/branch/revision記録
2. 英日active plan
3. baseline harness/race
4. bilingual provider docs
5. manifest provider
6. Docker refactor
7. Docker revalidation
8. Podman Doctor/identity
9. config normalization/policy
10. Up/Inspect/endpoint
11. ownership
12. logs
13. Down/residual
14. anonymous volume regression
15. two-Podman isolation
16. Docker+Podman coexistence
17. default connection regression
18. Linux rootless integration
19. same E2E both providers
20. native OS tests
21. possibleならPodman Machine
22. bilingual docs
23. final harness/race/integration
24. direct evidence
25. bilingual retrospective
26. completed/link/hash

## 検証と受け入れ

| ID | 必須動作 | 証拠 |
| --- | --- | --- |
| P1 | provider省略existing ComposeはDockerで既存integration非回帰 | baseline `aee3a3d`で実Docker integrationとVerify `34213899668`が成功。 |
| P2 | explicit docker-composeはdefaultと同等 | providerのplan/config互換性テストがローカルで成功。 |
| P3 | explicit podman-composeのみ選択しDocker fallback無し | 明示dispatcherとfallbackなしのCLI/appテストがローカルで成功。 |
| P4 | unknown/non-Compose providerをeffect前reject | strict manifestの拒否fixtureがローカルで成功。 |
| P5 | plan/snapshot/showにprovider identity | plan/domain snapshotと予約前create検査がローカルで成功。 |
| P6 | Docker context/cleanup safety非回帰 | 選択・cleanup変更後の実Docker integration再実行と最終ローカルfull check/raceが成功。 |
| P7 | Podman Doctorがprovider/client/server/mode/non-secret identity記録 | Doctor fixtureと実1.6.0/Podman 5.4.2 rootless lifecycleが成功。 |
| P8 | default connection変更後もrecorded engineへ固定 | local/remote識別とnative bridgeテストが`4a5de3d`のVerify `34216579481`のOS/Go 6 jobで成功。 |
| P9 | engine mismatch/ambiguityでcleanup block/quarantine | engine変更時の副作用拒否とcleanup隔離・再試行fixtureがローカルで成功。 |
| P10 | configがcommon host-policy modelへ入る | 正規化・Render policy回帰と実canonical JSON lifecycleが成功。 |
| P11 | unmodeled Podman extensionでpolicy bypass不可 | 再帰的拡張とprovider固有hostアクセスのRender拒否fixtureが成功。 |
| P12 | selected service closureのみdetached start | 選択closureとdetached実lifecycleのfixtureが成功（110.13秒）。 |
| P13 | structured Podman inspectでowned resource/readiness | native label/healthと残存のみの回帰に加え、実READY・resource観測が成功。 |
| P14 | real Linux rootless dynamic endpoint、fixed host port reject | 実dynamic loopback HTTP endpointとfixed port policy拒否テストが成功。 |
| P15 | unsupported Docker-only flag無しでtimestamp/service attribution logs | 実logsとnamed testのartifact・秘密値マスク検査が成功。 |
| P16 | destructive effect前にownership verify | native project/serviceと矛盾labelのfixture、実所有リソース削除が成功。 |
| P17 | two Podman leasesがdistinct resource/endpointでREADY | 実同時2leaseのREADY・resource/endpoint分離fixtureが成功（110.13秒）。 |
| P18 | A destroyでB/external resource維持 | 実lease削除後に他leaseと外部リソースを維持する検査が成功。 |
| P19 | Docker/Podman coexist、cross observation/cleanup無し | provider別inventoryテストと実Docker/Podman共存が成功。 |
| P20 | anonymous-volume差を検出しsafe residual handling/quarantine | 証拠・置換・再試行・保存失敗fixtureが成功。実anonymous証拠とcleanup後の接続されていた全volume消失も成功。 |
| P21 | global Podman prune無し | backendレビューと実範囲限定cleanupでglobal prune commandを使用していない。 |
| P22 | same E2E fixture両providerでpassまたは差document | Docker共存を有効にして同じnamed pass/failと証拠・artifact検査が成功。 |
| P23 | real Linux rootlessでcreate/endpoint/test/sibling/cleanup | 実Linux rootless lifecycleが110.13秒で成功。正確なversion・commandは以下。 |
| P24 | Windows/macOS/Linux native provider/path/argv/identity test、shell不要 | `4a5de3d`のVerify `34216579481`でWindows/macOS/Linux・Go 1.26/1.27のnative 6 jobとcross-build 5 jobがすべて成功。 |
| P25 | Podman Machine evidenceをavailable時記録、fakeで代替しない | 実Machine環境は利用不可。実Machine検証済みとは主張しない。 |
| P26 | Podman optional、core standaloneはPodman/Python不要 | ビルド済みstandalone help/versionが空PATH・Python/Podmanなしで成功。 |
| P27 | 英日durable docsがfinal contract/prerequisite説明 | 英日最終範囲・前提条件・証拠更新とdocs-checkが成功。 |
| P28 | final harness/translation/race pass | 最終ローカルcheck/race、実Docker/Podman integration、native 6 jobが成功。remote raceとDocker integrationも成功し、Verify `34216579481`全体が成功で完了した（12 job）。 |
| P29 | 英日ExecPlanにdirect evidence/retrospective後archive | 英日受け入れ証拠と振り返りを完成させ、両Planをarchive済み。 |

code存在だけではacceptanceではない。exact provider version、successful
command/test/runを記録する。

## 冪等性と復旧

plan/Doctorはread-only。

providerはlease内immutable。provider/engine identityをresource effect前にpersist。

effect不確実なPodman commandを自動retryせずrecorded engineを再observeする。

recovery中provider switch禁止。current/default connectionをcleanup authorityにしない。

repeated destroyはactual stateを再inspectしproven-owned residualのみremove。
ambiguousはquarantine。

Podman対応のためDocker assertion/common policyを弱めない。

## 成果物と注記

### 2026-09-08 実装チェックポイント

- 2026-09-08の追加検証: provider選択とcleanup永続化の変更後に、実Docker integration全体が再度成功し、ローカル`repoctl check`全体も成功した。その後の実Podman成功は以下に記録する。最終native 6 jobとremote integrationが成功し、Verify `34216579481`全体が成功で完了した。
- 独立した追加検証で`go test ./internal/runtime/compose -run 'TestPodman(ProviderFailurePreservesRedactedNativeDiagnostic|UpUsesDynamicPortWithoutChangingCanonicalSnapshot)$' -count=1`が成功した。native stderrの秘密値マスク、数値・文字列ゼロの変換、canonical内容の維持、host_ip/target/protocolの保持を確認する。8 KiB制限はコードレビューで確認したが、この対象テストは長い診断を別途検査するものではない。

- provider境界のbaseline commitは`aee3a3d`。native Verify run `34213899668`が成功した。これはbaselineの証拠であり、最終Podman backendのCIではない。
- 現在の未commit実装でローカル`go run ./tools/repoctl check`と`go test -race ./...`が成功した。backendの対象raceテストとapp cleanupテストも成功。最終port修正後にfull checkとfull raceも再度成功した。
- 独立した再確認で`go test ./internal/runtime/compose -run 'TestPodman(RejectsNestedExecutionExtensions|InspectRetainedAnonymousVolumeExists|RenderRejectsProviderSpecificHostAccess|NativeServiceOwnership|RelativeFileReferencesSurviveSnapshotRelocation|RejectsInitialFileDirectoryMismatchBeforeProvider)$' -count=1`と`go test ./internal/runtime/compose -run 'TestPodman(EnvironmentRequiresExplicitPassThroughValues|RenderRejectsProviderSpecificHostAccess)$' -count=1`が成功した。
- appの`TestCleanupRetainsProofAfterContainerDisappears`、`TestCleanupProofWriteFailurePreventsDown`、`TestInventoryDiscoversOrphansOutsideRecordedProviders`が成功した。副作用前の証拠保存、container消失後の安全な再試行、保存失敗時の停止、登録行が残っていないproviderの孤立リソース発見を検証する。
- 実Linux受け入れ: `AGENT_ENV_PODMAN_INTEGRATION=1`と`AGENT_ENV_PODMAN_DOCKER_COEXISTENCE=1`を設定した`go test -tags=integration ./internal/cli -run '^TestPodmanIntegrationConcurrentLeasesAndEvidence$' -count=1 -v`が110.13秒で成功。Go 1.27.1、native Linux、rootless Podman client/server 5.4.2、podman-compose 1.6.0を使用した。選択closure、同時2leaseのREADY、動的loopback endpoint疎通、時刻付きlogs、実anonymous volume証拠、成功・失敗のnamed test、artifactと秘密値マスク、cleanup後の接続されていた全volumeの消失、他lease・外部リソース・Dockerの維持を検証した。実際のcanonical JSON実行経路を通っている。永続的な証拠にlease IDやローカル実行ファイルpathは不要。
- 最終port修正後のローカル`go run ./tools/repoctl check`と`go test -race ./...`が成功した。ビルド済みstandaloneの`help`と`version`はPATHが空でも成功し、両操作にPythonもPodmanも不要と確認した。`4a5de3d`のVerify `34216579481`でnative 6 jobとcross-build 5 jobが成功。12 jobのworkflow全体が成功で完了した。実Podman Machine環境は利用不可。

2026-09-08のprovider境界の証拠: `go run ./tools/repoctl check`、`go test -race ./...`、`go run ./tools/repoctl test-integration`がすべて成功しました。integrationはPodmanの副作用を有効化する前に既存の実Docker lifecycleを検証しました。新規テストはmixed providerの予約前Doctor、不変snapshotによるcleanup、未知providerでrunnerを呼ばないこと、providerを含むinventory識別、manifestに基づくfallbackなしのCLI Doctorを検証します。

evidenceへ記録:

- provider
- podman-compose version
- Podman client/server
- local/remote mode
- non-secret engine fingerprint
- connection name/URI（credential除外）
- rootless
- project/config digest/services
- resource IDs/labels
- endpoints
- residual cleanup
- test run IDs

private key content/password/sensitive envをpersistしない。

## インターフェースと依存

実装済みdomain型:

```go
type ComposeProviderName string

const (
    ComposeProviderDocker ComposeProviderName = "docker-compose"
    ComposeProviderPodman ComposeProviderName = "podman-compose"
)
```

実装は`internal/runtime/compose`内に`dockerClient`、`podmanClient`、provider選択、正規化、native bridge、anonymous cleanupを置く。appがprovider選択と永続化を伴うcleanup手順を所有し、CLIが導入済みengineの検出を接続する。runtime adapter間のpackage importはない。

Docker prerequisite:
- docker
- Compose v2
- selected Docker engine/context

Podman prerequisite:
- podman
- podman-compose >= tested floor
- local engineまたはexplicit reachable service
- host OSで必要ならPodman Machine

core Python/shell/CGO dependencyを追加しない。

当初の設計論点と残る受け入れ:

1. key `provider` vs `compose_provider`
2. engine fingerprint field
3. podman-compose childをexact remote endpointへpinするenv/arg
4. canonical JSONをpodman-compose 1.6.0がacceptするか
5. network/volume agent-env label
6. generic x-* vs behavior-changing x-podman policy
7. Podman health mapping
8. Podman Machine loopback semantics
9. logs実装
10. anonymous volume ownership proof
11. common project-name subset
12. future profile overrideとの関係

選択、fingerprint、bridge、正規化の対応範囲、logs、anonymous証拠の永続化は上記の判断で決定済み。実LinuxでのJSON/label/endpoint動作は検証済み。最終native 6 jobとworkflow全体が成功した。profile overrideは将来範囲に残す。当初の論点は未検証事項を示すため保持し、決定済み事項を再度未決として扱わない。
