---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/current-mobile.md
source_sha256: 7e5109a5c83fee72043c89a8e51fe4390ffb00a982141f48166e698ad4deaa10
---

# Mobile領域の現在のcorrectness監査

[English](current-mobile.md) · [監査index](index.ja.md) · [過去mobile資料](history-mobile.ja.md) · [実行基準](../../exec-plans/completed/repository-correctness-audit.ja.md)

Phase A、reviewのみ。対象`031869c8b9073b8e23bc17fbc55243666a52f557`、branch `audit/repository-correctness`。製品・test変更、commit、remote操作は行わない。追加testは一時Go overlayでのみ実行する。製品contractは[Android](../../product-specs/android-emulator.md)、[Flutter](../../product-specs/flutter-android-runtime.ja.md)、[UI](../../product-specs/android-ui-observer.ja.md)。baseline/native行列はroot監査が管理し、ここで重複実行しない。

## Phase C の対応状況

以下の記述は、Phase A の固定対象で得た証拠と当時の検証不足を保存したものである。
Phase B では mobile の 8 件をすべて採用した。現在の分類は **ACCEPT** であり、
後述の「未分類」「未修正」は Phase A 当時を指す。候補実装には次の検査と、修正した動作を繰り返し確認できるテストを追加した。

| 指摘 | 実装した検査 | 修正を確認するテストと判定内容 |
| --- | --- | --- |
| AUDIT-BOUNDARY-001 | デバイスから 2001 件を要求し、時間範囲で絞って最新 2000 件を保持し、実際の省略を明示 | `TestUILogExactTailUsesOverflowProof`: 1999/2000/2001 件と時間範囲外の追加記録 |
| AUDIT-REDACTION-001 | redaction 後にフィールドと escape 済み snapshot/result の実バイト数を再検査し、node 順序を保持して省略を明示 | `TestUIAuditUIRedactionBounds`、`TestUIFieldBoundaryAndSerializedObservationBoundary`: 登録 artifact、保持識別情報、4096 文字と 1 MiB の直前・一致・直後 |
| AUDIT-UI-001 | 編集可能値の自動抑制と秘密値の一致を区別 | `TestUIAuditEditableSnapshotRemainsActionable`: snapshot から set-text、callback 実行と読み戻し、完了した永続 run 2 件。既存の error redaction test も callback 実行を必須化 |
| AUDIT-IDENTITY-001 | 検証済み helper backend を正確に設定し、現在の host ファイルに依存せず記録済み provenance で復旧 | `TestUIAuditHelperBackendCarriesVerifiedDigest`: 同じ build で入力成功。`TestUIRecoveryUsesRecordedHelperWithAbsentOrReplacedHostFiles`: install せず検証済み helper を実際に停止 |
| AUDIT-REDACTION-003 | window 秘密値から派生する node hash を消去し、秘密値照合データがある場合は非公開の操作後 hash を省略 | `TestUIAuditWindowSecretClearsDerivedNodeHashes`、`TestUIOpaqueAfterFingerprintIsNotPublishedWithConfiguredSecrets`: 操作後の返却値と保存済み証拠 |
| AUDIT-PREREQUISITE-001 | path 正規化前に未設定・空白だけの helper 指定を拒否 | `TestUIAuditUnsetHelperRefusesCurrentDirectory`: ファイルがある cwd も暗黙の利用許可にしない |
| AUDIT-BOUNDARY-002 | provider 実行と永続 run 作成前に小数秒を拒否 | `TestUIAuditFractionalLogLookback` |
| AUDIT-LIFECYCLE-001 | 型付き preflight 失敗では完了確認状態を維持し、実際の dispatch 後の失敗では不確実性を保持 | `TestUINativePreflightThroughAppDoesNotBlockCleanup`: 失敗 run を記録後に解放成功。`TestUINativeDispatchedFailureRemainsUnconfirmed`: Home/Back/tap/swipe |

ローカル検証では `go test -race ./internal/app ./internal/runtime/android ./internal/runtime/android/uihelper -run 'TestUI|TestVerify|TestLoad' -count=1` が境界・復旧 fixture 修正後に 2 回成功した（6.122s/1.353s/1.012s と 5.675s/1.340s/1.010s）。
初版 fixture は JSON の `log` プロパティに必要な 9 bytes と、復旧時の SDK 応答を欠いていた。
どちらも失敗を確認して修正し、製品側の検査は緩めていない。初期サイズが上限を超える場合は、
切り詰め flag の変更だけで 2 bytes 減る場合でも、少なくとも一つの node を省略する。
独立レビュアーが操作後の非公開 tree hash による追加の漏えいを再現し、保守的な省略で解消を確認した。
候補の native 検証と全 harness の証拠は集約 ExecPlan が管理する。これらのローカル結果で
Windows/macOS の native 挙動や過去のすべての不具合の再導入検証まで証明したとは扱わない。

