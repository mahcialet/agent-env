---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/product-specs/agent-env-mvp.md
source_sha256: 001a2af935f484f5e04adfcc15ad46c99e566ca5586fc629bbacb12e38b11b77
---

[English（正本）](agent-env-mvp.md)

# MVP 仕様

Compose を使う CLI とリポジトリ harness は実装済みです。この文書は当初の必須範囲と受け入れ基準を保持します。各プラットフォームと review gate の完了は、[実装計画](../exec-plans/completed/agent-env-mvp.md)の証拠で追跡します。現在のコマンド、field、制約、復旧動作は [CLI 契約](cli-contract.ja.md)と [manifest リファレンス](manifest-v1.ja.md)で説明します。

1 つ以上の Git リポジトリを固定 commit から、分離され、破棄可能で、検査可能な environment lease に materialize する再利用可能な CLI `agent-env` を実装します。

リポジトリまたは workspace は、起動方法を `.agent-env.yaml` に記述します。`agent-env` は共通のコントロールプレーン責務を扱います。

- ref を不変の commit ID に解決する
- 分離した Git worktree を作成する
- 要求された起動グループから、必要最小限のコンポーネント依存関係の閉包を解決する
- 一意の project 名を持つ分離した Docker Compose project を起動する
- lease、source、component、runtime、command、event の記録を SQLite に永続化する
- レジストリ状態を実際の Git/Docker 状態と照合し、正確な `list` と `show` を提供する
- named test を実行し、出力を証拠として収集する
- 環境を安全に destroy、garbage-collect、quarantine する
- agent 向けに安定した人間可読/機械可読の出力を提供する
- 単一の Go codebase から Windows、macOS、Linux 上でネイティブ動作する
- リポジトリ固有の開発 harness により、coding agent が `agent-env` 自体を理解し操作できるようにする
- chat 履歴に依存せず、設計意図、進捗、検証証拠、繰り返し得られる運用上の教訓を version 管理された artifact に保持する

リポジトリ開発 harness と製品の runtime manifest は別物です。

```text
リポジトリ開発 harness
  AGENTS.md + ARCHITECTURE.md + docs/ + repoctl + CI
  -> Codex に agent-env 自体の理解・変更・検証方法を伝える

対象リポジトリの runtime harness
  .agent-env.yaml
  -> agent-env に対象リポジトリ/workspace の materialize 方法を伝える
```

実装した workflow:

```text
agent-env plan   ./control-repo --stack api
agent-env create ./control-repo --stack api --ref refs/pull/3/head
agent-env list
agent-env test   <lease-id> api-smoke
agent-env logs   <lease-id>
agent-env destroy <lease-id>
```

CLI は worktree 位置、Compose file 位置、Compose project 名、一時 port、リソース削除順序などの実装詳細を、呼び出し元 agent から隠さなければなりません。

長期的な対象は Compose より広いものです。

```text
repository/workspace definition
              |
              v
          agent-env
              |
      environment lease
       /      |       \
 worktrees  runtimes  evidence
             /   \
        Compose  Android Emulator
```

ただし最初の実装では、不完全なアダプターの寄せ集めではなく、Compose を使った信頼できる一連の機能を端から端まで完成させなければなりません。

---

## 範囲

## 必須実装

1. リポジトリ固有 harness の bootstrap: 簡潔な `AGENTS.md`、`ARCHITECTURE.md`、索引付き `docs/`、`docs/PLANS.md`、active ExecPlan、初期 ADR、クロスプラットフォームの `tools/repoctl`。
2. Go CLI skeleton と version コマンド。
3. `AGENT_ENV_HOME` override に対応した、プラットフォーム別の状態パス解決。
4. SQLite database、migration、repository、lease、source、component、resource、event、command run、artifact。
5. 厳密な検証付き `.agent-env.yaml` 解析。
6. コンポーネント依存関係解決と stack planning。
7. 複数のローカル Git source repository と、不変の ref-to-commit 解決。
8. review lease 用の detached worktree materialize。
9. 一意の project 名と選択 service を持つ Compose v2 runtime adapter。
10. Compose config の展開、digest 計算、基本 policy 診断。
11. 補償とイベント記録を伴う create saga。
12. `list`、`show`、`renew`、`destroy`、`reconcile`、`gc`、`doctor`。
13. log/証拠収集付き named test 実行。
14. 人間向け table 出力と安定した JSON 出力。
15. unit test と fixture による integration test。
16. クロスプラットフォームの build/test CI。
17. README、`ARCHITECTURE.md`、manifest 文書、セキュリティ上の限界、Windows 向け注記。
18. `repoctl docs-check`、`generated-check`、安定した診断 code を持つ初期 `arch-check`。
19. リポジトリ harness を呼び出し、active plan、文書 index、生成 schema、architecture check の整合性を検証する CI。

