// Package pkg is Hector, built natively on the A2A (Agent-to-Agent) protocol
// using the a2a-go library.
//
// # Architecture Overview
//
// Hector follows a clean, interface-driven architecture inspired by Google's
// ADK-Go, with the following core concepts:
//
//   - Agent: The fundamental unit of execution, implementing the Agent interface
//   - Session: Manages conversation state and history
//   - Tool/Toolset: Capabilities that agents can invoke
//   - Runner: Orchestrates agent execution within sessions
//   - Server: Exposes agents via A2A protocol (JSON-RPC, gRPC, HTTP)
//
// # Key Design Principles
//
//   - Native A2A: Uses github.com/a2aproject/a2a-go directly, no custom protobuf
//   - Interface-First: All core components are defined as interfaces for testability
//   - Iterator Pattern: Uses Go 1.23+ iter.Seq2 for clean event streaming
//   - Context Hierarchy: Clear separation of read-only vs mutable context
//   - Lazy Loading: Toolsets connect to external services on first use
package pkg
