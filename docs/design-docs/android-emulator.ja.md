---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/design-docs/android-emulator.md
source_sha256: 1a6bfe77ed8536b54f3f3011014d5da25e6acb882af17ab4daa67e72f6134f6f
---

[English（翻訳元）](android-emulator.md)

# Android Emulator リソース設計

この設計では、共有 SDK サービスの所有権を引き取らずに、lease が Emulator を
所有・管理する仕組みを説明します。対応する設定と動作は
[製品仕様](../product-specs/android-emulator.ja.md)で定義します。

独立した `app.AndroidProvider` を `runtime/android` が実装します。
App は saga の順序、ロック、readiness、永続化を担当します。Domain は純粋な識別情報を
定義し、SQLite は排他的な予約を記録します。`execx` は OS ネイティブの切り離された
プロセスを操作します。

Android は Compose や Flutter を import しません。実行時間を制限したコマンドツリーは既存のキャンセル動作を維持し、常駐する Emulator プロセスには別の API を使います。

## 専用 AVD と予約

各ランタイムに、lease から導出した一意の AVD 名と、
`leases/<id>/android/<runtime>/avd/` 以下のパスを割り当てます。
テンプレートは読み取り専用の設定と不変の SDK イメージを提供します。
テンプレートの userdata、snapshot、lock を、書き込み可能な状態として共有することは
ありません。危険な書き込み先、シンボリックリンク、稼働中のテンプレートは検証で拒否します。

1 回の即時 SQLite トランザクションで、lease/worktree の識別情報、AVD の識別情報、console/ADB のポートペアを予約します。
console ポートは 5554〜5682 の偶数で、その次の奇数ポートを ADB が使います。一意性制約が同時実行する CLI プロセス間を調停します。

ポート確認は外部との競合を検出するだけで、所有権を証明せず、未記録の代替ポートを選びません。
隔離中はポートと容量を保持し、確認済みの `released` 状態だけが予約を解放します。

## Saga と保守的な復旧

App は副作用の発生前に起動意図を永続化します。アダプターは起動前に破棄可能な AVD 状態の外側へ所有権マーカーを書き、起動後に PID とプロセス生成時の識別情報を記録します。
起動識別情報の永続化前にクラッシュすると所有権が曖昧になるため、隔離が必要です。

保持する stdout/stderr はローカルプロセスの診断情報であり、logcat ではありません。レジストリと環境記述子は予約と観測結果を保持します。

停止処理では、`kill` に使うものと同じ認証済み console 接続で lease 由来の AVD 名を確認し、再利用されたポートでの確認と停止の競合を防ぎます。OS ネイティブの生成時識別情報で PID の再利用を検出します。書き込み可能な状態を削除する前に、ネイティブのプロセスグループ全体または Windows Job の終了と、ポートの不在を確認しなければなりません。

Unix では、ルートプロセス終了後にグループのメンバーが残っていると削除を阻止しますが、停止を許可する系統の証明にはなりません。グループが不在になるまで観測は不確定を報告します。

Windows では、リソースごとの専用 helper が CLI/ルートプロセス終了後も名前付き Job のハンドルを保持します。helper は明示したハンドルを継承し、全 Job メンバーの停止に応じた終了と、対応する空 Job の証拠の永続化を行います。その証拠なしに Job 名が失われた場合や、別の Windows ログオンセッションから観測した場合は不確定です。

手動終了を確認できれば degraded にし、所有権が不確定なら隔離します。Reconcile は Emulator を引き取ったり再起動したりしません。解放済みランタイムの旧ポートは、継続する所有権を意味しません。

## 共有 ADB サーバーの寿命

ポート 5037 のローカル ADB サーバーは共有 SDK 前提条件です。
Emulator 起動前に、アダプターは `adb version` で SDK クライアントのプロトコル版を読み、読み取り専用の `host:version` プロトコルで `127.0.0.1:5037` を直接確認します。既存サーバーが非互換、不正、観測不能なら、置き換えを試みず前提条件エラーにします。

