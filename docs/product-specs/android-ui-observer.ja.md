---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/product-specs/android-ui-observer.md
source_sha256: 20c7812ffb2c718ecab1d317a45ebd09f375025fd365306a8ed107f2c5829128
---

# Android UI observer

[English（翻訳元）](android-ui-observer.md)

observer は、既存のリースが所有し、その所有権を確認できた Android Emulator のみを操作します。
観測するのは Flutter が公開する情報を含む Android のアクセシビリティ情報であり、
Flutter のウィジェットではありません。対象プロジェクトへの instrumentation の追加や
マニフェストの変更は不要です。

## 対象の選択と状態

`ui` コマンドにはリース ID と、`--application NAME` または `--runtime NAME` を指定します。
application を指定すると、永続化された対応関係に従って Android runtime と package を選択します。
どちらも省略する場合、割り当て済みの Android runtime がちょうど一つである必要があります。
application の snapshot は既定でその package に絞り、`--all-windows` を明示した場合は
システム UI も含めます。runtime だけを指定した snapshot は、アクセス可能なすべての window を含めます。
対象が存在しない場合、指定が矛盾する場合、対象を一意に選べない場合は、副作用を起こす前に失敗します。
すべての操作はリースの operation fence を保持し、所有権 marker、process、console の識別情報、
明示的なローカル ADB routing を再検証します。

snapshot、screenshot、logcat は、リースが ready または degraded で、所有権を確認できる
runtime が稼働中の場合に利用できます。quarantined、期限切れ、released、cleanup 要求済みの
リースは拒否します。UI を変更する操作には、ready、active、かつ期限内のリースが必要です。
読み取り要求でも、使い捨ての所有デバイスに検証済み observer companion をインストールする場合があります。
この副作用は記録します。

## コマンド

既存の `--output table|json` と出力 envelope を使用します。コマンド体系案は次のとおりです。

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

任意の serial、shell コマンド、selector 式、artifact path は受け付けません。
timeout は正の値で、上限は 60 秒です。swipe の所要時間は 50〜2000 ms です。
wait の polling は毎秒一回以下とし、timeout 時に入力を実行したり、無制限に再試行したりしません。

## snapshot と action の仕様

version 1 の snapshot は、リース、runtime、serial、対象 package、backend version、
取得時刻、snapshot ID を記録します。window と node は class、package、resource ID、
label、text、bounds、関連する状態と action、親の文脈を保持します。node 参照は snapshot 内でのみ有効です。
bounds はデバイスのピクセル単位です。一定の走査順序から番号付きの簡潔なテキストを生成します。
上限は node 数 1000、深さ 64、各フィールド 4096 文字、応答データ 1 MiB です。
切り詰めた場合は明示します。Android window ID は観測情報としてのみ保持し、継続的な action の識別情報には使いません。
不完全な snapshot は診断専用です。

意味情報に基づく action は、登録済み snapshot の path と digest を検証して読み込み、
リースと runtime の識別情報および記録済み backend の provenance を確認したうえで、同じ operation fence の保持中に再観測します。
異なる helper build で取得した snapshot は、入力を許可する根拠にできません。
記録した意味情報の fingerprint 全体を現在の node と照合し、一致がちょうど一つであることを要求して、
その現在の accessibility node 経由で操作します。fingerprint には window の意味情報による識別要素（type/title/bounds/root package/class）、祖先の文脈、class、
package、resource ID、編集可能でない label と text、bounds、action と状態の flag を含めます。
走査順の参照番号と、編集可能な値・password 値は含めません。一致なし、または複数一致の場合は、
それぞれ安定した診断コード `AGENTENV-UI-STALE` / `AGENTENV-UI-AMBIGUOUS` を返し、入力を実行しません。
座標への fallback はありません。動的に移動した UI には、新しい snapshot が必要です。
検証後にアプリが変化する場合まで、操作の atomicity を保証するものではありません。

テキスト置換には `ACTION_SET_TEXT` を使用します。この action を提供し、focus がある編集可能な node が必要です。
focus が必要な場合は、先に tap して新しい snapshot を取得します。成功には読み戻した値との一致が必要です。
一致しない場合、結果は不確実とし、自動再試行してはいけません。accessibility 側の拒否を操作成功と扱いません。
座標 tap は独立した明示的操作です。Back、Home、単一ポインタの swipe には Android 標準の入力機構を使用します。
API が対応していない場合は `AGENTENV-UI-UNAVAILABLE` を返します。

