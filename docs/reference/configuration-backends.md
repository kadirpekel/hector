# Configuration Backends - Quick Reference

## Overview

Hector supports four configuration backends for different deployment scenarios:

| Backend | Format | Watch | Use Case |
|---------|--------|-------|----------|
| **File** | YAML | ✅ | Development, single instance |
| **Consul** | JSON | ✅ | Service mesh, microservices |
| **Etcd** | JSON | ✅ | Kubernetes deployments |
| **ZooKeeper** | YAML | ✅ | Hadoop/Kafka ecosystems |

## Quick Start

### File Backend

```bash
# Create config file
cat > config.yaml <<EOF
version: "1.0"
llms:
  openai:
    type: openai
    model: gpt-4
    api_key: \${OPENAI_API_KEY}
EOF

# Run with watch
hector serve --config config.yaml --config-watch
```

### Consul Backend

```bash
# 1. Start Consul (if needed)
docker run -d -p 8500:8500 hashicorp/consul

# 2. Upload JSON config
cat > config.json <<EOF
{
  "version": "1.0",
  "llms": {
    "openai": {
      "type": "openai",
      "model": "gpt-4",
      "api_key": "\${OPENAI_API_KEY}"
    }
  }
}
EOF

curl -X PUT -d @config.json http://localhost:8500/v1/kv/hector/config

# 3. Run with watch
hector serve --config hector/config --config-type consul --config-watch
```

### Etcd Backend

```bash
# 1. Start Etcd (if needed)
docker run -d -p 2379:2379 quay.io/coreos/etcd

# 2. Upload JSON config
etcdctl put /hector/config < config.json

# 3. Run with watch
hector serve --config /hector/config --config-type etcd --config-watch
```

### ZooKeeper Backend

```bash
# 1. Start ZooKeeper (if needed)
docker run -d -p 2181:2181 zookeeper

# 2. Upload YAML config (using zkCli.sh or similar)
create /hector ""
create /hector/config "<yaml-content>"

# 3. Run with watch
hector serve --config /hector/config --config-type zookeeper --config-watch
```

## Format Conversion

Convert between YAML and JSON:

```bash
# YAML to JSON (using Python)
python3 -c "import yaml, json, sys; print(json.dumps(yaml.safe_load(sys.stdin), indent=2))" < config.yaml > config.json

# JSON to YAML (using Python)
python3 -c "import yaml, json, sys; yaml.dump(json.load(sys.stdin), sys.stdout)" < config.json > config.yaml
```

## Configuration Flags

```bash
--config PATH                    # Config path or key
--config-type TYPE               # Backend type: file|consul|etcd|zookeeper (default: file)
--config-watch                   # Enable auto-reload
--config-endpoints ENDPOINTS     # Comma-separated endpoints (e.g., consul1:8500,consul2:8500)
```

## Default Endpoints

| Backend | Default |
|---------|---------|
| Consul | `localhost:8500` |
| Etcd | `localhost:2379` |
| ZooKeeper | `localhost:2181` |

## Environment Variables

All backends support environment variable expansion:

```yaml
api_key: ${OPENAI_API_KEY}
database_url: ${DATABASE_URL:postgresql://localhost/db}  # With default value
```

## Best Practices

### Development
- Use **File** backend with `--config-watch`
- Store configs in version control
- Use environment variables for secrets

### Staging
- Use **Consul** or **Etcd** for centralized management
- Enable `--config-watch` for dynamic updates
- Separate configs per environment

### Production
- Use **Consul** or **Etcd** cluster (3+ nodes)
- Enable `--config-watch` for zero-downtime updates
- Use ACLs for access control
- Enable audit logging on backend
- Encrypt data at rest and in transit

## Troubleshooting

### Config not loading

```bash
# Test locally first
hector serve --config configs/test.yaml --debug

# Check backend connectivity
curl http://localhost:8500/v1/status/leader  # Consul
etcdctl endpoint health                       # Etcd
```

### Watch not triggering

1. Verify `--config-watch` flag is set
2. Check logs for watch errors
3. Ensure backend is accessible
4. For file backend, check file permissions

### Format errors

1. Validate YAML/JSON syntax
2. Ensure correct format for backend (YAML for file/ZK, JSON for Consul/Etcd)
3. Test configuration locally before uploading

## See Also

- [Distributed Configuration](distributed-configuration.md) - Enterprise deployment guide
- [CLI Reference](cli.md) - Complete CLI documentation  
- [Configuration Reference](configuration.md) - Config file structure

