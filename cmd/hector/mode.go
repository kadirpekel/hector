package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/config"
)

// ============================================================================
// MODE DETECTION AND ROUTING
// ============================================================================

// ExecutionMode determines how commands should be executed
type ExecutionMode int

const (
	ModeDirect ExecutionMode = iota // Direct in-process execution
	ModeServer                      // A2A server protocol
)

// detectMode determines execution mode based on CLI arguments
func detectMode(args *CLIArgs) ExecutionMode {
	if args.ServerURL != "" {
		return ModeServer
	}
	return ModeDirect
}

// ============================================================================
// UNIFIED COMMAND HANDLERS
// ============================================================================

// executeCallCommand handles both direct and server modes
func executeCallCommand(args *CLIArgs, agentID string, prompt string) error {
	mode := detectMode(args)

	switch mode {
	case ModeDirect:
		return executeCallDirect(args, agentID, prompt)
	case ModeServer:
		serverURL := resolveServerURL(args.ServerURL)
		agentURL := buildAgentURL(serverURL, agentID)
		return executeCallServer(agentURL, prompt, args.Token, args.Stream)
	}

	return fmt.Errorf("unknown execution mode")
}

// executeChatCommand handles both direct and server modes
func executeChatCommand(args *CLIArgs, agentID string) error {
	mode := detectMode(args)

	switch mode {
	case ModeDirect:
		return executeChatDirect(args, agentID)
	case ModeServer:
		serverURL := resolveServerURL(args.ServerURL)
		agentURL := buildAgentURL(serverURL, agentID)
		return executeChatServer(agentURL, args.Token)
	}

	return fmt.Errorf("unknown execution mode")
}

// executeListCommand handles both direct and server modes
func executeListCommand(args *CLIArgs) error {
	mode := detectMode(args)

	switch mode {
	case ModeDirect:
		return executeListDirect(args)
	case ModeServer:
		serverURL := resolveServerURL(args.ServerURL)
		return executeListServer(serverURL, args.Token)
	}

	return fmt.Errorf("unknown execution mode")
}

// executeInfoCommand handles both direct and server modes
func executeInfoCommand(args *CLIArgs, agentID string) error {
	mode := detectMode(args)

	switch mode {
	case ModeDirect:
		return executeInfoDirect(args, agentID)
	case ModeServer:
		serverURL := resolveServerURL(args.ServerURL)
		agentURL := buildAgentURL(serverURL, agentID)
		return executeInfoServer(agentURL, args.Token)
	}

	return fmt.Errorf("unknown execution mode")
}

// ============================================================================
// DIRECT MODE IMPLEMENTATIONS
// ============================================================================

// executeCallDirect executes a call in direct mode
func executeCallDirect(args *CLIArgs, agentID string, prompt string) error {
	// Load or create configuration
	cfg, err := loadOrCreateConfig(args)
	if err != nil {
		return err
	}

	// Create direct executor
	executor, err := NewDirectExecutor(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize direct executor: %w", err)
	}

	// Execute task
	return executor.ExecuteTask(context.Background(), agentID, prompt, args.Stream)
}

// executeChatDirect executes chat in direct mode
func executeChatDirect(args *CLIArgs, agentID string) error {
	// Load or create configuration
	cfg, err := loadOrCreateConfig(args)
	if err != nil {
		return err
	}

	// Create direct executor
	executor, err := NewDirectExecutor(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize direct executor: %w", err)
	}

	// Start chat
	return executor.Chat(context.Background(), agentID)
}

// executeListDirect lists agents in direct mode
func executeListDirect(args *CLIArgs) error {
	// Load or create configuration
	cfg, err := loadOrCreateConfig(args)
	if err != nil {
		return err
	}

	// Create direct executor
	executor, err := NewDirectExecutor(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize direct executor: %w", err)
	}

	return executor.ListAgents()
}

// executeInfoDirect shows agent info in direct mode
func executeInfoDirect(args *CLIArgs, agentID string) error {
	// Load or create configuration
	cfg, err := loadOrCreateConfig(args)
	if err != nil {
		return err
	}

	// Create direct executor
	executor, err := NewDirectExecutor(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize direct executor: %w", err)
	}

	return executor.GetAgentInfo(agentID)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// buildAgentURL constructs agent URL from server URL and agent ID
func buildAgentURL(serverURL, agentID string) string {
	// If agentID is already a full URL, return as-is
	if strings.HasPrefix(agentID, "http://") || strings.HasPrefix(agentID, "https://") {
		return agentID
	}

	// Otherwise, construct from server URL + agent ID
	serverURL = strings.TrimSuffix(serverURL, "/")
	return fmt.Sprintf("%s/agents/%s", serverURL, agentID)
}

// ============================================================================
// CONFIGURATION LOADING
// ============================================================================

// loadOrCreateConfig loads config from file or creates zero-config
// Uses unified config loading from config_loader.go
func loadOrCreateConfig(args *CLIArgs) (*config.Config, error) {
	return loadConfigFromArgsOrFile(args, true)
}
