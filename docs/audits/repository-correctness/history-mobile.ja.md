---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/history-mobile.md
source_sha256: 5dbe7526c351d7dd7852741b6c32e25ba5c8d19dba2f025b2980cdfc2a058dc4
---

# Mobile領域の過去correctnessレビュー資料

[English](history-mobile.md) · [監査index](index.ja.md) · [実行基準](../../exec-plans/active/repository-correctness-audit.ja.md)

固定対象: `031869c8b9073b8e23bc17fbc55243666a52f557`。Phase Aとして製品・test fileは変更しない。実装時レビューとnative fixtureの判定基準不備も含め、記録された重要な指摘を整理する。前提環境の不足、一時的なformat/hash検査、原因未確認のscheduling問題は製品不具合と数えず別記する。

## 出典と読み方

- E: [Android feature](../../exec-plans/completed/android-emulator-lease.md)の想定外の発見。個別Thread IDは未記録。
- R1: [Android review](../../exec-plans/completed/android-emulator-review.md)のPR2指摘4件。comment ID: `3951260551`、`3951260553`、`3951260556`、`3951260560`。
- R2: [Android review2](../../exec-plans/completed/android-emulator-review-2.md)のPR2指摘7件と、独立して見つかったCLI preflight・Linux procfs不具合。comment ID: `3952452797`、`3952452807`、`3952452813`、`3952452817`、`3952452823`、`3952452826`、`3952452828`。
- F: [Flutter feature](../../exec-plans/completed/flutter-android-runtime.ja.md)の独立レビュー・実providerでの発見。
- FR: [Flutter review](../../exec-plans/completed/flutter-android-review.ja.md)のPR4指摘8件。個別Thread IDは未記録。
- U: [Android UI feature](../../exec-plans/completed/android-ui-observer.ja.md)の実装レビュー・native fixtureでの発見。Planにない後続PR5指摘はGitHub comments APIをpaginationして追加取得し、下記PR5節に整理した。

PR2の出典はcomment IDを指摘別に対応付けていないため、本報告でその対応を推測しない。Planのみの出典では当初の重要度ラベルは未記録。PR5外部commentのP1/P2は下表にそのまま保存する。各行には当時の影響を書き、reviewerが示していない重要度を遡及して付けない。fixtureの不変条件も含め、全行は現在も適用される（yes）。過去の不具合を自動的に新しい監査指摘とは扱わない。

`最早/検出/profile`はExecPlanのS0–S9を使う。検出段階は出典に従い、native CIはS5、独立レビューはS8とする。最早予防段階とescape profileは監査による分類であり、記録にないreviewerの考えを断定しない。E/M05はnative CI前にも実launcher-child fixtureを作れたため最早S3とし、OS固有Job挙動はS5とする。

回帰test名は現在その修正を検証するものを示す（過去Planがカテゴリしか記録していない場合もある）。下記のtest有効性はassertionのソース確認と今回のLinux実行であり、**元不具合を再導入するmutation replayの証明ではない**。全件を再導入して検出できると述べるには別途mutation検証が必要である。

| Code | 現testは不変条件を証明するか | 公開・全体entry pointか |
| --- | --- | --- |
| A | yes: app境界でprovider注入と永続状態のassertion | yes: Service use case。ただし実SDKは使わない |
| C | yes: CLI解析・出力境界 | yes: CLI commandまたは明示的host diagnostic entry |
| F | yes: factory順序 | no: factory+ServiceでありCobra dispatchではない |
| D | yes: concrete adapter・native primitive境界 | no: その回帰単独ではapp/CLI結合を検証しない |
| W | yes: ソース確認。Windows専用実行は今回localで再実行していない | no: native primitiveでありapp全体ではない |
| N | yes: native testソースと過去の実行記録。今回担当内では未再実行 | yes: 実provider entry経路 |
| S | 両側それぞれyes。結合は過去native fixtureで検証 | no: この2 test単独では相互動作を証明できない |
| G | no: 当該producer省略を直接検証する回帰を特定できなかった | no: 記載testはconsumer・隣接挙動のみ |
| H | yes: 障害注入fixture | no: test基盤であり製品entryではない |
| DOC | 意味の自動証明はno。docs-checkは構造・hashを検証 | no: 日英の意味レビューも必要 |

guardrail statusは上記の局所回帰についてexistingとする。profile表の広い予防策は監査のdispositionまでdeferred。G行は直接guardrailが不足している。Phase Aで製品の予防controlは追加していない。

## Escape profileと早期検出機会

各行はprofileの検出機会、早期段階で逃した理由、不足guardrail、予防策、再発探索を引き継ぐ。最早段階より前に必要な知識があったとは出典から確認できず、その段階の見落としと断定しない。S7で不足したnegative・結合・native fixtureを要求する機会はあったが、その判定基準がないgreen suiteから直接検出はできなかった。個人のレビュー失敗は推測しない。S8で発見した場合、それは機能した検出層である。

