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
