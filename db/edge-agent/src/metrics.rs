use chrono::{DateTime, Utc};
use std::sync::{Arc, Mutex};
use std::time::Duration;

#[derive(Debug, Clone)]
pub struct SyncMetrics {
    pub total_synced: u64,
    pub total_failed: u64,
    pub pending_records: u64,
    pub last_sync_time: Option<DateTime<Utc>>,
    pub avg_sync_latency_ms: f64,
    pub error_rate: f64,
}

impl Default for SyncMetrics {
    fn default() -> Self {
        Self {
            total_synced: 0,
            total_failed: 0,
            pending_records: 0,
            last_sync_time: None,
            avg_sync_latency_ms: 0.0,
            error_rate: 0.0,
        }
    }
}

#[derive(Debug, Clone)]
pub struct Alert {
    pub id: String,
    pub alert_type: AlertType,
    pub message: String,
    pub timestamp: DateTime<Utc>,
    pub severity: AlertSeverity,
}

#[derive(Debug, Clone, PartialEq)]
pub enum AlertType {
    HighErrorRate,
    SyncFailure,
    HighLatency,
    PendingRecordsThreshold,
}

#[derive(Debug, Clone, PartialEq)]
pub enum AlertSeverity {
    Info,
    Warning,
    Critical,
}

pub struct MetricsCollector {
    metrics: Arc<Mutex<SyncMetrics>>,
    alerts: Arc<Mutex<Vec<Alert>>>,
    sync_times: Arc<Mutex<Vec<Duration>>>,
}

impl MetricsCollector {
    pub fn new() -> Self {
        Self {
            metrics: Arc::new(Mutex::new(SyncMetrics::default())),
            alerts: Arc::new(Mutex::new(Vec::new())),
            sync_times: Arc::new(Mutex::new(Vec::new())),
        }
    }

    pub fn record_sync_success(&self, count: u64, latency: Duration) {
        let mut metrics = self.metrics.lock().unwrap();
        metrics.total_synced += count;
        metrics.last_sync_time = Some(Utc::now());
        
        let mut times = self.sync_times.lock().unwrap();
        times.push(latency);
        if times.len() > 100 {
            times.remove(0);
        }
        
        let avg_ms = times.iter().map(|d| d.as_millis() as f64).sum::<f64>() / times.len() as f64;
        metrics.avg_sync_latency_ms = avg_ms;
        drop(times);
        
        let total = metrics.total_synced + metrics.total_failed;
        if total > 0 {
            metrics.error_rate = (metrics.total_failed as f64 / total as f64) * 100.0;
        }
    }

    pub fn record_sync_failure(&self, count: u64) {
        let mut metrics = self.metrics.lock().unwrap();
        metrics.total_failed += count;
        
        let total = metrics.total_synced + metrics.total_failed;
        if total > 0 {
            metrics.error_rate = (metrics.total_failed as f64 / total as f64) * 100.0;
        }
    }

    pub fn update_pending_records(&self, count: u64) {
        let mut metrics = self.metrics.lock().unwrap();
        metrics.pending_records = count;
    }

    pub fn get_metrics(&self) -> SyncMetrics {
        self.metrics.lock().unwrap().clone()
    }

    pub fn add_alert(&self, alert_type: AlertType, message: String, severity: AlertSeverity) {
        let alert = Alert {
            id: uuid::Uuid::new_v4().to_string(),
            alert_type,
            message,
            timestamp: Utc::now(),
            severity,
        };
        
        let mut alerts = self.alerts.lock().unwrap();
        alerts.push(alert);
        if alerts.len() > 1000 {
            alerts.remove(0);
        }
    }

    pub fn get_alerts(&self, limit: usize) -> Vec<Alert> {
        let alerts = self.alerts.lock().unwrap();
        alerts.iter().rev().take(limit).cloned().collect()
    }

    pub fn check_error_rate_threshold(&self, threshold: f64) -> bool {
        let metrics = self.metrics.lock().unwrap();
        metrics.error_rate > threshold
    }

    pub fn check_pending_threshold(&self, threshold: u64) -> bool {
        let metrics = self.metrics.lock().unwrap();
        metrics.pending_records > threshold
    }

