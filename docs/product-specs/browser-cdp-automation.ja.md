---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/product-specs/browser-cdp-automation.md
source_sha256: 246e38b33249b8cdf87bf29ec0e3fdf1c00412a28a28e6dbc190d547400c1dad
---

# Browser/CDP自動操作

[英語版（翻訳元）](browser-cdp-automation.md)

browserコマンドは、persistent process leaseが所有する明示的なChromium系ブラウザーを
観測・操作します。別のブラウザーの起動、外部ブラウザーへの接続、個人用profileの再利用は
行いません。実装とnative環境での受け入れ状況は[実行中ExecPlan](../exec-plans/active/browser-cdp-automation.ja.md)に記録します。

## manifestと前提条件

必要なCDPメソッドを提供し、直接起動できるnativeのheadless Chromium系ブラウザーを
使います。検証にはChrome for Testingを推奨しますが、製品には同梱しません。
別のbrowser rootをforkするlauncherは非対応です。CDPが報告するPIDは、leaseが
所有するprocess rootと一致する必要があります。無関係なcoreコマンドにbrowser、
Node、Python、Playwright、Selenium、ChromeDriverは不要です。
既存のprocess実行ファイル・source相対パスの規則も適用します。

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

`browsers`は省略できます。宣言する場合は空でないmappingとし、各bindingから既存の
`process` runtimeと、そのruntimeの名前付きTCP portを参照します。
1つのruntimeにbindingは1つだけです。名前にはOS間で使えるものを指定し、
大文字・小文字を無視した衝突を認めません。`cdp`というport名だけでbrowser機能は有効になりません。

上の5つのswitchはすべて必須で、対象リポジトリがargv要素としてそれぞれ1回だけ、
記載どおりに宣言します。browser層は追加しません。保護対象switchの別表記・重複、
`--`終端、debugging pipe、profile選択の上書きを拒否します。
`--user-data-dir`は`${runtime_dir}/profile`に固定し、任意の専用subpathも認めません。
portと専用stateの管理はgeneric process runtimeが担当します。

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

`--browser`を省略できるのは、割り当て済みbindingが1つだけの場合です。
page一覧の順序は決定的です。対象pageが複数ある場合、pageを操作するコマンドでは
明示的に選びます。閉じる操作には常に正確なpage IDが必要です。
page作成時の既定URLは`about:blank`です。移動先はユーザー認証情報を含まない
HTTP(S) URLまたは`about:blank`に限り、移動すると以前のdocument handleは無効になります。
capabilitiesは移動・入力をせず、product/version/protocolを報告します。

すべてのbrowser操作でlease操作fenceを保持します。変更操作にはactive、期限内、readyの
leaseが必要です。read-only診断はnative所有権を証明できればdegradedでも使えます。
quarantined、所有権不明、未完了command runがあるleaseでは拒否します。
操作のたびにnative process、予約済みport、browser WebSocket identity、CDPが報告する
root PID、専用profileのcommand lineを再検証します。古いportの新しいlistenerは操作権限の根拠になりません。

## snapshot・入力・制限

version 1 snapshotは構造化JSONと簡潔なsemantic textを提供します。
`n4`などのhandleは、登録済みsnapshot、lease、browser、page、documentに限定されます。
Accessibility treeからrole・name・stateを取得し、上限付きDOM/layout snapshotを補助証拠にします。
same-origin iframeとshadow DOMの観測に対応します。cross-origin/OOPIF観測と、すべてのiframe semantic入力は今回明示的に非対応です。
曖昧なframe identityは安全側に倒して拒否します。

semantic入力には毎回`--snapshot`と`--node`が必要です。入力前に現在のbrowser/page、
document loader、frame、backend DOM fingerprintとの一致を確認します。
切り詰められたsnapshot、backend identityの欠落、nodeの変化、別lease・pageの参照を拒否し、
保存座標へのfallbackは行いません。Unicode textはCDP入力後に読み返して一致を確認します。
内部の固定JavaScriptは対象nodeの状態確認に使えますが、任意JavaScriptやraw CDPの公開コマンドはありません。
keyはEnter、Tab、Escape、Backspace、Delete、矢印、Home、End、PageUp、PageDownに対応します。

