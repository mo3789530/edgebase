use std::path::Path;
use std::time::{Duration, Instant};

pub struct WasmRuntime {
    memory_pages: u32,
    max_execution_ms: u32,
}

impl WasmRuntime {
    pub fn new(memory_pages: u32, max_execution_ms: u32) -> Result<Self, String> {
        if memory_pages == 0 {
            return Err("Memory pages must be greater than 0".to_string());
        }
        if max_execution_ms == 0 {
            return Err("Max execution time must be greater than 0".to_string());
        }

        Ok(WasmRuntime {
            memory_pages,
            max_execution_ms,
        })
    }

    pub fn load_module(&self, wasm_path: &Path) -> Result<Vec<u8>, String> {
        std::fs::read(wasm_path)
            .map_err(|e| format!("Failed to load WASM module: {}", e))
    }

    pub fn validate_module(&self, wasm_bytes: &[u8]) -> Result<(), String> {
        if wasm_bytes.len() < 4 {
            return Err("WASM module too small".to_string());
        }

        if &wasm_bytes[0..4] != b"\0asm" {
            return Err("Invalid WASM magic number".to_string());
        }

        Ok(())
    }

    pub fn execute_with_timeout<F, R>(&self, f: F) -> Result<R, String>
    where
        F: FnOnce() -> Result<R, String>,
    {
        let start = Instant::now();
        let timeout = Duration::from_millis(self.max_execution_ms as u64);

        let result = f()?;

        let elapsed = start.elapsed();
        if elapsed > timeout {
            return Err(format!(
                "Execution timeout: took {:?}, limit {:?}",
                elapsed, timeout
            ));
        }

        Ok(result)
    }

    pub fn get_memory_pages(&self) -> u32 {
        self.memory_pages
    }

    pub fn get_max_execution_ms(&self) -> u32 {
        self.max_execution_ms
    }
}

pub struct ExecutionResult {
    pub status_code: u16,
    pub body: Vec<u8>,
    pub execution_time_ms: u64,
}

impl ExecutionResult {
    pub fn new(status_code: u16, body: Vec<u8>, execution_time_ms: u64) -> Self {
        ExecutionResult {
            status_code,
            body,
            execution_time_ms,
        }
    }

    pub fn success(body: Vec<u8>, execution_time_ms: u64) -> Self {
        ExecutionResult {
            status_code: 200,
            body,
            execution_time_ms,
        }
    }

    pub fn error(message: String, execution_time_ms: u64) -> Self {
        ExecutionResult {
            status_code: 500,
            body: message.into_bytes(),
            execution_time_ms,
        }
    }

    pub fn timeout(execution_time_ms: u64) -> Self {
        ExecutionResult {
            status_code: 504,
            body: b"Gateway Timeout".to_vec(),
            execution_time_ms,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_wasm_runtime_creation() {
        let runtime = WasmRuntime::new(256, 5000).unwrap();
        assert_eq!(runtime.get_memory_pages(), 256);
        assert_eq!(runtime.get_max_execution_ms(), 5000);
    }

    #[test]
    fn test_wasm_runtime_invalid_memory() {
        let result = WasmRuntime::new(0, 5000);
        assert!(result.is_err());
    }

    #[test]
    fn test_wasm_runtime_invalid_timeout() {
        let result = WasmRuntime::new(256, 0);
        assert!(result.is_err());
    }

    #[test]
    fn test_validate_module_valid() {
        let runtime = WasmRuntime::new(256, 5000).unwrap();
        let mut wasm_data = vec![0u8; 100];
        wasm_data[0..4].copy_from_slice(b"\0asm");
        
        let result = runtime.validate_module(&wasm_data);
        assert!(result.is_ok());
    }

    #[test]
    fn test_validate_module_invalid_magic() {
        let runtime = WasmRuntime::new(256, 5000).unwrap();
        let wasm_data = b"invalid";
        
        let result = runtime.validate_module(wasm_data);
        assert!(result.is_err());
    }

    #[test]
    fn test_execute_with_timeout_success() {
        let runtime = WasmRuntime::new(256, 5000).unwrap();
        let result = runtime.execute_with_timeout(|| Ok("success".to_string()));
        assert!(result.is_ok());
    }

    #[test]
    fn test_execute_with_timeout_error() {
        let runtime = WasmRuntime::new(256, 5000).unwrap();
        let result: Result<String, String> = runtime.execute_with_timeout(|| Err("error".to_string()));
        assert!(result.is_err());
    }

    #[test]
    fn test_execution_result_success() {
        let result = ExecutionResult::success(b"test".to_vec(), 100);
        assert_eq!(result.status_code, 200);
        assert_eq!(result.execution_time_ms, 100);
    }

    #[test]
    fn test_execution_result_timeout() {
        let result = ExecutionResult::timeout(5000);
        assert_eq!(result.status_code, 504);
        assert_eq!(result.execution_time_ms, 5000);
    }
}
