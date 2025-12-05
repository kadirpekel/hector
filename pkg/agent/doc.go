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

// Package agent defines the core agent interfaces and types for Hector v2.
//
// # Agent Interface
//
// The Agent interface is the fundamental abstraction for all agents:
//
//	type Agent interface {
//	    Name() string
//	    Description() string
//	    Run(InvocationContext) iter.Seq2[*Event, error]
//	    SubAgents() []Agent
//	}
//
// # Context Hierarchy
//
// The package provides a three-tier context hierarchy:
//
//   - InvocationContext: Full access during agent execution
//   - ReadonlyContext: Read-only access for tools and callbacks
//   - CallbackContext: State modification for callbacks
//
// # Creating Agents
//
// Use the provided constructors to create agents:
//
//	agent, err := agent.New(agent.Config{
//	    Name:        "my-agent",
//	    Description: "A helpful assistant",
//	    Run:         myRunFunc,
//	})
//
// For LLM-based agents, use the llmagent subpackage:
//
//	agent, err := llmagent.New(llmagent.Config{
//	    Name:        "llm-agent",
//	    Model:       myModel,
//	    Instruction: "You are a helpful assistant.",
//	})
package agent
