---
status: completed
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/completed/flutter-android-runtime.md
source_sha256: d6fc605c123e74e8a5c7b3d838c3482a9452a0d2c548a6324eabef593e486f5b
---

# 環境リース内にFlutter Androidアプリケーションを実体化する

[English（翻訳元）](flutter-android-runtime.md)

このExecPlanは作業に合わせて更新する文書です。`docs/PLANS.md` に従って管理します。
予定ブランチは `feat/flutter-android-runtime` です。

実装開始時に変更前の `master` の正確なリビジョンを記録します。
PR #2（`Add isolated Android Emulator resources to leases`）は必須の前提で、マージ済みです。
そこで実装したEmulator所有、プロセス、AVD、ポート、ADBサーバー、クリーンアップ、隔離の
規則を再実装したり弱めたりしません。

開始リビジョン: `5e5c8b19b5880bff8e3dd97d0947ef47ac9ca066`（masterと作業開始時HEAD）。

## 目的 / 全体像

固定GitソースからFlutter Androidアプリを実体化し、リース所有のAndroid Emulatorに
インストールします。選択Compose APIにリース所有の `adb reverse` で接続し、起動して、
正確なビルド・インストール証拠を記録します。次のスタックを記述できます。

    api       = Web APIのみ
    dashboard = Web API + Dashboard
    mobile    = Web API + Android Emulator + Flutterアプリ
    full      = Web API + Dashboard + Android Emulator + Flutterアプリ

利用者が確認できる流れ:

    agent-env plan . --stack mobile
    agent-env create . --stack mobile
    agent-env show <lease-id>
    agent-env test <lease-id> mobile-e2e
    agent-env doctor <lease-id>
    agent-env destroy <lease-id>

同時に存在する二つのmobileリースは、ワークツリー、Composeプロジェクト、Android書き込み状態、
Emulator識別情報、ホストエンドポイント割り当てを分離します。Emulatorごとにループバック空間が
独立しているため、アプリは同じデバイス側バックエンドポートを使えます。
UI階層、スクリーンショット、タップ、入力、探索操作は後続の `android-ui-observer` で扱います。

## 対象範囲

対象:

- Flutter SDK前提条件の探索と診断
- `.agent-env.yaml` の明示的Flutter Androidアプリ宣言
- リースの固定ソースワークツリーからのビルド
- シェル解釈のない、設定可能なargvビルドコマンド
- APKパス検証とSHA-256証拠
- Androidパッケージ・アクティビティの宣言と検証
- リース所有Emulatorへのインストールと選択Androidランタイムでの起動
- デバイスTCPポートから選択Composeエンドポイントへの `adb reverse`
- レジストリに保持するアプリ・ビルド・ネットワーク識別情報
- 必要に応じたplan/create/show/list/doctor/reconcile/logs/test/destroyの挙動
- ビルド・インストール・ネットワーク・起動失敗時の逆順補償
- 名前付きテストでの選択Androidシリアル補間
- Compose + Android + Flutter混在スタックと同時リース隔離
- Windows/macOS/Linuxのパス・argv・プロセス挙動とネイティブ偽アダプター検証
- 適切なSDKと高速化が利用可能な環境での実Flutter + Emulator結合証拠
- この機能の英日製品文書・設計文書

対象外:

- UIAutomator/Accessibility階層、スクリーンショット、録画
- 探索的tap/type/back/homeコマンド、観測ストリームとしてのlogcat
- iOS Simulator、実機Android、リモートAndroidホスト
- Android/Flutter SDK・システムイメージの自動インストール、SDKライセンス承諾
- ローカルOCIレジストリ、APK昇格・長期保持、リース消滅後の過去APKの厳密再実行
- 修正用ワークツリー、悪意あるコードの隔離、今回と無関係な汎用プラグイン基盤

## アーキテクチャ上の意図

既存の `android-emulator` はデバイス・ランタイムリソースです。Flutterを
`internal/runtime/android` に統合したり、ランタイム型を `flutter-android` に改名したりしません。
独立したアプリ宣言で、固定ソース、既存Androidランタイム、Flutterプロジェクト、argvビルドコマンド、
期待APK、パッケージと起動アクティビティ、0個以上のreverseを結び付けます。
設定・domain・appの調査後にフィールド名を調整しても、この所有境界を維持します。
境界変更が必要なら実装前に判断を記録し、ADRに昇格します。

当初のマニフェスト案:

```yaml
version: 1

sources:
  backend:
    repository: ../backend
    default_ref: HEAD

  mobile:
    repository: ../mobile
    default_ref: HEAD

runtimes:
  backend:
    type: compose
    source: backend
    project_directory: .
    files:
      - compose.yaml

  phone:
    type: android-emulator
    source: mobile
    avd: Pixel_API_35

applications:
  mobile-app:
    type: flutter-android
    source: mobile
    runtime: phone
    project_directory: .
    build:
      command:
        - flutter
        - build
        - apk
        - --debug
      artifact: build/app/outputs/flutter-apk/app-debug.apk
    package: com.example.app
    activity: .MainActivity
    reverse:
      - device_port: 8080
        endpoint: api.http

components:
  api:
    runtime: backend
    compose_services:
      - api
    endpoints:
      http:
        service: api
        target: 8080

  mobile:
    runtime: phone
    application: mobile-app
    depends_on:
      - api
    provides:
      - android-ui
      - mobile-app

stacks:
  api:
    roots:
      - api

  mobile:
    roots:
      - mobile

tests:
  mobile-e2e:
    stack: mobile
    source: mobile
    working_directory: .
    command:
      - flutter
      - test
      - integration_test
      - -d
      - ${android:phone:serial}
    timeout: 20m
    artifacts:
      - build/test-results
```

これは設計案であり、厳密検証を省略する許可ではありません。最終構文は文書化し、要件を満たさないマニフェストが拒否されることをテストで検証します。

## 進捗

