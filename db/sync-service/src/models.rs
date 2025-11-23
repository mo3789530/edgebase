use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use std::collections::HashMap;
use uuid::Uuid;

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

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct Device {
    pub device_id: Uuid,
    pub device_name: String,
    pub device_type: String,
    pub location: Option<String>,
    pub registered_at: DateTime<Utc>,
    pub last_seen_at: Option<DateTime<Utc>>,
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct Command {
    pub command_id: Uuid,
    pub device_id: Uuid,
    pub command_type: String,
    pub payload: serde_json::Value,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Serialize)]
pub struct BatchResult {
    pub success: bool,
    pub inserted: usize,
    pub failed: usize,
}
