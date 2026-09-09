---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/design-docs/standalone-distribution.md
source_sha256: 60d76c2c8f468d75f3fc7c6eeca19b9e56af6e3be1279455e8c9db5803681df6
---

# スタンドアロン配布の設計

[English](standalone-distribution.md)

[製品契約](../product-specs/standalone-distribution.ja.md)が、利用者から見える
アーカイブと前提条件を定義する。ビルド識別情報は `internal/buildinfo`、
汎用immutable bytesは `internal/assets` が扱う。リリース処理はrepoctlに置き、
ローカルとCIでargument array・shellなしの同じ実装を使う。

通常のstate rootは `internal/paths` が所有する。リリース出力は明示された呼び出し
側ディレクトリであり、runtime stateと混同しない。将来のembedded companionは、
bytesを一度記述しdigestでmaterializeし、検証済みの既存内容を再利用できる。
asset coreにAndroid固有のコードは入れない。

## パッケージ生成と検査

### 不変のビルド入力

ビルド入力は、呼び出し元の作業ツリーではなく、解決したコミットの専用 checkout とする。
これにより、ignore 対象の Go ソースや assume-unchanged・skip-worktree フラグで隠れた編集を
除外する。

source検査はporcelain statusより
前にassume-unchanged/skip-worktreeのindex項目を拒否します。private checkoutは
compiler入力を追加で保護しますが、それだけで呼び出し元のclean状態を証明しません。

release用Gitコマンドは子プロセス環境でglobal/systemのGit設定と外部属性を無効化し、
専用cloneには空のGit templateを使う。外部のsmudge/process filterがコンパイラ入力を
書き換え、clean filterがその変更を隠すことを防ぐ。ユーザーのGit設定自体は変更せず、
releaseコマンドはglobal設定のcheckout filterに依存しない。

リリース作成はビルドによる副作用の前に不変な Git 識別情報を検証し、`CGO_ENABLED=0` と
パス除去を指定する。ソース作業ツリー外の専用一時領域で6対象すべてを作成する。
ソースの最終検査後、検証済みbytesを新規出力先と同じ親の下の専用領域へコピーし、
完成した一式をrenameで配置する。これにより、非ignoreのツリー内出力や一時領域と
出力先が異なるfilesystemの場合も、生成ファイルをソース変更と誤認せず扱える。完成した一式には、実際のアーカイブ・実行ファイルのバイト列から
得た checksum と schema 1 の manifest を含む。バージョンの根拠は Git のみであり、
ビルドツールは実際の Go runtime バージョンを記録する。リリース CI は Go 1.27.1 に固定する。
決定性の保証範囲は、同じソースとツールチェーンでの比較に限る。

### 同じバイト列を生成するアーカイブ

Go の tar/zip/gzip writer で、要素の順序、相対パス、mode、所有者、時刻を正規化する。
時刻は tag 対象コミットを使い、gzip metadata にホスト識別情報を含めない。
ZIP の UTC 拡張 timestamp は DOS フィールドで表現できない秒を保持する。
対応するコミット時刻は UTC の 1980-01-01 から 2106-02-07 06:28:15 までとし、
共通のアーカイブ形式で扱えるこの範囲の外は拒否する。
英日 README.txt はホストや現在時刻からではなく、ハーネスから決定的に生成する。
現在は runtime companion を埋め込まないため汎用資産のメタデータは空とし、
外部 UI helper の責務境界を維持する。

### 静的検査とネイティブ smoke

静的検査は Go の `debug/buildinfo` で全対象の実行ファイルを調べ、VCS 識別情報、CGO、
プラットフォーム設定を確認する。さらに、リンカーで与え、実行時のビルド識別情報からも読む
`ReleaseRecord` マーカーを検証する。`-trimpath` を使うと Go は `debug/buildinfo` から
リンカーフラグを省くため、静的比較に必要なバージョン・コミットの値はこのマーカーから取得する。
宣言された digest と実際の値も照合し、安全でない要素や想定外の要素を拒否する。
パス漏えいの検査範囲は、制御するメタデータと既知のリポジトリ・一時パスの接頭辞に限る。
考えられるすべてのホストパスを検出するものではない。他プラットフォームのバイナリは実行しない。
ネイティブ smoke は別の操作として、ホストに合う対象を展開し、制限した PATH、
外部の作業ディレクトリ、隔離した状態保存先で行う。workflow は再ビルドせず、
同じ候補バイト列を検証と smoke の条件に通してから公開する。

releaseのパス検査は、既知のcheckout/一時パスと、Go build metadataで
識別できる正確なmoduleパスを区別します。全binaryのパスを網羅する
scannerではなく限定的なmetadata/パス検査です。

### 検証済みの一式を配置する

配置用の隣接一時領域はソース検査後にだけ使用し、既存の出力先を拒否する。
配置済みの非ignore出力は、後続のreleaseコマンドでは通常の未追跡変更として数える。
後続コマンドでもソースをcleanに保つ場合は、ignore対象またはツリー外を出力先にする。
checksum 一覧はアーカイブのファイル名順とする。preview 検証は専用領域の `v0.1.0` tag を使い、
呼び出し元や公開 ref を変更しない。実リリースの繰り返し検証は要求された実際の tag 識別情報を維持する。
CI は tag をシェルへ展開せず、`AGENT_ENV_RELEASE_TAG` と `--tag-env` で渡す。

