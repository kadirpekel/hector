package hector

import (
	"fmt"
	"strings"
)

// ============================================================================
// MODEL CONFIGURATION
// ============================================================================

// ModelConfig holds configuration for a vector model
type ModelConfig struct {
	Name            string      `yaml:"name"`             // Model name (e.g., "document", "article")
	Collection      string      `yaml:"collection"`       // Vector collection name
	EmbeddingFields []string    `yaml:"embedding_fields"` // Fields to embed
	MetadataFields  []string    `yaml:"metadata_fields"`  // Fields to store as metadata
	DefaultTopK     int         `yaml:"default_top_k"`    // Default search results
	MaxTopK         int         `yaml:"max_top_k"`        // Maximum search results
	Fields          []FieldInfo `yaml:"fields"`           // Field definitions
}

// FieldInfo contains information about a model field
type FieldInfo struct {
	Name     string `yaml:"name"`     // Field name
	Type     string `yaml:"type"`     // Field type: "string", "number", "boolean", "array"
	Purpose  string `yaml:"purpose"`  // Field purpose: "key", "embed", "meta"
	Required bool   `yaml:"required"` // Whether field is required
	Array    bool   `yaml:"array"`    // Whether field is an array
}

// ============================================================================
// MODEL MANAGEMENT
// ============================================================================

// CreateModelMap creates a map of model configurations from YAML config
func CreateModelMap(models []ModelConfig) map[string]ModelConfig {
	modelMap := make(map[string]ModelConfig)
	for _, model := range models {
		// Set defaults if not specified
		if model.Collection == "" {
			model.Collection = model.Name + "s"
		}
		if model.DefaultTopK == 0 {
			model.DefaultTopK = 10
		}
		if model.MaxTopK == 0 {
			model.MaxTopK = 100
		}

		modelMap[model.Name] = model
	}
	return modelMap
}

// ExtractDocumentData extracts embedding text and metadata from document using YAML model config
func ExtractDocumentData(data interface{}, config ModelConfig) (string, map[string]interface{}, error) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return "", nil, fmt.Errorf("document data must be a map[string]interface{}")
	}

	// Extract embedding fields
	var embeddingTexts []string
	for _, fieldName := range config.EmbeddingFields {
		if value, exists := dataMap[fieldName]; exists {
			if strValue, ok := value.(string); ok {
				embeddingTexts = append(embeddingTexts, strValue)
			}
		}
	}

	// Extract metadata
	metadata := make(map[string]interface{})
	for _, fieldName := range config.MetadataFields {
		if value, exists := dataMap[fieldName]; exists {
			metadata[fieldName] = value
		}
	}

	return strings.Join(embeddingTexts, " "), metadata, nil
}

// GetAllModelNames returns all registered model names from a models map
func GetAllModelNames(models map[string]ModelConfig) []string {
	var names []string
	for name := range models {
		names = append(names, name)
	}
	return names
}

// ValidateModelConfig validates a model configuration
func ValidateModelConfig(config ModelConfig) error {
	if config.Name == "" {
		return fmt.Errorf("model name is required")
	}

	if len(config.EmbeddingFields) == 0 {
		return fmt.Errorf("at least one embedding field is required")
	}

	// Validate field purposes
	for _, field := range config.Fields {
		if field.Purpose != "key" && field.Purpose != "embed" && field.Purpose != "meta" {
			return fmt.Errorf("field purpose must be 'key', 'embed', or 'meta', got: %s", field.Purpose)
		}
	}

	return nil
}
