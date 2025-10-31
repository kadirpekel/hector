---
title: Quick Observability Setup
description: Enable metrics and tracing with a single flag
---

# Quick Observability Setup

Enable monitoring and tracing in development with a single flag - no config file needed!

---

## The --observe Flag

The `--observe` flag automatically configures:
- ✅ **Metrics** - Prometheus metrics at `/metrics` endpoint
- ✅ **Tracing** - OpenTelemetry traces to `localhost:4317`
- ✅ **Full instrumentation** - Agent calls, LLM requests, tool execution

### Quick Start

```bash
# 1. Start observability stack (Prometheus, Jaeger, Grafana)
docker-compose -f docker-compose.observability.yaml up -d

# 2. Start Hector with observability enabled
hector serve --observe

# 3. Make requests
curl -X POST http://localhost:8080/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{"message":{"role":"user","parts":[{"text":"Hello"}]}}'

# 4. View dashboards
# Metrics:  http://localhost:9090
# Traces:   http://localhost:16686
# Grafana:  http://localhost:3000 (admin/Dev12345)
```

---

## What Gets Configured

When you use `--observe`, Hector automatically enables:

### Global Observability
```yaml
global:
  observability:
    metrics:
      enabled: true
    tracing:
      enabled: true
      exporter_type: "otlp"
      endpoint_url: "localhost:4317"
      sampling_rate: 1.0
      service_name: "hector"
```

### Metrics Available

- `hector_agent_calls_total` - Total agent calls
- `hector_agent_call_duration_seconds` - Agent call latency
- `hector_agent_errors_total` - Agent errors
- `hector_agent_tokens_used_total` - Token usage
- `hector_llm_*` - LLM-specific metrics
- `hector_tool_*` - Tool execution metrics
- `hector_http_*` - HTTP request metrics
- `hector_grpc_*` - gRPC metrics

### Traces Collected

- 🔍 **Agent calls** - Full request/response lifecycle
- 🤖 **LLM requests** - Model interactions with token counts
- 🔧 **Tool executions** - Tool invocations with timing
- 📊 **HTTP/gRPC** - Transport layer spans

---

## Combine with Other Flags

The `--observe` flag works seamlessly with other zero-config flags:

```bash
# With tools enabled
hector serve --observe --tools

# With custom model
hector serve --observe --model gpt-4o

# With RAG/document store
hector serve --observe --tools --docs-folder ./knowledge

# All together
hector serve --observe --tools --docs-folder ./docs --model claude-sonnet-4
```

---

## Access Points

Once running with `--observe`:

| Service | URL | Credentials |
|---------|-----|-------------|
| **Metrics** | http://localhost:8080/metrics | - |
| **Prometheus** | http://localhost:9090 | - |
| **Jaeger** | http://localhost:16686 | - |
| **Grafana** | http://localhost:3000 | admin/Dev12345 |

---

## Grafana Dashboards

Pre-built dashboards automatically load:

1. **Hector Overview** - Service health, request rates, errors
2. **HTTP/gRPC** - Transport layer metrics
3. **LLM & Tools** - Model performance, tool usage
4. **Business Metrics** - Sessions, conversations, tokens

### Using Dashboards

1. Open Grafana: http://localhost:3000
2. Login: `admin` / `Dev12345`
3. Navigate to **Dashboards** → Browse
4. Select any **Hector** dashboard

---

## Custom Endpoint

Need to send traces somewhere else? Use a config file:

```yaml
global:
  observability:
    metrics:
      enabled: true
    tracing:
      enabled: true
      exporter_type: "otlp"
      endpoint_url: "my-collector.example.com:4317"
      sampling_rate: 0.1  # Sample 10% in production

llms:
  openai:
    type: openai
    api_key: "${OPENAI_API_KEY}"

agents:
  assistant:
    name: "Assistant"
    llm: openai
```

Then run:
```bash
hector serve --config my-config.yaml
```

---

## Troubleshooting

### No metrics showing up

**Check** metrics are being generated:
```bash
curl http://localhost:8080/metrics | grep hector_
```

If empty, metrics haven't been generated yet. Make some requests first.

### Prometheus not scraping

**Check** Prometheus targets:
```bash
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets'
```

Look for `job="hector"` with `health="up"`.

### Jaeger not receiving traces

**Verify** OTLP endpoint is reachable:
```bash
# From host machine
nc -zv localhost 4317
```

**Check** Jaeger is running:
```bash
docker ps | grep jaeger
```

### Grafana dashboards empty

1. **Check** Prometheus datasource is configured:
   - Go to **Configuration** → **Data Sources**
   - Prometheus should be listed and marked as default

2. **Verify** time range in top-right (Last 15 minutes → Last 1 hour)

3. **Generate some traffic** to create data points

---

## Production Use

For production, use a config file with:
- Lower sampling rates (e.g., 0.1 for 10%)
- External collectors (Datadog, Honeycomb, etc.)
- Secure endpoints
- Authentication

See: [Observability Guide](../core-concepts/observability.md) | [Deploy to Production](./deploy-production.md)

---

## Next Steps

- **[Observability Concepts](../core-concepts/observability.md)** - Deep dive into metrics & tracing
- **[Configuration Reference](../reference/configuration.md#globalobservability)** - All observability options
- **[Production Deployment](./deploy-production.md)** - Deploy with monitoring

