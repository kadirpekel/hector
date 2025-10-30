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
	chunkStartLine := 1 // Line number where current chunk starts (1-indexed)
	chunkStartByte := 0 // Byte offset where current chunk starts
	currentLine := 1    // Current line being processed (1-indexed)
	currentByte := 0    // Current byte offset

	for _, line := range lines {
		lineWithNewline := line + "\n"
		lineLen := len(lineWithNewline)

		// If adding this line would exceed the chunk size, save current chunk
		if currentChunk.Len() > 0 && currentChunk.Len()+lineLen > sc.config.Size {
			chunks = append(chunks, Chunk{
				Content:   currentChunk.String(),
				StartLine: chunkStartLine,
				EndLine:   currentLine - 1, // Exclusive: last line NOT included in this chunk
				StartByte: chunkStartByte,
				EndByte:   currentByte, // Exclusive: bytes up to but not including current line
				Index:     len(chunks),
				Total:     0, // Will be set after all chunks are created
			})

			currentChunk.Reset()
			chunkStartLine = currentLine // Next chunk starts at current line
			chunkStartByte = currentByte // Next chunk starts at current byte offset
		}

		// Add line to current chunk
		currentChunk.WriteString(lineWithNewline)
		currentLine++          // Move to next line (this line is now in the chunk)
		currentByte += lineLen // Advance byte offset
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
