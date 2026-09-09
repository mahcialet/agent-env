---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/product-specs/browser-cdp-automation.md
source_sha256: 5e5ed5bdbe662710a111fa24e9332b7f492278b3789ff316e5b6d38114affd76
---

# Browser/CDP自動操作

[英語版（翻訳元）](browser-cdp-automation.md)

ブラウザーコマンドは、常駐プロセスリースが所有する明示的なChromium系ブラウザーを
観測・操作します。別のブラウザーの起動、外部ブラウザーへの接続、個人用プロファイルの再利用は
行いません。本仕様は、実装済みの設定、ページ選択、安全な入力、証拠保持を定めます。
プロセスの寿命は[常駐プロセス仕様](persistent-process-runtime.ja.md)を参照してください。
実装と各 OS での受け入れ状況は[完了ExecPlan](../exec-plans/completed/browser-cdp-automation.ja.md)に記録します。

## manifestと前提条件

必要なCDPメソッドを提供し、直接起動できるネイティブのヘッドレス Chromium系ブラウザーを
使います。検証にはChrome for Testingを推奨しますが、製品には同梱しません。
別のブラウザールートを別プロセスとして起動するランチャーは非対応です。CDPが報告するPIDは、リースが
所有するプロセスルートと一致する必要があります。無関係な基本機能コマンドにブラウザー、
Node、Python、Playwright、Selenium、ChromeDriverは不要です。
既存のプロセス実行ファイル・ソース相対パスの規則も適用します。

```yaml
version: 1
sources:
  app: {repository: ., default_ref: HEAD}
runtimes:
  browser-process:
    type: process
    source: app
    working_directory: .
    command:
      - chrome
      - --headless=new
      - --enable-automation
      - --user-data-dir=${runtime_dir}/profile
      - --remote-debugging-address=127.0.0.1
      - --remote-debugging-port=${port:cdp}
      - about:blank
    ports:
      cdp: {protocol: tcp}
browsers:
  web:
    type: chromium-cdp
    runtime: browser-process
    cdp_port: cdp
components:
  browser: {runtime: browser-process}
stacks:
  browser: {roots: [browser]}
```

### ブラウザーの宣言と必須スイッチ

`browsers`は省略できます。宣言する場合は空でない対応付けとし、各関連付けから既存の
`process` ランタイムと、そのランタイムの名前付きTCP ポートを参照します。
1つのランタイムに関連付けは1つだけです。名前にはOS間で使えるものを指定し、
大文字・小文字を無視した衝突を認めません。`cdp`というポート名だけでブラウザー機能は有効になりません。

上の5つのスイッチはすべて必須で、対象リポジトリが引数配列要素としてそれぞれ1回だけ、
記載どおりに宣言します。ブラウザー層は追加しません。保護対象スイッチの別表記・重複、
`--`終端、debugging pipe、プロファイル選択の上書きを拒否します。
`--user-data-dir`は`${runtime_dir}/profile`に固定し、任意の専用サブパスも認めません。
ポートと専用状態の管理は汎用プロセスランタイムが担当します。

## コマンドと選択

```text
agent-env create . --stack browser
agent-env browser capabilities <lease> --browser web
agent-env browser pages <lease> --browser web
agent-env browser page-create <lease> --browser web --url about:blank
agent-env browser navigate <lease> --browser web --page <page> --url http://127.0.0.1:8080/
agent-env browser snapshot <lease> --browser web --page <page>
agent-env browser dom-snapshot <lease> --browser web --page <page>
agent-env browser screenshot <lease> --browser web --page <page>
agent-env browser click <lease> --snapshot <snapshot> --node n4
agent-env browser set-text <lease> --snapshot <snapshot> --node n2 --text <replacement>
agent-env browser key <lease> --snapshot <snapshot> --node n2 --key Enter
agent-env browser scroll <lease> --snapshot <snapshot> --node n4 --delta-y 400
agent-env browser wait <lease> --browser web --wait-for text --contains Ready
agent-env browser console <lease> --browser web --duration 1s
agent-env browser network <lease> --browser web --duration 1s
agent-env browser page-close <lease> --browser web --page <page>
agent-env destroy <lease>
```

