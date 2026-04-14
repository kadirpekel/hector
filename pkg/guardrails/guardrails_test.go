package guardrails_test

import (
	"testing"

	"github.com/verikod/hector/pkg/guardrails"
)

// =============================================================================
// Result Tests
// =============================================================================

func TestResult_IsBlocking(t *testing.T) {
	tests := []struct {
		name   string
		action guardrails.Action
		want   bool
	}{
		{"block action is blocking", guardrails.ActionBlock, true},
		{"allow action is not blocking", guardrails.ActionAllow, false},
		{"modify action is not blocking", guardrails.ActionModify, false},
		{"warn action is not blocking", guardrails.ActionWarn, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &guardrails.Result{Action: tt.action}
			if got := result.IsBlocking(); got != tt.want {
				t.Errorf("IsBlocking() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResult_IsAllowed(t *testing.T) {
	tests := []struct {
		name   string
		action guardrails.Action
		want   bool
	}{
		{"allow action is allowed", guardrails.ActionAllow, true},
		{"modify action is allowed", guardrails.ActionModify, true},
		{"warn action is allowed", guardrails.ActionWarn, true},
		{"block action is not allowed", guardrails.ActionBlock, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &guardrails.Result{Action: tt.action}
			if got := result.IsAllowed(); got != tt.want {
				t.Errorf("IsAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestAllow(t *testing.T) {
	result := guardrails.Allow("test_guardrail")

	if result.Action != guardrails.ActionAllow {
		t.Errorf("Action = %v, want %v", result.Action, guardrails.ActionAllow)
	}
	if result.GuardrailName != "test_guardrail" {
		t.Errorf("GuardrailName = %q, want %q", result.GuardrailName, "test_guardrail")
	}
	if result.Reason != "" {
		t.Errorf("Reason should be empty, got %q", result.Reason)
	}
}

func TestBlock(t *testing.T) {
	result := guardrails.Block("test_guardrail", "blocked for testing", guardrails.SeverityHigh)

	if result.Action != guardrails.ActionBlock {
		t.Errorf("Action = %v, want %v", result.Action, guardrails.ActionBlock)
	}
	if result.GuardrailName != "test_guardrail" {
		t.Errorf("GuardrailName = %q, want %q", result.GuardrailName, "test_guardrail")
	}
	if result.Reason != "blocked for testing" {
		t.Errorf("Reason = %q, want %q", result.Reason, "blocked for testing")
	}
	if result.Severity != guardrails.SeverityHigh {
		t.Errorf("Severity = %v, want %v", result.Severity, guardrails.SeverityHigh)
	}
}

func TestModify(t *testing.T) {
	modified := "modified content"
	result := guardrails.Modify("test_guardrail", modified, "content was modified")

	if result.Action != guardrails.ActionModify {
		t.Errorf("Action = %v, want %v", result.Action, guardrails.ActionModify)
	}
	if result.GuardrailName != "test_guardrail" {
		t.Errorf("GuardrailName = %q, want %q", result.GuardrailName, "test_guardrail")
	}
	if result.Reason != "content was modified" {
		t.Errorf("Reason = %q, want %q", result.Reason, "content was modified")
	}
	if result.Modified != modified {
		t.Errorf("Modified = %v, want %v", result.Modified, modified)
	}
}

func TestWarn(t *testing.T) {
	result := guardrails.Warn("test_guardrail", "warning message", guardrails.SeverityMedium)

	if result.Action != guardrails.ActionWarn {
		t.Errorf("Action = %v, want %v", result.Action, guardrails.ActionWarn)
	}
	if result.GuardrailName != "test_guardrail" {
		t.Errorf("GuardrailName = %q, want %q", result.GuardrailName, "test_guardrail")
	}
	if result.Reason != "warning message" {
		t.Errorf("Reason = %q, want %q", result.Reason, "warning message")
	}
	if result.Severity != guardrails.SeverityMedium {
		t.Errorf("Severity = %v, want %v", result.Severity, guardrails.SeverityMedium)
	}
}

// =============================================================================
// Constants Tests
// =============================================================================

func TestActionConstants(t *testing.T) {
	// Verify string values for serialization compatibility
	tests := []struct {
		action guardrails.Action
		want   string
	}{
		{guardrails.ActionAllow, "allow"},
		{guardrails.ActionBlock, "block"},
		{guardrails.ActionModify, "modify"},
		{guardrails.ActionWarn, "warn"},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if string(tt.action) != tt.want {
				t.Errorf("Action constant = %q, want %q", string(tt.action), tt.want)
			}
		})
	}
}

func TestSeverityConstants(t *testing.T) {
	// Verify string values for serialization compatibility
	tests := []struct {
		severity guardrails.Severity
		want     string
	}{
		{guardrails.SeverityLow, "low"},
		{guardrails.SeverityMedium, "medium"},
		{guardrails.SeverityHigh, "high"},
		{guardrails.SeverityCritical, "critical"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			if string(tt.severity) != tt.want {
				t.Errorf("Severity constant = %q, want %q", string(tt.severity), tt.want)
			}
		})
	}
}
