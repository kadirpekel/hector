---
title: Hector Documentation
description: Pure A2A-Native Declarative AI Agent Platform - Complete Documentation
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
    <p class="hero-slogan">Build AI agents without code</p>
    
    <p>A declarative A2A native AI agent platform. Define sophisticated agents through simple YAML configuration.</p>
    
    <p><strong>Built with Go</strong> for production performance, single-binary deployment, and true portability.</p>
    
    <div class="hero-example">
```yaml
agents:
  assistant:
    llm: gpt-4o
    tools: [search, write_file, execute_command]
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

<p><strong>Hector</strong> eliminates code from agent development. Get multi-agent orchestration, advanced memory management, and seamless interoperability through the <a href="https://a2a-protocol.org">Agent-to-Agent protocol</a> out of the box. Hector handles the complexity so you can focus on building intelligent systems.</p>

<p><strong>⚡️ From idea to production agent in minutes, not months.</strong></p>

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
      rows: 20,
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

## Why Hector?

<div class="grid cards" markdown>

-   :zap: __For Developers__

    ---

    - **Zero-code agent development** - YAML configuration only
    - **Instant setup** - Working agent in 5 minutes
    - **Advanced memory** - Working & long-term memory strategies
    - **RAG & semantic search** - Built-in vector store integration
    - **Rich tool ecosystem** - Built-in tools, MCP, and plugins

-   :building_construction: __For Enterprises__

    ---

    - **True interoperability** - Native A2A protocol support
    - **Multi-agent orchestration** - Coordinate specialized agents
    - **Production security** - JWT auth, API keys, agent-level security
    - **Distributed architecture** - Local, server, or client modes
    - **Multi-transport APIs** - REST, SSE, WebSocket, gRPC

-   :busts_in_silhouette: __For Teams__

    ---

    - **Simple configuration** - Human-readable YAML
    - **Declarative approach** - No code to maintain
    - **Built with Go** - Production performance, single binary, no dependencies
    - **Flexible deployment** - Docker, Kubernetes, systemd
    - **Extensible platform** - Custom plugins via gRPC
    - **Open source** - AGPL-3.0 licensed

</div>

## Get Started

<div class="grid cards" markdown>

-   :rocket: __[Getting Started](getting-started/installation.md)__

    ---

    New to Hector? Start here. Install Hector, run your first agent, and validate your setup in under 5 minutes.

-   :books: __[Core Concepts](core-concepts/overview.md)__

    ---

    Learn how Hector works. Understand agents, LLM providers, memory, tools, RAG, reasoning strategies, and multi-agent orchestration.

-   :hammer_and_wrench: __[How-To Guides](how-to/build-coding-assistant.md)__

    ---

    Step-by-step tutorials for common tasks. Build a coding assistant, set up RAG, deploy to production, or integrate external A2A agents.

-   :book: __[CLI Reference](reference/cli.md)__

    ---

    Complete command-line interface reference with all commands, flags, and options.

-   :gear: __[Configuration](reference/configuration.md)__

    ---

    Complete YAML configuration reference for agents, LLMs, tools, memory, and deployment.

-   :globe_with_meridians: __[API Reference](reference/api.md)__

    ---

    REST, gRPC, WebSocket, and JSON-RPC API documentation with examples.

</div>

If you're looking for something specific you can use the search bar at the top of the page.
