# Observability

Monitor your Hector deployment with structured logs, Prometheus metrics, and OpenTelemetry tracing.

## Logging

Hector uses structured logging for production observability.

### Configuration

```bash
hector serve \
  --log-level info \
  --log-format json \
  --log-file /var/log/hector.log
```

| Flag | Env Variable | Default | Options |
|------|--------------|---------|---------|
| `--log-level` | `HECTOR_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `--log-format` | `HECTOR_LOG_FORMAT` | `text` | `text`, `json` |
| `--log-file` | `HECTOR_LOG_FILE` | stdout | File path |

### JSON Log Format

Production-ready for log aggregators (Elasticsearch, Loki, Splunk):

```json
{
  "time": "2026-01-20T18:00:00Z",
  "level": "INFO",
  "msg": "Agent invocation completed",
  "agent": "assistant",
  "session_id": "sess_123",
  "duration_ms": 1250,
  "tokens_used": 450
}
```

## Metrics (Prometheus)

Enable the `/metrics` endpoint:

```bash
hector serve --metrics
```

### Available Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `hector_requests_total` | Counter | Total HTTP requests |
| `hector_request_duration_seconds` | Histogram | Request latency |
| `hector_agent_invocations_total` | Counter | Agent executions |
| `hector_llm_tokens_total` | Counter | Token usage (prompt/completion) |
| `hector_tool_calls_total` | Counter | Tool invocations |
| `hector_scheduler_triggers_total` | Counter | Scheduled trigger firings |
| `hector_notifications_total` | Counter | Outbound webhook notifications |
| `hector_guardrail_violations_total` | Counter | Blocked inputs/outputs |

### Labels

Common labels across metrics:

| Label | Description |
|-------|-------------|
| `app` | App/tenant name |
| `agent` | Agent name |
| `status` | `success`, `error` |
| `tool` | Tool name (for tool metrics) |

### Prometheus Scrape Config

```yaml
scrape_configs:
  - job_name: 'hector'
    static_configs:
      - targets: ['hector:8080']
    metrics_path: /metrics
```

## Tracing (OpenTelemetry)

Enable distributed tracing:

```bash
hector serve --tracing-endpoint "jaeger:4317"
```

### Trace Spans

| Span | Description |
|------|-------------|
| `http.request` | Full request lifecycle |
| `agent.run` | Agent execution |
| `llm.generate` | LLM API call |
| `tool.call` | Tool invocation |
| `vectorstore.search` | RAG retrieval |
| `guardrail.check` | Input/output validation |

### Configuration

| Flag | Env Variable | Default |
|------|--------------|---------|
| `--tracing-endpoint` | `HECTOR_TRACING_ENDPOINT` | `localhost:4317` |

### Jaeger Setup

```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4317:4317 \
  jaegertracing/all-in-one:latest

hector serve --tracing-endpoint "localhost:4317"
```

Access UI at `http://localhost:16686`.

## Grafana Dashboard

Import metrics into Grafana using the `hector_*` namespace.

### Key Panels

| Panel | Query |
|-------|-------|
| Request Rate | `rate(hector_requests_total[5m])` |
| Agent Latency P95 | `histogram_quantile(0.95, rate(hector_request_duration_seconds_bucket[5m]))` |
| Token Usage | `sum(rate(hector_llm_tokens_total[1h])) by (agent)` |
| Error Rate | `rate(hector_requests_total{status="error"}[5m])` |
| Tool Usage | `topk(10, sum by (tool) (hector_tool_calls_total))` |

### Alerting Examples

```yaml
# High error rate
- alert: HectorHighErrorRate
  expr: rate(hector_requests_total{status="error"}[5m]) > 0.1
  for: 5m
  labels:
    severity: warning

# Token budget exhaustion
- alert: HectorHighTokenUsage
  expr: sum(increase(hector_llm_tokens_total[1h])) > 100000
  for: 1h
  labels:
    severity: warning
```

## Health Endpoint

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "ok",
  "version": "v1.20.0",
  "database": "connected"
}
```

## Next Steps

*   [Deployment Guide](./deployment.md)
*   [Operations Guide](./operations.md)
*   [CLI Reference](../reference/cli.md)
