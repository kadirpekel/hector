# Changelog

All notable changes to the Hector project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

#### Smart Defaults: Structured Reflection Enabled by Default

Agents now use LLM-based structured reflection by default for better quality.

**Impact:**
- +13% quality improvement
- +20% token usage
- Better error recovery

**Configuration:**
```yaml
reasoning:
  enable_structured_reflection: true  # Default (disable if cost-sensitive)
```

**Related Features (opt-in):**
- `enable_completion_verification` - Task completion verification
- `enable_goal_extraction` - Goal decomposition for supervisor

See `docs/CONFIGURATION.md` for details.

### Added

- **Structured Reflection** - LLM-based tool execution analysis with confidence scores (default: enabled)
- **Completion Verification** - Task completion verification before stopping (default: disabled, opt-in)
- **Goal Extraction** - Task decomposition for supervisor strategy (default: disabled, opt-in)
- **Google Gemini Provider** - Full support for Gemini models with structured output
- **Structured Output** - Provider-agnostic interface for JSON, XML, and enum output across OpenAI, Anthropic, and Gemini
- **Benchmarking Framework** - Comprehensive testing lab in `docs/benchmarks/` for performance and behavioral testing
- **Multi-Provider Testing** - Automated benchmarking across OpenAI, Anthropic, and Gemini
- **MIGRATION.md** - Migration guide for smart defaults update
- **Cost Analysis Documentation** - Detailed cost/benefit analysis in `docs/CONFIGURATION.md`

### Fixed

- Configuration defaults now properly set `EnableStructuredReflection` to `true`
- Documentation updated to reflect smart defaults

---

## [Previous Releases]

### A2A Compliance Update

- **100% A2A Specification Compliance** - Full implementation of A2A protocol v1.0
- **SSE Streaming** - Server-Sent Events for real-time output (replaced WebSocket)
- **A2A Endpoints** - Complete support for `message/send`, `message/stream`, `tasks/get`, `tasks/cancel`, `tasks/resubscribe`
- **Agent Cards** - Full A2A AgentCard support with capabilities, authentication, and endpoints
- **Session Management** - Multi-turn conversations with session create/get/delete/list
- **External Agent Integration** - Seamless orchestration with external A2A agents

### Initial Release

- **Core Agent Framework** - Declarative agent configuration
- **Reasoning Strategies** - Chain-of-thought and supervisor strategies
- **Built-in Tools** - Command execution, file operations, search, todos
- **LLM Providers** - OpenAI and Anthropic support
- **Plugin System** - gRPC-based extensibility
- **Document Stores** - RAG with Qdrant
- **CLI & Server** - Complete command-line interface and HTTP server

---

## Versioning

This project uses [Semantic Versioning](https://semver.org/):
- **MAJOR** - Breaking changes
- **MINOR** - New features (backwards compatible)
- **PATCH** - Bug fixes (backwards compatible)

Current focus: Alpha stage, working toward v1.0.0

