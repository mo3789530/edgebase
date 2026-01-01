# Design Document: MinIO to Rustfs Migration

## Overview

This design document specifies the migration of the artifact storage backend from MinIO (S3-compatible object storage) to rustfs (local filesystem-based storage). The migration maintains API compatibility while simplifying the storage infrastructure. The system will continue to support all existing functionality including artifact upload, download, integrity verification, and presigned URL generation.

## Architecture

### Current Architecture (MinIO)

```
┌─────────────────────────────────────────────────────────┐
│                  Artifact Manager                        │
│  (Upload, Download, GenerateDownloadURL, Delete)        │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
        ┌────────────────────────┐
        │   MinIO Client         │
        │  (S3-compatible API)   │
        └────────────────────────┘
                     │
                     ▼
        ┌────────────────────────┐
        │   MinIO Server         │
        │  (Object Storage)      │
        └────────────────────────┘
```

### Target Architecture (Rustfs)

```
┌─────────────────────────────────────────────────────────┐
│                  Artifact Manager                        │
│  (Upload, Download, GenerateDownloadURL, Delete)        │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
        ┌────────────────────────┐
        │   Rustfs Backend       │
        │  (Filesystem Adapter)  │
        └────────────────────────┘
                     │
                     ▼
        ┌────────────────────────┐
        │   Local Filesystem     │
        │  (Directory Structure) │
        └────────────────────────┘
```

### Key Design Decisions

1. **Storage Backend Abstraction**: Maintain a storage interface that allows switching between MinIO and rustfs without changing the Artifact Manager logic.

2. **Directory Structure**: Organize artifacts in a hierarchical structure:
   ```
   {storage_root}/
   ├── {function_name}/
   │   ├── {version}/
   │   │   ├── function.wasm
   │   │   └── metadata.json
   ```

3. **Presigned URL Implementation**: Generate time-limited tokens that are validated server-side rather than relying on S3 presigned URLs.

4. **Integrity Verification**: Calculate and store SHA256 hashes for all artifacts, verified on download.

5. **Configuration-Driven**: Support configuration for storage path, allowing flexibility across deployment environments.

## Components and Interfaces

### 1. Storage Backend Interface

```go
type StorageBackend interface {
    // Upload stores an artifact and returns metadata
    Upload(ctx context.Context, path string, data []byte) (*ArtifactMetadata, error)
    
    // Download retrieves an artifact by path
    Download(ctx context.Context, path string) ([]byte, error)
    
    // Delete removes an artifact
    Delete(ctx context.Context, path string) error
    
    // Exists checks if an artifact exists
    Exists(ctx context.Context, path string) (bool, error)
    
    // GetSize returns the size of an artifact
    GetSize(ctx context.Context, path string) (int64, error)
    
    // List returns all artifacts in a directory
    List(ctx context.Context, prefix string) ([]string, error)
    
    // GetStats returns storage statistics
    GetStats(ctx context.Context) (*StorageStats, error)
    
    // Validate checks artifact integrity
    Validate(ctx context.Context, path string, expectedHash string) (bool, error)
}
```

### 2. Rustfs Backend Implementation

```go
type RustfsBackend struct {
    rootPath string
    mu       sync.RWMutex
}

func NewRustfsBackend(rootPath string) (*RustfsBackend, error) {
    // Create root directory if it doesn't exist
    // Validate directory is accessible
    // Return initialized backend
}

func (r *RustfsBackend) Upload(ctx context.Context, path string, data []byte) (*ArtifactMetadata, error) {
    // Create directory structure
    // Write artifact file
    // Calculate SHA256 hash
    // Store metadata
    // Return metadata
}

func (r *RustfsBackend) Download(ctx context.Context, path string) ([]byte, error) {
    // Check if file exists
    // Read file
    // Return data
}

// ... other methods
```

### 3. Presigned URL Token Manager

```go
type TokenManager struct {
    secret string
    mu     sync.RWMutex
}

type PresignedToken struct {
    Token     string
    ExpiresAt time.Time
    Path      string
}

func (t *TokenManager) GenerateToken(path string, ttl time.Duration) (*PresignedToken, error) {
    // Generate HMAC-based token
    // Set expiration time
    // Return token
}

func (t *TokenManager) ValidateToken(token string) (string, error) {
    // Verify HMAC signature
    // Check expiration
    // Return path if valid
}
```

