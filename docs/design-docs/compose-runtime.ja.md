---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/design-docs/compose-runtime.md
source_sha256: 00a2191e6c895f9a69afac1c09964de607f81404cced38ce51fd8bc710489edd
---

[English（翻訳元）](compose-runtime.md)

# Compose ランタイム

この文書では、サービスの選択、設定の展開、独立したプロジェクトの起動、
readiness と endpoint の観測という Compose 共通の処理を説明します。
Docker/Podman の選択、設定の制約、削除前に必要な証拠は
[provider の設計](compose-providers.ja.md)を参照してください。
以下の呼び出し例では、共通の引数を Docker で示します。

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

Compose MVP のポート分離設計では、次のいずれかを強く優先します。

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
