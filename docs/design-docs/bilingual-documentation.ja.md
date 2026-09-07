---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/design-docs/bilingual-documentation.md
source_sha256: 08de781ccf2b4a0b3758196583dc6c2382255c3cecc8c0dd1737b1fc7116ba78
---

# 英語・日本語のドキュメント

[正本の英語版](bilingual-documentation.md)

人が読む永続文書は英語と日本語で維持する。英語を規範となる正本とし、内容の不一致は修正すべき文書のずれとして扱う。日本語を古いまま残してよい理由にはしない。日本語訳は保守対象の成果物であり、任意の要約ではない。

## 適用範囲と名前

リポジトリ直下のMarkdownと`docs/`配下のMarkdownに適用する。日本語版は`.md`を`.ja.md`に置き換え、例えば`README.ja.md`や`docs/product-specs/android-emulator.ja.md`とする。索引、エージェント向け案内、ADR、新しいactive/completed ExecPlanも対象となる。ソースのコメント、CLIメッセージ、第三者コード、この文書範囲外にあるテストfixtureは翻訳しない。

英語と日本語は同じ一貫した変更単位で更新する。意味、制約、証拠を省略せず訳し、コマンド、識別子、診断コード、パスは保つ。英語の索引は両言語へのリンクを示す。日本語の索引は日本語版へリンクし、例外となる過去の記録・生成文書へのリンクは英語のままにする。各日本語文書には、正本の英語版へ読者が辿れるリンクを置く。

## 翻訳メタデータとレビュー

日本語版のfront matterにはstatus、owner、last_verifiedを維持し、以下を追加する。

```yaml
translation_of: docs/design-docs/bilingual-documentation.md
source_sha256: <64 lowercase hexadecimal characters>
```

メタデータの値は1行の通常の文字列、または両端の引用符が対応する文字列とする。入れ子や複数行のYAMLはハーネスのメタデータ形式の対象外である。

`translation_of`は、リポジトリ相対でスラッシュ区切りの正確な英語パスとする。`source_sha256`はfront matterを含む英語ファイル全体について、改行CRLFをLFへ正規化した後のSHA-256とする。末尾の改行を含む、それ以外のバイトは意味を持つ。これによりWindows、macOS、Linuxのネイティブ実行で同じ結果になる。

英語の差分を読み、日本語の意味とリンクを更新し、両方を見直してから新しいsource digestを記録する。翻訳を確認せずdigestだけを更新しない。検査が検出するのは未確認の原文変更であり、翻訳の意味の正確さや、虚偽の確認記録までは判定できない。通常の`docs-check`はファイルを書き換えず、外部翻訳サービスも呼び出さない。

## 明示的な例外

[例外登録ファイル](../translation-exceptions.json)に、正確なパスと空でない理由を列挙する。認める分類は`docs/generated/`の生成物、`docs/references/handoffs/`の参考資料アーカイブ、そこに登録された移行前の完了済みExecPlanである。ディレクトリ全体や暗黙の例外は認めない。索引はリンク先が過去の資料でも両言語で維持する。

初期の例外は、生成されたDB schema、元のhandoffアーカイブ、この移行より前のMVP・Androidの実装／レビューに関する完了済み計画4件である。これらの英語の証拠はそのまま保持する。新しい計画には完了まで両言語が必要であり、過去の計画を実質的に変更して現在の案内にする場合は、例外を外して翻訳を追加する。

## 検査と完了

`go run ./tools/repoctl docs-check`は、必要な対訳、正本のメタデータ、鮮度hash、言語別の索引、ローカルリンクを検証する。対訳の欠落と古い翻訳はエラーとして報告する。対応する英語文書がない日本語文書や、異なる正本パスを指す日本語文書も失敗する。既存のリンク、メタデータ、アーキテクチャ、生成物の検査も維持する。

変更した日本語文書を更新し、これらの検査が通るまではactive ExecPlanを完了できない。両言語の計画を一緒に移動し、リンクを修正し、最後の英語編集後に日本語計画のsourceメタデータを更新する。
