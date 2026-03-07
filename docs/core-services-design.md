# 必要サービス詳細設計

## 目的

Lambda風のFunction実行基盤を成立させるために必要な中核サービスについて、責務、インターフェース、データモデル、MVP範囲を整理する。

本資料は以下の設計を対象とする。

- [lambda-like-container-design.md](/Users/mo/repo/edgebase/docs/lambda-like-container-design.md)
- [service-catalog.md](/Users/mo/repo/edgebase/docs/service-catalog.md)

## 対象サービス

MVPで重要度が高い以下のサービスを対象とする。

- Control Plane API
- Function Controller
- Cluster Agent
- Gateway
- Function Runtime
- Invocation Recorder
- Metrics Collector

## 全体関係

```text
Client
  |
  v
Gateway
  |
  v
Function Runtime on k3s
  |
  +--> Invocation Recorder
  +--> Metrics Collector

Control Plane API
  |
  v
Function Controller
  |
  v
Cluster Agent
  |
  v
k3s API
```

## 1. Control Plane API

### 役割

システム全体の定義情報を管理する中核サービス。

### 責務

- Function定義管理
- Cluster管理
- Deployment Target管理
- Route管理
- Desired Stateの提供
- 実行履歴参照APIの提供

### 入力

- 管理者からのAPI操作
- Cluster Agentからの問い合わせ
- Gatewayからの参照要求

### 出力

- Function metadata
- cluster別のdesired state
- route情報
- invocation参照情報

### API

#### Function管理

- `POST /functions`
- `GET /functions`
- `GET /functions/:id`
- `POST /functions/:id/deploy`

#### Cluster管理

- `POST /clusters`
- `GET /clusters`
- `GET /clusters/:id`
- `POST /clusters/:id/heartbeat`
- `POST /clusters/:id/inventory`

#### Route管理

- `POST /routes`
- `GET /routes`
- `GET /routes/:id`

#### Sync

- `GET /clusters/:id/sync`
- `POST /clusters/:id/sync/ack`

#### Invocation参照

- `GET /invocations`
- `GET /invocations/:id`

### データモデル

- `Function`
- `FunctionRuntime`
- `Cluster`
- `ClusterNode`
- `FunctionDeploymentTarget`
- `Route`
- `InvocationRecord`

### 保存先

- PostgreSQL

### MVP範囲

- Function登録
- Cluster登録
- Route作成
- Sync API
- Invocation参照

### 非MVP

- 複雑なRBAC
- 高度なtenant分離
- canary割合制御

## 2. Function Controller

詳細設計は [function-controller-design.md](/Users/mo/repo/edgebase/docs/function-controller-design.md) を参照。

### 役割

Control Planeの定義をクラスタへ反映可能な配備計画へ変換する。

### 責務

- desired deployment plan生成
- cluster別配備差分判定
- rollout状態管理
- 配備ステータス集約

### 入力

- Function metadata
- Deployment target
- Cluster inventory
- 現在の配備状態

### 出力

- clusterごとの適用プラン
- deployment status

### 内部処理

1. 対象FunctionとTargetを取得
2. clusterごとのdesired versionを解決
3. inventoryからcurrent stateを取得
4. diffを生成
5. Cluster Agent向けsync payloadを生成

### 主要データ

- `FunctionDeploymentTarget`
- `ClusterSyncRecord`
- `ClusterInventorySnapshot`

### 想定メソッド

- `BuildDeploymentPlan(clusterID)`
- `BuildAllPlans()`
- `ReconcileStatus(clusterID, ack)`

### MVP範囲

- rolling deployment plan生成
- 単純な version 差分判定
- 成功/失敗の状態反映

### 非MVP

- blue/green
- canary
- 自動rollback

## 3. Cluster Agent

詳細設計は [cluster-agent-design.md](/Users/mo/repo/edgebase/docs/cluster-agent-design.md) を参照。

### 役割

各k3sクラスタに配置され、Control Planeの計画を受けて実リソースへ反映する。

### 責務

- heartbeat送信
- inventory送信
- desired state取得
- k3sへのmanifest反映
- 同期結果ACK

