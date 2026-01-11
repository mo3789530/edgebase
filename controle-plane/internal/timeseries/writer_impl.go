package timeseries

import (
	"context"
	"time"
)

// Writer implements LogWriter
type Writer struct {
	store TimeSeriesStore
}

// NewWriter creates a new LogWriter
func NewWriter(store TimeSeriesStore) LogWriter {
	return &Writer{store: store}
}

// WriteExecutionLog writes an execution log
func (w *Writer) WriteExecutionLog(ctx context.Context, log *ExecutionLog) error {
	tags := map[string]string{
		"function_id":   log.FunctionID,
		"function_name": log.FunctionName,
		"status":        string(log.Status),
		"environment":   log.Environment,
	}

	fields := map[string]interface{}{
		"execution_id": log.ExecutionID,
		"user_id":      log.UserID,
		"request_id":   log.RequestID,
		"duration_ms":  float64(log.Duration.Milliseconds()),
		"error_msg":    log.ErrorMessage,
	}

	if log.ResourceMetrics != nil {
		fields["memory_usage_mb"] = log.ResourceMetrics.MemoryUsageMB
		fields["cpu_time_ms"] = log.ResourceMetrics.CPUTimeMs
	}

	point := &MetricPoint{
		Timestamp:   log.Timestamp,
		Measurement: "execution_log",
		Tags:        tags,
		Fields:      fields,
	}

	return w.store.WritePoint(ctx, point)
}

// QueryLogs queries execution logs
func (w *Writer) QueryLogs(ctx context.Context, query *LogQuery) ([]*ExecutionLog, error) {
	metricsQuery := &MetricsQuery{
		StartTime:   query.StartTime,
		EndTime:     query.EndTime,
		Status:      query.Status,
		Limit:       query.Limit,
		Measurement: "execution_log",
		Tags:        make(map[string]string),
	}

	if query.FunctionName != "" {
		metricsQuery.Tags["function_name"] = query.FunctionName
	}

	points, err := w.store.Query(ctx, metricsQuery)
	if err != nil {
		return nil, err
	}

	logs := make([]*ExecutionLog, 0, len(points))
	for _, p := range points {
		log := &ExecutionLog{
			Timestamp:       p.Timestamp,
			FunctionID:      p.Tags["function_id"],
			FunctionName:    p.Tags["function_name"],
			Status:          ExecutionStatus(p.Tags["status"]),
			Environment:     p.Tags["environment"],
			ResourceMetrics: &ResourceMetrics{},
		}

		if eid, ok := p.Fields["execution_id"].(string); ok {
			log.ExecutionID = eid
		}
		if uid, ok := p.Fields["user_id"].(string); ok {
			log.UserID = uid
		}
		if rid, ok := p.Fields["request_id"].(string); ok {
			log.RequestID = rid
		}
		if dur, ok := p.Fields["duration_ms"].(float64); ok {
			log.Duration = time.Duration(dur) * time.Millisecond
		}
		if msg, ok := p.Fields["error_msg"].(string); ok {
			log.ErrorMessage = msg
		}

		// Resources
		if mem, ok := p.Fields["memory_usage_mb"].(float64); ok {
			log.ResourceMetrics.MemoryUsageMB = mem
		}
		if cpu, ok := p.Fields["cpu_time_ms"].(float64); ok {
			log.ResourceMetrics.CPUTimeMs = cpu
		}

		logs = append(logs, log)
	}

	return logs, nil
}
