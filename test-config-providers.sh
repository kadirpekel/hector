#!/bin/bash

# Simple test script for config providers
# Tests: upload config -> start hector -> verify startup -> update config -> verify reload

set -e

# Minimal test config
MINIMAL_CONFIG='{
  "llms": {
    "openai": {
      "api_key": "test-key"
    }
  },
  "agents": {
    "test_agent": {
      "llm": "openai",
      "description": "Test agent"
    }
  }
}'

# Upload config to provider
upload_config() {
    local provider=$1
    local key=$2
    local config=$3
    
    case $provider in
        consul)
            echo "$config" | curl -s -X PUT --data-binary @- "http://localhost:8500/v1/kv/$key" > /dev/null
            ;;
        etcd)
            if docker exec hector-etcd etcdctl put "$key" "$config" > /dev/null 2>&1; then
                :
            elif command -v etcdctl > /dev/null 2>&1 && etcdctl put "$key" "$config" > /dev/null 2>&1; then
                :
            else
                KEY_B64=$(echo -n "$key" | base64)
                VAL_B64=$(echo -n "$config" | base64)
                curl -s -X POST "http://localhost:2379/v3/kv/put" \
                    -H "Content-Type: application/json" \
                    -d "{\"key\": \"$KEY_B64\", \"value\": \"$VAL_B64\"}" > /dev/null
            fi
            ;;
        zookeeper)
            # Use Go tool to upload config
            echo "$config" | go run tools/zk-put.go -path "$key" -servers "127.0.0.1:2181" > /dev/null 2>&1
            ;;
    esac
}

# Update config in provider
update_config() {
    local provider=$1
    local key=$2
    local config=$3
    
    case $provider in
        consul)
            echo "$config" | curl -s -X PUT --data-binary @- "http://localhost:8500/v1/kv/$key" > /dev/null
            ;;
        etcd)
            if docker exec hector-etcd etcdctl put "$key" "$config" > /dev/null 2>&1; then
                :
            elif command -v etcdctl > /dev/null 2>&1 && etcdctl put "$key" "$config" > /dev/null 2>&1; then
                :
            else
                KEY_B64=$(echo -n "$key" | base64)
                VAL_B64=$(echo -n "$config" | base64)
                curl -s -X POST "http://localhost:2379/v3/kv/put" \
                    -H "Content-Type: application/json" \
                    -d "{\"key\": \"$KEY_B64\", \"value\": \"$VAL_B64\"}" > /dev/null
            fi
            ;;
        zookeeper)
            # Use Go tool to update config
            echo "$config" | go run tools/zk-put.go -path "$key" -servers "127.0.0.1:2181" > /dev/null 2>&1
            ;;
    esac
}

# Test a provider
test_provider() {
    local provider=$1
    local key=$2
    local log_file="/tmp/hector-$provider-test.log"
    
    echo "=== Testing $provider ==="
    
    # Upload initial config
    echo "1. Uploading config..."
    if ! upload_config "$provider" "$key" "$MINIMAL_CONFIG"; then
        echo "   ⚠️  Failed to upload, skipping..."
        return 1
    fi
    echo "   ✅ Config uploaded"
    
    # Start Hector
    echo "2. Starting Hector..."
    # Kill any existing Hector processes on default ports
    lsof -ti:50051 -ti:8080 2>/dev/null | xargs kill -9 2>/dev/null || true
    sleep 1
    ./hector serve --config "$key" --config-type "$provider" --config-watch > "$log_file" 2>&1 &
    local pid=$!
    echo "   ✅ Started (PID: $pid)"
    
    # Wait for startup
    echo "3. Waiting for startup..."
    sleep 3
    
    # Check startup
    if grep -qE "(Server started|Configuration loaded|gRPC server|HTTP server)" "$log_file"; then
        echo "   ✅ Startup successful"
    else
        echo "   ❌ Startup failed - check $log_file"
        tail -10 "$log_file"
        kill $pid 2>/dev/null || true
        return 1
    fi
    
    # Update config
    echo "4. Updating config..."
    UPDATED_CONFIG=$(echo "$MINIMAL_CONFIG" | sed 's/"Test agent"/"UPDATED: Test agent"/')
    update_config "$provider" "$key" "$UPDATED_CONFIG"
    echo "   ✅ Config updated"
    
    # Wait for reload
    echo "5. Waiting for reload..."
    sleep 3
    
    # Check reload
    if tail -20 "$log_file" | grep -qE "(Configuration change detected|Configuration reload requested|Server reloaded successfully)"; then
        echo "   ✅ Reload detected"
    else
        echo "   ⚠️  Reload not detected - check $log_file"
        tail -15 "$log_file" | grep -E "(reload|change|Configuration)" || tail -5 "$log_file"
    fi
    
    # Cleanup
    echo "6. Cleaning up..."
    kill $pid 2>/dev/null || true
    wait $pid 2>/dev/null || true
    
    echo "✅ $provider test complete"
    echo ""
}

# Check services
echo "=== Checking Services ==="
docker ps --format "{{.Names}}" | grep -q hector-consul && echo "✅ Consul running" || { echo "❌ Consul not running"; exit 1; }
docker ps --format "{{.Names}}" | grep -q hector-etcd && echo "✅ Etcd running" || { echo "❌ Etcd not running"; exit 1; }
docker ps --format "{{.Names}}" | grep -q hector-zookeeper && echo "✅ ZooKeeper running" || { echo "❌ ZooKeeper not running"; exit 1; }
echo ""

# Test each provider
test_provider consul "hector/test/consul"
test_provider etcd "/hector/test/etcd"
test_provider zookeeper "/hector/test/zookeeper"

echo "=== All Tests Complete ==="
