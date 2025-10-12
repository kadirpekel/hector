// Package hector provides a pure A2A-native declarative AI agent platform.
//
// This is the main entry point for the Hector Go library. It re-exports
// the most commonly used types and functions from the various sub-packages.
//
// # Quick Start
//
//	import "github.com/kadirpekel/hector/pkg"
//
//	// Create agent registry
//	registry := hector.NewAgentRegistry()
//
//	// Load configuration
//	cfg, err := hector.LoadConfig("config.yaml")
//
//	// Create and register agent
//	agent, err := hector.NewAgent(cfg.Agents["my_agent"], cfg)
//	registry.RegisterAgent("my_agent", agent, cfg, capabilities)
//
//	// Start A2A server
//	server := hector.NewA2AServer(registry, &hector.ServerConfig{Port: 8080})
//	server.Start()
package hector

import (
	// Re-export commonly used types and functions
	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/config"
)

// Re-export commonly used types
type (
	// Agent types
	Agent         = agent.Agent
	AgentConfig   = config.AgentConfig
	AgentRegistry = agent.AgentRegistry

	// Config is the main Hector configuration
	Config = config.Config
)

// Re-export commonly used functions
var (
	// Agent functions
	NewAgent         = agent.NewAgent
	NewAgentRegistry = agent.NewAgentRegistry

	// Config functions
	LoadConfig     = config.LoadConfig
	LoadConfigFrom = config.LoadConfigFromString
)
