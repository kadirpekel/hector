//go:build unrestricted

// Package commandtool provides a secure, streaming command execution tool.
//
// # UNRESTRICTED BUILD
//
// This file is only included when building with -tags=unrestricted.
// In this mode, SandboxEnforced = false, which means:
//   - Config can completely override DefaultDeniedCommands
//   - Config can completely override DefaultDeniedPatterns
//   - The operator takes full responsibility for security
//
// WARNING: Only use this in controlled environments where you explicitly
// need to allow commands that are blocked by default (rm, sudo, etc.).
package commandtool

// SandboxEnforced is false in unrestricted builds.
// This allows config to fully override security defaults.
const SandboxEnforced = false
