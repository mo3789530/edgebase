# WasmEdge エッジ関数基盤 仕様書

## 1. 概要（目的）

WasmEdge ランタイムを用いて、POP（エッジノード群）上で多量の短命 HTTP ハンドラ関数（WASM モジュール）を低遅延かつ安全に実行するプラットフォームを構築する。

設計は Pull ベース（Control Plane が通知 → Edge が S3/MinIO から WASM を取得）を基本とし、エッジ側は軽量な Go 実装の Runner を用いる。

## 2. 用語定義

- **CP**: Control Plane（関数管理、ビルド、Artifact 保管、通知）
- **Edge Node / Runner**: POP 上で実行されるエージェント（HTTP サーバ + WasmEdge 埋め込み）
- **Artifact Store**: MinIO / S3 互換ストレージ（WASM バイナリ格納）
- **Function**: デプロイ対象の WASM モジュール（メタ情報を伴う）
- **Route**: HTTP パス -> Function マッピング
- **Hot Instance**: メモリに常駐して何度も再呼び出しする Wasm インスタンス
- **Cold Start**: インスタンス生成から初回呼び出しまで

## 3. ハイレベルアーキテクチャ

```
[Developer] -> (push source) -> [Control Plane]

[Control Plane] -> build -> artifact (s3/minio) + metadata in DB
[Control Plane] -> notify -> [Edge Runner] (gRPC/WebSocket)

[Edge Runner] -> pull artifact -> local cache -> WasmEdge execute

Requests -> Edge Runner HTTP -> dispatch to cached Wasm

Metrics/Logs -> Control Plane (or push to Prometheus/Loki)
```

## 4. システムデザイン詳細

### 4.1 全体アーキテクチャ図

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Developer Zone                              │
│  ┌──────────┐      ┌──────────┐      ┌──────────┐                  │
│  │   Git    │      │   CLI    │      │ Web UI   │                  │
│  │  Repo    │      │  Tool    │      │ Console  │                  │
│  └────┬─────┘      └────┬─────┘      └────┬─────┘                  │
│       │                 │                   │                        │
│       └─────────────────┴───────────────────┘                        │
│                         │                                            │
└─────────────────────────┼────────────────────────────────────────────┘
                          │ HTTPS/API
                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       Control Plane (CP)                             │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                      API Gateway (TLS)                         │ │
