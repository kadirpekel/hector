package chunking

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/context/metadata"
)

// Chunker defines the interface for content chunking strategies
type Chunker interface {
	// Chunk splits content into smaller pieces
	Chunk(content string, meta *metadata.Metadata) ([]Chunk, error)

	// GetConfig returns the chunker configuration
	GetConfig() ChunkerConfig
}

// Chunk represents a piece of content with position information
type Chunk struct {
	Content   string                 `json:"content"`
	StartLine int                    `json:"start_line"`
	EndLine   int                    `json:"end_line"`
	StartByte int                    `json:"start_byte,omitempty"`
	EndByte   int                    `json:"end_byte,omitempty"`
	Index     int                    `json:"index"`
	Total     int                    `json:"total"`
	Context   *ChunkContext          `json:"context,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ChunkContext provides semantic context for a chunk
type ChunkContext struct {
	FunctionName string `json:"function_name,omitempty"`
	TypeName     string `json:"type_name,omitempty"`
	FilePath     string `json:"file_path,omitempty"`
}

// ChunkerConfig contains chunking configuration
type ChunkerConfig struct {
	Strategy string `json:"strategy"` // "simple", "overlapping", "semantic"
	Size     int    `json:"size"`     // Target size in characters
	Overlap  int    `json:"overlap"`  // Overlap size in characters
}

// DefaultChunkerConfig returns default chunking configuration
func DefaultChunkerConfig() ChunkerConfig {
	return ChunkerConfig{
		Strategy: "simple",
		Size:     800,
		Overlap:  0,
	}
}

// Validate checks if the configuration is valid
func (c *ChunkerConfig) Validate() error {
	if c.Size <= 0 {
		return fmt.Errorf("chunk size must be positive, got %d", c.Size)
	}
	if c.Overlap < 0 {
		return fmt.Errorf("chunk overlap cannot be negative, got %d", c.Overlap)
	}
	if c.Overlap >= c.Size {
		return fmt.Errorf("chunk overlap (%d) must be less than chunk size (%d)", c.Overlap, c.Size)
	}
	validStrategies := map[string]bool{
		"simple":      true,
		"overlapping": true,
		"semantic":    true,
	}
	if !validStrategies[c.Strategy] {
		return fmt.Errorf("invalid chunking strategy: %s (must be 'simple', 'overlapping', or 'semantic')", c.Strategy)
	}
	return nil
}

// NewChunker creates a chunker based on the strategy
func NewChunker(config ChunkerConfig) (Chunker, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	switch config.Strategy {
	case "simple":
		return NewSimpleChunker(config), nil
	case "overlapping":
		return NewOverlappingChunker(config), nil
	case "semantic":
		// Semantic chunking uses metadata to preserve function/type boundaries
		return NewSemanticChunker(config), nil
	default:
		return NewSimpleChunker(config), nil
	}
}