### 入力

- Control Planeからのsync plan
- k3s cluster state

### 出力

- inventory
- sync ack
- health status

### API利用

- `POST /clusters/:id/heartbeat`
- `POST /clusters/:id/inventory`
- `GET /clusters/:id/sync`
- `POST /clusters/:id/sync/ack`

### Agent内部構成

- `heartbeat loop`
- `inventory collector`
- `plan fetcher`
- `manifest applier`
- `ack reporter`

### Kubernetes操作

- Deploymentの作成更新
- Serviceの作成更新
- ConfigMap/Secretの反映

### MVP範囲

- 定期heartbeat
- inventory収集
- Deployment/Service反映
- ACK送信

### 非MVP

- admission webhook
- 複雑なdrift修復
- nodeレベル最適化

## 4. Gateway

詳細設計は [gateway-design.md](/Users/mo/repo/edgebase/docs/gateway-design.md) を参照。

### 役割

外部リクエストの唯一の入口として、Function実行を仲介する。

### 責務

- 認証
- route解決
- invocation id 発行
- cluster選択
- function呼び出し
- timeout/retry制御

### 入力

- HTTP request

### 出力

- Function response
- Invocation metadata
- Metrics

### 主要フロー

1. request受信
2. auth確認
3. route解決
4. cluster選定
5. target serviceへ転送
6. 応答返却
7. invocationを記録

### API

#### 外部向け

- `POST /invoke/:function_name`
- routeベース実行:
  - `POST /r/*`
  - `GET /r/*`

#### 内部向け

- `GET /health`

### 必要なデータ

- `Route`
- `Function`
- `ClusterHealth`

### MVP範囲

- HTTPのみ
- 単純なroute解決
- retryなしまたは1回まで
- synchronous invoke

### 非MVP

- WebSocket
- cron trigger
- queue trigger

## 5. Function Runtime

### 役割

Functionコンテナの共通実行契約を提供する。

### 責務

- invoke endpoint提供
- health endpoint提供
- request/response標準化
- timeout尊重

### 実行契約

- `POST /invoke`
- `GET /health`

### リクエスト構造

```json
{
  "invocation_id": "uuid",
  "function": {
    "name": "example",
    "version": "v1"
  },
  "request": {
    "headers": {},
    "body": {}
  },
  "context": {
    "timeout_ms": 3000,
    "cluster_id": "uuid"
  }
}
```

### レスポンス構造

```json
{
  "status_code": 200,
  "headers": {},
  "body": {}
}
```

### ランタイム要件

- stateless
- readiness/liveness対応
- JSON I/O
- 小さい起動時間

### MVP範囲

- HTTP invoke
- JSON payload
- 単一ポート

### 非MVP

- streaming response
- async callback
- language-specific SDK群

## 6. Invocation Recorder

### 役割

Function実行履歴の保存と追跡を担う。

### 責務

- invocation開始記録
- invocation完了記録
- status/duration/error保存
- request/responseサイズ保存

### 入力

- Gatewayからの開始/完了通知
- 必要時にRuntimeからの補足情報

### 出力

- 実行履歴参照API向けデータ
- メトリクス集計用データ

### データモデル

- `id`
- `function_id`
- `cluster_id`
- `route_id`
- `status`
- `started_at`
- `completed_at`
- `duration_ms`
- `status_code`
- `error_message`

### API

- `GET /invocations`
- `GET /invocations/:id`

### MVP範囲

- 成功/失敗記録
- duration記録
- 基本検索

### 非MVP

- payload全文保存
- distributed trace完全連携

## 7. Metrics Collector

### 役割

Function基盤全体の可観測性を提供する。

### 責務

- invocation数計測
- success rate計測
- latency計測
- cluster health計測
- pod restart計測

### 入力

- Gateway
- Function Runtime
- Cluster Agent
- Invocation Recorder

### 出力

- ダッシュボード用メトリクス
- alert条件判定用メトリクス

### 主要指標

- `invocation_total`
- `invocation_errors_total`
- `invocation_duration_ms`
- `gateway_requests_total`
- `cluster_sync_failures_total`
- `function_pod_restarts_total`

