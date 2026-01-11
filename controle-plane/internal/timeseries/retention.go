package timeseries

import (
	"context"
	"log"
)

// RetentionManager manages data retention policies
type RetentionManager struct {
	store         TimeSeriesStore
	retentionDays int
}

// NewRetentionManager creates a new RetentionManager
func NewRetentionManager(store TimeSeriesStore, retentionDays int) *RetentionManager {
	return &RetentionManager{
		store:         store,
		retentionDays: retentionDays,
	}
}

// ApplyPolicy applies the configured retention policy to the storage
func (rm *RetentionManager) ApplyPolicy(ctx context.Context) error {
	if rm.retentionDays <= 0 {
		return nil
	}
	log.Printf("Applying retention policy: %d days", rm.retentionDays)
	return rm.store.EnsureRetentionPolicy(ctx, rm.retentionDays)
}
