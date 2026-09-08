---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/design-docs/persistent-process-runtime.md
source_sha256: 6d969349a6f05899676df390db6bbda12b0652a6e19bcae33cd1adb514a3c83b
---

# 常駐プロセスのlifecycle設計

[English（翻訳元）](persistent-process-runtime.md)

manifest構文は[機能契約](../product-specs/persistent-process-runtime.ja.md)で定める。
実装と検証は[active ExecPlan](../exec-plans/active/persistent-process-runtime.ja.md)で追跡する。
この仕組みはprocess管理をCompose、Android Emulator、Flutter、将来のbrowser機能から分離する。

## 責務と永続化

configはruntimeとendpointのvariantを厳密に検証する。domainは追加field
`Runtime.process`のsnapshotを持ち、未展開のcommand/環境変数参照、固定source commit、
解決した実行ファイルの証拠、path、名前付きport、native PID/開始識別情報を記録する。
process fieldを省略した既存JSON snapshotは変わらない。appはsource、状態、予約、起動意図、
readiness、証拠、操作fence、heartbeat、補償、解放を調整する。process adapterは`execx`を
通してnative作用を担当し、ほかのruntime adapterをimportしない。

作用の前にappはimmutable sourceを展開し、runtime directory、予約port、stdout/stderr
pathを確定する。detached processの起動前に起動意図を保存し、readinessの前に返された
native識別情報を保存する。errorとともに返された0以外の識別情報もcleanup証拠として扱う。
起動結果が不確かな場合、何も起きなかったとして起動を再試行しない。展開したhost認証情報は
一時入力とし、desired stateのmetadataへ保存しない。

runtime rootは`leases/<id>/process-runtimes/<runtime>/`で、`state/`、`stdout.log`、
`stderr.log`、`owner.json`、`redaction.json`、`launch.json`を置く。`${runtime_dir}`として公開する
のは`state/`のみ。source相対のcwdと実行ファイルpathはnative pathで解決し、symlink解決後の
閉じ込めを確認する。PATH toolにはhost由来、絶対path、読み取れるfileのSHA-256を記録する。
この証拠はhost softwareの不変性を保証せず、差し替えの競合も解消しない。

## 観測と停止

detached primitiveはPIDだけに頼らずnative treeの同一性を証明する。後続の独立したCLIは
その識別情報を検査する。保存済みready行だけでは現在の正常性を示さない。foregroundの
起点processをlifecycleの基準とする。予期しない終了はleaseをdegradedにし、自動再起動は
予約しない。所有tree全体の不在を証明するまで、processがworktreeを使い続ける可能性がある。

停止ではsignal送信前と強制停止への移行前に正確な識別情報を再検証する。Unix groupは
起点終了や子processの残存で所有が不確実になることがある。不在を証明できなければ、
再利用されたgroupをkillせずquarantineにする。Unixの生成/group検査はsignal直前に行うが、native groupへのsignalと観測をatomicには
できない。Windowsのnative Job/guardian状態は数値PID以上のtree所有情報を持つ。停止時は
正確なJob handleを保持してJobを強制終了し、console signalによる穏当な終了は約束しない。取消し、永続化失敗、停止未完了では復旧証拠と予約を保持する。
tree全体の不在を証明した場合のみ可変状態とportを解放し、通常のtracked変更保護付きsource
cleanupへ進む。

## Port、endpoint、将来の利用側

SQLite予約はagent-env間の名前付きloopback TCP割り当てを直列化する。nativeの空き確認は
外部占有を検出するが、対象がbindするまでの競合は解消できない。socket activationと継承する
listen socketは対象外。processは渡されたloopback address/portへbindする必要がある。
appは`runtime_port`を共通の解決済みendpoint表現へ対応付けるため、HTTP readinessやFlutter
reverseの利用側は解決後にprocess固有構文を必要としない。

可変状態はruntimeごとに分離し、quarantine中は保持する。証拠storageには意図した診断だけを
保持し、browser profileやlocal databaseを自動で昇格しない。将来のbrowser観測はこの
process寿命、状態directory、logs、CDP状endpointを利用でき、process adapterへbrowser機能を
追加する必要はない。native Windows/macOS/Linux integration、crash復旧、兄弟leaseの存続、
起点の所有が不確かな場合の回帰検証が必要となる。cross-buildは追加証拠にとどまる。

専用の`launch.json`は所有情報とnative識別情報を保持する。独立した`redaction.json`は
native Start前に保存し、所有情報、形式version、secretの長さ/全文digest/先頭digestの
fingerprintを持つ。平文secretを永続化せず、host secret変数の変更・削除後や、起動後の
receipt保存失敗時も上限付きredactionを可能にする。
起動後のredaction証拠が不正・欠落ならlog exportを止める。未加工stdout/stderrには
引き続き専用storageでの保護が必要であり、fingerprintは暗号化ではない。
cleanup artifactとして保持するlogも同じ上限付きredaction経路を通る。
