package config

import (
	"testing"
)

// TestValidationIntegration tests the full validation pipeline end-to-end
func TestValidationIntegration(t *testing.T) {
	tests := []struct {
		name           string
		yaml           string
		expectValid    bool
		expectErrors   int
		expectWarnings int
	}{
		{
			name: "completely valid config",
			yaml: `
version: "1.0"
name: "test"
agents:
  my-agent:
    type: native
    name: Test Agent
`,
			expectValid:  true,
			expectErrors: 0,
		},
		{
			name: "multiple typos",
			yaml: `
ageents:
  test: {}
llmms:
  test: {}
toools:
  test: {}
`,
			expectValid:  false,
			expectErrors: 3,
		},
		{
			name: "mix of valid and invalid",
			yaml: `
version: "1.0"
agents:
  my-agent:
    type: native
invalid_field: value
`,
			expectValid:  false,
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := loadYAML(t, tt.yaml)
			result, err := ValidateConfigStructure(data)
			if err != nil {
				t.Fatalf("validation check failed: %v", err)
			}

			if result.Valid() != tt.expectValid {
				t.Errorf("Valid() = %v, expected %v. Errors: %s",
					result.Valid(), tt.expectValid, result.FormatErrors())
			}

			totalErrors := len(result.UnknownFields) + len(result.TypeErrors)
			if totalErrors != tt.expectErrors {
				t.Errorf("expected %d errors, got %d. Details: %s",
					tt.expectErrors, totalErrors, result.FormatErrors())
			}
		})
	}
}

// TestFuzzyMatchingQuality tests that fuzzy matching provides good suggestions
func TestFuzzyMatchingQuality(t *testing.T) {
	validFields := []string{"agents", "llms", "tools", "databases", "embedders"}

	tests := []struct {
		typo                  string
		expectFirstSuggestion string
	}{
		{"agent", "agents"},       // Missing 's'
		{"ageent", "agents"},      // Typo
		{"ageents", "agents"},     // Extra letter
		{"llm", "llms"},           // Missing 's'
		{"llmm", "llms"},          // Double letter
		{"llmms", "llms"},         // Double + s
		{"tool", "tools"},         // Missing 's'
		{"toools", "tools"},       // Extra letters
		{"databse", "databases"},  // Typo
		{"embedder", "embedders"}, // Missing 's'
	}

	for _, tt := range tests {
		t.Run(tt.typo, func(t *testing.T) {
			suggestions := findSimilarFields(tt.typo, validFields, 2)
			if len(suggestions) == 0 {
				t.Errorf("expected suggestions for %q, got none", tt.typo)
				return
			}
			if suggestions[0] != tt.expectFirstSuggestion {
				t.Errorf("expected first suggestion %q for %q, got %q",
					tt.expectFirstSuggestion, tt.typo, suggestions[0])
			}
		})
	}
}

// TestErrorMessageQuality verifies error messages are helpful
func TestErrorMessageQuality(t *testing.T) {
	yaml := `
ageents:
  test: {}
random_field: value
`
	data := loadYAML(t, yaml)
	result, err := ValidateConfigStructure(data)
	if err != nil {
		t.Fatalf("validation check failed: %v", err)
	}

	if result.Valid() {
		t.Fatal("expected validation to fail")
	}

	errMsg := result.FormatErrors()

	// Check for required elements in error message
	requiredElements := []string{
		"Configuration validation errors",
		"Unknown/Typo Fields",
		"Common causes",
		"Hints",
		"docs/reference/configuration.md",
	}

	for _, element := range requiredElements {
		if !stringContains(errMsg, element) {
			t.Errorf("error message missing required element: %q\nFull message:\n%s",
				element, errMsg)
		}
	}

	// Check that suggestions are present
	if !stringContains(errMsg, "Did you mean") {
		t.Error("error message should include suggestions")
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
