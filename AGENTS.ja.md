---
status: active
owner: maintainers
last_verified: 2026-09-10
translation_of: AGENTS.md
source_sha256: e841e3b9112f4af4e95336861061b72d5e3c1dcbb23387a40042c7d93abb7796
---

# エージェントの入口

[英語版（翻訳元）](AGENTS.md)

## 最初に読む文書

1. [アーキテクチャ](ARCHITECTURE.ja.md)：責務と依存の方向。
2. [ドキュメント索引](docs/index.ja.md)：リポジトリの正式な知識。
3. [MVP仕様](docs/product-specs/agent-env-mvp.ja.md)：当初の要件。現在の機能は[製品仕様](docs/product-specs/index.ja.md)に従います。
4. [計画の規則](docs/PLANS.ja.md)：active ExecPlanの要件と必須構成。
5. [完了済みMVP ExecPlan](docs/exec-plans/completed/agent-env-mvp.md)：提供済みの範囲と検証の記録。

## 標準的な作業手順

- 編集前に、現在のブランチ、作業ツリー、既存の差分を確認する。
- 実質的な変更ごとに専用ブランチとactive ExecPlanを使う。
- 想定ブランチと現在の作業は、active ExecPlanの記述に従う。
- ユーザーの変更、既存資産、MITライセンスを保護する。
- 動作を変える前に、関連する製品仕様と設計文書を読む。
- 意味のある区切りでactive ExecPlanを更新する。
- 計画の規則に従い、Plan ID、依存に基づく選出、Gitでの追跡を使う。
- Draftの自動昇格、人による検証の自動実行、古いレビューでのマージを行わない。
- 公開動作に不確実性がある場合は、それに依存する実装の前に解消する。
- 確認可能な変更を行い、関連動作を検証する。
- 発見、判断、失敗した検証、次の行動を計画に記録する。
- テストと動作変更を同じ一貫した変更単位に含める。
- すでに許可されている場合は、検証済みの一貫した区切りをcommit・pushする。
- 公開済みの履歴を書き換えず、force-pushや暗黙のremote再設定を行わない。
- 最終差分を点検し、自分の変更と他の作業者の変更を区別する。

## 安定したコマンド

サポート対象のGoツールチェーンをPATHに置く。以下はGo製のリポジトリハーネスが提供し、Bash、Make、PowerShellなしで動作しなければならない。

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go run ./tools/repoctl test-unit
go run ./tools/repoctl test-integration
go run ./tools/repoctl docs-check
go run ./tools/repoctl generated-check
go run ./tools/repoctl arch-check
go run ./tools/repoctl generate
```

内部コマンドと検証範囲は[品質](docs/QUALITY.ja.md)を参照。統合テストは明示的にDockerを必要とし、前提条件の不足を成功として扱わない。

## 必ず守る不変条件

- Windows、macOS、Linuxのネイティブ実行をサポートする。WSLは別のLinux実行環境として扱う。
- リリースビルドにCGO、POSIXシェル、symlink、必須daemonを要求しない。
- 外部ツールは引数配列とOSネイティブなパス処理で実行する。
- runtimeの起動前に、すべてのsource refを不変のcommitへ解決する。
- sourceの組を正規化し、lease全体を単一のcommitに縮約しない。
- 対象リポジトリ固有の起動処理は、そのmanifestに置く。
- componentが依存を所有し、stackが明示的なrootを宣言する。
- 各leaseは固有のworktreeと明示的なCompose project識別子を所有する。
- SQLiteは期待する状態を記録し、reconcileが外部リソースから観測状態を得る。
- 失敗した割り当て、清掃の証拠、隔離状態の可視性を維持する。
- 予期しない追跡対象ファイルの変更や、所有権が曖昧なリソースを黙って削除しない。
- GCの既定はdry-runとし、force削除は明示的かつ監査可能に保つ。
- 環境分離は、悪意あるコードに対するsandboxではない。
- ログ、データベースのメタデータ、commitする成果物に秘密情報を残さない。
- クロスビルドの成功だけでは、ネイティブ実行時の動作を証明できない。
- 未対応の前提条件と未検証の受け入れ条件を正直に報告する。

## テストの証拠

- 証拠は何を示すかで分類する。反復で実行順序の強制を証明しない。
- fixtureは失敗経路も含めcancel、資源close、完了joinを所有する。
- 拒否やエラーを確認するテストでは、検証対象の条件に到達し、意図した拒否・エラーを無関係な失敗と区別する。
- 指摘の判断と修正前失敗の対照をactive Planへ記録する。
- 実行順序・oracle・reviewは[品質の証拠規則](docs/QUALITY.ja.md#証拠の種類とテストアーキテクチャ)に従う。

## 知識を置く場所

- [製品仕様](docs/product-specs/index.ja.md)：ユーザーに見える契約。
- [設計文書](docs/design-docs/index.ja.md)：持続的な仕組みと根拠。
- [ADR](docs/adr/index.ja.md)：具体的な選択と不採用の代案。
- [計画](docs/PLANS.ja.md)：現在の実行、復旧、受け入れの証拠。
- [セキュリティ](docs/SECURITY.ja.md)：信頼とホストの方針。
- [信頼性](docs/RELIABILITY.ja.md)：reconcileと慎重な清掃。
- [移植性](docs/PORTABILITY.ja.md)：OSとプロセス実行の制約。
- [ロードマップ](docs/roadmap.ja.md)：延期した機能と未決事項。
- [参考資料](docs/references/index.ja.md)：過去の入力と出所。

schema文書はrepoctlでmigrationから生成し、手作業で維持しない。
強制できる不変条件はvalidatorとテストが担い、文書はその意図を説明する。
対応するvalidatorと、不正fixtureを拒否するテストが通るまで、規則が強制されるとは主張しない。

## ドキュメントの保守

英語版AGENTS.mdは案内に徹し、150行以下、目安80〜120行に保つ。
永続的な設計・製品・ADR文書はローカル索引で案内する。
永続文書と計画にstatus、owner、last_verifiedメタデータを維持する。
繰り返す発見は、このファイルを膨らませるよりテストや検査へ反映する。
下位ツリーの規則が実質的に異なる場合以外、入れ子の指示ファイルを避ける。
アーカイブされたhandoffは過去の入力であり、恒久的なプロジェクトマニュアルではない。
計画をcompletedへ移す前に、受け入れを完了しベースへのマージを確認する。
完了済み計画を移す際はリンクを直し、振り返りを残す。

人が読む永続文書は英語と日本語で維持する。
内容が食い違う場合は英語の`*.md`を優先し、日本語訳は対応する`*.ja.md`に置く。
永続文書を追加・変更する際は、同じ一貫した変更単位で両言語を更新する。
source hashを更新する前に翻訳の意味を確認する。hashの一致だけでは正確さを証明できない。
意味を省かず、各言語の読者に合わせて書く。文や節の順序を英語にそろえる必要はない。
文書の構成を実質的に変える場合は、言語の方針に従って独立レビューを行う。
生成文書と歴史的アーカイブには明示的な例外を設ける。
文書検査は翻訳の欠落と更新漏れを検出しなければならない。完了前にdocs-checkを実行する。
適用範囲、メタデータ、索引、例外登録は[言語の方針](docs/design-docs/bilingual-documentation.ja.md)を参照。
