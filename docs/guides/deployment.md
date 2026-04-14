# Deployment & Operations

This guide covers deploying Hector to production with proper database, queue, and observability configuration.

## Docker Deployment (Recommended)

Official images for standard and GPU workloads:

| Image | Use Case |
|-------|----------|
| `ghcr.io/verikod/hector:latest` | Standard (Alpine Linux) |
| `ghcr.io/verikod/hector:latest-gpu` | Nvidia CUDA (local LLMs) |

### Running a Container

```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/hector.yaml:/app/hector.yaml \
  -v $(pwd)/data:/data \
  -e ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY} \
  -e HECTOR_DATABASE="sqlite:///data/hector.db" \
  ghcr.io/verikod/hector:latest
```

### Production Example

```bash
docker run -d \
  -p 8080:8080 \
  -v /etc/hector/config.yaml:/app/hector.yaml:ro \
  -e HECTOR_DATABASE="postgres://user:pass@db:5432/hector" \
  -e HECTOR_AUTH_SECRET="${AUTH_SECRET}" \
  -e HECTOR_QUEUE_WORKERS=8 \
  -e HECTOR_LOG_FORMAT=json \
  ghcr.io/verikod/hector:latest
```

## Database Configuration

### SQLite (Default)

Best for single-instance deployments:

```bash
export HECTOR_DATABASE="sqlite:///data/hector.db"
```

### PostgreSQL (Production)

Required for multi-replica or high availability:

```bash
export HECTOR_DATABASE="postgres://user:pass@host:5432/hector?sslmode=require"
```

> **Note:** Migrations run automatically on startup.

## Queue Configuration

Tune the task queue for your workload:

```bash
hector serve \
  --queue-workers 8 \
  --queue-max-retries 5 \
  --queue-initial-delay 2s \
  --queue-max-delay 10m \
  --queue-stale-threshold 3m
```

| Setting | Default | Production Recommendation |
|---------|---------|---------------------------|
| `workers` | 4 | 2× CPU cores |
| `max_retries` | 3 | 5 (for transient failures) |
| `stale_threshold` | 5m | 3m (faster recovery) |

## Cloud Platforms

### Fly.io

```toml
app = "my-agent"

[env]
  PORT = "8080"
  HECTOR_LOG_FORMAT = "json"

[mounts]
  source = "hector_data"
  destination = "/data"

[[services]]
  internal_port = 8080
  protocol = "tcp"
  
  [[services.http_checks]]
    path = "/health"
    interval = 10000
    timeout = 2000
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hector
spec:
  replicas: 1  # SQLite requires single replica
  template:
    spec:
      containers:
      - name: hector
        image: ghcr.io/verikod/hector:latest
        env:
        - name: HECTOR_DATABASE
          value: "sqlite:///data/hector.db"
        - name: HECTOR_LOG_FORMAT
          value: "json"
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: data
          mountPath: /data
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: hector-data
```

### Multi-Replica with PostgreSQL

```yaml
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: hector
        env:
        - name: HECTOR_DATABASE
          valueFrom:
            secretKeyRef:
              name: hector-secrets
              key: database-url
```

## Health Checks

| Endpoint | Purpose |
|----------|---------|
| `GET /health` | Liveness + readiness |

**Response:**
```json
{
  "status": "ok",
  "version": "v1.20.0",
  "database": "connected"
}
```

## Recovery Behavior

On restart, Hector automatically:

1. **Recovers pending tasks** from the queue
2. **Resumes RAG indexing** via file checksums
3. **Restores HITL state** for approval-pending tasks

No manual intervention required if a database is configured.

## Graceful Shutdown

On SIGTERM/SIGINT:

1. Stops accepting new requests
2. Waits for in-flight requests (30s timeout)
3. Checkpoints active tasks
4. Closes database connections

## Observability Setup

Enable metrics and tracing for production:

```bash
hector serve \
  --log-format json \
  --metrics \
  --tracing-endpoint "jaeger:4317"
```

See [Observability Guide](./observability.md) for dashboard setup.

## Next Steps

*   [Observability & Monitoring](./observability.md)
*   [Security & Authentication](./security.md)
*   [Operations Guide](./operations.md)
*   [CLI Reference](../reference/cli.md)
