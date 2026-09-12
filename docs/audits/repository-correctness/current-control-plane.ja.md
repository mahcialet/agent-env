---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/current-control-plane.md
source_sha256: 8dd2e0c4e0a63dc4454b2998ff1ccf89e4689ce11833c3259a02c163ccdea2f5
---

# 制御処理の現行監査

[English](current-control-plane.md)

対象は `031869c8b9073b8e23bc17fbc55243666a52f557`。Phase A としてコード確認と隔離 overlay 再現だけを行った。全 unit/race/実 provider の基準検証は補助証拠であり、全不変条件を証明するものではない。

## 対象と不変条件

| 対象 | 確認した条件 / 実装入口 | 証拠と結果 |
| --- | --- | --- |
| Manifest と計画 | 厳格な解析、移植可能な相対パス、時間・argv、digest、stack のルートと直接・間接の全依存先の選択、provider 記録。`config.Parse/Validate`、`app.BuildPlan` | config/application/browser/process の不正な設定を拒否する既存テストと基準 race。未知フィールド、alias、大小文字衝突、source 脱出、APK 出力重複を拒否。新規不具合は未確認。 |
| Domain と policy | source の不変属性、GC 時間境界、明示的隔離。`SourceDigest`、`GCEligibleWithGrace`、TTL | 等値・ゼロ・負値の既存テスト。`Transition` は全状態更新を強制する境界ではなく、処理本体の直接代入も外部作用と照合した。 |
| SQLite | lease/source/process/Android 予約のトランザクション、準備済み identity の不変性、rollback、lease ごとの行。`Create/Save` と予約処理 | 正規化行を含む検証で非ゼロの PID identity の削除・再割当と予約復活を拒否。独立接続・プロセスを使う既存テストを基準 race に含む。DB だけから実資源の不存在を推論しない。 |
| Lock と中断 | 非公開 capability、更新失敗時の中断、書込み前の token/期限検査、destroy/run 調整。`AcquireContext`、`execBound`、`runWithCancellation` | lock 奪取と subprocess の既存テスト。`WithoutCancel` でも token 検査を回避できない。readiness 呼出し元で終了未確認の型が失われる: AUDIT-CLEANUP-001。 |
| Create と readiness | source/runtime 作用前の intent、起動前の設定保存、Exists AND Ready、未完了 runtime ごとの期限 | `Create`、`allocate`、`waitReady`、`probeReady` と混在構成の検証。command readiness に AUDIT-CLEANUP-001。 |
| Cleanup / reconcile / GC | 作用前の source identity・追跡差分、独立 sibling の削除、永続化失敗で停止、不存在確認後の解放 | `cleanup`、`Reconcile`、`GC`、`previewDestroy`、`Inventory` と既存 sibling/fence/foreign inventory テスト。実行中の名前付きテスト行は source を保護するが、readiness には同等の永続記録がない。Reconcile は repository command を再実行しない。 |
| Git とパス | pinned detached 登録 root、filesystem identity、symlink 脱出、消失 worktree の登録、追跡差分確認後のみ force | `gitcli.Inspect/Remove`、`paths.Within`、state-root テスト。削除後の再検査は呼出し元にある。悪意ある同時パス置換の全競合まで保証したとは扱わない。新規再現はない。 |
| Command と証拠 | 開始前の run intent、終了確認と証拠完了後のみ terminal 行、chunk を跨ぐ秘匿、通常ファイルと symlink 制約 | `Service.Test`、`collectTestArtifact`、`evidence.Redactor/AtomicWrite`。既存 stream-error/cancel 保護は名前付きテストに適用され、readiness で共有していない。Redactor は最長一致し置換を再走査しない。上限は呼出し元の責務で、mobile の再発は別記。 |
| Endpoint と health | protocol を保つキー、現行 provider 検査、process port 展開、HTTP status/期限、command 再実行禁止 | `Endpoints`、`probeHealth`、endpoint-config テスト。TCP/UDP を区別し実 Docker/Podman でも境界を検証。新規不具合は未確認。 |
| 文書 | 表示アンカー、翻訳メタデータと索引、歴史的除外範囲 | 過去の文書検査の修正を、docsCheck全体を通すテストで再確認。[文書監査](documentation.ja.md) の AUDIT-DOCS-001 を参照。native の主張は CI と照合。修正候補版の native 検証は未実施。 |

この表は範囲を区切った確認であり、全スケジュールや悪意ある filesystem 競合の列挙ではない。mobile、process/browser、release/provider の内部は別付録で扱う。

## AUDIT-CLEANUP-001