## 確認した不変条件の行列

「新規指摘なし」は確認したソースと局所assertionが一致するという意味で、網羅的証明ではない。現在の指摘はすべてPhase Bまで未分類。過去の修正を確認する既存テストの実行結果は、過去資料に記録した。

| 領域・production境界 | 確認した不変条件 | 証拠と結果 |
| --- | --- | --- |
| Android `Validate` / app allocation | tool、acceleration、ABI、共有ADB、path alias不備をsource/resource write前に拒否 | `app_test.go` preflightとCLI `create_store_test.go`。新規指摘なし。factoryのテストはCobraによるcommandの振り分けを通らない。 |
| Android `Inspect` / `Destroy` | kill/delete前にmarker、lease、tuple、PID/Job、console token、live portの一致が必要 | `lifecycle.go`とstale-marker/reused-port test。新規指摘なし。Windows再読取raceのtest不足M12は残る。 |
| Android process/port cleanup | root消失はtree消失ではなく、認証kill後もgroup/port不在を待つ | `DestroyWaitsForPostKillTreeConfirmation`、cancel test、containmentソース。新規指摘なし。native OS race証明は別。 |
| 共有ADB | Inspectは作成せず、非互換daemonを置換せず、lease containment外でstart | `adb.go` smart-socket preflightと、非互換・不正応答の共有ADB serverに操作用ADB commandを送らないことを確認するテスト。新規指摘なし。 |
| 混在app lifecycle | readiness期限を分離し、局所失敗後の独立cleanupを続け、全体fence喪失で停止 | `mixed_readiness_test.go`、`cleanup_siblings_test.go`、`waitReady`/`cleanup`。新規指摘なし。 |
| Flutter build path | 明示実行file、project/祖先symlink拒否、APK regularity、終了不明時のsource保持 | `flutter.go` Validate/Buildとapp guard。新規指摘なし。外部APK実行・実toolchainはnative証拠。 |
| Application desired state | 確認済install/reverse/launchと意図を区別し、reconcile用の実行file/directoryを保持 | `applicationConsistent`と必須field mutation test。新規指摘なし。 |
| Application cleanup | reverse削除前に現在endpointと記録所有を比較し、build/fence失敗時は保持 | `cleanupApplications`、reverse保持、build-barrier/preview test。新規指摘なし。 |
| UI intent/recovery | durable started intentと明確なrecovery分類がhelper限定停止に先行し、再実行せずhost errorをblock | `ui.go`、`ui_recovery.go`、`ui_recovery_test.go`。追加PR5追跡でAUDIT-LIFECYCLE-001、AUDIT-IDENTITY-001を確認。 |
| UI stale input | action前にsnapshot digest、lease/runtime/package/backend、fingerprint一意性・完全性を確認 | `loadUISnapshot`、Service選択、Java現tree確認、app stale test。追加PR5追跡でAUDIT-UI-001、AUDIT-IDENTITY-001を確認。同時UI変化が原子的に防げるとは主張しない。 |
| UI Java producer | 不完全hierarchyとaction読戻しには明確な証拠が必要 | `Observer.java`に実装あり。producer assertion不足M48/M49であり、現在の不具合確定ではない。 |
| UI log provider | attribution・時間窓と実truncationを上限等値と区別 | 現在の`AUDIT-BOUNDARY-001`。concrete adapter overlayが短い2000行ちょうどで失敗。 |
| UI秘匿化後の公開 | escape/metadata/秘匿化後にfield・正規化文書上限を守る | 現在の`AUDIT-REDACTION-001`。app entry overlayが過大・未truncated証拠を永続化。 |
| UI証拠永続化 | artifact/metadata失敗はmutation cleanup barrierを残し、処理完了済み不正captureはrunning誤barrierなしでfailedにできる | `TestUIEvidenceFailureKeepsCleanupBarrier`、`TestUIInvalidCompletedCaptureFinalizesFailed`。確認範囲で新規指摘なし。 |

