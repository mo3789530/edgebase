use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct HeartbeatPayload {
    pub node_id: String,
    pub pop_id: String,
    pub timestamp: u64,
    pub status: String,
    pub function_count: usize,
    pub cached_functions: Vec<CachedFunctionInfo>,
    pub metrics: NodeMetrics,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct CachedFunctionInfo {
    pub function_id: String,
    pub version: u32,
    pub status: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct NodeMetrics {
    pub memory_usage_mb: u64,
    pub cpu_usage_percent: f32,
    pub active_instances: usize,
    pub total_invocations: u64,
    pub error_count: u64,
}

impl HeartbeatPayload {
    pub fn new(node_id: String, pop_id: String) -> Self {
        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();

        HeartbeatPayload {
            node_id,
            pop_id,
            timestamp,
            status: "healthy".to_string(),
            function_count: 0,
            cached_functions: Vec::new(),
            metrics: NodeMetrics {
                memory_usage_mb: 0,
                cpu_usage_percent: 0.0,
                active_instances: 0,
                total_invocations: 0,
                error_count: 0,
            },
        }
    }

    pub fn add_cached_function(&mut self, function_id: String, version: u32, status: String) {
        self.cached_functions.push(CachedFunctionInfo {
            function_id,
            version,
            status,
        });
        self.function_count = self.cached_functions.len();
    }

    pub fn set_metrics(&mut self, metrics: NodeMetrics) {
        self.metrics = metrics;
    }

    pub fn set_status(&mut self, status: String) {
        self.status = status;
    }

    pub fn to_json(&self) -> Result<String, String> {
        serde_json::to_string(self)
            .map_err(|e| format!("Failed to serialize heartbeat: {}", e))
    }
}

pub struct HeartbeatManager {
    node_id: String,
    pop_id: String,
    last_heartbeat: std::sync::Arc<std::sync::Mutex<u64>>,
}

impl HeartbeatManager {
    pub fn new(node_id: String, pop_id: String) -> Self {
        HeartbeatManager {
            node_id,
            pop_id,
            last_heartbeat: std::sync::Arc::new(std::sync::Mutex::new(0)),
        }
    }

    pub fn create_heartbeat(&self) -> HeartbeatPayload {
        let mut payload = HeartbeatPayload::new(self.node_id.clone(), self.pop_id.clone());
        
        let mut last = self.last_heartbeat.lock().unwrap();
        *last = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();
        
        payload
    }

    pub fn get_last_heartbeat(&self) -> u64 {
        *self.last_heartbeat.lock().unwrap()
    }

    pub fn time_since_last_heartbeat(&self) -> u64 {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();
        
        let last = *self.last_heartbeat.lock().unwrap();
        if last == 0 {
            0
        } else {
            now - last
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_heartbeat_payload_creation() {
        let payload = HeartbeatPayload::new("node1".to_string(), "pop1".to_string());
        assert_eq!(payload.node_id, "node1");
        assert_eq!(payload.pop_id, "pop1");
        assert_eq!(payload.status, "healthy");
        assert_eq!(payload.function_count, 0);
    }

    #[test]
    fn test_heartbeat_payload_add_function() {
        let mut payload = HeartbeatPayload::new("node1".to_string(), "pop1".to_string());
        payload.add_cached_function("fn1".to_string(), 1, "cached".to_string());
        
        assert_eq!(payload.function_count, 1);
        assert_eq!(payload.cached_functions.len(), 1);
    }

    #[test]
    fn test_heartbeat_payload_set_metrics() {
        let mut payload = HeartbeatPayload::new("node1".to_string(), "pop1".to_string());
        let metrics = NodeMetrics {
            memory_usage_mb: 512,
            cpu_usage_percent: 25.5,
            active_instances: 5,
            total_invocations: 1000,
            error_count: 10,
        };
        
        payload.set_metrics(metrics.clone());
        assert_eq!(payload.metrics.memory_usage_mb, 512);
        assert_eq!(payload.metrics.cpu_usage_percent, 25.5);
    }

    #[test]
    fn test_heartbeat_payload_to_json() {
        let payload = HeartbeatPayload::new("node1".to_string(), "pop1".to_string());
        let json = payload.to_json();
        assert!(json.is_ok());
        
        let json_str = json.unwrap();
        assert!(json_str.contains("node1"));
        assert!(json_str.contains("pop1"));
    }

    #[test]
    fn test_heartbeat_manager_creation() {
        let manager = HeartbeatManager::new("node1".to_string(), "pop1".to_string());
        assert_eq!(manager.node_id, "node1");
        assert_eq!(manager.pop_id, "pop1");
    }

    #[test]
    fn test_heartbeat_manager_create_heartbeat() {
        let manager = HeartbeatManager::new("node1".to_string(), "pop1".to_string());
        let payload = manager.create_heartbeat();
        
        assert_eq!(payload.node_id, "node1");
        assert_eq!(payload.pop_id, "pop1");
        assert!(manager.get_last_heartbeat() > 0);
    }

    #[test]
    fn test_heartbeat_manager_time_since_last() {
        let manager = HeartbeatManager::new("node1".to_string(), "pop1".to_string());
        assert_eq!(manager.time_since_last_heartbeat(), 0);
        
        manager.create_heartbeat();
        std::thread::sleep(std::time::Duration::from_millis(100));
        
        let time_since = manager.time_since_last_heartbeat();
        assert!(time_since >= 0);
    }
}
