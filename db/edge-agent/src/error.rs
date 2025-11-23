use thiserror::Error;

#[derive(Error, Debug)]
pub enum EdgeError {
    #[error("Network error: {0}")]
    NetworkError(String),

    #[error("Authentication error: {0}")]
    AuthError(String),

    #[error("Data error: {0}")]
    DataError(String),

    #[error("Conflict error: {0}")]
    ConflictError(String),

    #[error("Resource error: {0}")]
    ResourceError(String),

    #[error("Database error: {0}")]
    DatabaseError(String),

    #[error("Validation error: {0}")]
    ValidationError(String),

    #[error("Timeout error: {0}")]
    TimeoutError(String),
}

impl EdgeError {
    pub fn is_retryable(&self) -> bool {
        matches!(
            self,
            EdgeError::NetworkError(_)
                | EdgeError::TimeoutError(_)
                | EdgeError::ResourceError(_)
        )
    }

    pub fn is_fatal(&self) -> bool {
        matches!(
            self,
            EdgeError::AuthError(_) | EdgeError::ValidationError(_)
        )
    }

    pub fn error_code(&self) -> u32 {
        match self {
            EdgeError::NetworkError(_) => 1001,
            EdgeError::AuthError(_) => 1002,
            EdgeError::DataError(_) => 1003,
            EdgeError::ConflictError(_) => 1004,
            EdgeError::ResourceError(_) => 1005,
            EdgeError::DatabaseError(_) => 1006,
            EdgeError::ValidationError(_) => 1007,
            EdgeError::TimeoutError(_) => 1008,
        }
    }
}

pub struct ErrorRecovery;

impl ErrorRecovery {
    pub fn should_retry(error: &EdgeError, attempt: u32, max_attempts: u32) -> bool {
        if attempt >= max_attempts {
            return false;
        }
        error.is_retryable()
    }

    pub fn calculate_backoff(attempt: u32) -> std::time::Duration {
        let base_delay = 1u64;
        let delay_ms = base_delay * 2_u64.pow(attempt);
        std::time::Duration::from_millis(delay_ms)
    }

