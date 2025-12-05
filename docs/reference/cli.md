---
title: CLI Reference
description: Hector CLI commands and flags
---

# CLI Reference

Hector ships a smaller CLI surface focused on running and inspecting agents. Legacy client/task commands are removed.

## Commands (current)
- `hector version` — show build version.
- `hector serve` — start the A2A server (zero-config or config file).
- `hector info` — list agents or show one (requires config).
- `hector validate` — validate a config file.
- `hector schema` — emit JSON Schema for the config builder UI.

## Global flags
| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--config PATH` | string | Config file path (required for config mode; auto-created if missing and not zero-config) | auto path |
| `--log-level` | string | `debug`, `info`, `warn`, `error` | `info` |
| `--log-file` | string | Log file path (empty = stderr) | empty |
| `--log-format` | string | `simple`, `verbose`, or custom | `simple` |

## `hector serve`
Start the A2A server with zero-config flags or a config file.

**Zero-config flags (no `--config`):**
- LLM: `--provider` (`anthropic|openai|gemini|ollama`), `--model`, `--api-key`, `--base-url`, `--temperature`, `--max-tokens`
- Behavior: `--instruction`, `--role`, `--tools` (comma list or `all`), `--approve-tools`, `--no-approve-tools`, `--thinking`, `--thinking-budget`, `--[no-]stream`
- RAG: `--docs-folder`, `--embedder-model`, `--rag-watch`, `--mcp-url`, `--mcp-parser-tool`
- Storage/obs: `--storage` (`inmemory|sqlite|postgres|mysql`), `--storage-db`, `--observe`
- Server: `--port` (default 8080), `--studio` (enables builder UI + watch), `--watch` (config reload when loader present)

**Config mode:**
- Provide `--config path` (auto-created if missing and not zero-config).
- `--port` can override `server.port` in config.

**Examples:**
- Zero-config RAG: `hector serve --model gpt-4o --docs-folder ./docs --tools`
- Zero-config + MCP parsing: `hector serve --docs-folder ./docs --mcp-url http://localhost:8000/mcp --mcp-parser-tool convert_document_into_docling_document`
- Config mode: `hector serve --config config.yaml --port 9000`
- Studio mode: `hector serve --studio --model gpt-4o`

## `hector info`
- Lists agents when no name is provided.
- `hector info assistant --config config.yaml`

## `hector validate`
- Validate a config file: `hector validate config.yaml`
- Formats: `--format=compact|verbose|json`
- Optional: `--print-config` to see the expanded config.

## `hector schema`
- Emit JSON Schema for the config builder UI: `hector schema > config.schema.json`
- `--compact` to disable indentation.

## Environment variables
- `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY` — provider keys
- `HECTOR_CONFIG` — default config path
- `LOG_LEVEL`, `LOG_FILE`, `LOG_FORMAT` — logging overrides

## Notes
- No client/task commands in v2; the CLI focuses on running the local server and inspecting configs/agents.
- Zero-config is preferred for quick starts; config files for repeatable setups and persistence.
