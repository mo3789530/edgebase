# クイックスタートガイド

## 1. データベースのセットアップ

### CockroachDBの起動
```bash
cockroach start-single-node --insecure --listen-addr=localhost:26257 --background
```

### データベースとテーブルの作成
```bash
cockroach sql --insecure -e "CREATE DATABASE iot_sync;"
cockroach sql --insecure --database=iot_sync < migrations/001_initial_schema.sql
```

### デバイスの登録（テスト用）
```bash
cockroach sql --insecure --database=iot_sync -e "
INSERT INTO devices (device_id, device_name, device_type, status) 
VALUES ('00000000-0000-0000-0000-000000000001', 'test-device', 'sensor', 'active');

INSERT INTO sync_status (device_id) 
VALUES ('00000000-0000-0000-0000-000000000001');
"
```

## 2. Sync Serviceの起動

ターミナル1:
```bash
export DATABASE_URL="postgresql://root@localhost:26257/iot_sync?sslmode=disable"
cargo run --bin sync-service
```

サービスが起動したら、ヘルスチェックで確認:
```bash
curl http://localhost:8080/health
```

## 3. Edge Agentの起動

ターミナル2:
```bash
export DEVICE_ID="00000000-0000-0000-0000-000000000001"
export API_URL="http://localhost:8080"
cargo run --bin edge-agent
```

## 4. サンプルデータの挿入

ターミナル3:
```bash
export DEVICE_ID="00000000-0000-0000-0000-000000000001"
cargo run --example insert_sample_data
```

## 5. 同期の確認

Edge Agentのログで同期が実行されていることを確認:
```
Synced 10 records
```

CockroachDBでデータを確認:
```bash
cockroach sql --insecure --database=iot_sync -e "
SELECT device_id, sensor_id, data_type, value, timestamp 
FROM telemetry_data 
ORDER BY timestamp DESC 
LIMIT 10;
"
```

## トラブルシューティング

### 接続エラーが発生する場合
- CockroachDBが起動しているか確認: `cockroach node status --insecure`
- Sync Serviceが起動しているか確認: `curl http://localhost:8080/health`

### データが同期されない場合
- Edge Agentのログを確認
- デバイスIDがCockroachDBに登録されているか確認
- ネットワーク接続を確認

## クリーンアップ

```bash
# CockroachDBの停止
cockroach quit --insecure

# データベースファイルの削除
rm -f edge.db edge.db-shm edge.db-wal
```
