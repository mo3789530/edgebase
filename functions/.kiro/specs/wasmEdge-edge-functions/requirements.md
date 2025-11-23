# Requirements Document: WasmEdge Edge Functions Platform

## Introduction

The WasmEdge Edge Functions Platform is a distributed system designed to execute short-lived HTTP handler functions (WASM modules) on edge nodes (POPs) with low latency and strong security guarantees. The system follows a pull-based architecture where the Control Plane manages function metadata and artifacts, while Edge Runners autonomously pull and cache WASM modules from an artifact store (MinIO/S3). The platform provides hot instance pooling for performance optimization, comprehensive observability through Prometheus metrics, and strict resource isolation through WASM sandboxing.

## Glossary

- **Control Plane (CP)**: Central management service responsible for function registration, versioning, artifact storage, deployment orchestration, and notification delivery
- **Edge Node / Edge Runner**: Lightweight agent deployed on POP (Point of Presence) that executes WASM functions via embedded WasmEdge runtime
- **Artifact Store**: MinIO or S3-compatible object storage containing WASM binary artifacts
- **Function**: A deployable WASM module with associated metadata (memory limits, execution timeout, entrypoint)
- **Route**: HTTP path-to-function mapping that determines which function handles a given request
- **Hot Instance**: A pre-instantiated WASM VM that remains in memory for rapid reuse across multiple invocations
- **Cold Start**: The process of creating a new WASM instance from scratch, including module loading and instantiation
- **WASI**: WebAssembly System Interface - standardized capability-based interface for WASM modules to interact with host systems
- **Heartbeat**: Periodic status message from Edge Runner to Control Plane containing node health and function inventory
- **Deployment**: The process of making a specific function version available on target edge nodes
- **POP Selector**: Expression-based rule (e.g., region=='tokyo') that determines which edge nodes receive a function deployment

## Requirements

### Requirement 1: Function Registration and Artifact Management

**User Story:** As a developer, I want to register WASM functions with the Control Plane, so that they can be deployed and executed on edge nodes.

#### Acceptance Criteria

1. WHEN a developer submits a function registration request with name, entrypoint, memory limit, and execution timeout THEN the Control Plane SHALL create a function record in the database and return a presigned artifact upload URL
   - _Requirements: 6.1_

2. WHEN a developer uploads a WASM artifact to the presigned URL THEN the Control Plane SHALL calculate the SHA256 checksum and store the artifact URL and checksum in the function metadata
   - _Requirements: 6.1_

3. WHEN a function is registered with invalid parameters (negative memory pages, zero timeout, missing entrypoint) THEN the Control Plane SHALL reject the registration and return a validation error
   - _Requirements: 6.1_

4. WHEN a function artifact is uploaded THEN the Control Plane SHALL verify the artifact is valid WASM bytecode before persisting the metadata
   - _Requirements: 6.1_

### Requirement 2: Function Deployment to Edge Nodes

**User Story:** As a platform operator, I want to deploy function versions to specific edge nodes based on geographic or performance criteria, so that functions are available where needed.

#### Acceptance Criteria

1. WHEN a deployment request specifies a function version and target POP selector THEN the Control Plane SHALL identify matching edge nodes and send deployment notifications containing function_id, version, artifact_url, sha256, memory_pages, and max_execution_ms
   - _Requirements: 6.2_

2. WHEN an Edge Runner receives a deployment notification THEN the Edge Runner SHALL pull the WASM artifact from the artifact store and verify the SHA256 checksum matches the expected value
   - _Requirements: 6.2_

3. WHEN an Edge Runner successfully caches a function THEN the Edge Runner SHALL report the deployment status back to the Control Plane with state='cached'
   - _Requirements: 6.2_

4. WHEN an Edge Runner fails to verify the SHA256 checksum of a downloaded artifact THEN the Edge Runner SHALL reject the artifact, report an error to the Control Plane, and not add it to the local cache
   - _Requirements: 6.2_

### Requirement 3: HTTP Request Routing and Execution

**User Story:** As an end user, I want to invoke edge functions via HTTP requests, so that I can execute serverless workloads at the edge.

