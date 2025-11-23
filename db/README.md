# Edge Agent

エッジデバイス上で動作するローカルデータベースエージェント。libSQLでデータを管理し、コントロールプレーンと同期します。

## アーキテクチャ

- **Edge Agent**: エッジデバイス上で動作し、libSQLからデータを読み取り、コントロールプレーンに同期

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

### Edge Agent起動

```bash
cd edge-agent
export DEVICE_ID="device-001"
export API_URL="http://localhost:8080"
cargo run --release
```

## 環境変数

### Edge Agent
- `DEVICE_ID`: デバイスID (デフォルト: ランダムUUID)
- `API_URL`: コントロールプレーン URL (デフォルト: `http://localhost:8080`)

## API エンドポイント

コントロールプレーンのエンドポイント:
- `POST /api/v1/sync/telemetry` - テレメトリデータ同期
- `GET /api/v1/sync/commands/:device_id` - コマンド取得
- `POST /api/v1/sync/ack/:command_id` - コマンド確認応答
- `GET /api/v1/sync/status/:device_id` - 同期ステータス取得
- `POST /api/v1/devices/register` - デバイス登録
