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

## 🚀 Quick Start

<div class="code-example" markdown="1">

**New to Hector?** Start with our **[Getting Started](getting-started)** guide, then try these popular paths:

### For Beginners
- **[Quick Start](QUICK_START)** - Install and run your first agent in 5 minutes
- **[Building Agents](AGENTS)** - Learn the fundamentals step-by-step

### For LangChain Users  
- **[LangChain vs Hector](tutorials/MULTI_AGENT_RESEARCH_PIPELINE)** - See 500+ lines of Python become 120 lines of YAML
- **[Migration Guide](tutorials/MULTI_AGENT_RESEARCH_PIPELINE#the-dramatic-difference)** - Compare approaches side-by-side

### For Advanced Users
- **[Multi-Agent Systems](ARCHITECTURE#multi-agent-orchestration-a2a-protocol)** - Orchestrate multiple agents
- **[Custom Tools](MCP_CUSTOM_TOOLS)** - Build custom MCP tools in 5 minutes

</div>

---

## 📚 Documentation Sections

<div class="grid-container">

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
