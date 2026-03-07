# Cluster Database Design

## 目的

各 k3s Cluster に配置される `Cluster Agent` が利用するローカル DB の設計を定義する。

この DB は Control Plane の集約 DB を置き換えるものではない。責務は以下に限定する。

- 最後に適用した desired state の記録
- sync plan の重複実行防止
- inventory のローカル保持
- 一時障害時の再試行
- cluster 内の operational state の短中期保存

## 結論

MVP では、各 Cluster のローカル DB は `SQLite` を採用する。

理由:
- Cluster Agent は基本 1 active instance
- ローカル状態は小さい
- セットアップが軽い
- agent 再起動後も state を保持したい

将来、同一 cluster 内で agent を active-active にするなら、`PostgreSQL` など外部 DB へ切り替える。

## DB の位置づけ

```text
Control Plane DB
  - source of truth
  - function / route / deployment target / cluster metadata

Cluster Local DB
  - applied state cache
  - sync execution state
  - inventory snapshots
  - retry queue
```

Control Plane が authoritative であり、Cluster DB は execution-side cache and journal として扱う。

## 保存しないもの

Cluster DB に以下は保存しない。

- tenant の最終 authoritative data
- function artifact 本体
- user account 情報
- 長期監査ログの正本
- secret の平文

## 推奨実装

- Engine: SQLite
- File path: `/var/lib/edgebase/cluster-agent/agent.db`
- Access pattern: single writer, small concurrent readers
- WAL mode: `ON`
- Foreign keys: `ON`
- Busy timeout: 5s 以上

## テーブル一覧

### 1. `agent_state`

Agent 自体のローカル状態。

用途:
- 最後に処理した generation
- 最終 heartbeat / inventory / sync 時刻
- agent version

```sql
CREATE TABLE agent_state (
  agent_id TEXT PRIMARY KEY,
  cluster_id TEXT NOT NULL,
  agent_version TEXT NOT NULL,
  last_heartbeat_at INTEGER,
  last_inventory_at INTEGER,
  last_sync_fetch_at INTEGER,
  last_applied_generation INTEGER DEFAULT 0,
  last_applied_sync_id TEXT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
```

### 2. `cluster_metadata`

cluster のローカルキャッシュ。

用途:
- cluster 名、環境、region、status のキャッシュ
- Control Plane 応答が一時的に不安定でも最低限の context を保持

```sql
CREATE TABLE cluster_metadata (
  cluster_id TEXT PRIMARY KEY,
  cluster_name TEXT NOT NULL,
  region TEXT,
  environment TEXT,
  status TEXT NOT NULL,
  kubernetes_version TEXT,
  token_ref TEXT,
  last_seen_cp_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
```

`token_ref` は secret 参照 ID のみで、token 本体は保存しない。

### 3. `desired_workloads`

Control Plane から見た cluster に適用されるべき desired state。

用途:
- 現在の desired deployment/service 定義
- diff 判定の基準

```sql
CREATE TABLE desired_workloads (
  id TEXT PRIMARY KEY,
  cluster_id TEXT NOT NULL,
  function_id TEXT NOT NULL,
  function_name TEXT NOT NULL,
  version TEXT NOT NULL,
  namespace TEXT NOT NULL,
  workload_type TEXT NOT NULL,
  desired_spec_json TEXT NOT NULL,
  desired_hash TEXT NOT NULL,
  generation INTEGER NOT NULL,
  active INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX idx_desired_workloads_unique
  ON desired_workloads(cluster_id, function_id, workload_type, namespace);

CREATE INDEX idx_desired_workloads_generation
  ON desired_workloads(cluster_id, generation);
```

### 4. `observed_workloads`

inventory から取得した実クラスタ状態。

用途:
- desired と observed の diff
- drift 検知
- apply 後の収束確認

```sql
CREATE TABLE observed_workloads (
  id TEXT PRIMARY KEY,
  cluster_id TEXT NOT NULL,
  namespace TEXT NOT NULL,
  workload_type TEXT NOT NULL,
  resource_name TEXT NOT NULL,
  function_id TEXT,
  version TEXT,
  observed_spec_json TEXT,
  observed_hash TEXT,
  ready_replicas INTEGER,
  available_replicas INTEGER,
  status TEXT,
  observed_at INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX idx_observed_workloads_unique
  ON observed_workloads(cluster_id, namespace, workload_type, resource_name);
```

