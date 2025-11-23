use crate::domain::{CachedFunction, DeploymentNotification};
use serde::{Deserialize, Serialize};

#[derive(Serialize)]
struct HeartbeatRequest {
    node_id: String,
    pop_id: String,
    status: String,
    cached_functions: Vec<CachedFunction>,
}

#[derive(Deserialize)]
struct HeartbeatResponse {
    deployments: Vec<DeploymentNotification>,
    routes: Option<Vec<RouteDto>>,
}

#[derive(Deserialize, Clone)]
pub struct RouteDto {
    pub id: String,
    pub host: String,
    pub path: String,
    pub function_id: String,
    pub methods: Vec<String>,
    pub priority: i32,
}

pub struct ControlPlaneClient {
    cp_url: String,
    client: reqwest::Client,
}

impl ControlPlaneClient {
    pub fn new(cp_url: String) -> Self {
        Self {
            cp_url,
            client: reqwest::Client::new(),
        }
    }
    
    pub async fn send_heartbeat(
        &self,
        node_id: &str,
        pop_id: &str,
        cached_functions: Vec<CachedFunction>,
    ) -> Result<(Vec<DeploymentNotification>, Vec<RouteDto>), String> {
        let req = HeartbeatRequest {
            node_id: node_id.to_string(),
            pop_id: pop_id.to_string(),
            status: "online".to_string(),
            cached_functions,
        };
        
        let url = format!("{}/api/v1/nodes/{}/heartbeat", self.cp_url, node_id);
        
        let resp = self.client.post(&url)
            .json(&req)
            .send()
            .await
            .map_err(|e| format!("Heartbeat failed: {}", e))?;
        
        let hb_resp: HeartbeatResponse = resp.json().await
            .map_err(|e| format!("Failed to parse response: {}", e))?;
        
        let routes = hb_resp.routes.unwrap_or_default();
        Ok((hb_resp.deployments, routes))
    }
}
