---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/matrix.md
source_sha256: 62758d03eed46457abe68a8fb98e74ab4cea8b8c60988a26754ebdc12f74bbdd
---

# subsystemと不変条件の監査表

[English](matrix.md) · [監査index](index.ja.md) · [処置台帳](findings.ja.md)

本表はPhase A固定revision `031869c8b9073b8e23bc17fbc55243666a52f557`に対する
限定した各資料を統合する。coverageと指摘の位置を記録し、普遍的な正しさや候補版検証の
代替を主張しない。19指摘の処置・最終解決状態は台帳が管理する。
以下のPhase C機構は作業候補版に存在し、修正前失敗・対象検証証拠は参照資料、
最終検証はindex/ExecPlanが管理する。履歴やcacheの成功を新規native証拠へ置き換えない。

## セルと参照元の意味

- `R(A)`など: 参照資料に限界を記したsource・contractと、それらの動作を確かめるテストの限定レビュー。
  全syscall順序・数値組合せ・native platformを実行したという意味ではない。
- `AUDIT-…`: 固定対象で確認した指摘のあるレビュー済セル。
  同一IDの複数掲載は境界をまたぐ1不具合を示し、別件数として数えない。解決状態は台帳を参照。
- `N-P`: この純粋schema/checker部分に独立したmanaged lifecycle・effect・cancel責務はない。
  呼出側はapp/runtime行で確認する。
- `N-T`: accessibility/page/nodeのsemantic actionや不在waitの入口はない。
  その他のidentity鮮度はidentity/並行性欄で確認する。
- `N-L`: release version・archive・公開責務はない。実行fileにcompileされるだけでは公開処理を担わない。

A: [制御処理](current-control-plane.ja.md)。config/domain/store、
source/path/evidence/readiness/reconcile/cleanupと過去対応付けを含む。
B: [mobile](current-mobile.ja.md)と[過去資料](history-mobile.ja.md)。
C: [process/browser/execx](current-process-browser.ja.md)と[過去資料](history-process-browser.ja.md)、遅れて確認した
[CDP](supplemental-browser.ja.md)・[CLI](supplemental-cli.ja.md)追加資料。
D: [Compose/release/asset](current-compose-release.ja.md)と[過去資料](history-compose-release.ja.md)。
E: [文書/harness](documentation.ja.md)。
全行に各資料のnative・coverage限界も適用する。
レビュー印によって明示的なoracle欠落や未実行testを消すことはできない。

## 監査表

3つの表で同じ20 subsystem行を使い、合わせて14の主要不変条件を扱う。
共通codeは各caller行にも意図的に現れる。例えばDocker/PodmanはCompose container経路を共有し、
readiness失敗はapp状態と後続cleanupへ伝播する。

### 上限・積極的証拠・遷移・永続化・identity

