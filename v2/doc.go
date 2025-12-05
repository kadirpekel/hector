// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package v2 is the next generation of Hector, built natively on the A2A
// (Agent-to-Agent) protocol using the a2a-go library.
//
// # Architecture Overview
//
// Hector v2 follows a clean, interface-driven architecture inspired by
// Google's ADK-Go, with the following core concepts:
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
//
// # Migration
//
// The v2 package is designed to coexist with the legacy pkg/ during migration.
// See the migration guide for transitioning existing agents to v2.
package v2