## AUDIT-REDACTION-001 — Android UI公開結果が秘匿化後の上限を超える

重要度: **Medium**（決定的な公開境界違反）。Disposition: **未分類**。サイズ・完全性の指摘であり、再現で設定secretは露出していないためsecret漏洩とは呼ばない。

### 不変条件・位置・trigger・影響

[UI contract](../../product-specs/android-ui-observer.ja.md)はfield4096文字、正規化証拠1 MiB以下を要求する。`internal/app/ui.go:434`のsanitizeは458–460行でtext/description/hintのsecretを置換するが、field上限を再適用しない。350行付近のsnapshot公開は秘匿化後にmarshalし、`2<<20`を超えた場合のみ拒否する。そのため1–2 MiBの証拠を成功・完全として登録できる。helperの秘匿化前budgetだけでは秘匿化後budgetを保証できない。

設定secretを`qz`、通常表示textを`qz/`の繰返しとする。置換は2文字を`[REDACTED]`へ拡大し、元textはprovider budget内に収まる。巨大な単一文字列や不正provider応答は必要ない。

既存`uiFixture`のService/store/runtimeを使い、providerは正常snapshotを返す。overlayは`Service.UI(snapshot)`を呼び、実際に登録された`ui-snapshot`artifactを読む。独立した2ケース:

| Case | Provider応答 | 公開結果 | 期待 |
| --- | --- | --- | --- |
| field | 1 node、text/descriptionそれぞれ`qz/`500回、JSON3612 bytes | textが5500文字になりsnapshot成功 | 最終field4096以下、省略時は明示 |
| aggregate | 550 nodes、それぞれtext/descriptionが`qz/`100回、完全provider JSON584693 bytesでJava700000-byte guard未満 | 登録snapshot1429604 bytes、`Tree.Truncated=false` | 正規化artifact1048576 bytes以下で正しいtruncation、または明確な上限付き拒否 |

一時testは期待どおり0.038秒で失敗した。node数は1000未満、元fieldは4096未満、元JSONはprovider budget未満。すでに不正なproviderデータの受理に依存しない。通常の安定node ref/fingerprintを使い、公開app use caseを通すが、実Java/ADBではない。rootのnative実行とは別の証拠である。

### 再現と既存検証

一時overlay testは`internal/app`の`TestAuditUIRedactionBounds`。tracked sourceを変更せずGo `-overlay`でpackage testを追加し、`AUDIT_PRIVATE_SECRET=qz`、`uiFixture`、上記node数・文字列を使う。`Service.UI`後のrune数と、登録`ui-snapshot`artifactの実byte長`1<<20`以下をassertする。local overlay pathは監査担当へ伝達したが、永続文書のmachine固有依存にはしていない。

既存`TestUIEvidenceRedactsEnteredAndEditableText`はsecret置換・抑制、`TestUIEditableDescriptionAndHintRedacted`はeditable各field、`TestUIInvalidCompletedCaptureFinalizesFailed`は明らかな過大入力を検証する。正常producer budgetと置換拡大、最終serialized-byte測定を独立に組み合わせていない。helper上限も秘匿化前である。Phase Aで製品修正と、修正を確認する恒久テストは追加していない。

### Escapeと同型分析

検出S9、現実的最早S2。分類は`BOUNDARY_GAP`、`COMPOSITION_GAP`、`ORACLE_COUPLING`。S2で置換率を独立した境界次元にできた。S3は秘匿化と上限を別検査し、S4は合成provider応答を受け取っても変換後永続byteを測定しなかった。S5 nativeの通常textは拡大境界に届かず、S6に意味的field/artifact size validatorはない。S7/S8で1 MiB contractとappの2 MiB guardを比較し、変換後の判定基準を要求する機会があったが、その実施証拠はない。

過去との関係はM46のadapter/app証拠結合、M49のproducer完全性不足。別領域ではBrowser PR10秘匿化後上限と同型。disposition待ち予防策は、最終公開field/serialized-byte checker、secret拡大・JSON escape後のlimit−1/exact/+1、実artifact byte測定、保持node本人性・truncationのassertion。期待検出S2/S4。testを通すため文書上限を黙って引き上げたり、truncationなしで証拠を捨てたりしない。

## 他の指摘とレビューの限界

[AUDIT-BOUNDARY-001](history-mobile.ja.md)にdevice-tail/helperの曖昧さとoverlay証拠を記録した。曖昧さを残しており、localの`>=`変更だけを検証済み修正とはしない。

