# IoT Data Sync System - Programming Guide

## プロジェクト構成

```
.
├── sync-service/          # コントロールプレーン APIサーバー
│   ├── src/
│   │   ├── main.rs
│   │   ├── models.rs      # データモデル
│   │   ├── handlers.rs    # APIハンドラー
│   │   ├── db.rs          # データベース操作
│   │   └── sync.rs        # 同期ロジック
│   └── Cargo.toml
├── edge-agent/            # エッジデバイス エージェント
│   ├── src/
│   │   ├── main.rs
│   │   ├── libsql.rs      # libSQL操作
│   │   ├── client.rs      # API クライアント
│   │   ├── sync.rs        # 同期ロジック
│   │   └── retry.rs       # リトライ処理
│   └── Cargo.toml
└── migrations/            # データベースマイグレーション
    └── 001_initial_schema.sql
```

## コア実装

### Sync Service

#### データモデル (models.rs)

```rust
#[derive(Serialize, Deserialize, Clone)]
pub struct TelemetryData {
    pub device_id: String,
    pub timestamp: i64,
    pub data: serde_json::Value,
    pub sync_status: SyncStatus,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub enum SyncStatus {
    Pending,
    Syncing,
    Synced,
    Failed,
}

#[derive(Serialize, Deserialize)]
pub struct Command {
    pub id: String,
    pub device_id: String,
    pub action: String,
    pub params: serde_json::Value,
    pub status: CommandStatus,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub enum CommandStatus {
    Pending,
    Executed,
    Acked,
}
```

#### APIハンドラー (handlers.rs)

```rust
use actix_web::{web, HttpResponse, Result};

pub async fn sync_telemetry(
    data: web::Json<Vec<TelemetryData>>,
    db: web::Data<DbPool>,
) -> Result<HttpResponse> {
    // バッチ同期処理
    for telemetry in data.iter() {
        db.insert_telemetry(telemetry).await?;
    }
    Ok(HttpResponse::Ok().json(json!({"status": "synced"})))
}

pub async fn get_commands(
    device_id: web::Path<String>,
    db: web::Data<DbPool>,
) -> Result<HttpResponse> {
    let commands = db.get_pending_commands(&device_id).await?;
    Ok(HttpResponse::Ok().json(commands))
}

pub async fn ack_command(
    command_id: web::Path<String>,
    db: web::Data<DbPool>,
) -> Result<HttpResponse> {
    db.update_command_status(&command_id, CommandStatus::Acked).await?;
    Ok(HttpResponse::Ok().json(json!({"status": "acked"})))
}

pub async fn register_device(
    device: web::Json<DeviceInfo>,
    db: web::Data<DbPool>,
) -> Result<HttpResponse> {
    db.register_device(&device).await?;
    Ok(HttpResponse::Created().json(device.into_inner()))
}

pub async fn get_sync_status(
    device_id: web::Path<String>,
    db: web::Data<DbPool>,
) -> Result<HttpResponse> {
    let status = db.get_sync_status(&device_id).await?;
    Ok(HttpResponse::Ok().json(status))
}
```

#### データベース操作 (db.rs)

