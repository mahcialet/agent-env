---
status: accepted
owner: maintainers
last_verified: 2026-09-09
translation_of: docs/adr/0005-separate-flutter-applications.md
source_sha256: 1fa718e9947b21f723adc0aa426cad8629768082a3a0ba9b6c5c62979179bf5e
---

# FlutterアプリケーションとAndroidリソースを分離する

[English（翻訳元）](0005-separate-flutter-applications.md)

この判断では、Android リソースの所有権を独立させ、その上で Flutter アプリを扱います。
実行順序と復旧の詳細は[ライフサイクル設計](../design-docs/flutter-android-runtime.ja.md)
を参照してください。

## 背景

Android Emulatorの所有、確保、ADBサーバー方針、クリーンアップはFlutterと独立しています。
アプリではビルドとインストール状態を追加しますが、このリソース境界を置き換えたり、
任意のデバイスに接続したりしてはいけません。

## 決定

`applications` と `component.application` を使います。appが独立したFlutterビルド
アダプターと、既存Androidアダプターのアプリ操作を調整します。runtimeアダプター間の
importは禁止し、Flutter/Android間とFlutter/Compose間の双方向にも適用します。
CLIが具体的な依存を接続します。config/domainは具体アダプターに依存しません。
アプリ記録は既存のシリアライズ済みLeaseに追加し、新しい関係スキーマは導入しません。

ランタイム確保前にビルドし、出自と意図する識別情報を保存してから、所有を確認した
シリアルに対してインストール、エンドポイント解決、reverse、起動を行います。
EmulatorライフサイクルとADBサーバー選択は引き続きAndroidアダプターだけが担います。
appは準備完了判定、reconcile、補償を担います。

## 代案と影響

FlutterをAndroidランタイムへ統合するとEmulator単独利用を妨げ、ビルド失敗にも高コストな
確保が先行します。汎用ワークロードプラグイン基盤は今回の機能範囲を超える抽象化です。
分離によりappが調整する操作は増えますが、リソース所有は変わりません。
APKハッシュはビルドを識別し、過去成果物の再実行を保証しません。

`repoctl arch-check` は既存の汎用規則でruntime間のimportを拒否します。
負例fixtureにはFlutterの新しいノードと入れ子パッケージを明示的に追加し、
app/domain/execxおよび同じアダプター内のimportには正常例を用意します。
