---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/SECURITY.md
source_sha256: 2c8713ed332784f61163508253a7fbcb3173a8193e4dfd374a7969869b0e0197
---

[英語版（翻訳元）](SECURITY.md)

# セキュリティと信頼

環境リースは名前、worktree、ライフサイクルの所有権を隔離します。**悪意あるコードを封じ込めるsandboxではありません。** リポジトリのDockerfile、Compose build、テスト、パッケージhook、コマンドprobeは、それぞれのツールが持つ権限でコードを実行できます。信頼できるリポジトリを使うか、信頼できないコードには別途管理する外側のsandboxを用意してください。

## マニフェストの権限

planとcreateは、既定で制御用checkoutの`.agent-env.yaml`を読みます。`--manifest`は信頼するマニフェストパスを明示的に選択し、正規化スナップショットとdigestをリースに保存します。ソースrefが選ぶのは固定したruntime/テストのソース内容であり、対象revisionのマニフェストに黙って差し替えるものではありません。マニフェストと、それが実行するコードの両方の変更をレビューしてください。信頼したbase/PR overlayのマージとremote認証情報の自動準備は今後の課題です。

### ローカルへのアクセスとホスト権限

ownerラベルと`--mine`は助言的なフィルターです。ローカル状態ディレクトリと選択したcontainer engineにアクセスできる人は、対応するホスト権限を持ちます。localモードはremote認証を提供せず、敵対的な複数ユーザーも隔離しません。明示的なremoteモードの認証は後述します。

## 組み込みホストポリシー

現在のCLIは、TTL 4時間、最大TTL 24時間、有効な予約数の上限8件という組み込みの既定値を使います。quarantinedのリースとcleanupが不完全なリースは予約を保持します。設定可能なホストポリシーファイルと、別個の最大並列create設定は未実装です。

### Composeの検査

起動前に正規化したCompose設定を検査します。対象はprivileged container、host networking、
固定コンテナー名、固定公開ホストポート、Docker socketへのアクセス、device passthrough、
安全でないmountです。

リソースの利用範囲についても、次の規則を適用します。

- bindパスは、symlink解決後も割り当てたソースroot内に収まる必要があります。
- 外部network/volumeと、選択したリソースのグローバル共有名は拒否します。
- 安全でない、またはカスタムのvolume driverとdriver optionは拒否します。

これらの検査は偶発的なホストアクセスと衝突を減らすためのものです。
Docker buildやリポジトリコマンド自体を信頼できるものにするわけではありません。

不変の実行スナップショットに入るのは、選択したサービスと、そこから到達可能なリソース定義だけです。所有権ラベル、一意のプロジェクト、記録したproviderとengineの識別情報、設定digestを保持します。実行とcleanupでは、保存設定と観測したリソース識別情報を検証します。未選択の名前付きリソースが、巻き添えでcleanup対象になることはありません。

### Podmanの制限

Podmanでも、正規化したYAMLをcanonical JSONへ変換して同じpolicyを適用します。
さらに次の範囲に制限します。

| 設定 | 対応範囲と拒否条件 |
| --- | --- |
| podと`x-podman*`拡張 | どの階層にあっても拒否 |
| mount型 | `bind`、`volume`、`tmpfs`のみ。`glob`など未対応の型は拒否 |
| `network_mode` | 省略・空文字列、`bridge`、`none`のみ。`host`は共通policyで拒否し、`ns:`、`pasta`、`slirp4netns`など未対応のmodeも拒否 |
| projectの`.env` | 予約済みの`PODMAN_*`、`CONTAINER_*`、`AGENT_ENV_PODMAN_*`、`COMPOSE_*`キーは設定禁止 |

継承した接続先の制御値は除去し、native bridgeが子プロセスの呼び出しを記録済みengineへ固定します。
Docker互換labelだけではPodmanの所有権を証明できません。Podman自身のproject labelと、正確な
リソース識別情報が必要です。

engineの構成fingerprintでは、初期化後に同じ構成が再作成されたことを検出できません。
リソースの所有権確認は引き続き必要です。

## 認証情報と証拠

プロセス環境全体をSQLiteや証拠へ出力することはありません。名前付きテストは`${env:NAME}`でホストの値を明示的に参照できます。認証情報らしいテスト環境キーには、マニフェストのリテラル値ではなく、正確なホスト変数参照を使う必要があります。認識された継承認証情報の値が正規化マニフェスト内の別の場所にリテラルで現れる場合も、予約前に拒否します。展開した環境は実行入力として使いますが、記録するargv、streamログ、コピーする成果物には、設定済みの秘密値と認識された継承秘密値に基づく伏字処理を適用します。

### 伏字処理の限界

これは既知の認証情報に対する値ベースの伏字処理であり、万能な秘密情報検出器ではありません。ツール内で生成された認証情報、encodeや変換された値、無関係な機微情報は認識できない場合があります。共有前に成果物をレビューしてください。既知のHugging Faceのboolean制御flagである`HF_HUB_DISABLE_IMPLICIT_TOKEN`は認証情報の値として扱いません。