#### Acceptance Criteria

1. WHEN an HTTP request arrives at an Edge Runner with a path matching a registered route THEN the Edge Runner SHALL look up the corresponding function and dispatch the request to the WASM handler
   - _Requirements: 4.2.2_

2. WHEN a function is available in the hot instance pool THEN the Edge Runner SHALL reuse the existing WASM instance without creating a new one
   - _Requirements: 4.5.1_

3. WHEN a function is not in the hot instance pool THEN the Edge Runner SHALL create a new WASM instance, execute the function, and manage the instance according to the LRU eviction policy
   - _Requirements: 4.5.1_

4. WHEN a function execution exceeds the max_execution_ms timeout THEN the Edge Runner SHALL terminate the execution and return a 504 Gateway Timeout response
   - _Requirements: 4.8_

5. WHEN a function execution completes successfully THEN the Edge Runner SHALL return the function's response as the HTTP response body with appropriate status code
   - _Requirements: 4.2.2_

### Requirement 4: Hot Instance Pool Management

**User Story:** As a platform architect, I want to manage a pool of pre-instantiated WASM instances per function, so that repeated invocations have minimal latency.

#### Acceptance Criteria

1. WHEN a function is deployed to an Edge Runner THEN the Edge Runner SHALL create at least min_hot_instances (default: 1) pre-instantiated WASM instances and keep them ready for execution
   - _Requirements: 4.5.1_

2. WHEN the hot instance pool size reaches max_hot_instances THEN the Edge Runner SHALL apply LRU eviction to remove the least recently used instance before creating new ones
   - _Requirements: 4.5.1_

3. WHEN a hot instance remains idle for longer than idle_timeout (default: 5 minutes) THEN the Edge Runner SHALL destroy the instance to free memory
   - _Requirements: 4.5.1_

4. WHEN memory pressure exceeds a threshold (e.g., 80% of available memory) THEN the Edge Runner SHALL evict hot instances using LRU policy until memory usage drops below the threshold
   - _Requirements: 4.5.1_

### Requirement 5: Local WASM Cache Management

**User Story:** As an Edge Runner operator, I want to efficiently cache WASM artifacts locally, so that functions can be executed quickly without repeated downloads.

#### Acceptance Criteria

1. WHEN a function is deployed to an Edge Runner THEN the Edge Runner SHALL store the WASM artifact in a local cache directory with the path pattern /var/cache/wasm/{function_id}/{version}.wasm
   - _Requirements: 4.5.2_

2. WHEN the local cache exceeds the maximum size limit (e.g., 10 GB) THEN the Edge Runner SHALL apply LRU eviction to remove the least recently used WASM files until the cache size is within limits
   - _Requirements: 4.5.2_

3. WHEN a WASM artifact is retrieved from the local cache THEN the Edge Runner SHALL verify the SHA256 checksum against the stored metadata to ensure integrity
   - _Requirements: 4.4.3_

4. WHEN a new version of a function is deployed THEN the Edge Runner SHALL treat it as a separate cache entry and not overwrite the previous version until explicitly undeployed
   - _Requirements: 4.5.2_

### Requirement 6: Heartbeat and Synchronization

**User Story:** As the Control Plane, I want to maintain awareness of edge node status and pending deployments, so that I can orchestrate function distribution and detect failures.

#### Acceptance Criteria

1. WHEN an Edge Runner is operational THEN the Edge Runner SHALL send a heartbeat message to the Control Plane every 30 seconds containing node_id, status, cpu_percent, mem_bytes, and a list of cached functions with their versions and states
   - _Requirements: 4.2.3_

2. WHEN the Control Plane receives a heartbeat from an Edge Runner THEN the Control Plane SHALL update the last_heartbeat timestamp and check for pending deployment notifications
   - _Requirements: 4.2.3_

3. WHEN the Control Plane has pending deployments for an Edge Runner THEN the Control Plane SHALL include deployment instructions in the heartbeat response containing function_id, version, artifact_url, sha256, memory_pages, and max_execution_ms
   - _Requirements: 4.2.3_

