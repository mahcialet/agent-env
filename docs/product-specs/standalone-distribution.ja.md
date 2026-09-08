---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/product-specs/standalone-distribution.md
source_sha256: bba29aba335224686e1e32bea003c6cf08f3887f487b41e88410d1e3abade9e3
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
相対パスの通常ファイルだけを含み、checksumsとrelease manifestを同梱する。
バージョンとソース識別子はリリースビルドのメタデータから与え、開発ビルドは
`devel`と正直に表示する。

書き込み可能な状態は既存のOS固有state rootと `AGENT_ENV_HOME` overrideを使う。
実行ファイルの隣へ書き込まない。agent-env所有のimmutable assetは状態root下へ
内容アドレス化し、digestを検証して、機能が必要とした時だけmaterializeする。

最初の配布面はGitHub archive downloadとする。署名、notarization、package
manager recipe、SBOM、attestationは後続作業とする。
