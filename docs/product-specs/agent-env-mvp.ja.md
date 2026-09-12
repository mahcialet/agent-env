---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/product-specs/agent-env-mvp.md
source_sha256: 6acfd154d037d669e6c25500b38e214df9990179bbb4ff1da732d0918f57a128
---

[English（翻訳元）](agent-env-mvp.md)

# MVP 仕様

この文書は、最初の MVP の要件と受け入れ基準を残すものです。Compose を使う CLI と
開発用ハーネスは実装済みで、各 OS とレビューの検証記録は
[完了済み実装計画](../exec-plans/completed/agent-env-mvp.md)にあります。
後述の対象外一覧は最初の MVP の範囲を示しており、現在利用できる機能の一覧ではありません。

現在の使い方は [CLI 仕様](cli-contract.ja.md)、[マニフェスト仕様](manifest-v1.ja.md)、
[機能別の索引](index.ja.md)から確認してください。後から追加した機能はそれぞれの仕様で
扱い、この初期要件の記録を書き換えずに拡張しています。

## 製品の目的

再利用可能な CLI `agent-env` を実装します。1 つ以上の Git リポジトリを固定 commit から、分離され、破棄可能で、検査可能な environment リースに materialize します。

リポジトリまたは workspace は、起動方法を `.agent-env.yaml` に記述します。`agent-env` は共通のコントロールプレーン責務を扱います。

- ref を不変の commit ID に解決する
- 分離した Git ワークツリーを作成する
- 要求された起動グループのルートと、その直接・間接の全依存先コンポーネントだけを選ぶ
- 一意の project 名を持つ分離した Docker Compose project を起動する
- リース、ソース、コンポーネント、ランタイム、コマンド、event の記録を SQLite に永続化する
- レジストリ状態を実際の Git/Docker 状態と照合し、正確な `list` と `show` を提供する
- named テストを実行し、出力を証拠として収集する
- 環境を安全に destroy、garbage-collect、隔離する
- agent 向けに安定した人間可読/機械可読の出力を提供する
- 単一の Go codebase から Windows、macOS、Linux 上でネイティブ動作する
- リポジトリ固有の開発 harness により、coding agent が `agent-env` 自体を理解し操作できるようにする
- chat 履歴に依存せず、設計意図、進捗、検証証拠、繰り返し得られる運用上の教訓をバージョン管理された成果物に保持する

リポジトリ開発 harness と製品のランタイムマニフェストは別物です。

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

CLI はワークツリー位置、Compose ファイル位置、Compose project 名、一時ポート、リソース削除順序などの実装詳細を、呼び出し元 agent から隠さなければなりません。

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

### 必須実装

1. リポジトリ固有 harness の bootstrap: 簡潔な `AGENTS.md`、`ARCHITECTURE.md`、索引付き `docs/`、`docs/PLANS.md`、active ExecPlan、初期 ADR、クロスプラットフォームの `tools/repoctl`。
2. Go CLI skeleton とバージョンコマンド。
3. `AGENT_ENV_HOME` override に対応した、プラットフォーム別の状態パス解決。
4. SQLite database、migration、リポジトリ、リース、ソース、コンポーネント、リソース、event、コマンド run、成果物。
5. 厳密な検証付き `.agent-env.yaml` 解析。
6. コンポーネント依存関係解決とスタック planning。
7. 複数のローカル Git ソースリポジトリと、不変の ref-to-commit 解決。
8. review リース用の detached ワークツリー materialize。
9. 一意の project 名と選択サービスを持つ Compose v2 ランタイム adapter。
10. Compose config の展開、ダイジェスト計算、基本 policy 診断。
11. 補償とイベント記録を伴う create saga。
12. `list`、`show`、`renew`、`destroy`、`reconcile`、`gc`、`doctor`。
13. log/証拠収集付き named テスト実行。
14. 人間向け table 出力と安定した JSON 出力。
15. unit テストと検証用フィクスチャによる integration テスト。
16. クロスプラットフォームの build/test CI。
17. README、`ARCHITECTURE.md`、マニフェスト文書、セキュリティ上の限界、Windows 向け注記。
18. `repoctl docs-check`、`generated-check`、安定した診断 code を持つ初期 `arch-check`。
19. リポジトリ harness を呼び出し、active plan、文書 index、生成スキーマ、architecture check の整合性を検証する CI。

### 一連の機能が安定した場合に実装すべきもの

- 単純な Compose リポジトリに対する `agent-env init` のマニフェスト候補生成
- 動的ループバック接続先公開用の生成 Compose override
- owner による `--mine` filter
- 軽量な Compose サービス health/readiness check
- 各リースディレクトリ以下の environment descriptor JSON
- 稼働サービスが実際に使った image ID/digest の記録