│  │              /functions  /deploy  /routes  /nodes              │ │
│  └──────┬──────────────────────────────────────────────┬──────────┘ │
│         │                                               │            │
│  ┌──────▼──────────┐  ┌──────────────┐  ┌─────────────▼─────────┐  │
│  │  Function Mgr   │  │ Build Worker │  │   Notification Svc    │  │
│  │  - Register     │  │ - Rust/TS    │  │   - gRPC Stream       │  │
│  │  - Versioning   │  │ - AOT        │  │   - WebSocket         │  │
│  │  - Routing      │  │ - Validation │  │   - Push Queue        │  │
│  └────────┬────────┘  └──────┬───────┘  └───────────┬───────────┘  │
│           │                  │                       │              │
│  ┌────────▼──────────────────▼───────┐  ┌───────────▼───────────┐  │
│  │      PostgreSQL (Metadata)        │  │   Redis (Queue/Cache) │  │
│  │  - functions, routes, nodes       │  │   - Pending deploys   │  │
│  │  - deployments, audit_logs        │  │   - Rate limiting     │  │
│  └───────────────────────────────────┘  └───────────────────────┘  │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │              MinIO / S3 (Artifact Store)                     │  │
│  │         s3://artifacts/functions/{id}/{version}.wasm         │  │
│  │         + sha256 checksums, presigned URLs                   │  │
│  └──────────────────────────────────────────────────────────────┘  │
└──────────────────────────────┬───────────────────────────────────────┘
                               │ gRPC/WebSocket (mTLS)
                               │ Heartbeat + Deploy Notifications
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Edge Network (POP)                           │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    Edge Node (Tokyo-1)                       │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │              HTTP Server (Go)                          │  │  │
│  │  │  - TLS termination                                     │  │  │
│  │  │  - Route matching (host + path)                        │  │  │
│  │  │  - Request validation                                  │  │  │
│  │  └──────────────────┬─────────────────────────────────────┘  │  │
│  │                     │                                         │  │
│  │  ┌──────────────────▼─────────────────────────────────────┐  │  │
│  │  │           Function Router & Dispatcher               │  │  │
│  │  │  - Route lookup                                       │  │  │
│  │  │  - Function resolution                                │  │  │
│  │  │  - Load balancing (hot instances)                     │  │  │
│  │  └──────────────────┬─────────────────────────────────────┘  │  │
│  │                     │                                         │  │
│  │  ┌──────────────────▼─────────────────────────────────────┐  │  │
│  │  │          WasmEdge Runtime Manager                     │  │  │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐            │  │  │
│  │  │  │ Hot Pool │  │ Hot Pool │  │ Hot Pool │            │  │  │
│  │  │  │ func-A   │  │ func-B   │  │ func-C   │            │  │  │
│  │  │  │ (reuse)  │  │ (reuse)  │  │ (reuse)  │            │  │  │
│  │  │  └──────────┘  └──────────┘  └──────────┘            │  │  │
│  │  │                                                        │  │  │
│  │  │  - Memory limit enforcement (pages)                   │  │  │
│  │  │  - Timeout enforcement (context)                      │  │  │
│  │  │  - WASI capability control                            │  │  │
│  │  │  - LRU eviction policy                                │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  │                                                               │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │              Local Cache Manager                      │  │  │
│  │  │  /var/cache/wasm/{function_id}/{version}.wasm         │  │  │
│  │  │  - SHA256 verification                                │  │  │
│  │  │  - LRU eviction (size/count limits)                   │  │  │
│  │  │  - Atomic updates                                     │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  │                                                               │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │         CP Communication Client                       │  │  │
│  │  │  - gRPC bidirectional stream / WebSocket              │  │  │
│  │  │  - Heartbeat (30s interval)                           │  │  │
│  │  │  - Deploy notification receiver                       │  │  │
│  │  │  - Exponential backoff retry                          │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  │                                                               │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │         Observability Exporter                        │  │  │
│  │  │  - Prometheus metrics (:9090/metrics)                 │  │  │
│  │  │  - Structured JSON logs                               │  │  │
│  │  │  - OpenTelemetry traces                               │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────────┐   │
│  │ Edge (Tokyo-2)   │  │ Edge (Osaka-1)   │  │ Edge (US-West)  │   │
│  │ (same structure) │  │ (same structure) │  │ (same structure)│   │
│  └──────────────────┘  └──────────────────┘  └─────────────────┘   │
└──────────────────────────────────────────────────────────────────────┘
                               │
                               │ HTTP/HTTPS
                               ▼
                        ┌──────────────┐
                        │  End Users   │
                        └──────────────┘
```

### 4.2 データフロー設計

#### 4.2.1 Function デプロイフロー

```
┌──────────┐
│Developer │
└────┬─────┘
     │ 1. git push / CLI deploy
     ▼
┌─────────────────┐
│  CP API Server  │
└────┬────────────┘
     │ 2. Create function record (DB)
     │    Generate presigned upload URL
     ▼
┌─────────────────┐
│  Build Worker   │
└────┬────────────┘
     │ 3. Compile to WASM
     │    Run AOT (optional)
     │    Calculate SHA256
     ▼
┌─────────────────┐
│  MinIO/S3       │ 4. Upload artifact
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│  PostgreSQL     │ 5. Update function record
└────┬────────────┘    (artifact_url, sha256)
     │
     ▼
┌─────────────────┐
│ Notification    │ 6. Push deploy notification
│    Service      │    to target edge nodes
└────┬────────────┘
     │ gRPC/WebSocket
     ▼
┌─────────────────┐
│  Edge Runners   │ 7. Receive notification
│  (Tokyo-1,2,..) │    Pull artifact from S3
└─────────────────┘    Verify SHA256
                       Cache locally
                       Report status to CP
```

#### 4.2.2 Request 実行フロー

```
┌──────────┐
│End User  │
└────┬─────┘
     │ HTTP Request
     │ GET /api/hello
     ▼
┌─────────────────────────────────┐
│  Edge Runner (HTTP Server)      │
└────┬────────────────────────────┘
     │ 1. TLS termination
     │ 2. Route matching
     │    (host + path -> function_id)
     ▼
