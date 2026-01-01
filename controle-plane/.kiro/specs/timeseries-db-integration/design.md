# 設計ドキュメント：時系列データベース統合

## 概要

このドキュメントは、時系列データベース（InfluxDB）をコントロールプレーンシステムに統合し、関数実行メトリクスと運用ログをキャプチャ、保存、クエリするための設計を説明しています。システムはすべての関数実行のパフォーマンスデータを記録し、履歴分析と監視のためのクエリ機能を提供します。

## アーキテクチャ

時系列データベース統合は、レイヤード・アーキテクチャに従います：

```
┌─────────────────────────────────────────────────────────────┐
│                    アプリケーション層                         │
│  （関数ハンドラー、サービス、ミドルウェア）                   │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│              メトリクス収集層                                 │
│  （メトリクス収集器、ログフォーマッター）                     │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│           時系列ストレージ抽象化層                            │
│  （ストレージインターフェース、バッチマネージャー、リトライロジック）
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│         時系列データベース実装                                │
│  （InfluxDBクライアント、接続管理）                           │
└─────────────────────────────────────────────────────────────┘
```

## コンポーネントとインターフェース

### 1. 時系列ストレージインターフェース

メトリクスの保存とクエリのコントラクトを定義する抽象化層：

```go
type TimeSeriesStore interface {
    // 単一のメトリクスポイントを書き込む
    WritePoint(ctx context.Context, point *MetricPoint) error
    
    // 複数のメトリクスポイントをバッチで書き込む
    WriteBatch(ctx context.Context, points []*MetricPoint) error
    
    // フィルター付きでメトリクスをクエリする
    Query(ctx context.Context, query *MetricsQuery) ([]*MetricPoint, error)
    
    // 集計統計をクエリする
    QueryAggregates(ctx context.Context, query *AggregateQuery) (*AggregateResult, error)
    
    // 接続を閉じる
    Close() error
}
```

### 2. メトリクス収集器

関数呼び出しから実行メトリクスを収集する責務を持ちます：

```go
type MetricCollector interface {
    // 関数実行開始を記録
    RecordExecutionStart(ctx context.Context, functionID string, executionID string) error
    
    // 関数実行完了を記録
    RecordExecutionEnd(ctx context.Context, functionID string, executionID string, 
                       duration time.Duration, status ExecutionStatus, err error) error
    
    // リソースメトリクスを記録
    RecordResourceMetrics(ctx context.Context, functionID string, executionID string,
                         metrics *ResourceMetrics) error
}
```

### 3. ログライター

時系列データベースに構造化ログを書き込む責務を持ちます：

```go
type LogWriter interface {
    // 実行ログを書き込む
    WriteExecutionLog(ctx context.Context, log *ExecutionLog) error
    
    // 実行ログをクエリする
    QueryLogs(ctx context.Context, query *LogQuery) ([]*ExecutionLog, error)
}
```

### 4. バッチマネージャー

効率性のためにメトリクス書き込みのバッチ処理を管理します：

```go
type BatchManager interface {
    // メトリクスをバッチに追加
    Add(point *MetricPoint) error
    
    // バッチをストレージにフラッシュ
    Flush(ctx context.Context) error
    
    // 現在のバッチサイズを取得
    Size() int
}
```

### 5. リトライマネージャー

失敗した書き込みに対してエクスポーネンシャルバックオフでリトライロジックを処理します：

```go
type RetryManager interface {
    // リトライロジック付きで操作を実行
    Execute(ctx context.Context, operation func() error) error
}
```

## データモデル

### MetricPoint

時系列データベース内の単一の測定値を表します：

```go
type MetricPoint struct {
    Timestamp   time.Time
    Measurement string                 // 例："function_execution"
    Tags        map[string]string      // 例：{"function_id": "...", "status": "success"}
    Fields      map[string]interface{} // 例：{"duration_ms": 150, "memory_mb": 256}
}
```

### ExecutionLog

関数実行の構造化ログエントリを表します：

```go
type ExecutionLog struct {
    Timestamp       time.Time
    FunctionID      string
    ExecutionID     string
    FunctionName    string
    Status          ExecutionStatus
    Duration        time.Duration
    ErrorMessage    string
    UserID          string
    RequestID       string
    Environment     string
    ResourceMetrics *ResourceMetrics
}
```

### ResourceMetrics

関数実行中のリソース使用量を表します：

