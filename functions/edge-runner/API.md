# WasmEdge Edge Runner API Documentation

## Overview

The WasmEdge Edge Runner is a lightweight WASM function execution runtime deployed on edge nodes (POPs). It receives function deployments from the Control Plane, manages a local cache of WASM artifacts, and executes functions via HTTP requests.

## HTTP Endpoints

### Function Invocation

**Endpoint:** `/*path`  
**Method:** `ANY`  
**Description:** Routes HTTP requests to registered WASM functions

**Request:**
```
GET/POST/PUT/DELETE /api/my-function
Content-Type: application/json

{
  "param1": "value1",
  "param2": "value2"
}
```

**Response (Success):**
```
HTTP/1.1 200 OK
Content-Type: application/json

{
  "result": "function output"
}
```

**Response (Timeout):**
```
HTTP/1.1 504 Gateway Timeout
Content-Type: text/plain

Gateway Timeout
```

**Response (Not Found):**
```
HTTP/1.1 404 Not Found
Content-Type: application/json

{
  "error": "Route not found"
}
```

### Metrics Endpoint

**Endpoint:** `/metrics`  
**Method:** `GET`  
**Description:** Returns Prometheus-format metrics

**Response:**
```
HTTP/1.1 200 OK
Content-Type: text/plain

# HELP edge_runner_total_invocations Total number of function invocations
# TYPE edge_runner_total_invocations counter
edge_runner_total_invocations 1000

# HELP edge_runner_total_errors Total number of errors
# TYPE edge_runner_total_errors counter
edge_runner_total_errors 5

# HELP edge_runner_average_execution_time_ms Average execution time in milliseconds
# TYPE edge_runner_average_execution_time_ms gauge
edge_runner_average_execution_time_ms 45.23

# HELP edge_runner_error_rate Error rate percentage
# TYPE edge_runner_error_rate gauge
edge_runner_error_rate 0.50

# HELP edge_runner_cache_hits Total cache hits
# TYPE edge_runner_cache_hits counter
edge_runner_cache_hits 950

# HELP edge_runner_cache_misses Total cache misses
# TYPE edge_runner_cache_misses counter
edge_runner_cache_misses 50

# HELP edge_runner_cache_hit_rate Cache hit rate percentage
# TYPE edge_runner_cache_hit_rate gauge
edge_runner_cache_hit_rate 95.00
```

## MQTT Messages

### Deployment Notification (from Control Plane)

**Topic:** `edge/deployments/{node_id}`

**Payload:**
```json
{
  "function_id": "fn_abc123",
  "version": 1,
  "entrypoint": "main",
  "memory_pages": 256,
  "max_execution_ms": 5000,
  "artifact_url": "http://minio:9000/artifacts/fn_abc123_v1.wasm",
  "sha256": "abc123def456..."
}
```

### Route Update (from Control Plane)

**Topic:** `edge/routes/{node_id}`

**Payload:**
```json
{
  "routes": [
    {
      "id": "r1",
      "host": "localhost",
      "path": "/api/my-function",
      "function_id": "fn_abc123",
      "methods": ["POST"],
      "priority": 0
    }
  ]
}
```

### Heartbeat (to Control Plane)

**Topic:** `edge/heartbeat/{node_id}`

**Payload:**
```json
{
  "node_id": "node_123",
  "pop_id": "tokyo",
  "timestamp": 1700000000,
  "status": "healthy",
  "function_count": 5,
  "cached_functions": [
    {
      "function_id": "fn_abc123",
      "version": 1,
      "status": "cached"
    }
  ],
  "metrics": {
    "memory_usage_mb": 512,
    "cpu_usage_percent": 25.5,
    "active_instances": 3,
    "total_invocations": 1000,
    "error_count": 5
  }
}
```

## Error Codes

| Status | Error Type | Retryable | Description |
|--------|-----------|-----------|-------------|
| 400 | ValidationError | No | Invalid request parameters |
| 404 | NotFoundError | No | Route or function not found |
| 500 | ExecutionError | Yes | Function execution failed |
| 502 | NetworkError | Yes | Network connectivity issue |
| 503 | ResourceError | Yes | Insufficient resources |
| 504 | TimeoutError | Yes | Function execution timeout |

## Configuration

Environment variables:

- `NODE_ID`: Unique node identifier (default: UUID)
- `POP_ID`: Point of Presence identifier (default: "default-pop")
- `CP_URL`: Control Plane URL (default: "http://localhost:8080")
- `LISTEN_ADDR`: Listen address (default: "0.0.0.0")
- `LISTEN_PORT`: Listen port (default: 3000)
- `CACHE_DIR`: WASM cache directory (default: "/tmp/wasm-cache")
- `CACHE_SIZE_GB`: Cache size in GB (default: 10)
- `MIN_HOT_INSTANCES`: Minimum hot instances per function (default: 1)
- `MAX_HOT_INSTANCES`: Maximum hot instances per function (default: 10)
- `IDLE_TIMEOUT_SECS`: Hot instance idle timeout (default: 300)
- `HEARTBEAT_INTERVAL_SECS`: Heartbeat interval (default: 30)
- `MINIO_ENDPOINT`: MinIO endpoint (default: "http://localhost:9000")
- `MINIO_ACCESS_KEY`: MinIO access key (default: "minioadmin")
- `MINIO_SECRET_KEY`: MinIO secret key (default: "minioadmin")
- `MQTT_BROKER`: MQTT broker URL (default: "mqtt://localhost:1883")

## Performance Characteristics

- **Cold Start Latency:** ~100-500ms (WASM module load + instantiation)
- **Hot Instance Latency:** ~10-50ms (reuse existing instance)
- **Memory Per Instance:** ~5-20MB (depends on WASM module size)
- **Max Concurrent Instances:** Configurable per function
- **Cache Hit Rate:** Typically 90%+ in production

## Security

- WASM modules are validated before execution
- Memory isolation enforced by WASM runtime
- API key authentication supported
- HMAC-SHA256 request signature verification
- TLS 1.3 support for MQTT connections