| Subsystem | 上限 | 積極的証拠 | 状態遷移 | intent/effect/result保存 | 所有/identity |
| --- | --- | --- | --- | --- | --- |
| config/domain | R(A) | R(A) | R(A) | N-P | R(A) |
| app orchestration | R(A) | AUDIT-CLEANUP-001; AUDIT-CLI-001 | R(A) | AUDIT-CLEANUP-001 | R(A) |
| SQLite store | R(A) | R(A) | R(A) | R(A) | R(A) |
| Git source/worktree | R(A) | R(A) | R(A) | R(A) | R(A) |
| Docker Compose | R(D) | AUDIT-OWNERSHIP-001 | R(D) | R(D) | AUDIT-OWNERSHIP-001 |
| Podman Compose | R(D) | AUDIT-OWNERSHIP-001 | R(D) | R(D) | AUDIT-OWNERSHIP-001 |
| Android Emulator | R(B) | R(B) | R(B) | R(B) | R(B) |
| Flutter Android | R(B) | R(B) | R(B) | R(B) | R(B) |
| Android UI | AUDIT-BOUNDARY-001; AUDIT-BOUNDARY-002 | AUDIT-LIFECYCLE-001 | AUDIT-LIFECYCLE-001 | AUDIT-LIFECYCLE-001 | AUDIT-IDENTITY-001; AUDIT-PREREQUISITE-001 |
| persistent process | R(C) | R(C) | R(C) | R(C) | R(C) |
| Browser/CDP | AUDIT-REDACTION-002; AUDIT-BOUNDARY-003 | AUDIT-STALE-001; AUDIT-STATE-001 | AUDIT-BOUNDARY-003 | R(C) | AUDIT-STALE-002 |
| execx | R(C) | R(C) | R(C) | R(C) | R(C) |
| evidence/redaction | AUDIT-REDACTION-001; AUDIT-REDACTION-002 | R(A) | N-P | R(A) | R(A) |
| path/filesystem | R(A) | R(A) | N-P | R(A) | R(A) |
| readiness/endpoint | R(A) | AUDIT-CLEANUP-001 | AUDIT-CLEANUP-001 | AUDIT-CLEANUP-001 | R(A) |
| reconcile | R(A) | R(A) | R(A) | R(A) | R(A) |
| destroy/GC/quarantine | R(A) | AUDIT-OWNERSHIP-001 | R(A) | R(A) | AUDIT-OWNERSHIP-001 |
| release/buildinfo | AUDIT-RELEASE-001; AUDIT-RELEASE-002 | R(D) | R(D) | R(D) | R(D) |
| embedded asset | R(D) | R(D) | R(D) | R(D) | R(D) |
| 文書/harness | R(E) | AUDIT-DOCS-001 | N-P | N-P | R(E) |

### cleanup・cancel・並行性・stale semantics・証拠

| Subsystem | cleanup証明 | cancel/期限/fence | 並行/予約 | stale semantic target | redaction/証拠上限 |
| --- | --- | --- | --- | --- | --- |
| config/domain | N-P | N-P | R(A) | N-T | R(A) |
| app orchestration | AUDIT-CLEANUP-001 | AUDIT-CLEANUP-001; AUDIT-CLI-001 | R(A) | R(A) | R(A) |
| SQLite store | R(A) | R(A) | R(A) | N-T | R(A) |
| Git source/worktree | R(A) | R(A) | R(A) | N-T | R(A) |
| Docker Compose | AUDIT-OWNERSHIP-001 | R(D) | R(D) | N-T | R(D) |
| Podman Compose | AUDIT-OWNERSHIP-001 | R(D) | R(D) | N-T | R(D) |
| Android Emulator | R(B) | R(B) | R(B) | N-T | R(B) |
| Flutter Android | R(B) | R(B) | R(B) | N-T | R(B) |
| Android UI | AUDIT-LIFECYCLE-001 | R(B) | R(B) | AUDIT-UI-001; AUDIT-IDENTITY-001 | AUDIT-REDACTION-001; AUDIT-REDACTION-003 |
| persistent process | R(C) | R(C) | R(C) | N-T | R(C) |
| Browser/CDP | R(C) | R(C) | AUDIT-STALE-001; AUDIT-STALE-002 | AUDIT-STALE-001; AUDIT-STALE-002; AUDIT-STATE-001 | AUDIT-REDACTION-002; AUDIT-CLI-001 |
| execx | R(C) | R(C) | R(C) | N-T | R(C) |
| evidence/redaction | R(A) | R(A) | R(A) | N-T | AUDIT-REDACTION-003 |
| path/filesystem | R(A) | R(A) | R(A) | N-T | R(A) |
| readiness/endpoint | AUDIT-CLEANUP-001 | AUDIT-CLEANUP-001 | R(A) | N-T | R(A) |
| reconcile | R(A) | R(A) | R(A) | N-T | R(A) |
| destroy/GC/quarantine | AUDIT-CLEANUP-001 | AUDIT-CLEANUP-001 | R(A) | N-T | R(A) |
| release/buildinfo | R(D) | R(D) | AUDIT-RELEASE-002 | N-T | AUDIT-RELEASE-001 |
| embedded asset | R(D) | R(D) | R(D) | N-T | R(D) |
| 文書/harness | N-P | N-P | N-P | N-T | R(E) |

