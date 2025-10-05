# Authentication in Hector

**JWT Token Validation for Enterprise Security**

## Overview

Hector implements JWT-based authentication as a **consumer** of external authentication providers. This means:

✅ **Hector validates tokens** issued by your auth provider  
✅ **Works with ANY OAuth2/OIDC provider** (Auth0, Keycloak, Okta, Google, etc.)  
✅ **Zero custom integration** - just configure JWKS URL  
✅ **Provider-agnostic** - switch providers with zero code changes  
✅ **Optional** - Auth disabled by default, enable when needed  

❌ **Hector does NOT handle** login/logout, user management, or token issuance  
❌ **Hector does NOT store** passwords or user credentials  
❌ **Hector does NOT implement** OAuth2 flows  

---

## Quick Start

### 1. Configure Authentication

```yaml
# hector-config.yaml
global:
  a2a_server:
    enabled: true
    host: "0.0.0.0"
    port: 8080
  
  # Enable JWT authentication
  auth:
    enabled: true
    jwks_url: "https://your-auth-provider.com/.well-known/jwks.json"
    issuer: "https://your-auth-provider.com"
    audience: "hector-api"

agents:
  secure_agent:
    visibility: public
    name: "Secure Agent"
    llm: "gpt-4o"
    # ... rest of agent config
```

### 2. Start Server with Auth

```bash
./hector serve --config hector-config.yaml

# Output:
# 🔒 Authentication: ENABLED
# ✅ JWT validator initialized
#    Provider: https://your-auth-provider.com
```

### 3. Get Token from Provider

```bash
# Example with Auth0
curl -X POST https://YOUR-TENANT.auth0.com/oauth/token \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "YOUR_CLIENT_ID",
    "client_secret": "YOUR_CLIENT_SECRET",
    "audience": "hector-api",
    "grant_type": "client_credentials"
  }'

# Response: {"access_token": "eyJ..."}
```

### 4. Call Hector with Token

```bash
# Authenticated request
curl -H "Authorization: Bearer eyJ..." \
  http://localhost:8080/agents/secure_agent/tasks \
  -d '{"task":"Process secure data"}'

# Without token (fails)
curl http://localhost:8080/agents/secure_agent/tasks \
  -d '{"task":"Process secure data"}'
# Response: 401 Unauthorized
```

---

## How It Works

### The Flow

```
1. User logs in at Auth Provider (Auth0/Keycloak/etc.)
   ↓
2. Auth Provider issues JWT token
   ↓
3. User sends request to Hector:
   Authorization: Bearer <token>
   ↓
4. Hector validates token:
   • Fetches JWKS from provider (cached)
   • Verifies JWT signature
   • Checks expiration
   • Extracts claims (user, role, tenant)
   ↓
5. If valid: Process request
   If invalid: Return 401 Unauthorized
```

### What Hector Does

✅ **Fetches JWKS** from provider (once, then cached)  
✅ **Auto-refreshes JWKS** every 15 minutes (handles key rotation)  
✅ **Validates signature** using provider's public keys  
✅ **Checks expiration** to prevent replay attacks  
✅ **Extracts claims** (user ID, email, role, tenant)  
✅ **Enforces permissions** based on claims  

### What Hector Does NOT Do

❌ Handle user login/logout UI  
❌ Manage users or passwords  
❌ Issue or refresh tokens  
❌ Implement OAuth2 flows  
❌ Terminate SSL/TLS  

**All of this is handled by your auth provider!**

---

## Supported Providers

Hector works with **ANY** provider that exposes a JWKS endpoint and issues standard JWTs.

### Auth0

```yaml
auth:
  enabled: true
  jwks_url: "https://YOUR-TENANT.auth0.com/.well-known/jwks.json"
  issuer: "https://YOUR-TENANT.auth0.com/"
  audience: "hector-api"
```

**Setup:**
1. Create API named "hector-api" in Auth0
2. Add custom claim "role" in Auth0 Action/Rule (optional)
3. Done!

### Keycloak

```yaml
auth:
  enabled: true
  jwks_url: "https://keycloak.example.com/realms/hector/protocol/openid-connect/certs"
  issuer: "https://keycloak.example.com/realms/hector"
  audience: "hector-api"
```

