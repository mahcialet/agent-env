---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/history-process-browser.md
source_sha256: 2f4a789a3df7202f7b20d01127f29d61ad918fba10642728a0c2cca647f34e25
---

# 過去レビュー資料: process・browser・MVP

[English](history-process-browser.md) · [作業を定める監査Plan](../../exec-plans/active/repository-correctness-audit.ja.md)

Phase A対象: `031869c8b9073b8e23bc17fbc55243666a52f557`。
本書はレビュー専用の過去資料であり、処置判断や修正実施ではない。
56項目はbrowserのPR指摘22件（9 + 8 + 5）と実装中・nativeでの発見、
process関連10件、MVPのまとめられた15件を区別する。
本資料の作成では製品・test変更もremote操作も行っていない。

## 参照元と読み方

- P: [persistent process Plan](../../exec-plans/completed/persistent-process-runtime.ja.md)の統合checkpoint、発見、初回native CI、Windows修正checkpoint、振り返り。
- D: [process destroy previewレビュー](../../exec-plans/completed/process-destroy-preview-review.ja.md)全体。
- B1/B2/B3: [browser Plan](../../exec-plans/completed/browser-cdp-automation.ja.md)のPR #10第1・2・3回対応、発見、判断、振り返り。BIは独立した実装・レビュー中の追加発見、BNはnative・fixtureの追加対応。
- M: [MVP Plan](../../exec-plans/completed/agent-env-mvp.md)の発見、独立review checkpoint、cancelレビュー、native・並行修正、renewal fixture、PR #1対応。これは翻訳例外の履歴文書。

多くの項目には元の重要度が記録されておらず、推測で補わない。
HB26は明記されたP1、HB27はP2、HM10/HM11はP1である。
他の行は事後に重要度を割り当てず影響を記す。参照元にはPR番号・commit・CI linkがあるが、
各行に対応するreview thread ID一覧はない。そのため本書のIDは資料内の安定IDであり、
元thread IDを捏造したものではない。MVPは複数修正をまとめており、個別commentまで
分解できない部分はHM08のまとめを維持する。通常のformat・翻訳準備の失敗や
前提toolの観測を独立した製品不具合として数えない。

段階は検出時/現実的に最も早く防げた段階を示す。S8は記録された独立・PRレビュー、
S5はnative実行、S4は実際の並行統合である。「implementation」は参照元に
それ以上細かな検出段階がないことを示し、勝手にS8としない。
最早段階は当時存在したcontractとinterfaceに基づく今回の評価であり、
そのtestが当時存在したという主張ではない。

## coverageの意味

以下の製品・test pathは、明示的な`tools/`・`docs/`・`.github/`・
`ARCHITECTURE.md`以外は`internal/`からの相対path。
後続のfile名なしtest名は直前のtest fileに属する。

- F: 公開package・app・store入口を実際のlocal永続化・filesystemまたは注入effectで通る回帰を確認。対象部分の証明でありCLI・provider全体実行ではない。
- C: protocol fixtureでCDP component入口を通る回帰を確認。これだけではJavaScriptや実browserを実行しない。
- H: helperのみの回帰。局所不変条件は確認するが公開入口での組合せや呼出順を証明しない。
- N: 実native fixtureが存在し、決定的component testを併用することもある。今回のnative実行証拠は監査baselineが管理し、source確認から実行済とは推定しない。
- D: 文書・processの主張。記述と記録済CIを確認したが、docs-checkは意味の正しさや全受入成功を証明しない。
- U: 元の回帰testとの対応が未確定。近傍coverageを証明として数えない。

全行は固定対象にも引き続き適用される。U以外は記載した局所不変条件のassertが存在し、
H/Dは挙動全体を証明しないことを明示する。製品側の呼出が生きていることは確認したが、
元不具合を再注入した際に56件全てが失敗するとは主張しない。
本資料ではmutation replayを実施せず、testも弱めていない。
今回のaudit finding IDは本資料では未割当。末尾のcoverage課題は処置判断前に
追加のPhase A証拠を必要とする。

## 過去の指摘

### HP01 — P

probe成功後にprocessが終了してもREADYになった。probe後に所有processを再観測する。

- 検出/最早段階: S8/S4、見逃し分類: `COMPOSITION_GAP`。
- 製品側: `app/process.go:confirmProcessesReady`。
- 現在の回帰: `app/process_lifecycle_test.go:TestProcessExitDuringSuccessfulProbeCannotBecomeReady`、coverage F。

### HP02 — P

生成前失敗の証明をcleanup不明としていた。型付き未生成証拠とPIDゼロでpreparedを解放し、証拠のないidentityゼロは解放しない。

