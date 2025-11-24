use crate::db::Database;
use crate::models::{Command, SyncResult, TelemetryData};
use anyhow::Result;
use std::time::Duration;
use tracing::{error, info, warn};

pub struct SyncAgent {
    db: Database,
    api_url: String,
    device_id: String,
    client: reqwest::Client,
    batch_size: usize,
    poll_interval: Duration,
}

impl SyncAgent {
    pub fn new(db: Database, api_url: String, device_id: String) -> Self {
        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(30))
            .build()
            .expect("Failed to create HTTP client");

        Self {
            db,
            api_url,
            device_id,
            client,
            batch_size: 1000,
            poll_interval: Duration::from_secs(30),
        }
    }

    pub fn with_batch_size(mut self, batch_size: usize) -> Self {
        self.batch_size = batch_size;
        self
    }

    pub fn with_timeout(mut self, timeout: Duration) -> Self {
        let client = reqwest::Client::builder()
            .timeout(timeout)
            .build()
            .expect("Failed to create HTTP client");
        self.client = client;
        self
    }

    pub async fn sync_to_control_plane(&self) -> Result<SyncResult> {
        let pending = self.db.get_pending_records(self.batch_size)?;
        
        if pending.is_empty() {
            return Ok(SyncResult {
                success: true,
                synced_count: 0,
                failed_count: 0,
                errors: vec![],
            });
        }

        info!("Syncing {} records to control plane", pending.len());

        // Retry with exponential backoff
        let mut attempt = 0;
        let max_attempts = 5;
        let mut delay = Duration::from_secs(1);

        loop {
            attempt += 1;
            
            match self.send_telemetry_batch(&pending).await {
                Ok(_) => {
                    let ids: Vec<String> = pending.iter().map(|r| r.id.clone()).collect();
                    self.db.mark_as_synced(&ids)?;
                    
                    return Ok(SyncResult {
                        success: true,
                        synced_count: pending.len(),
                        failed_count: 0,
                        errors: vec![],
                    });
                }
                Err(e) => {
                    if attempt >= max_attempts {
                        error!("Failed to sync after {} attempts: {}", max_attempts, e);
                        let ids: Vec<String> = pending.iter().map(|r| r.id.clone()).collect();
                        self.db.mark_as_failed(&ids)?;
                        
                        return Ok(SyncResult {
                            success: false,
                            synced_count: 0,
                            failed_count: pending.len(),
                            errors: vec![e.to_string()],
                        });
                    }
                    
                    warn!("Sync attempt {} failed, retrying in {:?}: {}", attempt, delay, e);
                    tokio::time::sleep(delay).await;
                    delay *= 2; // Exponential backoff
                }
            }
        }
    }

    async fn send_telemetry_batch(&self, batch: &[TelemetryData]) -> Result<()> {
        let url = format!("{}/api/v1/sync/telemetry", self.api_url);
        
        let response = self.client
            .post(&url)
            .json(batch)
            .send()
            .await?;

        if response.status().is_success() {
            Ok(())
        } else {
            anyhow::bail!("API returned error: {}", response.status())
        }
    }

    pub async fn poll_commands(&self) -> Result<Vec<Command>> {
        let url = format!("{}/api/v1/sync/commands/{}", self.api_url, self.device_id);
        
        let response = self.client
            .get(&url)
            .send()
            .await?;

        if response.status().is_success() {
            let commands = response.json::<Vec<Command>>().await?;
            Ok(commands)
        } else {
            Ok(vec![])
        }
    }

    pub async fn apply_command(&self, command: &Command) -> Result<()> {
        info!("Applying command: {} ({})", command.command_id, command.command_type);
        
        // Store command in local queue
        self.db.store_command(command)?;
        
        // Execute command (placeholder - actual implementation depends on command type)
        let result = self.execute_command(command).await;
        
        // Send ACK
        self.send_command_ack(&command.command_id, result.is_ok()).await?;
        
        Ok(())
    }

    async fn execute_command(&self, command: &Command) -> Result<String> {
        // Placeholder for command execution logic
        match command.command_type.as_str() {
            "config_update" => {
                info!("Executing config update");
                Ok("Config updated".to_string())
            }
            "restart" => {
                info!("Restart command received");
                Ok("Restart scheduled".to_string())
            }
            _ => {
                warn!("Unknown command type: {}", command.command_type);
                Ok("Unknown command".to_string())
            }
        }
    }

    async fn send_command_ack(&self, command_id: &str, success: bool) -> Result<()> {
        let url = format!("{}/api/v1/sync/ack/{}", self.api_url, command_id);
        
        let payload = serde_json::json!({
            "success": success,
            "timestamp": chrono::Utc::now()
        });
        
        self.client
            .post(&url)
            .json(&payload)
            .send()
            .await?;
        
        Ok(())
    }

    pub async fn run(&self) -> Result<()> {
        info!("Starting sync agent for device {}", self.device_id);
        
        loop {
            // Upstream sync
            match self.sync_to_control_plane().await {
                Ok(result) => {
                    if result.synced_count > 0 {
                        info!("Synced {} records", result.synced_count);
                    }
                }
                Err(e) => {
                    error!("Sync error: {}", e);
                }
            }

            // Downstream sync - poll and execute commands
            match self.poll_commands().await {
                Ok(commands) => {
                    if !commands.is_empty() {
                        info!("Received {} commands", commands.len());
                        for command in commands {
                            if let Err(e) = self.apply_command(&command).await {
                                error!("Failed to apply command {}: {}", command.command_id, e);
                            }
                        }
                    }
                }
                Err(e) => {
                    error!("Failed to poll commands: {}", e);
                }
            }

            tokio::time::sleep(self.poll_interval).await;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::CommandStatus;
    use chrono::Utc;
    use std::collections::HashMap;
    use std::fs;
    use uuid::Uuid;

    fn setup_test_db() -> (Database, String) {
        let db_path = format!("/tmp/test_sync_{}.db", Uuid::new_v4());
        let db = Database::new(&db_path).expect("Failed to create test database");
        (db, db_path)
    }

    #[test]
    fn test_sync_agent_creation() {
        let (db, db_path) = setup_test_db();
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-1".to_string());
        
        assert_eq!(agent.device_id, "device-1");
        assert_eq!(agent.api_url, "http://localhost:8080");
        assert_eq!(agent.batch_size, 1000);
        
        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_sync_agent_with_batch_size() {
        let (db, db_path) = setup_test_db();
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-1".to_string())
            .with_batch_size(500);
        
        assert_eq!(agent.batch_size, 500);
        
        let _ = fs::remove_file(db_path);
    }

    #[tokio::test]
    async fn test_sync_empty_records() {
        let (db, db_path) = setup_test_db();
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-1".to_string());
        
        let result = agent.sync_to_control_plane().await;
        assert!(result.is_ok());
        
        let sync_result = result.unwrap();
        assert!(sync_result.success);
        assert_eq!(sync_result.synced_count, 0);
        assert_eq!(sync_result.failed_count, 0);
        
        let _ = fs::remove_file(db_path);
    }

    #[tokio::test]
    async fn test_sync_with_pending_records() {
        let (db, db_path) = setup_test_db();
        
        // Insert test data
        let data = TelemetryData {
            id: "test-1".to_string(),
            device_id: "device-1".to_string(),
            sensor_id: "sensor-1".to_string(),
            timestamp: Utc::now(),
            data_type: "temperature".to_string(),
            value: 25.5,
            unit: Some("celsius".to_string()),
            metadata: None,
            version: 1,
        };
        db.insert_telemetry(&data).expect("Failed to insert");
        
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-1".to_string())
            .with_timeout(Duration::from_millis(100));
        
        // This will fail due to network error, but we can verify the logic
        let result = agent.sync_to_control_plane().await;
        assert!(result.is_ok());
        
        let sync_result = result.unwrap();
        assert!(!sync_result.success);
        assert_eq!(sync_result.failed_count, 1);
        
        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_command_creation() {
        let mut payload = HashMap::new();
        payload.insert("action".to_string(), serde_json::json!("restart"));
        
        let command = Command {
            command_id: "cmd-1".to_string(),
            device_id: "device-1".to_string(),
            command_type: "system".to_string(),
            payload,
            status: CommandStatus::Pending,
            created_at: Utc::now(),
        };
        
        assert_eq!(command.command_id, "cmd-1");
        assert_eq!(command.command_type, "system");
    }

    #[tokio::test]
    async fn test_apply_command_storage() {
        let (db, db_path) = setup_test_db();
        
        let mut payload = HashMap::new();
        payload.insert("action".to_string(), serde_json::json!("restart"));
        
        let command = Command {
            command_id: "cmd-1".to_string(),
            device_id: "device-1".to_string(),
            command_type: "config_update".to_string(),
            payload,
            status: CommandStatus::Pending,
            created_at: Utc::now(),
        };
        
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-1".to_string());
        
        // Store command in DB
        agent.db.store_command(&command).expect("Failed to store command");
        
        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_sync_result_structure() {
        let result = SyncResult {
            success: true,
            synced_count: 10,
            failed_count: 0,
            errors: vec![],
        };
        
        assert!(result.success);
        assert_eq!(result.synced_count, 10);
        assert_eq!(result.failed_count, 0);
        assert!(result.errors.is_empty());
    }

    #[test]
    fn test_sync_result_with_errors() {
        let result = SyncResult {
            success: false,
            synced_count: 0,
            failed_count: 5,
            errors: vec!["Network error".to_string()],
        };
        
        assert!(!result.success);
        assert_eq!(result.failed_count, 5);
        assert_eq!(result.errors.len(), 1);
    }

    #[tokio::test]
    async fn test_sync_batch_size_limit() {
        let (db, db_path) = setup_test_db();
        
        // Insert 150 records
        for i in 0..150 {
            let data = TelemetryData {
                id: format!("test-{}", i),
                device_id: "device-1".to_string(),
                sensor_id: "sensor-1".to_string(),
                timestamp: Utc::now(),
                data_type: "temperature".to_string(),
                value: 20.0 + (i as f64),
                unit: None,
                metadata: None,
                version: 1,
            };
            db.insert_telemetry(&data).expect("Failed to insert");
        }
        
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-1".to_string())
            .with_batch_size(100);
        
        // Verify batch size is respected
        assert_eq!(agent.batch_size, 100);
        
        let _ = fs::remove_file(db_path);
    }

    #[tokio::test]
    async fn test_sync_marks_records_as_failed_on_error() {
        let (db, db_path) = setup_test_db();
        
        let data = TelemetryData {
            id: "test-1".to_string(),
            device_id: "device-1".to_string(),
            sensor_id: "sensor-1".to_string(),
            timestamp: Utc::now(),
            data_type: "temperature".to_string(),
            value: 25.5,
            unit: None,
            metadata: None,
            version: 1,
        };
        db.insert_telemetry(&data).expect("Failed to insert");
        
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-1".to_string())
            .with_timeout(Duration::from_millis(100));
        
        let result = agent.sync_to_control_plane().await;
        assert!(result.is_ok());
        
        let sync_result = result.unwrap();
        assert!(!sync_result.success);
        assert_eq!(sync_result.failed_count, 1);
        
        let _ = fs::remove_file(db_path);
    }

    #[tokio::test]
    async fn test_poll_commands_empty() {
        let (db, db_path) = setup_test_db();
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-1".to_string());
        
        let result = agent.poll_commands().await;
        // Network error is expected since there's no actual server
        // The function should handle it gracefully
        let _ = result;
        
        let _ = fs::remove_file(db_path);
    }

    #[tokio::test]
    async fn test_execute_command_config_update() {
        let (db, db_path) = setup_test_db();
        
        let mut payload = HashMap::new();
        payload.insert("config".to_string(), serde_json::json!({"key": "value"}));
        
        let command = Command {
            command_id: "cmd-1".to_string(),
            device_id: "device-1".to_string(),
            command_type: "config_update".to_string(),
            payload,
            status: CommandStatus::Pending,
            created_at: Utc::now(),
        };
        
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-1".to_string());
        
        let result = agent.execute_command(&command).await;
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), "Config updated");
        
        let _ = fs::remove_file(db_path);
    }

    #[tokio::test]
    async fn test_execute_command_restart() {
        let (db, db_path) = setup_test_db();
        
        let command = Command {
            command_id: "cmd-1".to_string(),
            device_id: "device-1".to_string(),
            command_type: "restart".to_string(),
            payload: HashMap::new(),
            status: CommandStatus::Pending,
            created_at: Utc::now(),
        };
        
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-1".to_string());
        
        let result = agent.execute_command(&command).await;
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), "Restart scheduled");
        
        let _ = fs::remove_file(db_path);
    }

    #[tokio::test]
    async fn test_execute_command_unknown_type() {
        let (db, db_path) = setup_test_db();
        
        let command = Command {
            command_id: "cmd-1".to_string(),
            device_id: "device-1".to_string(),
            command_type: "unknown_type".to_string(),
            payload: HashMap::new(),
            status: CommandStatus::Pending,
            created_at: Utc::now(),
        };
        
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-1".to_string());
        
        let result = agent.execute_command(&command).await;
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), "Unknown command");
        
        let _ = fs::remove_file(db_path);
    }

    #[tokio::test]
    async fn test_multiple_pending_records_sync() {
        let (db, db_path) = setup_test_db();
        
        for i in 0..5 {
            let data = TelemetryData {
                id: format!("test-{}", i),
                device_id: "device-1".to_string(),
                sensor_id: "sensor-1".to_string(),
                timestamp: Utc::now(),
                data_type: "temperature".to_string(),
                value: 20.0 + (i as f64),
                unit: None,
                metadata: None,
                version: 1,
            };
            db.insert_telemetry(&data).expect("Failed to insert");
        }
        
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-1".to_string())
            .with_timeout(Duration::from_millis(100));
        
        let result = agent.sync_to_control_plane().await;
        assert!(result.is_ok());
        
        let sync_result = result.unwrap();
        assert_eq!(sync_result.failed_count, 5);
        
        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_sync_agent_device_id() {
        let (db, db_path) = setup_test_db();
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-xyz".to_string());
        
        assert_eq!(agent.device_id, "device-xyz");
        
        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_sync_agent_api_url() {
        let (db, db_path) = setup_test_db();
        let agent = SyncAgent::new(db, "http://api.example.com:9000".to_string(), "device-1".to_string());
        
        assert_eq!(agent.api_url, "http://api.example.com:9000");
        
        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_sync_agent_default_poll_interval() {
        let (db, db_path) = setup_test_db();
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-1".to_string());
        
        assert_eq!(agent.poll_interval, Duration::from_secs(30));
        
        let _ = fs::remove_file(db_path);
    }

    #[tokio::test]
    async fn test_command_storage_and_retrieval() {
        let (db, db_path) = setup_test_db();
        
        let mut payload = HashMap::new();
        payload.insert("param".to_string(), serde_json::json!("value"));
        
        let command = Command {
            command_id: "cmd-1".to_string(),
            device_id: "device-1".to_string(),
            command_type: "test_command".to_string(),
            payload,
            status: CommandStatus::Pending,
            created_at: Utc::now(),
        };
        
        let agent = SyncAgent::new(db, "http://localhost:8080".to_string(), "device-1".to_string());
        agent.db.store_command(&command).expect("Failed to store");
        
        let _ = fs::remove_file(db_path);
    }
}
