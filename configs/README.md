# Example Configurations

This directory contains curated example configurations for Hector. Each file demonstrates a specific feature or use case.

## Prompt Configuration

**`prompt-examples.yaml`** - Comprehensive prompt configuration examples
- 10 different prompt configuration patterns
- Pure strategy defaults, role customization, user guidance
- Full prompt_slots customization vs system_prompt override
- RAG integration, multi-agent orchestration
- Referenced in: [Prompts Guide](../docs/core-concepts/prompts.md)

## Essential Examples (11 files)

### Single Agent Examples

**`coding.yaml`** (33K) - Comprehensive coding assistant
- Full Cursor-like AI pair programming assistant
- Claude Sonnet 4, semantic search, chain-of-thought reasoning
- Rich toolset (git, npm, docker, testing, linters)
- Referenced in: [Build Your Own Cursor Tutorial](../docs/TUTORIAL_CURSOR.md)

**`research-assistant.yaml`** (6.0K) - Multi-agent research system
- LLM-driven orchestration with supervisor strategy
- Multiple specialized agents (coordinator, researcher, writer)
- Referenced in: [Multi-Agent Tutorial](../docs/TUTORIAL_MULTI_AGENT.md)

### Multi-Agent & External Integration

**`mixed-agents-example.yaml`** (5.3K) - Native + external A2A agents
- Demonstrates mixing local and remote A2A agents
- LLM-driven routing between agent types
- Referenced in: [External Agents Guide](../docs/EXTERNAL_AGENTS.md)

**`orchestrator-example.yaml`** (6.3K) - Multi-agent orchestration
- Supervisor strategy with agent delegation
- Task decomposition and synthesis
- LLM-driven specialist routing

**`external-agent-example.yaml`** (1.6K) - Connect to external A2A agents
- Simple external agent integration
- HTTP-based A2A protocol

### Feature-Specific Examples

**`auth-example.yaml`** (4.9K) - JWT authentication
- OAuth2/OIDC integration (Auth0, Keycloak, etc.)
- Public, internal, and private agent visibility
- Referenced in: [Authentication Guide](../docs/AUTHENTICATION.md)

**`long-term-memory-example.yaml`** (4.0K) - Long-term memory
- Session-scoped persistent memory
- Vector storage with semantic recall
- Qdrant + Ollama integration

**`memory-strategies-example.yaml`** (2.1K) - Memory strategies
- Summary buffer vs buffer window comparison
- Token-based management examples

**`tools-mcp-example.yaml`** (7.7K) - MCP protocol integration
- Multi-agent system with MCP tools
- Composio integration for 150+ apps
- Custom MCP server examples

**`security-example.yaml`** (1.4K) - Security & visibility
- Agent visibility control (public/internal/private)
- Tool sandboxing and restrictions
- Command whitelisting

**`structured-output-example.yaml`** (5.8K) - Structured JSON output
- JSON schema validation for reliable responses
- Sentiment analysis, data extraction, classification examples
- Works with OpenAI, Anthropic, Gemini

**`observability-example.yaml`** - Prometheus metrics & OpenTelemetry tracing
- Distributed tracing with Jaeger
- Prometheus metrics collection
- Grafana dashboard integration
- Referenced in: [Observability Guide](../docs/core-concepts/observability.md)

**`task-sql-example.yaml`** (849B) - SQL task persistence
- Task persistence with SQL backend
- SQLite/PostgreSQL/MySQL support

## Usage

```bash
# Run any example configuration
hector serve --config configs/<example>.yaml

# Example: Start the coding assistant
hector serve --config configs/coding.yaml
hector chat coding_assistant
```

## Prerequisites

Some examples require additional services:

- **Semantic search** (coding.yaml, research-assistant.yaml):
  - Qdrant: `docker run -p 6334:6333 qdrant/qdrant`
  - Ollama: `ollama pull nomic-embed-text`

- **SQL persistence** (task-sql-example.yaml):
  - PostgreSQL/MySQL/SQLite database

- **MCP tools** (tools-mcp-example.yaml):
  - MCP server running (e.g., Composio)

- **Observability** (observability-example.yaml):
  - `docker-compose -f deployments/docker-compose.observability.yaml up -d`
  - Starts Jaeger, Prometheus, and Grafana

## Documentation References

- [Configuration Reference](../docs/CONFIGURATION.md)
- [Building Agents](../docs/AGENTS.md)
- [Multi-Agent Tutorial](../docs/TUTORIAL_MULTI_AGENT.md)
- [Build Your Own Cursor](../docs/TUTORIAL_CURSOR.md)
- [Authentication Guide](../docs/AUTHENTICATION.md)
- [External Agents Guide](../docs/EXTERNAL_AGENTS.md)

---

**Note:** All examples use environment variables for API keys (e.g., `${OPENAI_API_KEY}`, `${ANTHROPIC_API_KEY}`).
