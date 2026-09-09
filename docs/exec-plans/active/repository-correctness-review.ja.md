---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/exec-plans/active/repository-correctness-review.md
source_sha256: 7d7285f44a7257a8ea5d56921156b4e036c7eb90a453c67520ca85f27041768d
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
- [x] (2026-09-09) 両不具合を再現して修正。不完全/type欠落列挙と全target不在確認の回帰も追加した。
- [ ] 独立レビュー、harness、race、関連native CIを実施する。
- [ ] 検証済み修正をpushし、両Threadへ返信してResolveする。
- [ ] 成果を記録し、日英Planを完了へ移す。

## 想定外の発見

- 2026-09-09 独立レビュー: 初期popup修正は全harness/raceとLinux Browser integrationに成功したが、作成IDを持ちtypeが欠落した列挙項目をfilterで落として不在と誤認した。隔離負例は0.024秒で失敗。target identity/typeの完全性を検査し、pageだけでなく全targetに対して正確な不在を確認する。push前に再検証する。Markdown独立race付き5回は2.663秒成功し、既存の表示された出典link処理は維持された。command確認lockのrace付き10回は元のassertionを全保持して4.390秒成功。

- 2026-09-09: inline codeのanchor回帰は修正前に4fixtureで失敗し、block/inline分離後のrepoctl package全raceは8.915秒成功。途中の全harnessはCDPテスト編集中のformat-checkで停止したため、安定後に再実行する。PR Verify34296197727のWindows Go1.26はTestNamedCommandFailuresRetainEvidence/sleepの最後の新規lock解放確認だけで失敗。commandのtimed_out/run/artifact検証は成功済み。確認lockのTTLは1秒で実commandは2分。同HEADのpush CIは成功。確認用TTLだけ1分にし、lock利用可能性の検査が秒未満のDB・scheduler遅延に依存しないようにする。全assertionと製品fenceは維持しnative CIで再検証する。

現行inline span処理はcodeの内容を単純に削除せず `code` に置換する。
表示されるanchorとの不一致という指摘は妥当。

## 判断の記録

- 2026-09-09、統合担当: 作成後は完全なtarget列挙で検証する。元の全target（workerを含む）に存在しなかった返却IDだけを補償削除し、直前にnative所有を再確認する。close応答と全target中の正確な不在が揃って初めて既存confirmedErrorで確認済み失敗を返す。所有・identity・protocol・不在の確認失敗は未確認のまま保持し、外部popupを削除しない。最終全harnessと全race成功（CDP8.233秒、CLI5.624秒）。実Linux Browser native race10.362秒成功。元Windows jobも未変更で再実行attempt2に成功し、確認用lock遅延の切り分けを補強した。修正版Windows CIは引き続き必要。

- 2026-09-09、統合担当: 3963647055と3963647059を採用。
  既存CDP lifecycle所有境界と非表示heading検査を維持する。
  block除外とinline文字処理を分け、link/provenance抽出の安全性は維持する。

## 成果と振り返り

実装・受入は継続中。

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

本Planに再現、検証、Thread結果を記録する。開発者固有SDK pathや秘密値を恒久文書へ含めない。

## インターフェースと依存

新dependency、CLI flag、runtime lifecycle所有者は追加しない。
Markdownのlink抽出はcode例中のlinkを引き続き無視する。
