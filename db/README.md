# IoT Data Sync System

エッジデバイス上のlibSQLとコントロールプレーンのCockroachDBの間で双方向のデータ同期を実現するシステム。

## アーキテクチャ

- **Edge Agent**: エッジデバイス上で動作し、libSQLからデータを読み取り、コントロールプレーンに同期
- **Sync Service**: コントロールプレーン上のAPIサーバー、CockroachDBにデータを保存

## 主要機能

### アップストリーム同期
- ✓ バッチ同期 (最大1000レコード)
- ✓ 指数バックオフリトライ (初期1秒、最大5回)
- ✓ 同期ステータス管理 (pending/syncing/synced/failed)
- ✓ 競合解決 (Last-Write-Wins戦略)

### ダウンストリーム同期
- ✓ コマンドポーリング
- ✓ コマンド実行
- ✓ 確認応答 (ACK)

### その他
- ✓ デバイス登録
- ✓ 同期ステータス取得
- ✓ エラーハンドリング

## セットアップ

### 前提条件

- Rust 1.70+
- CockroachDB (またはPostgreSQL)

### データベースセットアップ

```bash
# CockroachDBの起動
cockroach start-single-node --insecure --listen-addr=localhost:26257

# データベース作成
cockroach sql --insecure -e "CREATE DATABASE iot_sync;"

# マイグレーション実行
cockroach sql --insecure --database=iot_sync < migrations/001_initial_schema.sql
```

### Sync Service起動

```bash
cd sync-service
export DATABASE_URL="postgresql://root@localhost:26257/iot_sync?sslmode=disable"
cargo run --release
```

### Edge Agent起動

```bash
cd edge-agent
export DEVICE_ID="device-001"
export API_URL="http://localhost:8080"
cargo run --release
```

## 環境変数

### Sync Service
- `DATABASE_URL`: CockroachDB接続文字列 (デフォルト: `postgresql://localhost/iot_sync`)

### Edge Agent
- `DEVICE_ID`: デバイスID (デフォルト: ランダムUUID)
- `API_URL`: Sync Service URL (デフォルト: `http://localhost:8080`)

## API エンドポイント

- `POST /api/v1/sync/telemetry` - テレメトリデータ同期
- `GET /api/v1/sync/commands/:device_id` - コマンド取得
- `POST /api/v1/sync/ack/:command_id` - コマンド確認応答
- `GET /api/v1/sync/status/:device_id` - 同期ステータス取得
- `POST /api/v1/devices/register` - デバイス登録
- `GET /health` - ヘルスチェック

## 開発

```bash
# ビルド
cargo build

# テスト
cargo test

# フォーマット
cargo fmt

# Lint
cargo clippy
```
