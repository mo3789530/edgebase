use crate::models::{Command, TelemetryData};
use anyhow::Result;
use rusqlite::{params, Connection};

pub struct Database {
    conn: Connection,
}

impl Database {
    pub fn new(path: &str) -> Result<Self> {
        let conn = Connection::open(path)?;
        let db = Self { conn };
        db.init_schema()?;
        Ok(db)
    }

    fn init_schema(&self) -> Result<()> {
        self.conn.execute_batch(
            r#"
            CREATE TABLE IF NOT EXISTS telemetry_data (
                id TEXT PRIMARY KEY,
                device_id TEXT NOT NULL,
                sensor_id TEXT NOT NULL,
                timestamp INTEGER NOT NULL,
                data_type TEXT NOT NULL,
                value REAL NOT NULL,
                unit TEXT,
                metadata TEXT,
                sync_status TEXT DEFAULT 'pending',
                sync_timestamp INTEGER,
                version INTEGER DEFAULT 1,
                created_at INTEGER DEFAULT (strftime('%s', 'now'))
            );
            
            CREATE INDEX IF NOT EXISTS idx_sync_status ON telemetry_data(sync_status, timestamp);
            CREATE INDEX IF NOT EXISTS idx_device_sensor ON telemetry_data(device_id, sensor_id, timestamp);
            
            CREATE TABLE IF NOT EXISTS command_queue (
                id TEXT PRIMARY KEY,
                command_type TEXT NOT NULL,
                payload TEXT NOT NULL,
                status TEXT DEFAULT 'pending',
                received_at INTEGER NOT NULL,
                executed_at INTEGER,
                result TEXT
            );
            
            CREATE TABLE IF NOT EXISTS sync_log (
                id TEXT PRIMARY KEY,
                sync_type TEXT NOT NULL,
                started_at INTEGER NOT NULL,
                completed_at INTEGER,
                records_count INTEGER,
                status TEXT NOT NULL,
                error_message TEXT
            );
            "#,
        )?;
        Ok(())
    }

