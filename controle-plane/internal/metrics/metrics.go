package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	totalRequests      int64
	totalErrors        int64
	totalLatencyMs     int64
	requestCount       int64
	activeConnections  int64
	mu                 sync.RWMutex
	endpointMetrics    map[string]*EndpointMetrics
}

type EndpointMetrics struct {
	Path            string
	Method          string
	RequestCount    int64
	ErrorCount      int64
	TotalLatencyMs  int64
	MinLatencyMs    int64
	MaxLatencyMs    int64
	LastRequestTime time.Time
}

var defaultMetrics = &Metrics{
	endpointMetrics: make(map[string]*EndpointMetrics),
}

func RecordRequest(method, path string, latencyMs int64, isError bool) {
	atomic.AddInt64(&defaultMetrics.totalRequests, 1)
	atomic.AddInt64(&defaultMetrics.totalLatencyMs, latencyMs)
	atomic.AddInt64(&defaultMetrics.requestCount, 1)

	if isError {
		atomic.AddInt64(&defaultMetrics.totalErrors, 1)
	}

	key := method + " " + path
	defaultMetrics.mu.Lock()
	defer defaultMetrics.mu.Unlock()

	if em, exists := defaultMetrics.endpointMetrics[key]; exists {
		atomic.AddInt64(&em.RequestCount, 1)
		atomic.AddInt64(&em.TotalLatencyMs, latencyMs)
		if latencyMs < em.MinLatencyMs || em.MinLatencyMs == 0 {
			em.MinLatencyMs = latencyMs
		}
		if latencyMs > em.MaxLatencyMs {
			em.MaxLatencyMs = latencyMs
		}
		if isError {
			atomic.AddInt64(&em.ErrorCount, 1)
		}
		em.LastRequestTime = time.Now()
	} else {
		defaultMetrics.endpointMetrics[key] = &EndpointMetrics{
			Path:            path,
			Method:          method,
			RequestCount:    1,
			ErrorCount:      func() int64 { if isError { return 1 } else { return 0 } }(),
			TotalLatencyMs:  latencyMs,
			MinLatencyMs:    latencyMs,
			MaxLatencyMs:    latencyMs,
			LastRequestTime: time.Now(),
		}
	}
}

func IncrementActiveConnections() {
	atomic.AddInt64(&defaultMetrics.activeConnections, 1)
}

func DecrementActiveConnections() {
	atomic.AddInt64(&defaultMetrics.activeConnections, -1)
}

func GetMetrics() map[string]interface{} {
	defaultMetrics.mu.RLock()
	defer defaultMetrics.mu.RUnlock()

	totalRequests := atomic.LoadInt64(&defaultMetrics.totalRequests)
	totalErrors := atomic.LoadInt64(&defaultMetrics.totalErrors)
	totalLatencyMs := atomic.LoadInt64(&defaultMetrics.totalLatencyMs)
	activeConnections := atomic.LoadInt64(&defaultMetrics.activeConnections)

	avgLatency := int64(0)
	if totalRequests > 0 {
		avgLatency = totalLatencyMs / totalRequests
	}

	endpoints := make([]map[string]interface{}, 0)
	for _, em := range defaultMetrics.endpointMetrics {
		avgEpLatency := int64(0)
		if em.RequestCount > 0 {
			avgEpLatency = em.TotalLatencyMs / em.RequestCount
		}

		endpoints = append(endpoints, map[string]interface{}{
			"path":              em.Path,
			"method":            em.Method,
			"request_count":     em.RequestCount,
			"error_count":       em.ErrorCount,
			"avg_latency_ms":    avgEpLatency,
			"min_latency_ms":    em.MinLatencyMs,
			"max_latency_ms":    em.MaxLatencyMs,
			"last_request_time": em.LastRequestTime,
		})
	}

	return map[string]interface{}{
		"total_requests":       totalRequests,
		"total_errors":         totalErrors,
		"avg_latency_ms":       avgLatency,
		"active_connections":   activeConnections,
		"error_rate":           float64(totalErrors) / float64(totalRequests),
		"endpoint_metrics":     endpoints,
	}
}

func Reset() {
	atomic.StoreInt64(&defaultMetrics.totalRequests, 0)
	atomic.StoreInt64(&defaultMetrics.totalErrors, 0)
	atomic.StoreInt64(&defaultMetrics.totalLatencyMs, 0)
	atomic.StoreInt64(&defaultMetrics.requestCount, 0)

	defaultMetrics.mu.Lock()
	defer defaultMetrics.mu.Unlock()
	defaultMetrics.endpointMetrics = make(map[string]*EndpointMetrics)
}
