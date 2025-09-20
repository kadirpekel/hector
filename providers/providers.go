package providers

import (
	"fmt"
	"reflect"

	"github.com/kadirpekel/hector/databases"
	"github.com/kadirpekel/hector/embedders"
	"github.com/kadirpekel/hector/interfaces"
	"github.com/kadirpekel/hector/llms"
	"gopkg.in/yaml.v3"
)

// ============================================================================
// PROVIDER REGISTRY
// ============================================================================

// GlobalProviderRegistry is the default registry instance
var GlobalProviderRegistry = NewProviderRegistry()

// ProviderRegistry manages provider configurations and instantiation
type ProviderRegistry struct {
	providers map[interfaces.ProviderType]map[string]interfaces.ProviderConfig
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[interfaces.ProviderType]map[string]interfaces.ProviderConfig),
	}
}

// RegisterProvider registers a provider configuration
func RegisterProvider(config interfaces.ProviderConfig) error {
	providerType := config.GetProviderType()
	providerName := config.GetProviderName()

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid provider config for %s/%s: %w", providerType, providerName, err)
	}

	// Initialize the provider type map if it doesn't exist
	if GlobalProviderRegistry.providers[providerType] == nil {
		GlobalProviderRegistry.providers[providerType] = make(map[string]interfaces.ProviderConfig)
	}

	// Check if provider already exists
	if _, exists := GlobalProviderRegistry.providers[providerType][providerName]; exists {
		return fmt.Errorf("provider %s/%s already registered", providerType, providerName)
	}

	// Register the provider
	GlobalProviderRegistry.providers[providerType][providerName] = config

	return nil
}

// ============================================================================
// PROVIDER REGISTRATION
// ============================================================================

// RegisterDefaultProviders registers all default providers with the global registry
func RegisterDefaultProviders() error {
	// Register LLM providers
	ollamaConfig := &llms.OllamaConfig{
		Provider:    "ollama",
		Model:       "llama3.2", // Set default model for registration
		Host:        "http://localhost:11434",
		Temperature: 0.7,
		MaxTokens:   1000,
		Timeout:     60,
	}
	if err := RegisterProvider(ollamaConfig); err != nil {
		return fmt.Errorf("failed to register Ollama LLM provider: %w", err)
	}

	openaiConfig := &llms.OpenAIConfig{
		Provider:    "openai",
		APIKey:      "dummy", // Set dummy key for registration
		Model:       "gpt-3.5-turbo",
		Host:        "https://api.openai.com/v1",
		Temperature: 0.7,
		MaxTokens:   1000,
		Timeout:     60,
	}
	if err := RegisterProvider(openaiConfig); err != nil {
		return fmt.Errorf("failed to register OpenAI LLM provider: %w", err)
	}

	tgiConfig := &llms.TGIConfig{
		Provider:    "tgi",
		Model:       "microsoft/DialoGPT-medium", // Set default model for registration
		Host:        "http://localhost:8080",
		Temperature: 0.7,
		MaxTokens:   1000,
		Timeout:     60,
	}
	if err := RegisterProvider(tgiConfig); err != nil {
		return fmt.Errorf("failed to register TGI LLM provider: %w", err)
	}

	// Register Database providers
	qdrantConfig := &databases.QdrantConfig{
		Provider: "qdrant",
		Host:     "localhost",
		Port:     6334,
		Timeout:  30,
		UseTLS:   false,
		Insecure: false,
	}
	if err := RegisterProvider(qdrantConfig); err != nil {
		return fmt.Errorf("failed to register Qdrant database provider: %w", err)
	}

	// Register Embedder providers
	ollamaEmbedderConfig := &embedders.OllamaEmbedderConfig{
		Provider:   "ollama",
		Model:      "nomic-embed-text",
		Host:       "http://localhost:11434",
		Dimension:  768,
		Timeout:    30,
		MaxRetries: 3,
	}
	if err := RegisterProvider(ollamaEmbedderConfig); err != nil {
		return fmt.Errorf("failed to register Ollama embedder provider: %w", err)
	}

	tgiEmbedderConfig := &embedders.TGIEmbedderProviderConfig{
		Provider:     "tgi",
		Model:        "sentence-transformers/all-MiniLM-L6-v2",
		Host:         "http://localhost:8080",
		Dimension:    384,
		Timeout:      30,
		WaitForModel: true,
		Normalize:    true,
		Truncate:     true,
		MaxLength:    512,
	}
	if err := RegisterProvider(tgiEmbedderConfig); err != nil {
		return fmt.Errorf("failed to register TGI embedder provider: %w", err)
	}

	return nil
}

// ============================================================================
// DYNAMIC PROVIDER CREATION
// ============================================================================

