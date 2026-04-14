# Quick Start

Get an agent running in under 5 minutes.

## Prerequisites

- Hector installed ([Installation](installation.md))
- An LLM API key (Anthropic, OpenAI, or Gemini)

## 1. Set Your API Key

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
# or
export OPENAI_API_KEY="sk-..."
# or
export GEMINI_API_KEY="..."
```

## 2. Initialize Configuration

```bash
hector init
```

This creates `.hector/config.yaml` with a minimal configuration.

## 3. Start the Server

```bash
hector serve
```

The server starts at `http://localhost:8080`.

!!! tip "Open the Web UI"
    Navigate to **http://localhost:8080/** for the visual Studio interface where you can design flows, manage resources, and chat with agents. You can also **[try Studio online](https://studio.gohector.dev)** without installing anything. Just point it at your running server.

## 4. Test the Agent

Send a message using JSON-RPC 2.0:

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

Hector Studio is the web UI for designing and testing agents. **[Try it online](https://studio.gohector.dev)** or use the version embedded in Hector at **http://localhost:8080/**.

With Studio you can:

- **Design** agent flows visually with drag-and-drop
- **Chat** with agents in real-time with streaming and trace view
- **Configure** LLMs, tools, and guardrails without editing YAML

If you built Hector from source (not a release binary), see the [Studio Guide](../guides/studio.md) for setup options.

---

## Reference

- [CLI Reference](../reference/cli.md)
- [Configuration Reference](../reference/configuration.md)