```go
type ResourceMetrics struct {
    MemoryUsageMB  float64
    CPUTimeMs      float64
    DiskUsageMB    float64
    NetworkBytesIn  int64
    NetworkBytesOut int64
}
```

### MetricsQuery

メトリクスのクエリを表します：

```go
type MetricsQuery struct {
    FunctionID  string
    StartTime   time.Time
    EndTime     time.Time
    Status      ExecutionStatus // オプションフィルター
    Limit       int
}
```

### AggregateQuery

集計統計のクエリを表します：

```go
type AggregateQuery struct {
    FunctionID    string
    StartTime     time.Time
    EndTime       time.Time
    Aggregation   string // "mean", "min", "max", "p50", "p95", "p99"
    Interval      time.Duration
}
```

### AggregateResult

集計統計を表します：

```go
type AggregateResult struct {
    Aggregation string
    Values      []AggregateValue
}

type AggregateValue struct {
    Timestamp time.Time
    Value     float64
}
```

## 正確性プロパティ

プロパティは、システムのすべての有効な実行を通じて真であるべき特性または動作です。本質的には、システムが何をすべきかについての形式的なステートメントです。プロパティは、人間が読める仕様と機械検証可能な正確性保証の間の橋渡しとなります。

### プロパティ 1：実行メトリクスの完全性
*すべての*関数実行について、3つのタイミングメトリクス（開始時刻、終了時刻、期間）が記録され、期間は終了時刻 - 開始時刻に等しくなるべきです。
**検証対象：要件 1.1**

### プロパティ 2：実行ステータスの記録
*すべての*関数実行について、特定の結果（成功、失敗、またはタイムアウト）に対して、記録されたステータスは実際の実行結果と一致するべきです。
**検証対象：要件 1.2**

### プロパティ 3：リソースメトリクスの存在
*すべての*関数実行について、リソースメトリクス（メモリ使用量とCPU時間）がデータベースに記録されるべきです。
**検証対象：要件 1.3**

### プロパティ 4：メトリクス永続化ラウンドトリップ
*すべての*記録されたメトリクスについて、データベースをクエリすると、同じメトリクスが同じタイムスタンプとフィールド値で返されるべきです。
**検証対象：要件 1.4**

### プロパティ 5：クエリ結果のタイムスタンプ順序
*すべての*クエリ結果セットについて、返されたすべてのデータポイントはタイムスタンプで昇順に並べられるべきです。
**検証対象：要件 2.1**

### プロパティ 6：関数IDフィルタリング
*すべての*関数IDでフィルタリングされたクエリについて、返されたすべてのメトリクスは指定された関数IDと一致するタグを持つべきです。
**検証対象：要件 2.2**

### プロパティ 7：時間範囲フィルタリング
*すべての*開始時刻と終了時刻のフィルターを持つクエリについて、返されたすべてのメトリクスは指定された範囲内（包括的）のタイムスタンプを持つべきです。
**検証対象：要件 2.3**

### プロパティ 8：集計統計の正確性
*すべての*メトリクスセットについて、集計統計（平均、最小、最大、パーセンタイル）は基礎となるデータから正しく計算されるべきです。
**検証対象：要件 2.4**

### プロパティ 9：ログエントリの完全性
*すべての*関数実行について、記録されたログエントリはすべての必須フィールド（タイムスタンプ、関数名、実行ステータス、エラー詳細（該当する場合））を含むべきです。
**検証対象：要件 3.1**

### プロパティ 10：ログコンテキスト情報
*すべての*ログエントリについて、コンテキスト情報（ユーザーID、リクエストID、環境）が存在し、空でないべきです。
**検証対象：要件 3.2**

### プロパティ 11：ログ永続化ラウンドトリップ
*すべての*書き込まれたログエントリについて、そのタグを使用してデータベースをクエリすると、同じログエントリが返されるべきです。
**検証対象：要件 3.3**

### プロパティ 12：複数基準によるログフィルタリング
*すべての*フィルター（関数名、ステータス、時間範囲）を持つログクエリについて、返されたすべてのログはすべての指定されたフィルター基準と一致するべきです。
**検証対象：要件 3.4**

### プロパティ 13：保持ポリシーの適用
*すべての*設定された保持ポリシーについて、保持期間より古いメトリクスはデータベースから自動的に削除されるべきです。
**検証対象：要件 4.2、4.3**

### プロパティ 14：保持ポリシー更新の保存
*すべての*保持ポリシー更新について、新しいポリシーに従って保持されるべきメトリクスは削除されないべきです。
**検証対象：要件 4.4**

