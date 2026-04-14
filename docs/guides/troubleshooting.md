# Troubleshooting

Common issues and how to resolve them.

## Startup Issues

### "command not found: hector"

The binary isn't in your `$PATH`.

```bash
# If installed via go install
export PATH="$PATH:$(go env GOPATH)/bin"

# Verify
which hector
```

If you built from source, the binary is at `./hector` (or wherever you ran `go build -o`).

### "failed to load config"

Hector can't find or parse your configuration file. Check:

1. **File location**: Hector looks for `.hector/config.yaml` in the current directory, then `hector.yaml`. Use `--config` to specify explicitly:
   ```bash
   hector serve --config /path/to/config.yaml
   ```

2. **YAML syntax**: Validate your YAML:
   ```bash
   hector validate --config config.yaml
   ```

3. **Missing environment variables**: Unresolved `${VAR_NAME}` references cause errors. Check your `.env` file or environment:
   ```bash
   echo $ANTHROPIC_API_KEY  # Should not be empty
   ```

### Port already in use

```
listen tcp :8080: bind: address already in use
```

Another process is using port 8080. Either stop it or use a different port:

```bash
hector serve --port 9090
# or
lsof -i :8080  # Find the process
```

---

## LLM Issues

### "401 Unauthorized" from LLM provider

Your API key is missing or invalid.

- Verify the key is set: `echo $ANTHROPIC_API_KEY`
- Check for trailing whitespace in `.env` files
- Ensure the key has the right permissions (some providers require billing setup)
- For Ollama, verify the server is running: `curl http://localhost:11434/api/tags`

### Agent responds but seems "dumb"

- Check that the correct model is assigned: verify `agents.<name>.llm` points to the right LLM definition
- Ensure `max_tokens` isn't too low (the response gets truncated)
- For complex tasks, increase `reasoning.max_iterations` (default may be too low)
- Check if guardrails are modifying the input or output unexpectedly. Look for `guardrail` metadata in events

### Streaming not working

- Verify the agent has `streaming: true`
- Use `message/stream` instead of `message/send` in your API calls
- Check if a reverse proxy is buffering responses (disable response buffering in nginx/Cloudflare)

---

## Tool Issues

### MCP tool not connecting

**Stdio transport:**

- Verify the command exists: `which npx` or `which docker`
- Check that the MCP server package installs correctly: `npx -y @modelcontextprotocol/server-filesystem --help`
- Check environment variables: MCP servers often need tokens (e.g., `GITHUB_PERSONAL_ACCESS_TOKEN`)

**SSE/HTTP transport:**

- Verify the MCP server is running: `curl http://localhost:8080/sse`
- Check network connectivity (firewall, Docker networking)
- Verify the transport type matches: `sse` for SSE endpoints, `streamable-http` for newer HTTP endpoints

### "tool not found" errors

- Verify the tool is listed in the agent's `tools` array
- If using MCP with `filter`, check that the tool name matches exactly
- Tool names are case-sensitive
- Check if guardrail tool authorization is blocking the tool

### Command tool "permission denied"

- Check `allowed_commands` includes the command the agent is trying to run
- If `deny_by_default: true`, commands not in `allowed_commands` are blocked
- Verify `working_directory` exists and is readable
- Check if `require_approval: true` is set. The task may be waiting for approval

### Tool execution timeout

Command tools default to 5-minute timeout. Increase it:

```yaml
tools:
  slow_tool:
    type: command
    max_execution_time: "15m"
```

---

## RAG Issues

### Documents not being indexed

- Check `document_stores` configuration points to an existing directory
- Verify `include` glob patterns match your files: `["./docs/**/*.md"]`
- Check the `exclude` patterns aren't filtering out your files
- Look at logs for indexing errors: `hector serve --log-level debug`
- For incremental indexing, delete the checkpoint to force re-index:
  ```bash
  # SQLite
  sqlite3 .hector/hector.db "DELETE FROM checkpoints WHERE type = 'indexing';"
  ```

### Search returns irrelevant results

- **Increase `top_k`**: Return more candidates for better coverage
- **Lower `threshold`**: Default may be too strict, try `0.5`
- **Enable HyDE**: Hypothetical Document Embeddings improve recall for short queries:
  ```yaml
  document_stores:
    docs:
      search:
        enable_hyde: true
  ```
