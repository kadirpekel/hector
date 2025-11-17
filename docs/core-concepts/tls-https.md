---
title: TLS/HTTPS Configuration
description: Configure TLS certificates and HTTPS connections for enterprise deployments
---

# TLS/HTTPS Configuration

Hector supports HTTPS connections with configurable TLS certificate validation across all components. This enables secure connections to internal services, self-hosted deployments, and services with custom certificate authorities.

## Overview

Hector automatically supports HTTPS for all HTTP-based connections. By default, it uses the system's CA certificate store to validate certificates. For enterprise/internal deployments, you can configure:

- **Self-signed certificates** - Skip verification (dev/test only)
- **Custom CA certificates** - Use internal/private certificate authorities

## Components with TLS Support

| Component | TLS Config | Use Case |
|-----------|-----------|----------|
| **Vector Stores** | ✅ Full | Weaviate, Milvus, Chroma |
| **LLM Providers** | ✅ Full | OpenAI, Anthropic, Gemini, Ollama |
| **MCP Tools** | ✅ Full | External MCP servers |
| **A2A Agents** | ✅ Full | External A2A-compliant agents |
| **Web Request Tool** | ⚠️ System CA | Uses system certificate store |

## Configuration Options

### `insecure_skip_verify`

Skip TLS certificate verification. **⚠️ Use only in development/testing environments.**

```yaml
# Example: Self-signed certificate
vector_stores:
  weaviate:
    type: "weaviate"
    host: "internal.company.com"
    port: 443
    enable_tls: true
    insecure_skip_verify: true  # ⚠️ Dev/test only
```

**Security Warning:** When enabled, Hector will:
- Show a warning message at startup
- Accept any certificate (including invalid/expired ones)
- Not verify certificate hostnames
- Be vulnerable to man-in-the-middle attacks

### `ca_certificate`

Path to a custom CA certificate file (PEM format). Use this for internal/private certificate authorities.

```yaml
# Example: Custom CA certificate
vector_stores:
  weaviate:
    type: "weaviate"
    host: "internal.company.com"
    port: 443
    enable_tls: true
    ca_certificate: "/etc/ssl/certs/company-ca.pem"
```

**Best Practices:**
- Store CA certificates in a secure location (`/etc/ssl/certs/` on Linux)
- Use absolute paths
- Ensure the file is readable by the Hector process
- Keep CA certificates up to date

## Vector Stores

All vector store providers support TLS configuration:

```yaml
vector_stores:
  weaviate:
    type: "weaviate"
    host: "internal-weaviate.company.com"
    port: 443
    enable_tls: true
    insecure_skip_verify: false
    ca_certificate: "/etc/ssl/certs/company-ca.pem"

  milvus:
    type: "milvus"
    host: "internal-milvus.company.com"
    port: 443
    enable_tls: true
    ca_certificate: "/etc/ssl/certs/company-ca.pem"

  chroma:
    type: "chroma"
    host: "internal-chroma.company.com"
    port: 443
    enable_tls: true
    insecure_skip_verify: true  # Dev/test only
```

## LLM Providers

All LLM providers support TLS configuration for self-hosted deployments:

```yaml
llms:
  internal_openai:
    type: "openai"
    model: "gpt-4"
    host: "https://internal-llm.company.com"
    api_key: "${INTERNAL_API_KEY}"
    ca_certificate: "/etc/ssl/certs/company-ca.pem"

  internal_anthropic:
    type: "anthropic"
    model: "claude-sonnet-4-20250514"
    host: "https://internal-anthropic.company.com"
    api_key: "${INTERNAL_API_KEY}"
    insecure_skip_verify: true  # Dev/test only

  remote_ollama:
    type: "ollama"
    model: "qwen3"
    host: "https://ollama.company.com"
    ca_certificate: "/etc/ssl/certs/company-ca.pem"
```

## MCP Tools

MCP tools support TLS configuration for secure connections to MCP servers:

```yaml
tools:
  docling:
    type: "mcp"
    enabled: true
    server_url: "https://docling.company.com/mcp"
    description: "Docling - Document parsing"
    
    # TLS configuration
    insecure_skip_verify: false
    ca_certificate: "/etc/ssl/certs/company-ca.pem"
```

## A2A Agents

External A2A agents support TLS configuration for secure connections:

```yaml
agents:
  internal_agent:
    type: "a2a"
    url: "https://internal-agent.company.com"
    description: "Internal A2A agent"
    
    # TLS configuration
    insecure_skip_verify: false
    ca_certificate: "/etc/ssl/certs/company-ca.pem"
    
    credentials:
      type: "bearer"
      token: "${INTERNAL_TOKEN}"
```

**Note:** TLS configuration applies to both HTTP/REST and gRPC transports used by A2A agents.

## Common Scenarios

### Scenario 1: Internal Services with Custom CA

**Problem:** Your company uses an internal CA for all services.

**Solution:** Use `ca_certificate` to specify your company's CA:

```yaml
vector_stores:
  weaviate:
    type: "weaviate"
    host: "weaviate.internal.company.com"
    port: 443
    enable_tls: true
    ca_certificate: "/etc/ssl/certs/company-ca.pem"

llms:
  internal_llm:
    type: "openai"
    host: "https://llm.internal.company.com"
    ca_certificate: "/etc/ssl/certs/company-ca.pem"
```

### Scenario 2: Development with Self-Signed Certificates

**Problem:** Local development uses self-signed certificates.

**Solution:** Use `insecure_skip_verify: true` (dev/test only):

```yaml
vector_stores:
  weaviate:
    type: "weaviate"
    host: "localhost"
    port: 8443
    enable_tls: true
    insecure_skip_verify: true  # ⚠️ Dev/test only
```

### Scenario 3: Mixed Public and Internal Services

**Problem:** Some services use public certificates, others use internal CAs.

**Solution:** Configure TLS only for internal services:

```yaml
# Public service - no TLS config needed
llms:
  openai:
    type: "openai"
    host: "https://api.openai.com/v1"  # Public CA, works automatically

# Internal service - custom CA
llms:
  internal_llm:
    type: "openai"
    host: "https://llm.internal.company.com"
    ca_certificate: "/etc/ssl/certs/company-ca.pem"
```

## Security Best Practices

1. **Production:** Always use valid certificates or custom CA certificates
2. **Development:** `insecure_skip_verify: true` is acceptable for local testing
3. **Custom CAs:** Store CA certificates securely and keep them updated
4. **Monitoring:** Watch for TLS-related warnings in logs
5. **Rotation:** Rotate certificates and update CA bundles regularly

## Troubleshooting

### Certificate Verification Failed

**Error:** `x509: certificate signed by unknown authority`

**Solution:** Add the CA certificate:
```yaml
ca_certificate: "/path/to/ca-cert.pem"
```

### Self-Signed Certificate

**Error:** `x509: certificate is not trusted`

**Solution (dev/test only):**
```yaml
insecure_skip_verify: true
```

### Certificate File Not Found

**Error:** `failed to read CA certificate from /path/to/ca.pem`

**Solution:**
- Verify the file path is correct
- Ensure the file is readable
- Use absolute paths
- Check file permissions

## Related Documentation

- [Configuration Reference](../reference/configuration.md) - Complete configuration options
- [Security Guide](security.md) - Authentication and authorization
- [Vector Stores](../reference/configuration.md#vector-stores) - Database configuration
- [LLM Providers](../reference/configuration.md#llm-providers) - LLM configuration

