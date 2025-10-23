#!/bin/bash

# Script to run Hector with a2a-inspector for interactive validation
# This allows you to use the a2a-inspector web UI to validate Hector

set -e

echo "🔍 Starting Hector A2A Inspector Integration"
echo "============================================"
echo ""

# Configuration
HECTOR_GRPC_PORT=8080
HECTOR_REST_PORT=8081
INSPECTOR_PORT=5001

# Kill any existing processes
pkill -f "./hector serve" || true
pkill -f "a2a-inspector" || true
sleep 1

# Start Hector
echo "🚀 Starting Hector server..."
./hector serve --config configs/a2a-validation.yaml --port ${HECTOR_GRPC_PORT} > validation-logs/hector-server.log 2>&1 &
HECTOR_PID=$!

# Wait for Hector to start
echo "⏳ Waiting for Hector to start..."
for i in {1..30}; do
    if curl -s "http://localhost:${HECTOR_REST_PORT}/.well-known/agent-card.json" > /dev/null 2>&1; then
        echo "✅ Hector server is running on REST port ${HECTOR_REST_PORT}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ Hector failed to start"
        kill ${HECTOR_PID} 2>/dev/null || true
        exit 1
    fi
    sleep 1
done

# Start a2a-inspector
echo ""
echo "🔍 Starting A2A Inspector..."
cd a2a-inspector-temp
source venv/bin/activate
cd backend
python app.py --port ${INSPECTOR_PORT} > ../../validation-logs/inspector.log 2>&1 &
INSPECTOR_PID=$!
cd ../..

# Wait for inspector to start
echo "⏳ Waiting for A2A Inspector to start..."
for i in {1..30}; do
    if curl -s "http://localhost:${INSPECTOR_PORT}" > /dev/null 2>&1; then
        echo "✅ A2A Inspector is running"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ A2A Inspector failed to start"
        kill ${HECTOR_PID} ${INSPECTOR_PID} 2>/dev/null || true
        exit 1
    fi
    sleep 1
done

echo ""
echo "================================================================"
echo "🎉 Both services are running!"
echo "================================================================"
echo ""
echo "📡 Hector Server:"
echo "   - REST API: http://localhost:${HECTOR_REST_PORT}"
echo "   - Agent Card: http://localhost:${HECTOR_REST_PORT}/.well-known/agent-card.json"
echo "   - Discovery: http://localhost:${HECTOR_REST_PORT}/v1/agents"
echo ""
echo "🔍 A2A Inspector:"
echo "   - Web UI: http://localhost:${INSPECTOR_PORT}"
echo ""
echo "📝 To test with the inspector:"
echo "   1. Open http://localhost:${INSPECTOR_PORT} in your browser"
echo "   2. Enter the agent URL: http://localhost:${HECTOR_REST_PORT}"
echo "   3. Or test specific agents:"
echo "      - http://localhost:${HECTOR_REST_PORT}?agent=assistant"
echo "      - http://localhost:${HECTOR_REST_PORT}?agent=code-helper"
echo "      - http://localhost:${HECTOR_REST_PORT}?agent=research-assistant"
echo ""
echo "================================================================"
echo ""
echo "Press Ctrl+C to stop both services"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "🛑 Stopping services..."
    kill ${HECTOR_PID} ${INSPECTOR_PID} 2>/dev/null || true
    echo "✅ Services stopped"
    exit 0
}

trap cleanup INT TERM

# Wait for user to stop
wait ${HECTOR_PID}

