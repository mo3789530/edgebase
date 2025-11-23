use crate::domain::*;
use std::collections::HashMap;
use tokio::sync::RwLock;
use std::sync::Arc;

pub struct InMemoryFunctionRepository {
    functions: Arc<RwLock<HashMap<String, FunctionMetadata>>>,
}

impl InMemoryFunctionRepository {
    pub fn new() -> Self {
        Self {
            functions: Arc::new(RwLock::new(HashMap::new())),
        }
    }
}

#[async_trait::async_trait]
impl FunctionRepository for InMemoryFunctionRepository {
    async fn register(&self, metadata: FunctionMetadata) {
        self.functions.write().await.insert(metadata.function_id.clone(), metadata);
    }
    
    async fn get(&self, function_id: &str) -> Option<FunctionMetadata> {
        self.functions.read().await.get(function_id).cloned()
    }
    
    async fn remove(&self, function_id: &str) {
        self.functions.write().await.remove(function_id);
    }
}

pub struct InMemoryRouteRepository {
    routes: Arc<RwLock<Vec<Route>>>,
}

impl InMemoryRouteRepository {
    pub fn new() -> Self {
        Self {
            routes: Arc::new(RwLock::new(Vec::new())),
        }
    }
}

#[async_trait::async_trait]
impl RouteRepository for InMemoryRouteRepository {
    async fn add_route(&self, route: Route) {
        let mut routes = self.routes.write().await;
        routes.push(route);
        routes.sort_by(|a, b| b.priority.cmp(&a.priority));
    }
    
    async fn match_route(&self, host: &str, path: &str, method: &str) -> Option<RouteMatch> {
        let routes = self.routes.read().await;
        for route in routes.iter() {
            if (route.host == host || route.host == "*") && path_matches(&route.path, path) {
                if route.methods.contains(&method.to_string()) || route.methods.contains(&"*".to_string()) {
                    let path_params = extract_path_params(&route.path, path);
                    return Some(RouteMatch {
                        function_id: route.function_id.clone(),
                        path_params,
                    });
                }
            }
        }
        None
    }
    
    #[allow(dead_code)]
    async fn list_routes(&self) -> Vec<Route> {
        self.routes.read().await.clone()
    }
}

pub struct InMemoryCacheRepository {
    cached: Arc<RwLock<Vec<CachedFunction>>>,
}

impl InMemoryCacheRepository {
    pub fn new() -> Self {
        Self {
            cached: Arc::new(RwLock::new(Vec::new())),
        }
    }
}

#[async_trait::async_trait]
impl CacheRepository for InMemoryCacheRepository {
    async fn get_cached(&self) -> Vec<CachedFunction> {
        self.cached.read().await.clone()
    }
    
    async fn add_cached(&self, func: CachedFunction) {
        self.cached.write().await.push(func);
    }
    
    #[allow(dead_code)]
    async fn clear_cached(&self) {
        self.cached.write().await.clear();
    }
}

fn path_matches(pattern: &str, path: &str) -> bool {
    if pattern == "*" || pattern == "/*" {
        return true;
    }
    
    // プレフィックスワイルドカード処理
    if pattern.ends_with("/*") {
        let prefix = &pattern[..pattern.len() - 2];
        if prefix.is_empty() {
            return true;
        }
        return path.starts_with(prefix) && (path.len() == prefix.len() || path.chars().nth(prefix.len()) == Some('/'));
    }
    
    let pattern_parts: Vec<&str> = pattern.split('/').filter(|p| !p.is_empty()).collect();
    let path_parts: Vec<&str> = path.split('/').filter(|p| !p.is_empty()).collect();
    
    if pattern_parts.len() != path_parts.len() {
        if pattern.ends_with("*") && !pattern.ends_with("/*") {
            let prefix = &pattern[..pattern.len() - 1];
            return path.starts_with(prefix);
        }
        return false;
    }
    
    for (p_part, path_part) in pattern_parts.iter().zip(path_parts.iter()) {
        if p_part.starts_with(':') {
            continue;
        }
        if p_part != path_part {
            return false;
        }
    }
    true
}

fn extract_path_params(pattern: &str, path: &str) -> HashMap<String, String> {
    let mut params = HashMap::new();
    let pattern_parts: Vec<&str> = pattern.split('/').filter(|p| !p.is_empty()).collect();
    let path_parts: Vec<&str> = path.split('/').filter(|p| !p.is_empty()).collect();
    
    for (p_part, path_part) in pattern_parts.iter().zip(path_parts.iter()) {
        if p_part.starts_with(':') {
            let key = &p_part[1..];
            params.insert(key.to_string(), path_part.to_string());
        }
    }
    params
}
