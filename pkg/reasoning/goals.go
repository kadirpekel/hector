package reasoning

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/llms"
)

// ============================================================================
// GOAL EXTRACTION FOR SUPERVISOR STRATEGY
// Uses structured output to decompose tasks into subtasks and identify agents
// ============================================================================

// Subtask represents a single subtask in a decomposed plan
type Subtask struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	AgentType   string   `json:"agent_type"`
	DependsOn   []string `json:"depends_on"`
	Priority    int      `json:"priority"`
}

// TaskDecomposition represents a structured plan for task execution
type TaskDecomposition struct {
	MainGoal       string    `json:"main_goal"`
	Subtasks       []Subtask `json:"subtasks"`
	ExecutionOrder string    `json:"execution_order"` // "sequential", "parallel", "hierarchical"
	RequiredAgents []string  `json:"required_agents"`
	Strategy       string    `json:"strategy"`
	Reasoning      string    `json:"reasoning"`
}

// ExtractGoals uses structured output to decompose a task into subtasks
// This helps supervisor agents plan multi-agent orchestration
func ExtractGoals(
	ctx context.Context,
	userQuery string,
	availableAgents []string,
	services AgentServices,
) (*TaskDecomposition, error) {
	// Build goal extraction prompt
	prompt := buildGoalExtractionPrompt(userQuery, availableAgents)

	// Define structured output schema
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"main_goal": map[string]interface{}{
				"type":        "string",
				"description": "The main objective to accomplish",
			},
			"subtasks": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Unique identifier for this subtask",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "What needs to be done",
						},
						"agent_type": map[string]interface{}{
							"type":        "string",
							"description": "Type of agent needed (or specific agent name if known)",
						},
						"depends_on": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{"type": "string"},
							"description": "List of subtask IDs that must complete first",
						},
						"priority": map[string]interface{}{
							"type":        "integer",
							"description": "Priority level (1=highest, 5=lowest)",
						},
					},
					"required": []string{"id", "description", "agent_type", "depends_on", "priority"},
				},
			},
			"execution_order": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"sequential", "parallel", "hierarchical"},
				"description": "How subtasks should be executed",
			},
			"required_agents": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "List of agent types or names needed",
			},
			"strategy": map[string]interface{}{
				"type":        "string",
				"description": "High-level strategy for task completion",
			},
			"reasoning": map[string]interface{}{
				"type":        "string",
				"description": "Explanation of the decomposition approach",
			},
		},
		"required": []string{
			"main_goal",
			"subtasks",
			"execution_order",
			"required_agents",
			"strategy",
			"reasoning",
		},
	}

	config := &llms.StructuredOutputConfig{
		Format: "json",
		Schema: schema,
	}

	// Get LLM service
	llmService := services.LLM()

	// Check if LLM service is available and supports structured output
	if llmService == nil || !llmService.SupportsStructuredOutput() {
		// Fallback: simple decomposition
		return &TaskDecomposition{
			MainGoal: userQuery,
			Subtasks: []Subtask{
				{
					ID:          "task1",
					Description: userQuery,
					AgentType:   "general",
					DependsOn:   []string{},
					Priority:    1,
				},
			},
			ExecutionOrder: "sequential",
			RequiredAgents: []string{"general"},
			Strategy:       "Single-step execution",
			Reasoning:      "Provider doesn't support structured output; using simple decomposition",
		}, nil
	}

	// Make structured LLM call
	messages := []*pb.Message{
		{Role: pb.Role_ROLE_USER, Content: []*pb.Part{{Part: &pb.Part_Text{Text: prompt}}}},
	}

	text, _, _, err := llmService.GenerateStructured(messages, nil, config)
	if err != nil {
		// Fallback on error
		return &TaskDecomposition{
			MainGoal: userQuery,
			Subtasks: []Subtask{
				{
					ID:          "task1",
					Description: userQuery,
					AgentType:   "general",
					DependsOn:   []string{},
					Priority:    1,
				},
			},
			ExecutionOrder: "sequential",
			RequiredAgents: []string{"general"},
			Strategy:       "Single-step execution (fallback)",
			Reasoning:      "Error during goal extraction; using fallback",
		}, nil
	}

	// Parse response
	var decomposition TaskDecomposition
	if err := json.Unmarshal([]byte(text), &decomposition); err != nil {
		// Fallback on parse error
		return &TaskDecomposition{
			MainGoal: userQuery,
			Subtasks: []Subtask{
				{
					ID:          "task1",
					Description: userQuery,
					AgentType:   "general",
					DependsOn:   []string{},
					Priority:    1,
				},
			},
			ExecutionOrder: "sequential",
			RequiredAgents: []string{"general"},
			Strategy:       "Single-step execution (fallback)",
			Reasoning:      "Failed to parse goal extraction; using fallback",
		}, nil
	}

	return &decomposition, nil
}

// buildGoalExtractionPrompt creates the prompt for goal extraction
func buildGoalExtractionPrompt(userQuery string, availableAgents []string) string {
	agentsInfo := "No specific agents available (use general agent types)"
	if len(availableAgents) > 0 {
		agentsInfo = fmt.Sprintf("Available agents: %v", availableAgents)
	}

	return fmt.Sprintf(`You are a task planning expert helping a supervisor agent decompose a complex request.

**User Request:**
%s

**%s**

**Your Task:**
Break down this request into clear, actionable subtasks. For each subtask:
1. Identify what agent type or specific agent should handle it
2. Determine dependencies (which tasks must complete first)
3. Assign priorities (1=highest, 5=lowest)

Consider whether tasks can run in parallel or must be sequential.
Think about which agent types are best suited for each subtask.

Provide your decomposition in JSON format with:
- main_goal: Clear statement of the overall objective
- subtasks: Array of subtask objects with id, description, agent_type, depends_on, priority
- execution_order: "sequential", "parallel", or "hierarchical"
- required_agents: List of agent types/names needed
- strategy: High-level approach (1-2 sentences)
- reasoning: Why you decomposed it this way

Keep it practical and actionable. Don't over-decompose simple tasks.`, userQuery, agentsInfo)
}
