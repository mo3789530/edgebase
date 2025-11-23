use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TelemetryData {
    pub id: String,
    pub device_id: String,
    pub sensor_id: String,
    pub timestamp: DateTime<Utc>,
    pub data_type: String,
    pub value: f64,
    pub unit: Option<String>,
    pub metadata: Option<HashMap<String, serde_json::Value>>,
    pub version: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SyncStatus {
    Pending,
    Syncing,
    Synced,
    Failed,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Command {
    pub command_id: String,
    pub device_id: String,
    pub command_type: String,
    pub payload: HashMap<String, serde_json::Value>,
    pub status: CommandStatus,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum CommandStatus {
    Pending,
    Delivered,
    Executed,
    Failed,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SyncResult {
    pub success: bool,
    pub synced_count: usize,
    pub failed_count: usize,
    pub errors: Vec<String>,
}
