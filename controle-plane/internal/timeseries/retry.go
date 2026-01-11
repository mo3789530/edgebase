package timeseries

import "context"

// RetryManager handles retry logic with exponential backoff for failed writes
type RetryManager interface {
	// Execute executes an operation with retry logic
	Execute(ctx context.Context, operation func() error) error
}
