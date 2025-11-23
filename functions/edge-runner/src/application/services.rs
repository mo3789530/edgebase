use crate::domain::*;
use crate::infrastructure::*;
use std::sync::Arc;
use std::collections::HashMap;

pub struct FunctionService {
    function_repo: Arc<dyn FunctionRepository>,
    route_repo: Arc<dyn RouteRepository>,
    cache_repo: Arc<dyn CacheRepository>,
    _pool: Arc<HotInstancePool>,
}

impl FunctionService {
    pub fn new(
        function_repo: Arc<dyn FunctionRepository>,
        route_repo: Arc<dyn RouteRepository>,
        cache_repo: Arc<dyn CacheRepository>,
        pool: Arc<HotInstancePool>,
    ) -> Self {
        Self {
            function_repo,
            route_repo,
            cache_repo,
            _pool: pool,
        }
    }
    
    pub async fn register_function(&self, metadata: FunctionMetadata) {
        self.function_repo.register(metadata).await;
    }
    
    pub async fn add_route(&self, route: Route) {
        self.route_repo.add_route(route).await;
    }
    
    pub async fn resolve_function(&self, host: &str, path: &str, method: &str) -> Option<(FunctionMetadata, HashMap<String, String>)> {
        let route_match = self.route_repo.match_route(host, path, method).await?;
        let metadata = self.function_repo.get(&route_match.function_id).await?;
        Some((metadata, route_match.path_params))
    }
    
    pub async fn get_cached_functions(&self) -> Vec<CachedFunction> {
        self.cache_repo.get_cached().await
    }
}

pub struct HeartbeatService {
    cp_client: Arc<ControlPlaneClient>,
    function_service: Arc<FunctionService>,
    cache_repo: Arc<dyn CacheRepository>,
    wasm_cache: Arc<crate::infrastructure::LocalWasmCache>,
}

impl HeartbeatService {
    pub fn new(
        cp_client: Arc<ControlPlaneClient>,
        function_service: Arc<FunctionService>,
        cache_repo: Arc<dyn CacheRepository>,
        wasm_cache: Arc<crate::infrastructure::LocalWasmCache>,
    ) -> Self {
        Self {
            cp_client,
            function_service,
            cache_repo,
            wasm_cache,
        }
    }
    
    pub async fn send_heartbeat(&self, node_info: &NodeInfo) -> Result<(Vec<DeploymentNotification>, Vec<crate::infrastructure::RouteDto>), String> {
        let cached = self.function_service.get_cached_functions().await;
        self.cp_client.send_heartbeat(&node_info.node_id, &node_info.pop_id, cached).await
    }
    
    pub async fn handle_deployments(&self, deployments: Vec<DeploymentNotification>) {
        for deployment in deployments {
            let metadata = FunctionMetadata {
                function_id: deployment.function_id.clone(),
                version: deployment.version.clone(),
                artifact_url: deployment.artifact_url.clone(),
                sha256: deployment.sha256.clone(),
                memory_pages: deployment.memory_pages as u32,
                max_execution_ms: deployment.max_execution_ms as u32,
            };
            
            self.function_service.register_function(metadata).await;
            
            if let Ok(artifact_data) = download_artifact(&deployment.artifact_url).await {
                let _ = self.wasm_cache.put(
                    &deployment.function_id,
                    &deployment.version,
                    &artifact_data,
                    &deployment.sha256,
                ).await;
            }
            
            self.cache_repo.add_cached(CachedFunction {
                function_id: deployment.function_id,
                version: deployment.version,
                state: "cached".to_string(),
            }).await;
        }
    }
    
    pub async fn handle_routes(&self, routes: Vec<crate::infrastructure::RouteDto>) {
        for route_dto in routes {
            let route = Route {
                id: route_dto.id,
                host: route_dto.host,
                path: route_dto.path,
                function_id: route_dto.function_id,
                methods: route_dto.methods,
                priority: route_dto.priority,
            };
            
            self.function_service.add_route(route).await;
        }
    }
}

async fn download_artifact(url: &str) -> Result<Vec<u8>, String> {
    let client = reqwest::Client::new();
    let resp = client.get(url)
        .send()
        .await
        .map_err(|e| format!("Download failed: {}", e))?;
    
    resp.bytes()
        .await
        .map(|b| b.to_vec())
        .map_err(|e| format!("Failed to read response: {}", e))
}

pub struct InvocationService {
    function_service: Arc<FunctionService>,
    pool: Arc<HotInstancePool>,
    wasm_bytes: Vec<u8>,
    cache: Arc<crate::infrastructure::LocalWasmCache>,
}

impl InvocationService {
    pub fn new(
        function_service: Arc<FunctionService>,
        pool: Arc<HotInstancePool>,
        wasm_bytes: Vec<u8>,
        cache: Arc<crate::infrastructure::LocalWasmCache>,
    ) -> Self {
        Self {
            function_service,
            pool,
            wasm_bytes,
            cache,
        }
    }
    
    pub async fn invoke(&self, host: &str, path: &str, method: &str) -> Result<Vec<u8>, String> {
        let (metadata, _path_params) = self.function_service.resolve_function(host, path, method)
            .await
            .ok_or_else(|| "Route not found".to_string())?;
        
        let wasm_bytes = if let Some(cached) = self.cache.get(&metadata.function_id, &metadata.version, &metadata.sha256).await {
            cached
        } else {
            self.wasm_bytes.clone()
        };
        
        let mut pooled = self.pool.get_or_create(
            &metadata.function_id,
            &wasm_bytes,
            metadata.memory_pages,
        ).await?;
        
        let result = execute_wasm(&mut pooled, method, path)?;
        
        self.pool.return_instance(&metadata.function_id, pooled).await;
        
        Ok(result)
    }
}

fn execute_wasm(pooled: &mut PooledInstance, method: &str, path: &str) -> Result<Vec<u8>, String> {
    let handle = pooled.instance.exports.get_function("handle")
        .map_err(|_| "handle function not found".to_string())?;
    
    let memory = pooled.instance.exports.get_memory("memory")
        .map_err(|_| "memory not found".to_string())?;
    
    let method_bytes = method.as_bytes();
    let path_bytes = path.as_bytes();
    let response_buf = vec![0u8; 4096];
    
    {
        let mem_view = memory.view(&pooled.store);
        let _ = mem_view.write(0, method_bytes);
        let _ = mem_view.write(256, path_bytes);
        let _ = mem_view.write(1024, &response_buf);
    }
    
    let result = handle.call(&mut pooled.store, &[
        0i32.into(), (method_bytes.len() as i32).into(),
        256i32.into(), (path_bytes.len() as i32).into(),
        512i32.into(), 0i32.into(),
        768i32.into(), 0i32.into(),
        1024i32.into(), (response_buf.len() as i32).into(),
    ]).map_err(|e| format!("WASM error: {}", e))?;
    
    let len = result[0].i32().unwrap_or(0) as usize;
    let mut response_data = vec![0u8; len];
    let mem_view = memory.view(&pooled.store);
    let _ = mem_view.read(1024, &mut response_data);
    
    Ok(response_data)
}
