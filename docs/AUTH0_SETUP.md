# Testing Hector Authentication with Auth0

This guide walks you through testing Hector's JWT authentication using your Auth0 account.

## Overview

Hector validates JWT tokens issued by external OAuth2/OIDC providers like Auth0. The flow:

1. Client obtains JWT token from Auth0
2. Client sends requests to Hector with `Authorization: Bearer <token>` header
3. Hector validates token using Auth0's public keys (JWKS)
4. Request is accepted/rejected based on validation

---

## Step 1: Configure Auth0

### 1.1 Create an API in Auth0

1. Go to [Auth0 Dashboard](https://manage.auth0.com/)
2. Navigate to **Applications → APIs**
3. Click **Create API**
4. Fill in:
   - **Name**: `Hector API` (or any name)
   - **Identifier**: `https://hector.yourdomain.com` (this is your audience)
   - **Signing Algorithm**: `RS256`
5. Click **Create**

### 1.2 Note Your Auth0 Settings

From your Auth0 tenant settings, collect:

- **Domain**: `your-tenant.auth0.com` (or custom domain)
- **API Identifier** (Audience): `https://hector.yourdomain.com`
- **JWKS URI**: `https://your-tenant.auth0.com/.well-known/jwks.json`

---

## Step 2: Configure Hector

### 2.1 Create Auth Configuration

Create `configs/auth-test.yaml`:

```yaml
# Global settings
global:
  a2a_server:
    enabled: true
    host: "0.0.0.0"
    port: 8090
  
  # Authentication configuration
  auth:
    enabled: true
    jwks_url: "https://YOUR-TENANT.auth0.com/.well-known/jwks.json"
    issuer: "https://YOUR-TENANT.auth0.com/"
    audience: "https://hector.yourdomain.com"
    required_scopes: []  # Optional: require specific scopes

# LLMs
llms:
  gpt:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"

# Agents
agents:
  test_agent:
    name: "Test Agent"
    description: "Agent for testing authentication"
    visibility: "public"
    llm: "gpt"
    prompt:
      system_prompt: "You are a helpful assistant."
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 5
```

**Replace:**
- `YOUR-TENANT` with your actual Auth0 domain
- `https://hector.yourdomain.com` with your API identifier

### 2.2 Update Environment Variables

Add to your `.env`:

```bash
# Existing variables...

# Auth0 settings (for reference)
AUTH0_DOMAIN=your-tenant.auth0.com
AUTH0_API_IDENTIFIER=https://hector.yourdomain.com
```

---

## Step 3: Get a Test Token from Auth0

### Method 1: Using Auth0 Test Tab (Easiest)

1. Go to **Applications → APIs → Hector API**
2. Click the **Test** tab
3. Click **Copy Token**
4. Token is copied to clipboard!

### Method 2: Using Auth0 CLI

```bash
# Install Auth0 CLI
brew tap auth0/auth0-cli && brew install auth0

# Login
auth0 login

# Get a token
auth0 test token \
  --audience https://hector.yourdomain.com \
  --scopes "read:agents write:agents"
```

### Method 3: Using cURL (Requires Client Credentials)

First, create a Machine-to-Machine application:

1. Go to **Applications → Applications → Create Application**
2. Choose **Machine to Machine Applications**
3. Authorize it for your Hector API
4. Note the **Client ID** and **Client Secret**

Then:

```bash
curl --request POST \
  --url https://YOUR-TENANT.auth0.com/oauth/token \
  --header 'content-type: application/json' \
  --data '{
    "client_id": "YOUR_CLIENT_ID",
    "client_secret": "YOUR_CLIENT_SECRET",
    "audience": "https://hector.yourdomain.com",
    "grant_type": "client_credentials"
  }'
```

Response:
```json
{
  "access_token": "eyJhbGc...",
  "token_type": "Bearer",
  "expires_in": 86400
}
```

---

## Step 4: Test Authentication

### 4.1 Start Hector with Auth Enabled

```bash
./hector serve --config configs/auth-test.yaml --debug
```

You should see:
```
🚀 A2A Server starting on 0.0.0.0:8090
🔒 Authentication: ENABLED
   Issuer: https://your-tenant.auth0.com/
   Audience: https://hector.yourdomain.com
   JWKS URL: https://your-tenant.auth0.com/.well-known/jwks.json
```

### 4.2 Test Without Token (Should Fail)

```bash
curl http://localhost:8090/agents
```

Expected response:
```json
{
  "error": "Unauthorized",
  "message": "Missing or invalid authorization header"
}
```

### 4.3 Test With Valid Token (Should Succeed)

```bash
export TOKEN="eyJhbGc..."  # Your Auth0 token

curl http://localhost:8090/agents \
  -H "Authorization: Bearer $TOKEN"
```

Expected response:
```json
{
  "agents": [
    {
      "agentId": "test_agent",
      "name": "Test Agent",
      "description": "Agent for testing authentication"
    }
  ]
}
```

### 4.4 Test Agent Call with Token

```bash
curl -X POST http://localhost:8090/agents/test_agent/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "input": {
      "type": "text/plain",
      "content": "Hello!"
    }
  }'
```

### 4.5 Test with Hector CLI

```bash
# Set token as environment variable
export HECTOR_TOKEN="eyJhbGc..."

# Modify CLI to use token (or pass via header)
./hector call test_agent "Hello!" --server http://localhost:8090
```

---

## Step 5: Automated Test Script

Save this as `test-auth0.sh`:

```bash
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
    echo "To get a token:"
    echo "1. Go to Auth0 Dashboard → APIs → Your API → Test tab"
    echo "2. Click 'Copy Token'"
    echo "3. Run: export AUTH0_TOKEN='<paste token here>'"
    echo ""
    exit 1
fi

echo "✅ Token found"
echo ""

BASE_URL="http://localhost:8090"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 1: Request without token (should fail)"
echo ""

HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" $BASE_URL/agents)

if [ "$HTTP_STATUS" = "401" ]; then
    echo "✅ PASS: Request rejected without token (HTTP 401)"
else
    echo "❌ FAIL: Expected HTTP 401, got HTTP $HTTP_STATUS"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 2: Request with invalid token (should fail)"
echo ""

HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer invalid.token.here" \
    $BASE_URL/agents)

if [ "$HTTP_STATUS" = "401" ]; then
    echo "✅ PASS: Invalid token rejected (HTTP 401)"
else
    echo "❌ FAIL: Expected HTTP 401, got HTTP $HTTP_STATUS"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 3: List agents with valid token (should succeed)"
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
echo "TEST 4: Get agent card with valid token (should succeed)"
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
    echo "$BODY" | jq '.name, .description' 2>/dev/null || echo "$BODY"
else
    echo "❌ FAIL: Expected HTTP 200, got HTTP $HTTP_STATUS"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 5: Execute task with valid token (should succeed)"
echo ""

RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -X POST \
    -H "Authorization: Bearer $AUTH0_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "input": {
        "type": "text/plain",
        "content": "Say hello"
      }
    }' \
    $BASE_URL/agents/test_agent/tasks)

HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | grep -v "HTTP_STATUS")

if [ "$HTTP_STATUS" = "200" ] || [ "$HTTP_STATUS" = "202" ]; then
    echo "✅ PASS: Task executed (HTTP $HTTP_STATUS)"
    echo ""
    echo "Response:"
    echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
else
    echo "❌ FAIL: Expected HTTP 200/202, got HTTP $HTTP_STATUS"
    echo "Response: $BODY"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "✅ Authentication test complete!"
echo ""
echo "All endpoints are now protected by Auth0 JWT validation."
```

Make it executable:
```bash
chmod +x test-auth0.sh
```

Run it:
```bash
export AUTH0_TOKEN="<your token here>"
./test-auth0.sh
```

---

## Troubleshooting

### "Invalid token: failed to verify signature"

**Cause:** JWKS URL might be incorrect or network issue

**Fix:**
1. Verify JWKS URL in browser: `https://your-tenant.auth0.com/.well-known/jwks.json`
2. Check Hector can reach Auth0:
   ```bash
   curl https://your-tenant.auth0.com/.well-known/jwks.json
   ```

### "Invalid token: audience claim does not match"

**Cause:** Token `aud` claim doesn't match Hector's configured audience

**Fix:**
1. Check token audience: `jwt decode $TOKEN | jq '.aud'`
2. Ensure it matches `audience` in Hector config
3. When requesting token, use correct `audience` parameter

### "Invalid token: issuer claim does not match"

**Cause:** Token `iss` claim doesn't match Hector's configured issuer

**Fix:**
1. Check token issuer: `jwt decode $TOKEN | jq '.iss'`
2. Ensure `issuer` in Hector config matches exactly (including trailing slash)
3. Auth0 issuer format: `https://your-tenant.auth0.com/`

### Token expired

**Cause:** JWT tokens have expiration time (usually 24 hours)

**Fix:**
Get a new token from Auth0

---

## Advanced: Scope-Based Authorization

### Configure Required Scopes

In `configs/auth-test.yaml`:

```yaml
global:
  auth:
    enabled: true
    jwks_url: "https://your-tenant.auth0.com/.well-known/jwks.json"
    issuer: "https://your-tenant.auth0.com/"
    audience: "https://hector.yourdomain.com"
    required_scopes:
      - "read:agents"    # Required to list/read agents
      - "write:agents"   # Required to execute tasks
```

### Define Scopes in Auth0

1. Go to **APIs → Your API → Permissions**
2. Add permissions:
   - `read:agents` - View agent information
   - `write:agents` - Execute agent tasks
   - `admin:agents` - Manage agents

### Request Token with Scopes

```bash
curl --request POST \
  --url https://YOUR-TENANT.auth0.com/oauth/token \
  --data '{
    "client_id": "YOUR_CLIENT_ID",
    "client_secret": "YOUR_CLIENT_SECRET",
    "audience": "https://hector.yourdomain.com",
    "grant_type": "client_credentials",
    "scope": "read:agents write:agents"
  }'
```

---

## Production Considerations

1. **Use HTTPS**: Always use HTTPS in production
2. **Validate Scopes**: Implement granular scope checks per endpoint
3. **Rate Limiting**: Add rate limiting per token/user
4. **Token Refresh**: Implement token refresh logic in clients
5. **Logging**: Log authentication attempts for security monitoring
6. **Multi-Tenancy**: Use token claims (e.g., `sub`, `org_id`) for tenant isolation

---

## Next Steps

- ✅ Test with Auth0
- ✅ Verify JWT validation works
- ⬜ Implement scope-based authorization
- ⬜ Add user context to agents (use token claims)
- ⬜ Set up production Auth0 tenant
- ⬜ Configure custom domain in Auth0
- ⬜ Add HTTPS with TLS certificates

---

## Resources

- [Auth0 Documentation](https://auth0.com/docs/)
- [JWT.io](https://jwt.io/) - Decode and inspect JWT tokens
- [Auth0 CLI](https://github.com/auth0/auth0-cli)
- [Hector Authentication Docs](AUTHENTICATION.md)
