# EdgeBase Implementation Audit

2026-03-07 時点の実装棚卸し。

判定基準:
- `あり`: 基本機能が実装されている
- `不足あり`: 一部はあるが、MVP要件を満たしていない
- `未実装`: 実装が見当たらない

## サマリ

| 機能 | 状態 | メモ |
| --- | --- | --- |
| 認証・認可 | 不足あり | ノード JWT のみ。ユーザー認証、RBAC、失効がない |
| マルチテナント管理 | 未実装 | tenant/project モデルがない |
| API Gateway | 不足あり | middleware はあるが独立した gateway 機能は薄い |
| オブジェクトストレージ | あり | MinIO/S3 抽象あり |
| ノード管理プレーン | 不足あり | 登録、heartbeat はある。失効や証明書更新がない |
| デプロイ管理 | 不足あり | deploy API はあるが実処理が薄い |
| 同期エンジン | 不足あり | sync plan/ack はあるが簡易実装 |
| ルーティング / サービスディスカバリ | 不足あり | route API はあるが永続化されていない |
| 監査ログ | 不足あり | モデルとサービスはあるが未接続 |
| メトリクス / ログ | 不足あり | request ID、ログ、簡易 metrics はある。trace はない |
| シークレット管理 | 不足あり | env ベースのみ |
| キュー / 非同期ジョブ | 未実装 | 専用基盤なし |
| ジョブ再試行 / デッドレター | 未実装 | 一般ジョブ向け実装なし |
| 設定管理 | 不足あり | env ロードのみ |
| 証明書更新 / mTLS | 未実装 | JWT のみ |

## 詳細

### 1. 認証・認可

状態: `不足あり`

あるもの:
- JWT 発行・検証
- Bearer middleware
- ノード登録時の token 発行
- token refresh endpoint

根拠:
- [controle-plane/internal/auth/auth.go](/home/mo/repo/edgebase/controle-plane/internal/auth/auth.go)
- [controle-plane/internal/auth/middleware.go](/home/mo/repo/edgebase/controle-plane/internal/auth/middleware.go)
- [controle-plane/internal/handler/node_handler.go](/home/mo/repo/edgebase/controle-plane/internal/handler/node_handler.go)
- [controle-plane/internal/handler/auth_handler.go](/home/mo/repo/edgebase/controle-plane/internal/handler/auth_handler.go)

不足:
- ユーザー認証
- RBAC
- service account / API key
- token revocation
- actor 種別ごとの claims 分離

次の一手:
- `user`, `service`, `node` の認証主体を分ける
- role / permission を middleware で判定する
- revocation table か token version を追加する

### 2. マルチテナント管理

状態: `未実装`

あるもの:
- 該当モデルなし

根拠:
- [controle-plane/internal/model/model.go](/home/mo/repo/edgebase/controle-plane/internal/model/model.go)

不足:
- organization/project/membership
- tenant_id/project_id
- tenant 境界の認可

次の一手:
- `organizations`, `projects`, `memberships` を追加する
- function, route, node, artifact に owner 境界を持たせる

### 3. API Gateway

状態: `不足あり`

あるもの:
- CORS
- request ID
- logging middleware
- rate limit middleware
- auth middleware

根拠:
- [controle-plane/cmd/server/main.go](/home/mo/repo/edgebase/controle-plane/cmd/server/main.go)
- [controle-plane/internal/cors/cors.go](/home/mo/repo/edgebase/controle-plane/internal/cors/cors.go)
- [controle-plane/internal/logger/middleware.go](/home/mo/repo/edgebase/controle-plane/internal/logger/middleware.go)
- [controle-plane/internal/ratelimit/middleware.go](/home/mo/repo/edgebase/controle-plane/internal/ratelimit/middleware.go)

不足:
- API key 対応
- route policy
- tenant-aware rate limit
- gateway と control-plane application logic の分離

次の一手:
- 共通入口として policy を整理する
- authN/authZ/rate-limit/audit を gateway concern として明示する

### 4. オブジェクトストレージ

状態: `あり`

あるもの:
- MinIO client
- S3 client
- storage interface
- artifact upload/download

根拠:
- [controle-plane/internal/storage/interface.go](/home/mo/repo/edgebase/controle-plane/internal/storage/interface.go)
- [controle-plane/internal/storage/minio.go](/home/mo/repo/edgebase/controle-plane/internal/storage/minio.go)
- [controle-plane/internal/storage/s3.go](/home/mo/repo/edgebase/controle-plane/internal/storage/s3.go)
- [controle-plane/internal/service/artifact_service.go](/home/mo/repo/edgebase/controle-plane/internal/service/artifact_service.go)

不足:
- ログ / バックアップ保存の利用は未整備
- tenant ごとのアクセス境界がない

次の一手:
- artifact path に tenant / project 境界を入れる
- backup/log の責務を明確化する

### 5. ノード管理プレーン

状態: `不足あり`

あるもの:
- ノード登録
- heartbeat
- ノード状態更新
- schema status 更新

根拠:
- [controle-plane/internal/service/node_service.go](/home/mo/repo/edgebase/controle-plane/internal/service/node_service.go)
- [controle-plane/internal/repository/node_repository.go](/home/mo/repo/edgebase/controle-plane/internal/repository/node_repository.go)
- [controle-plane/internal/handler/node_handler.go](/home/mo/repo/edgebase/controle-plane/internal/handler/node_handler.go)

不足:
- ノード失効
- allowlist / denylist
- 証明書更新
- 管理用一覧・状態遷移 API

次の一手:
- `revoked_at` や `status=disabled` を導入する
- heartbeat 期限切れの offline 判定ジョブを入れる

### 6. デプロイ管理