### Composeへ渡す秘密情報

解決済みのCompose環境項目に含まれる認証情報と、認識された継承認証情報の値は、実行スナップショットを保存する前に拒否します。実行設定を黙って伏字化し、振る舞いを変えることはありません。コンテナー内の絶対パスを指定する`*_FILE`参照を含め、container secret fileを優先してください。argv、URL、label、Dockerfile、任意のマニフェスト項目に認証情報のリテラルを置かないでください。これらは対応する秘密情報伝達手段ではありません。ローカル状態ディレクトリと対象リポジトリ自体の出力も保護してください。

## cleanupの境界

cleanupは削除前に、固定ソースとruntimeの所有権を検証します。追跡対象が変更されたworktree、リソース識別情報の不一致、不確定なcleanupは、リースをquarantinedにします。`destroy --force`はbinary diffを保持した後に限り追跡対象の編集破棄を許可し、曖昧な所有権は上書きしません。管理対象worktree内の追跡対象外のbuild/test出力は、通常のcleanupで破棄可能です。

`gc`は既定でdry-runです。`gc --apply`は明示的な適用であり、quarantinedまたは処理中のリースを除外します。孤立リソースの観測は、一括Docker/Podman/Git cleanupの許可にはなりません。記録された操作lockは、協調するagent-envプロセス間のライフサイクル操作の競合を防ぎますが、ユーザーや無関係なプロセスによるGit、container engine、ファイルシステムの直接変更は防ぎません。

## Androidホストの信頼

インストール済みSDKツール、不変のsystem image、AVDテンプレートは、信頼するホスト入力です。テンプレートはhardware設定を提供します。各リースは新しい専用の書き込み可能状態で開始し、テンプレートのuserdataやsnapshotを共有しません。アダプターは安全でない書き込みパス、symlink、テンプレートlockを拒否します。console認証は有効のまま保ち、そのtokenはローカルで読み、レジストリmetadataには保存しません。ADBはローカルサーバーを明示的に指定します。ホストの直接変更は所有権を無効化し、quarantinedを引き起こす場合があります。このリースは敵対的なSDKツールやユーザーを隔離しません。

`127.0.0.1:5037`のサーバーは共有ホスト状態です。Emulator起動またはADB boot問い合わせの前に、アダプターは直接`host:version`を要求し、SDKクライアントのプロトコルバージョンと比較します。不一致時に既存サーバーをkillし得るクライアントを起動することなく、不一致や不正な応答を拒否します。サーバーがなければ、SDKのdetached起動経路で別途起動し、診断を保持できます。そのプロセス識別情報はリース所有権を与えるものではありません。補償、destroy、GCはグローバルサーバーを停止・置換しません。継承したADB routing変数とserial変数を解除し、boot問い合わせは予約済みserialを明示します。これらの保護は同時発生するホスト変更を締め出したり、敵対的なローカルサーバーを信頼できるものにしたりしません。

### ホストが同時に変更された場合

ADBプロトコル検査は現在の共有サーバーの観測であり、ホスト全体のlockではありません。観測からSDKコマンドまでの間に、外部からの直接的なサーバー置換やSDKバージョン変更が起こると、SDK自身のversion処理との競合は依然として起こり得ます。リースが有効な間は互換性のある共有サーバーを安定させてください。ツール間のサーバー置換はagent-envの調整範囲外です。

## リリースの整合性

Git tag がリリースバージョンを識別します。checksum はバイト列の変化を検出しますが、
署名や独立した出所証明にはなりません。署名、notarization、attestation は初回の
アーカイブリリースの対象外です。アーカイブと manifest は信頼する配布経路から取得してください。
静的検証は悪意ある実行ファイルを隔離する sandbox ではなく、ネイティブ smoke test は候補を実行します。

リリース作成は、追跡対象や index の変更、ignore 対象以外の未追跡ファイル、
バージョン tag の欠落・曖昧さ、HEAD や要求バージョンの不一致を拒否します。
ignore 対象のビルド出力は作業ツリーの変更に数えません。作成時は専用の一時領域を使い、
既存の出力ディレクトリを拒否します。検証は、想定外のアーカイブ要素、安全でないパスやリンク、
実行ファイルやメタデータの digest 不一致を拒否します。manifest のフィールドに
ホストパス、一時パス、認証情報を含めてはいけません。リリースコマンドは呼び出し元や公開 Git ref を変更しません。preview 検証のテスト tag は
専用 clone 内だけに作成します。公開の開始は maintainer が管理します。
ビルドは厳密なコミットの専用 checkout を使い、ignore 対象のソースや
index フラグで隠れたローカル編集を除外します。

runtime の前提ツールは引き続き信頼するホスト入力であり、リリース CLI はインストールしません。
任意の Android UI helper は外部でビルドし、現在のリリースには埋め込みません。
汎用資産の digest 検査は破損を検出するもので、既存のローカルな信頼境界を越えて
悪意あるホスト変更を防ぐものではありません。

