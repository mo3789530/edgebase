# Authentication

Control Plane API uses JWT authentication for protected endpoints.

## Overview

- Auth method: JWT (`HS256`)
- Token subject: registered edge node
- Token issue timing: returned by `POST /api/v1/nodes/register`
- Header format: `Authorization: Bearer <token>`
- Refresh endpoint: `POST /api/v1/auth/refresh`

## Required Settings

Set these environment variables for the control plane:

```bash
JWT_SECRET=change-this-in-production
TOKEN_EXPIRY_HOURS=24
```

### Variables

- `JWT_SECRET`: signing key for JWT verification and issuance. Do not use the default value in production.
- `TOKEN_EXPIRY_HOURS`: token lifetime in hours. Default is `24`.

## Example Configuration

```bash
SERVER_PORT=8000
DATABASE_URL=postgresql://root@localhost:26257/defaultdb?sslmode=disable
JWT_SECRET=super-secret-signing-key
TOKEN_EXPIRY_HOURS=24
MQTT_ENABLED=false
```

## Token Flow

### 1. Register a node

`POST /api/v1/nodes/register` is public. The response includes the node record and JWT.

Example request:

```bash
curl -X POST http://localhost:8000/api/v1/nodes/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "edge-tokyo-1",
    "region": "ap-northeast-1"
  }'
```

Example response shape:

```json
{
  "node": {
    "id": "8ec5d8c7-8d55-4d9d-8924-c9bf2f1eb7be",
    "name": "edge-tokyo-1",
    "region": "ap-northeast-1"
  },
  "token": "<jwt>"
}
```

### 2. Use the token for protected APIs

```bash
curl -X POST http://localhost:8000/api/v1/nodes/8ec5d8c7-8d55-4d9d-8924-c9bf2f1eb7be/heartbeat \
  -H "Authorization: Bearer <jwt>"
```

## Refresh Token

If the current token is still valid, issue a new token with:

```bash
curl -X POST http://localhost:8000/api/v1/auth/refresh \
  -H "Authorization: Bearer <jwt>"
```

## Protected Endpoints

The following route groups require `Authorization: Bearer <token>`:

- `/api/v1/nodes/:id/heartbeat`
- `/api/v1/nodes/:id/sync`
- `/api/v1/nodes/:id/sync/ack`
- `/api/v1/nodes/:id/schema_status`
- `/api/v1/functions`
- `/api/v1/artifacts`
- `/api/v1/functions/:function_id/deploy`
- `/api/v1/routes`
- `/api/v1/schemas`
- `/api/v1/sync`
- `/api/v1/devices`

## Notes

- Tokens are signed with the configured `JWT_SECRET`. Changing this value invalidates existing tokens.
- Current implementation issues node-scoped tokens with role `node`.
- Missing, malformed, or invalid tokens return `401 Unauthorized`.