状態: `不足あり`

あるもの:
- function deploy API
- node/function deployment モデル

根拠:
- [controle-plane/internal/handler/function_handler.go](/home/mo/repo/edgebase/controle-plane/internal/handler/function_handler.go)
- [controle-plane/internal/model/model.go](/home/mo/repo/edgebase/controle-plane/internal/model/model.go)

不足:
- `QueueDeployment` が実際には永続化や通知をしていない
- ロールバックなし
- deployment status 更新の流れが薄い

根拠:
- [controle-plane/internal/service/sync_service.go](/home/mo/repo/edgebase/controle-plane/internal/service/sync_service.go)

次の一手:
- deployment record を作成する
- node 別の pending / running / success / failed を更新する
- rollback API を追加する

### 7. 同期エンジン

状態: `不足あり`

あるもの:
- sync plan 生成
- sync ACK
- sync record 保存

根拠:
- [controle-plane/internal/service/sync_service.go](/home/mo/repo/edgebase/controle-plane/internal/service/sync_service.go)
- [controle-plane/internal/repository/sync_repository.go](/home/mo/repo/edgebase/controle-plane/internal/repository/sync_repository.go)

不足:
- pending sync の事前保存がない
- conflict resolution の明示がない
- function / route / deployment 状態の正確な反映が弱い

次の一手:
- plan 作成時に sync record を pending で保存する
- ACK payload に新状態を含める
- route/deployment 差分も同期対象に含める

### 8. ルーティング / サービスディスカバリ

状態: `不足あり`

あるもの:
- route 作成 API
- route 一覧 API
- edge runner 側の route repository 利用

根拠:
- [controle-plane/internal/handler/route_handler.go](/home/mo/repo/edgebase/controle-plane/internal/handler/route_handler.go)
- [functions/edge-runner/src/application/services.rs](/home/mo/repo/edgebase/functions/edge-runner/src/application/services.rs)

不足:
- control plane で route を永続化していない
- service discovery の実体がない
- pop/node 選択ロジックがない

根拠:
- [controle-plane/internal/service/sync_service.go](/home/mo/repo/edgebase/controle-plane/internal/service/sync_service.go)

次の一手:
- route model / repository を追加する
- host/path/method 一意制約を定義する
- POP 選択ルールを定義する

### 9. 監査ログ

状態: `不足あり`

あるもの:
- audit log model
- audit service

根拠:
- [controle-plane/internal/model/audit.go](/home/mo/repo/edgebase/controle-plane/internal/model/audit.go)
- [controle-plane/internal/service/audit_service.go](/home/mo/repo/edgebase/controle-plane/internal/service/audit_service.go)

不足:
- handler/service から呼ばれていない
- actor が node 前提
- user/service actor を扱えない

次の一手:
- create/update/delete/deploy/refresh を記録する
- actor_type, actor_id, tenant_id を追加する

### 10. メトリクス / ログ / トレース

状態: `不足あり`

あるもの:
- request ID
- request logging
- in-memory metrics
- 時系列ストア向け collector/writer の枠組み

根拠:
- [controle-plane/internal/logger/middleware.go](/home/mo/repo/edgebase/controle-plane/internal/logger/middleware.go)
- [controle-plane/internal/metrics/metrics.go](/home/mo/repo/edgebase/controle-plane/internal/metrics/metrics.go)
- [controle-plane/internal/timeseries](/home/mo/repo/edgebase/controle-plane/internal/timeseries)

不足:
- 分散トレースなし
- metrics は Prometheus 形式ではなく簡易 JSON
- log の集約先が未整備

次の一手:
- OpenTelemetry を導入する
- metrics exporter を整理する
- error/event のログ設計を統一する

### 11. シークレット管理

状態: `不足あり`

あるもの:
- env からの設定読込

根拠:
- [controle-plane/internal/config/config.go](/home/mo/repo/edgebase/controle-plane/internal/config/config.go)

不足:
- secret store 連携
- ローテーション
- 監査

次の一手:
- dev は `.env`、prod は Vault / cloud secret manager 前提に切り分ける

### 12. キュー / 非同期ジョブ

状態: `未実装`

あるもの:
- 専用キュー基盤は見当たらない

不足:
- deployment queue
- async worker
- scheduled reconciliation

次の一手:
- 最初は DB-backed job table で十分
- 後で Redis / NATS / SQS 等に切り替え可能な抽象を作る

### 13. ジョブ再試行 / デッドレター

状態: `未実装`

あるもの:
- timeseries write 向け retry interface はある

根拠:
- [controle-plane/internal/timeseries/retry.go](/home/mo/repo/edgebase/controle-plane/internal/timeseries/retry.go)

不足:
- deploy/sync/job 全般への retry policy
- dead-letter queue

次の一手:
- job table に `attempt_count`, `next_run_at`, `dead_lettered_at` を追加する

### 14. 設定管理

状態: `不足あり`

あるもの:
- env ロード
- `.env` 対応

根拠:
- [controle-plane/internal/config/config.go](/home/mo/repo/edgebase/controle-plane/internal/config/config.go)

不足:
- ノード別設定
- DB 管理設定
- feature flag

次の一手:
- `config_profiles` か `node_configs` を追加する

### 15. 証明書更新 / mTLS

状態: `未実装`

あるもの:
- 実コード上は JWT のみ

不足:
- node certificate
- issuance / rotation / revocation
- CP と edge 間の mTLS

次の一手:
- まず node JWT を維持しつつ、Phase 2 で mTLS を追加する

## 優先度の高い差分

1. マルチテナント管理
2. 認証・認可の actor 分離
3. 監査ログの本実装
4. route / deployment の永続化
5. deployment queue の実装