### 5. `inventory_snapshots`

inventory の履歴。

用途:
- apply 前後比較
- cluster 障害調査
- 直近 N 件の状態保存

```sql
CREATE TABLE inventory_snapshots (
  id TEXT PRIMARY KEY,
  cluster_id TEXT NOT NULL,
  snapshot_type TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  observed_at INTEGER NOT NULL,
  created_at INTEGER NOT NULL
);

CREATE INDEX idx_inventory_snapshots_cluster_time
  ON inventory_snapshots(cluster_id, observed_at DESC);
```

`snapshot_type` は `nodes`, `deployments`, `services`, `pods`, `summary` を想定する。

### 6. `sync_plans`

取得した sync plan の記録。

用途:
- plan の重複適用防止
- plan generation の追跡

```sql
CREATE TABLE sync_plans (
  sync_id TEXT PRIMARY KEY,
  cluster_id TEXT NOT NULL,
  generation INTEGER NOT NULL,
  status TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  fetched_at INTEGER NOT NULL,
  started_at INTEGER,
  completed_at INTEGER,
  error_message TEXT
);

CREATE UNIQUE INDEX idx_sync_plans_generation
  ON sync_plans(cluster_id, generation);
```

`status`:
- `fetched`
- `running`
- `succeeded`
- `failed`
- `superseded`

### 7. `sync_actions`

plan 内 action ごとの実行状態。

用途:
- どの action で失敗したかを把握
- 部分再試行

```sql
CREATE TABLE sync_actions (
  id TEXT PRIMARY KEY,
  sync_id TEXT NOT NULL,
  cluster_id TEXT NOT NULL,
  action_order INTEGER NOT NULL,
  action_type TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_name TEXT NOT NULL,
  namespace TEXT,
  desired_hash TEXT,
  status TEXT NOT NULL,
  result_json TEXT,
  error_message TEXT,
  started_at INTEGER,
  completed_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  FOREIGN KEY(sync_id) REFERENCES sync_plans(sync_id)
);

CREATE UNIQUE INDEX idx_sync_actions_order
  ON sync_actions(sync_id, action_order);
```

### 8. `deployment_revisions`

cluster 内で実際に適用した revision を保持する。

用途:
- rollback
- 現在稼働 version の把握

```sql
CREATE TABLE deployment_revisions (
  id TEXT PRIMARY KEY,
  cluster_id TEXT NOT NULL,
  function_id TEXT NOT NULL,
  namespace TEXT NOT NULL,
  deployment_name TEXT NOT NULL,
  version TEXT NOT NULL,
  artifact_ref TEXT,
  desired_hash TEXT NOT NULL,
  status TEXT NOT NULL,
  activated_at INTEGER,
  deactivated_at INTEGER,
  created_at INTEGER NOT NULL
);

CREATE INDEX idx_deployment_revisions_active
  ON deployment_revisions(cluster_id, function_id, activated_at DESC);
```

### 9. `route_cache`

cluster で有効な route のローカルキャッシュ。

用途:
- Gateway / runtime 側へ配布する route 情報の保持
- inventory と desired の突き合わせ

```sql
CREATE TABLE route_cache (
  route_id TEXT PRIMARY KEY,
  cluster_id TEXT NOT NULL,
  function_id TEXT NOT NULL,
  host TEXT NOT NULL,
  path TEXT NOT NULL,
  methods_json TEXT NOT NULL,
  priority INTEGER NOT NULL DEFAULT 0,
  route_hash TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  updated_at INTEGER NOT NULL
);

CREATE INDEX idx_route_cache_lookup
  ON route_cache(cluster_id, host, path, priority DESC);
```

### 10. `retry_queue`

一時失敗の再試行制御。

用途:
- apply 失敗
- ack 送信失敗
- inventory 送信失敗

```sql
CREATE TABLE retry_queue (
  id TEXT PRIMARY KEY,
  cluster_id TEXT NOT NULL,
  job_type TEXT NOT NULL,
  related_id TEXT,
  payload_json TEXT NOT NULL,
  attempt_count INTEGER NOT NULL DEFAULT 0,
  max_attempts INTEGER NOT NULL DEFAULT 10,
  next_run_at INTEGER NOT NULL,
  status TEXT NOT NULL,
  last_error TEXT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX idx_retry_queue_sched
  ON retry_queue(status, next_run_at);
```

`job_type`:
- `apply_action`
- `send_ack`
- `send_inventory`
- `send_heartbeat`

