package main

import (
	"fmt"
	"os"

	"github.com/verikod/hector/pkg/config"
)

// InitCmd creates a new app config file.
type InitCmd struct {
	Output   string `short:"o" help:"Output file path" default:".hector/config.yaml"`
	Name     string `help:"App name" default:"my-app"`
	Template string `help:"Template to use (minimal, full)" default:"minimal"`

	// Optional quick-start parameters
	Provider string `help:"LLM provider (anthropic, openai, gemini, ollama)"`
	Model    string `help:"Model name"`
}

func (c *InitCmd) Run() error {
	// Check if file already exists
	if _, err := os.Stat(c.Output); err == nil {
		return fmt.Errorf("config file already exists: %s\nUse a different --output path or remove the existing file", c.Output)
	}

	// Use existing EnsureConfigExists from config_generator.go
	opts := config.CLIOptions{
		Provider: c.Provider,
		Model:    c.Model,
	}

	result, err := config.EnsureConfigExists(opts, c.Output)
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	// Load the created config to show info
	appCfg, err := config.LoadAppConfig(c.Output)
	if err != nil {
		return fmt.Errorf("failed to load created config: %w", err)
	}

	fmt.Printf("✅ Created app config: %s\n", c.Output)
	fmt.Printf("   Name: %s\n", appCfg.Name)
	fmt.Printf("   Agents: %d\n", len(appCfg.Agents))
	fmt.Printf("   LLMs: %d\n", len(appCfg.LLMs))
	if result.SkillUsed {
		fmt.Printf("   (Generated from SKILL.md)\n")
	}
	fmt.Printf("\nEdit the config file to customize your app.\n")
	fmt.Printf("Then run: hector serve\n")

	return nil
}

// ValidateCmd validates an app config file.
type ValidateCmd struct {
	Config string `short:"c" help:"Path to config file" type:"path" default:".hector/config.yaml"`
}

func (c *ValidateCmd) Run() error {
	// Load config
	appCfg, err := config.LoadAppConfig(c.Config)
	if err != nil {
		return fmt.Errorf("❌ Invalid config: %w", err)
	}

	// Validate config
	if err := config.ValidateAppConfig(appCfg); err != nil {
		return fmt.Errorf("❌ Validation failed: %w", err)
	}

	fmt.Printf("✅ Config is valid: %s\n", c.Config)
	fmt.Printf("   Name: %s\n", appCfg.Name)
	fmt.Printf("   Agents: %d\n", len(appCfg.Agents))
	fmt.Printf("   LLMs: %d\n", len(appCfg.LLMs))
	fmt.Printf("   Tools: %d\n", len(appCfg.Tools))

	return nil
}
