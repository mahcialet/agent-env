---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/PORTABILITY.md
source_sha256: f7f6668e83b187a9bcc43fcef1a753a041a9619371f18c115dce8f47884826a8
---

[英語版（翻訳元）](PORTABILITY.md)

# 移植性

このmoduleはGo 1.26.xと1.27.x、ネイティブWindows・macOS・Linuxを対象とし、CGOを必要としません。Gitと選択したCompose providerのツールは外部runtimeの前提条件です。cross-compilationが証明するのはビルド互換性であり、ネイティブのプロセス、パス、SQLite、Dockerの振る舞いではありません。

## 状態とパス

| プラットフォーム | 既定の状態ディレクトリ |
| --- | --- |
| Linux | `$XDG_STATE_HOME/agent-env`。未設定時は`~/.local/state/agent-env` |
| macOS | `~/Library/Application Support/agent-env` |
| Windows | `%LOCALAPPDATA%/agent-env`。未設定時はユーザーの`AppData/Local/agent-env` |

`AGENT_ENV_HOME`はディレクトリ全体を上書きし、絶対パスである必要があります。永続的なSQLite状態と証拠にはネイティブのファイルシステムAPIを使います。管理対象worktreeは対象checkoutの外にある一意なlease/sourceパスを使います。テストには空白、Unicode、Windows drive URI、traversal、symlinkによる範囲外への移動を含めます。

ソース内のマニフェストパスにはforward slashを使います。ネイティブな絶対ローカルリポジトリパスは、対応するプラットフォームで使用できます。非Windowsホスト上のWindows driveパスや混在したパス形式は、検出できる範囲で拒否します。状態配置にsymbolic linkは必要なく、アプリケーションレベルのPOSIX lock fileの代わりにSQLite操作lockを使います。

## ネイティブツールと取消

コマンドは実行ファイルとargv、明示的な作業ディレクトリ、deadline、stream出力を使います。Git検査には機械可読出力を使い、Compose engine検査には構造化出力と記録したprovider/engine識別情報を使います。改行処理はCRLFを許容します。Go製のリポジトリharnessは標準ツールを直接呼び出し、shell script言語を必要としません。

Unixでは管理対象コマンドのprocess groupで子孫を取り消します。Windowsではコマンドの子プロセスを実行前にJob Objectへ割り当て、取消やtimeout時にはJobを終了します。これは時間を制限した名前付きテストとprobeを支えます。汎用の常駐process runtimeには後述の独立したmanaged detached interfaceを使います。OSの隔離機構から意図的に逃れるバックグラウンドプログラムは、信頼済みリポジトリモデルの範囲外です。終了を検証できない場合は、型で識別できるプロセスツリー未確認の結果として通知します。appはレジストリ記録をrunningに保ち、レビューした復旧により完了が確定するまでcleanupを拒否しなければなりません。

Windowsの`.cmd`と`.bat`の実行処理はWindowsアダプターに隔離します。wrapper引数はプラットフォーム固有の経路でquoteし、安全に表現できないtokenはargvを黙って変えるのではなく拒否します。ネイティブ実行ファイルのargvテストには、空白、引用符、末尾の区切り文字、Unicodeを含めます。shellに影響されるwrapperの振る舞いは別個にテストする境界です。

## Docker context

WindowsとmacOSは通常Docker Desktopを使います。LinuxはComposeが動くDocker Engineまたはrootless Dockerを使えます。選択したDocker contextを割り当て前に取得し、その後ユーザーのactive contextが変わっても、観測、ログ、cleanupで使います。contextに接続できるだけでは、daemonがローカルworktreeのbindパスへアクセスできるとは限りません。リモートdaemonからのパス可用性はホストの前提条件です。

WSLはLinuxとして扱います。リポジトリ、Git、Docker接続、パスを一貫してその境界の同じ側に置いてください。Windows/WSL混在リースと、WSLからWindowsホストのAndroid Emulatorを制御するworkflowは対応外です。

## Podmanの前提条件と検証の限界

Podman providerにはPodman 5.xと独立したpodman-compose >=1.6.0,<2.0.0が必要です。
Pythonはproviderのホスト側導入に属し、agent-env coreの依存関係ではありません。
現在の実行ファイルがnativeの子process bridgeとなり、生成shell scriptは不要です。
local Linuxではlocal engineを固定し、remote/Machine呼出しでは可変の接続名ではなく
解決済みendpointを保持します。remoteのbind pathはそのengineからアクセスできる必要が
あります。remote loopback endpointを報告するにはhost側の到達性確認が必要です。

