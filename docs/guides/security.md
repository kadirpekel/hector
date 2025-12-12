# Security

Secure Hector deployments with authentication, authorization, and tool sandboxing.

## Authentication

### JWT Authentication

Enable JWT with JWKS:

```yaml
server:
  auth:
    enabled: true
    jwks_url: https://auth.yourdomain.com/.well-known/jwks.json
```

Hector validates JWT tokens from the `Authorization: Bearer <token>` header using the public keys from the JWKS endpoint.

### API Key Authentication

Use static API keys:

```yaml
server:
  auth:
    enabled: true
    api_keys:
      - ${API_KEY_1}
      - ${API_KEY_2}
```

Environment variables:

```bash
export API_KEY_1="key-abc123..."
export API_KEY_2="key-def456..."
```

Clients send: `Authorization: Bearer key-abc123...`

### Combined Authentication

Support both JWT and API keys:

```yaml
server:
  auth:
    enabled: true
    jwks_url: https://auth.yourdomain.com/.well-known/jwks.json
    api_keys:
      - ${SERVICE_API_KEY}
```

Hector accepts either JWT tokens or API keys.

## Agent Visibility

Control agent discovery and access:

```yaml
agents:
  # Public agent (default)
  public_assistant:
    visibility: public
    # Visible in discovery
    # Accessible via HTTP (requires auth if enabled)

  # Internal agent
  internal_analyst:
    visibility: internal
    # Only visible when authenticated
    # Requires authentication

  # Private agent
  private_helper:
    visibility: private
    # Not exposed via HTTP
    # Only accessible internally (sub-agents, agent tools)
```

### Visibility Levels

**public** (default):
- Visible in agent discovery (`/agents`)
- Accessible via HTTP
- If auth enabled, requires valid credentials

**internal**:
- Visible in discovery only when authenticated
- Requires authentication for all access
- Hidden from unauthenticated users

**private**:
- Hidden from discovery
- Not accessible via HTTP endpoints
- Only callable by other agents (sub-agents, agent tools)

### Example

```yaml
server:
  auth:
    enabled: true
    jwks_url: https://auth.company.com/.well-known/jwks.json

agents:
  # Customer-facing agent
  customer_support:
    visibility: public
    instruction: Help customers with basic questions

  # Internal admin agent
  admin_assistant:
    visibility: internal
    tools: [execute_command, write_file]
    instruction: Administrative tasks

  # Backend helper (not directly accessible)
  data_processor:
    visibility: private
    instruction: Process data internally
```

## Tool Security

### Tool Approval (HITL)

Require human approval for sensitive tools:

```yaml
tools:
  write_file:
    type: function
    handler: write_file
    require_approval: true
    approval_prompt: "Allow writing to {file}?"

  execute_command:
    type: command
    require_approval: true
    approval_prompt: "Execute: {command}?"
```

When an agent calls an approval-required tool:
1. Execution pauses
2. User receives approval request
3. User approves or denies
4. Tool executes or returns error

### Command Sandboxing

Restrict command execution:

```yaml
tools:
  execute_command:
    type: command
    working_directory: ./workspace
    max_execution_time: 30s
    allowed_commands:
      - git
      - npm
      - python
      - pytest
    denied_commands:
      - rm
      - dd
      - sudo
    deny_by_default: false
```

**Whitelist Mode** (recommended):

```yaml
tools:
  execute_command:
    type: command
    deny_by_default: true  # Deny all except allowed
    allowed_commands:
      - ls
      - cat
      - grep
```

Only whitelisted commands can execute.

### Working Directory Restriction

Limit command scope:

```yaml
tools:
  execute_command:
    type: command
    working_directory: ./safe-workspace
    # Commands execute only in this directory
```

### Execution Timeout

Prevent long-running commands:

```yaml
tools:
  execute_command:
    type: command
    max_execution_time: 30s  # Kill after 30 seconds
```

## Secret Management

### Environment Variables

Never commit secrets to configuration:

```yaml
# ✅ Good - Environment variable
llms:
  default:
    api_key: ${OPENAI_API_KEY}

# ❌ Bad - Hardcoded secret
llms:
  default:
    api_key: sk-proj-abc123...
```

### .env Files

Store secrets in `.env` (add to `.gitignore`):

```bash
# .env
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
DATABASE_PASSWORD=secret
```

Hector automatically loads `.env` files.

### Kubernetes Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: hector-secrets
  namespace: hector
type: Opaque
stringData:
  OPENAI_API_KEY: sk-...
  ANTHROPIC_API_KEY: sk-ant-...
  DATABASE_PASSWORD: secret
```

Reference in deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: hector
        envFrom:
        - secretRef:
            name: hector-secrets
```

### HashiCorp Vault

Use Vault for secret management:

```bash
# Retrieve secrets from Vault
export OPENAI_API_KEY=$(vault kv get -field=api_key secret/hector/openai)
```

Or use Vault Agent for injection.

## Network Security

### TLS/HTTPS

Terminate TLS at reverse proxy:

```nginx
# nginx.conf
server {
    listen 443 ssl http2;
    server_name agents.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/agents.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/agents.yourdomain.com/privkey.pem;

    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

### Kubernetes Network Policies

Restrict pod communication:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: hector
  namespace: hector
spec:
  podSelector:
    matchLabels:
      app: hector
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  # Allow database
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - protocol: TCP
      port: 5432
  # Allow HTTPS for LLM APIs
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443
  # Allow DNS
  - to:
    - namespaceSelector: {}
      podSelector:
        matchLabels:
          k8s-app: kube-dns
    ports:
    - protocol: UDP
      port: 53
```

