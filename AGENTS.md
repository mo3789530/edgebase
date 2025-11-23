# EdgeBase Agent Guidelines

## プロジェクト概要

EdgeBaseは分散エッジコンピューティングプラットフォームです。エッジノードでサーバーレス関数を実行し、データベース同期とコントロールプレーン管理を提供します。

### 主要機能
- **エッジノード管理**: 分散ノードの登録・ハートビート・同期
- **関数管理**: WASM関数のアップロード・デプロイ・実行
- **データベース同期**: スキーマ・データの分散同期
- **ルーティング**: エッジノード間のリクエストルーティング

## フォルダ構成

```
edgebase/
├── db/                          # データベースサービス (Rust)
│   ├── edge-agent/              # エッジノード用DBエージェント
│   └── migrations/              # DB マイグレーション
├── functions/                   # サーバーレス関数ランタイム (Rust)
│   ├── edge-runner/             # エッジ実行ランタイム
│   └── hello-world/             # サンプル関数
├── platform/                    # コントロールプレーン (Go)
│   └── control-plane/           # 集約されたコントロールプレーン
│       ├── cmd/server/          # エントリーポイント
│       ├── internal/
│       │   ├── handler/         # HTTPハンドラー (機能別分割)
│       │   ├── service/         # ビジネスロジック
│       │   ├── repository/      # データアクセス層
│       │   ├── model/           # データモデル
│       │   ├── storage/         # MinIO クライアント
│       │   ├── db/              # DB接続
│       │   ├── config/          # 設定管理
│       │   └── mqtt/            # MQTT クライアント
│       └── go.mod
├── docker-compose.yml           # ローカル開発環境
├── Makefile                     # ビルドスクリプト
└── README.md                    # プロジェクト説明
```

## コーディングルール

### 言語別ガイドライン

#### Go (platform/control-plane)
1. **パッケージ構成**
   - `cmd/`: メイン処理
   - `internal/`: 内部パッケージ (外部から非公開)
   - 各層は責務を明確に分離

2. **命名規則**
   - インターフェース: `XxxService`, `XxxRepository`
   - 実装体: `xxxService`, `xxxRepository`
   - メソッド: PascalCase (公開), camelCase (非公開)

3. **エラーハンドリング**
   ```go
   if err != nil {
       return nil, fmt.Errorf("operation failed: %w", err)
   }
   ```

4. **テスト**
   - ファイル名: `xxx_test.go`
   - テスト関数: `TestXxx`
   - モック: `Mock` プレフィックス
   - `testify/mock` と `testify/assert` を使用

#### Rust (db, functions)
1. **モジュール構成**
   - `src/main.rs`: エントリーポイント
   - `src/lib.rs`: ライブラリ
   - 機能ごとにモジュール分割

2. **命名規則**
   - 構造体: PascalCase
   - 関数: snake_case
   - 定数: SCREAMING_SNAKE_CASE

3. **エラーハンドリング**
   - `Result<T>` を使用
   - カスタムエラー型を定義

4. **テスト**
   - ファイル名: `#[cfg(test)]` モジュール内
   - テスト関数: `#[test]` 属性
   - `assert!`, `assert_eq!` を使用

### Rustプロジェクト詳細

