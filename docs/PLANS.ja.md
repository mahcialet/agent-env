---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/PLANS.md
source_sha256: 726b3394a5ff92aea47dbfcb9688330d423acce7774c7ff5542d685f9fc69043
---

# ExecPlanの規則

[英語版（翻訳元）](PLANS.md)

ExecPlanには、実質的な作業を実行・レビュー・復旧するための情報をまとめます。
作業中の計画には現在の指示と受け入れの証拠を記録し、完了した計画は実装の履歴として残します。

## 実質的な作業を始める

実質的な変更には、専用ブランチと`docs/exec-plans/active/`配下でバージョン管理する
ExecPlanが必要です。想定ブランチと現在の作業はactive ExecPlanの記述に従い、両方を計画に明記します。

計画は単体で理解できる内容にします。会話をたどり直さなくても、実装担当者が正確なパス、
コマンド、結果、残る作業、判断、安全な復旧手順を確認できるようにしてください。
[完了済みMVP計画](exec-plans/completed/agent-env-mvp.md)は、過去の作業範囲と証拠の記録例です。

## 必須の構成

未完了のすべての計画と、新しいライフサイクル形式を使う完了済み計画に、次の節をMarkdownのレベル2見出しで記載します。
英語版には英語の見出しを使います。日本語版には対応する日本語名または英語名を使えます。
任意の名前では構造検査を通過しません。

| 英語の見出し | 日本語の見出し |
| --- | --- |
| Purpose / Big Picture | 目的 / 全体像 |
| Progress | 進捗 |
| Surprises & Discoveries | 想定外の発見 |
| Decision Log | 判断の記録 |
| Outcomes & Retrospective | 成果と振り返り |
| Context and Orientation | 背景と構成 |
| Plan of Work | 作業計画 |
| Concrete Steps | 具体的な手順 |
| Validation and Acceptance | 検証と受け入れ |
| Idempotence and Recovery | 冪等性と復旧 |
| Artifacts and Notes | 成果物と注記 |
| Interfaces and Dependencies | インターフェースと依存 |

## 実行中に証拠を更新する

永続的な計画のメタデータには、`status`、`owner`、`last_verified`を必須とします。
意味のある区切りごとに、次の内容を更新します。

- 進捗には日付付きの完了・未完了項目を使います。チェックは確認済みの完了を表し、予定には付けません。
- 正確なコマンドと結果を記録します。ローカルテスト、クロスビルド、実際のネイティブCIを区別し、
  失敗した検証と未解決のプラットフォーム上の不足も残します。
- 判断には日付、担当者の役割、根拠を記します。継続して適用する決定はADRへ反映します。
- 残る作業と安全な復旧手順を現状に合わせます。

## 完了して履歴へ移す

すべての受け入れ要件に直接の証拠があり、「成果と振り返り」を記入してから、計画をactiveから
completedへ移します。ただし、作業が指定したベースブランチへマージ済みであることも必須です。
実装が終わっていても未マージならactiveのままにします。`merge_commit`に完全なcommit SHAを記録し、
ベースの祖先に含まれることを確認してから移します。移動時はすべてのリンクを更新します。完了済み計画は削除せず履歴として
保持してください。過去の参考資料アーカイブは入力資料であり、現在の運用上の判断基準にはしません。

人が読む永続文書を追加・変更する実質的な作業では、ExecPlanを完了する前に対応する日本語訳を
含めなければなりません。更新を続けるExecPlan自身も対象です。両言語の計画を同じ
ライフサイクルのディレクトリに置き、移動時は両方のリンクを更新します。
例外は、移行前の歴史的計画として明示的に登録されたものに限ります。

[言語の方針](design-docs/bilingual-documentation.ja.md)に従い、実質的な構成変更では
英語、日本語、意味の一致をそれぞれ別にレビューします。完了前に`repoctl docs-check`を実行します。
更新確認用hashの一致だけで、翻訳の意味を確認したことにはできません。

## 状態と移行

`docs/exec-plans/`直下のディレクトリ名と計画の`status`を一致させます。

| 状態 | 意味と遷移 |
| --- | --- |
| `draft` | 提案段階。`promotion_criteria`を満たして明示的に昇格するとactiveになります。自動昇格も実行対象への選出も禁止です。 |
| `active` | 依存関係と競合の条件を満たせば実行できます。現在実行中という意味ではありません。 |
| `paused` | 実際の障害で進められない状態。`pause_reason`と`resume_when`を記録し、条件を満たしてから明示的に再開します。同時実行枠の待ち行列には使いません。 |
| `completed` | 受け入れ済みの作業がベースへマージされ、振り返りと両言語の計画が保存されています。 |
| `abandoned` | 意図的に中止した作業。`abandonment_reason`を残します。依存先として完了条件を満たすことはありません。 |

