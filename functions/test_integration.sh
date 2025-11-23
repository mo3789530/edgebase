#!/bin/bash

set -e

echo "=== Building WASM ==="
cargo build --package hello-world --target wasm32-unknown-unknown --release

echo "=== Starting Control Plane ==="
./target/release/control-plane &
CP_PID=$!
sleep 2

echo "=== Starting Edge Runner ==="
./target/release/edge-runner ./target/wasm32-unknown-unknown/release/hello_world.wasm http://localhost:8080 &
ER_PID=$!
sleep 2

echo "=== Registering function ==="
FUNC_ID=$(curl -s -X POST http://localhost:8080/api/v1/functions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hello-world",
    "entrypoint": "handle",
    "runtime": "wasm",
    "memory_pages": 16,
    "max_execution_ms": 500
  }' | jq -r '.function.id')

echo "Function ID: $FUNC_ID"

echo "=== Uploading artifact ==="
curl -s -X POST http://localhost:8080/api/v1/functions/$FUNC_ID/upload \
  -F "file=@./target/wasm32-unknown-unknown/release/hello_world.wasm" | jq .

echo "=== Getting node ID from Edge Runner logs ==="
sleep 1

echo "=== Testing heartbeat ==="
curl -s -X POST http://localhost:8080/api/v1/nodes/test-node/heartbeat \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": "test-node",
    "pop_id": "tokyo-1",
    "status": "online",
    "cached_functions": []
  }' | jq .

echo "=== Testing function invocation ==="
curl -s http://localhost:3000/api/test | head -c 100
echo ""

echo "=== Cleanup ==="
kill $CP_PID $ER_PID 2>/dev/null || true
wait $CP_PID $ER_PID 2>/dev/null || true

echo "=== Test complete ==="
