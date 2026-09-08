---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/active/android-ui-observer.md
source_sha256: d6707f35372f52faee596599698519edf851ff13863849934d5fb46b7eff7ba2
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
- [x] 2026-09-08: `AGENTENV-UI-STALE`、`AGENTENV-UI-AMBIGUOUS`、`AGENTENV-UI-UNAVAILABLE` を英日 product contract に定義。backend status の対応付けには専用の回帰テストを追加しており、最終検証は未完了。
- [ ] owned runtime/application の選択と identity proof を実装する。
- [ ] semantic snapshot capture と normalization を実装する。
- [ ] snapshot evidence 保存と compact rendering を実装する。
- [ ] screenshot capture と digest evidence を実装する。
- [ ] stale-safe semantic-node tap を実装する。
- [ ] 明示 coordinate tap を実装する。
- [ ] Unicode 対応 editable-node text replacement を実装する。
- [ ] Back / Home / swipe を実装する。
- [ ] bounded wait/poll を実装する。
- [ ] bounded package-scoped logcat を実装する。
- [ ] operation lock / heartbeat と統合する。
- [ ] CLI JSON/text contract と negative fixture を追加する。
- [ ] multi-runtime / multi-application selection test を追加する。
- [ ] stale snapshot / ambiguous node regression を追加する。
- [ ] evidence redaction / size-bound test を追加する。
- [ ] sibling lease が相互の device を観測・操作できないことを証明する。
- [ ] repository harness と Go race validation を完走する。
- [ ] Windows/macOS/Linux native evidence と real SDK evidence を区別して記録する。
- [ ] 条件が整えば real Flutter + Emulator observer integration を実行する。
- [ ] acceptance evidence と retrospective を完成させる。
- [ ] 英日双方の plan を `docs/exec-plans/completed/` へ移動する。

実装 checkpoint（2026-09-08）: snapshot、PNG、意味情報と native input、wait/logcat、fencing、
証拠のための app/domain/Android adapter/companion/CLI コードと専用の負例 fixture を追加済み。
最終検証とレビュー指摘の対応が終わるまで、これらの実装項目の checkbox は未完了のまま維持する。
復旧は文書化した仕様の下で実装中であり、最終受け入れ済みとはしない。
新しい実 observer fixture に対する `go test -tags flutterintegration ./internal/cli -run '^$'` は成功した。
これは compile-only の証拠であり、Emulator の実行証拠ではない。
`go test ./internal/runtime/android/uihelper ./internal/execx -run
'TestVerify|TestBuildRejects|TestEmbedded|TestLoad|TestCapture' -count=1` は Linux / Go 1.27.1 で成功した。
Windows/macOS の native 証拠、最終 harness/race 検査、二つの実リースを使う observer の全体検証は未完了。

追加の local 証拠（2026-09-08）: 実装担当が一つの所有 Emulator で snapshot、screenshot、
明示的な復旧、通常 destroy を実行し、成功した。keyboard 表示後に古い意味情報 snapshot を再使用すると、
stale として安全に拒否された。これは二台のデバイスを使う observer 全体の受け入れ試験の代わりにはならない。
最新の harness 実行では単体テストと vet が完了した。英日を同時編集中の docs-check 失敗は、
作業途中の鮮度検査であり最終検証結果ではない。同期後の単独の
`go run ./tools/repoctl docs-check` は Linux / Go 1.27.1 で成功した。

checkbox は意図ではなく観測済みの完了を表す。チェック時には日付、command/test ID、結果、必要な環境情報を記録する。

## 想定外の発見

- 2026-09-08、二回目の実二台 Emulator 試験: `TestRealAndroidUIObserver` は 276.86 秒実行され、両アプリの起動に成功した後、新しい snapshot を使った意味情報 tap を stale として拒否した。保持した前後の証拠は、Android window ID が 8 から 11 に変わったことと、それに由来する fingerprint 以外は一致していた。window ID は UiAutomation の再接続をまたぐと変わるため、継続的な意味情報の識別要素にはできない。helper を window の意味情報に基づく metadata を使うよう修正し、三回目の実装版 build を生成した。実際の回帰再試行は未完了。
- 2026-09-08、追加検査: 全単体テストと vet が再び成功した。CLI の検証には log 内容の直接出力、明示的な duration 0 の拒否、存在しない lease の終了コード 2、壊れた registry の終了コード 7 を含めた。今回の翻訳同期後、単独の `go run ./tools/repoctl docs-check` が成功した。Go race 検証は実行中である。native または実 observer の最終受け入れ済みとはしない。

- 2026-09-08、実統合試験: `TestRealAndroidUIObserver` は Android activity の起動中、`am start -W` の `Status: timeout` により 222.41 秒で失敗し、observer の assertion には到達しなかった。registry の確認と通常の保守的 cleanup により、両 lease の released を確認した。readiness、所有権、timeout の検査は緩めていない。独立した再試行は別項に記録し、この失敗を observer の受け入れ証拠とはしない。
- 2026-09-08、app/復旧の独立レビュー: 元の結果を登録した後に最終分類の `SaveRun` が失敗すると、分類のない run が残る場合があった。従来の復旧は明示的な禁止分類だけを拒否していたため、分類の欠落によって host-process/evidence barrier を回避できた。現在は明示的に永続化された `termination-unconfirmed` の分類と、検証済みの元の結果証拠を要求する。`TestUIRecoverRefusesUnpersistedFailureClassification` で分類の書き込み失敗を検証する。復旧・fence・backend 診断の focused test は Linux / Go 1.27.1 で成功し、最終受け入れは引き続き未完了。
- 2026-09-08、CLI の独立レビュー: 以前の UI error は分類されず、tool の不足や registry 障害も既定の終了コード 2 になっていた。型による error 分類の伝播と CLI の対応付けにより、前提条件不足は 3、無効な option・対象選択・存在しない lease・stale/ambiguous は 2、registry/観測障害は 7 と区別する。診断 error に秘密情報を露出させない。