Draft、active、pausedの作業は明示的に中止できます。証拠は保持します。
計画を移動・改名してもIDは変えません。中止した作業の代替には新しいIDを割り当て、
影響する依存関係を明示的に更新して判断を記録します。子計画がすべて完了しても親は自動で
完了しません。親自身の統合、受け入れ条件、文書、振り返りを照合する必要があります。
子を待つ親は、別に実際の障害を記録していない限りactiveのままです。

未完了の計画はすべて新しい形式を使い、`plan_type`に暗黙の既定値は設けません。
ハーネスが正確なパスで固定した移行前の完了済み計画一覧に限り、既存の内容とメタデータを
保持します。この移行例外は、翻訳を免除する歴史的計画4件の例外とは別です。
新しい計画がフィールドを省略するだけで例外になることはありません。
新たに完了する計画には、新形式のメタデータと必須節を残します。

## 識別子とメタデータ

`status`、`owner`、`last_verified`に加え、次のフィールドを使います。

| フィールド | 規則 |
| --- | --- |
| `plan_id` | リポジトリ内で一意かつ不変。`EP-[A-Z][A-Z0-9]*-[0-9]{3,}`に一致させ、選んだ名前空間の未使用番号を割り当てます。 |
| `plan_type` | `implementation`、`review`、`human-validation`のいずれかを明示します。 |
| `base_branch` | Gitのベースブランチを明示します。マージと依存関係の証拠はこのブランチを基準に評価します。 |
| `priority` | 0以上の整数。小さい値を先に選びます。 |
| `workstreams`、`conflicts` | 論理的な作業領域名のリスト。競合は状態を変えずに同時選出を防ぎます。 |
| `blockers` | 任意の明示的な障害リスト。項目があれば選出しません。 |
| `branch` | 後述の規則で決まるブランチの任意の宣言。初回の`EP-OPS-001`だけは`feat/execplan-lifecycle-orchestration`を維持します。 |
| `parent` | 任意の親Plan ID。階層関係だけでは実行依存を課しません。 |
| `depends_on` | `plan_id`と任意の`satisfaction`を持つオブジェクトのリスト。充足条件は`merged`（既定）または`stacked`です。 |
| `merge_policy` | `automatic`、`guarded`、`manual`のいずれか。機械検証できる承認がなければ自動マージできません。 |

Draftの`promotion_criteria`は、空でない文字列を一つ以上含むリストです。Pausedの
`pause_reason`と`resume_when`、abandonedの`abandonment_reason`は空でない説明文字列です。
Completedには`merge_commit`として完全な16進commit SHAが必要です。
人による検証計画には`execution_mode: human-kick`を指定し、自動実行しません。
ライフサイクルの値は日英で一致させ、本文の説明を翻訳します。ExecPlanメタデータには
定義されたYAMLのリストとオブジェクトを使えます。通常文書のメタデータは引き続き1行のスカラーです。

参照先の欠落、重複、自己参照、循環を拒否します。Merged依存の充足には、依存先がcompletedで、
そのマージcommitが依存する計画のベースから到達可能であることが必要です。
`stacked`は明示的な例外です。前提ブランチのcommitが依存するブランチに含まれる証拠を確認し、
マージやPRの対象変更後にベースと対象の関係を再確認します。親子関係と実行依存は別々に検証します。

## 照会・選出・Gitでの追跡

ハーネスは決定的な結果を返す読み取り専用の照会を提供します。

```text
go run ./tools/repoctl plans list
go run ./tools/repoctl plans check
go run ./tools/repoctl plans graph
go run ./tools/repoctl plans ready
```

依存条件を満たし、障害がなく、作業領域が競合しないactiveの計画だけを選びます。
`priority`、`plan_id`の順に並べ、当初の実装同時実行数は1とします。
子のマージ後や中断からの復帰時には、リポジトリとGitの状態から再評価します。
照会はエージェントを起動せず、状態も変更しません。スケジューラdaemonも必須ではありません。

