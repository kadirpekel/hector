---
layout: default
title: Security & Sandboxing
nav_order: 2
parent: Production & Security
description: "Safe execution and isolation"
---

# Security & Sandboxing

Secure your agent deployments with safe execution and isolation features.

## Tool Security

### Command Execution Security

```yaml
tools:
  execute_command:
    type: "command"
    enabled: true
    allowed_commands:
      - "cat"
      - "ls"
      - "grep"
      - "git"
    working_directory: "./"  # Restrict to specific directory
    max_execution_time: "30s"
    enable_sandboxing: true  # Enable process isolation
```

### File Operations Security

```yaml
tools:
  write_file:
    type: "write_file"
    enabled: true
    allowed_extensions:
      - ".go"
      - ".yaml"
      - ".md"
    forbidden_paths:
      - "/etc"
      - "/usr"
      - "/bin"
    max_file_size: 10485760  # 10MB limit
```

## Sandboxing Options

### Process Isolation

```yaml
tools:
  execute_command:
    enable_sandboxing: true
    sandbox_config:
      type: "docker"  # or "gvisor", "firecracker"
      image: "ubuntu:20.04"
      memory_limit: "512m"
      cpu_limit: "0.5"
```

### Network Restrictions

```yaml
tools:
  execute_command:
    network_policy:
      allowed_hosts:
        - "github.com"
        - "api.example.com"
      blocked_ports:
        - "22"   # SSH
        - "3389" # RDP
```

## Authentication & Authorization

### JWT Token Validation

```yaml
auth:
  enabled: true
  jwt:
    secret: "${JWT_SECRET}"
    issuer: "hector-server"
    audience: "hector-clients"
    expiration: "24h"
```

### API Key Authentication

```yaml
auth:
  enabled: true
  api_keys:
    - key: "${API_KEY_1}"
      permissions: ["read", "write"]
    - key: "${API_KEY_2}"
      permissions: ["read"]
```

## Security Best Practices

1. **Principle of Least Privilege** - Only grant necessary permissions
2. **Whitelist Commands** - Only allow explicitly approved commands
3. **Resource Limits** - Set memory, CPU, and time limits
4. **Network Isolation** - Restrict network access
5. **Audit Logging** - Log all tool executions
6. **Regular Updates** - Keep dependencies updated

## Monitoring & Logging

```yaml
logging:
  level: "info"
  format: "json"
  audit:
    enabled: true
    log_tool_calls: true
    log_auth_events: true
```

## See Also

- **[Authentication](authentication)** - JWT security configuration
- **[Built-in Tools](../tools-actions/built-in-tools)** - Tool security features
- **[Production Deployment](../development/PLUGINS)** - Deployment security
