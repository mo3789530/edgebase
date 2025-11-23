# HTTP リクエストルーティング (Requirement 3)

## 概要

Edge Runnerは、HTTP リクエストを受け取り、登録されたルートに基づいて適切なWASM関数にディスパッチするルーティング機能を実装しています。

## ルーティング仕様

### ルートマッチング

ルートマッチングは以下の優先順位で行われます：

1. **ホスト名マッチング**
   - 完全一致: `example.com`
   - ワイルドカード: `*` (すべてのホストに対応)

2. **パスマッチング**
   - 完全一致: `/api/users`
   - パラメータ: `/api/users/:id` (`:id`は動的パラメータ)
   - ワイルドカード: `/api/*` (プレフィックスマッチ)

3. **HTTPメソッドマッチング**
   - 完全一致: `GET`, `POST`, `PUT`, `DELETE`
   - ワイルドカード: `*` (すべてのメソッドに対応)

4. **優先度制御**
   - `priority` フィールドで優先度を指定（高い値ほど優先）
   - 同じ優先度の場合は登録順

### ルート定義

```rust
pub struct Route {
    pub id: String,                    // ルートID
    pub host: String,                  // ホスト名 ("*" でワイルドカード)
    pub path: String,                  // パスパターン
    pub function_id: String,           // 関連するWASM関数ID
    pub methods: Vec<String>,          // HTTPメソッド
    pub priority: i32,                 // 優先度（高いほど優先）
}
```

## パスパターンマッチング

### パターン例

| パターン | マッチ例 | 説明 |
|---------|---------|------|
| `/api/users` | `/api/users` | 完全一致 |
| `/api/users/:id` | `/api/users/123` | パラメータ抽出 |
| `/api/*` | `/api/users`, `/api/posts` | プレフィックスマッチ |
| `/*` | すべてのパス | ルートワイルドカード |

### パスパラメータ抽出

パスパターンに `:name` 形式のパラメータを含めると、マッチ時に抽出されます：

```
パターン: /api/users/:id/posts/:post_id
パス: /api/users/123/posts/456

抽出結果:
{
  "id": "123",
  "post_id": "456"
}
```

## ルーティング処理フロー

```
HTTP Request
    ↓
[Host, Path, Method] 抽出
    ↓
RouteRepository.match_route()
    ↓
優先度順にルートをスキャン
    ↓
ホスト名マッチング確認
    ↓
パスパターンマッチング確認
    ↓
HTTPメソッドマッチング確認
    ↓
RouteMatch {
  function_id,
  path_params
}
    ↓
FunctionRepository から FunctionMetadata 取得
    ↓
WASM実行
```

## エラーハンドリング

### HTTP ステータスコード

| コード | 説明 | 原因 |
|-------|------|------|
| 200 | OK | 正常に実行完了 |
| 404 | Not Found | ルートが見つからない |
| 405 | Method Not Allowed | メソッドが許可されていない |
| 500 | Internal Server Error | WASM実行エラー |

### エラーメッセージ

```
"Route not found" → 404 Not Found
"method not allowed" → 405 Method Not Allowed
その他 → 500 Internal Server Error
```

## 実装詳細

### RouteRepository トレイト

```rust
#[async_trait]
pub trait RouteRepository: Send + Sync {
    async fn add_route(&self, route: Route);
    async fn match_route(&self, host: &str, path: &str, method: &str) -> Option<RouteMatch>;
    async fn list_routes(&self) -> Vec<Route>;
}
```

### RouteMatch 構造体

```rust
#[derive(Clone, Debug)]
pub struct RouteMatch {
    pub function_id: String,
    pub path_params: HashMap<String, String>,
}
```

### パスマッチング実装

```rust
fn path_matches(pattern: &str, path: &str) -> bool {
    // ワイルドカード処理
    if pattern == "*" || pattern == "/*" {
        return true;
    }
    
    // パターンとパスを "/" で分割
    let pattern_parts: Vec<&str> = pattern.split('/').filter(|p| !p.is_empty()).collect();
    let path_parts: Vec<&str> = path.split('/').filter(|p| !p.is_empty()).collect();
    
    // 部分数が異なる場合
    if pattern_parts.len() != path_parts.len() {
        // サフィックスワイルドカード処理
        if pattern.ends_with("*") && !pattern.ends_with("/*") {
            let prefix = &pattern[..pattern.len() - 1];
            return path.starts_with(prefix);
        }
        return false;
    }
    
    // 各部分をマッチング
    for (p_part, path_part) in pattern_parts.iter().zip(path_parts.iter()) {
        if p_part.starts_with(':') {
            // パラメータ部分はスキップ
            continue;
        }
        if p_part != path_part {
            return false;
        }
    }
    true
}
```

### パスパラメータ抽出実装

```rust
fn extract_path_params(pattern: &str, path: &str) -> HashMap<String, String> {
    let mut params = HashMap::new();
    let pattern_parts: Vec<&str> = pattern.split('/').filter(|p| !p.is_empty()).collect();
    let path_parts: Vec<&str> = path.split('/').filter(|p| !p.is_empty()).collect();
    
    for (p_part, path_part) in pattern_parts.iter().zip(path_parts.iter()) {
        if p_part.starts_with(':') {
            let key = &p_part[1..];
            params.insert(key.to_string(), path_part.to_string());
        }
    }
    params
}
```

## 使用例

### ルート登録

```bash
# Control Plane API でルート登録
curl -X POST http://localhost:8080/api/v1/routes \
  -H "Content-Type: application/json" \
  -d '{
    "host": "*",
    "path": "/api/users/:id",
    "function_id": "user-handler",
    "methods": ["GET", "POST"],
    "priority": 100
  }'
```

### リクエスト処理

```bash
# GET /api/users/123 → user-handler 関数が呼び出される
curl http://localhost:3000/api/users/123

# POST /api/users/456 → user-handler 関数が呼び出される
curl -X POST http://localhost:3000/api/users/456

# GET /api/posts → ルートが見つからない → 404
curl http://localhost:3000/api/posts
```

## パフォーマンス特性

- **ルートマッチング**: O(n) - ルート数に比例
- **パスパラメータ抽出**: O(m) - パスセグメント数に比例
- **メモリ使用量**: O(n) - ルート数に比例

## 今後の拡張

- 正規表現パターンマッチング
- キャッシュされたルートマッチング結果
- ルート統計情報の収集
- 動的ルート更新（ホットリロード）
