---
title: Authentication & Security
description: Secure your agents with JWT authentication and agent-level security
---

# Authentication & Security

Hector provides two layers of security: **global JWT authentication** for the A2A server and **agent-level security** for fine-grained access control.

## Security Layers

```
┌──────────────────────────────────────────┐
│         Incoming Request                 │
└─────────────────┬────────────────────────┘
                  │
                  ▼
┌──────────────────────────────────────────┐
│   Global Authentication (Optional)       │
│   - JWT validation                       │
│   - Token verification                   │
└─────────────────┬────────────────────────┘
                  │
                  ▼
┌──────────────────────────────────────────┐
│   Agent-Level Security (Optional)        │
│   - Bearer auth                          │
│   - API key auth                         │
│   - Security schemes                     │
└─────────────────┬────────────────────────┘
                  │
                  ▼
┌──────────────────────────────────────────┐
│         Agent Processes Request          │
└──────────────────────────────────────────┘
```

---

## Global JWT Authentication

Validate JWT tokens from external auth providers (Auth0, Keycloak, Okta, etc.).

### Quick Setup

```yaml
global:
  a2a_server:
    host: "0.0.0.0"
    port: 8080
  
  auth:
    jwks_url: "https://your-provider.com/.well-known/jwks.json"
    issuer: "https://your-provider.com"
    audience: "hector-api"

agents:
  secure_agent:
    llm: "gpt-4o"
    # ... agent config
```

**Start server:**

```bash
hector serve --config config.yaml

# Output:
# Authentication: ENABLED
# JWT validator initialized
```

**Make authenticated request:**

```bash
# Get token from your auth provider first
TOKEN="eyJ..."

# Call Hector with token
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/agents/secure_agent/tasks \
  -d '{"task": "Process request"}'
```

### How It Works

1. User logs in at auth provider (Auth0, Keycloak, etc.)
2. Provider issues JWT token
3. User includes token in request: `Authorization: Bearer <token>`
4. Hector validates token:
   - Fetches JWKS from provider (cached)
   - Verifies signature
   - Checks expiration
   - Validates issuer and audience
5. If valid → process request
6. If invalid → return 401 Unauthorized

### Configuration

```yaml
global:
  auth:
    jwks_url: "https://provider.com/.well-known/jwks.json"  # Required
    issuer: "https://provider.com"                         # Required
    audience: "hector-api"                                 # Required
    
    # Optional
    leeway: 60              # Clock skew tolerance (seconds)
    cache_duration: "15m"   # JWKS cache duration
```

### Supported Providers

Hector works with **any** OAuth2/OIDC provider:

#### Auth0

```yaml
auth:
  jwks_url: "https://YOUR-TENANT.auth0.com/.well-known/jwks.json"
  issuer: "https://YOUR-TENANT.auth0.com/"
  audience: "hector-api"
```

#### Keycloak

```yaml
auth:
  jwks_url: "https://keycloak.example.com/realms/YOUR_REALM/protocol/openid-connect/certs"
  issuer: "https://keycloak.example.com/realms/YOUR_REALM"
  audience: "hector-api"
```

#### Okta

```yaml
auth:
  jwks_url: "https://YOUR-DOMAIN.okta.com/oauth2/default/v1/keys"
  issuer: "https://YOUR-DOMAIN.okta.com/oauth2/default"
  audience: "api://hector"
```

#### Google

```yaml
auth:
  jwks_url: "https://www.googleapis.com/oauth2/v3/certs"
  issuer: "https://accounts.google.com"
  audience: "YOUR_CLIENT_ID.apps.googleusercontent.com"
```

### Token Claims

Hector extracts standard JWT claims:

- `sub` - Subject (user ID)
- `iss` - Issuer
- `aud` - Audience
- `exp` - Expiration
- `iat` - Issued at
- `custom_claims` - Provider-specific claims

Access claims in agent prompts (future feature):

```yaml
agents:
  personalized:
    prompt:
      system_role: |
        You are assisting user: ${jwt.sub}
        User email: ${jwt.email}
        User role: ${jwt.role}
```

---

## Agent-Level Security

Fine-grained security per agent using OpenAPI-style security schemes.

### Quick Example

```yaml
agents:
  protected_agent:
    llm: "gpt-4o"
    
    security:
      # Define security schemes
      schemes:
        bearer_auth:
          type: "http"
          scheme: "bearer"
          bearer_format: "JWT"
        
        api_key_auth:
          type: "apiKey"
          name: "X-API-Key"
          in: "header"
      
      # Require authentication (AND relationship)
      require:
        - bearer_auth
        - api_key_auth
```