┌─────────────────────────────────┐
│  Function Router                │
└────┬────────────────────────────┘
     │ 3. Check local cache
     │    function_id exists?
     ▼
     ├─ YES ─┐              ├─ NO ─┐
     │        │              │       │
     │        ▼              │       ▼
     │  ┌─────────────┐     │  ┌──────────────┐
     │  │ Hot Pool?   │     │  │ Pull from S3 │
     │  └──┬──────────┘     │  │ Verify SHA   │
     │     │                │  │ Cache local  │
     │     ├─ YES ─┐        │  └──────┬───────┘
     │     │        │        │         │
     │     │        ▼        │         ▼
     │     │  ┌──────────┐  │    ┌──────────┐
     │     │  │ Reuse VM │  │    │ Cold     │
     │     │  │ (fast)   │  │    │ Start    │
     │     │  └────┬─────┘  │    └────┬─────┘
     │     │       │        │         │
     │     ├─ NO ─┐│        │         │
     │     │      ││        │         │
     │     ▼      ││        │         │
     │  ┌─────────┘│        │         │
     │  │ Create   │        │         │
     │  │ New VM   │        │         │
     │  └────┬─────┘        │         │
     │       │              │         │
     └───────┴──────────────┴─────────┘
             │
             ▼
┌─────────────────────────────────┐
│  WasmEdge Runtime               │
│  - Set memory limit (pages)     │
│  - Set timeout (context)        │
│  - Restrict WASI capabilities   │
└────┬────────────────────────────┘
     │ 4. Execute WASM function
     │    entrypoint(request) -> response
     ▼
┌─────────────────────────────────┐
│  Response Handler               │
│  - Collect metrics              │
│  - Log execution                │
│  - Update hot pool (LRU)        │
└────┬────────────────────────────┘
     │ 5. Return HTTP response
     ▼
┌──────────┐
│End User  │
└──────────┘
```

#### 4.2.3 Heartbeat & Sync フロー

```
┌─────────────────┐
│  Edge Runner    │
└────┬────────────┘
     │ Every 30s
     │ POST /api/v1/nodes/:id/heartbeat
     │ {status, cpu, mem, functions[]}
     ▼
┌─────────────────┐
│  CP API Server  │
└────┬────────────┘
     │ 1. Update last_heartbeat
     │ 2. Check pending notifications
     ▼
     ├─ Has pending deploy? ─┐
     │                        │
     │ YES                    │ NO
     │                        │
     ▼                        ▼
┌──────────────────┐    ┌─────────┐
│ Return deploy    │    │ Return  │
│ instructions:    │    │ 200 OK  │
│ - function_id    │    └─────────┘
│ - version        │
│ - artifact_url   │
│ - sha256         │
└────┬─────────────┘
     │
     ▼
┌─────────────────┐
│  Edge Runner    │ 3. Process instructions
│  - Pull artifact│    asynchronously
│  - Verify SHA   │
│  - Update cache │
│  - Report back  │
└─────────────────┘
```

### 4.3 コンポーネント間インターフェース設計

#### 4.3.1 CP ↔ Edge Runner (gRPC)

```protobuf
service EdgeControl {
  // Bidirectional stream for real-time communication
  rpc StreamControl(stream EdgeMessage) returns (stream ControlMessage);
  
  // Heartbeat (alternative to stream)
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
  string status = 2; // online, degraded, offline
  double cpu_percent = 3;
  int64 mem_bytes = 4;
  repeated FunctionStatus functions = 5;
}

message FunctionStatus {
  string function_id = 1;
  string version = 2;
  string state = 3; // cached, loading, error
  int64 invocation_count = 4;
  double avg_latency_ms = 5;
}
```

#### 4.3.2 Edge Runner ↔ WasmEdge (Host Functions)

```rust
// Host functions exposed to WASM modules

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

#### 4.3.3 WASM Module Interface (Expected Exports)

```rust
// Standard HTTP handler interface
#[no_mangle]
pub extern "C" fn handle(
    method_ptr: *const u8, method_len: usize,
    path_ptr: *const u8, path_len: usize,
    headers_ptr: *const u8, headers_len: usize,
    body_ptr: *const u8, body_len: usize,
    response_ptr: *mut u8, response_cap: usize
) -> i32; // Returns response length or negative error code

// Alternative: WASI HTTP interface (future)
// Uses component model with typed interfaces
```

