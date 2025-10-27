package chunking

import (
	"strings"

	"github.com/kadirpekel/hector/pkg/context/metadata"
)

// OverlappingChunker implements chunking with configurable overlap
type OverlappingChunker struct {
	config ChunkerConfig
}

// NewOverlappingChunker creates a new overlapping chunker
func NewOverlappingChunker(config ChunkerConfig) *OverlappingChunker {
	return &OverlappingChunker{
		config: config,
	}
}

// Chunk splits content into overlapping chunks
func (oc *OverlappingChunker) Chunk(content string, meta *metadata.Metadata) ([]Chunk, error) {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// If content fits in one chunk, return it
	if len(content) <= oc.config.Size {
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
	var overlapBuffer strings.Builder
	chunkStartLine := 1
	chunkStartByte := 0
	currentLine := 1
	currentByte := 0
	var overlapStartLine int

	for _, line := range lines {
		lineWithNewline := line + "\n"
		lineLen := len(lineWithNewline)

		// Add line to current chunk
		currentChunk.WriteString(lineWithNewline)

		// If chunk is large enough, save it
		if currentChunk.Len() >= oc.config.Size {
			chunks = append(chunks, Chunk{
				Content:   currentChunk.String(),
				StartLine: chunkStartLine,
				EndLine:   currentLine,
				StartByte: chunkStartByte,
				EndByte:   currentByte + lineLen,
				Index:     len(chunks),
				Total:     0,
			})

			// Prepare overlap for next chunk
			if oc.config.Overlap > 0 {
				overlapBuffer.Reset()
				overlapSize := 0
				overlapStartLine = currentLine

				// Go backwards to collect overlap
				for i := currentLine - 1; i >= chunkStartLine && overlapSize < oc.config.Overlap; i-- {
					if i-1 < len(lines) {
						overlapLine := lines[i-1] + "\n"
						overlapSize += len(overlapLine)
						// Prepend to overlap buffer
						temp := overlapLine + overlapBuffer.String()
						overlapBuffer.Reset()
						overlapBuffer.WriteString(temp)
						overlapStartLine = i
					}
				}

				// Start next chunk with overlap
				currentChunk.Reset()
				currentChunk.WriteString(overlapBuffer.String())
				chunkStartLine = overlapStartLine
				chunkStartByte = currentByte + lineLen - overlapBuffer.Len()
			} else {
				// No overlap - start fresh
				currentChunk.Reset()
				chunkStartLine = currentLine + 1
				chunkStartByte = currentByte + lineLen
			}
		}

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
func (oc *OverlappingChunker) GetConfig() ChunkerConfig {
	return oc.config
}
