package cli

import (
	"strings"
	"testing"
)

func TestValidateConfigMutualExclusivity(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectFlag  string
	}{
		{
			name:        "no config, no zero-config flags",
			args:        []string{"hector", "call", "hello"},
			expectError: false,
		},
		{
			name:        "config only",
			args:        []string{"hector", "call", "hello", "--config", "myconfig.yaml"},
			expectError: false,
		},
		{
			name:        "zero-config flags only",
			args:        []string{"hector", "call", "hello", "--provider", "anthropic"},
			expectError: false,
		},
		{
			name:        "config + provider",
			args:        []string{"hector", "call", "hello", "--config", "myconfig.yaml", "--provider", "anthropic"},
			expectError: true,
			expectFlag:  "--provider",
		},
		{
			name:        "config + model",
			args:        []string{"hector", "call", "hello", "--config", "myconfig.yaml", "--model", "gpt-4"},
			expectError: true,
			expectFlag:  "--model",
		},
		{
			name:        "config + tools",
			args:        []string{"hector", "call", "hello", "--config", "myconfig.yaml", "--tools"},
			expectError: true,
			expectFlag:  "--tools",
		},
		{
			name:        "config + multiple flags",
			args:        []string{"hector", "call", "hello", "--config", "myconfig.yaml", "--provider", "anthropic", "--model", "claude-3-5-sonnet-20241022", "--tools"},
			expectError: true,
			expectFlag:  "--provider",
		},
		{
			name:        "short config flag -c + provider",
			args:        []string{"hector", "call", "hello", "-c", "myconfig.yaml", "--provider", "anthropic"},
			expectError: true,
			expectFlag:  "--provider",
		},
		{
			name:        "config with = syntax",
			args:        []string{"hector", "call", "hello", "--config=myconfig.yaml", "--provider", "anthropic"},
			expectError: true,
			expectFlag:  "--provider",
		},
		{
			name:        "flag with = syntax",
			args:        []string{"hector", "call", "hello", "--config", "myconfig.yaml", "--provider=anthropic"},
			expectError: true,
			expectFlag:  "--provider",
		},
		{
			name:        "api-key is redacted",
			args:        []string{"hector", "call", "hello", "--config", "myconfig.yaml", "--api-key", "sk-12345"},
			expectError: true,
			expectFlag:  "[REDACTED]",
		},
		{
			name:        "docs-folder",
			args:        []string{"hector", "serve", "--config", "myconfig.yaml", "--docs-folder", "/path/to/docs"},
			expectError: true,
			expectFlag:  "--docs-folder",
		},
		{
			name:        "observe flag",
			args:        []string{"hector", "serve", "--config", "myconfig.yaml", "--observe"},
			expectError: true,
			expectFlag:  "--observe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfigMutualExclusivity(tt.args)

			if (err != nil) != tt.expectError {
				t.Errorf("ValidateConfigMutualExclusivity() error = %v, expectError = %v", err, tt.expectError)
				return
			}

			if err != nil {
				errMsg := err.Error()
				if !strings.Contains(errMsg, "Mutually Exclusive") {
					t.Errorf("error should mention mutual exclusivity, got: %v", errMsg)
				}
				if tt.expectFlag != "" && !strings.Contains(errMsg, tt.expectFlag) {
					t.Errorf("error should mention flag %q, got: %v", tt.expectFlag, errMsg)
				}
			}
		})
	}
}

func TestShouldSkipValidation(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		expect bool
	}{
		{
			name:   "validate command",
			args:   []string{"hector", "validate", "config.yaml"},
			expect: true,
		},
		{
			name:   "version command",
			args:   []string{"hector", "version"},
			expect: true,
		},
		{
			name:   "help flag",
			args:   []string{"hector", "--help"},
			expect: true,
		},
		{
			name:   "short help flag",
			args:   []string{"hector", "-h"},
			expect: true,
		},
		{
			name:   "call command - should not skip",
			args:   []string{"hector", "call", "hello"},
			expect: false,
		},
		{
			name:   "serve command - should not skip",
			args:   []string{"hector", "serve"},
			expect: false,
		},
		{
			name:   "chat command - should not skip",
			args:   []string{"hector", "chat"},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipValidation(tt.args)
			if result != tt.expect {
				t.Errorf("ShouldSkipValidation() = %v, expected %v", result, tt.expect)
			}
		})
	}
}

func TestBuildMutualExclusivityError(t *testing.T) {
	flags := []string{
		"--provider=anthropic",
		"--model=claude-3-5-sonnet-20241022",
		"--tools=true",
	}

	err := buildMutualExclusivityError(flags)

	if err == nil {
		t.Fatal("buildMutualExclusivityError() should return an error")
	}

	errMsg := err.Error()

	requiredPhrases := []string{
		"Mutually Exclusive",
		"--config",
		"zero-config flags",
		"--provider=anthropic",
		"--model=claude-3-5-sonnet-20241022",
		"--tools=true",
		"Choose one approach",
		"Config file mode",
		"Zero-config mode",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(errMsg, phrase) {
			t.Errorf("error message should contain %q, got:\n%v", phrase, errMsg)
		}
	}
}

func TestRealWorldScenarios(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		description string
	}{
		{
			name:        "typical config usage",
			args:        []string{"hector", "call", "hello world", "--config", "production.yaml"},
			expectError: false,
			description: "Using config file alone should work",
		},
		{
			name:        "typical zero-config usage",
			args:        []string{"hector", "call", "hello", "--provider", "anthropic", "--model", "claude-3-5-sonnet-20241022", "--tools"},
			expectError: false,
			description: "Using zero-config flags alone should work",
		},
		{
			name:        "mixed usage - development mistake",
			args:        []string{"hector", "serve", "--config", "prod.yaml", "--tools", "--docs-folder", "./docs"},
			expectError: true,
			description: "Accidentally mixing config with zero-config should fail",
		},
		{
			name:        "CI/CD with config",
			args:        []string{"hector", "serve", "--config", "/etc/hector/config.yaml"},
			expectError: false,
			description: "CI/CD using config file should work",
		},
		{
			name:        "quick testing with zero-config",
			args:        []string{"hector", "call", "test", "--provider", "openai", "--model", "gpt-4"},
			expectError: false,
			description: "Quick testing with zero-config should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfigMutualExclusivity(tt.args)
			if (err != nil) != tt.expectError {
				t.Errorf("%s: error = %v, expectError = %v", tt.description, err, tt.expectError)
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "empty args",
			args:        []string{},
			expectError: false,
		},
		{
			name:        "only command",
			args:        []string{"hector"},
			expectError: false,
		},
		{
			name:        "config value looks like flag",
			args:        []string{"hector", "call", "hello", "--config", "--provider"},
			expectError: true, // --provider after --config is treated as a zero-config flag
		},
		{
			name:        "multiple config flags (Kong will handle)",
			args:        []string{"hector", "call", "hello", "--config", "a.yaml", "--config", "b.yaml"},
			expectError: false, // Kong will handle duplicate flags
		},
		{
			name:        "flag after message argument",
			args:        []string{"hector", "call", "hello world", "--config", "a.yaml", "--provider", "anthropic"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfigMutualExclusivity(tt.args)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateConfigMutualExclusivity() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}