| Profile | 検出機会と逃した理由 | 不足guardrail・予防策・将来の検出段階 | 確認済みまたは次の同型探索候補 |
| --- | --- | --- | --- |
| I | S2/S3で独立した不正入力・本人性変更を試せたが、positive fixtureは正常値のみだった。`NEGATIVE_FIXTURE_GAP`、場合により`ORACLE_COUPLING`。 | 要件別identity/path/environment負例と操作なしassertion、S2/S3。 | Android marker/ADB、application reverse、process receipt、browser target。 |
| C | S3/S4時点で公開entryは存在したがhelper・単一provider・JSON専用testでは次の依存や任意分岐を省略した。`HELPER_ONLY`、`COMPOSITION_GAP`。 | CLI/factory/app/adapterと任意機能有無の行列、操作なしの観測、S3/S4。 | Inventory/Doctorのfrontend依存、runtime別preview、CLI store factory、Raw/Binary証拠。 |
| P | store/provider注入でfailure/cancel/restart窓を表現できたが、成功・初回失敗testだけで後続復旧を見なかった。`FAILURE_INJECTION_GAP`、`COMPOSITION_GAP`、場合により`INVARIANT_GAP`。 | durable intent/effect/finalization行列とstore回復後Destroy/Reconcile、callback実行assertion、S2/S4。 | build guard、UI recovery、process receipt、browser mutation barrier。 |
| N | 実SDK・OS lifecycleはfake identity/tool出力に表れず、cross-buildでも判定できなかった。`NATIVE_EVIDENCE_GAP`、`CONCURRENCY_GAP`。 | native lifecycle、親終了・子生存、同時2 leaseと兄弟cleanup、S5（M05は実process fixtureのS3）。 | Windows Job完了、Linux procfs race、Android helper専用discovery、共有ADB containment。 |
| T | lock取得・可視device状態についてfixtureの仮定を確かめず、意図した注入・操作前に失敗し得た。`ORACLE_COUPLING`、`FAILURE_INJECTION_GAP`。 | 注入行数・error、操作前可視状態をassertし、native viewportからgestureを算出、S3/S5。 | lock-loss注入、browser mutation callback、UI Back/swipe。 |
| B | producer走査省略を独立検証せず、consumerは合成済みcomplete/truncated値を見ていた。`BOUNDARY_GAP`、`HELPER_ONLY`。 | producer limit−1/limit/limit+1と欠落child fixture、実際の省略を観測、S2。 | Android UI helper走査、Android logcat、browser AX/DOM上限。 |
| D | 文書構造・hash検査だけではroadmapの意味と実装の一致を判定できない。`REVIEW_CHECKLIST_GAP`。 | 実装とcontract/README/roadmapの最終意味レビュー、S7。hash一致を意味の証明としない。 | feature status、CLI table、任意前提。 |

## 過去指摘と現時点の検証

repository prefixなしのpathは`internal/`配下。production symbolは現在のentry・責務境界を示す。helper testを全体結合回帰とは扱わない。

