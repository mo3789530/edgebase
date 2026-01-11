package timeseries

import "context"

// LogWriter is responsible for writing structured logs to the time-series database
type LogWriter interface {
	// WriteExecutionLog writes an execution log
	WriteExecutionLog(ctx context.Context, log *ExecutionLog) error

	// QueryLogs queries execution logs
	QueryLogs(ctx context.Context, query *LogQuery) ([]*ExecutionLog, error)
}
