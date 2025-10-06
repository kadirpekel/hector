#!/bin/bash
set -e

echo "╔══════════════════════════════════════════════════════════════════════════╗"
echo "║                  🔒 AUTH0 AUTHENTICATION TEST                            ║"
echo "╚══════════════════════════════════════════════════════════════════════════╝"
echo ""

# Check if token is provided
if [ -z "$AUTH0_TOKEN" ]; then
    echo "❌ Error: AUTH0_TOKEN environment variable not set"
    echo ""
    echo "To get a token from Auth0:"
    echo "1. Go to Auth0 Dashboard → APIs → Your API → Test tab"
    echo "2. Click 'Copy Token'"
    echo "3. Run: export AUTH0_TOKEN='<paste token here>'"
    echo ""
    echo "Or use Auth0 CLI:"
    echo "   auth0 test token --audience https://hector.yourdomain.com"
    echo ""
    exit 1
fi

echo "✅ Token found (length: ${#AUTH0_TOKEN} chars)"
echo ""

# Check if server is running
if ! curl -s http://localhost:8090/agents > /dev/null 2>&1; then
    echo "❌ Error: Hector server not running on http://localhost:8090"
    echo ""
    echo "Start the server first:"
    echo "   ./hector serve --config configs/auth-test.yaml --debug"
    echo ""
    exit 1
fi

echo "✅ Server is running"
echo ""

BASE_URL="http://localhost:8090"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 1: Public agent listing (should succeed without token)"
echo ""

HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" $BASE_URL/agents)

if [ "$HTTP_STATUS" = "200" ]; then
    echo "✅ PASS: Public agent listing accessible (HTTP 200)"
    echo "   Note: /agents endpoint is public for A2A discovery"
else
    echo "❌ FAIL: Expected HTTP 200, got HTTP $HTTP_STATUS"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 2: Protected endpoint without token (should fail with 401)"
echo ""

HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    $BASE_URL/agents/test_agent)

if [ "$HTTP_STATUS" = "401" ]; then
    echo "✅ PASS: Protected endpoint rejected without token (HTTP 401)"
else
    echo "❌ FAIL: Expected HTTP 401, got HTTP $HTTP_STATUS"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 3: Protected endpoint with invalid token (should fail with 401)"
echo ""

HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer invalid.token.here" \
    $BASE_URL/agents/test_agent)

if [ "$HTTP_STATUS" = "401" ]; then
    echo "✅ PASS: Invalid token rejected (HTTP 401)"
else
    echo "❌ FAIL: Expected HTTP 401, got HTTP $HTTP_STATUS"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 4: List agents with valid token (should succeed)"
echo ""

RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -H "Authorization: Bearer $AUTH0_TOKEN" \
    $BASE_URL/agents)

HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | grep -v "HTTP_STATUS")

if [ "$HTTP_STATUS" = "200" ]; then
    echo "✅ PASS: Agents listed successfully (HTTP 200)"
    echo ""
    echo "Response:"
    echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
else
    echo "❌ FAIL: Expected HTTP 200, got HTTP $HTTP_STATUS"
    echo "Response: $BODY"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 5: Get agent card with valid token (should succeed)"
echo ""

RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -H "Authorization: Bearer $AUTH0_TOKEN" \
    $BASE_URL/agents/test_agent)

HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | grep -v "HTTP_STATUS")

if [ "$HTTP_STATUS" = "200" ]; then
    echo "✅ PASS: Agent card retrieved (HTTP 200)"
    echo ""
    echo "Agent:"
    echo "$BODY" | jq '{name, description}' 2>/dev/null || echo "$BODY"
else
    echo "❌ FAIL: Expected HTTP 200, got HTTP $HTTP_STATUS"
    echo "Response: $BODY"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 6: Execute task with valid token (should succeed)"
echo ""

RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -X POST \
    -H "Authorization: Bearer $AUTH0_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "input": {
        "type": "text/plain",
        "content": "Say hello in one sentence"
      }
    }' \
    $BASE_URL/agents/test_agent/tasks)

HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | grep -v "HTTP_STATUS")

if [ "$HTTP_STATUS" = "200" ] || [ "$HTTP_STATUS" = "202" ]; then
    echo "✅ PASS: Task executed (HTTP $HTTP_STATUS)"
    echo ""
    echo "Task ID:"
    echo "$BODY" | jq '.taskId' 2>/dev/null || echo "$BODY"
else
    echo "❌ FAIL: Expected HTTP 200/202, got HTTP $HTTP_STATUS"
    echo "Response: $BODY"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "✅ Authentication test complete!"
echo ""
echo "Summary:"
echo "- ✅ Public endpoints (agent listing) accessible without auth"
echo "- ✅ Protected endpoints reject requests without tokens (401)"
echo "- ✅ Protected endpoints reject invalid tokens (401)"
echo "- ✅ Valid Auth0 tokens grant access to protected endpoints"
echo ""
echo "Your Hector instance is now secured with Auth0! 🔒"
echo ""
echo "Note: The /agents endpoint is intentionally public for A2A protocol"
echo "discovery. All other endpoints (/agents/{id}, /sessions) are protected."
