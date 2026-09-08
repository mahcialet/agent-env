---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/product-specs/persistent-process-runtime.md
source_sha256: fd0fcbc195e5207bec312eb6391fc50ac56bfdae1d94085b3ee873575d8eeef6
---

# 常駐プロセスruntime

[English（翻訳元）](persistent-process-runtime.md)

`process` runtimeは、leaseに属するforegroundのnative host processを管理する。
固定したsource、argvの直接実行、専用の可変状態directory、fileへの出力、永続化した
native process treeの識別情報を使う。Composeやdaemonは不要である。
実装とnative受け入れの状況は[active ExecPlan](../exec-plans/active/persistent-process-runtime.ja.md)
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

`working_directory`は必須で、sourceからの相対pathを指定する。symlink解決後も選択した
worktree内に収まる必要がある。実行ファイルはsource相対pathかPATH上の単純なtool名とする。
実行ファイル名と作業directoryでは補間を使えない。Windowsの`.bat`/`.cmd` wrapperは
非対応である。command配列は暗黙のshellを介さず直接実行する。shell command型は設けない。

commandの引数と環境変数値で使える補間は、`${runtime_dir}`、`${lease_id}`、
`${port:name}`、`${env:NAME}`のみ。`runtime_dir`はruntime専用の可変状態directoryを
示す。未知の参照は検証時に拒否し、必要なhost環境変数がなければ起動前に失敗する。
認証情報を示す環境変数keyの値は、host環境変数への参照1つだけにする。literalの認証情報を
永続manifestへ保存してはならない。展開した認証情報も起動metadataへ保存しない。
applicationは専用の未加工stdout/stderr fileにもsecretを出力しないようにする必要がある。

名前付きportには`protocol: tcp`の明示が必須となる。固定host port、UDP、log解析による
port検出は非対応。割り当てたloopback portは起動前に予約する。予約はagent-env間の
重複割り当てを防ぐが、割り当てから起動までに外部processが同じportへbindすることは
防げない。対象commandは渡されたportを使いloopbackへbindする必要がある。
process実行はnetworkや悪意あるcodeのsandboxではない。

processのendpointは、宣言した名前付きportを指す`runtime_port`のみで指定する。
Compose用の`service`、`target`、`protocol`と`compose_services`は、nullや空でも拒否する。
process runtimeではCompose用の`provider`、`project_directory`、`files`とAndroid用の
`avd`をfieldの存在で拒否する。ほかのruntime型でprocess専用fieldを指定することも拒否する。
runtime名はpathとして安全で、大文字小文字を無視した重複がない必要がある。
新fieldを省略した既存Compose manifestのcanonical表現は変わらない。

HTTP readinessは同じcomponentの宣言済みendpointを`${endpoint:localName}`で参照できる。
数値の予約portへ展開するため、URLには`http://127.0.0.1:${endpoint:http}/health`と書ける。
host:port全体への展開ではない。command readinessの引数はこのendpoint参照と上記process引数の
参照も使える。実行ファイル名の補間は禁止のままとする。

runtime fileは`leases/<id>/process-runtimes/<runtime>/`配下の`state/`、`stdout.log`、
`stderr.log`、所有markerの`owner.json`、専用の`launch.json`、独立した起動前`redaction.json`
に保存する。CLI logsは上限付きで、host環境変更後も起動前に保存したsecret fingerprintを
使ってredactionする。起動後にredaction証拠が欠落・不正なら
出力を止める。未加工fileはapplication secretを含み得るため、receiptと未加工logは非公開の
入力として扱う。redactionは起動後の識別情報receiptの保存成功に依存しない。

## 寿命とcleanup

processの起動だけではREADYとしない。既存のHTTP/command readinessが成功する必要がある。
後続のCLIも記録したnative識別情報を観測し、logsはruntimeに帰属するfile出力を返す。
予期しない終了はleaseをdegradedにする。自動再起動は行わない。foregroundの起点processは
停止時まで存続する必要がある。自己daemon化、外部processの取り込み、対話stdin、PTY、
remote実行、service導入は対象外である。

destroyはnative識別情報を再確認して期限付きの停止を要求し、tree全体の不在を確認してから
port、可変状態、source worktreeを解放する。所有が不確実ならresourceを保持してquarantine
とする。Unixでは起点processの終了後、PIDが存在しなくてもtreeの所有が不確実なことがある。
Windowsではnative Job/guardianによる所有を使う。cross-build成功だけではnative実行の
証拠にならない。destroyの再実行とGCも同じ所有確認を使う。可変状態には機密性のあるprofileや
databaseが含まれ得るため、自動でartifactとして保持しない。

実行ファイルpath、source/hostの由来、読み取れるfileのdigestは起動証拠となるが、可変の
PATH toolの再現性を保証しない。Browser/CDPの機能はこの汎用lifecycleの上に置く将来の
利用側で扱う。[設計](../design-docs/persistent-process-runtime.ja.md)を参照する。
