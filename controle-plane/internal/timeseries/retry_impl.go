package timeseries

import (
	"context"
	"math"
	"time"
)

// Retryer implements RetryManager
type Retryer struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxAttempts     int
}

// NewRetryer creates a new Retryer with default settings
func NewRetryer() *Retryer {
	return &Retryer{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     30 * time.Second,
		MaxAttempts:     5,
	}
}

// Execute executes an operation with retry logic
func (r *Retryer) Execute(ctx context.Context, operation func() error) error {
	var err error
	for i := 0; i < r.MaxAttempts; i++ {
		err = operation()
		if err == nil {
			return nil
		}

		// If context is canceled, stop retrying
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Calculate backoff
		// Exponential: initial * 2^i
		backoff := float64(r.InitialInterval) * math.Pow(2, float64(i))
		if backoff > float64(r.MaxInterval) {
			backoff = float64(r.MaxInterval)
		}

		// Sleep with context support
		select {
		case <-time.After(time.Duration(backoff)):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}