- 検出/最早段階: S8/S3、見逃し分類: `FAILURE_INJECTION_GAP`。
- 製品側: `app/process.go:startProcess; runtime/process/process.go:Start`。
- 現在の回帰: `app/process_lifecycle_test.go:TestProcessKnownNoSpawnCompensatesPreparedState; runtime/process/process_test.go:TestProvenNoSpawnFailureCanReleaseWithoutQuarantine`、coverage F。

### HP03 — P

symlink経由state homeの不変pathが予約後に変化した。予約前に実pathを確定する。

- 検出/最早段階: S8/S4、見逃し分類: `COMPOSITION_GAP`。
- 製品側: `app/lifecycle.go:CanonicalFuture before reservation`。
- 現在の回帰: `cli/process_native_test.go:TestPersistentProcessNativeCLI`、coverage N。

### HP04 — P

生port名が明示endpoint aliasを上書きした。宣言を優先する。

- 検出/最早段階: S8/S2、見逃し分類: `NEGATIVE_FIXTURE_GAP`。
- 製品側: `app/readiness.go:Endpoints`。
- 現在の回帰: `app/plan_process_test.go:TestProcessEndpointAliasWinsOverRawRuntimePort`、coverage F。

### HP05 — P

readinessリテラルが継承secretを保存できた。snapshot前に拒否し明示環境参照は許可する。

- 検出/最早段階: S8/S2、見逃し分類: `HELPER_ONLY,NEGATIVE_FIXTURE_GAP`。
- 製品側: `app/plan.go:BuildPlan -> validateProcessSecrets`。
- 現在の回帰: `app/plan_process_test.go:TestProcessReadinessRejectsLiteralInheritedSecretBeforeSnapshot`、coverage H。

### HP06 — P

完了所有Jobを過去PID再利用で拒否した。durableなJob消滅証拠を先に調べ、実行中の検査は厳格に維持する。

- 検出/最早段階: S5/S5、見逃し分類: `NATIVE_EVIDENCE_GAP,CONCURRENCY_GAP`。
- 製品側: `execx/detached_windows.go; execx/detached.go`。
- 現在の回帰: `execx/managed_windows_test.go:TestManagedWindowsCompletedJobIgnoresReusedHistoricalPID`、coverage N。

### HP07 — P

終端予約検査が再destroyや解放後quarantineを拒否した。予約履歴を観測状態から分離しportを再取得しない。

- 検出/最早段階: implementation/S3、見逃し分類: `COMPOSITION_GAP`。
- 製品側: `store/sqlite/process.go:saveProcesses`。
- 現在の回帰: `store/sqlite/process_test.go:TestProcessReleasedReservationsCannotResurrect; TestProcessTerminalReservationKeepsPostReleaseQuarantineVisible`、coverage F。

### HP08 — P

現環境と起動receiptだけでは起動時secret証拠を失った。独立fingerprintを起動前に保存しreceipt失敗でも維持する。

- 検出/最早段階: implementation/S3、見逃し分類: `FAILURE_INJECTION_GAP`。
- 製品側: `runtime/process/process.go; runtime/process/redaction.go`。
- 現在の回帰: `runtime/process/process_test.go:TestLogsRedactOriginalSecretsWhenEnvironmentChanges; TestReceiptFailureStillHasDurableLogRedactionForCompensation`、coverage F。

### HP09 — P

固定範囲割当が使用中の先頭portを繰り返し選び得た。OS選択loopback listenerを予約commitまで保持する。

- 検出/最早段階: implementation/S2、見逃し分類: `BOUNDARY_GAP,ORACLE_COUPLING`。
- 製品側: `store/sqlite/process.go:allocateProcesses`。
- 現在の回帰: `store/sqlite/process_test.go:TestProcessDynamicReservationAvoidsExternallyBoundPort; TestProcessConcurrentReservationPortsAreDisjoint`、coverage F。

### HP10 — D

正しいprocess照会後にCompose previewへ分岐し終了stateのcleanupを表示しなかった。状態とforceの6通りで表示と無副作用を確認する。

- 検出/最早段階: S8/S4、見逃し分類: `HELPER_ONLY,COMPOSITION_GAP`。
- 製品側: `app/destroy_preview.go`。
- 現在の回帰: `app/process_lifecycle_test.go:TestProcessDestroyPreviewReportsCleanupWithoutEffects`、coverage F。

### HB01 — B1

native受入成功前にPlanを完了扱いにした。古い成功CIで後続修正を受け入れず、最終検証後に移動する。

- 検出/最早段階: S8/S7、見逃し分類: `REVIEW_CHECKLIST_GAP`。
- 製品側: `docs/exec-plans/completed/browser-cdp-automation.md`。
- 現在の回帰: `repoctl docs-check plus recorded final native CI`、coverage D。

### HB02 — B1

URL比較がopaque・継承・blob frameを誤分類した。native placeholder originと抜けたOOPIFにブラウザによるorigin・構造証明が必要。

