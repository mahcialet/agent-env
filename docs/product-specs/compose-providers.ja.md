---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/product-specs/compose-providers.md
source_sha256: cf4f21cbedb3e7a8ca8809649272510ad247b4e14cdc5b85a1c17b8781328f86
---

[English（翻訳元）](compose-providers.md)

# Compose provider

Compose ランタイムでは Docker Compose v2 と単独の podman-compose を選べる。
本仕様は、プロバイダーの選択、後続操作で使うエンジンの識別情報、リソースの削除条件を
定める。記載した範囲の実装は完了しているが、実際の Podman Machine による転送は
未検証である。検証環境と結果は末尾の受け入れ記録を参照する。

## 選択と前提条件

Compose ランタイムは次のようにプロバイダーを1つ選択できる。

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
明示した場合と同じランタイム動作となる。ほかに受け付ける値は`podman-compose`のみ。
未知の値、明示した空文字列・null、Compose以外のランタイムのプロバイダーフィールドは、
ランタイムへの作用より前にマニフェスト検証で拒否する。コンポーネントとスタックはプロバイダーに
依存しない。選択したプロバイダーをplanとリースの実行スナップショットに記録し、showとランタイム
診断に表示する。

実行ファイルの有無による自動切り替えはない。Podmanが利用できない場合にPodman
ランタイムをDocker ランタイムへ変更しない。プロバイダー情報を持たない既存スナップショットはDocker
として扱う。その後のinspect、logs、reconcile、destroyは、対象マニフェストが変更されても
保存済みの選択を使う。

| Provider | 外部の前提条件 | バージョン契約 |
| --- | --- | --- |
| `docker-compose` | Docker クライアント、Compose v2、到達可能な選択済みエンジン | 既存のDocker契約 |
| `podman-compose` | Podman 5.x client/engine、独立したpodman-compose | podman-compose >=1.6.0,<2.0.0。検証したclient/serverの正確なバージョンは受け入れ記録に残す |

Podman、podman-compose、およびそれらの導入に必要な依存関係はホスト側の前提条件で
あり、同梱する配布物やスタンドアロン CLI の基本機能の依存関係ではない。podman-compose 1.3を含む
古い導入済みバージョンはこの契約を満たさない。`podman compose` ラッパーは別プロバイダー
として扱わない。Doctorは不足する前提条件を報告し、ツールの導入やエンジンの切替えは
行わない。version/helpと無関係な機能は、引き続きPodmanやPythonなしで動作する。

## 識別情報、endpoint、cleanup

### 記録したエンジンの識別

各リースは、記録したエンジン上の明示的なCompose projectを所有する。後続操作は、
既定の接続が変わっても同じエンジンを使う。再観測で記録済みの同一性を確認できない
場合は破壊的クリーンアップを停止し、隔離の証拠を保持する。フィンガープリントは接続先、
ホストの OS とアーキテクチャ、ストレージルートを使って対応範囲のローカル・リモート構成を区別する。
ただし、同じ構成のままエンジンがその場で初期化されていないことまでは証明できない。
リソースの所有確認は引き続き必要となる。

### 動的な接続先と到達確認

両プロバイダーともリソース作成前に既存のホストポリシーを適用する。選択したサービスの
依存関係閉包だけを起動する。固定ホストポートは引き続き拒否し、動的接続先には
観測した対応付けを使う。共通スナップショットには公開ポートの`0`を保持し、Podmanへ渡すプロバイダー専用のコピー
ではそのフィールドだけを省略して`host_ip`を保持する。これによりループバックへのバインドを広げずに
エンジンによるポート割り当てを要求する。記録済みリモート URLを使う場合、TCP 対応付けには
agent-envホストからの接続成功も必要となる。TCP接続に失敗した対応付けは返さず、
起動確認をfalseにする。UDP 対応付けはエンジンの観測情報として保持し、TCPによる検査や
UDP アプリケーションとの通信成功は推定しない。UDP 応答の検証が必要なアプリケーションは
自身の起動確認検査を定義する。ネイティブ環境での代替アダプターテストやクロスビルドは、実機のMachineの
転送動作を証明しない。

