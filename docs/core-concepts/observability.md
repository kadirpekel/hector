# Observability

Hector provides enterprise-grade observability through OpenTelemetry and Prometheus integration, enabling comprehensive monitoring, tracing, and metrics collection for your AI agents.

## Overview

Hector's observability stack includes:

- **Metrics** (Prometheus): Aggregate statistics about agent performance, throughput, and resource usage
- **Distributed Tracing** (OpenTelemetry/Jaeger): Detailed request-level visibility with timing and context
- **Dashboards** (Grafana): Pre-built visualizations for monitoring and alerting

## Architecture

```
┌─────────┐
│ Hector  │
└────┬────┘
     │
     ├─────────────────┐
     │                 │
     ▼                 ▼
┌─────────┐      ┌──────────┐
│Prometheus│      │  Jaeger  │
│(Metrics) │      │ (Traces) │
└────┬─────┘      └──────────┘
     │
     ▼
┌─────────┐
│ Grafana │
│(Dashboards)
└─────────┘
```

## Configuration

### Basic Configuration

Add observability configuration to your Hector config file:

```yaml
global:
  observability:
    tracing:
      enabled: true
      exporter_type: "jaeger"
      endpoint_url: "localhost:4317"
      sampling_rate: 1.0
      service_name: "hector"
    metrics:
      enabled: true
      port: 8080
```

### Configuration Options

#### Tracing

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable distributed tracing |
| `exporter_type` | string | `"jaeger"` | Trace exporter type (jaeger, otlp) |
| `endpoint_url` | string | `"localhost:4317"` | OTLP gRPC endpoint |
| `sampling_rate` | float | `1.0` | Sampling rate (0.0-1.0, 1.0 = 100%) |
| `service_name` | string | `"hector"` | Service name in traces |

**Note:** All tracing defaults are applied automatically via `SetDefaults()`.

#### Metrics

Metrics can be enabled with a simple boolean flag:

```yaml
observability:
  metrics_enabled: true
```

**Note:** Metrics are served on the HTTP server at `/metrics` endpoint.

## Metrics Collected

### Agent Metrics

- `hector_agent_calls_total` - Total number of agent executions
- `hector_agent_call_duration_seconds` - Agent execution duration (histogram)
- `hector_agent_errors_total` - Total agent execution errors
- `hector_agent_tokens_total` - Total tokens used across all agents

### Tool Metrics

- `hector_tool_calls_total` - Total tool executions
- `hector_tool_errors_total` - Total tool execution failures
- `hector_tool_execution_duration_seconds` - Tool execution duration

### LLM Metrics

- `hector_llm_calls_total` - Total LLM API calls
- `hector_llm_errors_total` - Total LLM API failures
- `hector_llm_request_duration_seconds` - LLM request duration
- `hector_llm_input_tokens_total` - Total input tokens
- `hector_llm_output_tokens_total` - Total output tokens

## Traces

Each agent execution creates a trace with the following structure:

```
Span: agent.call
├── Attributes:
│   ├── agent.name: "assistant"
│   ├── agent.llm: "gpt-4o"
│   └── input_preview: "User input..." (first 100 chars)
└── Duration: Execution time in milliseconds
```

## Quick Start

### 1. Start Observability Stack

Use the provided Docker Compose file:

```bash
docker-compose -f deployments/docker-compose.observability.yaml up -d
```

This starts:

- **Jaeger** on port 16686 (UI) and 4317 (OTLP gRPC)
- **Prometheus** on port 9090
- **Grafana** on port 3000 (login: admin/Dev12345)

### 2. Configure Hector

Use the example configuration:

```bash
cp configs/observability-example.yaml my-config.yaml
# Edit my-config.yaml with your settings
```

### 3. Start Hector

```bash
hector serve --config my-config.yaml
```

### 4. Access Dashboards

- **Metrics Endpoint**: http://localhost:8080/metrics
- **Prometheus**: http://localhost:9090
- **Jaeger**: http://localhost:16686
- **Grafana**: http://localhost:3000