### 4. Artifact Manager (Updated)

```go
type ArtifactManager struct {
    backend StorageBackend
    tokenMgr *TokenManager
    db      *sql.DB
}

func (a *ArtifactManager) UploadFunction(ctx context.Context, name string, version string, binary []byte) (*Function, error) {
    // Calculate hash
    // Store in backend
    // Update database
    // Return function metadata
}

func (a *ArtifactManager) GenerateDownloadURL(ctx context.Context, functionID string, ttl time.Duration) (string, error) {
    // Get artifact path from database
    // Generate presigned token
    // Return URL with token
}

func (a *ArtifactManager) Download(ctx context.Context, token string) ([]byte, error) {
    // Validate token
    // Get artifact path
    // Download from backend
    // Verify hash
    // Return data
}
```

## Data Models

### Artifact Metadata

```go
type ArtifactMetadata struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Version   string    `json:"version"`
    Hash      string    `json:"hash"`      // SHA256
    Size      int64     `json:"size"`      // bytes
    Path      string    `json:"path"`      // filesystem path
    CreatedAt time.Time `json:"created_at"`
    Status    string    `json:"status"`    // available, corrupted, missing
}
```

### Storage Statistics

```go
type StorageStats struct {
    TotalUsed      int64 `json:"total_used"`      // bytes
    ArtifactCount  int   `json:"artifact_count"`
    AvailableSpace int64 `json:"available_space"` // bytes
}
```

### Presigned Token

```go
type PresignedToken struct {
    Token     string    `json:"token"`
    ExpiresAt time.Time `json:"expires_at"`
    Path      string    `json:"path"`
}
```

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: Upload-Download Round Trip
*For any* valid WASM artifact, uploading it and then downloading it should return the exact same binary data.
**Validates: Requirements 1.2, 1.3, 5.2, 5.4**

### Property 2: Hash Integrity Verification
*For any* uploaded artifact, the calculated SHA256 hash should match the hash of the downloaded artifact.
**Validates: Requirements 3.1, 3.2, 3.4**

### Property 3: Presigned URL Expiration
*For any* presigned URL with a TTL of 0 seconds, accessing it should immediately fail with a 401 Unauthorized response.
**Validates: Requirements 2.3**

### Property 4: Artifact Deletion
*For any* uploaded artifact, after deletion, attempting to download it should fail with a 404 Not Found response.
**Validates: Requirements 4.1**

### Property 5: Directory Structure Consistency
*For any* uploaded artifact with name and version, the artifact should be stored in the path `{storage_root}/{name}/{version}/function.wasm`.
**Validates: Requirements 1.2, 7.1**

### Property 6: Storage Statistics Accuracy
*For any* set of uploaded artifacts, the total storage used reported by GetStats should equal the sum of all artifact file sizes.
**Validates: Requirements 8.1, 8.3**

### Property 7: Artifact Count Accuracy
*For any* set of uploaded and deleted artifacts, the artifact count reported by GetStats should equal the number of artifacts currently in storage.
**Validates: Requirements 8.2, 8.4**

### Property 8: Metadata Completeness
*For any* uploaded artifact, querying its metadata should return name, version, hash, size, and status fields.
**Validates: Requirements 3.4, 4.2, 4.4**

### Property 9: API Compatibility
*For any* call to the Artifact Manager interface, the response format should match the pre-migration format exactly.
**Validates: Requirements 5.1, 5.3**

### Property 10: Storage Path Configuration
*For any* configured storage path, the system should create the directory structure if it doesn't exist and use it for artifact storage.
**Validates: Requirements 7.1, 7.2, 7.3**

## Error Handling

### Error Categories

1. **File System Errors**
   - Directory not accessible
   - Insufficient disk space
   - File permission denied
   - File not found

2. **Integrity Errors**
   - Hash mismatch
   - Corrupted artifact
   - Invalid artifact format

