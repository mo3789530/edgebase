# Cluster Agent (MVP WIP)

`docs/cluster-agent-design.md` の Task 1-3 を開始した初期実装です。

## 実装済み

- `cmd/agent` 起動エントリーポイント
- Control Plane HTTP client
- heartbeat loop
- inventory loop (collectorは暫定)
- sync fetch/apply/ack loop (applierは暫定)
- ローカル状態ストア (メモリ)

## 環境変数

- `AGENT_CLUSTER_ID` (required, UUID)
- `AGENT_TOKEN` (required)
- `AGENT_CONTROL_PLANE_URL` (default: `http://localhost:8000`)
- `AGENT_VERSION` (default: `dev`)
- `AGENT_KUBECONFIG` (default: empty; emptyなら in-cluster config)
- `AGENT_TARGET_NAMESPACES` (default: `edge-functions`, comma-separated)
- `AGENT_REQUEST_TIMEOUT` (default: `10s`)
- `AGENT_HEARTBEAT_INTERVAL` (default: `15s`)
- `AGENT_INVENTORY_INTERVAL` (default: `60s`)
- `AGENT_SYNC_INTERVAL` (default: `10s`)
- `AGENT_PATH_HEARTBEAT` (default: `/api/v1/nodes/%s/heartbeat`)
- `AGENT_PATH_INVENTORY` (default: `/api/v1/clusters/%s/inventory`)
- `AGENT_PATH_SYNC` (default: `/api/v1/nodes/%s/sync`)
- `AGENT_PATH_ACK` (default: `/api/v1/nodes/%s/sync/ack`)

## 実行

```bash
cd cluster-agent
go test ./...
go run ./cmd/agent
```

## 現在の同期アクション対応

- `APPLY_DEPLOYMENT`
- `APPLY_SERVICE`
- `DELETE_DEPLOYMENT`
- `DELETE_SERVICE`
- `RESTART_DEPLOYMENT`

未対応アクションは `skipped` としてACKに記録します。
