use axum::{Router, routing::{get, post}, extract::{State, Path, Multipart}, http::StatusCode, Json, response::IntoResponse};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tokio::sync::RwLock;
use std::collections::HashMap;
use uuid::Uuid;
use sha2::{Sha256, Digest};
use chrono::{DateTime, Utc};

#[derive(Clone, Serialize, Deserialize)]
struct Function {
    id: String,
    name: String,
    version: String,
    runtime: String,
    entrypoint: String,
    artifact_url: Option<String>,
    sha256: Option<String>,
    memory_pages: i32,
    max_execution_ms: i32,
}

#[derive(Clone, Serialize, Deserialize)]
struct Node {
    id: String,
    pop_id: String,
    last_heartbeat: Option<DateTime<Utc>>,
    status: String,
    cached_functions: Vec<CachedFunction>,
}

#[derive(Clone, Serialize, Deserialize)]
struct CachedFunction {
    function_id: String,
    version: String,
    state: String,
}

#[derive(Deserialize)]
struct HeartbeatRequest {
    node_id: String,
    pop_id: String,
    status: String,
    cached_functions: Vec<CachedFunction>,
}

#[derive(Serialize)]
struct HeartbeatResponse {
    deployments: Vec<DeploymentNotification>,
    routes: Vec<Route>,
}

#[derive(Clone, Serialize, Deserialize)]
struct DeploymentNotification {
    function_id: String,
    version: String,
    artifact_url: String,
    sha256: String,
    memory_pages: i32,
    max_execution_ms: i32,
}

#[derive(Deserialize)]
struct CreateFunctionRequest {
    name: String,
    entrypoint: String,
    runtime: String,
    memory_pages: i32,
    max_execution_ms: i32,
}

#[derive(Serialize)]
struct CreateFunctionResponse {
    function: Function,
}

#[derive(Clone, Serialize, Deserialize)]
struct Route {
    id: String,
    host: String,
    path: String,
    function_id: String,
    methods: Vec<String>,
    priority: i32,
    pop_selector: Option<String>,
}

#[derive(Deserialize)]
struct CreateRouteRequest {
    host: String,
    path: String,
    function_id: String,
    methods: Vec<String>,
    priority: i32,
    pop_selector: Option<String>,
}

#[derive(Serialize)]
struct CreateRouteResponse {
    route: Route,
}

struct AppState {
    functions: RwLock<HashMap<String, Function>>,
    artifacts: RwLock<HashMap<String, Vec<u8>>>,
    nodes: RwLock<HashMap<String, Node>>,
    pending_deployments: RwLock<HashMap<String, Vec<DeploymentNotification>>>,
    routes: RwLock<Vec<Route>>,
}

#[tokio::main]
async fn main() {
    let state = Arc::new(AppState {
        functions: RwLock::new(HashMap::new()),
        artifacts: RwLock::new(HashMap::new()),
        nodes: RwLock::new(HashMap::new()),
        pending_deployments: RwLock::new(HashMap::new()),
        routes: RwLock::new(Vec::new()),
    });
    
    let app = Router::new()
        .route("/api/v1/functions", post(create_function))
        .route("/api/v1/functions/:id", get(get_function))
        .route("/api/v1/functions/:id/upload", post(upload_artifact))
        .route("/api/v1/artifacts/:id/:version", get(download_artifact))
        .route("/api/v1/nodes/:id/heartbeat", post(heartbeat))
        .route("/api/v1/functions/:function_id/deploy/:node_id", post(deploy_function))
        .route("/api/v1/routes", post(create_route))
        .route("/api/v1/routes", get(list_routes))
        .with_state(state);
    
    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080").await.unwrap();
    println!("Control Plane listening on http://0.0.0.0:8080");
    axum::serve(listener, app).await.unwrap();
}

async fn create_function(
    State(state): State<Arc<AppState>>,
    Json(req): Json<CreateFunctionRequest>,
) -> impl IntoResponse {
    let id = Uuid::new_v4().to_string();
    let function = Function {
        id: id.clone(),
        name: req.name,
        version: "1.0.0".to_string(),
        runtime: req.runtime,
        entrypoint: req.entrypoint,
        artifact_url: None,
        sha256: None,
        memory_pages: req.memory_pages,
        max_execution_ms: req.max_execution_ms,
    };
    
    state.functions.write().await.insert(id.clone(), function.clone());
    
    Json(CreateFunctionResponse { function })
}

