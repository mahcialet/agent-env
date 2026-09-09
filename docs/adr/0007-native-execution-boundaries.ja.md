---
status: accepted
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/adr/0007-native-execution-boundaries.md
source_sha256: 05dd7ecd80718af8c7bee8c9b0e6e40fa40e007acd28754017249f37e9e8a8fb
---

# Native実行パスとWindows/WSL interopの範囲を定める

[English](0007-native-execution-boundaries.md)

## 背景

Windowsのnative診断では、333文字の作業ディレクトリからGo子processもGitも起動できず、
拡張パスprefixでも失敗した。Gitの長いパス設定も早期の`-C`処理には適用されない。
一般のファイルAPIの長いパス対応だけではprocess起動の対応を証明できない。
ユーザーは、プラットフォームの制限を暗黙にまたいだり永続的な実行用aliasを追加したりせず、
WSL2も含めて安全な互換性範囲に収めることを求めた。

WSLはprocess所有権の面ではLinuxである。Linuxのprocess group/identityによるcleanupでは、
interopで起動したWindows子孫の不在を証明できない。DrvFS/9p上の共有stateについても、
本repositoryの耐久性の前提に対する検証がない。破損や実際の孤児processを再現したわけではない。

## 決定

Windowsの解決後の実行ディレクトリを240 UTF-16単位以内とし、MAX_PATHに余裕を持たせる。
派生するsource/worktree/runtime/test/probeの配置を予約・展開前に検査し、process起動前にも
再検査する。非対応のパスは対処方法を示すエラーで拒否する。native Linux/macOSの深いパスの
動作は維持する。Windowsのglobal設定、8.3名、junctionを必須にしない。

WindowsとWSLには別々のnative workerとstate rootを使う。kernel/filesystem/mountの観測と
既存の祖先パスの解決により、DrvFS/9p上のWSL stateをディレクトリ・DB作成前に拒否する。
Windowsは既知のWSL UNC上のstateと実行パスを拒否し、拡張表記や解決後のaliasも検査する。
この規則により、任意のUNC共有や読み取り専用source mountを新たに拒否するものではない。

Windows以外では直接起動するPEバイナリを、Windowsでは`wsl.exe`の起動を、process起動と
detached出力作成前に拒否する。`.exe`という拡張子ではなく形式を検査する。
source bundleのGitコマンドにも同じ検査を適用する。信頼するscriptは単一OS内での実行に責任を持つ。
これはsandboxではなく、全子孫、wrapper、改名したbridge、並行して置換された実行ファイルについて
証明するものでもない。

## 代案と影響

OSの長いパス設定だけに依存する方法ではnative検証を満たせなかった。一時junctionは削除後に
無効なGit worktree参照を残し得る。永続的なaliasには、source/runtimeをまたぐ所有権の永続化、
再起動時の復元、cleanupが必要となり、symbolic linkを必須にしない規則も変更することになる。

利用者はhome/repository配置を短くする必要がある場合がある。ユーザー合意により、従来のWindowsの
深いパスで無条件に成功する期待を、作用前に明示的に拒否する証拠へ変更する。Windows以外の深いパスは
引き続き成功を検査する。WSL mount検出は観測を注入するテストで保守的に検証する。
実WSL2のmount/interop受け入れは未実施であり、これらのテストから実行済みとは扱わない。
証拠は[PORTABILITY](../PORTABILITY.ja.md)と
[ExecPlan](../exec-plans/completed/multi-host-control-plane.ja.md)を参照する。