4. WHEN an Edge Runner fails to send a heartbeat for longer than a timeout period (e.g., 2 minutes) THEN the Control Plane SHALL mark the node as offline and stop routing requests to it
   - _Requirements: 4.2.3_

### Requirement 7: WASM Execution Isolation and Security

**User Story:** As a security architect, I want to enforce strict resource limits and capability restrictions on WASM execution, so that functions cannot escape the sandbox or consume excessive resources.

#### Acceptance Criteria

1. WHEN a WASM function is instantiated THEN the Edge Runner SHALL enforce the memory_pages limit by configuring the WASM linear memory to not exceed memory_pages * 64 KiB
   - _Requirements: 4.4.2_

2. WHEN a WASM function attempts to access WASI capabilities outside the whitelist (e.g., raw socket access, filesystem write, process execution) THEN the Edge Runner SHALL deny the access and the function execution SHALL fail with an error
   - _Requirements: 4.4.2_

3. WHEN a WASM function is executed THEN the Edge Runner SHALL enforce the max_execution_ms timeout using host-level context timeout to prevent runaway execution
   - _Requirements: 4.4.2_

4. WHEN a WASM function attempts to make an HTTP request via the http_fetch host function THEN the Edge Runner SHALL validate the target URL against a whitelist and reject requests to disallowed destinations
   - _Requirements: 4.4.2_

### Requirement 8: Observability and Metrics

**User Story:** As an operator, I want to collect comprehensive metrics and logs from edge nodes and function executions, so that I can monitor system health and debug issues.

#### Acceptance Criteria

1. WHEN a function is invoked on an Edge Runner THEN the Edge Runner SHALL increment the wasm_invoke_count_total metric and record the execution latency in the wasm_invoke_latency_seconds histogram, both tagged with function_id
   - _Requirements: 10_

2. WHEN a function execution fails THEN the Edge Runner SHALL increment the wasm_invoke_errors_total metric tagged with function_id and error_code
   - _Requirements: 10_

3. WHEN a WASM artifact is retrieved from the local cache THEN the Edge Runner SHALL increment the wasm_cache_hits_total metric; when a cache miss occurs THEN the Edge Runner SHALL increment the wasm_cache_misses_total metric
   - _Requirements: 10_

4. WHEN a function is executed THEN the Edge Runner SHALL emit structured JSON logs containing request_id, function_id, node_id, start_timestamp, end_timestamp, execution_status, and error_message (if applicable)
   - _Requirements: 10_

5. WHEN the Edge Runner is running THEN the Edge Runner SHALL expose Prometheus metrics at the endpoint :9090/metrics in Prometheus text format
   - _Requirements: 10_

### Requirement 9: Authentication and Authorization

**User Story:** As a security administrator, I want to authenticate developers and edge nodes, and enforce role-based access control, so that only authorized parties can register functions and manage deployments.

#### Acceptance Criteria

1. WHEN a developer submits an API request to the Control Plane THEN the Control Plane SHALL validate the JWT token in the Authorization header and verify the developer's project membership
   - _Requirements: 4.4.1_

2. WHEN an Edge Runner connects to the Control Plane THEN the Edge Runner SHALL present a valid mTLS certificate and the Control Plane SHALL verify the certificate and check the node_id against an allowlist
   - _Requirements: 4.4.1_

3. WHEN a developer attempts to deploy a function to a project they do not own THEN the Control Plane SHALL reject the request with a 403 Forbidden response
   - _Requirements: 4.4.1_

### Requirement 10: Artifact Integrity Verification

**User Story:** As a platform operator, I want to ensure that WASM artifacts have not been tampered with or corrupted, so that functions execute as intended.

#### Acceptance Criteria

1. WHEN a WASM artifact is uploaded to the Control Plane THEN the Control Plane SHALL calculate the SHA256 checksum and store it in the function metadata
   - _Requirements: 4.4.3_

2. WHEN an Edge Runner downloads a WASM artifact from the artifact store THEN the Edge Runner SHALL calculate the SHA256 checksum of the downloaded file and compare it against the expected checksum from the deployment notification
   - _Requirements: 4.4.3_

