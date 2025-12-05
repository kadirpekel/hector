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

package embedder

import (
	"fmt"
	"time"

	"github.com/kadirpekel/hector/v2/config"
)

// NewEmbedderFromConfig creates an Embedder from configuration.
func NewEmbedderFromConfig(cfg *config.EmbedderConfig) (Embedder, error) {
	if cfg == nil {
		return nil, fmt.Errorf("embedder config is required")
	}

	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid embedder config: %w", err)
	}

	switch cfg.Provider {
	case "openai":
		return NewOpenAIEmbedder(OpenAIConfig{
			APIKey:          cfg.APIKey,
			BaseURL:         cfg.BaseURL,
			Model:           cfg.Model,
			Dimension:       cfg.Dimension,
			Timeout:         time.Duration(cfg.Timeout) * time.Second,
			BatchSize:       cfg.BatchSize,
			EncodingFormat:  cfg.EncodingFormat,
			User:            cfg.User,
		})

	case "ollama":
		return NewOllamaEmbedder(OllamaConfig{
			BaseURL:   cfg.BaseURL,
			Model:     cfg.Model,
			Dimension: cfg.Dimension,
			Timeout:   time.Duration(cfg.Timeout) * time.Second,
		})

	case "cohere":
		var outputDim *int
		if cfg.OutputDimension > 0 {
			outputDim = &cfg.OutputDimension
		}
		return NewCohereEmbedder(CohereConfig{
			APIKey:          cfg.APIKey,
			BaseURL:         cfg.BaseURL,
			Model:           cfg.Model,
			Dimension:       cfg.Dimension,
			Timeout:         time.Duration(cfg.Timeout) * time.Second,
			BatchSize:       cfg.BatchSize,
			InputType:       cfg.InputType,
			OutputDimension: outputDim,
			Truncate:        cfg.Truncate,
		})

	default:
		return nil, fmt.Errorf("unsupported embedder provider: %s (supported: openai, ollama, cohere)", cfg.Provider)
	}
}
