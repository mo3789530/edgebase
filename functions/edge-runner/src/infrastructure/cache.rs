use std::collections::HashMap;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use tokio::sync::RwLock;
use sha2::{Sha256, Digest};
use std::time::SystemTime;

#[derive(Clone)]
struct CacheEntry {
    path: PathBuf,
    size: u64,
    sha256: String,
    last_used: u64,
}

pub struct LocalWasmCache {
    cache_dir: PathBuf,
    entries: Arc<RwLock<HashMap<String, CacheEntry>>>,
    max_size_bytes: u64,
    current_size: Arc<RwLock<u64>>,
}

impl LocalWasmCache {
    pub fn new(cache_dir: impl AsRef<Path>, max_size_bytes: u64) -> std::io::Result<Self> {
        let cache_dir = cache_dir.as_ref().to_path_buf();
        std::fs::create_dir_all(&cache_dir)?;
        
        Ok(Self {
            cache_dir,
            entries: Arc::new(RwLock::new(HashMap::new())),
            max_size_bytes,
            current_size: Arc::new(RwLock::new(0)),
        })
    }
    
    pub async fn get(&self, function_id: &str, version: &str, expected_sha256: &str) -> Option<Vec<u8>> {
        let key = format!("{}/{}", function_id, version);
        let mut entries = self.entries.write().await;
        
        if let Some(entry) = entries.get_mut(&key) {
            // Verify SHA256
            if entry.sha256 != expected_sha256 {
                return None;
            }
            
            // Update last_used
            entry.last_used = SystemTime::now()
                .duration_since(SystemTime::UNIX_EPOCH)
                .unwrap()
                .as_secs();
            
            std::fs::read(&entry.path).ok()
        } else {
            None
        }
    }
    
    pub async fn put(&self, function_id: &str, version: &str, data: &[u8], expected_sha256: &str) -> std::io::Result<()> {
        // Verify SHA256
        let mut hasher = Sha256::new();
        hasher.update(data);
        let hash = format!("{:x}", hasher.finalize());
        
        if hash != expected_sha256 {
            return Err(std::io::Error::new(
                std::io::ErrorKind::InvalidData,
                "SHA256 mismatch",
            ));
        }
        
        let key = format!("{}/{}", function_id, version);
        let path = self.cache_dir.join(&key).with_extension("wasm");
        
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        
        std::fs::write(&path, data)?;
        
        let size = data.len() as u64;
        let now = SystemTime::now()
            .duration_since(SystemTime::UNIX_EPOCH)
            .unwrap()
            .as_secs();
        
        let mut entries = self.entries.write().await;
        entries.insert(key, CacheEntry {
            path,
            size,
            sha256: hash,
            last_used: now,
        });
        
        let mut current_size = self.current_size.write().await;
        *current_size += size;
        drop(current_size);
        drop(entries);
        
        self.evict_if_needed().await;
        
        Ok(())
    }
    
    async fn evict_if_needed(&self) {
        loop {
            let current_size = *self.current_size.read().await;
            if current_size <= self.max_size_bytes {
                break;
            }
            
            let mut entries = self.entries.write().await;
            if entries.is_empty() {
                break;
            }
            
            // Find LRU entry
            let oldest_key = entries.iter()
                .min_by_key(|(_, e)| e.last_used)
                .map(|(k, _)| k.clone());
            
            if let Some(key) = oldest_key {
                if let Some(entry) = entries.remove(&key) {
                    let _ = std::fs::remove_file(&entry.path);
                    let mut current_size = self.current_size.write().await;
                    *current_size -= entry.size;
                }
            }
        }
    }
    
    #[allow(dead_code)]
    pub async fn remove(&self, function_id: &str, version: &str) -> std::io::Result<()> {
        let key = format!("{}/{}", function_id, version);
        let mut entries = self.entries.write().await;
        
        if let Some(entry) = entries.remove(&key) {
            std::fs::remove_file(&entry.path)?;
            let mut current_size = self.current_size.write().await;
            *current_size -= entry.size;
        }
        
        Ok(())
    }
    
    #[allow(dead_code)]
    pub async fn get_size(&self) -> u64 {
        *self.current_size.read().await
    }
}
