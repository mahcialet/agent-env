---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/SECURITY.md
source_sha256: 6a49fed49d58dba821926d397b3a8b73874621773ed4d120ff55dd0b474a268c
---

[英語版（翻訳元）](SECURITY.md)

# セキュリティと信頼

環境リースは名前、worktree、ライフサイクルの所有権を隔離します。**悪意あるコードを封じ込めるsandboxではありません。** リポジトリのDockerfile、Compose build、テスト、パッケージhook、コマンドprobeは、それぞれのツールが持つ権限でコードを実行できます。信頼できるリポジトリを使うか、信頼できないコードには別途管理する外側のsandboxを用意してください。

## マニフェストの権限

planとcreateは、既定で制御用checkoutの`.agent-env.yaml`を読みます。`--manifest`は信頼するマニフェストパスを明示的に選択し、正規化スナップショットとdigestをリースに保存します。ソースrefが選ぶのは固定したruntime/テストのソース内容であり、対象revisionのマニフェストに黙って差し替えるものではありません。マニフェストと、それが実行するコードの両方の変更をレビューしてください。信頼したbase/PR overlayのマージとリモート認証情報管理は今後の課題です。

ownerラベルと`--mine`は助言的なフィルターです。ローカル状態ディレクトリと選択したcontainer engineにアクセスできる人は、対応するホスト権限を持ちます。分散認証や敵対的な複数ユーザーの隔離はありません。

## 組み込みホストポリシー

現在のCLIは、TTL 4時間、最大TTL 24時間、有効な予約数の上限8件という組み込みの既定値を使います。quarantinedのリースとcleanupが不完全なリースは予約を保持します。設定可能なホストポリシーファイルと、別個の最大並列create設定は未実装です。

起動前に正規化Compose設定を検査し、privileged container、host networking、固定コンテナー名、固定公開ホストポート、Docker socketへのアクセス、device passthrough、安全でないmountを確認します。bindパスはsymlink解決後も含め、割り当てたソースroot内に収まる必要があります。外部network/volume、選択したリソースのグローバル共有名、安全でない・カスタムのvolume driverやdriver optionは拒否します。これらの検査は偶発的なホストアクセスと衝突を減らしますが、Docker buildやリポジトリコマンドを信頼できるものにするわけではありません。

不変の実行スナップショットに入るのは、選択したサービスと、そこから到達可能なリソース定義だけです。所有権ラベル、一意のプロジェクト、記録したproviderとengineの識別情報、設定digestを保持します。実行とcleanupでは、保存設定と観測したリソース識別情報を検証します。未選択の名前付きリソースが、巻き添えでcleanup対象になることはありません。

Podmanは、正規化YAMLをcanonical JSONへ変換して同じpolicyを適用します。
pod作成と、どの階層にある`x-podman*`拡張も拒否します。
対応するmount型は`bind`、`volume`、`tmpfs`です。Podman固有の`glob`や未model化の型は
明示的に拒否します。対応する`network_mode`は省略・空文字列、`bridge`、`none`です。
`host`は共通policyで拒否し、`ns:`、`pasta`、`slirp4netns`など未model化のmodeも、
ホストへのアクセス範囲を広げず拒否します。projectの`.env`には、予約済みの
`PODMAN_*`、`CONTAINER_*`、`AGENT_ENV_PODMAN_*`、`COMPOSE_*` keyを設定できません。
継承したrouting制御を除去し、native bridgeが子の呼出しを記録済みengineへ固定します。
Docker互換labelだけではPodmanの所有を証明できず、native project labelと正確なresource
識別情報が必要です。engineの構成fingerprintでは、同じ構成を再作成するその場の初期化を
検出できないため、resourceの所有確認は引き続き必要です。

## 認証情報と証拠

プロセス環境全体をSQLiteや証拠へ出力することはありません。名前付きテストは`${env:NAME}`でホストの値を明示的に参照できます。認証情報らしいテスト環境キーには、マニフェストのリテラル値ではなく、正確なホスト変数参照を使う必要があります。認識された継承認証情報の値が正規化マニフェスト内の別の場所にリテラルで現れる場合も、予約前に拒否します。展開した環境は実行入力として使いますが、記録するargv、streamログ、コピーする成果物には、設定済みの秘密値と認識された継承秘密値に基づく伏字処理を適用します。

これは既知の認証情報に対する値ベースの伏字処理であり、万能な秘密情報検出器ではありません。ツール内で生成された認証情報、encodeや変換された値、無関係な機微情報は認識できない場合があります。共有前に成果物をレビューしてください。既知のHugging Faceのboolean制御flagである`HF_HUB_DISABLE_IMPLICIT_TOKEN`は認証情報の値として扱いません。

