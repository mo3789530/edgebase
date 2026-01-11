package timeseries

import (
	"context"
	"time"
)

// Collector implements MetricCollector
type Collector struct {
	store TimeSeriesStore
}

// NewCollector creates a new MetricCollector
func NewCollector(store TimeSeriesStore) MetricCollector {
	return &Collector{store: store}
}

// RecordExecutionStart records the start of a function execution
func (c *Collector) RecordExecutionStart(ctx context.Context, functionID string, executionID string) error {
	// We currently don't record a separate event for start.
	// We could record a "function_start" measurement if needed for concurrency tracking.
	return nil
}

// RecordExecutionEnd records the completion of a function execution
func (c *Collector) RecordExecutionEnd(ctx context.Context, functionID string, executionID string, duration time.Duration, status ExecutionStatus, errVal error) error {
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	tags := map[string]string{
		"function_id":  functionID,
		"execution_id": executionID,
		"status":       string(status),
	}

	fields := map[string]interface{}{
		"duration_ms": float64(duration.Milliseconds()), // InfluxDB works well with float64 for fields
		"start_time":  startTime.UnixNano(),
		"end_time":    endTime.UnixNano(),
	}

	if errVal != nil {
		fields["error"] = errVal.Error()
	}

	point := &MetricPoint{
		Timestamp:   endTime,
		Measurement: "function_execution",
		Tags:        tags,
		Fields:      fields,
	}

	return c.store.WritePoint(ctx, point)
}

// RecordResourceMetrics records resource usage metrics
func (c *Collector) RecordResourceMetrics(ctx context.Context, functionID string, executionID string, metrics *ResourceMetrics) error {
	tags := map[string]string{
		"function_id":  functionID,
		"execution_id": executionID,
	}

	fields := map[string]interface{}{
		"memory_usage_mb":   metrics.MemoryUsageMB,
		"cpu_time_ms":       metrics.CPUTimeMs,
		"disk_usage_mb":     metrics.DiskUsageMB,
		"network_bytes_in":  metrics.NetworkBytesIn,
		"network_bytes_out": metrics.NetworkBytesOut,
	}

	point := &MetricPoint{
		Timestamp:   time.Now(),
		Measurement: "resource_usage",
		Tags:        tags,
		Fields:      fields,
	}

	return c.store.WritePoint(ctx, point)
}