- **Enable multi-query**: Expands the query for broader search:
  ```yaml
  search:
    enable_multi_query: true
  ```
- **Check chunking**: If chunks are too large, relevant passages get diluted. Try smaller `size`:
  ```yaml
  chunking:
    size: 500
    overlap: 100
  ```

### Embedding errors

- Verify your embedder API key is valid
- For Ollama embeddings, ensure the model is pulled: `ollama pull nomic-embed-text`
- Check that the embedding model matches the vector store dimensions

---

## Authentication Issues

### "unauthorized" on all requests

- If using `--auth-secret`, include: `Authorization: Bearer <your-secret>`
- If using JWKS, verify the token hasn't expired
- Check that `--auth-issuer` matches the `iss` claim in your JWT
- Check that `--auth-audience` matches the `aud` claim (if set)

### JWKS key fetch failing

- Verify the JWKS URL is accessible: `curl https://auth.example.com/.well-known/jwks.json`
- Keys are refreshed every 15 minutes by default. A new key may take up to 15 min to be recognized
- Check for TLS certificate issues with self-signed certs

---

## Session and State Issues

### Sessions lost after restart

- Default storage is SQLite at `.hector/hector.db`. Verify the file persists between restarts
- If using Docker, mount the data directory:
  ```bash
  docker run -v $(pwd)/.hector:/app/.hector ghcr.io/verikod/hector:latest
  ```
- For PostgreSQL, verify the connection string and that the database exists

### "session not found" errors

- Sessions are scoped per app in multi-tenant mode. Ensure you're using the right app token
- Check if the session was cleaned up (sessions may be pruned by retention policies)

---

## Guardrail Issues

### Legitimate messages being blocked

- Check logs for which guardrail triggered: look for `intervention_source` in event metadata
- Prompt injection detection uses pattern matching. Custom patterns may be too broad
- Lower the moderation `threshold` (e.g., from `0.8` to `0.9`) for fewer false positives
- Use `action: warn` instead of `action: block` during testing to see what would be blocked without blocking it
- Switch chain mode to `collect_all` to see all triggers at once:
  ```yaml
  guardrails:
    debug:
      input:
        chain_mode: collect_all
  ```

### PII redaction too aggressive

- Disable specific detectors you don't need: `detect_phone: false`
- Phone number detection is US-focused. International numbers may false-positive
- Use `redact_mode: hash` to see patterns without revealing the data

---

## Performance Issues

### Slow response times

1. **Check LLM latency**: The LLM call is usually the bottleneck. Use a faster model or provider.
2. **Reduce context size**: Large conversation histories slow down each call:
   ```yaml
   context:
     strategy: token_window
     budget: 8000
   ```
3. **Limit tool iterations**: Set `reasoning.max_iterations` to prevent runaway loops
4. **Monitor metrics**: Enable Prometheus (`--metrics`) and check `hector_llm_call_duration_seconds`

### High memory usage

- Large RAG indexes consume memory. Use an external vector store (Qdrant, Pinecone) instead of embedded chromem
- For many concurrent sessions, switch from SQLite to PostgreSQL
- Check queue depth: `curl http://localhost:8080/admin/queue/stats`

---

## Debugging Tips

### Enable debug logging

```bash
hector serve --log-level debug
```

### Use JSON log format for parsing

```bash
hector serve --log-format json | jq '.msg'
```

### Inspect events via Admin API

```bash
# List recent sessions
curl http://localhost:8080/admin/sessions \
  -H "Authorization: Bearer $AUTH_SECRET"

# Get session details with events
curl http://localhost:8080/admin/sessions/<session-id> \
  -H "Authorization: Bearer $AUTH_SECRET"
```

### Validate config without starting

```bash
hector validate --config config.yaml
```

### Check task queue health

```bash
curl http://localhost:8080/admin/queue/stats \
  -H "Authorization: Bearer $AUTH_SECRET"

# Check dead-letter queue for failed tasks
curl http://localhost:8080/admin/queue/dlq \
  -H "Authorization: Bearer $AUTH_SECRET"
```

## Still Stuck?

- Check the [GitHub Issues](https://github.com/verikod/hector/issues) for known problems
- Search the [Configuration Reference](../reference/configuration.md) for field-level details
- Review the [Architecture](../reference/architecture.md) to understand system internals