Java field limiterはapp秘匿化前に値を短縮しているためsecret prefixの扱いは追加security review候補。ただしproducerを通す具体的再現なしで追加の現在指摘とはしない。今回範囲では不正APK実行、悪意あるaccessibility service、malicious-code sandbox、全OS syscallのinterleaving証明を行っていない。未確認は成功としない。

## PR5最終指摘の現在再現

過去資料を24件補完する過程で、PR5最終6指摘が固定対象にも残ることを確認した。下記はすべて未分類で、製品修正・恒久test追加はまだ行っていない。`-overlay`で追加したtestのみ実行し、app0.016秒、Android0.007秒、helper0.003秒で各条件が失敗した。Go overlayは一時領域に保管して監査担当へ渡した。

### AUDIT-UI-001

重要度: **High**。出典: [PR5 comment 3954080090](https://github.com/mahcialet/agent-env/pull/5#discussion_r3954080090)、過去M72。位置: `internal/app/ui.go:461`。再現: `TestAuditEditableSnapshotRemainsActionable`。

自動抑制されたeditable fieldでも安全なsemantic fingerprintを保持する。helperが空text/description/hintと有効fingerprintを持つeditable nodeを返すと、sanitizerは空値を`[REDACTED]`へ変え、その変化を設定secret一致と扱って全fingerprintを除去する。Service snapshotは成功するが、保持nodeへの後続set-textはprovider前に`AGENTENV-UI-STALE: missing or ambiguous snapshot node`で失敗する。overlayで保存fingerprint欠落とapp拒否を再現した。さらに監査担当が十分なdisk空きのあるvolume上のcleanな固定cloneで未変更の実UI fixtureを実行し、2 leaseともREADYになった後、最初のtapが同じsnapshot-node欠落・曖昧errorで失敗した（75.11秒）。実Flutter fixtureは別に成功した（73.54秒）。これはsnapshot action不具合のnative裏付けであり、UI baseline成功ではない。

既存`TestUIEvidenceRedactsEnteredAndEditableText`は入力text確認callbackの実行をassertしないため、意図するerror秘匿化経路を通らなくても成功する。stale拒否もsecretなしerrorなので条件を満たす。検出S9、最早S3。`ORACLE_COUPLING`、`FAILURE_INJECTION_GAP`、`COMPOSITION_GAP`。予防策はsnapshot→set-textの結合成功とcallback回数、別途実secret由来fingerprint拒否を検証すること。無差別なhash復活ではなく、抑制とsecret由来本人性を区別する。

### AUDIT-IDENTITY-001

重要度: **High**。出典: [PR5 comment 3954080094](https://github.com/mahcialet/agent-env/pull/5#discussion_r3954080094)、過去M73。位置: `internal/runtime/android/ui.go:72`。再現: `TestAuditHelperBackendCarriesVerifiedDigest`。

semantic snapshot/action/recoveryの基準は検証済source/APK digestに結び付く必要がある。ObserveUIはBackendを`android-shell-v1`に初期化し、空の場合のみhelper本人性を設定するため、通常helper branchでは設定されない。有効helperと正しいexpected digestを使うoverlayでも一般名が返り、helper buildを区別できない。recovery fallbackは`uiautomation-v1:source=...:apk=...`を要求するため、このbranchの証拠ではhost directory消失後に本人性を復元できない。

既存`TestUISemanticActionRejectsChangedBackendBeforeDeviceInput`は任意の異なる文字列を使い、正しいdigestでも誤った一般名でも成功する。検出S9、最早S3。`ORACLE_COUPLING`、`INVARIANT_GAP`、`COMPOSITION_GAP`。予防策はsnapshotの正確なbackend、同build許可・別build拒否、host fileなしで記録本人性からのrecovery検証。未検証identityへのfallbackは不可。

### AUDIT-REDACTION-003

重要度: **High**。出典: [PR5 comment 3954080097](https://github.com/mahcialet/agent-env/pull/5#discussion_r3954080097)、過去M74。位置: `internal/app/ui.go:447`。再現: `TestAuditWindowSecretClearsDerivedNodeHashes`。

secret由来識別子をoffline照合oracleとして残さない。設定secretを含むwindow titleは秘匿化されKeyも消えるが、node Fingerprint除去はnodeSecretだけを条件にする。Java producerはwindow keyをroot ancestryへ含めてnode本人性をhashする。app entry overlayではwindow keyが消えても登録snapshotのnode fingerprintが残った。secretを含むwindow情報への入れ子依存はソースで確認。overlayは代表hashを用い、native password crackingの実行は主張しない。

既存秘匿化testは平文探索のみで派生識別子を確認せず、windowの秘密値から派生したhashが消えることを直接確認するテストを特定できなかった。検出S9、最早S2。`INVARIANT_GAP`、`NEGATIVE_FIXTURE_GAP`、`ORACLE_COUPLING`。予防策はproducer相当hashをwindow metadataから生成し、title/root secret秘匿化後のraw・normalized・result・runで派生key/hashが残らないことを検証する。secretと無関係なmetadataの安全なfingerprintだけを維持する。

### AUDIT-PREREQUISITE-001

重要度: **High**。出典: [PR5 comment 3954080101](https://github.com/mahcialet/agent-env/pull/5#discussion_r3954080101)、過去M75。位置: `internal/runtime/android/uihelper/helper.go:56`。再現: `TestAuditUnsetHelperRefusesCurrentDirectory`。

companion installには明示directory設定が必要。Load("")はfilepath.Absでcwdを解決する。整合したobserver metadata/APKを置いた一時cwdを空argumentで受理した。ObserveUIはos.Getenvを直接Loadへ渡すため、opt-in未設定でもcwd fileを選択・installし得る。overlayは実loaderを通し、実device installは行わない。受理metadataから検証済byte installへの経路はソースで確認。

既存provenance fixtureは必ず明示directoryを渡し、未設定とdigest不一致を区別していない。検出S9、最早S2。`NEGATIVE_FIXTURE_GAP`、`COMPOSITION_GAP`。予防策は空・空白・path行列と、file入りcwdで環境変数未設定のapp/adapter testによりinstallなしをassertすること。正規化前に未設定を拒否し、記録identityだけによるrecovery例外を明示する。

### AUDIT-BOUNDARY-002

重要度: **Medium**。出典: [PR5 comment 3954080103](https://github.com/mahcialet/agent-env/pull/5#discussion_r3954080103)、過去M76。位置: `internal/app/ui.go:190`。再現: `TestAuditFractionalLogLookback`。

受け入れたlookback時間から要求logを黙って除外しない。Service.UIはSince=1900msを受理するが、整数除算でproviderへ1秒を送る。app overlayで実requestを確認して失敗した。1–1.9秒前のlogが要求内なのに除外される。行数上限ちょうどの問題とは別。

既存log fixtureは整数秒のみ。検出S9、最早S2。`BOUNDARY_GAP`、`NEGATIVE_FIXTURE_GAP`。予防策は秒境界・最大時間の−epsilon/exact/+epsilonでprovider要求とcutoffを確認すること。既存公開contractに合わせ切上げまたは非対応小数の明示拒否をdispositionで選び、黙った短縮は認めない。

### AUDIT-LIFECYCLE-001

重要度: **High**。出典: [PR5 comment 3954080105](https://github.com/mahcialet/agent-env/pull/5#discussion_r3954080105)、過去M77/M65。位置: `internal/runtime/android/ui.go:211`。再現: `TestAuditNativePreflightRemainsConfirmed`。

native入力前ownership/protocol preflight失敗をdispatch不明と扱わない。native branchは結合preflight/input呼出前にConfirmed=falseとし、adbPreflightError認識は追加のfalse設定を防ぐだけで元のfalseを戻さない。adapter overlayでconsole名を変更するとErrResourceIdentity、runner commandゼロ、Confirmed=falseとなった。appはtermination-unconfirmedとしてrunning cleanup barrierを保持し、RecoverUIはnative Back/Home/tap/swipeを対象外とする。adapter結果は実行証拠、app barrierはソース追跡であり追加結合replay済みとはしない。

既存TestUIOwnershipFailureCannotDispatchInputはobservationを捨て、error本人性と入力ゼロしかassertしない。過去修正はtyped errorを認識したが最終返却状態を見なかった。検出S9、最早S4。`FAILURE_INJECTION_GAP`、`COMPOSITION_GAP`、`ORACLE_COUPLING`。予防策はapp経由preflight注入でdurable failed・running barrierなし・後続安全cleanupを確認し、実入力開始後の不明状態は別に保持すること。全errorをConfirmed=trueにする修正は安全性を弱める。