### ブラウザーとページの選択

`--browser`を省略できるのは、割り当て済み関連付けが1つだけの場合です。
ページ一覧の順序は決定的です。対象ページが複数ある場合、ページを操作するコマンドでは
明示的に選びます。閉じる操作には常に正確なページ IDが必要です。
ページ作成時の既定URLは`about:blank`です。移動先はユーザー認証情報を含まない
HTTP(S) URLまたは`about:blank`に限り、移動すると以前の文書の参照は無効になります。
`capabilities` は移動・入力をせず、製品名・バージョン・プロトコルを報告します。

### リースの状態と所有確認

すべてのブラウザー操作でリース操作を排他制御する fenceを保持します。変更操作にはactive、期限内、readyの
リースが必要です。読み取り専用診断はOS の識別情報に基づく所有権を証明できればdegradedでも使えます。
quarantined、所有権不明、未完了コマンド runがあるリースでは拒否します。
操作のたびにネイティブプロセス、予約済みポート、ブラウザー WebSocket 識別情報、CDPが報告する
ルート PID、専用プロファイルのコマンドラインを再検証します。古いポートの新しい待ち受けプロセスは操作権限の根拠になりません。

## snapshot・入力・制限

### スナップショットの識別と対応フレーム

バージョン 1 スナップショットは構造化JSONと簡潔な意味情報をまとめたテキストを提供します。
`n4`などの参照は、登録済みスナップショット、リース、ブラウザー、ページ、文書に限定されます。
アクセシビリティツリーから役割・名前・状態を取得し、上限付きDOM/layout スナップショットを補助証拠にします。
同一オリジンの iframe と shadow DOMの観測に対応します。別オリジンの iframe と OOPIF の観測と、すべてのiframe 意味情報による入力は今回明示的に非対応です。
曖昧なフレーム識別情報は安全側に倒して拒否します。

### 入力前の検証とフォーカス

意味情報による入力には毎回`--snapshot`と`--node`が必要です。入力前に現在のブラウザーとページ、
文書の loader、フレーム、backend DOM フィンガープリントとの一致を確認します。
切り詰められたスナップショット、バックエンドの識別情報の欠落、ノードの変化、別リース・ページの参照を拒否し、
保存座標への切り替えは行いません。Unicode テキストはCDP入力後に読み返して一致を確認します。
内部の固定JavaScriptは対象ノードの状態確認に使えますが、任意JavaScriptや未加工の CDPの公開コマンドはありません。
キーはEnter、Tab、Escape、Backspace、Delete、矢印、Home、End、PageUp、PageDownに対応します。

キーとテキストの入力は対象ページを前面にしてから対象ノードへフォーカスします。
文書と対象ノードのフォーカスを、入力の送信前と全選択後に確認します。
ページの前面化やフォーカスにも副作用があり、その後に入力結果を確認できず失敗した場合は不確定状態を保ちます。

明示的なページ移動はURL の hash や履歴の変化でも以前のブラウザースナップショットを無効にします。文書の識別トークンは
元の URLのダイジェスト（クエリー文字列そのものは保存しない）、ノードフィンガープリントは許可した非テキスト AX 状態を含みます。
set-textは明示的な`--text`が必須で、明示した空文字列はフィールドを消去します。CLIは操作の必須引数を検証し、
期間を明示的にゼロとした指定は、保存先を開く前に拒否します。

### 操作の上限と待機条件

操作タイムアウトは既定30秒、最大60秒です。console/network 取得は既定1秒、1 msから10秒までです。
置換テキストはNULを含まないUTF-8で4096 byte以内、scrollは各軸±10000以内です。
waitはload、URL、accessible text/role、消失を対象とし、タイムアウト後はポーリングしません。
応答・ノード・成果物の上限に達したら切り詰めまたは失敗を明示し、部分的な証拠を完全とは扱いません。

