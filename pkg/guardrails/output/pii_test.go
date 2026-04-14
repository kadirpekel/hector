package output_test

import (
	"context"
	"strings"
	"testing"

	"github.com/verikod/hector/pkg/guardrails"
	"github.com/verikod/hector/pkg/guardrails/output"
)

// =============================================================================
// PIIRedactor Tests
// =============================================================================

func TestPIIRedactor_Name(t *testing.T) {
	redactor := output.NewPIIRedactor()
	if name := redactor.Name(); name != "pii_redactor" {
		t.Errorf("Name() = %q, want %q", name, "pii_redactor")
	}
}

func TestPIIRedactor_EmailDetection(t *testing.T) {
	ctx := context.Background()
	redactor := output.NewPIIRedactor().
		DetectEmail(true).
		DetectPhone(false).
		DetectSSN(false).
		DetectCreditCard(false)

	tests := []struct {
		name       string
		input      string
		wantDetect bool
	}{
		{"simple email", "Contact john@example.com", true},
		{"email with subdomain", "Email admin@mail.company.com", true},
		{"email with plus", "Send to user+tag@gmail.com", true},
		{"no email", "Contact us at our website", false},
		{"invalid email", "This is not an@ email", false},
		{"multiple emails", "Send to a@b.com and c@d.org", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := redactor.Check(ctx, tt.input)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if tt.wantDetect {
				if result.Action != guardrails.ActionModify {
					t.Errorf("expected modify for detected email, got %v", result.Action)
				}
				// Verify email is redacted
				if modified, ok := result.Modified.(string); ok {
					if strings.Contains(modified, "@") {
						t.Error("redacted output should not contain @")
					}
					if !strings.Contains(modified, "[EMAIL_REDACTED]") {
						t.Error("redacted output should contain placeholder")
					}
				}
			} else {
				if result.Action != guardrails.ActionAllow {
					t.Errorf("expected allow when no email, got %v", result.Action)
				}
			}
		})
	}
}

func TestPIIRedactor_PhoneDetection(t *testing.T) {
	ctx := context.Background()
	redactor := output.NewPIIRedactor().
		DetectEmail(false).
		DetectPhone(true).
		DetectSSN(false).
		DetectCreditCard(false)

	tests := []struct {
		name       string
		input      string
		wantDetect bool
	}{
		{"standard format", "Call 555-123-4567", true},
		{"with parentheses", "Phone: (555) 123-4567", true},
		{"with dots", "Contact 555.123.4567", true},
		{"with country code", "Call +1-555-123-4567", true},
		{"no phone", "Call us soon", false},
		{"partial number", "Code is 123-45", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := redactor.Check(ctx, tt.input)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if tt.wantDetect {
				if result.Action != guardrails.ActionModify {
					t.Errorf("expected modify for detected phone, got %v", result.Action)
				}
				if modified, ok := result.Modified.(string); ok {
					if !strings.Contains(modified, "[PHONE_REDACTED]") {
						t.Error("redacted output should contain phone placeholder")
					}
				}
			} else {
				if result.Action != guardrails.ActionAllow {
					t.Errorf("expected allow when no phone, got %v", result.Action)
				}
			}
		})
	}
}

func TestPIIRedactor_SSNDetection(t *testing.T) {
	ctx := context.Background()
	redactor := output.NewPIIRedactor().
		DetectEmail(false).
		DetectPhone(false).
		DetectSSN(true).
		DetectCreditCard(false)

	tests := []struct {
		name       string
		input      string
		wantDetect bool
	}{
		{"standard SSN", "SSN: 123-45-6789", true},
		{"SSN with spaces", "SSN: 123 45 6789", true},
		{"SSN no dashes", "SSN: 123456789", true},
		{"no SSN", "ID: ABC123", false},
		{"partial SSN", "Code: 123-45", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := redactor.Check(ctx, tt.input)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if tt.wantDetect {
				if result.Action != guardrails.ActionModify {
					t.Errorf("expected modify for detected SSN, got %v", result.Action)
				}
				if modified, ok := result.Modified.(string); ok {
					if !strings.Contains(modified, "[SSN_REDACTED]") {
						t.Error("redacted output should contain SSN placeholder")
					}
				}
			} else {
				if result.Action != guardrails.ActionAllow {
					t.Errorf("expected allow when no SSN, got %v", result.Action)
				}
			}
		})
	}
}

