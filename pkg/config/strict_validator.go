package config

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/v2"
	"github.com/mitchellh/mapstructure"
)

// StrictValidationResult contains validation errors from strict unmarshaling
type StrictValidationResult struct {
	UnknownFields []string
	TypeErrors    []string
}

// Valid returns true if there are no validation errors
func (r *StrictValidationResult) Valid() bool {
	return len(r.UnknownFields) == 0 && len(r.TypeErrors) == 0
}

// FormatErrors returns a human-readable error message
func (r *StrictValidationResult) FormatErrors() string {
	if r.Valid() {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("❌ Configuration validation errors:\n\n")

	if len(r.UnknownFields) > 0 {
		sb.WriteString("📝 Unknown/Typo Fields (not recognized):\n")
		for _, field := range r.UnknownFields {
			sb.WriteString(fmt.Sprintf("   • %s\n", field))
		}
		sb.WriteString("\n")
		sb.WriteString("   These fields are not part of the configuration structure.\n")
		sb.WriteString("   Common causes:\n")
		sb.WriteString("   - Typos in field names\n")
		sb.WriteString("   - Incorrect nesting level\n")
		sb.WriteString("   - Using removed/deprecated fields\n")
		sb.WriteString("   - Copy-paste errors from examples\n\n")
	}

	if len(r.TypeErrors) > 0 {
		sb.WriteString("🔧 Type Errors:\n")
		for _, err := range r.TypeErrors {
			sb.WriteString(fmt.Sprintf("   • %s\n", err))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("💡 Hints:\n")
	sb.WriteString("   • Check field names against: docs/reference/configuration.md\n")
	sb.WriteString("   • Verify correct nesting (e.g., 'agents.my-agent.llm' not 'agents.llm')\n")
	sb.WriteString("   • Use 'hector validate <file> --print-config' to see expanded config\n")
	sb.WriteString("   • Compare with working examples in configs/ directory\n")

	return sb.String()
}

// ValidateConfigStructure performs strict validation on raw config data
// This catches typos, unknown fields, and incorrect nesting BEFORE
// the config is processed, providing early feedback to users
func ValidateConfigStructure(k *koanf.Koanf) (*StrictValidationResult, error) {
	result := &StrictValidationResult{}

	// Try to unmarshal with strict error detection
	cfg := &Config{}

	// Use mapstructure with ErrorUnused to catch unknown fields
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      cfg,
		ErrorUnused: true, // This is the key: error on unknown fields
		TagName:     "yaml",
		// Weak type coercion disabled - catch type mismatches
		WeaklyTypedInput: false,
		// Decode hook for better error messages
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	// Unmarshal and collect errors
	rawMap := k.Raw()
	if err := decoder.Decode(rawMap); err != nil {
		// Parse the error to categorize unknown fields vs type errors
		errStr := err.Error()

		// Check for unused key errors
		if strings.Contains(errStr, "unused key") || strings.Contains(errStr, "has invalid keys:") {
			// Extract field names from error
			result.UnknownFields = extractUnknownFields(errStr)
		} else {
			// Other errors are likely type errors
			result.TypeErrors = append(result.TypeErrors, errStr)
		}
	}

	return result, nil
}

// extractUnknownFields parses mapstructure error messages to extract field names
func extractUnknownFields(errMsg string) []string {
	var fields []string

	// mapstructure error format: "...has invalid keys: key1, key2, key3"
	if idx := strings.Index(errMsg, "has invalid keys:"); idx != -1 {
		keysStr := errMsg[idx+len("has invalid keys:"):]
		keysStr = strings.TrimSpace(keysStr)
		for _, key := range strings.Split(keysStr, ",") {
			key = strings.TrimSpace(key)
			if key != "" {
				fields = append(fields, key)
			}
		}
	}

	// If we couldn't parse it, return the raw error
	if len(fields) == 0 {
		fields = []string{errMsg}
	}

	return fields
}

// ValidateConfigBytes validates raw YAML/JSON bytes for structural issues
// This would require additional implementation to parse and create a koanf instance
func ValidateConfigBytes(data []byte, format string) (*StrictValidationResult, error) {
	// This is a placeholder for future implementation if needed
	// For now, users should use the full config loader which does strict validation
	return nil, fmt.Errorf("use the config loader which includes strict validation automatically")
}
