---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/supplemental-browser.md
source_sha256: ea9a0b1d4d287697121b11e0b3824f33fc19a972bd6951767c6c57ddbcedc1e6
---

# Browser追加レビュー: 遅れて確認したPR #10指摘

[English](supplemental-browser.md) · [監査index](index.ja.md) · [台帳](findings.ja.md)

本書は既存repository監査の一部であり、別のPR対応作業ではない。
最終資料照合で、完了browser Planにないmerge後PR #10未解決commentを4件発見した。
ここではCDPの3件を記録し、[CLI結果の残る1件](supplemental-cli.ja.md)は別資料に記す。
全て固定revision `031869c8b9073b8e23bc17fbc55243666a52f557`で再現した後、
統合担当が追加Phase B checkpointで修正を採用した。
元発見はS8、監査再現はS9。本書の作成でthread返信・Resolve・remote変更は行っていない。

## 再現の分離と共通証拠

追跡済CDP Go source/testを全て固定Git revisionから一時Go overlayへ読み込んだ。
追加testは実公開`Client.Observe`またはprotocol componentの`wait`/`act`と
既存fixtureを使い、Phase A中は監査製品fileを変更しなかった。3回帰全て失敗した。
恒久testも今回修正前の監査候補版で失敗し、先行監査修正では直っていないことを確認した。
入力dispatchはmock call記録で証明し、この安全な再現で実browserへ有害clickを送ったとはしない。

## AUDIT-BOUNDARY-003 — Page作成が対応page上限を越える

- 重要度: Medium、処置: ACCEPT。
- 参照: [PR comment 3963154175](https://github.com/mahcialet/agent-env/pull/10#discussion_r3963154175)。
- 不変条件: 上限付きresource作成は結果が対応件数を越すならeffect前に拒否する。
- 場所: `internal/browser/cdp/client.go`のpages列挙とpage-create。
- 条件: 検証済所有browserにpage targetがちょうど128件ある。
- 観測: 公開page-createがTarget.createTargetを送り成功し129件となる。
  後続の明示page-closeは既存列挙上限で失敗しclose commandを送らない。
  adapter自身が対応管理経路を使えなくした。
- 期待: 128件では作成effect前に拒否し、127から128への作成と128時の既存page closeは許す。
- 再現: `TestSupplementPageLimitPreventsIrrecoverableGrowth`が正しいPID/flag/discoveryと
  状態を保持するtarget一覧で公開Observeを通る。固定版ではcreate error nil、
  close errorはpage上限、create呼出1、close呼出0、件数129。
- 見逃し: 最早S3、COMPOSITION_GAP + BOUNDARY_GAP + NEGATIVE_FIXTURE_GAP。
  正しい列挙上限が後続作成effectを制限しなかった。上限超過listだけでは境界での作成を通らない。
- Phase C: 共通128-page定数を列挙と作成前検証で使い、上限以上でtarget作成を送らない。
- Guardrail: 恒久127/128/129表で作成・close回数と最終件数を検査し、
  近傍正常例と従来の上限超過拒否の両方を確認する。将来期待検出S3。
- 限界: 外部から129-page状態を与えたbrowserは従来通り上限で拒否する。
  adapter自身がその状態を作ることを防ぐ修正で、無制限discoveryや復旧scopeの追加ではない。

## AUDIT-STATE-001 — Ignored AX nodeがaccessibility waitを満たす

- 重要度: Medium、処置: ACCEPT。
- 参照: [PR comment 3963154186](https://github.com/mahcialet/agent-env/pull/10#discussion_r3963154186)。
- 不変条件: text/role・gone waitはChromiumがignoredとしたnodeを除くaccessible semantic nodeを使う。
  不完全証拠は引き続き不在証明にならない。
- 場所: `internal/browser/cdp/client.go:wait`のtext/gone走査。
- 条件: roleとnameが一致するignored nodeだけがsnapshotにある。
- 観測: textが偽成功し、goneはそのnodeを一致と数えてtimeoutする。
  snapshot自体はIgnored bitを正しく保持している。
- 期待: 両predicateでignoredを除き、既存truncated-gone拒否と通常accessible一致は維持する。
- 再現: `TestSupplementIgnoredAXCannotSatisfyWait`がNeedle名のignored buttonでwaitを呼び、
  AX取得到達もassertする。固定版textは成功、goneはdeadline exceeded。
- 見逃し: 最早S3、COMPOSITION_GAP + INVARIANT_GAP + NEGATIVE_FIXTURE_GAP。
  snapshotは正しい状態を保持しactionもignored入力を拒否したが、別consumerのwaitに
  semantic対象条件を適用していなかった。
- Phase C: role/name一致前にIgnoredをskipする。goneの完全snapshot条件は変えない。
- Guardrail: text偽陽性・gone偽陰性の両回帰を追加し、既存visible-text・truncated-gone回帰を維持。
  将来期待S3。DOM selectorや任意scriptへのfallbackは追加しない。

## AUDIT-STALE-002 — Pressed変化がaction fingerprintを無効化しない

- 重要度: Medium、処置: ACCEPT。
- 参照: [PR comment 3963154191](https://github.com/mahcialet/agent-env/pull/10#discussion_r3963154191)。
- 不変条件: snapshot後に意味のある対象状態が変われば、role/name/backend/boundsが
  同じでもsemantic actionは拒否する。
- 場所: `internal/browser/cdp/snapshot.go`のnode fingerprint用state allowlist。
  `actions.go`がそのfingerprintを使う。
- 条件: toggle buttonのpressedがsnapshot後、入力検証直前のnative所有検証callbackで変わる。
- 観測: pressedがStates/fingerprintに含まれずstale clickをdispatchしperformed=true・error nilとなる。
- 期待: false/true/mixed変化で保存fingerprintを無効化し入力前に拒否する。
  別browserや別nodeへの攻撃を証明したものではない。
- 再現: `TestSupplementPressedStateRefusesStaleInput`は現行所有mutation回帰をpressed変更へ適用。
  mutation callback到達と入力回数をassertし固定版で失敗する。
- 見逃し: 最早S3、NEGATIVE_FIXTURE_GAP + ORACLE_COUPLING + COMPOSITION_GAP。
  checked回帰だけでは別protocol propertyのpressed保持を証明しない。
  fingerprintの正しさはhash実装だけでなくproducer property集合に依存する。
- Phase C: 既存の制限付き`axState`変換へpressedを追加。
  checked・selected・expanded・readonly・required・focusable・focused・multiselectableは
  既に保持されることを確認した。自由文value・description・relationshipは追加しない。
- Guardrail: 恒久表でpressed bool/mixedと既存8flagを検査し、注入到達・無入力をassertする。
  将来期待S3。任意state文字列拒否は維持する。

## Phase C検証と残る最終検証

追加protocol回帰のrace検査3回はPASS、1.668s。独立したread-only reviewに阻害指摘はなく、
そのrace検査5回もPASS、2.039s。
page127/128/129、ignored text/gone、10状態変化を含む。
統合担当からCDP全体race PASS 8.446s、sandbox有効実Browser native race
PASS 10.157sも報告された。候補版harness/native/CI最終受入は
統合担当と台帳が管理し、これだけで監査を完了へ移動しない。

広い汎用checkerは追加せず、既存公開protocol fixtureでresource増加、semantic対象条件、
state由来を個別に検証した。これらは既存のちょうど件数・origin・redaction後testを補い、
既存testが全後続consumerを網羅すると仮定しない。
