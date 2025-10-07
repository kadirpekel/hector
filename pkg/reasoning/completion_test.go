package reasoning

import (
	"context"
	"testing"
)

func TestAssessTaskCompletion_EmptyResponse(t *testing.T) {
	services := &mockAgentServices{}

	assessment, err := AssessTaskCompletion(
		context.Background(),
		"Test query",
		"",
		services,
	)

	if err != nil {
		t.Errorf("AssessTaskCompletion() should not return error for empty response, got: %v", err)
	}

	if assessment == nil {
		t.Fatal("AssessTaskCompletion() returned nil assessment")
	}

	// Should default to complete for providers without structured output
	if !assessment.IsComplete {
		t.Errorf("Expected IsComplete=true for provider without structured output, got false")
	}

	if assessment.Recommendation != "stop" {
		t.Errorf("Expected Recommendation='stop', got '%s'", assessment.Recommendation)
	}
}

func TestBuildCompletionPrompt(t *testing.T) {
	query := "Calculate 2+2"
	response := "The answer is 4"

	prompt := buildCompletionPrompt(query, response)

	if prompt == "" {
		t.Error("buildCompletionPrompt() returned empty string")
	}

	// Check that prompt contains key elements
	if len(prompt) < 100 {
		t.Errorf("buildCompletionPrompt() returned suspiciously short prompt: %s", prompt)
	}

	// Should mention the query and response
	// Basic sanity check - actual content is in the prompt
	if len(query) > 0 && len(response) > 0 && len(prompt) < len(query)+len(response) {
		t.Error("buildCompletionPrompt() seems to be missing query or response")
	}
}

func TestAssessTaskCompletion_WithMockServices(t *testing.T) {
	services := &mockAgentServices{}

	tests := []struct {
		name     string
		query    string
		response string
	}{
		{
			name:     "simple query",
			query:    "What is 2+2?",
			response: "The answer is 4",
		},
		{
			name:     "complex query",
			query:    "Analyze the weather and send an email",
			response: "I've analyzed the weather (sunny) and sent an email to john@example.com",
		},
		{
			name:     "incomplete response",
			query:    "Do three things: A, B, and C",
			response: "I've done A and B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment, err := AssessTaskCompletion(
				context.Background(),
				tt.query,
				tt.response,
				services,
			)

			if err != nil {
				t.Errorf("AssessTaskCompletion() error = %v", err)
			}

			if assessment == nil {
				t.Fatal("AssessTaskCompletion() returned nil")
			}

			// Validate fields
			if assessment.Confidence < 0.0 || assessment.Confidence > 1.0 {
				t.Errorf("Confidence = %f, should be between 0.0 and 1.0", assessment.Confidence)
			}

			validQualities := map[string]bool{
				"excellent":         true,
				"good":              true,
				"needs_improvement": true,
			}
			if !validQualities[assessment.Quality] {
				t.Errorf("Quality = %s, should be one of [excellent, good, needs_improvement]", assessment.Quality)
			}

			validRecommendations := map[string]bool{
				"stop":     true,
				"continue": true,
				"clarify":  true,
			}
			if !validRecommendations[assessment.Recommendation] {
				t.Errorf("Recommendation = %s, should be one of [stop, continue, clarify]", assessment.Recommendation)
			}
		})
	}
}
