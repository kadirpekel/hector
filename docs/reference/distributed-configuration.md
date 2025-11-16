# Distributed Configuration Management

Hector supports distributed configuration backends for enterprise deployments, enabling centralized configuration management, high availability, and dynamic updates without service restarts.

## Overview

Traditional file-based configuration works well for development and small deployments, but enterprises need:

- **Centralized Management**: Single source of truth across all instances
- **High Availability**: Configuration survives individual node failures
- **Dynamic Updates**: Change configuration without restarting services
- **Audit Trail**: Track who changed what and when
- **Access Control**: Role-based access to configuration
- **Multi-Environment**: Separate configs for dev/staging/production

Hector addresses these needs through distributed configuration backends.

## Supported Backends

| Backend | Use Case | HA | Watch | Best For |
|---------|----------|----|----|---------|
| **File** | Development, single instance | ❌ | ✅ | Local development |
| **Consul** | Service mesh environments | ✅ | ✅ | Microservices with Consul |
| **Etcd** | Kubernetes environments | ✅ | ✅ | K8s deployments |
| **ZooKeeper** | Hadoop/Kafka ecosystems | ✅ | ✅ | Big data infrastructure |

## Configuration Storage Format

### YAML: Standard Format for All Providers

**All providers (File, Consul, Etcd, ZooKeeper) support YAML as the standard format:**

```yaml
version: "1.0"
name: "Production Configuration"

llms:
  openai:
    type: openai
    model: gpt-4
    api_key: ${OPENAI_API_KEY}

agents:
  assistant:
    name: "Production Assistant"
    llm: openai
```

### JSON: Fallback Support

JSON is supported as a fallback for all providers. The system automatically detects and parses both formats:

```json
{
  "version": "1.0",
  "name": "Production Configuration",
  "llms": {
    "openai": {
      "type": "openai",
      "model": "gpt-4",
      "api_key": "${OPENAI_API_KEY}"
    }
  },
  "agents": {
    "assistant": {
      "name": "Production Assistant",
      "llm": "openai"
    }
  }
}
```

**Format Detection**: The loader tries YAML first (since YAML is a superset of JSON), then falls back to JSON if YAML parsing fails. This means:

- ✅ YAML configs work with all providers (recommended)
- ✅ JSON configs work with all providers (fallback)
- ✅ No format conversion needed between providers

**Note**: You can convert between formats if needed:
```bash
# YAML to JSON
yq eval -o=json configs/production.yaml > configs/production.json

# JSON to YAML  
yq eval -P configs/production.json > configs/production.yaml
```

## Usage Examples

### File Backend (Default)

```bash
# Standard file-based configuration
hector serve --config configs/production.yaml

# With auto-reload on file changes
hector serve --config configs/production.yaml --config-watch
```

### Consul Backend

```bash
# Upload YAML directly (recommended)
curl -X PUT --data-binary @configs/production.yaml \
  http://localhost:8500/v1/kv/hector/production

# Or upload JSON (also supported)
curl -X PUT --data-binary @configs/production.json \
  http://localhost:8500/v1/kv/hector/production

# Run with Consul backend
hector serve --config hector/production --config-type consul

# With auto-reload (reactive, no polling)
hector serve --config hector/production --config-type consul --config-watch

# Custom Consul endpoint
hector serve --config hector/production --config-type consul \
  --config-endpoints "consul.prod.example.com:8500" \
  --config-watch
```

### Etcd Backend

```bash
# Upload YAML directly (recommended)
cat configs/production.yaml | etcdctl put /hector/production

# Or upload JSON (also supported)
cat configs/production.json | etcdctl put /hector/production

# Run with Etcd backend
hector serve --config /hector/production --config-type etcd

# With Etcd cluster (high availability)
hector serve --config /hector/production --config-type etcd \
  --config-endpoints "etcd1:2379,etcd2:2379,etcd3:2379" \
  --config-watch
```

