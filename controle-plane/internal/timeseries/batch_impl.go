package timeseries

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Manager implements BatchManager
type Manager struct {
	store        TimeSeriesStore
	batchSize    int
	batchTimeout time.Duration

	inputCh chan *MetricPoint
	buffer  []*MetricPoint
	
	closeCh chan struct{}
	wg      sync.WaitGroup
}

// NewBatchManager creates a new BatchManager
func NewBatchManager(store TimeSeriesStore, batchSize int, batchTimeout time.Duration) *Manager {
	bm := &Manager{
		store:        store,
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		inputCh:      make(chan *MetricPoint, 1000), // Buffer size 1000 to absorb bursts
		buffer:       make([]*MetricPoint, 0, batchSize),
		closeCh:      make(chan struct{}),
	}

	bm.wg.Add(1)
	go bm.processLoop()

	return bm
}

// Add adds a metric to the batch asynchronously
func (bm *Manager) Add(point *MetricPoint) error {
	select {
	case bm.inputCh <- point:
		return nil
	default:
		// Channel full, drop metric to avoid blocking
		log.Printf("Warning: Metrics buffer full, dropping point")
		return fmt.Errorf("metrics buffer full")
	}
}

// Flush currently does nothing in async mode as the loop handles it.
// Use Close to ensure all pending metrics are written.
func (bm *Manager) Flush(ctx context.Context) error {
	return nil
}

func (bm *Manager) processLoop() {
	defer bm.wg.Done()
	
	var tickerC <-chan time.Time
	if bm.batchTimeout > 0 {
		ticker := time.NewTicker(bm.batchTimeout)
		defer ticker.Stop()
		tickerC = ticker.C
	}

	for {
		select {
		case p := <-bm.inputCh:
			bm.buffer = append(bm.buffer, p)
			if len(bm.buffer) >= bm.batchSize {
				bm.flushBuffer(context.Background())
			}
		case <-tickerC:
			bm.flushBuffer(context.Background())
		case <-bm.closeCh:
			// Drain input channel
			// Note: We cannot simply range over inputCh because we need to know when to stop if it's not closed.
			// But here we are in shutdown, so we assume no more Adds are coming?
			// Add checks inputCh availability.
			// Ideally we should signal Add to stop.
			// But simple drain is:
		drainLoop:
			for {
				select {
				case p := <-bm.inputCh:
					bm.buffer = append(bm.buffer, p)
				default:
					break drainLoop
				}
			}
			bm.flushBuffer(context.Background())
			return
		}
	}
}

func (bm *Manager) flushBuffer(ctx context.Context) {
	if len(bm.buffer) == 0 {
		return
	}
	
	// WriteBatch handles the write (synchronously). 
	// In this loop, it blocks the loop. This is fine, as Add is async.
	err := bm.store.WriteBatch(ctx, bm.buffer)
	if err != nil {
		log.Printf("Error writing metrics batch: %v", err)
	}
	
	// Reset buffer
	// To reuse underlying array logic effectively or just make new slice
	bm.buffer = make([]*MetricPoint, 0, bm.batchSize)
}

// Size returns the approximate size (channel + buffer)
func (bm *Manager) Size() int {
	return len(bm.inputCh) + len(bm.buffer)
}

// Close gracefully shuts down the batch manager and flushes pending metrics
func (bm *Manager) Close(ctx context.Context) error {
	select {
	case <-bm.closeCh:
		// Already closed
	default:
		close(bm.closeCh)
		bm.wg.Wait()
	}
	return nil
}