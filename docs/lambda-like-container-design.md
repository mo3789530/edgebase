# Lambda風コンテナ実行基盤設計

## 目的

EdgeBase上で、AWS Lambdaに近い操作感を持つコンテナベースのFunction実行基盤を提供する。

この設計で目指すもの:

- Functionを登録できる
- Functionをコンテナとして配布できる
- HTTPでFunctionを起動できる
- k3s上でFunctionを安全に実行できる
- 実行ログ、メトリクス、実行履歴を追跡できる
- revision管理、traffic split、scale-to-zeroを実現できる

## 方針変更

当初は `Deployment + Service + 独自Gateway` を中心にした自前FaaSを想定していたが、Lambda-likeな実行感に近づけるには、Knative ServingをExecution Planeに採用する方が適切である。

この文書では、自前Gateway中心の構成ではなく、Knative Serviceを中心にした構成へ切り替える。

## 位置づけ

既存構成との関係:

- `controle-plane`: Function定義、revision、target、route、監査
- `cluster-agent`: Control Planeからsync planをpullし、k3sへ反映
- k3s: 実行基盤
- Knative Serving: Lambda-likeなコンテナ実行基盤

本設計では、WASM中心の軽量実行ではなく、コンテナ実行型のFaaSを対象とする。

## スコープ

### 対象

- コンテナイメージとしてFunctionを登録
- Knative ServiceとしてFunctionを配備
- HTTP起動
- revision管理
- traffic split
- scale-to-zero
- 実行履歴の保存

### 対象外

- 完全なAWS Lambda互換
- 汎用CI/CD
- 複雑なマルチテナント課金
- 完全なIAM互換
- GPUジョブや長時間バッチ基盤
- 初期フェーズでのEvent trigger

## 設計方針

### 1. Control PlaneとExecution Planeを分離する

Control Planeは定義とDesired Stateを持ち、実行自体はk3s上のExecution Planeが担当する。

- Control Plane: Function定義、Knative向けDesired State、Route管理、監査
- Execution Plane: Knative Serving上での実行、autoscaling、結果返却

### 2. Execution PlaneはKnative Servingを使う

Lambda-likeの核心である以下を自前実装せず、Knativeへ委譲する。

- revision生成
- traffic split
- request-driven autoscaling
- scale-to-zero / scale-from-zero
- queue-proxy による concurrency 制御

### 3. 配備はCluster Agent pull型に統一する

Control PlaneがKubernetes APIを直接操作するのではなく、Function Controllerが `SyncPlan` を生成し、Cluster Agentがpullして適用する。

ただし apply 対象は `Deployment` や `Service` ではなく、Knativeの `Service` を基本とする。

### 4. 外部入口はKnative Ingressを活用する

MVPでは独自Gatewayを作らず、Knativeのingress layerを利用する。

- Control Planeは論理Routeを保持する
- cluster側ではKnative ingressまたはKourier/Istio経由で受ける
- 必要に応じて外部LBまたはIngress ControllerからKnativeに流す

### 5. Functionは短時間・statelessを前提にする

推奨特性:

- 数秒から数十秒で完了
- stateless
- idempotent
- 外部ストレージ依存を明示

## 全体アーキテクチャ

```text
+-----------------------------+
| EdgeBase Control Plane      |
|  - Function Definitions     |
|  - Function Revisions       |
|  - Deployment Targets       |
|  - Route Definitions        |
|  - Invocation Metadata      |
|  - Function Controller      |
+--------------+--------------+
               |
               | sync plan
               v
+--------------+--------------+
| Cluster Agent / Reconciler  |
|  - Pull sync plan           |
|  - Apply Knative Service    |
|  - Report ack / inventory   |
+--------------+--------------+
               |
               v
+--------------+--------------+
| k3s Cluster                 |
|  - Knative Serving          |
|  - Activator / Autoscaler   |
|  - Queue Proxy              |
|  - Ingress                  |
+--------------+--------------+
               |
               v
+--------------+--------------+
| Function Container          |
|  - User handler             |
|  - Runtime adapter          |
+-----------------------------+
```

## 期待する効果

Knative採用により、以下の自前実装を避けられる。

- 独自Gateway
- 独自scale-to-zero
- 独自request autoscaling
- revision rolloutの制御
- warm/coldの切替ロジック

