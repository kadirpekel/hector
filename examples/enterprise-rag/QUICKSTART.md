# Enterprise RAG Example - Quick Start Guide

Get a complete, production-ready RAG system running in **5 minutes** with a single command.

## Prerequisites

- Docker and Docker Compose installed
- 4GB+ free disk space (for models)
- 2GB+ free RAM

## One-Command Setup

```bash
cd examples/enterprise-rag
./setup-docker.sh
```

**That's it!** The script will:
1. Start all services
2. Download required models
3. Initialize the RAG system
4. Index all data sources

## Verify It Works

```bash
# Test the system
docker exec lab-hector hector call \
  "What are our password requirements?" \
  --agent enterprise_assistant \
  --config /etc/hector/config.yaml
```

Expected output:
```
The password requirements at our organization are:

- **Minimum Length**: Passwords must be at least 12 characters long.
- **Character Composition**: Must include uppercase, lowercase, numbers, and special characters.
...
```

## What You Get

✅ **Multi-Source RAG**: Directory, SQL, and API sources  
✅ **100% On-Premise**: No external API calls  
✅ **Parallel Search**: Fast queries across all sources  
✅ **Production-Ready**: Observability, health checks, metrics  
✅ **Zero Configuration**: Everything pre-configured  

## Access Points

- **Hector API**: http://localhost:8080
- **Qdrant Dashboard**: http://localhost:6334/dashboard
- **Prometheus**: http://localhost:9090

## Next Steps

- Read the full [README.md](README.md) for detailed documentation
- Try different queries to test multi-source search
- Explore the configuration in `configs/enterprise-rag-lab-docker.yaml`
- Check out the blog post for enterprise deployment patterns

## Troubleshooting

**Services not starting?**
```bash
docker-compose logs
```

**Models not downloading?**
```bash
docker exec lab-ollama ollama pull nomic-embed-text
docker exec lab-ollama ollama pull qwen3
```

**Need to restart?**
```bash
docker-compose restart hector
```

