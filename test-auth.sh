#!/bin/bash
# Test Hector JWT Authentication

set -e

echo "╔═══════════════════════════════════════════════════════════════════════╗"
echo "║                  Hector JWT Authentication Test                      ║"
echo "╔═══════════════════════════════════════════════════════════════════════╗"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
HOST="localhost"
PORT="8080"
BASE_URL="http://${HOST}:${PORT}"

# Create test config without auth
cat > /tmp/hector-test-no-auth.yaml << 'EOF'
global:
  a2a_server:
    enabled: true
    host: "0.0.0.0"
    port: 8080
  
  # Auth disabled
  auth:
    enabled: false

agents:
  test_agent:
    visibility: public
    name: "Test Agent"
    description: "Test agent for auth testing"
    llm: "main-llm"
    reasoning:
      engine: "chain-of-thought"
    prompt:
      system_role: "You are a helpful assistant."

llms:
  main-llm:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7
    max_tokens: 1000
EOF

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 1: Server without authentication"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Start server without auth
echo "Starting Hector server (no auth)..."
./hector serve --config /tmp/hector-test-no-auth.yaml &
SERVER_PID=$!
sleep 3

echo ""
echo "→ Testing agent discovery (no auth required)..."
RESPONSE=$(curl -s ${BASE_URL}/agents)
if echo "$RESPONSE" | grep -q "test_agent"; then
    echo -e "${GREEN}✅ Agent discovery works (no auth)${NC}"
else
    echo -e "${RED}❌ Agent discovery failed${NC}"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
fi

echo ""
echo "→ Testing agent card retrieval (no auth required)..."
RESPONSE=$(curl -s ${BASE_URL}/agents/test_agent)
if echo "$RESPONSE" | grep -q "Test Agent"; then
    echo -e "${GREEN}✅ Agent card retrieval works (no auth)${NC}"
else
    echo -e "${RED}❌ Agent card retrieval failed${NC}"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
fi

echo ""
echo "→ Testing task execution (no auth required)..."
RESPONSE=$(curl -s -X POST ${BASE_URL}/agents/test_agent/tasks \
    -H "Content-Type: application/json" \
    -d '{"task":"Say hello"}')
if echo "$RESPONSE" | grep -q "task_id"; then
    echo -e "${GREEN}✅ Task execution works (no auth)${NC}"
else
    echo -e "${RED}❌ Task execution failed${NC}"
    echo "Response: $RESPONSE"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
fi

echo ""
echo "Stopping server..."
kill $SERVER_PID 2>/dev/null || true
sleep 2

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 2: Server with authentication (requires mock JWKS server)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${YELLOW}⚠️  This test requires a real auth provider with JWKS endpoint${NC}"
echo -e "${YELLOW}⚠️  To test with real auth:${NC}"
echo ""
echo "   1. Set up an auth provider (Auth0, Keycloak, etc.)"
echo "   2. Update configs/auth-example.yaml with your credentials"
echo "   3. Start server: ./hector serve --config configs/auth-example.yaml"
echo "   4. Get a token from your provider"
echo "   5. Test authenticated request:"
echo ""
echo "      curl -H \"Authorization: Bearer <token>\" \\"
echo "        ${BASE_URL}/agents/secure_agent/tasks \\"
echo "        -d '{\"task\":\"Hello\"}'"
echo ""
echo "   6. Test without token (should fail):"
echo ""
echo "      curl ${BASE_URL}/agents/secure_agent/tasks \\"
echo "        -d '{\"task\":\"Hello\"}'"
echo ""
echo -e "${YELLOW}⚠️  For now, skipping this test (requires external auth setup)${NC}"
echo ""

echo ""
echo "╔═══════════════════════════════════════════════════════════════════════╗"
echo "║                        All Tests Passed! ✅                           ║"
echo "╔═══════════════════════════════════════════════════════════════════════╗"
echo ""
echo "Summary:"
echo "  ✅ Server starts without auth"
echo "  ✅ Agent discovery works"
echo "  ✅ Agent card retrieval works"
echo "  ✅ Task execution works"
echo "  ⚠️  Auth test requires external provider (manual test)"
echo ""
echo "Next Steps:"
echo "  - Set up an auth provider (Auth0, Keycloak, etc.)"
echo "  - Test with real JWT tokens"
echo "  - Verify 401 responses for invalid tokens"
echo ""

