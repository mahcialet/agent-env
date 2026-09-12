---
status: completed
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/exec-plans/completed/repository-correctness-review.md
source_sha256: c94623a4978d8926a1af4c72e059e420ded197cbf91cdc2da80fc60581f2d86a
---

# PR #11 correctnessレビューへの対応

[English](repository-correctness-review.md)

期待branch: `audit/repository-correctness`。
本PlanをPR #11追加レビューの実行基準とする。
[完了済み監査](../completed/repository-correctness-audit.ja.md) の元の証拠は保持し、履歴を書き換えずに対応する。

## 目的 / 全体像

PR #11の2指摘を直す。page列挙から作成の間のpopupで128上限を超える問題と、
heading anchor抽出がinline codeの内容を置換して有効linkを拒否する問題。
検証した修正をpush後、両Threadへ返信してResolveする。

## 進捗

- [x] (2026-09-09) 6b13cd4のcleanなPR branchと未解決2Threadを確認した。
- [x] (2026-09-09) 両不具合を再現して修正。不完全/type欠落列挙と全target不在確認の誤りを検出するテストも追加した。
- [x] (2026-09-09) 独立レビュー、harness、race、関連native CIを実施する。
- [x] (2026-09-09) f10ecd4/de6da4f/ab71b64をpush。両Threadへ返信しResolve状態を確認した。
- [x] (2026-09-09) 成果を記録し、日英Planを完了へ移す。

## 想定外の発見

- 2026-09-09 完了時点: 下記検証は未変更Windows再実行attempt2を含め全完了。過去の継続中の記述はその時点の記録で、受入残件はない。

- 2026-09-09: ab71b64のPR Verify34298063439全job、Browser native両run、Release preview34298063300は成功。同HEADのpush Verify34298060578 Windows1.27は未変更のTestRunnerReapsOrdinaryDescendants/timeoutで失敗し、helperの起動確認出力より先に300ms timeoutとなった。fixture起動前提の失敗で、子孫がcleanup後に残った証拠ではない。同HEADのPR Windows1.27 jobは成功。製品・テストを変更せず失敗jobを再実行し、両結果を残す。具体的なscheduler遅延原因は確定していない。

- 2026-09-09 独立レビュー: 初期popup修正は全harness/raceとLinux Browser integrationに成功したが、作成IDを持ちtypeが欠落した列挙項目をfilterで落として不在と誤認した。typeが欠落した列挙を不在の証拠として扱わないことを確認する隔離テストは、0.024秒で失敗してこの不具合を検出した。target identity/typeの完全性を検査し、pageだけでなく全targetに対して正確な不在を確認する。push前に再検証する。Markdown独立race付き5回は2.663秒成功し、既存の表示された出典link処理は維持された。command確認lockのrace付き10回は元のassertionを全保持して4.390秒成功。

- 2026-09-09: inline codeのanchorを正しく認識するテストは修正前に4fixtureで失敗し、block/inline分離後のrepoctl package全raceは8.915秒成功。途中の全harnessはCDPテスト編集中のformat-checkで停止したため、安定後に再実行する。PR Verify34296197727のWindows Go1.26はTestNamedCommandFailuresRetainEvidence/sleepの最後の新規lock解放確認だけで失敗。commandのtimed_out/run/artifact検証は成功済み。確認lockのTTLは1秒で実commandは2分。同HEADのpush CIは成功。確認用TTLだけ1分にし、lock利用可能性の検査が秒未満のDB・scheduler遅延に依存しないようにする。全assertionと製品fenceは維持しnative CIで再検証する。

現行inline span処理はcodeの内容を単純に削除せず `code` に置換する。
表示されるanchorとの不一致という指摘は妥当。

## 判断の記録

- 2026-09-09、統合担当: 作成後は完全なtarget列挙で検証する。元の全target（workerを含む）に存在しなかった返却IDだけを補償削除し、直前にnative所有を再確認する。close応答と全target中の正確な不在が揃って初めて既存confirmedErrorで確認済み失敗を返す。所有・identity・protocol・不在の確認失敗は未確認のまま保持し、外部popupを削除しない。最終全harnessと全race成功（CDP8.233秒、CLI5.624秒）。実Linux Browser native race10.362秒成功。元Windows jobも未変更で再実行attempt2に成功し、確認用lock遅延の切り分けを補強した。修正版Windows CIは引き続き必要。

- 2026-09-09、統合担当: 3963647055と3963647059を採用。
  既存CDP lifecycle所有境界と非表示heading検査を維持する。
  block除外とinline文字処理を分け、link/provenance抽出の安全性は維持する。

## 成果と振り返り