3. **Token Errors**
   - Invalid token signature
   - Expired token
   - Token not found

4. **Configuration Errors**
   - Invalid storage path
   - Missing configuration
   - Invalid configuration values

### Error Responses

```json
{
  "error": {
    "code": "ARTIFACT_NOT_FOUND",
    "message": "Artifact not found in storage",
    "details": {
      "artifact_id": "abc123",
      "path": "/artifacts/function/v1.0"
    }
  }
}
```

## Testing Strategy

### Unit Testing

- Test StorageBackend interface implementations
- Test TokenManager token generation and validation
- Test ArtifactManager business logic
- Test error handling for various failure scenarios
- Test configuration loading and validation

### Property-Based Testing

Property-based tests will verify universal properties that should hold across all inputs:

- **Property 1 (Upload-Download Round Trip)**: Generate random binary data, upload, download, verify equality
- **Property 2 (Hash Integrity)**: Generate artifacts, verify hash consistency across upload/download
- **Property 3 (Presigned URL Expiration)**: Generate tokens with various TTLs, verify expiration behavior
- **Property 4 (Artifact Deletion)**: Upload artifacts, delete, verify they're no longer accessible
- **Property 5 (Directory Structure)**: Verify all artifacts follow the expected directory structure
- **Property 6 (Storage Statistics)**: Verify storage stats match actual artifact sizes
- **Property 7 (Artifact Count)**: Verify artifact count matches actual artifacts in storage
- **Property 8 (Metadata Completeness)**: Verify all required metadata fields are present
- **Property 9 (API Compatibility)**: Verify response formats match pre-migration expectations
- **Property 10 (Storage Path Configuration)**: Verify configuration is read and applied correctly

### Testing Framework

- **Unit Tests**: Go's built-in `testing` package
- **Property-Based Tests**: `gopter` (Go property testing library)
- **Mocking**: `testify/mock` for mocking dependencies
- **Fixtures**: Pre-generated test artifacts and configurations

### Test Configuration

- Minimum 100 iterations per property-based test
- Temporary directories for filesystem tests
- Isolated test cases to prevent interference
- Cleanup of test artifacts after each test

## Migration Strategy

### Phase 1: Implementation
1. Implement RustfsBackend with StorageBackend interface
2. Implement TokenManager for presigned URLs
3. Update ArtifactManager to use StorageBackend abstraction
4. Write comprehensive tests

### Phase 2: Validation
1. Run all property-based tests
2. Verify API compatibility
3. Test error handling scenarios
4. Performance testing

### Phase 3: Deployment
1. Deploy with rustfs backend
2. Monitor artifact operations
3. Verify no data loss
4. Decommission MinIO

## Configuration

### Environment Variables

```
STORAGE_BACKEND=rustfs              # Storage backend type
STORAGE_PATH=/data/artifacts        # Root path for artifact storage
STORAGE_TOKEN_SECRET=<secret>       # Secret for token generation
STORAGE_TOKEN_TTL=900               # Default token TTL in seconds
```

### Configuration File

```yaml
storage:
  backend: rustfs
  path: /data/artifacts
  token:
    secret: ${STORAGE_TOKEN_SECRET}
    ttl: 900
```

## Performance Considerations

### Optimization Strategies

1. **Caching**: Cache frequently accessed artifact metadata
2. **Batch Operations**: Support batch upload/download for efficiency
3. **Lazy Validation**: Validate artifacts on access rather than on startup
4. **Concurrent Access**: Use read-write locks for thread-safe operations

### Scalability

- Horizontal scaling through shared storage (NFS, etc.)
- Vertical scaling through local SSD storage
- Monitoring of disk I/O and space usage

## Security Considerations

### Access Control

- Presigned tokens are time-limited and cryptographically signed
- Token validation prevents unauthorized access
- Artifact paths are not exposed in URLs

### Data Integrity

- SHA256 hashing for artifact verification
- Hash validation on every download
- Detection of corrupted or tampered artifacts

### Storage Security

- File permissions restrict access to artifact directory
- Sensitive configuration stored in environment variables
- Audit logging of all artifact operations

