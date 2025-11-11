package agent

import (
	"testing"
)

// TestNewAgent is REMOVED - agent.NewAgent() was removed in favor of programmatic API
// Use hector.NewAgent().Build() or hector.NewConfigAgentBuilder() instead
func TestNewAgent(t *testing.T) {
	t.Skip("agent.NewAgent() was removed - use hector.NewAgent().Build() or hector.NewConfigAgentBuilder() instead")
}

func TestAgent_GetAgentCard(t *testing.T) {

	t.Skip("Skipping test that requires full component manager setup")
}

func TestAgent_ExecuteTask(t *testing.T) {

	t.Skip("Skipping test that requires full component manager setup")
}

func TestAgent_ExecuteTaskStreaming(t *testing.T) {

	t.Skip("Skipping test that requires full component manager setup")
}

func TestAgent_ClearHistory(t *testing.T) {

	t.Skip("Skipping test that requires full component manager setup")
}

func TestAgent_GetName(t *testing.T) {

	t.Skip("Skipping test that requires full component manager setup")
}

func TestAgent_GetDescription(t *testing.T) {

	t.Skip("Skipping test that requires full component manager setup")
}

func TestAgent_GetConfig(t *testing.T) {

	t.Skip("Skipping test that requires full component manager setup")
}

type MockComponentManager struct{}
