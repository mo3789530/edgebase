# 設計ドキュメント：ローカルDB OTA (Schema & Data Update)

## 概要

この機能は、エッジデバイス上で動作するデータベース（SQLiteを想定）のスキーマおよびマスタデータを、Control Planeから遠隔で安全に更新（OTA: Over-The-Air Update）する仕組みを提供します。

## アーキテクチャ

```
┌──────────────────┐          MQTT / HTTP           ┌──────────────────┐
│  Control Plane   │ ─────────────────────────> │    Edge Agent    │
│   (Go Server)    │ <───────────────────────── │   (Rust Client)  │
└────────┬─────────┘        Status Report       └────────┬─────────┘
         │                                               │
         │ Updates                                       │ Migrations
         ▼                                               ▼
┌──────────────────┐                            ┌──────────────────┐
│ Schema Repository│                            │    Local DB      │
│ (Postgres/S3)    │                            │    (SQLite)      │
└──────────────────┘                            └──────────────────┘
```

## コンポーネント

### 1. Control Plane (Server)

*   **スキーマ管理**: マイグレーションファイル（Up/Down SQL）のバージョン管理。
*   **配信管理**: どのノードグループにどのバージョンを適用するかを制御。
*   **状態監視**: 各エッジノードの現在のDBバージョンを追跡。

### 2. Edge Agent (Client)

*   **更新検知**: MQTT通知または定期ポーリングで更新を検知。
*   **マイグレーション実行**:
    *   現在のアクティブなバージョンを確認。
    *   差分SQLのダウンロード。
    *   トランザクション内でのSQL適用。
    *   失敗時のロールバック。
*   **結果報告**: 適用成功/失敗と現在のバージョンをControl Planeへ通知。

## データフロー

1.  **Upload**: 管理者が新しいスキーマ（v2.sql）をControl Planeにアップロード。
2.  **Notify**: Control Planeが対象ノードへMQTTで「更新あり（v2）」を通知。
3.  **Download**: Edge Agentが通知を受け、Control Planeからv2.sqlを取得。
4.  **Apply**: Edge AgentがローカルDBに対し `BEGIN TRANSACTION` -> `Execute v2.sql` -> `COMMIT` を実行。
5.  **Report**: Edge Agentが「v2適用完了」をControl Planeへ送信。

## データモデル (Control Plane)

```go
type SchemaMigration struct {
    ID          uuid.UUID
    Version     int       // 1, 2, 3...
    Description string
    UpSQL       string    // 適用するSQL
    DownSQL     string    // ロールバック用SQL
    Checksum    string    // ファイル整合性チェック用
    CreatedAt   time.Time
}

type NodeSchemaStatus struct {
    NodeID        uuid.UUID
    CurrentVersion int
    LastUpdated   time.Time
    Status        string    // "synced", "pending", "failed"
    ErrorMessage  string
}
```

## API 設計

### Control Plane

*   `POST /api/v1/schemas`: 新しいマイグレーションの登録
*   `GET /api/v1/schemas`: マイグレーション一覧取得
*   `GET /api/v1/schemas/{version}/download`: SQLファイル内容の取得

### Edge Agent (Internal Logic)

*   `MigrationManager`: バージョン管理とSQL実行を担当。
*   `SchemaState`: `_schema_migrations` テーブルで適用済みバージョンを管理。

## エラーハンドリング

*   **ダウンロード失敗**: リトライ（Exponential Backoff）。
*   **適用失敗**: トランザクションロールバックし、エラーをControl Planeへ報告。再試行は行わず、手動介入（または修正版の配布）を待つ。