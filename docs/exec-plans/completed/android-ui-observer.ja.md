---
status: completed
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/completed/android-ui-observer.md
source_sha256: 2e032e25bcc95d3054680f21c2f7fb46a2c3e156428a1229170db5d26e90f426
---

# Lease が所有する Android UI の観測と操作

[English](android-ui-observer.md)

この ExecPlan は living document であり、`docs/PLANS.md` に従って継続的に更新する。

想定ブランチ: `feat/android-ui-observer`

PR #4 (`feat: run Flutter Android applications on owned emulator leases`) のマージを必須の前提とする。PR #4 が `master` にマージされた後にのみこの ExecPlan を開始し、実装変更の前にマージ後の正確な `master` revision を以下へ記録する。

開始 revision: `f6167becefa81082c914e32769fc015cdb1865fc`

既存の Git source、Compose、Android Emulator、Flutter build/install、shared ADB、reverse mapping、cleanup、fencing、quarantine、application provenance の規則を再実装したり弱めたりしない。本作業はそれらの既存 identity を利用する。

## 目的 / 全体像

この作業の完了後、coding agent は target application にテスト用 hook を追加せず、既存 environment lease 内で表示されている Android UI を直接観測・操作できる。

確実に所有が確認された Android Emulator を含む lease に対して `agent-env` は以下を行える。

- semantics/accessibility を中心とした UI snapshot の取得
- Agent が読みやすい簡潔な tree と一時的 node reference の生成
- raw snapshot の evidence 保存
- PNG screenshot の取得
- semantic node または明示座標の tap
- 編集可能な semantic node の文字列置換
- Back / Home
- 明示的な swipe
- package scope を持つ bounded logcat evidence の収集
- すべての観測・操作を正確な lease/runtime/device identity と関連付けて記録
- 古い画面の座標を暗黙に押すのではなく stale node reference を拒否

典型的な流れ:

    agent-env create . --stack mobile
    agent-env ui snapshot <lease-id> --application mobile-app
    agent-env ui screenshot <lease-id> --application mobile-app
    agent-env ui tap <lease-id> --snapshot <snapshot-id> --node n7
    agent-env ui set-text <lease-id> --snapshot <snapshot-id> --node n3 --text "user@example.com"
    agent-env ui snapshot <lease-id> --application mobile-app
    agent-env ui logcat <lease-id> --application mobile-app --since 30s
    agent-env destroy <lease-id>

CLI syntax は product contract 確定前に調整してよいが、本計画に記した capability と safety property は維持する。

observer は Flutter 専用ではない。Flutter の標準 widget は Android の Semantics/accessibility tree に情報を公開するため恩恵を受けるが、native Android UI や permission dialog 等の system UI も同じ Android accessibility surface から観測できる。

これは browser agent に DOM/accessibility snapshot、screenshot、navigation、runtime diagnostics を与える方式の Android 版であり、新しい environment runtime ではない。

## 進捗

- [x] 2026-09-08: PR #4 の 00:16:34 UTC のマージを確認。`git pull --ff-only` で上記開始 revision と一致することを確認し、`feat/android-ui-observer` を作成した。
- [x] 2026-09-08: repository harness、移植性・セキュリティ・信頼性・品質方針、完了済み Android/Flutter の検証記録を確認した。
- [x] 2026-09-08: app の `AcquireContext` / `SaveRun` / `recordRunArtifact`、Android の `applicationADB`、CLI の `serviceForStore` を拡張点として確認。adapter 境界と cleanup barrier を維持する。
- [x] 2026-09-08: 所有確認済み API 35 Emulator で shell の tree dump、独立 UiAutomation APK のツリー、Flutter の意味的クリック、フォーカス後の Unicode 入力・置換を検証。標準 API の companion を採用した。詳細は下記。
- [x] 2026-09-08: public CLI の実装前に observer の英日 product contract を作成。初回翻訳後の `repoctl docs-check` が成功した。
- [x] 2026-09-08: observer の英日 design doc と索引を作成。既存の adapter と SQL の境界を維持した。
- [x] 2026-09-08: version 1 の snapshot/node/window 型と一定順序の簡潔な text 表示を product contract、`internal/domain/ui.go`、`internal/cli/ui.go` に定義した。
- [x] 2026-09-08: `AGENTENV-UI-STALE`、`AGENTENV-UI-AMBIGUOUS`、`AGENTENV-UI-UNAVAILABLE` を英日 product contract に定義。backend status の対応付けに対する回帰テストは、最新の Linux 全単体テストで成功した。
- [x] 2026-09-08: owned runtime/application の選択と identity proof を実装する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: semantic snapshot capture と normalization を実装する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: snapshot evidence 保存と compact rendering を実装する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: screenshot capture と digest evidence を実装する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: stale-safe semantic-node tap を実装する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: 明示 coordinate tap を実装する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: Unicode 対応 editable-node text replacement を実装する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: Back / Home / swipe を実装する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: bounded wait/poll を実装する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: bounded package-scoped logcat を実装する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: operation lock / heartbeat と統合する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: CLI JSON/text contract と negative fixture を追加する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: multi-runtime / multi-application selection test を追加する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: stale snapshot / ambiguous node regression を追加する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: evidence redaction / size-bound test を追加する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: 実六回目で別 lease の参照拒否、一方の destroy 後の兄弟 Count0/UI と backend HTTP、両 lease の通常 cleanup を確認した。
- [x] 2026-09-08: repository harness と Go race validation を完走する。 `867a862` の `repoctl check` で Linux 単体検証が成功。最終の実機・native の結果は後述。
- [x] 2026-09-08: `867a862` の OS/Go 全 6 native CI job と、別途 5 cross-build が成功。実 SDK の証拠と区別して記録した。
- [x] 2026-09-08: 最終の dialog/Home/viewport assertion を含む `TestRealAndroidUIObserver` 六回目が 217.34 秒で成功した。
- [x] 2026-09-08: native CI と Linux SDK の証拠を区別し、U1–U23 の証拠と振り返りを完成させた。
- [x] 2026-09-08: 英日双方を `docs/exec-plans/completed/` に移動し、参照リンクを更新した。

