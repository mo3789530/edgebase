use crate::domain::models::*;
use async_trait::async_trait;

#[async_trait]
pub trait FunctionRepository: Send + Sync {
    async fn register(&self, metadata: FunctionMetadata);
    async fn get(&self, function_id: &str) -> Option<FunctionMetadata>;
    #[allow(dead_code)]
    async fn remove(&self, function_id: &str);
}

#[async_trait]
pub trait RouteRepository: Send + Sync {
    async fn add_route(&self, route: Route);
    async fn match_route(&self, host: &str, path: &str, method: &str) -> Option<RouteMatch>;
    #[allow(dead_code)]
    async fn list_routes(&self) -> Vec<Route>;
}

#[async_trait]
pub trait CacheRepository: Send + Sync {
    async fn get_cached(&self) -> Vec<CachedFunction>;
    async fn add_cached(&self, func: CachedFunction);
    #[allow(dead_code)]
    async fn clear_cached(&self);
}
