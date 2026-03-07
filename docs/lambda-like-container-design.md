# Lambda風コンテナ実行基盤設計

## 目的

EdgeBase上で、AWS Lambdaに近い操作感を持つコンテナベースのFunction実行基盤を提供する。

この設計で目指すもの:

- Functionを登録できる
- Functionをコンテナとして配布できる
- HTTP/EventでFunctionを起動できる
- k3s上でFunctionを安全に実行できる
- 実行ログ、メトリクス、実行履歴を追跡できる

## 位置づけ

既存構成との関係:

- `controle-plane`: Function管理、配布管理、ルーティング、監視
- `functions/edge-runner`: 将来的な実行ノード側コンポーネントの参考実装
- k3s: 実行基盤

本設計では、WASM中心の軽量実行ではなく、コンテナ実行型のFaaSを対象とする。

## スコープ

### 対象

- コンテナイメージとしてFunctionを登録
- k3sへのFunction配備
- HTTP/Event起動
- 実行制限
- バージョン管理
- 実行履歴の保存

### 対象外

- 完全なAWS Lambda互換
- 汎用CI/CD
- 複雑なマルチテナント課金
- 完全なIAM互換
- GPUジョブや長時間バッチ基盤

## 設計方針

### 1. Control PlaneとExecution Planeを分離する

Control Planeは定義とDesired Stateを持ち、実行自体はk3s上のExecution Planeが担当する。

- Control Plane: 管理、配布、ルーティング、監査
- Execution Plane: 実行、スケーリング、結果返却

### 2. Functionは短時間実行に限定する

Lambda風の使い勝手を維持するため、Functionは短時間・小粒度な処理を前提とする。

推奨特性:

- 数秒から数十秒で完了
- stateless
- idempotent
- 外部ストレージ依存を明示

### 3. k3sは実行基盤として扱う

k3sはFunctionの管理主体ではなく、Container実行基盤として利用する。

### 4. 初期フェーズではコンテナを常駐させる

MVPでは完全なscale-to-zeroを目指さず、少数レプリカ常駐で安定動作を優先する。

## 全体アーキテクチャ

```text
+-----------------------------+
| EdgeBase Control Plane      |
|  - Function Registry        |
|  - Deployment Target Mgmt   |
|  - Route / Trigger Mgmt     |
|  - Invocation Metadata      |
+--------------+--------------+
               |
               | desired state
               v
+--------------+--------------+
| Cluster Agent / Reconciler  |
|  - Pull deployment plans    |
|  - Apply to k3s             |
|  - Report status            |
+--------------+--------------+
               |
               v
+--------------+--------------+
| k3s Cluster                 |
|  - Gateway                  |
|  - Function Runtime Pods    |
|  - Autoscaler               |
|  - Service / Ingress        |
+--------------+--------------+
               |
               v
+--------------+--------------+
| Function Container          |
|  - User handler             |
|  - Runtime adapter          |
|  - Request/response bridge  |
+-----------------------------+
```

## 主要コンポーネント

## 1. Function Registry

Functionのメタデータを管理する。

保持項目:

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
- `runtime_mode`
- `created_at`

`runtime_mode` の例:

- `http`
- `event`

## 2. Deployment Manager

どのFunctionをどのk3sクラスタへ配備するかを管理する。

責務:

- deployment target管理
- rollout方針管理
- version切替
- 段階配備

## 3. Gateway

外部からのHTTP/Eventを受け、対象Functionへルーティングする。

責務:

- 認証
- route解決
- invocation id 発行
- timeout管理
- retry方針の適用

MVPではHTTP triggerのみ対応でよい。

## 4. Runtime Adapter

Functionコンテナ内で共通の実行インターフェースを提供する。

責務:

- HTTP request を handler へ橋渡し
- event payload の標準化
- ヘルスチェック応答
- 実行メタデータの返却

Functionごとに自由なWebアプリを置くより、最低限の実行契約を持たせたほうが運用しやすい。

## 5. Invocation Recorder

各実行のメタデータを保存する。

保持項目:

- `invocation_id`
- `function_id`
- `cluster_id`
- `status`
- `started_at`
- `completed_at`
- `duration_ms`
- `status_code`
- `error_message`
- `request_size`
- `response_size`

## 実行モデル

## HTTP Trigger

### フロー

1. ClientがGatewayへリクエスト
2. GatewayがRouteからFunctionを解決
3. Gatewayが対象ClusterのFunction Serviceへ転送
4. Function Podが処理
5. 応答をGateway経由で返却
6. Invocation metadata を記録

### 特徴

- 同期応答しやすい
- API Gatewayに近い使い方ができる

## Event Trigger

### フロー

1. Event source がControl PlaneまたはBrokerへ送信
2. Event dispatcher が対象Functionを決定
3. QueueまたはHTTP経由でFunctionへ配送
4. 非同期結果を記録

### 初期判断

MVPでは後回しにしてよい。

## Kubernetesリソース設計

各Function versionごとに以下を持つ。

- `Deployment`
- `Service`
- `ConfigMap`
- `Secret` 必要時のみ
- `HorizontalPodAutoscaler` 任意

Function 1つに対して1 Deploymentを基本とする。

### Deployment設計

推奨:

- replica 1 以上
- readiness probe 必須
- liveness probe 必須
- resource requests / limits 設定
- rolling update 利用

### Service設計

- ClusterIP を基本とする
- GatewayからService名解決で呼ぶ

