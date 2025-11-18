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
		sb.WriteString("ERROR: Configuration validation errors:\n\n")
	}

	if len(r.UnknownFields) > 0 {
		sb.WriteString("UNKNOWN: Unknown/Typo Fields (not recognized):\n")
		for _, field := range r.UnknownFields {
			sb.WriteString(fmt.Sprintf("   • %s: %s\n", field.Field, field.Message))
			if len(field.Suggestions) > 0 {
				sb.WriteString(fmt.Sprintf("     TIP: Did you mean: %s?\n", strings.Join(field.Suggestions, ", ")))
			}
			if field.Context != "" {
				sb.WriteString(fmt.Sprintf("     INFO: %s\n", field.Context))
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
		sb.WriteString("TYPE_ERROR: Type Errors:\n")
		for _, err := range r.TypeErrors {
			sb.WriteString(fmt.Sprintf("   • %s: %s\n", err.Field, err.Message))
			if err.Context != "" {
				sb.WriteString(fmt.Sprintf("     INFO: %s\n", err.Context))
			}
		}
		sb.WriteString("\n")
	}

	if len(r.Warnings) > 0 {
		sb.WriteString("WARN: Warnings (non-fatal):\n")
		for _, warn := range r.Warnings {
			sb.WriteString(fmt.Sprintf("   • %s: %s\n", warn.Field, warn.Message))
			if warn.Context != "" {
				sb.WriteString(fmt.Sprintf("     INFO: %s\n", warn.Context))
			}
		}
		sb.WriteString("\n")
	}

	if hasErrors {
		sb.WriteString("TIP: Hints:\n")
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
		// Parse unknown fields - pass the full error string for better context
		unknownFields := extractUnknownFieldsImproved(errStr, rawMap, errStr)
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
func extractUnknownFieldsImproved(errMsg string, rawMap map[string]interface{}, fullErrMsg string) []FieldError {
	var fieldErrors []FieldError

	// mapstructure error format for nested structs:
	// "1 error(s) decoding:\n\n* 'agents[enterprise_assistant].search' has invalid keys: search_mode, hybrid_alpha, rerank"
	// or simpler: "...'search' has invalid keys: key1, key2, key3"
	if idx := strings.Index(errMsg, "has invalid keys:"); idx != -1 {
		// Extract the parent path (what comes before "has invalid keys:")
		beforeKeys := errMsg[:idx]
		parentPath := ""

		// Try multiple patterns to extract parent path
		// Pattern 1: "...'search' has invalid keys: ..."
		// Pattern 2: "...'agents[enterprise_assistant].search' has invalid keys: ..."
		if lastQuote := strings.LastIndex(beforeKeys, "'"); lastQuote != -1 && lastQuote > 0 {
			// Find the opening quote for this field
			// Look backwards from lastQuote to find matching opening quote
			openingQuote := -1
			for i := lastQuote - 1; i >= 0; i-- {
				if beforeKeys[i] == '\'' {
					openingQuote = i
					break
				}
			}

			if openingQuote != -1 && openingQuote < lastQuote {
				parentPath = beforeKeys[openingQuote+1 : lastQuote]
				// Clean up path (remove array notation like [enterprise_assistant])
				if bracketIdx := strings.Index(parentPath, "["); bracketIdx != -1 {
					parentPath = parentPath[:bracketIdx]
				}
				// Remove any remaining path separators at the start
				parentPath = strings.TrimPrefix(parentPath, "agents.")
			}
		}

		keysStr := errMsg[idx+len("has invalid keys:"):]
		keysStr = strings.TrimSpace(keysStr)

		// Get valid field names from Config struct for fuzzy matching
		validFields := getValidFieldNames(reflect.TypeOf(Config{}))

		for _, key := range strings.Split(keysStr, ",") {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}

			// Build full field path
			// parentPath could be "search" or "agents[enterprise_assistant].search"
			// We need to normalize it to just "search" for matching
			normalizedParent := parentPath
			if strings.Contains(parentPath, "[") {
				// Extract just the last component after the last dot or bracket
				parts := strings.Split(parentPath, ".")
				if len(parts) > 0 {
					lastPart := parts[len(parts)-1]
					if bracketIdx := strings.Index(lastPart, "["); bracketIdx != -1 {
						lastPart = lastPart[:bracketIdx]
					}
					normalizedParent = lastPart
				}
			}
			// Remove "agents." prefix if present
			normalizedParent = strings.TrimPrefix(normalizedParent, "agents.")

			// Build full field path for matching
			fullPath := key
			if normalizedParent != "" {
				fullPath = normalizedParent + "." + key
			}

			// Check if this field actually exists in the schema
			// validFields contains patterns like:
			// - "search.search_mode"
			// - "agents.search.search_mode"
			// - "agents.<agent-name>.search.search_mode"
			fieldExists := false

			for _, validField := range validFields {
				// Match exact path
				if validField == fullPath {
					fieldExists = true
					break
				}
				// Match with "search." prefix (e.g., "search.search_mode")
				if normalizedParent == "search" && validField == "search."+key {
					fieldExists = true
					break
				}
				// Match nested paths (e.g., "agents.search.search_mode")
				if strings.HasSuffix(validField, ".search."+key) ||
					strings.Contains(validField, ".search."+key+".") ||
					strings.Contains(validField, "search."+key) {
					fieldExists = true
					break
				}
			}

			// Only report as error if field doesn't exist
			if !fieldExists {
				// Find suggestions using fuzzy matching (try both with and without prefix)
				suggestions := findSimilarFields(fullPath, validFields, 2)
				if len(suggestions) == 0 {
					// Also try without prefix for better suggestions
					suggestions = findSimilarFields(key, validFields, 2)
				}

				fieldError := FieldError{
					Field:       fullPath,
					Message:     "field is not recognized in configuration structure",
					Suggestions: suggestions,
					Severity:    SeverityError,
					Context:     "This field does not exist in the configuration schema",
				}
				fieldErrors = append(fieldErrors, fieldError)
			}
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

	if t.Kind() == reflect.Map {
		// For maps, recurse into the value type
		return getValidFieldNames(t.Elem())
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

			// Handle maps (e.g., agents map[string]*AgentConfig)
			if fieldType.Kind() == reflect.Map {
				// Recurse into map value type with prefix
				mapValueType := fieldType.Elem()
				if mapValueType.Kind() == reflect.Ptr {
					mapValueType = mapValueType.Elem()
				}
				nestedFields := getValidFieldNames(mapValueType)
				for _, nf := range nestedFields {
					fields = append(fields, fieldName+".<agent-name>."+nf)
					// Also add without agent name for direct matching
					fields = append(fields, fieldName+"."+nf)
				}
			} else if fieldType.Kind() == reflect.Struct {
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
