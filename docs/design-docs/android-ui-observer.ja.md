---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/design-docs/android-ui-observer.md
source_sha256: 1aeb2a9a24e22a43edc5e452090b83df06ed137782bd699fa9da62c3914f53d5
---

# Android UI observer の設計

[English（翻訳元）](android-ui-observer.md)

動作は[製品仕様](../product-specs/android-ui-observer.ja.md)で定義します。
実装と受け入れの証拠は[進行中の計画](../exec-plans/active/android-ui-observer.ja.md)に記録します。
対象マニフェストへの section 追加や SQL migration はありません。

app は小さな `AndroidUIProvider` interface を通じて、対象選択、policy、operation fencing、
操作意図の永続化、最終 artifact を管理します。domain は直列化できる observer の値を定義します。
既存の Android adapter がデバイスへの副作用を担当し、`applicationADB` の同一性・routing 検証を再利用します。
Flutter や他の runtime adapter は import しません。CLI は引数解析、provider の接続、結果表示のみを担当します。

操作意図と cleanup barrier には既存の `CommandRun` row を使用します。
状態を読み込む前に、context に紐づく lease lock を取得します。その context をすべての registry 操作と
provider 操作へ渡します。必要な証拠を atomic に書き込み、digest を登録してから、run の終端状態を公開します。
キャンセル、lock 喪失、process 完了未確認、証拠の不足では、永続化した barrier を解除しません。
永続化する argv に text 入力値を含めません。最終証拠の記録には、既存の fence を保持する
WithoutCancel パターンを使用できます。

登録済み snapshot artifact の読み込みは、期待する lease artifact root 内に限定します。
通常ファイルであること、root 内に収まること、サイズ、digest を検証します。
別リースの参照をデバイス入力に使用することはできません。照合は companion の一回の呼び出し内で実行します。
再観測して意味情報の fingerprint を比較し、候補がゼロまたは複数なら拒否したうえで、
選択した現在の AccessibilityNodeInfo 経由で操作します。切り詰めた観測結果では一意性を証明できないため、
意味情報に基づく変更操作を許可しません。編集可能な値は fingerprint とすべての直列化出力から除外します。
CLI で対象を指定しなかった場合も、参照した snapshot の対象範囲と runtime 識別情報を維持します。

companion は自分自身を対象とする instrumentation で、対象アプリから独立しています。
platform の UiAutomation により、AndroidX への依存なしで Flutter の semantics を観測できます。
backend の試作検証では、ACTION_SET_TEXT が受理されるだけでは不十分であると判明しました。
focus のない Flutter node は、text を更新せずに true を返す場合があります。
対応 action の提供と focus を要求し、置換後に改めて読み戻して検証してから成功を報告します。
keyboard の表示によって走査順の番号も変わるため、この番号を識別情報として使用してはいけません。

companion は native Go から javac、aapt2、Java D8、ZIP、Java apksigner を制御して明示的に build します。
core workflow では SDK の shell/batch wrapper を使用しません。provenance に version、source digest、
APK digest を含め、インストール済み digest を検証し、管理外の helper package は拒否します。
helper は利用者が選んで local build する形で配布します。自動 download や license 受諾は行わず、
通常の Go 利用に JVM を必須としません。初期の対応下限は API 26 です。
実際の SDK 検証の証拠では、対応するすべての Android image を検証したかのように示さず、
試験した API を明記する必要があります。

unit test と fake adapter の test は、対象選択、stale/ambiguous な参照、プライバシー、
証拠記録の失敗、fence による cleanup 保護を検証します。native OS CI は argv、filesystem、
永続化の動作を検証します。実際の Linux Emulator test では、Flutter semantics、Unicode 置換、
PNG 証拠、別リースとの分離を独立に検証します。cross-build は SDK の検証証拠ではありません。

復旧でもリースの fence を維持します。helper の内部期限切れ時は、元の operation context が引き続き有効な間に、
fence の下で新たに操作意図を書き込み、期限付きの停止・静止確認を行う場合があります。
呼び出し元によるキャンセルや lock 喪失では、実行中 barrier を保持します。
明示的な `ui recover --run` は、新しい fence の下で、登録済みの中断された helper run を扱います。
復旧可能な `termination-unconfirmed` という分類が明示的に保存され、元の結果が登録済みで
digest を検証できることが必要です。禁止する分類がないことだけでは不十分です。
最終分類の保存が失敗すると、registry に古い操作意図だけが残る場合があるためです。
crash により分類が記録されなかった中断は、引き続き拒否します。
保存された runtime と serial を検証し、インストール済み companion の digest を確認して、その package だけを停止します。
PID の不在を確認し、失敗または結果不確実という outcome と復旧 artifact を記録してから、cleanup barrier を解除します。
入力の再試行や、任意の native command の復旧は行いません。

Android window ID は instrumentation の接続をまたぐと変わる一時的な値です。
生の観測には保持しますが、node の照合には含めず、window の意味情報を示す metadata を使います。
変化していない Flutter tree は再接続後も操作できる必要があります。
意味情報が同じ window が複数ある場合も、node の照合結果が一意であることを要求します。
