---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/product-specs/standalone-distribution.md
source_sha256: d1bd3435bc12a8fafd34f6647d39026445f15baab8586ae476bb888852bd02bd
---

# スタンドアロン配布

[English](standalone-distribution.md)

スタンドアロン配布物は、ネイティブの `agent-env` 実行ファイルと、
ライセンスおよび簡潔な導入説明で構成する。`agent-env version`、`--help`、
基本診断の実行にGo、リポジトリ、シェル、Docker、Android SDK、Flutter、
Java、Python、Node.jsは必要ない。機能ごとの外部前提は維持し、必要な機能を
実行するときに報告する。
本仕様は実装済みの配布物を扱い、利用者が導入するものと保守担当者が満たすべきリリース条件を
分けて説明する。作成処理の責務は[配布設計](../design-docs/standalone-distribution.ja.md)を参照する。

## アーカイブの構成とリリースの識別

リリースアーカイブはGo製リポジトリハーネスが作成・検証する。対象はWindows
amd64/arm64、macOS amd64/arm64、Linux amd64/arm64の6通りとする。アーカイブは
相対パスのトップレベルディレクトリと通常ファイルを含む。チェックサムとリリースマニフェストは
6 アーカイブの外に添付する。
開発ビルドは `devel` と表示する。

リリースバージョンの唯一の根拠は Git tag とする。次の四条件をすべて満たす必要がある。

| 条件 | 必要な値 |
| --- | --- |
| ソースのコミット | `HEAD` と tag のコミットが一致 |
| 作業ツリー | 後述の定義に従って clean |
| tag | 厳密な `v<semver>`。後述の安定版の形式だけを許可 |
| 要求バージョン | tag から先頭の `v` を除いた値 |

アーカイブ名は `agent-env_v0.1.0_linux_amd64.tar.gz` のようにし、`agent-env_v0.1.0_linux_amd64/` のトップレベルディレクトリに実行ファイル、`LICENSE`、`README.txt` を含める。全ファイルの mtime は tag 対象コミットの時刻 に統一し、作成時の現在時刻 は使わない。

## 状態の保存先と配布の範囲

書き込み可能な状態は既存のOS固有状態ルートと `AGENT_ENV_HOME` による上書きを使う。
実行ファイルの隣へ書き込まない。agent-env が所有する不変のアセットは状態ルート下へ
内容のハッシュに基づいて配置し、ダイジェストを検証する。実際に展開するのは、その機能が必要になったときだけとする。

絶対パスの `AGENT_ENV_HOME` は、ユーザーのホームを取得できない環境でも
使用できる。検証済みUI 補助ツールのインストール用コピーを含む実行時の一時
ファイルは、解決済みの状態保存先内に置く。Gitのワークツリー登録情報や
共有ADBサーバーなど、外部ツールが管理するホスト側の状態は既存の契約に従う。

最初の配布面はGitHub のアーカイブダウンロードとする。署名、Apple の公証、パッケージマネージャー向けの定義、SBOM、attestationは後続作業とする。


## リリースの作成と検証

### 受け付ける tag とソースの変更検査

受け付ける tag は `vMAJOR.MINOR.PATCH` で、各要素は 10 進整数とし、`0` 自体を除き
先頭のゼロを認めない。プレリリースやビルドの接尾辞 は受け付けない。有効なバージョン tag のうち
HEAD に解決されるものが、ちょうど一つ必要である。軽量 tag と注釈付き tag を受け付けるが、
署名検証は対象外とする。要求バージョンは tag から `v` を除いた値であり、VERSION ファイルなど
別のバージョン管理元は持たない。clean とは、追跡対象の作業ツリー変更、ステージ済み変更、
ignore 対象以外の未追跡ファイルがない状態を指す。ignore 対象の出力は妨げにならない。

リリース検証はassume-unchangedまたはskip-worktreeが設定された追跡対象ファイルを、
現在の内容が一致していても拒否する。リリース前に明示的にフラグを解除する。
リリースコマンドは呼び出し元のインデックスを変更しない。開発用実行ファイルはリンカーで
識別情報が明示されていなければ、Goに埋め込まれたVCS revision/modifiedを使い、
VCS メタデータがない場合はunknownを維持する。

### リリースの作成、検査、比較

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

### チェックサムとリリースマニフェスト

`checksums.txt` はファイル名順に並べ、各行を小文字の SHA-256、空白二つ、アーカイブの
ファイル名、LF で構成する。`release-manifest.json` はスキーマバージョン 1 を使い、
製品・バージョン・tag・commit、ソース時刻、作成に使った Go バージョン、対象識別情報、
アーカイブと実行ファイルの名前・ダイジェスト、実行ファイルのビルド識別情報、埋め込み資産の
メタデータを記録する。現在のリリースはランタイム補助ツールの資産を埋め込まないため、
資産一覧は空である。任意の Android UI 補助ツールは引き続き外部ビルドとする。
静的検査では、他プラットフォームの実行ファイルを起動せずにビルド情報を確認する。
クロスビルドの成功はネイティブ実行の証拠にはならない。

### tag の指定と再現性

リリースコマンドは `--version X.Y.Z` の代わりに `--tag vX.Y.Z` も受け付ける。
CI は `--tag-env` で `AGENT_ENV_RELEASE_TAG` を直接読み、tag をシェルコマンドへ展開しない。
最終出力は 6 アーカイブ、`checksums.txt`、`release-manifest.json` の計 8 ファイルに限る。
`release-repeat` は既存の tag 対応成果物を検証し、新たなビルドと比較する。
作成には厳密なコミットの専用のチェックアウト を使うため、ignore 対象のソースや
Git index のフラグで隠れたローカル編集が実行ファイルに混入することはない。

## 実行ファイルの同梱アセット一覧

`version --output json` の `data.assets` は、`name`、`version`、`sha256`
（小文字の16進数）、`size`（バイト数）を持つオブジェクトの配列である。
現在の製品には同梱アセットがないため、省略やnullではなく `[]` を返す。
表形式では `Bundled assets: 0` と表示する。外部から指定された補助ツール APKや
テスト用フィクスチャは一覧に含めない。一覧の取得で状態保存先を作成したり、
外部ツールを探索したりすることはない。

## browserの前提条件

Browser/CDPコマンドには、対象マニフェストが宣言する直接起動できる互換性のあるネイティブのヘッドレス Chromium 系実行ファイルが追加で必要である。
検証にはChrome for Testingを推奨するが、同梱しない。ブラウザーのダウンロード、Node、Python、Playwright、
Selenium、ChromeDriverを基本機能ランタイムの依存にはしない。バージョン、help、無関係な機能はブラウザーなしで動く。
[browser契約](browser-cdp-automation.ja.md)と[前提条件matrix](../../README.ja.md)を参照する。

## 検証記録

[完了済み配布計画](../exec-plans/completed/standalone-distribution.ja.md)と
[リリース実装計画](../exec-plans/completed/standalone-release-finalization.ja.md)に、検証した成果物と
ネイティブ実行の結果を記録している。署名や未実施のプラットフォーム検証まで保証するものではない。
