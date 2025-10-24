package reasoning

import (
	"context"
	"testing"
)

func TestExtractGoals_EmptyQuery(t *testing.T) {
	services := &mockAgentServices{}

	decomposition, err := ExtractGoals(
		context.Background(),
		"",
		[]string{},
		services,
	)

	if err != nil {
		t.Errorf("ExtractGoals() should not return error for empty query, got: %v", err)
	}

	if decomposition == nil {
		t.Fatal("ExtractGoals() returned nil decomposition")
	}

	if len(decomposition.Subtasks) == 0 {
		t.Error("ExtractGoals() returned no subtasks")
	}
}

func TestExtractGoals_WithAgents(t *testing.T) {
	services := &mockAgentServices{}
	availableAgents := []string{"weather_agent", "email_agent", "calculator_agent"}

	decomposition, err := ExtractGoals(
		context.Background(),
		"Check the weather and send an email",
		availableAgents,
		services,
	)

	if err != nil {
		t.Errorf("ExtractGoals() error = %v", err)
	}

	if decomposition == nil {
		t.Fatal("ExtractGoals() returned nil")
	}

	if decomposition.MainGoal == "" {
		t.Error("MainGoal should not be empty")
	}

	if len(decomposition.Subtasks) == 0 {
		t.Error("Subtasks should not be empty")
	}

	if len(decomposition.RequiredAgents) == 0 {
		t.Error("RequiredAgents should not be empty")
	}

	validOrders := map[string]bool{
		"sequential":   true,
		"parallel":     true,
		"hierarchical": true,
	}
	if !validOrders[decomposition.ExecutionOrder] {
		t.Errorf("ExecutionOrder = %s, should be one of [sequential, parallel, hierarchical]", decomposition.ExecutionOrder)
	}
}

func TestExtractGoals_SubtaskValidation(t *testing.T) {
	services := &mockAgentServices{}

	decomposition, err := ExtractGoals(
		context.Background(),
		"Complex task with multiple steps",
		[]string{"agent1", "agent2"},
		services,
	)

	if err != nil {
		t.Errorf("ExtractGoals() error = %v", err)
	}

	if decomposition == nil || len(decomposition.Subtasks) == 0 {
		t.Fatal("Expected at least one subtask")
	}

	for i, subtask := range decomposition.Subtasks {
		if subtask.ID == "" {
			t.Errorf("Subtask %d has empty ID", i)
		}

		if subtask.Description == "" {
			t.Errorf("Subtask %d has empty Description", i)
		}

		if subtask.AgentType == "" {
			t.Errorf("Subtask %d has empty AgentType", i)
		}

		if subtask.Priority < 1 || subtask.Priority > 5 {
			t.Errorf("Subtask %d has invalid Priority = %d, should be 1-5", i, subtask.Priority)
		}

		if subtask.DependsOn == nil {
			t.Errorf("Subtask %d has nil DependsOn", i)
		}
	}
}

func TestBuildGoalExtractionPrompt(t *testing.T) {
	query := "Test query"
	agents := []string{"agent1", "agent2"}

	prompt := buildGoalExtractionPrompt(query, agents)

	if prompt == "" {
		t.Error("buildGoalExtractionPrompt() returned empty string")
	}

	if len(prompt) < 100 {
		t.Errorf("buildGoalExtractionPrompt() returned suspiciously short prompt: %s", prompt)
	}
}

func TestBuildGoalExtractionPrompt_NoAgents(t *testing.T) {
	query := "Test query"
	agents := []string{}

	prompt := buildGoalExtractionPrompt(query, agents)

	if prompt == "" {
		t.Error("buildGoalExtractionPrompt() returned empty string")
	}

	if len(prompt) < 50 {
		t.Error("buildGoalExtractionPrompt() returned too short prompt")
	}
}

func TestExtractGoals_FallbackScenarios(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "simple task",
			query: "Calculate 2+2",
		},
		{
			name:  "complex task",
			query: "Research competitors, analyze market, prepare report",
		},
		{
			name:  "vague task",
			query: "Help me with something",
		},
	}

	services := &mockAgentServices{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decomposition, err := ExtractGoals(
				context.Background(),
				tt.query,
				[]string{},
				services,
			)

			if err != nil {
				t.Errorf("ExtractGoals() error = %v", err)
			}

			if decomposition == nil {
				t.Fatal("ExtractGoals() returned nil")
			}

			if len(decomposition.Subtasks) == 0 {
				t.Error("Fallback should still produce at least one subtask")
			}

			if decomposition.MainGoal == "" {
				t.Error("Fallback should still have MainGoal")
			}
		})
	}
}
