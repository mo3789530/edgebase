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

    #[test]
    fn test_max_backoff_limit() {
        let delay_10 = ErrorRecovery::calculate_backoff(10);
        let delay_20 = ErrorRecovery::calculate_backoff(20);
        
        // Verify exponential growth
        assert!(delay_20.as_millis() > delay_10.as_millis());
    }

    #[test]
    fn test_error_message_format() {
        let error = EdgeError::NetworkError("Connection refused".to_string());
        let message = ErrorRecovery::handle_error(&error);
        
        assert!(message.contains("Retryable error"));
        assert!(message.contains("1001"));
        assert!(message.contains("Connection refused"));
    }

    #[test]
    fn test_conflict_error_not_retryable() {
        let error = EdgeError::ConflictError("Version mismatch".to_string());
        assert!(!error.is_retryable());
        assert!(!error.is_fatal());
    }

    #[test]
    fn test_database_error_not_retryable() {
        let error = EdgeError::DatabaseError("Connection failed".to_string());
        assert!(!error.is_retryable());
        assert!(!error.is_fatal());
    }

    #[test]
    fn test_data_error_not_retryable() {
        let error = EdgeError::DataError("Invalid JSON".to_string());
        assert!(!error.is_retryable());
        assert!(!error.is_fatal());
    }

    #[test]
    fn test_should_retry_at_boundary() {
        let error = EdgeError::NetworkError("Connection failed".to_string());
        
        // At max attempts, should not retry
        assert!(!ErrorRecovery::should_retry(&error, 5, 5));
        
        // Just before max attempts, should retry
        assert!(ErrorRecovery::should_retry(&error, 4, 5));
    }

    #[test]
    fn test_backoff_sequence() {
        let mut delays = Vec::new();
        for attempt in 0..5 {
            delays.push(ErrorRecovery::calculate_backoff(attempt).as_millis());
        }
        
        // Verify exponential sequence: 1, 2, 4, 8, 16
        assert_eq!(delays, vec![1, 2, 4, 8, 16]);
    }

    #[test]
    fn test_error_recovery_workflow() {
        let error = EdgeError::NetworkError("Connection failed".to_string());
        let max_attempts = 5;
        
        let mut attempt = 0;
        while attempt < max_attempts {
            if !ErrorRecovery::should_retry(&error, attempt, max_attempts) {
                break;
            }
            
            let delay = ErrorRecovery::calculate_backoff(attempt);
            assert!(delay.as_millis() > 0);
            
            attempt += 1;
        }
        
        assert_eq!(attempt, max_attempts);
    }

    #[test]
    fn test_fatal_error_no_recovery() {
        let error = EdgeError::ValidationError("Invalid input".to_string());
        
        for attempt in 0..5 {
            assert!(!ErrorRecovery::should_retry(&error, attempt, 5));
        }
    }

    #[test]
    fn test_timeout_error_retryable() {
        let error = EdgeError::TimeoutError("Request timeout".to_string());
        assert!(error.is_retryable());
        assert!(!error.is_fatal());
        
        assert!(ErrorRecovery::should_retry(&error, 0, 5));
        assert!(ErrorRecovery::should_retry(&error, 2, 5));
    }

    #[test]
    fn test_resource_error_retryable() {
        let error = EdgeError::ResourceError("Out of memory".to_string());
        assert!(error.is_retryable());
        assert!(!error.is_fatal());
        
        assert!(ErrorRecovery::should_retry(&error, 0, 5));
    }

    #[test]
    fn test_error_classification() {
        let retryable_errors = vec![
            EdgeError::NetworkError("".to_string()),
            EdgeError::TimeoutError("".to_string()),
            EdgeError::ResourceError("".to_string()),
        ];
        
        let fatal_errors = vec![
            EdgeError::AuthError("".to_string()),
            EdgeError::ValidationError("".to_string()),
        ];
        
        for error in retryable_errors {
            assert!(error.is_retryable());
        }
        
        for error in fatal_errors {
            assert!(error.is_fatal());
        }
    }

    #[test]
    fn test_error_code_range() {
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
        
        for error in errors {
            let code = error.error_code();
            assert!(code >= 1001 && code <= 1008);
        }
    }

    #[test]
    fn test_handle_error_consistency() {
        let error = EdgeError::NetworkError("Connection failed".to_string());
        let message1 = ErrorRecovery::handle_error(&error);
        let message2 = ErrorRecovery::handle_error(&error);
        
        assert_eq!(message1, message2);
    }

    #[test]
    fn test_multiple_error_types() {
        let errors = vec![
            EdgeError::NetworkError("Network issue".to_string()),
            EdgeError::AuthError("Auth issue".to_string()),
            EdgeError::DataError("Data issue".to_string()),
            EdgeError::ConflictError("Conflict issue".to_string()),
            EdgeError::ResourceError("Resource issue".to_string()),
            EdgeError::DatabaseError("DB issue".to_string()),
            EdgeError::ValidationError("Validation issue".to_string()),
            EdgeError::TimeoutError("Timeout issue".to_string()),
        ];
        
        for error in errors {
            let message = ErrorRecovery::handle_error(&error);
            assert!(!message.is_empty());
            assert!(message.contains(&error.error_code().to_string()));
        }
    }

    #[test]
    fn test_retry_logic_with_max_attempts() {
        let error = EdgeError::NetworkError("Connection failed".to_string());
        let max_attempts = 3;
        
        for attempt in 0..max_attempts {
            assert!(ErrorRecovery::should_retry(&error, attempt, max_attempts));
        }
        
        assert!(!ErrorRecovery::should_retry(&error, max_attempts, max_attempts));
    }
}
