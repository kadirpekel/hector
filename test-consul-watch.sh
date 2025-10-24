#!/bin/bash

# Test script for Consul configuration watching

echo "=== Testing Consul Configuration Watch ==="
echo ""

# Upload initial config (JSON format for Consul)
echo "1. Uploading initial configuration to Consul (JSON format)..."
curl -s -X PUT -d @configs/coding.json http://localhost:8500/v1/kv/hector/watch-test > /dev/null
echo "   ✅ Initial config uploaded"
echo ""

# Start hector with watch in background
echo "2. Starting Hector with config watch enabled..."
echo "   Command: hector serve --config hector/watch-test --config-type consul --config-watch"
echo ""

./hector serve --config hector/watch-test --config-type consul --config-watch --debug > /tmp/hector-consul-test.log 2>&1 &
HECTOR_PID=$!

echo "   ✅ Hector started (PID: $HECTOR_PID)"
echo "   📝 Logs: /tmp/hector-consul-test.log"
echo ""

# Wait for startup
echo "3. Waiting for Hector to initialize..."
sleep 5
echo "   ✅ Initialized"
echo ""

# Check initial logs
echo "4. Initial startup logs:"
head -20 /tmp/hector-consul-test.log | grep -E "(Configuration loaded|Config watcher|Server started)" || echo "   (checking logs...)"
echo ""

# Update config
echo "5. Updating configuration in Consul..."
cat configs/coding.json | sed 's/AI Coding Assistant/UPDATED: Modified AI Assistant (v2)/' | \
  curl -s -X PUT -d @- http://localhost:8500/v1/kv/hector/watch-test > /dev/null
echo "   ✅ Config updated in Consul"
echo ""

# Wait for reload (server graceful restart)
echo "6. Waiting for server reload (graceful shutdown + restart)..."
sleep 5
echo ""

# Check logs for reload
echo "7. Checking for reload messages:"
tail -30 /tmp/hector-consul-test.log | grep -E "(Configuration change detected|Shutting down|Configuration reload|Server started)" || \
  tail -15 /tmp/hector-consul-test.log
echo ""

# Cleanup
echo "8. Cleanup..."
kill $HECTOR_PID 2>/dev/null
wait $HECTOR_PID 2>/dev/null
echo "   ✅ Hector stopped"
echo ""

echo "=== Test Complete ==="
echo ""
echo "Full logs available at: /tmp/hector-consul-test.log"
echo ""
echo "To view full logs:"
echo "  cat /tmp/hector-consul-test.log"

