package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// ValidationSeverity indicates whether an issue is an error or warning
type ValidationSeverity string

const (
	SeverityError   ValidationSeverity = "error"
	SeverityWarning ValidationSeverity = "warning"
)

// FieldError represents a validation error for a specific field
type FieldError struct {
	Field       string             // Full path to the field (e.g., "agents.my-agent.llm")
	Message     string             // Error message
	Suggestions []string           // Suggested corrections (for typos)
	Severity    ValidationSeverity // Error or warning
	Context     string             // Additional context about the error
}

// StrictValidationResult contains validation errors from strict unmarshaling
type StrictValidationResult struct {
	UnknownFields []FieldError // Unknown/typo fields
	TypeErrors    []FieldError // Type mismatch errors
	Warnings      []FieldError // Non-fatal warnings
}

// Valid returns true if there are no validation errors (warnings are allowed)
func (r *StrictValidationResult) Valid() bool {
	return len(r.UnknownFields) == 0 && len(r.TypeErrors) == 0
}

// HasIssues returns true if there are any errors or warnings
func (r *StrictValidationResult) HasIssues() bool {
	return len(r.UnknownFields) > 0 || len(r.TypeErrors) > 0 || len(r.Warnings) > 0
}

// FormatErrors returns a human-readable error message
func (r *StrictValidationResult) FormatErrors() string {
	if !r.HasIssues() {
		return ""
	}

	var sb strings.Builder

	hasErrors := !r.Valid()
	if hasErrors {
		sb.WriteString("❌ Configuration validation errors:\n\n")
	}

	if len(r.UnknownFields) > 0 {
		sb.WriteString("📝 Unknown/Typo Fields (not recognized):\n")
		for _, field := range r.UnknownFields {
			sb.WriteString(fmt.Sprintf("   • %s: %s\n", field.Field, field.Message))
			if len(field.Suggestions) > 0 {
				sb.WriteString(fmt.Sprintf("     💡 Did you mean: %s?\n", strings.Join(field.Suggestions, ", ")))
			}
			if field.Context != "" {
				sb.WriteString(fmt.Sprintf("     ℹ️  %s\n", field.Context))
			}
		}
		sb.WriteString("\n")
		sb.WriteString("   Common causes:\n")
		sb.WriteString("   - Typos in field names\n")
		sb.WriteString("   - Incorrect nesting level\n")
		sb.WriteString("   - Using removed/deprecated fields\n")
		sb.WriteString("   - Copy-paste errors from examples\n\n")
	}

	if len(r.TypeErrors) > 0 {
		sb.WriteString("🔧 Type Errors:\n")
		for _, err := range r.TypeErrors {
			sb.WriteString(fmt.Sprintf("   • %s: %s\n", err.Field, err.Message))
			if err.Context != "" {
				sb.WriteString(fmt.Sprintf("     ℹ️  %s\n", err.Context))
			}
		}
		sb.WriteString("\n")
	}

	if len(r.Warnings) > 0 {
		sb.WriteString("⚠️  Warnings (non-fatal):\n")
		for _, warn := range r.Warnings {
			sb.WriteString(fmt.Sprintf("   • %s: %s\n", warn.Field, warn.Message))
			if warn.Context != "" {
				sb.WriteString(fmt.Sprintf("     ℹ️  %s\n", warn.Context))
			}
		}
		sb.WriteString("\n")
	}

	if hasErrors {
		sb.WriteString("💡 Hints:\n")
		sb.WriteString("   • Check field names against: docs/reference/configuration.md\n")
		sb.WriteString("   • Verify correct nesting (e.g., 'agents.my-agent.llm' not 'agents.llm')\n")
		sb.WriteString("   • Use 'hector validate <file> --print-config' to see expanded config\n")
		sb.WriteString("   • Compare with working examples in configs/ directory\n")
	}

	return sb.String()
}

