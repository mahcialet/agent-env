---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/design-docs/compose-providers.md
source_sha256: 0a6799e3efdcbc9745cb19a65dd49495bccadf04064026a9f9ee92fb94ba9ac9
---

[English（翻訳元）](compose-providers.md)

# Compose providerの設計

[製品契約](../product-specs/compose-providers.ja.md)が選択と安全性の要件を定める。
実装と受け入れは[進行中のExecPlan](../exec-plans/active/compose-provider-podman.ja.md)
に従う。以下の構成は実装中であり、backendの正規化の詳細と実providerの検証証拠は、
同Planの判断と検証で確定する。

## 責務と永続化

`compose.Client`が非公開の`dockerClient`と`podmanClient`へ処理を振り分ける。
appはオーケストレーション、readiness、quarantine方針を引き続き担当し、provider adapterは
process argvとengine観測を担当する。AndroidとFlutterは別のruntime境界を維持する。
providerは他のruntime adapterをimportしない。

`domain.ComposeProviderName`は`docker-compose`または`podman-compose`を識別する。
新しいCompose planでは、実効的な選択を`domain.Runtime.Provider`に記録する。
`EffectiveComposeProvider`は旧snapshotの未設定値をDockerへ対応づけ、未知の空でない
識別子は拒否できるようそのまま保持する。AndroidはCompose providerを持たない。
manifestの任意フィールドは、省略時にcanonical JSONにも含めないため、defaultを追加
しても旧manifestのdigestは変わらない。leaseのsnapshotは、その後のmanifest編集と
独立してprovider情報を固定する。

## Engineの固定とnative実行

Dockerは既存のcontext確認と所有確認を維持する。Podmanはengineがlocalかremoteか、
解決済みendpoint、観測したホストOS/architecture、storage rootを記録する。この非秘密の
fingerprintは記録した構成の変化を検出するが、不変のengine世代識別子ではない。
同じ構成を再作成するその場の初期化はfingerprintでは検出できないため、cleanup前には
resourceの識別確認も必要となる。

remote操作は、可変の接続名や現在の既定値ではなく、記録済みendpointのパラメータを
使う。継承したrouting環境変数を除去してから固定した経路を設定する。秘密鍵の内容、
password、機密の環境変数値をregistry metadata、log、commitする証拠に含めない。
再観測で記録済みの同一性を確認できなければ、破壊的操作を停止する。

podman-composeはPodmanの子processを起動する。現在のagent-env実行ファイルに実装した
native bridgeが、その子processへ固定したPodmanのglobal引数を渡す。これによりproviderと
直接のengine観測が同じ経路を使う。実行にはnativeの引数配列と既存のexecx process境界を
使用し、bridgeを生成shell scriptとして実装しない。endpoint解析とargv/envの正確な動作は、
空白や非ASCII文字を含むpathを含めてnativeの回帰検証で証明する必要がある。

## 共通configと実観測

Dockerの正規化済みJSONとpodman-composeの正規化済みYAMLを同じhost-policy modelへ
入力する。作用の前にpolicyを適用し、選択したservice/resourceの到達可能な依存関係閉包を
digestとともに記録する。共有する保存形式にはcanonical JSONを予定している。このbackend
契約を検証済みとするには、完全な表現をpodman-compose 1.6.0が受け付けることを証明する。
provider別のserializationへ変更する場合は、明示的なPlan上の判断と対応するtestが必要となる。

pod作成は無効にする。動作を変える`x-podman`拡張と、model化していないprovider固有の
resource型を拒否する。一般的な拡張から共通policyを迂回できてはならない。
providerのdetached起動を使っても、providerの`--wait`をreadinessの判断元にはしない。
appが観測に基づいて時間制限つきのreadiness確認を行う。

実container、network、volume、health、公開portには、構造化されたPodmanの直接inspectionを
使う。`io.podman.compose.*`などのPodman固有のproject/service識別情報と、
`io.agent-env.lease`、`io.agent-env.runtime`を併用する。Docker互換labelだけではPodmanの
所有確認として不十分である。network/volumeへのagent-env label付与の移植性を証明できない
場合は、より強いlabel保証を主張せず、文書化したproviderのproject labelと記録済みIDによる
証明を維持する。リスト順、`latest`、生成名だけでresourceを選択しない。

Docker専用のCompose flagに依存しない方法として、timestampつきのPodman直接logsで
container/serviceへの安定した帰属を得てもよい。動的mappingをendpointとして提供するのは、
必要なhost到達性の契約を確認してからとする。Podman Machineのmappingは、host到達性が
未確認なら利用不可のままとする。fakeのinspectionデータではMachineの転送動作は証明できない。

## Cleanupと証拠

down前に、所有を証明済みのcontainerへ接続したresourceを匿名volume IDも含めて観測する。
down後は実engineを再観測する。残存する匿名volumeを直接削除できるのは、過去の接続を
証明でき、現在のengine観測で外部・兄弟containerからの参照がないと確認できる場合のみ。
所有の曖昧さ、識別情報の不一致、観測不足があればquarantineと証拠を維持する。
global pruneで補償処理を行わない。

testでは、Podmanの2 leaseが独立したlifecycleを維持すること、DockerとPodmanが共存すること、
既定の接続を変更してもcleanup先が変わらないことを証明する。同じnamed/E2E fixtureを両方の
providerで実行する。native platform testでは解析、argv、path処理、engine固定を検証し、
実Linux rootless integrationではendpoint、兄弟leaseの存続、cleanupを証明する。
実機のMachine testは別枠で、環境がある場合に実施する。検証したバージョンと残る不足は
すべてExecPlanに記録する。podman-compose 1.6.0未満をrelease受け入れ環境として扱わない。