URL待機には空でない`--contains`が必要です。`--role`はテキストやノード消失の条件でのみ使えます。
不完全なスナップショットからノードの消失を断定しません。入力runの証拠には操作前にページ・参照元
スナップショット・ノードを記録し、ブラウザーの応答が失われても残します。入力テキストは引き続き伏せ字にします。

## 証拠・privacy・復旧

### 登録する成果物

成果物は`leases/<lease>/artifacts/<run>/`に保存します。`run.json`、スナップショットのJSON/text、
必要に応じてDOM JSONとPNGを含み、結果には上限付きコンソールとネットワークの収集結果を記録します。
登録ダイジェストとリース識別情報でスナップショットの再利用を検証します。
スクリーンショットは有効なPNGであることを確認し、観測したページと結び付く寸法・ダイジェストを保持します。
秘密が写る可能性があり、画素の自動伏せ字処理は行いません。

### 秘密情報を伏せる範囲

編集可能フィールドとパスワードフィールドの値は意味情報と DOM のテキスト証拠から除きます。入力テキストを平文の操作メタデータには残しません。
認識した継承秘密情報と入力値は構造化結果から伏せ字にし、URL認証情報・query値も伏せ字にします。
ネットワークはrequest 識別情報、伏せ字処理済みURL、method、status、type、timestamp、failureを記録し、
headerとresponse bodyは保存しません。コンソールは接続期間内に限定し、接続前の永続履歴は保証しません。
ページテキスト・URL・コンソールには未認識の秘密が残り得るため、ブラウザー成果物はすべて非公開に扱います。

set-text前に長さと全体と接頭辞の SHA フィンガープリントだけを成果物登録します。後続観測がこの証拠を読み込み、
ページやコンソールへ表示された入力も伏せ字にします。証拠欠落・破損は安全側に倒して失敗し、検証CPUにも上限があります。
フィンガープリントは非公開な復旧証拠であり、暗号化ではありません。

### 結果が不確実な操作と削除

入力中の切断は結果不明であり、未実行の証明ではありません。完了を確認できないrunはクリーンアップを阻止する記録として
残します。自動再実行や、destroyを通すためのrun破棄は行いません。証拠を保持し、実際の結果を確認して
レビューを経た復旧を判断します。destroy/GCは既存のプロセスツリーの所有確認規則に従い、
不在を証明してからプロファイルを削除します。ブラウザーの自動再起動、ブラウザー側でのPID クリーンアップ、
ネイティブ終了の代わりの`Browser.close`はありません。downloadと再利用するログインプロファイルは対象範囲外です。
BrowserとAndroid UIは別の契約を維持します。

### protocolと証拠の上限

接続先の探索は64 KiB、WebSocket メッセージは8 MiB、個々の CDP 呼び出しは操作期限内で5秒までです。
ページは128、フレームは32、AX/DOM ノードは2048までです。意味情報と DOM の JSONは秘匿処理後もそれぞれ1 MiB、AX文字列が4096 byteを超える場合は全体を`[TRUNCATED]`へ置き換え、
スナップショットをtruncatedと表示します。truncated スナップショットで入力は許可しません。
console/networkはそれぞれ256 record、文字列合計64 KiB、各文字列4096 byteまでです。超過した文字列は全体を`[TRUNCATED]`へ置き換えます。
512 件のイベントを格納する転送バッファーがあふれると切断して取得を失敗させます。
DOM証拠は構造と配置のみで、テキスト・属性・入力値は保存しません。

保存証拠の上限は伏せ字処理後にも適用します。取得時間は購読とドメインの有効化の前から計測し、
期限までに有効化が完了しなければ取得を失敗させます。省略したコンソール引数はtruncatedと明示します。

詳細な責務分担は[設計](../design-docs/browser-cdp-automation.ja.md)、今後の機能は[ロードマップ](../roadmap.ja.md)を参照してください。