// ValidateConfigStructure validates config structure from a map[string]interface{}
// This catches typos, unknown fields, and incorrect nesting BEFORE
// the config is processed, providing early feedback to users
func ValidateConfigStructure(rawMap map[string]interface{}) (*StrictValidationResult, error) {
	result := &StrictValidationResult{
		UnknownFields: []FieldError{},
		TypeErrors:    []FieldError{},
		Warnings:      []FieldError{},
	}

	cfg := &Config{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           cfg,
		ErrorUnused:      true,
		TagName:          "yaml",
		WeaklyTypedInput: false,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(rawMap); err != nil {
		collectValidationErrors(err, rawMap, result)
	}

	return result, nil
}

// collectValidationErrors processes mapstructure errors and categorizes them
func collectValidationErrors(err error, rawMap map[string]interface{}, result *StrictValidationResult) {
	errStr := err.Error()

	// Try to extract structured error information
	if strings.Contains(errStr, "has invalid keys:") {
		// Parse unknown fields
		unknownFields := extractUnknownFieldsImproved(errStr, rawMap)
		result.UnknownFields = append(result.UnknownFields, unknownFields...)
	} else if strings.Contains(errStr, "'") && (strings.Contains(errStr, "expected") || strings.Contains(errStr, "cannot unmarshal") || strings.Contains(errStr, "cannot decode")) {
		// Type error
		typeError := parseTypeError(errStr)
		result.TypeErrors = append(result.TypeErrors, typeError)
	} else {
		// Generic error - try to categorize
		if strings.Contains(errStr, "unused") || strings.Contains(errStr, "unknown") {
			result.UnknownFields = append(result.UnknownFields, FieldError{
				Field:    "unknown",
				Message:  errStr,
				Severity: SeverityError,
			})
		} else {
			result.TypeErrors = append(result.TypeErrors, FieldError{
				Field:    "unknown",
				Message:  errStr,
				Severity: SeverityError,
			})
		}
	}
}

// extractUnknownFieldsImproved parses mapstructure error messages and provides suggestions
func extractUnknownFieldsImproved(errMsg string, rawMap map[string]interface{}) []FieldError {
	var fieldErrors []FieldError

	// mapstructure error format: "...has invalid keys: key1, key2, key3"
	if idx := strings.Index(errMsg, "has invalid keys:"); idx != -1 {
		keysStr := errMsg[idx+len("has invalid keys:"):]
		keysStr = strings.TrimSpace(keysStr)

		// Get valid field names from Config struct for fuzzy matching
		validFields := getValidFieldNames(reflect.TypeOf(Config{}))

		for _, key := range strings.Split(keysStr, ",") {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}

			// Find suggestions using fuzzy matching
			suggestions := findSimilarFields(key, validFields, 2)

			fieldError := FieldError{
				Field:       key,
				Message:     "field is not recognized in configuration structure",
				Suggestions: suggestions,
				Severity:    SeverityError,
				Context:     "This field does not exist in the configuration schema",
			}
			fieldErrors = append(fieldErrors, fieldError)
		}
	}

	// If we couldn't parse it, return a generic error
	if len(fieldErrors) == 0 {
		fieldErrors = []FieldError{{
			Field:    "unknown",
			Message:  errMsg,
			Severity: SeverityError,
		}}
	}

	return fieldErrors
}

// parseTypeError extracts information from type conversion errors
func parseTypeError(errStr string) FieldError {
	// Try to extract field name from error message
	// Common patterns: "'field' expected type X, got Y"
	//                  "cannot unmarshal X into Go value of type Y (field: 'field')"

	fieldName := "unknown"

	// Try to find field name in single quotes
	if start := strings.Index(errStr, "'"); start != -1 {
		if end := strings.Index(errStr[start+1:], "'"); end != -1 {
			fieldName = errStr[start+1 : start+1+end]
		}
	}

	return FieldError{
		Field:    fieldName,
		Message:  errStr,
		Severity: SeverityError,
		Context:  "Check that the value type matches the expected type (string, number, boolean, etc.)",
	}
}

// getValidFieldNames recursively extracts all valid field names from a struct type
func getValidFieldNames(t reflect.Type) []string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	var fields []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Get yaml tag
		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "" || yamlTag == "-" {
			continue
		}

		// Extract field name from yaml tag (before comma)
		parts := strings.Split(yamlTag, ",")
		fieldName := strings.TrimSpace(parts[0])

		if fieldName != "" {
			fields = append(fields, fieldName)

			// If it's a struct or pointer to struct, recurse with prefix
			fieldType := field.Type
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}

			if fieldType.Kind() == reflect.Struct {
				// Add nested fields with prefix
				nestedFields := getValidFieldNames(fieldType)
				for _, nf := range nestedFields {
					fields = append(fields, fieldName+"."+nf)
				}
			}
		}
	}

	return fields
}

// findSimilarFields finds fields similar to the typo using Levenshtein distance
func findSimilarFields(typo string, validFields []string, maxDistance int) []string {
	var suggestions []string

	// Normalize the typo
	typoLower := strings.ToLower(typo)

	// Score each valid field
	type scoredField struct {
		field    string
		distance int
	}
	var scored []scoredField

	for _, validField := range validFields {
		validLower := strings.ToLower(validField)

		// Calculate Levenshtein distance
		distance := levenshteinDistance(typoLower, validLower)

		// Only consider if distance is within threshold
		if distance <= maxDistance {
			scored = append(scored, scoredField{validField, distance})
		}

		// Also check if typo is a substring or vice versa
		if strings.Contains(validLower, typoLower) || strings.Contains(typoLower, validLower) {
			if distance > maxDistance {
				scored = append(scored, scoredField{validField, maxDistance})
			}
		}
	}

	// Sort by distance (best matches first) and take top 3
	for i := 0; i < len(scored) && i < 3; i++ {
		minIdx := i
		for j := i + 1; j < len(scored); j++ {
			if scored[j].distance < scored[minIdx].distance {
				minIdx = j
			}
		}
		if minIdx != i {
			scored[i], scored[minIdx] = scored[minIdx], scored[i]
		}
		suggestions = append(suggestions, scored[i].field)
	}

	return suggestions
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create a matrix to store distances
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Calculate distances
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
