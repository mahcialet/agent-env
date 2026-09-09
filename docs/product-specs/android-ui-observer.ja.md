---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/product-specs/android-ui-observer.md
source_sha256: ceb49f9210ac487b5cf39f0c8ed07981e19501c1f1ab48ac84d154a6c5dd3dba
---

# Android UI observer

[English（翻訳元）](android-ui-observer.md)

observer は、既存のリースが所有し、その所有権を確認できた Android Emulator のみを操作します。
観測するのは Flutter が公開する情報を含む Android のアクセシビリティ情報であり、
Flutter のウィジェットではありません。対象プロジェクトへの instrumentation の追加や
マニフェストの変更は不要です。この機能は実装済みです。
本仕様では導入、対象選択、入力の安全条件、証拠の上限を確認できます。端末の割り当てと
削除は [Android ランタイム仕様](android-emulator.ja.md)を参照してください。

## companion と移植性

agent-env が所有する独立した instrumentation APK を使い、instrumentation の対象も自身とします。
安定した Android platform の `UiAutomation` を使用し、Android API 26 以降を必要とします。
AndroidX や対象アプリへの依存はありません。ソース、プロトコルバージョン、ビルド入力はこのリポジトリで管理します。
明示的に実行するネイティブ Go ビルドツールが APK と、そのダイジェストおよびソースダイジェストを含むメタデータを生成します。
通常の Go ビルドとテストに Android SDK や JDK は不要です。
通常の補助ツール利用では `AGENT_ENV_UI_HELPER` にディレクトリを明示します。
未設定や空白だけの値からカレントディレクトリのファイルを選ぶことはありません。
復旧には記録済みの検証済み補助ツール識別情報を使い、ホストの補助ツールファイルが消えたり置換されたりしていても、
別ビルドを採用しません。ランタイムは使用前にメタデータと
インストール済み APK の同一性を検証します。競合する補助ツールが既にインストールされている場合は置換せず、拒否します。
使い捨て AVD の削除に伴って補助ツールも削除されます。global ADB サービスやホストツールの設定は変更しません。

### 補助 APK をビルドする

導入済みの SDK と JDK で補助 APK を一度ビルドし、ホストの環境変数設定で
`AGENT_ENV_UI_HELPER` に生成先を指定してください。出力ディレクトリは新規である必要があります。
例の platform と build-tools も導入済みでなければなりません。このビルダーはツールを
インストールせず、ライセンスの承諾も行いません。通常の Go ビルドと無関係なコマンドには、
補助 APK や JDK は不要です。

```text
go run ./tools/uihelper --sdk <sdk> --jdk <jdk> --platform android-35 --build-tools 36.0.0 --output <new-directory>
```

以下の `agent-env ui ...` の例は、ソースから実行する場合には
`go run ./cmd/agent-env ui ...` に置き換えられます。

## 対象の選択と状態

`ui` コマンドにはリース ID と、`--application NAME` または `--runtime NAME` を指定します。
application を指定すると、永続化された対応関係に従って Android ランタイムとパッケージを選択します。
どちらも省略する場合、割り当て済みの Android ランタイムがちょうど一つである必要があります。
application のスナップショットは既定でそのパッケージに絞り、`--all-windows` を明示した場合は
システム UI も含めます。ランタイムだけを指定したスナップショットは、アクセス可能なすべてのウィンドウを含めます。
対象が存在しない場合、指定が矛盾する場合、対象を一意に選べない場合は、副作用を起こす前に失敗します。
すべての操作はリースの operation fence を保持し、所有権 marker、プロセス、コンソールの識別情報、
明示的なローカル ADB routing を再検証します。

スナップショット、screenshot、logcat は、リースが ready または degraded で、所有権を確認できる
ランタイムが稼働中の場合に利用できます。quarantined、期限切れ、released、クリーンアップ要求済みの
リースは拒否します。UI を変更する操作には、ready、active、かつ期限内のリースが必要です。
読み取り要求でも、使い捨ての所有デバイスに検証済み observer companion をインストールする場合があります。
この副作用は記録します。

## コマンド

既存の `--output table|json` と出力 envelope を使用します。実装済みのコマンドは次のとおりです。