- 検出/最早段階: S8+S5/S3、見逃し分類: `ORACLE_COUPLING,NATIVE_EVIDENCE_GAP`。
- 製品側: `browser/cdp/snapshot.go:frame proof`。
- 現在の回帰: `browser/cdp/snapshot_test.go:TestFrameClassificationUsesSecurityOrigin; TestOutOfProcessFrameCannotBeSilentlyOmitted; cli/browser_native_test.go`、coverage N。

### HB03 — B1

roleのみのURL waitが空predicateを受理した。依存先に触れる前にsubstring必須とrole拒否を確認する。

- 検出/最早段階: S8/S2、見逃し分類: `NEGATIVE_FIXTURE_GAP`。
- 製品側: `app/browser.go:validation`。
- 現在の回帰: `app/browser_test.go:TestBrowserURLWaitRequiresSubstring; cli/browser_test.go:TestBrowserRejectsMissingOrUnsafeCLIInputBeforeStore`、coverage F。

### HB04 — B1

network metadataが文字列上限を迂回した（1,230,500対65,536 byte）。全保存文字列を加算する。

- 検出/最早段階: S8/S2、見逃し分類: `BOUNDARY_GAP`。
- 製品側: `browser/cdp/capture.go:boundCaptureStrings`。
- 現在の回帰: `browser/cdp/capture_test.go:TestNetworkCaptureBoundsEveryPersistedString; TestNetworkCaptureCountsAllStringsAgainstTotalBudget`、coverage C。

### HB05 — B1

短縮DOM名を完全な証拠として返した。実際の省略を明示する。

- 検出/最早段階: S8/S2、見逃し分類: `BOUNDARY_GAP`。
- 製品側: `browser/cdp/snapshot.go:domSnapshot`。
- 現在の回帰: `browser/cdp/snapshot_test.go:TestDOMNameTruncationIsReported`、coverage C。

### HB06 — B1

実装済browser providerを将来作業と説明した。日英の記述を実装範囲に合わせる。

- 検出/最早段階: S8/S7、見逃し分類: `REVIEW_CHECKLIST_GAP`。
- 製品側: `ARCHITECTURE.md:Android/browser paragraphs`。
- 現在の回帰: `paired architecture inspection; docs-check`、coverage D。

### HB07 — B1

不完全AX証拠でgoneを成功扱いにした。不完全な証拠は不在証明にならない。

- 検出/最早段階: S8/S2、見逃し分類: `INVARIANT_GAP,NEGATIVE_FIXTURE_GAP`。
- 製品側: `browser/cdp/client.go:wait`。
- 現在の回帰: `browser/cdp/snapshot_test.go:TestGoneCannotSucceedWithTruncatedSnapshot`、coverage C。

### HB08 — B1

無関係eventがqueueを埋め通常commandを切断した。対象session/methodだけ格納しoverflowは失敗とする。

- 検出/最早段階: S8/S3、見逃し分類: `COMPOSITION_GAP`。
- 製品側: `browser/cdp/transport.go:subscribe/read loop`。
- 現在の回帰: `browser/cdp/transport_test.go:TestTransportIgnoresUnsubscribedEvents; TestTransportCaptureIgnoresOtherSessionsAndMethods; TestTransportSubscribedOverflowFailsClosed`、coverage C。

### HB09 — B1

入力intentがpage/snapshot/nodeを欠いた。不明結果も含め全semantic effect前にtarget identityを保存する。

- 検出/最早段階: S8/S4、見逃し分類: `FAILURE_INJECTION_GAP`。
- 製品側: `app/browser.go:Browser run intent`。
- 現在の回帰: `app/browser_test.go:TestBrowserSemanticProvenanceSurvivesUncertainInput`、coverage F。

### HB10 — BI

AX取得中のframe変化で旧identityと証拠を組み合わせた。構造・originを再検査し変更時は全証拠を破棄する。

- 検出/最早段階: S8/S3、見逃し分類: `CONCURRENCY_GAP`。
- 製品側: `browser/cdp/snapshot.go:post-collection proof`。
- 現在の回帰: `browser/cdp/snapshot_test.go:TestSnapshotDiscardsAXWhenFrameProofChanges; TestWaitRetriesFrameChangesButNeverPublishesPartialEvidence`、coverage C。

### HB11 — BN

tree消滅証明後もWindows共有違反でprofile削除に失敗した。そのOS errorだけ期限と毎回のpath再証明付きでretryする。

- 検出/最早段階: S5/S5、見逃し分類: `NATIVE_EVIDENCE_GAP`。
- 製品側: `runtime/process/cleanup.go; cleanup_windows.go`。
- 現在の回帰: `runtime/process/cleanup_windows_test.go:TestWindowsStateRemovalRetriesHeldFile; cleanup_test.go:TestStateRemovalRevalidatesOwnerBeforeRetry`、coverage N。

### HB12 — B2

入力後protocol/readback errorでaction全体を確認済にした。例外やbool欠落は不明のまま維持する。