async fn get_function(
    State(state): State<Arc<AppState>>,
    Path(id): Path<String>,
) -> impl IntoResponse {
    let functions = state.functions.read().await;
    match functions.get(&id) {
        Some(func) => (StatusCode::OK, Json(func.clone())).into_response(),
        None => StatusCode::NOT_FOUND.into_response(),
    }
}

async fn upload_artifact(
    State(state): State<Arc<AppState>>,
    Path(id): Path<String>,
    mut multipart: Multipart,
) -> impl IntoResponse {
    while let Some(field) = multipart.next_field().await.unwrap() {
        if field.name() == Some("file") {
            let data = field.bytes().await.unwrap();
            
            let mut hasher = Sha256::new();
            hasher.update(&data);
            let sha256 = format!("{:x}", hasher.finalize());
            
            let mut functions = state.functions.write().await;
            if let Some(func) = functions.get_mut(&id) {
                let artifact_key = format!("{}/{}", id, func.version);
                func.artifact_url = Some(format!("/api/v1/artifacts/{}", artifact_key));
                func.sha256 = Some(sha256.clone());
                
                state.artifacts.write().await.insert(artifact_key, data.to_vec());
                
                return (StatusCode::OK, Json(func.clone())).into_response();
            }
        }
    }
    
    StatusCode::BAD_REQUEST.into_response()
}

async fn download_artifact(
    State(state): State<Arc<AppState>>,
    Path((id, version)): Path<(String, String)>,
) -> impl IntoResponse {
    let artifact_key = format!("{}/{}", id, version);
    let artifacts = state.artifacts.read().await;
    
    match artifacts.get(&artifact_key) {
        Some(data) => (StatusCode::OK, data.clone()).into_response(),
        None => StatusCode::NOT_FOUND.into_response(),
    }
}

async fn heartbeat(
    State(state): State<Arc<AppState>>,
    Path(node_id): Path<String>,
    Json(req): Json<HeartbeatRequest>,
) -> impl IntoResponse {
    let mut nodes = state.nodes.write().await;
    nodes.insert(node_id.clone(), Node {
        id: node_id.clone(),
        pop_id: req.pop_id,
        last_heartbeat: Some(Utc::now()),
        status: req.status,
        cached_functions: req.cached_functions,
    });
    drop(nodes);
    
    let deployments = state.pending_deployments.read().await
        .get(&node_id)
        .cloned()
        .unwrap_or_default();
    
    if !deployments.is_empty() {
        state.pending_deployments.write().await.remove(&node_id);
    }
    
    let routes = state.routes.read().await.clone();
    
    (StatusCode::OK, Json(HeartbeatResponse { deployments, routes })).into_response()
}

async fn deploy_function(
    State(state): State<Arc<AppState>>,
    Path((function_id, node_id)): Path<(String, String)>,
) -> impl IntoResponse {
    let functions = state.functions.read().await;
    
    match functions.get(&function_id) {
        Some(func) => {
            if let (Some(artifact_url), Some(sha256)) = (&func.artifact_url, &func.sha256) {
                let notification = DeploymentNotification {
                    function_id: func.id.clone(),
                    version: func.version.clone(),
                    artifact_url: artifact_url.clone(),
                    sha256: sha256.clone(),
                    memory_pages: func.memory_pages,
                    max_execution_ms: func.max_execution_ms,
                };
                
                let mut deployments = state.pending_deployments.write().await;
                deployments.entry(node_id).or_insert_with(Vec::new).push(notification);
                
                (StatusCode::OK, Json(serde_json::json!({"status": "queued"}))).into_response()
            } else {
                StatusCode::BAD_REQUEST.into_response()
            }
        }
        None => StatusCode::NOT_FOUND.into_response(),
    }
}

async fn create_route(
    State(state): State<Arc<AppState>>,
    Json(req): Json<CreateRouteRequest>,
) -> impl IntoResponse {
    let route = Route {
        id: Uuid::new_v4().to_string(),
        host: req.host,
        path: req.path,
        function_id: req.function_id,
        methods: req.methods,
        priority: req.priority,
        pop_selector: req.pop_selector,
    };
    
    let mut routes = state.routes.write().await;
    routes.push(route.clone());
    routes.sort_by(|a, b| b.priority.cmp(&a.priority));
    
    (StatusCode::CREATED, Json(CreateRouteResponse { route })).into_response()
}

async fn list_routes(
    State(state): State<Arc<AppState>>,
) -> impl IntoResponse {
    let routes = state.routes.read().await;
    Json(routes.clone()).into_response()
}