### 4.4 セキュリティ設計詳細

#### 4.4.1 認証・認可フロー

```
Developer Authentication:
┌──────────┐
│Developer │
└────┬─────┘
     │ 1. Login (email/password or OAuth)
     ▼
┌─────────────┐
│  CP Auth    │ 2. Validate credentials
│  Service    │    Check RBAC (project membership)
└────┬────────┘
     │ 3. Issue JWT
     │    {sub: user_id, projects: [...], exp: ...}
     ▼
┌──────────┐
│Developer │ 4. Include JWT in API requests
└──────────┘    Authorization: Bearer <token>

Edge Node Authentication:
┌─────────────┐
│ Edge Runner │
└────┬────────┘
     │ 1. On startup, load node certificate + key
     ▼
┌─────────────┐
│  CP Auth    │ 2. mTLS handshake
│  Service    │    Verify node certificate
└────┬────────┘    Check node_id in allowlist
     │ 3. Issue node token (long-lived JWT)
     ▼
┌─────────────┐
│ Edge Runner │ 4. Use token for API calls
└─────────────┘    Include in gRPC metadata
```

#### 4.4.2 WASM サンドボックス制約

```
Memory Isolation:
- Each WASM instance has isolated linear memory
- Max pages enforced by host (e.g., 16 pages = 1 MiB)
- No shared memory between functions

WASI Capabilities (Whitelist):
✓ Allowed:
  - wasi:logging/logging
  - wasi:clocks/wall-clock
  - wasi:random/random
  - custom:metrics/post
  - custom:kv/get-put (if enabled)

✗ Denied:
  - wasi:sockets/* (raw network access)
  - wasi:filesystem/* (except read-only config)
  - wasi:process/* (no exec/spawn)

Execution Limits:
- CPU time: max_execution_ms (enforced by host timeout)
- Memory: memory_pages * 64 KiB
- Network: Only via host functions (http_fetch with allowlist)
- File I/O: None (or read-only mounted config)

Resource Quotas (per tenant):
- Max concurrent executions: 100
- Max invocations/minute: 10,000
- Max total CPU time/day: 1 hour
```

#### 4.4.3 Artifact 整合性検証

```
Build Phase:
┌──────────────┐
│Build Worker  │
└────┬─────────┘
     │ 1. Compile WASM
     │ 2. Calculate SHA256
     │    sha256sum function.wasm
     ▼
┌──────────────┐
│  MinIO/S3    │ 3. Upload artifact
└────┬─────────┘    PUT /artifacts/{id}/{version}.wasm
     │
     ▼
┌──────────────┐
│ PostgreSQL   │ 4. Store metadata
└──────────────┘    INSERT INTO functions
                    (id, version, artifact_url, sha256)

Deploy Phase:
┌──────────────┐
│ Edge Runner  │
└────┬─────────┘
     │ 1. Receive deploy notification
     │    {artifact_url, sha256}
     ▼
┌──────────────┐
│  MinIO/S3    │ 2. Download artifact
└────┬─────────┘    GET /artifacts/{id}/{version}.wasm
     │
     ▼
┌──────────────┐
│ Edge Runner  │ 3. Verify SHA256
└────┬─────────┘    calculated_sha256 == expected_sha256?
     │
     ├─ YES ─┐              ├─ NO ─┐
     │        │              │       │
     │        ▼              │       ▼
     │  ┌──────────┐        │  ┌──────────┐
     │  │ Cache    │        │  │ Reject   │
     │  │ locally  │        │  │ Report   │
     │  └──────────┘        │  │ error    │
     │                      │  └──────────┘
     │                      │
     └──────────────────────┘
```

### 4.5 パフォーマンス設計

#### 4.5.1 Hot/Cold Instance 管理戦略