- [x] 2026-09-08: masterと作業HEADが `5e5c8b19b5880bff8e3dd97d0947ef47ac9ca066`、ブランチが既に `feat/flutter-android-runtime`、未追跡差分が提供された計画だけであることを確認。
- [x] 2026-09-08: config/domain/app/runtime/store/CLIを調査。BuildPlan、ソース実体化境界、waitReady/probeReady、cleanup、Reconcile、名前付きテスト展開、CLI接続を拡張する。SQLiteはLease JSON全体を保存するため、追加アプリ記録にSQLマイグレーションは不要。
- [x] 2026-09-08: 公開CLI変更前に英日製品契約を書く。
- [x] 2026-09-08: 永続スキーマ変更前に英日設計を書く。
- [x] 2026-09-08: アプリ宣言の形を決定し文書化する。
- [x] 2026-09-08: 厳密パースと、不正なマニフェストを拒否するテストを追加する。
- [x] 2026-09-08: Flutter SDK・プロジェクトの前提条件診断を追加する。
- [x] 2026-09-08: 固定ソースからのビルドを実装する。
- [x] 2026-09-08: APKダイジェストとビルド識別・証拠を保存する。
- [x] 2026-09-08: 所有Emulatorシリアルでインストール・起動する。
- [x] 2026-09-08: エンドポイント解決とリース所有reverseを実装する。
- [x] 2026-09-08: create補償にアプリを統合する。
- [x] 2026-09-08: show/doctor/reconcileのアプリ観測を統合する。
- [x] 2026-09-08: Androidランタイム識別を名前付きテストで明示補間する。
- [x] 2026-09-08: api/dashboard/mobile/fullの混在fixtureを追加する。
- [x] 2026-09-08: 同時mobileリース二つの非衝突を証明する。
- [x] 2026-09-08: 失敗・復旧・隔離の検証を追加する。
- [x] 2026-09-08: 最終実装のGo 1.26.8全harnessが全工程成功、Go 1.27.1の `go test -race ./...` も成功。最新ソースCIは `81102f1`、検証済み文書CIは `d4d4289` で成功。
- [x] 2026-09-08: ネイティブCI `34166963420` は `8975096` で全OS・Go版ジョブ成功。実SDK証拠は別扱いでLinuxのみ。
- [x] 2026-09-08: Linuxで実Flutter + Emulator結合 `TestRealFlutterAndroidBackendLease` が248.46秒で成功。二つの実リースへの拡張もその後88.99秒で成功。Windows/macOSの実SDK実行は未検証。
- [x] 2026-09-08: 遅れて判明したCI失敗後の受け入れを再検証。テスト限定修正 `bb55393` は両方の全CIで成功。前回の証拠・振り返り・独立レビューも保持。
- [x] 2026-09-08: 修正CI成功後に両言語を再度completedへ移動。前回の移動・索引更新履歴も保持。

チェックは予定でなく観測済みの完了を示します。UTC日付、コマンド・テスト・実行識別子、結果を添えます。


## 想定外の発見

- 2026-09-08: 完了計画commit `233192d` の後、PR CI `34169150614` がUbuntu Go 1.27の
  `TestCleanupStopsIndependentEffectsAfterOperationLockLoss` で失敗したため再開（3.04秒）。
  `down:d-failing` だけを期待したが、`down:c-failing` も観測された。原因は未確定。
  ロック喪失fixture・ストア同期は仮説の一つであり、確認済みの発見ではない。
  記録時のpush CI `34169148759` は12ジョブ中11成功。
  コード調査前にactiveの実行判断基準を復元し、過去の成功証拠と前回の移動履歴を保持する。
  必要な修正の検証後だけ再検証・再移動する。


- 2026-09-08: 提供計画には日本語版がないため、マイルストーンのcommit前に追加・維持する。
  既存のAndroid予約と操作フェンスは所有規則を変えず利用できる。

OS別Flutter CLI差、Gradle/Java/SDKの設計への影響、追跡ソースへの書き込み、APK・パッケージ・
アクティビティ探索の制限、install/reverse/起動のcleanupへの影響、Composeエンドポイント解決の制限、
Flutter結合テストによるAPK置換、高速化不足、既存Android所有・ADB規則との衝突を少なくとも記録します。
予期しない制限を黙ったskipや所有確認の弱体化に変えません。

- 2026-09-08: 単体・raceテストで、同時mobileリースのソース/APKパス、ダイジェスト証拠、Composeプロジェクト、Android識別、reverseホストポートの分離を検証。失敗fixtureはbuild/install/launch、未確認マッピング所有、パッケージ・reverse欠落、強制cleanupを扱う。
- 2026-09-08: 独立レビューで初回ビルド隔離だけでは後の強制destroyが生存中かもしれないビルドのソースを削除できると判明。実行前の `build_unconfirmed` 永続化で、再起動後もcleanupを阻止する。復旧には終了証拠の調査が必要で、CLIはガードを黙って解除しない。
- 2026-09-08: reverse要求は所有の証明ではない。設定を確認できずマッピングが存在する場合は、cleanupで保持・隔離する。不在になれば続行できる。永続書き込みエラーではcleanupを停止する。

- 2026-09-08: 最終snapshotレビューで、起動前のクラッシュ後でも起動意図だけを根拠にreconcileがREADYにできる問題を発見。
  `launch_confirmed` は起動成功後だけ保存する。観測ではビルド終了確認と実行ファイル・プロジェクトディレクトリ一致も要求する。
  `TestMobileReconcileRejectsIncompleteApplicationSnapshot` が起動・実行ファイル・ディレクトリの不整合を検証する。
  このガードはその後 `e2c23f8` としてcommitされ、最終ローカルharness/raceで成功した。

- 2026-09-08: 二つの実リースのfixtureで両APKのビルドは成功したが、両Emulatorがuserdataパーティションの空き容量不足を報告。
  既定の一時ディレクトリはメモリー上にあり、テンプレートは12 GiBの空きを必要とする。
  通常の準備完了待ち失敗と補償の終了を待ち、十分な容量があるディスクをプロセス内だけの `TMPDIR` に指定して再実行する。
  テンプレート縮小や検査弱体化は行わなかった。その後ディスク上で補助隔離を行った二つのリース実行が88.99秒で成功。最終証拠を参照。

- 2026-09-08: 最終安全性レビューで、ビルド終了後にプロセス未確認ガードを解除すると、必須証拠の保存に失敗していてもストア復旧後のcleanupがソース/APKを削除できる問題を発見。
  この方式では不十分だった。実行終了と証拠保存は独立して扱う必要がある。
  永続的な `build_evidence_incomplete` をビルド前に設定し、必須の両ログ成果物と最終リース状態の保存後だけ解除する。
  `TestMobileIncompleteBuildEvidenceBlocksCleanup` は `SaveArtifact` 失敗を注入してAPK保持を検証する。
  このガードはその後、最終ローカルharness/raceで成功した。

- 2026-09-08: 最新の厳密な二つのdebug実リース実行では、両方がREADYになり、両方のFlutter HTTPマーカーと名前付きデバイステストが成功。
  その後、一方のDestroyはEmulatorリーダー終了後も生存するプロセスグループを検出して隔離した。
  他方が生存中のcleanup再試行は失敗したが、他方のcleanupは成功し、その終了後に両プロセスグループが空になった。
  Emulatorログに同じnetsim Wi-Fi localhostポート34339があり、共有 `netsimd` が最初のグループを生存させた可能性があるが、原因はまだ証明されていない。
  調査により下記の専用補助隔離を実装した。通常復旧はその後成功し、最終の二つのリース実行も88.99秒で成功。所有検査や隔離は弱めていない。

- 2026-09-08: 他方の終了後、最初のリースの通常Destroyが22:52:12 UTCに成功。
  両リースがRELEASEDとなり、対象を正確に絞ったCompose照会、ワークツリー、AVD状態、ADBデバイス、プロセスグループがすべて空であることを確認。
  元の150.28秒のdebug実実行は失敗のままであり、遡って成功とはしない。
