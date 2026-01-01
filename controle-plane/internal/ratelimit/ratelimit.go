package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
	mu       sync.RWMutex
}

func NewLimiter(requestsPerSecond int, window time.Duration) *Limiter {
	limiter := &Limiter{
		requests: make(map[string][]time.Time),
		limit:    requestsPerSecond,
		window:   window,
	}

	// Cleanup old entries every minute
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.cleanup()
		}
	}()

	return limiter
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// Get or create request list
	requests := l.requests[key]

	// Remove old requests outside the window
	var filtered []time.Time
	for _, t := range requests {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}

	// Check if limit exceeded
	if len(filtered) >= l.limit {
		l.requests[key] = filtered
		return false
	}

	// Add current request
	filtered = append(filtered, now)
	l.requests[key] = filtered
	return true
}

func (l *Limiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	for key, requests := range l.requests {
		var filtered []time.Time
		for _, t := range requests {
			if t.After(cutoff) {
				filtered = append(filtered, t)
			}
		}

		if len(filtered) == 0 {
			delete(l.requests, key)
		} else {
			l.requests[key] = filtered
		}
	}
}

func (l *Limiter) GetRemaining(key string) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	requests := l.requests[key]
	var count int
	for _, t := range requests {
		if t.After(cutoff) {
			count++
		}
	}

	remaining := l.limit - count
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}
