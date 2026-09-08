---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/current-process-browser.md
source_sha256: 1f77e6e8cbbe6f8ce18c58da7f6834a01e765eef20e6e6e5cc064e308e0033cc
---

# 現在のprocess・browser・executorの正しさレビュー

[English](current-process-browser.md) · [監査Plan](../../exec-plans/active/repository-correctness-audit.ja.md) · [過去レビュー資料](history-process-browser.ja.md)

固定対象: `031869c8b9073b8e23bc17fbc55243666a52f557`。
Phase Aのみで製品・test sourceは変更していない。再現にはrepository外のGo overlayを用いた。
指摘は統合担当の処置判断まで未分類。本書はprocess runtime・Browser/CDP・execxの
限定したsource・再現確認を完了するが、全体監査表・native baseline・最終独立reviewを代替しない。

## 確認した不変条件の表

| 領域・不変条件 | 確認入口と証拠 | 結果・限界 |
| --- | --- | --- |
| Browser権限と分離 | CDP接続・discovery、loopback WebSocket、PID/version/flag、native callback、target session、page上限 | 既存identity負例は有効。誤browser採用の新経路は見つからず。同一userによる悪意のprotocol偽造は契約範囲外。 |
| Frameとstale node証明 | snapshot/approvedFrames/recheckFrames、owner census、action直前AX/node/hit、isolated focus、native closed-shadow | 前後origin/構造・stale入力guardを維持。load waitに下記混在document問題があるが後続入力は再検証する。 |
| Browser境界と完全性 | AX/DOM2048、frame32、page128、target/event512、semantic JSON1 MiB、field4 KiB、capture64 KiB/256件 | AX境界回帰は2047/2048/2049と複数・末尾空frameを検査。DOMは等値ではなく次node省略で判定。redaction後DOM不整合は下記。全組合せ実行済とはしない。 |
| Capture時間と省略 | 購読/enable前context、eventごとの期限、原子的unsubscribe残件/overflow、console省略flag | 既存期限・overflow・引数回帰は成功。helperだけの終了coverage限界HB18は残る。 |
| Browser effect/evidence barrier | 確認分類、入力/focus前action_performed、appのdurable run/provenance、型を区別するredaction | effect後不明・保存testを維持。app全体fence/finalize監査は統合担当。 |
| Process起動identityと復旧 | Prepare -> Start -> identity/receipt、型付き未生成、receipt不一致/復旧 | 不完全native identityは不明を維持。未生成解放は型付き証拠とPIDゼロ必須。完全identityが返ればreceipt欠落でも補償可能、不一致証拠は拒否。 |
| Process path/cleanup | 所有/path検査、予約canonical path、Destroy -> Terminate -> Observe -> state削除 | 全tree消滅を再検査しpath/symlink・予期しないfileは拒否。Windows retryは共有違反だけで毎回path再検証。一般TOCTOU監査は統合担当でtrusted-code限界を維持。 |
| Process port/readiness/log | OS動的予約の保持、起動前bind検査、最後のreadiness観測、独立startup fingerprint | 既存衝突/readiness/終了root/receipt-secret回帰は有効。listener解放後の外部bind競合は説明済で解決済としない。command-readiness再発はAUDIT-CLEANUP-001。 |
| Unix identity/cancel | birth/group証拠、signal前再検証、census制限付きretry、managed消滅待機 | managedの新たな偽不在は未確認。原子的PID/group signal不可の限界は説明済。通常Runnerのsignal成功だけではcensusせず下記仮説は未確定。 |
| Windows identity/cancel | suspended Job割当、session/nonce/birth、消滅前marker、過去PID優先順、正確なJob handleと空Job待機 | 過去修正を維持。Windows source/testの確認であり今回native実行はbaseline CIが担当。 |
| Platform横断error伝播 | OSRunner ExitError unwrap、tree不明/output不完全、named command/readiness呼出 | readiness呼出側が型付き不明を失うことを独立確認しAUDIT-CLEANUP-001を裏付ける。重複指摘にしない。 |

## AUDIT-REDACTION-002 — DOM証拠がadapterの最終size制限を越える