| ID / 出典 | 元の不変条件・不具合・影響 | 最早/検出/profile | 現回帰test位置 | 現production位置 | 検証範囲 |
| --- | --- | --- | --- | --- | --- |
| M01 / E | console本人性確認後のcancelでもkill・削除を止める。確認後のcancelで操作が続き得た。 | S2/S8/P | `runtime/android/android_test.go:TestCancellationDuringIdentityHandshakeCannotKill` | `runtime/android/lifecycle.go:Destroy` | D |
| M02 / E | ADBは所有するローカルserverを使う。同じserialのremote deviceを継承設定で観測し得た。 | S2/S8/I | `runtime/android/android_test.go:TestReadinessPinsLocalADBServer` | `runtime/android/adb.go:adbRoutingEnvironment` | D |
| M03 / E | adapter結果にはappが要求するlease所有metadataが必要。adapter単体成功では接続不整合を見逃した。 | S4/S8/C | `runtime/android/app_test.go:TestRealAdapterThroughAppPersistsOwnedResource` | `runtime/android/lifecycle.go:Create` | A |
| M04 / E | boot期限はappが管理する。adapter独自waitがその期限を迂回した。 | S4/S8/C | `runtime/android/android_test.go:TestBootingLaunchReturnsForAppReadinessAndCompensation` | `runtime/android/lifecycle.go:Create` | D |
| M05 / E | root消失だけで削除を許可しない。launcher終了後もQEMUが残り得る。 | S3/S8/N | `execx/detached_test.go:TestDetachedSurvivesLaunchingCLI` | `execx/detached.go:Observe` | D |
| M06 / E | 対応SDKのdataディレクトリseedを受け入れる。userdata.img必須という仮定で新形式を拒否した。 | S5/S5/N | `runtime/android/android_test.go:TestModernSystemImageSeedsPrivateDataWithoutFabricatedImage` | `runtime/android/discovery.go:Validate` | D |
| M07 / E | stopped markerでも再出現process・占有portの削除は許可しない。 | S2/S8/I | `runtime/android/android_test.go:TestStoppedMarkerNeverAuthorizesCleanupOfReappearedResource` | `runtime/android/lifecycle.go:Inspect/Destroy` | D |
| M08 / E | 別logon sessionのWindows Jobは不明であり、不在ではない。 | S5/S8/N | `execx/detached_windows_test.go:TestDetachedObservationRejectsDifferentWindowsSession` | `execx/detached_windows.go:observeDetachedJob` | W |
| M09 / E | 起動CLI終了後もWindows Job名を維持する。最後のhandle closeで生存processの名前付き本人性を失った。 | S5/S5/N | `execx/detached_test.go:TestDetachedSurvivesLaunchingCLI` | `execx/detached_windows.go:startDetached` | D |
| M10 / E | ローカルADB指定と独立startupを両立する。数値-Hは自動起動を無効にした。 | S5/S5/N | `runtime/android/adb_test.go:TestSharedADBStartsOutsideEmulatorContainment` | `runtime/android/adb.go:ensureADBServer` | D |
| M11 / E | 認証kill後もtree・port不在を待つ。10秒ではEmulator37のgraceful stopより短かった。 | S5/S5/N | `runtime/android/android_test.go:TestDestroyWaitsForPostKillTreeConfirmation` | `runtime/android/lifecycle.go:Destroy` | D |
| M12 / E | Job欠落時の証明でも2回目のidentity読取でPID再利用を拒否する。 | S2/S8/I | `execx/detached_windows_test.go:TestDetachedMissingGuardianRequiresDurableEmptyEvidence` | `execx/detached_windows.go:observeDetachedJob` | G |
| M13 / E | 共有ADB daemonをEmulator containmentへ入れない。入るとcleanupが永続的に生存扱いとなり得る。 | S5/S8/N | `runtime/android/adb_test.go:TestSharedADBStartsOutsideEmulatorContainment` | `runtime/android/adb.go:ensureADBServer` | D |
| M14 / E | SDK clientが非互換共有ADBを置換しない。操作前にprotocolを確認する。 | S3/S8/I | `runtime/android/adb_test.go:TestIncompatibleOrMalformedSharedServerNeverInvokesOperationalADB` | `runtime/android/adb.go:compatibleADB` | D |
| M15 / E | 空Jobには証拠公開の完了も必要。guardianの遅いwriteとディレクトリ削除が競合した。 | S5/S5/N | `execx/detached_windows_test.go:TestDetachedEmptyJobWaitsForPublishedCompletion` | `execx/detached_windows.go:observeDetachedJob` | W |
| M16 / R1 | registryがAndroid行のみでもglobal inventoryはCompose孤立resourceを表示する。 | S4/S8/C | `app/reconciliation_inventory_test.go:TestInventoryFindsComposeOrphansWithOnlyAndroidLeases` | `app/reconciliation_inventory.go:Inventory` | A |
| M17 / R1 | Android destroy previewはCompose専用diagnosticや操作なしでprivate AVD cleanupを説明する。 | S3/S8/C | `app/android_test.go:TestAndroidDestroyPreviewReportsPrivateCleanupWithoutEffects` | `app/destroy_preview.go:previewDestroy` | A |
| M18 / R1 | 各runtimeは未完readinessの期限を持つ。完了済みの短いCompose期限でAndroidを短縮せず、期限切れComposeも遅延させない。 | S4/S8/C | `app/mixed_readiness_test.go:TestMixedReadinessSatisfiedComposeDoesNotShortenAndroidBoot;TestMixedReadinessPendingComposeBoundsEarlierAndroidInspection` | `app/lifecycle.go:waitReady` | A |
| M19 / R1 | 実行可能SDK tool・acceleration不備はallocation前に拒否する。Doctorの検査がValidateになかった。 | S4/S8/C | `runtime/android/app_test.go:TestSDKPrerequisitesFailBeforeAppAllocation` | `runtime/android/discovery.go:sdkPrerequisites/Validate` | A |
| M20 / R2 | repository doctorはSDK一般状態だけでなく選択AVDすべてを検査する。 | S3/S8/C | `cli/android_test.go:TestAndroidRepositoryDoctorValidatesEverySelectedTemplate` | `cli/lifecycle.go:doctorManifest` | C |
| M21 / R2 | Android logはrelease後も読取・秘匿化でき、componentを混同しない。 | S4/S8/C | `cli/logs_test.go:TestAndroidProcessLogsActiveReleasedAndComponentIsolation` | `cli/lifecycle.go:runtimeLogEntries` | C |
| M22 / R2 | image ABIは対応hostに適合し、不正architectureを予約前に拒否する。 | S2/S8/I | `runtime/android/app_test.go:TestImageArchitectureMatrix;TestArchitectureAndADBFailBeforeAppAllocation` | `runtime/android/discovery.go:validateImageArchitecture` | A |
| M23 / R2 | 共有ADB互換性を予約前のpreflightで検査する。 | S4/S8/C | `runtime/android/app_test.go:TestArchitectureAndADBFailBeforeAppAllocation` | `runtime/android/discovery.go:Validate` | A |
| M24 / R2 | state・outputはSDK/template/imageの不変入力と重ならない。alias・他runtime入力も含む。 | S2/S8/I | `app/android_paths_test.go:TestAndroidInputOverlapResolvesAliasesAndAncestors;TestAndroidPathsCompareOtherRuntimeInputs` | `app/android_paths.go` | A |
| M25 / R2 | portable runtime名は大文字小文字を同一視する衝突をallocation前に拒否する。 | S2/S8/I | `app/android_paths_test.go:TestAndroidCaseCollisionFailsBeforeAllocation` | `config/manifest.go` | A |
| M26 / R2 | 局所失敗後も独立cleanupを継続し、source・予約を保持する。全体cancel・fence喪失では操作を止める。 | S4/S8/P | `app/cleanup_siblings_test.go:TestCleanupContinuesIndependentRuntimesAndRetainsSourcesUntilRetry;TestCleanupContinuesAfterRuntimeLocalTimeout` | `app/lifecycle.go:cleanup` | A |
| M27 / R2 | CLIはapp入力検査前にregistryを作らない。app単体の不変性testではfactoryのwriteを見逃した。 | S4/S8/C | `cli/create_store_test.go:TestCreateServiceRejectsInputOverlapBeforeOpeningRegistry` | `cli/create_store.go:openCreateService` | F |
| M28 / R2 | process終了競合でprocfs readはESRCHを返す。消失したcensus項目はgroup不在の証明にならない。 | S5/S5/N | `execx/detached_linux_test.go:TestDetachedProcStatReadAfterExitReportsVanishedEntry;TestDetachedProcDisappearanceDoesNotHideInspectionFailures` | `execx/detached_linux.go:detachedProcEntryGone` | D |
| M29 / F | build終了未確認は初回quarantineだけでなく後のforce cleanupも止める。 | S4/S8/P | `app/applications_test.go:TestMobileUnconfirmedBuildBlocksLaterForcedCleanup` | `app/applications.go:applicationCleanupBarrier` | A |
| M30 / F | reverse要求だけでは所有確認にならない。不明な既存mappingを残しdevice cleanupを止める。 | S3/S8/I | `app/applications_test.go:TestMobileUnconfirmedReversePreservesMapping` | `app/applications.go:cleanupApplications` | A |
| M31 / F | launch意図はREADYではない。reconcileにはlaunch確認と実行file・directory本人性が必要。 | S3/S8/P | `app/application_identity_test.go:TestMobileReconcileRejectsIncompleteApplicationSnapshot` | `app/applications.go:applicationConsistent` | A |
| M32 / F | build終了と証拠永続化は別。artifact保存失敗後はstore回復後もsource・APKを残す。 | S4/S8/P | `app/application_identity_test.go:TestMobileIncompleteBuildEvidenceBlocksCleanup` | `app/applications.go:buildApplications/applicationCleanupBarrier` | A |
| M33 / F | SDK helper discoveryはlease専用にする。2 Emulatorでnetsim discoveryを共有し兄弟cleanupが不明となった。 | S5/S5/N | `runtime/android/android_test.go:TestEmulatorEnvironmentUsesPrivateNativeDiscoveryPaths;cli/flutter_integration_test.go:TestRealFlutterAndroidBackendLease` | `runtime/android/lifecycle.go` | N |
| M34 / F | lock喪失fixtureは実際のtoken置換とwrite成功を確認する。raw SQLite writer競合で注入自体が失敗し得た。 | S3/S5/T | `app/lockloss_test.go:TestLockLossFixtureWaitsForSQLiteWriter` | `app/lockloss_test.go:replaceOwner` | H |
| M35 / FR | 選択APK出力は操作前に衝突拒否する。build-firstでは前appの出力を上書きする。 | S2/S8/I | `app/flutter_review_test.go:TestSelectedApplicationAPKOutputsCannotCollide` | `app/plan.go:BuildPlan` | A |
| M36 / FR | 公開文書をFlutter実装と合わせる。古いroadmap・manifest・CLI説明は利用者を誤らせる。 | S7/S8/D | `tools/repoctl:docs-check plus paired meaning review` | `README.md and docs/product-specs` | DOC |
| M37 / FR | host Flutter doctorもAndroidを要求し、install・allocationは行わない。 | S3/S8/C | `cli/flutter_test.go:TestFlutterHostDoctorRequiresBothToolchains;TestFlutterHostDoctorMissingPrerequisitesIsPure` | `cli/flutter.go:doctorFlutterHost` | C |
| M38 / FR | 既定plan tableは選択app要件を表示する。JSON専用testで既定出力を見逃した。 | S3/S8/C | `cli/flutter_test.go:TestFlutterPlanTableIncludesSelectedApplicationRequirements` | `cli/lifecycle.go:addLifecycle` | C |
| M39 / FR | reverseなしではreverse・endpointを問い合わせない。任意機能なしでも利用できる。 | S3/S8/C | `app/flutter_review_test.go:TestApplicationWithoutReverseSkipsNetworkObservation` | `app/applications.go:observeApplication` | A |
| M40 / FR | project・祖先symlinkは参照先がtree内でもbuild前後に拒否する。 | S2/S8/I | `runtime/flutter/flutter_test.go:TestBuildRejectsInternalProjectSymlinks` | `runtime/flutter/flutter.go:Validate/Build` | D |
| M41 / FR | dry-runもforceを含め2種のbuild barrierを尊重し、危険な削除を可能と表示しない。 | S3/S8/C | `app/flutter_review_test.go:TestDestroyPreviewHonorsApplicationBuildBarriers` | `app/destroy_preview.go:previewDestroy` | A |
| M42 / FR | ADB非zero終了でも上限付き秘匿diagnosticとerror本人性を保持する。 | S2/S8/I | `runtime/android/application_test.go:TestApplicationExecutionFailureRetainsDiagnosticsAndErrorIdentity` | `runtime/android/application.go` | D |
| M43 / U | instrumentation再接続でwindow IDが変わるため、semantic本人性には安定したwindow metadataを使う。 | S5/S5/N | `cli/ui_integration_test.go:TestRealAndroidUIObserver` | `runtime/android/uihelper/source/Observer.java:tree` | N |
| M44 / U | recoveryには永続化済みの明確な資格が必要。最終分類欠落でhost・証拠barrierを解除しない。 | S4/S8/P | `app/ui_recovery_test.go:TestUIRecoverRefusesUnpersistedFailureClassification` | `app/ui_recovery.go:RecoverUI` | A |
| M45 / U | CLI UI終了codeは前提・入力・registryを区別する。raw errorがすべて入力exit2となっていた。 | S4/S8/C | `cli/ui_test.go:TestUIExitClassificationPreservesCauses;TestUIRegistryOpenFailureReturnsObservationExit;TestUICorruptRegistryReturnsObservationExit` | `cli/ui.go` | C |
| M46 / U | 実logcat証拠をadapter/app境界で渡す。RawとBinaryの不一致でlogを失った。 | S4/S8/C | `runtime/android/ui_test.go:TestUILogPIDAttributionAndBounds;app/ui_test.go:TestUILogcatReturnsBoundedRedactedInlineEvidence` | `runtime/android/ui.go:ObserveUI;app/ui.go:UI` | S |
| M47 / U | backendのstale・ambiguous・unavailable状態を安定diagnosticとして保つ。 | S3/S8/C | `app/ui_test.go:TestUIBackendRefusalsRetainStableDiagnostics` | `app/ui.go:UI` | A |
| M48 / U | mutation helperは操作後fingerprintを返す。初期実装では欠落した。 | S3/S8/C | `cli/ui_integration_test.go:TestRealAndroidUIObserver (no explicit fingerprint assertion found)` | `runtime/android/uihelper/source/Observer.java:onStart` | G |
| M49 / U | 不完全なaccessibility走査を完全な証拠と表示しない。 | S2/S8/B | `app/ui_test.go:TestUIRejectsSnapshotTamperingAndAmbiguity (consumer only)` | `runtime/android/uihelper/source/Observer.java:node/tree` | G |
| M50 / U | ACTION_SET_TEXT成功は読戻し成功ではない。focus変化でordinalが無効化し別nodeへ操作し得る。 | S5/S5/N | `cli/ui_integration_test.go:TestRealAndroidUIObserver` | `runtime/android/uihelper/source/Observer.java:onStart` | N |
| M51 / U | ADB timeoutはremote helper停止の証明ではない。recoveryは再実行せずhelper限定の静止を証明する。 | S1/S8/P | `app/ui_test.go:TestUIHelperRecoveryAfterInternalTimeout;app/ui_recovery_test.go:TestUIRecoverFailureRetainsBarrier` | `app/ui.go:UI;app/ui_recovery.go:RecoverUI` | A |
| M52 / U | Back fixtureは対象を確定する。IMEを閉じるという仮定でappから退出した。 | S5/S5/T | `cli/ui_integration_test.go:TestRealAndroidUIObserver` | `cli/ui_integration_test.go (fixture only)` | N |
| M53 / U | swipe fixtureはIME resize後の実viewportを使う。固定screenshot座標でScrollView外を操作した。 | S5/S5/T | `cli/ui_integration_test.go:TestRealAndroidUIObserver` | `cli/ui_integration_test.go (fixture only)` | N |