    pub fn insert_telemetry(&self, data: &TelemetryData) -> Result<()> {
        let metadata_json = data.metadata.as_ref().and_then(|m| serde_json::to_string(m).ok());
        
        self.conn.execute(
            r#"INSERT INTO telemetry_data 
               (id, device_id, sensor_id, timestamp, data_type, value, unit, metadata, version)
               VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9)"#,
            params![
                data.id,
                data.device_id,
                data.sensor_id,
                data.timestamp.timestamp(),
                data.data_type,
                data.value,
                data.unit,
                metadata_json,
                data.version,
            ],
        )?;
        Ok(())
    }

    pub fn get_pending_records(&self, limit: usize) -> Result<Vec<TelemetryData>> {
        let mut stmt = self.conn.prepare(
            r#"SELECT id, device_id, sensor_id, timestamp, data_type, value, unit, metadata, version
               FROM telemetry_data 
               WHERE sync_status = 'pending'
               ORDER BY timestamp ASC
               LIMIT ?1"#,
        )?;

        let rows = stmt.query_map(params![limit], |row| {
            Ok(TelemetryData {
                id: row.get(0)?,
                device_id: row.get(1)?,
                sensor_id: row.get(2)?,
                timestamp: chrono::DateTime::from_timestamp(row.get(3)?, 0).unwrap_or_default(),
                data_type: row.get(4)?,
                value: row.get(5)?,
                unit: row.get(6)?,
                metadata: row.get::<_, Option<String>>(7)?.and_then(|s| serde_json::from_str(&s).ok()),
                version: row.get(8)?,
            })
        })?;

        rows.collect::<Result<Vec<_>, _>>().map_err(Into::into)
    }

    pub fn mark_as_synced(&self, record_ids: &[String]) -> Result<()> {
        let tx = self.conn.unchecked_transaction()?;
        
        for id in record_ids {
            tx.execute(
                "UPDATE telemetry_data SET sync_status = 'synced', sync_timestamp = strftime('%s', 'now') WHERE id = ?1",
                params![id],
            )?;
        }
        
        tx.commit()?;
        Ok(())
    }

    pub fn mark_as_failed(&self, record_ids: &[String]) -> Result<()> {
        let tx = self.conn.unchecked_transaction()?;
        
        for id in record_ids {
            tx.execute(
                "UPDATE telemetry_data SET sync_status = 'failed' WHERE id = ?1",
                params![id],
            )?;
        }
        
        tx.commit()?;
        Ok(())
    }

    pub fn store_command(&self, command: &Command) -> Result<()> {
        let payload_json = serde_json::to_string(&command.payload)?;
        
        self.conn.execute(
            r#"INSERT OR REPLACE INTO command_queue 
               (id, command_type, payload, status, received_at)
               VALUES (?1, ?2, ?3, ?4, strftime('%s', 'now'))"#,
            params![
                command.command_id,
                command.command_type,
                payload_json,
                "pending",
            ],
        )?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::{Command, CommandStatus};
    use chrono::Utc;
    use std::collections::HashMap;
    use std::fs;
    use uuid::Uuid;

    fn setup_test_db() -> (Database, String) {
        let db_path = format!("/tmp/test_edge_{}.db", Uuid::new_v4());
        let db = Database::new(&db_path).expect("Failed to create test database");
        (db, db_path)
    }

    #[test]
    fn test_insert_telemetry() {
        let (db, db_path) = setup_test_db();
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

        let result = db.insert_telemetry(&data);
        assert!(result.is_ok());

        let pending = db.get_pending_records(10).expect("Failed to get pending records");
        assert_eq!(pending.len(), 1);
        assert_eq!(pending[0].id, "test-1");
        assert_eq!(pending[0].value, 25.5);
        
        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_insert_multiple_telemetry() {
        let (db, db_path) = setup_test_db();
        
        for i in 0..5 {
            let data = TelemetryData {
                id: format!("test-{}", i),
                device_id: "device-1".to_string(),
                sensor_id: "sensor-1".to_string(),
                timestamp: Utc::now(),
                data_type: "temperature".to_string(),
                value: 20.0 + i as f64,
                unit: Some("celsius".to_string()),
                metadata: None,
                version: 1,
            };
            db.insert_telemetry(&data).expect("Failed to insert");
        }

        let pending = db.get_pending_records(10).expect("Failed to get pending records");
        assert_eq!(pending.len(), 5);
        
        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_mark_as_synced() {
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
        
        let result = db.mark_as_synced(&["test-1".to_string()]);
        assert!(result.is_ok());

        let pending = db.get_pending_records(10).expect("Failed to get pending records");
        assert_eq!(pending.len(), 0);
        
        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_mark_as_failed() {
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
        
        let result = db.mark_as_failed(&["test-1".to_string()]);
        assert!(result.is_ok());

        let pending = db.get_pending_records(10).expect("Failed to get pending records");
        assert_eq!(pending.len(), 0);
        
        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_store_command() {
        let (db, db_path) = setup_test_db();
        
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

        let result = db.store_command(&command);
        assert!(result.is_ok());
        
        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_transaction_consistency() {
        let (db, db_path) = setup_test_db();
        
        for i in 0..3 {
            let data = TelemetryData {
                id: format!("test-{}", i),
                device_id: "device-1".to_string(),
                sensor_id: "sensor-1".to_string(),
                timestamp: Utc::now(),
                data_type: "temperature".to_string(),
                value: 20.0 + i as f64,
                unit: None,
                metadata: None,
                version: 1,
            };
            db.insert_telemetry(&data).expect("Failed to insert");
        }

        let result = db.mark_as_synced(&["test-0".to_string(), "test-1".to_string()]);
        assert!(result.is_ok());

        let pending = db.get_pending_records(10).expect("Failed to get pending records");
        assert_eq!(pending.len(), 1);
        assert_eq!(pending[0].id, "test-2");
        
        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_telemetry_with_metadata() {
        let (db, db_path) = setup_test_db();
        
        let mut metadata = HashMap::new();
        metadata.insert("location".to_string(), serde_json::json!("room-1"));
        metadata.insert("status".to_string(), serde_json::json!("active"));
        
        let data = TelemetryData {
            id: "test-1".to_string(),
            device_id: "device-1".to_string(),
            sensor_id: "sensor-1".to_string(),
            timestamp: Utc::now(),
            data_type: "temperature".to_string(),
            value: 25.5,
            unit: Some("celsius".to_string()),
            metadata: Some(metadata),
            version: 1,
        };

        db.insert_telemetry(&data).expect("Failed to insert");
        
        let pending = db.get_pending_records(10).expect("Failed to get pending records");
        assert_eq!(pending.len(), 1);
        assert!(pending[0].metadata.is_some());
        assert_eq!(pending[0].metadata.as_ref().unwrap().len(), 2);
        
        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_get_pending_records_limit() {
        let (db, db_path) = setup_test_db();
        
        for i in 0..10 {
            let data = TelemetryData {
                id: format!("test-{}", i),
                device_id: "device-1".to_string(),
                sensor_id: "sensor-1".to_string(),
                timestamp: Utc::now(),
                data_type: "temperature".to_string(),
                value: 20.0 + i as f64,
                unit: None,
                metadata: None,
                version: 1,
            };
            db.insert_telemetry(&data).expect("Failed to insert");
        }

        let pending = db.get_pending_records(5).expect("Failed to get pending records");
        assert_eq!(pending.len(), 5);
        
        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_version_tracking() {
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
            version: 3,
        };

        db.insert_telemetry(&data).expect("Failed to insert");
        
        let pending = db.get_pending_records(10).expect("Failed to get pending records");
        assert_eq!(pending[0].version, 3);
        
        let _ = fs::remove_file(db_path);
    }
}
