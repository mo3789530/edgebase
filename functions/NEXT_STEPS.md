# 次のステップ - Requirement 4 以降の実装計画

## 完了した実装

✅ **Requirement 1**: WASM関数サンプル実装 (hello-world)
✅ **Requirement 2**: Edge Runner基本実装 (Wasmerランタイム、メトリクス、キャッシュ)
✅ **Requirement 3**: HTTP リクエストルーティング (完全実装、11テスト成功、警告ゼロ)

## 次の実装順序

### Requirement 4: リクエスト/レスポンス処理の拡張

**目的**: HTTP リクエストボディ、ヘッダー、クエリパラメータの処理

**実装内容**:
1. リクエストボディの読み込みと WASM メモリへの書き込み
2. レスポンスボディの WASM メモリからの読み込み
3. HTTP ヘッダーの処理（リクエスト/レスポンス）
4. クエリパラメータの抽出と処理
5. Content-Type の自動判定

**ファイル変更**:
- `edge-runner/src/application/services.rs` - InvocationService拡張
- `edge-runner/src/presentation/handlers.rs` - ヘッダー処理
- `hello-world/src/lib.rs` - WASM関数の拡張

**テスト**:
- POST リクエストボディ処理
- ヘッダー伝播
- クエリパラメータ抽出

---

### Requirement 5: エラーハンドリングと復旧

**目的**: WASM実行エラー、タイムアウト、リソース制限の処理

**実装内容**:
1. WASM実行タイムアウト制御
2. メモリ制限の強制
3. パニック/エラーハンドリング
4. リトライロジック
5. エラーログ記録

**ファイル変更**:
- `edge-runner/src/application/services.rs` - タイムアウト処理
- `edge-runner/src/infrastructure/pool.rs` - メモリ制限
- `edge-runner/src/presentation/handlers.rs` - エラーレスポンス

**テスト**:
- タイムアウト検出
- メモリ超過検出
- エラーレスポンス

---

### Requirement 6: ホットインスタンス管理の最適化

**目的**: インスタンスプールの効率化と LRU キャッシュ

**実装内容**:
1. LRU キャッシュ戦略の実装
2. インスタンス再利用率の向上
3. メモリ効率の最適化
4. アイドルインスタンスの自動削除

**ファイル変更**:
- `edge-runner/src/infrastructure/pool.rs` - LRU実装
- `edge-runner/src/infrastructure/cache.rs` - キャッシュ戦略

**テスト**:
- インスタンス再利用
- LRU削除
- メモリ効率

---

### Requirement 7: Control Plane 統合の強化

**目的**: ハートビート、デプロイ通知、ノード管理

**実装内容**:
1. ハートビート通信の改善
2. デプロイ通知の処理
3. ノード登録/削除
4. ステータスレポート

**ファイル変更**:
- `edge-runner/src/infrastructure/cp_client.rs` - CP通信
- `edge-runner/src/application/services.rs` - ハートビート処理

**テスト**:
- ハートビート送信
- デプロイ通知受信
- ノード登録

---

### Requirement 8: メトリクスと監視の拡張

**目的**: Prometheus メトリクスの充実

**実装内容**:
1. 関数別メトリクス
2. ルーティングメトリクス
3. キャッシュヒット率
4. インスタンスプール統計

**ファイル変更**:
- `edge-runner/src/infrastructure/metrics.rs` - メトリクス追加

**テスト**:
- メトリクス出力
- 値の正確性

---

### Requirement 9: セキュリティ強化

**目的**: WASM サンドボックス、WASI 制御、認証

**実装内容**:
1. WASI capability 制限
2. メモリアクセス制御
3. リソースアクセス制限
4. 認証/認可

**ファイル変更**:
- `edge-runner/src/infrastructure/pool.rs` - WASI制御
- `edge-runner/src/presentation/handlers.rs` - 認証

**テスト**:
- WASI制限
- リソースアクセス

---

### Requirement 10: パフォーマンス最適化

**目的**: レイテンシ削減、スループット向上

**実装内容**:
1. ルーティングキャッシュ
2. メモリプール最適化
3. 非同期処理の改善
4. バッチ処理

**ファイル変更**:
- `edge-runner/src/infrastructure/repositories.rs` - ルーティングキャッシュ
- `edge-runner/src/application/services.rs` - 非同期最適化

**テスト**:
- レイテンシ測定
- スループット測定

---

## 実装優先度

1. **高優先度** (次々週)
   - Requirement 4: リクエスト/レスポンス処理
   - Requirement 5: エラーハンドリング

2. **中優先度** (翌月)
   - Requirement 6: ホットインスタンス最適化
   - Requirement 7: CP統合強化

3. **低優先度** (その後)
   - Requirement 8: メトリクス拡張
   - Requirement 9: セキュリティ
   - Requirement 10: パフォーマンス最適化

## 現在の状態

**ビルド**: ✅ 成功（警告ゼロ）
**テスト**: ✅ 11/11 成功
**コード品質**: ✅ レイヤードアーキテクチャ、完全なドキュメント

## 推奨される次のアクション

```bash
# Requirement 4 の実装開始
# 1. リクエストボディ処理の実装
# 2. レスポンスボディ処理の実装
# 3. ヘッダー処理の実装
# 4. テストの追加
```
