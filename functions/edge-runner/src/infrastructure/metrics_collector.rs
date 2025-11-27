use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;

pub struct MetricsCollector {
    total_invocations: Arc<AtomicU64>,
    total_errors: Arc<AtomicU64>,
    total_execution_time_ms: Arc<AtomicU64>,
    cache_hits: Arc<AtomicU64>,
    cache_misses: Arc<AtomicU64>,
}

impl MetricsCollector {
    pub fn new() -> Self {
        MetricsCollector {
            total_invocations: Arc::new(AtomicU64::new(0)),
            total_errors: Arc::new(AtomicU64::new(0)),
            total_execution_time_ms: Arc::new(AtomicU64::new(0)),
            cache_hits: Arc::new(AtomicU64::new(0)),
            cache_misses: Arc::new(AtomicU64::new(0)),
        }
    }

    pub fn record_invocation(&self, execution_time_ms: u64, success: bool) {
        self.total_invocations.fetch_add(1, Ordering::Relaxed);
        self.total_execution_time_ms.fetch_add(execution_time_ms, Ordering::Relaxed);
        
        if !success {
            self.total_errors.fetch_add(1, Ordering::Relaxed);
        }
    }

    pub fn record_cache_hit(&self) {
        self.cache_hits.fetch_add(1, Ordering::Relaxed);
    }

    pub fn record_cache_miss(&self) {
        self.cache_misses.fetch_add(1, Ordering::Relaxed);
    }

    pub fn get_total_invocations(&self) -> u64 {
        self.total_invocations.load(Ordering::Relaxed)
    }

    pub fn get_total_errors(&self) -> u64 {
        self.total_errors.load(Ordering::Relaxed)
    }

    pub fn get_total_execution_time_ms(&self) -> u64 {
        self.total_execution_time_ms.load(Ordering::Relaxed)
    }

    pub fn get_cache_hits(&self) -> u64 {
        self.cache_hits.load(Ordering::Relaxed)
    }

    pub fn get_cache_misses(&self) -> u64 {
        self.cache_misses.load(Ordering::Relaxed)
    }

    pub fn get_average_execution_time_ms(&self) -> f64 {
        let total_invocations = self.get_total_invocations();
        if total_invocations == 0 {
            return 0.0;
        }
        
        let total_time = self.get_total_execution_time_ms();
        total_time as f64 / total_invocations as f64
    }

    pub fn get_error_rate(&self) -> f64 {
        let total_invocations = self.get_total_invocations();
        if total_invocations == 0 {
            return 0.0;
        }
        
        let total_errors = self.get_total_errors();
        (total_errors as f64 / total_invocations as f64) * 100.0
    }

    pub fn get_cache_hit_rate(&self) -> f64 {
        let hits = self.get_cache_hits();
        let misses = self.get_cache_misses();
        let total = hits + misses;
        
        if total == 0 {
            return 0.0;
        }
        
        (hits as f64 / total as f64) * 100.0
    }

    pub fn to_prometheus_format(&self) -> String {
        format!(
            "# HELP edge_runner_total_invocations Total number of function invocations\n\
             # TYPE edge_runner_total_invocations counter\n\
             edge_runner_total_invocations {}\n\
             # HELP edge_runner_total_errors Total number of errors\n\
             # TYPE edge_runner_total_errors counter\n\
             edge_runner_total_errors {}\n\
             # HELP edge_runner_average_execution_time_ms Average execution time in milliseconds\n\
             # TYPE edge_runner_average_execution_time_ms gauge\n\
             edge_runner_average_execution_time_ms {:.2}\n\
             # HELP edge_runner_error_rate Error rate percentage\n\
             # TYPE edge_runner_error_rate gauge\n\
             edge_runner_error_rate {:.2}\n\
             # HELP edge_runner_cache_hits Total cache hits\n\
             # TYPE edge_runner_cache_hits counter\n\
             edge_runner_cache_hits {}\n\
             # HELP edge_runner_cache_misses Total cache misses\n\
             # TYPE edge_runner_cache_misses counter\n\
             edge_runner_cache_misses {}\n\
             # HELP edge_runner_cache_hit_rate Cache hit rate percentage\n\
             # TYPE edge_runner_cache_hit_rate gauge\n\
             edge_runner_cache_hit_rate {:.2}\n",
            self.get_total_invocations(),
            self.get_total_errors(),
            self.get_average_execution_time_ms(),
            self.get_error_rate(),
            self.get_cache_hits(),
            self.get_cache_misses(),
            self.get_cache_hit_rate(),
        )
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_metrics_collector_creation() {
        let collector = MetricsCollector::new();
        assert_eq!(collector.get_total_invocations(), 0);
        assert_eq!(collector.get_total_errors(), 0);
    }

    #[test]
    fn test_record_invocation_success() {
        let collector = MetricsCollector::new();
        collector.record_invocation(100, true);
        
        assert_eq!(collector.get_total_invocations(), 1);
        assert_eq!(collector.get_total_errors(), 0);
        assert_eq!(collector.get_total_execution_time_ms(), 100);
    }

    #[test]
    fn test_record_invocation_error() {
        let collector = MetricsCollector::new();
        collector.record_invocation(100, false);
        
        assert_eq!(collector.get_total_invocations(), 1);
        assert_eq!(collector.get_total_errors(), 1);
    }

    #[test]
    fn test_cache_metrics() {
        let collector = MetricsCollector::new();
        collector.record_cache_hit();
        collector.record_cache_hit();
        collector.record_cache_miss();
        
        assert_eq!(collector.get_cache_hits(), 2);
        assert_eq!(collector.get_cache_misses(), 1);
    }

    #[test]
    fn test_average_execution_time() {
        let collector = MetricsCollector::new();
        collector.record_invocation(100, true);
        collector.record_invocation(200, true);
        
        let avg = collector.get_average_execution_time_ms();
        assert!((avg - 150.0).abs() < 0.01);
    }

    #[test]
    fn test_error_rate() {
        let collector = MetricsCollector::new();
        collector.record_invocation(100, true);
        collector.record_invocation(100, true);
        collector.record_invocation(100, false);
        
        let error_rate = collector.get_error_rate();
        assert!((error_rate - 33.33).abs() < 0.1);
    }

    #[test]
    fn test_cache_hit_rate() {
        let collector = MetricsCollector::new();
        collector.record_cache_hit();
        collector.record_cache_hit();
        collector.record_cache_miss();
        
        let hit_rate = collector.get_cache_hit_rate();
        assert!((hit_rate - 66.67).abs() < 0.1);
    }

    #[test]
    fn test_prometheus_format() {
        let collector = MetricsCollector::new();
        collector.record_invocation(100, true);
        collector.record_cache_hit();
        
        let prometheus = collector.to_prometheus_format();
        assert!(prometheus.contains("edge_runner_total_invocations 1"));
        assert!(prometheus.contains("edge_runner_cache_hits 1"));
    }
}
