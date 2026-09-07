---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/QUALITY.md
source_sha256: a13fbbd00e069d7c5b0f75440b1da081dc032baf8f97af9605fc26bf343ccb64
---

# 品質と検証

[正本の英語版](QUALITY.md)

## 正式な検査

```text
go run ./tools/repoctl doctor
go run ./tools/repoctl check
go run ./tools/repoctl test-integration
```

`doctor`はGo、gofmt、Gitの場所を確認する。Dockerが必要なのは明示的な統合コマンドだけである。`check`は、整形確認、`go test ./...`、`go vet ./...`、文書検証、生成物のずれ検出、アーキテクチャ検査を、実行内容が分かる形で組み合わせる。Bash、Make、PowerShellは不要である。内部の各コマンドも直接実行できる。

| ハーネスのコマンド | 範囲 |
| --- | --- |
| `test-unit` | 通常のGoテスト。Docker daemonは不要 |
| `test-integration` | Docker daemon／Composeを確認し、明示的な統合テストのopt-in付きでtag指定のテストをキャッシュなしで実行 |
| `docs-check` | メタデータ、リンク／見出し、ローカル索引、AGENTSの長さ／パス、active planの必須構成、英語／日本語の対訳の欠落・更新漏れ |
| `generate` | 埋め込まれたSQL migrationを正本としてDB文書を再生成 |
| `generated-check` | 生成schemaとmigrationが異なれば失敗 |
| `arch-check` | 修正方法を示す診断により、文書化されたimportの境界を強制 |

schema文書の正本は、手書きの説明ではなく[migrations](../migrations/001_initial.sql)配下の番号付きSQL群である。安定した`AGENTENV-*`診断により、違反した不変条件と修正の方向を特定する。不正fixtureでは、リンク、索引、メタデータ、計画セクション、ソース整形、schema生成、import境界、欠落／孤立した翻訳、誤ったsourceメタデータ、古いhash、言語別リンク／索引を意図的に壊して検証する。

## 製品のテスト

unit／adapterテストは、厳密なmanifest解析、決定的なcomponent／source識別、状態遷移、TTLとGC適格性、正規化されたCompose方針、パスの包含、command argv、ストリームの伏字化、証拠保存の失敗経路、ライフサイクルの補償処理を対象とする。fake runnerはシェル文字列を組み立てずに、実行ファイル、argv、作業ディレクトリ、環境制御、キャンセル、出力処理、エラー変換を検査する。

SQLiteテストは実際の一時DBを使い、Unicodeパス、再オープン／複数接続、migrationの冪等性、外部キー、容量／project予約の原子性、正規化された行のrollback、永続的なlock動作を確認する。プロセステストはネイティブargvと子プロセスのキャンセルを実行する。OS固有の実装を検証するには、そのプラットフォームでのネイティブ実行が必要である。

## 実Dockerのfixture

[統合テスト群](../internal/cli/integration_test.go)は、独立した一時Gitリポジトリとstate homeを作り、[小さなCompose fixture](../testdata/compose/compose.yaml)を使う。テストには`integration` build tagと明示的なopt-inの両方が必要であり、ハーネスが自動で設定する。通常のunitテストはDocker containerを起動しない。

同時に動くAPI／Dashboard lease、選択された依存closure、sourceにportを宣言せずmanifestから生成する動的loopback HTTP、バージョン付き環境descriptor、component単位の実行中／保存済みログ、固有のproject／worktree識別、兄弟leaseのcontainer／network／volume IDの維持を確認する。また、未選択の外部volumeが清掃後も残ること、手動削除したprojectがdegradedになること、複数リポジトリがsource ref overrideに従うこと、名前付きテストが伏字化されたstdout／stderr／artifactと非ゼロ終了状態を保存すること、readiness失敗が実リソースをrollbackすること、変更済みの追跡対象worktreeはdiff証拠を伴う明示的なforceまでGCで削除されないことも検証する。

最後のローカルLinux CLI統合テストは109.95秒で成功し、生成endpoint、診断descriptor、component単位の実行中／アーカイブログを確認した。これはそのリビジョンでの実Docker動作の証拠であり、後続の変更や全ネイティブプラットフォームの完了を示すものではない。fixtureの全リソースは固有の追跡可能な識別子とlease単位の清掃を使い、一般的なDocker pruneは実行しない。

## CIと完了の証拠

CIはGo 1.26.xと1.27.xを使い、Windows、macOS、LinuxでハーネスとCLIビルドをネイティブ実行する。Linuxでは`go test -race ./...`と明示的なDocker統合テストも実行する。別のクロスビルドjobで`CGO_ENABLED=0`の5対象を確認する。

[完了済み実装計画](exec-plans/completed/agent-env-mvp.md)には33件すべての受け入れ条件と、解決した独立レビューの指摘を記録している。c641286のCI 34124194139では、OS／Goのネイティブ検査6件、CGOを無効にしたビルド5件、全raceテストと実Docker統合テストを行うLinux jobの計12件が成功した。ローカルのGo 1.26.8／1.27.1での検査と実Docker統合テストも成功した。hosted macOS／WindowsでのDocker統合テストは実施済みとはしていない。

## 翻訳の検証

永続文書は英語を正本とし、日本語の`.ja.md`と対にする。[言語の方針](design-docs/bilingual-documentation.ja.md)でメタデータと正確なパス単位の例外を定義する。`docs-check`は、英語ファイル全体のCRLFをLFへ正規化したSHA-256と、翻訳のsource hashを比較する。そのためWindowsのcheckoutでもLinux／macOSと一致する。検査は読み取り専用であり、翻訳メタデータを自動更新しない。hashの一致は、どの原文リビジョンを確認したかを示すだけで、翻訳の正確さは証明しない。hashを更新する前に実際の日本語文を見直す。
