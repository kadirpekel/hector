---
layout: default
title: Knowledge Management
nav_order: 2
parent: Knowledge & RAG
description: "Organizing and retrieving information"
---

# Knowledge Management

Organize and structure information for optimal retrieval and use by your agents.

## Information Architecture

### Domain-Based Organization

Structure your knowledge base around business domains and use cases:

```yaml
# Document stores configuration
document_stores:
  # Product & Service Knowledge
  product_specs:
    type: "qdrant"
    url: "http://localhost:6333"
    collection: "product_specs"
    api_key: "${QDRANT_API_KEY}"

  # Technical Documentation
  api_reference:
    type: "qdrant"
    url: "http://localhost:6333"
    collection: "api_docs"

  # Support & Troubleshooting
  troubleshooting:
    type: "qdrant"
    url: "http://localhost:6333"
    collection: "troubleshooting"

  # Company & Policy
  company_policies:
    type: "qdrant"
    url: "http://localhost:6333"
    collection: "policies"

# Agent using multiple document stores
agents:
  support_agent:
    name: "Support Agent"
    llm: "gpt-4o"
    document_stores: ["product_specs", "api_reference", "troubleshooting"]
```

### Document Metadata

Documents can include metadata for better organization and search:

```yaml
# Example document with metadata
document_content = """
# API Authentication

Authentication is required for all API endpoints.

## Methods

### JWT Tokens
Use JWT tokens for stateless authentication...

### API Keys
API keys can be used for simple integrations...
"""

# Metadata is stored as simple key-value pairs
metadata = {
    "category": "api",
    "difficulty": "intermediate",
    "last_updated": "2024-01-15",
    "tags": "authentication,security,jwt"
}
```

## Search Configuration

### Basic Search Setup

```yaml
agents:
  assistant:
    name: "Assistant"
    llm: "gpt-4o"
    document_stores: ["knowledge_base"]

    search:
      enabled: true
      top_k: 10
      threshold: 0.7
```

### Advanced Search Features

```yaml
agents:
  researcher:
    name: "Research Assistant"
    llm: "claude-3-5-sonnet"
    document_stores: ["research_docs", "api_docs"]

    search:
      enabled: true
      top_k: 15
      threshold: 0.75
      max_context_length: 4000

    prompt:
      tool_usage: |
        SEARCH DECISION FRAMEWORK:

        🔍 SEARCH when user asks about:
        - Specific features or capabilities ("Does Hector support X?")
        - Technical specifications ("What are the API rate limits?")
        - Troubleshooting ("I'm getting error Y, how do I fix it?")
        - Documentation ("How do I configure Z?")

        🚫 DON'T search for:
        - General knowledge ("What is AI?")
        - Creative tasks ("Write a story about...")
        - Personal opinions ("What do you think about...")
        - Mathematical calculations
        - Current events or news
```

## Document Store Types

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

#### Production Setup
```yaml
document_stores:
  production_kb:
    type: "qdrant"
    url: "https://api.qdrant.cloud"
    collection: "production_docs"
    api_key: "${QDRANT_CLOUD_API_KEY}"

    # Performance tuning
    timeout: "30s"
    batch_size: 100
    parallel_requests: 3

    # High availability
    replicas: 3
    shard_number: 2
```

### Chroma (Alternative)

```yaml
document_stores:
  local_docs:
    type: "chroma"
    url: "http://localhost:8000"
    collection: "my_collection"
    persist_directory: "./chroma_db"
```

## Real-World Implementation Examples

### Customer Support Knowledge Base

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
    document_stores: ["support_kb"]

    search:
      enabled: true
      top_k: 5          # Focused results for support
      threshold: 0.8     # High relevance for accuracy

    prompt:
      system_role: |
        You are a helpful customer support assistant.
        Use the search tool to find accurate information from our knowledge base.
        Always cite your sources and provide step-by-step solutions.
```

### Developer Documentation Portal

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

agents:
  dev_assistant:
    name: "Developer Assistant"
    llm: "claude-3-5-sonnet"
    document_stores: ["api_reference", "code_examples"]

    search:
      enabled: true
      top_k: 8
      threshold: 0.75

    prompt:
      tool_usage: |
        For API questions: Search api_reference first
        For code examples: Search code_examples
        Always provide working code examples
```

## Search Tool Configuration

The search tool is configured per agent and references document stores:

```yaml
agents:
  my_agent:
    name: "My Agent"
    llm: "gpt-4o"
    document_stores: ["knowledge_base", "api_docs"]

    search:
      enabled: true
      top_k: 10        # Number of results to retrieve
      threshold: 0.7    # Minimum similarity threshold
      max_context_length: 4000 # Maximum context tokens
```

## Document Indexing

Documents are automatically indexed when added to document stores:

```yaml
# Documents are indexed based on their content and metadata
# - Text content is embedded for semantic search
# - Metadata is stored for filtering and organization
# - File type detection (markdown, code, text, etc.)
```

## Best Practices

### Content Organization
1. **Organize by domain** - Group related content together
2. **Use descriptive names** - Clear document store and collection names
3. **Include metadata** - Add relevant tags and categories
4. **Keep content current** - Regular updates for accuracy

### Search Optimization
1. **Tune similarity thresholds** - Balance precision vs. recall
2. **Configure result limits** - Appropriate for use case
3. **Use prompt guidance** - Help agents decide when to search
4. **Monitor performance** - Track search effectiveness

### Document Quality
1. **Clear structure** - Use headings and sections
2. **Descriptive titles** - Help with search relevance
3. **Complete examples** - Include working code and configurations
4. **Regular maintenance** - Update outdated information

## Troubleshooting

### Common Issues

#### Poor Search Relevance
**Symptoms:** Users getting irrelevant results

**Solutions:**
```yaml
agents:
  my_agent:
search:
  threshold: 0.8      # Increase threshold
  top_k: 5         # Reduce result count
```

#### No Search Results
**Symptoms:** Search returns empty results

**Solutions:**
```yaml
# Check document store configuration
document_stores:
  my_store:
    type: "qdrant"
    url: "http://localhost:6333"  # Ensure URL is correct
    collection: "my_collection"   # Ensure collection exists
```

#### Slow Search Performance
**Symptoms:** Search queries taking too long

**Solutions:**
```yaml
agents:
  my_agent:
search:
  top_k: 5         # Reduce for speed
  max_context_length: 2000 # Limit context size
```

## Performance Considerations

### Document Store Selection
- **Qdrant**: Best for production, scalable, feature-rich
- **Chroma**: Good for development, simple setup, local storage

### Search Performance
- **Result limits**: Balance between context and speed
- **Similarity thresholds**: Higher = more relevant but fewer results
- **Context length**: More context = better answers but slower

## See Also

- **[Document Stores](document-stores)** - Semantic search setup and configuration
- **[Built-in Tools](../tools-actions/built-in-tools)** - Search tool capabilities and usage
- **[Memory Management](../memory-context)** - Agent memory systems and context management