---
layout: default
title: Document Stores
nav_order: 1
parent: Knowledge & RAG
description: "Semantic search and RAG setup"
---

# Document Stores

Enable Retrieval-Augmented Generation (RAG) to give your agents domain knowledge through semantic search.

## Quick Start

### 1. Define Document Store

```yaml
document_stores:
  company_knowledge:
    type: "qdrant"
    url: "http://localhost:6333"
    collection: "company_docs"
    api_key: "${QDRANT_API_KEY}"

  api_docs:
    type: "qdrant"
    url: "http://localhost:6333"
    collection: "api_reference"
```

### 2. Link to Agent

```yaml
agents:
  support_agent:
    llm: "gpt-4o"
    document_stores:
      - "company_knowledge"
      - "api_docs"

    search:
      enabled: true
      result_limit: 10
      min_similarity: 0.7
```

### 3. Automatic Usage

```bash
# Agent automatically uses search when relevant
User: How do I reset my password?
Agent: Let me check our documentation...
[Automatic search in company_knowledge and api_docs]
Agent: Here's how to reset your password: [answer with citations]
```

## How RAG Works

### The RAG Pipeline

1. **Question Analysis** - Agent determines if external knowledge is needed
2. **Semantic Search** - Query embedded and compared against document vectors
3. **Context Retrieval** - Most relevant chunks retrieved with metadata
4. **Answer Synthesis** - LLM generates response using retrieved context
5. **Source Citation** - Response includes references to source documents

### Search Decision Making

**When to search:**
- User asks about specific features or capabilities
- Technical questions requiring documentation
- Troubleshooting or error resolution
- Policy or procedural questions

**When NOT to search:**
- General knowledge questions
- Creative tasks or brainstorming
- Mathematical calculations
- Current events or news

## Document Store Configuration

### Qdrant (Recommended)

#### Basic Setup
```yaml
document_stores:
  my_docs:
    type: "qdrant"
    url: "http://localhost:6333"
    collection: "my_collection"
    api_key: "${QDRANT_API_KEY}"  # Optional
```

#### Advanced Configuration
```yaml
document_stores:
  enterprise_kb:
    type: "qdrant"
    url: "https://api.qdrant.cloud"
    collection: "enterprise_docs"
    api_key: "${QDRANT_CLOUD_API_KEY}"

    # Performance tuning
    timeout: "30s"
    batch_size: 100
    parallel_requests: 3

    # Security
    tls_enabled: true
    verify_ssl: true

    # Monitoring
    health_check_interval: "60s"
```

#### Production Deployment
```yaml
document_stores:
  production_docs:
    type: "qdrant"
    url: "https://qdrant.production.company.com:6333"
    collection: "prod_docs"

    # High availability
    replicas: 3
    shard_number: 2

    # Performance
    write_consistency_factor: 2
    read_consistency: "majority"

    # Backup
    backup_enabled: true
    backup_schedule: "0 2 * * 0"  # Weekly
```

### Chroma (Alternative)

#### Local Setup
```yaml
document_stores:
  local_docs:
    type: "chroma"
    url: "http://localhost:8000"
    collection: "my_collection"
    persist_directory: "./chroma_db"
```

#### Cloud Deployment
```yaml
document_stores:
  cloud_docs:
    type: "chroma"
    url: "https://chroma-cloud.company.com"
    collection: "cloud_collection"
    api_key: "${CHROMA_API_KEY}"
```

### Pinecone (Enterprise)

```yaml
document_stores:
  enterprise_search:
    type: "pinecone"
    api_key: "${PINECONE_API_KEY}"
    environment: "us-west1-gcp"
    index_name: "enterprise-docs"

    # Vector configuration
    dimension: 1536  # OpenAI ada-002 dimensions
    metric: "cosine"
    pods: 2
```

## Search Configuration

### Basic Search Setup

```yaml
search:
  enabled: true
  result_limit: 10        # Number of results to retrieve
  min_similarity: 0.7    # Minimum similarity threshold
  max_tokens: 4000       # Maximum context tokens per result

  # Search behavior
  reranking: true        # Re-rank results for better quality
  hybrid_search: false   # Combine semantic + keyword search
  query_expansion: true  # Expand queries with synonyms
```

### Advanced Search Features

#### Reranking for Quality
```yaml
search:
  reranking:
    enabled: true
    model: "cross-encoder"  # More accurate but slower
    top_k: 20              # Re-rank top 20 results
    threshold: 0.8         # Only keep results above threshold
```

#### Hybrid Search (Semantic + Keyword)
```yaml
search:
  hybrid_search:
    enabled: true
    semantic_weight: 0.7   # Weight for semantic search
    keyword_weight: 0.3    # Weight for keyword search
    keyword_boost: 1.2     # Boost for exact matches
```

#### Query Enhancement
```yaml
search:
  query_expansion:
    enabled: true
    synonyms: true         # Add related terms
    stemming: true         # Handle word variations
    spell_check: true      # Correct typos
```