**Request must include both:**

```bash
curl -H "Authorization: Bearer token" \
     -H "X-API-Key: my-api-key" \
     http://localhost:8080/agents/protected_agent/tasks \
     -d '{"task": "Secure task"}'
```

### Security Schemes

#### HTTP Bearer Authentication

```yaml
agents:
  bearer_agent:
    security:
      schemes:
        bearer_auth:
          type: "http"
          scheme: "bearer"
          bearer_format: "JWT"  # Optional
      require:
        - bearer_auth
```

**Request:**

```bash
curl -H "Authorization: Bearer your-token" \
  http://localhost:8080/agents/bearer_agent/tasks
```

#### API Key Authentication

```yaml
agents:
  api_key_agent:
    security:
      schemes:
        api_key:
          type: "apiKey"
          name: "X-API-Key"    # Header/query/cookie name
          in: "header"         # header|query|cookie
      require:
        - api_key
```

**Request:**

```bash
# Header
curl -H "X-API-Key: your-api-key" \
  http://localhost:8080/agents/api_key_agent/tasks

# Query
curl "http://localhost:8080/agents/api_key_agent/tasks?api_key=your-api-key"

# Cookie
curl -b "api_key=your-api-key" \
  http://localhost:8080/agents/api_key_agent/tasks
```

#### Multiple Schemes (AND)

```yaml
agents:
  multi_auth:
    security:
      schemes:
        bearer:
          type: "http"
          scheme: "bearer"
        api_key:
          type: "apiKey"
          name: "X-API-Key"
          in: "header"
      require:
        - bearer
        - api_key  # Both required
```

#### Alternative Schemes (OR)

```yaml
agents:
  flexible_auth:
    security:
      schemes:
        bearer:
          type: "http"
          scheme: "bearer"
        api_key:
          type: "apiKey"
          name: "X-API-Key"
          in: "header"
      require:
        # If you need OR logic, use at the application level
        - bearer  # Currently only AND supported
```

---

## Combining Both Layers

Use global JWT + agent-level security together:

```yaml
# Global JWT validation
global:
  auth:
    jwks_url: "https://auth.example.com/.well-known/jwks.json"
    issuer: "https://auth.example.com"
    audience: "hector-api"

agents:
  # Public agent (JWT only)
  public_agent:
    llm: "gpt-4o"
  
  # Protected agent (JWT + API key)
  admin_agent:
    llm: "gpt-4o"
    security:
      schemes:
        api_key:
          type: "apiKey"
          name: "X-Admin-Key"
          in: "header"
      require:
        - api_key
```

**Requests:**

```bash
# Public agent: JWT required (global auth)
curl -H "Authorization: Bearer $JWT" \
  http://localhost:8080/agents/public_agent/tasks

# Admin agent: JWT + API key required
curl -H "Authorization: Bearer $JWT" \
     -H "X-Admin-Key: admin-key" \
  http://localhost:8080/agents/admin_agent/tasks
```

---

## Authenticating to External A2A Agents

When calling external A2A agents, provide their credentials:

```yaml
agents:
  # Local coordinator
  coordinator:
    llm: "gpt-4o"
    tools: ["agent_call"]
  
  # External agent with Bearer token
  external_researcher:
    type: "a2a"
    url: "https://external-agent.example.com"
    credentials:
      type: "bearer"
      token: "${EXTERNAL_AGENT_TOKEN}"
  
  # External agent with API key
  external_analyst:
    type: "a2a"
    url: "https://analyst.example.com"
    credentials:
      type: "api_key"
      key: "${ANALYST_API_KEY}"
      header: "X-API-Key"
  
  # External agent with Basic auth
  external_writer:
    type: "a2a"
    url: "https://writer.example.com"
    credentials:
      type: "basic"
      username: "${WRITER_USER}"
      password: "${WRITER_PASS}"
```

**Environment variables:**

```bash
export EXTERNAL_AGENT_TOKEN="bearer-token-here"
export ANALYST_API_KEY="api-key-here"
export WRITER_USER="username"
export WRITER_PASS="password"
```

See [How to Integrate External Agents](../how-to/integrate-external-agents.md) for details.

---

## Security Best Practices

### 1. Use Environment Variables

Never hardcode secrets:

```yaml
# ✅ Good
global:
  auth:
    jwks_url: "${JWKS_URL}"
    issuer: "${AUTH_ISSUER}"
    audience: "${AUTH_AUDIENCE}"

agents:
  external:
    type: "a2a"
    credentials:
      token: "${EXTERNAL_TOKEN}"

# ❌ Bad
global:
  auth:
    jwks_url: "https://hardcoded-url.com/jwks.json"

agents:
  external:
    credentials:
      token: "hardcoded-token-123"
```

### 2. Use HTTPS in Production

```yaml
global:
  a2a_server:
    tls:
      
      cert_file: "/path/to/cert.pem"
      key_file: "/path/to/key.pem"
```

Or use a reverse proxy (nginx, Caddy, Traefik).

### 3. Rotate Credentials Regularly

- Rotate API keys periodically
- Use short-lived JWT tokens
- Implement token refresh flows
- Monitor for compromised credentials

### 4. Principle of Least Privilege

```yaml
agents:
  # Public access (read-only)
  public_reader:
    llm: "gpt-4o"
    # No additional security
  
  # Authenticated access
  authenticated_writer:
    llm: "gpt-4o"
    security:
      schemes:
        bearer:
          type: "http"
          scheme: "bearer"
      require:
        - bearer
  
  # Admin access (JWT + API key)
  admin:
    llm: "gpt-4o"
    security:
      schemes:
        bearer:
          type: "http"
          scheme: "bearer"
        admin_key:
          type: "apiKey"
          name: "X-Admin-Key"
          in: "header"
      require:
        - bearer
        - admin_key
```

### 5. Monitor and Log

Enable observability to track authentication attempts:

```yaml
global:
  observability:
    enabled: true
    metrics_endpoint: "localhost:4317"
```

---

## Common Scenarios

### Public API with Authentication

```yaml
global:
  auth:
    jwks_url: "${JWKS_URL}"
    issuer: "${ISSUER}"
    audience: "public-api"

agents:
  chatbot:
    llm: "gpt-4o"
    # No additional security
```

All agents require JWT token.

### Mixed Public/Private Agents

```yaml
# No global auth

agents:
  # Public agent
  public:
    llm: "gpt-4o"
  
  # Private agent
  private:
    llm: "gpt-4o"
    security:
      schemes:
        bearer:
          type: "http"
          scheme: "bearer"
      require:
        - bearer
```

Public agent accessible to all, private agent requires token.

### Multi-Tenant System

```yaml
global:
  auth:
    jwks_url: "${JWKS_URL}"
    issuer: "${ISSUER}"
    audience: "multi-tenant"

agents:
  tenant_agent:
    llm: "gpt-4o"
    security:
      schemes:
        tenant_key:
          type: "apiKey"
          name: "X-Tenant-ID"
          in: "header"
      require:
        - tenant_key
    
    prompt:
      system_role: |
        You are assisting tenant: ${tenant_id}
        Respect tenant boundaries.
```

---

## Debugging Authentication

### Enable Debug Logging

Use standard logging tools or enable observability for auth debugging:

```bash
# Run with verbose output
HECTOR_LOG_LEVEL=debug hector serve --config config.yaml
```

### Test Authentication

```bash
# Without token (should fail)
curl http://localhost:8080/agents/protected/tasks
# Expected: 401 Unauthorized

# With invalid token (should fail)
curl -H "Authorization: Bearer invalid" \
  http://localhost:8080/agents/protected/tasks
# Expected: 401 Unauthorized

# With valid token (should work)
curl -H "Authorization: Bearer $VALID_TOKEN" \
  http://localhost:8080/agents/protected/tasks
# Expected: 200 OK
```

### Check JWKS

```bash
# Verify JWKS is accessible
curl https://your-provider.com/.well-known/jwks.json

# Should return JSON with keys:
# {"keys": [{"kty": "RSA", "kid": "...", ...}]}
```

---

## Next Steps

- **[How to Deploy to Production](../how-to/deploy-production.md)** - Production security setup
- **[How to Integrate External Agents](../how-to/integrate-external-agents.md)** - External agent authentication
- **[API Reference](../reference/api.md)** - Authentication headers and responses
- **[Configuration Reference](../reference/configuration.md)** - All security options

---

## Related Topics

- **[Agent Overview](overview.md)** - Understanding agents
- **[Sessions](sessions.md)** - Session-based authentication
- **[Multi-Agent](multi-agent.md)** - Secure multi-agent systems