### path・native意味論・release・文書

| Subsystem | path/filesystem | native移植性 | releaseの正しさ | 文書主張/検査 |
| --- | --- | --- | --- | --- |
| config/domain | R(A) | R(A) | N-L | R(A) |
| app orchestration | R(A) | R(A) | N-L | R(A) |
| SQLite store | R(A) | R(A) | N-L | R(A) |
| Git source/worktree | R(A) | R(A) | N-L | R(A) |
| Docker Compose | R(D) | R(D) | N-L | R(D) |
| Podman Compose | R(D) | R(D) | N-L | R(D) |
| Android Emulator | R(B) | R(B) | N-L | R(B) |
| Flutter Android | R(B) | R(B) | N-L | R(B) |
| Android UI | AUDIT-PREREQUISITE-001 | R(B) | N-L | R(B) |
| persistent process | R(C) | R(C) | N-L | R(C) |
| Browser/CDP | R(C) | R(C) | N-L | R(C) |
| execx | R(C) | R(C) | N-L | R(C) |
| evidence/redaction | R(A) | R(A) | N-L | R(A) |
| path/filesystem | R(A) | R(A) | N-L | R(A) |
| readiness/endpoint | R(A) | R(A) | N-L | R(A) |
| reconcile | R(A) | R(A) | N-L | R(A) |
| destroy/GC/quarantine | R(A) | R(A) | N-L | R(A) |
| release/buildinfo | AUDIT-RELEASE-001; AUDIT-RELEASE-002 | R(D) | AUDIT-RELEASE-001; AUDIT-RELEASE-002 | R(D) |
| embedded asset | R(D) | R(D) | R(D) | R(D) |
| 文書/harness | R(E) | R(E) | R(E) | AUDIT-DOCS-001 |

## 現在の指摘の見逃しと予防策

19件全てS9で再現し初回/追加Phase BでACCEPTとした。初回15件の初発見は本監査S9、
追加4件の初発見は外部S8 commentで、その後S9で独立再現した。各資料と同じ最早段階は
S2が9件、S3が7件、S4が3件。所有の欠落field oracleはS2、
公開Down barrierの証明はS3である。既に検査可能な契約があった部分に
S0/S1要件欠落を推測で追加しない。将来の期待段階は追加test/checkが働く場所で、
同型の全変種を検出する保証ではない。test名は候補版に実在するものを示し、
実行証拠は各担当資料を参照する。

### AUDIT-CLEANUP-001

- 検出/最早段階: S9/S4、分類: `COMPOSITION_GAP,FAILURE_INJECTION_GAP,HELPER_ONLY,REVIEW_CHECKLIST_GAP`。
- より早い機会と見逃し: executorの型付き安全契約は存在したがreadinessがretry・文字列化した。新しい組合せtestは即時errorだけでなく後続Destroyも検査する。
- 実装した機構と確認テスト: `readinessCommand -> runWithCancellation + durable CommandRun; TestReadinessRetainsUnconfirmedCommand, TestReadinessOrdinaryFailureMayRetryAndRelease, TestReviewReadinessCancellationDoesNotRetry`。
- 将来の期待検出: S3/S4。候補版証拠/解決状態は担当資料と台帳を参照。

### AUDIT-OWNERSHIP-001

- 検出/最早段階: S9/S2、分類: `NEGATIVE_FIXTURE_GAP,ORACLE_COUPLING,COMPOSITION_GAP`。
- より早い機会と見逃し: label正常fixtureは常にIDを供給した。欠落fieldをlabelから独立させ破壊入口未到達を確認する。
- 実装した機構と確認テスト: `dockerClient.Inspect shared by provider Down; TestMissingContainerIdentityRefusesInspectionAndDown`。
- 将来の期待検出: S2/S3。候補版証拠/解決状態は担当資料と台帳を参照。