- 2026-09-08: 所有確認済み API 35 Emulator と一時 Flutter アプリで、`uiautomator dump` と自己対象の UiAutomation APK が Flutter の accessibility を取得できた。意味的クリックで Count 0 から Count 1 に変化。フォーカスした欄への `日本語 🙂 café` の入力後、新しいノード参照で `置換済み 🚀` への置換を行い、取得した accessibility text で確認した。対象アプリに instrumentation 依存は追加していない。
- 2026-09-08、失敗した方法: 未フォーカスの Flutter ノードは ACTION_SET_TEXT に true を返しても値を変更しなかった。フォーカスによるキーボード表示で window とノード番号が変化し、古い番号は別の対象で成功を返す場合もあった。実装では対応 action・focus の確認、意味情報の fingerprint 照合、入力後の値の一致確認を必須にする。dispatch の成功だけでは検証済みとしない。

- 2026-09-08: 初回の `go run ./tools/repoctl check` は単体テストと vet に成功した後、提供された日本語計画の `translation_of` と `source_sha256` 欠落で失敗した（AGENTENV-DOC-008）。検査を変更せず必須 metadata を補い、`repoctl docs-check` が成功した。

- 2026-09-08、独立レビュー: adapter と app の logcat データ受け渡し（`Raw` と `Binary`）の不一致、backend status の診断コードへの対応付け漏れ、操作後 fingerprint の欠落、不完全な階層取得の扱いを発見した。修正と回帰テストは本実装に含め、最終レビューと検証は引き続き未完了とする。app の fake だけでは具体的な adapter と証拠保存の境界を証明できないことが明らかになった。
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

## 成果と振り返り

未完了。

完了時には以下をまとめる。

- 採用した Android UI backend と選定理由
- helper を用いた場合の artifact/versioning model
- normalized snapshot contract
- stale-node semantics
- 実装した action と意図的に除外した action
- evidence/retention behavior
- Flutter semantics で得た知見
- native platform evidence
- real Android/Flutter validation
- API level / Unicode の既知制約
- richer gesture、visual regression、browser/CDPとの共通化等の follow-up

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
| U1 | UI機能を使わない既存Compose/Android/Flutter manifestとleaseが無変更で動く。 | Pending |
| U2 | UI commandはselected confirmed owned Android runtimeだけを解決しfirst-device fallbackを持たない。 | Pending |
| U3 | semantic snapshotがlease/runtime/snapshot identity付きdeterministic JSON/textを返す。 | Pending |
| U4 | raw/normalized snapshot evidenceがboundedかつatomicにpublishされる。 | Pending |
| U5 | real integrationでFlutter semantic label/text/editable controlがdocumented Android accessibility surfaceに現れる。 | Pending |
| U6 | screenshotがvalid PNG artifact、digest、exact device identityを持つ。 | Pending |
| U7 | fresh snapshot nodeのsemantic tapが意図したnodeを操作する。 | Pending |
| U8 | stale/ambiguous nodeはinput前に失敗しold coordinateへfallbackしない。 | Pending |
| U9 | explicit coordinate tapをsemantic tapと別に記録する。 | Pending |
| U10 | editable-node text replacementが定義したUnicode caseをshell escaping破損なしで扱う。 | Pending |
| U11 | Back/Home/swipeがselected serialだけに作用する。 | Pending |
| U12 | wait/pollはboundedでtimeout後にhidden effectを残さない。 | Pending |
| U13 | bounded logcatがpackage/PID scopeまたはbroader scopeを正直に記録しsecret redactionする。 | Pending |
| U14 | UI operationがlease fenceに参加しdestroyがin-flight mutating actionを追い越さない。 | Pending |
| U15 | 一方のleaseのsnapshot/node refを他方leaseに使えない。 | Pending |
| U16 | concurrent mobile leaseを独立観測・操作でき、一方destroy後も他方が利用可能。 | Pending |
| U17 | ambiguous Android identityをforce/fallbackで観測・操作しない。 | Pending |
| U18 | target Flutter/app repoへUI-test dependencyやsource変更を要求しない。 | Pending |
| U19 | helper使用時、そのartifact/version/digest/lifecycleを記録しtarget instrumentationを要求しない。 | Pending |
| U20 | observer docs/ExecPlanが英日双方に存在しtranslation checkを通る。 | Pending |
| U21 | architecture/docs/schema/full repository harnessがfinal implementationでpass。 | Pending |
| U22 | Go raceとWindows/macOS/Linux native fake/backend testがpass。 | Pending |
| U23 | real Flutter+Emulatorでsnapshot/screenshot/Unicode/stale rejection/action/logcat/cleanupを実証するか、具体的外部blockerをfake evidenceで代替せず記録する。 | Pending |

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

Milestone 1で解決する未確定事項:

1. platform shell primitive vs companion instrumentation APK
2. helper使用時のAndroidX stable/pre-release API選択
3. helper packaging/build/release方式
4. Flutter nodeにresource IDがない場合のstale fingerprint
5. snapshot defaultを全visible windowにするかselected app中心にするか
6. visible text / entered text / screenshot privacy policy
7. logcatを`ui logcat`に置くかfuture generic observationへ分離するか
8. identityが確実なDEGRADED/QUARANTINED lease上action policy
9. node/byte limitとtruncation
10. backendのAndroid API compatibility floor

public contract固定前にDecision Logへ記録する。