    pub fn check_latency_threshold(&self, threshold_ms: f64) -> bool {
        let metrics = self.metrics.lock().unwrap();
        metrics.avg_sync_latency_ms > threshold_ms
    }

    pub fn clear_alerts(&self) {
        let mut alerts = self.alerts.lock().unwrap();
        alerts.clear();
    }
}

impl Clone for MetricsCollector {
    fn clone(&self) -> Self {
        Self {
            metrics: Arc::clone(&self.metrics),
            alerts: Arc::clone(&self.alerts),
            sync_times: Arc::clone(&self.sync_times),
        }
    }
}

impl Default for MetricsCollector {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_metrics_creation() {
        let collector = MetricsCollector::new();
        let metrics = collector.get_metrics();
        
        assert_eq!(metrics.total_synced, 0);
        assert_eq!(metrics.total_failed, 0);
        assert_eq!(metrics.pending_records, 0);
        assert_eq!(metrics.error_rate, 0.0);
    }

    #[test]
    fn test_record_sync_success() {
        let collector = MetricsCollector::new();
        collector.record_sync_success(10, Duration::from_millis(100));
        
        let metrics = collector.get_metrics();
        assert_eq!(metrics.total_synced, 10);
        assert!(metrics.last_sync_time.is_some());
        assert!(metrics.avg_sync_latency_ms > 0.0);
    }

    #[test]
    fn test_record_sync_failure() {
        let collector = MetricsCollector::new();
        collector.record_sync_failure(5);
        
        let metrics = collector.get_metrics();
        assert_eq!(metrics.total_failed, 5);
    }

    #[test]
    fn test_error_rate_calculation() {
        let collector = MetricsCollector::new();
        collector.record_sync_success(100, Duration::from_millis(50));
        collector.record_sync_failure(25);
        
        let metrics = collector.get_metrics();
        assert_eq!(metrics.error_rate, 20.0);
    }

    #[test]
    fn test_update_pending_records() {
        let collector = MetricsCollector::new();
        collector.update_pending_records(50);
        
        let metrics = collector.get_metrics();
        assert_eq!(metrics.pending_records, 50);
    }

    #[test]
    fn test_add_alert() {
        let collector = MetricsCollector::new();
        collector.add_alert(
            AlertType::SyncFailure,
            "Sync failed".to_string(),
            AlertSeverity::Critical,
        );
        
        let alerts = collector.get_alerts(10);
        assert_eq!(alerts.len(), 1);
        assert_eq!(alerts[0].alert_type, AlertType::SyncFailure);
        assert_eq!(alerts[0].severity, AlertSeverity::Critical);
    }

    #[test]
    fn test_get_alerts_limit() {
        let collector = MetricsCollector::new();
        for i in 0..20 {
            collector.add_alert(
                AlertType::HighErrorRate,
                format!("Alert {}", i),
                AlertSeverity::Warning,
            );
        }
        
        let alerts = collector.get_alerts(5);
        assert_eq!(alerts.len(), 5);
    }

    #[test]
    fn test_check_error_rate_threshold() {
        let collector = MetricsCollector::new();
        collector.record_sync_success(100, Duration::from_millis(50));
        collector.record_sync_failure(30);
        
        assert!(collector.check_error_rate_threshold(20.0));
        assert!(!collector.check_error_rate_threshold(40.0));
    }

    #[test]
    fn test_check_pending_threshold() {
        let collector = MetricsCollector::new();
        collector.update_pending_records(100);
        
        assert!(collector.check_pending_threshold(50));
        assert!(!collector.check_pending_threshold(150));
    }

    #[test]
    fn test_check_latency_threshold() {
        let collector = MetricsCollector::new();
        collector.record_sync_success(10, Duration::from_millis(200));
        
        assert!(collector.check_latency_threshold(100.0));
        assert!(!collector.check_latency_threshold(300.0));
    }

    #[test]
    fn test_clear_alerts() {
        let collector = MetricsCollector::new();
        collector.add_alert(AlertType::SyncFailure, "Test".to_string(), AlertSeverity::Critical);
        
        assert_eq!(collector.get_alerts(10).len(), 1);
        
        collector.clear_alerts();
        assert_eq!(collector.get_alerts(10).len(), 0);
    }

