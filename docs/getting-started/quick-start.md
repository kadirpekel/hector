# Quick Start

Get an agent running in under 5 minutes.

## Prerequisites

- Hector installed (see below or [Installation](installation.md))
- An LLM API key (Anthropic, OpenAI, or Gemini)

## Install Hector

=== "Install Script"

    ```bash
    curl -fsSL https://gohector.dev/install.sh | sh
    ```

=== "Download Binary"

    1. Go to the [latest release](https://github.com/verikod/hector/releases/latest)
    2. Download the archive for your OS and architecture
    3. Extract it and move the `hector` binary somewhere in your `PATH`

    ```bash
    # Example for macOS ARM64
    tar xzf hector_*_darwin_arm64.tar.gz
    sudo mv hector /usr/local/bin/
    ```

=== "Homebrew"

    ```bash
    brew install verikod/tap/hector
    ```

=== "Docker"

    ```bash
    docker run -p 8080:8080 ghcr.io/verikod/hector:latest serve
    ```

    If using Docker, skip to [Open Studio](#2-open-studio) — the server is already running.

## 1. Start the Server

```bash
hector serve
```

The server starts at `http://localhost:8080`. An admin secret is auto-generated and printed in the terminal — use it to log in to Studio.

To set your own secret, use `--auth-secret` or the `HECTOR_AUTH_SECRET` environment variable:

```bash
hector serve --auth-secret my-secret
```

## 2. Open Studio

Navigate to **http://localhost:8080/** and enter the admin secret shown in your terminal. From Studio you can:

- **Add LLM providers** — configure Anthropic, OpenAI, Gemini, or Ollama with your API keys
- **Create agents** — define agent instructions, tools, and models visually
- **Chat** — test agents in real-time with streaming and tool-call traces

Studio is embedded in Hector and available at **http://localhost:8080/**.

## 3. Test the Agent

Once you've configured an LLM and created an agent in Studio, test it via the API:

```bash
curl -X POST http://localhost:8080/agents/assistant \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "message": {
        "role": "user",
        "parts": [{"text": "Hello, who are you?"}]
      }
    },
    "id": 1
  }'
```

## Next Steps

### Add Tools

Enable built-in tools (like web search) by editing `.hector/config.yaml`:

```yaml
tools:
  internet:
    type: function
    handler: web_search

agents:
  assistant:
    llm: default
    tools: [internet]
```

Restart with `hector serve` or use `--watch` for hot-reload.

### Enable RAG

Turn a folder into a knowledge base:

```yaml
document_stores:
  my_knowledge:
    source:
      type: directory
      include: ["./documents/**/*"]
    vector_store: default
    embedder: default

vector_stores:
  default:
    type: chromem

embedders:
  default:
    provider: openai
    model: text-embedding-3-small
    api_key: ${OPENAI_API_KEY}
```

### Use Hector Studio

Hector Studio is the web UI for designing and testing agents. It's embedded in Hector at **http://localhost:8080/**.

With Studio you can:

- **Design** agent flows visually with drag-and-drop
- **Chat** with agents in real-time with streaming and trace view
- **Configure** LLMs, tools, and guardrails without editing YAML

If you built Hector from source (not a release binary), see the [Studio Guide](../guides/studio.md) for setup options.

---

## Reference

- [CLI Reference](../reference/cli.md)
- [Configuration Reference](../reference/configuration.md)
