---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/roadmap.md
source_sha256: c7688e6bd6e80e3a516b9ead059746ddf8e5ae4ef9f76a9bfd67f8f186fb017e
---

# Roadmapと未決定事項

[English](roadmap.md)

このroadmapでは、提供済みの機能と、新たな作業・判断が必要な事項を区別します。現在の挙動は
[製品仕様](product-specs/index.ja.md)を参照してください。進行中の実装はactive ExecPlanに従います。
開始・再開の手順は[Planの方針](PLANS.ja.md)にあります。ここに延期事項として載せた機能は、
利用可能なコマンドでも、承認済みの実装設計でもありません。

## MVPで確定した選択

module名は`github.com/mahcialet/agent-env`で、既存のMIT licenseを維持します。
固定したローカルGit commit、detached review worktree、componentとその直接・間接の依存先すべての選択、名前付きargvテスト、
証拠の保持、移植可能なrepository harnessが実装済みの基盤です。

AGENTSの上限は150行です。package境界は構造検査で強制し、番号付きmigrationからDB文書を生成します。
SQLite transactionと更新可能なfence付きoperation lockがlocal process間を調整します。
実行snapshotには、選択したサービスとその直接・間接の依存サービス、およびそれらが参照するresourceだけを保存します。組み込みpolicyは
固定公開portと、指定された外部・共有Compose resourceを拒否します。

時間を制限したコマンドのprocess treeキャンセルと、常駐processの管理には、別のnative interfaceを使います。
以下の機能群もこの基盤に従い、localの所有権検査やcleanup条件を置き換えません。

## 信頼境界とホストポリシー

**実装済み:** 信頼できる、または管理下のリポジトリに組み込みhost policyを適用します。
owner labelは補助情報であり、ユーザー認証ではありません。安全でないリポジトリを受け入れるために
policyを緩めてはいけません。[セキュリティ](SECURITY.ja.md)を参照してください。

**未決定:** host policy file、猶予・保持期間、並列割当数の設定は今後の作業です。
信頼したbase manifestと対象overlayのmerge、`--manifest-ref`、未信頼forkの明示実行には、
別途信頼設計が必要です。

## ソースと変更を残す作業

**実装済み:** ソースはcommitを固定したローカルGitです。remote modeはcommit済みGit bundleを転送しますが、
一般的なremote Git認証やfetchは提供しません。

**延期:** mirror/cache管理、HTTPS/SSH認証、provider固有PR短縮指定、自動fetch。
変更を残すfix leaseには、branch所有権、sourceごとの書き込み選択、復旧規則が必要です。
fork/checkpoint/reproduceや稼働中stackの拡大・縮小には、identityとartifactのモデルを定める必要があります。

## Runtimeの拡張

### Compose provider

**実装済み:** [DockerとPodmanの選択](product-specs/compose-providers.ja.md)をleaseごとに固定します。
既定はDockerで、自動fallbackはありません。

**証拠の範囲:** [完了済みprovider Plan](exec-plans/completed/compose-provider-podman.ja.md)に、
Linux rootless受け入れとDocker併存、Windows/macOS/Linuxのnative provider CIの記録があります。
実Podman Machine環境は利用できませんでした。検証したversion、revision、run IDはそのPlanを参照してください。

**延期:** `podman compose` wrapper、Quadlet/Kubernetes、pod作成、任意のprovider実行ファイルは対象外です。

### 常駐processとブラウザ

**実装済み:** [process runtime](product-specs/persistent-process-runtime.ja.md)は、引数配列による直接実行、
専用の可変状態、名前付きloopback TCP port、所有権を確認したうえでのプロセスツリー削除を提供します。
[Browser/CDP](product-specs/browser-cdp-automation.ja.md)は、processの所有権管理の上に独立した
interfaceで観測・操作を加えます。両方ともWindows、macOS、Linuxのnative受け入れを完了しています。
証拠は[process](exec-plans/completed/persistent-process-runtime.ja.md)と
[browser](exec-plans/completed/browser-cdp-automation.ja.md)の完了Planにあります。