- 2026-09-08: SDK 37.1.11 build 15917651 / netsimd 0.3.114の重点デーモンprobeで、リース専用netsim.ini、gRPCポート39107、HCI 0設定、libslirp有効を確認し、probeデーモンを停止した。
  探索理解に使った公開ソースがインストール版と完全一致するとは主張しない。
  実装は整合したリース専用探索ディレクトリと動的HCI設定を渡すようになった。その後、全ローカルharness/raceと二つの実リース再実行が成功。最終証拠を参照。

- 2026-09-08: `TestLockLossFixtureWaitsForSQLiteWriter` でテストのロック喪失注入器の欠陥を再現。
  実SQLite writerを保持すると旧helperはトークン置換前に0.063秒で `database is locked (5) (SQLITE_BUSY)` に失敗した。
  元テストは50回反復で成功（30.267秒）し、過去CIにそのSQLiteエラー自体の直接ログはない。
  これは再現したfixture欠陥であり、CIにも整合する有力な仕組みだが、過去失敗の厳密な原因を証明したとはしない。
  helperだけを `SetMaxOpenConns(1)` と `PRAGMA busy_timeout=10000` に変更して既存ストア方針と揃え、
  エラーは即座に `t.Errorf` で報告し、影響行数が正確に1であることを要求する。
  production fencingと元の後続作用がないことの厳密assertionは不変。
  実writer保持中のロック喪失注入を確認するテストと既存cleanupロック喪失テストは `-race -count=30` で成功（33.834秒）。
  Go 1.27全harnessも成功（app 5.794秒）、全 `go test -race ./...` も成功（app 25.304秒）。
  このテストの50ms待ちは保持writerの解放を有限時間後に行うためで、短い成功期限ではない。
  修正ソースのCIはその後 `bb55393` で成功し、両計画を再移動した。

## 判断の記録

- 決定: `applications` と `component.application` を使い、既存Lease JSONに識別情報を追加する。
  選択アプリはソース実体化直後、すべてのランタイムUp前にビルドする。
  理由: リソース境界と旧行・マニフェスト互換性を保ち、ビルド失敗時の高コスト確保を避ける。
  日付・担当: 2026-09-08 / implementation。
- 決定: Flutterアプリのライフサイクルを既存Androidプロバイダーから分離する。
  理由: PR #2はEmulator単独利用とFlutter非依存を意図し、アプリ処理で所有やcleanupを弱められない。
  日付・担当: 2026-09-08 / maintainers。
- 決定: 固定された管理ソースと同じリース内の選択Androidランタイムを必須とする。
  理由: 出自と所有を明示し、任意の外部Emulatorに接続させない。
  日付・担当: 2026-09-08 / maintainers。
- 決定: リース別ホストポートで再ビルドせず、解決TCPエンドポイントへ明示的所有reverseで接続する。
  理由: デバイスURLを一定にし、ホスト割り当てをComposeプロジェクト別に隔離できる。
  日付・担当: 2026-09-08 / maintainers。
- 決定: ビルドはシェル評価のないargv直接実行とする。
  理由: 3 OSの移植性と既存の実行安全境界を維持する。
  日付・担当: 2026-09-08 / maintainers。
- 決定: create時に実際にインストールしたAPKのSHA-256と証拠を保持し、長期昇格は追加しない。
  理由: レジストリ・保持基盤を早まって導入せず監査可能性を得る。
  日付・担当: 2026-09-08 / maintainers。
- 決定: 長期維持する製品・設計文書は英日で提供する。
  理由: 今後リポジトリ文書を二言語で維持するため。
  日付・担当: 2026-09-08 / maintainers。

- 決定: 宣言パッケージが既に存在する場合はAPKインストール前に拒否し、インストール後にそのパッケージを必須とする。
  理由: テンプレートのシステムイメージ由来のパッケージを無関係なAPKダイジェストに結び付けないため。
  日付・担当: 2026-09-08 / 独立レビューを受けたimplementation。
- 決定: reverse参照先はループバックTCPホストに限定し、リモートDockerのホストアドレスを捨てて続行せず拒否する。activityでは `$` 入れ子クラスを含むシェル特殊文字を除外する。
  日付・担当: 2026-09-08 / implementation。
- 決定: リポジトリのFlutter doctorは現在の宣言ソースcheckoutと全アプリを検査し、ワークツリー確保やDockerを必要としない。createは固定プロジェクトと選択依存先を別途検査する。
  日付・担当: 2026-09-08 / implementation。

- 決定: 起動成功と起動意図を別に保存し、アプリ観測で実行識別を記録ビルドに照合する。
  理由: パッケージ存在と起動意図だけでは、クラッシュ後に初回ライフサイクルが完了した証拠にならない。
  前面状態は引き続き意図的に要求しない。
  日付・担当: 2026-09-08 / snapshotレビューを受けたimplementation。

- 決定: 証拠保存の未完了状態をプロセス未確認状態とは独立して永続化する。
  理由: 必須ビルドログやリース最終状態を保存できなかった場合、プロセス終了成功だけで削除を許可できない。
  ストア復旧後も、証拠の失敗を調査・復旧するまで、reconcileは隔離し、通常・強制cleanupはソース/APKを保持する。
  日付・担当: 2026-09-08 / 最終安全性レビューを受けたimplementation。

- 決定: 残る当初の契約課題を明示的に解決する。APK推測でなくpackage/activityを必須とし、ビルドargv先頭でPATHまたは明示パスのFlutterと版探索を選ぶ。シリアル補間はAndroid専用、変更されるキャッシュ情報は証拠対象外、実Flutter/Emulator検証は高速化ホストの明示選択とする。
  理由: 厳密で移植可能な挙動を保ち、追加APKツールや時期尚早な汎用基盤を避け、出自と実SDK実行を未証明の再現性・OS対応の主張から区別する。
  既存のapplications/Lease JSON判断と合わせ、当初のマイルストーン1の7課題をすべて解決した。
  日付・担当: 2026-09-08 / 実装契約の照合。

- 決定: 汎用Android Emulatorの補助探索を子プロセスだけの `TMPDIR`/`TMP`/`TEMP`/`XDG_RUNTIME_DIR` で `AVDHome/emulator-data/Temp` に隔離する。
  Windows子の `LOCALAPPDATA` は `AVDHome/emulator-data` にし、`NETSIM_INSTANCE=1`、`NETSIM_HCI_PORT=0`、`-netsim-args --no-web-ui` を設定する。
  理由: クライアントとデーモンの専用探索先・既定インスタンスを一致させ、HCIと補助Web UIの固定ポート（8080等）を避ける。
  無線・ゲストネットワーク、共有ADB方針、プロセス所有・cleanupの証明は変更しない。
  インストールSDKの重点probeで設定を裏付けたが、その後88.99秒の二つの実リース実行で、他方を生存させる全ライフサイクルを証明した。
  日付・担当: 2026-09-08 / SDK調査を受けたimplementation。

