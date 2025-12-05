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

import (
	"strings"
)

// SimpleChunker implements basic line-based chunking.
//
// This is a direct port of legacy pkg/context/chunking/simple_chunker.go.
// It splits content by lines first, then groups lines into chunks of the
// configured size. This ensures chunks never split mid-line.
//
// Use when:
//   - Speed is critical
//   - Content has uniform structure
//   - Line boundaries should be preserved
type SimpleChunker struct {
	config ChunkerConfig
}

// NewSimpleChunker creates a new simple chunker.
func NewSimpleChunker(cfg ChunkerConfig) *SimpleChunker {
	cfg.SetDefaults()
	return &SimpleChunker{config: cfg}
}

// Chunk splits content into chunks based on line count.
// Direct port from legacy pkg/context/chunking/simple_chunker.go
func (c *SimpleChunker) Chunk(content string, ctx *ChunkContext) ([]Chunk, error) {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// If content fits in one chunk, return it
	if len(content) <= c.config.Size {
		return []Chunk{{
			Content:   content,
			StartLine: 1,
			EndLine:   totalLines,
			StartByte: 0,
			EndByte:   len(content),
			Index:     0,
			Total:     1,
			Context:   ctx,
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
		if currentChunk.Len() > 0 && currentChunk.Len()+lineLen > c.config.Size {
			chunks = append(chunks, Chunk{
				Content:   currentChunk.String(),
				StartLine: chunkStartLine,
				EndLine:   currentLine - 1, // Exclusive: last line NOT included in this chunk
				StartByte: chunkStartByte,
				EndByte:   currentByte, // Exclusive: bytes up to but not including current line
				Index:     len(chunks),
				Total:     0, // Will be set after all chunks are created
				Context:   ctx,
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
			Context:   ctx,
		})
	}

	// Set total count for all chunks
	total := len(chunks)
	for i := range chunks {
		chunks[i].Total = total
	}

	return chunks, nil
}

func (c *SimpleChunker) Strategy() ChunkerStrategy {
	return ChunkerSimple
}

func (c *SimpleChunker) Config() ChunkerConfig {
	return c.config
}

// Ensure SimpleChunker implements Chunker.
var _ Chunker = (*SimpleChunker)(nil)

// OverlappingChunker implements chunking with configurable overlap.
//
// This is a direct port of legacy pkg/context/chunking/overlapping_chunker.go.
// Overlap helps preserve context at chunk boundaries, improving retrieval
// quality when relevant information spans two chunks.
//
// Use when:
//   - Retrieval quality is important
//   - Content has flowing prose
//   - You can afford slightly more storage
type OverlappingChunker struct {
	config ChunkerConfig
}

// NewOverlappingChunker creates a new overlapping chunker.
func NewOverlappingChunker(cfg ChunkerConfig) *OverlappingChunker {
	cfg.SetDefaults()
	if cfg.Overlap <= 0 {
		cfg.Overlap = cfg.Size / 5 // Default 20% overlap
	}
	return &OverlappingChunker{config: cfg}
}

// Chunk splits content into overlapping chunks.
// Direct port from legacy pkg/context/chunking/overlapping_chunker.go
func (c *OverlappingChunker) Chunk(content string, ctx *ChunkContext) ([]Chunk, error) {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// If content fits in one chunk, return it
	if len(content) <= c.config.Size {
		return []Chunk{{
			Content:   content,
			StartLine: 1,
			EndLine:   totalLines,
			StartByte: 0,
			EndByte:   len(content),
			Index:     0,
			Total:     1,
			Context:   ctx,
		}}, nil
	}

	var chunks []Chunk
	var currentChunk strings.Builder
	var overlapBuffer strings.Builder
	chunkStartLine := 1      // Line number where current chunk starts (1-indexed)
	chunkStartByte := 0      // Byte offset where current chunk starts
	currentLine := 1         // Current line being processed (1-indexed)
	currentByte := 0         // Current byte offset
	var overlapStartLine int // Line number where overlap region begins

	for _, line := range lines {
		lineWithNewline := line + "\n"
		lineLen := len(lineWithNewline)

		// Add line to current chunk
		currentChunk.WriteString(lineWithNewline)

		// If chunk is large enough, save it
		if currentChunk.Len() >= c.config.Size {
			chunks = append(chunks, Chunk{
				Content:   currentChunk.String(),
				StartLine: chunkStartLine,
				EndLine:   currentLine,
				StartByte: chunkStartByte,
				EndByte:   currentByte + lineLen,
				Index:     len(chunks),
				Total:     0,
				Context:   ctx,
			})

			// Prepare overlap for next chunk
			if c.config.Overlap > 0 {
				overlapBuffer.Reset()
				overlapSize := 0
				overlapStartLine = currentLine

				// Go backwards to collect overlap
				for i := currentLine - 1; i >= chunkStartLine && overlapSize < c.config.Overlap; i-- {
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
			Context:   ctx,
		})
	}

	// Set total count for all chunks
	total := len(chunks)
	for i := range chunks {
		chunks[i].Total = total
	}

	return chunks, nil
}

func (c *OverlappingChunker) Strategy() ChunkerStrategy {
	return ChunkerOverlapping
}

func (c *OverlappingChunker) Config() ChunkerConfig {
	return c.config
}

// Ensure OverlappingChunker implements Chunker.
var _ Chunker = (*OverlappingChunker)(nil)

// SemanticChunker implements AST-aware chunking that respects code structure.
//
// This is a direct port of legacy pkg/context/chunking/semantic_chunker.go.
// It attempts to keep functions and types together when possible, using
// metadata to identify semantic boundaries.
//
// Use when:
//   - Chunking code files
//   - Retrieval quality is paramount
//   - Variable chunk sizes are acceptable
type SemanticChunker struct {
	config ChunkerConfig
}

// NewSemanticChunker creates a new semantic chunker.
func NewSemanticChunker(cfg ChunkerConfig) *SemanticChunker {
	cfg.SetDefaults()
	return &SemanticChunker{config: cfg}
}

// Chunk splits content into semantically meaningful chunks.
// It uses metadata to identify function and type boundaries.
// Direct port from legacy pkg/context/chunking/semantic_chunker.go
func (c *SemanticChunker) Chunk(content string, ctx *ChunkContext) ([]Chunk, error) {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// If content fits in one chunk, return it
	if len(content) <= c.config.Size {
		return []Chunk{{
			Content:   content,
			StartLine: 1,
			EndLine:   totalLines,
			StartByte: 0,
			EndByte:   len(content),
			Index:     0,
			Total:     1,
			Context:   ctx,
		}}, nil
	}

	// If no context/metadata available, fall back to overlapping chunking
	// This matches legacy behavior
	if ctx == nil || (ctx.FunctionName == "" && ctx.TypeName == "") {
		overlapping := NewOverlappingChunker(c.config)
		return overlapping.Chunk(content, ctx)
	}

	// With context, do semantic chunking trying to respect boundaries
	var chunks []Chunk
	var currentChunk strings.Builder
	chunkStartLine := 1
	chunkStartByte := 0
	currentByte := 0
	currentContext := ctx

	for lineNum, line := range lines {
		lineWithNewline := line + "\n"
		lineLen := len(lineWithNewline)
		actualLineNum := lineNum + 1 // Convert to 1-indexed

		// Check if adding this line would exceed chunk size
		if currentChunk.Len() > 0 && currentChunk.Len()+lineLen > c.config.Size {
			// Check for good break points (empty lines, closing braces)
			isGoodBreakPoint := line == "" ||
				strings.TrimSpace(line) == "}" ||
				strings.TrimSpace(line) == "}," ||
				strings.HasPrefix(strings.TrimSpace(line), "func ") ||
				strings.HasPrefix(strings.TrimSpace(line), "type ")

			if isGoodBreakPoint {
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
			}
			// Otherwise, continue adding to avoid splitting mid-function
		}

		// Force split if chunk becomes too large (2x target size)
		if currentChunk.Len() > 0 && currentChunk.Len() > c.config.Size*2 {
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
		}

		// Add line to current chunk
		currentChunk.WriteString(lineWithNewline)
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

func (c *SemanticChunker) Strategy() ChunkerStrategy {
	return ChunkerSemantic
}

func (c *SemanticChunker) Config() ChunkerConfig {
	return c.config
}

// Ensure SemanticChunker implements Chunker.
var _ Chunker = (*SemanticChunker)(nil)