**Setup:**
1. Create realm "hector"
2. Create client "hector-api"
3. Add role mapper (optional)
4. Done!

### Google

```yaml
auth:
  enabled: true
  jwks_url: "https://www.googleapis.com/oauth2/v3/certs"
  issuer: "https://accounts.google.com"
  audience: "YOUR-CLIENT-ID.apps.googleusercontent.com"
```

### Okta

```yaml
auth:
  enabled: true
  jwks_url: "https://YOUR-DOMAIN.okta.com/oauth2/default/v1/keys"
  issuer: "https://YOUR-DOMAIN.okta.com/oauth2/default"
  audience: "hector-api"
```

### Custom Provider

Any provider that exposes JWKS and issues standard JWTs:

```yaml
auth:
  enabled: true
  jwks_url: "https://your-auth.com/jwks.json"
  issuer: "https://your-auth.com"
  audience: "hector-api"
```

**Provider Requirements:**
1. Expose JWKS endpoint (public keys for JWT verification)
2. Issue JWTs with standard claims (`iss`, `aud`, `exp`, `sub`)
3. (Optional) Include custom claims (`email`, `role`, `tenant_id`)

---

## Protected vs Public Endpoints

### With Authentication Enabled

| Endpoint | Protected | Notes |
|----------|-----------|-------|
| `GET /agents` | ❌ No | Agent discovery is always public |
| `GET /agents/{id}` | ✅ Yes | Requires valid token |
| `POST /agents/{id}/tasks` | ✅ Yes | Requires valid token |
| `POST /sessions` | ✅ Yes | Requires valid token |
| `GET /sessions/{id}` | ✅ Yes | Requires valid token |

### Without Authentication (Default)

All endpoints are public.

### Why is `GET /agents` Always Public?

Agent discovery (`GET /agents`) is always public to enable:
- Agent marketplace browsing
- Capability discovery
- Integration planning

If you want to hide specific agents, use the `visibility` field:

```yaml
agents:
  public_agent:
    visibility: public  # Listed in GET /agents

  internal_agent:
    visibility: internal  # Not listed, but accessible if you know the ID

  private_agent:
    visibility: private  # Not accessible via API at all
```

---

## Claims Extraction

Hector extracts standard and custom claims from validated tokens:

### Standard Claims

- `sub` (subject) - User ID
- `iss` (issuer) - Token issuer
- `aud` (audience) - Token audience
- `exp` (expiration) - Token expiration
- `iat` (issued at) - Token issue time

### Custom Claims (Optional)

- `email` - User email
- `role` - User role (for RBAC)
- `tenant_id` - Tenant ID (for multi-tenancy)

**Note:** Custom claims are provider-specific. Configure them in your auth provider.

---

## Testing Authentication

### Test Without Auth (Default)

```bash
# Start server
./hector serve --config configs/a2a-server.yaml

# No authentication required
curl http://localhost:8080/agents/test_agent/tasks \
  -d '{"task":"Hello"}'
```

### Test With Auth

```bash
# 1. Start server with auth
./hector serve --config configs/auth-example.yaml

# 2. Try without token (should fail)
curl http://localhost:8080/agents/secure_agent/tasks \
  -d '{"task":"Hello"}'
# Response: 401 Unauthorized

# 3. Get token from provider
TOKEN=$(curl -s -X POST https://provider.com/oauth/token \
  -d '{"client_id":"...","audience":"hector-api"}' \
  | jq -r '.access_token')

# 4. Try with token (should succeed)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/agents/secure_agent/tasks \
  -d '{"task":"Hello"}'
# Response: 200 OK
```

### Automated Test Script

```bash
./test-auth.sh
```

---

## Security Best Practices

### 1. Use HTTPS in Production

```yaml
# Use reverse proxy (nginx, traefik) for TLS termination
# Never expose Hector directly to the internet without HTTPS
```

### 2. Validate Issuer and Audience

```yaml
auth:
  issuer: "https://your-provider.com"  # Must match token's 'iss' claim
  audience: "hector-api"               # Must match token's 'aud' claim
```

### 3. Use Short-Lived Tokens

Configure your auth provider to issue short-lived tokens (e.g., 15 minutes).

### 4. Monitor Token Usage

Log authentication events for audit trails:

