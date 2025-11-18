package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kadirpekel/hector/pkg/config"
	"gopkg.in/yaml.v3"
)

type ValidateCmd struct {
	Config      string `arg:"" name:"config" help:"Configuration file path" placeholder:"PATH"`
	Format      string `short:"f" help:"Output format: compact, verbose, json" default:"compact" enum:"compact,verbose,json"`
	PrintConfig bool   `short:"p" name:"print-config" help:"Print the expanded configuration (with defaults, shortcuts, and env vars resolved)"`
}

func ValidateCommand(args *ValidateCmd) error {
	// Load configuration using file type
	configType, err := config.ParseConfigType("file")
	if err != nil {
		return fmt.Errorf("invalid config type: %w", err)
	}

	loaderOpts := config.LoaderOptions{
		Type: configType,
		Path: args.Config,
	}

	cfg, err := config.LoadConfig(loaderOpts)
	if err != nil {
		return printLoadError(args.Format, args.Config, err)
	}

	// Use the same config processing pipeline as the main application
	// This expands shortcuts (docs_folder, enable_tools), sets defaults, and validates
	// Note: ProcessConfigPipeline calls cfg.Validate() internally - single source of truth!
	// The loader now also does strict validation to catch typos and unknown fields
	cfg, err = config.ProcessConfigPipeline(cfg)
	if err != nil {
		return printProcessError(args.Format, args.Config, err)
	}

	// If --print-config is specified, print the expanded configuration
	if args.PrintConfig {
		return printExpandedConfig(args.Format, args.Config, cfg)
	}

	// Success - configuration is valid (ProcessConfigPipeline already validated it)
	printSuccess(args.Format, args.Config)
	return nil
}

type ValidationError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func printLoadError(format, file string, err error) error {
	switch format {
	case "json":
		printJSONResult(false, file, []ValidationError{{Type: "load", Message: err.Error()}})
	case "verbose":
		fmt.Fprintf(os.Stderr, "Configuration Load Error\n")
		fmt.Fprintf(os.Stderr, "========================\n\n")
		fmt.Fprintf(os.Stderr, "File:    %s\n", file)
		fmt.Fprintf(os.Stderr, "Error:   %s\n", err.Error())
	default:
		fmt.Fprintf(os.Stderr, "%s: load error: %s\n", file, err.Error())
	}
	return fmt.Errorf("config load failed")
}

func printProcessError(format, file string, err error) error {
	switch format {
	case "json":
		printJSONResult(false, file, []ValidationError{{Type: "process", Message: err.Error()}})
	case "verbose":
		fmt.Fprintf(os.Stderr, "Configuration Processing Error\n")
		fmt.Fprintf(os.Stderr, "==============================\n\n")
		fmt.Fprintf(os.Stderr, "File:    %s\n", file)
		fmt.Fprintf(os.Stderr, "Error:   %s\n", err.Error())
	default:
		fmt.Fprintf(os.Stderr, "%s: process error: %s\n", file, err.Error())
	}
	return fmt.Errorf("config processing failed")
}

func printSuccess(format, file string) {
	switch format {
	case "json":
		printJSONResult(true, file, nil)
	case "verbose":
		fmt.Fprintf(os.Stdout, "Configuration Validation Successful\n")
		fmt.Fprintf(os.Stdout, "===================================\n\n")
		fmt.Fprintf(os.Stdout, "File:   %s\n", file)
		fmt.Fprintf(os.Stdout, "Status: OK Valid\n")
	default:
		fmt.Fprintf(os.Stdout, "%s: valid\n", file)
	}
}

func printExpandedConfig(format, file string, cfg *config.Config) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(cfg); err != nil {
			return fmt.Errorf("failed to encode config as JSON: %w", err)
		}
	case "verbose", "compact":
		// Use YAML for human-readable output (both verbose and compact use same format)
		fmt.Fprintf(os.Stdout, "# Expanded Configuration from: %s\n", file)
		fmt.Fprintf(os.Stdout, "# (shortcuts expanded, defaults applied, env vars resolved)\n\n")

		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(2)
		if err := encoder.Encode(cfg); err != nil {
			return fmt.Errorf("failed to encode config as YAML: %w", err)
		}
		encoder.Close()
	}
	return nil
}

type jsonOutput struct {
	Valid  bool              `json:"valid"`
	File   string            `json:"file"`
	Errors []ValidationError `json:"errors,omitempty"`
}

func printJSONResult(valid bool, file string, errors []ValidationError) {
	output := jsonOutput{
		Valid:  valid,
		File:   file,
		Errors: errors,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}

// FormatConfigError formats a config processing error in a user-friendly way
// This function is reused by other commands to provide better error messages
// when config loading/validation fails
func FormatConfigError(configPath string, err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// Extract the core error message from ProcessConfigPipeline errors
	if strings.Contains(errMsg, "ProcessConfigPipeline: validation failed:") {
		parts := strings.SplitN(errMsg, "ProcessConfigPipeline: validation failed: ", 2)
		if len(parts) == 2 {
			return fmt.Sprintf("Configuration validation failed in %s:\n  %s\n\nHint: Run 'hector validate %s' for detailed diagnostics",
				configPath, parts[1], configPath)
		}
	}

	// For load errors
	if strings.Contains(errMsg, "failed to load config:") || strings.Contains(errMsg, "failed to unmarshal") {
		return fmt.Sprintf("Configuration load error in %s:\n  %s\n\nHint: Run 'hector validate %s' to see the full error details",
			configPath, errMsg, configPath)
	}

	// For other config errors
	return fmt.Sprintf("Configuration error in %s:\n  %s\n\nHint: Run 'hector validate %s' for detailed diagnostics",
		configPath, errMsg, configPath)
}
