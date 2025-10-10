package main

import (
	"fmt"
	"os"

	"github.com/kadirpekel/hector/pkg/config"
)

// ============================================================================
// UNIFIED CONFIG LOADING
// ============================================================================

// loadConfigFromArgsOrFile loads configuration from file or creates zero-config
// This is the single source of truth for config loading across all commands
func loadConfigFromArgsOrFile(args *CLIArgs, requireAPIKey bool) (*config.Config, error) {
	// 1. Try explicit config file (non-default path)
	if args.ConfigFile != "" && args.ConfigFile != defaultConfigFile {
		return loadAndValidateConfigFile(args.ConfigFile)
	}

	// 2. Try default config file if it exists
	if fileExists(defaultConfigFile) {
		return loadAndValidateConfigFile(defaultConfigFile)
	}

	// 3. Zero-config mode - create config from CLI flags
	if requireAPIKey {
		apiKey, err := getOrRequireAPIKey(args.APIKey)
		if err != nil {
			return nil, err
		}
		args.APIKey = apiKey
	}

	return createZeroConfigFromArgs(args)
}

// loadAndValidateConfigFile loads a config file and validates it
func loadAndValidateConfigFile(path string) (*config.Config, error) {
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", path, err)
	}

	cfg.SetDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration in %s: %w", path, err)
	}

	return cfg, nil
}

// createZeroConfigFromArgs creates zero-config from CLI arguments
func createZeroConfigFromArgs(args *CLIArgs) (*config.Config, error) {
	opts := config.ZeroConfigOptions{
		APIKey:      args.APIKey,
		BaseURL:     args.BaseURL,
		Model:       args.Model,
		EnableTools: args.Tools,
		MCPURL:      args.MCPURL,
		DocsFolder:  args.DocsFolder,
	}

	cfg := config.CreateZeroConfig(opts)
	cfg.SetDefaults()

	// Validate zero-config too
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid zero-config: %w", err)
	}

	return cfg, nil
}

// ============================================================================
// API KEY HANDLING
// ============================================================================

// getOrRequireAPIKey retrieves API key from flag or environment
// Returns the API key or an error with clear instructions
func getOrRequireAPIKey(flagValue string) (string, error) {
	// Check flag first
	if flagValue != "" {
		return flagValue, nil
	}

	// Check environment variable
	if envKey := os.Getenv("OPENAI_API_KEY"); envKey != "" {
		return envKey, nil
	}

	// Not found - return descriptive error
	return "", fmt.Errorf(
		"OpenAI API key required for zero-config mode\n\n" +
			"Provide it via:\n" +
			"  1. Command line flag:     --api-key sk-...\n" +
			"  2. Environment variable:  export OPENAI_API_KEY=sk-...\n\n" +
			"Or create a hector.yaml configuration file for custom setups")
}

// ============================================================================
// UTILITIES
// ============================================================================

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