## 現在の証拠と未解決の検証不足

固定対象のLinux/Go 1.27.1で過去回帰の対象選択を`-race -count=1`実行し、Android 2.461秒、Flutter 1.037秒、execx 1.300秒、app 19.000秒、CLI 2.313秒で成功した。Android/SDK/ADB/Detached/MixedReadiness/Cleanup/CreateService/Flutter/Mobile/UIの記載回帰と、APK衝突・reverse・preview・symlink・障害注入testを選択した。UI helper subpackageはこのregexで**実行testなし**だったため、test実行の証拠に数えない。Windows専用・実SDK/UI testはソース確認のみで、今回担当内では未実行。baseline担当が別途実行する。既存ソースは変更していない。

M12も2回目PID再読取の強制raceを直接検証するtestは特定できなかった。記載Windows testは不在PIDでのproof検証であり、一般PID再利用testもその特定race窓を強制しない。

M48/M49は検証不足であり、現在のproducer不具合を確定したものではない。Java helperは`after_fingerprint`を計算し、アクセス不能・欠落childでtruncatedを設定する。既存Go helper testはprovenance/build入力を検証し、Java走査・操作algorithmは検証しない。`TestRealAndroidUIObserver`は通常treeの完全性と実効果をassertするが、操作後fingerprintを明示assertせず、アクセス不能child・producer上限境界も強制しない。`TestUIRejectsSnapshotTamperingAndAmbiguity`は渡されたtruncated値をappが拒否する証明であり、helperがその値を出力する証明ではない。次の候補は、producer fixture（または制御したAndroid instrumentation fixture）で返却fieldと省略条件を明示検証すること。statusは検証不足のdisposition待ちで、この2件に現在のcorrectness指摘IDは付けていない。

