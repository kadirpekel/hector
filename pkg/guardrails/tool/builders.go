package tool

import (
	"github.com/verikod/hector/pkg/guardrails"
)

// DefaultBuilders returns the standard tool guardrail builders.
// Use with Config.BuildToolChain(tool.DefaultBuilders()).
func DefaultBuilders() guardrails.ToolChainBuilders {
	return guardrails.ToolChainBuilders{
		Authorizer: BuildAuthorizer,
	}
}

// BuildAuthorizer creates an Authorizer from config.
func BuildAuthorizer(cfg *guardrails.AuthorizationConfig) guardrails.ToolGuardrail {
	a := NewAuthorizer()
	if len(cfg.AllowedTools) > 0 {
		a.AllowOnly(cfg.AllowedTools...)
	}
	if len(cfg.BlockedTools) > 0 {
		a.Block(cfg.BlockedTools...)
	}
	if cfg.Action != "" {
		a.WithAction(cfg.Action)
	}
	if cfg.Severity != "" {
		a.WithSeverity(cfg.Severity)
	}
	return a
}
