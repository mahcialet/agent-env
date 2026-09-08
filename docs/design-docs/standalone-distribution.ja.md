---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/design-docs/standalone-distribution.md
source_sha256: fdf53733afa90f5855359a46b4069b6c8c3ba961b81524eabd4f4b6f6054e706
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

リリース作成はビルドによる副作用の前に不変な Git 識別情報を検証し、`CGO_ENABLED=0` と
パス除去を指定する。ソース作業ツリー外の専用一時領域で6対象すべてを作成する。
ソースの最終検査後、検証済みbytesを新規出力先と同じ親の下の専用領域へコピーし、
完成した一式をrenameで配置する。これにより、非ignoreのツリー内出力や一時領域と
出力先が異なるfilesystemの場合も、生成ファイルをソース変更と誤認せず扱える。完成した一式には、実際のアーカイブ・実行ファイルのバイト列から
得た checksum と schema 1 の manifest を含む。バージョンの根拠は Git のみであり、
ビルドツールは実際の Go runtime バージョンを記録する。リリース CI は Go 1.27.1 に固定する。
決定性の保証範囲は、同じソースとツールチェーンでの比較に限る。

Go の tar/zip/gzip writer で、要素の順序、相対パス、mode、所有者、時刻を正規化する。
時刻は tag 対象コミットを使い、gzip metadata にホスト識別情報を含めない。
ZIP の UTC 拡張 timestamp は DOS フィールドで表現できない秒を保持する。
対応するコミット時刻は UTC の 1980-01-01 から 2106-02-07 06:28:15 までとし、
共通のアーカイブ形式で扱えるこの範囲の外は拒否する。
英日 README.txt はホストや現在時刻からではなく、ハーネスから決定的に生成する。
現在は runtime companion を埋め込まないため汎用資産のメタデータは空とし、
外部 UI helper の責務境界を維持する。

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

ビルド入力は、呼び出し元の作業ツリーではなく、解決したコミットの専用 checkout とする。
これにより、ignore 対象の Go ソースや assume-unchanged・skip-worktree フラグで隠れた編集を
除外する。配置用の隣接一時領域はソース検査後にだけ使用し、既存の出力先を拒否する。
配置済みの非ignore出力は、後続のreleaseコマンドでは通常の未追跡変更として数える。
後続コマンドでもソースをcleanに保つ場合は、ignore対象またはツリー外を出力先にする。
checksum 一覧はアーカイブのファイル名順とする。preview 検証は専用領域の `v0.1.0` tag を使い、
呼び出し元や公開 ref を変更しない。実リリースの繰り返し検証は要求された実際の tag 識別情報を維持する。
CI は tag をシェルへ展開せず、`AGENT_ENV_RELEASE_TAG` と `--tag-env` で渡す。

release用Gitコマンドは子プロセス環境でglobal/systemのGit設定と外部属性を無効化し、
専用cloneには空のGit templateを使う。外部のsmudge/process filterがコンパイラ入力を
書き換え、clean filterがその変更を隠すことを防ぐ。ユーザーのGit設定自体は変更せず、
releaseコマンドはglobal設定のcheckout filterに依存しない。
