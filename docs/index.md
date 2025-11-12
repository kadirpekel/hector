---
title: Hector Documentation
description: Declarative A2A-Native AI Agent Platform - Complete Documentation
hide:
  - navigation
  - toc
---

<style>
.md-content h1:first-child {
  display: none;
}
.md-content {
  padding-top: 0.5rem;
}
</style>

<p class="hero-slogan">Production-Ready AI Agents. Zero Code Required.</p>

<div class="hero-section">
  <div class="hero-demo">
    <video controls autoplay muted loop playsinline class="hector-video">
      <source src="assets/hector_chat.mp4" type="video/mp4">
      Your browser does not support the video tag.
    </video>
  </div>

  <div class="hero-content">

```bash
export OPENAI_API_KEY="sk-..." MCP_URL="http://localhost:3000"

cat > weather.yaml << 'EOF'
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"

tools:
  weather:
    type: "mcp"
    server_url: "${MCP_URL}"

agents:
  weather_assistant:
    name: Weather Assistant
    llm: "gpt-4o"
    tools: [weather]
EOF

hector serve --config weather.yaml

# Open your browser and experience the interactive chat interface
open http://localhost:8080
```

  </div>
</div>

<div class="hero-intro">
  <p>Deploy observable, secure, and scalable AI agent systems with hot reload, distributed configuration, and A2A-native federation. Configure in YAML or build programmatically in Go with full support for document stores, observability, and all agent features.</p>

  <p>Built for real-world complexity: orchestrate multiple specialized agents, equip them with advanced reasoning strategies like chain-of-thought and tree search, give them long-term memory with RAG and vector stores, connect them to any tool or API, and stream responses in real-time. Enable production-ready human-in-the-loop workflows with async state persistence and checkpoint recovery that survive server restarts. All while maintaining production-grade observability and security.</p>
</div>

[Get Started →](getting-started/quick-start.md){ .md-button .md-button--primary }

## Key Features

<div class="grid cards" markdown>

-   :zap: __Zero-Code Configuration__

    Pure YAML agent definition. No Python/Go required—define sophisticated agents declaratively. Or use the [programmatic API](core-concepts/programmatic-api.md) for Go code integration with full support for document stores, observability, and all agent features.

-   :chart_with_upwards_trend: __Production Observability__

    Built-in Prometheus metrics and OpenTelemetry tracing. Monitor latency, token usage, costs, and errors.

-   :shield: __Security-First__

    JWT auth, RBAC, and command sandboxing out of the box. Production-grade security controls.

-   :arrows_counterclockwise: __Hot Reload__

    Update configurations without downtime. Reload from Consul/Etcd/ZooKeeper automatically.

-   :link: __A2A Protocol Native__

    Standards-compliant agent communication and federation for distributed, interoperable deployments.

-   :rocket: __Resource Efficient__

    Single 30MB binary, <100ms startup. 10-20x less resource usage than Python frameworks.

</div>

