package timeseries

import "context"

// BufferedStore wraps a TimeSeriesStore and uses a BatchManager for writes
type BufferedStore struct {
	store    TimeSeriesStore
	batchMgr BatchManager
}

// NewBufferedStore creates a new BufferedStore
func NewBufferedStore(store TimeSeriesStore, batchMgr BatchManager) *BufferedStore {
	return &BufferedStore{
		store:    store,
		batchMgr: batchMgr,
	}
}

func (s *BufferedStore) WritePoint(ctx context.Context, point *MetricPoint) error {
	return s.batchMgr.Add(point)
}

func (s *BufferedStore) WriteBatch(ctx context.Context, points []*MetricPoint) error {
	for _, p := range points {
		if err := s.batchMgr.Add(p); err != nil {
			return err
		}
	}
	return nil
}

func (s *BufferedStore) Query(ctx context.Context, query *MetricsQuery) ([]*MetricPoint, error) {
	return s.store.Query(ctx, query)
}

func (s *BufferedStore) QueryAggregates(ctx context.Context, query *AggregateQuery) (*AggregateResult, error) {
	return s.store.QueryAggregates(ctx, query)
}

func (s *BufferedStore) EnsureRetentionPolicy(ctx context.Context, retentionDays int) error {
	return s.store.EnsureRetentionPolicy(ctx, retentionDays)
}

func (s *BufferedStore) Close() error {
	return s.store.Close()
}