### ZooKeeper Backend

```bash
# Upload configuration to ZooKeeper (using Python kazoo)
python3 << PYTHON
from kazoo.client import KazooClient
import yaml

zk = KazooClient(hosts='127.0.0.1:2181')
zk.start()

with open('configs/production.yaml', 'r') as f:
    config_yaml = f.read()

zk.ensure_path('/hector')
if zk.exists('/hector/production'):
    zk.set('/hector/production', config_yaml.encode('utf-8'))
else:
    zk.create('/hector/production', config_yaml.encode('utf-8'), ephemeral=False)

zk.stop()
PYTHON

# Or using zkCli
docker exec -it hector-zookeeper zkCli.sh -server localhost:2181
create /hector ""
create /hector/production "$(cat configs/production.yaml)"

# Run with ZooKeeper backend
hector serve --config /hector/production --config-type zookeeper

# With ZooKeeper ensemble
hector serve --config /hector/production --config-type zookeeper \
  --config-endpoints "zk1:2181,zk2:2181,zk3:2181" \
  --config-watch
```

## CLI Reference

### Flags

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `--config` | Configuration path or key | - | `configs/app.yaml` or `hector/prod` |
| `--config-type` | Backend type | `file` | `file`, `consul`, `etcd`, `zookeeper` |
| `--config-watch` | Enable auto-reload | `false` | - |
| `--config-endpoints` | Backend endpoints (comma-separated) | Backend-specific | `consul1:8500,consul2:8500` |

### Backend Defaults

| Backend | Default Endpoint |
|---------|-----------------|
| Consul | `localhost:8500` |
| Etcd | `localhost:2379` |
| ZooKeeper | `localhost:2181` |

## Configuration Watching

When `--config-watch` is enabled, Hector automatically reloads configuration when changes are detected:

- **File**: Watches file for modifications
- **Consul**: Uses Consul's blocking queries (reactive, instant updates)
- **Etcd**: Uses Etcd's watch API (reactive, instant updates)
- **ZooKeeper**: Uses ZooKeeper's watch mechanism (reactive, instant updates)

### Reload Process

1. **Change Detection**: Backend detects configuration change reactively (no polling)
2. **Download & Parse**: New configuration is fetched and parsed
3. **Validation**: Configuration is validated (schema, required fields, etc.)
4. **If validation fails**: Current configuration remains active, error logged
5. **If successful**:

   - Graceful shutdown initiated (30s timeout for in-flight requests)
   - All components cleaned up (runtime, agents, LLMs, memory, observability)
   - Server restarts with new configuration
   - Ready to accept new requests

**Note**: This is a graceful server restart, not zero-downtime hot-reload. There will be a brief moment (1-2 seconds) where new connections are refused during the reload. For true zero-downtime updates, use a load balancer with multiple Hector instances.

## Enterprise Deployment Patterns

### Development Environment

**Pattern**: File-based with auto-reload

```bash
hector serve --config configs/dev.yaml --config-watch --debug
```

**Benefits**: Fast iteration, no infrastructure dependencies

### Staging Environment

**Pattern**: Consul with single instance

```bash
# Upload config
curl -X PUT --data-binary @configs/staging.yaml \
  http://localhost:8500/v1/kv/staging/hector

# Run
hector serve --config staging/hector --config-type consul --config-watch
```

**Benefits**: Mimics production, easy configuration management

### Production Environment

**Pattern**: Distributed backend cluster with HA

```bash
# Consul cluster (3+ nodes)
hector serve --config production/hector --config-type consul \
  --config-endpoints "consul1.prod:8500,consul2.prod:8500,consul3.prod:8500" \
  --config-watch

# Or Etcd cluster (3+ nodes)
hector serve --config /production/hector --config-type etcd \
  --config-endpoints "etcd1.prod:2379,etcd2.prod:2379,etcd3.prod:2379" \
  --config-watch

# Or ZooKeeper ensemble (3+ nodes)
hector serve --config /hector/production --config-type zookeeper \
  --config-endpoints "zk1.prod:2181,zk2.prod:2181,zk3.prod:2181" \
  --config-watch
```

