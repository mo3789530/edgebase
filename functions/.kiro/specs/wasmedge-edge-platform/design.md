# 設計書

## 概要

WasmEdgeエッジ関数基盤は、地理的に分散したエッジノード（POP）上でWASM関数を低レイテンシかつ安全に実行するプラットフォームです。Control Plane（CP）が関数のライフサイクル管理を担当し、Edge NodeがWasmEdgeランタイムを用いて実際の関数実行を行います。

設計の核心は以下の3点です：

1. **Pullベースのデプロイメント**: CPが通知を送信し、Edge NodeがArtifact Storeから必要なWASMモジュールを取得
2. **Hot/Cold実行戦略**: メモリ常駐インスタンスの再利用による低レイテンシ実行
3. **多層セキュリティ**: mTLS通信、WASI機能制限、リソース制約による安全な実行環境

## アーキテクチャ

### システム全体構成

```
┌─────────────────────────────────────────────────────────────┐
│                     Developer Zone                          │
│  ┌──────────┐      ┌──────────┐      ┌──────────┐          │
│  │   Git    │      │   CLI    │      │ Web UI   │          │
│  └────┬─────┘      └────┬─────┘      └────┬─────┘          │
│       └─────────────────┴───────────────────┘                │
└─────────────────────────┼────────────────────────────────────┘
                          │ HTTPS/API
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                   Control Plane (CP)                        │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              API Gateway (TLS)                       │  │
│  └──────┬───────────────────────────────────┬───────────┘  │
│         │                                    │              │
│  ┌──────▼──────────┐  ┌──────────────┐  ┌──▼──────────┐   │
│  │  Function Mgr   │  │ Build Worker │  │ Notification│   │
│  │  - Register     │  │ - Rust/TS    │  │ Service     │   │
│  │  - Versioning   │  │ - AOT        │  │ - gRPC      │   │
│  │  - Routing      │  │ - Validation │  │ - WebSocket │   │
│  └────────┬────────┘  └──────┬───────┘  └───────┬─────┘   │
│           │                  │                   │          │
│  ┌────────▼──────────────────▼───────┐  ┌───────▼───────┐ │
│  │   PostgreSQL (Metadata)           │  │ Redis (Queue) │ │
│  └───────────────────────────────────┘  └───────────────┘ │
│  ┌──────────────────────────────────────────────────────┐ │
│  │         MinIO / S3 (Artifact Store)                  │ │
│  └──────────────────────────────────────────────────────┘ │
└──────────────────────────┬───────────────────────────────────┘
                           │ gRPC/WebSocket (mTLS)
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Edge Network (POP)                       │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Edge Node (Tokyo-1)                     │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │         HTTP Server (Go)                       │  │  │
│  │  └──────────────────┬─────────────────────────────┘  │  │
│  │  ┌──────────────────▼─────────────────────────────┐  │  │
│  │  │      Function Router & Dispatcher              │  │  │
│  │  └──────────────────┬─────────────────────────────┘  │  │
│  │  ┌──────────────────▼─────────────────────────────┐  │  │
│  │  │      WasmEdge Runtime Manager                  │  │  │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐     │  │  │
│  │  │  │ Hot Pool │  │ Hot Pool │  │ Hot Pool │     │  │  │
│  │  │  │ func-A   │  │ func-B   │  │ func-C   │     │  │  │
│  │  │  └──────────┘  └──────────┘  └──────────┘     │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │         Local Cache Manager                    │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │      CP Communication Client                   │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │      Observability Exporter                    │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### データフロー

#### Function デプロイフロー

```
Developer → CP API → Build Worker → Artifact Store → PostgreSQL
                                                          ↓
                                                   Notification Service
                                                          ↓
                                                   Edge Runners
                                                          ↓
                                                   Pull Artifact
                                                          ↓
                                                   Local Cache
```

#### Request 実行フロー

```
End User → Edge HTTP Server → Route Matching → Function Router
                                                      ↓
                                              Check Local Cache
                                                      ↓
                                    ┌─────────────────┴─────────────────┐
                                    │                                   │
                                 Cached                            Not Cached
                                    │                                   │
                                    ▼                                   ▼
                            Check Hot Pool                      Pull from S3
                                    │                                   │
                        ┌───────────┴───────────┐                      │
                        │                       │                      │
                    Available              Not Available               │
                        │                       │                      │
                        ▼                       ▼                      ▼
                   Reuse VM               Create New VM          Cold Start
                        │                       │                      │
                        └───────────────────────┴──────────────────────┘
                                            │
                                            ▼
                                   WasmEdge Execute
                                            │
                                            ▼
                                   Return Response
