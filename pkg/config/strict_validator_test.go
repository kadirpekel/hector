package config

import (
	"reflect"
	"strings"
	"testing"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

func loadYAML(t *testing.T, yamlStr string) *koanf.Koanf {
	t.Helper()

	// Parse YAML into a map first
	parser := yaml.Parser()
	data, err := parser.Unmarshal([]byte(yamlStr))
	if err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	// Load into koanf using confmap
	k := koanf.New(".")
	if err := k.Load(confmap.Provider(data, "."), nil); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	return k
}

func TestValidateConfigStructure_ValidConfig(t *testing.T) {
	validYAML := `
version: "1.0"
name: "test-config"
agents:
  my-agent:
    type: native
    name: My Agent
llms:
  openai:
    model: gpt-4
`
	k := loadYAML(t, validYAML)

	result, err := ValidateConfigStructure(k)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if !result.Valid() {
		t.Errorf("expected valid config, got errors: %s", result.FormatErrors())
	}
}

func TestValidateConfigStructure_UnknownField(t *testing.T) {
	tests := []struct {
		name           string
		yaml           string
		expectUnknown  []string
		expectSuggests bool
	}{
		{
			name: "typo in top-level field",
			yaml: `
ageents:
  my-agent:
    type: native
`,
			expectUnknown:  []string{"ageents"},
			expectSuggests: true, // Should suggest "agents"
		},
		{
			name: "completely unknown field",
			yaml: `
random_field: value
agents:
  my-agent:
    type: native
`,
			expectUnknown: []string{"random_field"},
		},
		{
			name: "typo in nested field",
			yaml: `
agents:
  my-agent:
    typpe: native
`,
			expectUnknown: []string{"typpe"},
		},
		{
			name: "multiple unknown fields",
			yaml: `
ageents:
  my-agent:
    type: native
llmms:
  openai:
    provider: openai
unknown_top: value
`,
			expectUnknown: []string{"ageents", "llmms", "unknown_top"},
		},
		{
			name: "field at wrong level",
			yaml: `
version: "1.0"
agents:
  my-agent:
    type: native
global:
  unknown_field: value
`,
			expectUnknown: []string{"unknown_field"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := loadYAML(t, tt.yaml)

			result, err := ValidateConfigStructure(k)
			if err != nil {
				t.Fatalf("validation check failed: %v", err)
			}

			if result.Valid() {
				t.Errorf("expected validation to fail, but it passed")
			}

			if len(result.UnknownFields) == 0 {
				t.Errorf("expected unknown fields errors, got none")
			}

			// Check that expected fields are reported
			for _, expectedField := range tt.expectUnknown {
				found := false
				for _, fieldErr := range result.UnknownFields {
					if strings.Contains(fieldErr.Field, expectedField) || fieldErr.Field == expectedField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected unknown field %q to be reported, but it wasn't", expectedField)
				}
			}

			// Check suggestions are provided when expected
			if tt.expectSuggests {
				hasSuggestions := false
				for _, fieldErr := range result.UnknownFields {
					if len(fieldErr.Suggestions) > 0 {
						hasSuggestions = true
						break
					}
				}
				if !hasSuggestions {
					t.Errorf("expected suggestions for typos, but none were provided")
				}
			}

			// Check that error message is helpful
			errMsg := result.FormatErrors()
			if !strings.Contains(errMsg, "Unknown/Typo Fields") {
				t.Errorf("error message should mention unknown/typo fields")
			}
		})
	}
}

func TestValidateConfigStructure_TypeErrors(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
	}{
		{
			name: "string where number expected",
			yaml: `
global:
  performance:
    max_concurrent_requests: "not-a-number"
`,
			expectError: true,
		},
		{
			name: "number where string expected",
			yaml: `
version: 123
`,
			expectError: true,
		},
		{
			name: "object where string expected",
			yaml: `
name:
  nested: value
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := loadYAML(t, tt.yaml)

			result, err := ValidateConfigStructure(k)
			if err != nil {
				t.Fatalf("validation check failed: %v", err)
			}

			if tt.expectError {
				if result.Valid() {
					t.Errorf("expected type error, but validation passed")
				}
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "adc", 1},
		{"abc", "dbc", 1},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"ageents", "agents", 1},
		{"llmms", "llms", 1},
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_"+tt.s2, func(t *testing.T) {
			distance := levenshteinDistance(tt.s1, tt.s2)
			if distance != tt.expected {
				t.Errorf("levenshteinDistance(%q, %q) = %d, expected %d",
					tt.s1, tt.s2, distance, tt.expected)
			}
		})
	}
}

func TestFindSimilarFields(t *testing.T) {
	validFields := []string{"agents", "llms", "tools", "databases", "embedders", "plugins"}

	tests := []struct {
		typo           string
		maxDistance    int
		expectSuggests bool
		expectContains string
	}{
		{
			typo:           "ageents",
			maxDistance:    2,
			expectSuggests: true,
			expectContains: "agents",
		},
		{
			typo:           "llmms",
			maxDistance:    2,
			expectSuggests: true,
			expectContains: "llms",
		},
		{
			typo:           "toools",
			maxDistance:    2,
			expectSuggests: true,
			expectContains: "tools",
		},
		{
			typo:           "completely_wrong",
			maxDistance:    2,
			expectSuggests: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.typo, func(t *testing.T) {
			suggestions := findSimilarFields(tt.typo, validFields, tt.maxDistance)

			if tt.expectSuggests {
				if len(suggestions) == 0 {
					t.Errorf("expected suggestions for %q, got none", tt.typo)
				}
				if tt.expectContains != "" {
					found := false
					for _, s := range suggestions {
						if s == tt.expectContains {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected suggestions to contain %q, got %v", tt.expectContains, suggestions)
					}
				}
			} else {
				if len(suggestions) > 0 {
					t.Errorf("expected no suggestions for %q, got %v", tt.typo, suggestions)
				}
			}
		})
	}
}

func TestGetValidFieldNames(t *testing.T) {
	// Test with a simple struct
	type TestStruct struct {
		Field1 string `yaml:"field1"`
		Field2 int    `yaml:"field2,omitempty"`
		Field3 bool   `yaml:"-"` // Should be ignored
	}

	fields := getValidFieldNames(reflect.TypeOf(TestStruct{}))

	expectedFields := []string{"field1", "field2"}
	if len(fields) != len(expectedFields) {
		t.Errorf("expected %d fields, got %d: %v", len(expectedFields), len(fields), fields)
	}

	for _, expected := range expectedFields {
		found := false
		for _, field := range fields {
			if field == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected field %q not found in %v", expected, fields)
		}
	}

	// Field3 should not be included
	for _, field := range fields {
		if field == "field3" {
			t.Errorf("field3 should be ignored due to yaml:\"-\" tag")
		}
	}
}

func TestStrictValidationResult_FormatErrors(t *testing.T) {
	tests := []struct {
		name           string
		result         StrictValidationResult
		expectContains []string
		expectValid    bool
	}{
		{
			name: "no errors",
			result: StrictValidationResult{
				UnknownFields: []FieldError{},
				TypeErrors:    []FieldError{},
				Warnings:      []FieldError{},
			},
			expectValid: true,
		},
		{
			name: "unknown field with suggestion",
			result: StrictValidationResult{
				UnknownFields: []FieldError{
					{
						Field:       "ageents",
						Message:     "field is not recognized",
						Suggestions: []string{"agents"},
						Severity:    SeverityError,
					},
				},
			},
			expectContains: []string{"ageents", "Unknown/Typo Fields", "Did you mean: agents"},
			expectValid:    false,
		},
		{
			name: "type error",
			result: StrictValidationResult{
				TypeErrors: []FieldError{
					{
						Field:    "version",
						Message:  "expected string, got int",
						Severity: SeverityError,
					},
				},
			},
			expectContains: []string{"version", "Type Errors", "expected string, got int"},
			expectValid:    false,
		},
		{
			name: "warnings only",
			result: StrictValidationResult{
				Warnings: []FieldError{
					{
						Field:    "deprecated_field",
						Message:  "this field is deprecated",
						Severity: SeverityWarning,
					},
				},
			},
			expectContains: []string{"deprecated_field", "Warnings"},
			expectValid:    true, // Warnings don't make config invalid
		},
		{
			name: "multiple errors and warnings",
			result: StrictValidationResult{
				UnknownFields: []FieldError{
					{
						Field:       "ageents",
						Suggestions: []string{"agents"},
						Severity:    SeverityError,
					},
				},
				TypeErrors: []FieldError{
					{
						Field:    "version",
						Severity: SeverityError,
					},
				},
				Warnings: []FieldError{
					{
						Field:    "old_field",
						Severity: SeverityWarning,
					},
				},
			},
			expectContains: []string{"ageents", "version", "old_field", "Unknown/Typo Fields", "Type Errors", "Warnings"},
			expectValid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.Valid() != tt.expectValid {
				t.Errorf("Valid() = %v, expected %v", tt.result.Valid(), tt.expectValid)
			}

			formatted := tt.result.FormatErrors()

			if tt.expectValid && formatted != "" && len(tt.result.Warnings) == 0 {
				t.Errorf("expected no error message for valid config without warnings, got: %s", formatted)
			}

			for _, expected := range tt.expectContains {
				if !strings.Contains(formatted, expected) {
					t.Errorf("expected error message to contain %q, got:\n%s", expected, formatted)
				}
			}
		})
	}
}

func TestStrictValidationResult_HasIssues(t *testing.T) {
	tests := []struct {
		name        string
		result      StrictValidationResult
		expectIssue bool
	}{
		{
			name:        "no issues",
			result:      StrictValidationResult{},
			expectIssue: false,
		},
		{
			name: "has unknown fields",
			result: StrictValidationResult{
				UnknownFields: []FieldError{{Field: "test"}},
			},
			expectIssue: true,
		},
		{
			name: "has type errors",
			result: StrictValidationResult{
				TypeErrors: []FieldError{{Field: "test"}},
			},
			expectIssue: true,
		},
		{
			name: "has warnings only",
			result: StrictValidationResult{
				Warnings: []FieldError{{Field: "test"}},
			},
			expectIssue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.HasIssues() != tt.expectIssue {
				t.Errorf("HasIssues() = %v, expected %v", tt.result.HasIssues(), tt.expectIssue)
			}
		})
	}
}

func TestFieldError_Severity(t *testing.T) {
	errorField := FieldError{
		Field:    "test",
		Severity: SeverityError,
	}

	warningField := FieldError{
		Field:    "test",
		Severity: SeverityWarning,
	}

	if errorField.Severity != SeverityError {
		t.Errorf("expected error severity, got %v", errorField.Severity)
	}

	if warningField.Severity != SeverityWarning {
		t.Errorf("expected warning severity, got %v", warningField.Severity)
	}
}
