# デザインドキュメント: WasmEdge エッジ関数基盤

## 概要

WasmEdge エッジ関数基盤は、短命の HTTP ハンドラ関数（WASM モジュール）をエッジノード上で低遅延かつ強力なセキュリティ保証の下で実行する分散型プルベースシステムです。アーキテクチャは、関数・アーティファクト・オーケストレーションを管理する中央の Control Plane と、WasmEdge ランタイムを組み込んだ軽量な Edge Runner に関心を分離しています。システムはホットインスタンスプーリングとローカルキャッシングによるパフォーマンス、WASM サンドボックスとケーパビリティ制限によるセキュリティ、包括的なメトリクスと構造化ログによる可観測性を優先します。

## アーキテクチャ

### ハイレベルシステムアーキテクチャ

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Developer Zone                              │
│  ┌──────────┐      ┌──────────┐      ┌──────────┐                  │
│  │   Git    │      │   CLI    │      │ Web UI   │                  │
│  │  Repo    │      │  Tool    │      │ Console  │                  │
│  └────┬─────┘      └────┬─────┘      └────┬─────┘                  │
│       │                 │                   │                        │
│       └─────────────────┴───────────────────┘                        │
│                         │                                            │
└─────────────────────────┼────────────────────────────────────────────┘
                          │ HTTPS/API
                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       Control Plane (CP)                             │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                      API Gateway (TLS)                         │ │
│  │              /functions  /deploy  /routes  /nodes              │ │
│  └──────┬──────────────────────────────────────────┬──────────────┘ │
│         │                                           │                │
│  ┌──────▼──────────┐  ┌──────────────┐  ┌─────────▼─────────┐      │
│  │  Function Mgr   │  │ Build Worker │  │   Notification    │      │
│  │  - Register     │  │ - Rust/TS    │  │   Service         │      │
│  │  - Versioning   │  │ - AOT        │  │   - gRPC Stream   │      │
│  │  - Routing      │  │ - Validation │  │   - WebSocket     │      │
│  └────────┬────────┘  └──────┬───────┘  └───────┬───────────┘      │
│           │                  │                   │                  │
│  ┌────────▼──────────────────▼───────┐  ┌───────▼───────────┐      │
│  │      PostgreSQL (Metadata)        │  │   Redis (Queue)   │      │
│  │  - functions, routes, nodes       │  │   - Pending       │      │
│  │  - deployments, audit_logs        │  │     deploys       │      │
│  └───────────────────────────────────┘  └───────────────────┘      │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │              MinIO / S3 (Artifact Store)                     │  │
│  │         s3://artifacts/functions/{id}/{version}.wasm         │  │
│  │         + sha256 checksums, presigned URLs                   │  │
│  └──────────────────────────────────────────────────────────────┘  │
└──────────────────────────────┬───────────────────────────────────────┘
                               │ gRPC/WebSocket (mTLS)
                               │ Heartbeat + Deploy Notifications
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Edge Network (POP)                           │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    Edge Node (Tokyo-1)                       │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │              HTTP Server (Go)                          │  │  │
│  │  │  - TLS termination                                     │  │  │
│  │  │  - Route matching (host + path)                        │  │  │
│  │  │  - Request validation                                  │  │  │
│  │  └──────────────────┬─────────────────────────────────────┘  │  │
│  │                     │                                         │  │
│  │  ┌──────────────────▼─────────────────────────────────────┐  │  │
│  │  │           Function Router & Dispatcher               │  │  │
│  │  │  - Route lookup                                       │  │  │
│  │  │  - Function resolution                                │  │  │
│  │  │  - Load balancing (hot instances)                     │  │  │
│  │  └──────────────────┬─────────────────────────────────────┘  │  │
│  │                     │                                         │  │
│  │  ┌──────────────────▼─────────────────────────────────────┐  │  │
│  │  │          WasmEdge Runtime Manager                     │  │  │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐            │  │  │
│  │  │  │ Hot Pool │  │ Hot Pool │  │ Hot Pool │            │  │  │
│  │  │  │ func-A   │  │ func-B   │  │ func-C   │            │  │  │
│  │  │  │ (reuse)  │  │ (reuse)  │  │ (reuse)  │            │  │  │
│  │  │  └──────────┘  └──────────┘  └──────────┘            │  │  │
│  │  │                                                        │  │  │
│  │  │  - Memory limit enforcement (pages)                   │  │  │
│  │  │  - Timeout enforcement (context)                      │  │  │
│  │  │  - WASI capability control                            │  │  │
│  │  │  - LRU eviction policy                                │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  │                                                               │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │              Local Cache Manager                      │  │  │
│  │  │  /var/cache/wasm/{function_id}/{version}.wasm         │  │  │
│  │  │  - SHA256 verification                                │  │  │
│  │  │  - LRU eviction (size/count limits)                   │  │  │
│  │  │  - Atomic updates                                     │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  │                                                               │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │         CP Communication Client                       │  │  │
│  │  │  - gRPC bidirectional stream / WebSocket              │  │  │
│  │  │  - Heartbeat (30s interval)                           │  │  │
│  │  │  - Deploy notification receiver                       │  │  │
│  │  │  - Exponential backoff retry                          │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  │                                                               │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │         Observability Exporter                        │  │  │
│  │  │  - Prometheus metrics (:9090/metrics)                 │  │  │
│  │  │  - Structured JSON logs                               │  │  │
│  │  │  - OpenTelemetry traces                               │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────────┐   │
│  │ Edge (Tokyo-2)   │  │ Edge (Osaka-1)   │  │ Edge (US-West)  │   │
│  │ (same structure) │  │ (same structure) │  │ (same structure)│   │
│  └──────────────────┘  └──────────────────┘  └─────────────────┘   │
└──────────────────────────────────────────────────────────────────────┘
                               │
                               │ HTTP/HTTPS
                               ▼
                        ┌──────────────┐
                        │  End Users   │
                        └──────────────┘
