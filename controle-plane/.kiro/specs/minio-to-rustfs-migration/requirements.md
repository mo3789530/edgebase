# Requirements Document: MinIO to Rustfs Migration

## Introduction

This document specifies the migration of artifact storage from MinIO (S3-compatible object storage) to rustfs (local filesystem-based storage). The migration aims to simplify the storage infrastructure while maintaining all existing functionality for WASM artifact management, including upload, download, integrity verification, and access control.

## Glossary

- **Artifact**: WASM binary files that are uploaded, stored, and distributed to edge nodes
- **Rustfs**: A Rust-based filesystem abstraction library for managing local file storage
- **MinIO**: S3-compatible object storage service currently used for artifact storage
- **Artifact Manager**: The component responsible for uploading, downloading, and managing WASM artifacts
- **Storage Backend**: The underlying storage system (MinIO or rustfs) used to persist artifacts
- **Integrity Verification**: Process of validating artifact authenticity using SHA256 hash
- **Presigned URL**: Time-limited URL that allows direct download without authentication
- **TTL (Time To Live)**: Expiration time for presigned URLs and temporary access tokens

## Requirements

### Requirement 1

**User Story:** As a system operator, I want to migrate from MinIO to rustfs, so that I can reduce infrastructure complexity and operational overhead.

#### Acceptance Criteria

1. WHEN the Artifact Manager initializes THEN the system SHALL connect to the rustfs storage backend instead of MinIO
2. WHEN a WASM artifact is uploaded THEN the system SHALL store it in the rustfs filesystem with proper directory structure
3. WHEN a WASM artifact is downloaded THEN the system SHALL retrieve it from rustfs and return the binary data
4. WHEN the system starts THEN the system SHALL verify that the rustfs storage directory exists and is accessible

### Requirement 2

**User Story:** As an edge node, I want to download WASM artifacts reliably, so that I can deploy functions without authentication overhead.

#### Acceptance Criteria

1. WHEN an edge node requests a download URL THEN the system SHALL generate a time-limited presigned URL for direct access
2. WHEN an edge node accesses a presigned URL THEN the system SHALL validate the token and serve the artifact if valid
3. WHEN a presigned URL expires THEN the system SHALL reject access and return a 401 Unauthorized response
4. WHEN an edge node downloads an artifact THEN the system SHALL verify the artifact integrity using SHA256 hash

### Requirement 3

**User Story:** As a system administrator, I want to ensure artifact integrity, so that corrupted or tampered artifacts are detected.

#### Acceptance Criteria

1. WHEN a WASM artifact is uploaded THEN the system SHALL calculate and store the SHA256 hash of the artifact
2. WHEN a WASM artifact is downloaded THEN the system SHALL verify the downloaded data matches the stored hash
3. IF the downloaded artifact hash does not match the stored hash THEN the system SHALL reject the download and log an error
4. WHEN querying artifact metadata THEN the system SHALL return the hash value for integrity verification

### Requirement 4

**User Story:** As a system operator, I want to manage artifact storage efficiently, so that I can control disk usage and maintain system performance.

#### Acceptance Criteria

1. WHEN a WASM artifact is deleted THEN the system SHALL remove the artifact file from rustfs storage
2. WHEN listing artifacts THEN the system SHALL return artifact metadata including name, version, size, and hash
3. WHEN the storage directory reaches capacity THEN the system SHALL log a warning and prevent new uploads
4. WHEN querying artifact information THEN the system SHALL return accurate file size in bytes

### Requirement 5

**User Story:** As a developer, I want the storage migration to be transparent, so that existing code continues to work without modification.

#### Acceptance Criteria

1. WHEN the Artifact Manager interface is called THEN the system SHALL provide the same API as before the migration
2. WHEN existing code calls UploadFunction THEN the system SHALL store the artifact in rustfs and return the same response format
3. WHEN existing code calls GenerateDownloadURL THEN the system SHALL generate a valid presigned URL for rustfs access
4. WHEN existing code calls Download THEN the system SHALL retrieve the artifact from rustfs and return the binary data

### Requirement 6

**User Story:** As a system operator, I want to ensure data consistency during migration, so that no artifacts are lost or corrupted.

#### Acceptance Criteria

1. WHEN the system starts with rustfs backend THEN the system SHALL validate all existing artifacts in storage
2. WHEN an artifact file is missing from storage THEN the system SHALL log an error and mark the artifact as unavailable
3. WHEN the system detects a corrupted artifact THEN the system SHALL log an error with details for recovery
4. WHEN querying artifact status THEN the system SHALL indicate whether the artifact is available or corrupted

### Requirement 7

**User Story:** As a system operator, I want to configure storage paths flexibly, so that I can adapt to different deployment environments.

#### Acceptance Criteria

1. WHEN the system starts THEN the system SHALL read the storage path from configuration
2. WHEN the storage path is not specified THEN the system SHALL use a default path (e.g., ./artifacts)
3. WHEN the storage path does not exist THEN the system SHALL create the directory structure automatically
4. WHEN the storage path is invalid THEN the system SHALL fail startup with a clear error message

### Requirement 8

**User Story:** As a system operator, I want to monitor storage usage, so that I can plan capacity and prevent disk exhaustion.

#### Acceptance Criteria

1. WHEN querying storage statistics THEN the system SHALL return total storage used in bytes
2. WHEN querying storage statistics THEN the system SHALL return the number of artifacts stored
3. WHEN an artifact is uploaded THEN the system SHALL update storage usage metrics
4. WHEN an artifact is deleted THEN the system SHALL update storage usage metrics

