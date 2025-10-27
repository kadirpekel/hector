package chunking

import (
	"strings"

	"github.com/kadirpekel/hector/pkg/context/metadata"
)

// SimpleChunker implements basic line-based chunking
type SimpleChunker struct {
	config ChunkerConfig
}

// NewSimpleChunker creates a new simple chunker
func NewSimpleChunker(config ChunkerConfig) *SimpleChunker {
	return &SimpleChunker{
		config: config,
	}
}

// Chunk splits content into chunks based on line count
func (sc *SimpleChunker) Chunk(content string, meta *metadata.Metadata) ([]Chunk, error) {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// If content fits in one chunk, return it
	if len(content) <= sc.config.Size {
		return []Chunk{{
			Content:   content,
			StartLine: 1,
			EndLine:   totalLines,
			StartByte: 0,
			EndByte:   len(content),
			Index:     0,
			Total:     1,
		}}, nil
	}

	var chunks []Chunk
	var currentChunk strings.Builder
	chunkStartLine := 1
	chunkStartByte := 0
	currentLine := 1
	currentByte := 0

	for _, line := range lines {
		lineWithNewline := line + "\n"
		lineLen := len(lineWithNewline)

		// If adding this line would exceed the chunk size, save current chunk
		if currentChunk.Len() > 0 && currentChunk.Len()+lineLen > sc.config.Size {
			chunks = append(chunks, Chunk{
				Content:   currentChunk.String(),
				StartLine: chunkStartLine,
				EndLine:   currentLine - 1,
				StartByte: chunkStartByte,
				EndByte:   currentByte,
				Index:     len(chunks),
				Total:     0, // Will be set after all chunks are created
			})

			currentChunk.Reset()
			chunkStartLine = currentLine
			chunkStartByte = currentByte
		}

		currentChunk.WriteString(lineWithNewline)
		currentLine++
		currentByte += lineLen
	}

	// Add the last chunk if there's content
	if currentChunk.Len() > 0 {
		chunks = append(chunks, Chunk{
			Content:   currentChunk.String(),
			StartLine: chunkStartLine,
			EndLine:   totalLines,
			StartByte: chunkStartByte,
			EndByte:   len(content),
			Index:     len(chunks),
			Total:     0,
		})
	}

	// Set total count for all chunks
	total := len(chunks)
	for i := range chunks {
		chunks[i].Total = total
	}

	return chunks, nil
}

// GetConfig returns the chunker configuration
func (sc *SimpleChunker) GetConfig() ChunkerConfig {
	return sc.config
}