実装 checkpoint（2026-09-08）: 文書 milestone `a717d59` と実装 `867a862` を
`feat/android-ui-observer` に commit・push 済み。app/domain/Android adapter/companion/CLI、
復旧、専用の負例 fixture は Linux の単体テストで確認できる範囲まで完了した。
`go run ./tools/repoctl check` は整形、全単体テスト、vet、文書、生成物、architecture の検査に成功した。
Go 1.27.1 で `go run ./tools/repoctl doctor` と `go test -race ./...` も成功した。
`go run ./tools/repoctl test-integration` は既存の実 Docker/Compose 統合試験に成功したが、SDK の証拠ではない。
`go test -tags flutterintegration ./internal/cli -run '^$'` による observer fixture の compile は既に成功している。
六回目の実二台 Emulator `TestRealAndroidUIObserver` は、Linux amd64、Go 1.27.1、Flutter 3.47.2、
API 35、三回目の companion build で 217.34 秒で成功した。最終 fixture は dialog の Back 前後の表示、
Home 後に可視アプリ node がないこと、現在の表示 viewport 内の scroll を検証する。
両 lease が通常 cleanup に成功し、一方の destroy 後も兄弟 UI と backend HTTP が維持された。
`867a862` の CI run `34175369767` は全 12 job（native OS/Go 6、cross-build 5、integration）が成功した。
全受け入れ要件の直接の証拠を後述する。archive 後の full `repoctl check` は、整形、単体、vet、文書、生成物、architecture の
全検査に終了コード 0 で成功した。

追加の local 証拠（2026-09-08）: 実装担当が一つの所有 Emulator で snapshot、screenshot、
明示的な復旧、通常 destroy を実行し、成功した。keyboard 表示後に古い意味情報 snapshot を再使用すると、
stale として安全に拒否された。これは二台のデバイスを使う observer 全体の受け入れ試験の代わりにはならない。
最新の harness 実行では単体テストと vet が完了した。英日を同時編集中の docs-check 失敗は、
作業途中の鮮度検査であり最終検証結果ではない。同期後の単独の
`go run ./tools/repoctl docs-check` は Linux / Go 1.27.1 で成功した。

checkbox は意図ではなく観測済みの完了を表す。チェック時には日付、command/test ID、結果、必要な環境情報を記録する。

## 想定外の発見

- 2026-09-08、五回目の実試験: 177.24 秒後、swipe の assertion で失敗した。Back は Count1 を正しく復元していた。IME が viewport を縮めていたため、表示中 ScrollView の bounds は `[0,80,320,346]` だったが、固定の screenshot 基準の swipe 開始点 y=480 はその外にあった。fixture は、現在表示されている一意な scrollable node の bounds から八分の一だけ内側に入った明示座標を求めるよう修正した。scroll 後に、それまで見えていなかった label が可視になる assertion は維持している。さらに dialog が Back 前に表示され、Back 後には存在しないことと、Home 後に可視のアプリ node がないことを、それぞれ独立に検証するよう強化した。これらをすべて含む六回目はその後成功し、最終証拠を前述した。

- 2026-09-08、四回目の実試験: 144.49 秒後、swipe の assertion で失敗した。Back がアプリを終了させたため、swipe 前後の accessibility tree が両方とも空だった。fixture が Back で IME を閉じられると仮定したのが誤りだった。試行後、両 lease の released を確認した。fixture は dialog を明示的に開いて待ち、Back を送って Count1 を待ってから swipe するよう修正した。swipe の検証を緩めず、Back の対象を確定するための変更である。五回目の結果は別項に記録し、この checkpoint では最終の実受け入れは未完了だったが、その後成功した。最終証拠を参照。
- 2026-09-08、証拠の訂正: 一回目と二回目の試験時間はそれぞれ 104.97 秒と 127.31 秒である。以前の 222.41 秒 / 276.86 秒は不正確な経過時間の要約だったため、この計画全体で訂正した。失敗原因と保持する証拠は変わらない。

- 2026-09-08、三回目の実試験: Flutter の依存解決中に fixture 作成が 342.98 秒で終了し、lease の割り当てには到達しなかった。pub.dev への直接 HTTPS probe も 10 秒で timeout した。失敗した試行を終えるため、所有する fixture の pub 子 process だけを明示的に終了した。lease は一つも割り当てられておらず、observer の受け入れは実行していない。
- 2026-09-08: fixture 専用の明示設定 `AGENT_ENV_FLUTTER_OFFLINE_FIXTURE=1` を追加した。Flutter 標準の `flutter create --offline` を付加し、既存 cache を利用する。不足があれば失敗する。core の通常 build/runtime とすべての受け入れ検査は変更していない。四回目はこの設定と三回目の helper build を使い、その結果は別項に記録した。
- 2026-09-08、native 証拠: head `867a862` の Windows/macOS/Linux × 対応 Go の全 6 native CI job と全 5 cross-build job が成功した。native fake/backend と build の証拠であり、実 Android SDK の証拠ではない。CI integration job も成功した。`867a862` の CI run `34175369767` は全 12 job が成功した。

- 2026-09-08、二回目の実二台 Emulator 試験: `TestRealAndroidUIObserver` は 127.31 秒実行され、両アプリの起動に成功した後、新しい snapshot を使った意味情報 tap を stale として拒否した。保持した前後の証拠は、Android window ID が 8 から 11 に変わったことと、それに由来する fingerprint 以外は一致していた。window ID は UiAutomation の再接続をまたぐと変わるため、継続的な意味情報の識別要素にはできない。helper を window の意味情報に基づく metadata を使うよう修正し、三回目の実装版 build を生成した。この checkpoint では実際の回帰再試行は未完了だったが、その後成功した。最終証拠を参照。
- 2026-09-08、追加検査: 全単体テストと vet が再び成功した。CLI の検証には log 内容の直接出力、明示的な duration 0 の拒否、存在しない lease の終了コード 2、壊れた registry の終了コード 7 を含めた。今回の翻訳同期後、単独の `go run ./tools/repoctl docs-check` が成功した。この checkpoint では Go race と native/実試験は未完了だったが、その後成功した。最終証拠を参照。