**Benefits**:

- High availability (survives node failures)
- Consistent configuration across all Hector instances
- Dynamic updates without restarts
- Audit trail and access control

## Multi-Environment Strategy

### Directory Structure in Backend

```
/hector/
  /development/
    /config          # Dev environment config
  /staging/
    /config          # Staging environment config
  /production/
    /config          # Production environment config
    /config-backup   # Backup of previous config
```

### Environment-Specific Deployment

```bash
# Development
hector serve --config /hector/development/config \
  --config-type etcd --config-watch

# Staging
hector serve --config /hector/staging/config \
  --config-type etcd --config-watch

# Production
hector serve --config /hector/production/config \
  --config-type etcd \
  --config-endpoints "etcd1:2379,etcd2:2379,etcd3:2379" \
  --config-watch
```

## Configuration Versioning

### Using Consul

Consul KV automatically maintains version history:

```bash
# View configuration history
consul kv get -detailed hector/production

# Rollback to previous version (manual)
consul kv get hector/production-backup | consul kv put hector/production -
```

### Using Etcd

Etcd maintains revision history:

```bash
# Get specific revision
etcdctl get /hector/production --rev=123

# Watch history
etcdctl watch /hector/production --rev=1
```

## Security Best Practices

### Access Control

**Consul ACLs:**
```hcl
key "hector/production" {
  policy = "write"  # Only authorized services can modify
}
```

**Etcd RBAC:**
```bash
# Create role with read-only access
etcdctl role add hector-reader
etcdctl role grant-permission hector-reader read /hector/
```

### Sensitive Data

**Never store secrets directly in configuration:**

```yaml
# ❌ Bad: Secrets in config
llms:
  openai:
    api_key: "sk-actual-key-here"  # DON'T DO THIS

# ✅ Good: Environment variables
llms:
  openai:
    api_key: "${OPENAI_API_KEY}"  # References environment variable
```

**Best Practices:**
1. Store secrets in dedicated secret managers (Vault, AWS Secrets Manager)
2. Reference secrets via environment variables
3. Use ACLs to restrict configuration access
4. Enable audit logging on backend
5. Encrypt data at rest and in transit

## Integration with A2A Protocol

Distributed configuration enables **dynamic agent orchestration** across multiple Hector instances:

### Multi-Instance Deployment

```
                  ┌─────────────────┐
                  │  Consul Cluster │
                  │   (Config KV)   │
                  └────────┬────────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
        ┌─────▼─────┐┌────▼─────┐┌────▼─────┐
        │ Hector-1  ││ Hector-2 ││ Hector-3 │
        │ (A2A)     ││ (A2A)    ││ (A2A)    │
        └───────────┘└──────────┘└──────────┘
              │            │            │
              └────────────┼────────────┘
                           │
                  ┌────────▼────────┐
                  │   Load Balancer │
                  └─────────────────┘
```

**Benefits:**
- All instances share same configuration
- Update once, applies to all instances
- Agents can discover and call other agents via A2A
- Horizontal scaling without configuration drift

### Agent Discovery via A2A

```yaml
# Configuration in Consul/Etcd
agents:
  orchestrator:
    type: "native"
    # ... config
  
  external-agent:
    type: "a2a"
    url: "https://another-hector-instance:8080"
    # Discovered via A2A agent card
```

When configuration changes:

1. New agent added to configuration
2. Configuration reloaded on all instances
3. Agents immediately available for A2A calls
4. No restart, no downtime

See [A2A Protocol Reference](a2a-protocol.md) for agent-to-agent communication patterns.

## Monitoring and Observability

### Configuration Change Events

Monitor configuration reload events:

