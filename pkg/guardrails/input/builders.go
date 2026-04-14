package input

import (
	"github.com/verikod/hector/pkg/guardrails"
)

// DefaultBuilders returns the standard input guardrail builders.
// Use with Config.BuildInputChain(input.DefaultBuilders()).
func DefaultBuilders() guardrails.InputChainBuilders {
	return guardrails.InputChainBuilders{
		LengthValidator:   BuildLengthValidator,
		InjectionDetector: BuildInjectionDetector,
		Sanitizer:         BuildSanitizer,
	}
}

// BuildLengthValidator creates a LengthValidator from config.
func BuildLengthValidator(cfg *guardrails.LengthConfig) guardrails.InputGuardrail {
	v := NewLengthValidator(cfg.MinLength, cfg.MaxLength)
	if cfg.Action != "" {
		v.WithAction(cfg.Action)
	}
	if cfg.Severity != "" {
		v.WithSeverity(cfg.Severity)
	}
	return v
}

// BuildInjectionDetector creates an InjectionDetector from config.
func BuildInjectionDetector(cfg *guardrails.InjectionConfig) guardrails.InputGuardrail {
	d := NewInjectionDetector()
	if len(cfg.Patterns) > 0 {
		d.AddPatterns(cfg.Patterns...)
	}
	d.CaseSensitive(cfg.CaseSensitive)
	if cfg.Action != "" {
		d.WithAction(cfg.Action)
	}
	if cfg.Severity != "" {
		d.WithSeverity(cfg.Severity)
	}
	return d
}

// BuildSanitizer creates a Sanitizer from config.
func BuildSanitizer(cfg *guardrails.SanitizerConfig) guardrails.InputGuardrail {
	s := NewSanitizer()
	s.TrimWhitespace(cfg.TrimWhitespace)
	s.NormalizeUnicode(cfg.NormalizeUnicode)
	s.StripHTML(cfg.StripHTML)
	if cfg.MaxLength > 0 {
		s.MaxLength(cfg.MaxLength)
	}
	return s
}