```rust
use sqlx::PgPool;

pub struct DbPool {
    pool: PgPool,
}

impl DbPool {
    pub async fn insert_telemetry(&self, data: &TelemetryData) -> Result<()> {
        sqlx::query(
            "INSERT INTO telemetry (device_id, timestamp, data, sync_status) 
             VALUES ($1, $2, $3, $4)
             ON CONFLICT (device_id, timestamp) DO UPDATE SET 
             data = $3, sync_status = $4"
        )
        .bind(&data.device_id)
        .bind(data.timestamp)
        .bind(&data.data)
        .bind(data.sync_status.to_string())
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn get_pending_commands(&self, device_id: &str) -> Result<Vec<Command>> {
        let commands = sqlx::query_as::<_, Command>(
            "SELECT id, device_id, action, params, status FROM commands 
             WHERE device_id = $1 AND status = 'Pending'"
        )
        .bind(device_id)
        .fetch_all(&self.pool)
        .await?;
        Ok(commands)
    }

    pub async fn update_command_status(
        &self,
        command_id: &str,
        status: CommandStatus,
    ) -> Result<()> {
        sqlx::query("UPDATE commands SET status = $1 WHERE id = $2")
            .bind(status.to_string())
            .bind(command_id)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    pub async fn register_device(&self, device: &DeviceInfo) -> Result<()> {
        sqlx::query(
            "INSERT INTO devices (device_id, name, status) VALUES ($1, $2, $3)"
        )
        .bind(&device.device_id)
        .bind(&device.name)
        .bind("active")
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn get_sync_status(&self, device_id: &str) -> Result<SyncStatusResponse> {
        let result = sqlx::query_as::<_, SyncStatusResponse>(
            "SELECT device_id, COUNT(*) as total, 
                    SUM(CASE WHEN sync_status = 'Synced' THEN 1 ELSE 0 END) as synced,
                    SUM(CASE WHEN sync_status = 'Failed' THEN 1 ELSE 0 END) as failed
             FROM telemetry WHERE device_id = $1 GROUP BY device_id"
        )
        .bind(device_id)
        .fetch_one(&self.pool)
        .await?;
        Ok(result)
    }
}
```

### Edge Agent

#### libSQL操作 (libsql.rs)

```rust
use libsql::Connection;

pub struct LibSqlClient {
    conn: Connection,
}

impl LibSqlClient {
    pub async fn new(db_path: &str) -> Result<Self> {
        let conn = libsql::open(db_path).await?;
        Ok(Self { conn })
    }

    pub async fn get_pending_telemetry(&self, limit: usize) -> Result<Vec<TelemetryData>> {
        let rows = self.conn
            .query(
                "SELECT device_id, timestamp, data FROM telemetry 
                 WHERE sync_status = 'pending' LIMIT ?",
                [limit as i32],
            )
            .await?;

        let mut data = Vec::new();
        for row in rows {
            data.push(TelemetryData {
                device_id: row.get(0)?,
                timestamp: row.get(1)?,
                data: serde_json::from_str(&row.get::<String>(2)?)?,
                sync_status: SyncStatus::Pending,
            });
        }
        Ok(data)
    }

    pub async fn mark_synced(&self, device_id: &str, timestamp: i64) -> Result<()> {
        self.conn
            .execute(
                "UPDATE telemetry SET sync_status = 'synced' 
                 WHERE device_id = ? AND timestamp = ?",
                [device_id, &timestamp.to_string()],
            )
            .await?;
        Ok(())
    }

    pub async fn execute_command(&self, command: &Command) -> Result<()> {
        self.conn
            .execute(
                "INSERT INTO command_log (command_id, action, params, executed_at) 
                 VALUES (?, ?, ?, ?)",
                [&command.id, &command.action, &command.params.to_string(), &chrono::Utc::now().to_rfc3339()],
            )
            .await?;
        Ok(())
    }
}
```

#### APIクライアント (client.rs)

```rust
use reqwest::Client;

pub struct SyncClient {
    client: Client,
    api_url: String,
    device_id: String,
}

impl SyncClient {
    pub fn new(api_url: String, device_id: String) -> Self {
        Self {
            client: Client::new(),
            api_url,
            device_id,
        }
    }

    pub async fn sync_telemetry(&self, data: Vec<TelemetryData>) -> Result<()> {
        self.client
            .post(&format!("{}/api/v1/sync/telemetry", self.api_url))
            .json(&data)
            .send()
            .await?;
        Ok(())
    }

    pub async fn get_commands(&self) -> Result<Vec<Command>> {
        let response = self.client
            .get(&format!("{}/api/v1/sync/commands/{}", self.api_url, self.device_id))
            .send()
            .await?;
        Ok(response.json().await?)
    }

    pub async fn ack_command(&self, command_id: &str) -> Result<()> {
        self.client
            .post(&format!("{}/api/v1/sync/ack/{}", self.api_url, command_id))
            .send()
            .await?;
        Ok(())
    }

    pub async fn register(&self, name: &str) -> Result<()> {
        self.client
            .post(&format!("{}/api/v1/devices/register", self.api_url))
            .json(&json!({
                "device_id": self.device_id,
                "name": name
            }))
            .send()
            .await?;
        Ok(())
    }
}
```

