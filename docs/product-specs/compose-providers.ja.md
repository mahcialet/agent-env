---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/product-specs/compose-providers.md
source_sha256: dbc5df54bb1e08db87b6d6b415632888a1972396208defb49f21456d58c93ea7
---

[English（翻訳元）](compose-providers.md)

# Compose provider

この文書は[進行中のExecPlan](../exec-plans/active/compose-provider-podman.ja.md)
で実装するproviderの契約を定める。実装と受け入れ検証は未完了であり、記載した
すべてのホスト構成で検証済みのサポートを提供しているという意味ではない。

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
| `podman-compose` | Podman client/engine、独立したpodman-compose | podman-compose 1.6.0以上。検証したclient/serverの正確なバージョンは受け入れ記録に残す |

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
観測したmappingを使う。Podman Machineのhost-loopback mappingは、agent-envホスト
からの到達性を示す証拠なしに利用可能と表示してはならない。非対応または未確認の
mappingは安全側で拒否する。nativeのfake testやcross-buildは、実機のMachine対応を
証明しない。

container、network、volumeの実観測では、provider固有の識別情報とagent-envの所有
証拠を併用して所有を確認する。生成名の一致だけでは不十分である。logsはtimestampと
service/containerへの帰属を保持する。destroyはprovider down後にresourceを再観測し、
兄弟leaseと無関係なresourceを保持する。

匿名volumeには明示的なcleanup証拠が必要となる。残存volumeを直接削除できるのは、
所有を証明済みのcontainerに接続されており、外部または兄弟containerが現在参照して
いない場合のみ。不確実ならquarantineとする。leaseのcleanupでPodmanのglobal prune
コマンドを使用しない。

## 対象範囲と受け入れ

podの作成は無効にする。動作を変える`x-podman`拡張は、共通policyを迂回させず拒否する。
任意のprovider実行ファイル、Docker Compose v1、Quadlet/Kubernetes、OCI保持、
悪意あるコードのsandbox化は対象外である。

受け入れには、既存の実Docker integrationの維持、podman-compose 1.6.0以上を使った
Linux rootless Podman、Podmanの2 lease同時実行、到達可能な動的endpoint、同じnamed/E2E
fixture、兄弟leaseの存続、cleanupの確認、DockerとPodmanの共存が必要となる。
Windows/macOS/Linuxのnative testでは選択、解析、path、argv、識別情報の固定を検証する。
実機のPodman Machine検証は、環境がある場合に別途記録する。正確なバージョン、コマンド、
結果、未解決事項はExecPlanを参照する。実装の責務境界は
[設計](../design-docs/compose-providers.ja.md)を参照する。
