package hector

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
)

// DatabaseBuilder provides a fluent API for building database providers
type DatabaseBuilder struct {
	dbType string
	host   string
	port   int
	apiKey string
	useTLS *bool
}

// NewDatabase creates a new database builder
func NewDatabase(dbType string) *DatabaseBuilder {
	builder := &DatabaseBuilder{
		dbType: dbType,
		port:   6333,
		useTLS: boolPtr(false),
	}

	// Set defaults based on type
	switch dbType {
	case "qdrant":
		builder.host = "localhost"
		builder.port = 6333
	default:
		builder.host = "localhost"
	}

	return builder
}

// Host sets the database host
func (b *DatabaseBuilder) Host(host string) *DatabaseBuilder {
	b.host = host
	return b
}

// Port sets the database port
func (b *DatabaseBuilder) Port(port int) *DatabaseBuilder {
	if port <= 0 {
		panic("port must be positive")
	}
	b.port = port
	return b
}

// APIKey sets the API key
func (b *DatabaseBuilder) APIKey(key string) *DatabaseBuilder {
	b.apiKey = key
	return b
}

// UseTLS enables/disables TLS
func (b *DatabaseBuilder) UseTLS(use bool) *DatabaseBuilder {
	b.useTLS = &use
	return b
}

// Build creates the database provider
func (b *DatabaseBuilder) Build() (databases.DatabaseProvider, error) {
	if b.host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if b.port <= 0 {
		return nil, fmt.Errorf("port must be positive")
	}

	switch b.dbType {
	case "qdrant":
		cfg := &config.DatabaseProviderConfig{
			Type:   "qdrant",
			Host:   b.host,
			Port:   b.port,
			APIKey: b.apiKey,
			UseTLS: b.useTLS,
		}
		return databases.NewQdrantDatabaseProviderFromConfig(cfg)
	default:
		return nil, fmt.Errorf("unknown database type: %s (supported: 'qdrant')", b.dbType)
	}
}