#### db/ (データベースサービス)
**edge-agent/**
- エッジノードのローカルDB管理
- SQLite ベース
- スキーマ同期機能
- 主要ファイル:
  - `src/main.rs`: エージェント起動
  - `src/db.rs`: DB操作
  - `src/sync.rs`: 同期ロジック
  - `src/models.rs`: データモデル

#### functions/ (関数ランタイム)
**edge-runner/**
- WASM関数実行ランタイム
- エッジノード上で動作
- 関数のライフサイクル管理
- 主要ファイル:
  - `src/main.rs`: ランタイム起動
  - `src/domain/`: ドメインロジック
  - `src/application/`: ユースケース
  - `src/infrastructure/`: 外部連携

**hello-world/**
- サンプルWASM関数
- テスト用途

#### platform/control-plane/ (コントロールプレーン)
**ノード管理**
- エッジノード登録・ハートビート・同期

**関数管理**
- WASM関数のアップロード・デプロイ・実行

**テレメトリ・同期**
- デバイス登録・管理
- テレメトリデータ同期（Last-Write-Wins競合検出）
- コマンド管理・実行

**スキーマ管理**
- スキーマバージョン管理
- マイグレーション

### 共通ルール

1. **最小限の実装**
   - 必要な機能のみ実装
   - 冗長なコードは避ける
   - 未使用の変数・インポートは削除

2. **コメント**
   - 複雑なロジックのみコメント
   - 自明なコードにはコメント不要
   - 日本語コメント可

3. **ファイルサイズ**
   - 1ファイル 300行以下を目安
   - 責務が大きい場合は分割
   - handler層は機能別に分割

## 実装フロー

### 1. 要件確認
- 機能の目的を明確化
- 既存コードとの関係を確認
- 影響範囲を把握

### 2. 設計
- インターフェース定義
- データモデル設計
- エラーケース検討

### 3. 実装
- 最小限の実装
- 依存性注入を活用
- テストを念頭に

### 4. テスト実装 (必須)
- ユニットテスト作成
- モックを使用した隔離テスト
- エッジケースをカバー

### 5. ビルド確認 (必須)
```bash
# Go
cd platform/control-plane
go mod tidy
go build ./cmd/server
go vet ./...
go test ./...

# Rust - Database Services
cd db
cargo build
cargo test
cargo clippy

# Rust - Functions
cd functions
cargo build
cargo test
cargo clippy

# WASM Build (hello-world example)
cd functions/hello-world
cargo build --target wasm32-unknown-unknown --release
```

### 6. 検証
- 警告なしでビルド
- すべてのテスト合格
- 既存テストの破損なし

## テスト実装ガイド

### Go テスト例
```go
func TestCreateFunction(t *testing.T) {
    mockRepo := new(MockFunctionRepository)
    mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
    
    svc := NewArtifactService(mockRepo, nil)
    fn, err := svc.CreateFunction(context.Background(), "test", "main", "wasm", 256, 5000)
    
    assert.NoError(t, err)
    assert.NotNil(t, fn)
    mockRepo.AssertExpectations(t)
}
```

### Rust テスト例
```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_create_function() {
        let result = create_function("test_fn", "main", 256);
        assert!(result.is_ok());
        
        let fn_obj = result.unwrap();
        assert_eq!(fn_obj.name, "test_fn");
        assert_eq!(fn_obj.entrypoint, "main");
    }

    #[test]
    fn test_invalid_memory() {
        let result = create_function("test", "main", 0);
        assert!(result.is_err());
    }
}
```

### テストカバレッジ
- 正常系: 必須
- エラー系: 必須
- エッジケース: 推奨

## 統合ガイドライン

### 新機能追加時
1. インターフェース定義
2. モック実装
3. テスト実装
4. 実装
5. ビルド・テスト確認

### 既存機能修正時
1. 影響範囲確認
2. テスト追加/修正
3. 実装修正
4. ビルド・テスト確認
5. 既存テスト確認

## チェックリスト

実装完了時に確認:

- [ ] コード実装完了
- [ ] ユニットテスト実装
- [ ] ビルド成功 (`go build` / `cargo build`)
- [ ] 警告なし (`go vet` / `cargo clippy`)
- [ ] テスト合格 (`go test` / `cargo test`)
- [ ] 既存テスト破損なし
- [ ] コメント・ドキュメント更新
- [ ] 不要なコード削除

## トラブルシューティング

### ビルドエラー
1. `go mod tidy` で依存関係を更新
2. 未使用の変数・インポートを削除
3. インターフェース実装を確認

### テスト失敗
1. モック設定を確認
2. テストデータを確認
3. 依存関係の初期化を確認

### 警告
1. `go vet` の出力を確認
2. 型キャストを確認
3. エラーハンドリングを確認

## 参考資料

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Rust API Guidelines](https://rust-lang.github.io/api-guidelines/)
- [EdgeBase README](./README.md)
- [Consolidation Notes](./CONSOLIDATION.md)
