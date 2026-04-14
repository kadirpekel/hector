package input_test

import (
	"context"
	"strings"
	"testing"

	"github.com/verikod/hector/pkg/guardrails"
	"github.com/verikod/hector/pkg/guardrails/input"
)

// =============================================================================
// Sanitizer Tests
// =============================================================================

func TestSanitizer_Name(t *testing.T) {
	sanitizer := input.NewSanitizer()
	if name := sanitizer.Name(); name != "input_sanitizer" {
		t.Errorf("Name() = %q, want %q", name, "input_sanitizer")
	}
}

func TestSanitizer_TrimWhitespace(t *testing.T) {
	ctx := context.Background()
	sanitizer := input.NewSanitizer().TrimWhitespace(true)

	tests := []struct {
		name         string
		input        string
		wantModified bool
		wantResult   string
	}{
		{"leading spaces", "   hello", true, "hello"},
		{"trailing spaces", "hello   ", true, "hello"},
		{"both sides", "   hello   ", true, "hello"},
		{"mixed whitespace", "\t\n hello \r\n", true, "hello"},
		{"no whitespace", "hello", false, "hello"},
		{"only whitespace", "   \t\n   ", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizer.Check(ctx, tt.input)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if tt.wantModified {
				if result.Action != guardrails.ActionModify {
					t.Errorf("expected modify action, got %v", result.Action)
				}
				if modified, ok := result.Modified.(string); ok {
					if modified != tt.wantResult {
						t.Errorf("Modified = %q, want %q", modified, tt.wantResult)
					}
				} else {
					t.Error("Modified should be a string")
				}
			} else {
				if result.Action != guardrails.ActionAllow {
					t.Errorf("expected allow action (no change), got %v", result.Action)
				}
			}
		})
	}
}

func TestSanitizer_TrimWhitespaceDisabled(t *testing.T) {
	ctx := context.Background()
	sanitizer := input.NewSanitizer().TrimWhitespace(false)

	result, _ := sanitizer.Check(ctx, "   hello   ")

	// Whitespace should not be trimmed, but control chars might still be processed
	// The specific behavior depends on other settings
	if result.Action == guardrails.ActionModify {
		if modified, ok := result.Modified.(string); ok {
			// If modified, check that leading/trailing spaces are preserved
			if !strings.HasPrefix(modified, " ") {
				t.Error("leading spaces should be preserved when trim disabled")
			}
		}
	}
}

func TestSanitizer_StripHTML(t *testing.T) {
	ctx := context.Background()
	sanitizer := input.NewSanitizer().
		TrimWhitespace(false).
		StripHTML(true)

	tests := []struct {
		name         string
		input        string
		wantModified bool
		wantContains string // What the result should contain
		wantExcludes string // What the result should NOT contain
	}{
		{
			name:         "simple tag",
			input:        "<b>bold</b>",
			wantModified: true,
			wantContains: "bold",
			wantExcludes: "<b>",
		},
		{
			name:         "script tag",
			input:        "Hello<script>alert('xss')</script>World",
			wantModified: true,
			wantContains: "Hello", // Tags stripped, content between preserved
			wantExcludes: "<script>",
		},

		{
			name:         "nested tags",
			input:        "<div><p>text</p></div>",
			wantModified: true,
			wantContains: "text",
			wantExcludes: "<div>",
		},
		{
			name:         "no HTML",
			input:        "plain text",
			wantModified: false,
			wantContains: "plain text",
		},
		{
			name:         "self-closing tag",
			input:        "Line 1<br/>Line 2",
			wantModified: true,
			wantContains: "Line 1Line 2",
			wantExcludes: "<br/>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizer.Check(ctx, tt.input)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			var output string
			if result.Action == guardrails.ActionModify {
				output = result.Modified.(string)
			} else {
				output = tt.input
			}

			if tt.wantModified && result.Action != guardrails.ActionModify {
				t.Errorf("expected modify action, got %v", result.Action)
			}
			if !strings.Contains(output, tt.wantContains) {
				t.Errorf("output %q should contain %q", output, tt.wantContains)
			}
			if tt.wantExcludes != "" && strings.Contains(output, tt.wantExcludes) {
				t.Errorf("output %q should not contain %q", output, tt.wantExcludes)
			}
		})
	}
}