- 2026-09-08、実統合試験: `TestRealAndroidUIObserver` は Android activity の起動中、`am start -W` の `Status: timeout` により 104.97 秒で失敗し、observer の assertion には到達しなかった。registry の確認と通常の保守的 cleanup により、両 lease の released を確認した。readiness、所有権、timeout の検査は緩めていない。独立した再試行は別項に記録し、この失敗を observer の受け入れ証拠とはしない。
- 2026-09-08、app/復旧の独立レビュー: 元の結果を登録した後に最終分類の `SaveRun` が失敗すると、分類のない run が残る場合があった。従来の復旧は明示的な禁止分類だけを拒否していたため、分類の欠落によって host-process/evidence barrier を回避できた。現在は明示的に永続化された `termination-unconfirmed` の分類と、検証済みの元の結果証拠を要求する。`TestUIRecoverRefusesUnpersistedFailureClassification` で分類の書き込み失敗を検証する。復旧・fence・backend 診断の focused test は Linux / Go 1.27.1 で成功し、この checkpoint では最終受け入れは未完了だったが、その後完了した。最終証拠を参照。
- 2026-09-08、CLI の独立レビュー: 以前の UI error は分類されず、tool の不足や registry 障害も既定の終了コード 2 になっていた。型による error 分類の伝播と CLI の対応付けにより、前提条件不足は 3、無効な option・対象選択・存在しない lease・stale/ambiguous は 2、registry/観測障害は 7 と区別する。診断 error に秘密情報を露出させない。

- 2026-09-08: 所有確認済み API 35 Emulator と一時 Flutter アプリで、`uiautomator dump` と自己対象の UiAutomation APK が Flutter の accessibility を取得できた。意味的クリックで Count 0 から Count 1 に変化。フォーカスした欄への `日本語 🙂 café` の入力後、新しいノード参照で `置換済み 🚀` への置換を行い、取得した accessibility text で確認した。対象アプリに instrumentation 依存は追加していない。
- 2026-09-08、失敗した方法: 未フォーカスの Flutter ノードは ACTION_SET_TEXT に true を返しても値を変更しなかった。フォーカスによるキーボード表示で window とノード番号が変化し、古い番号は別の対象で成功を返す場合もあった。実装では対応 action・focus の確認、意味情報の fingerprint 照合、入力後の値の一致確認を必須にする。dispatch の成功だけでは検証済みとしない。

- 2026-09-08: 初回の `go run ./tools/repoctl check` は単体テストと vet に成功した後、提供された日本語計画の `translation_of` と `source_sha256` 欠落で失敗した（AGENTENV-DOC-008）。検査を変更せず必須 metadata を補い、`repoctl docs-check` が成功した。

- 2026-09-08、独立レビュー: adapter と app の logcat データ受け渡し（`Raw` と `Binary`）の不一致、backend status の診断コードへの対応付け漏れ、操作後 fingerprint の欠落、不完全な階層取得の扱いを発見した。修正と回帰テストは本実装に含め、この checkpoint では最終レビューと検証は未完了だったが、その後完了した。最終証拠を参照。app の fake だけでは具体的な adapter と証拠保存の境界を証明できないことが明らかになった。
- 2026-09-08、復旧設計の発見: ローカル ADB の timeout だけでは remote instrumentation の停止を証明できない。実行中 barrier だけを残して対応する復旧手段がないと、通常 destroy を進められなくなる。所有権を検証した helper だけの停止・静止確認を有効な fence の下で行い、登録済み helper run の明示的な復旧を追加する。入力の再実行や任意コマンドの復旧は行わない。

少なくとも以下は発見時に記録する。

- Android API level による UI Automator/accessibility の差
- Flutter semantics が merge / 欠落 / 想定外の表現になるケース
- system dialog/window の cross-package 観測要件
- semantic fingerprint を一意にできない duplicate node
- Unicode text entry の制約
- helper/instrumentation artifact の packaging 制約
- host OS による screenshot 取得差
- package attribution が曖昧になる logcat behavior
- command 成功に対して evidence persistence が不確実なケース
- fake backend test と実 Emulator の不一致
- 既存 Android identity/cleanup rule を弱めたくなる圧力

設計に影響した失敗した試行は削除せず保持する。

## 判断の記録

- 2026-09-08、実装担当: 編集可能な node/password node のフィールドは引き続き伏せ、set-text 入力値はそのコマンド自身の証拠から伏せる。入力値を将来の秘密値照合データとして永続化しない。後の編集可能でない表示へのアプリの出力や、明示的に要求したアプリの logcat には値が含まれ得るが、設定済み秘密値の redaction は継続する。理由: コマンド自身の秘匿を維持しつつ、意図的に保持しない入力を後から認識できると約束しないため。

- 2026-09-08、実装担当: window の識別には type、title、bounds、root package/class、active 状態を使い、一時的な Android window ID、走査順番号、layer を除外する。生の ID は観測証拠として保持し、曖昧な一致は引き続き拒否する。理由: UiAutomation の再接続後も変化していない Flutter UI を操作できるようにしつつ、意味情報が重複する window を一意とは扱わないため。
- 2026-09-08、実装担当: 意味情報に基づく action は登録済み snapshot の `ExpectedBackend` を引き継ぐ。backend の provenance が一致しない場合は意味情報 action の dispatch 前に拒否し、別の helper build で得た snapshot で現在の backend を操作できないようにする。理由: fingerprint の規則は特定の検証済み helper 実装に属し、既存の参照が使う規則を暗黙に変更してはいけないため。

