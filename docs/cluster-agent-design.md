# Cluster Agent設計

## 目的

Cluster Agentは、各k3sクラスタ内に配置され、Control Planeから取得したdesired stateを実クラスタへ反映し、現在状態をControl Planeへ返すための常駐プロセスである。

本資料では以下を定義する。

- Cluster Agentの責務
- 内部コンポーネント構成
- Control Planeとの通信
- inventory収集方式
- k3s反映方式
- エラー処理
- MVP範囲

## 位置づけ

```text
Control Plane API
  ^
  | heartbeat / inventory / sync / ack
  v
Cluster Agent
  ^
  | Kubernetes API
  v
k3s Cluster
```

Cluster Agentは、Control Planeとk3sの間の翻訳層である。

## 役割

- クラスタの生存状態をControl Planeへ通知する
- クラスタ内リソースのinventoryを送信する
- Control Planeからsync planを取得する
- sync planをk3sへ反映する
- 反映結果をACKとして返す

## 非役割

- Function定義の管理
- Route解決
- 外部クライアントからのinvoke受付
- Build処理
- コンテナイメージ生成

## 設計原則

### 1. Pull型同期

Control Planeからのpushは行わない。

- Agentが定期的にplanを取りに行く
- ネットワーク制約下でも扱いやすい

### 2. 冪等apply

同じplanを複数回処理しても破綻しないことを前提とする。

### 3. inventoryとackを分離する

- inventory: 現状態の報告
- ack: planに対する適用結果

### 4. Kubernetesへの直接アクセスはAgentに閉じる

Control Planeはk3s APIに直接依存しない。

## 配置

### 配置単位

Clusterごとに1つのAgentを基本とする。

### 配置形態

初期は以下のいずれか。

- Deployment 1 replica
- system service

k3s内へ統一的に置くならDeploymentが自然である。

### namespace

推奨:

- `edgebase-system`

## 内部コンポーネント

Agentは以下のモジュールで構成する。

### 1. Heartbeat Loop

責務:

- 定期heartbeat送信
- agent version、cluster version、health summary送信

### 2. Inventory Collector

責務:

- Deployment, Service, Pod, Node 情報を収集
- Controller比較用に正規化

### 3. Plan Fetcher

責務:

- `GET /clusters/:id/sync` の実行
- generation確認
- 重複planの判定

### 4. Manifest Builder

責務:

- SyncActionをk8sリソース操作へ変換
- desired_spec を Kubernetes object に変換

### 5. Resource Applier

責務:

- create/update/delete/restart の実行
- apply結果の収集

### 6. Ack Reporter

責務:

- `POST /clusters/:id/sync/ack` を送信
- actionごとの結果を返す

### 7. Local State Store

責務:

- 最後に処理した generation 記録
- plan重複実行の抑制
- 一時失敗の記録

初期はローカルファイルまたはメモリで十分。

## Control Planeとの通信

### 必要API

- `POST /clusters/:id/heartbeat`
- `POST /clusters/:id/inventory`
- `GET /clusters/:id/sync`
- `POST /clusters/:id/sync/ack`

### 認証

- bearer token またはJWT
- cluster登録時に払い出されたtokenを保持する

### 通信方針

- heartbeat: 高頻度
- inventory: 中頻度
- sync fetch: 中頻度
- ack: apply直後

### 推奨周期

- heartbeat: 10秒から30秒
- inventory: 30秒から120秒
- sync fetch: 5秒から30秒

MVPでは固定周期でよい。

## inventory設計

Agentが送るinventoryは、Controllerが差分判定に使える粒度に限定する。

### ClusterInventory例

```json
{
  "cluster_id": "uuid",
  "observed_at": "2026-03-06T12:00:00Z",
  "kubernetes_version": "v1.31.2+k3s1",
  "nodes": [
    {
      "name": "edge-node-1",
      "role": "worker",
      "internal_ip": "10.0.0.11",
      "status": "Ready"
    }
  ],
  "deployments": [
    {
      "namespace": "edge-functions",
      "name": "fn-telemetry-normalizer-v1",
      "image": "registry.local/telemetry-normalizer:v1",
      "ready_replicas": 1,
      "available_replicas": 1
    }
  ],
  "services": [
    {
      "namespace": "edge-functions",
      "name": "fn-telemetry-normalizer",
      "selector": {
        "edgebase.io/function-name": "telemetry-normalizer",
        "edgebase.io/function-version": "v1"
      }
    }
  ]
}
```

### 収集対象

- Node
- Deployment
- Service
- Pod概要

### 初期では不要

- Event全文
- ReplicaSet詳細
- kube-system配下の汎用リソース

## Sync Plan処理

### フロー

1. Agentが `GET /clusters/:id/sync` を呼ぶ
2. planの `sync_id` と `generation` を確認する
3. 未処理なら action を順に実行する
4. 成功/失敗を収集する
5. ACKを返す

### generation扱い

- 同じgenerationかつ同じsync_idなら再実行抑制可能
- generationが進んでいれば新planとして処理する

## Sync Action対応

### APPLY_DEPLOYMENT

処理:

- Deployment manifestを作成
- create or update を実行

### APPLY_SERVICE

処理:

- Service manifestを作成
- create or update を実行