    #[test]
    fn test_multiple_sync_operations() {
        let collector = MetricsCollector::new();
        
        for _ in 0..5 {
            collector.record_sync_success(20, Duration::from_millis(100));
        }
        
        let metrics = collector.get_metrics();
        assert_eq!(metrics.total_synced, 100);
    }

    #[test]
    fn test_alert_timestamp() {
        let collector = MetricsCollector::new();
        let before = Utc::now();
        
        collector.add_alert(AlertType::SyncFailure, "Test".to_string(), AlertSeverity::Critical);
        
        let after = Utc::now();
        let alerts = collector.get_alerts(1);
        
        assert!(alerts[0].timestamp >= before);
        assert!(alerts[0].timestamp <= after);
    }

    #[test]
    fn test_alert_id_uniqueness() {
        let collector = MetricsCollector::new();
        
        collector.add_alert(AlertType::SyncFailure, "Alert 1".to_string(), AlertSeverity::Critical);
        collector.add_alert(AlertType::SyncFailure, "Alert 2".to_string(), AlertSeverity::Critical);
        
        let alerts = collector.get_alerts(10);
        assert_ne!(alerts[0].id, alerts[1].id);
    }

    #[test]
    fn test_metrics_clone() {
        let collector = MetricsCollector::new();
        collector.record_sync_success(10, Duration::from_millis(100));
        
        let collector2 = collector.clone();
        let metrics = collector2.get_metrics();
        
        assert_eq!(metrics.total_synced, 10);
    }

    #[test]
    fn test_average_latency_calculation() {
        let collector = MetricsCollector::new();
        
        collector.record_sync_success(5, Duration::from_millis(100));
        collector.record_sync_success(5, Duration::from_millis(200));
        
        let metrics = collector.get_metrics();
        assert_eq!(metrics.avg_sync_latency_ms, 150.0);
    }

    #[test]
    fn test_high_error_rate_alert() {
        let collector = MetricsCollector::new();
        collector.record_sync_success(10, Duration::from_millis(50));
        collector.record_sync_failure(90);
        
        if collector.check_error_rate_threshold(50.0) {
            collector.add_alert(
                AlertType::HighErrorRate,
                "Error rate exceeded 50%".to_string(),
                AlertSeverity::Critical,
            );
        }
        
        let alerts = collector.get_alerts(1);
        assert_eq!(alerts.len(), 1);
        assert_eq!(alerts[0].alert_type, AlertType::HighErrorRate);
    }

    #[test]
    fn test_pending_records_alert() {
        let collector = MetricsCollector::new();
        collector.update_pending_records(1000);
        
        if collector.check_pending_threshold(500) {
            collector.add_alert(
                AlertType::PendingRecordsThreshold,
                "Pending records exceeded 500".to_string(),
                AlertSeverity::Warning,
            );
        }
        
        let alerts = collector.get_alerts(1);
        assert_eq!(alerts.len(), 1);
        assert_eq!(alerts[0].alert_type, AlertType::PendingRecordsThreshold);
    }

    #[test]
    fn test_latency_alert() {
        let collector = MetricsCollector::new();
        collector.record_sync_success(10, Duration::from_millis(500));
        
        if collector.check_latency_threshold(300.0) {
            collector.add_alert(
                AlertType::HighLatency,
                "Latency exceeded 300ms".to_string(),
                AlertSeverity::Warning,
            );
        }
        
        let alerts = collector.get_alerts(1);
        assert_eq!(alerts.len(), 1);
        assert_eq!(alerts[0].alert_type, AlertType::HighLatency);
    }

    #[test]
    fn test_zero_error_rate() {
        let collector = MetricsCollector::new();
        collector.record_sync_success(100, Duration::from_millis(50));
        
        let metrics = collector.get_metrics();
        assert_eq!(metrics.error_rate, 0.0);
    }

    #[test]
    fn test_100_percent_error_rate() {
        let collector = MetricsCollector::new();
        collector.record_sync_failure(100);
        
        let metrics = collector.get_metrics();
        assert_eq!(metrics.error_rate, 100.0);
    }
}