#### リトライ処理 (retry.rs)

```rust
pub struct RetryConfig {
    pub max_retries: u32,
    pub initial_delay_ms: u64,
    pub max_delay_ms: u64,
}

impl Default for RetryConfig {
    fn default() -> Self {
        Self {
            max_retries: 5,
            initial_delay_ms: 1000,
            max_delay_ms: 32000,
        }
    }
}

pub async fn retry_with_backoff<F, T, E>(
    mut f: F,
    config: RetryConfig,
) -> Result<T, E>
where
    F: FnMut() -> futures::future::BoxFuture<'static, Result<T, E>>,
    E: std::fmt::Debug,
{
    let mut delay = config.initial_delay_ms;
    for attempt in 0..config.max_retries {
        match f().await {
            Ok(result) => return Ok(result),
            Err(e) => {
                if attempt == config.max_retries - 1 {
                    return Err(e);
                }
                tokio::time::sleep(tokio::time::Duration::from_millis(delay)).await;
                delay = (delay * 2).min(config.max_delay_ms);
            }
        }
    }
    unreachable!()
}
```

#### メインロジック (main.rs)

```rust
#[tokio::main]
async fn main() -> Result<()> {
    let device_id = std::env::var("DEVICE_ID")
        .unwrap_or_else(|_| uuid::Uuid::new_v4().to_string());
    let api_url = std::env::var("API_URL")
        .unwrap_or_else(|_| "http://localhost:8080".to_string());

    let libsql = LibSqlClient::new("./edge.db").await?;
    let client = SyncClient::new(api_url, device_id.clone());

    // デバイス登録
    client.register("edge-device").await?;

    loop {
        // アップストリーム同期
        if let Ok(telemetry) = libsql.get_pending_telemetry(1000).await {
            if !telemetry.is_empty() {
                match retry_with_backoff(
                    || Box::pin(client.sync_telemetry(telemetry.clone())),
                    RetryConfig::default(),
                ).await {
                    Ok(_) => {
                        for t in &telemetry {
                            let _ = libsql.mark_synced(&t.device_id, t.timestamp).await;
                        }
                    }
                    Err(e) => eprintln!("Sync failed: {:?}", e),
                }
            }
        }

        // ダウンストリーム同期
        if let Ok(commands) = client.get_commands().await {
            for command in commands {
                if let Ok(_) = libsql.execute_command(&command).await {
                    let _ = client.ack_command(&command.id).await;
                }
            }
        }

        tokio::time::sleep(tokio::time::Duration::from_secs(5)).await;
    }
}
```

## データベーススキーマ

```sql
CREATE TABLE devices (
    device_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255),
    status VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE telemetry (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(255) REFERENCES devices(device_id),
    timestamp BIGINT,
    data JSONB,
    sync_status VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(device_id, timestamp)
);

CREATE TABLE commands (
    id VARCHAR(255) PRIMARY KEY,
    device_id VARCHAR(255) REFERENCES devices(device_id),
    action VARCHAR(255),
    params JSONB,
    status VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE command_log (
    id SERIAL PRIMARY KEY,
    command_id VARCHAR(255) REFERENCES commands(id),
    action VARCHAR(255),
    params JSONB,
    executed_at TIMESTAMP
);

CREATE INDEX idx_telemetry_device_status ON telemetry(device_id, sync_status);
CREATE INDEX idx_commands_device_status ON commands(device_id, status);
```

## 開発

```bash
# ビルド
cargo build

# テスト
cargo test

# フォーマット
cargo fmt

# Lint
cargo clippy
```
