// Package config provides configuration types and utilities for the AI agent framework.
// This file contains environment variable utilities for configuration processing.
package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// ============================================================================
// ENVIRONMENT VARIABLE UTILITIES
// ============================================================================

var (
	// Pre-compiled regex patterns for better performance
	envVarPatterns = struct {
		withDefault *regexp.Regexp // ${VAR:-default}
		braced      *regexp.Regexp // ${VAR}
		simple      *regexp.Regexp // $VAR
	}{
		withDefault: regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*):-(.*?)\}`),
		braced:      regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`),
		simple:      regexp.MustCompile(`\$([A-Z_][A-Z0-9_]*)`),
	}
)

// expandEnvVars expands environment variables in a string
// Supports formats: ${VAR:-default}, ${VAR}, $VAR
// Processes patterns in order to avoid conflicts
func expandEnvVars(s string) string {
	// Early return if no environment variables detected
	if !strings.Contains(s, "$") {
		return s
	}

	// Process ${VAR:-default} first (most specific)
	s = envVarPatterns.withDefault.ReplaceAllStringFunc(s, func(match string) string {
		parts := envVarPatterns.withDefault.FindStringSubmatch(match)
		if len(parts) == 3 {
			envVar := parts[1]
			defaultVal := parts[2]
			if val := os.Getenv(envVar); val != "" {
				return val
			}
			return defaultVal
		}
		return match
	})

	// Process ${VAR} format (must come after ${VAR:-default})
	s = envVarPatterns.braced.ReplaceAllStringFunc(s, func(match string) string {
		parts := envVarPatterns.braced.FindStringSubmatch(match)
		if len(parts) == 2 {
			return os.Getenv(parts[1])
		}
		return match
	})

	// Process $VAR format (simple, least specific)
	s = envVarPatterns.simple.ReplaceAllStringFunc(s, func(match string) string {
		parts := envVarPatterns.simple.FindStringSubmatch(match)
		if len(parts) == 2 {
			return os.Getenv(parts[1])
		}
		return match
	})

	return s
}

// parseValue attempts to parse a string value to its appropriate type
// Returns the parsed value or the original string if parsing fails
func parseValue(value string) interface{} {
	// Try boolean first
	switch strings.ToLower(value) {
	case "true":
		return true
	case "false":
		return false
	}

	// Try integer
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Try float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// Return as string if no conversion applies
	return value
}

// ExpandEnvVarsInData recursively expands environment variables in structured data
// and preserves original data types through intelligent parsing
func ExpandEnvVarsInData(data interface{}) interface{} {
	switch v := data.(type) {
	case string:
		expanded := expandEnvVars(v)
		// If the string was expanded and looks like it should be a different type
		if expanded != v {
			return parseValue(expanded)
		}
		return expanded

	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, value := range v {
			result[key] = ExpandEnvVarsInData(value)
		}
		return result

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = ExpandEnvVarsInData(item)
		}
		return result

	default:
		return v
	}
}

// LoadEnvFiles loads environment variables from .env files
// Loads in priority order: .env.local (highest) → .env → system environment (lowest)
func LoadEnvFiles() error {
	envFiles := []string{".env.local", ".env"}

	for _, file := range envFiles {
		if err := godotenv.Load(file); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to load %s: %w", file, err)
		}
	}

	return nil
}
