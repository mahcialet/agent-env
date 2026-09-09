---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/product-specs/persistent-process-runtime.md
source_sha256: 71478d2c4c7765e0dd021a3409865e06ba8110e5fa013831cf153f6993d177d8
---

# 常駐プロセスruntime

[English（翻訳元）](persistent-process-runtime.md)

`process` ランタイムは、リースに属するフォアグラウンドのネイティブホストプロセスを管理する。
固定したソース、引数配列の直接実行、専用の可変状態ディレクトリ、ファイルへの出力、永続化した
OS が管理するプロセスツリーの識別情報を使う。Compose やデーモンは不要である。
このライフサイクルは実装済みであり、本仕様で常駐サービスの設定と安全な削除条件を確認できる。
実装とネイティブ受け入れの状況は[完了ExecPlan](../exec-plans/completed/persistent-process-runtime.ja.md)
に記録する。この文書は機能の契約を定める。

## Manifest

```yaml
runtimes:
  api:
    type: process
    source: app
    working_directory: .
    command: [./bin/server, '--listen=127.0.0.1:${port:http}', '--state=${runtime_dir}']
    ports:
      http: {protocol: tcp}
    env:
      API_TOKEN: ${env:API_TOKEN}
components:
  api:
    runtime: api
    endpoints:
      http: {runtime_port: http}
    readiness:
      - type: http
        url: http://127.0.0.1:${endpoint:http}/health
```

### 実行ファイルと作業ディレクトリ

`working_directory`は必須で、ソースからの相対パスを指定する。シンボリックリンク解決後も選択した
ワークツリー内に収まる必要がある。実行ファイルはソース相対パスかPATH上の単純なツール名とする。
実行ファイル名と作業ディレクトリでは補間を使えない。Windows の `.bat`/`.cmd` ラッパーは
非対応である。コマンド配列はシェルを暗黙に介在させず、直接実行する。シェル実行用のコマンド型は設けない。

### 引数と秘密情報

コマンドの引数と環境変数値で使える補間は、`${runtime_dir}`、`${lease_id}`、
`${port:name}`、`${env:NAME}`のみ。`runtime_dir`はランタイム専用の可変状態ディレクトリを
示す。未知の参照は検証時に拒否し、必要なホスト環境変数がなければ起動前に失敗する。
認証情報を示す環境変数キーの値は、ホスト環境変数への参照1つだけにする。リテラルの認証情報を
永続マニフェストへ保存してはならない。展開した認証情報も起動メタデータへ保存しない。
アプリケーションは、専用の未加工 stdout/stderr ファイルにも秘密情報を出力しないようにする必要がある。

### ポートとコンポーネントの接続先

名前付きポートには`protocol: tcp`の明示が必須となる。固定ホストポート、UDP、ログ解析による
ポート検出は非対応。割り当てたループバックポートは起動前に予約する。予約はagent-env間の
重複割り当てを防ぐが、割り当てから起動までに外部プロセスが同じポートを使用することは
防げない。対象コマンドは渡されたポートを使いループバックで待ち受ける必要がある。
プロセス実行はネットワークや悪意あるコードを隔離するサンドボックスではない。

プロセスの接続先は、宣言した名前付きポートを指す`runtime_port`のみで指定する。
Compose用の`service`、`target`、`protocol`と`compose_services`は、nullや空でも拒否する。
プロセスランタイムではCompose用の`provider`、`project_directory`、`files`とAndroid用の
`avd`をフィールドの存在で拒否する。ほかのランタイム型でプロセス専用フィールドを指定することも拒否する。
ランタイム名はパスとして安全で、大文字小文字を無視した重複がない必要がある。
新フィールドを省略した既存Compose マニフェストの正規化表現は変わらない。

### 起動確認で使う参照

HTTP 起動確認は同じコンポーネントの宣言済み接続先を`${endpoint:localName}`で参照できる。
数値の予約ポートへ展開するため、URLには`http://127.0.0.1:${endpoint:http}/health`と書ける。
host:port全体への展開ではない。コマンド起動確認の引数はこの接続先参照と上記プロセス引数の
参照も使える。実行ファイル名の補間は禁止のままとする。

## 専用ファイルと診断ログ

ランタイムファイルは`leases/<id>/process-runtimes/<runtime>/`配下の`state/`、`stdout.log`、
`stderr.log`、所有マーカーの`owner.json`、専用の`launch.json`、独立した起動前`redaction.json`
に保存する。CLI のログ表示は上限付きで、ホスト環境変更後も起動前に保存した秘密情報フィンガープリントを
使って伏せ字処理する。起動後に伏せ字処理証拠が欠落・不正なら
出力を止める。未加工ファイルにはアプリケーションの秘密情報が含まれる場合があるため、記録と未加工ログは非公開の
入力として扱う。伏せ字処理は起動後の識別情報記録の保存成功に依存しない。

## 寿命とcleanup

プロセスの起動だけではREADYとしない。既存のHTTP/command 起動確認が成功する必要がある。
後続の CLI も記録した OS の識別情報を観測し、`logs` はランタイムに帰属するファイル出力を返す。
予期しない終了はリースをdegradedにする。自動再起動は行わない。フォアグラウンドの起点プロセスは
停止時まで存続する必要がある。自己デーモン化、外部プロセスの取り込み、対話stdin、PTY、
リモート実行、サービス導入は対象外である。

### リソースを解放できる条件

destroyはネイティブ識別情報を再確認して期限付きの停止を要求し、プロセスツリー全体の不在を確認してから
ポート、可変状態、ソースワークツリーを解放する。所有が不確実ならリソースを保持して隔離
とする。Unixでは起点プロセスの終了後、PIDが存在しなくてもプロセスツリーの所有が不確実なことがある。
Windowsではネイティブ Job/guardianによる所有を使う。クロスビルドの成功だけではネイティブ実行の
証拠にならない。destroyの再実行とGCも同じ所有確認を使う。可変状態には機密性のあるプロファイルや
データベースが含まれ得るため、自動で成果物として保持しない。

## 証拠と関連機能

実行ファイルパス、source/hostの由来、読み取れるファイルのダイジェストは起動証拠となるが、可変の
PATH ツールの再現性を保証しない。[Browser/CDP](browser-cdp-automation.ja.md) は、この汎用ライフサイクルを利用する
独立した層として実装済みである。[設計](../design-docs/persistent-process-runtime.ja.md)を参照する。

監視と所有確認の仕組みは[設計](../design-docs/persistent-process-runtime.ja.md)、実装と各 OS での
検証記録は[完了済み ExecPlan](../exec-plans/completed/persistent-process-runtime.ja.md)を参照する。
