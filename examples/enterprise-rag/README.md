# Enterprise RAG Example

This example demonstrates a complete, production-ready enterprise on-premise RAG system with multiple data sources.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Lab Environment                       │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │  Qdrant  │  │  Ollama  │  │ Postgres │            │
│  │ (Vector) │  │(Embedding)│  │(Knowledge)│            │
│  └──────────┘  └──────────┘  └──────────┘            │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │ Wiki API │  │Prometheus │  │  Hector  │            │
│  │  (Mock)  │  │(Metrics)  │  │  Server  │            │
│  └──────────┘  └──────────┘  └──────────┘            │
│                                                          │
│  Advanced Features:                                      │
│  • Hybrid Search (keyword + vector)                     │
│  • LLM-based Re-ranking                                  │
│  • Multi-Query Expansion                                 │
│  • RAG Evaluation Tools                                  │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Data Sources

1. **Internal Documentation** (`./docs/internal/`)
   - Security policies
   - Deployment guides
   - Architecture documentation

2. **Knowledge Base** (PostgreSQL)
   - 10 sample knowledge articles
   - Categories: Security, Development, Operations, Compliance
   - Includes metadata (author, category, timestamps)

3. **Internal Wiki** (Mock REST API)
   - 5 wiki pages
   - Technical documentation
   - Infrastructure guides

## Prerequisites

