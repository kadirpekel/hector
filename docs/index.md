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
    <p class="hero-slogan">AI Agents with Confidence!</p>
    
    <p>Hector is a declarative A2A-Native AI Agent platform for building, deploying, and orchestrating AI agents at scale.</p>
    
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

Built in Go, Hector lets you deploy powerful distributed AI agents on top of A2A protocol using straightforward YAML—no custom coding required. Ideal for production environments that demand speed and simplicity.

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
      rows: 21,
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

[Get Started Now! →](getting-started/quick-start.md){ .md-button .md-button--primary }

## Key Features

<div class="grid cards" markdown>

-   :zap: __Declarative Configuration__

    YAML-based agent definition. No programming required for agent creation.

-   :link: __A2A Protocol Native__

    Standards-compliant agent communication and federation for distributed deployments.

-   :globe_with_meridians: __Distributed Architecture__

    Support for local, server, and federated deployments with seamless scaling.

-   :shield: __Production Ready__

    Built-in observability, authentication, distributed configuration, and security controls.

-   :package: __Single Binary Deployment__

    No runtime dependencies or complex installations. One executable, zero configuration.

-   :rocket: __Featureful Agent System__

    Multi-agent orchestration, advanced memory, rich tool ecosystem, reasoning engines, builtin rag, session persistence and many more.

</div>