```text
agent-env ui snapshot LEASE --application mobile-app
agent-env ui screenshot LEASE --application mobile-app
agent-env ui tap LEASE --snapshot SNAPSHOT --node n7
agent-env ui set-text LEASE --snapshot SNAPSHOT --node n3 --text VALUE
agent-env ui tap-coordinate LEASE --runtime phone --x 120 --y 240
agent-env ui back LEASE --runtime phone
agent-env ui home LEASE --runtime phone
agent-env ui swipe LEASE --runtime phone --x 120 --y 400 --to-x 120 --to-y 100
agent-env ui wait LEASE --application mobile-app --contains Ready --timeout 10s
agent-env ui logcat LEASE --application mobile-app --since 30s
agent-env ui recover LEASE --run RUN
```

任意のシリアル、shell コマンド、selector 式、成果物パスは受け付けません。
タイムアウトは正の値で、上限は 60 秒です。swipe の所要時間は 50〜2000 ms です。
wait のポーリングは毎秒一回以下とし、タイムアウト時に入力を実行したり、無制限に再試行したりしません。

## snapshot と action の仕様

### スナップショットの内容と上限

バージョン 1 のスナップショットは、リース、ランタイム、シリアル、対象パッケージ、backend バージョン、
取得時刻、スナップショット ID を記録します。ウィンドウとノードは class、パッケージ、リソース ID、
label、テキスト、bounds、関連する状態と action、親の文脈を保持します。ノード参照はスナップショット内でのみ有効です。
bounds はデバイスのピクセル単位です。一定の走査順序から番号付きの簡潔なテキストを生成します。
上限はノード数 1000、深さ 64、各フィールド 4096 文字、応答データ 1 MiB です。
切り詰めた場合は明示します。Android ウィンドウ ID は観測情報としてのみ保持し、継続的な action の識別情報には使いません。
不完全なスナップショットは診断専用です。

### 登録済み参照と古い対象の拒否

意味情報に基づく action は、登録済みスナップショットのパスとダイジェストを検証して読み込み、
リースとランタイムの識別情報および記録済み backend の provenance を確認したうえで、同じ operation fence の保持中に再観測します。
異なる補助ツールビルドで取得したスナップショットは、入力を許可する根拠にできません。
記録した意味情報のフィンガープリント全体を現在のノードと照合し、一致がちょうど一つであることを要求して、
その現在の accessibility ノード経由で操作します。フィンガープリントにはウィンドウの意味情報による識別要素（type/title/bounds/root package/class）、祖先の文脈、class、
パッケージ、リソース ID、編集可能でない label とテキスト、bounds、action と状態のフラグを含めます。
走査順の参照番号と、編集可能な値・password 値は含めません。一致なし、または複数一致の場合は、
それぞれ安定した診断コード `AGENTENV-UI-STALE` / `AGENTENV-UI-AMBIGUOUS` を返し、入力を実行しません。
座標への fallback はありません。動的に移動した UI には、新しいスナップショットが必要です。
検証後にアプリが変化する場合まで、操作の atomicity を保証するものではありません。

### 入力と読み戻し確認

テキスト置換には `ACTION_SET_TEXT` を使用します。この action を提供し、focus がある編集可能なノードが必要です。
focus が必要な場合は、先に tap して新しいスナップショットを取得します。成功には読み戻した値との一致が必要です。
一致しない場合、結果は不確実とし、自動再試行してはいけません。accessibility 側の拒否を操作成功と扱いません。
座標 tap は独立した明示的操作です。Back、Home、単一ポインタの swipe には Android 標準の入力機構を使用します。
API が対応していない場合は `AGENTENV-UI-UNAVAILABLE` を返します。

## 証拠とプライバシー

### テキストの証拠と秘密情報の抑制

