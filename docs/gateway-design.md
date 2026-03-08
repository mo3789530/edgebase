# Gateway設計

## ステータス

この文書は独自Gatewayを前提にした旧案を含む。

Lambda-like container 基盤のMVPでは、外部公開はKnative ingressを使うため、独自Gatewayは必須ではない。
現時点では以下の位置づけとする。

- MVP: 採用しない
- 将来: path-based routing、認証、WAF、rate limit、多cluster振り分けが必要になった時点で再導入を検討する

## 目的

Gatewayは、将来的にKnative ingressの前段へ置く拡張用のL7ルーティング層であり、認証、route解決、cluster選択、ポリシー適用を担う候補である。

本資料では以下を定義する。

- Gatewayの責務
- リクエスト処理フロー
- route解決
- cluster選択
- Function Runtimeへの転送
- エラー処理
- MVP範囲

## 位置づけ

```text
Client
  |
  v
Gateway
  |
  +--> Control Plane API
  |      - Route lookup
  |      - Function metadata
  |      - Cluster health
  |
  +--> Function Runtime on k3s
  |
  +--> Invocation Recorder
  |
  +--> Metrics Collector
```

MVPでは外部公開される唯一の入口としては扱わず、Knative ingress を優先する。

## 役割

- 外部HTTPリクエストの受け口
- 認証と基本ポリシー適用
- route解決
- 実行先clusterの選択
- Function Runtimeへの転送
- invocation id 発行
- 実行結果の返却

## 非役割

- Function定義の管理
- k3sリソース反映
- Build処理
- Kubernetes APIの直接操作

## 設計原則

### 1. MVPでは外部公開をKnative ingressへ集約する

Function PodやServiceを直接外部公開せず、Gatewayが必要になるまではKnative ingressを使う。

### 2. Gatewayはできるだけstatelessにする

状態はControl PlaneやRecorderに持たせ、Gatewayは水平分散可能にする。

### 3. route解決と転送を分離する

内部的に次の段階に分ける。

- 認証
- route解決
- target解決
- invoke

### 4. request単位の追跡IDを必ず付与する

`request_id` と `invocation_id` を持たせ、ログと履歴を相関できるようにする。

## MVPとの関係

Knative ingress を使うMVPでは、Route と外部公開の結合は次の制約で扱う。

- `host` は必須
- 1つの `host` は1つの KService に対応する
- `path` は Function container に渡す補助情報として扱う
- 同一host配下でのpath分岐は扱わない
- 単一routeの公開先clusterは1つに制限する

この制約を超える要件が出た時点で、Gatewayを再度有効化する。

## 提供API

### 外部向け

#### Routeベース実行

- `GET /r/*`
- `POST /r/*`
- `PUT /r/*`
- `DELETE /r/*`

#### Function名ベース実行

- `POST /invoke/:function_name`

MVPでは `GET` と `POST` を優先する。

### 内部向け

- `GET /health`
- `GET /ready`
- `GET /live`

## リクエスト処理フロー

### 標準フロー

1. requestを受信する
2. `request_id` を採番する
3. 認証を行う
4. routeを解決する
5. Functionとtarget clusterを決定する
6. `invocation_id` を採番する
7. Function Runtimeへ転送する
8. 応答を受ける
9. Invocation Recorderへ記録する
10. Metrics Collectorへメトリクスを送る
11. クライアントへ応答する

## 認証

### MVP

- API key または Bearer token を利用する
- routeごとの公開/非公開設定を持てるようにする

### 将来

- tenant別認証
- RBAC
- signed request

### 認証失敗時

- `401 Unauthorized`
- request body はFunctionへ転送しない

## Route解決

Gatewayは受信したHTTP requestからRouteを解決する。

### Route解決キー

- host
- path
- method

### Routeモデル

- `id`
- `host`
- `path`
- `methods`
- `function_id`
- `cluster_selector`
- `timeout_ms`
- `retry_policy`
- `auth_mode`
- `enabled`

### path一致ルール

MVPでは以下の順序でよい。

1. 完全一致
2. prefix一致

複雑なpath parameterやregexは後回しにする。

## Cluster選択

route解決後、どのclusterのFunction Runtimeへ送るか決定する。

### 入力

- Route
- cluster health
- function deployment status

### MVP選択ルール

1. `cluster_selector` が明示されていればそれを使う
2. 配備済みで `healthy` なclusterを選ぶ
3. 複数ある場合は単純な round-robin

### 将来拡張

- region優先
- latency優先
- tenant affinity
- canary weighted routing

## Function Runtimeへの転送

GatewayはFunction Runtimeの `POST /invoke` へ内部リクエストを送る。

### 転送先解決

- cluster内のService名
- namespace
- port

### 内部リクエスト例