func TestSanitizer_MaxLength(t *testing.T) {
	ctx := context.Background()
	sanitizer := input.NewSanitizer().
		TrimWhitespace(false).
		MaxLength(10)

	tests := []struct {
		name         string
		input        string
		wantModified bool
		wantLen      int
	}{
		{"under limit", "hello", false, 5},
		{"at limit", "helloworld", false, 10},
		{"over limit", "hello world!", true, 10},
		{"way over limit", "this is a very long string", true, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizer.Check(ctx, tt.input)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			var output string
			if result.Action == guardrails.ActionModify {
				output = result.Modified.(string)
			} else {
				output = tt.input
			}

			if len(output) != tt.wantLen {
				t.Errorf("len(output) = %d, want %d", len(output), tt.wantLen)
			}
			if tt.wantModified && result.Action != guardrails.ActionModify {
				t.Errorf("expected modify action, got %v", result.Action)
			}
		})
	}
}

func TestSanitizer_ControlCharacters(t *testing.T) {
	ctx := context.Background()
	sanitizer := input.NewSanitizer().TrimWhitespace(false)

	tests := []struct {
		name         string
		input        string
		wantModified bool
		desc         string
	}{
		{
			name:         "null byte",
			input:        "hello\x00world",
			wantModified: true,
			desc:         "null byte should be removed",
		},
		{
			name:         "bell character",
			input:        "hello\x07world",
			wantModified: true,
			desc:         "bell should be removed",
		},
		{
			name:         "backspace",
			input:        "hello\x08world",
			wantModified: true,
			desc:         "backspace should be removed",
		},
		{
			name:         "preserved newline",
			input:        "hello\nworld",
			wantModified: false,
			desc:         "newline should be preserved",
		},
		{
			name:         "preserved tab",
			input:        "hello\tworld",
			wantModified: false,
			desc:         "tab should be preserved",
		},
		{
			name:         "preserved carriage return",
			input:        "hello\rworld",
			wantModified: false,
			desc:         "carriage return should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizer.Check(ctx, tt.input)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if tt.wantModified && result.Action != guardrails.ActionModify {
				t.Errorf("[%s] expected modify, got %v", tt.desc, result.Action)
			}
			if !tt.wantModified && result.Action == guardrails.ActionModify {
				t.Errorf("[%s] expected no modification, got modify", tt.desc)
			}
		})
	}
}

func TestSanitizer_NormalizeUnicode(t *testing.T) {
	ctx := context.Background()
	sanitizer := input.NewSanitizer().
		TrimWhitespace(false).
		NormalizeUnicode(true)

	// Test with decomposed vs composed Unicode
	// é can be: U+00E9 (composed) or U+0065 U+0301 (decomposed)
	decomposed := "e\u0301" // e + combining acute accent
	composed := "é"         // single character

	result, _ := sanitizer.Check(ctx, decomposed)

	if result.Action != guardrails.ActionModify {
		t.Error("expected modification for decomposed Unicode")
	}

	if modified, ok := result.Modified.(string); ok {
		if modified != composed {
			t.Errorf("normalized = %q (len %d), want %q (len %d)",
				modified, len(modified), composed, len(composed))
		}
	}
}

func TestSanitizer_CombinedOperations(t *testing.T) {
	ctx := context.Background()
	sanitizer := input.NewSanitizer().
		TrimWhitespace(true).
		StripHTML(true).
		MaxLength(20)

	input := "   <b>Hello</b> World!   "
	result, err := sanitizer.Check(ctx, input)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if result.Action != guardrails.ActionModify {
		t.Fatal("expected modify action")
	}

	output := result.Modified.(string)

	// Should be trimmed, HTML stripped
	if strings.HasPrefix(output, " ") || strings.HasSuffix(output, " ") {
		t.Error("should be trimmed")
	}
	if strings.Contains(output, "<b>") {
		t.Error("should have HTML stripped")
	}
	if len(output) > 20 {
		t.Error("should be truncated to max length")
	}
}

func TestSanitizer_NoChanges(t *testing.T) {
	ctx := context.Background()
	sanitizer := input.NewSanitizer().
		TrimWhitespace(true).
		StripHTML(true)

	// Input that needs no modification
	input := "clean text"
	result, err := sanitizer.Check(ctx, input)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if result.Action != guardrails.ActionAllow {
		t.Errorf("expected allow for clean input, got %v", result.Action)
	}
}

func TestSanitizer_InterfaceCompliance(t *testing.T) {
	var _ guardrails.InputGuardrail = input.NewSanitizer()
}
