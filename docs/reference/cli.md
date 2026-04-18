# CLI Reference

Complete reference for Hector CLI commands and flags. Every flag documented with type, default, environment variable, and description.

---

## Commands

| Command | Description |
|---------|-------------|
| [`hector serve`](#hector-serve) | Start the Hector server |
| [`hector init`](#hector-init) | Create a new app config file |
| [`hector validate`](#hector-validate) | Validate an app config file |
| [`hector version`](#hector-version) | Show version information |

---

## hector serve

Start the Hector server. If no config file exists at the specified path, a minimal one is auto-generated.

```bash
hector serve [flags]
```

### All Flags

#### Configuration

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-c, --config` | string | `.hector/config.yaml` | Path to app config file |
| `--watch` | boolean | `false` | Watch config file for changes (hot-reload) |
| `--sync` | boolean | `false` | Force sync config file to database on startup (overwrites DB changes) |

#### Server

| Flag | Type | Default | Env Variable | Description |
|------|------|---------|--------------|-------------|
| `--database` | string | `sqlite://.hector/hector.db` | `HECTOR_DATABASE` | Database DSN |
| `--host` | string | `0.0.0.0` | `HECTOR_HOST` | Host to bind to |
| `--port` | integer | `8080` | `HECTOR_PORT` | Port to listen on |

**Database DSN formats:**

- SQLite: `sqlite://.hector/hector.db` or `sqlite:///absolute/path/to/db`
- PostgreSQL: `postgres://user:password@host:port/database`
- MySQL: `mysql://user:password@tcp(host:port)/database`

#### Logging

| Flag | Type | Default | Env Variable | Description |
|------|------|---------|--------------|-------------|
| `--log-level` | string | `info` | `HECTOR_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` |
| `--log-format` | string | `text` | `HECTOR_LOG_FORMAT` | Log format: `json`, `text` |
| `--log-file` | string | _(stderr)_ | `HECTOR_LOG_FILE` | Log file path |

#### Authentication

| Flag | Type | Default | Env Variable | Description |
|------|------|---------|--------------|-------------|
| `--auth-secret` | string | _(auto-generated)_ | `HECTOR_AUTH_SECRET` | Admin API secret. Auto-generated and persisted to `.hector/secret` if not set. |
| `--auth-jwks-url` | string | - | `HECTOR_AUTH_JWKS_URL` | JWKS URL for JWT validation |
| `--auth-issuer` | string | - | `HECTOR_AUTH_ISSUER` | Expected JWT issuer |
| `--auth-audience` | string | - | `HECTOR_AUTH_AUDIENCE` | Expected JWT audience |
| `--auth-client-id` | string | - | `HECTOR_AUTH_CLIENT_ID` | Public client ID for OIDC flows |
| `--auth-signing-seed` | string | - | `HECTOR_AUTH_SIGNING_SEED` | Deterministic seed for signing key (Ed25519). When set, the same key is generated on every restart. |

**Auth behavior:**

- If neither `--auth-secret` nor `--auth-jwks-url` is set, Hector auto-generates a secret, persists it to `.hector/secret`, and prints it on startup.
- The auto-generated secret survives restarts (read from `.hector/secret`).
- JWT auth requires all three: `--auth-jwks-url`, `--auth-issuer`, `--auth-audience`.

#### Queue

| Flag | Type | Default | Env Variable | Description |
|------|------|---------|--------------|-------------|
| `--queue-workers` | integer | `4` | `HECTOR_QUEUE_WORKERS` | Number of concurrent background workers |
| `--queue-max-retries` | integer | `3` | `HECTOR_QUEUE_MAX_RETRIES` | Max retries for failed tasks |
| `--queue-initial-delay` | duration | `1s` | `HECTOR_QUEUE_INITIAL_DELAY` | Initial retry delay (exponential backoff) |
| `--queue-max-delay` | duration | `5m` | `HECTOR_QUEUE_MAX_DELAY` | Maximum retry delay cap |
| `--queue-stale-threshold` | duration | `5m` | `HECTOR_QUEUE_STALE_THRESHOLD` | Stale task recovery threshold |

#### Observability

| Flag | Type | Default | Env Variable | Description |
|------|------|---------|--------------|-------------|
| `--metrics` | boolean | `false` | - | Enable Prometheus metrics at `/metrics` |
| `--tracing-endpoint` | string | - | `HECTOR_TRACING_ENDPOINT` | OTLP endpoint for distributed tracing |

#### Rate Limiting

| Flag | Type | Default | Env Variable | Description |
|------|------|---------|--------------|-------------|
| `--rate-limit-enabled` | boolean | `false` | `HECTOR_RATE_LIMIT_ENABLED` | Enable rate limiting |
| `--rate-limit-scope` | string | `session` | `HECTOR_RATE_LIMIT_SCOPE` | Rate limit scope: `session`, `user` |
| `--rate-limit-limits` | string | - | `HECTOR_RATE_LIMIT_LIMITS` | Rate limits as JSON array (see below) |
| `--rate-limit-ip-headers` | string | - | `HECTOR_RATE_LIMIT_IP_HEADERS` | Comma-separated IP headers for client identification |

**Rate limit JSON format:**

```bash
hector serve --rate-limit-enabled \
  --rate-limit-limits '[{"type":"token","window":"day","limit":100000},{"type":"count","window":"minute","limit":60}]'
```

### Examples

```bash
# Basic — auto-creates config, auto-generates admin secret
hector serve

# Custom config file
hector serve -c my-config.yaml

# Development mode with hot-reload and debug logging
hector serve --watch --log-level debug

# Custom port
hector serve --port 3000

# PostgreSQL database
hector serve --database "postgres://user:pass@localhost:5432/hector"

# Set your own admin secret
hector serve --auth-secret "my-super-secret-token"

# JWT/OIDC authentication
hector serve \
  --auth-jwks-url "https://auth.example.com/.well-known/jwks.json" \
  --auth-issuer "https://auth.example.com/" \
  --auth-audience "hector-api"

# Production with observability
hector serve \
  --log-format json \
  --log-file /var/log/hector.log \
  --metrics \
  --tracing-endpoint "jaeger:4317"

# Rate limiting
hector serve \
  --rate-limit-enabled \
  --rate-limit-scope session \
  --rate-limit-limits '[{"type":"count","window":"minute","limit":60}]'
```

---

## hector init

Create a new app configuration file. Fails if the file already exists.

```bash
hector init [flags]
```

### All Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-o, --output` | string | `.hector/config.yaml` | Output file path |
| `--provider` | string | _(auto-detected)_ | LLM provider: `anthropic`, `openai`, `gemini`, `ollama` |
| `--model` | string | _(provider default)_ | Model name |
| `--api-key` | string | - | API key for the LLM provider |
| `--base-url` | string | - | Base URL for the LLM provider |
| `-i, --instruction` | string | - | System instruction for the agent |
| `--tools` | string | - | Comma-separated tools to enable (e.g., `bash,text_editor`) or `all` |
| `--mcp-url` | string | - | MCP server URL |

**Behavior:**

- If `--name` is not set, derives the name from the workspace directory (e.g., `my-project`).
- If `--provider` is not set, detects the provider from environment variables (`ANTHROPIC_API_KEY` → `anthropic`, etc.).
- If a `SKILL.md` file exists in the workspace root, it is automatically used as the agent's instruction file.
- A `.env.example` file is generated alongside the config if relevant env vars are detected.

### Examples

```bash
# Create minimal config (auto-detects provider from env)
hector init

# Specify provider and model
hector init --provider anthropic --model claude-sonnet-4

# Full quick-start with instruction and tools
hector init \
  --provider openai \
  --model gpt-4o \
  --instruction "You are a coding assistant" \
  --tools bash,text_editor

# Create config with MCP server
hector init --mcp-url "http://localhost:3001/sse"

# Custom output path
hector init -o configs/production.yaml

# Ollama (local, no API key needed)
hector init --provider ollama --model llama3.2
```

---

## hector validate

Validate an app configuration file. Checks YAML syntax, field types, and cross-references (e.g., agents referencing valid LLMs and tools).

```bash
hector validate [flags]
```

### All Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-c, --config` | string | `.hector/config.yaml` | Path to config file |

### Exit Codes

| Code | Description |
|------|-------------|
| `0` | Configuration is valid |
| `1` | Configuration has errors |

### Examples

```bash
# Validate default config
hector validate

# Validate specific file
hector validate -c production.yaml

# CI/CD usage
hector validate -c config.yaml && echo "Valid" || echo "Invalid"
```

---

## hector version

Show version information.

```bash
hector version
```

### Example Output

```
Hector v1.20.0 (cf6a7bc) built 2026-01-18T22:29:05Z
```

---

## Environment Variables

### LLM API Keys

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Anthropic API key for Claude models |
| `OPENAI_API_KEY` | OpenAI API key |
| `GEMINI_API_KEY` | Google Gemini API key |
| `GOOGLE_API_KEY` | Alternative Google API key |
| `DEEPSEEK_API_KEY` | DeepSeek API key |
| `GROQ_API_KEY` | Groq API key |
| `MISTRAL_API_KEY` | Mistral API key |
| `COHERE_API_KEY` | Cohere API key |

### Server

| Variable | Default | Description |
|----------|---------|-------------|
| `HECTOR_DATABASE` | `sqlite://.hector/hector.db` | Database DSN |
| `HECTOR_HOST` | `0.0.0.0` | Server host |
| `HECTOR_PORT` | `8080` | Server port |

### Logging

| Variable | Default | Description |
|----------|---------|-------------|
| `HECTOR_LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `HECTOR_LOG_FORMAT` | `text` | Log format (`json`, `text`) |
| `HECTOR_LOG_FILE` | - | Log file path |

### Authentication

| Variable | Description |
|----------|-------------|
| `HECTOR_AUTH_SECRET` | Admin API secret |
| `HECTOR_AUTH_JWKS_URL` | JWKS URL for JWT validation |
| `HECTOR_AUTH_ISSUER` | Expected JWT issuer |
| `HECTOR_AUTH_AUDIENCE` | Expected JWT audience |
| `HECTOR_AUTH_CLIENT_ID` | Public client ID for OIDC |
| `HECTOR_AUTH_SIGNING_SEED` | Deterministic seed for signing key |

### Queue

| Variable | Default | Description |
|----------|---------|-------------|
| `HECTOR_QUEUE_WORKERS` | `4` | Number of background workers |
| `HECTOR_QUEUE_MAX_RETRIES` | `3` | Max retries for failed tasks |
| `HECTOR_QUEUE_INITIAL_DELAY` | `1s` | Initial retry delay |
| `HECTOR_QUEUE_MAX_DELAY` | `5m` | Maximum retry delay |
| `HECTOR_QUEUE_STALE_THRESHOLD` | `5m` | Stale task threshold |

### Observability

| Variable | Description |
|----------|-------------|
| `HECTOR_TRACING_ENDPOINT` | OTLP tracing endpoint |

### Rate Limiting

| Variable | Default | Description |
|----------|---------|-------------|
| `HECTOR_RATE_LIMIT_ENABLED` | `false` | Enable rate limiting (`true`/`false`) |
| `HECTOR_RATE_LIMIT_SCOPE` | `session` | Rate limit scope (`session`, `user`) |
| `HECTOR_RATE_LIMIT_LIMITS` | - | Rate limits as JSON array |
| `HECTOR_RATE_LIMIT_IP_HEADERS` | - | Comma-separated IP headers |

### Tools

| Variable | Description |
|----------|-------------|
| `MCP_URL` | Default MCP server URL (auto-detected during config generation) |
| `TAVILY_API_KEY` | API key for the `web_search` built-in tool |

---

## Configuration Precedence

For server settings, later sources override earlier ones:

1. Built-in defaults
2. Environment variables
3. CLI flags

```bash
# Environment variable
export HECTOR_PORT=3000

# CLI flag overrides environment variable
hector serve --port 8080  # Server runs on port 8080
```

For app configuration, there is no merging — the YAML file is the single source of truth. Multi-app configurations are managed via the Admin API.