```json
{
  "invocation_id": "uuid",
  "request_id": "uuid",
  "function": {
    "name": "telemetry-normalizer",
    "version": "v1"
  },
  "request": {
    "method": "POST",
    "path": "/normalize",
    "headers": {
      "content-type": "application/json"
    },
    "query": {},
    "body": {
      "device_id": "dev-1",
      "temperature": 31.2
    }
  },
  "context": {
    "cluster_id": "cluster-1",
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

## タイムアウト

### 適用順序

1. routeの `timeout_ms`
2. functionの `timeout_seconds`
3. gateway global default

最も厳しい値を使うと安全である。

### timeout時の応答

- `504 Gateway Timeout`

### timeout時の記録

- invocation status を `timeout`
- duration を保存

## 再試行

### MVP

- retryしない

理由:

- Functionが非冪等な可能性がある
- 最初から複雑な再試行を入れると事故が起きやすい

### 将来

- routeごとにretry policy
- idempotent routeのみ再試行

## エラー処理

### route未発見

- `404 Not Found`

### 認証失敗

- `401 Unauthorized`

### 配備先clusterなし

- `503 Service Unavailable`

### Function Runtime到達不可

- `503 Service Unavailable`

### Function内部エラー

- runtimeの `status_code` を優先
- 不正レスポンスなら `502 Bad Gateway`

## Invocation Recorder連携

Gatewayは少なくとも以下を記録する。

- `invocation_id`
- `request_id`
- `function_id`
- `route_id`
- `cluster_id`
- `status`
- `started_at`
- `completed_at`
- `duration_ms`
- `status_code`
- `error_message`

### 記録タイミング

- request開始時に start record
- 応答返却前に completion record

## Metrics Collector連携

Gatewayが送るべき主要メトリクス:

- `gateway_requests_total`
- `gateway_request_duration_ms`
- `gateway_errors_total`
- `invocation_total`
- `invocation_timeout_total`

ラベル例:

- route
- function
- cluster
- status_code

## キャッシュ

Gatewayはroute解決のための短期キャッシュを持てる。

### キャッシュ対象

- Route
- Function metadata
- cluster health

### 注意

- TTLは短くする
- 更新の伝播遅延を許容する

MVPではメモリキャッシュで十分。

## ログ

すべてのリクエストログに以下を含める。

- `request_id`
- `invocation_id`
- `route_id`
- `function_name`
- `cluster_id`
- `status_code`
- `duration_ms`

## セキュリティ

- 外部からFunction Runtimeへ直接到達させない
- 認証ヘッダを必要に応じてマスクする
- 機密ヘッダはログに出さない
- upstream先は内部サービス名に限定する

## 想定ディレクトリ構成

新規 `gateway/` を作る場合の例。

```text
gateway/
  cmd/gateway/main.go
  internal/config/
  internal/handler/
  internal/service/auth/
  internal/service/route/
  internal/service/selector/
  internal/service/invoke/
  internal/client/controlplane/
  internal/client/runtime/
  internal/model/
```

## 内部コンポーネント

### 1. Auth Middleware

責務:

- API key / token 検証
- 公開routeの判定

### 2. Request Context Builder

責務:

- request_id 発行
- invocation context 構築

### 3. Route Resolver

責務:

- host/path/method から route 解決

### 4. Cluster Selector

責務:

- 利用可能clusterを1つ選ぶ

### 5. Runtime Client

責務:

- Function Runtime へのHTTP転送
- timeout設定
- レスポンス正規化

### 6. Invocation Reporter

責務:

- start / complete イベント保存

## 想定インターフェース

```go
type RouteResolver interface {
    Resolve(ctx context.Context, host, method, path string) (*Route, error)
}

type ClusterSelector interface {
    Select(ctx context.Context, route *Route, functionID uuid.UUID) (*ClusterTarget, error)
}

type RuntimeInvoker interface {
    Invoke(ctx context.Context, target ClusterTarget, req InvokeRequest) (*InvokeResponse, error)
}

type InvocationReporter interface {
    Start(ctx context.Context, record InvocationStart) error
    Complete(ctx context.Context, record InvocationComplete) error
}
```

## 状態遷移

Gateway自体はstatelessだが、ヘルス状態は持つ。

### 状態

- `starting`
- `healthy`
- `degraded`

### 判定

- Control Plane参照可能
- Runtimeへ転送可能
- Recorder書き込み可能

## MVP範囲

- HTTPのみ
- routeの完全一致とprefix一致
- API keyまたはBearer token認証
- 単純なcluster選択
- synchronous invoke
- invocation記録

## 非MVP

- event trigger
- queue連携
- WebSocket
- cron
- weighted canary routing
- streaming response

## 実装タスク

### Task 1

- `gateway/` の雛形を作る

### Task 2

- auth middleware を実装する

### Task 3

- route resolver を実装する

### Task 4

- cluster selector を実装する

### Task 5

- runtime client を実装する

### Task 6

- invocation reporter を実装する

### Task 7

- timeout とエラー応答を実装する

### Task 8

- metrics と logging を実装する

### Task 9

- health endpoint を実装する

### Task 10

- build/test とサンプルFunctionへの疎通確認を行う

## 結論

Gatewayは、外部からのリクエストを安全かつ一貫した方法でFunction Runtimeへ届けるための中核サービスである。

重要なのは以下の点である。

- 外部公開をGatewayに集約すること
- route解決とcluster選択を明確に分けること
- invocationの追跡情報を確実に残すこと

MVPでは、HTTP同期実行に絞った単純なGatewayとして作るのが妥当である。