```go
// Hector automatically logs authentication attempts
// 2024/10/05 20:31:00 [INFO] Auth: Valid token for user user@example.com
// 2024/10/05 20:32:00 [WARN] Auth: Invalid token from IP 192.168.1.100
```

### 5. Use Agent Visibility

Combine auth with visibility controls:

```yaml
agents:
  public_api:
    visibility: public  # Anyone can discover and use (if authenticated)

  internal_tool:
    visibility: internal  # Hidden from discovery, only accessible if you know the ID

  orchestrator_helper:
    visibility: private  # Not accessible via API, only for local orchestrators
```

---

## Troubleshooting

### Error: "Failed to fetch JWKS"

**Cause:** Hector can't reach your auth provider's JWKS endpoint.

**Solution:**
1. Verify JWKS URL is correct: `curl https://provider.com/.well-known/jwks.json`
2. Check network connectivity
3. Ensure firewall allows outbound HTTPS

### Error: "Invalid token: signature verification failed"

**Cause:** Token was not issued by the configured provider, or keys have rotated.

**Solution:**
1. Verify token issuer matches config
2. Get a fresh token from your provider
3. Wait for JWKS auto-refresh (15 minutes)

### Error: "Invalid token: token is expired"

**Cause:** Token has expired.

**Solution:**
1. Get a new token from your provider
2. Implement token refresh in your client

### Error: "Missing Authorization header"

**Cause:** Request didn't include `Authorization: Bearer <token>`.

**Solution:**
```bash
# Correct format
curl -H "Authorization: Bearer eyJ..." http://localhost:8080/...

# Not: curl -H "Token: eyJ..."
# Not: curl -H "Bearer eyJ..."
```

---

## Advanced: Per-Agent Authorization (Future)

Currently, all authenticated users can access all agents. Future versions may support:

```yaml
agents:
  admin_agent:
    auth:
      required: true
      allowed_roles: ["admin", "superuser"]
      allowed_tenants: ["tenant-1", "tenant-2"]
  
  user_agent:
    auth:
      required: true
      allowed_roles: ["user", "admin"]
```

---

## Benefits of This Approach

✅ **Provider-Agnostic** - Works with ANY OAuth2/OIDC provider  
✅ **Zero Custom Integration** - Just JWKS URL + validation  
✅ **Auto Key Rotation** - JWKS auto-refreshes every 15 minutes  
✅ **Stateless** - No database needed for auth  
✅ **Scalable** - JWT validation is fast (in-memory)  
✅ **Secure** - Industry-standard cryptography  
✅ **Simple** - ~300 lines of code total  
✅ **Future-Proof** - New providers work without code changes  

---

## Example Configurations

### Development (No Auth)

```yaml
global:
  a2a_server:
    enabled: true
    host: "0.0.0.0"
    port: 8080
  
  auth:
    enabled: false  # Default
```

### Production (With Auth0)

```yaml
global:
  a2a_server:
    enabled: true
    host: "0.0.0.0"
    port: 8080
  
  auth:
    enabled: true
    jwks_url: "https://your-tenant.auth0.com/.well-known/jwks.json"
    issuer: "https://your-tenant.auth0.com/"
    audience: "hector-api"
```

### Production (With Keycloak)

```yaml
global:
  a2a_server:
    enabled: true
    host: "0.0.0.0"
    port: 8080
  
  auth:
    enabled: true
    jwks_url: "https://keycloak.company.com/realms/hector/protocol/openid-connect/certs"
    issuer: "https://keycloak.company.com/realms/hector"
    audience: "hector-api"
```

---

## Summary

Hector's authentication is:
- **Simple** - Just 3 config values (JWKS URL, issuer, audience)
- **Secure** - Industry-standard JWT validation
- **Flexible** - Works with any OAuth2/OIDC provider
- **Optional** - Enable only when needed

**What Hector does:** Validate tokens  
**What you do:** Choose your auth provider  
**What your provider does:** Everything else (login, users, tokens)

This clean separation of concerns makes Hector enterprise-ready without reinventing authentication.

---

## See Also

- [Configuration Guide](CONFIGURATION.md) - Full config reference
- [Quick Start](QUICK_START.md) - Get started quickly
- [Architecture](ARCHITECTURE.md) - System design
- Example: `configs/auth-example.yaml`