- 2026-09-08、実装担当: 復旧の許可は、禁止分類がないことではなく、永続化された明示的な適格性に基づく。helper 復旧の dispatch 前に `termination-unconfirmed` の分類と、登録済みの元の結果証拠を要求する。理由: 最終分類の保存失敗や crash によって、host process や証拠の不確実性が barrier 解除の許可へ変わることを防ぐため。分類のない中断は、明示的に調査するまで引き続き拒否する。

- 2026-09-08、実装担当: helper の内部期限切れでは、元の fence が有効な間、検証済み companion だけの停止・静止確認に追加で最大 10 秒を使える。呼び出し元のキャンセルや lock 喪失では自動復旧しない。`ui recover LEASE --run RUN` は新しい fence を取得し、元の登録済み runtime/serial と helper の同一性を検証して process の不在を証明する。失敗または結果不確実という復旧証拠を保持してから、その run の barrier だけを解除する。理由: 所有権を弱めず、不確実性を隠さず、入力を再実行せず、native/test process に触れずに、不完全な remote 操作を復旧可能にするため。
- 2026-09-08、実装担当: 安定した platform UiAutomation（API 26 以降）を使う agent-env 所有の自己対象 APK を採用し、AndroidX には依存しない。shell の text input では Unicode 置換を満たせない。native Go の明示 build で local 配布用 APK と version/source/APK digest を生成し、通常の Go build は SDK/JDK から独立させる。template に競合する helper があれば拒否し、上書きしない。
- 2026-09-08、実装担当: application の既定 scope はその package とし、runtime-only/all-windows は system UI を含む。quarantined・期限切れ・cleanup 中の lease を拒否し、ready/degraded の所有 Android では読み取り診断を許可する。変更操作には ready を要求する。fingerprint は意味情報の祖先・window・bounds・state を含み、走査順番号と編集可能な値を除く。切り詰めた tree は action を許可しない。入力値・編集可能な値・password 値は完全に伏せ、読み戻しは一致したかだけを報告する。上限、API 下限、logcat の PID 帰属は product/design contract に記録する。manifest や adapter 間の依存境界の変更は不要。

- 判断: `android-ui-observer` は既存 owned Android runtime 上の observation/interaction capability とし、新 runtime type にしない。
  理由: device lifecycle と application lifecycle は既に独立した所有権を持っており、observer が第三の owner になるべきではない。
  日付/担当: 2026-09-08 / maintainers.

- 判断: semantic black-box surface は Android accessibility/UI Automator から見える状態とし、screenshot は補完情報とする。
  理由: target-app hook 無しで Flutter semantics、native Android UI、system dialog を同じ方式で扱え、browser agent の semantic tree + screenshot と同型にできる。
  日付/担当: 2026-09-08 / maintainers.

- 判断: snapshot node reference は一時的で snapshot scope に限定する。
  理由: accessibility node と bounds は UI の変化で変わり、durable selector として扱うと Agent が別 control を誤操作し得る。
  日付/担当: 2026-09-08 / maintainers.

- 判断: semantic-node action は stale/ambiguous snapshot を拒否し、保存済み座標へ暗黙 fallback しない。
  理由: 観測が古い場合に Agent がそれを認識できる必要がある。
  日付/担当: 2026-09-08 / maintainers.

- 判断: `agent-env ui` のために target app へ instrumentation dependency を追加させない。
  理由: 任意 repository と system UI に再利用できる black-box infrastructure とするため。
  日付/担当: 2026-09-08 / maintainers.

- 判断: UI evidence は lease artifact system に保存し、textual data は既存 redaction/digest boundary を利用する。
  理由: unmanaged な証拠置き場を作らず、build/test evidence と同じ audit/recovery model に載せるため。
  日付/担当: 2026-09-08 / maintainers.

- 判断: durable docs と living ExecPlan は英語・日本語双方で維持する。
  理由: bilingual documentation policy を継続するため。
  日付/担当: 2026-09-08 / maintainers.

最終検証 checkpoint（2026-09-08）: tagged CLI `TestUI` の対象検証と `repoctl doctor` が成功した。
最終の `go test -race ./...` も成功した（app 32.784 秒）。最終 fixture 変更は `2001eac` として commit・push 済み。
この head の CI `34176592070` は 10 job が成功し、integration と macOS Go 1.27 が未完了である。
production code は変更しておらず、`867a862` で native/integration CI 全体の証拠がある。
archive 後の full `repoctl check` は、整形/単体/vet/docs/generated/architecture の全検査に終了コード 0 で成功した。

## 成果と振り返り

2026-09-08 完了。version と上限を持つ accessibility snapshot、一時的な node 参照、検証済み PNG、
古い参照を拒否する意味情報 tap、focus と読み戻し確認を伴う Unicode 置換、明示的な座標 tap、
Back/Home/swipe、上限付き wait、帰属と上限を示す logcat を実装した。独立して build する自己対象の
platform UiAutomation companion により、対象アプリへの instrumentation と AndroidX 依存を不要にした。
検証済み version/source/APK provenance で snapshot と backend を結び付ける。app の制御、Android 資源管理、
Flutter build は分離したままとし、既存 registry の run/artifact で操作意図、保守的 cleanup barrier、
証拠に基づく限定的な復旧を実現した。schema の変更はない。

実 Flutter 試験では、platform action の成功だけでは focus/読み戻しなしの text 置換を保証できないことと、
Android window ID が instrumentation 再接続で変化することを確認した。window の意味情報に基づく識別で
有効な参照を維持し、一意な照合で古い・曖昧な入力を拒否する。IME で縮む viewport や Back の遷移は、
画面サイズや keyboard の存在を仮定せず、観測可能な fixture 状態で検証する必要があった。
最終 fixture はその assertion を強化している。五回の失敗は以下に保持し、テストや所有権検査は緩めていない。

