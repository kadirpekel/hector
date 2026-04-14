# Configuration Guide

Hector uses a single YAML file to define your entire agent application: LLMs, tools, agents, guardrails, RAG pipelines, and more. This guide covers the structure, patterns, and best practices for writing configurations.

## File Structure

The default config file is `.hector/config.yaml` (created by `hector init`). You can also use `hector.yaml` in the current directory or specify a path with `--config`.

```yaml
# .hector/config.yaml

name: "my-app"
description: "My agent application"

llms:        # LLM provider definitions
  # ...
tools:       # Tool definitions
  # ...
agents:      # Agent tree
  # ...
guardrails:  # Safety policies
  # ...
vector_stores:    # Vector database config
  # ...
embedders:        # Embedding model config
  # ...
document_stores:  # RAG data sources
  # ...
```

### Path Resolution

Relative paths (e.g., `instruction_file: ./prompts/agent.md`) resolve relative to the config file's directory.

---

## Environment Variables

Use `${VAR_NAME}` for secrets and environment-specific values:

```yaml
llms:
  claude:
    provider: anthropic
    api_key: ${ANTHROPIC_API_KEY}

server:
  port: ${PORT:-8080}          # Default value syntax
```

Hector automatically loads `.env` files from the working directory.

---

## Configuration Profiles

Overlay environment-specific settings on a base config:

```bash
# Base:    hector.yaml
# Profile: hector.prod.yaml (merged on top)
hector serve --profile prod
```

Use profiles for dev/prod differences: database URLs, model selection, logging levels.

---

## Common Patterns

### Pattern 1: Simple Chatbot

The minimal configuration for a useful agent:

```yaml
llms:
  default:
    provider: anthropic
    model: claude-sonnet-4
    api_key: ${ANTHROPIC_API_KEY}

agents:
  assistant:
    llm: default
    instruction: |
      You are a helpful AI assistant. Be concise and accurate.
      When you don't know something, say so.
```

### Pattern 2: Agent with Tools

Add web search and file editing capabilities:

```yaml
llms:
  default:
    provider: openai
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}

tools:
  search:
    type: function
    handler: web_search

  editor:
    type: function
    handler: text_editor

  github:
    type: mcp
    transport: stdio
    command: npx
    args: [-y, "@modelcontextprotocol/server-github"]
    env:
      GITHUB_PERSONAL_ACCESS_TOKEN: ${GITHUB_TOKEN}
    filter: [get_issue, create_issue, search_repositories]

agents:
  developer:
    llm: default
    tools: [search, editor, github]
    instruction: |
      You are a software development assistant.
      Use tools to research, edit code, and manage issues.
```

### Pattern 3: Multi-Agent Pipeline

A content creation pipeline with sequential orchestration:

```yaml
llms:
  fast:
    provider: anthropic
    model: claude-sonnet-4
    api_key: ${ANTHROPIC_API_KEY}

  precise:
    provider: openai
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}

agents:
  blog_pipeline:
    type: sequential
    sub_agents: [researcher, writer, editor]

  researcher:
    llm: fast
    tools: [search]
    instruction: "Research the topic. Return key facts and sources."

  writer:
    llm: fast
    instruction: "Write a blog post from the research. Be engaging."

  editor:
    llm: precise
    instruction: "Fix grammar, improve clarity, verify facts."

tools:
  search:
    type: function
    handler: web_search
```

### Pattern 4: RAG Knowledge Base

Turn a documentation folder into a searchable knowledge base:

```yaml
llms:
  default:
    provider: anthropic
    model: claude-sonnet-4
    api_key: ${ANTHROPIC_API_KEY}

embedders:
  default:
    provider: openai
    model: text-embedding-3-small
    api_key: ${OPENAI_API_KEY}

vector_stores:
  default:
    type: chromem

document_stores:
  docs:
    source:
      type: directory
      include: ["./knowledge/**/*.md", "./knowledge/**/*.pdf"]
    embedder: default
    vector_store: default
    watch: true                    # Re-index on file changes
    incremental_indexing: true     # Only re-index changed files

agents:
  support:
    llm: default
    instruction: |
      You are a support agent. Answer questions using the knowledge base.
      Always cite your sources. If you can't find an answer, say so.
    context:
      include_context: true        # Auto-inject relevant docs
      include_context_limit: 5
```

