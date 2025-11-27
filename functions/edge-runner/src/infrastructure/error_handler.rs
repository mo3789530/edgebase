use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Serialize, Deserialize)]
pub enum ErrorType {
    ExecutionError,
    TimeoutError,
    ResourceError,
    NetworkError,
    ValidationError,
    NotFoundError,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ErrorResponse {
    pub error_type: String,
    pub message: String,
    pub status_code: u16,
    pub retryable: bool,
}

impl ErrorResponse {
    pub fn new(error_type: ErrorType, message: String, status_code: u16, retryable: bool) -> Self {
        ErrorResponse {
            error_type: format!("{:?}", error_type),
            message,
            status_code,
            retryable,
        }
    }

    pub fn execution_error(message: String) -> Self {
        ErrorResponse::new(ErrorType::ExecutionError, message, 500, true)
    }

    pub fn timeout_error() -> Self {
        ErrorResponse::new(
            ErrorType::TimeoutError,
            "Function execution timeout".to_string(),
            504,
            true,
        )
    }

    pub fn resource_error(message: String) -> Self {
        ErrorResponse::new(ErrorType::ResourceError, message, 503, true)
    }

    pub fn network_error(message: String) -> Self {
        ErrorResponse::new(ErrorType::NetworkError, message, 502, true)
    }

    pub fn validation_error(message: String) -> Self {
        ErrorResponse::new(ErrorType::ValidationError, message, 400, false)
    }

    pub fn not_found_error(message: String) -> Self {
        ErrorResponse::new(ErrorType::NotFoundError, message, 404, false)
    }

    pub fn to_json(&self) -> Result<String, String> {
        serde_json::to_string(self)
            .map_err(|e| format!("Failed to serialize error: {}", e))
    }
}

pub struct FallbackManager {
    fallback_versions: std::sync::Arc<std::sync::RwLock<std::collections::HashMap<String, String>>>,
}

impl FallbackManager {
    pub fn new() -> Self {
        FallbackManager {
            fallback_versions: std::sync::Arc::new(std::sync::RwLock::new(
                std::collections::HashMap::new(),
            )),
        }
    }

    pub fn register_fallback(&self, function_id: String, fallback_version: String) -> Result<(), String> {
        let mut versions = self.fallback_versions.write().unwrap();
        versions.insert(function_id, fallback_version);
        Ok(())
    }

    pub fn get_fallback(&self, function_id: &str) -> Option<String> {
        let versions = self.fallback_versions.read().unwrap();
        versions.get(function_id).cloned()
    }

    pub fn clear_fallback(&self, function_id: &str) -> Result<(), String> {
        let mut versions = self.fallback_versions.write().unwrap();
        versions.remove(function_id);
        Ok(())
    }
}

pub struct CircuitBreaker {
    failure_count: std::sync::Arc<std::sync::atomic::AtomicU32>,
    failure_threshold: u32,
    state: std::sync::Arc<std::sync::RwLock<CircuitBreakerState>>,
}

#[derive(Clone, Debug, PartialEq)]
pub enum CircuitBreakerState {
    Closed,
    Open,
    HalfOpen,
}

impl CircuitBreaker {
    pub fn new(failure_threshold: u32) -> Self {
        CircuitBreaker {
            failure_count: std::sync::Arc::new(std::sync::atomic::AtomicU32::new(0)),
            failure_threshold,
            state: std::sync::Arc::new(std::sync::RwLock::new(CircuitBreakerState::Closed)),
        }
    }

    pub fn record_success(&self) {
        self.failure_count.store(0, std::sync::atomic::Ordering::Relaxed);
        let mut state = self.state.write().unwrap();
        *state = CircuitBreakerState::Closed;
    }

    pub fn record_failure(&self) {
        let count = self.failure_count.fetch_add(1, std::sync::atomic::Ordering::Relaxed) + 1;
        
        if count >= self.failure_threshold {
            let mut state = self.state.write().unwrap();
            *state = CircuitBreakerState::Open;
        }
    }

    pub fn get_state(&self) -> CircuitBreakerState {
        self.state.read().unwrap().clone()
    }

    pub fn is_open(&self) -> bool {
        self.get_state() == CircuitBreakerState::Open
    }

    pub fn attempt_reset(&self) {
        let mut state = self.state.write().unwrap();
        if *state == CircuitBreakerState::Open {
            *state = CircuitBreakerState::HalfOpen;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_error_response_execution_error() {
        let error = ErrorResponse::execution_error("Test error".to_string());
        assert_eq!(error.status_code, 500);
        assert!(error.retryable);
    }

    #[test]
    fn test_error_response_timeout() {
        let error = ErrorResponse::timeout_error();
        assert_eq!(error.status_code, 504);
        assert!(error.retryable);
    }

    #[test]
    fn test_error_response_validation() {
        let error = ErrorResponse::validation_error("Invalid input".to_string());
        assert_eq!(error.status_code, 400);
        assert!(!error.retryable);
    }

    #[test]
    fn test_error_response_to_json() {
        let error = ErrorResponse::execution_error("Test error".to_string());
        let json = error.to_json();
        assert!(json.is_ok());
    }

    #[test]
    fn test_fallback_manager_register() {
        let manager = FallbackManager::new();
        manager.register_fallback("fn1".to_string(), "v1".to_string()).unwrap();
        
        let fallback = manager.get_fallback("fn1");
        assert_eq!(fallback, Some("v1".to_string()));
    }

    #[test]
    fn test_fallback_manager_clear() {
        let manager = FallbackManager::new();
        manager.register_fallback("fn1".to_string(), "v1".to_string()).unwrap();
        manager.clear_fallback("fn1").unwrap();
        
        let fallback = manager.get_fallback("fn1");
        assert!(fallback.is_none());
    }

    #[test]
    fn test_circuit_breaker_creation() {
        let breaker = CircuitBreaker::new(3);
        assert_eq!(breaker.get_state(), CircuitBreakerState::Closed);
    }

    #[test]
    fn test_circuit_breaker_record_success() {
        let breaker = CircuitBreaker::new(3);
        breaker.record_failure();
        breaker.record_success();
        
        assert_eq!(breaker.get_state(), CircuitBreakerState::Closed);
    }

    #[test]
    fn test_circuit_breaker_open() {
        let breaker = CircuitBreaker::new(3);
        breaker.record_failure();
        breaker.record_failure();
        breaker.record_failure();
        
        assert!(breaker.is_open());
    }

    #[test]
    fn test_circuit_breaker_half_open() {
        let breaker = CircuitBreaker::new(3);
        breaker.record_failure();
        breaker.record_failure();
        breaker.record_failure();
        
        assert!(breaker.is_open());
        breaker.attempt_reset();
        assert_eq!(breaker.get_state(), CircuitBreakerState::HalfOpen);
    }
}
