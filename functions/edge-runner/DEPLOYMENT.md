# WasmEdge Edge Runner Deployment Guide

## Prerequisites

- Rust 1.70+ (for building from source)
- Docker (for containerized deployment)
- MQTT broker (for deployment notifications)
- MinIO or S3-compatible storage (for artifact storage)

## Building from Source

### 1. Clone the Repository

```bash
cd /path/to/edgebase/functions/edge-runner
```

### 2. Build the Binary

```bash
cargo build --release
```

The binary will be available at `target/release/edge-runner`.

### 3. Verify the Build

```bash
cargo test
cargo clippy
```

## Docker Deployment

### 1. Build Docker Image

```bash
docker build -t edge-runner:latest .
```

### 2. Run Container

```bash
docker run -d \
  --name edge-runner \
  -p 3000:3000 \
  -e NODE_ID=node_1 \
  -e POP_ID=tokyo \
  -e CP_URL=http://control-plane:8080 \
  -e MQTT_BROKER=mqtt://mqtt-broker:1883 \
  -e MINIO_ENDPOINT=http://minio:9000 \
  -e MINIO_ACCESS_KEY=minioadmin \
  -e MINIO_SECRET_KEY=minioadmin \
  -v /var/cache/wasm:/var/cache/wasm \
  edge-runner:latest /path/to/sample.wasm
```

## Local Development Setup

### 1. Start Dependencies

```bash
docker-compose up -d
```

This starts:
- MQTT broker (port 1883)
- MinIO (port 9000)
- Control Plane (port 8080)

### 2. Build and Run

```bash
cargo build
./target/debug/edge-runner /path/to/sample.wasm
```

### 3. Test Function Invocation

```bash
curl -X POST http://localhost:3000/api/test \
  -H "Content-Type: application/json" \
  -d '{"input": "test"}'
```

### 4. Check Metrics

```bash
curl http://localhost:3000/metrics
```

## Configuration

### Environment Variables

Create a `.env` file:

```bash
NODE_ID=node_1
POP_ID=tokyo
CP_URL=http://localhost:8080
LISTEN_ADDR=0.0.0.0
LISTEN_PORT=3000
CACHE_DIR=/var/cache/wasm
CACHE_SIZE_GB=10
MIN_HOT_INSTANCES=1
MAX_HOT_INSTANCES=10
IDLE_TIMEOUT_SECS=300
HEARTBEAT_INTERVAL_SECS=30
MINIO_ENDPOINT=http://localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MQTT_BROKER=mqtt://localhost:1883
```

Load with:

```bash
export $(cat .env | xargs)
./target/release/edge-runner /path/to/sample.wasm
```

## Monitoring

### Prometheus Metrics

Access metrics at `http://localhost:3000/metrics`

Key metrics:
- `edge_runner_total_invocations`: Total function invocations
- `edge_runner_total_errors`: Total errors
- `edge_runner_average_execution_time_ms`: Average execution time
- `edge_runner_cache_hit_rate`: Cache hit rate percentage

### Logs

Logs are printed to stdout. For production, redirect to a file:

```bash
./target/release/edge-runner /path/to/sample.wasm > edge-runner.log 2>&1 &
```

## Troubleshooting

### Issue: Connection refused to Control Plane

**Solution:** Verify Control Plane is running and accessible:

```bash
curl http://localhost:8080/health
```

### Issue: WASM module not found

**Solution:** Ensure the WASM file path is correct and readable:

```bash
ls -la /path/to/sample.wasm
```

### Issue: Cache directory permission denied

**Solution:** Ensure cache directory is writable:

```bash
mkdir -p /var/cache/wasm
chmod 755 /var/cache/wasm
```

### Issue: MQTT connection timeout

**Solution:** Verify MQTT broker is running:

```bash
telnet localhost 1883
```

### Issue: High memory usage

**Solution:** Reduce `MAX_HOT_INSTANCES` or `CACHE_SIZE_GB`:

```bash
export MAX_HOT_INSTANCES=5
export CACHE_SIZE_GB=5
```

## Performance Tuning

### For High Throughput

```bash
export MAX_HOT_INSTANCES=20
export CACHE_SIZE_GB=20
export IDLE_TIMEOUT_SECS=600
```

### For Low Latency

```bash
export MIN_HOT_INSTANCES=5
export MAX_HOT_INSTANCES=10
export IDLE_TIMEOUT_SECS=300
```

### For Resource-Constrained Environments

```bash
export MAX_HOT_INSTANCES=2
export CACHE_SIZE_GB=2
export IDLE_TIMEOUT_SECS=60
```

## Production Checklist

- [ ] Build with `--release` flag
- [ ] Set appropriate `NODE_ID` and `POP_ID`
- [ ] Configure `CP_URL` to production Control Plane
- [ ] Set up persistent cache directory
- [ ] Configure MQTT broker for notifications
- [ ] Set up MinIO/S3 for artifact storage
- [ ] Enable monitoring and logging
- [ ] Test function invocation
- [ ] Verify metrics collection
- [ ] Set up alerting for errors
- [ ] Configure resource limits
- [ ] Test failover scenarios

## Scaling

### Horizontal Scaling

Deploy multiple Edge Runner instances with different `NODE_ID` values:

```bash
# Node 1
NODE_ID=node_1 POP_ID=tokyo ./target/release/edge-runner fn.wasm

# Node 2
NODE_ID=node_2 POP_ID=tokyo ./target/release/edge-runner fn.wasm
```

### Vertical Scaling

Increase resources per node:

```bash
export MAX_HOT_INSTANCES=50
export CACHE_SIZE_GB=50
```

## Maintenance

### Clearing Cache

```bash
rm -rf /var/cache/wasm/*
```

### Updating WASM Functions

New versions are automatically deployed via MQTT notifications. No restart required.

### Monitoring Health

```bash
curl http://localhost:3000/metrics | grep edge_runner_total_errors
```

## Support

For issues or questions, refer to:
- API Documentation: `API.md`
- Architecture: `../edge-runner/ARCHITECTURE.md`
- Requirements: `../.kiro/specs/wasmEdge-edge-functions/requirements.md`
