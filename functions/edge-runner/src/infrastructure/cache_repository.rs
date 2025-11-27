use crate::domain::{Function, Deployment, CacheEntry, DeploymentStatus};
use std::collections::HashMap;
use std::sync::{Arc, RwLock};

pub trait LocalFunctionRepository: Send + Sync {
    fn create(&self, function: Function) -> Result<(), String>;
    fn get(&self, id: &str) -> Result<Option<Function>, String>;
    fn get_by_name(&self, name: &str) -> Result<Option<Function>, String>;
    fn list(&self) -> Result<Vec<Function>, String>;
    fn delete(&self, id: &str) -> Result<(), String>;
}

pub trait LocalDeploymentRepository: Send + Sync {
    fn create(&self, deployment: Deployment) -> Result<(), String>;
    fn get(&self, id: &str) -> Result<Option<Deployment>, String>;
    fn update_status(&self, id: &str, status: DeploymentStatus) -> Result<(), String>;
    fn list_by_function(&self, function_id: &str) -> Result<Vec<Deployment>, String>;
}

pub trait LocalCacheRepository: Send + Sync {
    fn create(&self, entry: CacheEntry) -> Result<(), String>;
    fn get(&self, id: &str) -> Result<Option<CacheEntry>, String>;
    fn get_by_function(&self, function_id: &str) -> Result<Option<CacheEntry>, String>;
    fn update(&self, entry: CacheEntry) -> Result<(), String>;
    fn delete(&self, id: &str) -> Result<(), String>;
    fn list_all(&self) -> Result<Vec<CacheEntry>, String>;
}

pub struct InMemoryLocalFunctionRepository {
    functions: Arc<RwLock<HashMap<String, Function>>>,
}

impl InMemoryLocalFunctionRepository {
    pub fn new() -> Self {
        InMemoryLocalFunctionRepository {
            functions: Arc::new(RwLock::new(HashMap::new())),
        }
    }
}

impl LocalFunctionRepository for InMemoryLocalFunctionRepository {
    fn create(&self, function: Function) -> Result<(), String> {
        let mut funcs = self.functions.write().unwrap();
        if funcs.contains_key(&function.id) {
            return Err("Function already exists".to_string());
        }
        funcs.insert(function.id.clone(), function);
        Ok(())
    }

    fn get(&self, id: &str) -> Result<Option<Function>, String> {
        let funcs = self.functions.read().unwrap();
        Ok(funcs.get(id).cloned())
    }

    fn get_by_name(&self, name: &str) -> Result<Option<Function>, String> {
        let funcs = self.functions.read().unwrap();
        Ok(funcs.values().find(|f| f.name == name).cloned())
    }

    fn list(&self) -> Result<Vec<Function>, String> {
        let funcs = self.functions.read().unwrap();
        Ok(funcs.values().cloned().collect())
    }

    fn delete(&self, id: &str) -> Result<(), String> {
        let mut funcs = self.functions.write().unwrap();
        funcs.remove(id);
        Ok(())
    }
}

pub struct InMemoryLocalDeploymentRepository {
    deployments: Arc<RwLock<HashMap<String, Deployment>>>,
}

impl InMemoryLocalDeploymentRepository {
    pub fn new() -> Self {
        InMemoryLocalDeploymentRepository {
            deployments: Arc::new(RwLock::new(HashMap::new())),
        }
    }
}

impl LocalDeploymentRepository for InMemoryLocalDeploymentRepository {
    fn create(&self, deployment: Deployment) -> Result<(), String> {
        let mut deps = self.deployments.write().unwrap();
        if deps.contains_key(&deployment.id) {
            return Err("Deployment already exists".to_string());
        }
        deps.insert(deployment.id.clone(), deployment);
        Ok(())
    }

    fn get(&self, id: &str) -> Result<Option<Deployment>, String> {
        let deps = self.deployments.read().unwrap();
        Ok(deps.get(id).cloned())
    }

