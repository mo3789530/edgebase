use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Clone, Debug)]
pub struct ResourceQuota {
    pub function_id: String,
    pub memory_quota_mb: u64,
    pub execution_time_quota_ms: u32,
    pub concurrent_limit: u32,
}

impl ResourceQuota {
    pub fn new(
        function_id: String,
        memory_quota_mb: u64,
        execution_time_quota_ms: u32,
        concurrent_limit: u32,
    ) -> Self {
        ResourceQuota {
            function_id,
            memory_quota_mb,
            execution_time_quota_ms,
            concurrent_limit,
        }
    }
}

pub struct RateLimiter {
    quotas: Arc<RwLock<HashMap<String, ResourceQuota>>>,
    request_counts: Arc<RwLock<HashMap<String, Vec<u64>>>>,
    window_size_secs: u64,
}

impl RateLimiter {
    pub fn new(window_size_secs: u64) -> Self {
        RateLimiter {
            quotas: Arc::new(RwLock::new(HashMap::new())),
            request_counts: Arc::new(RwLock::new(HashMap::new())),
            window_size_secs,
        }
    }

    pub fn register_quota(&self, quota: ResourceQuota) -> Result<(), String> {
        let mut quotas = self.quotas.write().unwrap();
        if quotas.contains_key(&quota.function_id) {
            return Err("Quota already registered".to_string());
        }
        quotas.insert(quota.function_id.clone(), quota);
        Ok(())
    }

    pub fn get_quota(&self, function_id: &str) -> Option<ResourceQuota> {
        let quotas = self.quotas.read().unwrap();
        quotas.get(function_id).cloned()
    }

    pub fn check_rate_limit(&self, function_id: &str, max_requests: u32) -> Result<bool, String> {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();

        let mut counts = self.request_counts.write().unwrap();
        let requests = counts.entry(function_id.to_string()).or_insert_with(Vec::new);

        // Remove old requests outside the window
        requests.retain(|&timestamp| now - timestamp < self.window_size_secs);

        if requests.len() >= max_requests as usize {
            return Ok(false);
        }

        requests.push(now);
        Ok(true)
    }

    pub fn get_request_count(&self, function_id: &str) -> usize {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();

        let mut counts = self.request_counts.write().unwrap();
        if let Some(requests) = counts.get_mut(function_id) {
            requests.retain(|&timestamp| now - timestamp < self.window_size_secs);
            requests.len()
        } else {
            0
        }
    }

    pub fn reset_quota(&self, function_id: &str) -> Result<(), String> {
        let mut counts = self.request_counts.write().unwrap();
        counts.remove(function_id);
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_resource_quota_creation() {
        let quota = ResourceQuota::new("fn1".to_string(), 512, 5000, 10);
        assert_eq!(quota.function_id, "fn1");
        assert_eq!(quota.memory_quota_mb, 512);
        assert_eq!(quota.concurrent_limit, 10);
    }

    #[test]
    fn test_rate_limiter_creation() {
        let limiter = RateLimiter::new(60);
        assert_eq!(limiter.window_size_secs, 60);
    }

    #[test]
    fn test_register_quota() {
        let limiter = RateLimiter::new(60);
        let quota = ResourceQuota::new("fn1".to_string(), 512, 5000, 10);
        
        assert!(limiter.register_quota(quota).is_ok());
    }

    #[test]
    fn test_register_duplicate_quota() {
        let limiter = RateLimiter::new(60);
        let quota1 = ResourceQuota::new("fn1".to_string(), 512, 5000, 10);
        let quota2 = ResourceQuota::new("fn1".to_string(), 256, 3000, 5);
        
        limiter.register_quota(quota1).unwrap();
        assert!(limiter.register_quota(quota2).is_err());
    }

    #[test]
    fn test_get_quota() {
        let limiter = RateLimiter::new(60);
        let quota = ResourceQuota::new("fn1".to_string(), 512, 5000, 10);
        
        limiter.register_quota(quota).unwrap();
        let retrieved = limiter.get_quota("fn1");
        assert!(retrieved.is_some());
    }

    #[test]
    fn test_check_rate_limit_allowed() {
        let limiter = RateLimiter::new(60);
        let allowed = limiter.check_rate_limit("fn1", 10);
        
        assert!(allowed.is_ok());
        assert!(allowed.unwrap());
    }

    #[test]
    fn test_check_rate_limit_exceeded() {
        let limiter = RateLimiter::new(60);
        
        for _ in 0..5 {
            limiter.check_rate_limit("fn1", 5).unwrap();
        }
        
        let allowed = limiter.check_rate_limit("fn1", 5);
        assert!(!allowed.unwrap());
    }

    #[test]
    fn test_get_request_count() {
        let limiter = RateLimiter::new(60);
        
        limiter.check_rate_limit("fn1", 100).unwrap();
        limiter.check_rate_limit("fn1", 100).unwrap();
        limiter.check_rate_limit("fn1", 100).unwrap();
        
        let count = limiter.get_request_count("fn1");
        assert_eq!(count, 3);
    }

    #[test]
    fn test_reset_quota() {
        let limiter = RateLimiter::new(60);
        
        limiter.check_rate_limit("fn1", 100).unwrap();
        assert_eq!(limiter.get_request_count("fn1"), 1);
        
        limiter.reset_quota("fn1").unwrap();
        assert_eq!(limiter.get_request_count("fn1"), 0);
    }
}
