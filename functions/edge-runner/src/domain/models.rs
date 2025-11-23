use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Clone, Serialize, Deserialize)]
pub struct CachedFunction {
    pub function_id: String,
    pub version: String,
    pub state: String,
}

#[derive(Clone, Serialize, Deserialize)]
pub struct FunctionMetadata {
    pub function_id: String,
    pub version: String,
    pub artifact_url: String,
    pub sha256: String,
    pub memory_pages: u32,
    pub max_execution_ms: u32,
}

#[derive(Clone, Serialize, Deserialize)]
pub struct Route {
    pub id: String,
    pub host: String,
    pub path: String,
    pub function_id: String,
    pub methods: Vec<String>,
    pub priority: i32,
}

#[derive(Clone, Serialize, Deserialize)]
pub struct DeploymentNotification {
    pub function_id: String,
    pub version: String,
    pub artifact_url: String,
    pub sha256: String,
    pub memory_pages: i32,
    pub max_execution_ms: i32,
}

#[derive(Clone, Debug)]
pub struct RouteMatch {
    pub function_id: String,
    pub path_params: HashMap<String, String>,
}

pub struct PooledInstance {
    pub instance: wasmer::Instance,
    pub store: wasmer::Store,
    pub last_used: u64,
}

#[derive(Clone)]
pub struct NodeInfo {
    pub node_id: String,
    pub pop_id: String,
    #[allow(dead_code)]
    pub cp_url: String,
}