## 証拠とプライバシー

操作意図、対象の識別情報、完了または不確実な結果、artifact の digest を永続化します。
編集可能な node と password node のフィールドは常に伏せます。set-text の入力値は、そのコマンド自身が
保持する JSON、生の tree 証拠、log、error を含む証拠から完全に伏せます。読み戻しとの比較は helper 内で実施し、その外へは一致したかどうかの真偽値だけを返します。
label や編集可能でない text にも秘密情報が含まれ得るため、設定済みの秘密値を伏せます。
入力値を将来のコマンド用の秘密値照合データとして保持することはありません。後の snapshot には、アプリが
編集可能でない場所に表示した同じ値が含まれる場合があります。明示的に要求した logcat にも、アプリが log に
書いた値が含まれ得ます。設定済み秘密値の redaction は引き続き適用します。
アプリが表示する任意の text に機密情報がないことまでは保証できません。証拠ファイルには限定的なアクセス権を設定します。

screenshot は application を指定していても display 全体を取得します。PNG のピクセルに text redaction は適用できません。
完全な PNG を decode し、寸法を検証して 16 MiB / 1600 万ピクセル以内であることを確認した後、
atomic に永続化して digest を登録します。

logcat には application の指定が必要です。現在観測した PID に対象を絞り、その PID と
デバイス時刻による取得範囲の下限を報告します。process 再起動をまたぐ履歴や subprocess の網羅は保証しません。
global log を消去せず、対象を絞れない収集へ暗黙に切り替えることもありません。
出力の秘密値を伏せ、256 KiB / 2000 行を上限とします。登録済み artifact に加え、text/JSON 出力内でも内容を返します。
PID が再利用されると、以前の process の履歴行が含まれる場合があります。帰属を示すのは現在の数値 PID であり、
その package の履歴であることまで証明するものではありません。

## companion と移植性

agent-env が所有する独立した instrumentation APK を使い、instrumentation の対象も自身とします。
安定した Android platform の `UiAutomation` を使用し、Android API 26 以降を必要とします。
AndroidX や対象アプリへの依存はありません。source、protocol version、build input はこのリポジトリで管理します。
明示的に実行する native Go build tool が APK と、その digest および source digest を含む metadata を生成します。
通常の Go build と test に Android SDK や JDK は不要です。runtime は使用前に metadata と
インストール済み APK の同一性を検証します。競合する helper が既にインストールされている場合は置換せず、拒否します。
使い捨て AVD の削除に伴って helper も削除されます。global ADB service や host tool の設定は変更しません。

内部の操作 timeout 時には、元のリースの fence が有効な間に、この検証済み companion だけを停止し、
その不在を確認するため、追加で最大 10 秒を使う場合があります。入力は再試行しません。
呼び出し元によるキャンセルや fence 喪失の場合、自動復旧は行いません。
中断された操作や完了を確認できないデバイス操作では、既存の実行中コマンドに対する cleanup barrier を保持します。
Destroy が入力操作と競合して先に進むことはありません。期限付き observer 操作が fence を保持している間は、
busy を返す場合があります。入力結果が不確実な場合は調査が必要であり、暗黙に再試行してはいけません。
中断後は、registry に復旧可能な `termination-unconfirmed` という分類が明示的に保存され、
元の結果 artifact が登録済みで検証可能な場合に限り、`ui recover LEASE --run RUN` で
登録済みの実行中 UI helper 操作を終了できます。分類を保存する前の crash を含め、分類がない場合は拒否します。
分類の書き込み失敗によって、host process や証拠が不確実な操作を helper 復旧の対象にしてはいけません。
このコマンドは新しい fence を使い、helper の同一性と process の不在を証明し、復旧の証拠を保持してから、元の操作を
失敗または結果不確実として記録します。native input や任意の test process の復旧、別 run の barrier 解除、
失われた screenshot の再構築、中断された操作の成功扱いは行いません。
証拠の記録が失敗した場合も、cleanup を進める前に、保持された完了または復旧の証拠が必要です。