- 検出/最早段階: S8/S3、見逃し分類: `FAILURE_INJECTION_GAP`。
- 製品側: `browser/cdp/client.go:Observe; actions.go:act`。
- 現在の回帰: `browser/cdp/input_review_test.go:TestPostInsertErrorRemainsUncertain; app/browser_test.go:TestBrowserUncertainMutationRetainsCleanupBarrier`、coverage C+F。

### HB13 — B2

focus handlerが入力先を変更し非active tabのactiveElementだけではevent配信を保証できなかった。page有効化・再検証後、focusとselect-all後にも対象とdocumentのfocusを証明する。

- 検出/最早段階: S8+S5/S3、見逃し分類: `COMPOSITION_GAP,NATIVE_EVIDENCE_GAP`。
- 製品側: `browser/cdp/actions.go:act/verifyFocus`。
- 現在の回帰: `browser/cdp/input_review_test.go:TestInputFocusRedirectRefusesKeyboardDispatch; TestSelectionFocusChangeRefusesTextInsertion; TestInactiveDocumentRefusesKeyboardDispatch; TestActivationMutationRefusesKeyboardDispatch; cli/browser_native_test.go`、coverage N。

### HB14 — B2

DOM snapshotがAX同等のorigin・構造保護を欠いた。両経路で前後証明と承認frame IDが必要。

- 検出/最早段階: S8/S3、見逃し分類: `COMPOSITION_GAP`。
- 製品側: `browser/cdp/snapshot.go:domSnapshot; snapshot.go`。
- 現在の回帰: `browser/cdp/snapshot_test.go:TestDOMSnapshotOriginAndDocumentProof`、coverage C。

### HB15 — B2

他tabのOOPIFが選択pageを拒否させた。親不明targetを選択sessionのownerと照合し外部contentは採用しない。

- 検出/最早段階: S8/S3、見逃し分類: `NEGATIVE_FIXTURE_GAP`。
- 製品側: `browser/cdp/snapshot.go:iframe ownership`。
- 現在の回帰: `browser/cdp/snapshot_test.go:TestUnparentedIframeTargetsArePageScoped; cli/browser_native_test.go`、coverage N。

### HB16 — B2

redactionでconsole/network文字列が上限超過した。provider値だけでなく保存run.jsonを再制限する。

- 検出/最早段階: S8/S4、見逃し分類: `COMPOSITION_GAP,BOUNDARY_GAP`。
- 製品側: `app/browser_redaction.go:boundRedactedBrowserCapture`。
- 現在の回帰: `app/browser_review_test.go:TestBrowserCaptureBoundsAfterRedaction`、coverage F。

### HB17 — B2

URL predicateが加工済dataを使いmockのFrame.urlも実ChromeのurlFragmentと異なった。raw URLで判定し証拠保存時に加工する。

- 検出/最早段階: S8+S5/S3、見逃し分類: `ORACLE_COUPLING`。
- 製品側: `browser/cdp/client.go:wait; snapshot.go:frame URL`。
- 現在の回帰: `browser/cdp/input_review_test.go:TestURLWaitUsesRawURLButPersistsScrubbedEvidence; cli/browser_native_test.go`、coverage N。

### HB18 — B2

期限終了時にqueue内eventを黙って省略した。格納停止と残件確認を原子的に行い省略を明示しoverflowはerrorのまま保持する。

- 検出/最早段階: S8/S3、見逃し分類: `CONCURRENCY_GAP,BOUNDARY_GAP`。
- 製品側: `browser/cdp/capture.go:finishCapture; transport.go:subscribe`。
- 現在の回帰: `browser/cdp/capture_test.go:TestCaptureDeadlineMarksPendingEventsTruncated`、coverage H。

### HB19 — B2

保存browser manifestのdigestを検証していなかった。照会・effect前に変更を拒否する。

- 検出/最早段階: S8/S2、見逃し分類: `NEGATIVE_FIXTURE_GAP,HELPER_ONLY`。
- 製品側: `app/browser.go:Browser -> selectBrowser`。
- 現在の回帰: `app/browser_review_test.go:TestBrowserRejectsChangedStoredManifest`、coverage H。

### HB20 — B3

redaction後semantic証拠がfield/1 MiB制限超過や旧2 MiB guardで失敗した。最終encodingを制限しnode identityを保ちread-only runを完了しtruncated入力は拒否する。

- 検出/最早段階: S8/S4、見逃し分類: `COMPOSITION_GAP,BOUNDARY_GAP`。
- 製品側: `app/browser_redaction.go:boundRedactedBrowserSnapshot`。
- 現在の回帰: `app/browser_semantic_bounds_test.go:TestBrowserSemanticBoundsAfterRedaction`、coverage F。

### HB21 — B3

AXで見えるclosed-root controlをhost.shadowRoot探索が拒否した。targetから外向きに全hostのhit/focusを証明しoverlayは拒否する。

