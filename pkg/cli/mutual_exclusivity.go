package cli

import (
	"fmt"
	"strings"
)

// zeroConfigFlagNames are all the flag names that indicate zero-config mode
var zeroConfigFlagNames = []string{
	"--provider",
	"--model",
	"--api-key",
	"--base-url",
	"--role",
	"--instruction",
	"--tools",
	"--mcp-url",
	"--docs-folder",
	"--embedder-model",
	"--vectordb",
	"--observe",
}

// ValidateConfigMutualExclusivity checks if --config and zero-config flags are mutually exclusive
// This is called early with raw command-line arguments, before any processing
func ValidateConfigMutualExclusivity(args []string) error {
	hasConfig := false
	var zeroConfigFlags []string

	for i, arg := range args {
		// Check for --config or -c (both --flag and --flag=value syntax)
		if arg == "--config" || arg == "-c" ||
			strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "-c=") {
			hasConfig = true
			continue
		}

		// Check for zero-config flags
		for _, zcFlag := range zeroConfigFlagNames {
			var matched bool
			var value string

			if strings.HasPrefix(arg, zcFlag+"=") {
				// Handle --flag=value format
				matched = true
				parts := strings.SplitN(arg, "=", 2)
				if len(parts) == 2 {
					value = parts[1]
				}
			} else if arg == zcFlag {
				// Handle --flag value format
				matched = true
				// For boolean flags, they don't need a value
				if zcFlag == "--tools" || zcFlag == "--observe" {
					value = "true"
				} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					value = args[i+1]
				}
			}

			if matched {
				// Redact sensitive values
				if zcFlag == "--api-key" && value != "" {
					value = "[REDACTED]"
				}
				zeroConfigFlags = append(zeroConfigFlags, fmt.Sprintf("%s=%s", zcFlag, value))
				break // Found this flag, move to next arg
			}
		}
	}

	// Return error if both config and zero-config flags are present
	if hasConfig && len(zeroConfigFlags) > 0 {
		return buildMutualExclusivityError(zeroConfigFlags)
	}

	return nil
}

// buildMutualExclusivityError creates a helpful error message
func buildMutualExclusivityError(flags []string) error {
	var sb strings.Builder

	sb.WriteString("\nERROR: Configuration Error: Mutually Exclusive Options\n\n")
	sb.WriteString("You cannot use --config with zero-config flags at the same time.\n\n")

	sb.WriteString("Detected zero-config flags:\n")
	for _, flag := range flags {
		sb.WriteString(fmt.Sprintf("  • %s\n", flag))
	}
	sb.WriteString("\n")

	sb.WriteString("TIP: Choose one approach:\n\n")
	sb.WriteString("  1. Config file mode:\n")
	sb.WriteString("     Remove zero-config flags and use --config only\n")
	sb.WriteString("     Example: hector call \"message\" --config myconfig.yaml\n\n")

	sb.WriteString("  2. Zero-config mode:\n")
	sb.WriteString("     Remove --config and use zero-config flags\n")
	sb.WriteString("     Example: hector call \"message\" --provider anthropic --model claude-3-5-sonnet-20241022\n\n")

	sb.WriteString("INFO: Zero-config mode: Quick setup via CLI flags (no config file needed)\n")
	sb.WriteString("   Config file mode: Full control with YAML configuration\n\n")

	return fmt.Errorf("%s", sb.String())
}

// ShouldSkipValidation checks if the command should skip mutual exclusivity validation
func ShouldSkipValidation(args []string) bool {
	// Skip validation for commands that don't support zero-config mode
	skipCommands := []string{"validate", "version", "help", "--help", "-h"}

	for _, arg := range args {
		for _, skipCmd := range skipCommands {
			if arg == skipCmd {
				return true
			}
		}
	}

	return false
}