### DELETE_DEPLOYMENT

処理:

- 対象Deploymentを削除

### DELETE_SERVICE

処理:

- 対象Serviceを削除

### RESTART_DEPLOYMENT

処理:

- pod template annotation更新などでrollout restart相当を実行

## Kubernetes反映方針

### 適用戦略

MVPでは server-side apply か upsert 的な create/update を採用する。

要件:

- 同じspecで再実行しても安全
- 差分更新が可能
- Controller管理ラベルを付与する

### 管理対象

- Deployment
- Service
- ConfigMap
- Secret

初期は Deployment と Service を優先する。

### ラベル方針

- `edgebase.io/managed-by=cluster-agent`
- `edgebase.io/function-name=<name>`
- `edgebase.io/function-version=<version>`

## Local State Store

Agentが最低限保持したい情報:

- last successful sync id
- last seen generation
- last inventory timestamp
- recent apply errors

### 保存先候補

- メモリ
- ローカルファイル

MVPでは再起動時に失われても致命的ではないため、単純な実装でよい。

## ACK設計

### ACK例

```json
{
  "sync_id": "uuid",
  "success": false,
  "results": [
    {
      "resource_type": "Deployment",
      "resource_name": "fn-telemetry-normalizer-v2",
      "status": "failed",
      "error_message": "image pull backoff"
    },
    {
      "resource_type": "Service",
      "resource_name": "fn-telemetry-normalizer",
      "status": "skipped"
    }
  ]
}
```

### result status

- `applied`
- `deleted`
- `skipped`
- `failed`

## エラー処理

### 通信失敗

- Control Planeへの送信失敗時はログ出力
- 次の周期で再送を試みる

### inventory取得失敗

- heartbeatは継続
- inventoryだけ失敗として扱う

### plan apply失敗

- actionごとに失敗を記録
- 可能なら残りaction継続
- 最終的に部分失敗ACKを返す

### Kubernetes API失敗

- 一時エラーと恒久エラーを区別する
- MVPでは単純にfailedで返し、次回pollingで再試行する

## リソース作成ルール

### Namespace

- 対象namespaceが存在しなければ作成するか、事前作成を前提とする

MVPでは事前作成でもよい。

### Image Pull Secret

- namespaceに必要なSecretがある前提
- 後続でSecret Distributionと連携する

### Replicas

- planの `replicas` を尊重する
- 未指定なら1

## Health判定

Agent自身のhealthは以下で判定する。

- Control Plane疎通可否
- Kubernetes API疎通可否
- 直近syncの成否

### health状態

- `healthy`
- `degraded`
- `unreachable`

## 状態遷移

Agent全体の動作状態を持つ。

### 状態

- `starting`
- `healthy`
- `syncing`
- `degraded`
- `unreachable`

### 遷移

```text
starting -> healthy -> syncing -> healthy
                   \-> degraded -> healthy
                   \-> unreachable -> healthy
```

## セキュリティ

- agent tokenは平文でログ出力しない
- TLSでControl Planeと通信する
- k8s権限は最小権限のServiceAccountにする
- 管理対象namespaceのみ権限を絞る

## 想定ディレクトリ構成

新規 `cluster-agent/` を作る場合の例。

```text
cluster-agent/
  cmd/agent/main.go
  internal/config/
  internal/client/controlplane/
  internal/client/k8s/
  internal/service/heartbeat/
  internal/service/inventory/
  internal/service/sync/
  internal/service/apply/
  internal/model/
```

## 想定インターフェース

```go
type PlanFetcher interface {
    Fetch(ctx context.Context, clusterID uuid.UUID) (*SyncPlan, error)
}

type InventoryReporter interface {
    Report(ctx context.Context, clusterID uuid.UUID, inventory ClusterInventory) error
}

type HeartbeatReporter interface {
    Report(ctx context.Context, clusterID uuid.UUID, heartbeat Heartbeat) error
}

type ResourceApplier interface {
    Apply(ctx context.Context, plan *SyncPlan) (*SyncAck, error)
}
```

## MVP範囲

- Deploymentとしてagentを配置
- heartbeat loop
- inventory送信
- sync fetch
- Deployment/Serviceのapply/delete
- ACK送信

## 非MVP

- watchベース同期
- 複雑なrollback
- Secret自動配布
- Pod log収集
- admission hook

## 実装タスク

### Task 1

- `cluster-agent/` の雛形を作る

### Task 2

- Control Plane client を実装する

### Task 3

- heartbeat loop を実装する

### Task 4

- inventory collector を実装する

### Task 5

- sync fetcher を実装する

### Task 6

- Kubernetes resource applier を実装する

### Task 7

- ACK reporter を実装する

### Task 8

- local state store を実装する

### Task 9

- エラー処理と再試行を整える

### Task 10

- build/test とサンプルclusterでの疎通確認を行う

## 結論

Cluster Agentは、Control Planeの意図をk3sの実体へ変換する実行側の中核である。

重要なのは以下の3点である。

- pull型でControl Planeと疎結合にすること
- inventoryとackを分離して扱うこと
- applyを冪等に保つこと

MVPでは、Deployment/Service同期に絞った単純なAgentとして始めるのが妥当である。