### 最初の MVP では明示的に範囲外

- Android Emulator lifecycle
- Flutter APK build/install
- browser/CDP 制御
- UI スナップショットコマンド
- リモート Git 資格情報管理
- GitHub/GitLab プロバイダー固有の PR 省略記法
- 信頼できない fork の sandbox 化
- ローカル OCI registry 管理
- image の自動 promotion
- environment checkpoint/clone
- 稼働スタックの拡張/縮小
- 汎用の長時間稼働 host-process adapter
- 分散/複数ホストでのリース調停
- GUI/TUI

---

## 受け入れ基準

### 基本動作

1. `api` と `dashboard` コンポーネントを持つ検証用フィクスチャリポジトリの検証が成功する。
2. `plan --stack api` は API stack のルートと、その直接・間接の全依存先コンポーネントだけを選ぶ。
3. `plan --stack dashboard` は API と Dashboard を決定的なトポロジカル順序で解決する。
4. 不正な cycle と未知の参照は、対処可能な診断とともに失敗する。
5. `create` はランタイム起動前に、要求された正確な ref と解決済み commit を記録する。
6. 同じリポジトリと commit からの 2 つの同時リースは、異なるワークツリーと Compose project 名を受け取る。
7. 片方のリースを destroy しても、他方のコンテナー、ボリューム、ネットワーク、ワークツリーを変更しない。
8. `list --output json` は両リースをソース、スタック、コンポーネント、desired 状態、observed 状態とともに返す。
9. Compose project を手動停止または削除した場合、続く list/reconcile はリースを ready のままにせず degraded にする。
10. ワークツリー作成後の create 失敗は補償削除を起動する。削除も失敗した場合、リースは quarantined として見える状態で残る。
11. 追跡対象が dirty なワークツリーを GC が黙って削除しない。
12. `--apply` のない `gc` は何も削除しない。
13. named テストは出力をストリーム配信し、exit code を記録し、stdout/stderr の証拠を保存する。
14. 複数のローカルリポジトリソースを解決し、その commit の組が `show` と JSON 出力で見える。

### クロスプラットフォーム動作

15. CLI は少なくとも次の対象で `CGO_ENABLED=0` により compile できる。

    ```text
    windows/amd64
    darwin/amd64
    darwin/arm64
    linux/amd64
    linux/arm64
    ```

16. unit テストが Windows、macOS、Linux の CI runner で成功する。
17. 空白と Unicode を含むパスをテストする。
18. Windows で Bash を要求するテストがない。
19. 空白や引用符を含むコマンド引数が、各 OS の実行で往復しても保たれる。
20. `doctor` は `git`、`docker`、Compose v2 の欠如を、panic や誤った成功なしに報告する。

### 文書と安全性

21. README は、環境分離が悪意あるコードの sandbox ではないことを明示する。
22. `ARCHITECTURE.md` は低水準の実装詳細を重複させず、lease/source/runtime/reconciliation の境界を説明する。
23. `.agent-env.yaml` のスキーマと例を文書化する。
24. 破壊的コマンドについて dry-run、force、隔離の動作を文書化する。
25. 延期した Android/browser/registry 機能は roadmap 項目として文書化し、実装済みとして示さない。
26. `AGENTS.md` は 150 行以下の案内図として機能し、参照する全リポジトリパスが存在する。
27. `docs/exec-plans/active/agent-env-mvp.md` は living plan の必須セクションを含み、実装中を通じて現在の進捗と次の行動を正確に示す。
28. すべての設計文書、製品仕様、ADR はローカル index から見つけられる。意図的に index から外したファイルにより `repoctl docs-check` が対処可能な診断で失敗する。
29. `go run ./tools/repoctl check` は Windows、macOS、Linux で Bash、Make、PowerShell を必須とせずに動作する。
30. migration が存在する状態で、`repoctl generated-check` が `docs/generated/db-schema.md` に意図的に加えたずれを検出する。
31. `repoctl arch-check` は少なくとも 1 つの検証用フィクスチャまたは合成した禁止依存を検出し、期待する修正方向を説明する。
32. commit された active ExecPlan とリポジトリ文書だけで、新しい Codex 実行がこの chat を参照せずに、branch、現在の milestone、必要なコマンド、受け入れ動作、復旧経路を特定できる。
33. 完了時に ExecPlan を outcomes/retrospective の記載付きで `docs/exec-plans/completed/` に移す。過去 handoff の由来を commit する場合は `docs/references/` 以下に保持する。

---
