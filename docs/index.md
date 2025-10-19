---
title: Hector Documentation
description: Pure A2A-Native Declarative AI Agent Platform - Complete Documentation
hide:
  - navigation
  - toc
---

<style>
.md-typeset h1 {
  position: absolute;
  left: -10000px;
  opacity: 0;
}
</style>

**Hector** is a declarative AI agent platform that eliminates code from agent development. Define sophisticated AI agents through YAML configuration, with built-in support for multi-agent orchestration, advanced memory management, and seamless interoperability through the [Agent-to-Agent protocol](https://a2a-protocol.org).

  **⚡️ From idea to production agent in minutes, not months.**

Build AI assistants, automate complex workflows, create multi-agent research systems, or integrate with external A2A services—all without writing code. Hector handles the complexity so you can focus on building intelligent systems.

## See Hector in Action

<div id="hector-demo"></div>

<script>
  // Load asciinema player CSS
  var link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = 'https://unpkg.com/asciinema-player@3.6.3/dist/bundle/asciinema-player.css';
  document.head.appendChild(link);
  
  // Load asciinema player script
  var script = document.createElement('script');
  script.src = 'https://unpkg.com/asciinema-player@3.6.3/dist/bundle/asciinema-player.js';
  script.onload = function() {
        AsciinemaPlayer.create('assets/hector-demo.cast', document.getElementById('hector-demo'), {
          theme: 'asciinema',
          cols: 120,
          rows: 30,
          autoplay: false,
          loop: false,
          speed: 1,
          startAt: 0,
          fontSize: 'medium',
          poster: 'npt:0:2',
          pauseOnMarkers: true,
          markers: [
            [17.0, 'Server & Client Demo']
          ]
        });
  };
  document.head.appendChild(script);
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
    - **Flexible deployment** - Docker, Kubernetes, systemd
    - **Extensible platform** - Custom plugins via gRPC
    - **Open source** - MIT licensed

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

-   :book: __[Reference](reference/cli.md)__

    ---

    Technical reference documentation. CLI commands, configuration syntax, API endpoints, architecture details, and A2A protocol specifications.

</div>

If you're looking for something specific you can use the search bar at the top of the page.