```
Hot Pool Management:
┌─────────────────────────────────────┐
│  Hot Instance Pool (per function)   │
│  ┌─────────┐  ┌─────────┐           │
│  │ Inst 1  │  │ Inst 2  │  ...      │
│  │ (ready) │  │ (ready) │           │
│  └─────────┘  └─────────┘           │
│                                      │
│  Config:                             │
│  - min_hot_instances: 1              │
│  - max_hot_instances: 10             │
│  - idle_timeout: 5 minutes           │
│  - warmup_on_deploy: true            │
└─────────────────────────────────────┘

Request arrives:
1. Check hot pool for available instance
2. If available: reuse (< 1ms overhead)
3. If not: create new instance (cold start: 10-50ms)
4. After execution: return to pool or destroy based on:
   - Pool size < max_hot_instances: keep
   - Pool size >= max: destroy
   - Idle > idle_timeout: destroy

LRU Eviction (when memory pressure):
- Track last_used timestamp per instance
- Evict least recently used when:
  - Total memory > threshold (e.g., 80% of limit)
  - Need space for new function
```

#### 4.5.2 キャッシュ階層

```
L1: Hot Instance Pool (Memory)
- WASM modules loaded in WasmEdge VM
- Fastest access (< 1ms)
- Limited by memory (e.g., 100 instances)

L2: Local File Cache (Disk)
- /var/cache/wasm/{function_id}/{version}.wasm
- Fast access (1-5ms to load)
- Limited by disk space (e.g., 10 GB)
- LRU eviction policy

L3: Artifact Store (S3/MinIO)
- Authoritative source
- Slower access (50-200ms)
- Unlimited storage
- Accessed on cache miss

Cache Coherence:
- Version-based: Each version is immutable
- SHA256 verification on L3 -> L2 transfer
- No invalidation needed (new version = new cache entry)
```

#### 4.5.3 レイテンシ目標

```
Cold Start (P99): < 50ms
- Download from cache: 5ms
- Load WASM module: 10ms
- Instantiate VM: 20ms
- First execution: 15ms

Hot Execution (P99): < 5ms
- Route lookup: 0.1ms
- Instance selection: 0.1ms
- Function call: 2-4ms
- Response serialization: 0.5ms

End-to-End (P99): < 100ms
- TLS handshake: 20ms (reused)
- Request parsing: 1ms
- Execution: 5-50ms (hot/cold)
- Response: 1ms
- Network: 20-50ms (depends on location)
```

## 5. コンポーネント一覧と役割

### CP API Server (Go/Rust)
- `/functions`, `/deploy`, `/routes`, `/nodes`, `/metrics`
- ビルドパイプライン、Artifact管理、通知配信（gRPC/WebSocket）
- DB: PostgreSQL（メタデータ）
- Artifact Store: MinIO

### Edge Runner (Go)
- HTTP サーバ（ルーティング）
- WasmEdge 埋め込み（wasmedge-go）
- キャッシュマネージャ（WASMファイル、LRU）
- CP 通信クライアント（gRPC または WebSocket）
- メトリクスエクスポーター（Prometheus）

### Build Worker
- esbuild / wasm-pack / Rust toolchain を用いてビルド
- AOT を必要に応じて実行

### Observability
- Prometheus (metrics), Grafana (dashboards), Loki (logs)

### Auth / IAM
- JWT / API Keys + RBAC（プロジェクト単位）

## 5. データモデル（抜粋）

### Function (functions table)
```
id: uuid
name: string
version: semver
owner_project_id: uuid
runtime: enum {wasm, wasm-aot}
entrypoint: string (exported func name or wasm http shim)
artifact_url: s3://bucket/path.wasm
sha256: hex
memory_pages: int (WASM pages, 64KiB/page)
max_execution_ms: int
created_at, updated_at
```

### Route (routes table)
```
id: uuid
host: string
path: string (path pattern)
function_id: uuid
methods: [GET,POST,...]
pop_selector: expression (e.g. region=="tokyo")
priority: int
```

### Node (nodes table)
```
id: uuid
pop_id: string
ip: string
last_heartbeat: timestamp
status: enum {online, degraded, offline}
metadata: json (cpu,mem,labels)
```

## 6. API 仕様（主要なもの）

### 6.1 Function登録

**POST /api/v1/functions**

Request (multipart/form-data or json pointing to git/zip):
```json
{
  "name": "hello",
  "owner_project_id": "...",
  "entrypoint": "handle",
  "runtime": "wasm",
  "max_execution_ms": 500,
  "memory_pages": 16,
  "pop_selector": "region=='tokyo'"
}
```

Response: function resource with presigned artifact upload URL.

ビルド後、CP は artifact を MinIO に置き、artifact_url と sha256 を登録。

