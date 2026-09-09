---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/design-docs/browser-cdp-automation.md
source_sha256: 34712fd936ff46bbef295935beda554ae906f59a2dff2e5285428cbe2eb8e5db
---

# Browser/CDP設計

[英語版（翻訳元）](browser-cdp-automation.md)

コマンドと上限は[製品契約](../product-specs/browser-cdp-automation.ja.md)、
実装判断とnative環境での直接証拠は[完了ExecPlan](../exec-plans/completed/browser-cdp-automation.ja.md)に記録します。

## 所有権と依存関係

configはgeneric process設定の上に明示的browser bindingを定義します。
domainは保存可能なbinding・identity・page・snapshot・node・observationを定義します。
`app.BrowserProvider.Observe`はruntime、binding、型付きrequest、native所有権確認callbackを受け取ります。
appは選択、操作fence、run intent保存、snapshot出所の確認、redaction、artifact登録を担当します。
`internal/browser/cdp`はbrowser固有のdiscovery・protocol検証・型付き操作を実装し、
CDP transportは`github.com/gorilla/websocket` v1.5.3で接続します。
具体的runtime adapterをimportせず、browserを起動・終了しません。
CLIがproviderを接続し、引数の解釈と表示を担当します。storeは既存のrun/artifact recordを使い、
browser lifecycle用tableや別のprocess所有者を追加しません。

## 接続の根拠

appはlease fenceを保持して、記録済みpersistent process identityを先に検査します。
adapterは展開・保存済みdebugging/profile switchがそれぞれ1つであることを確認し、
予約済みloopback portを解決します。discoveryには上限を設け、HTTP redirectや環境のproxyを使いません。
browser-level WebSocketの接続先はそのloopback authorityと一致する必要があります。
discoveryと`Browser.getVersion`のproduct/protocolを照合し、`SystemInfo.getProcessInfo`のbrowser PIDが
記録済みnative rootと一致することを確認します。`Browser.getBrowserCommandLine`で専用profile・automation・
debugging設定も確認します。protocol副作用前にnative所有権を再確認し、通常の終了・再利用の変化を検出します。
これは信頼するリポジトリ間の誤操作を防ぐ隔離であり、同一ユーザーの悪意あるCDP serverが
protocol応答を偽装することに対する防御ではありません。

## page・session・古い参照

browser-level接続とtarget sessionを使い、期限・cancel・event振り分けを管理します。
pageは決定的に選び、省略時に曖昧なら拒否します。snapshot identityにはlease、browser、
native PID/birth/port/WebSocket、page、document loader、backend DOM/frame fingerprintを含めます。
semantic入力にはdigestを検証した登録済みsnapshotが必要です。見せかけのファイル、別leaseの参照、
切り詰められたsnapshot、古いdocumentは入力を許可しません。入力直前にframe/nodeの現在状態と一致を確認し、
保存座標へのfallbackはしません。same-origin iframeとshadow DOMは観測し、
cross-origin/OOPIF観測とすべてのiframe semantic入力は今回明示的に非対応です。内部の固定node readbackは使いますが、
任意評価は公開しません。Browser/Android UIの共通抽象化はまだ導入しません。

### Navigation と引数の検証

明示navigationはhash/history変化でも以前のbrowser snapshotを無効にします。document tokenは
raw URLのdigest（query文字列そのものは保存しない）、node fingerprintは許可した非text AX stateを含みます。
set-textは明示的な`--text`が必須で、明示した空文字列はfieldを消去します。CLIは操作の必須引数を検証し、
明示zero durationはstoreを開く前に拒否します。

### Frame の同一 origin を確認する

frameへのアクセスはbrowser報告のsecurity originで判定します。継承した`about:blank`や
`about:srcdoc`でChromeがopaqueの代用値を返す場合、分離した親world内の非公開・固定
predicateで、nativeの`contentDocument` getterへのアクセスがChromiumの同一origin制約で
許されるか確認します。worldにはuniversal accessを与えず、page scriptによる上書きを
信用しません。上限付きtarget一覧で、frame treeから省かれた選択pageの別process iframeを
拒否します。origin不明かつ未commitでURLが空のframeは、waitの既存期限内でのみ再観測し、
その状態で観測や入力を許可しません。

## 副作用・診断・privacy

すべての操作でprotocol処理と証拠確定までlease fenceを保持します。変更操作はrun intentを先に保存し、
入力を1回だけ試みます。完了や証拠を確認できなければ未完了runのbarrierを残します。
fenceを失った操作は後続所有者の状態を確定できません。native process終了とprofile削除は
既存のprocess adapterが担当します。

AXを主要semantic表現、DOM/layoutを上限付き補助証拠とし、editable/password valueは除外します。
PNGは保存前に画像の完全性・サイズ・寸法を検証します。secretを含む入力は一時的に扱い、
構造化データ中の文字列をredactしてJSONの妥当性を保ちます。networkはheader/bodyを保存せず、
console/networkは接続期間を限定します。screenshotと未認識のpage textには秘密が残り得るためprivateに扱います。
永続background診断、download、browser再起動、外部接続、公開CDP passthroughは提供しません。

### reviewで強化した証拠と待機の境界

semantic入力では、検証済みのpage ID、参照元snapshotのrun ID、node参照をCDP呼出し前に
永続CommandRunへ記録します。入力結果が不確定な場合も`run.json`へ同じ根拠を保存し、
set-textの内容はredactします。URL待機には空でない`--contains`が必須で、text/nodeの
条件に使う`--role`は指定できません。省略のあるAX観測では、一致するnodeの消失を証明
できません。DOM名を短縮した場合もtruncation flagを設定します。

console/networkで保持するすべての文字列（IDやmetadataも含む）に、1文字列4 KiB、
文字列内容の合計64 KiBの予算を適用します。長すぎる文字列は機密を含み得る先頭部分を
残さず、全体を置き換えます。上限付きqueueには、実行中captureのsessionと必要なevent
種別だけを入れ、無関係な通知は無視します。購読対象eventがあふれた場合は引き続き拒否します。

## 検証方針

configの負例で明示的bindingと正確なswitch要件を検証します。app testは所有権再検証、
古い参照・別lease参照、run/証拠の失敗、destroy fencingを対象にします。
transport fixtureは別接続先discovery、不正・過大応答を拒否することを検証します。
native testはWindows・macOS・Linux、Go 1.27、Chrome for Testing 152.0.7977.82で実行します。
`391288c`の3 native jobはすべて成功し（Browser native 34247636411）、CDP 1.3を報告しました。
browser/backend fixtureは同じleaseのendpointを使います。Planには実行したnative検証の証拠を、
cross-buildの結果と分けて記録しています。