```

## コンポーネントとインターフェース

### Control Plane コンポーネント

#### API Gateway
- **責務**: 外部APIリクエストの受付、認証、ルーティング
- **技術**: Go/Rust、TLS終端
- **エンドポイント**:
  - `POST /api/v1/functions` - Function登録
  - `POST /api/v1/functions/:id/deploy` - デプロイ
  - `POST /api/v1/routes` - Route作成
  - `POST /api/v1/nodes/:id/heartbeat` - Heartbeat受信

#### Function Manager
- **責務**: Function メタデータ管理、バージョニング、ルーティング設定
- **データストア**: PostgreSQL
- **主要操作**:
  - Function CRUD
  - Route管理
  - デプロイ履歴追跡

#### Build Worker
- **責務**: ソースコードからWASMモジュールのビルド
- **サポート言語**: Rust、TypeScript/JavaScript
- **ツールチェーン**: 
  - Rust: `cargo build --target wasm32-wasi`
  - TypeScript: `esbuild` + `wasm-pack`
- **AOT最適化**: WasmEdge AOTコンパイラによる事前コンパイル（オプション）

#### Notification Service
- **責務**: Edge Nodeへのデプロイ通知配信
- **プロトコル**: gRPC bidirectional stream または WebSocket
- **再試行戦略**: 指数バックオフ（初期: 1秒、最大: 60秒）
- **永続化**: Redis キューによる未配信通知の保持

### Edge Node コンポーネント

#### HTTP Server
- **責務**: TLS終端、リクエスト受付、レスポンス返却
- **技術**: Go `net/http`
- **機能**:
  - HTTP/1.1, HTTP/2サポート
  - リクエストバリデーション
  - タイムアウト管理

#### Function Router
- **責務**: HTTPリクエストからFunction IDへのマッピング
- **ルーティングロジック**:
  1. ホスト名マッチング
  2. パスパターンマッチング（prefix、exact、regex）
  3. 優先度による選択
- **データ構造**: Trie木またはRadix treeによる高速ルックアップ

#### WasmEdge Runtime Manager
- **責務**: WASMインスタンスのライフサイクル管理
- **技術**: `wasmedge-go` SDK
- **Hot Pool管理**:
  - Function IDごとにプール管理
  - 設定可能なプールサイズ（min: 1, max: 10）
  - アイドルタイムアウト（デフォルト: 5分）
- **リソース制限**:
  - メモリページ上限の適用
  - 実行タイムアウトの強制（Go `context.WithTimeout`）
  - WASI機能のホワイトリスト制御

#### Local Cache Manager
- **責務**: WASMファイルのローカルキャッシュ管理
- **ストレージ**: `/var/cache/wasm/{function_id}/{version}.wasm`
- **Evictionポリシー**: LRU
- **制限**:
  - 最大サイズ: 10 GB
  - 最大ファイル数: 1000
- **整合性**: SHA256検証

#### CP Communication Client
- **責務**: CPとの双方向通信
- **プロトコル**: gRPC bidirectional stream
- **機能**:
  - Heartbeat送信（30秒間隔）
  - デプロイ通知受信
  - ステータス報告
- **認証**: mTLS + JWT

#### Observability Exporter
- **責務**: メトリクス、ログ、トレースの出力
- **メトリクス**: Prometheus形式（`:9090/metrics`）
- **ログ**: 構造化JSON（stdout）
- **トレース**: OpenTelemetry

### インターフェース定義

#### CP ↔ Edge Node (gRPC)

```protobuf
service EdgeControl {
  rpc StreamControl(stream EdgeMessage) returns (stream ControlMessage);
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
}

message EdgeMessage {
  string node_id = 1;
  oneof payload {
    HeartbeatRequest heartbeat = 2;
    DeploymentStatus deployment_status = 3;
    MetricsReport metrics = 4;
  }
}

message ControlMessage {
  oneof payload {
    DeployInstruction deploy = 1;
    UndeployInstruction undeploy = 2;
    ConfigUpdate config = 3;
  }
}