- **Docker and Docker Compose** (that's it!)
- No need to install Go, Hector binary, or configure API keys
- Everything runs in containers

## Quick Start (One Command Setup)

```bash
cd examples/enterprise-rag
./setup-docker.sh
```

This single command will:
1. ✅ Start all services (Qdrant, Ollama, PostgreSQL, Wiki API, Hector, Prometheus)
2. ✅ Wait for services to be healthy
3. ✅ Download embedding model (nomic-embed-text)
4. ✅ Download LLM model (qwen3)
5. ✅ Initialize the complete RAG system

**That's it!** Your enterprise RAG system is ready to use.

## Manual Setup (Alternative)

If you prefer manual setup:

### 1. Start All Services

```bash
cd examples/enterprise-rag
docker-compose up -d
```

This starts:
- **Qdrant** (vector DB) on port 6334
- **Ollama** (embeddings + LLM) on port 11434
- **PostgreSQL** (knowledge base) on port 5433
- **Wiki API** (mock) on port 8081
- **Hector** (RAG server) on port 8080
- **Prometheus** (metrics) on port 9090

### 2. Initialize Ollama Models

```bash
# Wait for Ollama to be ready
sleep 30

# Pull embedding model
docker exec lab-ollama ollama pull nomic-embed-text

# Pull LLM model (qwen3 - native tool calling)
docker exec lab-ollama ollama pull qwen3
```

### 3. Verify Services

```bash
# Check Qdrant
curl http://localhost:6334/health

# Check Ollama
curl http://localhost:11434/api/tags

# Check PostgreSQL
docker exec lab-postgres pg_isready -U hector

# Check Wiki API
curl http://localhost:8081/health

# Check Hector
curl http://localhost:8080/health

# Check Prometheus
curl http://localhost:9090/-/healthy
```

Hector will automatically:
1. ✅ Index internal documentation from mounted `./docs/internal/`
2. ✅ Index knowledge base articles from PostgreSQL
3. ✅ Index wiki content from the REST API
4. ✅ Create vector embeddings using Ollama
5. ✅ Store everything in Qdrant

### 4. Test the System

```bash
# Query via Docker
docker exec lab-hector hector call \
  "What are our password requirements?" \
  --agent enterprise_assistant \
  --config /etc/hector/config.yaml

# Or use the API directly
curl -X POST http://localhost:8080/v1/agents/enterprise_assistant/call \
  -H "Content-Type: application/json" \
  -d '{"message": "What are our password requirements?"}'
```

## Example Queries

Try these queries to test multi-source RAG and advanced search features:

1. **Security Questions:**
   - "What are our password requirements?"
   - "How do we handle security incidents?"
   - "What is our data retention policy?"

2. **Operations Questions:**
   - "What is our deployment process?"
   - "How do we handle rollbacks?"
   - "What are our monitoring requirements?"

3. **Architecture Questions:**
   - "What is our system architecture?"
   - "How do services communicate?"
   - "What databases do we use?"

4. **Cross-Source Questions:**
   - "What security measures are in place for deployments?"
   - "How does our architecture support compliance?"

5. **Testing Advanced Search:**
   - Compare results with `search_mode: "vector"` vs `search_mode: "hybrid"`
   - Test ambiguous queries with `search_mode: "multi_query"`
   - Evaluate search quality using the `evaluate_rag` tool

## Service URLs

| Service | URL | Description |
|---------|-----|-------------|
| **Hector API** | http://localhost:8080 | A2A protocol endpoint |
| **Hector Health** | http://localhost:8080/health | Health check |
| **Hector Metrics** | http://localhost:8080/metrics | Prometheus metrics |
| **Qdrant Dashboard** | http://localhost:6334/dashboard | Vector database UI |
| **Ollama API** | http://localhost:11434 | Embeddings & LLM |
| **Prometheus** | http://localhost:9090 | Metrics dashboard |
| **Wiki API** | http://localhost:8081/health | Mock internal wiki |

## Monitoring

### Qdrant Dashboard
- URL: http://localhost:6334/dashboard
- View collections and vector counts
- Inspect indexed documents

### Prometheus
- URL: http://localhost:9090
- Query: `hector_agent_requests_total`
- View query latency, token usage, error rates

### Hector Metrics
- URL: http://localhost:8080/metrics
- Prometheus format metrics
- Real-time observability

## Troubleshooting

### Ollama not responding
```bash
docker logs lab-ollama
docker restart lab-ollama
```

### PostgreSQL connection issues
```bash
docker exec -it lab-postgres psql -U hector -d knowledge_base
SELECT COUNT(*) FROM knowledge_articles;
```

### Wiki API not working
```bash
docker logs lab-wiki-api
curl http://localhost:8081/api/pages
```

### Indexing issues
- Check Hector logs for errors
- Verify all services are running
- Check file permissions for docs directory
- Verify database credentials

## Architecture Details

### Docker Network
All services run in a shared `rag-network` for secure inter-container communication:
- Services use Docker service names (e.g., `qdrant`, `ollama`, `postgres`)
- No need to expose internal ports to host
- Only necessary ports are exposed for external access

### Configuration
- **Docker Config**: `configs/enterprise-rag-lab-docker.yaml` (uses service names)
- **Local Config**: `configs/enterprise-rag-lab.yaml` (uses localhost, for local testing)

### Data Persistence
All data is persisted in Docker volumes:
- `qdrant-data`: Vector embeddings
- `ollama-data`: Downloaded models
- `postgres-data`: Knowledge base articles
- `prometheus-data`: Metrics history

## Cleanup

```bash
# Stop all services (keeps data)
docker-compose down

# Stop and remove volumes (deletes all data)
docker-compose down -v

# Remove everything including images
docker-compose down -v --rmi all
```

## Advanced Features

This example showcases Hector's advanced RAG capabilities:

### Hybrid Search
The configuration uses `search_mode: "hybrid"` which combines:
- **Vector search**: Semantic similarity matching
- **Keyword search**: Exact term matching
- **Result fusion**: Reciprocal Rank Fusion (RRF) to combine results

This provides better recall for enterprise documentation with specific technical terms.

### LLM-based Re-ranking
Enabled with `rerank.enabled: true`:
- Re-ranks top results using LLM semantic understanding
- Improves relevance beyond simple vector similarity
- Uses fast/cheap LLM model (qwen3 or gpt-4o-mini) for cost efficiency

**How It Works:**
1. Initial search (vector/hybrid/keyword) retrieves candidate results
2. Reranker sends query + results to LLM for semantic evaluation
3. LLM ranks results by actual usefulness (not just vector similarity)
4. Scores are replaced with position-based values (1.0, 0.95, 0.90, ...)
5. Threshold filtering is applied AFTER reranking

**Score Semantics (Important):**
- **Before reranking**: Scores represent vector similarity (0.0-1.0, e.g., 0.8 = "80% similar")
- **After reranking**: Scores represent ranking position (1.0 = "ranked first by LLM")
- Original vector scores are **replaced**, not preserved
- Threshold semantics change: `threshold: 0.5` after reranking means "keep top 11 results or better"

**When to Use:**
- Vector search returns low similarity scores but may include relevant results
- Query requires deep semantic understanding
- Results need ordering by actual usefulness, not just vector distance

**When to Skip:**
- Vector search already returns high-quality results (saves latency and cost)
- Latency is critical (adds 100-500ms per search)
- Simple keyword matching is sufficient
- Cost constraints are strict (each search incurs LLM API cost)

**Performance:**
- Latency: +100-500ms per search (depends on LLM speed)
- Cost: ~$0.001-0.01 per search (for gpt-4o-mini)
- Token usage: ~500 chars per result × max_results

**Configuration Tips:**
```yaml
search:
  top_k: 10           # Final results to return
  threshold: 0.5      # Applied AFTER reranking (keeps top 11 or better)
  rerank:
    enabled: true
    llm: "reranker"   # Use fast model (gpt-4o-mini, qwen3)
    max_results: 20   # Only rerank top 20 (balance quality vs cost)
```

### Multi-Query Expansion
Available via `search_mode: "multi_query"`:
- Generates multiple query variations
- Searches each variation in parallel
- Merges results with score boosting

### RAG Evaluation
The `evaluate_rag` tool allows agents to:
- Measure search quality metrics
- Evaluate context precision and recall
- Assess answer relevance and faithfulness
- Compare different search configurations

### Vector Database Options
While this example uses Qdrant, you can switch to:
- **Weaviate**: Native hybrid search support
- **Milvus**: High-performance, large-scale
- **Chroma**: Lightweight, embedded
- **Pinecone**: Managed cloud service

## Next Steps

1. ✅ Verify all three sources index correctly
2. ✅ Test semantic search across all sources
3. ✅ Compare search modes: vector vs hybrid vs multi-query
4. ✅ Test re-ranking impact on result quality
5. ✅ Use `evaluate_rag` tool to measure performance
6. ✅ Verify metadata filtering works
7. ✅ Test incremental indexing (SQL)
8. ✅ Try different vector databases
9. ✅ Validate security features
10. ✅ Check observability metrics
11. ✅ Document any issues or limitations

## Lab Validation Checklist

- [ ] All services start successfully
- [ ] Ollama embedding model downloaded
- [ ] PostgreSQL database initialized with sample data
- [ ] Wiki API returns data
- [ ] Hector indexes all three sources
- [ ] Qdrant collections created
- [ ] Semantic search works across all sources
- [ ] Metadata filtering works
- [ ] Prometheus metrics available
- [ ] Multi-source queries return relevant results

