use crate::models::{BatchResult, TelemetryData};
use crate::repository::Repository;
use axum::{
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    Json,
};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tracing::{error, info};
use uuid::Uuid;

#[derive(Debug, Deserialize)]
pub struct CommandAck {
    pub success: bool,
    pub timestamp: chrono::DateTime<chrono::Utc>,
}

#[derive(Debug, Serialize)]
pub struct SyncStatus {
    pub device_id: String,
    pub last_sync_at: Option<chrono::DateTime<chrono::Utc>>,
    pub last_sync_status: String,
    pub pending_records_count: i32,
    pub total_synced_records: i64,
}

#[derive(Debug, Deserialize)]
pub struct DeviceRegistration {
    pub device_name: String,
    pub device_type: String,
    pub location: Option<String>,
}

pub async fn sync_telemetry(
    State(repo): State<Arc<Repository>>,
    Json(batch): Json<Vec<TelemetryData>>,
) -> impl IntoResponse {
    if batch.is_empty() {
        return (StatusCode::BAD_REQUEST, Json(BatchResult {
            success: false,
            inserted: 0,
            failed: 0,
        }));
    }

    info!("Received telemetry batch with {} records", batch.len());

    match repo.insert_telemetry_batch(&batch).await {
        Ok(inserted) => {
            if let Ok(device_id) = Uuid::parse_str(&batch[0].device_id) {
                let _ = repo.update_device_last_seen(device_id).await;
            }

            (StatusCode::OK, Json(BatchResult {
                success: true,
                inserted,
                failed: 0,
            }))
        }
        Err(e) => {
            error!("Failed to insert telemetry batch: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Json(BatchResult {
                success: false,
                inserted: 0,
                failed: batch.len(),
            }))
        }
    }
}

pub async fn get_commands(
    State(repo): State<Arc<Repository>>,
    Path(device_id): Path<String>,
) -> impl IntoResponse {
    let device_uuid = match Uuid::parse_str(&device_id) {
        Ok(uuid) => uuid,
        Err(_) => return (StatusCode::BAD_REQUEST, Json(vec![])),
    };

    match repo.get_pending_commands(device_uuid).await {
        Ok(commands) => (StatusCode::OK, Json(commands)),
        Err(e) => {
            error!("Failed to get commands: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Json(vec![]))
        }
    }
}

pub async fn health_check() -> impl IntoResponse {
    (StatusCode::OK, "OK")
}

pub async fn command_ack(
    State(repo): State<Arc<Repository>>,
    Path(command_id): Path<String>,
    Json(ack): Json<CommandAck>,
) -> impl IntoResponse {
    let command_uuid = match Uuid::parse_str(&command_id) {
        Ok(uuid) => uuid,
        Err(_) => return StatusCode::BAD_REQUEST,
    };

    match repo.update_command_status(command_uuid, ack.success).await {
        Ok(_) => {
            info!("Command {} acknowledged: {}", command_id, ack.success);
            StatusCode::OK
        }
        Err(e) => {
            error!("Failed to update command status: {}", e);
            StatusCode::INTERNAL_SERVER_ERROR
        }
    }
}

pub async fn get_sync_status(
    State(repo): State<Arc<Repository>>,
    Path(device_id): Path<String>,
) -> impl IntoResponse {
    let device_uuid = match Uuid::parse_str(&device_id) {
        Ok(uuid) => uuid,
        Err(_) => return (StatusCode::BAD_REQUEST, Json(None)),
    };

    match repo.get_sync_status(device_uuid).await {
        Ok(status) => (StatusCode::OK, Json(Some(status))),
        Err(e) => {
            error!("Failed to get sync status: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Json(None))
        }
    }
}

pub async fn register_device(
    State(repo): State<Arc<Repository>>,
    Json(registration): Json<DeviceRegistration>,
) -> impl IntoResponse {
    match repo.register_device(registration).await {
        Ok(device_id) => {
            info!("Device registered: {}", device_id);
            (StatusCode::CREATED, Json(serde_json::json!({ "device_id": device_id })))
        }
        Err(e) => {
            error!("Failed to register device: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({ "error": e.to_string() })))
        }
    }
}
