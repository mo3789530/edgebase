# k3s集約サービス設計

## 目的

EdgeBaseの既存Control Planeを拡張し、複数のk3sクラスタを一元管理する。

この設計で目指すのは、各k3sクラスタに対して以下を提供すること。

- クラスタ登録
- クラスタ状態収集
- Desired Stateの配布
- 差分同期の実行管理
- EdgeBaseのFunction/Schema配布先の集約管理

## 前提

既存の `controle-plane` には以下の基盤がある。

- ノード登録とハートビート
- 同期計画の生成
- Function管理
- Schema管理

関連箇所:

- `controle-plane/internal/service/node_service.go`
- `controle-plane/internal/service/sync_service.go`
- `controle-plane/internal/repository/node_repository.go`
- `controle-plane/internal/model/model.go`

既存構成を踏まえると、k3s集約は別サービスとして完全分離するより、Control Plane内の新しいドメインとして追加するほうが自然である。

## スコープ

### 対象

- 複数k3sクラスタのメタデータ管理
- クラスタのInventory収集
- クラスタごとのDesired State管理
- EdgeBase Functionの配布ターゲット管理
- 同期履歴と状態監視

### 対象外

- Kubernetes APIの常時直接制御
- すべてのKubernetesリソースの汎用管理
- Helm/ArgoCD相当の完全なGitOps基盤
- マルチクラスタネットワーク制御

## 設計方針

### 1. Clusterを新設する

既存の `Node` はEdgeBaseの実行単位として残し、k3sを表す上位概念として `Cluster` を追加する。

- `Node`: EdgeBase agent / edge runtimeの単位
- `Cluster`: k3sクラスタの単位

`Node` にk3sの情報を寄せると責務が混ざるため、Clusterを独立モデルとして扱う。

### 2. Control PlaneはDesired Stateを持つ

Control Planeは各クラスタに対するDesired Stateを保持し、agentがActual Stateを報告する。

- Control Plane: あるべき状態を保持
- Cluster Agent: 実際の状態を収集し、差分を適用

これは既存の `sync_service.go` の同期思想に揃う。

### 3. 初期フェーズはAgent経由で運用する

Control Planeが各クラスタのKubernetes APIを直接常時叩く方式は避ける。

代わりに、各k3sクラスタ側に配置したagentが以下を行う。

- クラスタInventory収集
- Desired State取得
- 差分適用
- 結果ACK送信

これにより認証情報管理と疎通要件を簡素化できる。

## 全体アーキテクチャ

```text
+------------------------+
| EdgeBase Control Plane |
|  - Cluster Service     |
|  - Cluster Sync        |
|  - Function Service    |
|  - Schema Service      |
+-----------+------------+
            |
     HTTP / MQTT
            |
+-----------v------------+
| Cluster Agent          |
|  - Inventory Collector |
|  - Reconciler          |
|  - k3s API Client      |
+-----------+------------+
            |
      Kubernetes API
            |
+-----------v------------+
| k3s Cluster            |
|  - Nodes               |
|  - Workloads           |
|  - Configs/Secrets     |
+------------------------+
```

## ドメインモデル

## Cluster

k3sクラスタそのものを表す。

想定項目:

- `id`
- `name`
- `region`
- `environment`
- `status`
- `api_endpoint`
- `labels`
- `last_heartbeat_at`
- `last_inventory_at`
- `created_at`
- `updated_at`

## ClusterCredential

クラスタ接続に必要な情報を保持する。

想定項目:

- `id`
- `cluster_id`
- `auth_type`
- `server`
- `token_encrypted`
- `client_cert_encrypted`
- `client_key_encrypted`
- `ca_cert`
- `created_at`
- `updated_at`

注意:

- kubeconfig全文の平文保存は避ける
- Tokenや秘密鍵は暗号化して保存する

## ClusterNode

クラスタ配下ノードのInventoryを管理する。

想定項目:

- `id`
- `cluster_id`
- `node_name`
- `role`
- `internal_ip`
- `status`
- `kubelet_version`
- `os_image`
- `container_runtime`
- `last_seen_at`

## ClusterWorkloadTarget

どのFunctionや構成をどのクラスタへ配るかを管理する。

想定項目:

- `id`
- `cluster_id`
- `target_type`
- `target_id`
- `namespace`
- `status`
- `desired_version`
- `last_applied_version`

`target_type` の例:

- `function`
- `schema`
- `config`

## ClusterSyncRecord

同期実行の履歴を保持する。

想定項目:

- `id`
- `cluster_id`
- `sync_type`
- `status`
- `started_at`
- `completed_at`
- `error_message`
- `changes_summary`

## サービス分割

既存の `controle-plane/internal/service` に以下を追加する。

### ClusterService

責務:

- クラスタ登録
- クラスタ一覧/詳細取得
- ステータス更新
- 認証情報関連の登録

### ClusterInventoryService

責務:

- agentから送られたInventoryの保存
- ノード一覧更新
- クラスタ状態の正規化

