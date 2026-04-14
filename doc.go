// Package hector is an open-source, self-sovereign AI agent runtime for Go.
//
// One self-contained binary with production-ready defaults. Deploy on-premise,
// in air-gapped environments, or in any cloud without external dependencies.
// Define agents in YAML or Go, and keep every byte of data under your control.
// Built on open standards (A2A, MCP) with SQL-backed persistence, multi-agent
// orchestration, RAG, guardrails, and a built-in visual IDE.
//
// # Quick Start (CLI)
//
// Install Hector:
//
//	go install github.com/verikod/hector/cmd/hector@latest
//
// Create a minimal agent configuration:
//
//	# config.yaml
//	llms:
//	  default:
//	    provider: anthropic
//	    model: claude-sonnet-4
//	    api_key: ${ANTHROPIC_API_KEY}
//
//	agents:
//	  assistant:
//	    llm: default
//	    instruction: You are a helpful assistant.
//
// Start the server:
//
//	export ANTHROPIC_API_KEY="sk-ant-..."
//	hector serve --config config.yaml
//
// The server starts at http://localhost:8080. Open it in a browser for Hector
// Studio, or send A2A JSON-RPC requests directly to /agents/assistant.
//
// # Programmatic API
//
// Import the top-level package for a fluent builder API:
//
//	import "github.com/verikod/hector/pkg"
//
// Load agents from YAML:
//
//	h, err := pkg.FromConfig("config.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer h.Close()
//	h.Serve(":8080")
//
// Or build programmatically:
//
//	h, err := pkg.New(
//	    pkg.WithAnthropic(anthropic.Config{APIKey: os.Getenv("ANTHROPIC_API_KEY")}),
//	    pkg.WithInstruction("You are a helpful assistant."),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer h.Close()
//
//	response, err := h.Generate(ctx, "Hello!")
//
// Import specific sub-packages for lower-level control:
//
//	import (
//	    "github.com/verikod/hector/pkg/agent"
//	    "github.com/verikod/hector/pkg/config"
//	    "github.com/verikod/hector/pkg/runner"
//	    "github.com/verikod/hector/pkg/server"
//	    "github.com/verikod/hector/pkg/session"
//	)
//
// # Key Features
//
//   - Config-First: Complete agents defined in YAML, no code required
//   - A2A Protocol: Full A2A v0.3.0 compliance for agent interoperability
//   - Multi-Agent: Sequential, parallel, loop, conditional, and remote agent types
//   - MCP Tools: All MCP transports (stdio, SSE, streamable-HTTP)
//   - RAG Pipeline: Directory, SQL, API, and cloud blob document sources
//   - Guardrails: Deterministic and LLM-powered input/output protection
//   - Auth: JWKS/OIDC integration and simple token auth
//   - Observability: Prometheus metrics and OpenTelemetry tracing
//   - Persistence: SQL-backed sessions, tasks, and checkpoints (SQLite / PostgreSQL)
//
// # Documentation
//
//   - Getting Started: https://gohector.dev/getting-started/quick-start/
//   - Configuration Reference: https://gohector.dev/reference/configuration/
//   - CLI Reference: https://gohector.dev/reference/cli/
//   - API Reference: https://gohector.dev/reference/api/
//   - Go API Reference: https://gohector.dev/reference/go-api/
//   - Source: https://github.com/verikod/hector
//
// # License
//
// MIT. See LICENSE for details.
package hector
