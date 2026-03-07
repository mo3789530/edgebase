# Function Controller設計

## 目的

Function Controllerは、Control Planeに保存されたFunction定義とDeployment Targetをもとに、各k3sクラスタへ適用可能な配備計画を生成し、その反映状態を追跡するサービスである。

本資料では以下を定義する。

- Function Controllerの責務
- 内部コンポーネント構成
- 入出力データ
- 配備計画の生成ルール
- reconcileの状態遷移
- 失敗時の扱い
- MVP実装範囲

## 背景

Lambda風のコンテナ実行基盤では、Function登録だけでは不十分である。

以下の橋渡しが必要になる。

- Control Plane上の定義
- Cluster Agentが理解できるsync plan
- k3s上に実際に存在するDeployment/Service

この変換責務を担うのがFunction Controllerである。

## 位置づけ

```text
Control Plane API
  |
  | Function / Route / Target / Cluster
  v
Function Controller
  |
  | Sync Plan
  v
Cluster Agent
  |
  | Apply
  v
k3s Cluster
```

Function ControllerはKubernetes APIを直接操作しない。

初期フェーズでは、Cluster Agentに渡す `desired state plan` の生成と、ACKを受けた結果反映に責務を限定する。

## 役割

- Function配備計画の生成
- clusterごとの差分判定
- rollout状態管理
- apply結果の反映
- drift検知のための基準状態提供

## 非役割

- Gatewayのリクエスト転送
- Function実行そのもの
- Build処理
- Docker image作成
- Kubernetes APIへの直接apply

## 設計原則

### 1. Desired State中心

Controllerは「今あるもの」ではなく「あるべきもの」を基準にする。

### 2. Cluster Agent pull型

Controllerはpushしない。

- agentが `GET /clusters/:id/sync` でplanを取りにくる
- 反映結果は `POST /clusters/:id/sync/ack` で返る

### 3. 冪等なplan生成

同じ入力からは同じplanが生成されるべきである。

### 4. Deployment planと実行結果を分離

planは「指示」、ackは「結果」である。混ぜない。

## Controllerの責務分解

Function Controllerは内部的に以下へ分割して考える。

### 1. Target Resolver

責務:

- clusterごとの配備対象Functionを解決する
- enabled/disabled状態を解釈する
- desired versionを確定する

### 2. Inventory Reader

責務:

- cluster inventoryから現在状態を取得する
- Deployment, Service, Podの概要を正規化する
- Controller内部の比較モデルに変換する

### 3. Plan Builder

責務:

- desired stateとcurrent stateの差分を判定する
- create/update/delete/restart planを生成する

### 4. Rollout Tracker

責務:

- plan発行後の状態を保持する
- ackで進行状態を更新する
- 成功/失敗/要再試行を判定する

### 5. Status Aggregator

責務:

- Function単位、Cluster単位の配備状態を集約する
- API参照用の状態を返せるようにする

## 入力データ

Controllerが利用する主な入力は以下。

### Function

- `id`
- `name`
- `version`
- `image`
- `command`
- `args`
- `env`
- `timeout_seconds`
- `memory_mb`
- `cpu_millis`
- `max_concurrency`

### FunctionDeploymentTarget

- `id`
- `function_id`
- `cluster_id`
- `namespace`
- `desired_version`
- `replicas`
- `rollout_strategy`
- `enabled`

### Cluster

- `id`
- `status`
- `region`
- `environment`

### Cluster Inventory

- `cluster_id`
- `deployments`
- `services`
- `pods`
- `observed_at`

## 出力データ

Controllerの主な出力は `SyncPlan` である。

### SyncPlan

- `sync_id`
- `cluster_id`
- `generated_at`
- `generation`
- `actions`

### SyncAction

- `type`
- `resource_type`
- `resource_name`
- `namespace`
- `desired_spec`
- `reason`
- `order`

### Action Type

- `APPLY_DEPLOYMENT`
- `APPLY_SERVICE`
- `DELETE_DEPLOYMENT`
- `DELETE_SERVICE`
- `RESTART_DEPLOYMENT`
- `NOOP`

