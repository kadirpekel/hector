package utils

// ============================================================================
// TOKEN UTILITIES
// ============================================================================

// EstimateTokens provides a rough token estimation
func EstimateTokens(text string) int {
	// Rough estimation: 4 characters per token
	return len(text) / 4
}