- 検出/最早段階: S8/S5、見逃し分類: `NATIVE_EVIDENCE_GAP,ORACLE_COUPLING`。
- 製品側: `browser/cdp/actions.go:nodeHit/verifyFocus`。
- 現在の回帰: `cli/browser_native_test.go:TestBrowserNativeCLI /closed-shadow`、coverage N。

### HB22 — B3

enable後に計時し20 ms要求が500 ms以上待てた。購読・enableを期限に含め未完enableは失敗とする。

- 検出/最早段階: S8/S3、見逃し分類: `BOUNDARY_GAP,COMPOSITION_GAP`。
- 製品側: `browser/cdp/capture.go:capture`。
- 現在の回帰: `browser/cdp/capture_test.go:TestCaptureDurationIncludesDomainEnable`、coverage C。

### HB23 — B3

単一・複数frameや末尾空frameでAXちょうど2048件をtruncatedにした。実際の省略が必要。

- 検出/最早段階: S8/S2、見逃し分類: `BOUNDARY_GAP`。
- 製品側: `browser/cdp/snapshot.go:snapshot`。
- 現在の回帰: `browser/cdp/snapshot_test.go:TestSnapshotNodeLimitMarksOnlyOmittedNodes`、coverage C。

### HB24 — B3

非string・不正・null console引数を黙って完全扱いで捨てた。private詳細は保存せず省略を明示する。

- 検出/最早段階: S8/S2、見逃し分類: `NEGATIVE_FIXTURE_GAP`。
- 製品側: `browser/cdp/capture.go:console decoding`。
- 現在の回帰: `browser/cdp/capture_test.go:TestConsoleMarksOmittedArgumentsTruncated`、coverage C。

### HB25 — BI

isolated context欠落でstale-target回帰がmutation前に失敗し合格した。失敗・無effectだけでなく注入点到達をassertする。

- 検出/最早段階: S8/S2、見逃し分類: `ORACLE_COUPLING,NEGATIVE_FIXTURE_GAP`。
- 製品側: `browser/cdp/actions.go:act`。
- 現在の回帰: `browser/cdp/actions_test.go:TestNodeChangesDuringOwnershipVerificationNeverInputs`、coverage C。

### HB26 — BI

P1: fence喪失errorがredaction前の生観測を露出した。出力を消し旧ownerのfinalizeを禁止する。

- 検出/最早段階: S8/S4、見逃し分類: `FAILURE_INJECTION_GAP`。
- 製品側: `app/browser.go:lost-fence return`。
- 現在の回帰: `app/browser_test.go:TestBrowserLockLossDoesNotExposeObservation`、coverage F。

### HB27 — BI

P2: 入力とbrowser名が一致するとredactionが権限identityを壊した。contentだけ加工しidentityを維持する。

- 検出/最早段階: S8/S4、見逃し分類: `ORACLE_COUPLING,COMPOSITION_GAP`。
- 製品側: `app/browser_redaction.go:structured redaction`。
- 現在の回帰: `app/browser_test.go:TestBrowserPriorTextRedactionPreservesAuthority; cli/browser_native_test.go`、coverage N。

### HB28 — BN

macOS synthetic Meta+Aで確実に全選択できなかった。明示selectAll編集commandとreadbackが必要。

- 検出/最早段階: S5/S5、見逃し分類: `NATIVE_EVIDENCE_GAP`。
- 製品側: `browser/cdp/actions.go:selectAll`。
- 現在の回帰: `browser/cdp/actions_test.go:TestSelectAllUsesExplicitEditingCommandOnEveryPlatform; cli/browser_native_test.go`、coverage N。

### HB29 — BN

Windows privacy判定が正当なUnicode identity pathをsecretと誤認した。JSONをdecodeしescape込みの実入力を照合する。

- 検出/最早段階: S5/S2、見逃し分類: `ORACLE_COUPLING`。
- 製品側: `cli/browser_native_test.go:privacy oracle`。
- 現在の回帰: `cli/browser_native_test.go:TestBrowserEvidenceSecretDetection`、coverage H。

### HB30 — BN

Linux sandbox起動とWindows LPAC実行file参照が失敗した。固定installationへの必要accessだけ設定しsandbox全体は緩めない。

- 検出/最早段階: S5/S5、見逃し分類: `NATIVE_EVIDENCE_GAP`。
- 製品側: `.github/workflows/browser.yml:AppArmor/LPAC ACL`。
- 現在の回帰: `cli/browser_native_test.go:TestBrowserNativeCLI startup/sandbox checks`、coverage N。

### HB31 — BN

Create fixtureの50 ms期限がbrowser assert前のSQLiteで切れた。setup予算を分離し操作設定は戻す。