message DeployInstruction {
  string function_id = 1;
  string version = 2;
  string artifact_url = 3;
  string sha256 = 4;
  int32 memory_pages = 5;
  int32 max_execution_ms = 6;
  repeated Route routes = 7;
}

message HeartbeatRequest {
  string node_id = 1;
  string status = 2;
  double cpu_percent = 3;
  int64 mem_bytes = 4;
  repeated FunctionStatus functions = 5;
}

message FunctionStatus {
  string function_id = 1;
  string version = 2;
  string state = 3;
  int64 invocation_count = 4;
  double avg_latency_ms = 5;
}
```

#### Edge Node ↔ WasmEdge (Host Functions)

```rust
// Logging
fn log(level: i32, message: &str);

// Metrics
fn metrics_increment(name: &str, value: f64, tags: &[(&str, &str)]);

// KV Store (optional)
fn kv_get(key: &str) -> Option<Vec<u8>>;
fn kv_put(key: &str, value: &[u8], ttl_seconds: i32) -> Result<(), Error>;

// HTTP Client (restricted)
fn http_fetch(url: &str, options: &HttpOptions) -> Result<HttpResponse, Error>;

// Environment
fn env_get(key: &str) -> Option<String>;

// Request context
fn request_id() -> String;
fn request_header(name: &str) -> Option<String>;
```

#### WASM Module Interface

```rust
// Standard HTTP handler interface
#[no_mangle]
pub extern "C" fn handle(
    method_ptr: *const u8, method_len: usize,
    path_ptr: *const u8, path_len: usize,
    headers_ptr: *const u8, headers_len: usize,
    body_ptr: *const u8, body_len: usize,
    response_ptr: *mut u8, response_cap: usize
) -> i32;
```

## データモデル

### Function

```sql
CREATE TABLE functions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    owner_project_id UUID NOT NULL,
    runtime VARCHAR(20) NOT NULL CHECK (runtime IN ('wasm', 'wasm-aot')),
    entrypoint VARCHAR(255) NOT NULL DEFAULT 'handle',
    artifact_url TEXT NOT NULL,
    sha256 CHAR(64) NOT NULL,
    memory_pages INTEGER NOT NULL CHECK (memory_pages > 0 AND memory_pages <= 256),
    max_execution_ms INTEGER NOT NULL CHECK (max_execution_ms > 0),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(name, version)
);

CREATE INDEX idx_functions_owner ON functions(owner_project_id);
CREATE INDEX idx_functions_name_version ON functions(name, version);
```

### Route

```sql
CREATE TABLE routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host VARCHAR(255) NOT NULL,
    path VARCHAR(500) NOT NULL,
    function_id UUID NOT NULL REFERENCES functions(id) ON DELETE CASCADE,
    methods TEXT[] NOT NULL DEFAULT '{"GET"}',
    pop_selector TEXT,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_routes_host_path ON routes(host, path);