Plan IDを小文字にした接尾辞を使い、実装は`feat/ep-ops-001`、レビューは`fix/ep-ops-001`、
人による検証は`validate/ep-ops-001`とします。IDの切り詰めやタイトルへの置換は禁止です。
導入用の`EP-OPS-001`だけは、明示的な例外として`feat/execplan-lifecycle-orchestration`を使います。
対象の実装commitには`ExecPlan: EP-OPS-001` trailerを付け、PR本文にも同じ独立した行を置きます。
ブランチ削除後もcommit・PR履歴からIDを照会できます。過去のcommitを書き換えてtrailerを
追加しません。無関係な継承履歴を除き、現在の計画が提供するcommit範囲を検証します。

## レビューとマージの条件

`automatic`でも、機械検証できる条件をすべて満たした場合だけマージできます。
`guarded`と`manual`は明示的な人間の承認を必要とし、自動マージできません。
適用する方針は未レビューのPR変更ではなく、信頼するベースから取得します。
GitHubのブランチ保護とrulesetの制約にも従います。

現在のPR HEADに結び付いた信頼できる構造化承認、必須CI、該当するネイティブ・統合検証、
日英文書検査の成功が必要です。ブロックするレビュー、未解決Thread、明示的な障害がなく、
マージ・保存以外の受け入れ条件を完了していなければなりません。
マージ直前にベースとrulesetが最新か再確認します。後続commitがあれば以前の承認は無効です。
自由記述の`LGTM`などは承認として解釈しません。レビュー提供元から現在のHEADに対する
信頼できる機械可読の承認を取得できない場合は、guardedまたはmanualを維持して制約を記録し、
条件を緩めません。

マージ失敗時に結果が不明なら、再試行前にリモートのPR、マージ、ベース、検査状態を読み直します。
無条件の再試行や、要求送信だけを根拠にした完了扱いは禁止です。自動マージの証拠と残る手作業は
active計画に記録します。将来の親子連携、人による検証の事前確認・証拠コマンドは、該当する
マイルストーンの検証が通る前に提供済みと宣伝しません。機械的な事前確認やシナリオ実行の前にも、
人間による明示的な開始操作が必要です。

### コマンドの手順と現在の制限

remoteの進捗に基づいて判断する前に`git fetch origin`を行います。`plans`はlocalの参照を読み、
localの`master`がなければ`origin/master`を使えます。nativeのVerify jobは全履歴を取得し、
移植可能な文書検査とは別に`plans check`を実行します。

```text
go run ./tools/repoctl plans list
go run ./tools/repoctl plans check
go run ./tools/repoctl plans graph
go run ./tools/repoctl plans ready
go run ./tools/repoctl plans provenance --plan EP-OPS-001 --pr-body <file>
go run ./tools/repoctl plans gate --plan EP-OPS-001 --pr <number> --repo <owner/repository>
go run ./tools/repoctl plans replay --plan <completed-plan-path> --merge <full-sha> --pr <number>
```

`check`は全Planを削除したりbaseを変更したりしても、`master`の既存IDを保持するよう求めます。
merge証拠には、merge元のheadまたはsquash結果のcommitに、そのPlanを示す一意のtrailerが必要です。
treeにdraft Planを含むだけのcommitでは足りません。`ready`はstacked依存を利用側の実際のbranchで
確認し、branch作成前だけ明示された開始baseを使います。これらのコマンドはbranch作成、commit、
push、merge、archiveを自動実行しません。

`gate`は読み取り専用で、根拠が不足すれば拒否します。PRのbase commitから、任意のversion 1の
`.github/execplan-gates.json`を読みます。`policies`はPlan IDをキーとし、各値は
`tools/repoctl/plans_gate.go`の`PlanMergePolicy`に従います。今回の初回実装では信頼するreviewerや
checkの方針を設定しません。実際のadapterでは原子的なruleset保護と機械可読の受け入れ証拠を
まだ確認できないため、判定modelが完全な模擬証拠を許可できてもBLOCKEDを返し、guarded/manualの
手順を求めます。後から発火するauto-mergeも設定しません。replayはGitの事実と再現上の仮定を
区別し、任意のPR番号はGitHubで別途照合します。merge履歴だけでreview承認を証明しません。

人間検証のコマンドとscenario契約は[multi-host検証Plan](exec-plans/paused/multi-host-human-validation.ja.md)に
記載します。`preflight`は明示的な`--kick`、名前を指定したlocal実行ファイルとendpoint、
未使用の証拠ディレクトリを必要とします。`record`は操作担当の観測を記録し、runtimeのPASSを推測しません。
FINDINGには追跡中のdraft/active review Planを求め、BLOCKEDではpaused Planの更新が必要と記録します。
人間によるscenarioの実施、証拠の確認、Planへの反映は別途必要です。
