use chrono::{DateTime, Duration, Utc};
use std::collections::HashMap;

#[derive(Debug, Clone)]
pub struct DeviceCertificate {
    pub device_id: String,
    pub certificate_pem: String,
    pub private_key_pem: String,
    pub issued_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
    pub thumbprint: String,
}

impl DeviceCertificate {
    pub fn is_valid(&self) -> bool {
        Utc::now() < self.expires_at
    }

    pub fn is_expired(&self) -> bool {
        Utc::now() >= self.expires_at
    }

    pub fn days_until_expiry(&self) -> i64 {
        (self.expires_at - Utc::now()).num_days()
    }
}

#[derive(Debug, Clone)]
pub struct JwtToken {
    pub token: String,
    pub device_id: String,
    pub issued_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
}

impl JwtToken {
    pub fn is_valid(&self) -> bool {
        Utc::now() < self.expires_at
    }

    pub fn is_expired(&self) -> bool {
        Utc::now() >= self.expires_at
    }
}

pub struct SecurityManager {
    certificates: HashMap<String, DeviceCertificate>,
    tokens: HashMap<String, JwtToken>,
    pinned_certificates: HashMap<String, String>,
}

impl SecurityManager {
    pub fn new() -> Self {
        Self {
            certificates: HashMap::new(),
            tokens: HashMap::new(),
            pinned_certificates: HashMap::new(),
        }
    }

    pub fn register_certificate(&mut self, cert: DeviceCertificate) -> Result<(), String> {
        if !cert.is_valid() {
            return Err("Certificate is expired".to_string());
        }
        self.certificates.insert(cert.device_id.clone(), cert);
        Ok(())
    }

    pub fn get_certificate(&self, device_id: &str) -> Option<DeviceCertificate> {
        self.certificates.get(device_id).cloned()
    }

    pub fn verify_certificate(&self, device_id: &str) -> Result<(), String> {
        match self.certificates.get(device_id) {
            Some(cert) => {
                if cert.is_expired() {
                    Err("Certificate is expired".to_string())
                } else {
                    Ok(())
                }
            }
            None => Err("Certificate not found".to_string()),
        }
    }

    pub fn issue_token(&mut self, device_id: String, ttl_hours: i64) -> JwtToken {
        let now = Utc::now();
        let expires_at = now + Duration::hours(ttl_hours);
        
        let token = JwtToken {
            token: format!("jwt_{}_{}_{}", device_id, now.timestamp(), ttl_hours),
            device_id: device_id.clone(),
            issued_at: now,
            expires_at,
        };
        
        self.tokens.insert(device_id, token.clone());
        token
    }

    pub fn verify_token(&self, device_id: &str, token: &str) -> Result<(), String> {
        match self.tokens.get(device_id) {
            Some(jwt) => {
                if jwt.token != token {
                    Err("Token mismatch".to_string())
                } else if jwt.is_expired() {
                    Err("Token is expired".to_string())
                } else {
                    Ok(())
                }
            }
            None => Err("Token not found".to_string()),
        }
    }

    pub fn pin_certificate(&mut self, device_id: String, thumbprint: String) {
        self.pinned_certificates.insert(device_id, thumbprint);
    }

    pub fn verify_pinned_certificate(&self, device_id: &str, thumbprint: &str) -> Result<(), String> {
        match self.pinned_certificates.get(device_id) {
            Some(pinned) => {
                if pinned == thumbprint {
                    Ok(())
                } else {
                    Err("Certificate pinning verification failed".to_string())
                }
            }
            None => Err("No pinned certificate found".to_string()),
        }
    }

    pub fn revoke_certificate(&mut self, device_id: &str) -> Result<(), String> {
        if self.certificates.remove(device_id).is_some() {
            Ok(())
        } else {
            Err("Certificate not found".to_string())
        }
    }

    pub fn revoke_token(&mut self, device_id: &str) -> Result<(), String> {
        if self.tokens.remove(device_id).is_some() {
            Ok(())
        } else {
            Err("Token not found".to_string())
        }
    }