Podman 5.4.2とpodman-compose 1.6.0で、Docker共存、動的endpoint、匿名volume cleanupを
含む実Linux rootless integrationが成功しました。Windows/macOS/Linuxのnative provider CIは4a5de3d（run 34216579481）で成功です。
実機のPodman Machine環境はなく、cross-buildやfake接続testではMachine証拠を代替できません。正確な証拠は
[provider plan](exec-plans/completed/compose-provider-podman.ja.md)に記録します。

## 検証範囲

CIは、対応する両Go minorバージョンについてWindows・macOS・Linuxのネイティブ単体/harness jobを定義します。release-build matrixは、windows/amd64、darwin/amd64、darwin/arm64、linux/amd64、linux/arm64で`CGO_ENABLED=0`を設定します。Linuxではrace testと明示的な実Docker統合も実行します。

c641286に対するCI 34124194139では、6件すべてのネイティブOS/Go jobと5件すべてのCGO無効buildが成功しました。同じrunでLinux raceと実Compose統合も成功しました。ローカル実fixtureも、並行プロジェクト、Unicode worktree、複数リポジトリの固定、名前付き証拠、rollback、変更済みソースのcleanupを含め成功しました。[完了済み実装計画](exec-plans/completed/agent-env-mvp.md)に、すべての証拠とネイティブ回帰修正を記録しています。Windows/macOSでの実Docker統合は未実行であり、適切なrunnerが必要です。

## Androidの永続プロセス

Androidは、ネイティブのファイル出力を持つ独立したdetached-process APIを使います。呼び出したCLIとその要求contextが終了してもプロセスは存続します。Linux/macOSはprocess-group識別情報を保持し、root終了後に残る子孫を系譜不確定として扱い、cleanupを禁止します。Windowsは中断状態のプロセスを名前付きJobに割り当ててから再開します。専用helperがJobが空になるまでhandleを保持し、その証拠を記録します。helperの証拠が欠けている、またはlogon sessionが異なる場合はcleanupを禁止します。生成時識別情報により再利用されたプロセスを拒否し、観測が不確定なら書き込み可能状態を削除しません。永続起動にはネイティブ実行ファイルを使い、batch wrapperやshellには依存しません。

SDK探索は`ANDROID_HOME`、次に`ANDROID_SDK_ROOT`、最後にプラットフォーム既定値を使います。両方の変数が設定され、その値が競合する場合は失敗します。テンプレートは`ANDROID_AVD_HOME`、`ANDROID_USER_HOME/avd`、またはユーザーの`.android/avd`を使います。Emulatorのarchitectureと利用可能なホストアクセラレーションは前提条件です。

`127.0.0.1:5037`の互換ローカルADBサーバーは共有前提条件です。アダプターは起動またはboot検査の前に、直接の読み取り専用`host:version`応答をSDKクライアントのプロトコルバージョンと比較します。互換性がない、または不正な応答のサーバーは拒否します。サーバーがなければ、Emulatorより先にSDKの`adb -L tcp:localhost:5037 start-server`を別個のdetached起動で実行し、起動診断を保持します。時間制限付きboot検査は`-H 127.0.0.1 -P 5037 -s <reserved-serial>`を使い、継承したserver-routing変数を解除します。このboot検査では、欠けたサーバーを起動しません。互換性probeは、通常のSDKクライアントのversion不一致による置換経路を防ぎます。共有ADBはリースcleanupの対象外です。

ネイティブ単体CIと実Emulator統合は別です。実SDK統合はLinuxで実施しています。Windows/macOSでの実SDK起動、アクセラレーション、共有サーバーの寿命は未検証です。Android ExecPlanにテストしたrevisionと残るプラットフォーム上の不足を記録しています。

## スタンドアロンリリースの対象

アーカイブの仕様では、従来の 5 ビルド対象に windows/arm64 を加えます。
Windows、macOS（`darwin`）、Linux のそれぞれに amd64 と arm64 を用意します。
Windows は ZIP、macOS/Linux は tar.gz を使い、全対象を `CGO_ENABLED=0` でビルドします。
パッケージ生成は Go ライブラリで行い、外部のシェル、tar、zip、checksum ツールは不要です。
アーカイブはバージョン付きディレクトリを一つ持ち、作成時刻ではなく tag 対象コミットの
時刻を使います。ZIP は精度の粗い DOS フィールドに加え、UTC の拡張 timestamp を保持します。
状態の保存先は引き続き OS 標準の state root または絶対パスの `AGENT_ENV_HOME` であり、
空白や非 ASCII 文字を含むパスも扱います。

