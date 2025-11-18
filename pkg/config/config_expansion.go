package config

import (
	"fmt"
	"strings"
)

// applyDefaults applies default values from the defaults section to agents
func (c *Config) applyDefaults() {
	if c.Defaults == nil {
		return
	}

	for _, agent := range c.Agents {
		if agent == nil {
			continue
		}

		// Only apply defaults to native agents
		if agent.Type != "" && agent.Type != "native" {
			continue
		}

		// Apply defaults only if agent doesn't have the field set
		if agent.LLM == "" && agent.LLMInline == nil && c.Defaults.LLM != "" {
			agent.LLM = c.Defaults.LLM
		}
		if agent.VectorStore == "" && agent.VectorStoreInline == nil && c.Defaults.VectorStore != "" {
			agent.VectorStore = c.Defaults.VectorStore
		}
		if agent.Embedder == "" && agent.EmbedderInline == nil && c.Defaults.Embedder != "" {
			agent.Embedder = c.Defaults.Embedder
		}
		if agent.SessionStore == "" && c.Defaults.SessionStore != "" {
			agent.SessionStore = c.Defaults.SessionStore
		}
	}
}

// expandInlineConfigs expands inline configs to top-level providers and updates agent references
func (c *Config) expandInlineConfigs(agent *AgentConfig) {
	if agent == nil {
		return
	}

	// Expand inline LLM config
	if agent.LLMInline != nil {
		if agent.LLM != "" {
			// Both inline and reference specified - error
			// This will be caught in validation
			return
		}
		// Generate a unique name for the inline config
		inlineName := generateInlineProviderName("llm", agent.Name)
		c.LLMs[inlineName] = agent.LLMInline
		agent.LLM = inlineName
		agent.LLMInline = nil // Clear inline config after expansion
		fmt.Printf("INFO: Expanded inline LLM config for agent '%s' to top-level provider '%s'\n", agent.Name, inlineName)
	}

	// Expand inline vector store config
	if agent.VectorStoreInline != nil {
		if agent.VectorStore != "" {
			// Both inline and reference specified - error
			return
		}
		inlineName := generateInlineProviderName("vector-store", agent.Name)
		c.VectorStores[inlineName] = agent.VectorStoreInline
		agent.VectorStore = inlineName
		agent.VectorStoreInline = nil
		fmt.Printf("INFO: Expanded inline vector store config for agent '%s' to top-level provider '%s'\n", agent.Name, inlineName)
	}

	// Expand inline embedder config
	if agent.EmbedderInline != nil {
		if agent.Embedder != "" {
			// Both inline and reference specified - error
			return
		}
		inlineName := generateInlineProviderName("embedder", agent.Name)
		c.Embedders[inlineName] = agent.EmbedderInline
		agent.Embedder = inlineName
		agent.EmbedderInline = nil
		fmt.Printf("INFO: Expanded inline embedder config for agent '%s' to top-level provider '%s'\n", agent.Name, inlineName)
	}
}

// generateInlineProviderName generates a unique name for an inline provider config
func generateInlineProviderName(providerType, agentName string) string {
	// Use agent name as base, sanitize it
	sanitized := strings.ToLower(strings.ReplaceAll(agentName, " ", "-"))
	return fmt.Sprintf("%s-%s-inline", sanitized, providerType)
}
