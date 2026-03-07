# サービス一覧

## 目的

EdgeBaseでLambda風のFunction実行基盤を構築する際に必要となるサービス群を整理する。

本資料では以下を明確にする。

- どのサービスが必要か
- それぞれの責務
- 優先度
- 他サービスとの依存関係
- MVPで必須かどうか

## 全体像

サービスは以下の4層に分ける。

- Control Plane層
- Execution Plane層
- Platform Support層
- Observability / Security層

```text
+----------------------------------+
| Control Plane                    |
|  - Control Plane API             |
|  - Function Controller           |
|  - Route Resolver                |
|  - Deployment Target Manager     |
+----------------+-----------------+
                 |
                 v
+----------------+-----------------+
| Execution Plane                  |
|  - Gateway                       |
|  - Cluster Agent                 |
|  - Function Runtime              |
|  - Event Dispatcher              |
+----------------+-----------------+
                 |
                 v
+----------------+-----------------+
| Platform Support                 |
|  - Image Registry Integration    |
|  - Config Distribution Service   |
|  - Build Service                 |
|  - Secret Manager Integration    |
+----------------+-----------------+
                 |
                 v
+----------------+-----------------+
| Observability / Security         |
|  - Invocation Recorder           |
|  - Log Collector                 |
|  - Metrics Collector             |
|  - Auth / Policy Service         |
+----------------------------------+
```

## 1. Control Plane API

### 役割

Function、Route、Deployment Target、Clusterの定義を管理する中核サービス。

### 主な責務

- Function登録
- バージョン管理
- Route管理
- Cluster管理
- Desired State管理

### 依存

- PostgreSQL
- Invocation Recorder
- Function Controller

### 優先度

必須

### MVP対象

対象

## 2. Function Controller

### 役割

Control PlaneのDesired Stateを、各k3sクラスタ上のDeployment/Serviceへ反映する。

### 主な責務

- Function配備計画の生成
- rollout管理
- version切替
- k3s反映対象の決定

### 依存

- Control Plane API
- Cluster Agent
- Deployment Target Manager

### 優先度

必須

### MVP対象

対象

## 3. Deployment Target Manager

### 役割

どのFunctionをどのClusterに配るかを管理する。

### 主な責務

- cluster単位の配備先管理
- namespace管理
- rollout strategy管理
- desired version管理

### 依存

- Control Plane API
- Function Controller

### 優先度

高

### MVP対象

対象

## 4. Route Resolver

### 役割

受信したリクエストから、どのFunctionをどのClusterで実行するか解決する。

### 主な責務

- host/path/method から route 解決
- tenant/region による振り分け
- cluster health に基づく切替
- timeout / retry policy 解決

### 依存

- Control Plane API
- Gateway
- Metrics Collector

### 優先度

高

### MVP対象

最小機能のみ対象

## 5. Gateway

### 役割

外部からのHTTP/Eventの入口。

### 主な責務

- 認証
- request ID 発行
- route 解決
- invocation forwarding
- timeout制御
- レスポンス返却

### 依存

- Route Resolver
- Function Runtime
- Invocation Recorder

### 優先度

必須

### MVP対象

対象

## 6. Cluster Agent

### 役割

各k3sクラスタに常駐し、Control Planeの計画を受けて実クラスタへ適用する。

### 主な責務

- inventory収集
- desired state取得
- k3sリソース反映
- 実行結果ACK

### 依存

- Control Plane API
- Function Controller
- k3s API

### 優先度

必須

### MVP対象

対象

## 7. Function Runtime

### 役割

実際にFunctionコンテナを処理する実行面。

### 主な責務

- Functionコンテナ起動
- リクエスト処理
- ヘルスチェック
- 実行結果返却

### 依存

- Gateway
- k3s
- Config Distribution Service
- Secret Manager Integration

### 優先度

必須

### MVP対象

対象

## 8. Event Dispatcher

### 役割

非同期イベントを適切なFunctionへ配送する。

### 主な責務

- event source受信
- function選択
- queue投入または配送
- retry / dead-letter 制御

### 依存

