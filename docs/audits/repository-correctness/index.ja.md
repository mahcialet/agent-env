---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/audits/repository-correctness/index.md
source_sha256: ff9a8ea78e290d38c35e470a1c8b76addddd88ecad65956ca36f3cb276ac3537
---

# リポジトリ正確性監査

[English](index.md)

実行指示: [進行中の監査 Plan](../../exec-plans/active/repository-correctness-audit.ja.md)。
Phase A の固定対象は PR #10 merge 後 master の `031869c8b9073b8e23bc17fbc55243666a52f557`。作業ブランチは `audit/repository-correctness`。

監査は進行中。採否確定までは観測結果と修正案として扱い、製品コードは変更していない。過去テスト名の対応表は、過去の全不具合版を mutation replay したという意味ではない。

## レポート

- [文書の過去指摘とフラグメント監査](documentation.ja.md) / [English](documentation.md)
- [制御処理の監査](current-control-plane.ja.md) / [English](current-control-plane.md)
- [mobile の過去指摘](history-mobile.ja.md) / [English](history-mobile.md)
- [mobile の現行監査](current-mobile.ja.md) / [English](current-mobile.md)
- [process/browser/MVP の過去指摘](history-process-browser.ja.md) / [English](history-process-browser.md)
- [Compose/release の過去指摘](history-compose-release.ja.md) / [English](history-compose-release.md)

- [過去指摘の全体索引と見逃し分析](historical-corpus.ja.md) / [English](historical-corpus.md)

- [Compose と release の現行監査](current-compose-release.ja.md) / [English](current-compose-release.md)

- [process と Browser の現行監査](current-process-browser.ja.md) / [English](current-process-browser.md)

- [指摘の採否と修正](findings.ja.md) / [English](findings.md)

## 基準検証

| 固定した製品コードの検査 | 結果 |
| --- | --- |
| `repoctl check` | 提供された日本語監査 Plan のメタデータ欠落修正後に成功。最初の文書エラーは Plan に残した。 |
| `go test -race ./...` | 全 package で成功。 |
| `repoctl test-integration` | 実 Docker とタグ付き検証に成功。process/Compose 共存も含む。 |
| Linux Browser native と race | sandbox 有効で成功、10.130秒。 |
| 固定 master の native CI | Verify34290359477 と Browser native34290359439 が Windows/macOS/Linux で成功。修正後の候補版 CI は別途必要。 |
| 実 Podman 並行リースと Docker 共存 | 明示した integration flag と導入済み podman-compose で成功、126.631秒。 |
| 実 Android Emulator 並行リース | 専用の一時 template home で独立リースと手動終了後の照合に成功、48.587秒。 |
| Flutter Android backend lease | 342.51秒で失敗。両 build は完了したが runtime readiness が期限切れ。両 Emulator ログで tmpfs 空き不足（必要12GiB、空き約10GiB）を確認。成功扱いにしない。UI observer も同様に332.31秒で失敗。テスト上限を変えず、固定した隔離 clone とディスク上の一時保存先で再実行中。 |
| 隔離 clone の `release-verify` | 6対象を2回生成し8ファイルが完全一致。Linux 展開後 smoke も成功。tag は検証用の隔離 clone だけに作成。 |

CLI のタグと名前を推測した2回は `no tests to run` であり、検証実績に含めない。正しい Podman と process integration は上表に記載した。Android template は未導入のイメージ種別で最初に失敗し、その後導入済みイメージで成功した。開発者固有の SDK/Flutter パスは記載しない。
