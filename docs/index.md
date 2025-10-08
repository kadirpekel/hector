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

[Get started now](getting-started){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
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
  <p>Authentication, streaming, sessions, monitoring. Built for enterprise from day one.</p>
</div>

<div class="feature-card">
  <h4>🛠️ Extensible</h4>
  <p>MCP protocol support, gRPC plugins. Add custom LLMs, databases, and tools easily.</p>
</div>

</div>

---

## See the Difference

### Traditional Approach (LangChain)
```python
# 500+ lines across 8+ Python files
from langchain.agents import Agent, AgentExecutor
from langchain.tools import Tool
from langchain.memory import ConversationBufferMemory
from langchain.prompts import PromptTemplate
# ... hundreds more lines of setup, state management, 
# error handling, orchestration logic, etc.
```

### Hector Approach (Pure YAML)
```yaml
# 120 lines of YAML - same functionality
agents:
  research_coordinator:
    name: "Research Coordinator"
    llm: "gpt-4o"
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
    
  researcher:
    name: "Web Researcher"
    llm: "gpt-4o-mini"
    tools: ["execute_command"]
```

**Want to see the complete comparison?** Check out our [**LangChain vs Hector Tutorial**](tutorials/MULTI_AGENT_RESEARCH_PIPELINE) 🔥

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

## Popular Learning Paths

<div class="learning-paths">

### 🎯 **New to AI Agents?**
Start with the fundamentals and build your first agent.

1. **[Getting Started](getting-started)** - Set up Hector in 5 minutes
2. **[Quick Start](QUICK_START)** - Run your first agent
3. **[Building Agents](AGENTS)** - Learn core concepts
4. **[Tools & Extensions](TOOLS)** - Add capabilities

### 🔄 **Coming from LangChain/AutoGen?**
See how Hector simplifies what you already know.

1. **[LangChain vs Hector](tutorials/MULTI_AGENT_RESEARCH_PIPELINE)** - Direct comparison
2. **[Multi-Agent Systems](ARCHITECTURE#multi-agent-orchestration-a2a-protocol)** - Orchestration patterns
3. **[Migration Benefits](tutorials/MULTI_AGENT_RESEARCH_PIPELINE#the-dramatic-difference)** - Why switch?

### 🚀 **Building Production Systems?**
Advanced patterns for enterprise deployments.

1. **[Architecture](ARCHITECTURE)** - System design patterns
2. **[Authentication](AUTHENTICATION)** - JWT security
3. **[External Agents](EXTERNAL_AGENTS)** - Distributed systems
4. **[Plugin Development](PLUGINS)** - Custom extensions

</div>

---

## 📚 Documentation Sections

<div class="doc-grid">

<div class="doc-section">
  <h3><a href="getting-started">🎯 Getting Started</a></h3>
  <p>New to Hector? Start here for quick setup and your first agent.</p>
  <ul>
    <li><a href="QUICK_START">Quick Start Guide</a></li>
    <li><a href="AGENTS#your-first-agent">Your First Agent</a></li>
  </ul>
</div>

<div class="doc-section">
  <h3><a href="tutorials/">🎓 Tutorials</a></h3>
  <p>Hands-on learning with real-world examples and comparisons.</p>
  <ul>
    <li><a href="tutorials/MULTI_AGENT_RESEARCH_PIPELINE">LangChain vs Hector</a></li>
    <li><a href="tutorials/BUILD_YOUR_OWN_CURSOR">Build Cursor-like Assistant</a></li>
  </ul>
</div>

<div class="doc-section">
  <h3><a href="core-guides">📖 Core Guides</a></h3>
  <p>Essential knowledge for building production-ready agents.</p>
  <ul>
    <li><a href="AGENTS">Building Agents</a></li>
    <li><a href="TOOLS">Tools & Extensions</a></li>
    <li><a href="MCP_CUSTOM_TOOLS">Custom MCP Tools</a></li>
  </ul>
</div>

<div class="doc-section">
  <h3><a href="advanced">🚀 Advanced</a></h3>
  <p>Complex deployments, integrations, and production patterns.</p>
  <ul>
    <li><a href="ARCHITECTURE">Architecture</a></li>
    <li><a href="EXTERNAL_AGENTS">External Agents</a></li>
    <li><a href="AUTHENTICATION">Authentication</a></li>
  </ul>
</div>

<div class="doc-section">
  <h3><a href="reference">📋 Reference</a></h3>
  <p>Complete technical documentation and API specifications.</p>
  <ul>
    <li><a href="API_REFERENCE">API Reference</a></li>
    <li><a href="A2A_COMPLIANCE">A2A Compliance</a></li>
  </ul>
</div>

</div>

---

## 💡 Why Choose Hector?

| Feature | Hector | LangChain | AutoGen | CrewAI |
|:--------|:-------|:----------|:--------|:-------|
| **Configuration** | Pure YAML | Python code | Python code | Python code |
| **A2A Native** | ✅ 100% | ❌ No | ❌ No | ❌ No |
| **External Agents** | ✅ Seamless | ⚠️ Custom | ⚠️ Custom | ❌ No |
| **Zero Code** | ✅ Yes | ❌ No | ❌ No | ❌ No |
| **Multi-Agent** | ✅ LLM-driven | ✅ Hard-coded | ✅ Hard-coded | ✅ Hard-coded |
| **Production Ready** | ✅ Built-in | ⚠️ DIY | ⚠️ DIY | ⚠️ DIY |

---

## 🔗 Quick Links

- [🏠 Project Homepage]({{ site.hector.repo_url }})
- [📊 A2A Protocol Specification](https://a2a-protocol.org)
- [🐛 Report Issues]({{ site.hector.repo_url }}/issues)
- [🤝 Contributing Guide](CONTRIBUTING)

---

<div style="text-align: center; margin: 2rem 0;">
  <img src="hector-gopher.png" alt="Hector Gopher Mascot" style="width: 80px; height: auto;">
  <p><em>Meet Hector, your AI agent companion!</em></p>
</div>

<style>
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

.learning-paths {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 2rem;
  margin: 2rem 0;
}

.learning-paths > div {
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 1.5rem;
  background: var(--code-background-color);
}

.learning-paths h3 {
  margin-top: 0;
  margin-bottom: 1rem;
}

.learning-paths ol {
  margin: 0;
  padding-left: 1.2rem;
}

.learning-paths li {
  margin-bottom: 0.5rem;
  font-size: 0.9rem;
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
