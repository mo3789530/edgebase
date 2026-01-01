# Implementation Plan: MinIO to Rustfs Migration

## Overview

This implementation plan converts the feature design into a series of actionable coding tasks. Each task builds incrementally on previous tasks, with no orphaned code. The plan focuses on implementing the rustfs storage backend while maintaining API compatibility with the existing MinIO implementation.

---

## Tasks

- [ ] 1. Set up storage backend abstraction and interfaces
  - Create `internal/storage/backend.go` with StorageBackend interface
  - Define ArtifactMetadata, StorageStats, and PresignedToken types
  - Create interface documentation with method signatures
  - _Requirements: 1.1, 5.1_

- [ ] 2. Implement Rustfs backend
  - Create `internal/storage/rustfs.go` with RustfsBackend struct
  - Implement Upload method with directory structure creation
  - Implement Download method with file reading
  - Implement Delete method with file removal
  - Implement Exists method for artifact existence checking
  - Implement GetSize method for file size retrieval
  - Implement List method for directory listing
  - Implement GetStats method for storage statistics
  - Implement Validate method for hash verification
  - _Requirements: 1.2, 1.3, 4.1, 4.2, 4.4, 8.1, 8.2_

- [ ]* 2.1 Write property test for upload-download round trip
  - **Property 1: Upload-Download Round Trip**
  - **Validates: Requirements 1.2, 1.3, 5.2, 5.4**

- [ ]* 2.2 Write property test for hash integrity verification
  - **Property 2: Hash Integrity Verification**
  - **Validates: Requirements 3.1, 3.2, 3.4**

- [ ]* 2.3 Write property test for directory structure consistency
  - **Property 5: Directory Structure Consistency**
  - **Validates: Requirements 1.2, 7.1**

- [ ]* 2.4 Write property test for storage statistics accuracy
  - **Property 6: Storage Statistics Accuracy**
  - **Validates: Requirements 8.1, 8.3**

- [ ]* 2.5 Write property test for artifact count accuracy
  - **Property 7: Artifact Count Accuracy**
  - **Validates: Requirements 8.2, 8.4**

- [ ]* 2.6 Write property test for metadata completeness
  - **Property 8: Metadata Completeness**
  - **Validates: Requirements 3.4, 4.2, 4.4**

- [ ] 3. Implement token manager for presigned URLs
  - Create `internal/storage/token.go` with TokenManager struct
  - Implement GenerateToken method with HMAC-based token generation
  - Implement ValidateToken method with signature verification and expiration checking
  - Add token secret configuration support
  - _Requirements: 2.1, 2.2, 2.3_

- [ ]* 3.1 Write property test for presigned URL expiration
  - **Property 3: Presigned URL Expiration**
  - **Validates: Requirements 2.3**

- [ ]* 3.2 Write property test for API compatibility
  - **Property 9: API Compatibility**
  - **Validates: Requirements 5.1, 5.3**

- [ ] 4. Update Artifact Manager to use storage backend abstraction
  - Modify `internal/handler/artifact_handler.go` to accept StorageBackend interface
  - Update UploadFunction to use backend.Upload
  - Update Download to use backend.Download and token validation
  - Update GenerateDownloadURL to use TokenManager
  - Update DeleteFunction to use backend.Delete
  - Ensure all responses maintain pre-migration format
  - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [ ]* 4.1 Write unit tests for Artifact Manager compatibility
  - Test UploadFunction response format
  - Test GenerateDownloadURL response format
  - Test Download response format
  - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [ ] 5. Implement configuration loading for storage backend
  - Create `internal/config/storage.go` with storage configuration
  - Add support for STORAGE_BACKEND, STORAGE_PATH, STORAGE_TOKEN_SECRET environment variables
  - Implement default values (backend=rustfs, path=./artifacts)
  - Add configuration validation
  - _Requirements: 7.1, 7.2, 7.4_

- [ ]* 5.1 Write property test for storage path configuration
  - **Property 10: Storage Path Configuration**
  - **Validates: Requirements 7.1, 7.2, 7.3**