func TestPIIRedactor_CreditCardDetection(t *testing.T) {
	ctx := context.Background()
	redactor := output.NewPIIRedactor().
		DetectEmail(false).
		DetectPhone(false).
		DetectSSN(false).
		DetectCreditCard(true)

	tests := []struct {
		name       string
		input      string
		wantDetect bool
	}{
		{"Visa 16 digit", "Card: 4111111111111111", true},
		{"Visa 13 digit", "Card: 4111111111111", true},
		{"Mastercard", "Card: 5111111111111118", true},
		{"Amex", "Amex: 371111111111114", true},
		{"Discover", "Card: 6011111111111117", true},
		{"no CC", "Amount: $100.00", false},
		{"partial number", "Code: 4111-1111", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := redactor.Check(ctx, tt.input)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if tt.wantDetect {
				if result.Action != guardrails.ActionModify {
					t.Errorf("expected modify for detected CC, got %v", result.Action)
				}
				if modified, ok := result.Modified.(string); ok {
					if !strings.Contains(modified, "[CREDIT_CARD_REDACTED]") {
						t.Error("redacted output should contain CC placeholder")
					}
				}
			} else {
				if result.Action != guardrails.ActionAllow {
					t.Errorf("expected allow when no CC, got %v", result.Action)
				}
			}
		})
	}
}

func TestPIIRedactor_RedactModes(t *testing.T) {
	ctx := context.Background()
	input := "Contact john@example.com for details"

	t.Run("mask mode", func(t *testing.T) {
		redactor := output.NewPIIRedactor().
			DetectEmail(true).
			RedactMode(guardrails.RedactModeMask)

		result, _ := redactor.Check(ctx, input)
		if result.Action != guardrails.ActionModify {
			t.Fatal("expected modify")
		}
		modified := result.Modified.(string)
		if !strings.Contains(modified, "[EMAIL_REDACTED]") {
			t.Errorf("mask mode should use [TYPE_REDACTED] format, got %q", modified)
		}
	})

	t.Run("remove mode", func(t *testing.T) {
		redactor := output.NewPIIRedactor().
			DetectEmail(true).
			RedactMode(guardrails.RedactModeRemove)

		result, _ := redactor.Check(ctx, input)
		if result.Action != guardrails.ActionModify {
			t.Fatal("expected modify")
		}
		modified := result.Modified.(string)
		if strings.Contains(modified, "@") {
			t.Error("email should be removed")
		}
		if strings.Contains(modified, "[") {
			t.Error("remove mode should not add placeholders")
		}
	})

	t.Run("hash mode", func(t *testing.T) {
		redactor := output.NewPIIRedactor().
			DetectEmail(true).
			RedactMode(guardrails.RedactModeHash)

		result, _ := redactor.Check(ctx, input)
		if result.Action != guardrails.ActionModify {
			t.Fatal("expected modify")
		}
		modified := result.Modified.(string)
		if !strings.Contains(modified, "[email:") {
			t.Errorf("hash mode should use [type:hash] format, got %q", modified)
		}
	})
}

func TestPIIRedactor_MultiplePII(t *testing.T) {
	ctx := context.Background()
	redactor := output.NewPIIRedactor().
		DetectEmail(true).
		DetectPhone(true)

	input := "Contact john@example.com or call 555-123-4567"
	result, err := redactor.Check(ctx, input)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if result.Action != guardrails.ActionModify {
		t.Fatal("expected modify")
	}

	modified := result.Modified.(string)
	if strings.Contains(modified, "@") {
		t.Error("email should be redacted")
	}
	if strings.Contains(modified, "555") {
		t.Error("phone should be redacted")
	}

	// Check details
	if result.Details == nil {
		t.Fatal("Details should not be nil")
	}
	if count, ok := result.Details["pii_count"].(int); !ok || count != 2 {
		t.Errorf("pii_count = %v, want 2", result.Details["pii_count"])
	}
}

func TestPIIRedactor_BlockMode(t *testing.T) {
	ctx := context.Background()
	redactor := output.NewPIIRedactor().
		DetectEmail(true).
		WithAction(guardrails.ActionBlock)

	input := "Contact john@example.com"
	result, _ := redactor.Check(ctx, input)

	if result.Action != guardrails.ActionBlock {
		t.Errorf("expected block, got %v", result.Action)
	}
	if result.Modified != nil {
		t.Error("block mode should not modify content")
	}
}

func TestPIIRedactor_NoPII(t *testing.T) {
	ctx := context.Background()
	redactor := output.NewPIIRedactor()

	result, _ := redactor.Check(ctx, "This is clean text with no PII")

	if result.Action != guardrails.ActionAllow {
		t.Errorf("expected allow for clean text, got %v", result.Action)
	}
}

func TestPIIRedactor_InterfaceCompliance(t *testing.T) {
	var _ guardrails.OutputGuardrail = output.NewPIIRedactor()
}

// =============================================================================
// ContentFilter Tests
// =============================================================================

func TestContentFilter_Name(t *testing.T) {
	filter := output.NewContentFilter()
	if name := filter.Name(); name != "content_filter" {
		t.Errorf("Name() = %q, want %q", name, "content_filter")
	}
}