### AUDIT-DOCS-001

- 検出/最早段階: S9/S2、分類: `COMPOSITION_GAP,NEGATIVE_FIXTURE_GAP,REVIEW_CHECKLIST_GAP`。
- より早い機会と見逃し: 既存prose filterはlink・必須headingを保護したがtarget heading側には適用しなかった。
- 実装した機構と確認テスト: `documentProse reused in fragment target scan; TestFragmentsRequireRenderedTargetHeadings`。
- 将来の期待検出: S2/S6。候補版証拠/解決状態は担当資料と台帳を参照。

### AUDIT-RELEASE-001

- 検出/最早段階: S9/S2、分類: `BOUNDARY_GAP,NEGATIVE_FIXTURE_GAP`。
- より早い機会と見逃し: 全root例が長さによる除外より長かった。短い合法rootで検査し真のvolume rootの明示policyは維持する。
- 実装した機構と確認テスト: `structural root classification; TestReleaseBinaryShortCheckoutPaths`。
- 将来の期待検出: S2。候補版証拠/解決状態は担当資料と台帳を参照。

### AUDIT-RELEASE-002

- 検出/最早段階: S9/S2、分類: `CONCURRENCY_GAP,BOUNDARY_GAP,FAILURE_INJECTION_GAP`。
- より早い機会と見逃し: 固定size超過fileだけではstat後の増大を見逃し、asset修正もrelease reader/writerへ広がらなかった。
- 実装した機構と確認テスト: `bounded opened-file reader shared by check/copy/compare and writer; TestReleaseReadBoundsGrowthAfterOpenedStat, TestReleaseRegularReadExactBounds`。
- 将来の期待検出: S2/S3。候補版証拠/解決状態は担当資料と台帳を参照。

### AUDIT-BOUNDARY-001

- 検出/最早段階: S9/S2、分類: `BOUNDARY_GAP,ORACLE_COUPLING`。
- より早い機会と見逃し: 以前の件数超過fixtureはbyte上限も越えた。ちょうど件数は>=だけでなく独立したoverflow証拠が必要。
- 実装した機構と確認テスト: `2001-record provider probe -> boundedUILog; TestUILogExactTailUsesOverflowProof`。
- 将来の期待検出: S2/S3。候補版証拠/解決状態は担当資料と台帳を参照。

### AUDIT-REDACTION-001

- 検出/最早段階: S9/S2、分類: `BOUNDARY_GAP,COMPOSITION_GAP,ORACLE_COUPLING`。
- より早い機会と見逃し: producer上限とredactionは別々に通ったが、最終encodingと拡張fieldの組合せoracleがなかった。
- 実装した機構と確認テスト: `boundUIEvidence reused for raw/normalized/result envelopes; TestUIAuditUIRedactionBounds, TestUIFieldBoundaryAndSerializedObservationBoundary`。
- 将来の期待検出: S2/S4。候補版証拠/解決状態は担当資料と台帳を参照。

### AUDIT-REDACTION-002

- 検出/最早段階: S9/S4、分類: `COMPOSITION_GAP,BOUNDARY_GAP`。
- より早い機会と見逃し: semantic/capture最終上限が隣接DOM種別を省いた。実保存byteと保持identityを読む。
- 実装した機構と確認テスト: `boundBrowserDOM before artifact publication; TestBrowserDOMBoundsAfterRedaction, TestBrowserDOMEncodedBoundary`。
- 将来の期待検出: S4。候補版証拠/解決状態は担当資料と台帳を参照。

### AUDIT-STALE-001

- 検出/最早段階: S9/S3、分類: `CONCURRENCY_GAP,COMPOSITION_GAP`。
- より早い機会と見逃し: snapshot整合性が後続predicateも保護すると考えた。両段階の間に変化を注入する必要がある。
- 実装した機構と確認テスト: `load predicate post-evaluation frameDocument check; TestLoadWaitRechecksDocumentAfterPredicate`。
- 将来の期待検出: S3。候補版証拠/解決状態は担当資料と台帳を参照。

