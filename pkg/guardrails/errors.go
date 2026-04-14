package guardrails

import "fmt"

// GuardrailError represents an error from a guardrail check.
type GuardrailError struct {
	GuardrailName string
	Reason        string
	Severity      Severity
	Details       map[string]any
}

func (e *GuardrailError) Error() string {
	return fmt.Sprintf("guardrail %q blocked: %s (severity: %s)", e.GuardrailName, e.Reason, e.Severity)
}

// NewGuardrailError creates a new GuardrailError from a Result.
func NewGuardrailError(result *Result) *GuardrailError {
	return &GuardrailError{
		GuardrailName: result.GuardrailName,
		Reason:        result.Reason,
		Severity:      result.Severity,
		Details:       result.Details,
	}
}

// ChainError represents multiple errors from a guardrail chain.
type ChainError struct {
	Errors []*GuardrailError
}

func (e *ChainError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("%d guardrails blocked execution", len(e.Errors))
}

// Add adds an error to the chain.
func (e *ChainError) Add(err *GuardrailError) {
	e.Errors = append(e.Errors, err)
}

// HasErrors returns true if any errors were collected.
func (e *ChainError) HasErrors() bool {
	return len(e.Errors) > 0
}