操作timeoutは既定30秒、最大60秒です。console/network captureは既定1秒、1 msから10秒までです。
置換textはNULを含まないUTF-8で4096 byte以内、scrollは各軸±10000以内です。
waitはload、URL、accessible text/role、消失を対象とし、timeout後はpollしません。
応答・node・artifactの上限に達したら切り詰めまたは失敗を明示し、部分的な証拠を完全とは扱いません。

## 証拠・privacy・復旧

artifactは`leases/<lease>/artifacts/<run>/`に保存します。`run.json`、snapshotのJSON/text、
必要に応じてDOM JSONとPNGを含み、結果には上限付きconsole/network collectionを記録します。
登録digestとlease identityでsnapshotの再利用を検証します。
screenshotは有効なPNGであることを確認し、観測したpageと結び付く寸法・digestを保持します。
秘密が写る可能性があり、pixelの自動redactionは行いません。

editable/password valueはsemantic・DOM text証拠から除きます。入力textを平文の操作metadataには残しません。
認識した継承secretと入力値は構造化結果からredactし、URL認証情報・query値もredactします。
networkはrequest identity、redact済みURL、method、status、type、timestamp、failureを記録し、
headerとresponse bodyは保存しません。consoleは接続期間内に限定し、接続前の永続履歴は保証しません。
page text・URL・consoleには未認識の秘密が残り得るため、browser artifactはすべてprivateに扱います。

入力中の切断は結果不明であり、未実行の証明ではありません。完了を確認できないrunはcleanup barrierとして
残します。自動再実行や、destroyを通すためのrun破棄は行いません。証拠を保持し、実際の結果を確認して
reviewを伴う復旧を判断します。destroy/GCは既存のprocess tree所有権規則に従い、
不在を証明してからprofileを削除します。browserの自動再起動、browser側でのPID cleanup、
native終了の代わりの`Browser.close`はありません。downloadと再利用するlogin profileはscope外です。
BrowserとAndroid UIは別の契約を維持します。

### protocolと証拠の上限

Discoveryは64 KiB、WebSocket messageは8 MiB、個別CDP callは操作期限内で5秒までです。
pageは128、frameは32、AX/DOM nodeは2048までです。semantic JSONは1 MiB、AX文字列が4096 byteを超える場合は全体を`[TRUNCATED]`へ置き換え、
snapshotをtruncatedと表示します。truncated snapshotで入力は許可しません。
console/networkはそれぞれ256 record、文字列合計64 KiB、各文字列4096 byteまでです。超過した文字列は全体を`[TRUNCATED]`へ置き換えます。
512 eventのtransport bufferがあふれると切断してcaptureを失敗させます。
DOM証拠はstructure/layoutのみで、text・attribute・input valueは保存しません。

set-text前に長さと全体/接頭辞SHA fingerprintだけをartifact登録します。後続観測がこの証拠を読み込み、
pageやconsoleへechoされた入力もredactします。証拠欠落・破損は安全側に倒して失敗し、検証CPUにも上限があります。
fingerprintはprivateな復旧証拠であり、暗号化ではありません。

明示navigationはhash/history変化でも以前のbrowser snapshotを無効にします。document tokenは
raw URLのdigest（query文字列そのものは保存しない）、node fingerprintは許可した非text AX stateを含みます。
set-textは明示的な`--text`が必須で、明示した空文字列はfieldを消去します。CLIは操作の必須引数を検証し、
明示zero durationはstoreを開く前に拒否します。

URL待機には空でない`--contains`が必要です。`--role`はtextやnode消失の条件でのみ使えます。
不完全なsnapshotからnodeの消失を断定しません。入力runの証拠には操作前にpage・参照元
snapshot・nodeを記録し、browserの応答が失われても残します。入力textは引き続きredactします。