### 11. `outbox_events`

Control Plane に送るべきイベントの outbox。

用途:
- HTTP 送信前後で state を壊さない
- 再送可能にする

```sql
CREATE TABLE outbox_events (
  id TEXT PRIMARY KEY,
  cluster_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  aggregate_id TEXT,
  payload_json TEXT NOT NULL,
  status TEXT NOT NULL,
  published_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX idx_outbox_events_status
  ON outbox_events(status, created_at);
```

`event_type`:
- `heartbeat_reported`
- `inventory_reported`
- `sync_ack_reported`
- `deployment_status_changed`

### 12. `local_audit_logs`

agent ローカルの短期監査ログ。

用途:
- cluster 内で何を apply したかの追跡
- Control Plane 未送信時の調査

```sql
CREATE TABLE local_audit_logs (
  id TEXT PRIMARY KEY,
  cluster_id TEXT NOT NULL,
  actor_type TEXT NOT NULL,
  actor_id TEXT,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT,
  status TEXT NOT NULL,
  details_json TEXT,
  created_at INTEGER NOT NULL
);

CREATE INDEX idx_local_audit_logs_time
  ON local_audit_logs(cluster_id, created_at DESC);
```

これは長期保存しない。Control Plane 側監査ログが正本。

## 主要リレーション

```text
sync_plans 1 --- n sync_actions
desired_workloads 1 --- n deployment_revisions
cluster_metadata 1 --- n inventory_snapshots
cluster_metadata 1 --- n retry_queue
```

## 状態遷移

### plan 適用

1. `sync_plans` に `fetched`
2. `sync_actions` を展開して保存
3. 実行開始時に `running`
4. 成功時に `succeeded`
5. 失敗 action は `retry_queue` に投入
6. すべて完了後に `sync_plans.status = succeeded`

### inventory 更新

1. k8s から収集
2. `observed_workloads` を upsert
3. `inventory_snapshots` に summary を追加
4. `outbox_events` へ report を積む

## 保持期間

- `agent_state`: 永続
- `cluster_metadata`: 永続
- `desired_workloads`: 現行 + 直前 generation
- `observed_workloads`: 現行のみ
- `inventory_snapshots`: 7 日または 1,000 件
- `sync_plans`: 30 日
- `sync_actions`: 30 日
- `deployment_revisions`: 直近 20 revision
- `retry_queue`: 完了後 7 日で削除
- `outbox_events`: 送信済みは 7 日で削除
- `local_audit_logs`: 30 日

## インデックス方針

重視するクエリ:
- 現在 generation の desired state 取得
- resource ごとの observed state lookup
- retry 対象のスキャン
- latest sync plan の取得

そのため、以下を優先する。

- `(cluster_id, generation)`
- `(cluster_id, namespace, workload_type, resource_name)`
- `(status, next_run_at)`
- `(cluster_id, created_at DESC)`

## 競合と整合性

- Cluster DB は source of truth ではない
- Control Plane と不一致があれば Control Plane を優先する
- 同一 generation の再取得は idempotent に処理する
- apply 済み判定は `sync_id` と `generation` の両方で行う

## マルチテナントとの関係

Cluster DB には tenant の正本を持たないが、最低限以下は保持してよい。

- `tenant_id`
- `project_id`

用途:
- ローカル route / workload の境界判定
- ログ出力タグ

ただし認可判定の正本は Control Plane 側に置く。

## 実装メモ

Rust で Cluster Agent を作る前提なら、以下の方針がよい。

- DB access: `rusqlite` か `sqlx + sqlite`
- migration 管理: `refinery` または plain SQL
- JSON payload: `TEXT` に JSON 保存
- timestamp: Unix epoch seconds

## MVP の最小テーブル

最初から全部は要らない。MVP では以下で十分。

1. `agent_state`
2. `desired_workloads`
3. `observed_workloads`
4. `sync_plans`
5. `sync_actions`
6. `retry_queue`
7. `route_cache`

## 将来拡張

- PostgreSQL への切替
- 複数 agent での leader election
- local secret envelope encryption
- pod/event レベルの詳細 inventory
- long-term local analytics

## 判断基準

この DB 設計の原則は単純で、Cluster 側には `実行に必要な状態` だけを置く。

Control Plane に置くべきものまで Cluster 側へ持ち込むと、整合性と運用コストが急に悪化する。
