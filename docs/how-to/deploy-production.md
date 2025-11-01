---
title: Deploy to Production
description: Deploy Hector agents to production with Docker, Kubernetes, or systemd
---

# How to Deploy Hector to Production

Deploy Hector agents to production environments with proper security, monitoring, and reliability.

**Time:** 30-60 minutes  
**Difficulty:** Advanced

---

## Deployment Options

| Method | Best For | Complexity | Scalability |
|--------|----------|------------|-------------|
| **Docker** | Small to medium deployments | Low | Medium |
| **Docker Compose** | Multi-service setup | Low | Medium |
| **Kubernetes** | Large-scale production | High | High |
| **systemd** | Traditional Linux servers | Medium | Low |

---

## Option 1: Docker (Recommended)

### Step 1: Create Dockerfile

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o hector ./cmd/hector

# Runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary
COPY --from=builder /app/hector .

# Copy config
COPY config.yaml .

EXPOSE 8080 8080 8080

ENTRYPOINT ["./hector"]
CMD ["serve", "--config", "config.yaml"]
```

### Step 2: Build Image

```bash
docker build -t hector:latest .
```

### Step 3: Run Container

```bash
docker run -d \
  --name hector \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/root/config.yaml \
  -e OPENAI_API_KEY="${OPENAI_API_KEY}" \
  -e ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY}" \
  --restart unless-stopped \
  hector:latest
```

### Step 4: Verify

```bash
# Check logs
docker logs hector

# Test endpoint
curl http://localhost:8080/agents

# Check health
curl http://localhost:8080/health
```

---

## Option 2: Docker Compose

For multi-service deployments with Qdrant, Ollama, etc.

### docker-compose.yml

```yaml
version: '3.8'

services:
  hector:
    build: .
    ports:
      - "8080:8080"
      - "8080:8080"
      - "8080:8080"
    volumes:
      - ./config.yaml:/root/config.yaml:ro
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - QDRANT_HOST=qdrant
      - OLLAMA_HOST=http://ollama:11434
    depends_on:
      - qdrant
      - ollama
    restart: unless-stopped
    networks:
      - hector-network

  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6334:6334"
      - "6334:6334"
    volumes:
      - qdrant_data:/qdrant/storage
    restart: unless-stopped
    networks:
      - hector-network

  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama_data:/root/.ollama
    restart: unless-stopped
    networks:
      - hector-network
    command: serve

volumes:
  qdrant_data:
  ollama_data:

networks:
  hector-network:
    driver: bridge
```

### Deploy

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f hector

# Scale Hector
docker-compose up -d --scale hector=3

# Stop all
docker-compose down
```

---

## Option 3: Kubernetes

For large-scale production deployments.

### hector-deployment.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hector
  labels:
    app: hector
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hector
  template:
    metadata:
      labels:
        app: hector
    spec:
      containers:
      - name: hector
        image: your-registry/hector:latest
        ports:
        - containerPort: 8080
          name: grpc
        - containerPort: 8080
          name: rest
        - containerPort: 8080
          name: jsonrpc
        env:
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: hector-secrets
              key: openai-api-key
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: hector-secrets
              key: anthropic-api-key
        volumeMounts:
        - name: config
          mountPath: /root/config.yaml
          subPath: config.yaml
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: hector-config
---
apiVersion: v1
kind: Service
metadata:
  name: hector
spec:
  selector:
    app: hector
  ports:
  - name: grpc
    port: 8080
    targetPort: 8080
  - name: rest
    port: 8080
    targetPort: 8080
  - name: jsonrpc
    port: 8080
    targetPort: 8080
  type: LoadBalancer
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: hector-config
data:
  config.yaml: |
    # Your Hector configuration
    agents:
      assistant:
        llm: "gpt-4o"
        # ...
---
apiVersion: v1
kind: Secret
metadata:
  name: hector-secrets
type: Opaque
stringData:
  openai-api-key: "sk-..."
  anthropic-api-key: "sk-ant-..."
```

### Deploy to Kubernetes

```bash
# Create secret
kubectl create secret generic hector-secrets \
  --from-literal=openai-api-key=$OPENAI_API_KEY \
  --from-literal=anthropic-api-key=$ANTHROPIC_API_KEY

# Deploy
kubectl apply -f hector-deployment.yaml

# Check status
kubectl get pods -l app=hector
kubectl get svc hector

# View logs
kubectl logs -f deployment/hector

# Scale
kubectl scale deployment hector --replicas=5
```

---

## Option 4: systemd (Linux)

Traditional Linux service deployment.

### hector.service

```ini
[Unit]
Description=Hector AI Agent Platform
After=network.target

[Service]
Type=simple
User=hector
WorkingDirectory=/opt/hector
ExecStart=/opt/hector/hector serve --config /opt/hector/config.yaml
Restart=always
RestartSec=10

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/hector/data

# Environment
Environment="OPENAI_API_KEY=sk-..."
Environment="ANTHROPIC_API_KEY=sk-ant-..."

[Install]
WantedBy=multi-user.target
```

### Deploy

```bash
# Create user
sudo useradd -r -s /bin/false hector

# Install binary
sudo mkdir -p /opt/hector
sudo cp hector /opt/hector/
sudo cp config.yaml /opt/hector/
sudo chown -R hector:hector /opt/hector

# Install service
sudo cp hector.service /etc/systemd/system/
sudo systemctl daemon-reload

# Start service
sudo systemctl start hector
sudo systemctl enable hector

# Check status
sudo systemctl status hector

# View logs
sudo journalctl -u hector -f
```

---

## Production Checklist

### Security

#### 1. Enable Authentication

```yaml
global:
  auth:
    jwks_url: "${JWKS_URL}"
    issuer: "${AUTH_ISSUER}"
    audience: "hector-api"
