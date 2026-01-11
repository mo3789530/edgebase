package timeseries

import "time"

// ExecutionStatus represents the status of a function execution
type ExecutionStatus string

const (
	StatusSuccess ExecutionStatus = "success"
	StatusFailure ExecutionStatus = "failure"
	StatusTimeout ExecutionStatus = "timeout"
)

// MetricPoint represents a single measurement in the time-series database
type MetricPoint struct {
	Timestamp   time.Time
	Measurement string                 // e.g., "function_execution"
	Tags        map[string]string      // e.g., {"function_id": "...", "status": "success"}
	Fields      map[string]interface{} // e.g., {"duration_ms": 150, "memory_mb": 256}
}

// ResourceMetrics represents resource usage during function execution
type ResourceMetrics struct {
	MemoryUsageMB   float64
	CPUTimeMs       float64
	DiskUsageMB     float64
	NetworkBytesIn  int64
	NetworkBytesOut int64
}

// ExecutionLog represents a structured log entry for a function execution
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

// MetricsQuery represents a query for metrics
type MetricsQuery struct {
	FunctionID  string
	Measurement string            // Optional: override default measurement
	Tags        map[string]string // Optional: generic tag filtering
	StartTime   time.Time
	EndTime     time.Time
	Status      ExecutionStatus // Optional filter
	Limit       int
}

// LogQuery represents a query for execution logs
type LogQuery struct {
	FunctionName string
	StartTime    time.Time
	EndTime      time.Time
	Status       ExecutionStatus
	Limit        int
}

// AggregateQuery represents a query for aggregated statistics
type AggregateQuery struct {
	FunctionID  string
	StartTime   time.Time
	EndTime     time.Time
	Aggregation string // "mean", "min", "max", "p50", "p95", "p99"
	Interval    time.Duration
}

// AggregateResult represents aggregated statistics
type AggregateResult struct {
	Aggregation string
	Values      []AggregateValue
}

// AggregateValue represents a single aggregated value
type AggregateValue struct {
	Timestamp time.Time
	Value     float64
}
