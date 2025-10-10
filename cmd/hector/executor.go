package main

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/pkg/a2a"
	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
)

// ============================================================================
// DIRECT EXECUTOR - IN-PROCESS AGENT EXECUTION
// ============================================================================

// DirectExecutor handles direct (non-server) agent execution
type DirectExecutor struct {
	config           *config.Config
	componentManager *component.ComponentManager
}

// NewDirectExecutor creates a new direct executor
func NewDirectExecutor(cfg *config.Config) (*DirectExecutor, error) {
	// Create component manager
	componentManager, err := component.NewComponentManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	return &DirectExecutor{
		config:           cfg,
		componentManager: componentManager,
	}, nil
}

// ExecuteTask executes a task directly on an agent without A2A server
func (e *DirectExecutor) ExecuteTask(ctx context.Context, agentName string, prompt string, stream bool) error {
	// Get agent config
	agentConfig, exists := e.config.Agents[agentName]
	if !exists {
		return fmt.Errorf("agent '%s' not found in configuration", agentName)
	}

	// Create agent instance
	agentInstance, err := agent.NewAgent(&agentConfig, e.componentManager)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	if stream {
		return e.executeStreaming(ctx, agentInstance, prompt)
	}
	return e.executeNonStreaming(ctx, agentInstance, prompt)
}

// executeStreaming executes task with streaming output
func (e *DirectExecutor) executeStreaming(ctx context.Context, agentInstance *agent.Agent, prompt string) error {
	// Create task from prompt
	task := e.createTask(prompt)

	// Execute with streaming
	streamCh, err := agentInstance.ExecuteTaskStreaming(ctx, task)
	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	// Stream output to stdout
	for event := range streamCh {
		if event.Message != nil {
			// Extract text from message parts
			for _, part := range event.Message.Parts {
				if part.Type == a2a.PartTypeText {
					fmt.Print(part.Text)
				}
			}
		}
		// Check for errors in status
		if event.Status != nil && event.Status.State == a2a.TaskStateFailed {
			return fmt.Errorf("task failed")
		}
	}
	fmt.Println() // Final newline

	return nil
}

// executeNonStreaming executes task and returns complete response
func (e *DirectExecutor) executeNonStreaming(ctx context.Context, agentInstance *agent.Agent, prompt string) error {
	// Create task from prompt
	task := e.createTask(prompt)

	// Execute task
	result, err := agentInstance.ExecuteTask(ctx, task)
	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	// Extract and print response
	response := e.extractResponse(result)
	fmt.Println(response)

	return nil
}

// createTask creates an a2a.Task from a simple prompt string
func (e *DirectExecutor) createTask(prompt string) *a2a.Task {
	return &a2a.Task{
		Messages: []a2a.Message{
			{
				Role: a2a.MessageRoleUser,
				Parts: []a2a.Part{
					{
						Type: a2a.PartTypeText,
						Text: prompt,
					},
				},
			},
		},
	}
}

// extractResponse extracts text response from completed task
func (e *DirectExecutor) extractResponse(task *a2a.Task) string {
	if len(task.Messages) == 0 {
		return ""
	}

	// Get last message (should be assistant's response)
	lastMsg := task.Messages[len(task.Messages)-1]
	if lastMsg.Role != a2a.MessageRoleAssistant {
		return ""
	}

	// Collect text from all parts
	var text string
	for _, part := range lastMsg.Parts {
		if part.Type == a2a.PartTypeText {
			text += part.Text
		}
	}

	return text
}

// Chat starts an interactive chat session
func (e *DirectExecutor) Chat(ctx context.Context, agentName string) error {
	// Get agent config
	agentConfig, exists := e.config.Agents[agentName]
	if !exists {
		return fmt.Errorf("agent '%s' not found in configuration", agentName)
	}

	// Create agent instance
	agentInstance, err := agent.NewAgent(&agentConfig, e.componentManager)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// Start interactive chat (reuse existing chat logic)
	return startDirectChat(ctx, agentInstance, agentName)
}

// ListAgents lists available agents from configuration
func (e *DirectExecutor) ListAgents() error {
	fmt.Printf("\n📋 Available Agents (Direct Mode):\n\n")

	for agentID, agentConfig := range e.config.Agents {
		fmt.Printf("  🤖 %s\n", agentConfig.Name)
		fmt.Printf("     ID: %s\n", agentID)
		if agentConfig.Description != "" {
			fmt.Printf("     Description: %s\n", agentConfig.Description)
		}
		fmt.Printf("     LLM: %s\n", agentConfig.LLM)
		fmt.Printf("     Type: %s\n", agentConfig.Type)
		fmt.Println()
	}

	return nil
}

// GetAgentInfo returns detailed information about an agent
func (e *DirectExecutor) GetAgentInfo(agentName string) error {
	agentConfig, exists := e.config.Agents[agentName]
	if !exists {
		return fmt.Errorf("agent '%s' not found in configuration", agentName)
	}

	fmt.Printf("\n🤖 Agent: %s\n", agentConfig.Name)
	fmt.Printf("   ID: %s\n", agentName)
	if agentConfig.Description != "" {
		fmt.Printf("   Description: %s\n", agentConfig.Description)
	}
	fmt.Printf("   Type: %s\n", agentConfig.Type)
	fmt.Printf("   LLM: %s\n", agentConfig.LLM)
	if len(agentConfig.Tools) > 0 {
		fmt.Printf("   Tools: %v\n", agentConfig.Tools)
	}
	fmt.Printf("   Reasoning: %s (max %d iterations)\n",
		agentConfig.Reasoning.Engine,
		agentConfig.Reasoning.MaxIterations)
	fmt.Println()

	return nil
}
