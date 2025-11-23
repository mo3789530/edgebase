#!/bin/bash

set -e

RUNNER_PID=""
CONTROL_PLANE_PID=""

cleanup() {
    if [ -n "$RUNNER_PID" ]; then
        kill $RUNNER_PID 2>/dev/null || true
    fi
    if [ -n "$CONTROL_PLANE_PID" ]; then
        kill $CONTROL_PLANE_PID 2>/dev/null || true
    fi
}

trap cleanup EXIT

echo "=== Building WASM module ==="
cargo build --package hello-world --target wasm32-unknown-unknown --release

echo "=== Starting Control Plane ==="
./target/release/control-plane &
CONTROL_PLANE_PID=$!
sleep 2

echo "=== Starting Edge Runner ==="
./target/release/edge-runner ./target/wasm32-unknown-unknown/release/hello_world.wasm &
RUNNER_PID=$!
sleep 2

echo "=== Testing HTTP Routing ==="

# Test 1: Basic route
echo "Test 1: Basic GET request"
curl -s http://localhost:3000/api/test -H "Host: localhost" | head -c 100
echo ""

# Test 2: Different path
echo "Test 2: Different path"
curl -s http://localhost:3000/api/users -H "Host: localhost" | head -c 100
echo ""

# Test 3: 404 - Route not found
echo "Test 3: 404 - Route not found"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/nonexistent)
if [ "$HTTP_CODE" = "404" ]; then
    echo "✓ Correctly returned 404"
else
    echo "✗ Expected 404, got $HTTP_CODE"
fi

# Test 4: Metrics endpoint
echo "Test 4: Metrics endpoint"
curl -s http://localhost:3000/metrics | grep -q "wasm_invoke_count_total" && echo "✓ Metrics available" || echo "✗ Metrics not found"

# Test 5: Multiple requests
echo "Test 5: Multiple requests"
for i in {1..3}; do
    curl -s http://localhost:3000/api/test -H "Host: localhost" > /dev/null
done
echo "✓ Completed 3 requests"

echo ""
echo "=== Routing Tests Complete ==="
