use std::collections::HashMap;
use std::sync::{Arc, RwLock};

#[derive(Clone, Debug)]
pub struct FunctionVersion {
    pub function_id: String,
    pub version: u32,
    pub artifact_url: String,
    pub sha256: String,
    pub status: VersionStatus,
    pub created_at: u64,
}

#[derive(Clone, Debug, PartialEq)]
pub enum VersionStatus {
    Active,
    Inactive,
    Deprecated,
}

impl FunctionVersion {
    pub fn new(
        function_id: String,
        version: u32,
        artifact_url: String,
        sha256: String,
    ) -> Self {
        let created_at = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs();

        FunctionVersion {
            function_id,
            version,
            artifact_url,
            sha256,
            status: VersionStatus::Active,
            created_at,
        }
    }
}

pub struct VersionManager {
    versions: Arc<RwLock<HashMap<String, Vec<FunctionVersion>>>>,
    active_versions: Arc<RwLock<HashMap<String, u32>>>,
}

impl VersionManager {
    pub fn new() -> Self {
        VersionManager {
            versions: Arc::new(RwLock::new(HashMap::new())),
            active_versions: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    pub fn register_version(&self, version: FunctionVersion) -> Result<(), String> {
        let mut versions = self.versions.write().unwrap();
        let func_versions = versions
            .entry(version.function_id.clone())
            .or_insert_with(Vec::new);

        if func_versions.iter().any(|v| v.version == version.version) {
            return Err("Version already exists".to_string());
        }

        func_versions.push(version.clone());
        
        // Set as active if first version
        if func_versions.len() == 1 {
            let mut active = self.active_versions.write().unwrap();
            active.insert(version.function_id.clone(), version.version);
        }

        Ok(())
    }

    pub fn get_version(&self, function_id: &str, version: u32) -> Option<FunctionVersion> {
        let versions = self.versions.read().unwrap();
        versions
            .get(function_id)
            .and_then(|v| v.iter().find(|fv| fv.version == version).cloned())
    }

    pub fn get_active_version(&self, function_id: &str) -> Option<FunctionVersion> {
        let active = self.active_versions.read().unwrap();
        if let Some(&version) = active.get(function_id) {
            self.get_version(function_id, version)
        } else {
            None
        }
    }

    pub fn set_active_version(&self, function_id: &str, version: u32) -> Result<(), String> {
        let versions = self.versions.read().unwrap();
        
        if !versions
            .get(function_id)
            .map(|v| v.iter().any(|fv| fv.version == version))
            .unwrap_or(false)
        {
            return Err("Version not found".to_string());
        }

        let mut active = self.active_versions.write().unwrap();
        active.insert(function_id.to_string(), version);
        Ok(())
    }

    pub fn list_versions(&self, function_id: &str) -> Vec<FunctionVersion> {
        let versions = self.versions.read().unwrap();
        versions
            .get(function_id)
            .map(|v| v.clone())
            .unwrap_or_default()
    }

    pub fn rollback(&self, function_id: &str) -> Result<u32, String> {
        let previous_version = {
            let versions = self.versions.read().unwrap();
            
            let func_versions = versions
                .get(function_id)
                .ok_or("Function not found".to_string())?;

            if func_versions.len() < 2 {
                return Err("No previous version to rollback to".to_string());
            }

            let active = self.active_versions.read().unwrap();
            let current_version = active
                .get(function_id)
                .ok_or("No active version".to_string())?;

            // Find previous version
            func_versions
                .iter()
                .filter(|v| v.version < *current_version)
                .max_by_key(|v| v.version)
                .ok_or("No previous version found".to_string())?
                .version
        };

        self.set_active_version(function_id, previous_version)?;
        Ok(previous_version)
    }

    pub fn deprecate_version(&self, function_id: &str, version: u32) -> Result<(), String> {
        let mut versions = self.versions.write().unwrap();
        
        if let Some(func_versions) = versions.get_mut(function_id) {
            if let Some(v) = func_versions.iter_mut().find(|fv| fv.version == version) {
                v.status = VersionStatus::Deprecated;
                return Ok(());
            }
        }

        Err("Version not found".to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_function_version_creation() {
        let version = FunctionVersion::new(
            "fn1".to_string(),
            1,
            "http://example.com/fn.wasm".to_string(),
            "abc123".to_string(),
        );
        
        assert_eq!(version.function_id, "fn1");
        assert_eq!(version.version, 1);
        assert_eq!(version.status, VersionStatus::Active);
    }

    #[test]
    fn test_version_manager_register() {
        let manager = VersionManager::new();
        let version = FunctionVersion::new(
            "fn1".to_string(),
            1,
            "http://example.com/fn.wasm".to_string(),
            "abc123".to_string(),
        );
        
        assert!(manager.register_version(version).is_ok());
    }

    #[test]
    fn test_version_manager_duplicate() {
        let manager = VersionManager::new();
        let version1 = FunctionVersion::new(
            "fn1".to_string(),
            1,
            "http://example.com/fn.wasm".to_string(),
            "abc123".to_string(),
        );
        let version2 = FunctionVersion::new(
            "fn1".to_string(),
            1,
            "http://example.com/fn2.wasm".to_string(),
            "def456".to_string(),
        );
        
        manager.register_version(version1).unwrap();
        assert!(manager.register_version(version2).is_err());
    }

    #[test]
    fn test_get_version() {
        let manager = VersionManager::new();
        let version = FunctionVersion::new(
            "fn1".to_string(),
            1,
            "http://example.com/fn.wasm".to_string(),
            "abc123".to_string(),
        );
        
        manager.register_version(version).unwrap();
        let retrieved = manager.get_version("fn1", 1);
        assert!(retrieved.is_some());
    }

    #[test]
    fn test_get_active_version() {
        let manager = VersionManager::new();
        let version = FunctionVersion::new(
            "fn1".to_string(),
            1,
            "http://example.com/fn.wasm".to_string(),
            "abc123".to_string(),
        );
        
        manager.register_version(version).unwrap();
        let active = manager.get_active_version("fn1");
        assert!(active.is_some());
        assert_eq!(active.unwrap().version, 1);
    }

    #[test]
    fn test_set_active_version() {
        let manager = VersionManager::new();
        let v1 = FunctionVersion::new(
            "fn1".to_string(),
            1,
            "http://example.com/fn1.wasm".to_string(),
            "abc123".to_string(),
        );
        let v2 = FunctionVersion::new(
            "fn1".to_string(),
            2,
            "http://example.com/fn2.wasm".to_string(),
            "def456".to_string(),
        );
        
        manager.register_version(v1).unwrap();
        manager.register_version(v2).unwrap();
        manager.set_active_version("fn1", 2).unwrap();
        
        let active = manager.get_active_version("fn1").unwrap();
        assert_eq!(active.version, 2);
    }

    #[test]
    fn test_rollback() {
        let manager = VersionManager::new();
        let v1 = FunctionVersion::new(
            "fn1".to_string(),
            1,
            "http://example.com/fn1.wasm".to_string(),
            "abc123".to_string(),
        );
        let v2 = FunctionVersion::new(
            "fn1".to_string(),
            2,
            "http://example.com/fn2.wasm".to_string(),
            "def456".to_string(),
        );
        
        manager.register_version(v1).unwrap();
        manager.register_version(v2).unwrap();
        manager.set_active_version("fn1", 2).unwrap();
        
        let rolled_back = manager.rollback("fn1").unwrap();
        assert_eq!(rolled_back, 1);
        
        let active = manager.get_active_version("fn1").unwrap();
        assert_eq!(active.version, 1);
    }

    #[test]
    fn test_deprecate_version() {
        let manager = VersionManager::new();
        let version = FunctionVersion::new(
            "fn1".to_string(),
            1,
            "http://example.com/fn.wasm".to_string(),
            "abc123".to_string(),
        );
        
        manager.register_version(version).unwrap();
        manager.deprecate_version("fn1", 1).unwrap();
        
        let v = manager.get_version("fn1", 1).unwrap();
        assert_eq!(v.status, VersionStatus::Deprecated);
    }
}