    fn update_status(&self, id: &str, status: DeploymentStatus) -> Result<(), String> {
        let mut deps = self.deployments.write().unwrap();
        if let Some(dep) = deps.get_mut(id) {
            dep.status = status;
            Ok(())
        } else {
            Err("Deployment not found".to_string())
        }
    }

    fn list_by_function(&self, function_id: &str) -> Result<Vec<Deployment>, String> {
        let deps = self.deployments.read().unwrap();
        Ok(deps.values()
            .filter(|d| d.function_id == function_id)
            .cloned()
            .collect())
    }
}

pub struct InMemoryLocalCacheRepository {
    entries: Arc<RwLock<HashMap<String, CacheEntry>>>,
}

impl InMemoryLocalCacheRepository {
    pub fn new() -> Self {
        InMemoryLocalCacheRepository {
            entries: Arc::new(RwLock::new(HashMap::new())),
        }
    }
}

impl LocalCacheRepository for InMemoryLocalCacheRepository {
    fn create(&self, entry: CacheEntry) -> Result<(), String> {
        let mut entries = self.entries.write().unwrap();
        if entries.contains_key(&entry.id) {
            return Err("Cache entry already exists".to_string());
        }
        entries.insert(entry.id.clone(), entry);
        Ok(())
    }

    fn get(&self, id: &str) -> Result<Option<CacheEntry>, String> {
        let entries = self.entries.read().unwrap();
        Ok(entries.get(id).cloned())
    }

    fn get_by_function(&self, function_id: &str) -> Result<Option<CacheEntry>, String> {
        let entries = self.entries.read().unwrap();
        Ok(entries.values()
            .find(|e| e.function_id == function_id)
            .cloned())
    }

    fn update(&self, entry: CacheEntry) -> Result<(), String> {
        let mut entries = self.entries.write().unwrap();
        if !entries.contains_key(&entry.id) {
            return Err("Cache entry not found".to_string());
        }
        entries.insert(entry.id.clone(), entry);
        Ok(())
    }

    fn delete(&self, id: &str) -> Result<(), String> {
        let mut entries = self.entries.write().unwrap();
        entries.remove(id);
        Ok(())
    }

    fn list_all(&self) -> Result<Vec<CacheEntry>, String> {
        let entries = self.entries.read().unwrap();
        Ok(entries.values().cloned().collect())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_function_repository_create_and_get() {
        let repo = InMemoryLocalFunctionRepository::new();
        let fn_obj = Function::new(
            "fn1".to_string(),
            "test".to_string(),
            1,
            "main".to_string(),
            256,
            5000,
            "http://example.com/fn.wasm".to_string(),
            "abc123".to_string(),
        ).unwrap();
        
        repo.create(fn_obj.clone()).unwrap();
        let retrieved = repo.get("fn1").unwrap();
        assert!(retrieved.is_some());
        assert_eq!(retrieved.unwrap().name, "test");
    }

    #[test]
    fn test_cache_repository_crud() {
        let repo = InMemoryLocalCacheRepository::new();
        let entry = CacheEntry::new(
            "ce1".to_string(),
            "fn1".to_string(),
            "/tmp/fn.wasm".to_string(),
            1024,
            "abc123".to_string(),
        );
        
        repo.create(entry.clone()).unwrap();
        let retrieved = repo.get("ce1").unwrap();
        assert!(retrieved.is_some());
        
        repo.delete("ce1").unwrap();
        let deleted = repo.get("ce1").unwrap();
        assert!(deleted.is_none());
    }

    #[test]
    fn test_deployment_repository_status_update() {
        let repo = InMemoryLocalDeploymentRepository::new();
        let dep = Deployment::new("dep1".to_string(), "fn1".to_string());
        
        repo.create(dep).unwrap();
        repo.update_status("dep1", DeploymentStatus::Cached).unwrap();
        
        let retrieved = repo.get("dep1").unwrap().unwrap();
        assert_eq!(retrieved.status, DeploymentStatus::Cached);
    }
}

