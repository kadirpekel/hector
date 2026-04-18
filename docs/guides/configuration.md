# Configuration Guide

Hector separates configuration into two layers:

- **App config** (YAML file): Defines *what* your application does — agents, LLMs, tools, guardrails, RAG pipelines.
- **Server config** (CLI flags / env vars): Controls *how* the server runs — port, database, auth, logging.

This guide covers the app config YAML file. For server flags, see the [CLI Reference](../reference/cli.md).

## File Location

The default config file is `.hector/config.yaml`, created automatically on first `hector serve` or explicitly with `hector init`.

Specify a different path with:

```bash
hector serve -c path/to/config.yaml
```

## Structure

```yaml
# .hector/config.yaml

llms:            # LLM provider definitions
  # ...
tools:           # Tool definitions (MCP, function, command)
  # ...
agents:          # Agent definitions
  # ...
guardrails:      # Safety policies
  # ...
vector_stores:   # Vector database backends
  # ...
embedders:       # Embedding model providers
  # ...
document_stores: # RAG document sources
  # ...
defaults:        # Default values (e.g., default LLM)
  # ...
```

All sections are optional. An empty file is valid — Hector applies sensible defaults.

### Path Resolution

Relative paths (e.g., `instruction_file: ./prompts/agent.md`) resolve relative to the config file's directory.

---

## Environment Variables

Use `${VAR_NAME}` syntax for secrets and environment-specific values:

```yaml
llms:
  claude:
    provider: anthropic
    api_key: ${ANTHROPIC_API_KEY}
```

Hector automatically loads `.env` files from the workspace root (the parent of `.hector/`).

---

## Auto-Generated Config

When you run `hector serve` without an existing config file, Hector auto-generates a minimal `.hector/config.yaml`. The generated config:

- Detects your LLM provider from environment variables (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, etc.)
- Creates a single `default` LLM and `assistant` agent
- Detects `SKILL.md` in the workspace root and uses it as the agent instruction
- Derives the app name from the directory name

For more control over the generated config, use `hector init`:

```bash
# Generate with specific provider and instruction
hector init --provider anthropic --model claude-sonnet-4 \
  --instruction "You are a coding assistant" \
  --tools bash,text_editor
```

---

## Common Patterns

### Pattern 1: Simple Chatbot

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
      type: blob
      blob:
        url: "file://./knowledge"
    embedder: default
    vector_store: default
    watch: true
    incremental_indexing: true

agents:
  support:
    llm: default
    include_context: true
    include_context_limit: 5
    instruction: |
      You are a support agent. Answer questions using the knowledge base.
      Always cite your sources. If you can't find an answer, say so.
```

### Pattern 5: Production with Security

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
  --database "postgres://user:pass@db:5432/hector" \
  --auth-secret "$HECTOR_AUTH_SECRET" \
  --metrics \
  --tracing-endpoint "jaeger:4317"
```

### Pattern 6: Webhook-Triggered Agent

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
# Validate syntax and cross-references
hector validate

# Validate a specific file
hector validate -c production.yaml

# JSON Schema available at runtime
curl http://localhost:8080/schema
```

## Hot Reload

Use `--watch` for development. Config changes apply without restart:

```bash
hector serve --watch
```

Changes made via the Admin API (e.g., from Studio) take effect immediately without restart.

### Config File vs Database

On first startup, the config file seeds the default app into the database. After that, the database owns the config — changes made in Studio or via the Admin API survive restarts.

To force the config file to overwrite the database:

```bash
hector serve --sync
```

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
