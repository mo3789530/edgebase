use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct Function {
    pub id: String,
    pub name: String,
    pub version: u32,
    pub entrypoint: String,
    pub memory_pages: u32,
    pub max_execution_ms: u32,
    pub artifact_url: String,
    pub sha256: String,
    pub created_at: u64,
}

impl Function {
    pub fn new(
        id: String,
        name: String,
        version: u32,
        entrypoint: String,
        memory_pages: u32,
        max_execution_ms: u32,
        artifact_url: String,
        sha256: String,
    ) -> Result<Self, String> {
        Self::validate(&name, memory_pages, max_execution_ms, &entrypoint)?;
        
        let created_at = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();
        
        Ok(Function {
            id,
            name,
            version,
            entrypoint,
            memory_pages,
            max_execution_ms,
            artifact_url,
            sha256,
            created_at,
        })
    }

    fn validate(name: &str, memory_pages: u32, max_execution_ms: u32, entrypoint: &str) -> Result<(), String> {
        if name.is_empty() {
            return Err("Function name cannot be empty".to_string());
        }
        if memory_pages == 0 {
            return Err("Memory pages must be greater than 0".to_string());
        }
        if max_execution_ms == 0 {
            return Err("Max execution time must be greater than 0".to_string());
        }
        if entrypoint.is_empty() {
            return Err("Entrypoint cannot be empty".to_string());
        }
        Ok(())
    }
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct Deployment {
    pub id: String,
    pub function_id: String,
    pub status: DeploymentStatus,
    pub deployed_at: u64,
}

#[derive(Clone, Debug, Serialize, Deserialize, PartialEq)]
pub enum DeploymentStatus {
    Pending,
    Cached,
    Failed,
}

impl Deployment {
    pub fn new(id: String, function_id: String) -> Self {
        let deployed_at = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();
        
        Deployment {
            id,
            function_id,
            status: DeploymentStatus::Pending,
            deployed_at,
        }
    }
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct CacheEntry {
    pub id: String,
    pub function_id: String,
    pub artifact_path: String,
    pub size_bytes: u64,
    pub sha256: String,
    pub last_accessed: u64,
    pub created_at: u64,
}

impl CacheEntry {
    pub fn new(
        id: String,
        function_id: String,
        artifact_path: String,
        size_bytes: u64,
        sha256: String,
    ) -> Self {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();
        
        CacheEntry {
            id,
            function_id,
            artifact_path,
            size_bytes,
            sha256,
            last_accessed: now,
            created_at: now,
        }
    }

    pub fn update_access_time(&mut self) {
        self.last_accessed = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_function_creation() {
        let fn_obj = Function::new(
            "fn1".to_string(),
            "test_fn".to_string(),
            1,
            "main".to_string(),
            256,
            5000,
            "http://example.com/fn.wasm".to_string(),
            "abc123".to_string(),
        ).unwrap();
        
        assert_eq!(fn_obj.name, "test_fn");
        assert_eq!(fn_obj.memory_pages, 256);
    }

    #[test]
    fn test_function_validation_empty_name() {
        let result = Function::new(
            "fn1".to_string(),
            "".to_string(),
            1,
            "main".to_string(),
            256,
            5000,
            "http://example.com/fn.wasm".to_string(),
            "abc123".to_string(),
        );
        assert!(result.is_err());
    }

    #[test]
    fn test_function_validation_zero_memory() {
        let result = Function::new(
            "fn1".to_string(),
            "test_fn".to_string(),
            1,
            "main".to_string(),
            0,
            5000,
            "http://example.com/fn.wasm".to_string(),
            "abc123".to_string(),
        );
        assert!(result.is_err());
    }

    #[test]
    fn test_cache_entry_creation() {
        let entry = CacheEntry::new(
            "ce1".to_string(),
            "fn1".to_string(),
            "/tmp/fn.wasm".to_string(),
            1024,
            "abc123".to_string(),
        );
        
        assert_eq!(entry.function_id, "fn1");
        assert_eq!(entry.size_bytes, 1024);
    }

    #[test]
    fn test_cache_entry_update_access_time() {
        let mut entry = CacheEntry::new(
            "ce1".to_string(),
            "fn1".to_string(),
            "/tmp/fn.wasm".to_string(),
            1024,
            "abc123".to_string(),
        );
        
        let old_time = entry.last_accessed;
        std::thread::sleep(std::time::Duration::from_millis(10));
        entry.update_access_time();
        
        assert!(entry.last_accessed >= old_time);
    }
}