    pub fn get_expiring_certificates(&self, days: i64) -> Vec<DeviceCertificate> {
        self.certificates
            .values()
            .filter(|cert| cert.days_until_expiry() <= days && cert.days_until_expiry() > 0)
            .cloned()
            .collect()
    }
}

impl Default for SecurityManager {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_test_certificate(device_id: &str, valid_days: i64) -> DeviceCertificate {
        let now = Utc::now();
        DeviceCertificate {
            device_id: device_id.to_string(),
            certificate_pem: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----".to_string(),
            private_key_pem: "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----".to_string(),
            issued_at: now,
            expires_at: now + Duration::days(valid_days),
            thumbprint: format!("thumbprint_{}", device_id),
        }
    }

    #[test]
    fn test_certificate_is_valid() {
        let cert = create_test_certificate("device-1", 30);
        assert!(cert.is_valid());
        assert!(!cert.is_expired());
    }

    #[test]
    fn test_certificate_is_expired() {
        let cert = create_test_certificate("device-1", -1);
        assert!(!cert.is_valid());
        assert!(cert.is_expired());
    }

    #[test]
    fn test_certificate_days_until_expiry() {
        let cert = create_test_certificate("device-1", 30);
        let days = cert.days_until_expiry();
        assert!(days >= 29 && days <= 30);
    }

    #[test]
    fn test_register_certificate() {
        let mut manager = SecurityManager::new();
        let cert = create_test_certificate("device-1", 30);
        
        let result = manager.register_certificate(cert);
        assert!(result.is_ok());
    }

    #[test]
    fn test_register_expired_certificate() {
        let mut manager = SecurityManager::new();
        let cert = create_test_certificate("device-1", -1);
        
        let result = manager.register_certificate(cert);
        assert!(result.is_err());
    }

    #[test]
    fn test_get_certificate() {
        let mut manager = SecurityManager::new();
        let cert = create_test_certificate("device-1", 30);
        manager.register_certificate(cert.clone()).unwrap();
        
        let retrieved = manager.get_certificate("device-1");
        assert!(retrieved.is_some());
        assert_eq!(retrieved.unwrap().device_id, "device-1");
    }

    #[test]
    fn test_verify_certificate() {
        let mut manager = SecurityManager::new();
        let cert = create_test_certificate("device-1", 30);
        manager.register_certificate(cert).unwrap();
        
        let result = manager.verify_certificate("device-1");
        assert!(result.is_ok());
    }

    #[test]
    fn test_verify_nonexistent_certificate() {
        let manager = SecurityManager::new();
        let result = manager.verify_certificate("device-1");
        assert!(result.is_err());
    }

    #[test]
    fn test_issue_token() {
        let mut manager = SecurityManager::new();
        let token = manager.issue_token("device-1".to_string(), 24);
        
        assert_eq!(token.device_id, "device-1");
        assert!(token.is_valid());
    }

    #[test]
    fn test_verify_token() {
        let mut manager = SecurityManager::new();
        let token = manager.issue_token("device-1".to_string(), 24);
        
        let result = manager.verify_token("device-1", &token.token);
        assert!(result.is_ok());
    }

    #[test]
    fn test_verify_invalid_token() {
        let mut manager = SecurityManager::new();
        manager.issue_token("device-1".to_string(), 24);
        
        let result = manager.verify_token("device-1", "invalid_token");
        assert!(result.is_err());
    }

    #[test]
    fn test_pin_certificate() {
        let mut manager = SecurityManager::new();
        manager.pin_certificate("device-1".to_string(), "thumbprint_123".to_string());
        
        let result = manager.verify_pinned_certificate("device-1", "thumbprint_123");
        assert!(result.is_ok());
    }

    #[test]
    fn test_verify_pinned_certificate_mismatch() {
        let mut manager = SecurityManager::new();
        manager.pin_certificate("device-1".to_string(), "thumbprint_123".to_string());
        
        let result = manager.verify_pinned_certificate("device-1", "thumbprint_456");
        assert!(result.is_err());
    }

