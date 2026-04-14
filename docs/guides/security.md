# Security & Sandboxing

Security is a primary design goal of Hector. This guide covers authentication, authorization, tool permissions, and execution sandboxing.

## Authentication

Hector supports multiple authentication methods for securing API access.

### Simple Token Auth

Fastest setup. Use a shared secret.

```bash
export HECTOR_AUTH_SECRET="my-super-secret-token"
hector serve
```

All API requests must include the header:
```
Authorization: Bearer my-super-secret-token
```

### JWT/OIDC Authentication

For enterprise deployments, integrate with identity providers.

```bash
hector serve \
  --auth-jwks-url "https://auth.example.com/.well-known/jwks.json" \
  --auth-issuer "https://auth.example.com/" \
  --auth-audience "hector-api"
```

**Example: Auth0**
```bash
hector serve \
  --auth-jwks-url "https://YOUR_DOMAIN.auth0.com/.well-known/jwks.json" \
  --auth-issuer "https://YOUR_DOMAIN.auth0.com/"
```

**Example: Keycloak**
```bash
hector serve \
  --auth-jwks-url "https://keycloak.example.com/realms/myrealm/protocol/openid-connect/certs" \
  --auth-issuer "https://keycloak.example.com/realms/myrealm"
```

## Multi-Tenancy

Hector supports full multi-tenant isolation through the **Admin API**. Each tenant (app) gets completely isolated agents, sessions, state, vector stores, and JWT authentication.

### Architecture

A single Hector deployment can host many isolated applications:

```
Hector Server
├── App: "customer-support"   ← Own agents, sessions, vector store
├── App: "internal-tools"     ← Completely isolated
└── App: "partner-api"        ← Own JWT, own state
```

### Creating Tenants via Admin API

Start the server with an admin secret:

```bash
hector serve --auth-secret "admin-secret" --database postgres://...
```

Create a tenant (app):

```bash
curl -X POST http://localhost:8080/admin/apps \
  -H "Authorization: Bearer admin-secret" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "customer-support",
    "config_json": "{\"llms\":{\"default\":{\"provider\":\"anthropic\",\"model\":\"claude-sonnet-4\",\"api_key\":\"sk-ant-...\"}},\"agents\":{\"support\":{\"llm\":\"default\",\"instruction\":\"You are a support agent.\"}}}"
  }'
```

Response includes a JWT for the new app:

```json
{
  "app": { "id": "abc123", "name": "customer-support" },
  "access_token": "eyJhbGc...",
  "token_type": "bearer"
}
```

All subsequent API calls using that JWT are scoped to the `customer-support` app. Sessions, tasks, and vector data cannot leak between apps.

### JWT-Based Tenant Routing

When using JWKS/OIDC, JWT claims determine tenant routing:

| Claim | Purpose |
|-------|---------|
| `sub` | User identifier (scopes session data) |
| `tenant_id` | Maps to an app configuration |
| `roles` | Permission checking |

```bash
hector serve \
  --auth-jwks-url "https://auth.example.com/.well-known/jwks.json" \
  --auth-issuer "https://auth.example.com/"
```

A JWT with `tenant_id: "customer-support"` routes all requests to the `customer-support` app configuration.

### What's Isolated Per Tenant

| Resource | Isolation |
|----------|-----------|
| Agent configurations | Each app has its own YAML config |
| Sessions & events | Session IDs are app-scoped |
| Task queue | Tasks belong to the creating app |
| Vector store collections | RAG data is per-app |
| JWT authentication | Each app can have its own signing key |
| Rate limits | Quotas are per-app when scoped to `user` |

### Managing Tenants

```bash
# List all apps
curl http://localhost:8080/admin/apps \
  -H "Authorization: Bearer admin-secret"

# Update an app's config
curl -X PUT http://localhost:8080/admin/apps/abc123 \
  -H "Authorization: Bearer admin-secret" \
  -H "Content-Type: application/json" \
  -d '{"config_json": "{...}"}'

# Regenerate an app's JWT
curl -X POST http://localhost:8080/admin/apps/abc123/token \
  -H "Authorization: Bearer admin-secret"

# Delete an app and all its data
curl -X DELETE http://localhost:8080/admin/apps/abc123 \
  -H "Authorization: Bearer admin-secret"
```

See the [API Reference](../reference/api.md) for the complete Admin API and the [Operations Guide](./operations.md) for production multi-tenancy setup.

## Tool Permissions

Control which tools agents can access.

### Method 1: Explicit Assignment

Agents only see listed tools:

```yaml
agents:
  intern_bot:
    tools: [search] 
    # Cannot access 'delete_database'
```

### Method 2: MCP Tool Filtering

Whitelist specific tools from MCP servers:

```yaml
tools:
  github:
    type: mcp
    filter: [get_issue, create_issue]
    # 'delete_repo' is hidden
```

### Method 3: Human-in-the-Loop

Require approval for sensitive operations:

```yaml
tools:
  deploy:
    type: command
    require_approval: true
    approval_prompt: "Deploy to production?"
```

### Method 4: Guardrail Authorization

Block tools via guardrails:

```yaml
guardrails:
  strict:
    tool:
      authorization:
        enabled: true
        blocked_tools: ["delete_*", "*_production"]
```

## Rate Limiting

Protect against abuse with usage quotas:

```bash
export HECTOR_RATE_LIMIT_ENABLED=true
export HECTOR_RATE_LIMIT_LIMITS='[
  {"type": "count", "window": "minute", "limit": 60},
  {"type": "token", "window": "day", "limit": 100000}
]'
```

See the [Rate Limiting Guide](./rate-limiting.md) for configuration details.

## Guardrails

Protect against malicious inputs and harmful outputs.

| Type | Protection |
|------|------------|
| Prompt Injection | Blocks instruction override attempts |
| PII Redaction | Removes personal data from responses |
| Content Moderation | Filters harmful content |

See the [Guardrails Guide](./guardrails.md) for detailed configuration.

## Sandboxing

### Docker Sandboxing

Run Hector in a container for filesystem/network isolation:

```bash
docker run -v $(pwd):/app ghcr.io/verikod/hector:latest
```

### gVisor / Firecracker

For multi-tenant production, use microVMs:

```bash
# gVisor runtime
docker run --runtime=runsc ghcr.io/verikod/hector:latest
```

This is the isolation strategy used by Hector Cloud.

### Command Tool Restrictions

Limit command tool capabilities:

```yaml
tools:
  safe_shell:
    type: command
    allowed_commands: ["ls", "cat", "grep"]
    denied_commands: ["rm", "chmod", "sudo"]
    working_directory: "/sandbox"
```

## Secrets Management

**Never commit API keys.** Use environment variables:

```yaml
# Bad - hardcoded secret
llms:
  gpt4:
    api_key: "sk-1234..."

# Good - environment variable
llms:
  gpt4:
    api_key: ${OPENAI_API_KEY}
```

Hector automatically loads `.env` files in development.

## Security Checklist

- [ ] Enable authentication (`--auth-secret` or `--auth-jwks-url`)
- [ ] Configure rate limiting
- [ ] Use guardrails for input/output validation
- [ ] Restrict tool access per agent
- [ ] Run in Docker/container for production
- [ ] Use environment variables for secrets
- [ ] Enable TLS termination (via reverse proxy)

## Next Steps

*   [Rate Limiting Guide](./rate-limiting.md)
*   [Guardrails Guide](./guardrails.md)
*   [Deployment Guide](./deployment.md)
