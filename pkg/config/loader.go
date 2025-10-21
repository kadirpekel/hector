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

// ============================================================================
// SERVE COMMAND CONFIG LOADER
// ============================================================================

// LoadResult contains the loaded configuration and metadata about how it was loaded
type LoadResult struct {
	Config         *Config
	IsZeroConfig   bool
	ResolvedOpts   ZeroConfigOptions // Resolved options (with env vars, defaults applied)
	LoadedFromFile bool
}

// LoadOrCreateConfig loads configuration from a file or creates zero-config
// This is the recommended way to load configuration for the serve command
func LoadOrCreateConfig(configFile string, zeroOpts ZeroConfigOptions) (*LoadResult, error) {
	result := &LoadResult{}

	// Check if config file exists
	if configFile != "" {
		if _, err := os.Stat(configFile); err == nil {
			// File exists - load from file
			cfg, loadErr := LoadConfig(configFile)
			if loadErr != nil {
				return nil, fmt.Errorf("failed to load config: %w", loadErr)
			}
			result.Config = cfg
			result.LoadedFromFile = true
			result.IsZeroConfig = false
			return result, nil
		}
	}

	// No file or file doesn't exist - use zero-config
	cfg := CreateZeroConfig(zeroOpts)
	result.Config = cfg
	result.IsZeroConfig = true
	result.LoadedFromFile = false

	// Store resolved options (after env var resolution and defaults)
	// This is useful for debug output
	result.ResolvedOpts = extractResolvedOptions(cfg, zeroOpts.AgentName)

	return result, nil
}

// extractResolvedOptions extracts the resolved configuration options from a zero-config
// This helps with debug output by showing what was actually used
func extractResolvedOptions(cfg *Config, agentName string) ZeroConfigOptions {
	opts := ZeroConfigOptions{
		AgentName: agentName,
	}

	// Extract LLM config (first one)
	for provider, llmCfg := range cfg.LLMs {
		opts.Provider = provider
		opts.Model = llmCfg.Model
		opts.BaseURL = llmCfg.Host
		opts.APIKey = llmCfg.APIKey
		break
	}

	// Extract tools config
	if len(cfg.Tools.Tools) > 0 {
		if mcpTool, hasMCP := cfg.Tools.Tools["mcp"]; hasMCP && mcpTool.Enabled {
			opts.MCPURL = mcpTool.ServerURL
		}
	}

	// Check if agent has all tools enabled
	if agentName != "" {
		if agent, hasAgent := cfg.Agents[agentName]; hasAgent {
			opts.EnableTools = agent.Tools == nil || len(agent.Tools) > 0
		}
	}

	// Extract document store config (first one if any)
	for _, docStore := range cfg.DocumentStores {
		if docStore.Source == "directory" {
			opts.DocsFolder = docStore.Path
			break
		}
	}

	return opts
}

// FormatZeroConfigDebug returns formatted debug output for zero-config mode
func FormatZeroConfigDebug(opts ZeroConfigOptions) string {
	output := "🔧 Zero-config mode:\n"
	output += fmt.Sprintf("  Provider: %s\n", opts.Provider)
	output += fmt.Sprintf("  Model: %s\n", opts.Model)
	if opts.BaseURL != "" {
		output += fmt.Sprintf("  Base URL: %s\n", opts.BaseURL)
	}
	if opts.EnableTools {
		output += "  Tools: Enabled\n"
	}
	if opts.MCPURL != "" {
		output += fmt.Sprintf("  MCP: %s\n", opts.MCPURL)
	}
	if opts.DocsFolder != "" {
		output += fmt.Sprintf("  Docs: %s\n", opts.DocsFolder)
	}
	if opts.AgentName != "" && opts.AgentName != DefaultAgentName {
		output += fmt.Sprintf("  Agent: %s\n", opts.AgentName)
	}
	return output
}