### Ingress設計

- 直接公開はしない
- 外部公開はGateway経由に寄せる

## Functionコンテナ契約

Functionコンテナは最低限以下を満たす。

- 指定ポートでHTTP待受
- `POST /invoke` を実装
- `GET /health` を実装
- JSON入力を受けられる
- JSON出力を返せる

### リクエスト例

```json
{
  "invocation_id": "uuid",
  "function": {
    "name": "telemetry-normalizer",
    "version": "v1"
  },
  "request": {
    "headers": {
      "content-type": "application/json"
    },
    "body": {
      "device_id": "dev-1",
      "temperature": 31.2
    }
  },
  "context": {
    "cluster_id": "uuid",
    "timeout_ms": 3000
  }
}
```

### レスポンス例

```json
{
  "status_code": 200,
  "headers": {
    "content-type": "application/json"
  },
  "body": {
    "normalized": true
  }
}
```

## データモデル

既存 `controle-plane/internal/model` へ追加を想定する。

## FunctionRuntime

- `id`
- `function_id`
- `runtime_type`
- `image`
- `command`
- `args`
- `port`
- `healthcheck_path`

## FunctionDeploymentTarget

- `id`
- `function_id`
- `cluster_id`
- `namespace`
- `desired_version`
- `rollout_strategy`
- `status`

## InvocationRecord

- `id`
- `function_id`
- `cluster_id`
- `route_id`
- `status`
- `request_id`
- `started_at`
- `completed_at`
- `duration_ms`
- `error_message`

## Route

- `id`
- `host`
- `path`
- `methods`
- `function_id`
- `cluster_selector`
- `timeout_ms`
- `retry_policy`

## API設計

## Function登録

`POST /functions`

```json
{
  "name": "telemetry-normalizer",
  "version": "v1",
  "runtime": "container",
  "image": "registry.local/telemetry-normalizer:v1",
  "timeout_seconds": 3,
  "memory_mb": 128,
  "cpu_millis": 250,
  "max_concurrency": 20
}
```

## Function一覧

`GET /functions`

## Function配備

`POST /functions/:id/deploy`

```json
{
  "cluster_ids": ["uuid-1", "uuid-2"],
  "namespace": "edge-functions",
  "rollout_strategy": "rolling"
}
```

## Route作成

`POST /routes`

```json
{
  "host": "api.edgebase.local",
  "path": "/normalize",
  "methods": ["POST"],
  "function_id": "uuid",
  "timeout_ms": 3000
}
```

## Invocation実行

`POST /invoke/:function_name`

## Invocation詳細

`GET /invocations/:id`

## スケーリング方針

### MVP

- min replicas 1
- HPAはCPUまたはRPS近似値で制御
- scale-to-zeroは対象外

### 将来拡張

- queue length ベース
- request latency ベース
- scale-from-zero

## セキュリティ

- 外部公開はGatewayのみに限定する
- Function Podの直接公開を避ける
- registry認証情報はSecretで管理する
- tenant境界が必要な場合はnamespaceまたはclusterで分離する
- 実行権限は最小権限のServiceAccountにする

## ロギングと監視

取得対象:

- invocation count
- error rate
- p50/p95 latency
- pod restart count
- cluster別成功率

ログ方針:

- invocation_id を全ログに含める
- Gateway と Function Pod の相関を取れるようにする

## エラーハンドリング

分類:

- route resolution error
- deployment unavailable
- function timeout
- function error
- cluster unreachable

応答方針:

- 4xx: 入力不正
- 5xx: 実行基盤またはFunction内部エラー
- timeout: 明示的に区別する

## 実装フェーズ

### Phase 1: MVP

- Function登録
- Route作成
- k3sへのDeployment生成
- Gateway経由のHTTP起動
- Invocation記録

### Phase 2

- HPA連携
- version rollout
- canary配備
- Cluster target配布管理

### Phase 3

- Event trigger
- queue連携
- retry / DLQ
- scale-to-zero検討

## MVPの制約

- HTTP triggerのみ
- コンテナ常駐
- 単純なrolling deploy
- 同期レスポンス中心

## 既存リポジトリへの反映方針

### `controle-plane`

追加候補:

- `internal/model/function_runtime.go`
- `internal/model/invocation.go`
- `internal/repository/function_runtime_repository.go`
- `internal/repository/invocation_repository.go`
- `internal/service/invocation_service.go`
- `internal/handler/invocation_handler.go`

### `functions`

追加候補:

- `functions/container-runtime/`
- 共通のruntime adapter
- sample function container

## タスク

### Task 1

- Function runtime用モデル追加
- AutoMigrate組み込み

### Task 2

- Function登録APIへcontainer runtime属性を追加

### Task 3

- Deployment target管理を追加

### Task 4

- Gateway設計と最小実装

### Task 5

- Invocation record の保存

### Task 6

- Cluster Agentからk3sへDeployment反映

### Task 7

- サンプルFunctionコンテナ作成

## 結論

Lambda風のシステムをEdgeBaseで実現するなら、Control PlaneでFunction定義と配備を管理し、k3sをExecution Planeとして使う形が最も現実的である。

特に重要なのは以下の点である。

- Control PlaneとExecution Planeを分離する
- Functionは短時間・statelessに寄せる
- 初期はHTTP triggerと常駐Podに限定する
- 完全なLambda互換ではなく、Edge向けFaaSとして段階的に育てる
