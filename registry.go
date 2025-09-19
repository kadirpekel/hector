package hector

import (
	"fmt"
	"reflect"

	"github.com/kadirpekel/hector/interfaces"
)

// ============================================================================
// PROVIDER REGISTRY SYSTEM
// ============================================================================

// ProviderType represents the type of provider
type ProviderType = interfaces.ProviderType

const (
	ProviderTypeLLM      = interfaces.ProviderTypeLLM
	ProviderTypeDatabase = interfaces.ProviderTypeDatabase
	ProviderTypeEmbedder = interfaces.ProviderTypeEmbedder
)

// ProviderConfig represents a configuration that can create a provider
type ProviderConfig = interfaces.ProviderConfig

// ProviderRegistry manages provider configurations and instantiation
type ProviderRegistry struct {
	providers map[ProviderType]map[string]ProviderConfig
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[ProviderType]map[string]ProviderConfig),
	}
}

// RegisterProvider registers a provider configuration
func (r *ProviderRegistry) RegisterProvider(config ProviderConfig) error {
	providerType := config.GetProviderType()
	providerName := config.GetProviderName()

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid provider config for %s/%s: %w", providerType, providerName, err)
	}

	// Initialize the provider type map if it doesn't exist
	if r.providers[providerType] == nil {
		r.providers[providerType] = make(map[string]ProviderConfig)
	}

	// Check if provider already exists
	if _, exists := r.providers[providerType][providerName]; exists {
		return fmt.Errorf("provider %s/%s already registered", providerType, providerName)
	}

	// Register the provider
	r.providers[providerType][providerName] = config

	return nil
}

// CreateProvider creates a provider instance by type and name
func (r *ProviderRegistry) CreateProvider(providerType ProviderType, providerName string) (interface{}, error) {
	providers, exists := r.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("no providers registered for type %s", providerType)
	}

	config, exists := providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s/%s not found", providerType, providerName)
	}

	return config.CreateProvider()
}

// GetProviderConfig returns the configuration for a specific provider
func (r *ProviderRegistry) GetProviderConfig(providerType ProviderType, providerName string) (ProviderConfig, error) {
	providers, exists := r.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("no providers registered for type %s", providerType)
	}

	config, exists := providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s/%s not found", providerType, providerName)
	}

	return config, nil
}

// ListProviders returns all registered providers for a given type
func (r *ProviderRegistry) ListProviders(providerType ProviderType) []string {
	providers, exists := r.providers[providerType]
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
	result := make(map[ProviderType][]string)

	for providerType := range r.providers {
		result[providerType] = r.ListProviders(providerType)
	}

	return result
}

// ============================================================================
// GLOBAL REGISTRY INSTANCE
// ============================================================================

// GlobalProviderRegistry is the default registry instance
var GlobalProviderRegistry = NewProviderRegistry()

// RegisterProvider registers a provider configuration in the global registry
func RegisterProvider(config ProviderConfig) error {
	return GlobalProviderRegistry.RegisterProvider(config)
}

// CreateProvider creates a provider instance using the global registry
func CreateProvider(providerType ProviderType, providerName string) (interface{}, error) {
	return GlobalProviderRegistry.CreateProvider(providerType, providerName)
}

// GetProviderConfig gets a provider configuration from the global registry
func GetProviderConfig(providerType ProviderType, providerName string) (ProviderConfig, error) {
	return GlobalProviderRegistry.GetProviderConfig(providerType, providerName)
}

// ============================================================================
// CONFIGURATION HELPERS
// ============================================================================

// ConfigFromMap creates a provider config from a map of values
// This is useful for dynamic configuration loading
func ConfigFromMap(providerType ProviderType, providerName string, configMap map[string]interface{}) (ProviderConfig, error) {
	// This is a generic helper that can be extended for specific provider types
	// For now, we'll return an error as this needs to be implemented per provider type
	return nil, fmt.Errorf("ConfigFromMap not implemented for provider type %s", providerType)
}

// ValidateConfigMap validates a configuration map against expected fields
func ValidateConfigMap(configMap map[string]interface{}, requiredFields []string) error {
	for _, field := range requiredFields {
		if _, exists := configMap[field]; !exists {
			return fmt.Errorf("required field '%s' is missing", field)
		}
	}
	return nil
}

// ============================================================================
// REFLECTION HELPERS
// ============================================================================

// CreateProviderFromStruct creates a provider from a struct using reflection
// This is useful for dynamic provider creation from YAML configs
func CreateProviderFromStruct(configStruct interface{}, providerType ProviderType, providerName string) (ProviderConfig, error) {
	// Get the type of the config struct
	configType := reflect.TypeOf(configStruct)
	if configType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("config must be a struct, got %s", configType.Kind())
	}

	// Create a new instance of the config struct
	configValue := reflect.New(configType).Elem()

	// Copy values from the input struct
	inputValue := reflect.ValueOf(configStruct)
	for i := 0; i < configType.NumField(); i++ {
		fieldValue := inputValue.Field(i)
		configValue.Field(i).Set(fieldValue)
	}

	// Create the config instance
	configInstance := configValue.Interface()

	// Cast to ProviderConfig interface
	providerConfig, ok := configInstance.(ProviderConfig)
	if !ok {
		return nil, fmt.Errorf("struct does not implement ProviderConfig interface")
	}

	return providerConfig, nil
}