代わりに、Control Plane側は「Functionをどう見せるか」と「どのclusterへどのrevisionを出すか」に集中する。

## 既存実装との整合

本設計は以下の見直しを前提とする。

- `controle-plane/internal/service/function_controller_service.go`
  - 現在の `Deployment/Service` 向け plan 生成は、Knative Service向けへ変更する
- `cluster-agent/internal/service/apply/k8s_applier.go`
  - `Deployment/Service` apply中心から、Knative Service apply中心へ変更する
- `cluster-agent/internal/service/gateway`
  - 独自Gatewayは主経路から外し、不要なら削除する
- `docs/function-controller-design.md`
  - Sync planのresource typeをKnative前提へ更新する

## ドメインモデル

Function関連の責務は以下のモデルへ分離する。

### 1. FunctionDefinition

論理的なFunction名と公開設定を表す。

保持項目:

- `id`
- `name`
- `description`
- `runtime_kind`
- `default_timeout_seconds`
- `default_memory_mb`
- `default_cpu_millis`
- `created_at`
- `updated_at`

### 2. FunctionRevision

配布可能なイメージ単位の不変レコード。

保持項目:

- `id`
- `function_definition_id`
- `version`
- `image`
- `image_digest`
- `command`
- `args`
- `env`
- `port`
- `healthcheck_path`
- `created_at`

### 3. FunctionDeploymentTarget

どのclusterへ、どのrevisionを、どういうスケーリング条件で出すかを表す desired state。

保持項目:

- `id`
- `function_definition_id`
- `cluster_id`
- `namespace`
- `desired_revision_id`
- `rollout_strategy`
- `traffic_percent`
- `min_scale`
- `max_scale`
- `container_concurrency`
- `enabled`
- `status`
- `last_applied_revision_id`
- `updated_at`

### 4. Route

外部入口からFunctionDefinitionへのマッピング。

保持項目:

- `id`
- `host`
- `path`
- `methods`
- `function_definition_id`
- `cluster_selector`
- `timeout_ms`
- `retry_policy`
- `enabled`

### 5. Invocation / InvocationAttempt

実行履歴は親子2段で保持する。

`Invocation`:

- `id`
- `route_id`
- `function_definition_id`
- `trigger_type`
- `request_id`
- `started_at`
- `completed_at`
- `final_status`
- `client_status_code`

`InvocationAttempt`:

- `id`
- `invocation_id`
- `cluster_id`
- `knative_service`
- `knative_revision`
- `pod_name`
- `attempt_no`
- `started_at`
- `completed_at`
- `duration_ms`
- `status`
- `status_code`
- `error_type`
- `error_message`

## 主要コンポーネント

## 1. Function Registry

FunctionDefinition と FunctionRevision を管理する。

責務:

- 論理Functionの作成
- 新revisionの登録
- revision immutabilityの維持
- image digest固定

## 2. Deployment Manager

FunctionDeploymentTarget を管理する。

責務:

- clusterごとの配備対象管理
- desired revision切替
- scale設定管理
- rollout方針管理

## 3. Function Controller

Control Plane上のdesired stateをcluster-agent向け `SyncPlan` へ変換する。

責務:

- target解決
- inventoryとの差分判定
- Knative Service manifest生成
- rollout状態更新
- drift判定

Function ControllerはKubernetes APIを直接操作しない。

## 4. Knative Serving

実際のFunction実行を担当する。

責務:

- container起動
- revision管理
- traffic split
- autoscaling
- scale-to-zero

## 実行モデル

## HTTP Trigger

### フロー

1. Clientがcluster入口へリクエスト
2. 外部LBまたはDNSが対象clusterのKnative ingressへ到達させる
3. IngressがHostベースでKnative Serviceへ転送
4. Knativeがrevisionへルーティング
5. Function containerが処理
6. 応答を返却
7. Invocation / InvocationAttempt を記録

### 特徴

- cold startを許容しつつ scale-to-zero が可能
- revision単位の切替ができる
- 自前Gatewayを薄くできる

## Function ControllerとSyncPlan

Cluster Agentに渡すplanは、Knative Service適用を前提にする。

### Action Type

- `APPLY_KSERVICE`
- `DELETE_KSERVICE`

### APPLY_KSERVICE payload例