### AUDIT-UI-001

- 検出/最早段階: S9/S3、分類: `ORACLE_COUPLING,FAILURE_INJECTION_GAP,COMPOSITION_GAP`。
- より早い機会と見逃し: errorだけのassertが入力callback前のstale拒否でも通った。組合せ正常例でcallback到達とdurable成功を確認する。
- 実装した機構と確認テスト: `editable suppression distinct from secret-derived hash invalidation; TestUIAuditEditableSnapshotRemainsActionable`。
- 将来の期待検出: S3/S4。候補版証拠/解決状態は担当資料と台帳を参照。

### AUDIT-IDENTITY-001

- 検出/最早段階: S9/S3、分類: `ORACLE_COUPLING,INVARIANT_GAP,COMPOSITION_GAP`。
- より早い機会と見逃し: 任意の別backendは正しいdigestでも誤った汎用文字列でも拒否できた。同じbuildの正確な正常例がなかった。
- 実装した機構と確認テスト: `exact verified backend + recorded-identity recovery; TestUIAuditHelperBackendCarriesVerifiedDigest, TestUIRecoveryUsesRecordedHelperWithAbsentOrReplacedHostFiles`。
- 将来の期待検出: S3/S4。候補版証拠/解決状態は担当資料と台帳を参照。

### AUDIT-REDACTION-003

- 検出/最早段階: S9/S2、分類: `INVARIANT_GAP,NEGATIVE_FIXTURE_GAP,ORACLE_COUPLING`。
- より早い機会と見逃し: 平文検索ではsecretを含むwindow ancestry由来identifierを検査できなかった。
- 実装した機構と確認テスト: `window-secret invalidation clears dependent node hashes; TestUIAuditWindowSecretClearsDerivedNodeHashes`。
- 将来の期待検出: S2/S4。候補版証拠/解決状態は担当資料と台帳を参照。

### AUDIT-PREREQUISITE-001

- 検出/最早段階: S9/S2、分類: `NEGATIVE_FIXTURE_GAP,COMPOSITION_GAP`。
- より早い機会と見逃し: 正常/不正provenance fixtureは全て明示directoryを渡した。validなcwd fileがあるopt-in欠落は別条件。
- 実装した機構と確認テスト: `uihelper.Load explicit nonblank directory before Abs; TestUIAuditUnsetHelperRefusesCurrentDirectory`。
- 将来の期待検出: S2/S3。候補版証拠/解決状態は担当資料と台帳を参照。

### AUDIT-BOUNDARY-002

- 検出/最早段階: S9/S2、分類: `BOUNDARY_GAP,NEGATIVE_FIXTURE_GAP`。
- より早い機会と見逃し: 整数秒の例では情報を失う整数変換に入らない。非対応の端数は黙って短縮せず拒否する。
- 実装した機構と確認テスト: `whole-second UI lookback validation before store/provider; TestUIAuditFractionalLogLookback`。
- 将来の期待検出: S2。候補版証拠/解決状態は担当資料と台帳を参照。

### AUDIT-LIFECYCLE-001

- 検出/最早段階: S9/S4、分類: `FAILURE_INJECTION_GAP,COMPOSITION_GAP,ORACLE_COUPLING`。
- より早い機会と見逃し: error/無入力testが戻り値confirmationを捨てた。公開run/cleanupでpreflightと実起動後不明を区別する。
- 実装した機構と確認テスト: `effect-aware ADB invocation preserves preflight certainty; TestUIAuditNativePreflightRemainsConfirmed, TestUINativePreflightThroughAppDoesNotBlockCleanup, TestUINativeDispatchedFailureRemainsUnconfirmed`。
- 将来の期待検出: S3/S4。候補版証拠/解決状態は担当資料と台帳を参照。


### AUDIT-BOUNDARY-003

