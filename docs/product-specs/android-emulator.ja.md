---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/product-specs/android-emulator.md
source_sha256: 4a7d0daec9f3d00db12dca884a0ed13258e75a325175f2969acd805b4c38be9f
---

[English（翻訳元）](android-emulator.md)

# Android Emulator lease

`android-emulator` は、リース専用の Emulator と書き込み可能な AVD 状態を割り当てる、実装済みのランタイムです。この文書は、その設定、起動完了の条件、安全に削除するための条件を定めます。Flutter とは独立して利用できます。

APK のビルドとインストール、adb reverse は [Flutter](flutter-android-runtime.ja.md)、UI 操作、スクリーンショット、logcat は [Android UI](android-ui-observer.ja.md) の仕様を参照してください。実機とリモートホストの配置管理は、このランタイム仕様の対象外です。リモート配置は[複数ホストの仕様](multi-host-control-plane.ja.md)で扱います。

## マニフェストとリース間の隔離

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

`source` は固定された管理対象ソースを識別します。`avd` は既存のローカル AVD テンプレートを選びます。Android ランタイムでは Compose ファイル、プロジェクトディレクトリ、サービス、接続先を指定できません。

Compose と Android は 1 つのスタックに共存できます。同じ Android ランタイムを共有するコンポーネントは、そのリースの Emulator を共有します。別のリースには常に異なる書き込み状態、AVD 識別情報、console/ADB ポート、シリアルを割り当てます。

Android ランタイム名は、すべての OS で大文字小文字を同一視しても区別できなければなりません。生成するランタイムディレクトリは、既存シンボリックリンクによる別名も含め、選択されたどのテンプレートや system-image のツリーとも重複してはいけません。

## 計画と起動の前提条件

`plan . --stack android-runtime --output json` は SDK の探索、予約、ランタイムの副作用なしに要件を示します。

`create` は予約前に、実行可能な SDK ツール、ホストのアクセラレーション、AVD を検証します。Android のみのスタックに Docker は不要です。ツールやテンプレートの欠如は前提条件エラー（終了 status 3）です。インストール済みイメージの ABI メタデータ（`source.properties`）は、アクセラレーション対応でネイティブホストと互換性があるイメージを識別しなければならず、テンプレートの ABI/CPU 設定も一致する必要があります。既存の共有 ADB サーバーは予約前に確認します。不在は許容し、後で起動します。

既定の起動待ち時間は 2 分です。Ready になるには、所有する端末の Android プロパティで起動完了を確認する必要があります。

ローカル ADB サーバーは選択 SDK と互換性がなければなりません。作成時に不在サーバーを Emulator のプロセス管理範囲とは別に起動します。不正または非互換の既存サーバーは、自動的に置き換えず拒否します。共有サーバーの診断情報を保持し、リースを destroy しても共有 SDK サービスは停止しません。

## 状態の観測とログ

`runtimes[].android` はテンプレート、SDK/system image、専用パス、一意の AVD 名、console/ADB ポート、シリアル、OS ネイティブのプロセス生成時識別情報、状態を記録します。

`doctor --runtime android-emulator` は SDK と AVD の前提条件を報告し、`doctor <lease-id>` は実際のリースの診断を報告します。リポジトリを指定した Android doctor は、宣言された各 Android ランタイムが選んだ AVD のテンプレート、イメージ、ホストの前提条件を検証します。

`logs <lease-id>` は active または released なリースの Emulator と共有 ADB の保持済み stdout/stderr を公開し、`--component` 選択と秘密情報の伏せ字処理に対応します。これはプロセスログであり logcat ではありません。

Show/list/reconcile は実際の端末を観測し、手動終了を確認すると active リースを degraded にします。

## 失敗時の処理と安全な削除

起動失敗は逆順で補償します。あるランタイムの削除が失敗した場合、その証拠と予約を保持しつつ、独立したランタイムの削除を続けます。全ランタイムの削除を確認するまでソースは残します。操作のキャンセルやレジストリの fence 喪失があれば、それ以降の副作用を止めます。

Destroy は一意の AVD 名を確認し、同じ認証済みコンソール接続で停止を要求してから、終了を確認した後に専用の書き込み状態を削除します。プロセス/AVD の識別情報が曖昧な場合、所有権の証拠が変わった場合、パスが危険な場合、削除が不完全な場合はリースを隔離し、予約を保持します。Force でもこの条件を迂回しません。Destroy を繰り返しても、解放済みポートの新しい利用者に影響しません。テンプレートと他のリースの状態は削除しません。

予約ポートを外部が使用していれば作成を失敗させて補償します。割り当てが黙って別ポートへ移ることはありません。グローバルな adb shutdown や Emulator 一括削除は禁止です。

## Emulator補助プロセスの専用状態

Emulatorの子プロセスには、リースのAndroid状態内の専用一時領域とnetsim探索パスを渡します。
これによりリース間の補助デーモンを隔離し、ホスト環境や共有ADBサーバー方針は変更しません。
各インスタンスは動的HCIポートを要求し、固定ホストポートを避けるためnetsimの補助Web UIを
無効にします。無線シミュレーションとゲストネットワークは有効のままです。
専用の補助プロセスも既存のネイティブ所有確認・クリーンアップ完了確認の対象であり、
生存するプロセスグループの識別が曖昧なら隔離を維持します。

## 検証記録

SDK イメージと互換性のあるホストアクセラレーションは外部の前提条件です。ネイティブの fake-adapter とプロセスのテストは Windows/macOS/Linux を対象とし、実 Emulator の証拠は [ExecPlan](../exec-plans/completed/android-emulator-lease.md) に別途記録します。SDK 不在やクロスビルドを実 Emulator の検証成功として報告しません。復旧と予約については[設計](../design-docs/android-emulator.ja.md)を参照してください。

## 実環境の統合検証

互換性のある SDK イメージをインストールし、停止済み AVD テンプレートを作成します。`AGENT_ENV_ANDROID_TEMPLATE` にその名前を設定し、必要に応じ SDK/AVD 探索用変数を設定してから、次のネイティブ Go コマンドを実行します。

```text
go test -tags=androidintegration ./internal/runtime/android -run ^TestRealAndroidEmulatorLeases$ -v -count=1
```

この明示実行のテストには、空いている予約対象 Emulator ポートと使用可能なアクセラレーションが必要です。前提条件が欠けていれば skip ではなく失敗します。実際の SQLite とアプリケーション層の制御処理、検証用のソースプロバイダーで 2 台の実 Emulator を起動し、兄弟の生存と手動終了を確認してから、所有を確認したリソースだけを削除します。削除が不確定なら、復旧用にテストが表示した一時状態パスを保持します。別途用意された Git/Compose の統合検証用フィクスチャを置き換えるものではありません。
