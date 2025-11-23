# Control Plane Consolidation

## Summary

コントロールプレーンを集約しました。分散していたRust実装(`functions/control-plane`)をGo実装(`platform/control-plane`)に統合しました。

## Changes Made

### 1. Platform Control Plane (Go) - Enhanced

#### Handler (`internal/handler/handler.go`)
- `/api/v1/functions` - POST: Create function
- `/api/v1/functions/:id` - GET: Get function metadata
- `/api/v1/functions/:id/upload` - POST: Upload artifact
- `/api/v1/artifacts/:id/:version` - GET: Download artifact
- `/api/v1/functions/:function_id/deploy/:node_id` - POST: Deploy function
- `/api/v1/routes` - POST/GET: Create and list routes

#### Service Layer (`internal/service/`)
- **ArtifactService**: 
  - `CreateFunction()` - Create new function
  - `UploadArtifact()` - Upload function artifact
  - `GetArtifactData()` - Download artifact data

- **SyncService**:
  - `QueueDeployment()` - Queue function for deployment
  - `CreateRoute()` - Create routing rule
  - `ListRoutes()` - List all routes

#### Model (`internal/model/model.go`)
- **Function**: Added fields
  - `Entrypoint` - Function entry point
  - `Runtime` - Runtime type
  - `MemoryPages` - Memory allocation
  - `MaxExecutionMs` - Execution timeout

- **NodeFunctionDeployment**: Restructured
  - Added `ID` primary key
  - Added `Status` field
  - Added timestamps

#### Repository (`internal/repository/`)
- **FunctionRepository**: Added `Update()` method

#### Storage (`internal/storage/minio.go`)
- Added `Download()` method for artifact retrieval

### 2. Deprecated

The Rust implementation at `functions/control-plane/` should be removed as all functionality has been migrated to the Go implementation.

## Migration Path

1. All function management APIs are now available in `platform/control-plane`
2. Deployment and routing features are integrated
3. Artifact storage uses MinIO (consistent with existing setup)
4. Database models support all required fields

## Next Steps

1. Remove `functions/control-plane/` directory
2. Update documentation to reference only `platform/control-plane`
3. Update deployment configurations
4. Run integration tests to verify all endpoints work correctly
