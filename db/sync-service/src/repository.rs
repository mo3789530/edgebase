use crate::models::{Command, TelemetryData};
use crate::handlers::{DeviceRegistration, SyncStatus};
use anyhow::Result;
use sqlx::PgPool;
use uuid::Uuid;

pub struct Repository {
    pool: PgPool,
}

impl Repository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn insert_telemetry_batch(&self, batch: &[TelemetryData]) -> Result<usize> {
        let mut tx = self.pool.begin().await?;
        let mut inserted = 0;

        for data in batch {
            let device_uuid = Uuid::parse_str(&data.device_id)?;
            let metadata_json = data.metadata.as_ref().map(|m| serde_json::to_value(m).ok()).flatten();
            
            // Check for existing record with same ID (conflict detection)
            let existing: Option<(i32,)> = sqlx::query_as(
                "SELECT version FROM telemetry_data WHERE id = $1"
            )
            .bind(Uuid::parse_str(&data.id).ok())
            .fetch_optional(&mut *tx)
            .await?;

            if let Some((existing_version,)) = existing {
                // Conflict detected - use Last-Write-Wins strategy
                if data.version > existing_version {
                    sqlx::query(
                        r#"UPDATE telemetry_data 
                           SET sensor_id = $1, timestamp = $2, data_type = $3, value = $4, 
                               unit = $5, metadata = $6, version = $7, synced_at = NOW()
                           WHERE id = $8"#
                    )
                    .bind(&data.sensor_id)
                    .bind(data.timestamp)
                    .bind(&data.data_type)
                    .bind(data.value)
                    .bind(&data.unit)
                    .bind(metadata_json)
                    .bind(data.version)
                    .bind(Uuid::parse_str(&data.id)?)
                    .execute(&mut *tx)
                    .await?;
                    
                    inserted += 1;
                }
                // Else: skip older version
            } else {
                // No conflict - insert new record
                sqlx::query(
                    r#"INSERT INTO telemetry_data 
                       (device_id, sensor_id, timestamp, data_type, value, unit, metadata, version)
                       VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"#
                )
                .bind(device_uuid)
                .bind(&data.sensor_id)
                .bind(data.timestamp)
                .bind(&data.data_type)
                .bind(data.value)
                .bind(&data.unit)
                .bind(metadata_json)
                .bind(data.version)
                .execute(&mut *tx)
                .await?;

                inserted += 1;
            }
        }

        sqlx::query(
            "UPDATE sync_status SET last_sync_at = NOW(), last_sync_status = 'success', total_synced_records = total_synced_records + $1 WHERE device_id = $2"
        )
        .bind(inserted as i64)
        .bind(Uuid::parse_str(&batch[0].device_id)?)
        .execute(&mut *tx)
        .await?;

        tx.commit().await?;
        Ok(inserted)
    }

    pub async fn get_pending_commands(&self, device_id: Uuid) -> Result<Vec<Command>> {
        let commands = sqlx::query_as::<_, Command>(
            r#"SELECT command_id, device_id, command_type, payload, status, created_at
               FROM commands
               WHERE device_id = $1 AND status = 'pending'
               ORDER BY created_at ASC
               LIMIT 100"#
        )
        .bind(device_id)
        .fetch_all(&self.pool)
        .await?;

        Ok(commands)
    }

    pub async fn update_device_last_seen(&self, device_id: Uuid) -> Result<()> {
        sqlx::query("UPDATE devices SET last_seen_at = NOW() WHERE device_id = $1")
            .bind(device_id)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    pub async fn update_command_status(&self, command_id: Uuid, success: bool) -> Result<()> {
        let status = if success { "executed" } else { "failed" };
        
        sqlx::query(
            "UPDATE commands SET status = $1, executed_at = NOW() WHERE command_id = $2"
        )
        .bind(status)
        .bind(command_id)
        .execute(&self.pool)
        .await?;
        
        Ok(())
    }

    pub async fn get_sync_status(&self, device_id: Uuid) -> Result<SyncStatus> {
        let status = sqlx::query_as::<_, (Option<chrono::DateTime<chrono::Utc>>, Option<String>, i32, i64)>(
            r#"SELECT last_sync_at, last_sync_status, pending_records_count, total_synced_records
               FROM sync_status
               WHERE device_id = $1"#
        )
        .bind(device_id)
        .fetch_one(&self.pool)
        .await?;

        Ok(SyncStatus {
            device_id: device_id.to_string(),
            last_sync_at: status.0,
            last_sync_status: status.1.unwrap_or_else(|| "unknown".to_string()),
            pending_records_count: status.2,
            total_synced_records: status.3,
        })
    }

    pub async fn register_device(&self, registration: DeviceRegistration) -> Result<Uuid> {
        let device_id = Uuid::new_v4();
        
        let mut tx = self.pool.begin().await?;
        
        sqlx::query(
            r#"INSERT INTO devices (device_id, device_name, device_type, location, status)
               VALUES ($1, $2, $3, $4, 'active')"#
        )
        .bind(device_id)
        .bind(&registration.device_name)
        .bind(&registration.device_type)
        .bind(&registration.location)
        .execute(&mut *tx)
        .await?;

        sqlx::query(
            "INSERT INTO sync_status (device_id) VALUES ($1)"
        )
        .bind(device_id)
        .execute(&mut *tx)
        .await?;

        tx.commit().await?;
        
        Ok(device_id)
    }
}