    #[test]
    fn test_revoke_certificate() {
        let mut manager = SecurityManager::new();
        let cert = create_test_certificate("device-1", 30);
        manager.register_certificate(cert).unwrap();
        
        let result = manager.revoke_certificate("device-1");
        assert!(result.is_ok());
        
        let retrieved = manager.get_certificate("device-1");
        assert!(retrieved.is_none());
    }

    #[test]
    fn test_revoke_nonexistent_certificate() {
        let mut manager = SecurityManager::new();
        let result = manager.revoke_certificate("device-1");
        assert!(result.is_err());
    }

    #[test]
    fn test_revoke_token() {
        let mut manager = SecurityManager::new();
        manager.issue_token("device-1".to_string(), 24);
        
        let result = manager.revoke_token("device-1");
        assert!(result.is_ok());
    }

    #[test]
    fn test_get_expiring_certificates() {
        let mut manager = SecurityManager::new();
        
        let cert1 = create_test_certificate("device-1", 5);
        let cert2 = create_test_certificate("device-2", 15);
        let cert3 = create_test_certificate("device-3", 60);
        
        manager.register_certificate(cert1).unwrap();
        manager.register_certificate(cert2).unwrap();
        manager.register_certificate(cert3).unwrap();
        
        let expiring = manager.get_expiring_certificates(10);
        assert!(expiring.len() >= 1);
    }

    #[test]
    fn test_multiple_certificates() {
        let mut manager = SecurityManager::new();
        
        for i in 0..5 {
            let cert = create_test_certificate(&format!("device-{}", i), 30);
            manager.register_certificate(cert).unwrap();
        }
        
        for i in 0..5 {
            let result = manager.verify_certificate(&format!("device-{}", i));
            assert!(result.is_ok());
        }
    }

    #[test]
    fn test_token_expiry() {
        let mut manager = SecurityManager::new();
        let token = manager.issue_token("device-1".to_string(), 0);
        
        assert!(!token.is_valid());
        assert!(token.is_expired());
    }

    #[test]
    fn test_security_manager_default() {
        let manager = SecurityManager::default();
        let result = manager.verify_certificate("device-1");
        assert!(result.is_err());
    }

    #[test]
    fn test_certificate_thumbprint() {
        let cert = create_test_certificate("device-1", 30);
        assert_eq!(cert.thumbprint, "thumbprint_device-1");
    }

    #[test]
    fn test_jwt_token_structure() {
        let mut manager = SecurityManager::new();
        let token = manager.issue_token("device-1".to_string(), 24);
        
        assert!(token.token.contains("jwt_"));
        assert!(token.token.contains("device-1"));
        assert_eq!(token.device_id, "device-1");
    }

    #[test]
    fn test_certificate_pem_format() {
        let cert = create_test_certificate("device-1", 30);
        assert!(cert.certificate_pem.contains("BEGIN CERTIFICATE"));
        assert!(cert.private_key_pem.contains("BEGIN PRIVATE KEY"));
    }

    #[test]
    fn test_revoke_and_reregister() {
        let mut manager = SecurityManager::new();
        let cert = create_test_certificate("device-1", 30);
        
        manager.register_certificate(cert.clone()).unwrap();
        manager.revoke_certificate("device-1").unwrap();
        
        let result = manager.register_certificate(cert);
        assert!(result.is_ok());
    }

    #[test]
    fn test_pin_multiple_certificates() {
        let mut manager = SecurityManager::new();
        
        manager.pin_certificate("device-1".to_string(), "thumbprint_1".to_string());
        manager.pin_certificate("device-2".to_string(), "thumbprint_2".to_string());
        
        assert!(manager.verify_pinned_certificate("device-1", "thumbprint_1").is_ok());
        assert!(manager.verify_pinned_certificate("device-2", "thumbprint_2").is_ok());
    }
}
