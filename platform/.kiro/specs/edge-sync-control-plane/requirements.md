# 要件定義書

## はじめに

本システムは、エッジロケーションに配置されたlibsqlデータベースとWASMEdge関数を、中央のコントロールプレーンから同期・管理するための分散システムです。コントロールプレーンはCockroachDBを使用してメタデータを管理し、MinIOにWASMアーティファクトを保存します。

## 用語集

- **Control Plane**: 中央管理システム。エッジリソースの状態を管理し、同期を制御する
- **Edge Node**: libsqlデータベースとWASMEdge実行環境を持つエッジロケーション
- **Sync Manager**: エッジノードとコントロールプレーン間の同期を管理するコンポーネント
- **Artifact Store**: MinIOベースのWASMバイナリ保存システム
- **Metadata DB**: CockroachDBベースのコントロールプレーンデータベース
- **WASM Function**: WASMEdgeで実行される関数
- **Schema Version**: libsqlデータベースのスキーマバージョン

## 要件

### 要件1: エッジノード登録管理

**ユーザーストーリー:** システム管理者として、新しいエッジノードをコントロールプレーンに登録し、その状態を追跡したい

#### 受入基準

1. WHEN エッジノードが登録リクエストを送信する時、THE Control Plane SHALL エッジノードの情報をMetadata DBに保存する
2. THE Control Plane SHALL 各エッジノードに一意の識別子を割り当てる
3. WHEN エッジノードがヘルスチェックを送信する時、THE Control Plane SHALL 最終接続時刻をMetadata DBに更新する
4. THE Control Plane SHALL エッジノードの状態（オンライン、オフライン、同期中）を記録する

### 要件2: WASMアーティファクト管理

**ユーザーストーリー:** 開発者として、WASM関数をアップロードし、エッジノードにデプロイしたい

#### 受入基準

1. WHEN 開発者がWASM関数をアップロードする時、THE Control Plane SHALL アーティファクトをArtifact Storeに保存する
2. THE Control Plane SHALL 各WASM関数にバージョン番号を付与する
3. THE Control Plane SHALL WASM関数のメタデータ（名前、バージョン、ハッシュ値、作成日時）をMetadata DBに記録する
4. WHEN WASM関数が削除される時、THE Control Plane SHALL Artifact StoreとMetadata DBから関連データを削除する

### 要件3: データベーススキーマ同期

**ユーザーストーリー:** データベース管理者として、スキーマ変更をすべてのエッジノードに配信したい

#### 受入基準

1. WHEN スキーマ変更が登録される時、THE Control Plane SHALL 変更内容をMetadata DBに保存する
2. THE Control Plane SHALL スキーマ変更に順序番号を付与する
3. WHEN エッジノードが同期リクエストを送信する時、THE Control Plane SHALL 未適用のスキーマ変更を返す
4. THE Control Plane SHALL 各エッジノードの現在のスキーマバージョンを追跡する

### 要件4: エッジノードへの同期配信

**ユーザーストーリー:** システムとして、エッジノードに最新のWASM関数とスキーマを配信したい

#### 受入基準

1. WHEN エッジノードが同期リクエストを送信する時、THE Sync Manager SHALL 配信すべきWASM関数のリストを返す
2. WHEN エッジノードがWASM関数をリクエストする時、THE Control Plane SHALL Artifact Storeからバイナリを取得して返す
3. THE Sync Manager SHALL エッジノードの現在のデプロイ状態とターゲット状態を比較する
4. WHEN 同期が完了する時、THE Control Plane SHALL エッジノードのデプロイ状態をMetadata DBに記録する

### 要件5: 同期状態の監視

**ユーザーストーリー:** 運用担当者として、各エッジノードの同期状態を監視したい

#### 受入基準

1. THE Control Plane SHALL 各エッジノードの最終同期時刻を記録する
2. WHEN エッジノードの同期が5分以上遅延している時、THE Control Plane SHALL 警告状態を記録する
3. THE Control Plane SHALL 同期失敗の回数とエラー内容を記録する
4. THE Control Plane SHALL エッジノードごとのWASM関数とスキーマバージョンの一覧を提供する

### 要件6: トランザクション整合性

**ユーザーストーリー:** システムとして、同期処理中の障害に対して整合性を保ちたい

#### 受入基準

1. WHEN 同期処理が開始される時、THE Sync Manager SHALL トランザクションを開始する
2. IF 同期処理中にエラーが発生する時、THEN THE Sync Manager SHALL 変更をロールバックする
3. WHEN 同期処理が成功する時、THE Sync Manager SHALL トランザクションをコミットする
4. THE Control Plane SHALL 同期処理のログをMetadata DBに記録する

### 要件7: セキュリティと認証

**ユーザーストーリー:** セキュリティ管理者として、エッジノードとの通信を安全に保ちたい

#### 受入基準

1. WHEN エッジノードが接続する時、THE Control Plane SHALL 認証トークンを検証する
2. THE Control Plane SHALL TLS暗号化を使用して通信する
3. THE Control Plane SHALL 各エッジノードに固有の認証情報を発行する
4. WHEN 認証が失敗する時、THE Control Plane SHALL 接続を拒否しログに記録する