- 重要度: Medium（証拠上限の不整合。仕様上の留保は後述）。
- 処置: 未分類。
- 不変条件: 証拠変換後も意図した最終artifact予算を維持するか超過を明示・拒否する。
- 場所: `internal/app/browser.go`のDOM分岐、`redactBrowserJSON`後、
  `save("browser-dom", ...)`前。`internal/browser/cdp/snapshot.go:domSnapshot`と
  `internal/app/browser_redaction.go:boundRedactedBrowserSnapshot`を比較。
- 条件: set-textで短い`qz`のfingerprintを記録。合法custom-element名
  `qz-`42回（126 byte）のDOM nodeを2048個観測する。
  provider DOMは411,520 byteでadapterの1 MiB/名前128 byte以下。
- 観測: 公開`Service.Browser(dom-snapshot)`が1,099,648 byteの
  `browser-dom`を`Truncated=false`、durable run passedで保存する。
  secret漏えいではなく最終上限の問題。汎用save helperは16 MiBまで許す。
- 期待: 意図したDOM最終出力予算を確認しredaction後に適用する。
  adapterの1 MiBが保たれないのに保たれると示さない。
- 影響: adapter/capabilitiesのsnapshot上限に依存するcallerが、完全表示付きの
  より大きいartifactを受け取る。DOMからsemantic入力を許可する問題ではない。
- 既存coverage: semanticとconsole/networkのredaction後保存上限はtestするが、
  DOM最終byteのassertはない。
- 再現: Go overlayで`internal/app/browser_review_test.go`へ
  `TestAuditDOMPostRedactionBudget`を追加し既存`browserFixture`を使用。
  snapshot、set-text `qz`後、上記2048 nodeの確認済DOMを注入しdom-snapshotを呼ぶ。
  登録artifactの存在とencodingが1 MiB以下であることをassertする。
  実際は上記sizeでFAIL、0.079s。製品fileは無変更。
- 回帰: 公開app経由の保存artifact回帰を提案。処置判断待ち。
- 修正: Phase Aではなし。
- 検証: 決定的なapp全体再現、保存artifact実読取、passed runとtruncation確認。
  app変換の証明にnative Chrome再現は不要。合法custom名を使い未知DOM fieldに依存しない。
- 関連: 過去HB16/HB20（上限適用後の変換という同型）。
- 見逃し分析: 検出S9、最早S4。appは元からredactionと保存を両方担当する。
  COMPOSITION_GAP + BOUNDARY_GAP。過去修正はsemantic/capture最終出力を測ったが
  全browser artifact種別を列挙しなかった。不足oracleは最終DOM byte長。
  予防策はartifactごとの最終encoding上限表でS4検出を期待。
  実装・証拠は処置待ち、既存guardはsemantic/captureのみ。

留保: product文書が1 MiB以下と明示するのは**semantic JSON**で、
DOMはboundedと説明するが最終DOMの独立数値はない。adapterはDOM JSONが1 MiBを
越すと拒否しcapabilitiesは「1 MiB snapshots」と説明する。
再現した不整合は確定だが、durable契約で最終DOMにも同じ上限を課すかは処置判断が必要。
semantic限定の文章を明示DOM仕様として引用しない。

## AUDIT-STALE-001 — Load waitが別documentの証拠を返せる

- 重要度: Medium。
- 処置: 未分類。
- 不変条件: wait成功の証拠は成功predicateを確立したdocumentを記述する。
- 場所: `internal/browser/cdp/client.go:wait`の`case "load"`。
- 条件: snapshot自身の取得前後証明が完了した後、
  `Runtime.evaluate(document.readyState)`前にnavigationでdocumentが変わる。
- 観測: 新documentのreadyState `complete`で成功し、旧snapshot/tokenを返す。
  load分岐は評価後にidentity比較せず、URL分岐にはその検査がある。
- 期待: load状態の証明をsnapshot documentに結び付けるか、既存期限内で変更時にretryする。
- 影響: load wait成功にstale semantic証拠を返す。後続入力は別途documentを検証するため、
  この再現は他nodeへのstale入力を示さない。
- 既存coverage: snapshot中frame変化とURL wait identityはtestするが、
  追加のload評価段階をまたぐ回帰がない。
