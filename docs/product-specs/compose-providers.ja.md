---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/product-specs/compose-providers.md
source_sha256: 9e495342a6459ab091c8d688abc6b644eec4b5651d071f73e099129eab6ed694
---

[English（翻訳元）](compose-providers.md)

# Compose provider

この文書は[完了済みExecPlan](../exec-plans/completed/compose-provider-podman.ja.md)
で実装したproviderの契約を定める。Podman 5.4.2とpodman-compose 1.6.0で、Docker共存を
含む実Linux rootless受け入れが成功した。Windows/macOS/Linuxのnative provider CIは4a5de3d（run 34216579481）で成功であり、実機の
Podman Machine環境はない。

## 選択と前提条件

Compose runtimeは次のようにproviderを1つ選択できる。

```yaml
runtimes:
  backend:
    type: compose
    provider: podman-compose
    source: backend
    project_directory: .
    files: [compose.yaml]
```

`provider`の省略時は`docker-compose`を選択し、`provider: docker-compose`を
明示した場合と同じruntime動作となる。ほかに受け付ける値は`podman-compose`のみ。
未知の値、明示した空文字列・null、Compose以外のruntimeのproviderフィールドは、
runtimeへの作用より前にmanifest検証で拒否する。componentとstackはproviderに
依存しない。選択したproviderをplanとleaseの実行snapshotに記録し、showとruntime
診断に表示する。

実行ファイルの有無による自動fallbackはない。Podmanが利用できない場合にPodman
runtimeをDocker runtimeへ変更しない。provider情報を持たない既存snapshotはDocker
として扱う。その後のinspect、logs、reconcile、destroyは、対象manifestが変更されても
保存済みの選択を使う。

| Provider | 外部の前提条件 | バージョン契約 |
| --- | --- | --- |
| `docker-compose` | Docker client、Compose v2、到達可能な選択済みengine | 既存のDocker契約 |
| `podman-compose` | Podman 5.x client/engine、独立したpodman-compose | podman-compose >=1.6.0,<2.0.0。検証したclient/serverの正確なバージョンは受け入れ記録に残す |

Podman、podman-compose、およびそれらの導入に必要な依存関係はホスト側の前提条件で
あり、同梱assetやstandalone coreの依存関係ではない。podman-compose 1.3を含む
古い導入済みバージョンはこの契約を満たさない。`podman compose` wrapperは別provider
として扱わない。Doctorは不足する前提条件を報告し、ツールの導入やengineの切替えは
行わない。version/helpと無関係な機能は、引き続きPodmanやPythonなしで動作する。

## 識別情報、endpoint、cleanup

各leaseは、記録したengine上の明示的なCompose projectを所有する。後続操作は、
既定の接続が変わっても同じengineを使う。再観測で記録済みの同一性を確認できない
場合は破壊的cleanupを停止し、quarantineの証拠を保持する。fingerprintはendpoint、
ホストOS/architecture、storage rootを使って対応範囲のlocal/remote構成を区別する。
ただし、同じ構成のままengineがその場で初期化されていないことまでは証明できない。
resourceの所有確認は引き続き必要となる。

両providerともresource作成前に既存のhost policyを適用する。選択したserviceの
依存関係閉包だけを起動する。固定host portは引き続き拒否し、動的endpointには
観測したmappingを使う。共通snapshotには公開portの`0`を保持し、Podmanへ渡すprivate copy
ではそのfieldだけを省略して`host_ip`を保持する。これによりloopback bindingを広げずに
engineによるport割り当てを要求する。記録済みremote URLを使う場合、TCP mappingには
agent-envホストからの接続成功も必要となる。TCP接続に失敗したmappingは返さず、
readinessをfalseにする。UDP mappingはengineの観測情報として保持し、TCPによる検査や
UDP applicationとの通信成功は推定しない。UDP応答の検証が必要なapplicationは
自身のreadiness検査を定義する。nativeのfake testやcross-buildは、実機のMachineの
転送動作を証明しない。

Podman inventoryは記録済みnative engineからlabel付きcontainer、network、volumeを
列挙する。記録済みCompose実行ファイルが削除・移動された場合も含め、
podman-composeの存在や実行を必要としない。

container、network、volumeの実観測では、provider固有の識別情報とagent-envの所有
証拠を併用して所有を確認する。生成名の一致だけでは不十分である。logsはtimestampと
service/containerへの帰属を保持する。destroyはprovider down後にresourceを再観測し、
兄弟leaseと無関係なresourceを保持する。

匿名volumeには明示的なcleanup証拠が必要となる。appはdown前に、正確なnative volume
fingerprintと証明済みのcontainer接続を`Runtime.cleanup_evidence`へ保存する。
復旧時はcontainer消失後もその証拠を保持し、現在の識別情報と参照を再確認する。
外部または兄弟containerが現在参照していない場合のみ、残存volumeを削除できる。
不確実ならquarantineとする。leaseのcleanupでPodmanのglobal pruneコマンドを使用しない。

## 対象範囲と受け入れ

pod作成は無効にし、`x-podman*`拡張は階層を問わず再帰的に拒否する。
mount型は`bind`、`volume`、`tmpfs`に限定し、`glob`など未model化の型は明示的に拒否する。
`network_mode`は省略、空文字列、`bridge`、`none`に対応する。`host`は共通policyで拒否し、
`ns:`、`pasta`、`slirp4netns`など、ほかの明示的modeは非対応となる。

Podmanでは、最初のCompose fileの親directoryを`project_directory`と一致させる必要がある。
`env_file`、config、secretのfile参照は、symlink解決後もそのdirectory内に収まる通常fileで
なければならず、正規化後は絶対pathを使う。正規化設定の環境値は明示的な値として固定する。
未解決の裸のkeyやnullによるpass-throughは拒否する。これらの制限は、ホストへのアクセスを
黙って広げたり、後の環境変数値を読み直したりせず、作用の前にエラーとする。

任意のprovider実行ファイル、Docker Compose v1、Quadlet/Kubernetes、OCI保持、
悪意あるコードのsandbox化は対象外である。

受け入れには、既存の実Docker integrationの維持、Podman 5.xとpodman-compose
>=1.6.0,<2.0.0を使ったLinux rootless Podman、Podmanの2 lease同時実行、到達可能な動的endpoint、同じnamed/E2E
fixture、兄弟leaseの存続、cleanupの確認、DockerとPodmanの共存が必要となる。
Windows/macOS/Linuxのnative testでは選択、解析、path、argv、識別情報の固定を検証する。
実機のPodman Machine検証は、環境がある場合に別途記録する。正確なバージョン、コマンド、
結果、未解決事項はExecPlanを参照する。実装の責務境界は
[設計](../design-docs/compose-providers.ja.md)を参照する。