## 一連の機能が安定した場合に実装すべきもの

- 単純な Compose repository に対する `agent-env init` の manifest 候補生成
- 動的 loopback endpoint 公開用の生成 Compose override
- owner による `--mine` filter
- 軽量な Compose service health/readiness check
- 各 lease directory 以下の environment descriptor JSON
- 稼働 service が実際に使った image ID/digest の記録

## 最初の MVP では明示的に範囲外

- Android Emulator lifecycle
- Flutter APK build/install
- browser/CDP 制御
- UI snapshot コマンド
- リモート Git 資格情報管理
- GitHub/GitLab provider 固有の PR 省略記法
- 信頼できない fork の sandbox 化
- ローカル OCI registry 管理
- image の自動 promotion
- environment checkpoint/clone
- 稼働 stack の拡張/縮小
- 汎用の長時間稼働 host-process adapter
- 分散/複数ホストでの lease 調停
- GUI/TUI

---

## 受け入れ基準

## 基本動作

1. `api` と `dashboard` コンポーネントを持つ fixture repository の検証が成功する。
2. `plan --stack api` は API の依存関係閉包だけを解決する。
3. `plan --stack dashboard` は API と Dashboard を決定的なトポロジカル順序で解決する。
4. 不正な cycle と未知の参照は、対処可能な診断とともに失敗する。
5. `create` はランタイム起動前に、要求された正確な ref と解決済み commit を記録する。
6. 同じ repository と commit からの 2 つの同時 lease は、異なる worktree と Compose project 名を受け取る。
7. 片方の lease を destroy しても、他方の container、volume、network、worktree を変更しない。
8. `list --output json` は両 lease を source、stack、component、desired state、observed state とともに返す。
9. Compose project を手動停止または削除した場合、続く list/reconcile は lease を ready のままにせず degraded にする。
10. worktree 作成後の create 失敗は補償削除を起動する。削除も失敗した場合、lease は quarantined として見える状態で残る。
11. 追跡対象が dirty な worktree を GC が黙って削除しない。
12. `--apply` のない `gc` は何も削除しない。
13. named test は出力をストリーム配信し、exit code を記録し、stdout/stderr の証拠を保存する。
14. 複数のローカル repository source を解決し、その commit の組が `show` と JSON 出力で見える。

## クロスプラットフォーム動作

15. CLI は少なくとも次の対象で `CGO_ENABLED=0` により compile できる。

    ```text
    windows/amd64
    darwin/amd64
    darwin/arm64
    linux/amd64
    linux/arm64
    ```

16. unit test が Windows、macOS、Linux の CI runner で成功する。
17. 空白と Unicode を含む path をテストする。
18. Windows で Bash を要求する test がない。
19. 空白や引用符を含むコマンド引数が、各 OS の実行で往復しても保たれる。
20. `doctor` は `git`、`docker`、Compose v2 の欠如を、panic や誤った成功なしに報告する。

## 文書と安全性

21. README は、環境分離が悪意あるコードの sandbox ではないことを明示する。
22. `ARCHITECTURE.md` は低水準の実装詳細を重複させず、lease/source/runtime/reconciliation の境界を説明する。
23. `.agent-env.yaml` の schema と例を文書化する。
24. 破壊的コマンドについて dry-run、force、quarantine の動作を文書化する。
25. 延期した Android/browser/registry 機能は roadmap 項目として文書化し、実装済みとして示さない。
26. `AGENTS.md` は 150 行以下の案内図として機能し、参照する全リポジトリパスが存在する。
27. `docs/exec-plans/active/agent-env-mvp.md` は living plan の必須セクションを含み、実装中を通じて現在の進捗と次の行動を正確に示す。
28. すべての設計文書、製品仕様、ADR はローカル index から見つけられる。意図的に index から外した file により `repoctl docs-check` が対処可能な診断で失敗する。
29. `go run ./tools/repoctl check` は Windows、macOS、Linux で Bash、Make、PowerShell を必須とせずに動作する。
30. migration が存在する状態で、`repoctl generated-check` が `docs/generated/db-schema.md` に意図的に加えたずれを検出する。
31. `repoctl arch-check` は少なくとも 1 つの fixture または合成した禁止依存を検出し、期待する修正方向を説明する。
32. commit された active ExecPlan とリポジトリ文書だけで、新しい Codex 実行がこの chat を参照せずに、branch、現在の milestone、必要な command、受け入れ動作、復旧経路を特定できる。
33. 完了時に ExecPlan を outcomes/retrospective の記載付きで `docs/exec-plans/completed/` に移す。過去 handoff の由来を commit する場合は `docs/references/` 以下に保持する。

---
