# Docker Compose Setup - Complete RAG Environment

This document describes the consolidated Docker Compose setup that provides a **complete, ready-to-use enterprise RAG system** with a single command.

## Overview

The `docker-compose.yaml` file includes **all services** needed for a production-ready RAG system:

1. **Qdrant** - Vector database for embeddings
2. **Ollama** - Embeddings and LLM (100% local)
3. **PostgreSQL** - Knowledge base database
4. **Wiki API** - Mock internal wiki (sample data)
5. **Hector** - RAG orchestration server (official Docker image)
6. **Prometheus** - Metrics and observability

## Key Features

### ✅ Zero Configuration
- All services pre-configured
- Automatic service discovery via Docker networking
- Health checks ensure proper startup order
- Models automatically downloaded on first run

### ✅ Production-Ready
- Health checks for all services
- Proper dependency management
- Data persistence via Docker volumes
- Network isolation for security

### ✅ 100% On-Premise
- No external API dependencies
- All data stays local
- Complete data sovereignty

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│              Docker Network: rag-network                │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │  Qdrant  │  │  Ollama  │  │ Postgres │            │
│  │  :6334   │  │  :11434  │  │  :5432   │            │
│  └──────────┘  └──────────┘  └──────────┘            │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │ Wiki API │  │Prometheus│  │  Hector  │            │
│  │  :8080   │  │  :9090   │  │  :8080   │            │
│  └──────────┘  └──────────┘  └──────────┘            │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Service Configuration

### Hector Service

```yaml
hector:
  image: kadirpekel/hector:1.8.1  # Official Docker Hub image
  ports:
    - "8080:8080"  # A2A API
  volumes:
    - ./configs/enterprise-rag-lab-docker.yaml:/etc/hector/config.yaml:ro
      - ./docs:/examples/enterprise-rag/docs:ro  # Sample documentation
  depends_on:
    qdrant: { condition: service_healthy }
    ollama: { condition: service_healthy }
    postgres: { condition: service_healthy }
    wiki-api: { condition: service_healthy }
```

**Key Points:**
- Uses official `kadirpekel/hector:1.8.1` from Docker Hub
- Waits for all dependencies to be healthy before starting
- Configuration uses Docker service names (not localhost)
- Sample docs mounted for indexing

### Configuration Files

- **`enterprise-rag-lab-docker.yaml`**: Uses Docker service names
  - `host: "qdrant"` (not `localhost`)
  - `host: "postgres"` (not `localhost`)
  - `host: "http://ollama:11434"` (not `http://localhost:11434`)

- **`enterprise-rag-lab.yaml`**: Uses localhost (for local testing)

## Network Communication

All services communicate via Docker service names:

- Hector → Qdrant: `qdrant:6334`
- Hector → Ollama: `ollama:11434`
- Hector → PostgreSQL: `postgres:5432`
- Hector → Wiki API: `wiki-api:8080`

This provides:
- ✅ Secure inter-container communication
- ✅ No need to expose internal ports
- ✅ Automatic DNS resolution
- ✅ Network isolation

## Data Persistence

All data is stored in Docker volumes:

```yaml
volumes:
  qdrant-data:      # Vector embeddings
  ollama-data:      # Downloaded models (nomic-embed-text, qwen3)
  postgres-data:    # Knowledge base articles
  prometheus-data:  # Metrics history
```

**Benefits:**
- Data persists across container restarts
- Models cached (no re-download on restart)
- Easy backup/restore
- Can be removed with `docker-compose down -v`

## Startup Sequence

1. **Infrastructure Services** (Qdrant, Ollama, PostgreSQL, Wiki API)
   - Start in parallel
   - Health checks verify readiness

2. **Hector Service**
   - Waits for all dependencies (`depends_on` with `condition: service_healthy`)
   - Starts automatically when ready
   - Begins indexing on startup

3. **Prometheus**
   - Waits for Hector
   - Starts scraping metrics

## Usage

### Quick Start
```bash
./setup-docker.sh
```

### Manual Start
```bash
docker-compose up -d
```

### Check Status
```bash
docker-compose ps
```

### View Logs
```bash
docker-compose logs -f hector
```

### Stop Services
```bash
docker-compose down
```

### Clean Everything
```bash
docker-compose down -v  # Removes volumes too
```

## Troubleshooting

### Services Not Starting
```bash
# Check logs
docker-compose logs

# Check specific service
docker-compose logs hector
docker-compose logs ollama
```

### Models Not Downloading
```bash
# Manual download
docker exec lab-ollama ollama pull nomic-embed-text
docker exec lab-ollama ollama pull qwen3
```

### Network Issues
```bash
# Verify network exists
docker network ls | grep rag-network

# Inspect network
docker network inspect lab-rag-network
```

### Configuration Issues
```bash
# Validate config
docker exec lab-hector hector validate /etc/hector/config.yaml

# Check mounted files
docker exec lab-hector ls -la /etc/hector/
docker exec lab-hector ls -la /examples/enterprise-rag/docs/
```

## Production Considerations

For production deployments:

1. **Use Environment Variables**
   ```yaml
   environment:
     - POSTGRES_PASSWORD=${DB_PASSWORD}
     - OPENAI_API_KEY=${OPENAI_API_KEY}
   ```

2. **Add Resource Limits**
   ```yaml
   deploy:
     resources:
       limits:
         cpus: '2'
         memory: 4G
   ```

3. **Use Secrets Management**
   ```yaml
   secrets:
     - db_password
     - api_key
   ```

4. **Enable Logging**
   ```yaml
   logging:
     driver: "json-file"
     options:
       max-size: "10m"
       max-file: "3"
   ```

5. **Add Restart Policies**
   ```yaml
   restart: unless-stopped
   ```

## Next Steps

- Read [README.md](README.md) for detailed documentation
- Check [QUICKSTART.md](QUICKSTART.md) for quick setup
- Explore the blog post for enterprise patterns