- 検出/最早段階: 元発見S8、監査S9/S3。分類:
  `COMPOSITION_GAP,BOUNDARY_GAP,NEGATIVE_FIXTURE_GAP`。
- 機会と見逃し: 上限時のlistは検査したが、その一覧後に1 page増やす作成を通らなかった。
  producer自身がconsumer上限を越えた。
- 機構と確認テスト: 作成前の共通page上限。
  `TestSupplementPageLimitPreventsIrrecoverableGrowth`で127/128/129、
  effect回数、後続closeを検査する。
- 将来期待S3。[追加資料](supplemental-browser.ja.md)に固定版/候補版失敗、
  対象・独立race証拠を記録。

### AUDIT-CLI-001

- 検出/最早段階: 元発見S8、監査S9/S3。分類:
  `COMPOSITION_GAP,NEGATIVE_FIXTURE_GAP,ORACLE_COUPLING`。
- 機会と見逃し: adapterは作成ID/close結果を返したが通常text表示が捨てた。
  JSON testだけでは人間向け結果を証明しない。
- 機構と確認テスト: `internal/cli/browser.go`のRunE内operation分岐、
  `TestBrowserNativeCLI/page_mutation_table_identifies_affected_page`による実作成targetと
  close後不在の検査。
- 将来期待S3/S5。[CLI追加資料](supplemental-cli.ja.md)にnative race10.500sと
  修正前公開text失敗を記録。

### AUDIT-STATE-001

- 検出/最早段階: 元発見S8、監査S9/S3。分類:
  `COMPOSITION_GAP,INVARIANT_GAP,NEGATIVE_FIXTURE_GAP`。
- 機会と見逃し: snapshotはIgnoredを保持し入力も拒否したが、別consumerのwaitがaccessibleと数えた。
- 機構と確認テスト: ignored filter、
  `TestSupplementIgnoredAXCannotSatisfyWait`でtext/goneの両極性とAX取得到達を検査。
  truncated-gone guardは維持する。
- 将来期待S3。追加証拠は上の資料を参照。

### AUDIT-STALE-002

- 検出/最早段階: 元発見S8、監査S9/S3。分類:
  `NEGATIVE_FIXTURE_GAP,ORACLE_COUPLING,COMPOSITION_GAP`。
- 機会と見逃し: checkedの変化によるstale入力の拒否だけを検証しても、別fieldのpressedがfingerprintに含まれる証明にならなかった。
- 機構と確認テスト: 既存の制限付きaxStateへpressedを追加し、
  `TestSupplementPressedStateRefusesStaleInput`でbool/mixedと既存8flag、
  mutation到達・無入力を検査する。
- 将来期待S3。任意state文字列は許さない。

## 繰り返す分類: 再利用する機構と限界

