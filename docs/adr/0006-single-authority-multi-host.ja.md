---
status: accepted
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/adr/0006-single-authority-multi-host.md
source_sha256: c4bbb16fda1c1f364819c7c7c9bb130e603e7e4afb219ab6e2167a0514a1de5e
---

# 一つの controller 管理主体と lease 全体の worker 割当

[English](0006-single-authority-multi-host.md)

## 背景

local registry は既に、保守的な resource lifecycle、operation fence、証拠を管理しています。
remote の配置には global scheduling と通信の復旧が必要ですが、heartbeat の途絶を resource 不在と見なしたり、
既存の local 規則を緩めたりしてはいけません。local モードは daemon 不要のまま維持します。
[ExecPlan](../exec-plans/completed/multi-host-control-plane.ja.md) にこの仕様と、
文書化した範囲で完了した受け入れの証拠を記録しています。

## 決定

専用の controller SQLite を管理主体とし、登録済み role の相互 TLS と version 付き型付き JSON を使って、
worker から接続します。lease 全体を worker 一つに配置し、global の canonical ULID を local でも再利用します。
controller/host/host-instance と assignment epoch を local の作用より先に永続化します。
epoch は一つの assignment の各操作を通じて固定し、通常の registry 保存では metadata を変更できなくします。
通常の local mutation と GC は、force や期限切れでも controller の管理を回避できません。
worker 操作は完全一致の assignment tuple を持ち、既存 app/local operation fence を維持します。

双方で operation の識別情報と payload を永続化します。重複配送では journal に記録した結果を復旧します。
作用開始の可能性がある時点以降の不確実性には照合が必要であり、無条件の mutation 再実行を許可しません。
artifact 配送前に結果を local で永続化します。global RELEASED には worker の cleanup/不在の証明が必要です。
heartbeat の途絶では OFFLINE・古い観測・UNKNOWN とし、生存中や不確実な resource を再配置しません。
分かりやすい host 名だけを根拠に、別 instance へ所有権を移せません。

commit 済みで独立に検証できる Git bundle と証拠を、上限付き SHA-256 CAS で転送します。
source object は 1 GiB、artifact は 64 MiB が上限です。呼出側の path を保存先操作の根拠にしません。
worker-local の endpoint と環境変数解決を明示します。
[設計](../design-docs/multi-host-control-plane.ja.md) と
[製品仕様](../product-specs/multi-host-control-plane.ja.md) を参照してください。

## 代案と結果

local SQLite を分散 DB として拡張すると cleanup の管理主体が曖昧になります。
heartbeat による failover は生存中 resource を重複させるおそれがあります。
操作ごとの epoch 更新では assignment の識別と配送の識別が混同されます。
応答消失後の mutation 再試行では入力や test の作用が重複するおそれがあります。
lease を worker 間で分割するには、この段階を超える network と分散 cleanup の設計が必要です。

controller を一つにすると単純になりますが、HA や合意形成は提供しません。
独立に複製した有効な controller DB の同時稼働は安全ではありません。
永続 journal と digest 単位の転送には保存領域と復旧処理が必要で、観測が古い lease では自動置換の代わりに
明示的な照合が必要になる場合があります。これらの負担を受け入れることで、不確実性を保持し、
外部作用の exactly-once を提供するかのような扱いを避けます。

Windows/macOS/Linux の native role test と、実際の TLS を使った二 worker の統合検証が必要です。
同じ host の role process や cross-build では、物理マシン・VM による複数 host の稼働を証明できません。
ExecPlan はこの区別と未検証の前提条件を保持します。この ADR は設計の採用を記録するもので、完了の宣言ではありません。
