# Hector Documentation

Complete documentation for Hector - Pure A2A-Native AI Agent Platform

---

## Getting Started

**[Quick Start Guide →](QUICK_START.md)**

Get up and running in 5 minutes. Install, configure, and start using Hector.

---

## Core Documentation

### [Building Agents](AGENTS.md) ← Start Here
Complete guide to building single agents: prompts, tools, RAG, sessions, streaming, and more.

### [Tools & Extensions](TOOLS.md)
Comprehensive guide to Hector's tool system: built-in tools, MCP protocol, gRPC plugins, and custom integrations.

### [Custom MCP Tools](MCP_CUSTOM_TOOLS.md) 🔥
Build custom tools in 5 minutes with Python or TypeScript. Real-world examples, best practices, and deployment guide.

### [API Reference](API_REFERENCE.md)
Complete A2A protocol API reference: HTTP endpoints, SSE streaming, request/response schemas, authentication, and client examples.

### [A2A Protocol Compliance](A2A_COMPLIANCE.md)
Complete A2A specification compliance documentation with spec section references and implementation details.

### [Architecture](ARCHITECTURE.md)
System design, A2A protocol implementation, sessions, orchestration patterns, and core components.

### [Configuration](CONFIGURATION.md)
Complete configuration reference with examples and all available options.

### [CLI Guide](CLI_GUIDE.md)
Command-line interface reference for all `hector` commands.

### [Testing Guide](TESTING.md)
Comprehensive testing practices, strategies, and tools for Hector development.

---

## Multi-Agent Features

### [External Agents](EXTERNAL_AGENTS.md)
How to integrate external A2A agents with Hector. Discover, register, and orchestrate remote agents transparently.

### [Multi-Agent Orchestration](ARCHITECTURE.md#orchestrator-pattern)
Multi-agent orchestration patterns. Use the `agent_call` tool to coordinate multiple agents for complex tasks.

### [Authentication](AUTHENTICATION.md)
Securing Hector with JWT token validation. Works with any OAuth2/OIDC provider (Auth0, Keycloak, etc.).

---

## Quick Navigation

### I want to...

**Build my first agent**
→ [Building Agents Guide](AGENTS.md)

**Get started quickly**
→ [Quick Start](QUICK_START.md)

**Add tools & integrations**
→ [Tools & Extensions Guide](TOOLS.md)

**Build custom tools quickly**
→ [Custom MCP Tools Guide](MCP_CUSTOM_TOOLS.md) 🔥

**Connect to 150+ apps (MCP)**
→ [Tools - MCP Integration](TOOLS.md#mcp-integration)

**Integrate via A2A protocol**
→ [API Reference](API_REFERENCE.md)

**Verify A2A compliance**
→ [A2A Protocol Compliance](A2A_COMPLIANCE.md)

**Customize prompts and tools**
→ [Building Agents Guide](AGENTS.md)

**Add RAG/semantic search**
→ [Building Agents - RAG Section](AGENTS.md#document-stores--rag)

**Use sessions & streaming**
→ [Building Agents - Sessions](AGENTS.md#sessions--streaming)

**Configure an agent**
→ [Configuration Reference](CONFIGURATION.md)

**Orchestrate multiple agents**
→ [Orchestration Patterns](ARCHITECTURE.md#orchestrator-pattern)

**Use external agents**
→ [External Agents](EXTERNAL_AGENTS.md)

**Secure with authentication**
→ [Authentication](AUTHENTICATION.md)

**Use the CLI**
→ [CLI Guide](CLI_GUIDE.md)

**Understand the system**
→ [Architecture](ARCHITECTURE.md)

**Contribute to the project**
→ [Contributing Guide](CONTRIBUTING.md)

**Write tests**
→ [Testing Guide](TESTING.md)

---

## Documentation Index

| Document | Purpose |
|----------|---------|
| [Building Agents](AGENTS.md) | Complete single-agent guide (Start here) |
| [Tools & Extensions](TOOLS.md) | Built-in tools, MCP, plugins |
| [Custom MCP Tools](MCP_CUSTOM_TOOLS.md) | Build custom tools in 5 minutes 🔥 |
| [Plugins](PLUGINS.md) | Plugin development guide (gRPC extensions) |
| [API Reference](API_REFERENCE.md) | A2A protocol HTTP+JSON/SSE API |
| [A2A Compliance](A2A_COMPLIANCE.md) | A2A specification compliance details |
| [Quick Start](QUICK_START.md) | Get started in 5 minutes |
| [Architecture](ARCHITECTURE.md) | System design, sessions, orchestration |
| [Configuration](CONFIGURATION.md) | Complete config reference |
| [CLI Guide](CLI_GUIDE.md) | Command-line interface |
| [External Agents](EXTERNAL_AGENTS.md) | External A2A agent integration |
| [Authentication](AUTHENTICATION.md) | JWT token validation |
| [Contributing](CONTRIBUTING.md) | How to contribute to Hector |
| [Testing](TESTING.md) | Testing practices and guidelines |

---

## External Resources

- [A2A Protocol Specification](https://a2a-protocol.org) - Official specification
- [GitHub Repository](https://github.com/kadirpekel/hector) - Source code
- [Main README](../README.md) - Project overview

---

## Tips

- Start with [Building Agents](AGENTS.md) to understand core capabilities
- Use [Quick Start](QUICK_START.md) for a 5-minute setup
- Check [Configuration](CONFIGURATION.md) for all YAML options
- Learn [orchestration patterns](ARCHITECTURE.md#orchestrator-pattern) for multi-agent systems
- See [External Agents](EXTERNAL_AGENTS.md) for integration examples
- Secure with [Authentication](AUTHENTICATION.md) for enterprise deployments

---

**Documentation Version:** 1.0  
**Last Updated:** October 2025

## 📊 Benchmarks

The `benchmarks/` directory contains a comprehensive testing lab for validating structured output features across multiple LLM providers.

**Quick Start:**
```bash
cd docs/benchmarks
./run_all_benchmarks.sh
```

**Documentation:**
- [Benchmarks README](benchmarks/README.md) - Overview and quick start
- [Results Interpretation](benchmarks/RESULTS_INTERPRETATION.md) - Understanding results
- [Executive Summary](benchmarks/EXECUTIVE_SUMMARY.md) - Business insights

**What it tests:**
- 3 structured output features (Reflection, Completion, Goals)
- 3 LLM providers (OpenAI, Anthropic, Gemini)
- 240 total tests (performance + behavioral)
- Expected: 10-20% quality improvement, 15-37% cost increase