### ClusterSyncService

責務:

- Desired State生成
- Actual Stateとの差分判定
- 同期計画作成
- 実行結果のACK処理

### ClusterTargetService

責務:

- Function/Schema/Configの配布先管理
- クラスタごとのデプロイ意図の保持

## Repository分割

既存の `controle-plane/internal/repository` に以下を追加する。

- `cluster_repository.go`
- `cluster_credential_repository.go`
- `cluster_inventory_repository.go`
- `cluster_sync_repository.go`
- `cluster_target_repository.go`

## Handler/API設計

既存の handler 分割に合わせて `cluster_handler.go` を追加する。

### クラスタ登録

`POST /clusters`

リクエスト例:

```json
{
  "name": "tokyo-edge-1",
  "region": "ap-northeast-1",
  "environment": "prod",
  "api_endpoint": "https://10.0.0.10:6443",
  "labels": {
    "tier": "edge",
    "provider": "onprem"
  }
}
```

レスポンス例:

```json
{
  "cluster": {
    "id": "uuid",
    "name": "tokyo-edge-1",
    "status": "online"
  },
  "token": "agent-token"
}
```

### クラスタ一覧

`GET /clusters`

### クラスタ詳細

`GET /clusters/:id`

### ハートビート

`POST /clusters/:id/heartbeat`

用途:

- 生存確認
- version情報更新
- last_seen更新

### Inventory送信

`POST /clusters/:id/inventory`

用途:

- ノード一覧
- ラベル
- 稼働中ワークロード
- クラスタVersion

### Desired State取得

`GET /clusters/:id/sync`

用途:

- agentが同期計画を取得する

### 同期結果ACK

`POST /clusters/:id/sync/ack`

用途:

- 適用結果送信
- 差分反映
- 監査/履歴記録

## 同期フロー

### 1. 登録

1. agentがクラスタ登録を実行
2. Control PlaneがClusterを作成
3. agent用トークンを返却

### 2. 定期収集

1. agentがheartbeat送信
2. agentがinventory送信
3. Control PlaneがClusterNodeなどを更新

### 3. 差分同期

1. agentが `GET /clusters/:id/sync` を呼ぶ
2. Control PlaneがDesired Stateを計算
3. 差分アクションを返す
4. agentがk3sへ適用
5. agentがACKを送信
6. Control PlaneがSyncRecordを保存

## Desired Stateの内容

初期フェーズでは以下に絞る。

- EdgeBase Functionの配布対象
- ConfigMap相当の設定配布
- Schema/Version情報

将来的には以下も追加可能。

- Namespace作成
- Secret参照設定
- DaemonSet/Deploymentテンプレート

## データベース変更方針

初期追加候補テーブル:

- `clusters`
- `cluster_credentials`
- `cluster_nodes`
- `cluster_workload_targets`
- `cluster_sync_records`

既存 `model.go` にすべて詰め込むより、Cluster系モデルを別ファイルへ分割したほうが保守しやすい。

推奨:

- `internal/model/cluster.go`
- `internal/model/cluster_sync.go`

## セキュリティ方針

- agent認証は既存JWT方式を流用可能
- クラスタ秘密情報は暗号化保存
- 平文kubeconfigの保存は避ける
- Control Planeからクラスタへ直接接続する設計は初期フェーズでは採らない

## 実装フェーズ

### Phase 1

- Clusterモデル追加
- Cluster登録API追加
- Cluster heartbeat API追加
- Cluster一覧/詳細API追加

成果:

- 複数k3sクラスタをControl Planeで識別できる

### Phase 2

- Inventory API追加
- ClusterNode管理追加
- クラスタ状態の可視化追加

成果:

- 各クラスタ配下ノードの把握ができる

### Phase 3

- ClusterSyncService追加
- Desired State生成
- Sync ACK処理追加

成果:

- Control Plane主導の差分同期が可能になる

### Phase 4

- Function配布ターゲット管理
- Schema/Config配布
- 監査ログ拡充

成果:

- EdgeBaseの配布管理をk3sクラスタ単位で実行できる

## 初回実装の推奨PR分割

### PR 1

- Cluster系モデル
- Cluster repository
- Cluster service
- Cluster handler

### PR 2

- Inventoryモデル
- Inventory API
- ClusterNode保存処理

### PR 3

- ClusterSyncService
- Desired State API
- SyncRecord保存

## 実装タスク

### Task 1: Clusterドメイン追加

- `controle-plane/internal/model` にCluster系モデルを追加する
- `Cluster`, `ClusterCredential`, `ClusterNode`, `ClusterWorkloadTarget`, `ClusterSyncRecord` を定義する
- `cmd/server/main.go` の AutoMigrate 対象に追加する

完了条件:

- Cluster系テーブルが起動時に作成される
- 既存モデルとの責務衝突がない

### Task 2: Cluster Repository実装