実六回目は Linux amd64 / Go 1.27.1 / Flutter 3.47.2 / API 35 で 217.34 秒で成功し、二つの同時 lease、
要求された observer 操作、Unicode/秘匿/PNG、兄弟分離、通常 cleanup を確認した。
Linux full harness、race、実 Docker 検査も成功し、`867a862` の CI `34175369767` は全 12 job が成功した。
Windows/macOS の native fake/backend と cross-build は、それらの OS で実 SDK を検証した証拠ではない。
対応下限は API 26 だが、実 SDK の証拠は API 35 であり、他の API image は未検証である。
より豊富な gesture、OCR、visual regression、物理 device、remote host、browser/CDP との対称性は今後の課題とする。

編集可能な field/password field と set-text 自身の証拠は伏せるが、将来の任意のアプリ表示/logcat と PNG の
ピクセルには機密情報が残り得る。分類のない crash、host process や証拠の不確実性は意図的に cleanup barrier とする。
archive 後の最終 harness 再実行は全検査に終了コード 0 で成功し、未完了の機能受け入れはない。

## 背景と構成

実装前に以下を読む。

- `AGENTS.md`
- `ARCHITECTURE.md` / `ARCHITECTURE.ja.md`
- `docs/PLANS.md` / `docs/PLANS.ja.md`
- `docs/PORTABILITY.md` / `.ja.md`
- `docs/RELIABILITY.md` / `.ja.md`
- `docs/SECURITY.md` / `.ja.md`
- `docs/QUALITY.md` / `.ja.md`
- `docs/roadmap.md` / `.ja.md`
- Android Emulator の英日 product/design docs と completed ExecPlan
- PR #4 の最終 Flutter Android 英日 product/design docs と completed ExecPlan

PR #4 は Flutter application を `android-emulator` runtime とは別概念とし、installed APK digest/package/activity、exact owned serial、reverse mapping を lease identity に含め、UI automation を後続作業として残している。

PR #4 マージ後の想定境界:

- config: manifest strict decode
- stack: deterministic component closure
- app: orchestration / fencing / readiness / compensation / evidence policy
- Android adapter: concrete Emulator/ADB effect
- Flutter application lifecycle: build/install/launch/reverse identity
- SQLite: lease snapshot / events / runs / artifacts
- `execx`: portable process boundary
- `evidence`: redaction / digest
- CLI: parse/renderのみ

observer のために新しい manifest section は原則追加しない。observer は既存 lease resource を操作する。

semantic snapshot は Android accessibility tree に基づくものであり、Flutter widget tree ではない。custom Flutter control が適切な Semantics を公開していなければ、視覚的には存在しても semantic snapshot には現れない。その場合は制約として正直に示し、screenshot から架空の semantic node を生成しない。

snapshot 例:

    Snapshot: ui-01K...
    Window: com.example.app

    [n1] text    "Sign in"
    [n2] textbox "Email"       bounds=[72,240][1008,344]
    [n3] textbox "Password"    bounds=[72,376][1008,480] password=true
    [n4] button  "Login"       bounds=[72,528][1008,640] enabled=true

`n4` はその snapshot 内だけで有効で、manifest へ durable selector として保存しない。

## 作業計画

### Milestone 1 — Backend spike と product contract

PR #4 で作れる real lease-owned Emulator を使い、以下を満たす最小 black-box backend を比較する。

- visible accessibility window/node の列挙
- text/contentDescription/bounds/state の保持
- semantic element click
- non-ASCII を含む editable text replacement
- screenshot
- structured command failure
- target app と Android system dialog の cross-package 操作

target app の test dependency を前提にしない。

候補:

1. AndroidX UI Automator/accessibility API を利用する versioned `agent-env` companion instrumentation APK
2. 同等の Unicode / structured error / stale safety / evidence を満たせる Android platform built-in primitive

public `agent-env ui` contract から backend implementation は隠す。

companion helper を選ぶ場合:

- helper は agent-env の artifact であり target repo dependency ではない
- version と SHA-256 を UI evidence に記録
- confirmed owned serial にだけ install
- Emulator の owner にはならない
- Windows/macOS/Linux で runtime 利用可能な packaging を定義
- UI 無関係の通常 Go command に Android build toolchain を要求しない
- pre-release AndroidX dependency を暗黙採用しない

以下を英日双方で作る。

    docs/product-specs/android-ui-observer.md
    docs/product-specs/android-ui-observer.ja.md
    docs/design-docs/android-ui-observer.md
    docs/design-docs/android-ui-observer.ja.md

### Milestone 2 — Snapshot schema と capture

versioned normalized snapshot schema を定義する。

最低限検討する field:

- snapshot/lease/runtime/serial identity
- capture timestamp
- display size
- window/package
- class
- resource ID
- text
- content description
- bounds
- clickable / enabled / focused / editable / password / scrollable / selected / checked 等

要件:

- node ref は snapshot 内のみ一意
- raw backend output を有用な場合に別 artifact として保持
- compact text output は deterministic
- JSON output を structured client 向けに提供
- node/byte 上限と明確な truncation marker
- semantic snapshot は頻繁に取得できる程度に軽量
- screenshot を毎 snapshot に暗黙取得しない

### Milestone 3 — Screenshot evidence

exact owned serial から PNG を取得し、lease/runtime/serial、timestamp、dimensions、SHA-256を記録する。不完全な image output は成功 artifact として publish しない。

### Milestone 4 — Stale-safe action

semantic-node actionは:

1. lease operation fence取得
2. Android ownership再確認
3. referenced snapshot読込
4. lease/runtime一致確認
5. current UI再観測
6. old node fingerprintを一意に再解決
7. stale/ambiguousなら入力せずstable error
8. freshならaction
9. action evidenceとcurrent-state fingerprint記録

初期 action:

- tap
- set-text

`set-text` は Unicode を扱う。backend が忠実に文字列を表現できない場合、`adb shell input text` と同等だと document しない。

coordinate tap は semantic action と明確に区別して記録する。

### Milestone 5 — Navigation / swipe / wait

