use sha2::{Sha256, Digest};
use std::path::Path;
use tokio::fs;
use std::io::Write;

pub struct ArtifactDownloader {
    minio_endpoint: String,
    access_key: String,
    secret_key: String,
}

impl ArtifactDownloader {
    pub fn new(endpoint: String, access_key: String, secret_key: String) -> Self {
        ArtifactDownloader {
            minio_endpoint: endpoint,
            access_key,
            secret_key,
        }
    }

    pub async fn download(&self, artifact_url: &str, dest_path: &Path) -> Result<String, String> {
        let response = reqwest::Client::new()
            .get(artifact_url)
            .send()
            .await
            .map_err(|e| format!("Download failed: {}", e))?;

        if !response.status().is_success() {
            return Err(format!("HTTP error: {}", response.status()));
        }

        let bytes = response.bytes().await
            .map_err(|e| format!("Failed to read response: {}", e))?;

        let sha256 = self.calculate_sha256(&bytes);

        fs::write(dest_path, &bytes)
            .await
            .map_err(|e| format!("Failed to write file: {}", e))?;

        Ok(sha256)
    }

    pub fn verify_checksum(&self, file_path: &Path, expected_sha256: &str) -> Result<(), String> {
        let bytes = std::fs::read(file_path)
            .map_err(|e| format!("Failed to read file: {}", e))?;

        let calculated = self.calculate_sha256(&bytes);
        if calculated != expected_sha256 {
            return Err(format!(
                "Checksum mismatch: expected {}, got {}",
                expected_sha256, calculated
            ));
        }

        Ok(())
    }

    pub fn calculate_sha256(&self, data: &[u8]) -> String {
        let mut hasher = Sha256::new();
        hasher.update(data);
        format!("{:x}", hasher.finalize())
    }

    pub fn validate_wasm(&self, file_path: &Path) -> Result<(), String> {
        let bytes = std::fs::read(file_path)
            .map_err(|e| format!("Failed to read file: {}", e))?;

        if bytes.len() < 4 {
            return Err("File too small to be WASM".to_string());
        }

        if &bytes[0..4] != b"\0asm" {
            return Err("Invalid WASM magic number".to_string());
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use tempfile::TempDir;

    #[test]
    fn test_calculate_sha256() {
        let downloader = ArtifactDownloader::new(
            "http://localhost:9000".to_string(),
            "minioadmin".to_string(),
            "minioadmin".to_string(),
        );

        let data = b"test data";
        let hash = downloader.calculate_sha256(data);
        assert_eq!(hash.len(), 64);
    }

    #[test]
    fn test_verify_checksum_success() {
        let downloader = ArtifactDownloader::new(
            "http://localhost:9000".to_string(),
            "minioadmin".to_string(),
            "minioadmin".to_string(),
        );

        let temp_dir = TempDir::new().unwrap();
        let file_path = temp_dir.path().join("test.bin");
        let data = b"test data";
        fs::write(&file_path, data).unwrap();

        let hash = downloader.calculate_sha256(data);
        let result = downloader.verify_checksum(&file_path, &hash);
        assert!(result.is_ok());
    }

    #[test]
    fn test_verify_checksum_mismatch() {
        let downloader = ArtifactDownloader::new(
            "http://localhost:9000".to_string(),
            "minioadmin".to_string(),
            "minioadmin".to_string(),
        );

        let temp_dir = TempDir::new().unwrap();
        let file_path = temp_dir.path().join("test.bin");
        fs::write(&file_path, b"test data").unwrap();

        let result = downloader.verify_checksum(&file_path, "wronghash");
        assert!(result.is_err());
    }

    #[test]
    fn test_validate_wasm_valid() {
        let downloader = ArtifactDownloader::new(
            "http://localhost:9000".to_string(),
            "minioadmin".to_string(),
            "minioadmin".to_string(),
        );

        let temp_dir = TempDir::new().unwrap();
        let file_path = temp_dir.path().join("test.wasm");
        let mut wasm_data = vec![0u8; 100];
        wasm_data[0..4].copy_from_slice(b"\0asm");
        fs::write(&file_path, wasm_data).unwrap();

        let result = downloader.validate_wasm(&file_path);
        assert!(result.is_ok());
    }

    #[test]
    fn test_validate_wasm_invalid_magic() {
        let downloader = ArtifactDownloader::new(
            "http://localhost:9000".to_string(),
            "minioadmin".to_string(),
            "minioadmin".to_string(),
        );

        let temp_dir = TempDir::new().unwrap();
        let file_path = temp_dir.path().join("test.bin");
        fs::write(&file_path, b"invalid").unwrap();

        let result = downloader.validate_wasm(&file_path);
        assert!(result.is_err());
    }
}
