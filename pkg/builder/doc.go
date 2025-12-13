// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel
//
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.gnu.org/licenses/agpl-3.0.en.html
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package builder provides fluent builder APIs for programmatic agent construction.
//
// This package provides an ergonomic, chainable API for building agents, LLMs,
// memory strategies, tools, RAG systems, and other components programmatically.
// The builders wrap the underlying Config structs, providing the best of both worlds:
//
//   - Fluent, discoverable API for programmatic use
//   - Config structs remain available for direct use (ADK-Go aligned)
//
// # Quick Start
//
// Build a complete agent with LLM, reasoning, and tools:
//
//	agent, err := builder.NewAgent("assistant").
//	    WithName("Assistant").
//	    WithDescription("A helpful AI assistant").
//	    WithLLM(
//	        builder.NewLLM("openai").
//	            Model("gpt-4o-mini").
//	            APIKeyFromEnv("OPENAI_API_KEY").
//	            Temperature(0.7).
//	            MustBuild(),
//	    ).
//	    WithReasoning(
//	        builder.NewReasoning().
//	            MaxIterations(100).
//	            EnableExitTool(true).
//	            Build(),
//	    ).
//	    WithTools(weatherTool, searchTool).
//	    Build()
//
// # Architecture
//
// The builder package is a convenience layer over the core packages:
//
//	┌─────────────────────────────────────────────────────────────────┐
//	│         Builder Package (Convenience Layer)                      │
//	│  Fluent API for ergonomic programmatic construction             │
//	│                                                                  │
//	│  Agents:                                                         │
//	│    AgentBuilder → llmagent.Config → llmagent.New()              │
//	│    ReasoningBuilder → llmagent.ReasoningConfig                   │
//	│                                                                  │
//	│  LLMs & Embeddings:                                              │
//	│    LLMBuilder → model.LLM (OpenAI, Anthropic, Gemini, Ollama)   │
//	│    EmbedderBuilder → embedder.Embedder                           │
//	│                                                                  │
//	│  Tools:                                                          │
//	│    MCPBuilder → mcptoolset.Toolset (MCP servers)                 │
//	│    ToolsetBuilder → tool.Toolset (wrap tools)                    │
//	│    FunctionTool → tool.CallableTool (from Go functions)          │
//	│                                                                  │
//	│  Memory:                                                         │
//	│    WorkingMemoryBuilder → memory.WorkingMemoryStrategy           │
//	│                                                                  │
//	│  RAG (Retrieval-Augmented Generation):                           │
//	│    DocumentStoreBuilder → rag.DocumentStore                      │
//	│    VectorProviderBuilder → vector.Provider                       │
//	│                                                                  │
//	│  Runtime:                                                        │
//	│    RunnerBuilder → runner.Runner                                 │
//	│    ServerBuilder → A2A HTTP server                               │
//	└─────────────────────────────────────────────────────────────────┘
//
// # Available Builders
//
// Core:
//   - [AgentBuilder]: Build LLM agents with fluent API
//   - [LLMBuilder]: Build LLM providers (OpenAI, Anthropic, Gemini, Ollama)
//   - [ReasoningBuilder]: Configure chain-of-thought reasoning
//   - [WorkingMemoryBuilder]: Configure working memory strategies
//
// Tools:
//   - [MCPBuilder]: Connect to MCP (Model Context Protocol) servers
//   - [ToolsetBuilder]: Wrap tools into a toolset
//   - [FunctionTool]: Create tools from typed Go functions
//
// RAG:
//   - [DocumentStoreBuilder]: Build document stores for RAG
//   - [EmbedderBuilder]: Build embedding providers
//   - [VectorProviderBuilder]: Build vector database providers
//
// Runtime:
//   - [RunnerBuilder]: Build agent runners
//   - [ServerBuilder]: Build A2A HTTP servers
//
// # Example: Multi-Agent System
//
//	// Build LLM
//	llm := builder.NewLLM("openai").
//	    Model("gpt-4o").
//	    APIKeyFromEnv("OPENAI_API_KEY").
//	    MustBuild()
//
//	// Build specialized agents
//	researcher, _ := builder.NewAgent("researcher").
//	    WithDescription("Researches topics in depth").
//	    WithLLM(llm).
//	    Build()
//
//	writer, _ := builder.NewAgent("writer").
//	    WithDescription("Writes content based on research").
//	    WithLLM(llm).
//	    Build()
//
//	// Build parent agent with sub-agents
//	parent, _ := builder.NewAgent("coordinator").
//	    WithDescription("Coordinates research and writing").
//	    WithLLM(llm).
//	    WithSubAgents(researcher, writer).
//	    Build()
//
// # Example: RAG with Document Store
//
//	// Build embedder
//	emb := builder.NewEmbedder("openai").
//	    Model("text-embedding-3-small").
//	    MustBuild()
//
//	// Build vector store
//	vecStore := builder.NewVectorProvider("chromem").
//	    PersistPath(".hector/vectors").
//	    MustBuild()
//
//	// Build document store
//	docStore, _ := builder.NewDocumentStore("docs").
//	    FromDirectory("./documents").
//	    WithVectorProvider(vecStore).
//	    WithEmbedder(emb).
//	    EnableWatching(true).
//	    Build()
//
// # Example: Custom Function Tool
//
//	type WeatherArgs struct {
//	    City string `json:"city" jsonschema:"required,description=City name"`
//	}
//
//	weatherTool, _ := builder.FunctionTool(
//	    "get_weather",
//	    "Get current weather for a city",
//	    func(ctx tool.Context, args WeatherArgs) (map[string]any, error) {
//	        return map[string]any{"temp": 22, "city": args.City}, nil
//	    },
//	)
//
// # Example: MCP Tool Integration
//
//	// SSE transport
//	mcpTools, _ := builder.NewMCP("weather").
//	    URL("http://localhost:9000").
//	    Build()
//
//	// Stdio transport
//	fsTool, _ := builder.NewMCP("filesystem").
//	    Command("npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp").
//	    Build()
package builder