### プロパティ 15：接続復旧力
*すべての*データベース接続失敗について、システムはエラーをログに記録し、関数実行をブロックせずに新しいメトリクスの受け入れを続けるべきです。
**検証対象：要件 5.2**

### プロパティ 16：エクスポーネンシャルバックオフ付きリトライロジック
*すべての*失敗した書き込み操作について、システムは成功するか最大リトライ回数に達するまでエクスポーネンシャルバックオフでリトライするべきです。
**検証対象：要件 5.3**

### プロパティ 17：グレースフルシャットダウンフラッシュ
*すべての*シャットダウン時の保留中メトリクスについて、接続を閉じる前にすべての保留中の書き込みをデータベースにフラッシュするべきです。
**検証対象：要件 5.4**

### プロパティ 18：バッチ効率
*すべての*メトリクスバッチについて、複数のメトリクスは個別ではなく単一のバッチ操作で一緒に書き込まれるべきです。
**検証対象：要件 6.3**

### プロパティ 19：バッチサイズまたはタイムアウト時のフラッシュ
*すべての*バッチマネージャーについて、バッチが最大サイズに達するか、タイムアウトが発生するかのいずれかの場合、バッチはフラッシュされるべきです。
**検証対象：要件 6.4**

## エラーハンドリング

システムは包括的なエラーハンドリングを実装します：

1. **接続エラー**：ログに記録されますが、関数実行をブロックしません。メトリクスはリトライのためにキューに入れられます。
2. **書き込み失敗**：エクスポーネンシャルバックオフでリトライされます（初期：100ms、最大：30秒、最大リトライ：5回）。
3. **クエリエラー**：呼び出し元に説明的なエラーメッセージとともに返されます。
4. **バッチタイムアウト**：部分的なバッチは設定可能なタイムアウト後にフラッシュされます（デフォルト：5秒）。
5. **設定エラー**：スタートアップ時にログに記録されます。システムはデフォルト値で続行します。

## テスト戦略

### ユニットテスト

ユニットテストは特定の例とエッジケースを検証します：

- 環境変数からの設定ロード
- メトリクスポイントの作成と検証
- ログエントリのフォーマット
- バッチマネージャーの動作（追加、フラッシュ、サイズ）
- 様々な失敗シナリオでのリトライロジック
- クエリ結果のフィルタリングとソート

### プロパティベーステスト

プロパティベーステストは`gopter`ライブラリ（Goのプロパティテストフレームワーク）を使用して普遍的なプロパティを検証します：

- **最小100回の反復**プロパティテストごと
- 各テストは要件参照でタグ付けされます
- テストは有効な入力空間に制約するスマートジェネレーターを使用します
- テストはランダム入力全体でプロパティが成立することを検証します

**プロパティベーステストフレームワーク**：gopter (https://github.com/leanovate/gopter)

**テスト注釈形式**：
```go
// **機能：timeseries-db-integration、プロパティ 1：実行メトリクスの完全性**
// **検証対象：要件 1.1**
```

### 統合テスト

統合テストはエンドツーエンド機能を検証します：

- メトリクス記録を伴う完全な関数実行
- 様々なフィルターを使用したクエリ実行
- 保持ポリシーの適用
- 接続失敗と復旧
- 保留中メトリクスを伴うグレースフルシャットダウン

## 設定

システムは環境変数を通じて設定されます：

```
TIMESERIES_DB_TYPE=influxdb              # データベースタイプ（influxdb、prometheus等）
TIMESERIES_DB_URL=http://localhost:8086  # データベースURL
TIMESERIES_DB_TOKEN=<token>              # 認証トークン
TIMESERIES_DB_ORG=<org>                  # 組織（InfluxDB v2+）
TIMESERIES_DB_BUCKET=metrics             # バケット/データベース名
TIMESERIES_BATCH_SIZE=100                # フラッシュ前のバッチサイズ
TIMESERIES_BATCH_TIMEOUT=5s              # バッチタイムアウト
TIMESERIES_RETENTION_DAYS=30             # デフォルト保持期間
TIMESERIES_ENABLED=true                  # 時系列収集の有効/無効
```

## 依存関係

- **InfluxDB Goクライアント**：github.com/influxdata/influxdb-client-go/v2
- **プロパティテスト**：github.com/leanovate/gopter
- **既存**：Fiber、GORM、PostgreSQL

