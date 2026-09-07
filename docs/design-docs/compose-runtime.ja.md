---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/design-docs/compose-runtime.md
source_sha256: bd2af518c3e30976eca3d78e2262f6911c3a8eb00a2a688ebca25480479b9bbc
---

[English（翻訳元）](compose-runtime.md)

# Compose ランタイム

## 呼び出し

引数を直接組み立てます。代表的な呼び出しは次のとおりです。

```text
docker compose
  -p <normalized-project-name>
  --project-directory <absolute-worktree-project-directory>
  -f <absolute-compose-file-1>
  -f <absolute-compose-file-2>
  up -d <selected-services...>
```

`up` の前に `docker compose config` を実行し、展開結果とハッシュを保存します。

## サービス選択

解決したコンポーネント依存関係の閉包から `compose_services` を統合し、決定的な順序を保ちながら重複を除きます。

コンポーネント名と Compose サービス名が同じだと仮定してはいけません。

## Readiness

少なくとも次をサポートします。

- Compose コンテナの状態/health の検査
- manifest で宣言した HTTP probe
- argv 配列として宣言した command probe

上限付きの timeout を使い、失敗した probe を診断情報に含めます。

## ポート

MVP では次のいずれかを強く優先します。

- ホストへ公開せず、Compose 内でテストを実行する
- Compose で設定した動的なホスト公開
- 宣言した endpoint の target ポートに対する生成済み Compose override

固定ホストポートの競合が分かっているときに、黙って第 2 の lease を起動してはいけません。分離した override を生成するか、`up` の前に対処可能な診断とともに失敗させます。

生成した override は lease の generated ディレクトリ以下に置き、その内容の digest を記録しなければなりません。

## 検査

次を記録します。

- Compose プロジェクト名
- Docker context
- 選択したサービス
- コンテナ ID
- イメージ参照と、取得可能ならイメージ ID/digest
- プロジェクトに帰属する network と volume
- health/status
- 検出した endpoint の対応関係

プロジェクトリソースは、明示したプロジェクト名と Compose label で識別します。推測したコンテナ名の書式だけに依存してはいけません。

---
