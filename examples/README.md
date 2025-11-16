# Hector Examples

This directory contains example implementations and configurations for common Hector use cases.

## Available Examples

### Enterprise RAG System

A complete, production-ready RAG system that demonstrates:

- **Multi-source indexing**: Directory, SQL database, and REST API
- **Parallel search**: Concurrent queries across all sources
- **100% on-premise**: No external API dependencies
- **Enterprise features**: Observability, health checks, metrics

**Quick Start:**
```bash
cd enterprise-rag
./setup-docker.sh
```

See [enterprise-rag/README.md](./enterprise-rag/README.md) for detailed documentation.

## Contributing Examples

When adding new examples:

1. Create a new directory under `examples/` with a descriptive name
2. Include a `README.md` with:
   - Overview of what the example demonstrates
   - Prerequisites
   - Setup instructions
   - Usage examples
3. Keep configurations minimal but complete
4. Include sample data if needed
5. Document any special requirements

