use std::env;

#[derive(Clone, Debug)]
pub struct Config {
    pub node_id: String,
    pub pop_id: String,
    pub cp_url: String,
    pub listen_addr: String,
    pub listen_port: u16,
    pub cache_dir: String,
    pub cache_size_gb: u64,
    pub min_hot_instances: usize,
    pub max_hot_instances: usize,
    pub idle_timeout_secs: u64,
    pub heartbeat_interval_secs: u64,
    pub minio_endpoint: String,
    pub minio_access_key: String,
    pub minio_secret_key: String,
    pub mqtt_broker: String,
}

impl Config {
    pub fn from_env() -> Self {
        Config {
            node_id: env::var("NODE_ID").unwrap_or_else(|_| uuid::Uuid::new_v4().to_string()),
            pop_id: env::var("POP_ID").unwrap_or_else(|_| "default-pop".to_string()),
            cp_url: env::var("CP_URL").unwrap_or_else(|_| "http://localhost:8080".to_string()),
            listen_addr: env::var("LISTEN_ADDR").unwrap_or_else(|_| "0.0.0.0".to_string()),
            listen_port: env::var("LISTEN_PORT")
                .ok()
                .and_then(|p| p.parse().ok())
                .unwrap_or(3000),
            cache_dir: env::var("CACHE_DIR").unwrap_or_else(|_| "/tmp/wasm-cache".to_string()),
            cache_size_gb: env::var("CACHE_SIZE_GB")
                .ok()
                .and_then(|s| s.parse().ok())
                .unwrap_or(10),
            min_hot_instances: env::var("MIN_HOT_INSTANCES")
                .ok()
                .and_then(|s| s.parse().ok())
                .unwrap_or(1),
            max_hot_instances: env::var("MAX_HOT_INSTANCES")
                .ok()
                .and_then(|s| s.parse().ok())
                .unwrap_or(10),
            idle_timeout_secs: env::var("IDLE_TIMEOUT_SECS")
                .ok()
                .and_then(|s| s.parse().ok())
                .unwrap_or(300),
            heartbeat_interval_secs: env::var("HEARTBEAT_INTERVAL_SECS")
                .ok()
                .and_then(|s| s.parse().ok())
                .unwrap_or(30),
            minio_endpoint: env::var("MINIO_ENDPOINT").unwrap_or_else(|_| "http://localhost:9000".to_string()),
            minio_access_key: env::var("MINIO_ACCESS_KEY").unwrap_or_else(|_| "minioadmin".to_string()),
            minio_secret_key: env::var("MINIO_SECRET_KEY").unwrap_or_else(|_| "minioadmin".to_string()),
            mqtt_broker: env::var("MQTT_BROKER").unwrap_or_else(|_| "mqtt://localhost:1883".to_string()),
        }
    }
}