- 決定: テストの外部ロック喪失注入器だけを直し、実SQLite writerを待ってトークンを1行置換したことを証明してから後続fencingを検証する。注入エラーは発生箇所で報告する。
  理由: 未設定のraw接続はロック喪失を注入する前に失敗し、その後の作用を誤解させる失敗につながり得る。
  既存ストアのbusy処理に合わせ、production fencingや正確な作用順序のassertionは変更しない。
  日付・担当: 2026-09-08 / 遅れて判明したCIのfixture再現を受けたimplementation。

## 成果と振り返り

`233192d` の完了計画CI失敗を受けた再開から、再度完了した。テスト限定のロック喪失注入器修正
`bb55393` はローカル全harness/raceと両方の全CIで成功し、2026-09-08に再移動した。
fixture欠陥は再現したが、過去CIの厳密なSQLiteエラーはログがなく未証明のまま。
production fencingと後続作用がないことの厳密assertionは保持した。


提供成果: `applications` により厳密な固定ソースFlutterビルドを独立所有のAndroidランタイムに結び付ける。
高コストなランタイム作成前にビルドし、install、正確なパッケージ確認、loopback reverse、起動成功確認の後にREADYとする。
既存Lease JSONにソース・ビルド・APK・ネットワーク識別と、プロセス・証拠・起動の独立した確認を保持する。
名前付きテストは観測した所有シリアルを使い、テストAPKの再ビルド可能性とcreateの出自を明確に区別する。
実debugリース二つで別々のゲスト接続、専用補助プロセス寿命、他方を保つforceなしcleanupを証明した。
UI観測、長期APK昇格、Windows/macOSの実SDK検証は後続作業。

振り返り: 注入テスト成功では実netsimd共有の寿命結合を検出できなかった。
厳密なプロセスグループcleanupは正しく隔離し、補助探索とポート所有を直すことで検査を維持できた。
クラッシュや後の強制cleanupに備え、プロセス終了、必須証拠保存、起動成功を独立して永続確認する必要があった。
プロセス終了の未確認、証拠保存の失敗、起動の失敗をそれぞれ注入するテストで、これらの区別を検証する。偽準備完了の時間依存失敗は、待機キャンセル検証を残して決定的な発火条件に直した。
一時容量は基盤の前提であり、テンプレート縮小や検査弱体化の理由ではない。ローカル最終race/harnessと最新ソースのネイティブCIは成功。過去失敗と実SDKの残るOS差を保持し、両言語を完了計画として保存した。


2026-09-08に完了。契約（`4457dfd`）、初回ライフサイクル（`8975096`）、復旧ガード（`e2c23f8`）、
診断・テスト修正（`aee0982`）、最終の専用補助隔離と二つのリースfixture（`81102f1`）をPR #4で提供した。
検証済み二言語文書のチェックポイントは `d4d4289`。F1〜F21すべての直接証拠を下記に記録し、
Linuxの二つの実リース成功と全ネイティブCIを含む。人間によるPR承認・mergeは実装・計画完了とは別。
Windows/macOSの実Flutter/Emulator実行は、F20/F21で認める明示的なOS検証差として残す。

## 背景と構成

実装前に読む文書:

- `AGENTS.md`、`ARCHITECTURE.md`、`docs/PLANS.md`
- `docs/PORTABILITY.md`、`docs/RELIABILITY.md`、`docs/SECURITY.md`、`docs/QUALITY.md`、`docs/roadmap.md`
- `docs/product-specs/agent-env-mvp.md`、`docs/product-specs/android-emulator.md`
- `docs/design-docs/android-emulator.md`
- `docs/exec-plans/completed/agent-env-mvp.md`、`docs/exec-plans/completed/android-emulator-lease.md`

現在の責務:

- config: マニフェストの厳密デコード。stack: ルートcomponentとその直接・間接の依存先すべてを、同じ入力なら同じ順序で選択。
- domain: 具体アダプターに依存しないリース・ソース・コンポーネント・ランタイム・リソース・イベント状態。
- app: 順序、準備完了判定、補償、証拠、reconcile方針。
- runtime/compose: Composeの具体操作。runtime/android: Emulatorプロセス・AVD・ポート・デバイス操作。Flutterに依存させない。
- execx: 移植可能なコマンド・プロセス境界。
- store/sqlite: 期待状態、リソース識別、予約、イベント、実行、成果物。
- evidence: 秘匿処理とダイジェスト。paths: ソース内へのパス制限。
- CLI: 入出力整形・解析。ライフサイクル方針は持たない。

現行名前付きテストの補間は `${lease_id}` と `${env:NAME}` だけです。
Flutter結合には `${android:phone:serial}` のような安定したランタイム指定を加え、
環境変数ANDROID_SERIAL、暗黙のadb devices選択、シェル置換に依存させません。
現Android契約はFlutterビルド、APKインストール、reverse、UI、スクリーンショット、logcatを
意図的に除外しています。本計画はその安定境界を利用し、所有保証は変更しません。

## 作業計画

### マイルストーン1 — 契約とアプリモデル

現マニフェスト、domainのランタイム・コンポーネント、実行snapshot、SQLiteを調査します。
製品と設計の `flutter-android-runtime.md` / `flutter-android-runtime.ja.md` を作成し、
英日索引を更新します。全体移行が未マージでも新規文書の両言語は提供し、無関係な翻訳は重複しません。
最終名をapplications/workloads等から決め、次を維持します。

1. アプリはEmulatorランタイムとは別。
2. ソースは固定されリースが実体化する。
3. 明示選択した所有Emulatorを対象にする。
4. 設定は厳密かつ移植可能。
5. 未使用なら旧マニフェストは有効。
6. アプリ識別は永続化され観測できる。

新しい依存ノードを追加する場合、ARCHITECTUREと日本語版、arch-check、ADRを同時に更新します。
`agent-env plan . --stack mobile --output json` はアプリ、ソース、対象ランタイム、APKパス、
reverse要件を表示し、ビルド・ポート確保・Docker/Emulator起動・SDK変更を行いません。

### マイルストーン2 — Flutter前提条件の探索

注入可能でテストできるCLI探索を実装します。インストールやツール変更をせず、実行可能性、
解析可能なバージョンと証拠、固定ソース内のプロジェクト存在、Androidビルドに必要なメタデータ、
制限された相対APKパス、package/activity構文、Androidランタイム、選択依存先エンドポイント、
TCPおよびポート範囲・重複を検査します。通常診断でSDK取得やホスト変更を伴うflutter doctorを使いません。
`agent-env doctor <repository> --runtime flutter-android` または最終的な文書化済み同等コマンドが、
環境を確保せず前提条件を報告します。

### マイルストーン3 — ビルドとAPKの出自

