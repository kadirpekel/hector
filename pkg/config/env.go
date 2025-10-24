package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

var (
	envVarPatterns = struct {
		withDefault *regexp.Regexp
		braced      *regexp.Regexp
		simple      *regexp.Regexp
	}{
		withDefault: regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*):-(.*?)\}`),
		braced:      regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`),
		simple:      regexp.MustCompile(`\$([A-Z_][A-Z0-9_]*)`),
	}
)

func expandEnvVars(s string) string {

	if !strings.Contains(s, "$") {
		return s
	}

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

	s = envVarPatterns.braced.ReplaceAllStringFunc(s, func(match string) string {
		parts := envVarPatterns.braced.FindStringSubmatch(match)
		if len(parts) == 2 {
			return os.Getenv(parts[1])
		}
		return match
	})

	s = envVarPatterns.simple.ReplaceAllStringFunc(s, func(match string) string {
		parts := envVarPatterns.simple.FindStringSubmatch(match)
		if len(parts) == 2 {
			return os.Getenv(parts[1])
		}
		return match
	})

	return s
}

func parseValue(value string) interface{} {

	switch strings.ToLower(value) {
	case "true":
		return true
	case "false":
		return false
	}

	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	return value
}

func ExpandEnvVarsInData(data interface{}) interface{} {
	switch v := data.(type) {
	case string:
		expanded := expandEnvVars(v)

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

func LoadEnvFiles() error {
	envFiles := []string{".env.local", ".env"}

	for _, file := range envFiles {
		if err := godotenv.Load(file); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to load %s: %w", file, err)
		}
	}

	return nil
}

func GetProviderAPIKey(providerType string) string {
	switch providerType {
	case "openai":
		return os.Getenv("OPENAI_API_KEY")
	case "anthropic":
		return os.Getenv("ANTHROPIC_API_KEY")
	case "gemini":
		return os.Getenv("GEMINI_API_KEY")
	default:
		return ""
	}
}
