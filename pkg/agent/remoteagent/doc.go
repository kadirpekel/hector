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

// Package remoteagent provides remote A2A agent support.
//
// Remote agents allow communication with agents running in different
// processes or on different hosts using the A2A (Agent-to-Agent) protocol.
//
// # Basic Usage
//
// Create a remote agent from a URL:
//
//	agent, _ := remoteagent.NewA2A(remoteagent.Config{
//	    Name:        "remote_helper",
//	    Description: "A remote helper agent",
//	    URL:         "http://localhost:9000",
//	})
//
// # With Agent Card
//
// Provide an agent card directly:
//
//	agent, _ := remoteagent.NewA2A(remoteagent.Config{
//	    Name:        "remote_helper",
//	    Description: "A remote helper agent",
//	    AgentCard:   &a2a.AgentCard{...},
//	})
//
// # As Sub-Agent
//
// Remote agents can be used as sub-agents:
//
//	parent, _ := llmagent.New(llmagent.Config{
//	    Name:      "orchestrator",
//	    SubAgents: []agent.Agent{remoteAgent},
//	})
//
// # As Tool
//
// Remote agents can be wrapped as tools:
//
//	tool := agenttool.New(remoteAgent, nil)
package remoteagent