Back、Home、one-pointer swipe、semantic predicateまたはstable-stateへのbounded waitを追加する。wait は必ずdeadlineを持つ。arbitrary shell command は `agent-env ui` に追加しない。

### Milestone 6 — Logcat observation

selected application/package に可能な限り scope した bounded diagnostics を追加する。

要件:

- exact owned serial
- since/duration/line/byte bound
- package/PID attribution
- attribution不能ならbroader device scopeを明示
- text evidence secret redaction
- unbounded background stream禁止
- shared ADB lifecycle変更禁止

logcat 内容だけで lease readiness を変更しない。

### Milestone 7 — State policy / concurrency

read-only snapshot/screenshot/logcatは、device ownershipが確実なら DEGRADED lease でもdiagnosisのため利用可能にする方向で検討する。

QUARANTINED leaseの厳密な規則は実装前に決める。ambiguous device identityをforce/fallbackで越えない。

mutating actionにはactive/unexpired lease、confirmed owned Android identity、operation fence、cleanup非進行を要求する。

証明事項:

- 同じ package/device-local port を使う2 leaseが独立
- lease A snapshotをlease B actionへ渡すと拒否
- destroyとactionがfenceを越えて競合しない
- AからB serialを操作しない
- user/external Emulatorをimplicit selectionしない

### Milestone 8 — Agent-facing ergonomics

AgentがADB serial、AVD path、helper package、instrumentation runner、artifact directoryを知らなくても使えるCLIを作る。

例:

    $ agent-env ui snapshot lease-a --application mobile-app

    Snapshot ui-01K...
    Runtime phone (emulator-5554)
    Window com.example.app

    [n1] text    "Sign in"
    [n2] textbox "Email"       enabled
    [n3] textbox "Password"    enabled password
    [n4] button  "Login"       enabled

eligible target が1つなら deterministic default を許可してよい。複数なら guess せず application/runtime 指定を要求する。

### Milestone 9 — Real integration evidence

最小 Flutter fixture に labeled text、ASCII/Unicode editable field、button、tap後のstate change、scrollable elementを含める。

real Emulatorで以下を証明する。

1. create READY
2. expected semantic nodeをsnapshotで取得
3. valid PNG screenshot
4. ASCII / non-ASCII set-text
5. UI change後にold snapshot actionを拒否
6. fresh snapshot + tapでstate変化
7. Back/Home動作
8. bounded logcat取得
9. sibling lease非影響
10. normal destroy成功

Windows/macOS/Linuxのfake/native portability testとreal Android validationは区別する。

## 具体的な手順

1. PR #4 のmergeを待つ。
2. `git switch master && git pull --ff-only`。
3. 開始revisionを本planへ記録。
4. `feat/android-ui-observer`作成。
5. 英日planを `docs/exec-plans/active/` に追加。
6. 実装前repository harness baseline記録。
7. merged Flutter/application domainとAndroid adapter調査。
8. real lease-owned Emulatorでbackend spike実行。
9. public contract作成前にbackend decision記録。
10. bilingual product/design docs とindex更新。
11. concrete adapter wiring前にapp/domain observer interface追加。
12. snapshot capture/normalize/evidence実装。
13. screenshot実装。
14. stale-safe tap / Unicode set-text実装。
15. coordinate tap / navigation / swipe / wait実装。
16. bounded logcat実装。
17. CLI text/JSON wiring。
18. fake backend / negative / fencing / cross-lease test追加。
19. real Flutter/Emulator observer integration追加。
20. focused tests / `repoctl check` / race / native CI反復。
21. platform/evidence typeを区別してacceptance evidence記録。
22. 英日双方のOutcomes & Retrospective完成。
23. 両planをcompletedへ移動しlink更新。

## 検証と受け入れ