### CORS Configuration

Control allowed origins:

```yaml
server:
  cors:
    allowed_origins:
      - https://app.yourdomain.com
      - https://dashboard.yourdomain.com
    allowed_methods:
      - GET
      - POST
      - OPTIONS
    allowed_headers:
      - Authorization
      - Content-Type
    max_age: 86400
```

Wildcard (development only):

```yaml
server:
  cors:
    allowed_origins:
      - "*"
```

## Rate Limiting

Prevent abuse with rate limiting:

```yaml
server:
  rate_limiting:
    enabled: true
    requests_per_minute: 60
    burst: 10
```

Per-user rate limiting (with auth):

```yaml
server:
  rate_limiting:
    enabled: true
    requests_per_minute: 100
    burst: 20
    per_user: true  # Separate limits per authenticated user
```

## Audit Logging

Enable structured logging for auditing:

```yaml
logger:
  level: info
  format: json  # Structured logs for parsing
```

Logs include:
- Authentication attempts
- Agent requests
- Tool executions
- Errors

Example log:

```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "level": "info",
  "component": "auth",
  "action": "authenticate",
  "user": "user@example.com",
  "success": true
}
```

## Security Best Practices

### Principle of Least Privilege

Grant minimal permissions:

```yaml
# ✅ Good - Minimal tools
agents:
  reader:
    tools: [read_file, grep_search]

# ❌ Bad - Excessive permissions
agents:
  reader:
    tools: [read_file, write_file, execute_command, web_request]
```

### Agent Isolation

Separate agents by trust level:

```yaml
agents:
  # Untrusted: public-facing, limited tools
  public_assistant:
    visibility: public
    tools: [search]

  # Trusted: internal, more tools
  internal_assistant:
    visibility: internal
    tools: [search, read_file, write_file]

  # Privileged: admin only, all tools
  admin_assistant:
    visibility: internal
    tools: [execute_command, write_file, web_request]
```

### Tool Whitelisting

Use explicit whitelists:

```yaml
# ✅ Good - Explicit whitelist
tools:
  execute_command:
    deny_by_default: true
    allowed_commands: [ls, cat, grep]

# ❌ Bad - Blacklist (incomplete)
tools:
  execute_command:
    deny_by_default: false
    denied_commands: [rm]  # Many dangerous commands not listed
```

### Secrets Rotation

Rotate secrets regularly:

```bash
# Generate new API key
new_key=$(generate_api_key)

# Update Kubernetes secret
kubectl create secret generic hector-secrets \
  --from-literal=API_KEY=$new_key \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart pods to pick up new secret
kubectl rollout restart deployment/hector -n hector
```

### Input Validation

Validate inputs in custom tools:

```go
func ValidatePath(path string) error {
    // Prevent path traversal
    if strings.Contains(path, "..") {
        return errors.New("path traversal not allowed")
    }
    // Restrict to workspace
    if !strings.HasPrefix(path, "/workspace/") {
        return errors.New("path must be within workspace")
    }
    return nil
}
```

## Production Security Checklist

- [ ] Enable authentication (JWT or API keys)
- [ ] Use agent visibility controls
- [ ] Require approval for destructive tools
- [ ] Enable command sandboxing with whitelists
- [ ] Store secrets in environment variables or vault
- [ ] Use TLS/HTTPS (via reverse proxy)
- [ ] Configure network policies (Kubernetes)
- [ ] Set up CORS restrictions
- [ ] Enable rate limiting
- [ ] Use structured logging for auditing
- [ ] Implement secrets rotation
- [ ] Regular security updates
- [ ] Monitor for suspicious activity

## Example: Secure Production Setup

```yaml
# config.yaml
version: "2"

llms:
  default:
    provider: openai
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}  # From Kubernetes secret

tools:
  execute_command:
    type: command
    working_directory: ./workspace
    max_execution_time: 30s
    deny_by_default: true
    allowed_commands: [git, npm, python, pytest]
    require_approval: true

  write_file:
    type: function
    handler: write_file
    require_approval: true

  read_file:
    type: function
    handler: read_file
    # No approval needed for read-only

agents:
  # Public agent: minimal tools
  public_assistant:
    visibility: public
    llm: default
    tools: [search]

  # Internal agent: more tools
  internal_assistant:
    visibility: internal
    llm: default
    tools: [search, read_file, write_file]
    document_stores: [internal_docs]

  # Admin agent: full access
  admin_assistant:
    visibility: internal
    llm: default
    tools: [execute_command, write_file, read_file]

server:
  port: 8080
  auth:
    enabled: true
    jwks_url: https://auth.company.com/.well-known/jwks.json
  cors:
    allowed_origins:
      - https://app.company.com
  rate_limiting:
    enabled: true
    requests_per_minute: 100
    per_user: true
  observability:
    metrics:
      enabled: true
    tracing:
      enabled: true
      endpoint: jaeger-collector:4317

logger:
  level: info
  format: json
```

## Next Steps

- [Deployment Guide](deployment.md) - Deploy securely
- [Observability Guide](observability.md) - Monitor security events
- [Tools Guide](tools.md) - Configure tool security