```

### データフロー: 関数デプロイ

```
開発者 → CP API → ビルドワーカー → MinIO/S3 → PostgreSQL → 通知サービス → Edge Runner
```

1. 開発者が Git push または CLI で関数コードを送信
2. CP が関数レコードを作成し、署名付きアップロード URL を生成
3. ビルドワーカーが WASM にコンパイルし SHA256 を計算
4. アーティファクトを MinIO/S3 にアップロード
5. CP が関数メタデータを artifact_url と sha256 で更新
6. 通知サービスがターゲットエッジノードにデプロイ指示をプッシュ
7. Edge Runner がアーティファクトをプル、SHA256 を検証、ローカルにキャッシュ

### データフロー: リクエスト実行

```
エンドユーザー → Edge HTTP サーバー → ルート検索 → ホットプール/コールドスタート → WasmEdge ランタイム → レスポンス
```

1. HTTP リクエストが Edge Runner に到着
2. ルートマッチングがターゲット関数を決定
3. ホットインスタンスプール内で利用可能なインスタンスをチェック
4. 利用可能な場合: 再利用（< 1ms オーバーヘッド）
5. 利用不可の場合: キャッシュまたは S3 からプル、新しいインスタンスを作成（コールドスタート: 10-50ms）
6. 強制された制限で WASM 関数を実行
7. HTTP レスポンスを返し、メトリクスを更新

### データフロー: ハートビート & 同期

```
Edge Runner（30秒ごと） → CP API → 保留中のデプロイをチェック → 指示を返す
```

1. Edge Runner がノードステータスと関数インベントリを含むハートビートを送信
2. CP が last_heartbeat タイムスタンプを更新
3. CP が保留中のデプロイ/アンデプロイ通知をチェック
4. CP がハートビートレスポンスで指示を返す
5. Edge Runner が指示を非同期で処理

## コンポーネントとインターフェース

### コンポーネント 1: Control Plane API サーバー

**責務:**
- 関数登録とバージョニング
- ルート管理
- デプロイメントオーケストレーション
- ノード管理とヘルスチェック
- Edge Runner への通知配信
- 認証と認可

**主要エンドポイント:**
- `POST /api/v1/functions` - 新しい関数を登録
- `POST /api/v1/functions/:id/deploy` - 関数バージョンをデプロイ
- `POST /api/v1/routes` - HTTP ルートを作成
- `POST /api/v1/nodes/:id/heartbeat` - Edge Runner からハートビートを受信
- `GET /api/v1/nodes` - エッジノードをリスト表示

**技術スタック:**
- 言語: Go または Rust
- フレームワーク: Gin/Axum
- データベース: PostgreSQL
- キャッシュ: Redis（保留中のデプロイメント用）
- アーティファクトストア: MinIO/S3

### コンポーネント 2: Edge Runner

**責務:**
- HTTP リクエスト処理とルーティング
- WasmEdge 経由の WASM 関数実行
- ホットインスタンスプール管理
- ローカル WASM キャッシュ管理
- CP とのハートビートと同期
- メトリクス収集とエクスポート
- セキュリティ強制（WASI ケーパビリティ、リソース制限）

**主要モジュール:**
- HTTP サーバー: TLS 終了、ルートマッチング、リクエスト検証
- 関数ルーター: ルート検索、関数解決、ロードバランシング
- WasmEdge ランタイムマネージャー: インスタンスライフサイクル、メモリ/タイムアウト強制、WASI ポリシー
- ローカルキャッシュマネージャー: ファイルストレージ、SHA256 検証、LRU 削除
- CP 通信クライアント: gRPC/WebSocket、ハートビート、デプロイ通知
- 可観測性エクスポーター: Prometheus メトリクス、構造化ログ、OpenTelemetry トレース

**技術スタック:**
- 言語: Rust
- HTTP フレームワーク: Actix-web または Axum
- WASM ランタイム: wasmedge-rust-sdk (WasmEdge Rust SDK)
- メトリクス: Prometheus Rust クライアント
- ログ: 構造化 JSON（tracing または slog）
- 通信: gRPC または WebSocket

### コンポーネント 3: ビルドワーカー

**責務:**
- ソースコードを WASM にコンパイル
- 必要に応じて AOT（事前コンパイル）を実行
- SHA256 チェックサムを計算
- WASM バイトコードを検証
- アーティファクトを MinIO/S3 にアップロード

**技術スタック:**
- 言語: Go/Rust/TypeScript（ソース言語に応じて）
- ビルドツール: esbuild、wasm-pack、Rust ツールチェーン
- 検証: wasmtime または wasmedge CLI

### コンポーネント 4: 可観測性スタック

**責務:**
- メトリクスを収集・保存
- システムヘルスを可視化
- ログを保存・クエリ
- 分散トレーシング

**技術スタック:**
- メトリクス: Prometheus
- 可視化: Grafana
- ログ: Loki または ELK
- トレース: Jaeger または OpenTelemetry Collector

## データモデル

### Function（関数）

```sql
CREATE TABLE functions (
  id UUID PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  version VARCHAR(50) NOT NULL,
  owner_project_id UUID NOT NULL,
  runtime ENUM('wasm', 'wasm-aot') NOT NULL,
  entrypoint VARCHAR(255) NOT NULL,
  artifact_url VARCHAR(1024) NOT NULL,
  sha256 VARCHAR(64) NOT NULL,
  memory_pages INT NOT NULL,
  max_execution_ms INT NOT NULL,
  pop_selector VARCHAR(1024),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(name, version, owner_project_id)
);
```

### Route（ルート）

```sql
CREATE TABLE routes (
  id UUID PRIMARY KEY,
  host VARCHAR(255) NOT NULL,
  path VARCHAR(1024) NOT NULL,
  function_id UUID NOT NULL REFERENCES functions(id),
  methods VARCHAR(50) NOT NULL,
  pop_selector VARCHAR(1024),
  priority INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(host, path, methods)
);
```

### Node（ノード）

```sql
CREATE TABLE nodes (
  id UUID PRIMARY KEY,
  pop_id VARCHAR(255) NOT NULL,
  ip VARCHAR(45) NOT NULL,
  last_heartbeat TIMESTAMP,
  status ENUM('online', 'degraded', 'offline') NOT NULL,
  metadata JSONB,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Deployment（デプロイメント）

```sql
CREATE TABLE deployments (
  id UUID PRIMARY KEY,
  function_id UUID NOT NULL REFERENCES functions(id),
  version VARCHAR(50) NOT NULL,
  target_pop VARCHAR(1024) NOT NULL,
  status ENUM('pending', 'in_progress', 'completed', 'failed') NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 正確性プロパティ

プロパティは、システムのすべての有効な実行を通じて真であるべき特性または動作です。本質的には、システムが何をすべきかについての形式的なステートメントです。プロパティは、人間が読める仕様と機械検証可能な正確性保証の間の橋渡しとなります。

### プロパティ 1: 関数登録の永続性

*任意の* 有効な関数登録リクエスト（name、entrypoint、memory_pages、max_execution_ms を含む）に対して、登録成功後、Control Plane にその関数をクエリすると、同じメタデータが返されるべきです。

**検証: 要件 1.1, 1.2**

### プロパティ 2: アーティファクト SHA256 検証ラウンドトリップ

*任意の* Control Plane にアップロードされた WASM アーティファクトについて、Control Plane が計算した SHA256 チェックサムは、Edge Runner が同じアーティファクトをダウンロードするときに計算した SHA256 チェックサムと一致するべきです。

**検証: 要件 1.2, 10.1, 10.2**

### プロパティ 3: デプロイメント通知配信

*任意の* POP セレクターにマッチするエッジノードをターゲットとするデプロイメントリクエストについて、マッチするすべてのエッジノードは、正しい function_id、version、artifact_url、sha256、memory_pages、max_execution_ms を含むデプロイメント通知を受け取るべきです。

**検証: 要件 2.1, 2.3**

### プロパティ 4: ホットインスタンスプール再利用

*任意の* ホットインスタンスプール内に利用可能なインスタンスを持つ関数について、その関数の連続した呼び出しは、新しいインスタンスを作成せずに同じインスタンスを再利用し、コールドスタートより低いレイテンシを実現するべきです。

**検証: 要件 3.2, 4.1, 4.2**

### プロパティ 5: ローカルキャッシュ一貫性

*任意の* Edge Runner にローカルキャッシュされた WASM アーティファクトについて、キャッシュされたファイルの SHA256 チェックサムは、デプロイメント通知から期待されるチェックサムと一致するべきです。

**検証: 要件 5.3, 10.2, 10.3**

### プロパティ 6: ハートビートステータス同期

*任意の* キャッシュされた関数のリストを含むハートビートを送信する Edge Runner について、Control Plane はノードの last_heartbeat タイムスタンプを更新し、そのノード上の関数の正確なインベントリを維持するべきです。

**検証: 要件 6.1, 6.2**

### プロパティ 7: メモリ制限強制

*任意の* memory_pages 制限で インスタンス化された WASM 関数について、その関数は memory_pages * 64 KiB を超えるメモリを割り当てることができず、この制限を超えようとする試みは失敗するべきです。

**検証: 要件 7.1**

### プロパティ 8: WASI ケーパビリティ制限

*任意の* ホワイトリスト外の WASI ケーパビリティ（例：生ソケットアクセス、ファイルシステム書き込み）にアクセスしようとする WASM 関数について、アクセスは拒否され、関数実行はエラーで失敗するべきです。

**検証: 要件 7.2**

### プロパティ 9: 実行タイムアウト強制

*任意の* max_execution_ms タイムアウトを持つ WASM 関数について、関数実行がこのタイムアウトを超える場合、Edge Runner は実行を終了し、504 Gateway Timeout レスポンスを返すべきです。

**検証: 要件 3.4, 7.3**

### プロパティ 10: メトリクス収集精度

*任意の* Edge Runner 上の関数呼び出しについて、wasm_invoke_count_total メトリクスは 1 だけ増加し、wasm_invoke_latency_seconds ヒストグラムは実行レイテンシを記録するべきです。

**検証: 要件 8.1**

### プロパティ 11: ルートマッチング優先度

*任意の* 複数のマッチングルートを持つ Edge Runner に到着する HTTP リクエストについて、最も高い優先度を持つルートがディスパッチ用に選択されるべきです。

**検証: 要件 12.2**

### プロパティ 12: バージョン分離

*任意の* 複数のバージョンがデプロイされた関数について、各バージョンは個別のキャッシュエントリとホットインスタンスを維持し、異なるバージョンの同時実行を可能にするべきです。

**検証: 要件 13.1**

### プロパティ 13: 指数バックオフ再試行

*任意の* 失敗したアーティファクトダウンロード試行について、Edge Runner は指数バックオフ（初期遅延: 1s、最大遅延: 60s）で最大 5 回まで再試行してから失敗を報告するべきです。

**検証: 要件 11.1**

### プロパティ 14: オフラインノード処理

*任意の* タイムアウト期間（例：2 分）以上ハートビートを送信しない Edge Runner について、Control Plane はノードをオフラインとしてマークし、それへのリクエストルーティングを停止するべきです。

**検証: 要件 6.4**

### プロパティ 15: LRU 削除正確性

*任意の* サイズ制限を超えるホットインスタンスプールまたはローカルキャッシュについて、最も最近使用されていないアイテムが最初に削除され、削除はサイズが制限内に収まるまで続くべきです。

**検証: 要件 4.2, 4.3, 5.2**

## エラーハンドリング

### ネットワーク障害

- **アーティファクトダウンロード失敗**: Edge Runner は指数バックオフ（1s → 60s、最大 5 回）で再試行
- **CP 通信喪失**: Edge Runner はキャッシュされた関数の実行を継続し、再接続を試みる
- **ハートビートタイムアウト**: CP はハートビートなしで 2 分後にノードをオフラインとしてマーク

### 検証エラー

- **無効な関数登録**: CP は 400 Bad Request を返し、検証エラーの詳細を含む
- **無効な WASM アーティファクト**: CP はアーティファクトを拒否してエラーを返す；Edge Runner は破損したダウンロードを拒否
- **SHA256 不一致**: Edge Runner はアーティファクトを拒否、破損したファイルを削除、CP にエラーを報告

### リソース枯渇

- **メモリ圧力**: Edge Runner は LRU ポリシーを使用してホットインスタンスを削除
- **キャッシュサイズ超過**: Edge Runner は最も最近使用されていない WASM ファイルを削除
- **クォータ超過**: Edge Runner は 429 Too Many Requests を返す

### 実行エラー

- **WASM ランタイムエラー**: Edge Runner はエラーメッセージを含む 500 Internal Server Error を返す
- **タイムアウト超過**: Edge Runner は 504 Gateway Timeout を返す
- **WASI ケーパビリティ拒否**: 関数実行はエラーで失敗

## テスト戦略

### ユニットテスト

ユニットテストは特定の例とエッジケースを検証します：

- 有効/無効なパラメータでの関数登録
- さまざまなホスト/パスパターンでのルートマッチング
- 正しい/不正なチェックサムでの SHA256 検証
- ホットインスタンスプール操作（作成、再利用、削除）
- ローカルキャッシュ操作（保存、取得、削除）
- ハートビート処理とノードステータス更新
- エラーハンドリングと再試行ロジック

### プロパティベーステスト

プロパティベーステストは、すべての入力を通じて成立するべき普遍的なプロパティを検証します：

- **プロパティ 1**: 関数登録の永続性（ラウンドトリップ）
- **プロパティ 2**: アーティファクト SHA256 検証（ラウンドトリップ）
- **プロパティ 3**: デプロイメント通知配信（普遍的プロパティ）
- **プロパティ 4**: ホットインスタンスプール再利用（パフォーマンスプロパティ）
- **プロパティ 5**: ローカルキャッシュ一貫性（不変量）
- **プロパティ 6**: ハートビートステータス同期（不変量）
- **プロパティ 7**: メモリ制限強制（不変量）
- **プロパティ 8**: WASI ケーパビリティ制限（不変量）
- **プロパティ 9**: 実行タイムアウト強制（不変量）
- **プロパティ 10**: メトリクス収集精度（不変量）
- **プロパティ 11**: ルートマッチング優先度（不変量）
- **プロパティ 12**: バージョン分離（不変量）
- **プロパティ 13**: 指数バックオフ再試行（不変量）
- **プロパティ 14**: オフラインノード処理（不変量）
- **プロパティ 15**: LRU 削除正確性（不変量）

### プロパティベーステストフレームワーク

- **言語**: Rust（edge-runner および control-plane コンポーネント用）
- **フレームワーク**: Rust 用 `proptest` または `quickcheck`
- **設定**: プロパティテストあたり最小 100 回の反復
- **テスト注釈形式**: 各プロパティテストには以下の形式のコメントを含めるべきです：
  ```
  // **Feature: wasmEdge-edge-functions, Property N: [Property Description]**
  // **Validates: Requirements X.Y**
  ```

### テストカバレッジ目標

- すべての受け入れ基準は対応するプロパティベーステストを持つべき
- エッジケース（空の入力、境界値、エラー条件）がカバーされるべき
- コンポーネント間の統合ポイントは統合テストを持つべき
- パフォーマンス特性（レイテンシ、スループット）はベンチマークで検証されるべき