## 常駐host processの信頼とlogs

process runtimeは、宣言されたネイティブ実行ファイルをホストユーザーの権限で動かします。
固定ソース、専用状態、予約したloopbackポートは所有権の管理と衝突防止のためのもので、sandboxではありません。
loopbackへのbindはコマンド自身が行う必要があります。agent-envは、信頼するリポジトリのコードによる
任意のネットワーク・ファイルシステムへの作用を防ぎません。

cwdとソース相対の実行ファイルは、symlinkを解決してソース内に収まることを確認します。
PATHの解決先とdigestは証拠として記録しますが、ホストツールを固定するものではありません。

### 起動入力と非公開の出力

processの環境変数値にも、名前付きテストと同じ`${env:NAME}`による明示的な秘密値参照の規則を適用します。
展開した認証情報は一時的な起動入力に使い、永続化するコマンド・環境変数フィールドには参照を残します。

ネイティブstdout/stderrファイルは未加工の専用出力であり、アプリケーションが出した秘密値を含む場合があります。
状態root、起動receipt、起動前の伏字処理用証拠は非公開にし、通常の成果物として公開しないでください。

### ログの伏字処理と限界

CLIは上限を設けてログを読み、起動前に独立して保存した`redaction.json`の秘密値fingerprintを使って
伏字化します。後からホスト変数を削除・変更した場合や、起動後の識別情報receiptを保存できなかった場合も同じです。
起動済みプロセスの伏字処理用証拠が欠落、不正、または不一致なら、出力を拒否します。

fingerprintは平文値を保存しませんが、暗号化ではありません。保存領域の保護が必要です。
未知のアプリケーションの秘密値や、変換済みの秘密値を認識できる保証はありません。
書き換え可能なprofileやdatabaseは証拠へ自動コピーせず、ツリー全体の不在を確認してから削除します。
[process契約](product-specs/persistent-process-runtime.ja.md)を参照してください。

## browser profileと証拠

Browser/CDPは、リースが所有するネイティブプロセスへ明示的に結び付けた機能です。
アダプターはCDPによる作用の前に、記録したroot PID、専用profile、予約したloopbackポートを検証します。
ユーザーの既存browserは探索しません。HTTP discoveryではredirectとproxyを禁止します。

これらの検査はリース間の誤接続を防ぎますが、同じユーザー権限を持つ悪意あるサーバーによる
CDP応答の偽装は防ぎません。リポジトリが指定するbrowser実行ファイルは、信頼するホストコードです。

### 証拠に残るものと残らないもの

profileは非公開の書き換え可能な状態であり、自動収集する証拠には含めません。
各証拠の扱いは次のとおりです。

- semantic/DOM証拠からは、編集可能な値とパスワード値を除きます。入力テキストは一時的に扱い、操作メタデータでは伏字化します。
- network証拠にはheaderとbodyを保存せず、URLのqueryと認証情報を伏字化します。
- consoleの収集は接続期間内に限定し、認識した継承秘密値を伏字化します。
- screenshotは有効なPNGとして保存します。ただし画像や、認識できないpage/consoleのテキストから秘密が漏れる可能性があるため、成果物は非公開に保ちます。

型付き操作には、任意のJavaScriptやraw CDPを公開する経路はありません。
[browser契約](product-specs/browser-cdp-automation.ja.md)を参照してください。

## Remote の管理境界

[remote モード](product-specs/multi-host-control-plane.ja.md) は、信頼された一つの管理組織を前提にします。
証明書を事前に用意し、`control-plane enroll --certificate ... --role ...` で client/worker role を登録します。
worker の登録には `--host-id` も結び付けます。登録は controller の専用 local 状態を変更する操作であり、
remote の自己登録 endpoint ではありません。本番通信は `--controller`、`--tls-ca`、`--tls-cert`、`--tls-key` による
HTTPS 相互 TLS を使います。CA 署名だけで未登録の証明書や、client として使った worker 証明書は拒否します。
worker から接続するため、worker の受信用 listener や汎用 remote shell は公開しません。

### Remoteの機密データ

証明書の秘密鍵は保護したファイル入力として保持し、SQLite に記録しません。
commit 済み source bundle、operation payload、artifact は管理用の機密データとして扱います。
control-plane の保存領域は非公開ですが、保存時の暗号化は保証しません。
SHA-256 CAS key に呼出側の path を使わず、source object は 1 GiB、artifact は 64 MiB に制限します。
worker の source 検証は、runtime に作用する前に未 commit や未対応の source 形式を拒否します。
client の環境変数の秘密値は暗黙に転送せず、`${env:NAME}` は worker で解決します。
既存の repository の信頼、redaction、runtime の識別情報、host policy を worker でも適用します。

### 配置と操作の所有権

管理 tuple の検査で local mutation/force/GC と別 assignment の操作を拒否し、通常の registry Save では
管理 metadata の削除・置換をできなくします。暗黙の緊急管理引継ぎはありません。
返す loopback URL は worker-local であり、client から任意の host endpoint に接続する許可ではありません。