- 重要度: High。採否: Phase B 待ち。
- 不変条件: command のプロセス群終了が未確認なら再試行・外部作用を止め、source と永続的な削除禁止記録を保持する。
- 場所: `internal/app/readiness.go` の `runProbe`、`internal/app/lifecycle.go` の allocation エラー処理。
- 条件: command readiness が `execx.ErrProcessTreeUnconfirmed` を返す。実 Unix の group 終了失敗、Windows Job 終了・空状態確認失敗でも発生する。`ErrOutputIncomplete` も同じ経路で型が失われる。
- 観測: `Service.Create` 全体で7回再試行し、source を削除して `released` となった。期限後の probe artifact 書込みが失敗し、返却エラーは context deadline exceeded のみ。別タイミングでは最後の原因を `%v` 化するため `errors.Is` 分類がやはり失われる。
- 期待: 終了・証拠が不完全なら再試行せず、安全性の型を保持して隔離し worktree を残す。後続の明示 destroy/GC も未解決の command を無視しない。
- 影響: 生存し得る子プロセスの作業ツリーを削除する。再試行成功で以前の不確実性を隠す場合もある。診断だけの問題ではない。
- 既存検証: MVP の名前付きテストには不完全なプロセス群・出力と allocation の型付き防護があるが、command readiness を通す Create 全体の失敗注入がない。process readiness の secret テストも設定検証に限定される。
- 再現: 一時 Go overlay の `TestAuditReadinessUnconfirmedTree`。通常の lifecycle fixture と製品の型付きエラーを返す Runner を使用。`go test -overlay <temporary-overlay> ./internal/app -run TestAuditReadinessUnconfirmedTree -count=1 -v` は失敗（0.024秒）。`state=released sources=0 calls=7`、末尾操作は `remove:self`。製品ファイルは未変更。
- 独立確認: executor 担当が Unix/Windows の実失敗から OSRunner の ExitError 包装までを追跡し、注入値が実装契約に含まれることを確認した。
- 原因: 終了未確認を通常の probe 失敗として再試行し、秘匿・期限処理で型を失う。後続削除を止める永続 command 行もない。
- 修正案: 診断を秘匿しつつ安全性エラーを保持し即座に停止する。既存 run 行を使って readiness intent と完了を永続化し、後続 destroy も防護する。自動的に不存在を推定しない。
- 修正と、その動作を確認するテスト・最終検証: Phase B/C 待ち。両エラー、診断の secret、再試行停止、source 保持、後続 destroy を検証し、正常・通常失敗の挙動も保つ。
- 関連: process/browser 歴史付録の MVP HM10/HM11、不完全なプロセス群・出力の過去修正、readiness の補助関数だけの検証。
- 見逃し: 検出 S9、最も早い現実的防止 S4。型付き Runner 契約と Create 隔離分岐は既にあり、全体入口で失敗注入できた。S2/S3 は executor と名前付きテストを別々に確認し、S6 に呼出し元の合成テストがなく、S7/S8 は全利用箇所を辿らなかった。COMPOSITION_GAP、FAILURE_INJECTION_GAP、HELPER_ONLY、REVIEW_CHECKLIST_GAP。
- 防止策: 安全性分類の共有、Create 全体で終了・出力が未確認の場合をそれぞれ注入するテスト、永続 command による削除制御。今後 S6 より前の S3/S4 で検出する。まだ未実装で共通ポリシーは変更しない。

## Phase C の解決と独立レビュー

AUDIT-CLEANUP-001 は ACCEPT とし `readinessCommand`/`runProbe` で修正した。command の各試行前に running 行を保存し、終了・出力確認、artifact 登録、terminal 行の保存まで完了して初めて終了扱いにする。診断を秘匿しつつ安全性の型を保ち、未確認なら running 行と source を保持して Create を隔離する。強制 Destroy と GC は既存の永続 run 保護に従う。終了確認済みの通常失敗は再試行でき、成功後は failed と passed の terminal 行が残る。

`readiness_safety_test.go` で両未確認エラー、秘匿診断、1回だけの実行、source/running 行の保持、後続の実際の強制 Destroy、通常失敗から成功と解放を確認する。終了・出力が未確認の場合に再試行と削除を止めるテストは、修正前の実装で失敗した。独立レビューで最初の修正が永続中断要求後も再試行する問題を発見し、実 Store.RequestRunCancel の再現が0.119秒で失敗した。context.Canceled の型を保持して停止すると同じ再現が0.110秒で成功し、`TestReviewReadinessCancellationDoesNotRetry` として残した。別担当は GC/Destroy の lock 内再確認も辿り、対象 race を3回実行（2.559秒）した。途中の失敗した修正を最終成功だけで隠さない。候補版全体の検証は別途記録する。