### リソースの一覧と所有確認

Podman の一覧表示は、記録したエンジンからラベル付きのコンテナー、ネットワーク、ボリュームを
列挙する。記録済みCompose実行ファイルが削除・移動された場合も含め、
podman-composeの存在や実行を必要としない。

コンテナー、ネットワーク、ボリュームの実観測では、プロバイダー固有の識別情報とagent-envの所有
証拠を併用して所有を確認する。生成名の一致だけでは不十分である。`logs` は時刻と、サービスやコンテナーへの帰属を保持する。destroyはプロバイダー down後にリソースを再観測し、
他のリースと無関係なリソースを保持する。

### 匿名ボリュームの削除

匿名ボリュームには明示的なクリーンアップ証拠が必要となる。アプリケーション層は down 前に、正確なネイティブボリューム
フィンガープリントと証明済みのコンテナー接続を`Runtime.cleanup_evidence`へ保存する。
復旧時はコンテナー消失後もその証拠を保持し、現在の識別情報と参照を再確認する。
外部または兄弟コンテナーが現在参照していない場合のみ、残存ボリュームを削除できる。
不確実なら隔離とする。リースのクリーンアップでPodman のホスト全体に対する一括削除コマンドを使用しない。

## 対象範囲と受け入れ

### 対応する設定

pod作成は無効にし、`x-podman*`拡張は階層を問わず再帰的に拒否する。
mount型は`bind`、`volume`、`tmpfs`に限定し、`glob`などモデル化していない型は明示的に拒否する。
`network_mode`は省略、空文字列、`bridge`、`none`に対応する。`host`は共通ポリシーで拒否し、
`ns:`、`pasta`、`slirp4netns`など、ほかの明示モードは非対応となる。

Podmanでは、最初のCompose ファイルの親ディレクトリを`project_directory`と一致させる必要がある。
`env_file`、config と secret のファイル参照は、シンボリックリンク解決後もそのディレクトリ内に収まる通常ファイルで
なければならず、正規化後は絶対パスを使う。正規化設定の環境値は明示的な値として固定する。
未解決の裸のキーやnull による環境変数の暗黙継承は拒否する。これらの制限は、ホストへのアクセスを
黙って広げたり、後の環境変数値を読み直したりせず、作用の前にエラーとする。

任意のプロバイダー実行ファイル、Docker Compose v1、Quadlet/Kubernetes、OCI保持、
悪意あるコードのサンドボックス化は対象外である。

### 受け入れの検証記録

Podman 5.4.2 と podman-compose 1.6.0 による Linux の rootless 実行では、Docker との
共存を含めて受け入れ検証に成功した。Windows/macOS/Linux のネイティブ CI も
4a5de3d（run 34216579481）で成功した。実際の Podman Machine 環境は利用できなかった。
コマンドと結果は[完了済み ExecPlan](../exec-plans/completed/compose-provider-podman.ja.md)に残している。

受け入れには、既存の実 Docker 統合検証の維持、Podman 5.xとpodman-compose
>=1.6.0,<2.0.0を使ったLinux rootless Podman、Podmanの2 リース同時実行、到達可能な動的接続先、同じ名前付きテストと E2E の
検証用フィクスチャ、兄弟リースの存続、クリーンアップの確認、DockerとPodmanの共存が必要となる。
Windows/macOS/Linuxのネイティブテストでは選択、解析、パス、引数配列、識別情報の固定を検証する。
実機のPodman Machine検証は、環境がある場合に別途記録する。正確なバージョン、コマンド、
結果、未解決事項はExecPlanを参照する。実装の責務境界は
[設計](../design-docs/compose-providers.ja.md)を参照する。
