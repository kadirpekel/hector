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
