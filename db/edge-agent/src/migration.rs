use crate::db::Database;
use anyhow::{Context, Result};
use serde::Deserialize;
use tracing::{error, info};

#[derive(Debug, Deserialize)]
pub struct SchemaMigration {
    pub version: i32,
    pub description: String,
    pub created_at: String,
}

#[derive(Deserialize)]
struct SchemaResponse {
    data: Vec<SchemaMigration>,
}

pub struct MigrationManager {
    api_url: String,
    device_id: String,
    client: reqwest::Client,
}

impl MigrationManager {
    pub fn new(api_url: String, device_id: String) -> Self {
        let client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(30))
            .build()
            .expect("Failed to create HTTP client");

        Self { api_url, device_id, client }
    }

    pub async fn check_and_apply(&self, db: &Database) -> Result<()> {
        let current_version = db.get_current_version().context("Failed to get current DB version")?;
        info!("Current DB version: {}", current_version);

        let schemas = self.fetch_available_schemas().await.context("Failed to fetch schemas")?;
        
        let mut to_apply: Vec<SchemaMigration> = schemas.into_iter()
            .filter(|s| s.version > current_version)
            .collect();
            
        // Sort by version ascending
        to_apply.sort_by_key(|s| s.version);

        if to_apply.is_empty() {
            info!("No new migrations to apply.");
            return Ok(());
        }

        info!("Found {} new migrations.", to_apply.len());

        for schema in to_apply {
            info!("Applying migration version: {}", schema.version);
            match self.apply_single_migration(db, &schema).await {
                Ok(_) => {
                    info!("Successfully applied migration version: {}", schema.version);
                    self.report_status(schema.version, "synced", None).await;
                }
                Err(e) => {
                    error!("Failed to apply migration version {}: {}", schema.version, e);
                    self.report_status(schema.version, "failed", Some(e.to_string())).await;
                    return Err(e); // Stop on failure
                }
            }
        }

        Ok(())
    }

    async fn fetch_available_schemas(&self) -> Result<Vec<SchemaMigration>> {
        let url = format!("{}/api/v1/schemas?limit=100", self.api_url); // Fetch up to 100 schemas
        
        let resp = self.client.get(&url).send().await?;
        if !resp.status().is_success() {
            anyhow::bail!("Failed to fetch schemas: status {}", resp.status());
        }
        
        let wrapper: SchemaResponse = resp.json().await?;
        Ok(wrapper.data)
    }

    async fn apply_single_migration(&self, db: &Database, schema: &SchemaMigration) -> Result<()> {
        // Download SQL
        let url = format!("{}/api/v1/schemas/{}/download", self.api_url, schema.version);
        let resp = self.client.get(&url).send().await?;
        
        if !resp.status().is_success() {
            anyhow::bail!("Failed to download SQL for version {}: status {}", schema.version, resp.status());
        }

        let sql = resp.text().await?;
        
        // Apply
        // Note: rusqlite operations are blocking. In a high-throughput async app we might want spawn_blocking.
        // For this agent, blocking briefly during migration (which is rare) is acceptable.
        db.apply_migration(schema.version, &sql).context("Failed to execute migration SQL")?;
        
        Ok(())
    }

    async fn report_status(&self, version: i32, status: &str, error: Option<String>) {
        let url = format!("{}/api/v1/nodes/{}/schema_status", self.api_url, self.device_id);
        let payload = serde_json::json!({
            "version": version,
            "status": status,
            "error_message": error.unwrap_or_default()
        });
        
        if let Err(e) = self.client.post(&url).json(&payload).send().await {
             error!("Failed to report schema status: {}", e);
        }
    }
}

