package timeseries

import "context"

// TimeSeriesStore defines the contract for storing and querying metrics
type TimeSeriesStore interface {
	// WritePoint writes a single metric point
	WritePoint(ctx context.Context, point *MetricPoint) error

	// WriteBatch writes multiple metric points in a batch
	WriteBatch(ctx context.Context, points []*MetricPoint) error

	// Query queries metrics with filters
	Query(ctx context.Context, query *MetricsQuery) ([]*MetricPoint, error)

	// QueryAggregates queries aggregated statistics
	QueryAggregates(ctx context.Context, query *AggregateQuery) (*AggregateResult, error)

	// EnsureRetentionPolicy ensures the retention policy is set
	EnsureRetentionPolicy(ctx context.Context, retentionDays int) error

	// Close closes the connection
	Close() error
}
