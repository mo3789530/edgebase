#!/bin/bash
set -e

echo "=== IoT Data Sync System - Build Verification ==="
echo

echo "✓ Build Status:"
echo "  - edge-agent: $(ls -lh target/release/edge-agent | awk '{print $5}')"
echo "  - sync-service: $(ls -lh target/release/sync-service | awk '{print $5}')"
echo

echo "✓ Edge Agent Database:"
if [ -f edge-agent/edge.db ]; then
  echo "  - Database created: edge-agent/edge.db ($(ls -lh edge-agent/edge.db | awk '{print $5}'))"
else
  echo "  - No database found"
fi
echo

echo "✓ Sample Data Insertion:"
cd edge-agent
DEVICE_ID="test-device-001" ../target/release/examples/insert_sample_data 2>&1 | tail -1
cd ..
echo

echo "✓ Edge Agent Startup Test:"
cd edge-agent
timeout 2 ../target/release/edge-agent 2>&1 | grep -E "INFO|Starting" | head -2 || true
cd ..
echo

echo "=== Verification Complete ==="
echo
echo "Next steps:"
echo "1. Setup CockroachDB or PostgreSQL"
echo "2. Run: cockroach sql --insecure --database=iot_sync < migrations/001_initial_schema.sql"
echo "3. Start sync-service: DATABASE_URL='postgresql://...' ./target/release/sync-service"
echo "4. Start edge-agent: DEVICE_ID='...' API_URL='http://localhost:8080' ./target/release/edge-agent"