```json
{
  "apiVersion": "serving.knative.dev/v1",
  "kind": "Service",
  "metadata": {
    "name": "telemetry-normalizer",
    "namespace": "edge-functions",
    "labels": {
      "edgebase.io/function-name": "telemetry-normalizer",
      "edgebase.io/function-version": "v2",
      "edgebase.io/managed-by": "function-controller"
    }
  },
  "spec": {
    "template": {
      "metadata": {
        "annotations": {
          "autoscaling.knative.dev/min-scale": "0",
          "autoscaling.knative.dev/max-scale": "20"
        }
      },
      "spec": {
        "containerConcurrency": 10,
        "timeoutSeconds": 3,
        "containers": [
          {
            "image": "registry.local/telemetry-normalizer@sha256:abcd",
            "ports": [
              {
                "containerPort": 8080
              }
            ],
            "env": [
              {
                "name": "MODE",
                "value": "prod"
              }
            ]
          }
        ]
      }
    },
    "traffic": [
      {
        "latestRevision": true,
        "percent": 100
      }
    ]
  }
}
```

## Kubernetes / Knativeリソース設計

Function実体はKnativeの `Service` を基本とする。

必要に応じて以下を伴う。

- `Service.serving.knative.dev`
- `Configuration`
- `Revision`
- `Route`

ただし後ろ3つは通常Knativeが内部的に管理するため、Control Planeの直接責務は `KService` の desired state に集中させる。

## Functionコンテナ契約

Functionコンテナは最低限以下を満たす。

- 指定ポートでHTTP待受
- `POST /invoke` を実装
- `GET /health` を実装
- request envelope を受けられる
- response envelope を返せる

KnativeはHTTPコンテナを前提とするため、container contractはHTTPベースとする。

## API設計

## FunctionDefinition作成

`POST /functions`

```json
{
  "name": "telemetry-normalizer",
  "runtime_kind": "container",
  "default_timeout_seconds": 3,
  "default_memory_mb": 128,
  "default_cpu_millis": 250
}
```

## FunctionRevision登録

`POST /functions/:id/revisions`

```json
{
  "version": "v1",
  "image": "registry.local/telemetry-normalizer:v1",
  "image_digest": "sha256:abcd",
  "port": 8080,
  "healthcheck_path": "/health"
}
```

## Deployment Target作成

`POST /functions/:id/deployments`

```json
{
  "cluster_ids": ["uuid-1"],
  "namespace": "edge-functions",
  "desired_revision_id": "uuid-revision-v1",
  "min_scale": 0,
  "max_scale": 20,
  "container_concurrency": 10,
  "rollout_strategy": "direct"
}
```

## Route作成

`POST /routes`

```json
{
  "host": "api.edgebase.local",
  "path": "/normalize",
  "methods": ["POST"],
  "function_definition_id": "uuid",
  "timeout_ms": 3000
}
```

## Route と Knative ingress の結合方針

### 基本方針

- RouteはControl Plane上の論理定義として保持する
- 実際の外部公開はcluster側のKnative ingressが担う
- Cluster AgentはRouteごとの独自proxyを持たず、KService配備とinventory報告に責務を絞る

### Route の責務

- 公開host/path/methodの定義
- FunctionDefinitionへの対応付け
- timeoutやretryなどの論理ポリシー
- どのclusterへ公開するかの制約

### Knative ingress の責務

- 外部HTTPの受信
- HostベースのKService選択
- revisionへの転送
- scale-from-zeroの起動トリガ

### MVP の結合ルール

MVPでは Knative 単体で無理なく運用できる形に制限する。

- `host` は必須とする
- 1つの公開routeは1つのFunctionDefinitionにのみ対応する
- 1つの `host` は1つの KService に対応させる
- `path` は Function container にそのまま渡す補助情報として扱う
- 同一 `host` 上で複数Functionへ path 振り分けする構成はMVP対象外とする
- 外部DNSまたはLBは対象clusterのKnative ingressへ名前解決させる

### cluster 選択の扱い

MVPでは、外部公開routeは単一clusterへ束縛する。

- `cluster_selector` が1 clusterに解決できることを前提にする
- active-active の multi-cluster 同一host公開は非MVPとする
- failover は運用または将来のglobal LBで補う

### path の扱い

Knative ingress は host ベース公開と相性がよく、共有 host 上の path fan-out は別レイヤのL7ルータが必要になる。

