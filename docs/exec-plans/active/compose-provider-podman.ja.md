---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/active/compose-provider-podman.md
source_sha256: 50f24b392f6f0cb9856ac0fc09e81057e11989f383900fc8f52f8ff57fa91b66
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

Podman engineのminimum versionはreal integration前に固定しない。実際の
client/server/podman-compose versionをacceptance evidenceへ記録する。

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
- [ ] Podman Doctor/version
- [ ] local/remote engine identity
- [ ] recorded engine pinning
- [ ] podman-compose config common normalization
- [ ] canonical recorded config方式決定
- [ ] unmodeled Podman extension拒否
- [ ] Podman Up/Inspect/endpoint
- [ ] Podman Logs
- [ ] Down + residual reinspection
- [ ] anonymous volume regression
- [ ] network/volume ownership label強化検証
- [ ] two Podman lease isolation
- [ ] Docker+Podman coexistence
- [ ] default connection change regression
- [ ] real Linux rootless integration
- [ ] same E2EをDocker/Podmanで実行
- [ ] Windows/macOS/Linux native provider tests
- [ ] possibleならPodman Machine real integration
- [ ] bilingual durable docs/prerequisite matrix
- [ ] final harness/race/integration
- [ ] 英日acceptance/retrospective
- [ ] completed移動/link/hash更新

checkboxは観測済み完了のみ。UTC date、revision、command/test/run、resultを記録する。

## 想定外の発見

- 2026-09-08: ユーザーがPodman 5.4.2とpodman-compose 1.3.0を導入し、rootless infoは動作しました。composeは必須の1.6.0に届かないため、更新または隔離した検証環境への導入許可を問い合わせています。下限は変更しません。
- 2026-09-08: full checkとraceの並行実行中にAndroidのport所有権テストが一度失敗しました。raceとDocker integrationは成功しました。同じfull checkを単独で再実行した結果は成功しました。Androidには変更していません。

- 2026-09-08: baselineのGoテストとvetは成功しました。full checkは提供された日本語Planのtranslation_of/source_sha256不足で停止しました。翻訳を確認してmetadataを補い再実行します。
- 2026-09-08: Docker 29.7.2は利用できますが、Podmanとpodman-composeはPATH上にありません。導入はPlanの対象外のため、既存環境か導入の追加許可を問い合わせ、独立した実装を進めています。

config normalization、Compose option差、`x-podman`、label、health、dynamic port、
rootless networking、anonymous volume、Podman Machine forwarding、Windows process、
path/space/non-ASCII、same Compose fileのDocker/Podman behavior差を記録する。

Docker前提で差を隠さない。

## 判断の記録

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

未完了。

完了時にprovider syntax/architecture、Docker非回帰、tested Podman versions、
engine identity、config format、ownership、endpoint、rootless、cleanup差、
Docker-vs-Podman E2E、Podman Machine evidence/gapを記録する。

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
| P1 | provider省略existing ComposeはDockerで既存integration非回帰 | Pending |
| P2 | explicit docker-composeはdefaultと同等 | Pending |
| P3 | explicit podman-composeのみ選択しDocker fallback無し | Pending |
| P4 | unknown/non-Compose providerをeffect前reject | Pending |
| P5 | plan/snapshot/showにprovider identity | Pending |
| P6 | Docker context/cleanup safety非回帰 | Pending |
| P7 | Podman Doctorがprovider/client/server/mode/non-secret identity記録 | Pending |
| P8 | default connection変更後もrecorded engineへ固定 | Pending |
| P9 | engine mismatch/ambiguityでcleanup block/quarantine | Pending |
| P10 | configがcommon host-policy modelへ入る | Pending |
| P11 | unmodeled Podman extensionでpolicy bypass不可 | Pending |
| P12 | selected service closureのみdetached start | Pending |
| P13 | structured Podman inspectでowned resource/readiness | Pending |
| P14 | real Linux rootless dynamic endpoint、fixed host port reject | Pending |
| P15 | unsupported Docker-only flag無しでtimestamp/service attribution logs | Pending |
| P16 | destructive effect前にownership verify | Pending |
| P17 | two Podman leasesがdistinct resource/endpointでREADY | Pending |
| P18 | A destroyでB/external resource維持 | Pending |
| P19 | Docker/Podman coexist、cross observation/cleanup無し | Pending |
| P20 | anonymous-volume差を検出しsafe residual handling/quarantine | Pending |
| P21 | global Podman prune無し | Pending |
| P22 | same E2E fixture両providerでpassまたは差document | Pending |
| P23 | real Linux rootlessでcreate/endpoint/test/sibling/cleanup | Pending |
| P24 | Windows/macOS/Linux native provider/path/argv/identity test、shell不要 | Pending |
| P25 | Podman Machine evidenceをavailable時記録、fakeで代替しない | Pending |
| P26 | Podman optional、core standaloneはPodman/Python不要 | Pending |
| P27 | 英日durable docsがfinal contract/prerequisite説明 | Pending |
| P28 | final harness/translation/race pass | Pending |
| P29 | 英日ExecPlanにdirect evidence/retrospective後archive | Pending |

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

想定:

```go
type ComposeProviderName string

const (
    ComposeProviderDocker ComposeProviderName = "docker-compose"
    ComposeProviderPodman ComposeProviderName = "podman-compose"
)
```

想定layout:

    internal/runtime/compose/
    internal/runtime/compose/docker/
    internal/runtime/compose/podman/

実architectureに合わせて変更可。

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

Milestone 1で解決:

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

public behavior確定前にDecision Logで解決する。
