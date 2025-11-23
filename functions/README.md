# EdgeBase Functions - WasmEdge エッジ関数基盤

SPEC.mdに基づいたWasmEdgeランタイムを用いたエッジ関数実行プラットフォームの実装です。

## コンポーネント

### 1. hello-world (WASM関数サンプル)
Rustで実装されたシンプルなHTTPハンドラ関数。

**ビルド:**
```bash
cargo build --package hello-world --target wasm32-unknown-unknown --release
```

**出力:** `target/wasm32-unknown-unknown/release/hello_world.wasm`

### 2. edge-runner (エッジランナー)
WASMモジュールを実行するHTTPサーバー。

**機能:**
- Wasmerランタイムを使用したWASM実行
- HTTPリクエストのルーティング
- Prometheusメトリクスエクスポート (`/metrics`)
- SHA256によるアーティファクト検証
- LRUキャッシュマネージャー

**ビルド:**
```bash
cargo build --package edge-runner --release
```

**実行:**
```bash
./target/release/edge-runner <wasm_file>
```

**例:**
```bash
./target/release/edge-runner ./target/wasm32-unknown-unknown/release/hello_world.wasm
```

サーバーは `http://0.0.0.0:3000` で起動します。

**テスト:**
```bash
# 関数呼び出し
curl http://localhost:3000/api/test

# メトリクス確認
curl http://localhost:3000/metrics
```

### 3. control-plane (コントロールプレーン)
関数の登録、アーティファクト管理を行うAPIサーバー。

**機能:**
- 関数の登録・取得
- WASMアーティファクトのアップロード・ダウンロード
- SHA256ハッシュ計算と保存
- メタデータ管理

**ビルド:**
```bash
cargo build --package control-plane --release
```

**実行:**
```bash
./target/release/control-plane
```

サーバーは `http://0.0.0.0:8080` で起動します。

**API例:**

```bash
# 関数登録
curl -X POST http://localhost:8080/api/v1/functions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hello-world",
    "entrypoint": "handle",
    "runtime": "wasm",
    "memory_pages": 16,
    "max_execution_ms": 500
  }'

# アーティファクトアップロード
curl -X POST http://localhost:8080/api/v1/functions/{id}/upload \
  -F "file=@./target/wasm32-unknown-unknown/release/hello_world.wasm"

# 関数取得
curl http://localhost:8080/api/v1/functions/{id}

# アーティファクトダウンロード
curl http://localhost:8080/api/v1/artifacts/{id}/{version}
```

## アーキテクチャ

```
[Developer] -> [Control Plane] -> [Artifact Store (in-memory)]
                                         |
                                         v
                                  [Edge Runner] -> [WASM Execution]
                                         |
                                         v
                                  [Prometheus Metrics]
```

## 実装済み機能

✅ WASM関数のサンプル実装 (hello-world)
✅ Edge Runner (Wasmerベース)
  - HTTPサーバー
  - WASM実行エンジン
  - メトリクスエクスポート
  - キャッシュマネージャー
✅ Control Plane
  - 関数登録API
  - アーティファクト管理
  - SHA256検証

## メトリクス

Edge Runnerは以下のPrometheusメトリクスを公開します:

- `wasm_invoke_count_total`: WASM関数の呼び出し回数
- `wasm_invoke_latency_seconds`: WASM関数の実行レイテンシ
- `wasm_invoke_errors_total`: WASM関数のエラー回数

## セキュリティ

- WASMモジュールのSHA256検証
- メモリページ制限 (16ページ = 1MB)
- 実行タイムアウト制御
- サンドボックス化された実行環境

## 今後の拡張

SPEC.mdに記載されている以下の機能は今後実装可能:

- PostgreSQLによる永続化
- MinIO/S3統合
- gRPC/WebSocketによるCP-Edge通信
- Hot/Coldインスタンス管理
- ルーティング機能
- WASI capability制御
- AOTコンパイル
- 複数エッジノードのサポート