    pub fn handle_error(error: &EdgeError) -> String {
        if error.is_fatal() {
            format!("Fatal error ({}): {}", error.error_code(), error)
        } else if error.is_retryable() {
            format!("Retryable error ({}): {}", error.error_code(), error)
        } else {
            format!("Error ({}): {}", error.error_code(), error)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_network_error_creation() {
        let error = EdgeError::NetworkError("Connection refused".to_string());
        assert_eq!(error.error_code(), 1001);
        assert!(error.is_retryable());
        assert!(!error.is_fatal());
    }

    #[test]
    fn test_auth_error_creation() {
        let error = EdgeError::AuthError("Invalid token".to_string());
        assert_eq!(error.error_code(), 1002);
        assert!(!error.is_retryable());
        assert!(error.is_fatal());
    }

    #[test]
    fn test_data_error_creation() {
        let error = EdgeError::DataError("Invalid JSON".to_string());
        assert_eq!(error.error_code(), 1003);
        assert!(!error.is_retryable());
        assert!(!error.is_fatal());
    }

    #[test]
    fn test_conflict_error_creation() {
        let error = EdgeError::ConflictError("Version mismatch".to_string());
        assert_eq!(error.error_code(), 1004);
        assert!(!error.is_retryable());
        assert!(!error.is_fatal());
    }

    #[test]
    fn test_resource_error_creation() {
        let error = EdgeError::ResourceError("Out of memory".to_string());
        assert_eq!(error.error_code(), 1005);
        assert!(error.is_retryable());
        assert!(!error.is_fatal());
    }

    #[test]
    fn test_database_error_creation() {
        let error = EdgeError::DatabaseError("Connection failed".to_string());
        assert_eq!(error.error_code(), 1006);
        assert!(!error.is_retryable());
        assert!(!error.is_fatal());
    }

    #[test]
    fn test_validation_error_creation() {
        let error = EdgeError::ValidationError("Invalid input".to_string());
        assert_eq!(error.error_code(), 1007);
        assert!(!error.is_retryable());
        assert!(error.is_fatal());
    }

    #[test]
    fn test_timeout_error_creation() {
        let error = EdgeError::TimeoutError("Request timeout".to_string());
        assert_eq!(error.error_code(), 1008);
        assert!(error.is_retryable());
        assert!(!error.is_fatal());
    }

    #[test]
    fn test_should_retry_retryable_error() {
        let error = EdgeError::NetworkError("Connection failed".to_string());
        assert!(ErrorRecovery::should_retry(&error, 0, 5));
        assert!(ErrorRecovery::should_retry(&error, 3, 5));
        assert!(!ErrorRecovery::should_retry(&error, 5, 5));
    }

    #[test]
    fn test_should_retry_fatal_error() {
        let error = EdgeError::AuthError("Invalid token".to_string());
        assert!(!ErrorRecovery::should_retry(&error, 0, 5));
        assert!(!ErrorRecovery::should_retry(&error, 3, 5));
    }

    #[test]
    fn test_calculate_backoff() {
        let delay0 = ErrorRecovery::calculate_backoff(0);
        let delay1 = ErrorRecovery::calculate_backoff(1);
        let delay2 = ErrorRecovery::calculate_backoff(2);

        assert_eq!(delay0.as_millis(), 1);
        assert_eq!(delay1.as_millis(), 2);
        assert_eq!(delay2.as_millis(), 4);
    }

    #[test]
    fn test_exponential_backoff() {
        for attempt in 0..5 {
            let delay = ErrorRecovery::calculate_backoff(attempt);
            let expected_ms = 2_u64.pow(attempt);
            assert_eq!(delay.as_millis() as u64, expected_ms);
        }
    }

    #[test]
    fn test_handle_fatal_error() {
        let error = EdgeError::AuthError("Invalid token".to_string());
        let message = ErrorRecovery::handle_error(&error);
        assert!(message.contains("Fatal error"));
        assert!(message.contains("1002"));
    }

    #[test]
    fn test_handle_retryable_error() {
        let error = EdgeError::NetworkError("Connection refused".to_string());
        let message = ErrorRecovery::handle_error(&error);
        assert!(message.contains("Retryable error"));
        assert!(message.contains("1001"));
    }

    #[test]
    fn test_handle_other_error() {
        let error = EdgeError::DataError("Invalid JSON".to_string());
        let message = ErrorRecovery::handle_error(&error);
        assert!(message.contains("Error"));
        assert!(message.contains("1003"));
    }

    #[test]
    fn test_error_display() {
        let error = EdgeError::NetworkError("Connection refused".to_string());
        let display = format!("{}", error);
        assert!(display.contains("Network error"));
        assert!(display.contains("Connection refused"));
    }

    #[test]
    fn test_error_debug() {
        let error = EdgeError::AuthError("Invalid token".to_string());
        let debug = format!("{:?}", error);
        assert!(debug.contains("AuthError"));
    }

    #[test]
    fn test_all_error_codes_unique() {
        let errors = vec![
            EdgeError::NetworkError("".to_string()),
            EdgeError::AuthError("".to_string()),
            EdgeError::DataError("".to_string()),
            EdgeError::ConflictError("".to_string()),
            EdgeError::ResourceError("".to_string()),
            EdgeError::DatabaseError("".to_string()),
            EdgeError::ValidationError("".to_string()),
            EdgeError::TimeoutError("".to_string()),
        ];

        let mut codes: Vec<u32> = errors.iter().map(|e| e.error_code()).collect();
        codes.sort();
        codes.dedup();

        assert_eq!(codes.len(), 8);
    }

    #[test]
    fn test_recovery_strategy_network_error() {
        let error = EdgeError::NetworkError("Connection failed".to_string());
        
        for attempt in 0..3 {
            assert!(ErrorRecovery::should_retry(&error, attempt, 5));
            let delay = ErrorRecovery::calculate_backoff(attempt);
            assert!(delay.as_millis() > 0);
        }
    }

    #[test]
    fn test_recovery_strategy_auth_error() {
        let error = EdgeError::AuthError("Invalid token".to_string());
        
        assert!(!ErrorRecovery::should_retry(&error, 0, 5));
        let message = ErrorRecovery::handle_error(&error);
        assert!(message.contains("Fatal"));
    }
}
