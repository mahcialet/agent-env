---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/design-docs/compose-providers.md
source_sha256: 075b95f5b78c92c1dd0c6f3503cbff369ada32ffcb555ccebe24ae3fc1b81423
---

[English（翻訳元）](compose-providers.md)

# Compose providerの設計

[製品契約](../product-specs/compose-providers.ja.md)が選択と安全性の要件を定める。
[完了済みExecPlan](../exec-plans/completed/compose-provider-podman.ja.md)に、
以下の構成と正規化についての実装判断と受け入れ証拠を記録している。
実機のMachine検証環境は利用できていない。

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
digestとともにcanonical JSONで記録する。podman-composeが変更操作のsnapshotを再解析
する前に、private copyのliteralなdollar記号をescapeし、固定値を再展開させない。
同じprivate copyで、記録portがzeroの場合だけ`published`を省略し、`host_ip`は保持する。
Podmanは指定したloopback制限のまま動的portを割り当てる。canonical設定とdigestは変えない。
受け付けるホスト構成はPodman 5.xとpodman-compose >=1.6.0,<2.0.0である。
5.4.2 / 1.6.0でDocker共存を含む実Linux rootless受け入れが成功した。以前の1.3 providerは
この条件で拒否した。Windows/macOS/Linuxのnative provider CIは4a5de3d（run 34216579481）で成功で、実機のMachine環境はない。

最初のCompose fileの親を`project_directory`と一致させ、Podmanの基準directoryの
意味の違いを避ける。`env_file`とconfig/secretのfile参照は、symlink解決後もそのdirectory
内に収まる通常fileへ解決し、絶対pathとする。nullや裸のkeyによる環境pass-throughは拒否し、
変わり得る実行時環境から値を得ず、正規化値を明示する。projectの`.env`では、routingや動作を
制御する予約済みの`PODMAN_*`、`CONTAINER_*`、`AGENT_ENV_PODMAN_*`、`COMPOSE_*` keyを
拒否する。

pod作成は無効にする。resource levelや深い階層の拡張も含め、`x-podman*`を再帰的に拒否する。
model化するmount型は`bind`、`volume`、`tmpfs`のみであり、ホスト参照を展開するPodmanの
`glob`など、ほかの型は拒否する。`network_mode`は省略・空文字列、`bridge`、`none`を
受け付ける。`host`は共通policyへ渡して拒否し、namespace path、`pasta`、`slirp4netns`など
ほかのmodeはこの段階で拒否する。
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

down前に`PrepareCleanup`が、所有を証明済みのcontainerへ接続したresourceをnative匿名
volumeも含めて観測する。appは返された`Runtime.cleanup_evidence`をDown呼出し前に
永続化する。この証拠には正確なvolume fingerprint、lease/runtimeの識別情報、証明済みの
container接続が含まれる。元のcontainerが消失しても復旧時に残存resourceを観測でき、
volume名の一致だけで削除の根拠を再構成しない。down後は実engineを再観測し、保持した
fingerprintと照合する。残存する匿名volumeを直接削除できるのは、過去の接続を証明でき、
現在のengine観測で外部・兄弟containerからの参照がないと確認できる場合のみ。
Composeで宣言した匿名mountは決定的なproject所有のnamed volumeへ正規化する。
imageで宣言したnative匿名volumeには、この別の証拠が必要となる。
所有の曖昧さ、識別情報の不一致、観測不足があればquarantineと証拠を維持する。
global pruneで補償処理を行わない。

testでは、Podmanの2 leaseが独立したlifecycleを維持すること、DockerとPodmanが共存すること、
既定の接続を変更してもcleanup先が変わらないことを証明する。同じnamed/E2E fixtureを両方の
providerで実行する。native platform testでは解析、argv、path処理、engine固定を検証し、
実Linux rootless integrationではendpoint、兄弟leaseの存続、cleanupを証明する。
実機のMachine testは別枠で、環境がある場合に実施する。検証したバージョンと残る不足は
すべてExecPlanに記録する。Podman 5.xおよびpodman-compose >=1.6.0,<2.0.0以外を
release受け入れ環境として扱わない。
