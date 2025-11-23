use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize)]
pub struct HeartbeatRequest {
    pub node_id: String,
    pub pop_id: String,
    pub status: String,
    pub cached_functions: Vec<CachedFunctionDto>,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct CachedFunctionDto {
    pub function_id: String,
    pub version: String,
    pub state: String,
}

#[derive(Deserialize)]
pub struct HeartbeatResponse {
    #[allow(dead_code)]
    pub deployments: Vec<DeploymentNotificationDto>,
}

#[derive(Deserialize, Clone)]
pub struct DeploymentNotificationDto {
    #[allow(dead_code)]
    pub function_id: String,
    #[allow(dead_code)]
    pub version: String,
    #[allow(dead_code)]
    pub artifact_url: String,
    #[allow(dead_code)]
    pub sha256: String,
    #[allow(dead_code)]
    pub memory_pages: i32,
    #[allow(dead_code)]
    pub max_execution_ms: i32,
}

#[derive(Serialize)]
pub struct InvocationRequest {
    pub method: String,
    pub path: String,
    pub host: String,
}

#[derive(Serialize)]
pub struct InvocationResponse {
    pub status_code: u16,
    pub body: Vec<u8>,
}