// CreateLLMProvider creates an LLM provider dynamically from config
func CreateLLMProvider(config map[string]interface{}) (llms.LLMProvider, error) {
	providerName, ok := config["provider"].(string)
	if !ok {
		return nil, fmt.Errorf("provider field is required and must be a string")
	}

	// Get the provider config from registry
	registry := GlobalProviderRegistry
	providerConfigs, exists := registry.providers[interfaces.ProviderTypeLLM]
	if !exists {
		return nil, fmt.Errorf("no LLM providers registered")
	}

	providerConfig, exists := providerConfigs[providerName]
	if !exists {
		return nil, fmt.Errorf("LLM provider '%s' not found in registry", providerName)
	}

	// Create a new instance of the provider config type using reflection
	configType := reflect.TypeOf(providerConfig)
	if configType.Kind() == reflect.Ptr {
		configType = configType.Elem()
	}

	newConfig := reflect.New(configType).Interface()

	// Populate the config struct using YAML (more appropriate for our use case)
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := yaml.Unmarshal(configYAML, newConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Cast to ProviderConfig interface
	providerConfigInterface, ok := newConfig.(interfaces.ProviderConfig)
	if !ok {
		return nil, fmt.Errorf("config does not implement ProviderConfig interface")
	}

	// Set defaults before validation
	if setDefaultsMethod := reflect.ValueOf(providerConfigInterface).MethodByName("SetDefaults"); setDefaultsMethod.IsValid() {
		setDefaultsMethod.Call(nil)
	}

	// Validate and create provider
	if err := providerConfigInterface.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config for provider %s: %w", providerName, err)
	}

	provider, err := providerConfigInterface.CreateProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	llmProvider, ok := provider.(llms.LLMProvider)
	if !ok {
		return nil, fmt.Errorf("provider %s does not implement LLMProvider interface", providerName)
	}

	return llmProvider, nil
}

// CreateDatabaseProvider creates a database provider dynamically from config
func CreateDatabaseProvider(config map[string]interface{}) (databases.VectorDB, error) {
	providerName, ok := config["provider"].(string)
	if !ok {
		return nil, fmt.Errorf("provider field is required and must be a string")
	}

	// Get the provider config from registry
	registry := GlobalProviderRegistry
	providerConfigs, exists := registry.providers[interfaces.ProviderTypeDatabase]
	if !exists {
		return nil, fmt.Errorf("no database providers registered")
	}

	providerConfig, exists := providerConfigs[providerName]
	if !exists {
		return nil, fmt.Errorf("database provider '%s' not found in registry", providerName)
	}

	// Create a new instance of the provider config type using reflection
	configType := reflect.TypeOf(providerConfig)
	if configType.Kind() == reflect.Ptr {
		configType = configType.Elem()
	}

	newConfig := reflect.New(configType).Interface()

	// Populate the config struct using YAML (more appropriate for our use case)
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := yaml.Unmarshal(configYAML, newConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Cast to ProviderConfig interface
	providerConfigInterface, ok := newConfig.(interfaces.ProviderConfig)
	if !ok {
		return nil, fmt.Errorf("config does not implement ProviderConfig interface")
	}

	// Set defaults before validation
	if setDefaultsMethod := reflect.ValueOf(providerConfigInterface).MethodByName("SetDefaults"); setDefaultsMethod.IsValid() {
		setDefaultsMethod.Call(nil)
	}

	// Validate and create provider
	if err := providerConfigInterface.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config for provider %s: %w", providerName, err)
	}

	provider, err := providerConfigInterface.CreateProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	dbProvider, ok := provider.(databases.VectorDB)
	if !ok {
		return nil, fmt.Errorf("provider %s does not implement VectorDB interface", providerName)
	}

	return dbProvider, nil
}

// CreateEmbedderProvider creates an embedder provider dynamically from config
func CreateEmbedderProvider(config map[string]interface{}) (embedders.EmbeddingProvider, error) {
	providerName, ok := config["provider"].(string)
	if !ok {
		return nil, fmt.Errorf("provider field is required and must be a string")
	}

	// Get the provider config from registry
	registry := GlobalProviderRegistry
	providerConfigs, exists := registry.providers[interfaces.ProviderTypeEmbedder]
	if !exists {
		return nil, fmt.Errorf("no embedder providers registered")
	}

	providerConfig, exists := providerConfigs[providerName]
	if !exists {
		return nil, fmt.Errorf("embedder provider '%s' not found in registry", providerName)
	}

	// Create a new instance of the provider config type using reflection
	configType := reflect.TypeOf(providerConfig)
	if configType.Kind() == reflect.Ptr {
		configType = configType.Elem()
	}

	newConfig := reflect.New(configType).Interface()

	// Populate the config struct using YAML (more appropriate for our use case)
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := yaml.Unmarshal(configYAML, newConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Cast to ProviderConfig interface
	providerConfigInterface, ok := newConfig.(interfaces.ProviderConfig)
	if !ok {
		return nil, fmt.Errorf("config does not implement ProviderConfig interface")
	}

	// Set defaults before validation
	if setDefaultsMethod := reflect.ValueOf(providerConfigInterface).MethodByName("SetDefaults"); setDefaultsMethod.IsValid() {
		setDefaultsMethod.Call(nil)
	}

	// Validate and create provider
	if err := providerConfigInterface.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config for provider %s: %w", providerName, err)
	}

	provider, err := providerConfigInterface.CreateProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	embedderProvider, ok := provider.(embedders.EmbeddingProvider)
	if !ok {
		return nil, fmt.Errorf("provider %s does not implement EmbeddingProvider interface", providerName)
	}

	return embedderProvider, nil
}