| ID | 必須動作 | 証拠 |
| --- | --- | --- |
| U1 | UI機能を使わない既存Compose/Android/Flutter manifestとleaseが無変更で動く。 | 2026-09-08: `867a862` の `repoctl check` で既存単体回帰が成功。`repoctl test-integration` で実 Docker/Compose が成功。二回目の実 Flutter 試験で両 lease がアプリ起動に到達した。observer 全体は実六回目で成功した。 |
| U2 | UI commandはselected confirmed owned Android runtimeだけを解決しfirst-device fallbackを持たない。 | 2026-09-08 Linux 成功: full `repoctl check` 内の `TestUIRejectsInvalidStateAndSelectionBeforeDevice`、`TestUIDegradedDiagnosticsAndApplicationScope`、`TestUIOwnershipFailureCannotDispatchInput`。 |
| U3 | semantic snapshotがlease/runtime/snapshot identity付きdeterministic JSON/textを返す。 | 2026-09-08 Linux 成功: `TestUISnapshotAndSemanticScope`、protocol test、単一実 Emulator の snapshot smoke。簡潔な表示を実装済み。最終実 observer workflow は実六回目で成功した。 |
| U4 | raw/normalized snapshot evidenceがboundedかつatomicにpublishされる。 | 2026-09-08 Linux 成功: `repoctl check` の snapshot 改ざん/symlink、protocol 上限、証拠失敗、atomic-write 回帰。単一実 Emulator の snapshot artifact も保持。 |
| U5 | real integrationでFlutter semantic label/text/editable controlがdocumented Android accessibility surfaceに現れる。 | 2026-09-08: 所有 API 35 spike で Flutter label、text、編集可能な control を取得。二回目の実試験でも Count0/Increment0 を取得した後、一時的 window ID の照合で失敗した。 |
| U6 | screenshotがvalid PNG artifact、digest、exact device identityを持つ。 | 2026-09-08 Linux 成功: `TestUIPNGValidation`、`TestUIInvalidCompletedCaptureFinalizesFailed`。単一の所有実 Emulator の PNG/artifact smoke も成功。 |
| U7 | fresh snapshot nodeのsemantic tapが意図したnodeを操作する。 | 2026-09-08 実六回目成功（217.34 秒）: 新しい意味情報 tap で Count0 から Count1 に変化し、古い参照の再使用で追加 increment は起きなかった。 |
| U8 | stale/ambiguous nodeはinput前に失敗しold coordinateへfallbackしない。 | 2026-09-08 Linux 成功: `TestUIRejectsSnapshotTamperingAndAmbiguity`、`TestUIBackendRefusalsRetainStableDiagnostics`、backend provenance 回帰。実 smoke でも古い snapshot の入力を stale として拒否。 |
| U9 | explicit coordinate tapをsemantic tapと別に記録する。 | 2026-09-08 実六回目成功: 明示的な座標 tap と表示 viewport 内の swipe を実施。座標範囲の負例テストも成功。 |
| U10 | editable-node text replacementが定義したUnicode caseをshell escaping破損なしで扱う。 | 2026-09-08 実六回目成功: ASCII と日本語/emoji/Greek の置換が読み戻しと一致し、編集可能な値/password の証拠は伏せられた。 |
| U11 | Back/Home/swipeがselected serialだけに作用する。 | 2026-09-08 実六回目成功: dialog は Back 前に表示され、Back 後は消えて Count1 が復元された。Home 後は可視アプリ node がなく、swipe は以前見えなかった label を表示した。 |
| U12 | wait/pollはboundedでtimeout後にhidden effectを残さない。 | 2026-09-08 実六回目成功: 存在しない条件の wait は上限内に期待どおり失敗し、両 lease の通常 cleanup が成功。単体の timeout/復旧/fence 回帰も成功。 |
| U13 | bounded logcatがpackage/PID scopeまたはbroader scopeを正直に記録しsecret redactionする。 | 2026-09-08 Linux 成功: `TestUILogPIDAttributionAndBounds`、`TestUILogcatReturnsBoundedRedactedInlineEvidence`、非有限 timestamp、CLI log 表示テスト。数値 PID の履歴の限界を文書化。 |
| U14 | UI operationがlease fenceに参加しdestroyがin-flight mutating actionを追い越さない。 | 2026-09-08 Linux 成功: `TestUIOperationFencePreventsDestroyRace` と復旧/host 完了未確認 barrier テスト。`go test -race ./...` も成功。 |
| U15 | 一方のleaseのsnapshot/node refを他方leaseに使えない。 | 2026-09-08 Linux 成功: `TestUISnapshotAndSemanticScope` の別 lease 拒否。二回目の実試験でも新しい tap が失敗する前に別 lease の参照を拒否。 |
| U16 | concurrent mobile leaseを独立観測・操作でき、一方destroy後も他方が利用可能。 | 2026-09-08 実六回目成功: 二つの独立 lease の一方を destroy 後、兄弟 snapshot の Count0 と backend HTTP が維持され、両方が通常 released になった。 |
| U17 | ambiguous Android identityをforce/fallbackで観測・操作しない。 | 2026-09-08 Linux 成功: `TestUIOwnershipFailureCannotDispatchInput`、状態/選択テスト、既存 Android 所有権 suite。force や serial 上書きは公開しない。 |
| U18 | target Flutter/app repoへUI-test dependencyやsource変更を要求しない。 | 2026-09-08: 独立 companion build と API 35 Flutter spike で対象アプリへの instrumentation 依存は不要だった。実 fixture は標準 Flutter widget を使い、observer manifest section は存在しない。 |
| U19 | helper使用時、そのartifact/version/digest/lifecycleを記録しtarget instrumentationを要求しない。 | 2026-09-08 Linux 成功: helper の provenance/source/version/digest と backend 変更テスト。三回の明示 companion build、単一実 Emulator の install/観測/復旧 smoke を記録。 |
| U20 | observer docs/ExecPlanが英日双方に存在しtranslation checkを通る。 | 2026-09-08 成功: product/design/index/README/CLI/architecture/plan を英日更新。full `repoctl check` の docs-check が成功し、今回の living plan 更新も別途再検査。 |
| U21 | architecture/docs/schema/full repository harnessがfinal implementationでpass。 | 2026-09-08、`867a862` で成功: `repoctl doctor`、整形/単体/vet/docs/generated/architecture を含む full `repoctl check`。実 Docker 統合も成功。 |
| U22 | Go raceとWindows/macOS/Linux native fake/backend testがpass。 | 2026-09-08 成功: Linux の `go test -race ./...`、`867a862` の Windows/macOS/Linux × 対応 Go の全 6 native CI job。5 cross-build も別途成功し、SDK 証拠とはしない。 |
| U23 | real Flutter+Emulatorでsnapshot/screenshot/Unicode/stale rejection/action/logcat/cleanupを実証するか、具体的外部blockerをfake evidenceで代替せず記録する。 | 2026-09-08 実六回目成功、217.34 秒、Linux amd64 / Go 1.27.1 / Flutter 3.47.2 / API 35 / companion build 3。要求された実 CLI observer 検査と両 lease の通常 cleanup が成功。一〜五回目の失敗は保持。 |

acceptanceは直接の成功証拠を必要とする。test名だけでは実行成功の証拠にならない。

## 冪等性と復旧

snapshot/screenshot/logcatはdesired lease stateを変更しない。operation fence下でheartbeat更新とevidence/event追加は許容する。

failed observationがincomplete temporary artifactを残してもterminal success artifact rowをpublishしない。lease artifact directory内で明確に所有できるincomplete fileだけを回収してよい。

mutating UI actionは一般にidempotentではない。

- 可能ならinput前にaction intentとtarget identityを保存
- completionは別記録
- tap結果が不確実なら自動retryしない
- uncertaintyを返し、callerはfresh snapshotでretry要否を判断
- stale failure後にold node refを再利用しない

helper installが不確実ならunproven deviceのpackageをuninstallしない。通常はconfirmed private AVD destroyによりhelper/app stateも消える。

禁止:

