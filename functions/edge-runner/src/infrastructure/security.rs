use sha2::{Sha256, Digest};

pub struct SecurityManager {
    api_keys: std::sync::Arc<std::sync::RwLock<Vec<String>>>,
}

impl SecurityManager {
    pub fn new() -> Self {
        SecurityManager {
            api_keys: std::sync::Arc::new(std::sync::RwLock::new(Vec::new())),
        }
    }

    pub fn add_api_key(&self, key: String) -> Result<(), String> {
        let mut keys = self.api_keys.write().unwrap();
        if keys.contains(&key) {
            return Err("API key already exists".to_string());
        }
        keys.push(key);
        Ok(())
    }

    pub fn validate_api_key(&self, key: &str) -> Result<bool, String> {
        let keys = self.api_keys.read().unwrap();
        Ok(keys.contains(&key.to_string()))
    }

    pub fn verify_signature(
        &self,
        message: &[u8],
        signature: &str,
        secret: &str,
    ) -> Result<bool, String> {
        let expected = self.compute_hmac_sha256(message, secret);
        Ok(expected == signature)
    }

    pub fn compute_hmac_sha256(&self, message: &[u8], secret: &str) -> String {
        let mut hasher = Sha256::new();
        hasher.update(secret.as_bytes());
        hasher.update(message);
        format!("{:x}", hasher.finalize())
    }

    pub fn validate_wasm_sandbox(&self, wasm_bytes: &[u8]) -> Result<(), String> {
        if wasm_bytes.len() < 4 {
            return Err("WASM module too small".to_string());
        }

        if &wasm_bytes[0..4] != b"\0asm" {
            return Err("Invalid WASM magic number".to_string());
        }

        // Check for restricted imports
        let wasm_str = String::from_utf8_lossy(wasm_bytes);
        if wasm_str.contains("env") && wasm_str.contains("syscall") {
            return Err("Restricted syscall detected".to_string());
        }

        Ok(())
    }

    pub fn generate_api_key(&self) -> String {
        use uuid::Uuid;
        Uuid::new_v4().to_string()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_security_manager_creation() {
        let manager = SecurityManager::new();
        assert!(manager.validate_api_key("nonexistent").unwrap() == false);
    }

    #[test]
    fn test_add_api_key() {
        let manager = SecurityManager::new();
        let key = "test_key_123".to_string();
        
        assert!(manager.add_api_key(key.clone()).is_ok());
        assert!(manager.validate_api_key(&key).unwrap());
    }

    #[test]
    fn test_add_duplicate_api_key() {
        let manager = SecurityManager::new();
        let key = "test_key_123".to_string();
        
        manager.add_api_key(key.clone()).unwrap();
        assert!(manager.add_api_key(key).is_err());
    }

    #[test]
    fn test_compute_hmac_sha256() {
        let manager = SecurityManager::new();
        let message = b"test message";
        let secret = "secret";
        
        let signature1 = manager.compute_hmac_sha256(message, secret);
        let signature2 = manager.compute_hmac_sha256(message, secret);
        
        assert_eq!(signature1, signature2);
        assert_eq!(signature1.len(), 64);
    }

    #[test]
    fn test_verify_signature_valid() {
        let manager = SecurityManager::new();
        let message = b"test message";
        let secret = "secret";
        
        let signature = manager.compute_hmac_sha256(message, secret);
        let result = manager.verify_signature(message, &signature, secret);
        
        assert!(result.unwrap());
    }

    #[test]
    fn test_verify_signature_invalid() {
        let manager = SecurityManager::new();
        let message = b"test message";
        let secret = "secret";
        
        let result = manager.verify_signature(message, "invalid_signature", secret);
        assert!(!result.unwrap());
    }

    #[test]
    fn test_validate_wasm_sandbox_valid() {
        let manager = SecurityManager::new();
        let mut wasm_data = vec![0u8; 100];
        wasm_data[0..4].copy_from_slice(b"\0asm");
        
        let result = manager.validate_wasm_sandbox(&wasm_data);
        assert!(result.is_ok());
    }

    #[test]
    fn test_validate_wasm_sandbox_invalid_magic() {
        let manager = SecurityManager::new();
        let wasm_data = b"invalid";
        
        let result = manager.validate_wasm_sandbox(wasm_data);
        assert!(result.is_err());
    }

    #[test]
    fn test_generate_api_key() {
        let manager = SecurityManager::new();
        let key1 = manager.generate_api_key();
        let key2 = manager.generate_api_key();
        
        assert_ne!(key1, key2);
        assert!(!key1.is_empty());
    }
}
