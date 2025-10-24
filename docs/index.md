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
    <p class="hero-slogan">Declarative A2A-Native AI Agent Platform</p>
    
    <p>Production-ready framework for building, deploying, and orchestrating AI agents at scale.</p>
    
    <p><strong>Built with Go</strong> for production performance, single-binary deployment, and true portability.</p>
    
    <div class="hero-example">
```yaml
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

```
    </div>
  </div>
  
  <div class="hero-demo">
    <div id="hector-demo"></div>
  </div>
</div>

Hector is an AI agent framework designed for production deployment, built in Go for performance and operational simplicity. Define sophisticated multi-agent systems through declarative YAML configuration without writing code.

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
      rows: 25,
      autoplay: false,
      loop: false,
      speed: 1,
      startAt: 0,
      fontSize: 'medium',
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

# Configure
export OPENAI_API_KEY="sk-..."

# Run server
hector serve --config agents.yaml

# Or use locally
hector call "Explain quantum computing"
```

[Installation Guide →](getting-started/installation.md){ .md-button .md-button--primary }
[Quick Start Tutorial →](getting-started/quick-start.md){ .md-button }

## Key Features

<div class="grid cards" markdown>

-   :zap: __Zero-Code Development__

    Define agents through YAML configuration. No programming required.

-   :link: __A2A Protocol Native__

    Standards-compliant agent communication and federation.

-   :brain: __Advanced Memory__

    Working memory strategies and vector-based long-term memory with RAG.

-   :hammer_and_wrench: __Rich Tool Ecosystem__

    Built-in tools, MCP support, and custom plugin system.

-   :thought_balloon: __Reasoning Engines__

    Chain-of-thought and supervisor strategies for single and multi-agent workflows.

-   :shield: __Production Ready__

    Observability, authentication, distributed configuration, and security controls.

</div>

