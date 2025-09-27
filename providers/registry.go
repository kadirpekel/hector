package providers

import (
	"context"
	"sync"
)

// ============================================================================
// PROVIDER REGISTRY
// ============================================================================

// ProviderRegistry manages provider configurations and factories
type ProviderRegistry struct {
	mu        sync.RWMutex
	configs   map[ProviderType]map[string]ProviderConfig
	factories map[ProviderType]ProviderFactory
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		configs:   make(map[ProviderType]map[string]ProviderConfig),
		factories: make(map[ProviderType]ProviderFactory),
	}
}

// GlobalProviderRegistry is the default registry instance
var GlobalProviderRegistry = NewProviderRegistry()

// RegisterProvider registers a provider configuration
func (r *ProviderRegistry) RegisterProvider(config ProviderConfig) error {
	if config == nil {
		return NewProviderError("", "", "RegisterProvider", "config cannot be nil", nil)
	}

	providerType := config.GetProviderType()
	providerName := config.GetProviderName()

	if providerName == "" {
		return NewProviderError(providerType, "", "RegisterProvider", "provider name cannot be empty", nil)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return NewProviderError(providerType, providerName, "RegisterProvider", "invalid configuration", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize the provider type map if it doesn't exist
	if r.configs[providerType] == nil {
		r.configs[providerType] = make(map[string]ProviderConfig)
	}

	// Check if provider already exists
	if _, exists := r.configs[providerType][providerName]; exists {
		return NewProviderError(providerType, providerName, "RegisterProvider", "provider already registered", nil)
	}

	// Register the provider
	r.configs[providerType][providerName] = config

	return nil
}

// RegisterFactory registers a provider factory
func (r *ProviderRegistry) RegisterFactory(factory ProviderFactory) error {
	if factory == nil {
		return NewProviderError("", "", "RegisterFactory", "factory cannot be nil", nil)
	}

	providerType := factory.GetSupportedType()
	if providerType == "" {
		return NewProviderError("", "", "RegisterFactory", "factory must specify supported type", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if factory already exists
	if _, exists := r.factories[providerType]; exists {
		return NewProviderError(providerType, "", "RegisterFactory", "factory already registered for type", nil)
	}

	// Register the factory
	r.factories[providerType] = factory

	return nil
}

// GetProviderConfig returns the configuration for a specific provider
func (r *ProviderRegistry) GetProviderConfig(providerType ProviderType, providerName string) (ProviderConfig, error) {
	if providerName == "" {
		return nil, NewProviderError(providerType, "", "GetProviderConfig", "provider name cannot be empty", nil)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	providers, exists := r.configs[providerType]
	if !exists {
		return nil, NewProviderError(providerType, providerName, "GetProviderConfig", "no providers registered for type", nil)
	}

	config, exists := providers[providerName]
	if !exists {
		return nil, NewProviderError(providerType, providerName, "GetProviderConfig", "provider not found", nil)
	}

	return config, nil
}

// CreateProvider creates a provider instance using the registered factory
func (r *ProviderRegistry) CreateProvider(providerType ProviderType, providerName string) (interface{}, error) {
	if providerName == "" {
		return nil, NewProviderError(providerType, "", "CreateProvider", "provider name cannot be empty", nil)
	}

	// Get the configuration
	config, err := r.GetProviderConfig(providerType, providerName)
	if err != nil {
		return nil, err
	}

	r.mu.RLock()
	factory, exists := r.factories[providerType]
	r.mu.RUnlock()

	if !exists {
		return nil, NewProviderError(providerType, providerName, "CreateProvider", "no factory registered for type", nil)
	}

	// Create the provider using the factory
	provider, err := factory.CreateProvider(config)
	if err != nil {
		return nil, NewProviderError(providerType, providerName, "CreateProvider", "failed to create provider", err)
	}

	return provider, nil
}

// ListProviders returns all registered providers for a given type
func (r *ProviderRegistry) ListProviders(providerType ProviderType) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers, exists := r.configs[providerType]
	if !exists {
		return []string{}
	}

	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}

	return names
}

// ListAllProviders returns all registered providers organized by type
func (r *ProviderRegistry) ListAllProviders() map[ProviderType][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[ProviderType][]string)
	for providerType := range r.configs {
		result[providerType] = r.ListProviders(providerType)
	}

	return result
}

// GetHealthStatus returns the health status of all registered providers
func (r *ProviderRegistry) GetHealthStatus(ctx context.Context) map[ProviderType]map[string]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[ProviderType]map[string]bool)

	for providerType, providers := range r.configs {
		result[providerType] = make(map[string]bool)
		for providerName := range providers {
			// Try to create and check health of each provider
			if provider, err := r.CreateProvider(providerType, providerName); err == nil {
				if healthCheckable, ok := provider.(interface{ IsHealthy(context.Context) bool }); ok {
					result[providerType][providerName] = healthCheckable.IsHealthy(ctx)
				} else {
					result[providerType][providerName] = true // Assume healthy if no health check
				}
			} else {
				result[providerType][providerName] = false
			}
		}
	}

	return result
}

// ============================================================================
// GLOBAL REGISTRY FUNCTIONS
// ============================================================================

// RegisterProvider registers a provider configuration in the global registry
func RegisterProvider(config ProviderConfig) error {
	return GlobalProviderRegistry.RegisterProvider(config)
}

// RegisterFactory registers a provider factory in the global registry
func RegisterFactory(factory ProviderFactory) error {
	return GlobalProviderRegistry.RegisterFactory(factory)
}

// GetProviderConfig gets a provider configuration from the global registry
func GetProviderConfig(providerType ProviderType, providerName string) (ProviderConfig, error) {
	return GlobalProviderRegistry.GetProviderConfig(providerType, providerName)
}

// CreateProvider creates a provider instance using the global registry
func CreateProvider(providerType ProviderType, providerName string) (interface{}, error) {
	return GlobalProviderRegistry.CreateProvider(providerType, providerName)
}

// ListProviders returns all registered providers for a given type from the global registry
func ListProviders(providerType ProviderType) []string {
	return GlobalProviderRegistry.ListProviders(providerType)
}

// ListAllProviders returns all registered providers organized by type from the global registry
func ListAllProviders() map[ProviderType][]string {
	return GlobalProviderRegistry.ListAllProviders()
}

// GetHealthStatus returns the health status of all providers from the global registry
func GetHealthStatus(ctx context.Context) map[ProviderType]map[string]bool {
	return GlobalProviderRegistry.GetHealthStatus(ctx)
}