### 6.2 デプロイ（実体は「特定バージョンを特定POPへ有効化」）

**POST /api/v1/functions/:id/deploy**

Body:
```json
{
  "version": "1.2.0",
  "target_pop": ["jp-t1", "jp-t2"],
  "strategy": "pull"
}
```

Response: deployment task id

動作: CP は nodes を選び、各 node に通知（gRPC push）を行う（通知は function_id, artifact_url, sha256, version, memory_pages, max_execution_ms を含む）

### 6.3 Node Heartbeat（Edge → CP）

**POST /api/v1/nodes/:id/heartbeat**

Body:
```json
{
  "node_id": "..",
  "status": "online",
  "cpu": 12.3,
  "mem_bytes": 134217728,
  "functions": [
    {"id": "..", "version": "1.2.0", "state": "cached"}
  ]
}
```

CP は last_heartbeat を更新し、未配信の通知があればレスポンスで指示を返す（pull トリガーや immediate action）。

## 7. Edge Runner の振る舞い（詳細）

### 起動時
1. 認証トークン（CP 発行）を取得/設定して CP に登録
2. 初期 heartbeat -> CP から初期設定（routes, assigned functions）を受け取る
3. キャッシュ確認 → 必要な WASM を Artifact Store から pull（sha チェック）
4. 起動完了状態を CP に送信

### 受信リクエスト
1. HTTP 受信 → route lookup （host+path）
2. 該当 function がキャッシュにあるか確認
3. なければ同期（pull）してから実行

### 実行方法（hot/cold選択）
- **Hot**: すでに作成済み VM/instance の関数内で call（same VM reused）
- **Cold**: 新インスタンス生成 → exec → destroy or keep based on LRU

### 実行制御
- WasmEdge にて memory_pages を制限
- 実行タイムアウト enforced by host (context with timeout)
- WASI capability: ネットワーク/FS access をホワイトリスト化
- 結果返却 & metrics ログ送信

### キャッシュ管理
- LRU キャッシュポリシー（max_wasm_files, max_total_bytes）
- ファイル整合性は sha256 で保証。更新はバージョン単位で扱う。

## 8. Wasm 実行ポリシー（セキュリティ）

- **メモリページ上限**を memory_pages により厳格に設定（例: 16 pages = 1MiB）
- **Host 呼び出し**（WASI / host functions）は最小限に限定：
  - 允许: log, metrics.post, kv.get/put（必要時）
  - 禁止/制限: 任意の raw socket 書き込み、exec、mount（代替は CP 経由のサービス呼び出し）
- **実行タイムアウト**: max_execution_ms をホストで enforced（500ms〜数秒が標準）
- **Multi-tenant 分離**: namespace 毎にログ/metrics を分離、quota（呼び出し数/総実行時間/月）を適用

## 9. 通信プロトコル（CP ↔ Edge）

### 制御/通知
- gRPC（双方向ストリーム）または WebSocket（TLS）
- 利点: 双方向で即時通知が可能、切断→再接続耐性が必要
- Heartbeat: 定期 POST（30s 間隔推奨）＋即時通知
- Payload: JSON + base64 or小さいバイナリは直接 S3 参照

### 再試行方針
- 一時的失敗は指数バックオフ
- 重要通知は永続キュー（DBに保管）で再配送

## 10. Observability（必須メトリクス）

Edge Runner は Prometheus エクスポートを提供:

```
edge_node_up (gauge)
wasm_invoke_count_total{function_id}
wasm_invoke_latency_seconds_bucket{function_id}
wasm_invoke_errors_total{function_id, code}
wasm_cache_hits_total / misses_total
node_memory_bytes / node_cpu_percent
```

### ログ
- structured JSON logs including request_id, function_id, node_id, start/end timestamps, status, error_message

### トレース
- OpenTelemetry spans: CP → Edge → Function execution

## 11. リソース管理とスケジューリング方針

- Edge レベルでは「ランタイム(プロセス)数」を管理
- Runner は複数関数をホットで実行（同じランタイムインスタンスを再利用）することでメモリ使用を抑える
- **POP 配置ルール**: CP の pop_selector によりルーティング（例: region=="tokyo" and latency<30ms）
- **負荷時**: LRU による関数アンロード、最も古い hot instance を破棄

## 12. CI/CD と Build