### MVP範囲

- invocation success/failure
- p50/p95 latency
- cluster health

### 非MVP

- 高度なSLO管理
- コスト配賦

## サービス間シーケンス

### 配備

1. 管理者がControl Plane APIへFunction登録
2. 管理者がDeployment Targetを設定
3. Function Controllerが配備計画を生成
4. Cluster Agentが `GET /clusters/:id/sync` で取得
5. Cluster Agentがk3sへ反映
6. Cluster AgentがACK送信
7. Control Plane APIが状態更新

### 実行

1. ClientがGatewayへHTTPリクエスト
2. Gatewayがrouteを解決
3. GatewayがFunction Runtimeへ転送
4. Runtimeが処理
5. GatewayがInvocation Recorderへ記録
6. Metrics Collectorがメトリクス更新
7. GatewayがClientへ応答

## 実装優先順位

1. Control Plane API
2. Function Controller
3. Cluster Agent
4. Function Runtime
5. Gateway
6. Invocation Recorder
7. Metrics Collector

## 実装タスク

### Task 1: Control Plane APIのFunction拡張

- 既存 `controle-plane` のFunctionモデルにcontainer runtime向け属性を追加する
- `image`, `command`, `args`, `timeout_seconds`, `memory_mb`, `cpu_millis`, `max_concurrency` を扱えるようにする
- Function登録APIを拡張する

完了条件:

- container runtime向けFunctionを登録できる
- 既存WASM向け定義を壊さない

### Task 2: Clusterモデル追加

- `Cluster`, `ClusterNode`, `ClusterSyncRecord` を追加する
- Cluster登録、一覧、詳細、heartbeat APIを実装する
- AutoMigrate対象へ追加する

完了条件:

- k3sクラスタをControl Planeで一意に管理できる
- heartbeatで生存状態を更新できる

### Task 3: Deployment Target管理追加

- `FunctionDeploymentTarget` モデルを追加する
- `POST /functions/:id/deploy` を実装する
- FunctionとClusterの紐付けを永続化する

完了条件:

- Functionごとに配備先clusterを指定できる
- desired versionを保持できる

### Task 4: Route管理追加

- `Route` モデルを追加する
- `POST /routes`, `GET /routes`, `GET /routes/:id` を追加する
- host/path/method でFunctionを引けるようにする

完了条件:

- HTTP routeからFunctionを解決できる
- timeoutなどの基本設定を保持できる

### Task 5: Function Controller実装

- `controle-plane/internal/service` に `FunctionControllerService` を追加する
- cluster単位のdeployment planを生成する
- inventoryとdesired stateの差分を判定する

完了条件:

- clusterごとの適用計画を生成できる
- sync APIからagentへ返せる

### Task 6: Cluster Sync API実装

- `GET /clusters/:id/sync` を実装する
- `POST /clusters/:id/sync/ack` を実装する
- apply結果を `ClusterSyncRecord` に保存する

完了条件:

- agentがdesired stateを取得できる
- apply成否をControl Planeへ返せる

### Task 7: Cluster Inventory API実装

- `POST /clusters/:id/inventory` を実装する
- cluster配下ノード、deployment、podなどの状態を保存する
- inventory snapshotの保持方針を決める

完了条件:

- Control Planeがclusterの現在状態を参照できる
- Function Controllerが差分判定に使える

### Task 8: Cluster Agent雛形作成

- 新規 `cluster-agent/` を追加する
- heartbeat loop を実装する
- inventory送信を実装する
- sync plan取得とACK送信の雛形を作る

完了条件:

- Control Planeとagentが疎通できる
- 定期同期の基本ループが動く

### Task 9: k3s反映機能実装

- Cluster AgentにKubernetes client処理を追加する
- DeploymentとServiceの作成更新を実装する
- ConfigMapやSecretの基本反映を追加する

完了条件:

- planに基づいてk3sへFunctionを配備できる
- 再実行しても破綻しない

### Task 10: Function Runtime契約定義

- FunctionコンテナのHTTP契約を確定する
- `POST /invoke` と `GET /health` を定義する
- request/responseのJSONフォーマットを固定する