3. WHEN the calculated SHA256 checksum does not match the expected checksum THEN the Edge Runner SHALL reject the artifact, delete the corrupted file, and report an error to the Control Plane
   - _Requirements: 4.4.3_

### Requirement 11: Graceful Degradation and Error Handling

**User Story:** As a platform operator, I want the system to handle failures gracefully and provide meaningful error messages, so that issues can be diagnosed and resolved quickly.

#### Acceptance Criteria

1. WHEN an Edge Runner fails to download a WASM artifact from the artifact store THEN the Edge Runner SHALL retry with exponential backoff (initial delay: 1s, max delay: 60s) up to a maximum of 5 attempts
   - _Requirements: 9_

2. WHEN an Edge Runner loses connectivity to the Control Plane THEN the Edge Runner SHALL continue executing cached functions and attempt to reconnect with exponential backoff
   - _Requirements: 9_

3. WHEN a function execution fails due to a WASM runtime error THEN the Edge Runner SHALL return a 500 Internal Server Error response with an error message
   - _Requirements: 11_

4. WHEN the Control Plane receives an error report from an Edge Runner THEN the Control Plane SHALL log the error with full context (node_id, function_id, timestamp, error_message) for audit and debugging purposes
   - _Requirements: 11_

### Requirement 12: Route Management

**User Story:** As a developer, I want to define HTTP routes that map to specific functions, so that requests are correctly dispatched to the appropriate handler.

#### Acceptance Criteria

1. WHEN a developer creates a route with host, path pattern, function_id, HTTP methods, and POP selector THEN the Control Plane SHALL store the route in the database and propagate it to matching edge nodes
   - _Requirements: 4.3_

2. WHEN an HTTP request arrives at an Edge Runner THEN the Edge Runner SHALL match the request's host and path against registered routes in priority order and dispatch to the highest-priority matching function
   - _Requirements: 4.3_

3. WHEN no route matches an incoming HTTP request THEN the Edge Runner SHALL return a 404 Not Found response
   - _Requirements: 4.3_

4. WHEN multiple routes have the same host and path but different methods THEN the Edge Runner SHALL select the route matching the request's HTTP method
   - _Requirements: 4.3_

### Requirement 13: Function Versioning and Rollback

**User Story:** As a developer, I want to manage multiple versions of functions and roll back to previous versions if needed, so that I can safely deploy updates and recover from issues.

#### Acceptance Criteria

1. WHEN a new function version is deployed THEN the Control Plane SHALL maintain the previous version in the artifact store and allow edge nodes to cache both versions simultaneously
   - _Requirements: 4.2.1_

2. WHEN a rollback is requested for a function THEN the Control Plane SHALL send an undeploy notification for the current version and a deploy notification for the target version to affected edge nodes
   - _Requirements: 15_

3. WHEN an Edge Runner receives an undeploy notification THEN the Edge Runner SHALL remove the function from the hot instance pool, destroy all instances, and optionally remove the artifact from the local cache
   - _Requirements: 15_

### Requirement 14: Resource Quotas and Rate Limiting

**User Story:** As a platform operator, I want to enforce resource quotas per tenant to prevent resource exhaustion, so that the platform remains stable and fair for all users.

#### Acceptance Criteria

1. WHEN a function is invoked THEN the Edge Runner SHALL check if the tenant has exceeded their concurrent execution quota (e.g., max 100 concurrent executions) and reject the request with a 429 Too Many Requests response if the quota is exceeded
   - _Requirements: 4.4.2_

2. WHEN a tenant's invocation rate exceeds their per-minute quota (e.g., max 10,000 invocations/minute) THEN the Edge Runner SHALL reject excess requests with a 429 Too Many Requests response
   - _Requirements: 4.4.2_

3. WHEN a tenant's total CPU time usage exceeds their daily quota (e.g., max 1 hour/day) THEN the Control Plane SHALL prevent new function invocations for that tenant until the quota resets
   - _Requirements: 4.4.2_
