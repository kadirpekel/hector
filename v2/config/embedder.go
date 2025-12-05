// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import "fmt"

// EmbedderConfig configures an embedding provider for semantic search.
//
// Embedders convert text to vector embeddings for similarity search.
// They are used by the memory IndexService for semantic retrieval.
//
// Example:
//
//	embedders:
//	  default:
//	    provider: openai
//	    model: text-embedding-3-small
//	    api_key: ${OPENAI_API_KEY}
//
//	  local:
//	    provider: ollama
//	    model: nomic-embed-text
//	    base_url: http://localhost:11434
type EmbedderConfig struct {
	// Provider specifies the embedding service.
	// Values: "openai", "ollama", "cohere"
	Provider string `yaml:"provider,omitempty"`

	// Model is the embedding model name.
	// OpenAI: "text-embedding-3-small", "text-embedding-3-large"
	// Ollama: "nomic-embed-text", "all-minilm:l6-v2"
	// Cohere: "embed-english-v3.0", "embed-multilingual-v3.0", "embed-v4.0"
	Model string `yaml:"model,omitempty"`

	// APIKey for the embedding provider (OpenAI and Cohere require this).
	// Can use environment variable expansion: ${OPENAI_API_KEY}
	APIKey string `yaml:"api_key,omitempty"`

	// BaseURL for the API endpoint.
	// OpenAI default: https://api.openai.com/v1
	// Ollama default: http://localhost:11434
	// Cohere default: https://api.cohere.com
	BaseURL string `yaml:"base_url,omitempty"`

	// Dimension of the embedding vectors (auto-detected if 0).
	Dimension int `yaml:"dimension,omitempty"`

	// Timeout in seconds for API requests (default: 30).
	Timeout int `yaml:"timeout,omitempty"`

	// BatchSize for batch embedding requests (default: 100 for OpenAI/Ollama, 96 for Cohere).
	BatchSize int `yaml:"batch_size,omitempty"`

	// EncodingFormat for OpenAI API (optional).
	// Values: "float" (default), "base64"
	// Note: Currently only "float" is supported in response parsing.
	EncodingFormat string `yaml:"encoding_format,omitempty"`

	// User for OpenAI API (optional).
	// A unique identifier representing your end-user, which can help OpenAI monitor and detect abuse.
	User string `yaml:"user,omitempty"`

	// InputType for Cohere v3+ models (required).
	// Values: "search_document", "search_query", "classification", "clustering"
	// Default: "search_document"
	InputType string `yaml:"input_type,omitempty"`

	// OutputDimension for Cohere v4+ models (optional).
	// Values: 256, 512, 1024, 1536
	// If set, overrides model's default dimension.
	OutputDimension int `yaml:"output_dimension,omitempty"`

	// Truncate for Cohere API (optional).
	// Values: "NONE", "START", "END" (default: "END")
	Truncate string `yaml:"truncate,omitempty"`
}

// SetDefaults applies default values.
func (c *EmbedderConfig) SetDefaults() {
	if c.Provider == "" {
		c.Provider = "ollama"
	}

	if c.Model == "" {
		switch c.Provider {
		case "openai":
			c.Model = "text-embedding-3-small"
		case "ollama":
			c.Model = "nomic-embed-text"
		case "cohere":
			c.Model = "embed-english-v3.0"
		default:
			c.Model = "nomic-embed-text"
		}
	}

	if c.BaseURL == "" {
		switch c.Provider {
		case "openai":
			c.BaseURL = "https://api.openai.com/v1"
		case "ollama":
			c.BaseURL = "http://localhost:11434"
		case "cohere":
			c.BaseURL = "https://api.cohere.com"
		}
	}

	if c.Dimension == 0 {
		switch c.Provider {
		case "openai":
			switch c.Model {
			case "text-embedding-3-small":
				c.Dimension = 1536
			case "text-embedding-3-large":
				c.Dimension = 3072
			default:
				c.Dimension = 1536
			}
		case "ollama":
			switch c.Model {
			case "nomic-embed-text", "nomic-embed-text-v2":
				c.Dimension = 768
			case "all-minilm:l6-v2":
				c.Dimension = 384
			default:
				c.Dimension = 768
			}
		case "cohere":
			switch c.Model {
			case "embed-english-v3.0", "embed-multilingual-v3.0":
				c.Dimension = 1024
			case "embed-english-light-v3.0", "embed-multilingual-light-v3.0":
				c.Dimension = 384
			case "embed-v4.0":
				c.Dimension = 1536 // Default for v4, can be overridden with output_dimension
			default:
				c.Dimension = 1024
			}
		}
	}

	if c.Timeout == 0 {
		c.Timeout = 30
	}

	if c.BatchSize == 0 {
		switch c.Provider {
		case "cohere":
			c.BatchSize = 96 // Cohere's maximum per request
		default:
			c.BatchSize = 100
		}
	}

	// Cohere-specific defaults
	if c.Provider == "cohere" {
		if c.InputType == "" {
			c.InputType = "search_document" // Default for semantic search use case
		}
		if c.Truncate == "" {
			c.Truncate = "END" // Default per API spec
		}
	}
}

// Validate checks the embedder configuration.
func (c *EmbedderConfig) Validate() error {
	validProviders := map[string]bool{
		"openai":  true,
		"ollama":  true,
		"cohere":  true,
	}

	if !validProviders[c.Provider] {
		return fmt.Errorf("invalid provider %q (valid: openai, ollama, cohere)", c.Provider)
	}

	if (c.Provider == "openai" || c.Provider == "cohere") && c.APIKey == "" {
		return fmt.Errorf("api_key is required for %s embedder", c.Provider)
	}

	if c.Model == "" {
		return fmt.Errorf("model is required")
	}

	if c.Dimension <= 0 {
		return fmt.Errorf("dimension must be positive")
	}

	// Cohere-specific validation
	if c.Provider == "cohere" {
		validInputTypes := map[string]bool{
			"search_document": true,
			"search_query":    true,
			"classification":  true,
			"clustering":      true,
		}
		if c.InputType != "" && !validInputTypes[c.InputType] {
			return fmt.Errorf("invalid input_type %q for Cohere (valid: search_document, search_query, classification, clustering)", c.InputType)
		}

		if c.OutputDimension > 0 {
			validDims := map[int]bool{
				256:  true,
				512:  true,
				1024: true,
				1536: true,
			}
			if !validDims[c.OutputDimension] {
				return fmt.Errorf("invalid output_dimension %d for Cohere (valid: 256, 512, 1024, 1536)", c.OutputDimension)
			}
		}

		if c.Truncate != "" {
			validTruncate := map[string]bool{
				"NONE":  true,
				"START": true,
				"END":   true,
			}
			if !validTruncate[c.Truncate] {
				return fmt.Errorf("invalid truncate %q for Cohere (valid: NONE, START, END)", c.Truncate)
			}
		}
	}

	return nil
}
