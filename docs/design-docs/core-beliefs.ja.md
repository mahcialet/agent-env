---
status: active
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/design-docs/core-beliefs.md
source_sha256: 2ca7c26c0b4120d08bd0fc22a8df15432a0a078b78124cb50cba7921a12c90ba
---

[English（翻訳元）](core-beliefs.md)

# 基本原則

設計の選択では、以下の原則を守ります。構成と責務の関係は[アーキテクチャ](../../ARCHITECTURE.ja.md)、
ライフサイクルの仕組みは [lease の設計](lease-control-plane.ja.md)を参照してください。

不変の source set によって環境を再現可能にします。lease はランタイム起動前に、要求された全 ref と解決済み commit を記録します。lease を 1 つの commit 列だけに縮約してはいけません。

SQLite は desired state、予約、証拠を管理します。Git と各ランタイムアダプターは外部リソースを観測します。Reconciliation は外部への副作用がトランザクションであるかのように扱わず、これらの事実を結び付けます。

失敗はデータです。失敗した割り当て、補償処理、隔離イベントを保持します。削除では、楽観的な削除よりも追跡対象ファイルの編集とリソース識別情報の保護を優先します。

コンポーネントが依存関係を持ち、stack がルートを指定します。必要最小限の閉包を選び、高コストで無関係なサービスの起動を避けます。リポジトリの manifest を、起動構成を明示的に定義する基準とします。

Windows、macOS、Linux のネイティブ動作は製品要件です。引数配列、OS ネイティブのパス、CGo 不要のリリースを使い、暗黙の shell や必須 daemon を導入しません。

リポジトリの知識は索引付きの記録体系です。繰り返し得られる教訓を実行可能な検査へ昇格させます。指示は案内として保ち、計画は最新にし、主張を実際のテスト証拠と結び付けます。

lease は偶発的な競合を防ぎます。悪意あるリポジトリコードを sandbox 化するものではありません。
