package timeseries

import (
	"context"
	"time"
)

// MetricCollector is responsible for collecting execution metrics from function calls
type MetricCollector interface {
	// RecordExecutionStart records the start of a function execution
	RecordExecutionStart(ctx context.Context, functionID string, executionID string) error

	// RecordExecutionEnd records the completion of a function execution
	RecordExecutionEnd(ctx context.Context, functionID string, executionID string,
		duration time.Duration, status ExecutionStatus, err error) error

	// RecordResourceMetrics records resource usage metrics
	RecordResourceMetrics(ctx context.Context, functionID string, executionID string,
		metrics *ResourceMetrics) error
}