固定ワークツリー作成後、可能なら高コスト確保前にビルドします。コマンドはargvで、指定プロジェクト内で
文書化した環境だけを受け取り、execxで時間制限・キャンセル可能にし、パス逸脱を許さず、stdout/stderrを
証拠化します。可能なら失敗時に無関係な高コストリソースを起動しません。
成功後はAPK存在、シンボリックリンク・逸脱拒否、通常ファイル、SHA-256を検査し、コマンド、
ソースコミット、Flutter版、パス、ダイジェストを保存します。長期昇格とは区別し、ハッシュだけで
再現可能ビルドと主張しません。show JSONから「どの固定ソースとAPKをインストールしたか」を分かるようにします。

### マイルストーン4 — インストール、reverse、起動

AndroidがREADYになったら、正確な所有シリアルに記録APKをインストールし、パッケージ存在を確認し、
実Compose観測から依存エンドポイントを解決し、そのシリアルへreverseを設定・検証し、明示アクティビティを
起動します。すべて成功した後だけアプリをREADYにします。adb devicesの先頭や外部シリアルは使いません。

    emulator-5554 localhost:8080 -> host 127.0.0.1:49173
    emulator-5556 localhost:8080 -> host 127.0.0.1:49218

このように同じデバイスURLでホストポートを分離します。reverseは既存Androidの互換ローカルADBサーバー
方針を使い、第二の独立ADBデーモン方針を導入しません。

### マイルストーン5 — 補償、destroy、reconcile

create sagaにアプリ操作を加えます。失敗時は所有を確認したreverseを除去し、ビルド証拠を保持し、
既存ランタイムを逆順にcleanupし、確認後だけソースをcleanupします。専用Emulator書き込み状態の
破棄でアプリも消えるため、外部に状態が残らなければグローバルなアンインストール工程は不要です。
reconcileはAndroid識別、パッケージ存在、reverseの記録先一致、レジストリ内のAPK・ビルド整合性を調べます。
アプリの前面・実行継続は要求しません。手動アンインストールや必須reverse欠落はDEGRADED、
識別の曖昧さは未証明デバイスの修復・killでなく隔離にします。

### マイルストーン6 — 名前付きテストとFlutter結合

`${android:<runtime-name>:serial}` を追加し、たとえば
`[flutter, test, integration_test, -d, '${android:phone:serial}']` で所有デバイスを選べるようにします。
固定snapshotと観測した所有ランタイムから解決し、不明・不在・非Androidを拒否し、環境の暗黙選択を避け、
既存のlease_id/env補間を保ちます。integration_testは別テストAPKを再ビルド・再インストールすることがあるため、
createのAPKを実行したと主張しません。製品・設計とテスト出力に区別を記録します。
厳密なcreate APKのブラックボックス操作は、証明可能な手段がなければUI observerの後続範囲です。

### マイルストーン7 — スタックfixtureと並行実行

api/dashboard/mobile/fullで、意図したルートcomponentとその直接・間接の依存先すべてが選ばれることを検証します。apiとdashboardは明示要求がなければAndroid/Flutterを
確保せず、mobileはAPI + Emulator + Flutter、fullはさらにDashboardを選びます。
同時mobile二つのワークツリー、Compose、ホストエンドポイント、AVD状態、シリアル、APKビルド記録を分離し、
同じデバイスreverseポートを利用可能にし、一方の破棄後も他方がREADYで利用可能と証明します。

### マイルストーン8 — ネイティブ検証と実結合証拠

偽・注入アダプターとパス・argvのテストは3 OSのネイティブCIで実行します。
`go run ./tools/repoctl check` と文書化済みGo race検査を実行します。
明示選択する実結合fixtureでは、必要に応じた実Gitワークツリー、実Composeバックエンド、
Flutterビルド、Emulatorリース、APKインストール、reverse、起動、最終cleanupを使います。
明示選択後の前提条件不足はskipでなく失敗にします。実Windows/macOSの適切なランナーがなければ
未解決の実行差として記録し、クロスビルドや偽テストを実SDK検証と報告しません。

## 具体的な手順

1. `git switch master && git pull --ff-only`。
2. 正確な開始リビジョンを記録する。
3. `feat/flutter-android-runtime` を作成し切り替える。
4. harnessと上記の関連文書を読む。
5. config/domain/app/runtime/storeのスキーマを調査する。
6. 英日製品・設計を作る。
7. 最終モデル判断と必要なADRを記録する。
8. 厳密パースと、不正なマニフェストを拒否するテストを加える。
9. 永続フィールド決定後、必要な場合だけマイグレーションを加える。
10. 注入インターフェースでFlutterコマンド・探索を加える。
11. ビルドと証拠を実装する。
12. 既存Android所有モデルでinstall/launch/reverseを実装する。
13. create補償とdestroy/reconcileに統合する。
14. 明示シリアル補間を拡張する。
15. スタック解決と同時リースのテストを加える。
16. 実結合fixtureを加える。
17. 重点テスト、全harness、raceを繰り返し実行する。
18. 検証済みのまとまったcommitを履歴を書き換えずpushする。
19. 全受け入れ項目に直接証拠を記録する。
20. 成果と振り返りを完成する。
21. completedへ移動しリンクを更新する。

## 検証と受け入れ