### AUDIT-BOUNDARY-001 — Android log tailが上限ちょうどでtruncatedになる

- 重要度: 監査のexact-limit定義によりMedium。Disposition: 未分類。Phase Bで採用・不採用を決めていない。
- 不変条件: `truncated=true`には実際の証拠省略が必要。上限ちょうどの完全な応答だけでは省略の証明にならない。
- 位置: `internal/runtime/android/ui.go:357`の`boundedUILog`。`Adapter.ObserveUI`のlogcatが251行で呼ぶ。243行のdevice要求は`logcat -t 2000`。
- 再現条件: 既存`applicationFixture`でPID応答を`1234`、device時刻を`1000`とし、logcat command応答を`999.000 1234 1234 I Tag: current\n`の2000回繰り返しにする。package `com.example.app`、`SinceSeconds:30`で`ObserveUI`を呼ぶ。1999回を対照とする。
- 観測結果: 一時Go overlay回帰は2000件で失敗し、66,000 bytesをすべて返しながら`Truncated=true`だった。1999件は完全と返した。test packageは0.014秒で失敗。byte省略・local保持行の省略はない。runnerを注入してconcrete adapterを呼んだ結果であり、実deviceの証拠ではない。
- 期待・影響: 完全な上限ちょうどのcaptureは完全と示すべきだが、省略と誤表示する。device側に2000件超のlogがある場合はtailが古い行を省略し得るが、commandはoverflow bitを返さない。helperの`raw行数>=2000`は、上流省略の可能性と実証済み省略を混同している。比較だけを消すと実overflowを完全とするため、上流の追加1件取得・証明方式はdisposition/remediationで検討する。
- 既存検証: `TestUILogPIDAttributionAndBounds`は2行と、2001行かつbyte上限超の応答を検証する。短い完全応答1999/2000/2001行を個別に検証しない。consumer testも渡されたflagに依存する。
- 回帰・解決: 一時overlayのみ。製品・test変更、修正は未実施。過去との関係はM49のproducer完全性不足とBrowser exact-limit分類。同型探索候補はJava `node/tree`上限、UI text/log artifact上限、browser producer。他の不具合があるとは断定しない。
- Escape分析: 検出S9、現実的な最早検出S2。`BOUNDARY_GAP`と`ORACLE_COUPLING`。既存testは件数超過とbyte超過を同時に作り、上限ちょうどを独立検証しない。S3/S4 fixtureは短いlog、過去S5 UI native fixtureは2000件ちょうどを生成しない。S6はそのtestを実行するが意味的上限validatorを持たない。S7/S8では独立したexact-limit判定基準を用意する機会があったが、過去Planにその記録はない。予防策候補は共通境界ケース規約とprovider overflow証明、将来検出S2/S3。dispositionまでdeferred。

