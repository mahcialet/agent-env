---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/product-specs/standalone-distribution.md
source_sha256: 392aa3fe8c7c2d68ec92eb883fe851c733ff9957e72caf35b9fa06f7f4049c76
---

# スタンドアロン配布

[English](standalone-distribution.md)

スタンドアロン配布物は、ネイティブの `agent-env` 実行ファイルと、
ライセンスおよび簡潔な導入説明で構成する。`agent-env version`、`--help`、
基本診断の実行にGo、リポジトリ、シェル、Docker、Android SDK、Flutter、
Java、Python、Node.jsは必要ない。機能ごとの外部前提は維持し、必要な機能を
実行した時点で遅延して報告する。

リリースアーカイブはGo製リポジトリハーネスが作成・検証する。対象はWindows
amd64/arm64、macOS amd64/arm64、Linux amd64/arm64の6通りとする。アーカイブは
相対パスのトップレベルディレクトリと通常ファイルを含む。checksum と release manifest は
6 アーカイブの外に添付する。
バージョンとソース識別子はリリースビルドのメタデータから与え、開発ビルドは
`devel`と正直に表示する。

書き込み可能な状態は既存のOS固有state rootと `AGENT_ENV_HOME` overrideを使う。
実行ファイルの隣へ書き込まない。agent-env所有のimmutable assetは状態root下へ
内容アドレス化し、digestを検証して、機能が必要とした時だけmaterializeする。

最初の配布面はGitHub archive downloadとする。署名、notarization、package
manager recipe、SBOM、attestationは後続作業とする。


リリースバージョンの唯一の根拠は Git tag とする。リリース要求は、`HEAD` が tag の commit と一致し、作業ツリーが clean で、tag が厳密な `v<semver>` 形式で、要求バージョンが先頭の `v` を除いた tag と一致する場合だけ有効とする。

アーカイブ名は `agent-env_v0.1.0_linux_amd64.tar.gz` のようにし、`agent-env_v0.1.0_linux_amd64/` のトップレベルディレクトリに実行ファイル、`LICENSE`、`README.txt` を含める。全ファイルの mtime は tag 対象 commit の timestamp に統一し、wall clock は使わない。

## リリースの作成と検証

受け付ける tag は `vMAJOR.MINOR.PATCH` で、各要素は 10 進整数とし、`0` 自体を除き
先頭のゼロを認めない。prerelease や build suffix は受け付けない。有効なバージョン tag のうち
HEAD に解決されるものが、ちょうど一つ必要である。軽量 tag と注釈付き tag を受け付けるが、
署名検証は対象外とする。要求バージョンは tag から `v` を除いた値であり、VERSION ファイルなど
別のバージョン管理元は持たない。clean とは、追跡対象の作業ツリー変更、ステージ済み変更、
ignore 対象以外の未追跡ファイルがない状態を指す。ignore 対象の出力は妨げにならない。

```text
go run ./tools/repoctl release-build --version X.Y.Z --out <new-directory>
go run ./tools/repoctl release-check --dir <directory> --version X.Y.Z
go run ./tools/repoctl release-repeat --dir <directory> --version X.Y.Z
go run ./tools/repoctl release-smoke --dir <directory> --version X.Y.Z
```

ビルドは既存の出力ディレクトリを拒否し、専用の一時領域で作成してから完全な成果物一式を
ローカルの出力先に配置する。Windows は `.zip`、darwin と linux は `.tar.gz` とし、
それぞれ amd64 と arm64 を用意する。各アーカイブに含めるのはトップレベルディレクトリ、
ネイティブ実行ファイル、MIT の `LICENSE`、`README.txt` だけである。
README.txt はハーネス内の英日テキストから決定的に生成する。

`checksums.txt` はファイル名順に並べ、各行を小文字の SHA-256、空白二つ、アーカイブの
ファイル名、LF で構成する。`release-manifest.json` は schema version 1 を使い、
製品・バージョン・tag・commit、ソース時刻、実際の builder の Go バージョン、対象識別情報、
アーカイブと実行ファイルの名前・digest、実行ファイルのビルド識別情報、埋め込み資産の
メタデータを記録する。現在のリリースは runtime companion の資産を埋め込まないため、
資産一覧は空である。任意の Android UI helper は引き続き外部ビルドとする。
静的検査では、他プラットフォームの実行ファイルを起動せずにビルド情報を確認する。
クロスビルドの成功はネイティブ実行の証拠にはならない。

リリースコマンドは `--version X.Y.Z` の代わりに `--tag vX.Y.Z` も受け付ける。
CI は `--tag-env` で `AGENT_ENV_RELEASE_TAG` を直接読み、tag をシェルコマンドへ展開しない。
最終出力は 6 アーカイブ、`checksums.txt`、`release-manifest.json` の計 8 ファイルに限る。
`release-repeat` は既存の tag 対応成果物を検証し、新たなビルドと比較する。
作成には厳密なコミットの専用 checkout を使うため、ignore 対象のソースや
Git index のフラグで隠れたローカル編集が実行ファイルに混入することはない。

## 実行ファイルの同梱アセット一覧

`version --output json` の `data.assets` は、`name`、`version`、`sha256`
（小文字の16進数）、`size`（バイト数）を持つオブジェクトの配列です。
現在の製品には同梱アセットがないため、省略やnullではなく `[]` を返します。
表形式では `Bundled assets: 0` と表示します。外部から指定されたhelper APKや
テスト用fixtureは一覧に含めません。一覧の取得で状態保存先を作成したり、
外部ツールを探索したりすることはありません。

絶対パスの `AGENT_ENV_HOME` は、ユーザーのホームを取得できない環境でも
使用できます。検証済みUI helperのインストール用コピーを含む実行時の一時
ファイルは、解決済みの状態保存先内に置きます。Gitのworktree登録情報や
共有ADBサーバーなど、外部ツールが管理するホスト側の状態は既存の契約に従います。

release検証はassume-unchangedまたはskip-worktreeが設定されたtracked項目を、
現在の内容が一致していても拒否します。release前に明示的にflagを解除してください。
releaseコマンドは呼び出し元のindexを変更しません。開発用実行ファイルはlinkerで
識別情報が明示されていなければ、Goに埋め込まれたVCS revision/modifiedを使い、
VCS metadataがない場合はunknownを維持します。

## browserの前提条件

Browser/CDPコマンドには、対象manifestが宣言する互換native headless Chromium系実行ファイルが追加で必要です。
検証にはChrome for Testingを推奨しますが、同梱しません。browserのdownload、Node、Python、Playwright、
Selenium、ChromeDriverをcore runtimeの依存にはしません。version、help、無関係な機能はbrowserなしで動きます。
[browser契約](browser-cdp-automation.ja.md)と[前提条件matrix](../../README.ja.md)を参照してください。
