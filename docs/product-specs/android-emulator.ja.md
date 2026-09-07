---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/product-specs/android-emulator.md
source_sha256: 25633f8663cd793ae0c2bb04240bbea4f5ec10d21965f2eb5b437bd9d0b05e84
---

[English（翻訳元）](android-emulator.md)

# Android Emulator lease

`android-emulator` ランタイムは、Flutter とは独立して、1 台の Emulator と専用の書き込み可能な AVD 状態を所有します。APK のインストール、ビルド、adb reverse、UI 操作、スクリーンショット、logcat、実機、リモートホストはこの範囲に含みません。

```yaml
version: 1
sources:
  app: {repository: ., default_ref: HEAD}
runtimes:
  phone: {type: android-emulator, source: app, avd: Pixel_API_35}
components:
  device: {runtime: phone}
stacks:
  android-runtime: {roots: [device]}
```

`source` は固定された管理対象 source を識別します。`avd` は既存のローカル AVD テンプレートを選びます。Android ランタイムでは Compose file、project directory、service、endpoint を指定できません。Compose と Android は 1 つの stack に共存できます。同じ Android ランタイムを共有するコンポーネントは、その lease の Emulator を共有します。別の lease には常に異なる書き込み状態、AVD 識別情報、console/ADB ポート、serial を割り当てます。Android ランタイム名は、すべての OS で大文字小文字を同一視しても区別できなければなりません。生成する runtime directory は、既存シンボリックリンクによる別名も含め、選択されたどのテンプレートや system-image のツリーとも重複してはいけません。

`plan . --stack android-runtime --output json` は SDK の探索、予約、ランタイムの副作用なしに要件を示します。

`create` は予約前に、実行可能な SDK tool、ホストのアクセラレーション、AVD を検証します。Android のみの stack に Docker は不要です。tool やテンプレートの欠如は前提条件エラー（終了 status 3）です。インストール済みイメージの ABI metadata（`source.properties`）は、アクセラレーション対応でネイティブホストと互換性があるイメージを識別しなければならず、テンプレートの ABI/CPU 設定も一致する必要があります。既存の共有 ADB サーバーは予約前に確認します。不在は許容し、後で起動します。

既定の起動待ち時間は 2 分です。Ready には、所有する device の Android boot-completed property が必要です。

ローカル ADB サーバーは選択 SDK と互換性がなければなりません。作成時に不在サーバーを Emulator のプロセス管理範囲とは別に起動します。不正または非互換の既存サーバーは、自動的に置き換えず拒否します。共有サーバーの診断情報を保持し、lease を destroy しても共有 SDK service は停止しません。

`runtimes[].android` はテンプレート、SDK/system image、専用パス、一意の AVD 名、console/ADB ポート、serial、OS ネイティブのプロセス生成時識別情報、状態を記録します。

`doctor --runtime android-emulator` は SDK と AVD の前提条件を報告し、`doctor <lease-id>` は実際の lease の診断を報告します。リポジトリを指定した Android doctor は、宣言された各 Android ランタイムが選んだ AVD のテンプレート、イメージ、ホストの前提条件を検証します。

`logs <lease-id>` は active または released な lease の Emulator と共有 ADB の保持済み stdout/stderr を公開し、`--component` 選択と秘密情報の伏せ字処理に対応します。これはプロセスログであり logcat ではありません。

Show/list/reconcile は実際の device を観測し、手動終了を確認すると active lease を degraded にします。

起動失敗は逆順で補償します。あるランタイムの削除が失敗した場合、その証拠と予約を保持しつつ、独立したランタイムの削除を続けます。全ランタイムの削除を確認するまで source は残します。操作のキャンセルやレジストリの fence 喪失があれば、それ以降の副作用を止めます。

Destroy は一意の AVD 名を確認し、同じ認証済み console 接続で停止を要求してから、終了を確認した後に専用の書き込み状態を削除します。プロセス/AVD の識別情報が曖昧な場合、所有権の証拠が変わった場合、パスが危険な場合、削除が不完全な場合は lease を隔離し、予約を保持します。Force でもこの条件を迂回しません。Destroy を繰り返しても、解放済みポートの新しい利用者に影響しません。テンプレートと兄弟 lease の状態は削除しません。予約ポートを外部が使用していれば作成を失敗させて補償します。割り当てが黙って別ポートへ移ることはありません。グローバルな adb shutdown や Emulator prune は禁止です。

SDK イメージと互換性のあるホストアクセラレーションは外部の前提条件です。ネイティブの fake-adapter と process のテストは Windows/macOS/Linux を対象とし、実 Emulator の証拠は [ExecPlan](../exec-plans/completed/android-emulator-lease.md) に別途記録します。SDK 不在やクロスビルドを実 Emulator の検証成功として報告しません。復旧と予約については[設計](../design-docs/android-emulator.ja.md)を参照してください。

## 実環境の統合検証

互換性のある SDK イメージをインストールし、停止済み AVD テンプレートを作成します。`AGENT_ENV_ANDROID_TEMPLATE` にその名前を設定し、必要に応じ SDK/AVD 探索用変数を設定してから、次のネイティブ Go コマンドを実行します。

```text
go test -tags=androidintegration ./internal/runtime/android -run ^TestRealAndroidEmulatorLeases$ -v -count=1
```

この明示実行のテストには、空いている予約対象 Emulator ポートと使用可能なアクセラレーションが必要です。前提条件が欠けていれば skip ではなく失敗します。実際の SQLite/app orchestration と合成 source provider で 2 台の実 Emulator を起動し、兄弟の生存と手動終了を確認してから、所有を確認したリソースだけを削除します。削除が不確定なら、復旧用にテストが表示した一時状態パスを保持します。別途用意された Git/Compose の統合 fixture を置き換えるものではありません。
