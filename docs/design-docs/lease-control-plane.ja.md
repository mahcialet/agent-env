---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/design-docs/lease-control-plane.md
source_sha256: 5d301ed43612e744cda19b64835073f6cf0558667e28a599b55a9ec0a219be5b
---

[English（翻訳元）](lease-control-plane.md)

# Lease コントロールプレーン

この文書では、local lease の記録、アダプターの責務、作成・削除に伴う復旧の仕組みを説明します。
現在の公開仕様は[製品仕様の索引](../product-specs/index.ja.md)から参照できます。
[MVP 仕様](../product-specs/agent-env-mvp.ja.md)は初期の範囲を記録しています。
[複数 host の調整](multi-host-control-plane.ja.md)は、この local の規則を維持したまま、
remote の割り当てと操作配送の管理を追加します。

## Environment

アプリケーションを動かす実際のリソースです。worktree、Compose プロジェクト、コンテナ、network、volume、
生成設定、ポート、ログに加え、選択した Emulator や常駐プロセスを含みます。
Browser 操作は所有を確認した常駐プロセスを使い、別のライフサイクル管理主体は追加しません。

## Lease

環境の所有者、環境が表す不変の source set、存在する目的、割り当てを維持できる期間、リソースの状態を記述するコントロールプレーンの記録です。

lease は単なる PID でも、単なる Compose プロジェクト名でもありません。

## Source set

lease が使うリポジトリと解決済み Git commit の不変な組です。

例:

```text
mobile-app@aaaa1111
backend-api@bbbb2222
shared-schema@cccc3333
```

要求された ref も記録する必要がありますが、ランタイムの識別は解決済み commit に基づきます。

## Component

`api`、`dashboard`、`database`、`mobile` など、システムの論理的な構成要素です。コンポーネントは依存関係グラフを形成し、1 つ以上のランタイム固有リソースに対応できます。

## Stack

明示的なルートコンポーネントからなる名前付きの起動グループです。`agent-env` は、そのルートと、ルートが直接・間接に依存するすべてのコンポーネントを選びます。

例:

```text
api        -> api
Dashboard  -> dashboard + api
full       -> mobile + dashboard + api
```

manifest のキーと CLI の用語には `stack` を使います。MVP で stack 間の継承は使いません。

## Profile

`profile` は、*どの構成要素*を起動するかではなく、特定のホストや検証モードで stack を*どのように*実現するかのために予約します。例えば将来の profile は、Android API level 35 と 36、rootless Docker と Docker Desktop、ローカル基盤と外部提供の基盤を選択できます。

MVP は起動グループに `stack` を使い、profile overlay は実装しません。これにより、コンポーネント選択とランタイム設定を 1 つの用語に重ねることを避けます。

## Capability

解決済み環境が提供する機能を表し、`api`、`web-ui`、`logs`、`browser-e2e`、`android-ui` などを指します。
local のモデルでは環境の説明に使い、capability による stack の自動選択は将来の拡張です。
remote worker の capability は、複数 host の設計における配置判定に使う別の概念です。

## Scenario

capability を要求し、テストや観測を実行する、繰り返し可能な検証ワークフローです。用語の予約を除き scenario は MVP の範囲外で、初期段階では named test で十分です。

---

## 永続化と lifecycle

以下は domain の責務を表す概念上の名前です。現在の Go の型宣言の一覧ではありません。
実装された型は [domain](../../internal/domain/lease.go)を参照してください。

```text
Lease
LeaseState / DesiredState / ObservedState
LeaseOwner
SourceSpec
ResolvedSource
SourceSet
RuntimeSpec
Component
ResolvedComponentGraph
Stack
Capability
Resource
ResourceKind
CommandRun
TestRun
Artifact
Event
```

database row struct、YAML struct、domain struct を 1 つの共有型に統合してはいけません。解析、検証、domain の動作、永続化の境界を明示します。

## ランタイムアダプターの契約

この概念上の interface は、検証、計画、作用、観測、削除の責務を分けます。
[実装された app interface](../../internal/app/lifecycle.go) と各ランタイムの provider がその責務を担います。
以下は現在の API 宣言そのものではありません。

```go
type Runtime interface {
    Type() string
    Validate(ctx context.Context, req ValidateRequest) ([]Diagnostic, error)
    Plan(ctx context.Context, req PlanRequest) (RuntimePlan, error)
    Create(ctx context.Context, req CreateRequest) ([]Resource, error)
    Inspect(ctx context.Context, req InspectRequest) (ObservedRuntime, error)
    Collect(ctx context.Context, req CollectRequest) ([]Artifact, error)
    Destroy(ctx context.Context, req DestroyRequest) error
}
```

すべてのアダプターに、PID で制御する長時間稼働プロセスであることを要求してはいけません。Compose と Android はそれぞれ安定した外部識別子を持ちます。

## Source アダプターの契約

以下の例は、source の解決、展開、検査、削除の責務を分けたものです。
現在の interface は [app/plan.go](../../internal/app/plan.go)を参照してください。

