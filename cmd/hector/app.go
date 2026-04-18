package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/verikod/hector/pkg/config"
)

// InitCmd creates a new app config file.
type InitCmd struct {
	Output string `short:"o" help:"Output file path" default:".hector/config.yaml"`

	// LLM quick-start
	Provider string `help:"LLM provider (anthropic, openai, gemini, ollama)"`
	Model    string `help:"Model name"`
	APIKey   string `help:"API key for the LLM provider"`
	BaseURL  string `help:"Base URL for the LLM provider"`

	// Agent quick-start
	Instruction string `short:"i" help:"System instruction for the agent"`
	Tools       string `help:"Comma-separated tools to enable (e.g., bash,text_editor) or 'all'"`
	MCPURL      string `name:"mcp-url" help:"MCP server URL"`
}

func (c *InitCmd) Run() error {
	// Fail if config already exists (init = create, not ensure)
	if _, err := os.Stat(c.Output); err == nil {
		return fmt.Errorf("config file already exists: %s\nUse a different --output path or remove the existing file", c.Output)
	}

	// Build CLIOptions from flags
	opts := config.CLIOptions{
		Provider:    c.Provider,
		Model:       c.Model,
		APIKey:      c.APIKey,
		BaseURL:     c.BaseURL,
		Instruction: c.Instruction,
		Tools:       c.Tools,
		MCPURL:      c.MCPURL,
	}

	// Auto-detect SKILL.md in workspace root
	configDir := filepath.Dir(c.Output)
	workspaceRoot := configDir
	if filepath.Base(configDir) == ".hector" {
		workspaceRoot = filepath.Dir(configDir)
	}
	skillPath := filepath.Join(workspaceRoot, "SKILL.md")
	if _, err := os.Stat(skillPath); err == nil {
		opts.SkillFile = skillPath
	}

	// Generate config
	result, err := config.GenerateLeanConfig(opts, c.Output)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	// Write config file
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.WriteFile(c.Output, result.ConfigYAML, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Write .env.example if useful
	envPath := filepath.Join(workspaceRoot, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) && len(result.EnvVars) > 0 {
		if envContent := config.GenerateEnvExample(result.EnvVars); envContent != nil {
			envExamplePath := filepath.Join(workspaceRoot, ".env.example")
			if _, err := os.Stat(envExamplePath); os.IsNotExist(err) {
				_ = os.WriteFile(envExamplePath, envContent, 0644)
			}
		}
	}

	// Display info from result directly (no reload needed)
	cfg := result.AppConfig
	cfg.SetDefaults()

	fmt.Printf("✅ Created app config: %s\n", c.Output)
	fmt.Printf("   Agents: %d\n", len(cfg.Agents))
	fmt.Printf("   LLMs: %d\n", len(cfg.LLMs))
	fmt.Printf("   Tools: %d\n", len(cfg.Tools))
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
	fmt.Printf("   Agents: %d\n", len(appCfg.Agents))
	fmt.Printf("   LLMs: %d\n", len(appCfg.LLMs))
	fmt.Printf("   Tools: %d\n", len(appCfg.Tools))

	return nil
}