| ID | 必須の挙動 | 証拠 |
| --- | --- | --- |
| F1 | 旧Composeのみ・Androidのみのマニフェストは有効で挙動不変。 | `8975096` のGo 1.26.8全検査で成功: `TestStrictManifest`、`TestLifecycleCreatePersistedIntentAndUniqueIsolation`、`TestAndroidConcurrentLeasesAndSiblingCleanup`。 |
| F2 | 不正なアプリ設定を外部作用前の厳密検証で拒否。 | 同全検査で成功: `TestFlutterManifestContract`、`TestFlutterApplicationCollisionsAndDependencyClosure`、`TestApplicationEndpointReferenceWithDots`。 |
| F3 | mobile planがビルド・ランタイム操作なしで要件を表示。 | 同全検査で成功: `TestFlutterPlanJSONIsPureAndIncludesApplication`、`TestMobilePlanStackClosure`。 |
| F4 | Flutter実行ファイル・プロジェクト不足を前提失敗としリソースを漏らさない。 | 同全検査で成功: `TestMobilePrerequisiteNoReservation`、`TestFlutterDoctorMissingExecutableDoesNotRequireDockerOrAllocateState`、`TestInvalidPrerequisitesDoNotRunCommands`。 |
| F5 | 固定ソースでargvビルドしコミット・Flutter版・ログ・APK SHA-256を記録。 | 同全検査で成功: `TestBuildArgvDirectoryAndEvidence`、`TestMobileEvidenceDoesNotKeepRawBuildOutputInLease`、`TestNativeBuildPreservesArgvAndProjectDirectory`。下記の実Linux実行も成功。 |
| F6 | 記録APKだけを選択した所有Emulatorシリアルへインストール。 | 同全検査で成功: `TestApplicationOperationsUseOwnedSerialAndLocalServer`、`TestAPKInstallRequiresExplicitSuccess`、`TestMobileConcurrentLeasesAndObservation`。実Linuxインストールも成功。 |
| F7 | package/activity起動成功、または安全なcreate補償。 | 同全検査で成功: `TestActivityLaunchReportsExitZeroFailures`、`TestMobileFailureCompensation`。実Linux起動も成功。 |
| F8 | 宣言デバイスTCPを実選択Compose先へreverseし記録。 | 同全検査で成功: `TestApplicationOperationsUseOwnedSerialAndLocalServer`、`TestReverseEndpointRejectsRemoteAndMalformed`。実Dart HTTPがComposeバックエンドに到達。 |
| F9 | 二つのmobileは同じデバイスポートを使いホストとEmulatorを隔離。 | 注入アダプターの並行・raceと、実debugリース二つ（88.99秒）で成功。デバイス・バックエンド・reverseを隔離し、一方の通常Destroy後も他方READYと新しいゲストHTTPを確認。詳細は下記。 |
| F10 | reverse/install/launch失敗後に非所有操作を残さず証拠保持。 | 同全検査で成功: `TestMobileFailureCompensation`、`TestMobileUnconfirmedReversePreservesMapping`、`TestMobileUnconfirmedBuildBlocksLaterForcedCleanup`、`TestReverseCleanupRequiresExactMapping`。 |
| F11 | 手動パッケージ除去・reverse欠落をDEGRADEDとして観測。 | 同全検査の `TestMobileConcurrentLeasesAndObservation` がパッケージとマッピングを明示除去し、それぞれDEGRADEDとなることを確認。 |
| F12 | reconcileが未証明の外部デバイスに接続・killしない。 | 同全検査で成功: `TestMobileUnknownIdentityAndCleanupPreservation`、`TestApplicationOperationsRefuseOwnershipOrServerMismatch`、`TestApplicationChecksServerAgainAfterObservation`。 |
| F13 | 一方破棄後も他方がREADYで利用可能。 | 注入アダプターの並行・raceと、実debugリース二つ（88.99秒）で成功。デバイス・バックエンド・reverseを隔離し、一方の通常Destroy後も他方READYと新しいゲストHTTPを確認。詳細は下記。 |
| F14 | Androidシリアル補間は所有する選択Androidだけを解決し暗黙選択しない。 | 同全検査と反復重点実行で成功: `TestAndroidSerialInterpolation`、`TestMobileConcurrentNamedTestsUseOwnedSerialAndPersistWarning`。 |
| F15 | テスト証拠が再ビルド・再インストールとcreate APKを区別。 | 同全検査と反復重点実行で成功: `TestMobileConcurrentNamedTestsUseOwnedSerialAndPersistWarning`、`TestMobileFailedNamedTestRetainsEvidenceAndReadyLease`。 |
| F16 | 4スタックが文書化した最小のコンポーネント集合へ解決。 | 同全検査の `TestMobilePlanStackClosure` が4スタックを検証し成功。 |
| F17 | 新規製品・設計文書に英日版と索引がある。 | `8975096` のGo 1.26.8全検査内のdocs-checkで成功。英日製品・設計文書と両索引あり。 |
| F18 | 新しい責務境界追加後のarchitecture/docs検査が成功。 | 同全検査のarch-checkとdocs-checkが成功。`TestArchitectureBoundaries` にFlutterの禁止されたアダプター間依存を明示的に与えて検出するテストを追加。 |
| F19 | 最終実装の全harnessとraceが成功。 | 最終ローカル実装で成功: Go 1.26.8全harness全工程とGo 1.27.1全 `go test -race ./...`。最新実装CIは `81102f1`、検証済み文書CIは `d4d4289` で成功。過去失敗は下記に保持。 |
| F20 | 3 OSネイティブ証拠を実Flutter + Emulatorと区別して正確に記録。 | 区別して記録し成功: Linux実SDK、最終実装 `81102f1` と文書 `d4d4289` の6 OS/Goネイティブ・5クロスビルド・結合CI。Windows/macOS実SDKは未検証。 |
| F21 | 利用可能なら少なくとも1回の実Flutter + Emulator + backend結合が成功、または不足基盤を偽証拠で代用せず明記。 | `TestRealFlutterAndroidBackendLease` 成功: 初回1リース248.46秒、最新debug2リース88.99秒。他方の新しいゲストHTTPと両方の通常cleanupを含む。ツール・ソース・APK証拠は下記。 |

すべてに直接証拠が必要です。成功した実行を記録しないテスト名だけでは証拠になりません。

## 冪等性と復旧

planと前提検査は読み取り専用です。ビルド失敗で使い捨てワークツリーに未追跡出力が残っても、
既存のtracked-dirty・所有規則に従います。出力除去のために隔離を弱めません。
ビルド後の外部作用は後続が依存する前に永続識別情報を持ちます。
アプリcleanupが除去できるのは、記録シリアル・リースの所有を証明したreverse、
リース成果物・状態ディレクトリ内のアプリ証拠、所有AVD破棄で間接的に消える専用アプリ状態だけです。

グローバルにADB kill-server、SDK/Flutter共有キャッシュ削除、ユーザーAVDテンプレート削除、
無関係デバイスのuninstall、任意シリアルのclear、現在の空き具合だけによるポート・リソース除去を行いません。
反復destroy/reconcileを安全にし、再利用ホストポート・シリアル・PID・AVD名・別デバイスパッケージを
元リースの所有物と誤認しません。曖昧なら予約と証拠を保持して隔離し、forceでも所有証明を省略しません。

## 成果物と注記

再完了の証拠（2026-09-08）:

- テスト限定修正 `bb55393` の[push CI 34169765493](https://github.com/mahcialet/agent-env/actions/runs/34169765493)と
  [PR CI 34169767777](https://github.com/mahcialet/agent-env/actions/runs/34169767777)は両方とも全12ジョブ成功。
  下記のローカル全harness/raceと独立レビューと合わせ、`233192d` を契機とする再開を完了した。
- appのharness所要時間を5.794秒に訂正し、過去失敗とその厳密なSQLiteエラーに関する未確定事項を保持して、両計画を再度完了した。
  production fencingや元の厳密な作用assertionは変更していない。全受け入れ要件を満たしている。


- 最終自己点検で、writer保持テストの接続にもBegin前の `SetMaxOpenConns(1)` と `PRAGMA busy_timeout=10000` を設定し、競合準備自体がレジストリ更新と競わないようにした。
  productionと正確なassertionは不変。先行の全harness/raceは注入器修正を検証し、この小さな準備変更は重点race30反復で成功（33.561秒）。


再開後の修正チェックポイント（2026-09-08）:

- 完了計画commit `233192d` ではpush CI `34169148759` が成功し、PR CI `34169150614` はロック喪失テストで失敗。両結果を保持する。
- 再現したfixture修正は重点 `-race -count=30`（33.834秒）、Go 1.27全harness（app 5.794秒）、
  Go 1.27全 `go test -race ./...`（app 25.304秒）で成功。
- 別の読み取り専用レビュアーが実際のテスト限定差分を確認し、具体的問題なし。
  production fencingと正確なassertionは不変で、このテストのgoroutine合流・リソースcleanupは安全。
  50ms遅延は保持writerを解放するためであり、短い成功期限を設けるものではない。
- 過去CIにSQLite失敗そのもののログはない。writer保持テストはfixture欠陥を再現し、過去のエラーメッセージを再現したとはしない。
  その後、検証済み `bb55393` で修正ソースCIと再移動を完了した。


最終完了証拠（2026-09-08）:

- 最終実装 `81102f1` の[push CI 34168707969](https://github.com/mahcialet/agent-env/actions/runs/34168707969)と
  [PR CI 34168710704](https://github.com/mahcialet/agent-env/actions/runs/34168710704)は両方成功。
  検証済み文書 `d4d4289` の[push CI 34168751870](https://github.com/mahcialet/agent-env/actions/runs/34168751870)は
  [PR CI 34168753637](https://github.com/mahcialet/agent-env/actions/runs/34168753637)とともに
  全12ジョブ成功（Windows/macOS/Linux × Go 1.26/1.27の6ネイティブ、5クロスビルド、結合）。
- production namespace変更の実装者とは別の読み取り専用レビュアーが `81102f1` を確認。
  lifecycle/layout/所有マーカーとcleanup全体、Windows大小文字・Job検査を含むdetached環境、単体検証の対応、文書を調査し、具体的指摘なし。
  この独立レビューとネイティブCIはWindows/macOSの実Flutter/Emulator実行を代替しない。
- 全受け入れ項目は下記の検証範囲を明示した証拠で充足。両計画をcompletedへ移動し索引を更新した。
  過去の検査失敗、実実行失敗、実SDKのOS検証差は保持する。


- 最終実装と二つのリースfixtureを `81102f1` としてcommitし、最新ソースCIのためpushした。
  下記のローカルharness/raceと実実行成功はこの実装を検証する。このcommitのCIはその後成功した。


- 最終実装のローカル検証が成功。Go 1.27.1の `go test -race ./...`（app 22.990秒、Android 2.145秒、CLI 1.769秒、SQLite 11.985秒）と、Go 1.26.8の全harness全工程（Android 0.322秒、CLI 0.228秒、SQLite 5.810秒）が成功。
  最新実装CIは `81102f1`、検証済み文書CIは `d4d4289` で成功。


二つの実リースの成功証拠（2026-09-08）:

- `TestRealFlutterAndroidBackendLease` が同時debug APKリース二つで88.99秒で成功。
  Linux amd64、Go 1.26.8、Flutter 3.47.2、Dart 3.13.2、JBR Java 25、Gradle 9.3.1、
  NDK 28.2.13676358を使用。生成した固定ソースは `58f38ffb6598bd08d890cf742119c74654311457`。
- `emulator-5554` はバックエンドホストポート32832、APK SHA-256
  `adfc18293649617d5c969697811ce194974a396671813533560e021238564f62` を使用。
  `emulator-5556` は32833と `8bc3a8b1c27005b5097a63afb258a3491dba9e8a1aad2bff36d458e2cf835bef` を使用。
  両方のデバイスポートは8080、隔離したnetsimリスナーは43277と41769。
- 両方がREADYとなり、専用バックエンドへのFlutter HTTPと名前付きの対象限定ADBテストに成功。
  最初の通常Destroy後、他方ゲストの新しいHTTPリクエストで観測基準値が増え、過去リクエストや
  ホストだけの確認ではなく、存続するゲスト接続を証明した。両方の通常Destroyが成功しfixtureルートも削除。
- 22:59:38 UTCにADBと両方の対象限定Composeプロジェクトが空であることを確認。
  二つのリースの受け入れ不足は解消したが、下記の過去失敗履歴は残す。
  最新ソースの全raceとGo 1.26.8全harnessはその後成功。その後、最新実装と検証済み文書のネイティブCIが成功し、移動条件を満たした。


- `aee0982` のpush CI `34168074785` とPR CI `34168077225` は全ジョブ成功（6ネイティブOS/Go、5クロスビルド、結合）。
  汎用Android補助隔離変更より前のため、その後の実装を検証したとは扱わない。
- 補助隔離後のローカルGo 1.27全harnessは全工程成功。その後の二つの実リース再実行は88.99秒で成功。
  インストールSDKの証拠はLinuxのみ。Windows/macOSのネイティブ偽・プロセスCIやクロスビルドは、同OSでの実Flutter/Emulator挙動の証明ではない。


- 診断・テスト・計画のチェックポイント `aee0982` をpush済み。この時点では二つのリースfixtureの製品説明は成功待ちの未commit変更だった。その後の88.99秒実行で挙動を検証した。
  最新の実実行では両方の厳密debug起動、Flutter HTTPマーカー、名前付きデバイステストを証明したが、他方を生存させたままの最終cleanupは成功していない。
  上記のプロセスグループの発見を参照。この実行を二つのリースの結合全体の成功と数えない。


- `e2c23f8` のPR CI `34167569456` は全ジョブ成功。同じcommitのpush CI `34167566940` は下記の時間依存ケースで失敗した。両方を保持し、他方の成功で失敗を取り消さない。
- Android診断・準備完了の重点テストは `-race -count=30` で成功（1.502秒）。
  独立レビューを受け待機ループのキャンセルも明示検証する。第三の `canceled_while_waiting` ケースはStart後の最初の未準備probeでキャンセルし、`(false, nil)` を返してselectのキャンセル分岐を通す。反復検証は全3準備完了ケースと診断を `-race -count=30` で実行し成功（1.759秒）。arch-checkも成功。その後の二つのdebug実リース実行は88.99秒で成功。上記の成功証拠を参照。


最新検証チェックポイント（2026-09-08）:

- 実Emulatorのcleanup後、Go 1.27の全harnessを逐次実行して成功。未確認・証拠・起動ガードを `e2c23f8` としてcommit/pushした。
- push CI `34167566940` はUbuntu Go 1.27の既存 `TestSharedADBReadinessFailureNeverLaunchesEmulator/unavailable` で失敗。
  100ms期限が偽共有サーバーの開始前に切れた（`starts=[]`）ためで、Emulatorの予期しない起動ではない。
  決定的な修正としてStart時にキャンセルし、Start後だけProbeADBに利用不能エラーを注入する。
  正確な起動列とEmulator未起動のassertionは維持する。反復race検証はその後全3ケースで成功（`-race -count=30`、1.759秒）。
  この修正や後続診断変更が先行CIの成功で検証されたとは扱わない。
- install/launch失敗診断は長さを制限し秘匿したstdout/stderrを保持するようになった。
  起動には引き続き明示の `Status: ok` が必要。先行するディスク上の二つの実リース実行はactivity状態が不明だった。
  cleanup再試行でRELEASEDを確認した。その後の補助隔離で二つの実リース実行が88.99秒で成功。


証拠チェックポイント（2026-09-08、`8975096` 後）:

- Go 1.26.8の `go run ./tools/repoctl check` は `8975096` で成功。
  Go 1.27.1の `go test -race ./...` は先行実行で成功し、その後の重点race・復旧検査も成功。
  これは実装チェックポイントの証拠であり、その後のfixture拡張の検証完了を意味しない。
- 後のGo 1.27.1全検査は実Emulator結合と重なり、既存の
  `TestRealAdapterThroughAppPersistsOwnedResource` が予約する5554ポートを実結合が使っていたため失敗。
  テストは弱めていない。実リソースcleanup後のGo 1.27全harness逐次再実行はその後成功。
- [ネイティブCI `34166963420`](https://github.com/mahcialet/agent-env/actions/runs/34166963420)
  は `8975096` で全体成功。3 OS × Go 1.26/1.27の全6ネイティブジョブ、5 OS/アーキテクチャの
  クロスビルド、結合（raceと実Compose）が成功。Windows/macOSの実Flutter/Emulatorは未検証。
- ローカルGo 1.27.1の `go run ./tools/repoctl test-integration` はその後終了コード0で成功。
  その後 `e2c23f8` にcommitした後続変更で、Reconcileは期待状態にかかわらずビルド未確認の隔離を保持する。
  不確定なbuildの隔離維持を確認する重点テスト `TestMobileUnconfirmedBuildRemainsQuarantinedOnObservation` は成功。
  `8975096` のネイティブCIは、この後続変更の検証を意味しない。
- `go test -tags=flutterintegration -run TestRealFlutterAndroidBackendLease -v ./internal/cli -timeout=40m`
  はLinux amd64で248.46秒で成功。Go 1.26.8、Flutter 3.47.2、Dart 3.13.2、
  JBR Java 25、Gradle 9.3.1、NDK 28.2.13676358を使用。生成fixtureのソースコミットは
  `329768ca35b018608a31bb77b04655448f16d289`、インストールAPKのSHA-256は
  `285597924dfe685f75573f0c6abfefe0bfabe1e4eb0e465dd497362b29d6cb88`。
  build/install/パッケージ確認/reverse/launchが成功し、DartアプリのHTTPリクエストが
  nginxバックエンドに到達した。ShowはREADY、forceなしのDestroyも成功。
  同時に存在する二つの実リースへのfixture拡張はその後88.99秒で成功。正確な証拠は上記。


実装マイルストーンの検証（2026-09-08）:

- `go test ./internal/config`: 成功。不正な設定形式・依存関係を拒否するテストを含む。
- `go test ./internal/app`: 成功。mobileの並行実行、補償、DEGRADED観測、所有シリアル補間、永続的な未確認状態ガードを検証。
- Go 1.27.1で `go test -race ./...`: Linuxで成功。
- `go test ./internal/runtime/android -count=10`: 成功。Androidの汎用argv、識別、共有サーバー、パッケージ、アクティビティ、reverseのfixtureを検証。
- `go test ./internal/app -run 'TestMobile(ConcurrentNamed|FailedNamed|Unconfirmed)' -count=10`: 成功。名前付きテストは各リースのシリアルを使い、APK出自に関する注記を保持。
- 全harnessは当初、並行作業中の未整形の新規ファイルで失敗。その後、この計画の更新中に意図的に未同期だった日本語版で失敗。検査は弱めていない。commit前に整合したチェックポイントで再実行する必要がある。
- 実Flutter fixtureは `flutterintegration` で明示選択する。最初のローカル実行はSDK依存準備を確認するため実装担当が中断した。この失敗は製品の受け入れ証拠ではない。fixtureを変えず通常のFlutter/Gradle依存で再実行中。


成功・失敗の試行を再構成できる簡潔な証拠を保持します。

- リースID、ソース別名と固定コミット、マニフェストダイジェスト
- Flutter実行ファイル・版、秘匿済みargv、ビルド作業ディレクトリ、stdout/stderr
- APK相対パスとSHA-256
- Androidランタイム名とシリアル、パッケージとアクティビティ
- エンドポイント識別、デバイスポートと解決ホスト先
- install/reverse/launch結果、関連時刻、cleanup・補償結果

継承した秘密をSQLite、ログ、argv snapshot、成果物メタデータに保存せず既存秘匿処理を使います。
大きなGradle/Flutterキャッシュは証拠でないため成果物ストアにコピーしません。

## インターフェースと依存

app/domain分離を保つ追加・拡張を行います。概念上の形:

    FlutterProvider.Validate(...)
    FlutterProvider.Build(...)
    AndroidApplicationProvider.Install(...)
    AndroidApplicationProvider.Inspect(...)
    AndroidApplicationProvider.ConfigureReverse(...)
    AndroidApplicationProvider.Launch(...)
    AndroidApplicationProvider.Cleanup(...)

調査後に最終名を変えても、Emulator所有や独立start/stopを持つプロバイダーを作りません。
それは `app.AndroidProvider` のままです。appがsource → Flutter build → Compose/Android作成 →
APK install → endpoint解決 → reverse → activity起動 → observationの順序・補償を担い、
runtimeはapp方針が要求した具体外部操作だけを行います。

外部ツールはFlutter、互換Dart/Gradle/Java/Androidビルドツール、adb、既存プロバイダー経由の
Emulator、選択スタックに必要な場合のDocker Composeです。コア手順にPOSIX shell、Bash、Make、
PowerShell、symlink契約、CGO、暗黙の先頭デバイス、固定ホスト公開ポートを導入しません。

## マイルストーン1の解決済み事項

当初の7項目はすべて、判断の記録と実装済み契約で解決した。

1. `applications` と `component.application` を採用し、汎用ワークロード基盤は追加しない。
2. package/activityは常に明示し、識別子を厳密に検証する。APK調査ツールへの依存やactivity推測は導入しない。
3. `build.command[0]` でPATH上または明示ホストパスのFlutterを選び、同じ実行ファイルからバージョン証拠を得る。
4. アプリ・ビルド・reverse記録は既存Lease JSONに追加し、リース状態と一括保存する。SQLマイグレーションや別テーブルは不要。
5. `${android:<runtime>:serial}` はAndroid専用に留め、汎用ランタイム補間基盤は導入しない。
6. Flutter版、固定ソース、秘匿済みビルドargv・ログ、APKダイジェストを保持する。変更されるFlutter/Gradleキャッシュのメタデータは保持せず、再現性を主張しない。
7. 実Flutter/Emulator/backend検証は適切な高速化ローカル・ランナーホストで明示選択する `flutterintegration` fixtureにする。既存CIのネイティブ偽アダプターと実Composeは別の証拠であり、Windows/macOSの実Flutter/Emulator実行を意味しない。