操作意図、対象の識別情報、完了または不確実な結果、成果物のダイジェストを永続化します。
編集可能なノードと password ノードのフィールドは常に伏せます。set-text の入力値は、そのコマンド自身が
保持する JSON、生の tree 証拠、ログ、エラーを含む証拠から完全に伏せます。読み戻しとの比較は補助ツール内で実施し、その外へは一致したかどうかの真偽値だけを返します。
label や編集可能でないテキストにも秘密情報が含まれ得るため、設定済みの秘密値を伏せます。
入力値を将来のコマンド用の秘密値照合データとして保持することはありません。後のスナップショットには、アプリが
編集可能でない場所に表示した同じ値が含まれる場合があります。明示的に要求した logcat にも、アプリがログに
書いた値が含まれ得ます。設定済み秘密値の伏せ字処理は引き続き適用します。
アプリが表示する任意のテキストに機密情報がないことまでは保証できません。編集可能な値や password 値の自動抑制では、それらの値を含まないフィンガープリントを保持します。
設定済み秘密値がノードやウィンドウの意味情報に含まれる場合、すべてのノードフィンガープリントを
削除するため、そのスナップショットは意味情報による入力を許可できません。
ウィンドウの意味情報に秘密値が含まれる場合は、派生したウィンドウキーもすべて削除します。
操作に秘密値の照合データがある場合、操作後の tree が非公開で秘密値由来の識別情報を検証できないため、
任意項目である操作後のフィンガープリントを省略します。
各フィールドと応答全体の上限は、伏せ字処理後にも JSON の escape とスナップショットメタデータを含めて適用します。
フィールドやノード全体を省略した場合は切り詰めを明示し、メタデータだけで上限を超える場合は上限内の invalid 結果を返します。
証拠ファイルには限定的なアクセス権を設定します。

### スクリーンショット

screenshot は application を指定していても display 全体を取得します。PNG のピクセルにテキスト伏せ字処理は適用できません。
完全な PNG を decode し、寸法を検証して 16 MiB / 1600 万ピクセル以内であることを確認した後、
atomic に永続化してダイジェストを登録します。

### アプリケーションの logcat

logcat には application の指定が必要です。現在観測した PID に対象を絞り、その PID と
デバイス時刻による取得範囲の下限を報告します。プロセス再起動をまたぐ履歴や subprocess の網羅は保証しません。
global ログを消去せず、対象を絞れない収集へ暗黙に切り替えることもありません。
出力の秘密値を伏せ、256 KiB / 2000 行を上限とします。登録済み成果物に加え、text/JSON 出力内でも内容を返します。
PID が再利用されると、以前のプロセスの履歴行が含まれる場合があります。帰属を示すのは現在の数値 PID であり、
そのパッケージの履歴であることまで証明するものではありません。
`--since` は 1s〜1h の整数秒を受け付け、小数秒は実行前に拒否します。
デバイスから末尾の記録を一件余分に取得して実際の省略を検出します。
指定時間内の完全な行がちょうど 2000 行あるだけでは、切り詰めとは扱いません。

## 中断された操作の復旧

内部の操作タイムアウト時には、元のリースの fence が有効な間に、この検証済み companion だけを停止し、
その不在を確認するため、追加で最大 10 秒を使う場合があります。入力は再試行しません。
呼び出し元によるキャンセルや fence 喪失の場合、自動復旧は行いません。
中断された操作や完了を確認できないデバイス操作では、既存の実行中コマンドに対するクリーンアップ barrier を保持します。
Destroy が入力操作と競合して先に進むことはありません。期限付き observer 操作が fence を保持している間は、
busy を返す場合があります。入力結果が不確実な場合は調査が必要であり、暗黙に再試行してはいけません。

中断後は、registry に復旧可能な `termination-unconfirmed` という分類が明示的に保存され、
元の結果成果物が登録済みで検証可能な場合に限り、`ui recover LEASE --run RUN` で
登録済みの実行中 UI 補助ツール操作を終了できます。分類を保存する前の crash を含め、分類がない場合は拒否します。
分類の書き込み失敗によって、ホストプロセスや証拠が不確実な操作を補助ツール復旧の対象にしてはいけません。
このコマンドは新しい fence を使い、補助ツールの同一性とプロセスの不在を証明し、復旧の証拠を保持してから、元の操作を
失敗または結果不確実として記録します。ネイティブ入力や任意のテストプロセスの復旧、別 run の barrier 解除、
失われた screenshot の再構築、中断された操作の成功扱いは行いません。
証拠の記録が失敗した場合も、クリーンアップを進める前に、保持された完了または復旧の証拠が必要です。

## 設計と検証記録

補助 APK、操作の排他制御、復旧の責務は[設計](../design-docs/android-ui-observer.ja.md)、
実装時の検証結果は[完了済み ExecPlan](../exec-plans/completed/android-ui-observer.ja.md)を参照してください。
