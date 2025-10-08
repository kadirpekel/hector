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

[Get started now](/QUICK_START){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View on GitHub]({{ site.hector.repo_url }}){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## 🚀 Getting Started

<div class="code-example" markdown="1">

**Want to see the power of Hector?** Check out our featured tutorial:

### [LangChain vs Hector: Multi-Agent Systems](/tutorials/MULTI_AGENT_RESEARCH_PIPELINE)
{: .text-purple-000}

See how Hector transforms complex LangChain multi-agent implementations into simple YAML configuration. **What takes 500+ lines of Python code becomes 120 lines of YAML** - same functionality, dramatically simpler approach.

[Read the comparison →](/tutorials/MULTI_AGENT_RESEARCH_PIPELINE){: .btn .btn-outline }

</div>

---

## 📚 Popular Guides

| Guide | Description |
|:------|:------------|
| [**Quick Start**](/QUICK_START) | Get up and running in 5 minutes |
| [**Building Agents**](/AGENTS) | Complete single-agent guide with prompts, tools, RAG |
| [**LangChain vs Hector**](/tutorials/MULTI_AGENT_RESEARCH_PIPELINE) | Multi-agent systems comparison tutorial |
| [**Custom MCP Tools**](/MCP_CUSTOM_TOOLS) | Build custom tools in 5 minutes 🔥 |
| [**Tools & Extensions**](/TOOLS) | Built-in tools, MCP protocol, gRPC plugins |

---

## 🌟 Why Hector?

| Feature | Hector | LangChain | AutoGen | CrewAI |
|:--------|:-------|:----------|:--------|:-------|
| **Configuration** | Pure YAML | Python code | Python code | Python code |
| **A2A Native** | ✅ 100% | ❌ No | ❌ No | ❌ No |
| **External Agents** | ✅ Seamless | ⚠️ Custom | ⚠️ Custom | ❌ No |
| **Zero Code** | ✅ Yes | ❌ No | ❌ No | ❌ No |
| **Multi-Agent** | ✅ LLM-driven | ✅ Hard-coded | ✅ Hard-coded | ✅ Hard-coded |

---

## 💡 Quick Example

Here's a complete AI agent in pure YAML:

```yaml
agents:
  assistant:
    name: "My Assistant"
    llm: "gpt-4o"
    prompt:
      system_role: |
        You are a helpful assistant who explains concepts clearly.

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
```

**That's it!** Start the server and you have a working AI agent with streaming, sessions, and A2A protocol compliance.

---

## 🔗 Quick Links

- [🏠 Project Homepage]({{ site.hector.repo_url }})
- [📊 A2A Protocol Specification](https://a2a-protocol.org)
- [🐛 Report Issues]({{ site.hector.repo_url }}/issues)
- [🤝 Contributing Guide](/CONTRIBUTING)

---

<div style="text-align: center; margin: 2rem 0;">
  <img src="hector-gopher.png" alt="Hector Gopher Mascot" style="width: 80px; height: auto;">
  <p><em>Meet Hector, your AI agent companion!</em></p>
</div>