アダプターは接続拒否をサーバー不在と判定し、起動が必要なら切り離されたプロセス API で `adb -L tcp:localhost:5037 start-server` を実行します。Emulator のネイティブなプロセス管理範囲とは分離します。上限付きの readiness 待機で、Emulator 起動前に互換性を再確認します。

サーバー起動には専用の `adb-server.stdout.log`、`adb-server.stderr.log`、`adb-server-start.json` があり、保持されるランタイム証拠の隣に置きます。その識別情報を Emulator のプロセス識別情報として保存しません。
lease の補償処理、destroy、GC は、失敗した割り当て中に起動したものも含め、共有サーバーを停止しません。

起動状態の観測は、`adb -H 127.0.0.1 -P 5037 -s <reserved-serial> shell getprop sys.boot_completed` の実行前に共有プロトコルを再確認します。継承されたサーバー接続先と serial の変数はクリアします。このクライアント接続形式は、サーバー消失時の自動起動を避けます。

直接の互換性確認も必要です。[ADB のバージョン不一致処理は `-H` を指定しても既存サーバーを停止する可能性があります](https://android.googlesource.com/platform/packages/modules/adb/+/9084198a2d4b0f6a0f174260fb42da33485b684d/client/adb_client.cpp#311)。共有前提条件が失われれば readiness は degraded となり、観測処理は修復しません。

この分離は Windows で重要です。[ADB daemon の起動は `DETACHED_PROCESS` を使い](https://android.googlesource.com/platform/packages/modules/adb/+/9084198a2d4b0f6a0f174260fb42da33485b684d/adb.cpp#938)、console を切り離しますが、継承した Job への所属は解除しません。[Windows Job の規則](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects)により、分離しなければ自動起動したサーバーが終了対象の bounded command Job に入るか、Emulator の Job の生存メンバーとして数えられ続けます。

## netsim探索と補助プロセスの専用化

Emulatorとnetsimd補助プロセスは、同じリース専用デーモンを探索する必要があります。
子プロセスだけの `TMPDIR`・`TMP`・`TEMP`・`XDG_RUNTIME_DIR` を
`<AVDHome>/emulator-data/Temp` に設定し、Windowsでは子の `LOCALAPPDATA` を
`<AVDHome>/emulator-data` に設定します。ホスト全体の環境は変更しません。
これによりEmulatorクライアントの一時ファイル探索を、Linuxのデーモンruntimeパス、
Windowsの `LOCALAPPDATA/Temp`、macOSのOS標準一時パスと揃えます。

`NETSIM_INSTANCE=1` はクライアントの既定インスタンスに合わせ、`NETSIM_HCI_PORT=0` は
一時的に割り当てるHCIリスナーを要求します。Emulator argvに `-netsim-args --no-web-ui` を
渡し、補助UIの固定8080ポートを避けます。無効にするのは補助Web UIだけで、無線とゲスト
ネットワークは維持します。共有ADB方針、プロセス誕生・group・Jobの証明、保守的cleanupは変更しません。

SDK 37.1.11 build 15917651 / netsimd 0.3.114の専用デーモンprobeで、専用 `netsim.ini`、
gRPCリスナー、HCIポート0設定、libslirp有効を確認し、そのデーモンは停止しました。
この重点probeは二つのEmulatorの全ライフサイクル成功を証明しません。
その別検証と、隔離の契機となった共有補助プロセスのcleanup失敗はFlutter実行計画に記録します。

## 移植性と証拠

実装は、ネイティブの argv、明示したパスと環境、shell/CGO 不要という条件を守ります。
SDK とイメージのアーキテクチャ互換性は、ホスト側の前提条件です。
WSL は Linux ホストとして扱います。

テストは discovery、危険なテンプレート、console 所有権、切り離したプロセスの寿命、
PID 再利用、同時予約、補償、兄弟 lease の分離、隔離を対象にします。
ネイティブ CI とクロスビルドは、実際のアクセラレーション付き Emulator テストとは別です。

実 SDK の統合は Linux で実行済みです。Windows/macOS での実 SDK、アクセラレーション、
共有サーバーの起動動作は未検証です。
[ExecPlan](../exec-plans/completed/android-emulator-lease.md)に、証拠、実装判断、
未解決の前提条件、プラットフォームごとの検証の不足を記録します。