- 検出/最早段階: S5/S3、見逃し分類: `ORACLE_COUPLING`。
- 製品側: `app/browser_test.go:browserFixture`。
- 現在の回帰: `app/browser_test.go:TestBrowserLifecycleGuards`、coverage F。

### HM01 — M

同等sourceでもWindows CRLF checkoutでformat検査が失敗した。CRLFのみ正規化し実差分は拒否する。

- 検出/最早段階: S5/S2、見逃し分類: `NATIVE_EVIDENCE_GAP,BOUNDARY_GAP`。
- 製品側: `tools/repoctl format check`。
- 現在の回帰: `tools/repoctl/main_test.go:TestFormattingCRLFAndActualDrift`、coverage H。

### HM02 — M

欠落worktreeのaliasがGit登録と一致しなかった。登録証拠を保ち既存祖先identityを比較する。

- 検出/最早段階: S5/S3、見逃し分類: `NATIVE_EVIDENCE_GAP`。
- 製品側: `source/gitcli/git.go:registration identity`。
- 現在の回帰: `source/gitcli/missing_test.go:TestMissingWorktreeThroughParentAlias; TestMissingWorktreeRegistrationIsObservedAndRemoved`、coverage F。

### HM03 — M

Windows slash始まりpathが相対path前提を破った。root付き・脱出pathを明示拒否する。

- 検出/最早段階: S8/S2、見逃し分類: `NEGATIVE_FIXTURE_GAP`。
- 製品側: `paths/paths.go:source confinement`。
- 現在の回帰: `paths/paths_test.go:TestSourceConfinement`、coverage H。

### HM04 — M

未選択Compose resourceがcleanupへ入った。snapshotを絞り実destroyでもsibling・無関係volumeが残ることを証明する。

- 検出/最早段階: S8/S4、見逃し分類: `COMPOSITION_GAP`。
- 製品側: `runtime/compose/compose.go:selected snapshot`。
- 現在の回帰: `runtime/compose/compose_test.go:TestRenderSelectedClosureAndPolicy; cli/integration_test.go:TestIntegrationConcurrentLeasesClosureAndEvidence`、coverage F。

### HM05 — M

symlink・volume driver間接参照がhost mount policyを迂回した。effect許可前に間接参照を検査する。

- 検出/最早段階: S8/S2、見逃し分類: `NEGATIVE_FIXTURE_GAP`。
- 製品側: `policy/policy.go`。
- 現在の回帰: `policy/escape_test.go:TestBindSymlinkAndVolumeDriverCannotEscapePolicy`、coverage H。

### HM06 — M

初回readiness未完でもReconcileが昇格させ得た。近傍testは主に欠落resource・解放後残件を検査し、元の回帰との正確な対応は未確定。

- 検出/最早段階: S8/S3、見逃し分類: `INVARIANT_GAP`。
- 製品側: `app/lifecycle.go:Reconcile`。
- 現在の回帰: `app/lifecycle_test.go:TestLifecycleReconcileMissingAndReleasedLeftovers (nearby, not exact original mapping)`、coverage U。

### HM07 — M

secretリテラルは拒否が必要だが固定schema名をcredentialと誤認した。serialize済schema文字列でなく実data値と動的keyを検査する。

- 検出/最早段階: S8+S5/S2、見逃し分類: `ORACLE_COUPLING,NEGATIVE_FIXTURE_GAP`。
- 製品側: `app/plan.go; evidence/structured.go`。
- 現在の回帰: `app/lifecycle_test.go:TestLifecycleRejectsSecretSnapshots; app/manifest_credentials_test.go:TestInheritedCredentialDoesNotMatchManifestSchema; evidence/structured_test.go`、coverage F。

### HM08 — M

owner/default・manifest由来・exit・log scopeにCLI全体検証が必要だった。集約logをcomponent分離済としない。元記録は各thread IDなしで修正をまとめている。

- 検出/最早段階: S8/S4、見逃し分類: `HELPER_ONLY,NEGATIVE_FIXTURE_GAP`。
- 製品側: `cli/root.go; app/lifecycle.go; source/gitcli/origin.go`。
- 現在の回帰: `cli/diagnostics_test.go; cli/source_origin_test.go:TestPlanManifestOriginIndependentOfRuntimeRef; cli/logs_test.go:TestArchivedComponentLogsThroughCLI; TestLegacyAggregateCannotPretendComponentIsolation`、coverage F。

### HM09 — M

command完了後も子孫が書き込めた。returnやsource削除前に所有treeを停止する。

- 検出/最早段階: S3/S3、見逃し分類: `COMPOSITION_GAP`。
- 製品側: `execx/process_unix.go; process_windows.go`。
- 現在の回帰: `execx/process_tree_test.go:TestRunnerReapsOrdinaryDescendants; app/cancellation_test.go:TestDestroyCancelsActualCommandTreeBeforeSourceCleanup`、coverage N。

### HM10 — M

