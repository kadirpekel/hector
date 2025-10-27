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
</style>

<div class="hero-section">
  <div class="hero-content">
    <p class="hero-slogan">Production-Ready AI Agents. Zero Code Required.</p>

    <p>Deploy observable, secure, and scalable AI agent systems with hot reload, distributed configuration, and A2A-native federation—all configured in YAML.</p>

    <p>Built in Go for production environments, Hector delivers a single 30MB binary with &lt;100ms startup, built-in Prometheus metrics, OpenTelemetry tracing, and security controls—perfect for platform teams deploying AI infrastructure at scale.</p>
  </div>

  <div class="hero-demo">
    <div id="hector-demo"></div>
  </div>
</div>


<script>
(function() {
  var link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = 'https://unpkg.com/asciinema-player@3.6.3/dist/bundle/asciinema-player.css';
  document.head.appendChild(link);
  
  var script = document.createElement('script');
  script.src = 'https://unpkg.com/asciinema-player@3.6.3/dist/bundle/asciinema-player.js';
  script.onload = function() {
    AsciinemaPlayer.create('assets/hector-demo.cast', document.getElementById('hector-demo'), {
      theme: 'asciinema',
      cols: 80,
      rows: 22,
      autoplay: false,
      loop: false,
      speed: 1,
      startAt: 0,
      terminalFontSize: '16px',
      poster: 'npt:0:2',
      pauseOnMarkers: true,
      markers: [[17.0, 'Server & Client Demo']]
    });
  };
  document.head.appendChild(script);
})();
</script>

## Quick Start

```bash
# Install
go install github.com/kadirpekel/hector/cmd/hector@latest

# Create configuration
cat > agents.yaml << 'EOF'
agents:
  assistant:
    llm: gpt-4o
    tools: [search, write_file, search_replace]
    reasoning:
      engine: chain-of-thought
      max_iterations: 100
    memory:
      working:
        strategy: summary_buffer
EOF

# Configure credentials
export OPENAI_API_KEY="sk-..."

# Run server
hector serve --config agents.yaml

# Or use locally
hector call "Explain quantum computing" --config agents.yaml
```

[Get Started Now! →](getting-started/quick-start.md){ .md-button .md-button--primary }

## Key Features

<div class="grid cards" markdown>

-   :zap: __Zero-Code Configuration__

    Pure YAML agent definition. No Python/Go required—define sophisticated agents declaratively.

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

