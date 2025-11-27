mod domain;
mod application;
mod infrastructure;
mod presentation;
mod config;

use axum::{Router, routing::{any, get}, extract::State, http::Request, body::Body, response::IntoResponse};
use std::sync::Arc;
use std::time::Duration;

use config::Config;
use domain::NodeInfo;
use infrastructure::{
    InMemoryFunctionRepository, InMemoryRouteRepository, InMemoryCacheRepository,
    HotInstancePool, ControlPlaneClient, LocalWasmCache,
};
use application::{FunctionService, HeartbeatService, InvocationService};
use presentation::HttpHandler;

struct AppState {
    http_handler: Arc<HttpHandler>,
    heartbeat_service: Arc<HeartbeatService>,
    _node_info: NodeInfo,
}

#[tokio::main]
async fn main() {
    let config = Config::from_env();
    
    let wasm_path = std::env::args().nth(1).expect("Usage: edge-runner <wasm_file>");
    let wasm_bytes = std::fs::read(&wasm_path).expect("Failed to read WASM file");
    
    let node_info = NodeInfo {
        node_id: config.node_id.clone(),
        pop_id: config.pop_id.clone(),
        cp_url: config.cp_url.clone(),
    };
    
    // Initialize repositories
    let function_repo = Arc::new(InMemoryFunctionRepository::new());
    let route_repo = Arc::new(InMemoryRouteRepository::new());
    let cache_repo = Arc::new(InMemoryCacheRepository::new());
    
    // Initialize cache
    let cache_size_bytes = config.cache_size_gb * 1024 * 1024 * 1024;
    let wasm_cache = Arc::new(LocalWasmCache::new(&config.cache_dir, cache_size_bytes)
        .unwrap_or_else(|_| LocalWasmCache::new("/tmp/wasm-cache", cache_size_bytes).unwrap()));
    
    // Initialize pool
    let pool = Arc::new(HotInstancePool::new(config.max_hot_instances, config.idle_timeout_secs));
    
    // Initialize services
    let function_service = Arc::new(FunctionService::new(
        function_repo,
        route_repo,
        cache_repo.clone(),
        pool.clone(),
    ));
    
    let cp_client = Arc::new(ControlPlaneClient::new(config.cp_url.clone()));
    let heartbeat_service = Arc::new(HeartbeatService::new(
        cp_client,
        function_service.clone(),
        cache_repo,
        wasm_cache.clone(),
    ));
    
    let invocation_service = Arc::new(InvocationService::new(
        function_service,
        pool,
        wasm_bytes,
        wasm_cache,
    ));
    
    let http_handler = Arc::new(HttpHandler::new(invocation_service));
    
    let state = AppState {
        http_handler,
        heartbeat_service,
        _node_info: node_info.clone(),
    };
    
    let state_arc = Arc::new(state);
    
    // Start heartbeat task
    let heartbeat_state = state_arc.clone();
    let node_info_clone = node_info.clone();
    let heartbeat_interval = Duration::from_secs(config.heartbeat_interval_secs);
    tokio::spawn(async move {
        let mut interval = tokio::time::interval(heartbeat_interval);
        loop {
            interval.tick().await;
            match heartbeat_state.heartbeat_service.send_heartbeat(&node_info_clone).await {
                Ok((deployments, routes)) => {
                    heartbeat_state.heartbeat_service.handle_deployments(deployments).await;
                    heartbeat_state.heartbeat_service.handle_routes(routes).await;
                }
                Err(e) => eprintln!("Heartbeat error: {}", e),
            }
        }
    });
    
    let app = Router::new()
        .route("/metrics", get(presentation::metrics_handler))
        .route("/*path", any(handler))
        .with_state(state_arc);
    
    let bind_addr = format!("{}:{}", config.listen_addr, config.listen_port);
    let listener = tokio::net::TcpListener::bind(&bind_addr).await.unwrap();
    println!("Edge Runner listening on http://{} (node_id: {})", bind_addr, config.node_id);
    axum::serve(listener, app).await.unwrap();
}

async fn handler(
    State(state): State<Arc<AppState>>,
    req: Request<Body>,
) -> impl IntoResponse {
    state.http_handler.handle_request(req).await
}
