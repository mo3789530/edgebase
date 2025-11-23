use std::collections::HashMap;
use std::path::{Path, PathBuf};
use sha2::{Sha256, Digest};

pub struct WasmCache {
    cache_dir: PathBuf,
    entries: HashMap<String, CacheEntry>,
    max_size_bytes: u64,
    current_size: u64,
}

struct CacheEntry {
    path: PathBuf,
    size: u64,
    sha256: String,
    last_used: std::time::SystemTime,
}

impl WasmCache {
    pub fn new(cache_dir: impl AsRef<Path>, max_size_bytes: u64) -> std::io::Result<Self> {
        let cache_dir = cache_dir.as_ref().to_path_buf();
        std::fs::create_dir_all(&cache_dir)?;
        
        Ok(Self {
            cache_dir,
            entries: HashMap::new(),
            max_size_bytes,
            current_size: 0,
        })
    }
    
    pub fn get(&mut self, function_id: &str, version: &str) -> Option<Vec<u8>> {
        let key = format!("{}/{}", function_id, version);
        
        if let Some(entry) = self.entries.get_mut(&key) {
            entry.last_used = std::time::SystemTime::now();
            std::fs::read(&entry.path).ok()
        } else {
            None
        }
    }
    
    pub fn put(&mut self, function_id: &str, version: &str, data: &[u8], expected_sha256: &str) -> std::io::Result<()> {
        let mut hasher = Sha256::new();
        hasher.update(data);
        let hash = format!("{:x}", hasher.finalize());
        
        if hash != expected_sha256 {
            return Err(std::io::Error::new(
                std::io::ErrorKind::InvalidData,
                "SHA256 mismatch"
            ));
        }
        
        let key = format!("{}/{}", function_id, version);
        let path = self.cache_dir.join(&key).with_extension("wasm");
        
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        
        std::fs::write(&path, data)?;
        
        let size = data.len() as u64;
        self.current_size += size;
        
        self.entries.insert(key, CacheEntry {
            path,
            size,
            sha256: hash,
            last_used: std::time::SystemTime::now(),
        });
        
        self.evict_if_needed();
        
        Ok(())
    }
    
    fn evict_if_needed(&mut self) {
        while self.current_size > self.max_size_bytes && !self.entries.is_empty() {
            let oldest_key = self.entries.iter()
                .min_by_key(|(_, e)| e.last_used)
                .map(|(k, _)| k.clone());
            
            if let Some(key) = oldest_key {
                if let Some(entry) = self.entries.remove(&key) {
                    let _ = std::fs::remove_file(&entry.path);
                    self.current_size -= entry.size;
                }
            }
        }
    }
}
