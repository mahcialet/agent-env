---
status: completed
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/completed/compose-provider-podman-review.md
source_sha256: 2890b85003a28a940066b3965ea3bca821d254a201f109a295bfa51dfc5b9ada
---

# Podman providerのPRレビュー修正

[English（翻訳元）](compose-provider-podman-review.md)

## 目的 / 全体像

PR #8のUDP到達性と任意のComposeツールに依存するinventoryの指摘へ対応する。
作業ブランチは`feat/compose-provider-podman`、開始commitは`4800a1f`。
本Planが今回のレビュー対応の実行範囲を定める。完了済みprovider Planは
当初の実装・検証記録として保持する。

## 進捗

- [x] 2026-09-08: 未解決の2 threadを読み、実装と照合した。
- [x] 2026-09-08: 両指摘を再現し、protocol判定とnative inventoryを修正した。
  provider回帰テスト、全repository check、全raceテストが成功した。
- [x] 2026-09-08: 全harness、race、Docker integration、native Podman/Docker
  共存（104.081秒）、Verify 34221034636の全12 jobが成功した。
- [x] 2026-09-08: 53c141fをpushし、元の両threadへ修正と回帰検証の証拠を返信した。
  両threadをResolveし、本Planをcompletedへ移した。

## 想定外の発見

engineのみのInventoryDoctorテストはInventory呼び出し前で終わっていた。
続く共通Docker走査はCompose lsを実行する。remote endpointの観測では
全mappingにTCP接続を試みるためUDPも削除していた。開始時のCI
34217217740はWindowsのlock喪失テストを再実行して成功済み。

## 判断の記録

- 2026-09-08、実装担当: TCP到達性の検査をTCP endpoint keyだけに適用する。
  UDPのdial成功はapplication readinessの証明にならない。engineが観測した
  UDP mappingと既存のservice/readiness検査を保持し、UDP疎通確認は主張しない。
- 2026-09-08、実装担当: nativeなlabel付きresource走査をDocker Composeの
  project一覧取得から分離する。記録済み実行ファイルが移動した場合も含め、
  Podman inventoryはCompose frontendの存在や実行を必要としない。project labelだけを
  持つorphanも検出するため、値を指定しないlabel存在filterもPodman native labelへ変換する。

## 成果と振り返り

両指摘を解決した。remote UDP mappingにTCP専用のreadiness検査を適用せず、
native Podman inventoryはCompose frontendなしで動作する。Dockerのproject検出と
共通の所有／エラー処理は維持した。英日契約でUDP観測の保証範囲を明記した。

従来のテストは隣接するhelperで止まっていた。engineのみのDoctor検証はその後の
inventory経路を証明せず、endpoint fixtureはremoteでのprotocol混在を扱わなかった。
新しい回帰テストは入口からの全経路、任意ツールの欠落／古いpath、labelのみのorphan、
途中失敗、TCP/UDP混在を扱う。providerの組み合わせもhelperとともに検証する。
実Podman Machineの転送は未検証であり、remote回帰テストはrunnerによるinspectionと
nativeなローカルsocketを使用している。

## 背景と構成

`internal/runtime/compose/podman.go`がremote endpoint検査とPodman inventory
委譲を担う。`inventory.go`はlabel付きcontainer/network/volumeの共通走査を
持つ。`podman_review_test.go`でprovider入口からの経路を検証する。

## 作業計画

runnerを使ったproviderテストとnativeなローカルsocketで両問題を再現する。
provider identity、所有検査、Dockerのproject検出を維持する。英日provider契約に
protocol別の挙動とinventoryの要件を反映する。検証・push後、元の2 threadへ返信する。

## 具体的な手順

`go test ./internal/runtime/compose`、`go run ./tools/repoctl check`、
`go test -race ./...`、`go run ./tools/repoctl test-integration`を実行する。
共通走査の変更後に有効化が必要なPodman/Docker共存suiteを実行する。
最終native CIを確認し、正確な結果を本Planへ記録する。

## 検証と受け入れ

- remote UDP mappingは保持され、到達可能なTCPは利用でき、到達不能なTCPは
  引き続き観測をunreadyにする。
- InventoryDoctorForからInventoryForまでComposeなしでnativeのlabel付き
  resourceを観測できる。存在しない記録済みCompose実行ファイルでも検証する。
- native identity固定、エラー処理、Docker inventory挙動の既存テストを維持し、
  全harnessと関連integrationが成功する。
- 元の各review threadに修正と検証の具体的な返信があり、解決済みとなる。

## 冪等性と復旧

既存featureブランチで作業し、公開済み履歴を保持する。テストは隔離したfixtureを
使い、自分が所有するresourceだけをcleanupする。修正をpush・検証するまでは
threadをResolveしない。

## 成果物と注記

指摘: discussion_r3957283152（UDP）、discussion_r3957283162（inventory）。
修正前は`TestPodmanRemoteInspectPreservesUDPAndChecksTCP`でUDP mappingが消失し、
`TestPodmanInventoryWithoutComposeDiscoversNativeOrphans`でComposeの欠落・古いpathに
より失敗した。修正後は到達可能／不能なTCP、native resource全3種、label競合、
inventory途中失敗を含め成功した。別担当がproduction差分を独立確認し、providerの
raceテストを実行した。`go run ./tools/repoctl check`と`go test -race ./...`も成功した。
native Podman/Docker共存は104.081秒で成功し、既存Docker integrationも成功した。
53c141fの[Verify 34221034636](https://github.com/mahcialet/agent-env/actions/runs/34221034636)
は全12 job（Windows/macOS/LinuxとGo 1.26/1.27のnative 6 job、cross-build 5 job、
Linuxのrace／Docker integration）が成功した。元の両threadへ返信してResolveした。
返信はdiscussion_r3957440177（UDP）とdiscussion_r3957440447（inventory）。
完了移動後の文書もdocs-checkで成功した。

## インターフェースと依存

依存・公開fieldの追加はない。native Goのsubprocess/socket APIで
Windows/macOS/Linuxへの移植性を維持する。Android/Flutterの責務境界も維持する。
