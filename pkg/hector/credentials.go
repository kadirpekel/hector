package hector

import (
	"github.com/kadirpekel/hector/pkg/config"
)

// AgentCredentialsBuilder provides a fluent API for building agent credentials
type AgentCredentialsBuilder struct {
	creds *config.AgentCredentials
}

// NewAgentCredentials creates a new agent credentials builder
func NewAgentCredentials() *AgentCredentialsBuilder {
	return &AgentCredentialsBuilder{
		creds: &config.AgentCredentials{},
	}
}

// Type sets the credential type ("bearer", "api_key", "basic")
func (b *AgentCredentialsBuilder) Type(typ string) *AgentCredentialsBuilder {
	b.creds.Type = typ
	return b
}

// Token sets the bearer token
func (b *AgentCredentialsBuilder) Token(token string) *AgentCredentialsBuilder {
	b.creds.Token = token
	return b
}

// APIKey sets the API key
func (b *AgentCredentialsBuilder) APIKey(key string) *AgentCredentialsBuilder {
	b.creds.APIKey = key
	return b
}

// APIKeyHeader sets the API key header name
func (b *AgentCredentialsBuilder) APIKeyHeader(header string) *AgentCredentialsBuilder {
	b.creds.APIKeyHeader = header
	return b
}

// Username sets the username (for basic auth)
func (b *AgentCredentialsBuilder) Username(user string) *AgentCredentialsBuilder {
	b.creds.Username = user
	return b
}

// Password sets the password (for basic auth)
func (b *AgentCredentialsBuilder) Password(pass string) *AgentCredentialsBuilder {
	b.creds.Password = pass
	return b
}

// Build returns the credentials config
func (b *AgentCredentialsBuilder) Build() *config.AgentCredentials {
	b.creds.SetDefaults()
	return b.creds
}

// NewAgentCredentialsWithConfig creates a builder with existing config
func NewAgentCredentialsWithConfig(cfg *config.AgentCredentials) *AgentCredentialsBuilder {
	if cfg == nil {
		cfg = &config.AgentCredentials{}
	}
	return &AgentCredentialsBuilder{
		creds: cfg,
	}
}