そのためMVPでは:

- `path` は route の論理識別と監査に使う
- Function container は受信 path を自分で解釈してよい
- `host + path` の組み合わせで別Functionへ振り分けるのは将来拡張に回す

### 将来拡張

以下が必要になった時点で、Knative ingress の前段に専用のL7ルーティング層を追加する。

- 同一host配下でのpathベース振り分け
- route単位の認証
- WAF, rate limit, tenant別ポリシー
- multi-cluster weighted routing

候補:

- Kubernetes Gateway API `HTTPRoute`
- Ingress NGINX
- Envoy Gateway
- EdgeBase独自Gateway

## スケーリング方針

### MVP

- Knative autoscalingを使う
- `minScale=0` を許可する
- `maxScale` と `containerConcurrency` をFunction単位で制御する

### 将来拡張

- cluster別default autoscaling profile
- cold start軽減のための pre-warm
- request class別の scaling policy

## セキュリティ

- 外部公開はKnative ingressに限定する
- imageはdigest固定を基本とする
- registry認証情報はSecretで管理する
- 実行権限は最小権限のServiceAccountにする
- `securityContext` を明示する
- `runAsNonRoot` を有効化する
- 不要なLinux capabilityをdropする
- tenant境界が必要な場合はnamespaceまたはclusterで分離する

## ロギングと監視

取得対象:

- invocation count
- error rate
- p50/p95 latency
- cold start率
- scale up / scale down回数
- revision別成功率

ログ方針:

- `invocation_id` を全ログに含める
- `cluster_id`, `function_name`, `knative_service`, `knative_revision`, `pod_name` を含める
- ingressログとrevisionログの相関を取れるようにする

## エラーハンドリング

分類:

- route resolution error
- knative service unavailable
- cold start timeout
- function timeout
- function error
- cluster unreachable

応答方針:

- 4xx: 入力不正
- 5xx: 実行基盤またはFunction内部エラー
- timeoutは `cold start timeout` と `function timeout` を区別する

## 実装フェーズ

### Phase 1: MVP

- FunctionDefinition / FunctionRevision モデル追加
- Deployment Target モデル追加
- Knative前提のFunction Controller実装
- Cluster AgentからKnative Service反映
- HTTP triggerの疎通
- Invocation記録

### Phase 2

- traffic split
- canary配備
- revision rollback
- autoscaling tuning

### Phase 3

- Event trigger
- queue連携
- retry / DLQ
- マルチcluster routing最適化

## MVPの制約

- HTTP triggerのみ
- Event triggerは後回し
- Knative Serving導入が前提
- Invocation保存は最低限から開始

## 既存リポジトリへの反映方針

### `controle-plane`

追加または更新候補:

- `internal/model/function_definition.go`
- `internal/model/function_revision.go`
- `internal/model/function_deployment_target.go`
- `internal/model/route_definition.go`
- `internal/model/invocation.go`
- `internal/model/invocation_attempt.go`
- `internal/service/function_controller_service.go`
- `internal/service/route_service.go`

### `cluster-agent`

追加または更新候補:

- Knative Service applyロジック
- Knative inventory収集
- route同期はKnative ingress前提へ簡素化

### 廃止または縮小候補

- 独自Gateway
- Deployment/Service を直接applyするFaaS専用ロジック

## タスク

### Task 1

- Knative前提で設計文書を更新

### Task 2

- `Function Controller` 設計を Knative Service 中心に更新

### Task 3

- Cluster Agentへ Knative Service apply を追加

### Task 4

- Route と Knative ingress の結合方針を確定

### Task 5

- Invocation / InvocationAttempt 保存

### Task 6

- sample function containerでKnative上の疎通確認

実装メモ:

- sample container: `functions/sample-container`
- Knative manifest: `functions/sample-container/knative-service.yaml`

## 結論

Lambda風のシステムをEdgeBaseで実現するなら、Execution PlaneはKnative Servingを使う方が現実的である。

特に重要なのは以下の点である。

- Control PlaneとExecution Planeを分離する
- 配備はCluster Agent pull型に統一する
- 実行基盤はKnative Serviceを中心にする
- revision、traffic split、scale-to-zeroはKnativeに委譲する
- Control PlaneはFunction定義、route、監査に集中する