- `adb kill-server`
- device list orderによる選択
- user AVD template変更/削除
- 他lease package/reverse mapping変更
- cleanup shortcutとしてglobal logcat clear
- Android quarantine barrierの弱体化
- `--force`によるdevice ownership proof bypass

## 成果物と注記

最終の実証拠（2026-09-08）: `go test -tags flutterintegration ./internal/cli -run
'^TestRealAndroidUIObserver$' -count=1 -v -timeout=35m` が 217.34 秒で成功した。
Lease ID: `01M1Z9KEH18ZTC8AHQSQ06MHT9` と `01M1Z9KEJE3EZDVT8Y5RBXT9MS`。
fixture source: `be35f3179692baf0c3c160715d7c6f5987889de9`。Linux amd64、Go 1.27.1、
Flutter 3.47.2、Android API 35、三回目の検証済み companion build、明示的な cache 利用の fixture 作成で実施した。
実 CLI の操作、Unicode 読み戻し、stale/別 lease 拒否、秘匿と PNG/digest、viewport 内の swipe、
Back/Home、現在 PID の log、上限付き wait を検証した。一方の destroy 後も兄弟は Count0 を保持し、
新たな backend HTTP 要求に応答した。両 lease の通常 cleanup が成功した。


操作ごとのartifact例:

    <agent-env-home>/leases/<lease-id>/artifacts/<ui-run-id>/
      run.json
      snapshot.json
      snapshot.txt
      raw-hierarchy.xml
      screenshot.png
      logcat.txt

実際のlayoutは既存artifact conventionへ合わせてよい。

最低記録:

- UI run/observation ID
- lease ID
- component/application
- Android runtime
- serial
- package/window
- backend/helper name/version/digest
- operation
- semantic actionのsnapshot/node ref
- stale validation用fingerprint
- coordinate actionの座標
- timestamps
- truncation
- artifact SHA-256
- action result / uncertainty

passwordやtext-entry payloadをclear textでaction metadataへ永続化しない。fully redacted / length-only / opt-in retentionのいずれかを決めてdocumentする。defaultはsecret retentionを最小化する。

screenshot/raw snapshotそのものにはsecretが映る可能性がある。text redactionでPNG pixel内のsecretは消えないためprivacy propertyを明記する。

## インターフェースと依存

想定app boundary:

```go
type AndroidUIProvider interface {
    Capabilities(ctx context.Context, target AndroidUITarget) (...)
    Snapshot(ctx context.Context, target AndroidUITarget, opts SnapshotOptions) (...)
    Screenshot(ctx context.Context, target AndroidUITarget, opts ScreenshotOptions) (...)
    Act(ctx context.Context, target AndroidUITarget, action UIAction) (...)
    Logcat(ctx context.Context, target AndroidUITarget, opts LogcatOptions) (...)
}
```

名称・型は固定ではない。

責務:

- domain: CLI非依存data
- app: lease/runtime selection、fencing、stale policy、evidence policy
- concrete backend: Android device commandとparse
- store: persistenceのみ
- CLI: parse/renderのみ

依存候補:

- 既存Android SDK `adb`
- Android accessibility/UI Automator surface
- 必要ならagent-env companion instrumentation artifact
- 既存Emulator/Flutter support

target applicationは `agent-env ui` のためにAndroidX UI Automatorやinstrumentation runnerを追加しない。

Bash/POSIX shell/PowerShell/Make/implicit first-device selection/CGO/fixed host portをcore requirementにしない。

Milestone 1 の決定事項（解決済み。検証証拠は受け入れ表に記録）:

1. Backend: 意味情報 snapshot/action は agent-env の自己対象 companion を使う。PNG、navigation、座標、logcat は所有 serial を明示した platform command を使う。
2. API: 安定した platform `UiAutomation` / `AccessibilityNodeInfo` を使い、AndroidX や prerelease には依存しない。API 35 の実 spike 証拠はあるが、他の API で検証済みとはしない。
3. Packaging: companion source を埋め込み、native Go から SDK/JDK を明示的に呼んで新しい directory に build する。APK/metadata は `AGENT_ENV_UI_HELPER` で local 配布し、version/source/APK digest を検証する。通常の Go build は SDK/JDK から独立させる。
4. Fingerprint: window の意味情報を示す type/title/bounds/root package/class/active 状態と、祖先の意味情報、node の class/package/resource ID/編集可能でない label・text/bounds/action・state flag を使う。一時的な window ID・走査順番号・layer と、編集可能な値・password 値を除外する。一意な一致と backend provenance の一致を要求する。
5. Scope: application snapshot は記録済み package を既定の対象とし、runtime-only と明示的な `--all-windows` はアクセス可能な system window も含める。
6. Privacy: 入力値・編集可能な値・password text と、編集可能な node の description/hint を完全に伏せる。通常の label/log は設定済み秘密値を伏せる。PNG のピクセルには text redaction を適用できず、任意の表示 text には機密情報が残り得る。
7. Logs: `ui logcat` とし、現在の数値 PID とデバイス時刻の下限で取得する。device 全体への暗黙 fallback や global log の消去は行わない。PID 再利用と履歴の限界を報告し、直接出力と artifact の両方に上限を設ける。
8. State: 読み取り診断は active かつ期限内の ready/degraded lease で、稼働する Android の所有権を証明できる場合に許可する。変更には ready を要求する。quarantined・期限切れ・released・cleanup 中の通常 UI 操作は拒否し、対象を限定した明示的 helper 復旧には別途文書化した fence/証拠方針を適用する。
9. Limits: node 数 1000、深さ 64、snapshot の各 field 4096 文字、backend 応答 1 MiB とし、切り詰めを明示する。不完全な tree では意味情報による変更を許可しない。PNG と log の上限は product contract に定義する。
10. Compatibility: API 26 を下限とし、Windows/macOS/Linux の native argv/filesystem をサポートする。platform の証拠には、記録した実 SDK/native 実行だけを使用する。

理由と復旧の詳細は、判断の記録と英日 product/design contract に記録する。
