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
// memory strategies, and other components programmatically. The builders wrap
// the underlying Config structs, providing the best of both worlds:
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
//	            Build(),
//	    ).
//	    WithReasoning(
//	        builder.NewReasoning().
//	            MaxIterations(100).
//	            EnableExitTool(true).
//	            Build(),
//	    ).
//	    WithWorkingMemory(
//	        builder.NewWorkingMemory("summary_buffer").
//	            Budget(8000).
//	            Threshold(0.85).
//	            Build(),
//	    ).
//	    WithTools(tool1, tool2).
//	    Build()
//
// # Architecture
//
// The builder package is a convenience layer over the core v2 packages:
//
//	┌─────────────────────────────────────────────────────────────┐
//	│         Builder Package (Convenience Layer)                  │
//	│  Fluent API for ergonomic programmatic construction         │
//	│                                                              │
//	│  AgentBuilder → llmagent.Config → llmagent.New()            │
//	│  LLMBuilder → model.LLM                                      │
//	│  ReasoningBuilder → llmagent.ReasoningConfig                 │
//	│  MemoryBuilder → memory.WorkingMemoryStrategy                │
//	└─────────────────────────────────────────────────────────────┘
//	                            ▲
//	                            │ wraps
//	                            │
//	┌─────────────────────────────────────────────────────────────┐
//	│         Core V2 Packages (Foundation)                        │
//	│  Config structs aligned with ADK-Go                          │
//	│                                                              │
//	│  llmagent.Config, model.LLM, memory.WorkingMemoryStrategy   │
//	└─────────────────────────────────────────────────────────────┘
//
// # Available Builders
//
//   - [AgentBuilder]: Build LLM agents with fluent API
//   - [LLMBuilder]: Build LLM providers (OpenAI, Anthropic, Gemini, Ollama)
//   - [ReasoningBuilder]: Configure chain-of-thought reasoning
//   - [WorkingMemoryBuilder]: Configure working memory strategies
//   - [LongTermMemoryBuilder]: Configure long-term memory
//   - [CredentialsBuilder]: Configure authentication credentials
//   - [SecurityBuilder]: Configure security schemes
//
// # Example: Multi-Agent System
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
package builder
