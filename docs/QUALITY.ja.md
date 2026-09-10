---
status: active
owner: maintainers
last_verified: 2026-09-10
translation_of: docs/QUALITY.md
source_sha256: 830b054dfa0e556c5c699869c627e6f472701cbc52034875e76bb6c8a65dbd46
---

# 品質と検証

[英語版（翻訳元）](QUALITY.md)

通常の検査にはリポジトリハーネスを使い、変更する動作に応じて実runtimeのfixtureも実行します。
以下のコマンドは検証方法です。実行記録が証明するのは、テストしたrevisionと環境での結果に限ります。
前提条件の不足やクロスビルド成功を、ネイティブ実行の受け入れ完了として扱いません。

## 正式な検査

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go run ./tools/repoctl test-integration
```

`doctor`はGo、gofmt、Gitの場所を確認します。Dockerが必要なのは明示的な統合コマンドだけです。
内部の各コマンドも直接実行できます。ハーネスにBash、Make、PowerShellは不要です。

`check`は実行内容を示しながら、整形確認、`go test ./...`、`go vet ./...`、文書検証、
生成物のずれ検出、アーキテクチャ検査を実行します。

| ハーネスのコマンド | 範囲 |
| --- | --- |
| `test-unit` | 通常のGoテスト。Docker daemonは不要 |
| `test-integration` | Docker daemon／Composeを確認し、明示的な統合テストのopt-in付きでtag指定のテストをキャッシュなしで実行 |
| `docs-check` | メタデータ、リンク／見出し、ローカル索引、AGENTSの長さ／パス、active planの必須構成、英語／日本語の対訳の欠落・更新漏れ |
| `generate` | 埋め込まれたSQL migrationを入力にDB文書を再生成 |
| `generated-check` | 生成schemaとmigrationが異なれば失敗 |
| `arch-check` | 修正方法を示す診断により、文書化されたimportの境界を強制 |

schema文書の生成元は、手書きの説明ではなく[migrations](../migrations/001_initial.sql)配下の番号付きSQL群である。安定した`AGENTENV-*`診断により、違反した不変条件と修正の方向を特定する。不正fixtureでは、リンク、索引、メタデータ、計画セクション、ソース整形、schema生成、import境界、欠落／孤立した翻訳、誤ったsourceメタデータ、古いhash、言語別リンク／索引を意図的に壊して検証する。

## 製品のテスト

unit／adapterテストは、厳密なmanifest解析、決定的なcomponent／source識別、状態遷移、TTLとGC適格性、正規化されたCompose方針、パスの包含、command argv、ストリームの伏字化、証拠保存の失敗経路、ライフサイクルの補償処理を対象とする。fake runnerはシェル文字列を組み立てずに、実行ファイル、argv、作業ディレクトリ、環境制御、キャンセル、出力処理、エラー変換を検査する。

SQLiteテストは実際の一時DBを使い、Unicodeパス、再オープン／複数接続、migrationの冪等性、外部キー、容量／project予約の原子性、正規化された行のrollback、永続的なlock動作を確認する。プロセステストはネイティブargvと子プロセスのキャンセルを実行する。OS固有の実装を検証するには、そのプラットフォームでのネイティブ実行が必要である。

## 実Dockerのfixture

[統合テスト群](../internal/cli/integration_test.go)は、独立した一時Gitリポジトリとstate homeを作り、[小さなCompose fixture](../testdata/compose/compose.yaml)を使う。テストには`integration` build tagと明示的なopt-inの両方が必要であり、ハーネスが自動で設定する。通常のunitテストはDocker containerを起動しない。

同時に動くAPI／Dashboard lease、選択された依存closure、sourceにportを宣言せずmanifestから生成する動的loopback HTTP、バージョン付き環境descriptor、component単位の実行中／保存済みログ、固有のproject／worktree識別、兄弟leaseのcontainer／network／volume IDの維持を確認する。

また、未選択の外部volumeが清掃後も残ること、手動削除したprojectがdegradedになること、複数リポジトリがsource ref overrideに従うことを確認する。

名前付きテストが伏字化されたstdout／stderr／artifactと非ゼロ終了状態を保存すること、readiness失敗が実リソースをrollbackすること、変更済みの追跡対象worktreeはdiff証拠を伴う明示的なforceまでGCで削除されないことも検証する。

### Dockerの実行記録

最後のローカルLinux CLI統合テストは109.95秒で成功し、生成endpoint、診断descriptor、component単位の実行中／アーカイブログを確認した。これはそのリビジョンでの実Docker動作の証拠であり、後続の変更や全ネイティブプラットフォームの完了を示すものではない。fixtureの全リソースは固有の追跡可能な識別子とlease単位の清掃を使い、一般的なDocker pruneは実行しない。

## 実Podmanのfixture

プラットフォームのnativeな環境設定で`AGENT_ENV_PODMAN_INTEGRATION=1`を設定し、
次を実行する。

```text
go test -tags=integration ./internal/cli -run TestPodmanIntegration -count=1 -v
```

`AGENT_ENV_PODMAN_DOCKER_COEXISTENCE=1`も設定すると、Docker共存と、両engineでの
同じnamed test fixtureを要求する。suiteは実CLIをビルド・実行してnativeのPodman
子process bridgeを検証する。Linux rootless Podman 5.xとpodman-compose
>=1.6.0,<2.0.0が必要であり、opt-in時に前提条件が不足・非対応ならskipせず失敗する。
通常のtestはどちらのengineも起動しない。fixtureは並行lease、HTTP endpoint、
伏字化したnamed-test証拠、logs、兄弟・外部resourceの存続、残存resourceのcleanupを
検査する。engineへの変更はlease単位で行い、所有を確認したcleanupを使う。
global pruneは実行しない。

### Podmanの実行記録と未検証の範囲

2026-09-08に、両opt-inを有効にした`TestPodmanIntegrationConcurrentLeasesAndEvidence`が
110.13秒で成功した。Linux rootless Podman 5.4.2、podman-compose 1.6.0とDockerを併用した。
両Podman leaseが選択したservice閉包でreadyとなり、HTTP endpointに到達できた。
component logs、named pass/fail test、伏字化、artifact保持を確認した。image宣言のvolumeは
nativeの匿名かつlabelなしであり、destroy後の不在を確認した。destroyは兄弟lease、外部volume、
稼働中のDocker leaseを保持した。Dockerでも同じnamed testが期待どおり成功・失敗した。
Windows/macOS/Linuxのnative provider CIは4a5de3d（run 34216579481）で成功した。実機のMachine環境はない。正確な証拠は
[provider plan](exec-plans/completed/compose-provider-podman.ja.md)に記録する。

4a5de3dのrun [34216579481](https://github.com/mahcialet/agent-env/actions/runs/34216579481)
では、Go 1.26/1.27のWindows/macOS/Linux native job全6件、CGO無効cross-build全5件、
Linuxのrace／Docker integration jobが成功し、全12 jobが成功した。
前述のlocal DockerおよびPodman/Docker共存runも成功した。

## CIと完了の証拠

CIはGo 1.26.xと1.27.xを使い、Windows、macOS、LinuxでハーネスとCLIビルドをネイティブ実行する。Linuxでは`go test -race ./...`と明示的なDocker統合テストも実行する。別のクロスビルドjobで`CGO_ENABLED=0`の5対象を確認する。

### MVPの過去の検証記録

[完了済み実装計画](exec-plans/completed/agent-env-mvp.md)には33件すべての受け入れ条件と、解決した独立レビューの指摘を記録している。c641286のCI 34124194139では、OS／Goのネイティブ検査6件、CGOを無効にしたビルド5件、全raceテストと実Docker統合テストを行うLinux jobの計12件が成功した。ローカルのGo 1.26.8／1.27.1での検査と実Docker統合テストも成功した。hosted macOS／WindowsでのDocker統合テストは実施済みとはしていない。

## 翻訳の検証

永続文書は英語と日本語の`.ja.md`を対にし、内容が食い違う場合は英語を優先する。[言語の方針](design-docs/bilingual-documentation.ja.md)でメタデータと正確なパス単位の例外を定義する。`docs-check`は、英語ファイル全体のCRLFをLFへ正規化したSHA-256と、翻訳のsource hashを比較する。そのためWindowsのcheckoutでもLinux／macOSと一致する。検査は読み取り専用であり、翻訳メタデータを自動更新しない。hashの一致は、どの原文リビジョンを確認したかを示すだけで、翻訳の正確さは証明しない。hashを更新する前に実際の日本語文を見直す。

文書の構成を実質的に変更する場合は、言語の方針に従い、英語と日本語をそれぞれ独立にレビューした後、
意味の一致を照合します。指摘はactive ExecPlanに記録します。機械的な検査に通るだけでは、
読みやすさや意味の維持を確認したことにはなりません。

## リリースの検証

tag と完全に一致する変更のないソースから、まだ存在しない出力パスを指定して作成します。
繰り返しビルドの比較には、それぞれ別の出力ディレクトリを使います。

```text
go run ./tools/repoctl release-build --version X.Y.Z --out <new-directory>
go run ./tools/repoctl release-check --dir <directory> --version X.Y.Z
go run ./tools/repoctl release-repeat --dir <directory> --version X.Y.Z
go run ./tools/repoctl release-smoke --dir <directory> --version X.Y.Z
```

`release-check` は他プラットフォームの実行ファイルを起動せず、6 アーカイブ一式と
Go 実行ファイルのビルド情報を静的に検査します。`release-smoke` は checkout の外へ展開し、
実行ファイルの検索パスを制限して、ホストと一致する対象だけを実行します。
状態には空白・非 ASCII 文字を含む独立した home を使います。同一ソース・同一ツールチェーンで
繰り返し作成し、実行ファイルとアーカイブの digest、checksum、manifest を比較します。
コンパイル成功をバイト列比較の代わりにしません。リリース CI の builder は Go 1.27.1 に固定します。
Windows/macOS/Linux のネイティブ smoke 結果と arm64 の検証範囲は、
[リリース計画](exec-plans/completed/standalone-release-finalization.ja.md)に明記します。

### 公開の条件

tag workflow は、リポジトリ検査、成果物の静的検証、繰り返しビルドの比較、ネイティブ smoke job の
成功を公開条件にします。公開 job は再ビルドせず、検証済みの候補バイト列をアップロードします。
workflow が存在するだけでは、リリースやネイティブ検証が成功した証拠にはなりません。

### 繰り返しビルドと専用preview

`release-repeat` は tag に対応する候補を検証し、同じコミット・ツールチェーンで再ビルドして、
出力 8 ファイルすべてのバイト列を比較します。tag のないブランチや PR の検証には
専用の preview を使います。

```text
go run ./tools/repoctl release-verify --out <new-directory>
go run ./tools/repoctl release-preview-smoke --dir <directory>
```

`release-verify` は変更のないソースを要求し、専用 clone 内だけに `v0.1.0` を作成して
2 回のビルド、バイト列比較、ローカルのネイティブ smoke を実行した後、候補を新規出力先へコピーします。
`release-preview-smoke` は別の専用 clone で、preview を現在のコミットと照合します。
どちらも公開 ref の作成やリリース公開は行いません。preview workflow はこの比較と
3 OS のネイティブ smoke job を実行します。tag workflow は公開前に、独立して繰り返しビルドと
3 OS のネイティブ smoke の成功を要求します。これは workflow の条件であり、
個別の実行が成功したという主張ではありません。

## 常駐processの検証

[process ExecPlan](exec-plans/completed/persistent-process-runtime.ja.md)で直接の受け入れ証拠を
追跡します。config testはprocess/Compose/Androidの厳密なfield variant、null/空/mergeした
YAML field、名前付きTCP port、local readiness endpoint参照、portableなdirectory名、
既存canonical snapshotの維持を検証します。adapter testは閉じた実行ファイル/cwd解決、補間、
専用状態と識別情報/起動前redaction証拠、native識別情報の復旧、上限付きlogs、
host secret変更後のredactionを
検証します。native primitive testは識別情報不一致、停止の再実行、子孫が残る起点終了を扱い、
既存Android detached testも回帰範囲に残します。

lifecycleの受け入れには、独立CLIをまたぐ存続/観測、並行2 lease、兄弟/無関係processの存続、
永続化失敗、quarantine、TCP占有、source変更、実HTTP readiness、process/Compose共存、
Browser状の状態/CDP状fixtureも必要です。このfixtureでBrowser自体の機能は実装しません。
native Windows/macOS/Linux実行が必須であり、cross-buildだけでは受け入れ完了にできません。
local adapter、race、integration、cross-buildの証拠と、`f588960`のnative
Windows/macOS/Linux CI成功（Verify 34226859965）を完了Planに記録しています。

## Browser/CDPの検証

受け入れ証拠は[完了browser ExecPlan](exec-plans/completed/browser-cdp-automation.ja.md)で管理します。
`TestBrowserManifestContract`、`TestBrowserManifestNegativeFixtures`、
`TestBrowserRequiresProcessRuntime`、`TestBrowserAbsentPreservesLegacyCanonicalShape`は、
明示的binding、正確な専用profile/debugging flag、YAML null/merge/alias、互換性を検証します。
config単体・race testはローカルで成功しました。baselineの`go test -race ./...`も成功しました。
初回harnessはunit/vet成功後、提示された日本語planに翻訳metadataがなくdocs-checkで失敗しました。
この失敗と修正を英日planに記録します。

### Browserの実行記録

実際のheadless Chrome for Testing 152.0.7977.82 / CDP 1.3をGo 1.27で動かし、
`391288c`の3 OSすべてで成功しました（Browser native 34247636411）。fixtureではAX/DOM、
screenshot、Unicode入力と消去、古い参照の拒否、iframe/shadow観測、上限付き診断、
lease所有backend、永続的な入力redaction、安全なprofile cleanupを検証しました。
Planにはnativeの証拠とCI修正履歴を、mock testやcross-buildと分けて記録しています。

既存の local Browser/CDP matrix は、`440082b` の Windows・macOS・Linux で再び成功しました
（[run 34320519250](https://github.com/mahcialet/agent-env/actions/runs/34320519250)）。
この回帰検証の証拠は、remote Browser 操作の受け入れ検証とは区別します。

## 複数 host の native 検証

```text
go test -tags=multihostintegration ./internal/cli -run '^TestMultiHostNativeCLI$' -count=1 -v -timeout=12m
```

この明示実行 test には Go と Git が必要です。native の agent-env と commit 済み process fixture を build し、
Go で短期間の test 証明書を作って、別々の状態 root を持つ controller/client/二 worker の実 process を TLS で接続します。
shell script、Docker、SDK、browser は不要です。role/enrollment の拒否、lease 全体の配置、drain、
同時に稼働する二つの lease、port/worktree の分離、local force 拒否、controller 停止・再起動、
native process の識別情報を保持した worker 再起動、相対実行ファイル path で起動する名前付き test、
保持した log、digest を検証する登録済み artifact の download、期限更新、client/worker の環境変数の分離、
独立した cleanup を検証します。
cleanup を確認できなければ調査用に fixture 状態を保持します。

### Roleの実行記録と未検証の範囲

Linux/amd64 では 21.406s で成功しました。初回実行では、通信時の JSON object 順序の正規化によって
実際に plan digest が不一致になる問題を検出し、manifest の意味に基づく正規化と恒久的な source 往復回帰 test で修正しました。
拡張した fixture は、`440082b` の Windows・macOS・Linux の全 job で成功しました
（[native workflow run 34320519252](https://github.com/mahcialet/agent-env/actions/runs/34320519252)）。
各 runner は同じ host 上の二つの worker root を使っています。これを物理マシン・VM の複数 host 動作の証明とは扱わず、
cross-build も native role 実行の証明にはしません。文書化した範囲で最終受け入れは完了し、
正確な結果は [ExecPlan](exec-plans/completed/multi-host-control-plane.ja.md) に記録します。

### Remote runtimeのfixture

追加の remote runtime fixture も、同じ build tag を指定して明示的に実行します。

```text
go test -tags=multihostintegration ./internal/cli -run '^TestMultiHostRemoteBrowser$' -count=1 -v -timeout=12m
go test -tags=multihostintegration ./internal/cli -run '^TestMultiHostRemoteCompose$' -count=1 -v -timeout=12m
```

Browser fixture には、PATH 上で直接実行できる互換 `google-chrome` と、利用可能な browser sandbox が必要です。
Compose fixture には、動作する Docker Compose と Podman/podman-compose の両環境が必要です。
前提条件がない状態でこれらの test を選ぶと失敗します。実際の controller/worker/runtime process を起動し、
登録済みの証拠を保持して、lease 単位で cleanup します。通常の単体 test はこれらの外部 runtime を起動しません。
この実行方法の記載は、全 OS で remote の native 受け入れが成功したという主張ではありません。
検証した範囲と結果は 完了Plan を参照してください。

### パスとWSLの検証範囲

Windowsの実行パステストでは240 UTF-16単位の境界、補助文字、解決済みパス、派生する
worktree/runtimeディレクトリと、予約・展開・出力作成前の拒否を検査します。
Windows以外の深いsource lifecycleは成功テストを維持します。直接interopのテストは
絶対・相対・PATH・symlink経由のPEを拒否し、nativeの.exe名は維持します。
WSL stateテストではkernel/filesystem/mountの観測を注入し、独自mount、alias、未作成homeを検査します。
実WSL2のmount/interop実行に代わる証拠ではなく、その環境は未検証です。

## 証拠の種類とテストアーキテクチャ

証拠は何を示すかで分類する。回数を増やしても証拠の種類は変わらず、反復は決定的な再現の代わりにならない。受け入れの主張ごとにrevision・コマンド・環境・結果・限界を記録する。

| 種類 | 示せること | 示せないこと |
| --- | --- | --- |
| 不変条件の強制検証 | 明示的barrierや状態遷移で問題の順序・境界へ到達し、原因を区別するoracleで不変条件を確認 | 全スケジューリングや実OSの挙動 |
| native/integrationの直接観測 | 前提条件を満たした指定hostで実プロセス・プロトコル・engineの挙動を確認 | 別OSや未使用の基盤 |
| race・静的検査・ツール | 実行した経路や構造規則で問題が報告されなかった | 全実行順序、oracleの正しさ、翻訳の意味一致 |
| 安定性・反復 | 記録した回数・CPU設定で観測した安定性 | 決定的な再現や回数による証明の強化 |
| コンパイル・構造 | 型・import・生成物・対象platformのビルド検査への適合 | native実行やruntime受け入れの完了 |

種類は異なる主張を表し、上位の証拠で置換できる順位ではない。一つのテストが複数種類の証拠を持つ場合も、それぞれ区別する。

### fixtureの所有と実行順序

helperごとに所有者、資源、goroutine/callback、共有状態、完了通知、cleanup順序、失敗出口、利用箇所、仮定を記録する。cleanupでは受付を止め、必要に応じてcancelやtransport closeを行い、所有処理をjoinしてからDBや一時状態を破棄する。listenerのcloseは受付済みconnectionを閉じない。HTTP serverのcloseはhijack済みWebSocket callbackをjoinしない。cancel通知はjoinではない。

assertionより先に失敗時cleanupを登録する。native資源が残り得るcleanup失敗を無視しない。WaitGroupへの登録はWaitより前に順序付ける。atomicでcounterを保護しても完了順序は別途必要である。join対象自身からjoinしない。timeoutは失敗検査の時間を制限するだけで、callback開始・cleanup完了・副作用不在を示さない。適切な箇所ではchannel/barrierや`testing/synctest`を使う。実socket/native processの組み合わせを確認した証拠は別途区別する。

### oracle・指摘・予防検査

負例では、assertionを偶然満たす最も早い拒否経路を特定する。意図した変更・境界への到達を必須とし、特定のerror/stateを確認する。必要に応じて隣接する正常対照を加える。遅延ファイルの不在は独立したprocess完了証拠ではない。期待値を実装と同じロジックで計算せず、保存状態や副作用を独立して比較する。

修正前に指摘ID、不変条件、最初の現実的な検出機会、見逃し分類、test/productionの範囲、同種箇所への影響、ACCEPT/REJECT/DEFERと理由を記録する。採用した修正は可能なら同じoracleで修正前失敗・修正後成功を示す。新test API不足のcompile失敗は再現に数えない。native対照を実行できなければ明記する。

安定したlifecycle・境界の不変条件には対象を絞った回帰テストを優先する。既存`check`とnative/race CIで実行し、別policy runnerを増やさない。テスト名・文章・sleep使用・成功回数から証拠品質を機械判定しない。これらは文脈を踏まえてreviewする。独立reviewでは差分、到達性、指摘の判断、実際の証拠を確認する。英日読者reviewと意味一致reviewはsource hash確認と区別する。

固定のnative資源poolを共有するsuiteは、分離を確認できるまで直列実行する。競合後の単独再実行は診断証拠であり、元の失敗を消さない。