| 分類・subsystem横断検索 | 再利用可能または具体的な予防機構と証拠 | 普遍的checkerを主張しない理由 |
| --- | --- | --- |
| 上限/完全性: Browser AX/DOM、Android tree/log、console/network、release file | 正確な境界表と最終encoding assert。boundUIEvidenceをraw/normalized/resultへ適用し、DOMはIDを保つnode prefix、logはoverflow用1件、release readerはopened byteを検査。上記testは通常package/harness経路に入る。 | byte・Unicode文字・record・node・caller envelope・device tail省略は単位と権限意味が異なる。汎用truncateだけでは区別を隠すため、共通原則を明示domain oracleで検証する。 |
| 上限後変換/privacy: mobile、Browser、process log、共通evidence | 公開Service UI/Browserでredaction後artifactの存在/size/secret不在/identityとdurable完了を検査。共通Redactorのchunk横断検査を維持しwindowの秘密値に由来するhashが残らないことを確認するテストを追加。 | 文字列置換だけではsecret由来hashやaction権限fieldを判別できない。screenshot/private profileはpixel-redaction対象外で、包括的な秘密保護を偽って主張しない。 |
| Effect前identity証明: Compose、process/Job、Android、Browser、Git | container ID欠落で公開Down拒否、helperの同一/別buildを正確に検査。既存のOS birth/Job/session、frame/node、source登録の不一致や不備を拒否するテストを維持。 | engine label/ID、Job handle、AVD、Chromium origin、Git登録は異なる証拠。nameだけの共通所有抽象化は契約を弱めるため、証明の問いを共通化し形式は交換しない。 |
| 保存/effect/cleanup/cancel: named test、readiness、UI、Browser、release | readinessはCommandRunとrunWithCancellationを再利用し不明attemptを既存Destroy/GC gateのrunningに残す。UIはpreflightと実起動後不明の両側を検査、releaseは検証済byte公開を維持。 | 外部OS/device/engine effectを汎用transactionで原子的にcommitできない。明示intent/identity/finalize barrierと失敗注入が必要。Windows第2PID読取やrelease後段close/persist注入は明示残作業とする。 |
| Stale/時間的証拠: Browser snapshot/load/URL、Android semantic入力 | predicate後document再検査と単発・連続navigation後にstale証拠を返さないことを確認するテスト、既存frame変化・stale fingerprint・coordinate fallback禁止を維持。 | CDP/ADB/OS複数callに普遍的原子観測はない。各操作がtokenとretry/不明境界を定める。今回はload欠落境界を補い、全体原子性は保証しない。 |
| 環境依存前提/path採用: helper、Git、asset、release | helperはAbs前に空opt-in拒否。既存provenance/symlink/asset opened-readに短いrelease rootと実opened-file増大guardを加える。 | filesystem alias/handle/caseはOSで異なりnative testが必要。静的検査で契約外の同一user悪意置換全ての安全を証明しない。 |
| 弱い/空振りoracle: 全adapterの過去レビュー | 意図したcallback/effect到達、登録artifact存在、durable状態、近傍正常例成功を要求するtestへ改善。過去helper-only限界は資料に残す。 | 汎用testが他testの注入意味を判断することはできない。coverage率を対象分岐の証明とせず注入counterと独立出力で検査する。 |
| 文書主張/navigation | 必須section/linkと同じdocumentProseをfragment対象へ再利用。公開docsCheckで非表示headingへのリンクを拒否するテストと有効な重複headingのテスト、既存の日英hash不一致・index不備を拒否するテストを維持。 | link/hashは機械検査できるが翻訳意味やnative受入の真偽は人間・独立証拠reviewが必要。proseから実装を自動証明するcheckerは主張しない。 |
| Native/並行挙動 | 独立SQLite connection/process、process/Job/native provider fixtureと候補CIで非mock証拠を得る。fileの読込上限を確認するテストでは、openしたfileのstat取得直後に内容を増やす。 | Linux mock/cross-buildでWindows/macOS/ADB/Machineを保証できない。安定したlocal失敗hookがない範囲は台帳へriskを明記し、native証拠を装うframeworkを追加しない。 |

以上が具体的な共通化判断である。契約が共通なら既存製品/test入口を再利用し、
異なる場合は独立したdomain別の正常・異常oracleを加える。
global AGENTS・組織policy・依存・広大なharness frameworkを追加する必要はなかった。
browser選択時の保存manifest改変やreadiness内secretリテラルの拒否は、過去のテストでは
helperだけで検証している。この公開入口の検証不足やJava producer探索など、残るcoverageは
元資料と台帳へ明示し、完了testへ書き換えない。

## 受入での用途と検証限界

- A3: 全20 × 14表に明示N/A理由と指摘IDを示した。
- A27: 全ACCEPT指摘に検出/最早段階、分類、より早い具体的機会と実装機構の対応を記した。
- A28/A35: 各反復分類に再利用した機構と確認テストの証拠または広い自動予防が不適切・過大となる理由を記した。
  機構の範囲を定めるもので、最終test・独立review成功を意味しない。
- 解決件数は台帳、最終受入は完了Planとindexに記録し、native/integration成功と
  独立reviewを含む。matrixだけでその実行証拠を代替しない。
