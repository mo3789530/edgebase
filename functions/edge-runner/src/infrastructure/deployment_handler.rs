use crate::domain::{Function, Deployment, DeploymentStatus};
use crate::infrastructure::{LocalFunctionRepository, LocalDeploymentRepository, ArtifactDownloader};
use std::path::Path;
use uuid::Uuid;

pub struct DeploymentHandler {
    function_repo: std::sync::Arc<dyn LocalFunctionRepository>,
    deployment_repo: std::sync::Arc<dyn LocalDeploymentRepository>,
    artifact_downloader: ArtifactDownloader,
    cache_dir: String,
}

impl DeploymentHandler {
    pub fn new(
        function_repo: std::sync::Arc<dyn LocalFunctionRepository>,
        deployment_repo: std::sync::Arc<dyn LocalDeploymentRepository>,
        artifact_downloader: ArtifactDownloader,
        cache_dir: String,
    ) -> Self {
        DeploymentHandler {
            function_repo,
            deployment_repo,
            artifact_downloader,
            cache_dir,
        }
    }

    pub async fn handle_deployment_notification(
        &self,
        function_id: String,
        version: u32,
        entrypoint: String,
        memory_pages: u32,
        max_execution_ms: u32,
        artifact_url: String,
        sha256: String,
    ) -> Result<DeploymentStatus, String> {
        // Create deployment record
        let deployment_id = Uuid::new_v4().to_string();
        let deployment = Deployment::new(deployment_id.clone(), function_id.clone());
        self.deployment_repo.create(deployment)?;

        // Download artifact
        let artifact_path = Path::new(&self.cache_dir).join(format!("{}.wasm", function_id));
        self.artifact_downloader
            .download(&artifact_url, &artifact_path)
            .await?;

        // Verify checksum
        self.artifact_downloader.verify_checksum(&artifact_path, &sha256)?;

        // Validate WASM
        self.artifact_downloader.validate_wasm(&artifact_path)?;

        // Create function record
        let function = Function::new(
            function_id.clone(),
            format!("fn_{}", version),
            version,
            entrypoint,
            memory_pages,
            max_execution_ms,
            artifact_url,
            sha256,
        )?;

        self.function_repo.create(function)?;

        // Update deployment status
        self.deployment_repo
            .update_status(&deployment_id, DeploymentStatus::Cached)?;

        Ok(DeploymentStatus::Cached)
    }

    pub fn get_deployment_status(&self, deployment_id: &str) -> Result<Option<DeploymentStatus>, String> {
        self.deployment_repo
            .get(deployment_id)
            .map(|opt| opt.map(|d| d.status))
    }

    pub fn list_deployments(&self, function_id: &str) -> Result<Vec<Deployment>, String> {
        self.deployment_repo.list_by_function(function_id)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::infrastructure::{InMemoryLocalFunctionRepository, InMemoryLocalDeploymentRepository};

    #[test]
    fn test_deployment_handler_creation() {
        let func_repo = std::sync::Arc::new(InMemoryLocalFunctionRepository::new());
        let dep_repo = std::sync::Arc::new(InMemoryLocalDeploymentRepository::new());
        let downloader = ArtifactDownloader::new(
            "http://localhost:9000".to_string(),
            "minioadmin".to_string(),
            "minioadmin".to_string(),
        );

        let handler = DeploymentHandler::new(
            func_repo,
            dep_repo,
            downloader,
            "/tmp/cache".to_string(),
        );

        assert_eq!(handler.cache_dir, "/tmp/cache");
    }

    #[test]
    fn test_get_deployment_status_not_found() {
        let func_repo = std::sync::Arc::new(InMemoryLocalFunctionRepository::new());
        let dep_repo = std::sync::Arc::new(InMemoryLocalDeploymentRepository::new());
        let downloader = ArtifactDownloader::new(
            "http://localhost:9000".to_string(),
            "minioadmin".to_string(),
            "minioadmin".to_string(),
        );

        let handler = DeploymentHandler::new(
            func_repo,
            dep_repo,
            downloader,
            "/tmp/cache".to_string(),
        );

        let status = handler.get_deployment_status("nonexistent");
        assert!(status.is_ok());
        assert!(status.unwrap().is_none());
    }

    #[test]
    fn test_list_deployments_empty() {
        let func_repo = std::sync::Arc::new(InMemoryLocalFunctionRepository::new());
        let dep_repo = std::sync::Arc::new(InMemoryLocalDeploymentRepository::new());
        let downloader = ArtifactDownloader::new(
            "http://localhost:9000".to_string(),
            "minioadmin".to_string(),
            "minioadmin".to_string(),
        );

        let handler = DeploymentHandler::new(
            func_repo,
            dep_repo,
            downloader,
            "/tmp/cache".to_string(),
        );

        let deployments = handler.list_deployments("fn1");
        assert!(deployments.is_ok());
        assert_eq!(deployments.unwrap().len(), 0);
    }
}
