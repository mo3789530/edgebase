# Edgebase Control Plane

The Control Plane for the Edgebase platform, responsible for managing nodes, functions, deployments, and synchronization.

## Features

- Node Management & Heartbeats
- WASM Function Management & Deployment
- Schema Registry
- Telemetry & Sync
- Authentication (JWT)

## Prerequisites

- Go 1.25+
- PostgreSQL
- MinIO or AWS S3
- MQTT Broker (Optional)

## Getting Started

1. **Clone the repository**
2. **Install dependencies**
   ```bash
   go mod download
   ```
3. **Configuration**
   Copy `.env.example` to `.env` (if available) or set the environment variables.

### Storage Configuration

The Control Plane supports both MinIO (self-hosted) and AWS S3 for storing WASM artifacts.

#### Using MinIO (Default)
By default, the system uses MinIO.

```bash
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=admin
MINIO_SECRET_KEY=password
MINIO_BUCKET=wasm-functions
# S3_ENABLED=false (default)
```

#### Using AWS S3
To enable AWS S3, set `S3_ENABLED=true` and provide AWS credentials.

```bash
S3_ENABLED=true
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_BUCKET=your-bucket-name
```

### Other Configuration

```bash
SERVER_PORT=8000
DATABASE_URL=postgresql://root@localhost:26257/defaultdb?sslmode=disable
JWT_SECRET=your-secret-key
MQTT_ENABLED=false
```

## Running

```bash
go run cmd/server/main.go
```

## Build

```bash
go build -o bin/server ./cmd/server
```

## Testing

```bash
go test ./...
```
