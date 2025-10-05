#!/bin/bash

# Quick test to see what's happening with sessions

echo "Starting server with verbose output..."
./hector serve &
SERVER_PID=$!
sleep 2

echo ""
echo "Testing session history..."
echo ""

# Create a session
SESSION_ID=$(curl -s -X POST http://localhost:8080/sessions \
  -H "Content-Type: application/json" \
  -d '{"agentId":"weather_assistant"}' | jq -r '.sessionId')

echo "Created session: $SESSION_ID"

# First message
echo ""
echo "Message 1: hello, my name is john"
curl -s -X POST "http://localhost:8080/sessions/$SESSION_ID/tasks" \
  -H "Content-Type: application/json" \
  -d '{"input":{"type":"text/plain","content":"hello, my name is john"}}' | jq -r '.output.content'

sleep 2

# Second message  
echo ""
echo "Message 2: what is my name?"
curl -s -X POST "http://localhost:8080/sessions/$SESSION_ID/tasks" \
  -H "Content-Type: application/json" \
  -d '{"input":{"type":"text/plain","content":"what is my name?"}}' | jq -r '.output.content'

# Cleanup
kill $SERVER_PID 2>/dev/null