```bash
# Check logs for reload events
tail -f /var/log/hector/hector.log | grep "Configuration reloaded"

# Expected output:
# 2025/10/26 02:42:46 ✅ Configuration reloaded successfully from consul
```

### Health Checks

```bash
# Verify configuration is loaded
curl http://localhost:8080/v1/agents | jq '.agents | length'

# Should return number of configured agents
```

### Metrics

Key metrics to monitor:

- `config_reload_count` - Number of successful reloads
- `config_reload_errors` - Number of failed reloads
- `config_validation_errors` - Invalid configurations rejected
- `config_backend_errors` - Backend connection issues

## Troubleshooting

### Configuration Not Loading

**Symptoms**: "failed to load config from consul/etcd/zookeeper"

**Solutions**:

1. Verify backend is accessible:
   ```bash
   curl http://consul-host:8500/v1/status/leader  # Consul
   curl http://etcd-host:2379/version            # Etcd
   ```

2. Check if key exists:
   ```bash
   consul kv get hector/production     # Consul
   etcdctl get /hector/production      # Etcd
   ```

3. Verify YAML format is valid:
   ```bash
   # Test locally
   hector serve --config /tmp/test.yaml  # File mode first
   ```

### Watch Not Triggering

**Symptoms**: Configuration changes not detected

**Solutions**:

1. Verify `--config-watch` flag is set
2. Check network connectivity to backend
3. Review logs for watch errors:
   ```bash
   grep "watch error" /var/log/hector/hector.log
   ```

### Validation Errors

**Symptoms**: "config validation failed"

**Solutions**:

1. Check logs for specific validation error
2. Validate YAML syntax:
   ```bash
   yamllint configs/production.yaml
   ```
3. Test configuration locally before uploading:
   ```bash
   hector serve --config configs/test.yaml
   ```

## Migration from File-Based

### Step 1: Choose Backend

Select based on existing infrastructure:

- Have Consul? → Use Consul
- Have Kubernetes? → Use Etcd
- Have Hadoop/Kafka? → Use ZooKeeper
- None of above? → Start with Consul (easiest)

### Step 2: Deploy Backend (if needed)

```bash
# Consul (Docker)
docker run -d --name=consul -p 8500:8500 consul

# Etcd (Docker)
docker run -d --name=etcd -p 2379:2379 quay.io/coreos/etcd

# ZooKeeper (Docker)
docker run -d --name=zookeeper -p 2181:2181 zookeeper
```

### Step 3: Upload Configuration

```bash
# Current: File-based
hector serve --config configs/production.yaml

# Upload to backend (YAML works for all)
curl -X PUT --data-binary @configs/production.yaml \
  http://localhost:8500/v1/kv/hector/production

# Test new backend
hector serve --config hector/production --config-type consul

# Enable watch (optional)
hector serve --config hector/production --config-type consul --config-watch
```

### Step 4: Update Deployment

Update your deployment scripts/manifests to use new backend:

```yaml
# Kubernetes deployment example
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hector
spec:
  template:
    spec:
      containers:
      - name: hector
        image: hector:latest
        command:
          - /hector
          - serve
          - --config=/hector/production
          - --config-type=etcd
          - --config-endpoints=etcd-0:2379,etcd-1:2379,etcd-2:2379
          - --config-watch
```

## Summary

Distributed configuration management transforms Hector from a single-instance application into an enterprise-ready platform with:

✅ **Centralized Control**: Single source of truth  
✅ **High Availability**: Survives failures  
✅ **Dynamic Updates**: Zero-downtime changes  
✅ **Multi-Instance**: Consistent configuration across deployments  
✅ **A2A Integration**: Dynamic agent orchestration  
✅ **Enterprise Features**: ACLs, audit trails, versioning  
✅ **Format Flexibility**: YAML standard, JSON fallback for all providers

This enables large-scale AI agent deployments with operational excellence and reliability.
