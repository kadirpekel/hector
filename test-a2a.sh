#!/bin/bash

# Test script to verify Hector A2A implementation
# This tests the client against the server to ensure full A2A compliance

set -e

echo "🧪 Testing Hector A2A Implementation"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Configuration
CONFIG_FILE="configs/a2a-server.yaml"
SERVER_URL="http://localhost:8080"
TEST_AGENT="competitor_analyst"

# Check if hector binary exists
if [ ! -f "./hector" ]; then
    echo "❌ hector binary not found. Please build first:"
    echo "   go build -o hector cmd/hector/main.go"
    exit 1
fi

# Check if config exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ Config file not found: $CONFIG_FILE"
    exit 1
fi

echo "1️⃣  Starting Hector A2A Server..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Start server in background
./hector serve --config "$CONFIG_FILE" > /tmp/hector-server.log 2>&1 &
SERVER_PID=$!

echo "   Server PID: $SERVER_PID"
echo "   Waiting for server to start..."
sleep 3

# Check if server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ Server failed to start. Check /tmp/hector-server.log"
    cat /tmp/hector-server.log
    exit 1
fi

echo "   ✅ Server started successfully"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "🧹 Cleaning up..."
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    rm -f /tmp/hector-server.log
    echo "✅ Cleanup complete"
}

trap cleanup EXIT

echo "2️⃣  Testing: hector list"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
./hector list --server "$SERVER_URL"
echo ""

echo "3️⃣  Testing: hector info"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
./hector info "$SERVER_URL/agents/$TEST_AGENT"
echo ""

echo "4️⃣  Testing: hector call"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
./hector call "$TEST_AGENT" "What are the top 3 AI agent frameworks?" --server "$SERVER_URL"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ All tests passed!"
echo ""
echo "🎉 Hector A2A implementation is working correctly!"
echo ""
echo "Try it yourself:"
echo "  Terminal 1: ./hector serve --config $CONFIG_FILE"
echo "  Terminal 2: ./hector list"
echo "  Terminal 2: ./hector chat $TEST_AGENT"

