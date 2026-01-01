# Authentication & Authorization

## Overview

This control plane uses JWT (JSON Web Tokens) for authentication. Nodes must register first to obtain a token, then use that token for subsequent API calls.

## Configuration

Set these environment variables:

```bash
JWT_SECRET=your-secret-key-change-in-production
TOKEN_EXPIRY_HOURS=24
```

## Flow

### 1. Node Registration (Public)

Register a new node to get a JWT token:

```bash
curl -X POST http://localhost:8000/api/v1/nodes/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "edge-node-1",
    "region": "us-west-2"
  }'
```

Response:
```json
{
  "node": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "edge-node-1",
    "region": "us-west-2",
    "status": "online",
    "created_at": "2025-12-31T22:53:47Z"
  },
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### 2. Authenticated Requests

Use the token in the Authorization header for all other endpoints:

```bash
curl -X POST http://localhost:8000/api/v1/nodes/550e8400-e29b-41d4-a716-446655440000/heartbeat \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## Protected Endpoints

All endpoints except `/api/v1/nodes/register` require authentication:

- ✅ Public: `POST /api/v1/nodes/register`
- 🔒 Protected: All other endpoints

## Token Claims

JWT tokens contain:
- `node_id`: UUID of the node
- `role`: Always "node"
- `exp`: Expiration time
- `iat`: Issued at time
- `sub`: Subject (node ID)

## Error Responses

Missing or invalid token:
```json
{
  "error": "missing authorization header"
}
```

Invalid token:
```json
{
  "error": "invalid token"
}
```