### Performance Optimization

#### Caching Strategy
```yaml
search:
  caching:
    enabled: true
    ttl: "1_hour"          # Cache results for 1 hour
    max_size: "1GB"       # Maximum cache size
    strategy: "lru"        # Least recently used eviction

  # Batch processing for efficiency
  batch_processing:
    enabled: true
    batch_size: 10         # Process multiple queries together
    max_wait_time: "100ms" # Maximum wait for batching
```

#### Index Optimization
```yaml
search:
  index_optimization:
    rebuild_frequency: "weekly"
    compaction_enabled: true
    dead_chunk_cleanup: true
    vector_compression: true
```

## Real-World Implementation Examples

### Customer Support Knowledge Base

#### Setup Configuration
```yaml
document_stores:
  support_kb:
    type: "qdrant"
    url: "http://localhost:6333"
    collection: "customer_support"

agents:
  support_agent:
    name: "Customer Support Assistant"
    llm: "gpt-4o"
    document_stores:
      - "support_kb"

    search:
      enabled: true
      result_limit: 5          # Focused results for support
      min_similarity: 0.8     # High relevance for accuracy
      reranking: true         # Improve result quality

    prompt:
      system_role: |
        You are a helpful customer support assistant.
        Use the search tool to find accurate information from our knowledge base.
        Always cite your sources and provide step-by-step solutions.
```

#### Usage Example
```bash
# Customer asks about password reset
User: I forgot my password and need to reset it
Agent: [Searches support_kb for "password reset"]
Result: Found relevant docs in support_kb
Agent: Here's how to reset your password:
1. Go to our login page
2. Click "Forgot Password"
3. Enter your email address
4. Check your email for reset link

Source: Customer Support Guide, Section 3.2
```

### Developer Documentation Portal

#### Multi-Store Setup
```yaml
document_stores:
  # API documentation
  api_reference:
    type: "qdrant"
    collection: "api_docs"

  # Code examples
  code_examples:
    type: "qdrant"
    collection: "examples"

  # Troubleshooting
  troubleshooting:
    type: "qdrant"
    collection: "troubleshooting"

agents:
  dev_assistant:
    name: "Developer Assistant"
    llm: "claude-3-5-sonnet"
    document_stores:
      - "api_reference"
      - "code_examples"
      - "troubleshooting"

    search:
      enabled: true
      result_limit: 8
      min_similarity: 0.75

    prompt:
      tool_usage: |
        For API questions: Search api_reference first
        For code examples: Search code_examples
        For errors: Search troubleshooting
        Always provide working code examples
```

#### Query Routing
```yaml
prompt:
  reasoning_instructions: |
    1. Analyze user query type
    2. Route to appropriate document store
    3. Use specific search parameters per store
    4. Combine results when needed
```

## Performance Tuning

### Search Performance Metrics

#### Monitoring Dashboard
```yaml
monitoring:
  metrics:
    - search_latency: "<100ms"
    - result_relevance: ">0.8"
    - cache_hit_rate: ">80%"
    - index_freshness: "<1_hour"
```

#### Performance Benchmarks
```yaml
benchmarks:
  small_dataset: "<50ms"     # < 1000 documents
  medium_dataset: "<200ms"   # 1000-10000 documents
  large_dataset: "<500ms"    # > 10000 documents
```

### Scaling Considerations

#### Horizontal Scaling
```yaml
scaling:
  horizontal:
    enabled: true
    replicas: 3
    load_balancer: "round_robin"
    health_check_interval: "30s"
```

#### Vertical Scaling
```yaml
scaling:
  vertical:
    memory_limit: "8GB"
    cpu_limit: "4"
    vector_dimensions: 1536
```

## Security Considerations

### Access Control

#### API Key Authentication
```yaml
document_stores:
  secure_docs:
    type: "qdrant"
    url: "https://secure.qdrant.company.com"
    api_key: "${QDRANT_API_KEY}"
    tls_enabled: true
    verify_ssl: true
```

#### Role-Based Access
```yaml
security:
  rbac:
    enabled: true
    roles:
      - name: "read_only"
        permissions: ["read"]
      - name: "admin"
        permissions: ["read", "write", "delete"]

    agents:
      support_agent: "read_only"
      admin_agent: "admin"
```

### Data Protection

#### Encryption at Rest
```yaml
security:
  encryption:
    enabled: true
    key_management: "aws_kms"
    algorithm: "AES256"
```

#### Network Security
```yaml
security:
  network:
    allowed_ips: ["10.0.0.0/8", "172.16.0.0/12"]
    tls_required: true
    certificate_validation: true
```

## Troubleshooting Guide

### Common Issues

#### 1. Poor Search Relevance
**Symptoms:** Users getting irrelevant or no results

**Diagnosis:**
```bash
# Check similarity threshold
curl -X GET "http://localhost:6333/collections/my_collection/info"
```

