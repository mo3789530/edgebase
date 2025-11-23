use axum::{Router, routing::{get, post}, extract::{State, Path, Multipart}, http::StatusCode, Json, response::IntoResponse};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tokio::sync::RwLock;
use std::collections::HashMap;
use uuid::Uuid;
use sha2::{Sha256, Digest};

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

struct AppState {
    functions: RwLock<HashMap<String, Function>>,
    artifacts: RwLock<HashMap<String, Vec<u8>>>,
}

#[tokio::main]
async fn main() {
    let state = Arc::new(AppState {
        functions: RwLock::new(HashMap::new()),
        artifacts: RwLock::new(HashMap::new()),
    });
    
    let app = Router::new()
        .route("/api/v1/functions", post(create_function))
        .route("/api/v1/functions/:id", get(get_function))
        .route("/api/v1/functions/:id/upload", post(upload_artifact))
        .route("/api/v1/artifacts/:id/:version", get(download_artifact))
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