### Pattern 5: Production with Security

Full production setup with auth, guardrails, rate limiting, and monitoring:

```yaml
llms:
  default:
    provider: anthropic
    model: claude-sonnet-4
    api_key: ${ANTHROPIC_API_KEY}
    max_tokens: 4096

guardrails:
  strict:
    enabled: true
    input:
      chain_mode: fail_fast
      injection: { enabled: true, action: block, severity: critical }
      sanitizer: { enabled: true, trim_whitespace: true, strip_html: true }
      length: { enabled: true, max_length: 50000 }
    output:
      pii:
        enabled: true
        detect_email: true
        detect_phone: true
        detect_ssn: true
        detect_credit_card: true
        redact_mode: mask
      content:
        enabled: true
        blocked_patterns: ["sk-[a-zA-Z0-9]{20,}"]
    moderation:
      enabled: true
      strategy: openai
      openai: { model: omni-moderation-latest, threshold: 0.8 }

agents:
  assistant:
    llm: default
    guardrails: strict
    instruction: |
      You are a helpful assistant for Acme Corp customers.
    reasoning:
      max_iterations: 20
    context:
      strategy: token_window
      budget: 16000
```

Start the server with security flags:

```bash
hector serve \
  --config production.yaml \
  --database postgres://user:pass@db:5432/hector \
  --auth-jwks-url "https://auth.example.com/.well-known/jwks.json" \
  --auth-issuer "https://auth.example.com/" \
  --metrics \
  --tracing-endpoint "jaeger:4317"
```

### Pattern 6: Webhook-Triggered Agent

An agent that runs automatically when a GitHub PR is opened:

```yaml
llms:
  default:
    provider: openai
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}

tools:
  github:
    type: mcp
    transport: stdio
    command: npx
    args: [-y, "@modelcontextprotocol/server-github"]
    env:
      GITHUB_PERSONAL_ACCESS_TOKEN: ${GITHUB_TOKEN}

agents:
  reviewer:
    llm: default
    tools: [github]
    instruction: "Review the PR and post a comment with your findings."
    trigger:
      type: webhook
      path: /webhooks/github
      methods: [POST]
      secret: ${WEBHOOK_SECRET}
      signature_header: X-Hub-Signature-256
      webhook_input:
        template: "Review PR #{{.Body.pull_request.number}} in {{.Body.repository.full_name}}"
        session_id: "pr-{{.Body.pull_request.number}}"
      response:
        mode: async
```

---

## Validation

Always validate before deploying:

```bash
# Validate syntax and references
hector validate --config production.yaml

# JSON Schema available at runtime
curl http://localhost:8080/schema
```

## Hot Reload

Use `--watch` for development. Config changes apply without restart:

```bash
hector serve --watch
```

In production, configs stored in the database (via the Admin API) hot-reload automatically.

---

## Key Sections Reference

| Section | Purpose | Guide |
|---------|---------|-------|
| `llms` | LLM providers and models | [Configuration Reference](../reference/configuration.md#llms) |
| `tools` | Tool definitions | [Tools Guide](./tools.md) |
| `agents` | Agent tree and orchestration | [Agents Guide](./agents.md) |
| `guardrails` | Safety policies | [Guardrails Guide](./guardrails.md) |
| `vector_stores` | Vector database backends | [Knowledge Guide](./knowledge.md) |
| `embedders` | Embedding models | [Knowledge Guide](./knowledge.md) |
| `document_stores` | RAG data sources | [Knowledge Guide](./knowledge.md) |

## Next Steps

- [Full Configuration Reference](../reference/configuration.md): Complete YAML schema with all fields
- [CLI Reference](../reference/cli.md): Server flags, environment variables, and precedence rules
- [Agents Guide](./agents.md): Agent types and orchestration patterns
