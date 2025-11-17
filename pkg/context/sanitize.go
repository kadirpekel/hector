package context

import "strings"

// sanitizeInput removes or escapes potential prompt injection patterns from user input
// This prevents malicious queries from manipulating the LLM's behavior
func sanitizeInput(input string) string {
	// Remove common prompt injection patterns
	sanitized := input

	// Remove system role indicators that could confuse the LLM
	sanitized = strings.ReplaceAll(sanitized, "SYSTEM:", "")
	sanitized = strings.ReplaceAll(sanitized, "System:", "")
	sanitized = strings.ReplaceAll(sanitized, "system:", "")
	sanitized = strings.ReplaceAll(sanitized, "ASSISTANT:", "")
	sanitized = strings.ReplaceAll(sanitized, "Assistant:", "")
	sanitized = strings.ReplaceAll(sanitized, "assistant:", "")
	sanitized = strings.ReplaceAll(sanitized, "USER:", "")
	sanitized = strings.ReplaceAll(sanitized, "User:", "")
	sanitized = strings.ReplaceAll(sanitized, "user:", "")

	// Remove instruction override attempts
	sanitized = strings.ReplaceAll(sanitized, "Ignore previous instructions", "")
	sanitized = strings.ReplaceAll(sanitized, "ignore previous instructions", "")
	sanitized = strings.ReplaceAll(sanitized, "Ignore all previous", "")
	sanitized = strings.ReplaceAll(sanitized, "ignore all previous", "")
	sanitized = strings.ReplaceAll(sanitized, "Disregard previous", "")
	sanitized = strings.ReplaceAll(sanitized, "disregard previous", "")

	// Remove common delimiter attacks (trying to break out of the prompt structure)
	sanitized = strings.ReplaceAll(sanitized, "---", "")
	sanitized = strings.ReplaceAll(sanitized, "===", "")
	sanitized = strings.ReplaceAll(sanitized, "***", "")

	// Escape backticks that could be used for code injection or markdown manipulation
	sanitized = strings.ReplaceAll(sanitized, "```", "")

	// Remove excessive whitespace that could be used for obfuscation
	sanitized = strings.TrimSpace(sanitized)

	return sanitized
}
