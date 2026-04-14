//go:build !unrestricted

// Package commandtool provides a secure, streaming command execution tool.
//
// Sandbox Enforcement:
//
// By default, Hector is compiled with SandboxEnforced = true, which means:
//   - DefaultDeniedCommands are ALWAYS applied (cannot be emptied via config)
//   - DefaultDeniedPatterns are ALWAYS applied (cannot be removed via config)
//   - Config can ADD to deny lists, but not remove default protections
//
// To compile an unrestricted version (for advanced users who understand the risks):
//
//	go build -tags=unrestricted ./cmd/hector
//
// This should only be used in controlled environments where the operator
// explicitly wants to allow all commands.
package commandtool

// SandboxEnforced indicates whether sandbox protections are permanently enabled.
// When true (the default), DefaultDeniedCommands and DefaultDeniedPatterns
// cannot be bypassed via configuration.
const SandboxEnforced = true
