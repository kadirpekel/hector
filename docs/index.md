---
layout: default
title: Home
nav_order: 1
description: "Pure A2A-Native Declarative AI Agent Platform - Complete Documentation"
permalink: /
---

# Hector Documentation
{: .fs-9 }

Pure A2A-Native Declarative AI Agent Platform
{: .fs-6 .fw-300 }

![Hector Gopher](hector-gopher.png){: .float-right .ml-4 style="width: 120px; height: auto;"}

Build powerful AI agents in pure YAML. Compose single agents, orchestrate multi-agent systems, and integrate external A2A agents—all through declarative configuration and industry-standard protocols.
{: .fs-5 .fw-300 }

[Get started now](getting-started/QUICK_START){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View on GitHub]({{ site.hector.repo_url }}){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## What is Hector?

Hector is a **declarative AI agent platform** that eliminates code from agent development. Unlike Python-based frameworks (LangChain, AutoGen, CrewAI), Hector uses **pure YAML configuration** to define complete agent systems.

### The Hector Advantage

<div class="grid-container">

<div class="feature-card">
  <h4>🎯 Zero Code Required</h4>
  <p>Define agents, tools, prompts, and orchestration in pure YAML. No Python, no JavaScript, no complexity.</p>
</div>

<div class="feature-card">
  <h4>🌐 100% A2A Native</h4>
  <p>Built on the <a href="https://a2a-protocol.org">Agent-to-Agent protocol</a> for true interoperability and standardization.</p>
  <p><a href="A2A_COMPLIANCE">🏆 100% A2A Compliant →</a></p>
</div>

<div class="feature-card">
  <h4>🚀 Single & Multi-Agent</h4>
  <p>From individual agents to complex orchestration. Scale naturally from simple to sophisticated.</p>
</div>

<div class="feature-card">
  <h4>🔗 External Integration</h4>
  <p>Connect remote A2A agents seamlessly. Build distributed agent networks effortlessly.</p>
</div>

<div class="feature-card">
  <h4>⚡ Production Ready</h4>
  <p>Authentication, streaming, pluggable session stores (SQL/Redis/Memory), monitoring. Built for enterprise from day one.</p>
  <p><a href="SESSION_STORES">📦 Session Stores →</a></p>
</div>

<div class="feature-card">
  <h4>🛠️ Extensible</h4>
  <p>MCP protocol support, gRPC plugins. Add custom LLMs, databases, and tools easily.</p>
</div>

</div>

---

## Featured How-To Guides

<div class="featured-tutorials">

<div class="tutorial-card">
  <h3>🔥 <a href="architecture-design/TUTORIAL_MULTI_AGENT">LangChain vs Hector</a></h3>
  <p><strong>Most Popular!</strong> See how Hector transforms complex LangChain multi-agent implementations into simple YAML configuration.</p>
  <div class="tutorial-stats">
    <span class="stat">📊 500+ lines Python → 120 lines YAML</span>
    <span class="stat">⚡ Same functionality, dramatically simpler</span>
  </div>
</div>

<div class="tutorial-card">
  <h3>🤖 <a href="how-to/TUTORIAL_CURSOR">AI Coding Assistant Tutorial</a></h3>
  <p><strong>Build Your Own AI Coding Assistant!</strong> Create a powerful Cursor-like AI coding assistant with semantic search and chain-of-thought reasoning—all in pure YAML.</p>
  <div class="tutorial-stats">
    <span class="stat">💻 Full IDE-like capabilities</span>
    <span class="stat">🧠 Chain-of-thought reasoning</span>
  </div>
</div>

</div>

---

## Architecture Overview

Hector's clean architecture scales from single agents to complex multi-agent systems:

<div class="architecture-diagram">
<pre>
┌─────────────────────────────────────────────────────────────┐
│                        USER / CLIENT                        │
│                  (CLI, HTTP, A2A Protocol)                  │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          │ A2A Protocol (HTTP+JSON/SSE)
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                      A2A SERVER                             │
│         • Discovery (/agents)    • Execution (/tasks)       │
│         • Sessions               • Streaming (SSE)          │
└─────────────────────────┬───────────────────────────────────┘
                          │
      ┌───────────────────┼───────────────────┐
      │                   │                   │
      ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐   ┌──────────────┐
│Orchestrator  │    │   Native     │   │   External   │
│    Agent     │    │   Agents     │   │  A2A Agents  │
│              │    │              │   │              │
│ • Supervisor │    │ • Local      │   │ • Remote URL │
│ • agent_call │    │ • Full Ctrl  │   │ • HTTP Proxy │
│ • Synthesis  │    │              │   │ • Same Iface │
└──────┬───────┘    └──────────────┘   └──────────────┘
       │
       │ LLM-Driven Routing (agent_call tool)
       └──────────────────┐
                          ▼
                  ┌───────────────┐
                  │ Agent Registry│
                  │  (All Agents) │
                  └───────────────┘
</pre>
</div>

---

## Quick Example

Here's a complete AI agent in pure YAML:

```yaml
agents:
  assistant:
    name: "My Assistant"
    llm: "gpt-4o"
    prompt:
      system_role: |
        You are a helpful assistant who explains concepts clearly
        and provides actionable guidance.

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
```

**That's it!** Start the server and you have a working AI agent with:
- ✅ Streaming responses
- ✅ Session management  
- ✅ A2A protocol compliance
- ✅ Built-in security
- ✅ Production monitoring

---

## 📚 Documentation Sections

<div class="doc-grid">

<div class="doc-section">
  <h3><a href="getting-started/QUICK_START">🎯 Getting Started</a></h3>
  <p>New to Hector? Start here for quick setup and your first agent.</p>
  <ul>
    <li><a href="getting-started/INSTALLATION">Installation Guide</a></li>
    <li><a href="getting-started/CLI_GUIDE">CLI Guide</a></li>
  </ul>
</div>

<div class="doc-section">
  <h3><a href="agent-capabilities">🎯 Agent Capabilities</a></h3>
  <p>What agents can do - organized by capability.</p>
  <ul>
    <li><a href="agent-capabilities/intelligence-reasoning">Intelligence & Reasoning</a></li>
    <li><a href="agent-capabilities/memory-context">Memory & Context</a></li>
    <li><a href="agent-capabilities/tools-actions">Tools & Actions</a></li>
    <li><a href="agent-capabilities/knowledge-rag">Knowledge & RAG</a></li>
  </ul>
</div>

<div class="doc-section">
  <h3><a href="how-to">🎓 How-To Guides</a></h3>
  <p>Step-by-step guides for common Hector tasks and integrations.</p>
  <ul>
    <li><a href="how-to/TUTORIAL_CURSOR">AI Coding Assistant Tutorial</a></li>
    <li><a href="how-to/MCP_CUSTOM_TOOLS">Custom MCP Tools</a></li>
  </ul>
</div>

<div class="doc-section">
  <h3><a href="architecture-design">🏗️ Architecture & Design</a></h3>
  <p>System architecture, design patterns, and technical comparisons.</p>
  <ul>
    <li><a href="architecture-design/ARCHITECTURE">Architecture</a></li>
    <li><a href="architecture-design/A2A_NATIVE_ARCHITECTURE">A2A Native Architecture</a></li>
    <li><a href="architecture-design/TUTORIAL_MULTI_AGENT">LangChain vs Hector</a></li>
  </ul>
</div>

<div class="doc-section">
  <h3><a href="development">🛠️ Development</a></h3>
  <p>Development tools, plugins, and contributing to Hector.</p>
  <ul>
    <li><a href="development/PLUGINS">Plugin Development</a></li>
    <li><a href="development/CONTRIBUTING">Contributing</a></li>
  </ul>
</div>

<div class="doc-section">
  <h3><a href="reference">📋 Reference</a></h3>
  <p>Complete technical documentation and API specifications.</p>
  <ul>
    <li><a href="reference/API_REFERENCE">API Reference</a></li>
    <li><a href="reference/CONFIGURATION">Configuration</a></li>
    <li><a href="reference/A2A_COMPLIANCE">A2A Compliance</a></li>
  </ul>
</div>

</div>

---

## 🔗 Quick Links & License

- [🏠 Project Homepage]({{ site.hector.repo_url }})
- [📊 A2A Protocol Specification](https://a2a-protocol.org)
- [🐛 Report Issues]({{ site.hector.repo_url }}/issues)
- [🤝 Contributing Guide](CONTRIBUTING)

### License

**Dual License** - Hector uses different licenses for different use cases:

**🏠 Non-Commercial Use (AGPL-3.0):**
- ✅ **Free for personal, educational, research use**
- ✅ **Modify and redistribute freely**
- ⚠️ **Must provide source code when distributing**
- ⚠️ **Network services must offer source code**

**💼 Commercial Use (Separate License):**
- 💼 **For-profit companies and SaaS products**
- 💼 **No source code disclosure requirements**
- 💼 **Priority support and legal indemnification**
- 📞 **Contact via [GitHub Issues]({{ site.hector.repo_url }}/issues) for licensing**

See the [complete license details]({{ site.hector.repo_url }}/blob/main/LICENSE.md) for full terms and what constitutes commercial vs. non-commercial use.

---

<div style="text-align: center; margin: 2rem 0;">
  <img src="hector-gopher.png" alt="Hector Gopher Mascot" style="width: 80px; height: auto;">
  <p><em>Meet Hector, your AI agent companion!</em></p>
</div>

<style>
.architecture-diagram {
  background: var(--code-background-color);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 1.5rem;
  margin: 1.5rem 0;
  overflow-x: auto;
}

.architecture-diagram pre {
  margin: 0;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.85rem;
  line-height: 1.2;
  color: var(--body-text-color);
  background: transparent;
  border: none;
  padding: 0;
}

.grid-container {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1.5rem;
  margin: 2rem 0;
}

.feature-card {
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 1.5rem;
  background: var(--code-background-color);
  text-align: center;
}

.feature-card h4 {
  margin-top: 0;
  margin-bottom: 0.75rem;
  font-size: 1.1rem;
}

.feature-card p {
  font-size: 0.9rem;
  color: var(--body-text-color);
  margin: 0;
  line-height: 1.4;
}

.featured-tutorials {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 2rem;
  margin: 2rem 0;
}

.tutorial-card {
  border: 2px solid var(--accent-color);
  border-radius: 12px;
  padding: 2rem;
  background: var(--code-background-color);
  position: relative;
}

.tutorial-card h3 {
  margin-top: 0;
  margin-bottom: 1rem;
  font-size: 1.3rem;
}

.tutorial-card h3 a {
  text-decoration: none;
  color: inherit;
}

.tutorial-card h3 a:hover {
  text-decoration: underline;
}

.tutorial-card p {
  font-size: 1rem;
  color: var(--body-text-color);
  margin-bottom: 1rem;
  line-height: 1.5;
}

.tutorial-stats {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.tutorial-stats .stat {
  font-size: 0.9rem;
  color: var(--accent-color);
  font-weight: 500;
}

.capabilities-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1.5rem;
  margin: 2rem 0;
}

.capability-section {
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 1.5rem;
  background: var(--code-background-color);
}

.capability-section h3 {
  margin-top: 0;
  margin-bottom: 1rem;
  font-size: 1.1rem;
  color: var(--accent-color);
}

.capability-section ul {
  margin: 0;
  padding-left: 1.2rem;
}

.capability-section li {
  margin-bottom: 0.5rem;
  font-size: 0.9rem;
  line-height: 1.4;
}

.learning-paths {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 2rem;
  margin: 2rem 0;
}

.learning-path {
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 1.5rem;
  background: var(--code-background-color);
}

.learning-path h3 {
  margin-top: 0;
  margin-bottom: 0.75rem;
  font-size: 1.1rem;
}

.learning-path p {
  font-size: 0.9rem;
  color: var(--body-text-color);
  margin-bottom: 1rem;
  line-height: 1.4;
}

.learning-path ol {
  margin: 0;
  padding-left: 1.2rem;
}

.learning-path li {
  margin-bottom: 0.5rem;
  font-size: 0.9rem;
  line-height: 1.4;
}

.doc-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1.5rem;
  margin: 2rem 0;
}

.doc-section {
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 1.5rem;
  background: var(--code-background-color);
}

.doc-section h3 {
  margin-top: 0;
  margin-bottom: 0.5rem;
}

.doc-section h3 a {
  text-decoration: none;
  color: inherit;
}

.doc-section h3 a:hover {
  text-decoration: underline;
}

.doc-section p {
  font-size: 0.9rem;
  color: var(--body-text-color);
  margin-bottom: 1rem;
}

.doc-section ul {
  margin: 0;
  padding-left: 1.2rem;
}

.doc-section li {
  margin-bottom: 0.25rem;
  font-size: 0.9rem;
}
</style>
