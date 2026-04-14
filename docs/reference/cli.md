# CLI Reference

Complete, authoritative reference for Hector CLI commands and flags. Every flag documented with type, default, environment variable, and description.

**Version:** Hector v1.20.0

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

Start the Hector server.

```bash
hector serve [flags]
```

### All Flags

| Flag | Type | Default | Env Variable | Description |
|------|------|---------|--------------|-------------|
| `-h, --help` | - | - | - | Show context-sensitive help |
| `-c, --config` | string | `.hector/config.yaml` | - | Path to app config file |
| `--watch` | boolean | `false` | - | Watch config file for changes |
| `--database` | string | `sqlite://.hector/hector.db` | `HECTOR_DATABASE` | Database DSN |
| `--host` | string | `0.0.0.0` | `HECTOR_HOST` | Host to bind to |
| `--port` | integer | `8080` | `HECTOR_PORT` | Port to listen on |
| `--log-level` | string | `info` | `HECTOR_LOG_LEVEL` | Log level |
| `--log-format` | string | `text` | `HECTOR_LOG_FORMAT` | Log format (json, text) |
| `--log-file` | string | - | `HECTOR_LOG_FILE` | Log file path |
| `--auth-secret` | string | - | `HECTOR_AUTH_SECRET` | Admin API secret |
| `--auth-jwks-url` | string | - | `HECTOR_AUTH_JWKS_URL` | JWKS URL for JWT auth |
| `--auth-issuer` | string | - | `HECTOR_AUTH_ISSUER` | JWT issuer |
| `--auth-audience` | string | - | `HECTOR_AUTH_AUDIENCE` | JWT audience |
| `--auth-client-id` | string | - | `HECTOR_AUTH_CLIENT_ID` | Public client ID |
| `--queue-workers` | integer | `4` | `HECTOR_QUEUE_WORKERS` | Number of queue workers |
| `--queue-max-retries` | integer | `3` | `HECTOR_QUEUE_MAX_RETRIES` | Max retries for failed tasks |
| `--queue-initial-delay` | duration | `1s` | `HECTOR_QUEUE_INITIAL_DELAY` | Initial retry delay |
| `--queue-max-delay` | duration | `5m` | `HECTOR_QUEUE_MAX_DELAY` | Max retry delay |
| `--queue-stale-threshold` | duration | `5m` | `HECTOR_QUEUE_STALE_THRESHOLD` | Stale task recovery threshold |
| `--metrics` | boolean | `false` | - | Enable Prometheus metrics |
| `--tracing-endpoint` | string | `localhost:4317` | `HECTOR_TRACING_ENDPOINT` | OTLP endpoint |

### Flag Categories

#### Configuration

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-c, --config` | string | `.hector/config.yaml` | Path to app config file |
| `--watch` | boolean | `false` | Watch config file for changes (hot-reload) |

#### Server

| Flag | Type | Default | Env Variable | Description |
|------|------|---------|--------------|-------------|
| `--database` | string | `sqlite://.hector/hector.db` | `HECTOR_DATABASE` | Database DSN. Supports SQLite and PostgreSQL. |
| `--host` | string | `0.0.0.0` | `HECTOR_HOST` | Host to bind to |
| `--port` | integer | `8080` | `HECTOR_PORT` | Port to listen on |

**Database DSN Formats:**

- SQLite: `sqlite://.hector/hector.db` or `sqlite:///absolute/path/to/db`
- PostgreSQL: `postgres://user:password@host:port/database`

#### Logging

| Flag | Type | Default | Env Variable | Description |
|------|------|---------|--------------|-------------|
| `--log-level` | string | `info` | `HECTOR_LOG_LEVEL` | Log level. Values: `debug`, `info`, `warn`, `error` |
| `--log-format` | string | `text` | `HECTOR_LOG_FORMAT` | Log format. Values: `json`, `text` |
| `--log-file` | string | - | `HECTOR_LOG_FILE` | Log file path. If empty, logs to stdout |

#### Authentication

| Flag | Type | Default | Env Variable | Description |
|------|------|---------|--------------|-------------|
| `--auth-secret` | string | - | `HECTOR_AUTH_SECRET` | Admin API secret. Used for simple token-based auth. |
| `--auth-jwks-url` | string | - | `HECTOR_AUTH_JWKS_URL` | JWKS URL for JWT auth. For OIDC integration. |
| `--auth-issuer` | string | - | `HECTOR_AUTH_ISSUER` | JWT issuer. Must match token's `iss` claim. |
| `--auth-audience` | string | - | `HECTOR_AUTH_AUDIENCE` | JWT audience. Must match token's `aud` claim. |
| `--auth-client-id` | string | - | `HECTOR_AUTH_CLIENT_ID` | Public client ID for OIDC flows. |

#### Queue

| Flag | Type | Default | Env Variable | Description |
|------|------|---------|--------------|-------------|
| `--queue-workers` | integer | `4` | `HECTOR_QUEUE_WORKERS` | Number of queue workers for background tasks |
| `--queue-max-retries` | integer | `3` | `HECTOR_QUEUE_MAX_RETRIES` | Max retries for failed tasks |
| `--queue-initial-delay` | duration | `1s` | `HECTOR_QUEUE_INITIAL_DELAY` | Initial retry delay (exponential backoff) |
| `--queue-max-delay` | duration | `5m` | `HECTOR_QUEUE_MAX_DELAY` | Max retry delay cap |
| `--queue-stale-threshold` | duration | `5m` | `HECTOR_QUEUE_STALE_THRESHOLD` | Stale task recovery threshold |

#### Observability

| Flag | Type | Default | Env Variable | Description |
|------|------|---------|--------------|-------------|
| `--metrics` | boolean | `false` | - | Enable Prometheus metrics at `/metrics` |
| `--tracing-endpoint` | string | `localhost:4317` | `HECTOR_TRACING_ENDPOINT` | OTLP endpoint for distributed tracing |

