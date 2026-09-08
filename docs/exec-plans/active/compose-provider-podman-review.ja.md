---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/exec-plans/active/compose-provider-podman-review.md
source_sha256: 53cb9f87fff9f571fdfe98307810d8b6c3554c4fc049764d26b843f14923590f
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
- [ ] 共通Docker走査、Podman挙動、harness、native CIを検証する。
- [ ] 修正をpushし、各threadへの返信・Resolve後に本Planを完了へ移す。

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

実装・検証待ち。

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
native Podman/Docker共存と既存Docker integrationは実行中。CI・push・thread対応は未完了。

## インターフェースと依存

依存・公開fieldの追加はない。native Goのsubprocess/socket APIで
Windows/macOS/Linuxへの移植性を維持する。Android/Flutterの責務境界も維持する。
