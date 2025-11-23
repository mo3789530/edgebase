use crate::db::Database;
use crate::models::TelemetryData;
use anyhow::Result;
use chrono::Utc;
use std::collections::HashMap;
use uuid::Uuid;

pub struct TelemetryCollector {
    db: Database,
    device_id: String,
}

impl TelemetryCollector {
    pub fn new(db: Database, device_id: String) -> Self {
        Self { db, device_id }
    }

    pub fn collect_sensor_data(
        &self,
        sensor_id: &str,
        data_type: &str,
        value: f64,
        unit: Option<String>,
    ) -> Result<TelemetryData> {
        let data = TelemetryData {
            id: Uuid::new_v4().to_string(),
            device_id: self.device_id.clone(),
            sensor_id: sensor_id.to_string(),
            timestamp: Utc::now(),
            data_type: data_type.to_string(),
            value,
            unit,
            metadata: None,
            version: 1,
        };

        self.db.insert_telemetry(&data)?;
        Ok(data)
    }

    pub fn collect_sensor_data_with_metadata(
        &self,
        sensor_id: &str,
        data_type: &str,
        value: f64,
        unit: Option<String>,
        metadata: Option<HashMap<String, serde_json::Value>>,
    ) -> Result<TelemetryData> {
        let data = TelemetryData {
            id: Uuid::new_v4().to_string(),
            device_id: self.device_id.clone(),
            sensor_id: sensor_id.to_string(),
            timestamp: Utc::now(),
            data_type: data_type.to_string(),
            value,
            unit,
            metadata,
            version: 1,
        };

        self.db.insert_telemetry(&data)?;
        Ok(data)
    }

    pub fn collect_batch(
        &self,
        readings: Vec<(String, String, f64, Option<String>)>,
    ) -> Result<Vec<TelemetryData>> {
        let mut collected = Vec::new();

        for (sensor_id, data_type, value, unit) in readings {
            let data = self.collect_sensor_data(&sensor_id, &data_type, value, unit)?;
            collected.push(data);
        }

        Ok(collected)
    }

    pub fn get_pending_count(&self) -> Result<usize> {
        let pending = self.db.get_pending_records(10000)?;
        Ok(pending.len())
    }

    pub fn validate_reading(value: f64, min: f64, max: f64) -> bool {
        value >= min && value <= max
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    fn setup_test_db() -> (Database, String) {
        let db_path = format!("/tmp/test_telemetry_{}.db", Uuid::new_v4());
        let db = Database::new(&db_path).expect("Failed to create test database");
        (db, db_path)
    }

    #[test]
    fn test_collector_creation() {
        let (db, db_path) = setup_test_db();
        let collector = TelemetryCollector::new(db, "device-1".to_string());

        assert_eq!(collector.device_id, "device-1");

        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_collect_single_sensor_data() {
        let (db, db_path) = setup_test_db();
        let collector = TelemetryCollector::new(db, "device-1".to_string());

        let result = collector.collect_sensor_data("sensor-1", "temperature", 25.5, Some("celsius".to_string()));
        assert!(result.is_ok());

        let data = result.unwrap();
        assert_eq!(data.device_id, "device-1");
        assert_eq!(data.sensor_id, "sensor-1");
        assert_eq!(data.data_type, "temperature");
        assert_eq!(data.value, 25.5);
        assert_eq!(data.unit, Some("celsius".to_string()));

        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_collect_multiple_sensors() {
        let (db, db_path) = setup_test_db();
        let collector = TelemetryCollector::new(db, "device-1".to_string());

        for i in 0..5 {
            let result = collector.collect_sensor_data(
                &format!("sensor-{}", i),
                "temperature",
                20.0 + i as f64,
                Some("celsius".to_string()),
            );
            assert!(result.is_ok());
        }

        let pending_count = collector.get_pending_count().expect("Failed to get pending count");
        assert_eq!(pending_count, 5);

        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_collect_with_metadata() {
        let (db, db_path) = setup_test_db();
        let collector = TelemetryCollector::new(db, "device-1".to_string());

        let mut metadata = HashMap::new();
        metadata.insert("location".to_string(), serde_json::json!("room-1"));
        metadata.insert("status".to_string(), serde_json::json!("active"));

        let result = collector.collect_sensor_data_with_metadata(
            "sensor-1",
            "temperature",
            25.5,
            Some("celsius".to_string()),
            Some(metadata),
        );
        assert!(result.is_ok());

        let data = result.unwrap();
        assert!(data.metadata.is_some());
        assert_eq!(data.metadata.as_ref().unwrap().len(), 2);

        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_collect_batch() {
        let (db, db_path) = setup_test_db();
        let collector = TelemetryCollector::new(db, "device-1".to_string());

        let readings = vec![
            ("sensor-1".to_string(), "temperature".to_string(), 25.5, Some("celsius".to_string())),
            ("sensor-2".to_string(), "humidity".to_string(), 60.0, Some("%".to_string())),
            ("sensor-3".to_string(), "pressure".to_string(), 1013.25, Some("hPa".to_string())),
        ];

        let result = collector.collect_batch(readings);
        assert!(result.is_ok());

        let collected = result.unwrap();
        assert_eq!(collected.len(), 3);

        let pending_count = collector.get_pending_count().expect("Failed to get pending count");
        assert_eq!(pending_count, 3);

        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_validate_reading() {
        assert!(TelemetryCollector::validate_reading(25.0, 0.0, 50.0));
        assert!(TelemetryCollector::validate_reading(0.0, 0.0, 50.0));
        assert!(TelemetryCollector::validate_reading(50.0, 0.0, 50.0));
        assert!(!TelemetryCollector::validate_reading(-1.0, 0.0, 50.0));
        assert!(!TelemetryCollector::validate_reading(51.0, 0.0, 50.0));
    }

    #[test]
    fn test_collect_without_unit() {
        let (db, db_path) = setup_test_db();
        let collector = TelemetryCollector::new(db, "device-1".to_string());

        let result = collector.collect_sensor_data("sensor-1", "count", 42.0, None);
        assert!(result.is_ok());

        let data = result.unwrap();
        assert_eq!(data.value, 42.0);
        assert!(data.unit.is_none());

        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_collect_various_data_types() {
        let (db, db_path) = setup_test_db();
        let collector = TelemetryCollector::new(db, "device-1".to_string());

        let data_types = vec![
            ("sensor-1", "temperature", 25.5),
            ("sensor-2", "humidity", 60.0),
            ("sensor-3", "pressure", 1013.25),
            ("sensor-4", "co2", 400.0),
        ];

        for (sensor_id, data_type, value) in data_types {
            let result = collector.collect_sensor_data(sensor_id, data_type, value, None);
            assert!(result.is_ok());
        }

        let pending_count = collector.get_pending_count().expect("Failed to get pending count");
        assert_eq!(pending_count, 4);

        let _ = fs::remove_file(db_path);
    }

    #[test]
    fn test_offline_data_accumulation() {
        let (db, db_path) = setup_test_db();
        let collector = TelemetryCollector::new(db, "device-1".to_string());

        // Simulate offline data collection
        for i in 0..10 {
            let result = collector.collect_sensor_data(
                "sensor-1",
                "temperature",
                20.0 + i as f64,
                Some("celsius".to_string()),
            );
            assert!(result.is_ok());
        }

        let pending_count = collector.get_pending_count().expect("Failed to get pending count");
        assert_eq!(pending_count, 10);

        let _ = fs::remove_file(db_path);
    }
}