**延期:** processの自己daemon化、自動再起動、対話terminal、remote processへの直接attach、service登録。
browserの外部attach、headful mode、download、Firefox/BiDi、Safari/WebKit、Androidとbrowserに共通の
UI抽象化も現在の対象外です。

### AndroidとFlutter

**実装済み:** [Android lease](product-specs/android-emulator.ja.md)は専用AVD状態とlocal SDK processを
所有します。[Flutterアプリ](product-specs/flutter-android-runtime.ja.md)は、そのEmulator上でbuild、
install、launch、backend reverse mappingを行います。[UI observer](product-specs/android-ui-observer.ja.md)は
上限付きsemantic snapshot、PNG、Unicode置換、navigation、現在のアプリプロセスに限定したログを提供します。
任意のplatform companionに対象アプリへのinstrumentationは不要です。

**証拠の範囲:** observerの受け入れは[完了Plan](exec-plans/completed/android-ui-observer.ja.md)にあります。
追加のnative Emulator検証には仮想化支援を利用できるrunnerが必要です。Android統合テストにも適切なhardware accelerationが必要です。

**延期:** iOS、OCR、visual regression、より豊富なgesture、物理device、管理外remote Emulatorへのattach。

## Multi-hostの受け入れと延期した拡張

**実装済み:** [単一controller仕様](product-specs/multi-host-control-plane.ja.md)は、lease全体の明示配置、
commit済みソース転送、型付きworker操作を提供します。操作は順次dispatchしますが、作成済みleaseは並行稼働できます。
同じleaseで操作が実行中の場合は、テスト中のdestroyも含めて別の操作を拒否します。

**証拠の範囲:** [完了Plan](exec-plans/completed/multi-host-control-plane.ja.md)には、Windows・macOS・Linuxの
実TLS native受け入れ記録と、検証したrevision・run IDがあります。
各runnerで2つのworker rootを使い、名前付きテスト、log、artifact取得、renew、環境分離を検証しました。
物理multi-host/VMの証拠は未取得です。受け入れ完了はこの範囲に限り、すべての配置構成を保証するものではありません。

**延期:** remote cancel-active、並行operation dispatch、controller HA/consensus、live migration、
hostをまたぐlease、透過的endpoint tunnel、secret配布、緊急時のhost引き継ぎには別の設計が必要です。

## Artifactとrelease

**実装済み:** 初期配布は[standalone仕様](product-specs/standalone-distribution.ja.md)に従うGitHub Release
archiveです。[release Plan](exec-plans/completed/standalone-release-finalization.ja.md)にrelease工程と
直接native検証の証拠があります。runtime検査で実container imageのidentityを記録しますが、
再現可能なimage promotionを提供するわけではありません。

**延期:** local OCI registry、image promotion、image保持参照、同一artifactの再実行、署名、notarization、
package manager recipe、self-update、SBOM、attestation。

**未決定:** artifactの自動期限切れ、event圧縮、migration rollbackツール、生成CLI/JSON Schema参照、
長期handoff archive方針。releaseは検証したrevisionのnative・統合証拠を示す必要があります。
対応build targetの一覧で証拠を代用することはできません。

## CIの拡張

WindowsとmacOSのnative Docker統合には、Dockerを使えるself-hosted runnerが必要になる場合があります。
このインフラ条件はcross-build対応とは別です。現在の検証範囲は[品質](QUALITY.ja.md)と
[移植性](PORTABILITY.ja.md)を参照してください。

文書のmetadataと到達可能性は検査しますが、`last_verified`からの経過日数をCIの失効条件にはしていません。
意味の確認と翻訳検査には[言語方針](design-docs/bilingual-documentation.ja.md)を使い、有効なmetadataだけで
内容が最新だと判断しないでください。
