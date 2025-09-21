package hector

import (
	"fmt"
)

// ============================================================================
// MODEL CONFIGURATION
// ============================================================================

// ModelConfig holds configuration for a vector model
type ModelConfig struct {
	Name        string `yaml:"name"`          // Model name (e.g., "document", "article")
	Collection  string `yaml:"collection"`    // Vector collection name
	DefaultTopK int    `yaml:"default_top_k"` // Default search results
	MaxTopK     int    `yaml:"max_top_k"`     // Maximum search results

	// Each model can have its own ingestion configuration
	Ingestion *ModelIngestionConfig `yaml:"ingestion"`
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

// ExtractDocumentData extracts embedding text and metadata from document using universal document structure
func ExtractDocumentData(data interface{}) (string, map[string]interface{}, error) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return "", nil, fmt.Errorf("document data must be a map[string]interface{}")
	}

	// Universal document structure: always embed "content" field
	content, exists := dataMap["content"]
	if !exists {
		return "", nil, fmt.Errorf("document must have 'content' field")
	}

	contentStr, ok := content.(string)
	if !ok {
		return "", nil, fmt.Errorf("content field must be a string")
	}

	// Universal metadata: include all fields except content for embedding
	metadata := make(map[string]interface{})
	for key, value := range dataMap {
		if key != "content" {
			metadata[key] = value
		}
	}

	return contentStr, metadata, nil
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
	return nil
}
