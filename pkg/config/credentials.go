package config

// CredentialsConfig holds authentication credentials for HTTP requests.
// Used for API tools, webhook endpoints, and external service integrations.
type CredentialsConfig struct {
	// Type of credential: "bearer", "api_key", or "basic"
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// Token for bearer authentication
	Token string `yaml:"token,omitempty" json:"token,omitempty"`

	// APIKey for API key authentication
	APIKey string `yaml:"api_key,omitempty" json:"api_key,omitempty"`

	// APIKeyHeader is the header name for API key (default: X-API-Key)
	APIKeyHeader string `yaml:"api_key_header,omitempty" json:"api_key_header,omitempty"`

	// Username for basic authentication
	Username string `yaml:"username,omitempty" json:"username,omitempty"`

	// Password for basic authentication
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
}

// SetDefaults applies default values.
func (c *CredentialsConfig) SetDefaults() {
	if c.Type == "" {
		c.Type = "bearer"
	}
	if c.APIKeyHeader == "" {
		c.APIKeyHeader = "X-API-Key"
	}
}

// Validate checks the credentials configuration.
func (c *CredentialsConfig) Validate() error {
	return nil
}