解決済みのCompose環境項目に含まれる認証情報と、認識された継承認証情報の値は、実行スナップショットを保存する前に拒否します。実行設定を黙って伏字化し、振る舞いを変えることはありません。コンテナー内の絶対パスを指定する`*_FILE`参照を含め、container secret fileを優先してください。argv、URL、label、Dockerfile、任意のマニフェスト項目に認証情報のリテラルを置かないでください。これらは対応する秘密情報伝達手段ではありません。ローカル状態ディレクトリと対象リポジトリ自体の出力も保護してください。

## cleanupの境界

cleanupは削除前に、固定ソースとruntimeの所有権を検証します。追跡対象が変更されたworktree、リソース識別情報の不一致、不確定なcleanupは、リースをquarantinedにします。`destroy --force`はbinary diffを保持した後に限り追跡対象の編集破棄を許可し、曖昧な所有権は上書きしません。管理対象worktree内の追跡対象外のbuild/test出力は、通常のcleanupで破棄可能です。

`gc`は既定でdry-runです。`gc --apply`は明示的な適用であり、quarantinedまたは処理中のリースを除外します。孤立リソースの観測は、一括Docker/Podman/Git cleanupの許可にはなりません。記録された操作lockは、協調するagent-envプロセス間のライフサイクル操作の競合を防ぎますが、ユーザーや無関係なプロセスによるGit、container engine、ファイルシステムの直接変更は防ぎません。

## Androidホストの信頼

インストール済みSDKツール、不変のsystem image、AVDテンプレートは、信頼するホスト入力です。テンプレートはhardware設定を提供します。各リースは新しい専用の書き込み可能状態で開始し、テンプレートのuserdataやsnapshotを共有しません。アダプターは安全でない書き込みパス、symlink、テンプレートlockを拒否します。console認証は有効のまま保ち、そのtokenはローカルで読み、レジストリmetadataには保存しません。ADBはローカルサーバーを明示的に指定します。ホストの直接変更は所有権を無効化し、quarantinedを引き起こす場合があります。このリースは敵対的なSDKツールやユーザーを隔離しません。

`127.0.0.1:5037`のサーバーは共有ホスト状態です。Emulator起動またはADB boot問い合わせの前に、アダプターは直接`host:version`を要求し、SDKクライアントのプロトコルバージョンと比較します。不一致時に既存サーバーをkillし得るクライアントを起動することなく、不一致や不正な応答を拒否します。サーバーがなければ、SDKのdetached起動経路で別途起動し、診断を保持できます。そのプロセス識別情報はリース所有権を与えるものではありません。補償、destroy、GCはグローバルサーバーを停止・置換しません。継承したADB routing変数とserial変数を解除し、boot問い合わせは予約済みserialを明示します。これらの保護は同時発生するホスト変更を締め出したり、敵対的なローカルサーバーを信頼できるものにしたりしません。

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

process runtimeは宣言したnative実行ファイルをhost userの権限で動かします。固定source、
分離した状態、予約loopback portは所有と衝突の制御であり、sandboxではありません。
command自身がloopbackへbindする必要があります。agent-envは信頼するrepository codeの
任意のnetwork/file system作用を防ぎません。cwd/source相対実行ファイルの閉じ込めはsymlinkを
解決して確認します。PATH解決/digestはhost toolを固定せずに証拠を記録します。

process環境変数値にも名前付きtestと同じ明示的な`${env:NAME}` secret参照規則を適用します。
展開した認証情報は一時的な起動入力とし、永続command/環境変数fieldには参照を残します。
native stdout/stderr fileは専用の未加工出力であり、applicationが出したsecretを含むことが
あります。状態root、起動receipt、起動前redaction証拠は非公開とし、通常のartifactとして
公開しないでください。

CLIのlog読み取りは上限付きで、独立した起動前`redaction.json`内のsecret fingerprintを
使ってredactionします。host変数を後で削除・変更した場合や、起動後の識別情報receiptの保存に
失敗した場合も同じ値を隠します。起動済みprocessのredaction証拠が
欠落、不正、不一致の場合は出力を拒否します。fingerprintは平文値を保存しませんが暗号化では
なく、専用storageでの保護が必要です。未知のapplication secretや変換済みsecret値の検出は
保証しません。可変profile/databaseを自動で証拠へcopyせず、tree不在確認後にのみ削除します。
[process契約](product-specs/persistent-process-runtime.ja.md)を参照してください。