## 過去の制約を維持する

Android review Planはrootless live groupでの2 Emulator cleanup失敗を記録し、readinessだけで全体成功とはしていない。後のFlutter専用netsim discoveryと2 lease成功により受け入れ不足は解消したが、logだけでは過去の全shutdown失敗が同じ原因だったと証明できない。過去原因を遡及して断定しない。

SDK不足、offline package取得、一時disk空き不足、無関係fixtureの初期50ms readinessは環境・test前提であり、本資料では独立した製品修正と数えない。SQLite注入fixtureの不具合は再現されたが、過去CI失敗との厳密な因果は仮説のままである。実Windows/macOS SDK実行とnative Go testは区別する。UI feature Planに欠けていたPR5指摘は外部APIで補完済み。

今回の過去レビューで監査control追加・製品修正・commit・remote writeは行っていない。追加の境界・状態・本人性レビューは[現在のmobile監査](current-mobile.ja.md)に記録し、`AUDIT-REDACTION-001`を含む。最終dispositionは監査担当が決める。

## 追加取得したPR5外部レビュー

`repos/mahcialet/agent-env/pulls/5/comments`をpaginationして取得し、reviewerの実質指摘24件と返信を区別した。M54–M77は全指摘をcomment ID・当初の重要度付きで記録する。各IDは`https://github.com/mahcialet/agent-env/pull/5#discussion_r<ID>`で参照できる。修正返信は実装証拠ではなく、固定対象のコードとtestを別に確認した。全行は現時点でも適用yes。Gは「修正されていない」の意味ではなく、元不具合を直接検出するtestを特定できないことを示す。

