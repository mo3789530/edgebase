# Control Plane Specification

## Current Features ✅

### 1. Node Management
- Node registration with auto-generated UUID
- Heartbeat mechanism for status tracking
- Sync information retrieval
- Sync acknowledgment

### 2. Function (WASM) Management
- Create, read, delete functions
- Upload/download artifacts to MinIO
- Function metadata (entrypoint, runtime, memory, execution time)

### 3. Deployment
- Deploy functions to nodes
- Track deployment status

### 4. Schema Management
- Register and list schemas
- Schema versioning

### 5. Telemetry & Sync
- Collect telemetry data from devices
- Command execution tracking
- Sync status monitoring
- Device registration

### 6. Storage
- MinIO integration for artifact storage
- PostgreSQL for metadata

### 7. Authentication & Authorization ✅ (NEW)
- JWT-based token authentication
- Node registration returns token
- Protected endpoints require Bearer token

---

## Missing Features ❌

### 2. Error Handling & Logging
**Priority: HIGH**
- Global error handler middleware
- Structured logging (JSON format)
- Request/response logging
- Error tracking with request IDs

### 3. Input Validation
**Priority: HIGH**
- Request body validation
- Field constraints (min/max length, format)
- Custom validation rules
- Error response standardization

### 4. Rate Limiting
**Priority: MEDIUM**
- Per-node rate limiting
- Per-IP rate limiting
- Configurable limits
- Rate limit headers in responses

### 5. Caching
**Priority: MEDIUM**
- Cache frequently accessed data (functions, schemas)
- TTL-based cache invalidation
- Cache warming on startup

### 6. Health Check Endpoint
**Priority: HIGH**
- `GET /health` - basic health status
- `GET /health/ready` - readiness probe
- `GET /health/live` - liveness probe
- Database connectivity check

### 7. Monitoring & Metrics
**Priority: MEDIUM**
- Prometheus metrics export
- Request latency tracking
- Error rate monitoring
- Active connections tracking

### 8. Graceful Shutdown
**Priority: HIGH**
- Signal handling (SIGTERM, SIGINT)
- In-flight request completion
- Database connection cleanup
- MQTT client cleanup

### 9. Pagination
**Priority: MEDIUM**
- List endpoints support limit/offset
- Consistent pagination format
- Total count in responses

### 10. Request ID Tracking
**Priority: MEDIUM**
- Generate unique request IDs
- Include in logs and responses
- Trace requests across services

### 11. CORS Support
**Priority: LOW**
- Configurable CORS headers
- Allow cross-origin requests

### 12. API Documentation
**Priority: MEDIUM**
- OpenAPI/Swagger specification
- Auto-generated API docs endpoint
- Request/response examples

### 13. Database Migrations
**Priority: HIGH**
- Version-controlled migrations
- Migration status tracking
- Rollback capability

### 14. Token Refresh
**Priority: MEDIUM**
- Refresh token endpoint
- Token rotation mechanism
- Revocation list

### 15. Audit Logging
**Priority: MEDIUM**
- Track all state changes
- User/node action history
- Compliance audit trail

---

## Implementation Priority

### Phase 1 (Critical)
1. Error Handling & Logging
2. Input Validation
3. Health Check Endpoint
4. Graceful Shutdown
5. Database Migrations

### Phase 2 (Important)
6. Request ID Tracking
7. Rate Limiting
8. Pagination
9. Monitoring & Metrics

### Phase 3 (Nice to Have)
10. Caching
11. Token Refresh
12. Audit Logging
13. CORS Support
14. API Documentation
