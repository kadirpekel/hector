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

package rag

import "fmt"

// ChunkerStrategy identifies a chunking strategy.
type ChunkerStrategy string

const (
	// ChunkerSimple splits content by fixed character count.
	// Fast but may split mid-sentence/word.
	ChunkerSimple ChunkerStrategy = "simple"

	// ChunkerOverlapping splits with overlap between chunks.
	// Better for retrieval as context is preserved at boundaries.
	ChunkerOverlapping ChunkerStrategy = "overlapping"

	// ChunkerSemantic splits at natural boundaries (paragraphs, sections).
	// Best quality but more complex and slower.
	ChunkerSemantic ChunkerStrategy = "semantic"
)

// Chunker splits content into smaller pieces for indexing.
//
// Chunking is critical for RAG quality:
//   - Too small: loses context, retrieves fragments
//   - Too large: wastes tokens, dilutes relevance
//   - Good chunking: preserves semantic units, enables precise retrieval
//
// Derived from legacy pkg/context/chunking/chunker.go:Chunker
type Chunker interface {
	// Chunk splits content into pieces.
	//
	// The content is split according to the chunker's strategy.
	// Each chunk includes position information (line numbers, byte offsets)
	// for source mapping.
	//
	// Parameters:
	//   - content: the text to split
	//   - ctx: optional context (e.g., from metadata extraction)
	//
	// Returns chunks ordered by position in the original content.
	Chunk(content string, ctx *ChunkContext) ([]Chunk, error)

	// Strategy returns the chunker strategy name.
	Strategy() ChunkerStrategy

	// Config returns the chunker configuration.
	Config() ChunkerConfig
}

// ChunkerConfig configures chunking behavior.
type ChunkerConfig struct {
	// Strategy is the chunking strategy.
	// Values: "simple", "overlapping", "semantic"
	// Default: "simple"
	Strategy ChunkerStrategy `yaml:"strategy,omitempty"`

	// Size is the target chunk size in characters.
	// Default: 1000
	Size int `yaml:"size,omitempty"`

	// Overlap is the overlap size in characters (for overlapping strategy).
	// Default: 200
	Overlap int `yaml:"overlap,omitempty"`

	// MinSize is the minimum chunk size (chunks smaller than this are merged).
	// Default: 100
	MinSize int `yaml:"min_size,omitempty"`

	// MaxSize is the maximum chunk size (hard limit).
	// Default: 2000
	MaxSize int `yaml:"max_size,omitempty"`

	// Separators are the preferred split points for semantic chunking.
	// Default: ["\n\n", "\n", ". ", " "]
	Separators []string `yaml:"separators,omitempty"`

	// PreserveWords avoids splitting in the middle of words.
	// Default: true
	PreserveWords bool `yaml:"preserve_words,omitempty"`
}

// DefaultChunkerConfig returns sensible defaults.
func DefaultChunkerConfig() ChunkerConfig {
	return ChunkerConfig{
		Strategy:      ChunkerSimple,
		Size:          1000,
		Overlap:       200,
		MinSize:       100,
		MaxSize:       2000,
		Separators:    []string{"\n\n", "\n", ". ", " "},
		PreserveWords: true,
	}
}

// SetDefaults applies default values.
func (c *ChunkerConfig) SetDefaults() {
	if c.Strategy == "" {
		c.Strategy = ChunkerSimple
	}
	if c.Size <= 0 {
		c.Size = 1000
	}
	if c.Overlap < 0 {
		c.Overlap = 0
	}
	if c.MinSize <= 0 {
		c.MinSize = 100
	}
	if c.MaxSize <= 0 {
		c.MaxSize = 2000
	}
	if len(c.Separators) == 0 {
		c.Separators = []string{"\n\n", "\n", ". ", " "}
	}
}

// Validate checks the configuration for errors.
func (c *ChunkerConfig) Validate() error {
	switch c.Strategy {
	case ChunkerSimple, ChunkerOverlapping, ChunkerSemantic, "":
		// Valid
	default:
		return fmt.Errorf("invalid chunker strategy: %q", c.Strategy)
	}

	if c.Size <= 0 {
		return fmt.Errorf("chunk size must be positive, got %d", c.Size)
	}

	if c.Overlap < 0 {
		return fmt.Errorf("overlap must be non-negative, got %d", c.Overlap)
	}

	if c.Overlap >= c.Size {
		return fmt.Errorf("overlap (%d) must be less than size (%d)", c.Overlap, c.Size)
	}

	if c.MinSize > c.Size {
		return fmt.Errorf("min_size (%d) must not exceed size (%d)", c.MinSize, c.Size)
	}

	if c.MaxSize < c.Size {
		return fmt.Errorf("max_size (%d) must be at least size (%d)", c.MaxSize, c.Size)
	}

	return nil
}

// NewChunker creates a chunker from configuration.
func NewChunker(cfg ChunkerConfig) (Chunker, error) {
	cfg.SetDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid chunker config: %w", err)
	}

	switch cfg.Strategy {
	case ChunkerSimple:
		return NewSimpleChunker(cfg), nil

	case ChunkerOverlapping:
		return NewOverlappingChunker(cfg), nil

	case ChunkerSemantic:
		return NewSemanticChunker(cfg), nil

	default:
		return NewSimpleChunker(cfg), nil
	}
}

// NilChunker returns the entire content as a single chunk.
type NilChunker struct{}

func (NilChunker) Chunk(content string, ctx *ChunkContext) ([]Chunk, error) {
	return []Chunk{{
		Content:   content,
		Index:     0,
		Total:     1,
		StartLine: 1,
		EndLine:   countLines(content),
		Context:   ctx,
	}}, nil
}

func (NilChunker) Strategy() ChunkerStrategy {
	return "nil"
}

func (NilChunker) Config() ChunkerConfig {
	return ChunkerConfig{}
}

// countLines counts the number of lines in content.
func countLines(content string) int {
	if len(content) == 0 {
		return 0
	}
	lines := 1
	for _, c := range content {
		if c == '\n' {
			lines++
		}
	}
	return lines
}