## Prometheus Queries

### Basic Queries

```promql
# Total agent calls
hector_agent_calls_total

# Agent call rate (calls per second)
rate(hector_agent_calls_total[1m])

# Average call duration
rate(hector_agent_call_duration_seconds_sum[5m]) / rate(hector_agent_call_duration_seconds_count[5m])

# 95th percentile latency
histogram_quantile(0.95, rate(hector_agent_call_duration_seconds_bucket[5m]))

# Error rate
rate(hector_agent_errors_total[5m])
```

### Alerting Queries

```promql
# High error rate (>5%)
rate(hector_agent_errors_total[5m]) / rate(hector_agent_calls_total[5m]) > 0.05

# High latency (P95 > 10s)
histogram_quantile(0.95, rate(hector_agent_call_duration_seconds_bucket[5m])) > 10

# Scraping failure
up{job="hector"} == 0
```

## Grafana Setup

### Connect Prometheus Datasource

Prometheus is automatically configured when using the Docker Compose stack. To manually add:

1. Go to Configuration → Data Sources
2. Add Prometheus datasource
3. URL: `http://prometheus:9090` (if in Docker) or `http://localhost:9090`
4. Save & Test

### Create Dashboards

Use these panel queries:

**Agent Throughput**:
```promql
rate(hector_agent_calls_total[5m])
```

**Average Response Time**:
```promql
rate(hector_agent_call_duration_seconds_sum[5m]) / rate(hector_agent_call_duration_seconds_count[5m])
```

**Error Rate**:
```promql
rate(hector_agent_errors_total[5m])
```

## Jaeger Usage

### View Traces

1. Open http://localhost:16686
2. Select Service: **hector**
3. Click "Find Traces"
4. Click any trace to see:

   - Request timeline
   - Duration breakdown
   - Agent attributes (name, LLM, input)

### Search Traces

- **By duration**: Set min/max duration
- **By tags**: Filter by `agent.name`, `agent.llm`
- **By time**: Select time range

## Production Considerations

### Sampling

For high-volume production environments, reduce sampling rate:

```yaml
observability:
  tracing:
    sampling_rate: 0.1  # Sample 10% of requests
```

### Security

- **Metrics endpoint**: Consider adding authentication
- **Jaeger**: Deploy with authentication enabled
- **Grafana**: Change default password
- **Prometheus**: Enable basic auth or use reverse proxy

### Performance

- Metrics collection: ~0.1ms overhead per request
- Trace export: Asynchronous, no blocking
- Memory: ~10MB for observability components

### Storage

- **Prometheus**: Configure retention period
  ```yaml
  --storage.tsdb.retention.time=30d
  ```

- **Jaeger**: Configure storage backend (Cassandra, Elasticsearch)

## Troubleshooting

### No Metrics in Prometheus

Check Prometheus targets:
```bash
curl http://localhost:9090/api/v1/targets
```

Verify Hector metrics endpoint:
```bash
curl http://localhost:8080/metrics | grep hector_
```

### No Traces in Jaeger

Check Hector logs for trace export errors:
```bash
grep -i trace hector.log
```

Verify Jaeger OTLP endpoint:
```bash
curl http://localhost:4317
```

### Grafana Can't Query Prometheus

1. Check datasource configuration
2. Verify Prometheus is running: `curl http://localhost:9090/-/healthy`
3. Test query in Prometheus UI first

## Best Practices

1. **Always enable in production** - Observability is crucial for debugging and monitoring
2. **Set appropriate sampling** - Balance between visibility and overhead
3. **Create alerts** - Monitor error rates and latency
4. **Use dashboards** - Visualize trends over time
5. **Correlate metrics and traces** - Use both for complete visibility

## See Also

- [Configuration Reference](../reference/configuration.md)
- [Configuration Reference](../reference/configuration.md) - Production configuration
- [Example Configurations](https://github.com/kadirpekel/hector/tree/main/configs)