M70の修正返信はraw再marshal後の1 MiB確認を述べるが、typed resultとsnapshot publicationは別経路でありAUDIT-REDACTION-001がその残件を再現する。M72–M77は現在も残る挙動をoverlayで再現したため、現在の監査にも追加する。M65とM77は同じ不変条件の初回指摘と不完全修正の再指摘である。

| ID / comment / 元重要度 | 不変条件・影響 | 最早/検出/profile | 現test / 限界 | Production | 範囲 |
| --- | --- | --- | --- | --- | --- |
| M54 / 3953621034 / P1 | 通常の大量logをbounded tailへ渡し、永続host-process cleanup barrierにしない。 | S3/S8/B | `runtime/android/ui_test.go:TestUILogPIDAttributionAndBounds` | `runtime/android/ui.go:ObserveUI(logcat)` | G |
| M55 / 3953621041 / P2 | 不正・未登録・別lease snapshot参照はexit2とし、実storage障害のexit7を保つ。 | S4/S8/C | `cli/ui_test.go:TestUIExitClassificationPreservesCauses (typed synthetic errors)` | `app/ui.go:loadUISnapshot` | G |
| M56 / 3953621046 / P2 | helper install実効果と本人性を記録し、一般的なinstall可能性noteだけにしない。 | S4/S8/C | `runtime/android/ui_test.go:TestUIHelperStagingStaysInOwnedRuntimeAndCleansUp (staging only)` | `runtime/android/ui.go:ObserveUI;app/ui.go:UI` | G |
| M57 / 3953621054 / P2 | 全display screenshot・native入力をapplication package隔離と表示しない。 | S3/S8/C | `app/ui_test.go:TestUIDegradedDiagnosticsAndApplicationScope (request scope only)` | `app/ui.go:UI run notes` | G |
| M58 / 3953753802 / P2 | parsed helper応答・後続pollで以前のHelperInstalled=trueを失わない。 | S3/S8/C | `no direct installation-marker/poll regression located` | `runtime/android/ui.go:ObserveUI;app/ui.go:UI` | G |
| M59 / 3953753807 / P2 | load後の明示runtime/snapshot selector不一致も入力errorにする。 | S4/S8/C | `app/ui_test.go:TestUIRejectsInvalidStateAndSelectionBeforeDevice (no exact typed post-load assertion)` | `app/ui.go:UI` | G |
| M60 / 3953753811 / P2 | waitのsleep期限切れではtimeout・未充足を記録し、最後の正常pollをokとして保存しない。 | S3/S8/P | `app/ui_test.go:TestUIWaitIsBoundedAndDoesNotInput (run failure only)` | `app/ui.go:UI polling` | G |
| M61 / 3953753815 / P2 | semantic windowのtitle/bounds/root package/classを保持し、不透明keyだけにしない。 | S3/S8/C | `cli/ui_integration_test.go:TestRealAndroidUIObserver (no explicit window-field assertion)` | `domain/ui.go:UIWindow;runtime/android/uihelper/source/Observer.java` | G |
| M62 / 3953753816 / P2 | 後で再openされる置換可能pathではなく、検証済helper bytesをinstallする。 | S2/S8/I | `runtime/android/ui_test.go:TestUIHelperStagingStaysInOwnedRuntimeAndCleansUp` | `runtime/android/ui.go:ObserveUI install staging` | D |
| M63 / 3953839026 / P1 | raw・normalized・result保存前にwindow metadataを秘匿化する。 | S2/S8/I | `app/ui_test.go:TestUIEvidenceRedactsEnteredAndEditableText (no window secret fixture)` | `app/ui.go:sanitizeUIObservation` | G |
| M64 / 3953839032 / P2 | 不正・stale recover run参照をstorage exit7でなくexit2にする。 | S4/S8/C | `app/ui_recovery_test.go:TestUIRecoverRefusesUnsafeIntentAndEvidence (no CLI exact classification)` | `app/ui_recovery.go:RecoverUI` | G |
| M65 / 3953839035 / P1 | 入力前native preflight失敗で復旧不能uncertaintyを作らない。 | S4/S8/P | `runtime/android/ui_test.go:TestUIOwnershipFailureCannotDispatchInput (does not assert Confirmed)` | `runtime/android/ui.go:ObserveUI native branch` | G |
| M66 / 3953922442 / P1 | 設定secret由来node/window hashをoffline照合oracleとして残さない。 | S2/S8/I | `app/ui_test.go:TestUIEvidenceRedactsEnteredAndEditableText (plaintext only)` | `app/ui.go:sanitizeUIObservation` | G |
| M67 / 3953922444 / P2 | waitにapplication scopeを要求し、別app・system windowでpredicateを満たさない。 | S3/S8/C | `cli/ui_test.go:TestUIFlagValidationAndExitCodes` | `app/ui.go:UI option validation;cli/ui.go` | C |
| M68 / 3953922447 / P2 | wait predicate・timeout・applicationを秘匿化したintentとして保存する。 | S3/S8/C | `app/ui_test.go:TestUIWaitIsBoundedAndDoesNotInput (no argv assertion)` | `app/ui.go:UI run argv` | G |
| M69 / 3953922449 / P2 | 安全error化後もSDK adb欠落の前提exit3を保持する。 | S4/S8/C | `app/ui_test.go:TestUIPrerequisiteErrorSurvivesPrivacySanitization (synthetic provider marker)` | `runtime/android/ui.go:uiSafeError` | G |
| M70 / 3953922453 / P2 | 秘匿化で文字列・省略fieldが増えるためraw/result応答上限を再適用する。 | S2/S8/B | `app/ui_test.go:TestUIEvidenceRedactsEnteredAndEditableText (no final size assertion)` | `app/ui.go:sanitizeUIObservation/UI artifacts` | G |
| M71 / 3953922458 / P2 | host helper pathが変更・欠落しても、recoverは記録helper本人性を使う。 | S4/S8/I | `app/ui_recovery_test.go:TestUIRecoverInterruptedHelperPreservesFailedOutcome (no real digest reconstruction)` | `app/ui_recovery.go:RecoverUI;runtime/android/ui.go:quiesce` | G |
| M72 / 3954080090 / P1 | editable自動抑制でも安全なactionable fingerprintを保つ。 | S3/S8/C | `app/ui_test.go:TestUIEvidenceRedactsEnteredAndEditableText (callback not asserted)` | `app/ui.go:sanitizeUIObservation` | G |
| M73 / 3954080094 / P1 | helper branchでは一般shell backendを検証済source/APK digestへ置き換える。 | S3/S8/I | `runtime/android/ui_test.go:TestUISemanticActionRejectsChangedBackendBeforeDeviceInput (different-string only)` | `runtime/android/ui.go:ObserveUI helper branch` | G |
| M74 / 3954080097 / P1 | window keyだけでなく、window secret由来node fingerprintも除去する。 | S2/S8/I | `no direct nested window-hash regression located` | `app/ui.go:sanitizeUIObservation` | G |
| M75 / 3954080101 / P2 | helper opt-in未設定でcwdを解決し、そのAPKをinstallしてはならない。 | S2/S8/I | `runtime/android/uihelper/helper_test.go:TestVerifyRejectsMismatchedProvenance (explicit directory only)` | `runtime/android/uihelper/helper.go:Load` | G |
| M76 / 3954080103 / P2 | 小数秒lookbackを整数切捨てし、有効な要求範囲のlogを失わない。 | S2/S8/B | `runtime/android/ui_test.go:TestUILogPIDAttributionAndBounds (integral seconds only)` | `app/ui.go:UI request SinceSeconds` | G |
| M77 / 3954080105 / P1 | typed native preflight保護は先にfalseにしたConfirmedを回復するか、その変更を防ぐ。 | S4/S8/P | `runtime/android/ui_test.go:TestUIOwnershipFailureCannotDispatchInput (ignores observation)` | `runtime/android/ui.go:ObserveUI native branch` | G |
