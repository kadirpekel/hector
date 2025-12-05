// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel
//
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.gnu.org/licenses/agpl-3.0.en.html
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"strings"
)

// zeroConfigFlagNames are all the flag names that indicate zero-config mode.
// Ported from legacy pkg/cli/mutual_exclusivity.go with pkg-specific flags.
//
// pkg adaptations:
// - Added --storage, --storage-db (pkg storage options)
// - Added --rag-watch, --mcp-parser-tool (pkg RAG options)
// - Removed --vectordb (pkg uses different approach)
var zeroConfigFlagNames = []string{
	// LLM options
	"--provider",
	"--model",
	"--api-key",
	"--base-url",
	"--temperature",
	"--max-tokens",

	// Agent options
	"--role",
	"--instruction",

	// Tool options
	"--tools",
	"--approve-tools",
	"--no-approve-tools",
	"--mcp-url",

	// Thinking options
	"--thinking",
	"--no-thinking",
	"--thinking-budget",

	// Storage options (pkg)
	"--storage",
	"--storage-db",

	// Observability
	"--observe",

	// RAG options (pkg)
	"--docs-folder",
	"--embedder-model",
	"--rag-watch",
	"--no-rag-watch",
	"--mcp-parser-tool",
}

// booleanFlags are flags that don't require a value.
// Used to correctly detect flag presence.
var booleanFlags = map[string]bool{
	"--tools":        true, // Can be boolean (presence) or string (tool list)
	"--observe":      true,
	"--thinking":     true,
	"--no-thinking":  true,
	"--rag-watch":    true,
	"--no-rag-watch": true,
}

// ValidateConfigMutualExclusivity checks if --config and zero-config flags are mutually exclusive.
// This is called early with raw command-line arguments, before any processing.
// Ported line-by-line from legacy pkg/cli/mutual_exclusivity.go.
func ValidateConfigMutualExclusivity(args []string) error {
	hasConfig := false
	var zeroConfigFlags []string

	for i, arg := range args {
		// Check for --config or -c (both --flag and --flag=value syntax)
		// Ported from legacy: handles both formats
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
				if booleanFlags[zcFlag] {
					value = "true"
				} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					value = args[i+1]
				}
			}

			if matched {
				// Redact sensitive values (ported from legacy)
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

// buildMutualExclusivityError creates a helpful error message.
// Ported line-by-line from legacy pkg/cli/mutual_exclusivity.go.
// pkg adaptation: Updated examples to use pkg command structure.
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
	sb.WriteString("     Example: hector serve --config myconfig.yaml\n\n")

	sb.WriteString("  2. Zero-config mode:\n")
	sb.WriteString("     Remove --config and use zero-config flags\n")
	sb.WriteString("     Example: hector serve --provider anthropic --model claude-sonnet-4-20250514\n\n")

	sb.WriteString("INFO: Zero-config mode: Quick setup via CLI flags (no config file needed)\n")
	sb.WriteString("      Config file mode: Full control with YAML configuration\n\n")

	return fmt.Errorf("%s", sb.String())
}

// ShouldSkipValidation checks if the command should skip mutual exclusivity validation.
// Ported from legacy pkg/cli/mutual_exclusivity.go.
// pkg adaptation: Updated skip commands for pkg command structure.
func ShouldSkipValidation(args []string) bool {
	// Skip validation for commands that don't support zero-config mode
	skipCommands := []string{"validate", "version", "info", "help", "--help", "-h"}

	for _, arg := range args {
		for _, skipCmd := range skipCommands {
			if arg == skipCmd {
				return true
			}
		}
	}

	return false
}

// FormatConfigError formats a config processing error in a user-friendly way.
// This function is reused by other commands to provide better error messages
// when config loading/validation fails.
// Ported line-by-line from legacy pkg/cli/validate_command.go.
// pkg adaptation: Updated error message patterns for pkg's error format.
func FormatConfigError(configPath string, err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// Extract the core error message from validation errors
	// pkg adaptation: pkg uses different error format than legacy's ProcessConfigPipeline
	if strings.Contains(errMsg, "validation failed:") {
		parts := strings.SplitN(errMsg, "validation failed: ", 2)
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

	// For reference errors (pkg specific)
	if strings.Contains(errMsg, "reference errors:") {
		return fmt.Sprintf("Configuration reference error in %s:\n  %s\n\nHint: Check that all referenced llms, embedders, and databases are defined",
			configPath, errMsg)
	}

	// For other config errors
	return fmt.Sprintf("Configuration error in %s:\n  %s\n\nHint: Run 'hector validate %s' for detailed diagnostics",
		configPath, errMsg, configPath)
}

// Fatalf prints a formatted error message to stderr and exits with code 1.
// Ported from legacy pkg/cli/validation.go.
func Fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
