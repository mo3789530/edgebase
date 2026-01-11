package influxdb

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"

	"github.com/edgebase/platform/control-plane/internal/timeseries"
)

// Store implements timeseries.TimeSeriesStore for InfluxDB
type Store struct {
	client *Client
}

// NewStore creates a new InfluxDB store
func NewStore(client *Client) *Store {
	return &Store{
		client: client,
	}
}

// EnsureRetentionPolicy ensures the retention policy is set
func (s *Store) EnsureRetentionPolicy(ctx context.Context, retentionDays int) error {
	bucket, err := s.client.bucketsAPI.FindBucketByName(ctx, s.client.config.Bucket)
	if err != nil {
		return fmt.Errorf("failed to find bucket: %w", err)
	}

	if retentionDays <= 0 {
		// No retention policy or infinite
		return nil
	}

	seconds := int64((time.Duration(retentionDays) * 24 * time.Hour).Seconds())

	// Check if update is needed
	needsUpdate := true
	if len(bucket.RetentionRules) > 0 {
		if bucket.RetentionRules[0].EverySeconds == seconds {
			needsUpdate = false
		}
	}

	if needsUpdate {
		expire := domain.RetentionRuleTypeExpire
		rule := domain.RetentionRule{
			EverySeconds: seconds,
			Type:         &expire,
		}
		bucket.RetentionRules = []domain.RetentionRule{rule}

		_, err := s.client.bucketsAPI.UpdateBucket(ctx, bucket)
		if err != nil {
			return fmt.Errorf("failed to update bucket retention: %w", err)
		}
	}

	return nil
}

// WritePoint writes a single metric point
func (s *Store) WritePoint(ctx context.Context, point *timeseries.MetricPoint) error {
	p := influxdb2.NewPoint(
		point.Measurement,
		point.Tags,
		point.Fields,
		point.Timestamp,
	)
	return s.client.writeAPI.WritePoint(ctx, p)
}

// WriteBatch writes multiple metric points in a batch
func (s *Store) WriteBatch(ctx context.Context, points []*timeseries.MetricPoint) error {
	// WriteAPIBlocking writes point by point synchronously.
	// For true batching efficiency, we might want to use the asynchronous WriteAPI,
	// but the requirements say "Implement BatchManager" which implies we manage the batching logic on top.
	// So here we just iterate. If performance becomes an issue, we can optimize this part.
	for _, point := range points {
		p := influxdb2.NewPoint(
			point.Measurement,
			point.Tags,
			point.Fields,
			point.Timestamp,
		)
		if err := s.client.writeAPI.WritePoint(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// Query queries metrics with filters
func (s *Store) Query(ctx context.Context, query *timeseries.MetricsQuery) ([]*timeseries.MetricPoint, error) {
	flux := fmt.Sprintf(`from(bucket: "%s")`, s.client.config.Bucket)
	flux += fmt.Sprintf(` |> range(start: %s, stop: %s)`, query.StartTime.Format(time.RFC3339), query.EndTime.Format(time.RFC3339))

	measurement := "function_execution"
	if query.Measurement != "" {
		measurement = query.Measurement
	}
	flux += fmt.Sprintf(` |> filter(fn: (r) => r["_measurement"] == "%s")`, measurement)

	if query.FunctionID != "" {
		flux += fmt.Sprintf(` |> filter(fn: (r) => r["function_id"] == "%s")`, query.FunctionID)
	}
	if query.Status != "" {
		flux += fmt.Sprintf(` |> filter(fn: (r) => r["status"] == "%s")`, query.Status)
	}

	for k, v := range query.Tags {
		flux += fmt.Sprintf(` |> filter(fn: (r) => r["%s"] == "%s")`, k, v)
	}

	// Pivot to align fields in one row
	flux += ` |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")`

	if query.Limit > 0 {
		flux += fmt.Sprintf(` |> limit(n: %d)`, query.Limit)
	}

	result, err := s.client.queryAPI.Query(ctx, flux)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer result.Close()

	var points []*timeseries.MetricPoint

	for result.Next() {
		record := result.Record()
		point := &timeseries.MetricPoint{
			Timestamp:   record.Time(),
			Measurement: record.Measurement(),
			Tags:        make(map[string]string),
			Fields:      make(map[string]interface{}),
		}

		// Extract values
		// The record contains all columns.
		// We need to separate tags and fields.
		for key, value := range record.Values() {
			if key == "_time" || key == "_start" || key == "_stop" || key == "_measurement" || key == "result" || key == "table" {
				continue
			}
			
			// In Flux pivot, tags are preserved as columns. Fields are also columns.
			// It's hard to distinguish without schema knowledge, but typically tags are strings.
			// However, fields can also be strings.
			// In our model: function_id, status are tags. duration_ms, memory_mb are fields.
			// Ideally we know the schema.
			// For now, let's treat known tags as tags, others as fields.
			
			switch key {
			case "function_id", "execution_id", "status", "user_id", "request_id", "environment", "function_name":
				if strVal, ok := value.(string); ok {
					point.Tags[key] = strVal
				}
			default:
				point.Fields[key] = value
			}
		}
		points = append(points, point)
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("query result error: %w", result.Err())
	}

	return points, nil
}

// QueryAggregates queries aggregated statistics
func (s *Store) QueryAggregates(ctx context.Context, query *timeseries.AggregateQuery) (*timeseries.AggregateResult, error) {
	flux := fmt.Sprintf(`from(bucket: "%s")`, s.client.config.Bucket)
	flux += fmt.Sprintf(` |> range(start: %s, stop: %s)`, query.StartTime.Format(time.RFC3339), query.EndTime.Format(time.RFC3339))
	flux += ` |> filter(fn: (r) => r["_measurement"] == "function_execution")`
	
	if query.FunctionID != "" {
		flux += fmt.Sprintf(` |> filter(fn: (r) => r["function_id"] == "%s")`, query.FunctionID)
	}

	// For aggregation, we usually aggregate on a specific field, e.g., duration_ms
	// The interface implies generalized aggregation, but usually we care about specific metrics.
	// Let's assume we are aggregating "duration_ms" for now, or maybe the query should specify the field?
	// The `AggregateQuery` struct in design doc doesn't have a Field, but `MetricPoint` has `Fields`.
	// Assuming `duration_ms` as the primary metric for aggregation.
	
	flux += ` |> filter(fn: (r) => r["_field"] == "duration_ms")`

	// Window and Aggregate
	flux += fmt.Sprintf(` |> aggregateWindow(every: %s, fn: %s, createEmpty: false)`, query.Interval, query.Aggregation)

	result, err := s.client.queryAPI.Query(ctx, flux)
	if err != nil {
		return nil, fmt.Errorf("failed to execute aggregate query: %w", err)
	}
	defer result.Close()

	aggResult := &timeseries.AggregateResult{
		Aggregation: query.Aggregation,
		Values:      make([]timeseries.AggregateValue, 0),
	}

	for result.Next() {
		record := result.Record()
		val, ok := record.Value().(float64)
		if !ok {
			// Try to convert if it's int or other numeric
			if intVal, ok := record.Value().(int64); ok {
				val = float64(intVal)
			} else {
				continue // Skip non-numeric
			}
		}
		aggResult.Values = append(aggResult.Values, timeseries.AggregateValue{
			Timestamp: record.Time(),
			Value:     val,
		})
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("aggregate query result error: %w", result.Err())
	}

	return aggResult, nil
}

// Close closes the connection
func (s *Store) Close() error {
	s.client.Close()
	return nil
}