```go
type SourceProvider interface {
    Resolve(ctx context.Context, spec SourceSpec, requestedRef string) (ResolvedSource, error)
    Materialize(ctx context.Context, lease Lease, source ResolvedSource) (WorktreeResource, error)
    Inspect(ctx context.Context, resource WorktreeResource) (ObservedSource, error)
    Remove(ctx context.Context, resource WorktreeResource, force bool) error
}
```

Git 実装は、ライブラリで Git object/ref の動作を再実装するのではなく、インストール済み Git CLI を外部コマンドとして実行します。

---

## `agent-env plan`

`plan` は Git、ランタイムのリソース、SQLite の lease 状態を変更してはいけません。

次を実施します。

1. `.agent-env.yaml` を見つけて解析する
2. schema と参照を検証する
3. source リポジトリを特定する
4. worktree を作成せずに要求された ref を commit に解決する
5. stack のルートと、ルートが直接・間接に依存するすべてのコンポーネントを選ぶ
6. 必要なランタイム操作を特定する
7. 可能な範囲でホスト policy と前提条件を検証する
8. 決定的な plan と診断情報を表示する

人間向け出力の例:

```text
Repository: C:\src\control-repo
Stack:      dashboard

Sources:
  backend   requested=refs/pull/3/head   resolved=abc1234...

Components:
  1. api
  2. dashboard

Runtime:
  backend   compose
  files:    infra/compose.yaml, infra/compose.agent.yaml
  services: db, api, dashboard

Warnings:
  none
```

## `agent-env create`

概念上の saga:

```text
plan と完全な不変 source set を解決
  -> lease ID とリソース名を予約
  -> 要求された lease、source、イベントを永続化
  -> 全 worktree を作成
  -> ランタイム plan を展開して検証
  -> 選択したランタイムのリソースを作成/起動
  -> readiness を確認
  -> 観測リソースと証拠を永続化
  -> ready に変更
```

失敗時は逆順で補償します。補償が不完全なら、lease を quarantined として保持します。

必要な全コンポーネントが readiness に合格するまで、環境を `ready` として公開してはいけません。

## `agent-env destroy`

概念上の saga:

```text
releasing に変更
  -> 新しいコマンドを禁止
  -> 最終ログ/設定を収集
  -> 所有するランタイムのリソースを停止/down
  -> worktree の追跡対象変更を検査
  -> 安全な worktree を削除
  -> policy に従い artifact を保持
  -> released に変更
```

---

## 状態パスと所有権

### 保存先

AGENT_ENV_HOME は OS の既定値を上書きします。Linux は XDG_STATE_HOME/agent-env または ~/.local/state/agent-env、macOS は ~/Library/Application Support/agent-env、Windows は LOCALAPPDATA/agent-env を使います。

state.db と worktrees、repositories、leases、artifacts、logs、generated、locks はこの状態ルート以下に置きます。

独立して破棄できる cache は OS の cache 位置を使います。ローカル SQLite が必須で、NFS/SMB 上の状態 database は未対応です。

### 状態と所有者の記録

desired state は active、stopped、released です。observed state は allocating、starting、ready、degraded、stopped、failed、releasing、released、quarantined、unknown を区別します。割り当て/起動失敗は failed、不在/不健全なリソースは degraded、危険/不完全な解放は quarantined になります。timestamp は UTC/RFC3339 で永続化し、要求 stack、解決済みコンポーネント、manifest と source-set の digest、所有権、作成、heartbeat、期限を保存します。

lease の所有者を説明する情報は参考情報であり、認可には使いません。明示した --owner が AGENT_ENV_OWNER を上書きします。指定がなければ PID だけでなく生成した一意 token を使い、説明的なローカル所有者情報を導出します。lease コマンドは heartbeat を更新します。renew は期限を変更し、組み込みの host policy が既定 TTL を 4 時間に定めます。利用者が編集する host policy ファイルは未実装です。

### 永続化と source の識別

正規化した repositories、leases、lease_sources、lease_components、resources、events、command_runs、artifacts を永続化し、外部キーと owner/state/expiration/repository/resource 検索用 index を持たせます。一意なリソース名はトランザクションで予約します。YAML/domain/database のモデルは分けます。foreign_keys、少なくとも 5000 ミリ秒の busy_timeout、検証済み WAL を有効にします。秘密情報を含む任意の環境 map を serialize してはいけません。

alias/リポジトリ識別情報/解決済み commit をソートした組が source-set digest を決めます。各 source の正確な要求 ref、解決済み commit、checkout mode、書き込み policy、timestamp を materialize 前に保存します。review worktree は detached で、生成出力を書き込み可能です。ファイルシステム権限を review policy とみなすのではなく、削除前に staged/unstaged の追跡対象変更を検出します。

### Readiness の中断と削除の制御

command readiness は実行前に running の command 行を永続化します。プロセス群の終了確認と証拠の永続化が完了した場合だけ terminal 行に進めます。終了または出力が未確認なら再試行せず、作成を隔離し running 行を残して、後続の destroy/GC でも source を保持します。終了確認済みの通常 probe 失敗は再試行できます。永続化された中断要求では次の試行を開始せず readiness を停止します。