P1: artifact成功前にrunを終端化しforce destroyが証拠を削除できた。finalize失敗時はrunning barrierを維持する。

- 検出/最早段階: S8/S4、見逃し分類: `FAILURE_INJECTION_GAP`。
- 製品側: `app/commands.go:finalization`。
- 現在の回帰: `app/cancellation_test.go:TestDestroyRetainsSourceWhenCancellationFinalizationFails`、coverage F。

### HM11 — M

P1: cancel応答がtree停止不明やoutput欠落を隠した。型付き不明証拠で終端化とcleanupを止める。

- 検出/最早段階: S8/S4、見逃し分類: `FAILURE_INJECTION_GAP`。
- 製品側: `app/commands.go; execx/runner.go`。
- 現在の回帰: `app/cancellation_test.go:TestDestroyRetainsSourceWhenCancellationFinalizationFails; execx/runner_test.go:TestOutputWriteFailureIsIncompleteEvidence`、coverage F。

### HM12 — M

並行SQLite初期化がBUSYを返した。初期化競合のみ制限付きretryし通常操作・不変条件違反はretryしない。

- 検出/最早段階: S4/S3、見逃し分類: `CONCURRENCY_GAP`。
- 製品側: `store/sqlite/store.go:initializeWithRetry`。
- 現在の回帰: `store/sqlite/initialization_test.go:TestConcurrentColdOpen; TestConcurrentColdOpenProcesses; TestInitializationRetriesOnlyBoundedBusyContention; TestInitializationDoesNotRetryInvariantFailure`、coverage F。

### HM13 — M

native filepath.Rel区切りがportable artifact APIへ入った。境界で変換しnativeで保存artifactを検証する。

- 検出/最早段階: S5/S2、見逃し分類: `NATIVE_EVIDENCE_GAP`。
- 製品側: `app/commands.go:artifact relative paths`。
- 現在の回帰: `app/commands_test.go:TestNamedCommandPersistsRedactedEvidence; app/state_root_test.go:TestNamedCommandEvidenceStaysUnderStateRoot`、coverage F。

### HM14 — M

成功testの固定sleepが保守的renewal watchdogを越えた。sleepを所有証明とせず別connectionからdurable更新を観測する。

- 検出/最早段階: S5/S3、見逃し分類: `ORACLE_COUPLING,CONCURRENCY_GAP`。
- 製品側: `store/sqlite/store_test.go:renewal fixture`。
- 現在の回帰: `store/sqlite/store_test.go:TestOperationLockAcrossConnectionsAndExpiry`、coverage F。

### HM15 — M

macOSでWait後の一時zombie groupがEPERMを返した。Darwin EPERMのみ制限付きretryし成功/ESRCH必須と継続失敗時barrierを維持する。

- 検出/最早段階: S5/S5、見逃し分類: `NATIVE_EVIDENCE_GAP,CONCURRENCY_GAP`。
- 製品側: `execx/process_unix.go:terminateProcessGroup`。
- 現在の回帰: `execx/process_unix_test.go:TestTerminateProcessGroupWaitsForZombieReaping; TestTerminateProcessGroupPreservesFailures; app/cancellation_test.go`、coverage N。

## 各分類行に共通する見逃し分析

各行は以下の具体的な検出機会と説明を併せて持つ。「test不足」だけを理由としない。
参照元に失敗fixtureの記録がある場合を除き、理由は事後分析上の仮説である。

