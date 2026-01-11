package timeseries

import (
	"context"
	"log"
)

// ShutdownManager handles graceful shutdown of time-series components
type ShutdownManager struct {
	store    TimeSeriesStore
	batchMgr *Manager // Using concrete type Manager to access Close method not in BatchManager interface?
	// Ideally BatchManager interface should have Close, or we use concrete type.
	// In batch_impl.go we defined Close on *Manager.
}

// NewShutdownManager creates a new ShutdownManager
func NewShutdownManager(store TimeSeriesStore, batchMgr *Manager) *ShutdownManager {
	return &ShutdownManager{
		store:    store,
		batchMgr: batchMgr,
	}
}

// Shutdown performs graceful shutdown
func (sm *ShutdownManager) Shutdown(ctx context.Context) error {
	log.Println("Shutting down time-series system...")

	// 1. Close batch manager (flushes pending metrics)
	if sm.batchMgr != nil {
		if err := sm.batchMgr.Close(ctx); err != nil {
			log.Printf("Error closing batch manager: %v", err)
		}
	}

	// 2. Close store connection
	if sm.store != nil {
		if err := sm.store.Close(); err != nil {
			log.Printf("Error closing time-series store: %v", err)
		}
	}

	log.Println("Time-series system shutdown complete.")
	return nil
}