完了条件:

- GatewayとFunctionコンテナが共通契約で通信できる
- サンプルFunctionを作れる

### Task 11: サンプルFunctionコンテナ作成

- `functions/` 配下にサンプルのcontainer functionを追加する
- `POST /invoke` を実装する
- Dockerfileを用意する

完了条件:

- k3sへ配備して疎通確認できる
- Gatewayから呼び出せる

### Task 12: Gateway実装

- 新規 `gateway/` サービスを追加する
- route解決、request forwarding、timeout制御を実装する
- invocation_id を採番する

完了条件:

- 外部HTTPリクエストをFunctionへ転送できる
- routeベースのFunction呼び出しが動く

### Task 13: Invocation Recorder実装

- `InvocationRecord` モデルを追加する
- Gatewayから開始/完了イベントを保存する
- `GET /invocations`, `GET /invocations/:id` を実装する

完了条件:

- 実行履歴を参照できる
- 失敗理由とdurationを追える

### Task 14: Metrics Collector実装

- GatewayとControl Planeで基本メトリクスを出す
- invocation success/failure、duration、cluster sync failure を記録する
- 既存metrics機構へ統合する

完了条件:

- 基本的な成功率と遅延を見られる
- cluster異常を検知できる

### Task 15: 認証と最小権限制御

- Gatewayの呼び出し認証を追加する
- Cluster Agentの認証を追加する
- 管理APIに対する最小限の権限分離を入れる

完了条件:

- 無認証での配備更新や実行が防げる
- agentと管理者APIを分離できる

### Task 16: テスト追加

- repository層のテストを追加する
- service層のユニットテストを追加する
- handler/gatewayのHTTPテストを追加する
- agentの同期フローを最低限テストする

完了条件:

- Function登録、配備、sync、invoke、invocation記録の主要経路にテストがある

### Task 17: ビルドと検証

- `controle-plane` の `go build`, `go test` を通す
- `cluster-agent` の build/test を通す
- `gateway` の build/test を通す
- サンプルFunctionコンテナの起動確認を行う

完了条件:

- 各主要サービスがビルド可能
- 主要ユースケースがローカルで確認できる

## MVPタスク

MVPとして優先するタスクは以下。

1. Task 1: Control Plane APIのFunction拡張
2. Task 2: Clusterモデル追加
3. Task 3: Deployment Target管理追加
4. Task 4: Route管理追加
5. Task 5: Function Controller実装
6. Task 6: Cluster Sync API実装
7. Task 8: Cluster Agent雛形作成
8. Task 9: k3s反映機能実装
9. Task 10: Function Runtime契約定義
10. Task 11: サンプルFunctionコンテナ作成
11. Task 12: Gateway実装
12. Task 13: Invocation Recorder実装
13. Task 14: Metrics Collector実装
14. Task 16: テスト追加
15. Task 17: ビルドと検証

## MVP完了条件

以下を満たしたらMVP完了とみなす。

- Functionをcontainer runtimeとして登録できる
- Functionをclusterへ配備できる
- Cluster Agentがk3sへDeployment/Serviceを反映できる
- Gateway経由でHTTP invokeできる
- Invocation履歴を参照できる
- 基本メトリクスを確認できる

## 後続フェーズ

### Phase 2

- canary rollout
- 複数clusterへの動的振り分け
- Config/Secret配布強化
- retry policy強化

### Phase 3

- Event Dispatcher
- queue trigger
- scale-to-zero
- Build Service
- policy強化

## 結論

Lambda風のFunction基盤を実装するうえで必要なのは、単なるFunction実行機構ではなく、定義管理、配布、実行仲介、履歴保存、監視まで含んだ一連のサービス群である。

MVPでは以下の一式がそろえば成立する。

- 定義管理を行う `Control Plane API`
- 配備計画を生成する `Function Controller`
- k3sへ反映する `Cluster Agent`
- リクエスト入口となる `Gateway`
- 実処理を担う `Function Runtime`
- 実行履歴を保存する `Invocation Recorder`
- 可観測性を担う `Metrics Collector`
