// Package config provides configuration types and utilities for the AI agent framework.
// This file contains generic configuration loading utilities that work with any Config interface implementation.
package config

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"
)

// ============================================================================
// GENERIC CONFIG LOADER
// ============================================================================

// loadConfigFromBytes is the core implementation that handles YAML parsing,
// environment variable expansion, defaults, and validation
func loadConfigFromBytes(data []byte, config ConfigInterface) error {
	// Note: .env files should be loaded before calling this function (typically in main())

	// Parse YAML into generic structure for environment variable expansion
	var rawConfig interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("failed to parse config YAML: %w", err)
	}

	// Expand environment variables in the parsed data
	expandedConfig := ExpandEnvVarsInData(rawConfig)

	// Convert expanded data directly to the target config using YAML marshaling
	expandedData, err := yaml.Marshal(expandedConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal expanded config: %w", err)
	}

	// Unmarshal into the target config struct
	if err := yaml.Unmarshal(expandedData, config); err != nil {
		return fmt.Errorf("failed to parse final config: %w", err)
	}

	// Set default values
	config.SetDefaults()

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}

// loadConfig loads a configuration from a YAML file with environment variable expansion
// The config parameter should be a pointer to a struct that implements the ConfigInterface interface
func loadConfig(filePath string, config ConfigInterface) error {
	// Read the configuration file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	return loadConfigFromBytes(data, config)
}

// loadConfigFromString loads a configuration from a YAML string with environment variable expansion
// The config parameter should be a pointer to a struct that implements the ConfigInterface interface
func loadConfigFromString(yamlContent string, config ConfigInterface) error {
	return loadConfigFromBytes([]byte(yamlContent), config)
}
