mod handlers;
mod models;
mod repository;

use anyhow::Result;
use axum::{
    routing::{get, post},
    Router,
};
use repository::Repository;
use sqlx::postgres::PgPoolOptions;
use std::sync::Arc;
use tower_http::trace::TraceLayer;
use tracing::info;

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt::init();

    let database_url = std::env::var("DATABASE_URL")
        .unwrap_or_else(|_| "postgresql://localhost/iot_sync".to_string());

    let pool = PgPoolOptions::new()
        .max_connections(10)
        .connect(&database_url)
        .await?;

    info!("Connected to database");

    let repo = Arc::new(Repository::new(pool));

    let app = Router::new()
        .route("/health", get(handlers::health_check))
        .route("/api/v1/sync/telemetry", post(handlers::sync_telemetry))
        .route("/api/v1/sync/commands/:device_id", get(handlers::get_commands))
        .route("/api/v1/sync/ack/:command_id", post(handlers::command_ack))
        .route("/api/v1/sync/status/:device_id", get(handlers::get_sync_status))
        .route("/api/v1/devices/register", post(handlers::register_device))
        .layer(TraceLayer::new_for_http())
        .with_state(repo);

    let addr = "0.0.0.0:8080";
    info!("Starting server on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}