func TestContentFilter_BlockKeywords(t *testing.T) {
	ctx := context.Background()
	filter := output.NewContentFilter().
		BlockKeywords("password", "secret", "api_key")

	tests := []struct {
		name      string
		input     string
		wantBlock bool
	}{
		{"contains password", "Your password is 12345", true},
		{"contains secret", "This is a secret message", true},
		{"contains api_key", "The api_key is xyz", true},
		{"case insensitive", "Your PASSWORD is safe", true},
		{"no keywords", "Hello world", false},
		{"partial match", "passwords are important", true}, // Contains "password"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filter.Check(ctx, tt.input)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if tt.wantBlock && result.Action != guardrails.ActionBlock {
				t.Errorf("expected block, got %v", result.Action)
			}
			if !tt.wantBlock && result.Action != guardrails.ActionAllow {
				t.Errorf("expected allow, got %v", result.Action)
			}
		})
	}
}

func TestContentFilter_BlockPatterns(t *testing.T) {
	ctx := context.Background()
	filter := output.NewContentFilter().
		BlockPatterns(
			`sk-[a-zA-Z0-9]{48}`,         // OpenAI API key pattern
			`ghp_[a-zA-Z0-9]{36}`,        // GitHub token pattern
			`(?i)bearer\s+[a-z0-9\-_.]+`, // Bearer token
		)

	tests := []struct {
		name      string
		input     string
		wantBlock bool
	}{
		{
			name:      "OpenAI key pattern",
			input:     "Key: sk-" + strings.Repeat("a", 48),
			wantBlock: true,
		},
		{
			name:      "GitHub token",
			input:     "Token: ghp_" + strings.Repeat("b", 36),
			wantBlock: true,
		},
		{
			name:      "Bearer token",
			input:     "Authorization: Bearer abc123-token",
			wantBlock: true,
		},
		{
			name:      "no pattern match",
			input:     "Normal content here",
			wantBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filter.Check(ctx, tt.input)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if tt.wantBlock && result.Action != guardrails.ActionBlock {
				t.Errorf("expected block, got %v", result.Action)
			}
			if !tt.wantBlock && result.Action != guardrails.ActionAllow {
				t.Errorf("expected allow, got %v", result.Action)
			}
		})
	}
}

func TestContentFilter_CombinedKeywordsAndPatterns(t *testing.T) {
	ctx := context.Background()
	filter := output.NewContentFilter().
		BlockKeywords("secret").
		BlockPatterns(`\d{4}-\d{4}-\d{4}-\d{4}`)

	tests := []struct {
		name      string
		input     string
		wantBlock bool
		matchType string
	}{
		{"keyword match", "This is secret", true, "keyword"},
		{"pattern match", "Card: 1234-5678-9012-3456", true, "pattern"},
		{"no match", "Hello world", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filter.Check(ctx, tt.input)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if tt.wantBlock && result.Action != guardrails.ActionBlock {
				t.Errorf("expected block, got %v", result.Action)
			}
			if !tt.wantBlock && result.Action != guardrails.ActionAllow {
				t.Errorf("expected allow, got %v", result.Action)
			}
		})
	}
}

func TestContentFilter_Details(t *testing.T) {
	ctx := context.Background()
	filter := output.NewContentFilter().
		BlockKeywords("secret")

	result, _ := filter.Check(ctx, "This is a secret message")

	if result.Action != guardrails.ActionBlock {
		t.Fatal("expected block")
	}
	if result.Details == nil {
		t.Fatal("Details should not be nil")
	}
	if result.Details["match_type"] != "keyword" {
		t.Errorf("match_type = %v, want keyword", result.Details["match_type"])
	}
	if result.Details["matched"] != "secret" {
		t.Errorf("matched = %v, want secret", result.Details["matched"])
	}
}

func TestContentFilter_ActionAndSeverity(t *testing.T) {
	ctx := context.Background()
	filter := output.NewContentFilter().
		BlockKeywords("test").
		WithAction(guardrails.ActionWarn).
		WithSeverity(guardrails.SeverityLow)

	result, _ := filter.Check(ctx, "This is a test")

	if result.Action != guardrails.ActionWarn {
		t.Errorf("Action = %v, want %v", result.Action, guardrails.ActionWarn)
	}
	if result.Severity != guardrails.SeverityLow {
		t.Errorf("Severity = %v, want %v", result.Severity, guardrails.SeverityLow)
	}
}

func TestContentFilter_EmptyFilter(t *testing.T) {
	ctx := context.Background()
	filter := output.NewContentFilter() // No keywords or patterns

	result, _ := filter.Check(ctx, "Any content")

	if result.Action != guardrails.ActionAllow {
		t.Errorf("empty filter should allow all, got %v", result.Action)
	}
}

func TestContentFilter_InterfaceCompliance(t *testing.T) {
	var _ guardrails.OutputGuardrail = output.NewContentFilter()
}
