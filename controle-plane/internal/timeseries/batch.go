package timeseries

import "context"

// BatchManager manages batching of metrics writes for efficiency
type BatchManager interface {
	// Add adds a metric to the batch
	Add(point *MetricPoint) error

	// Flush flushes the batch to storage
	Flush(ctx context.Context) error

	// Size returns the current batch size
	Size() int
}
