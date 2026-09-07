---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/design-docs/reconciliation-and-gc.md
source_sha256: fde8bf1ba2534827fdece6c12b77377027ebe74fee8535d464f3020c9a17d556
---

[English（翻訳元）](reconciliation-and-gc.md)

# Reconciliation とガベージコレクション

## Reconciliation の規則

reconciler は desired record と観測リソースを比較し、見つけたものをすべて直接変更するのではなく、診断とイベントを生成します。

例:

```text
lease ready + project absent        -> degraded
lease active + worktree absent      -> degraded
lease released + project present    -> cleanup_failed/quarantined
no lease + project with agent label -> orphaned external resource
expired active lease                -> stale candidate
```

可能な場合は、生成リソースに Compose の project label に加えて `agent-env` label を付けます。

## GC の規則

GC 候補の条件:

- lease の期限切れから猶予期間が経過している
- 実行中のコマンドがない
- 最近の heartbeat がない
- リソースの識別情報がレジストリ記録と一致する
- worktree に予期しない追跡対象ファイルの変更がない

既定の `agent-env gc` は提案する操作を表示します。`--apply` が実行します。

artifact は稼働中ランタイムリソースとは独立した保持期間を持ちます。コンテナと worktree が解放されたという理由だけで証拠を削除してはいけません。

---

## 実装済みの安全条件

既定の policy は、期限切れ後の猶予が 5 分、heartbeat の猶予が 1 分です。Domain の対象判定は両方と保護対象の lifecycle state を確認します。App は preview 時に、永続化された `running` コマンド記録がある lease を対象から除外し、apply 前に操作ロック下で再確認します。期限切れや未完了の作業は reconciliation の診断から確認でき、未完了のコマンド行を自動的に完了扱いにはしません。

Destroy は対象を厳密に特定した実行中の named run ID だけにキャンセルを要求し、最大 10 秒待ちます。named run の実行担当は、子孫プロセスの終了を確認し、証拠を確定してから run の終了状態を公開します。終了未確認エラーや証拠確定の失敗があれば、レジストリの run は `running` のままです。削除には lease ロックとコマンド完了の確認が必要で、古い running 記録は force も阻止します。時間制約と復旧の限界は[信頼性](../RELIABILITY.ja.md)を参照してください。
