package output

import (
	"github.com/verikod/hector/pkg/guardrails"
)

// DefaultBuilders returns the standard output guardrail builders.
// Use with Config.BuildOutputChain(output.DefaultBuilders()).
func DefaultBuilders() guardrails.OutputChainBuilders {
	return guardrails.OutputChainBuilders{
		PIIRedactor:   BuildPIIRedactor,
		ContentFilter: BuildContentFilter,
	}
}

// BuildPIIRedactor creates a PIIRedactor from config.
func BuildPIIRedactor(cfg *guardrails.PIIConfig) guardrails.OutputGuardrail {
	r := NewPIIRedactor()
	r.DetectEmail(cfg.DetectEmail)
	r.DetectPhone(cfg.DetectPhone)
	r.DetectSSN(cfg.DetectSSN)
	r.DetectCreditCard(cfg.DetectCreditCard)
	if cfg.RedactMode != "" {
		r.RedactMode(cfg.RedactMode)
	}
	if cfg.Action != "" {
		r.WithAction(cfg.Action)
	}
	if cfg.Severity != "" {
		r.WithSeverity(cfg.Severity)
	}
	return r
}

// BuildContentFilter creates a ContentFilter from config.
func BuildContentFilter(cfg *guardrails.ContentConfig) guardrails.OutputGuardrail {
	f := NewContentFilter()
	if len(cfg.BlockedKeywords) > 0 {
		f.BlockKeywords(cfg.BlockedKeywords...)
	}
	if len(cfg.BlockedPatterns) > 0 {
		f.BlockPatterns(cfg.BlockedPatterns...)
	}
	if cfg.Action != "" {
		f.WithAction(cfg.Action)
	}
	if cfg.Severity != "" {
		f.WithSeverity(cfg.Severity)
	}
	return f
}