- 再現: overlayで`internal/browser/cdp/snapshot_test.go`へ
  `TestAuditLoadWaitDoesNotMixDocuments`を追加。protocol fixtureがsnapshot中はloader
  `old`を返し、Runtime.evaluateで`new`へ変え`complete`を返す。
  mutation到達をassertし戻りsnapshot tokenと再取得frameDocumentを比較する。
  FAIL、0.004s。戻り値`main:old:<同じURL digest>`に対し現状態は`main:new:<同じURL digest>`。
- 回帰: componentのnavigation境界fixtureを提案。処置判断待ち。
- 修正: Phase Aではなし。
- 検証: 決定的protocol component再現。native timing再現は未実行。
  mock独自fieldではなく説明されたloader変化と通常readyStateを用いている。
- 関連: 過去HB10/HB17（複数call観測前後のidentity検査）。
- 見逃し分析: 検出S9、最早S3。CONCURRENCY_GAP + COMPOSITION_GAP。
  snapshot自身の証明が後続predicate callも保護すると考えた。
  既存mutation testは追加段階前で止まる。全複数call wait predicateを列挙し
  観測・判定間にnavigationを注入すればS3検出が期待できる。
  guardrail実装・証拠は処置判断待ち。

## AUDIT-CLEANUP-001を裏付けるexecutor証拠

統合担当のreadiness再現の前提となる型付きexecutor契約を独立確認した。
Unixの`runProcessTree`はWait後group停止失敗で`ErrProcessTreeUnconfirmed`をjoinする。
WindowsもJob停止失敗、空Job確認未完、abort cleanup失敗で同じ型を返す。
`OSRunner`はunwrap可能なExitErrorで包む。`runCaptured`はoutput pump/drain失敗に
`ErrOutputIncomplete`をjoinする。任意に注入した文字列だけでなくnativeの失敗経路である。

`app/readiness.go:runProbe`はこれを`last`へ保存しretryした後`%v`で文字列化する。
named commandにある型付きrunning barrierと異なる。
統合担当は公開Create経由でretry・source削除・RELEASEDを再現した。
過去MVP HM10/HM11と同じcaller側安全契約の欠落であり統合担当の指摘を基準とする。

## 否定または未確定の仮説と限界

- **DOMちょうど上限で偽truncation:** source制御flowからは再現しない。
  省略する次nodeがある時にcapを検査し、次documentが空なだけでは付かない。
  名前短縮でtruncationを付け後続documentを止めるのは実省略である。新不具合としない。
- **Managed停止成功だけで未確認削除:** source追跡で否定。
  `Client.Destroy`はTerminate後Observeを呼び、alive/errorならpath検証済state削除前に拒否する。
- **Closed-shadow証明がpage overrideを信用:** 既存native fixtureとisolated world解決で否定。
  外向き探索は全root/hostを検査し4入力全てにoverlay負例がある。
- **通常Unix Runnerのsignal成功が全tree消滅証明:** sourceはWait後SIGKILL成功/ESRCHを
  group censusなしで受け入れる。Windows空Job検査やmanaged待機とは異なる。
  既存late-write testは1秒後を検査する。害のある残存process時間窓は再現しておらず、
  新たな確定findingや重要度を割り当てない。証明済安全とせず明示的調査限界とする。
- **過去回帰が全て公開組合せを検査:** 否定。HP05/HB19はhelperのみ、
  HB18はhelper終了coverage、HM06は元test特定が必要。
  Reconcileは現在昇格前にdurableな`lease_ready` eventを確認しており、
  対応付け未完は元の製品不具合が残る証拠ではない。
- Windows/macOS実行、Docker/Podman/Android統合、全体保存・fence・path不変条件は統合担当。
  本source確認はcross-buildや過去成功CIで今回の必要検証を代替しない。

## Local検証

選択した過去回帰のrace実行は成功し詳細を過去資料に記録した。
続くprocess・execx・CDP全体race呼出はGo cacheからPASSを返したため、新規実行とは数えない。
2つのoverlay再現は期待通り失敗しrepository製品・test fileを変更していない。
ここでは修正・commit・push・thread操作を行っていない。