**Solutions:**
```yaml
search:
  min_similarity: 0.8      # Increase threshold
  reranking: true         # Enable reranking
  result_limit: 5         # Reduce for quality
  query_expansion: true   # Enhance queries
```

#### 2. Slow Search Performance
**Symptoms:** Search queries taking >500ms

**Diagnosis:**
```bash
# Check index health
curl -X GET "http://localhost:6333/collections/my_collection"
```

**Solutions:**
```yaml
search:
  caching: true           # Enable result caching
  batch_processing: true  # Process queries in batches
  result_limit: 5         # Reduce result count
```

#### 3. Index Corruption
**Symptoms:** Inconsistent search results, missing documents

**Diagnosis:**
```bash
# Check collection status
curl -X GET "http://localhost:6333/collections"
```

**Solutions:**
```yaml
maintenance:
  index_rebuild:
    enabled: true
    schedule: "0 2 * * 0"  # Weekly rebuild
    backup_before_rebuild: true
```

### Debugging Tools

#### Search Query Analysis
```yaml
debug:
  search_analysis:
    enabled: true
    log_queries: true
    log_results: true
    performance_metrics: true
```

#### Vector Similarity Testing
```bash
# Test vector similarity
curl -X POST "http://localhost:6333/collections/my_collection/points/search" \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, 0.3, ...],
    "limit": 5,
    "with_payload": true
  }'
```

## Best Practices Summary

### Setup & Configuration
1. **Start simple** - Begin with basic Qdrant setup
2. **Test thoroughly** - Validate search results before production
3. **Monitor performance** - Track latency and relevance metrics
4. **Scale gradually** - Add complexity as needs grow

### Content Organization
1. **Domain-based stores** - Group related content together
2. **Consistent metadata** - Use standard schemas across stores
3. **Clear naming** - Descriptive collection and store names
4. **Version management** - Track content changes over time

### Search Optimization
1. **Tune thresholds** - Balance precision vs. recall
2. **Use reranking** - Improve result quality for important queries
3. **Cache results** - Improve performance for repeated queries
4. **Monitor usage** - Learn from search patterns

### Production Deployment
1. **Security first** - Implement proper access controls
2. **High availability** - Configure redundancy and backups
3. **Performance monitoring** - Track system health continuously
4. **Regular maintenance** - Keep indexes optimized and current

## Advanced Features

### Custom Embeddings

```yaml
embeddings:
  custom_model:
    enabled: true
    model: "sentence-transformers/all-MiniLM-L6-v2"
    dimension: 384
    batch_size: 32
```

### Multi-Modal Search

```yaml
search:
  multimodal:
    enabled: true
    image_search: true
    audio_search: false
    video_search: false
```

### Real-Time Updates

```yaml
updates:
  real_time:
    enabled: true
    webhook_url: "https://your-app.com/webhooks/documents"
    batch_size: 10
    retry_attempts: 3
```

## Integration Examples

### Slack Bot with RAG

```yaml
agents:
  slack_bot:
    name: "Slack Assistant"
    llm: "gpt-4o"

    # Document stores for different knowledge domains
    document_stores:
      - "company_handbook"
      - "product_docs"
      - "troubleshooting"

    search:
      enabled: true
      result_limit: 3  # Concise responses for chat
      min_similarity: 0.85  # High relevance for accuracy

    prompt:
      system_role: |
        You are a helpful assistant in our company Slack.
        Use search to find accurate information from our documentation.
        Keep responses concise but complete.
        Always cite your sources.
```

### API Documentation Assistant

```yaml
agents:
  api_assistant:
    name: "API Documentation Helper"
    llm: "claude-3-5-sonnet"

    document_stores:
      - "api_reference"
      - "sdk_examples"
      - "integration_guides"

    search:
      enabled: true
      search_types: ["content", "function", "endpoint"]
      result_limit: 10

    prompt:
      tool_usage: |
        For API questions:
        1. Search api_reference for endpoint documentation
        2. Search sdk_examples for code samples
        3. Search integration_guides for setup instructions

        Always provide:
        - Complete code examples
        - Parameter explanations
        - Error handling guidance
```

## Monitoring & Analytics

### Usage Analytics

```yaml
analytics:
  enabled: true
  track:
    - search_queries: true
    - result_clicks: true
    - search_success_rate: true
    - popular_documents: true
    - query_patterns: true
```

### Performance Monitoring

```yaml
monitoring:
  metrics:
    - search_latency: "<200ms"
    - result_relevance: ">0.8"
    - cache_efficiency: ">75%"
    - index_health: "healthy"
```

## See Also

- **[Knowledge Management](knowledge-management)** - Advanced content organization and lifecycle management
- **[Built-in Tools](../tools-actions/built-in-tools)** - Comprehensive search tool capabilities and usage examples
- **[Memory Management](../memory-context)** - How agents use long-term memory for persistent knowledge
- **[Advanced Reasoning](../intelligence-reasoning/advanced-reasoning)** - Complex reasoning with integrated knowledge retrieval
