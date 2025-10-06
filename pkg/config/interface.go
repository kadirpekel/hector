// Package config provides configuration types and utilities for the AI agent framework.
// This file defines the core Config interface that all configuration types must implement.
package config

// ConfigInterface defines the interface that all configuration types must implement
// This provides a consistent way to validate and set defaults for any configuration
type ConfigInterface interface {
	// Validate checks if the configuration is valid and returns an error if not
	Validate() error

	// SetDefaults sets default values for any unset fields
	SetDefaults()
}
