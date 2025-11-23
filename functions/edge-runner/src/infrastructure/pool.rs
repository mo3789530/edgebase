use crate::domain::PooledInstance;
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::Mutex;
use wasmer::{Store, Module, Instance, imports, Memory, MemoryType, Pages};
use std::time::{SystemTime, UNIX_EPOCH};

pub struct HotInstancePool {
    pools: Arc<Mutex<HashMap<String, Vec<PooledInstance>>>>,
    max_instances: usize,
    idle_timeout_secs: u64,
}

impl HotInstancePool {
    pub fn new(max_instances: usize, idle_timeout_secs: u64) -> Self {
        Self {
            pools: Arc::new(Mutex::new(HashMap::new())),
            max_instances,
            idle_timeout_secs,
        }
    }
    
    pub async fn get_or_create(
        &self,
        function_id: &str,
        wasm_bytes: &[u8],
        memory_pages: u32,
    ) -> Result<PooledInstance, String> {
        let mut pools = self.pools.lock().await;
        let pool = pools.entry(function_id.to_string()).or_insert_with(Vec::new);
        
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();
        
        pool.retain(|inst| now - inst.last_used < self.idle_timeout_secs);
        
        if let Some(mut pooled) = pool.pop() {
            pooled.last_used = now;
            return Ok(pooled);
        }
        
        if pool.len() < self.max_instances {
            let mut store = Store::default();
            let module = Module::new(&store, wasm_bytes)
                .map_err(|e| format!("Failed to compile WASM: {}", e))?;
            
            let memory = Memory::new(&mut store, MemoryType::new(
                Pages(memory_pages),
                Some(Pages(memory_pages)),
                false
            )).map_err(|e| format!("Failed to create memory: {}", e))?;
            
            let import_object = imports! {
                "env" => {
                    "memory" => memory.clone(),
                }
            };
            
            let instance = Instance::new(&mut store, &module, &import_object)
                .map_err(|e| format!("Failed to instantiate WASM: {}", e))?;
            
            Ok(PooledInstance {
                instance,
                store,
                last_used: now,
            })
        } else {
            Err("Pool at max capacity".to_string())
        }
    }
    
    pub async fn return_instance(&self, function_id: &str, pooled: PooledInstance) {
        let mut pools = self.pools.lock().await;
        let pool = pools.entry(function_id.to_string()).or_insert_with(Vec::new);
        
        if pool.len() < self.max_instances {
            pool.push(pooled);
        }
    }
}
