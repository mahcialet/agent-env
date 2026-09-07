---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/PORTABILITY.md
source_sha256: 53d10c89a2feb9efe5d7dbd7b1cfea407e6be97e4aa382340809215087a38eb4
---

[英語版（翻訳元）](PORTABILITY.md)

# 移植性

このmoduleはGo 1.26.xと1.27.x、ネイティブWindows・macOS・Linuxを対象とし、CGOを必要としません。GitとDocker Compose pluginは外部runtimeの前提条件です。cross-compilationが証明するのはビルド互換性であり、ネイティブのプロセス、パス、SQLite、Dockerの振る舞いではありません。

## 状態とパス

| プラットフォーム | 既定の状態ディレクトリ |
| --- | --- |
| Linux | `$XDG_STATE_HOME/agent-env`。未設定時は`~/.local/state/agent-env` |
| macOS | `~/Library/Application Support/agent-env` |
| Windows | `%LOCALAPPDATA%/agent-env`。未設定時はユーザーの`AppData/Local/agent-env` |

`AGENT_ENV_HOME`はディレクトリ全体を上書きし、絶対パスである必要があります。永続的なSQLite状態と証拠にはネイティブのファイルシステムAPIを使います。管理対象worktreeは対象checkoutの外にある一意なlease/sourceパスを使います。テストには空白、Unicode、Windows drive URI、traversal、symlinkによる範囲外への移動を含めます。

ソース内のマニフェストパスにはforward slashを使います。ネイティブな絶対ローカルリポジトリパスは、対応するプラットフォームで使用できます。非Windowsホスト上のWindows driveパスや混在したパス形式は、検出できる範囲で拒否します。状態配置にsymbolic linkは必要なく、アプリケーションレベルのPOSIX lock fileの代わりにSQLite操作lockを使います。

## ネイティブツールと取消

コマンドは実行ファイルとargv、明示的な作業ディレクトリ、deadline、stream出力を使います。Git検査には機械可読出力を使い、Docker検査にはJSONと記録したcontext識別情報を使います。改行処理はCRLFを許容します。Go製のリポジトリharnessは標準ツールを直接呼び出し、shell script言語を必要としません。

Unixでは管理対象コマンドのprocess groupで子孫を取り消します。Windowsではコマンドの子プロセスを実行前にJob Objectへ割り当て、取消やtimeout時にはJobを終了します。これは時間を制限した名前付きテストとprobeを支えるもので、汎用の永続ホストプロセスruntimeではありません。OSの隔離機構から意図的に逃れるバックグラウンドプログラムは、信頼済みリポジトリモデルの範囲外です。終了を検証できない場合は、型で識別できるプロセスツリー未確認の結果として通知します。appはレジストリ記録をrunningに保ち、レビューした復旧により完了が確定するまでcleanupを拒否しなければなりません。

Windowsの`.cmd`と`.bat`の実行処理はWindowsアダプターに隔離します。wrapper引数はプラットフォーム固有の経路でquoteし、安全に表現できないtokenはargvを黙って変えるのではなく拒否します。ネイティブ実行ファイルのargvテストには、空白、引用符、末尾の区切り文字、Unicodeを含めます。shellに影響されるwrapperの振る舞いは別個にテストする境界です。

## Docker context

WindowsとmacOSは通常Docker Desktopを使います。LinuxはComposeが動くDocker Engineまたはrootless Dockerを使えます。選択したDocker contextを割り当て前に取得し、その後ユーザーのactive contextが変わっても、観測、ログ、cleanupで使います。contextに接続できるだけでは、daemonがローカルworktreeのbindパスへアクセスできるとは限りません。リモートdaemonからのパス可用性はホストの前提条件です。

WSLはLinuxとして扱います。リポジトリ、Git、Docker接続、パスを一貫してその境界の同じ側に置いてください。Windows/WSL混在リースと、WSLからWindowsホストのAndroid Emulatorを制御するworkflowは対応外です。

## 検証範囲

CIは、対応する両Go minorバージョンについてWindows・macOS・Linuxのネイティブ単体/harness jobを定義します。release-build matrixは、windows/amd64、darwin/amd64、darwin/arm64、linux/amd64、linux/arm64で`CGO_ENABLED=0`を設定します。Linuxではrace testと明示的な実Docker統合も実行します。

c641286に対するCI 34124194139では、6件すべてのネイティブOS/Go jobと5件すべてのCGO無効buildが成功しました。同じrunでLinux raceと実Compose統合も成功しました。ローカル実fixtureも、並行プロジェクト、Unicode worktree、複数リポジトリの固定、名前付き証拠、rollback、変更済みソースのcleanupを含め成功しました。[完了済み実装計画](exec-plans/completed/agent-env-mvp.md)に、すべての証拠とネイティブ回帰修正を記録しています。Windows/macOSでの実Docker統合は未実行であり、適切なrunnerが必要です。

## Androidの永続プロセス

Androidは、ネイティブのファイル出力を持つ独立したdetached-process APIを使います。呼び出したCLIとその要求contextが終了してもプロセスは存続します。Linux/macOSはprocess-group識別情報を保持し、root終了後に残る子孫を系譜不確定として扱い、cleanupを禁止します。Windowsは中断状態のプロセスを名前付きJobに割り当ててから再開します。専用helperがJobが空になるまでhandleを保持し、その証拠を記録します。helperの証拠が欠けている、またはlogon sessionが異なる場合はcleanupを禁止します。生成時識別情報により再利用されたプロセスを拒否し、観測が不確定なら書き込み可能状態を削除しません。永続起動にはネイティブ実行ファイルを使い、batch wrapperやshellには依存しません。

SDK探索は`ANDROID_HOME`、次に`ANDROID_SDK_ROOT`、最後にプラットフォーム既定値を使います。両方の変数が設定され、その値が競合する場合は失敗します。テンプレートは`ANDROID_AVD_HOME`、`ANDROID_USER_HOME/avd`、またはユーザーの`.android/avd`を使います。Emulatorのarchitectureと利用可能なホストアクセラレーションは前提条件です。

`127.0.0.1:5037`の互換ローカルADBサーバーは共有前提条件です。アダプターは起動またはboot検査の前に、直接の読み取り専用`host:version`応答をSDKクライアントのプロトコルバージョンと比較します。互換性がない、または不正な応答のサーバーは拒否します。サーバーがなければ、Emulatorより先にSDKの`adb -L tcp:localhost:5037 start-server`を別個のdetached起動で実行し、起動診断を保持します。時間制限付きboot検査は`-H 127.0.0.1 -P 5037 -s <reserved-serial>`を使い、継承したserver-routing変数を解除します。このboot検査では、欠けたサーバーを起動しません。互換性probeは、通常のSDKクライアントのversion不一致による置換経路を防ぎます。共有ADBはリースcleanupの対象外です。

ネイティブ単体CIと実Emulator統合は別です。実SDK統合はLinuxで実施しています。Windows/macOSでの実SDK起動、アクセラレーション、共有サーバーの寿命は未検証です。Android ExecPlanにテストしたrevisionと残るプラットフォーム上の不足を記録しています。