## 一覧と将来の埋め込みアセット利用

`assets.Inventory()` は外部機能に依存しない製品の同梱一覧で、
`buildinfo.Current()` が参照します。現在は明示的な空配列を返します。
fixtureはテスト用実行ファイルにだけ埋め込みます。リリースのnative smokeは、
CLIの一覧が現在のmanifestの空一覧と一致することを要求します。実際の
companionを追加するときは、両方の一覧と検証を同じ変更で更新します。
現時点のrelease checkerは、検証していない内容を受け入れないよう、
空でないmanifestのアセット一覧を拒否します。

Browser/CDPは実行ファイルへGo WebSocket transportを加えますが、browser assetやhelper runtimeは
追加しません。browser binaryはmanifestで宣言する外部toolです。release packagingと空の同梱asset inventoryは
変わりません。[browser設計](browser-cdp-automation.ja.md)を参照してください。

### 将来 companion を組み込む手順

将来のhelperは、信頼するビルド時のバイト列を `go:embed` で埋め込み、
`assets.Describe(name, version, bytes)` で正確なメタデータを計算します。
選択された機能が必要とするときだけ
`assets.Materialize(resolvedStateRoot, info, bytes)` を呼び出します。
機能の識別、ライセンス確認、期待するバージョンの検証は呼び出し側が担当し、
assetsは不変のバイト列の検証と保存だけを担当します。新規保存時にmode 0600を指定します。
実際の権限はOSのファイルシステムに従います。APKにはホスト上の実行権限は不要です。
この契約にダウンローダーや対象アプリのビルドは含みません。既存の外部UI
helper指定は、別の機能変更で明示的に置き換えるまで維持します。
テストはAndroid SDK、Flutter、対象アプリ、runtimeの初期化なしで埋め込みを検証します。

### 同時書き込みと上限付きの再利用

複数プロセスのディレクトリ作成が競合しEEXISTになった場合、作成された
エントリを再検査し、symlinkではないディレクトリだけを受け入れます。
各書き込みは保存先ディレクトリ内の固有の一時ファイルから、同一の検証済み
バイト列を公開します。他の書き込みが作成した通常ファイルも、内容を検証して
から再利用します。既存内容の破損は拒否し、黙って修復しません。
状態保存先に対する悪意ある並行変更は信頼境界の外です。sandboxではありません。

Windowsでは置換フラグなしのMoveFileExで公開します。先行する書き込みが
完了していれば内容を検証して再利用し、別の読み取りが開いているファイルを
置き換えません。Unixでは同一内容をatomic renameします。埋め込みfixtureは
Git属性の-textを指定し、checkout時の改行変換を防ぎます。

Windowsでの内容検証は長いパスに対応するGoのファイル読み取りを使い、
共有/lock違反だけを10ms間隔、最大2秒で再試行します。他のopenエラーと内容不一致は直ちに失敗します。
期限を設けることで永続的な干渉を成功と扱わず、状態ディレクトリに対する
悪意ある書き込みがある状況での進行も保証しません。

アセット名は全OSでWindowsのdevice名、禁止文字、末尾のドット/空白を拒否します。
キャッシュは読み取り前に信頼するmetadataとサイズを比較し、その後にファイルが
変わっても読み取り量を制限します。通常の再利用と公開競合時の両方で、その上限内の
正確なバイト列を検証します。

## 永続的な書き込み先の監査

| agent-envが所有するデータ | 解決済み状態保存先からの相対位置 |
| --- | --- |
| registry、WAL、共有メモリ用ファイル | `state.db*` |
| 固定したソースのworktreeとビルド出力 | `worktrees/<lease>/<source>/` |
| 環境記述、Compose設定 | `leases/<lease>/` |
| コマンドログ、結果、コピーした成果物、UI復旧証拠 | `leases/<lease>/artifacts/` |
| Android所有者マーカー、専用AVD、emulator/ADBログ、起動識別情報 | `leases/<lease>/android/<runtime>/` |
| Emulatorのホストデータと一時ディレクトリ | `leases/<lease>/android/<runtime>/avd/emulator-data/` とその配下の `Temp/` |
| 検証済みhelperのインストール用コピー | 所有runtimeディレクトリ内の一時APK。成功時も失敗時も削除 |
| 不変アセットのキャッシュ | `assets/<name>/<sha256>/<name>` |

証拠とマーカーのatomic writeは保存先と同じディレクトリに一時ファイルを
作成します。Windowsのdetached process終了証拠も、所有するstdoutファイルと
同じ場所に置きます。テストでは実際のSQLiteとnative子プロセスを使い、
lifecycle providerはfakeに置き換えます。別のnative release smokeで、
3OSのcoreコマンドの状態保存先を検証します。

GitはソースリポジトリのGit管理情報にlinked worktreeを登録します。
Dockerリソース、共有ADBサービスと鍵、SDK/Flutter/Gradleキャッシュは、
既存の契約に従う外部ツールの状態です。対象リポジトリが定義するコマンドは
所有worktree内で実行され、信頼するコードとして他の副作用を持つ場合があります。
開発者向けrelease/helperビルダーの明示的な出力先は、実行時の状態保存先とは
別の契約です。この監査は任意の外部ツールの書き込みを封じ込める保証ではありません。