### Examples

```bash
# Basic usage with defaults
hector serve

# Custom config file
hector serve --config my-config.yaml

# Custom port
hector serve --port 3000

# Development mode with hot-reload and debug logging
hector serve --watch --log-level debug

# PostgreSQL database
hector serve --database "postgres://user:pass@localhost:5432/hector"

# Simple token authentication
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

# Custom queue configuration
hector serve \
  --queue-workers 8 \
  --queue-max-retries 5 \
  --queue-initial-delay 2s \
  --queue-max-delay 10m

# Full production example
hector serve \
  --config /etc/hector/config.yaml \
  --database "postgres://hector:password@db:5432/hector" \
  --host 0.0.0.0 \
  --port 8080 \
  --log-level info \
  --log-format json \
  --auth-jwks-url "https://auth.example.com/.well-known/jwks.json" \
  --auth-issuer "https://auth.example.com/" \
  --auth-audience "hector-api" \
  --metrics \
  --tracing-endpoint "jaeger:4317"
```

---

## hector init

Create a new app configuration file.

```bash
hector init [flags]
```

### All Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-h, --help` | - | - | Show context-sensitive help |
| `-o, --output` | string | `.hector/config.yaml` | Output file path |
| `--name` | string | `my-app` | App name |
| `--template` | string | `minimal` | Template to use. Values: `minimal`, `full` |
| `--provider` | string | - | LLM provider. Values: `anthropic`, `openai`, `gemini`, `ollama` |
| `--model` | string | - | Model name |

### Templates

| Template | Description |
|----------|-------------|
| `minimal` | Basic configuration with a single LLM and agent |
| `full` | Comprehensive configuration with examples of all features |

### Examples

```bash
# Create minimal config in default location
hector init

# Create config with specific provider and model
hector init --provider anthropic --model claude-sonnet-4

# Create config for OpenAI
hector init --provider openai --model gpt-4o

# Create config for Gemini
hector init --provider gemini --model gemini-2.0-flash

# Create config for Ollama
hector init --provider ollama --model llama3.2

# Create full template
hector init --template full

# Custom output path
hector init --output configs/production.yaml

# Custom app name
hector init --name my-ai-assistant

# Combined options
hector init \
  --output .hector/config.yaml \
  --name production-app \
  --template full \
  --provider anthropic \
  --model claude-sonnet-4
```

---

## hector validate

Validate an app configuration file.

```bash
hector validate [flags]
```

### All Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-h, --help` | - | - | Show context-sensitive help |
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
hector validate --config production.yaml

# Validate and exit with status code (for CI/CD)
hector validate --config config.yaml && echo "Valid" || echo "Invalid"
```

---

## hector version

Show version information.

```bash
hector version
```

### Output Format

```
Hector v<version>-<commits>-g<hash> (<short-hash>) built <timestamp>
```

### Example Output

```
Hector v1.20.0-26-gcf6a7bc (cf6a7bc) built 2026-01-18T22:29:05Z
```

### Fields

| Field | Description |
|-------|-------------|
| Version | Semantic version (e.g., `v1.20.0`) |
| Commits | Commits since last tag |
| Hash | Full commit hash |
| Short Hash | Short commit hash |
| Timestamp | Build timestamp (ISO 8601) |

---

## Environment Variables

Complete reference of all environment variables.

### LLM API Keys

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Anthropic API key for Claude models |
| `OPENAI_API_KEY` | OpenAI API key |
| `GEMINI_API_KEY` | Google Gemini API key |
| `GOOGLE_API_KEY` | Alternative for Gemini API key |

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HECTOR_DATABASE` | `sqlite://.hector/hector.db` | Database connection string |
| `HECTOR_HOST` | `0.0.0.0` | Server host |
| `HECTOR_PORT` | `8080` | Server port |

### Logging

| Variable | Default | Description |
|----------|---------|-------------|
| `HECTOR_LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `HECTOR_LOG_FORMAT` | `text` | Log format (`json`, `text`) |
| `HECTOR_LOG_FILE` | - | Log file path |

### Authentication

| Variable | Default | Description |
|----------|---------|-------------|
| `HECTOR_AUTH_SECRET` | - | Admin API secret for token-based auth |
| `HECTOR_AUTH_JWKS_URL` | - | JWKS URL for JWT validation |
| `HECTOR_AUTH_ISSUER` | - | Expected JWT issuer |
| `HECTOR_AUTH_AUDIENCE` | - | Expected JWT audience |
| `HECTOR_AUTH_CLIENT_ID` | - | Public client ID for OIDC |

### Queue

| Variable | Default | Description |
|----------|---------|-------------|
| `HECTOR_QUEUE_WORKERS` | `4` | Number of background workers |
| `HECTOR_QUEUE_MAX_RETRIES` | `3` | Max retries for failed tasks |
| `HECTOR_QUEUE_INITIAL_DELAY` | `1s` | Initial retry delay |
| `HECTOR_QUEUE_MAX_DELAY` | `5m` | Maximum retry delay |
| `HECTOR_QUEUE_STALE_THRESHOLD` | `5m` | Stale task threshold |

### Observability

| Variable | Default | Description |
|----------|---------|-------------|
| `HECTOR_TRACING_ENDPOINT` | `localhost:4317` | OTLP tracing endpoint |

---

## Configuration Precedence

Configuration is loaded in the following order (later overrides earlier):

1. Default values
2. Environment variables
3. CLI flags

**Example:**
```bash
# Environment variable
export HECTOR_PORT=3000

# CLI flag overrides environment variable
hector serve --port 8080  # Server runs on port 8080
```