6 対象のクロスビルド成功は、6 通りすべてのネイティブ動作を証明しません。
arm64 を含め、実際に smoke test を実行したネイティブ runner を個別に記録します。
アーカイブが生成されたことから実行済みとは判断しません。現在の証拠は
[リリース計画](exec-plans/completed/standalone-release-finalization.ja.md)で管理します。
展開した CLI の実行に Go は不要です。選択した機能の外部前提条件は
[README](../README.ja.md)に記載しています。

## 汎用の常駐process runtime

`type: process`はnative argvと、起動元CLIの終了後も存続するforegroundの起点processを
使います。source相対のcwd/実行ファイルpathはsymlink解決後もsource内に閉じます。
PATH toolにはsource/hostの由来と読み取れる実行ファイルのdigestを記録しますが、host toolの
再現性は主張しません。coreに暗黙のshell、batch wrapper、service manager、新たな言語runtime、
daemonは不要です。

Unixは各signal直前に生成識別情報とprocess groupへの所属を確認します。観測とgroupへの
signal送信はatomicではなく、最後のnative syscallとの競合を減らせても解消はできません。
子孫が生きたまま起点が消えた場合は所有が不確実となり、以後の破壊的作用を止めます。
Windowsは保持したhandleを使い正確なnamed Jobを再検証して停止します。これはJobの強制停止で
あり、console signalによる穏当な終了を約束しません。識別情報が不確実ならleaseのport、状態、
sourceを保持します。

名前付きportはagent-env lease間のloopback TCP割り当てを予約します。対象がbindするまでの
間は外部占有と競合し得ます。socket activationやlisten socket継承は使いません。実装中の
native adapter test、race test、cross-buildは成功しています。実常駐processの
Windows/macOS/Linux integrationも`f588960`（Verify 34226859965）で成功しました。
最終CI証拠は[完了process plan](exec-plans/completed/persistent-process-runtime.ja.md)
で別途追跡します。

## Browser/CDPの前提条件

browser自動操作には、process runtimeから直接起動できる互換native headless Chromium系実行ファイルが
必要です。shell wrapperや、CDP browser rootが所有native rootと異なるlauncherは非対応です。
argvには`--enable-automation`と専用`${runtime_dir}/profile`などを必須とし、接続ごとにCDPから
実際のcommand lineとPIDを検証します。Go WebSocket transportを使い、coreコマンドに
Node、Python、browser driver、CGO、shellの要件を追加しません。Chromeは同梱しません。
選定したnative matrixはChrome for Testing 152.0.7977.82、Go 1.27、Windows/macOS/Linuxです。
実測browser/protocol versionとnative成功・失敗の証拠は
[完了browser plan](exec-plans/active/browser-cdp-automation.ja.md)に記録しています。
`391288c`の3 native jobがすべて成功し（Browser native 34247636411）、CDP 1.3を報告しました。
この実行結果はcross-buildの証拠と分けて扱います。

Linuxでは、導入したbrowserのsandboxを利用できる必要があります。UbuntuのAppArmorは、
package profileの対象外へ展開したChrome for Testingのuser namespace利用を拒否する
場合があります。native CIでは[Chromiumの手順](https://chromium.googlesource.com/chromium/src/+/main/docs/security/apparmor-userns-restrictions.md)
に基づき、固定versionのChrome実行ファイルだけを対象にAppArmorで利用を許可します。
host全体のuser namespace制限とChrome sandboxは有効に保ちます。これはUbuntu runner
の準備処理であり、agent-env自体はhostのsecurity policyを変更しません。

Windowsでは、downloadしたCfTのインストール先に、ChromiumのLPAC sandboxが必要とする
read/execute ACLがない場合があります。native CIではChromium公式testの設定に従い、
制限付きapplication-package SID（S-1-15-2-2）へbrowserインストール先の権限だけを
付与します。lease profileや無関係なdirectoryには権限を付与せず、sandboxも無効化しません。

native process treeの不在を証明した後、汎用process state cleanupはWindows共有違反を
呼出し側context内・最大2秒で再試行し、毎回所有権とpathを再検査します。持続するfile lockを
cleanup成功とせず、無関係なerrorは再試行しません。通常のresource保持・隔離が適用されます。
filesystemのcleanupとして扱い、browserのlifecycle管理をCDP adapterへ移しません。

2秒の予算は再試行の開始を制限します。実行中の同期filesystem削除を中断するものではありません。
