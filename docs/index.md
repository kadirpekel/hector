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

[Get started now](QUICK_START){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
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
  <p><a href="../A2A_COMPLIANCE">🏆 100% A2A Compliant →</a></p>
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

## Featured Tutorials

<div class="featured-tutorials">

<div class="tutorial-card">
  <h3>🔥 <a href="tutorials/MULTI_AGENT_RESEARCH_PIPELINE">LangChain vs Hector</a></h3>
  <p><strong>Most Popular!</strong> See how Hector transforms complex LangChain multi-agent implementations into simple YAML configuration.</p>
  <div class="tutorial-stats">
    <span class="stat">📊 500+ lines Python → 120 lines YAML</span>
    <span class="stat">⚡ Same functionality, dramatically simpler</span>
  </div>
</div>

<div class="tutorial-card">
  <h3>🤖 <a href="tutorials/BUILD_YOUR_OWN_CURSOR">Build Cursor-like AI Assistant</a></h3>
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

### Single Agent Architecture

<div class="architecture-diagram">
<pre>
┌─────────────────────────────────────────────────────────────┐
│                        USER / CLIENT                        │
│                  (CLI, HTTP, A2A Protocol)                  │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          │ HTTP+JSON / SSE
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                      A2A INTERFACE                          │
│      GetAgentCard() • ExecuteTask() • Streaming (SSE)       │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                    REASONING ENGINE                         │
│  Chain-of-Thought Strategy    |    Supervisor Strategy      │
│  • Step-by-step reasoning     |    • Multi-agent coord      │
│  • Natural termination        |    • Task decomposition     │
└─────────────────────────┬───────────────────────────────────┘
                          │
      ┌───────────────────┼───────────────────┬────────────────┐
      │                   │                   │                │
      ▼                   ▼                   ▼                ▼
┌──────────────┐    ┌──────────────┐   ┌──────────────┐  ┌────────────┐
│    TOOLS     │    │     LLM      │   │     RAG      │  │   MEMORY   │
│              │    │              │   │              │  │            │
│ • Command    │    │ • OpenAI     │   │ • Qdrant     │  │ • Working  │
│ • File Ops   │    │ • Anthropic  │   │ • Semantic   │  │   (Session)│
│ • Search     │    │ • Gemini     │   │   Search     │  │ • Long-term│
│ • MCP        │    │ • Plugins    │   │ • Documents  │  │            │
└──────────────┘    └──────────────┘   └──────────────┘  └────────────┘
</pre>
</div>

### Multi-Agent Architecture

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

## Core Capabilities

Hector provides comprehensive features through pure YAML configuration:

<div class="capabilities-grid">

<div class="capability-section">
  <h3>🎛️ Declarative Configuration</h3>
  <ul>
    <li><strong>Pure YAML</strong> - Zero code for complete agent systems</li>
    <li><strong>6-slot prompt system</strong> - Role, reasoning, tools, output, style, additional</li>
    <li><strong>Environment variables</strong> - Secure API key management</li>
    <li><strong>Multiple LLM providers</strong> - OpenAI, Anthropic, Gemini</li>
  </ul>
</div>

<div class="capability-section">
  <h3>🛠️ Tools & Integrations</h3>
  <ul>
    <li><strong>Built-in tools</strong> - Command execution, file operations, search, todos</li>
    <li><strong>MCP Protocol</strong> - 150+ apps (GitHub, Slack, Gmail, Notion via Composio)</li>
    <li><strong>Custom MCP tools</strong> - Build your own in 5 minutes (Python/TypeScript) 🔥</li>
    <li><strong>Security controls</strong> - Command whitelisting, path restrictions, timeouts</li>
  </ul>
</div>

<div class="capability-section">
  <h3>🧠 Memory Management</h3>
  <ul>
    <li><strong>Working memory (short-term)</strong> - Pluggable strategies for session history: token-based with summarization (default) or simple LIFO</li>
    <li><strong>Accurate token counting</strong> - 100% accurate using tiktoken, never exceed limits</li>
    <li><strong>Recency-based selection</strong> - Most recent messages preserved automatically</li>
    <li><strong>Long-term memory</strong> - Session-scoped persistent memory with vector storage and semantic recall</li>
  </ul>
</div>

<div class="capability-section">
  <h3>📚 RAG & Knowledge</h3>
  <ul>
    <li><strong>Vector databases</strong> - Qdrant, Pinecone, or custom via plugins</li>
    <li><strong>Semantic search</strong> - Automatic document retrieval</li>
    <li><strong>Document stores</strong> - Organize knowledge by domain</li>
    <li><strong>Embeddings</strong> - Ollama or custom embedder plugins</li>
  </ul>
</div>

<div class="capability-section">
  <h3>💬 Sessions & Streaming</h3>
  <ul>
    <li><strong>Multi-turn conversations</strong> - Persistent conversation history</li>
    <li><strong>Server-Sent Events</strong> - Real-time A2A-compliant streaming</li>
    <li><strong>Session management</strong> - Create, list, delete sessions via API</li>
    <li><strong>Context retention</strong> - Agent remembers conversation across messages</li>
  </ul>
</div>

<div class="capability-section">
  <h3>🤝 Multi-Agent Orchestration</h3>
  <ul>
    <li><strong>LLM-driven routing</strong> - Agent decides which specialist to delegate to</li>
    <li><strong>Native + External</strong> - Mix local and remote A2A agents</li>
    <li><strong>agent_call tool</strong> - Automatic orchestration capability</li>
    <li><strong>Supervisor strategy</strong> - Optimized for coordination tasks</li>
  </ul>
</div>

<div class="capability-section">
  <h3>🔌 Plugin System (gRPC)</h3>
  <ul>
    <li><strong>Language-agnostic</strong> - Write in Go, Python, Rust, JavaScript, etc.</li>
    <li><strong>Custom LLMs</strong> - Integrate proprietary models or local inference</li>
    <li><strong>Custom databases</strong> - Add specialized vector stores</li>
    <li><strong>Process isolation</strong> - Plugins run in separate processes for stability</li>
  </ul>
</div>

<div class="capability-section">
  <h3>🔒 Security & Deployment</h3>
  <ul>
    <li><strong>JWT Authentication</strong> - OAuth2/OIDC integration</li>
    <li><strong>Visibility control</strong> - Public, internal, private agents</li>
    <li><strong>Tool security</strong> - Whitelisting, sandboxing, resource limits</li>
    <li><strong>Docker support</strong> - Production-ready containerization</li>
  </ul>
</div>

<div class="capability-section">
  <h3>📡 A2A Protocol Compliance</h3>
  <ul>
    <li><strong>Agent Cards</strong> - Standard capability discovery</li>
    <li><strong>HTTP+JSON transport</strong> - RESTful A2A endpoints</li>
    <li><strong>SSE streaming</strong> - Real-time output per spec</li>
    <li><strong>Task management</strong> - Create, get status, cancel tasks</li>
  </ul>
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

<div class="learning-path">
<h3>🎯 <strong>New to AI Agents?</strong></h3>
<p>Start with the fundamentals and build your first agent.</p>
<ol>
<li><strong><a href="INSTALLATION">Installation</a></strong> - Complete installation options</li>
<li><strong><a href="QUICK_START">Quick Start</a></strong> - Set up Hector in 5 minutes</li>
<li><strong><a href="QUICK_START">Quick Start</a></strong> - Run your first agent</li>
<li><strong><a href="AGENTS">Building Agents</a></strong> - Learn core concepts</li>
<li><strong><a href="TOOLS">Tools & Extensions</a></strong> - Add capabilities</li>
</ol>
</div>

<div class="learning-path">
<h3>🔄 <strong>Coming from LangChain/AutoGen?</strong></h3>
<p>See how Hector simplifies what you already know.</p>
<ol>
<li><strong><a href="tutorials/MULTI_AGENT_RESEARCH_PIPELINE">LangChain vs Hector</a></strong> - Direct comparison</li>
<li><strong><a href="ARCHITECTURE#multi-agent-orchestration-a2a-protocol">Multi-Agent Systems</a></strong> - Orchestration patterns</li>
<li><strong><a href="tutorials/MULTI_AGENT_RESEARCH_PIPELINE#the-dramatic-difference">Migration Benefits</a></strong> - Why switch?</li>
</ol>
</div>

<div class="learning-path">
<h3>🚀 <strong>Building Production Systems?</strong></h3>
<p>Advanced patterns for enterprise deployments.</p>
<ol>
<li><strong><a href="ARCHITECTURE">Architecture</a></strong> - System design patterns</li>
<li><strong><a href="AUTHENTICATION">Authentication</a></strong> - JWT security</li>
<li><strong><a href="EXTERNAL_AGENTS">External Agents</a></strong> - Distributed systems</li>
<li><strong><a href="PLUGINS">Plugin Development</a></strong> - Custom extensions</li>
</ol>
</div>

</div>

---

## 📚 Documentation Sections

<div class="doc-grid">

<div class="doc-section">
  <h3><a href="QUICK_START">🎯 Getting Started</a></h3>
  <p>New to Hector? Start here for quick setup and your first agent.</p>
  <ul>
    <li><a href="INSTALLATION">Installation Guide</a></li>
    <li><a href="CLI_GUIDE">CLI Guide</a></li>
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
    <li><a href="MEMORY">Memory Management</a></li>
    <li><a href="TOOLS">Tools & Extensions</a></li>
    <li><a href="MCP_CUSTOM_TOOLS">Custom MCP Tools</a></li>
  </ul>
</div>

<div class="doc-section">
  <h3><a href="advanced">🚀 Advanced</a></h3>
  <p>Complex deployments, integrations, and production patterns.</p>
  <ul>
    <li><a href="ARCHITECTURE">Architecture</a></li>
    <li><a href="MEMORY_CONFIGURATION">Memory Configuration</a></li>
    <li><a href="EXTERNAL_AGENTS">External Agents</a></li>
    <li><a href="AUTHENTICATION">Authentication</a></li>
    <li><a href="MEMORY_CONFIGURATION#long-term-memory-configuration">Long-Term Memory</a></li>
  </ul>
</div>

<div class="doc-section">
  <h3><a href="reference">📋 Reference</a></h3>
  <p>Complete technical documentation and API specifications.</p>
  <ul>
    <li><a href="API_REFERENCE">API Reference</a></li>
    <li><a href="CONFIGURATION">Configuration</a></li>
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