1. Git push → CP が webhook 受信 → ビルドワーカー（esbuild / wasm-pack / Rust toolchain）を起動
2. ビルド成果物に対して AOT を実行（可能なら）
3. 成果物は MinIO に保存、sha256 を DB に登録
4. **自動テスト**: unit wasm, e2e: deploy → edge runner に配布 → http → golden response

## 13. セキュリティ設計

- CP と Edge 通信は mTLS（推奨） or TLS + JWT
- **Artifact 認証**: presigned URL + artifact に対して署名（CP は sha256 を持つ）
- **Audit log**: 関数の登録/変更/デプロイは不可逆の audit table に記録
- **コードスキャン**: ビルドステップで依存関係の脆弱性スキャン（特に npm）
- **Secret 管理**: CP が Secrets を保管し、Edge は実行時に secrets id を渡されるが直接 value を保存しない（必要時は Vault）

## 14. テスト & 受け入れ基準（Acceptance Criteria）

### MVP (最小受け入れ条件)
- CP: Function register → artifact stored → metadata persisted
- Edge Runner: CP からの通知を受け、WASM を pull してローカルにキャッシュできる
- HTTP リクエストで Function を呼び出し、正しいレスポンスが返る
- メトリクス（invoke count, latency）が Prometheus に出る
- 実行タイムアウトが enforcement される（例えば max_execution_ms = 200 → 超過は 504）

### セキュア動作
- WASI 権限制御が有効で、外部TCP/ファイル書き込みが禁止される
- Artifact の sha256 チェックが実装される

## 15. デプロイ戦略（ローリング）

- **Canary**: まず 1 ノードで new function/version を有効化、ログ・メトリクス確認
- **Gradual rollout**: CP が残りノードへ段階的に通知
- **Rollback**: 異常時 CP は undeploy 指示を出し Edge は古いバージョンに戻す

## 16. 拡張 & 選択肢メモ

- AOT（事前コンパイル）を取り入れると cold start と実行性能が向上
- Firecracker を用いる場合は「VM 上で Runner を動かす」方式が推奨（高隔離が必要な関数のみ）
- libSQL をローカルに配置して低レイテンシ DB を提供する選択肢あり（同期設計要検討）

## 17. マイルストーン（提案）

- **Week0**: 要件確認・設計レビュー（本仕様の承認）
- **Week1**: ローカル PoC（WasmEdge + Go runner minimal: load wasm + call exported func）
- **Week2**: HTTP→WASM ランナー + local cache + prometheus metrics
- **Week3**: CP minimal (function register, artifact upload via MinIO, simple DB) + deploy notification（WebSocket）
- **Week4**: Pull 配布完了 → e2e テスト（deploy → edge pull → request）
- **Week5**: セキュリティ（WASI policy）実装、AOT 評価
- **Week6**: Canary rollout 支援機能（versioning, rollback）
- **Week7~**: Load test & optimization、運用ドキュメント作成

## 18. 開発上の注意（実装の"肝"まとめ）

- **Hot vs Cold のバランス**：hot を過剰に持つとメモリ負荷、cold だとレイテンシ。LRU + warming（preload）戦略が鍵
- **実行制限はホスト側で必ず enforced**（WasmEdge の設定だけでなく、Go の context timeout でガード）
- **Artifact の一貫性**：sha256 チェックは必須。CP は整合性が取れない artifact を拒否する。
- **Observability を最初から組み込む**（trace id を request に必ず付与）

## 19. サンプル manifest（function.json）

```json
{
  "id": "uuid-xxxx",
  "name": "hello-world",
  "version": "1.0.0",
  "entrypoint": "handle",
  "artifact_url": "s3://edge-artifacts/hello-world-1.0.0.wasm",
  "sha256": "abcd1234...",
  "memory_pages": 16,
  "max_execution_ms": 500,
  "pop_selector": "region=='tokyo'"
}
```

## 20. 受け渡し可能な成果物（この仕様からすぐ作れる）

- **repo A**: edge-runner (Go) — Dockerfile + WasmEdge example + Prometheus exporter
- **repo B**: control-plane (Go/Rust) — API + MinIO integration + DB schema + simple UI
- **repo C**: build-worker (CI script + wasm build example)
- **terraform**: MinIO, PostgreSQL (dev), Prometheus stack for dev
