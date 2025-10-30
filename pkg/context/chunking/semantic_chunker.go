package chunking

import (
	"strings"

	"github.com/kadirpekel/hector/pkg/context/metadata"
)

// SemanticChunker implements AST-aware chunking that respects code structure
// It attempts to keep functions and types together when possible
type SemanticChunker struct {
	config ChunkerConfig
}

// NewSemanticChunker creates a new semantic chunker
func NewSemanticChunker(config ChunkerConfig) *SemanticChunker {
	return &SemanticChunker{
		config: config,
	}
}

// Chunk splits content into semantically meaningful chunks
// It uses metadata to identify function and type boundaries
func (sc *SemanticChunker) Chunk(content string, meta *metadata.Metadata) ([]Chunk, error) {
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

	// If no metadata available, fall back to overlapping chunking
	if meta == nil || (len(meta.Functions) == 0 && len(meta.Types) == 0) {
		overlapping := NewOverlappingChunker(sc.config)
		return overlapping.Chunk(content, meta)
	}

	// Build a map of line numbers to semantic units (functions/types)
	type semanticUnit struct {
		startLine int
		endLine   int
		name      string
		unitType  string // "function" or "type"
	}

	var units []semanticUnit

	// Add functions to units
	for _, fn := range meta.Functions {
		units = append(units, semanticUnit{
			startLine: fn.StartLine,
			endLine:   fn.EndLine,
			name:      fn.Name,
			unitType:  "function",
		})
	}

	// Add types to units
	for _, typ := range meta.Types {
		units = append(units, semanticUnit{
			startLine: typ.StartLine,
			endLine:   typ.EndLine,
			name:      typ.Name,
			unitType:  "type",
		})
	}

	// If no semantic units found, fall back to overlapping
	if len(units) == 0 {
		overlapping := NewOverlappingChunker(sc.config)
		return overlapping.Chunk(content, meta)
	}

	var chunks []Chunk
	var currentChunk strings.Builder
	chunkStartLine := 1
	chunkStartByte := 0
	currentByte := 0
	var currentContext *ChunkContext

	// Helper to find semantic unit for a line
	findUnit := func(lineNum int) *semanticUnit {
		for i := range units {
			if lineNum >= units[i].startLine && lineNum <= units[i].endLine {
				return &units[i]
			}
		}
		return nil
	}

	for lineNum, line := range lines {
		lineWithNewline := line + "\n"
		lineLen := len(lineWithNewline)
		actualLineNum := lineNum + 1 // Convert to 1-indexed

		unit := findUnit(actualLineNum)

		// Check if adding this line would exceed chunk size
		if currentChunk.Len() > 0 && currentChunk.Len()+lineLen > sc.config.Size {
			// Try to find a good break point
			// If we're at a function/type boundary, this is a good place to split
			if unit == nil || unit.startLine == actualLineNum {
				// Save current chunk
				chunks = append(chunks, Chunk{
					Content:   currentChunk.String(),
					StartLine: chunkStartLine,
					EndLine:   actualLineNum - 1,
					StartByte: chunkStartByte,
					EndByte:   currentByte,
					Index:     len(chunks),
					Total:     0,
					Context:   currentContext,
				})

				currentChunk.Reset()
				chunkStartLine = actualLineNum
				chunkStartByte = currentByte
				currentContext = nil
			}
			// Otherwise, continue adding to avoid splitting mid-function
			// (unless we really have to - handled below)
		}

		// Force split if chunk becomes too large (2x target size)
		if currentChunk.Len() > 0 && currentChunk.Len() > sc.config.Size*2 {
			chunks = append(chunks, Chunk{
				Content:   currentChunk.String(),
				StartLine: chunkStartLine,
				EndLine:   actualLineNum - 1,
				StartByte: chunkStartByte,
				EndByte:   currentByte,
				Index:     len(chunks),
				Total:     0,
				Context:   currentContext,
			})

			currentChunk.Reset()
			chunkStartLine = actualLineNum
			chunkStartByte = currentByte
			currentContext = nil
		}

		// Add line to current chunk
		currentChunk.WriteString(lineWithNewline)

		// Update context if we're in a semantic unit
		if unit != nil && currentContext == nil {
			currentContext = &ChunkContext{}
			if unit.unitType == "function" {
				currentContext.FunctionName = unit.name
			} else {
				currentContext.TypeName = unit.name
			}
		}

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
			Context:   currentContext,
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
func (sc *SemanticChunker) GetConfig() ChunkerConfig {
	return sc.config
}