- `controle-plane/internal/repository/cluster_repository.go` を追加する
- Clusterの `Create`, `GetByID`, `List`, `UpdateStatus`, `UpdateHeartbeat` を実装する
- 必要に応じて credential / inventory / sync 向け repository を分割追加する

完了条件:

- service層がDB直接操作せず repository 経由で扱える
- 一覧取得と詳細取得が可能

### Task 3: ClusterService実装

- `controle-plane/internal/service/cluster_service.go` を追加する
- クラスタ登録、一覧、詳細取得、heartbeat更新を実装する
- 認証トークン払い出し方針をNode登録と揃える

完了条件:

- handlerからClusterServiceを呼び出せる
- 登録時にagent利用用トークンを返せる

### Task 4: Cluster Handler/API追加

- `controle-plane/internal/handler/cluster_handler.go` を追加する
- 以下のAPIを追加する
- `POST /clusters`
- `GET /clusters`
- `GET /clusters/:id`
- `POST /clusters/:id/heartbeat`
- request body のバリデーションを追加する

完了条件:

- Clusterの登録、一覧、詳細、heartbeatがHTTP経由で利用できる
- 異常系レスポンスが既存handlerの形式に揃う

### Task 5: ルーティングとDI追加

- `handler.NewHandler` または関連構造体へ ClusterService を注入する
- `RegisterRoutes` にCluster系エンドポイントを追加する
- `cmd/server/main.go` で repository / service / handler の配線を追加する

完了条件:

- サーバ起動時にCluster APIが有効化される
- 既存Node APIに影響がない

### Task 6: Cluster Inventory受信機能

- `ClusterInventoryService` を追加する
- `POST /clusters/:id/inventory` を追加する
- クラスタ配下ノード情報を `ClusterNode` に保存できるようにする

完了条件:

- agentから受けたノード一覧が永続化される
- clusterごとの最新inventory時刻が更新される

### Task 7: Cluster Sync基盤実装

- `ClusterSyncService` を追加する
- `GET /clusters/:id/sync` を実装する
- `POST /clusters/:id/sync/ack` を実装する
- Desired StateとActual Stateの差分から同期計画を返す

完了条件:

- cluster単位で同期計画を取得できる
- 同期結果を履歴保存できる

### Task 8: Function配布ターゲット管理

- `ClusterWorkloadTarget` を用いた配布先管理を追加する
- FunctionとClusterの紐付けAPIまたは内部管理機能を追加する
- clusterごとのDesired Function一覧を生成できるようにする

完了条件:

- どのFunctionをどのClusterへ配るか管理できる
- sync時にcluster別のFunction差分を返せる

### Task 9: テスト追加

- repository層のテストを追加する
- service層のユニットテストを追加する
- handler層のHTTPテストを追加する
- 正常系と異常系を最低限カバーする

完了条件:

- Cluster登録、一覧、heartbeatのテストがある
- inventory / sync APIに対する最低限のテストがある

### Task 10: ビルドと検証

- `cd controle-plane && go test ./...` を通す
- `cd controle-plane && go build ./cmd/server` を通す
- 必要に応じて `go vet ./...` を通す

完了条件:

- 既存テストを壊していない
- 新規追加分がビルド可能

## 実装順序

1. Task 1: Clusterドメイン追加
2. Task 2: Cluster Repository実装
3. Task 3: ClusterService実装
4. Task 4: Cluster Handler/API追加
5. Task 5: ルーティングとDI追加
6. Task 9: テスト追加
7. Task 10: ビルドと検証
8. Task 6: Cluster Inventory受信機能
9. Task 7: Cluster Sync基盤実装
10. Task 8: Function配布ターゲット管理

## MVP定義

MVPでは以下を完了条件とする。

- Clusterの登録ができる
- Clusterの一覧と詳細が見られる
- Cluster heartbeat を受けられる
- Inventoryの受信と保存ができる
- 将来のsync拡張に耐えるモデル分離ができている

MVPではまだ必須ではないもの:

- k3s APIの直接操作
- 汎用Kubernetesリソース管理
- 高度な差分同期
- Helm/ArgoCD連携

## リスク

### 1. NodeとClusterの責務衝突

`Node` にk3s情報を寄せると、既存のEdgeBaseノード概念と混ざる。

対策:

- Clusterを独立モデルにする

### 2. 認証情報の取り扱い

k3s接続情報を平文で保存すると事故リスクが高い。

対策:

- 暗号化保存
- 初期はagent pull方式を優先

### 3. 同期責務の肥大化

最初からKubernetes全般を扱うと設計が崩れる。

対策:

- Function/Config/Schemaに対象を限定する

## 結論

EdgeBaseにおけるk3s集約は、既存Control Planeを拡張する形で実装するのが適切である。

特に重要なのは以下の3点。

- `Node` ではなく `Cluster` を新設すること
- Control PlaneはDesired State管理に徹すること
- 実際の適用はCluster Agent経由で行うこと

この方針であれば、既存のNode/Sync/Function管理の延長として実装でき、段階的な導入もしやすい。
