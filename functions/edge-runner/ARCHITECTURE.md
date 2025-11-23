# Edge Runner - レイヤードアーキテクチャ

## ディレクトリ構造

```
src/
├── domain/                    # ドメインレイヤー
│   ├── mod.rs
│   ├── models.rs             # ドメインモデル
│   └── repository.rs         # リポジトリインターフェース
├── application/              # アプリケーションレイヤー
│   ├── mod.rs
│   ├── dto.rs               # データ転送オブジェクト
│   └── services.rs          # ビジネスロジック
├── infrastructure/           # インフラストラクチャレイヤー
│   ├── mod.rs
│   ├── repositories.rs      # リポジトリ実装
│   ├── pool.rs              # ホットインスタンスプール
│   ├── cp_client.rs         # Control Plane クライアント
│   ├── metrics.rs           # メトリクス
│   ├── cache.rs             # WASM キャッシュ
│   └── routing_tests.rs     # ルーティング単体テスト
├── presentation/             # プレゼンテーションレイヤー
│   ├── mod.rs
│   └── handlers.rs          # HTTP ハンドラ
└── main.rs                   # エントリーポイント
```

## レイヤー説明

### Domain Layer (ドメインレイヤー)
ビジネスロジックに関連するコアモデルと抽象化を定義します。

**models.rs:**
- `CachedFunction`: キャッシュ済み関数情報
- `FunctionMetadata`: 関数メタデータ
- `Route`: HTTP ルート定義
- `RouteMatch`: ルートマッチング結果（パスパラメータ含む）
- `DeploymentNotification`: デプロイメント通知
- `PooledInstance`: WASM インスタンス
- `NodeInfo`: ノード情報

**repository.rs:**
- `FunctionRepository`: 関数管理インターフェース
- `RouteRepository`: ルート管理インターフェース（ルーティング機能）
- `CacheRepository`: キャッシュ管理インターフェース

### Application Layer (アプリケーションレイヤー)
ユースケースとビジネスロジックを実装します。

**dto.rs:**
- `HeartbeatRequest/Response`: ハートビート通信
- `InvocationRequest/Response`: 関数呼び出し

**services.rs:**
- `FunctionService`: 関数管理ロジック（ルーティング統合）
- `HeartbeatService`: ハートビート処理
- `InvocationService`: 関数実行ロジック

### Infrastructure Layer (インフラストラクチャレイヤー)
外部システムとの連携を実装します。

**repositories.rs:**
- `InMemoryFunctionRepository`: メモリ内関数リポジトリ
- `InMemoryRouteRepository`: メモリ内ルートリポジトリ（ルーティング実装）
  - `path_matches()`: パスパターンマッチング
  - `extract_path_params()`: パスパラメータ抽出

**pool.rs:**
- `HotInstancePool`: WASM インスタンスプール管理

**cp_client.rs:**
- `ControlPlaneClient`: Control Plane との通信

**metrics.rs:**
- Prometheus メトリクス定義

**cache.rs:**
- `LocalWasmCache`: ローカル WASM キャッシュ

**routing_tests.rs:**
- ルーティング機能の単体テスト（11テスト）

### Presentation Layer (プレゼンテーションレイヤー)
HTTP リクエスト/レスポンスを処理します。

**handlers.rs:**
- `HttpHandler`: HTTP リクエストハンドラ
  - ルーティング結果に基づくディスパッチ
  - ステータスコード制御（404, 405, 500）
- `metrics_handler`: メトリクスエンドポイント

## 依存関係

```
Presentation → Application → Domain
                    ↓
            Infrastructure
```

- Presentation は Application に依存
- Application は Domain と Infrastructure に依存
- Infrastructure は Domain に依存
- Domain は外部に依存しない（独立）

## ルーティング機能の統合

### フロー

```
HTTP Request
    ↓
Presentation::HttpHandler
    ↓
Application::InvocationService::invoke()
    ↓
Application::FunctionService::resolve_function()
    ↓
Infrastructure::InMemoryRouteRepository::match_route()
    ↓
path_matches() + extract_path_params()
    ↓
RouteMatch { function_id, path_params }
    ↓
FunctionMetadata 取得
    ↓
WASM 実行
```

### ルーティング機能

1. **ホスト名マッチング**: 完全一致またはワイルドカード
2. **パスマッチング**: 完全一致、パラメータ、ワイルドカード
3. **メソッドマッチング**: 完全一致またはワイルドカード
4. **優先度制御**: priority フィールドで順序制御
5. **パスパラメータ抽出**: `:param` 形式で自動抽出

### テスト

- `test_exact_path_match`: 完全一致マッチング
- `test_path_parameter_extraction`: 単一パラメータ抽出
- `test_multiple_path_parameters`: 複数パラメータ抽出
- `test_prefix_wildcard_match`: プレフィックスワイルドカード
- `test_root_wildcard_match`: ルートワイルドカード
- `test_method_matching`: メソッドマッチング
- `test_wildcard_method`: ワイルドカードメソッド
- `test_host_matching`: ホスト名マッチング
- `test_priority_ordering`: 優先度順序
- `test_no_match`: マッチなし
- `test_list_routes`: ルート一覧取得

## 利点

1. **関心の分離**: 各レイヤーが明確な責務を持つ
2. **テスト容易性**: リポジトリをモック化可能、ルーティング単体テスト完備
3. **保守性**: 変更の影響が限定される
4. **拡張性**: 新しい実装を追加しやすい
5. **再利用性**: ドメインロジックが独立している
6. **ルーティング柔軟性**: パターンマッチング、パラメータ抽出、優先度制御