- [ ] 6. Implement storage initialization and validation
  - Create `internal/storage/init.go` with initialization logic
  - Implement directory creation if path doesn't exist
  - Implement directory accessibility validation
  - Implement artifact validation on startup
  - Add error handling for invalid paths
  - _Requirements: 1.4, 6.1, 6.2, 6.3, 6.4, 7.3, 7.4_

- [ ]* 6.1 Write unit tests for initialization and validation
  - Test directory creation
  - Test accessibility validation
  - Test error handling for invalid paths
  - Test artifact validation on startup
  - _Requirements: 1.4, 6.1, 6.2, 6.3, 6.4, 7.3, 7.4_

- [ ] 7. Implement artifact deletion with storage cleanup
  - Update DeleteFunction to remove artifact from storage
  - Update storage statistics after deletion
  - Add error handling for deletion failures
  - _Requirements: 4.1, 8.4_

- [ ]* 7.1 Write property test for artifact deletion
  - **Property 4: Artifact Deletion**
  - **Validates: Requirements 4.1**

- [ ] 8. Implement error handling and logging
  - Create `internal/storage/errors.go` with storage-specific error types
  - Implement error responses for file system errors
  - Implement error responses for integrity errors
  - Implement error responses for token errors
  - Implement error responses for configuration errors
  - Add structured logging for all operations
  - _Requirements: 3.3, 6.2, 6.3, 7.4_

- [ ]* 8.1 Write unit tests for error handling
  - Test file not found errors
  - Test hash mismatch errors
  - Test token validation errors
  - Test configuration errors
  - _Requirements: 3.3, 6.2, 6.3, 7.4_

- [ ] 9. Update database schema for artifact metadata
  - Modify functions table to include status field (available, corrupted, missing)
  - Add artifact_path column to track rustfs path
  - Add artifact_hash column for integrity verification
  - Create migration script for schema updates
  - _Requirements: 3.1, 3.4, 4.2, 6.4_

- [ ]* 9.1 Write unit tests for database schema updates
  - Test schema migration
  - Test artifact metadata storage
  - _Requirements: 3.1, 3.4, 4.2, 6.4_

- [ ] 10. Integrate storage backend into application startup
  - Update `cmd/server/main.go` to initialize storage backend
  - Load storage configuration
  - Create storage backend instance
  - Validate storage accessibility
  - Pass backend to Artifact Manager
  - _Requirements: 1.1, 1.4, 7.1, 7.2, 7.3_

- [ ]* 10.1 Write integration tests for application startup
  - Test storage backend initialization
  - Test configuration loading
  - Test error handling on startup
  - _Requirements: 1.1, 1.4, 7.1, 7.2, 7.3_

- [ ] 11. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 12. Implement artifact listing with metadata
  - Create endpoint for listing artifacts with metadata
  - Return artifact name, version, size, hash, and status
  - Implement pagination support
  - _Requirements: 4.2, 4.4_

- [ ]* 12.1 Write unit tests for artifact listing
  - Test metadata completeness
  - Test pagination
  - _Requirements: 4.2, 4.4_

- [ ] 13. Implement storage statistics endpoint
  - Create endpoint for querying storage statistics
  - Return total storage used and artifact count
  - Update statistics on artifact operations
  - _Requirements: 8.1, 8.2, 8.3, 8.4_

- [ ]* 13.1 Write unit tests for storage statistics
  - Test statistics accuracy
  - Test statistics updates
  - _Requirements: 8.1, 8.2, 8.3, 8.4_

- [ ] 14. Implement artifact status checking
  - Create method to check artifact availability
  - Detect missing artifacts
  - Detect corrupted artifacts
  - Return status in metadata queries
  - _Requirements: 6.1, 6.2, 6.3, 6.4_

- [ ]* 14.1 Write unit tests for artifact status checking
  - Test missing artifact detection
  - Test corrupted artifact detection
  - Test status reporting
  - _Requirements: 6.1, 6.2, 6.3, 6.4_

- [ ] 15. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 16. Create migration guide and documentation
  - Document storage backend abstraction
  - Document configuration options
  - Document error handling
  - Document presigned URL usage
  - Create examples for common operations
  - _Requirements: All_

- [ ] 17. Final Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

