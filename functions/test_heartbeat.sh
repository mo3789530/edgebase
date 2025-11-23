#!/bin/bash

# Start Control Plane
echo "Starting Control Plane..."
./target/release/control-plane &
CP_PID=$!
sleep 2

# Start Edge Runner
echo "Starting Edge Runner..."
./target/release/edge-runner ./target/wasm32-unknown-unknown/release/hello_world.wasm http://localhost:8080 &
ER_PID=$!
sleep 2

# Test heartbeat
echo "Testing heartbeat..."
curl -X POST http://localhost:8080/api/v1/nodes/test-node/heartbeat \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": "test-node",
    "pop_id": "tokyo-1",
    "status": "online",
    "cached_functions": []
  }' | jq .

# Test function invocation
echo "Testing function invocation..."
curl http://localhost:3000/api/test

# Cleanup
kill $CP_PID $ER_PID