2026-09-09に完了。PR #11の両指摘を修正し、返信・Resolveまで完了した。

- `f10ecd4`: heading anchorでinline code文字を保持し、既存の非表示blockと
  出典/link処理は維持した。docsCheckで有効なheading anchorへのlinkを受理し、
  存在しない置換anchorへのlinkを拒否する4件のテストは修正前に失敗し、
  独立した関連race付き5回は2.663秒成功。
- `de6da4f`: command後のlock解放確認用TTLだけ1秒から1分へ変更した。
  状態・証拠・再取得・解放の全assertionを保持。対象race付き10回4.390秒成功。
  元のWindows失敗jobも未変更の再実行で成功した。
- `ab71b64`: page作成後に再列挙し、超過または列挙確認不能なら新規targetだけを補償削除する。
  native所有、close応答、全target種別での正確な不在を必須とする。
  欠落・曖昧な証拠は未確認のまま保持し、既存targetと並行popupを残す。

元のpopupによる上限超過を検出するテストは修正前に失敗した。独立レビューではtype欠落をfilterで落として
不在と誤認する問題を発見し、修正後の17ケースと127/128/129境界は
独立race付き5回2.224秒成功。全 `repoctl check` と全 `go test -race ./...` は成功。
最終CDP race8.233秒、CLI race5.624秒、実Linux Browser native race10.362秒。

製品revision `ab71b64f3867ccced2a304640ade22db56706adf` でPR Verify34298063439、
push Verify34298060578（attempt2）、Browser native34298063305/34298060667、
Release preview34298063300はすべて成功。native Windows/macOS/Linuxと配布archiveの
3OS smokeを含む。push Windows1.27の初回は未変更の300ms helper起動前提で失敗したが、
同HEADのPR jobと未変更の再実行は成功した。fixture起動遅延への感度として残し、
子孫が生存した証拠や、検査を黙って緩めた成功とはしない。

両Threadにcommit・テストを示して返信し、isResolved=trueを確認した。
採用したレビュー指摘の残件はない。元の監査履歴を保持し、製品dependencyやlifecycle所有者は変えず、
検証した製品revision以後は文書完了のみを行う。今後の削除証明では表示用filterより前の
全identity一覧を保持する必要がある。そうしないと種別除外を不在と誤認する。

## 背景と構成

`internal/browser/cdp/client.go` がpage作用と列挙を担当し、appがbrowser実行の確実性を永続化する。
`tools/repoctl/main.go` はfragment、`translations.go` は非表示Markdownと出典linkを処理する。
テストは実際の利用先に到達させる。

## 作業計画

protocol件数・正確なtarget IDでpopup競合を再現し、所有再確認の後に作成targetだけを削除する。
確認できない結果はそのままエラーとして扱う。docsCheckでinline codeの有効anchorと誤った置換anchorを再現し、
block処理を再利用しながらheadingのinline文字を保持する。検証後に反映する。

## 具体的な手順

`go test -race ./internal/browser/cdp ./tools/repoctl`、
`go run ./tools/repoctl check`、`go test -race ./...`、既存の実Browser native integrationを実行する。
PR branchをpushしてnative CIを確認する。修正を確認可能にしてからGitHub Thread返信・Resolveを行う。

## 検証と受け入れ

- page作成後に再列挙し、超過時は自分が作成したtargetだけを削除する。
  並行popupと兄弟pageは残し、削除失敗をエラーとして保持する。
- inline codeの正しいanchorは通し、存在しない置換anchorは拒否する。
  fence/comment/indentの偽headingは引き続き拒否する。
- 全harness、関連race、独立レビュー、native Browser証拠が成功する。
- 両Threadに具体的返信がありResolve済み。最終日英文書検査が成功する。

## 冪等性と復旧

既存commitを保持しforce pushしない。target削除を証明できなければ不確実性と証拠を残し、作成成功としない。

## 成果物と注記

返信: [page rollback](https://github.com/mahcialet/agent-env/pull/11#discussion_r3963707422)、[heading anchor](https://github.com/mahcialet/agent-env/pull/11#discussion_r3963707551)。両Resolve応答はtrue。最終独立popup/rollback17ケースと境界テストはrace付き5回2.224秒成功。候補ab71b64のCI: Verify34298060578/34298063439、Browser34298060667/34298063305、Release preview34298063300はすべて成功し、結果を照合済み。

本Planに再現、検証、Thread結果を記録する。開発者固有SDK pathや秘密値を恒久文書へ含めない。

## インターフェースと依存

新dependency、CLI flag、runtime lifecycle所有者は追加しない。
Markdownのlink抽出はcode例中のlinkを引き続き無視する。