- Gateway または Broker
- Function Runtime
- Invocation Recorder

### 優先度

中

### MVP対象

対象外

## 9. Invocation Recorder

### 役割

実行履歴を保存し、トラブルシュートと可観測性の基盤を作る。

### 主な責務

- invocation metadata保存
- 実行結果保存
- duration保存
- 失敗理由保存

### 依存

- Gateway
- Control Plane API
- Metrics Collector

### 優先度

必須

### MVP対象

対象

## 10. Log Collector

### 役割

Functionログを収集し、invocation単位で追跡できるようにする。

### 主な責務

- Podログ収集
- invocation_id 相関
- エラー検索

### 依存

- Function Runtime
- Invocation Recorder

### 優先度

高

### MVP対象

最小機能のみ対象

## 11. Metrics Collector

### 役割

Function、Gateway、Clusterの状態を数値で観測する。

### 主な責務

- success rate
- error rate
- p50/p95 latency
- cluster health
- pod restart / saturation

### 依存

- Gateway
- Function Runtime
- Invocation Recorder

### 優先度

高

### MVP対象

対象

## 12. Auth / Policy Service

### 役割

誰が何を作成・更新・実行できるかを制御する。

### 主な責務

- API認証
- RBAC
- function更新権限
- invocation実行権限

### 依存

- Control Plane API
- Gateway

### 優先度

高

### MVP対象

最小機能のみ対象

## 13. Image Registry Integration

### 役割

Functionコンテナイメージを安全に登録し、digest単位で扱う。

### 主な責務

- image参照管理
- digest固定
- 署名検証
- registry認証

### 依存

- Control Plane API
- Build Service
- Function Controller

### 優先度

高

### MVP対象

対象

## 14. Secret Manager Integration

### 役割

Function実行に必要な秘密情報を安全に注入する。

### 主な責務

- APIキー管理
- DB接続情報管理
- Secretのnamespace反映
- rotation対応

### 依存

- Control Plane API
- Function Runtime
- Cluster Agent

### 優先度

高

### MVP対象

最小機能のみ対象

## 15. Config Distribution Service

### 役割

Function設定やfeature flagを各クラスタへ配布する。

### 主な責務

- 環境変数配布
- ConfigMap生成
- cluster別設定管理

### 依存

- Control Plane API
- Cluster Agent
- Function Runtime

### 優先度

中

### MVP対象

最小機能のみ対象

## 16. Build Service

### 役割

ソースコードからFunctionコンテナイメージを生成する。

### 主な責務

- build実行
- image push
- buildログ保存
- build provenance 追跡

### 依存

- Image Registry Integration
- Control Plane API

### 優先度

中

### MVP対象

対象外

## MVPで必須のサービス

- Control Plane API
- Function Controller
- Deployment Target Manager
- Gateway
- Cluster Agent
- Function Runtime
- Invocation Recorder
- Metrics Collector
- Image Registry Integration

## MVPでは最小機能で十分なサービス

- Route Resolver
- Log Collector
- Auth / Policy Service
- Secret Manager Integration
- Config Distribution Service

## 後回しでよいサービス

- Event Dispatcher
- Build Service
- 高度な独自Autoscaler
- 課金基盤
- 高度なマルチテナント制御

## 推奨実装順

1. Control Plane API
2. Deployment Target Manager
3. Function Controller
4. Cluster Agent
5. Function Runtime
6. Gateway
7. Invocation Recorder
8. Metrics Collector
9. Route Resolver
10. Image Registry Integration

## 結論

Lambda風のFunction基盤では、Functionの登録機能だけでは不十分である。

実際に運用可能な基盤にするには、少なくとも以下の一式が必要になる。

- 配布を担う `Function Controller`
- 実行入口となる `Gateway`
- クラスタ適用を行う `Cluster Agent`
- 実行面である `Function Runtime`
- 実行履歴を扱う `Invocation Recorder`
- 監視基盤となる `Metrics Collector`

まずはMVP対象サービスを優先し、その後にイベント処理、ビルド、ポリシー管理を段階追加する方針が現実的である。