CREATE INDEX idx_routes_function ON routes(function_id);
CREATE INDEX idx_routes_priority ON routes(priority DESC);
```

### Node

```sql
CREATE TABLE nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pop_id VARCHAR(50) NOT NULL,
    ip INET NOT NULL,
    last_heartbeat TIMESTAMP NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('online', 'degraded', 'offline')),
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_nodes_pop ON nodes(pop_id);
CREATE INDEX idx_nodes_status ON nodes(status);
CREATE INDEX idx_nodes_last_heartbeat ON nodes(last_heartbeat DESC);
```

### Deployment

```sql
CREATE TABLE deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES functions(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    target_pop VARCHAR(50),
    strategy VARCHAR(20) NOT NULL CHECK (strategy IN ('immediate', 'canary', 'rolling')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'in_progress', 'completed', 'failed', 'rolled_back')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deployments_function ON deployments(function_id);
CREATE INDEX idx_deployments_status ON deployments(status);
```

### Audit Log

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID NOT NULL,
    operation VARCHAR(20) NOT NULL CHECK (operation IN ('create', 'update', 'delete', 'deploy')),
    old_value JSONB,
    new_value JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at DESC);
```


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Function登録時のメタデータ完全性

*任意の*有効なFunction登録リクエストに対して、データベースに保存されるFunction recordは、Function ID、バージョン、Artifact URL、SHA256の全てのフィールドを含む必要がある。

**Validates: Requirements 1.3**

### Property 2: SHA256ハッシュの一貫性

*任意の*WASMファイルに対して、Build WorkerまたはEdge Nodeが計算するSHA256ハッシュは、同じファイルに対して常に同じ値を返す必要がある。

**Validates: Requirements 1.2, 3.2**

### Property 3: デプロイ通知の完全性

*任意の*デプロイ通知に対して、Function ID、バージョン、Artifact URL、SHA256、メモリページ数、最大実行時間の全ての必須フィールドが含まれている必要がある。

**Validates: Requirements 2.2**

### Property 4: SHA256検証による整合性保証

*任意の*Edge Nodeでのデプロイに対して、ダウンロードしたWASMファイルのSHA256ハッシュが通知に含まれるSHA256と一致する場合のみ、ファイルがキャッシュに保存される必要がある。

**Validates: Requirements 3.3**

### Property 5: ルーティングの決定性

*任意の*HTTPリクエスト（ホスト名、パス）に対して、Routeテーブルの検索結果は常に同じFunction IDを返す必要がある（Routeテーブルが変更されない限り）。

**Validates: Requirements 4.1**

### Property 6: キャッシュミス時のArtifact取得

*任意の*Function IDに対して、ローカルキャッシュに存在しない場合、Edge NodeはArtifact Storeから同期的にWASMモジュールを取得し、キャッシュに保存する必要がある。

**Validates: Requirements 4.3**

### Property 7: Hot Instance再利用の優先

*任意の*WASM関数実行リクエストに対して、Hot Instance Poolに利用可能なインスタンスが存在する場合、新しいインスタンスを作成する前に既存のインスタンスを再利用する必要がある。

**Validates: Requirements 5.1, 5.2**

### Property 8: プールサイズ制限の遵守

*任意の*関数実行完了後、Hot Instance Poolのサイズが上限に達している場合、インスタンスはプールに戻されず破棄される必要がある。

**Validates: Requirements 5.4**

### Property 9: LRU Evictionの正確性

*任意の*Hot Instance Poolに対して、メモリ使用量が閾値を超える場合、最も古い（last_usedが最小の）インスタンスが破棄される必要がある。

**Validates: Requirements 5.5**

### Property 10: メモリページ制限の適用

*任意の*WASM関数実行に対して、WasmEdgeランタイムに設定されるメモリページ上限は、Function metadataに指定された値と一致する必要がある。

**Validates: Requirements 6.1**

### Property 11: 実行タイムアウトの強制

*任意の*WASM関数実行に対して、実行時間がmax_execution_msを超える場合、実行がキャンセルされHTTP 504ステータスコードが返される必要がある。

**Validates: Requirements 6.2**

### Property 12: WASI機能のホワイトリスト制御

*任意の*WASM関数に対して、許可されていないWASI機能（raw socket、filesystem書き込み、process spawn）へのアクセスは拒否され、許可された機能（logging、clocks、random）へのアクセスは許可される必要がある。

**Validates: Requirements 6.3, 6.4**

### Property 13: Heartbeatの完全性

*任意の*HeartbeatリクエストはNode ID、ステータス、CPU使用率、メモリ使用量、キャッシュされたFunction一覧の全てのフィールドを含む必要がある。

**Validates: Requirements 7.2**

### Property 14: Heartbeatレスポンスの処理

*任意の*Heartbeatレスポンスに対して、未配信のデプロイ通知が含まれる場合、Edge Nodeは非同期にデプロイ通知を処理する必要がある。

**Validates: Requirements 7.3**

### Property 15: 再試行の指数バックオフ

*任意の*失敗した通信（デプロイ通知送信、Heartbeat送信、Artifact取得）に対して、再試行間隔は指数的に増加する必要がある（例: 1秒、2秒、4秒、8秒...）。

**Validates: Requirements 2.4, 3.5, 7.4**

### Property 16: メトリクスカウンターの単調増加

*任意の*WASM関数呼び出しに対して、wasm_invoke_count_totalカウンターは単調に増加する必要がある（減少しない）。

**Validates: Requirements 8.2**

### Property 17: レイテンシメトリクスの記録

*任意の*WASM関数実行完了に対して、wasm_invoke_latency_seconds_bucketヒストグラムに実行時間が記録される必要がある。

**Validates: Requirements 8.3**

### Property 18: エラーメトリクスの記録

*任意の*WASM関数実行エラーに対して、wasm_invoke_errors_totalカウンターがFunction IDとエラーコードごとにインクリメントされる必要がある。

**Validates: Requirements 8.4**

### Property 19: キャッシュヒット/ミスメトリクス

*任意の*WASMモジュール取得に対して、ローカルキャッシュから取得された場合はwasm_cache_hits_totalがインクリメントされ、Artifact Storeから取得された場合はwasm_cache_misses_totalがインクリメントされる必要がある。

**Validates: Requirements 8.5, 8.6**

### Property 20: Route優先度による選択

*任意の*HTTPリクエストに対して、複数のRouteがマッチする場合、最も高いpriorityフィールドを持つRouteが選択される必要がある。

**Validates: Requirements 9.2**

### Property 21: POP Selectorによるフィルタリング

*任意の*Routeに対して、POP selectorが指定されている場合、条件に一致するEdge Nodeにのみルーティング情報が配信される必要がある。

**Validates: Requirements 9.3**

### Property 22: Route変更時の通知

*任意の*Route作成または更新に対して、影響を受けるEdge Nodeにルーティングテーブル更新通知が送信される必要がある。

**Validates: Requirements 9.4**

### Property 23: ビルドツールチェーンの選択

*任意の*ソースコードに対して、Build Workerは言語（Rust、TypeScript/JavaScript）に応じた適切なツールチェーンを使用する必要がある。

**Validates: Requirements 10.3**

### Property 24: ビルド成功時のArtifactアップロード

*任意の*ビルド成功に対して、Build WorkerはWASMファイルのSHA256ハッシュを計算し、Artifact Storeにアップロードする必要がある。

**Validates: Requirements 10.4**

### Property 25: Canaryデプロイの段階的ロールアウト

*任意の*Canaryデプロイに対して、最初に1つのEdge Nodeにのみデプロイされ、成功後に段階的に残りのノードにデプロイされる必要がある。

**Validates: Requirements 11.1, 11.2**

### Property 26: Canary失敗時のロールアウト停止

*任意の*Canary Edge Nodeに対して、エラー率が閾値を超える場合、ロールアウトが停止される必要がある。

**Validates: Requirements 11.3**

### Property 27: ロールバック時の切り替え指示

*任意の*ロールバック要求に対して、全Edge Nodeに前バージョンへの切り替え指示が送信される必要がある。

**Validates: Requirements 11.4**

### Property 28: mTLS認証の実行

*任意の*Edge NodeからCPへの接続に対して、mTLSハンドシェイクが実行され、ノード証明書が検証される必要がある。

**Validates: Requirements 12.1, 12.2**

### Property 29: JWT トークンの発行と使用

*任意の*mTLS検証成功に対して、CPはJWTトークンを発行し、Edge NodeはAPI呼び出し時にトークンを含める必要がある。

**Validates: Requirements 12.3, 12.4**

### Property 30: 構造化ログの出力

*任意の*WASM関数からのlog呼び出しに対して、Edge Nodeはログレベル、メッセージ、Function ID、Request ID、タイムスタンプを含む構造化JSONログを出力する必要がある。

**Validates: Requirements 13.1**

### Property 31: ログのレート制限

*任意の*WASM関数に対して、1秒間に1000件を超えるログ出力が試みられた場合、レート制限が適用され超過分が破棄される必要がある。

**Validates: Requirements 13.4**

### Property 32: 監査ログの記録

*任意の*Function登録、更新、デプロイ操作に対して、ユーザーID、リソースID、操作タイプ、タイムスタンプを含む監査ログが記録される必要がある。

**Validates: Requirements 14.1, 14.2, 14.3**

### Property 33: 監査ログの不変性

*任意の*監査ログレコードに対して、一度記録されたログは削除や変更ができない必要がある。

**Validates: Requirements 14.4**

### Property 34: キャッシュLRU Eviction

*任意の*ローカルキャッシュに対して、サイズまたはファイル数が上限に達する場合、LRUポリシーに基づいて最も古いWASMファイルが削除される必要がある。

**Validates: Requirements 15.1, 15.2**

### Property 35: バージョン管理の独立性

*任意の*新バージョンのWASMファイル追加に対して、古いバージョンは自動的に削除されず、LRU evictionに従って管理される必要がある。

**Validates: Requirements 15.3**

## エラーハンドリング

### Control Plane エラー処理

#### Function登録エラー
- **無効なメモリページ数**: HTTP 400 Bad Request、エラーメッセージ「Memory pages must be between 1 and 256」
- **重複するFunction名とバージョン**: HTTP 409 Conflict、エラーメッセージ「Function with name and version already exists」
- **Artifact Storeアップロード失敗**: HTTP 500 Internal Server Error、再試行後にエラーログ記録

#### デプロイエラー
- **存在しないFunction ID**: HTTP 404 Not Found
- **ターゲットPOPにEdge Nodeが存在しない**: HTTP 400 Bad Request、エラーメッセージ「No edge nodes available in target POP」
- **通知送信失敗**: 永続キューに保存、指数バックオフで再試行

#### ビルドエラー
- **コンパイルエラー**: ビルドログをデータベースに保存、開発者に通知（webhook、メール）
- **依存関係解決失敗**: エラーログ記録、ビルドステータスを「failed」に更新

### Edge Node エラー処理

#### Artifact取得エラー
- **SHA256不一致**: ファイル破棄、CPにデプロイ失敗ステータス報告、エラーログ記録
- **ネットワークエラー**: 指数バックオフで再試行（最大5回）、失敗後にCPにエラー報告
- **ディスク容量不足**: 古いキャッシュファイルを削除、再試行

#### 実行エラー
- **Route未発見**: HTTP 404 Not Found、エラーログ記録
- **リクエストパースエラー**: HTTP 400 Bad Request
- **WASM実行タイムアウト**: HTTP 504 Gateway Timeout、メトリクス記録
- **WASM実行エラー**: HTTP 500 Internal Server Error、エラーログとメトリクス記録
- **メモリ制限超過**: 実行終了、HTTP 500 Internal Server Error

#### 通信エラー
- **CP接続失敗**: 指数バックオフで再接続試行、ローカルキャッシュで継続動作
- **Heartbeat送信失敗**: 再試行、接続回復まで継続

### エラーログ形式

```json
{
  "timestamp": "2024-01-15T10:30:45.123Z",
  "level": "error",
  "component": "edge-runner",
  "node_id": "tokyo-1-node-01",
  "function_id": "func-123",
  "request_id": "req-456",
  "error_type": "execution_timeout",
  "error_message": "Function execution exceeded max_execution_ms (500ms)",
  "stack_trace": "...",
  "context": {
    "max_execution_ms": 500,
    "actual_execution_ms": 523
  }
}
```

## テスト戦略

### ユニットテスト

#### Control Plane
- **Function Manager**: CRUD操作、バリデーション、バージョン管理
- **Route Manager**: ルート作成、優先度ソート、POP selector評価
- **Notification Service**: 通知キューイング、再試行ロジック
- **Build Worker**: ツールチェーン選択、SHA256計算、Artifactアップロード

#### Edge Node
- **Function Router**: ルートマッチング、優先度選択
- **WasmEdge Runtime Manager**: インスタンス作成、プール管理、LRU eviction
- **Local Cache Manager**: ファイル保存、SHA256検証、LRU eviction
- **CP Communication Client**: gRPC通信、Heartbeat送信、再接続ロジック

### プロパティベーステスト

プロパティベーステストには**fast-check**（TypeScript/JavaScript）または**QuickCheck**（Rust）を使用します。各テストは最低**100回**の反復を実行します。

#### CP プロパティテスト
- **Property 1**: 任意の有効なFunction登録リクエストに対して、全必須フィールドが保存される
- **Property 2**: 任意のWASMファイルに対して、SHA256ハッシュが一貫している
- **Property 3**: 任意のデプロイ通知に対して、全必須フィールドが含まれる
- **Property 15**: 任意の失敗した通信に対して、再試行間隔が指数的に増加する
- **Property 20**: 任意のHTTPリクエストに対して、最も高い優先度のRouteが選択される
- **Property 21**: 任意のRouteに対して、POP selectorに一致するノードにのみ配信される
- **Property 23**: 任意のソースコードに対して、適切なツールチェーンが選択される
- **Property 25**: 任意のCanaryデプロイに対して、段階的ロールアウトが実行される
- **Property 32**: 任意の操作に対して、監査ログが記録される
- **Property 33**: 任意の監査ログに対して、削除や変更ができない

#### Edge Node プロパティテスト
- **Property 4**: 任意のデプロイに対して、SHA256一致時のみキャッシュに保存される
- **Property 5**: 任意のHTTPリクエストに対して、ルーティングが決定的である
- **Property 6**: 任意のキャッシュミスに対して、Artifact Storeから取得される
- **Property 7**: 任意の実行リクエストに対して、Hot Instanceが優先的に再利用される
- **Property 8**: 任意の実行完了に対して、プールサイズ制限が遵守される
- **Property 9**: 任意のメモリ圧迫に対して、LRU evictionが正確に実行される
- **Property 10**: 任意の実行に対して、メモリページ制限が適用される
- **Property 11**: 任意の実行に対して、タイムアウトが強制される
- **Property 12**: 任意のWASM関数に対して、WASI機能がホワイトリスト制御される
- **Property 13**: 任意のHeartbeatに対して、全必須フィールドが含まれる
- **Property 16-19**: 任意の実行に対して、メトリクスが正しく記録される
- **Property 30**: 任意のlog呼び出しに対して、構造化ログが出力される
- **Property 31**: 任意のログ出力に対して、レート制限が適用される
- **Property 34**: 任意のキャッシュに対して、LRU evictionが実行される
- **Property 35**: 任意の新バージョン追加に対して、古いバージョンが即座に削除されない

### 統合テスト

#### End-to-End フロー
1. **デプロイフロー**: Function登録 → ビルド → Artifact保存 → 通知 → Edge取得 → キャッシュ保存
2. **実行フロー**: HTTPリクエスト → ルーティング → キャッシュ確認 → WASM実行 → レスポンス返却
3. **Canaryデプロイフロー**: Canary指定 → 1ノードデプロイ → メトリクス確認 → 段階的ロールアウト
4. **ロールバックフロー**: エラー検出 → ロールバック要求 → 全ノード切り替え

#### パフォーマンステスト
- **Cold Start レイテンシ**: P99 < 50ms
- **Hot Execution レイテンシ**: P99 < 5ms
- **スループット**: 1ノードあたり1000 req/s
- **同時実行**: 100並行リクエスト

### テスト環境

#### ローカル開発
- Docker Compose: PostgreSQL、MinIO、Redis
- モックEdge Node: 1ノード
- モックWASM関数: Hello World、Echo

#### ステージング
- Kubernetes: CP（3レプリカ）、Edge Node（5ノード）
- 実際のPostgreSQL、MinIO、Redis
- 実際のWASM関数: 複数言語（Rust、TypeScript）

#### 本番
- Kubernetes: CP（高可用性）、Edge Node（地理的分散）
- マネージドサービス: RDS PostgreSQL、S3、ElastiCache Redis
- 監視: Prometheus、Grafana、Loki、OpenTelemetry

## セキュリティ考慮事項

### 認証・認可

#### 開発者認証
- **方式**: OAuth 2.0（GitHub、Google）またはメール/パスワード
- **トークン**: JWT（有効期限: 24時間）
- **RBAC**: プロジェクト単位の権限管理（owner、developer、viewer）

#### Edge Node認証
- **方式**: mTLS（相互TLS認証）
- **証明書**: ノード固有の証明書（有効期限: 1年、自動更新）
- **トークン**: 長期有効JWT（有効期限: 30日、自動更新）

### 通信セキュリティ

#### CP ↔ Edge Node
- **プロトコル**: gRPC over TLS 1.3
- **認証**: mTLS + JWT
- **暗号化**: AES-256-GCM

#### Developer ↔ CP
- **プロトコル**: HTTPS（TLS 1.3）
- **認証**: JWT Bearer Token
- **CORS**: ホワイトリスト方式

### WASM サンドボックス

#### メモリ隔離
- 各WASMインスタンスは独立したリニアメモリを持つ
- 最大ページ数: 256ページ（16 MiB）
- インスタンス間でメモリ共有なし

#### WASI 機能制限
- **許可**: logging、clocks、random、metrics、kv（オプション）
- **禁止**: sockets、filesystem（書き込み）、process

#### リソース制限
- **CPU時間**: max_execution_ms（デフォルト: 500ms）
- **メモリ**: memory_pages * 64 KiB（デフォルト: 1 MiB）
- **ネットワーク**: http_fetchホスト関数のみ（URLホワイトリスト）

### Artifact 整合性

#### ビルド時
- SHA256ハッシュ計算
- Artifact Storeへの署名付きアップロード
- メタデータへのSHA256保存

#### デプロイ時
- Artifact Storeからのダウンロード
- SHA256検証（不一致時は拒否）
- ローカルキャッシュへの保存

### 監査とコンプライアンス

#### 監査ログ
- 全操作（Function登録、更新、デプロイ）を記録
- 不可逆的な保存（削除・変更不可）
- 保持期間: 最低1年

#### コンプライアンス
- GDPR: 個人データの暗号化、削除権の実装
- SOC 2: アクセス制御、監査ログ、暗号化
- ISO 27001: セキュリティポリシー、リスク管理

## パフォーマンス最適化

### Hot/Cold Instance 戦略

#### Hot Pool 設定
- **min_hot_instances**: 1（常に1インスタンスを維持）
- **max_hot_instances**: 10（Function IDごと）
- **idle_timeout**: 5分（アイドル後に破棄）
- **warmup_on_deploy**: true（デプロイ時に事前ウォームアップ）

#### Cold Start 最適化
- AOTコンパイル（事前コンパイル）の使用
- WASMモジュールサイズの最小化（< 1 MiB推奨）
- 依存関係の削減

### キャッシュ階層

#### L1: Hot Instance Pool（メモリ）
- アクセス時間: < 1ms
- 容量: 100インスタンス（メモリ制限による）

#### L2: Local File Cache（ディスク）
- アクセス時間: 1-5ms
- 容量: 10 GB
- Eviction: LRU

#### L3: Artifact Store（S3/MinIO）
- アクセス時間: 50-200ms
- 容量: 無制限
- 整合性: SHA256検証

### ネットワーク最適化

#### gRPC 最適化
- HTTP/2多重化
- Keepalive（30秒）
- 圧縮（gzip）

#### Artifact 配信最適化
- CDN（CloudFront、Fastly）の使用
- Presigned URL（有効期限: 1時間）
- 並列ダウンロード

### データベース最適化

#### インデックス
- functions: (name, version)、(owner_project_id)
- routes: (host, path)、(function_id)、(priority DESC)
- nodes: (pop_id)、(status)、(last_heartbeat DESC)

#### クエリ最適化
- Prepared statements
- Connection pooling（最大100接続）
- Read replica（読み取り専用クエリ）

## デプロイメント

### Control Plane デプロイ

#### Kubernetes マニフェスト
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: control-plane
spec:
  replicas: 3
  selector:
    matchLabels:
      app: control-plane
  template:
    metadata:
      labels:
        app: control-plane
    spec:
      containers:
      - name: api-server
        image: wasmedge-cp:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: cp-secrets
              key: database-url
        - name: MINIO_ENDPOINT
          value: "minio:9000"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
```

### Edge Node デプロイ

#### Systemd サービス
```ini
[Unit]
Description=WasmEdge Edge Runner
After=network.target

[Service]
Type=simple
User=edge-runner
ExecStart=/usr/local/bin/edge-runner \
  --config /etc/edge-runner/config.yaml \
  --node-id tokyo-1-node-01 \
  --cp-endpoint https://cp.example.com:8080
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### 監視とアラート

#### Prometheus アラート
```yaml
groups:
- name: edge-runner
  rules:
  - alert: HighErrorRate
    expr: rate(wasm_invoke_errors_total[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High error rate on {{ $labels.node_id }}"
  
  - alert: HighLatency
    expr: histogram_quantile(0.99, wasm_invoke_latency_seconds_bucket) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High P99 latency on {{ $labels.node_id }}"
```

## 今後の拡張

### 短期（3ヶ月）
- WebAssembly Component Model対応
- より細かいWASI機能制御
- ローカルKVストア（libSQL）統合

### 中期（6ヶ月）
- Firecracker統合（高隔離要件）
- マルチテナント強化（namespace分離）
- A/Bテスト機能

### 長期（12ヶ月）
- エッジ間通信（Edge Mesh）
- 機械学習推論ワークロード対応
- ステートフルFunction（Durable Functions）
