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

**Hector** is a declarative AI agent platform that eliminates code from agent development. Built on the [Agent-to-Agent protocol](https://a2a-protocol.org), Hector enables true interoperability between agents across networks, servers, and organizations.

⚡ **From idea to production agent in minutes, not months.**

Its typical use cases include building AI assistants, automating workflows, creating multi-agent systems, and integrating with external services. It focuses on simplicity and declarative configuration, which makes it a popular choice among developers and organizations working with AI agents.

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
      poster: 'npt:0:2'
    });
  };
  document.head.appendChild(script);
</script>

<div class="grid cards" markdown>
-   :rocket: __[Getting Started](installation.md)__

    ---

    New to Hector? Start here with the essentials. Learn how to install Hector, run your first agent, and understand the core concepts that make Hector powerful.

-   :brain: __[Core Concepts](agents.md)__

    ---

    Ready to build sophisticated AI agents? Explore advanced features like multi-agent orchestration, memory management, reasoning strategies, and production deployment.

-   :wrench: __[How To](tutorial-cursor.md)__

    ---

    Practical guides and tutorials for common use cases. Learn how to integrate with external services, build custom tools, configure authentication, and optimize performance.
</div>

If you're looking for something specific you can use the search bar at the top of the page.