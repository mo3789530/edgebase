use anyhow::Result;
use edge_agent::{db::Database, sync::SyncAgent, migration::MigrationManager};
use tracing::{info, error};
use uuid::Uuid;

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt::init();

    let db = Database::new("edge.db")?;
    info!("Database initialized");

    let device_id = std::env::var("DEVICE_ID").unwrap_or_else(|_| Uuid::new_v4().to_string());
    let api_url = std::env::var("API_URL").unwrap_or_else(|_| "http://localhost:8080".to_string());

    // Run Migrations
    let migration_mgr = MigrationManager::new(api_url.clone(), device_id.clone());
    if let Err(e) = migration_mgr.check_and_apply(&db).await {
        error!("Failed to apply migrations: {}", e);
        return Err(e);
    }

    let agent = SyncAgent::new(db, api_url, device_id);
    
    agent.run().await?;

    Ok(())
}