| 分類 | 発見前の機会と見逃し理由 | より早い予防策と状態 |
| --- | --- | --- |
| BOUNDARY_GAP | 数値上限や時間契約は存在したが、通常・overflow fixtureはちょうど上限、metadata合計、setup時間、変換後byteを省いた。 | 各行の境界回帰は既存。共通0/1/limit-1/limit/limit+1と最終encoding表はPhase Aの残作業で、局所修正を全体guardrailとは呼ばない。 |
| NEGATIVE_FIXTURE_GAP | 公開validation・所有契約から独立した不正入力を作れたが、正常fixtureで各要件を個別変異していなかった。 | 独立不正入力fixtureは既存。H/Uが残る入口に拡張する。期待検出はS2またはS3。 |
| ORACLE_COUPLING | mockや期待値がURL形状、全選択、広いUnicode照合、sleep所有証明、早期拒否の成功扱いなど実装と同じ前提を共有した。 | native protocol fixture、注入点到達・artifact存在assertは既存。負testは対象分岐への到達と近傍正常例を証明する。interfaceに応じS2–S5。 |
| COMPOSITION_GAP | 一層の成功では終了後readiness、alias優先、effect順、redaction後上限を保証できなかった。interface自体はレビュー前に存在した。 | app・store・effect全体fixtureと保存artifact検査は既存。共通effect段階・最終encoding表は処置判断待ちでhelperのみでは不足。S3/S4を期待。 |
| FAILURE_INJECTION_GAP | intent/effect/result保存段階は説明されていたが、store・receipt・fence喪失・入力後・output finalizeの各失敗を注入していなかった。 | 各行の失敗注入とdurable running barrier assertは既存。runtime横断の段階表は監査残作業。S3/S4を期待。 |
| NATIVE_EVIDENCE_GAP | cross-buildやfakeではJob完了/PID再利用、Darwin zombie、Chrome realm/accelerator、sandbox ACL、native区切りを再現できない。 | Windows/macOS/Linux CIと実process/browser fixtureは既存。未実行platformを成功扱いにしない。OS意味論が本質ならS5。 |
| CONCURRENCY_GAP | 単一connection・逐次processでは初期化、queue終了、renewal、identity観測の競合を隠した。 | 独立SQLite connection/process、決定的mutation点、native identity負例は既存。runtime横断の再検証は残る。S3–S5を期待。 |
| HELPER_ONLY | 正しい照会・validation helperが後続format・保存・routingも保証すると考え、公開呼出をoracleにしなかった。 | HP10はDestroy経由へ修正済。HP05/HB19の特定負例はhelperのみ。coverage作業を採用する場合は公開呼出と注入assertを追加する。 |
| INVARIANT_GAP | 「targetが見えない」「resourceが動く」に完全な不在や初回readinessの積極的証拠がなかった。 | gone/truncated回帰は既存。HM06を追跡してから完了を主張する。積極的証拠の判断表はPhase A残作業。 |
| REVIEW_CHECKLIST_GAP | 文書とCIは存在したが古い説明・旧revisionの成功を最終受入と突き合わせていなかった。 | 日英/hash検査は意味を証明しない。最終revision受入確認は手動。証拠modelなしのCI-to-Plan自動真偽検査は現実的でなく、そのguardがあるとは主張しない。 |

## 現在のcoverage限界と同型問題の引継ぎ

以下は確認したcoverage限界または調査候補であり、新たな製品不具合の確定ではない。

1. HP05は`validateProcessSecrets`を直接呼ぶ。現在`BuildPlan`はdigest/source解決前に
   これを呼び、近傍`TestPlanProcessPinsSourceWithoutRuntimeEffects`もruntime環境リテラルで
   全planningを通るが、readinessリテラルの同じ注入ではない。公開plan/createの負例なら
   将来の呼出迂回を検出できる。
2. HB19は`selectBrowser`を直接呼ぶ。現在`Browser`はprovider照会/run intent前に呼ぶが、
   回帰は保存manifestを実app入口から改変してprovider呼出ゼロを確認していない。
3. HB18は`finishCapture`と購読状態を直接testする。現在のcapture loopは期限終了をそこへ
   委譲するが、helper回帰だけでは将来の別return経路を検出できない。enable期限と購読overflowの
   protocol testは隣接挙動を確認するが、この競合そのものではない。
4. HM06の初回readinessに関する元の回帰testは今回の範囲では特定できなかった。
   `TestLifecycleReconcileMissingAndReleasedLeftovers`をsetup確認なしに代替証拠としない。
   この対応付けを完了扱いにしない。
5. 範囲を絞った検索で得た再発候補: `app/commands.go`・`app/browser.go`・`app/ui.go`の
   cancel/evidence barrier、browserとAndroid UIの不完全snapshot/gone、browser・process log・
   共通evidenceのredaction後上限、process/Android destroy preview、source/Android/processの
   canonical path、独立resource cleanupと混在readiness。これらは検索先であり現在の不具合や
   広範レビュー完了の主張ではない。
6. HB25は6通り全てで所有検証mutation callbackへの到達をassertするようになった。
   HB16/HB20は登録artifactの存在とbyteを検査し、HB20は保存passed run状態とtruncated入力拒否も
   確認する。以前の空振り成功を防ぐ具体的guardである。

全行のguardrail状態は既存の局所回帰・process統制で、広い予防策は監査の処置判断まで
明示的に保留する。本資料は新しいtest guardrailを追加しない。
ここで現在のfindingをACCEPT/REJECT/DEFERとはしていない。

## 検証証拠と限界

固定対象で選択した過去回帰を`go test -race`・`-count=1`で実行:
browser/cdp PASS 7.293s、runtime/process PASS 1.092s、
store/sqlite PASS 10.883s、app PASS 11.129s、execx PASS 1.115s。
選択範囲はbrowser、capture、frame/DOM/gone/stale/focus/transport、
process readiness/未生成/preview/alias/readiness-secret/予約guard、
receipt/redaction、cancel finalize、初期化競合、process group停止。
上記全testを選択したわけではなく、特にbuild tag付きWindows/native CLI/Dockerは
source確認のみで今回の再実行に含まれない。過去PlanのWindows/macOS証拠は履歴であり
今回のnative監査証拠ではない。全baseline・native/integration再実行は統合担当が管理する。
元不具合のmutation replayは行っていない。
