use chrono::Utc;
use edge_agent::{db::Database, models::TelemetryData};
use uuid::Uuid;

fn main() -> anyhow::Result<()> {
    let db = Database::new("edge.db")?;
    
    let device_id = std::env::var("DEVICE_ID").unwrap_or_else(|_| Uuid::new_v4().to_string());
    
    for i in 0..10 {
        let data = TelemetryData {
            id: Uuid::new_v4().to_string(),
            device_id: device_id.clone(),
            sensor_id: format!("sensor-{}", i % 3),
            timestamp: Utc::now(),
            data_type: "temperature".to_string(),
            value: 20.0 + (i as f64) * 0.5,
            unit: Some("celsius".to_string()),
            metadata: None,
            version: 1,
        };
        
        db.insert_telemetry(&data)?;
        println!("Inserted telemetry record: {}", data.id);
    }
    
    println!("Successfully inserted 10 sample records");
    Ok(())
}
