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

package vector

import (
	"fmt"
	"sync"
)

// ProviderType identifies a vector provider implementation.
type ProviderType string

const (
	// ProviderChromem uses chromem-go for embedded vector storage.
	// Zero-config, no external dependencies. Best for development and small deployments.
	ProviderChromem ProviderType = "chromem"

	// ProviderQdrant uses Qdrant vector database.
	// High-performance, supports distributed deployments.
	ProviderQdrant ProviderType = "qdrant"

	// ProviderChroma uses Chroma vector database.
	// Python-native but has Go client support.
	ProviderChroma ProviderType = "chroma"

	// ProviderPinecone uses Pinecone managed vector database.
	// Fully managed cloud service.
	ProviderPinecone ProviderType = "pinecone"

	// ProviderMilvus uses Milvus vector database.
	// Open-source, supports large-scale deployments.
	ProviderMilvus ProviderType = "milvus"

	// ProviderWeaviate uses Weaviate vector database.
	// Supports GraphQL queries and hybrid search.
	ProviderWeaviate ProviderType = "weaviate"
)

// ProviderConfig is the configuration for creating vector providers.
type ProviderConfig struct {
	// Type identifies which provider to create.
	Type ProviderType `yaml:"type"`

	// Chromem configuration (used when Type == "chromem").
	Chromem *ChromemConfig `yaml:"chromem,omitempty"`

	// Qdrant configuration (used when Type == "qdrant").
	Qdrant *QdrantConfig `yaml:"qdrant,omitempty"`

	// Pinecone configuration (used when Type == "pinecone").
	Pinecone *PineconeConfig `yaml:"pinecone,omitempty"`

	// Weaviate configuration (used when Type == "weaviate").
	Weaviate *WeaviateConfig `yaml:"weaviate,omitempty"`

	// Milvus configuration (used when Type == "milvus").
	Milvus *MilvusConfig `yaml:"milvus,omitempty"`

	// Chroma configuration (used when Type == "chroma").
	Chroma *ChromaConfig `yaml:"chroma,omitempty"`
}

// SetDefaults applies default values.
func (c *ProviderConfig) SetDefaults() {
	if c.Type == "" {
		c.Type = ProviderChromem
	}
	if c.Type == ProviderChromem && c.Chromem == nil {
		c.Chromem = &ChromemConfig{}
	}
}

// Validate checks the configuration.
func (c *ProviderConfig) Validate() error {
	switch c.Type {
	case ProviderChromem:
		// Chromem has no required fields
		return nil
	case ProviderQdrant:
		if c.Qdrant == nil {
			return fmt.Errorf("qdrant configuration is required")
		}
		if c.Qdrant.Host == "" {
			return fmt.Errorf("qdrant host is required")
		}
		return nil
	case ProviderPinecone:
		if c.Pinecone == nil {
			return fmt.Errorf("pinecone configuration is required")
		}
		if c.Pinecone.APIKey == "" {
			return fmt.Errorf("pinecone api_key is required")
		}
		return nil
	case ProviderWeaviate:
		if c.Weaviate == nil {
			return fmt.Errorf("weaviate configuration is required")
		}
		if c.Weaviate.Host == "" {
			return fmt.Errorf("weaviate host is required")
		}
		return nil
	case ProviderMilvus:
		if c.Milvus == nil {
			return fmt.Errorf("milvus configuration is required")
		}
		if c.Milvus.Host == "" {
			return fmt.Errorf("milvus host is required")
		}
		return nil
	case ProviderChroma:
		if c.Chroma == nil {
			return fmt.Errorf("chroma configuration is required")
		}
		if c.Chroma.Host == "" {
			return fmt.Errorf("chroma host is required")
		}
		return nil
	case "":
		return fmt.Errorf("provider type is required")
	default:
		return fmt.Errorf("unknown provider type: %q", c.Type)
	}
}

// NewProvider creates a vector provider from configuration.
func NewProvider(cfg *ProviderConfig) (Provider, error) {
	if cfg == nil {
		return NilProvider{}, nil
	}

	switch cfg.Type {
	case ProviderChromem:
		chromemCfg := ChromemConfig{}
		if cfg.Chromem != nil {
			chromemCfg = *cfg.Chromem
		}
		return NewChromemProvider(chromemCfg)

	case ProviderQdrant:
		if cfg.Qdrant == nil {
			return nil, fmt.Errorf("qdrant configuration is required")
		}
		return NewQdrantProvider(*cfg.Qdrant)

	case ProviderPinecone:
		if cfg.Pinecone == nil {
			return nil, fmt.Errorf("pinecone configuration is required")
		}
		return NewPineconeProvider(*cfg.Pinecone)

	case ProviderWeaviate:
		if cfg.Weaviate == nil {
			return nil, fmt.Errorf("weaviate configuration is required")
		}
		return NewWeaviateProvider(*cfg.Weaviate)

	case ProviderMilvus:
		if cfg.Milvus == nil {
			return nil, fmt.Errorf("milvus configuration is required")
		}
		return NewMilvusProvider(*cfg.Milvus)

	case ProviderChroma:
		if cfg.Chroma == nil {
			return nil, fmt.Errorf("chroma configuration is required")
		}
		return NewChromaProvider(*cfg.Chroma)

	default:
		return nil, fmt.Errorf("unknown provider type: %q", cfg.Type)
	}
}

// Registry manages named vector providers.
//
// This allows multiple providers to be configured and accessed by name,
// similar to how databases or embedders are managed.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(name string, provider Provider) error {
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %q already registered", name)
	}

	r.providers[name] = provider
	return nil
}

// Get retrieves a provider by name.
func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// MustGet retrieves a provider by name or panics.
func (r *Registry) MustGet(name string) Provider {
	p, ok := r.Get(name)
	if !ok {
		panic(fmt.Sprintf("vector provider %q not found", name))
	}
	return p
}

// List returns all registered provider names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// Close closes all registered providers.
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for name, p := range r.providers {
		if err := p.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close provider %q: %w", name, err))
		}
	}

	r.providers = make(map[string]Provider)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing providers: %v", errs)
	}
	return nil
}
