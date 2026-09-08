---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/design-docs/standalone-distribution.md
source_sha256: c5447e5d398f9c7e57c588f880a051430504b2a64088c11dfbc16bb5f075affc
---

# スタンドアロン配布の設計

[English](standalone-distribution.md)

[製品契約](../product-specs/standalone-distribution.md)が、利用者から見える
アーカイブと前提条件を定義する。ビルド識別情報は `internal/buildinfo`、
汎用immutable bytesは `internal/assets` が扱う。リリース処理はrepoctlに置き、
ローカルとCIでargument array・shellなしの同じ実装を使う。

通常のstate rootは `internal/paths` が所有する。リリース出力は明示された呼び出し
側ディレクトリであり、runtime stateと混同しない。将来のembedded companionは、
bytesを一度記述しdigestでmaterializeし、検証済みの既存内容を再利用できる。
asset coreにAndroid固有のコードは入れない。
