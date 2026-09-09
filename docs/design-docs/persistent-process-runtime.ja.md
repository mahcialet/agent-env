---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/design-docs/persistent-process-runtime.md
source_sha256: b0b1657a8331aafa079292ae5d12863c13dac2f9fb0c74aec972ab8f4a4cca54
---

# 常駐プロセスのlifecycle設計

[English（翻訳元）](persistent-process-runtime.md)

この文書では、常駐プロセスの起動、所有権の記録、停止確認を説明します。
manifest 構文は[機能契約](../product-specs/persistent-process-runtime.ja.md)、
実装と検証の証拠は[完了 ExecPlan](../exec-plans/completed/persistent-process-runtime.ja.md)
を参照してください。プロセス管理は Compose、Android Emulator、Flutter、Browser/CDP の
機能から独立しています。[browser アダプター](browser-cdp-automation.ja.md)は app の
インターフェースを通じてこのライフサイクルを利用し、process アダプターには browser の
動作を持ち込みません。

## 責務と永続化

config は runtime と endpoint の種類ごとに設定を厳密に検証します。
Domain は追加した `Runtime.process` snapshot に、未展開のコマンドと環境変数参照、
固定コミット、解決した実行ファイルの証拠、パス、名前付きポート、OS 上の PID と
開始時の識別情報を記録します。process field のない既存 JSON snapshot は変わりません。

App はソース、状態、予約、起動意図を管理し、readiness、証拠、操作 fence、heartbeat、
補償、解放を調整します。process アダプターは `execx` を通して OS を操作し、
ほかの runtime アダプターは import しません。

App は操作前に不変のソースを展開し、runtime のディレクトリ、予約ポート、stdout/stderr
のパスを確定します。切り離したプロセスを起動する前に起動意図を保存し、返された
OS 上の識別情報を readiness 確認の前に保存します。エラーとともに返されたゼロでない
識別情報も、削除判断の証拠として保持します。

起動結果が不確かな場合、何も起きなかったとして再試行してはいけません。
展開した host の認証情報は一時的な入力として扱い、desired state の metadata へ保存しません。

runtime の保存先は `leases/<id>/process-runtimes/<runtime>/` です。
ここに `state/`、`stdout.log`、`stderr.log`、`owner.json`、`redaction.json`、`launch.json`
を置きます。`${runtime_dir}` として公開するのは `state/` だけです。

ソースを基準とする cwd と実行ファイルの相対パスは、OS ネイティブのパスとして解決します。
symlink の解決後にも所定の範囲内にあることを確認します。PATH から選んだツールには、
host 由来であること、絶対パス、読み取れるファイルの SHA-256 を記録します。
この証拠は host のソフトウェアが不変であることを保証せず、差し替えとの競合も解消しません。

### 起動記録と非公開ログの秘匿処理

専用の `launch.json` は所有情報と OS 上の識別情報を保持します。
それとは独立した `redaction.json` を OS 上での Start の前に保存します。
後者には所有情報、形式の version、secret の長さ・全文 digest・先頭 digest からなる
fingerprint を記録し、平文の secret は永続化しません。

この記録により、host の secret 変数を変更・削除した後や、起動後の記録保存に失敗した
場合も、処理量に上限を設けた秘匿処理ができます。起動後に必要な秘匿処理の証拠が
欠落している、または不正な場合は、ログの export を止めます。

fingerprint は暗号化ではありません。未加工の stdout/stderr は、引き続き非公開の保存先で
保護する必要があります。削除時の artifact として保持するログも、同じ上限付きの
秘匿処理を通します。

## 観測と停止

切り離したプロセスを扱う基盤 API は、PID だけに頼らず、OS 上のプロセスツリーの同一性を
確認します。後から独立して起動した CLI も、その識別情報を検査します。
保存済みの ready 行だけでは、現在も正常であるとは判断できません。

foreground の起点プロセスをライフサイクルの基準とします。予期せず終了した場合は
lease を degraded とし、自動再起動は予約しません。所有するツリー全体の不在を
証明するまでは、プロセスが worktree を使い続けている可能性があります。

停止時には、signal を送る前と強制停止へ移る前に、正確な識別情報を再検証します。
Unix の group は、起点の終了や子プロセスの残存により所有権が不確かになることがあります。
不在を証明できない場合は、再利用された group を kill せず quarantine にします。
プロセス生成時の識別情報と group は signal の直前に確認しますが、OS 上の group の
観測と signal 送信を不可分の操作にはできません。

Windows の Job/guardian の状態は、数値 PID よりも詳しいツリーの所有情報を持ちます。
停止時は対象の Job handle を保持して Job を強制終了します。
console signal による穏当な終了は保証しません。

キャンセル、永続化の失敗、停止の未完了があれば、復旧の証拠と予約を保持します。
ツリー全体の不在を証明した場合だけ可変状態とポートを解放し、通常のソース削除へ
進みます。その際も tracked ファイルの変更を保護します。

## Port、endpoint、将来の利用側

Browser/CDP は実装済みの利用側です。今後の機能もこの境界を通じて所有するプロセスを
利用でき、汎用のプロセス管理を変更する必要はありません。

SQLite の予約は、agent-env 同士の名前付き loopback TCP ポートの割り当てを直列化します。
OS 上での空き確認は外部による占有を検出しますが、対象プロセスが bind するまでの
競合は解消できません。socket activation と listen socket の継承は対象外です。
プロセスは渡された loopback のアドレスとポートへ bind する必要があります。

App は `runtime_port` を共通の解決済み endpoint 表現へ変換します。
そのため、HTTP readiness や Flutter reverse の利用側は、解決後に process 固有の構文を
扱う必要がありません。

可変状態は runtime ごとに分離し、quarantine 中は保持します。証拠の保存先には、
診断のために意図して採取したものだけを残します。browser profile やローカル DB を
自動的に証拠として取り込むことはありません。Browser/CDP の観測はこの
プロセスの寿命、状態ディレクトリ、ログ、endpoint を利用します。browser 固有の動作は
別のアダプターが担当します。

## 検証

Windows・macOS・Linux でのネイティブ統合、クラッシュからの復旧、兄弟 lease の存続、
起点プロセスの所有権が不確かな場合の回帰検証が必要です。
cross-build は追加の証拠にとどまり、ネイティブ実行の代わりにはなりません。