## 内部データモデル

実装では以下の内部DTOを持つと整理しやすい。

### DesiredFunctionInstance

clusterに対して最終的に存在してほしいFunction実体。

項目:

- `function_id`
- `function_name`
- `version`
- `cluster_id`
- `namespace`
- `image`
- `replicas`
- `env`
- `resources`
- `service_port`

### ObservedFunctionInstance

inventoryから得られた現状態。

項目:

- `cluster_id`
- `namespace`
- `deployment_name`
- `image`
- `available_replicas`
- `ready`
- `service_exists`
- `observed_generation`

### DeploymentDiff

差分判定結果。

項目:

- `action_required`
- `deployment_changed`
- `service_changed`
- `reason`

## 命名ルール

Controllerが生成するk8sリソース名は一貫させる。

推奨:

- Deployment名: `fn-{function_name}-{version}`
- Service名: `fn-{function_name}`
- Label:
  - `edgebase.io/function-name`
  - `edgebase.io/function-version`
  - `edgebase.io/managed-by=function-controller`

versionをService名に含めないのは、ルーティング先を安定させるためである。

## Plan生成アルゴリズム

### 手順

1. clusterに紐づく有効な `FunctionDeploymentTarget` を取得
2. targetからdesired function instancesを構築
3. inventoryからobserved function instancesを構築
4. desiredとobservedを `function_name + namespace` で対応付け
5. 差分を生成
6. action順序を決定
7. `SyncPlan` を返す

### 生成ルール

#### 新規作成

条件:

- desiredは存在する
- observedが存在しない

生成するaction:

- `APPLY_DEPLOYMENT`
- `APPLY_SERVICE`

#### 更新

条件:

- desiredとobservedが存在する
- image, env, resources, replicas のいずれかが異なる

生成するaction:

- `APPLY_DEPLOYMENT`
- 必要なら `APPLY_SERVICE`

#### 削除

条件:

- observedは存在する
- desiredが存在しない

生成するaction:

- `DELETE_SERVICE`
- `DELETE_DEPLOYMENT`

#### 再起動

条件:

- specは同じ
- pod不健康が続いている

生成するaction:

- `RESTART_DEPLOYMENT`

## action順序

基本順序は以下。

1. `APPLY_DEPLOYMENT`
2. `APPLY_SERVICE`
3. `RESTART_DEPLOYMENT`
4. `DELETE_SERVICE`
5. `DELETE_DEPLOYMENT`

削除は最後に回す。

理由:

- 先に削除すると通信断が発生しやすい
- rollout中のService欠落を避ける

## Sync Plan例

```json
{
  "sync_id": "uuid",
  "cluster_id": "cluster-1",
  "generated_at": "2026-03-06T12:00:00Z",
  "generation": 12,
  "actions": [
    {
      "type": "APPLY_DEPLOYMENT",
      "resource_type": "Deployment",
      "resource_name": "fn-telemetry-normalizer-v2",
      "namespace": "edge-functions",
      "desired_spec": {
        "image": "registry.local/telemetry-normalizer:v2",
        "replicas": 1,
        "env": {
          "LOG_LEVEL": "info"
        }
      },
      "reason": "version changed from v1 to v2",
      "order": 1
    },
    {
      "type": "APPLY_SERVICE",
      "resource_type": "Service",
      "resource_name": "fn-telemetry-normalizer",
      "namespace": "edge-functions",
      "desired_spec": {
        "selector": {
          "edgebase.io/function-name": "telemetry-normalizer",
          "edgebase.io/function-version": "v2"
        }
      },
      "reason": "service selector must point to v2",
      "order": 2
    }
  ]
}
```

## ACK処理

Cluster Agentはapply後にACKを返す。

### ACK例

```json
{
  "sync_id": "uuid",
  "success": true,
  "results": [
    {
      "resource_type": "Deployment",
      "resource_name": "fn-telemetry-normalizer-v2",
      "status": "applied"
    },
    {
      "resource_type": "Service",
      "resource_name": "fn-telemetry-normalizer",
      "status": "applied"
    }
  ]
}
```

