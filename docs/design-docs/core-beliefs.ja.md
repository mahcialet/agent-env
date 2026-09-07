---
status: active
owner: maintainers
last_verified: 2026-09-08
translation_of: docs/design-docs/core-beliefs.md
source_sha256: 16cf89f7fcabe1e8b208563cecced20aca8104ba4cf3691fd48a6eb6c1c7bcb8
---

[English（正本）](core-beliefs.md)

# 基本原則

不変の source set によって環境を再現可能にします。lease はランタイム起動前に、要求された全 ref と解決済み commit を記録します。lease を 1 つの commit 列だけに縮約してはいけません。

SQLite は desired state、予約、証拠を管理します。Git と Docker は観測対象リソースの状態を管理します。Reconciliation は外部への副作用がトランザクションであるかのように扱わず、これらの事実を結び付けます。

失敗はデータです。失敗した割り当て、補償処理、隔離イベントを保持します。削除では、楽観的な削除よりも追跡対象ファイルの編集とリソース識別情報の保護を優先します。

コンポーネントが依存関係を持ち、stack がルートを指定します。必要最小限の閉包を選び、高コストで無関係なサービスの起動を避けます。リポジトリの manifest が明示的な起動の正本です。

Windows、macOS、Linux のネイティブ動作は製品要件です。引数配列、OS ネイティブのパス、CGo 不要のリリースを使い、暗黙の shell や必須 daemon を導入しません。

リポジトリの知識は索引付きの記録体系です。繰り返し得られる教訓を実行可能な検査へ昇格させます。指示は案内として保ち、計画は最新にし、主張を実際のテスト証拠と結び付けます。

lease は偶発的な競合を防ぎます。悪意あるリポジトリコードを sandbox 化するものではありません。