```

#### 2. Use HTTPS

**With reverse proxy (recommended):**

```nginx
# nginx.conf
server {
    listen 443 ssl http2;
    server_name hector.example.com;

    ssl_certificate /etc/ssl/certs/hector.crt;
    ssl_certificate_key /etc/ssl/private/hector.key;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

**With built-in TLS:**

```yaml
global:
  a2a_server:
    tls:
      
      cert_file: "/path/to/cert.pem"
      key_file: "/path/to/key.pem"
```

#### 3. Secure Secrets

**Use environment variables:**

```bash
# .env (never commit)
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
QDRANT_API_KEY=...
```

**Or use secret management:**

```bash
# AWS Secrets Manager
aws secretsmanager get-secret-value --secret-id hector/openai-key

# HashiCorp Vault
vault kv get secret/hector/openai-key

# Kubernetes Secrets
kubectl get secret hector-secrets -o jsonpath='{.data.openai-api-key}' | base64 -d
```

#### 4. Restrict Tool Permissions

```yaml
tools:
  execute_command:
    type: command
    
    # Production security: optionally restrict commands
    # allowed_commands: ["npm", "go", "python"]
    # denied_commands: ["rm", "dd", "sudo"]
  
  write_file:
    type: write_file
    
    # Production security: optionally restrict paths
    # allowed_paths: ["./output/"]
    # denied_paths: ["./secrets/", "./.env"]
```

### Monitoring

#### 1. Health Checks

```yaml
# Built-in health endpoint
http://localhost:8080/health
```

#### 2. Logging

Hector logs to stdout. Configure container logging to aggregate:

**Aggregate logs:**

```bash
# Docker
docker logs -f hector | jq

# Kubernetes
kubectl logs -f deployment/hector | jq

# systemd
journalctl -u hector -f | jq
```

#### 3. Metrics

**Coming soon:** Prometheus metrics endpoint.

**Workaround:** Parse JSON logs:

```bash
# Count agent calls
cat hector.log | jq 'select(.agent_call) | .agent' | sort | uniq -c

# Average response time
cat hector.log | jq -r '.response_time' | awk '{sum+=$1; n++} END {print sum/n}'
```

### Performance

#### 1. Resource Limits

**Docker:**
```bash
docker run -d \
  --memory="2g" \
  --cpus="2" \
  hector:latest
```

**Kubernetes:**
```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "2000m"
```

#### 2. Connection Pooling

```yaml
databases:
  qdrant:
    connection_pool_size: 10
    max_idle_connections: 5
```

#### 3. Caching

```yaml
document_stores:
  - name: "codebase"
    cache_embeddings: true
    cache_duration: "1h"
```

### Reliability

#### 1. Auto-Restart

**Docker:**
```bash
docker run --restart unless-stopped hector:latest
```

**Kubernetes:**
```yaml
restartPolicy: Always
```

**systemd:**
```ini
[Service]
Restart=always
RestartSec=10
```

#### 2. Health Probes (Kubernetes)

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
```

#### 3. Graceful Shutdown

Hector handles SIGTERM gracefully:

```bash
# Sends SIGTERM, waits 30s, then SIGKILL
docker stop -t 30 hector
```

---

## Scaling Strategies

### Horizontal Scaling

**Multiple Hector instances:**

```bash
# Docker Compose
docker-compose up -d --scale hector=3

# Kubernetes
kubectl scale deployment hector --replicas=5
```

**Load balancer:**

```nginx
upstream hector {
    least_conn;
    server hector-1:8080;
    server hector-2:8080;
    server hector-3:8080;
}

server {
    location / {
        proxy_pass http://hector;
    }
}
```

### Vertical Scaling

Increase resources per instance:

```yaml
resources:
  limits:
    memory: "4Gi"  # Up from 2Gi
    cpu: "4000m"   # Up from 2000m
```

---

## Backup & Disaster Recovery

### Backup Configuration

```bash
# Backup config and secrets
tar czf hector-backup-$(date +%Y%m%d).tar.gz \
  config.yaml \
  .env \
  certs/
```

### Backup Qdrant Data

```bash
# Backup vector database
docker exec qdrant tar czf /tmp/qdrant-backup.tar.gz /qdrant/storage
docker cp qdrant:/tmp/qdrant-backup.tar.gz ./backups/
```

### Restore Procedure

```bash
# Restore config
tar xzf hector-backup-20241019.tar.gz

# Restore Qdrant
docker cp ./backups/qdrant-backup.tar.gz qdrant:/tmp/
docker exec qdrant tar xzf /tmp/qdrant-backup.tar.gz -C /

# Restart services
docker-compose restart
```

---

## Troubleshooting Production Issues

### High Memory Usage

```bash
# Check memory
docker stats hector

# Reduce batch sizes
document_stores:
  - name: "codebase"
    batch_size: 50  # Lower from 100
```

### Slow Response Times

```bash
# Check LLM latency
tail -f hector.log | grep llm_latency

# Add timeout
llms:
  gpt-4o:
    timeout: 30  # Seconds
```

### Connection Errors

```bash
# Check network
docker network inspect hector-network

# Check DNS
docker exec hector ping qdrant
```

---

## Next Steps

- **[Authentication & Security](../core-concepts/security.md)** - Secure your deployment
- **[Configuration Reference](../reference/configuration.md)** - Production configs
- **[Architecture](../reference/architecture.md)** - Understanding internals
- **[CLI Reference](../reference/cli.md)** - Command-line options

---

## Related Topics

- **[Agent Overview](../core-concepts/overview.md)** - Understanding agents
- **[Build a Coding Assistant](build-coding-assistant.md)** - Complete example
- **[Set Up RAG](setup-rag.md)** - Production RAG setup