### ACK処理でやること

1. `ClusterSyncRecord` を作成または更新
2. actionごとの結果を保存
3. targetの状態を更新
4. clusterの最終同期時刻を更新
5. 必要なら retry候補を作る

## 状態遷移

Function Deployment Targetごとに状態を持たせる。

### 状態一覧

- `pending`
- `planning`
- `applying`
- `ready`
- `degraded`
- `failed`
- `disabled`

### 遷移

```text
pending -> planning -> applying -> ready
                           |
                           v
                        degraded
                           |
                           v
                         failed
```

### 遷移ルール

- target作成時は `pending`
- planが発行されたら `planning`
- agentがplan取得して処理開始したら `applying`
- ack成功で `ready`
- 部分失敗なら `degraded`
- 継続失敗またはtimeout超過で `failed`
- 明示停止で `disabled`

## 失敗時の扱い

### 分類

- plan generation failure
- cluster unreachable
- apply failure
- drift detected
- stale inventory

### 対応

#### plan generation failure

- sync planを返さない
- エラーをログに出す
- targetを `failed` にしない

#### cluster unreachable

- cluster statusを `offline` 寄りに更新
- 最後に成功したplanを保持

#### apply failure

- `ClusterSyncRecord` に保存
- targetを `degraded` または `failed` にする

#### stale inventory

- 一定時間以上古いinventoryを比較に使わない
- 必要なら `NOOP` ではなく `inventory stale` エラーを返す

## 再試行方針

MVPでは単純化する。

- 自動retryはしない
- 次回agent polling時に再度planを返す
- 同一内容なら同一generationのplanを返してよい

将来的には以下を追加できる。

- backoff
- max retry count
- 部分成功時の再試行

## 一貫性

Controllerは厳密な分散トランザクションを目指さない。

採用する一貫性モデル:

- Control Planeは最終的整合性
- Cluster Agentのpollingで収束
- inventoryとackを組み合わせて現状態を把握

## 必要なRepository

想定追加:

- `function_deployment_target_repository.go`
- `cluster_inventory_repository.go`
- `cluster_sync_repository.go`
- `function_runtime_repository.go`

## 想定Serviceインターフェース

```go
type FunctionControllerService interface {
    BuildDeploymentPlan(ctx context.Context, clusterID uuid.UUID) (*SyncPlan, error)
    AcknowledgePlan(ctx context.Context, clusterID uuid.UUID, syncID uuid.UUID, ack SyncAck) error
    GetDeploymentStatus(ctx context.Context, functionID uuid.UUID) ([]DeploymentStatus, error)
}
```

## Control Plane APIとの接点

Controllerは独立プロセスでなく、初期は `controle-plane/internal/service` の1サービスとして実装してよい。

理由:

- 初期構成が単純
- 既存のsync serviceと近い責務
- DBアクセスを共有しやすい

将来的には独立サービス化も可能。

## MVP実装範囲

- FunctionとTargetからdesired instanceを生成
- inventoryとの差分判定
- Deployment/Serviceのapply/delete planを返す
- ACKを保存して状態更新
- 単純なrolling updateを前提とする

## 非MVP

- canary rollout
- weighted traffic split
- multi-cluster failover自動切替
- pod単位の細かい自己修復
- progressive delivery

## 実装タスク

### Task 1

- `SyncPlan`, `SyncAction`, `SyncAck` DTOを定義する

### Task 2

- desired state構築処理を実装する

### Task 3

- inventory正規化処理を実装する

### Task 4

- diff判定ロジックを実装する

### Task 5

- action ordering を実装する

### Task 6

- ack反映と状態遷移を実装する

### Task 7

- deployment status参照API用の集約処理を実装する

## 結論

Function Controllerは、Function基盤の中で最も重要な「定義から配備への変換点」である。

ここで重要なのは以下の3点である。

- desired state中心であること
- Cluster Agentへ渡すplanを冪等に生成すること
- apply結果を状態遷移として正しく反映すること

MVPでは、Deployment/Serviceの差分同期に絞って設計するのが妥当である。
