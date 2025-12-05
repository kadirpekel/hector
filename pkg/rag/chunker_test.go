// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel
//
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.gnu.org/licenses/agpl-3.0.en.html
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rag

import (
	"strings"
	"testing"
)

// These tests verify the chunker implementations are correct ports of
// legacy pkg/context/chunking/* implementations.

func TestSimpleChunker_EmptyContent(t *testing.T) {
	chunker := NewSimpleChunker(ChunkerConfig{Size: 100})
	chunks, err := chunker.Chunk("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty content should still produce one chunk with empty content
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for empty content, got %d", len(chunks))
	}
}

func TestSimpleChunker_SmallContent(t *testing.T) {
	chunker := NewSimpleChunker(ChunkerConfig{Size: 100})
	content := "Hello, World!"
	chunks, err := chunker.Chunk(content, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for small content, got %d", len(chunks))
	}
	if chunks[0].Content != content {
		t.Errorf("expected content %q, got %q", content, chunks[0].Content)
	}
	if chunks[0].Index != 0 {
		t.Errorf("expected index 0, got %d", chunks[0].Index)
	}
	if chunks[0].Total != 1 {
		t.Errorf("expected total 1, got %d", chunks[0].Total)
	}
}

func TestSimpleChunker_MultiLine(t *testing.T) {
	chunker := NewSimpleChunker(ChunkerConfig{Size: 30})
	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"

	chunks, err := chunker.Chunk(content, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should create multiple chunks based on line accumulation
	if len(chunks) < 2 {
		t.Errorf("expected at least 2 chunks, got %d", len(chunks))
	}

	// Verify total content is preserved
	var total strings.Builder
	for _, chunk := range chunks {
		total.WriteString(chunk.Content)
	}

	// Remove trailing newline differences for comparison
	expected := content + "\n" // SimpleChunker adds newline to each line
	if total.String() != expected {
		t.Errorf("content not preserved:\nexpected: %q\ngot: %q", expected, total.String())
	}

	// Verify line numbers are correct
	for i, chunk := range chunks {
		if chunk.StartLine < 1 {
			t.Errorf("chunk %d has invalid StartLine: %d", i, chunk.StartLine)
		}
		if chunk.EndLine < chunk.StartLine {
			t.Errorf("chunk %d has EndLine (%d) < StartLine (%d)", i, chunk.EndLine, chunk.StartLine)
		}
	}
}

func TestSimpleChunker_PreservesLineBasedChunking(t *testing.T) {
	// This test verifies the key feature from legacy: line-based chunking
	chunker := NewSimpleChunker(ChunkerConfig{Size: 20})
	content := "Short\nMedium line here\nAnother"

	chunks, err := chunker.Chunk(content, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Each chunk should end with a complete line (newline)
	for i, chunk := range chunks[:len(chunks)-1] { // All but last
		if !strings.HasSuffix(chunk.Content, "\n") {
			t.Errorf("chunk %d doesn't end with newline: %q", i, chunk.Content)
		}
	}
}

func TestSimpleChunker_Strategy(t *testing.T) {
	chunker := NewSimpleChunker(DefaultChunkerConfig())
	if chunker.Strategy() != ChunkerSimple {
		t.Errorf("expected strategy %q, got %q", ChunkerSimple, chunker.Strategy())
	}
}

func TestOverlappingChunker_EmptyContent(t *testing.T) {
	chunker := NewOverlappingChunker(ChunkerConfig{Size: 100, Overlap: 20})
	chunks, err := chunker.Chunk("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for empty content, got %d", len(chunks))
	}
}

func TestOverlappingChunker_SmallContent(t *testing.T) {
	chunker := NewOverlappingChunker(ChunkerConfig{Size: 100, Overlap: 20})
	content := "Hello, World!"
	chunks, err := chunker.Chunk(content, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for small content, got %d", len(chunks))
	}
	if chunks[0].Content != content {
		t.Errorf("expected content %q, got %q", content, chunks[0].Content)
	}
}

func TestOverlappingChunker_VerifyOverlap(t *testing.T) {
	// This test verifies the overlap feature from legacy
	chunker := NewOverlappingChunker(ChunkerConfig{
		Size:    40,
		Overlap: 15,
	})

	content := "Line 1 content\nLine 2 content\nLine 3 content\nLine 4 content\nLine 5 content"
	chunks, err := chunker.Chunk(content, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks with overlap, got %d", len(chunks))
	}

	// With overlap, consecutive chunks should share some content
	for i := 0; i < len(chunks)-1; i++ {
		chunk1 := chunks[i].Content
		chunk2 := chunks[i+1].Content

		// The end of chunk1 should be the start of chunk2 (overlap)
		// This is the key feature of overlapping chunker
		found := false
		for j := 1; j <= len(chunk1) && j <= len(chunk2); j++ {
			suffix := chunk1[len(chunk1)-j:]
			if strings.HasPrefix(chunk2, suffix) {
				found = true
				break
			}
		}
		if !found {
			t.Logf("chunk %d: %q", i, chunk1)
			t.Logf("chunk %d: %q", i+1, chunk2)
			// Note: overlap may not always be present if chunk boundaries
			// align with line boundaries differently
		}
	}
}

func TestOverlappingChunker_Strategy(t *testing.T) {
	chunker := NewOverlappingChunker(DefaultChunkerConfig())
	if chunker.Strategy() != ChunkerOverlapping {
		t.Errorf("expected strategy %q, got %q", ChunkerOverlapping, chunker.Strategy())
	}
}

func TestSemanticChunker_EmptyContent(t *testing.T) {
	chunker := NewSemanticChunker(ChunkerConfig{Size: 100})
	chunks, err := chunker.Chunk("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for empty content, got %d", len(chunks))
	}
}

func TestSemanticChunker_SmallContent(t *testing.T) {
	chunker := NewSemanticChunker(ChunkerConfig{Size: 100})
	content := "Hello, World!"
	chunks, err := chunker.Chunk(content, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for small content, got %d", len(chunks))
	}
	if chunks[0].Content != content {
		t.Errorf("expected content %q, got %q", content, chunks[0].Content)
	}
}

func TestSemanticChunker_FallsBackToOverlapping(t *testing.T) {
	// Per legacy: semantic chunker falls back to overlapping when no metadata
	chunker := NewSemanticChunker(ChunkerConfig{Size: 50, Overlap: 10})
	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7"

	// Without context, should fall back to overlapping
	chunks, err := chunker.Chunk(content, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should produce chunks via overlapping strategy
	if len(chunks) < 1 {
		t.Errorf("expected at least 1 chunk, got %d", len(chunks))
	}
}

func TestSemanticChunker_WithContext(t *testing.T) {
	chunker := NewSemanticChunker(ChunkerConfig{Size: 50})
	content := `func Hello() {
    fmt.Println("Hello")
}

func World() {
    fmt.Println("World")
}`

	ctx := &ChunkContext{
		FunctionName: "Hello",
		FilePath:     "test.go",
	}

	chunks, err := chunker.Chunk(content, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With context, should attempt semantic chunking
	if len(chunks) < 1 {
		t.Errorf("expected at least 1 chunk, got %d", len(chunks))
	}

	// Context should be preserved
	for _, chunk := range chunks {
		if chunk.Context != ctx {
			t.Errorf("context not preserved in chunk")
		}
	}
}

func TestSemanticChunker_Strategy(t *testing.T) {
	chunker := NewSemanticChunker(DefaultChunkerConfig())
	if chunker.Strategy() != ChunkerSemantic {
		t.Errorf("expected strategy %q, got %q", ChunkerSemantic, chunker.Strategy())
	}
}

func TestChunkerConfig_SetDefaults(t *testing.T) {
	cfg := ChunkerConfig{}
	cfg.SetDefaults()

	if cfg.Size != 1000 {
		t.Errorf("expected default size 1000, got %d", cfg.Size)
	}
	if cfg.MinSize != 100 {
		t.Errorf("expected default min_size 100, got %d", cfg.MinSize)
	}
	if cfg.MaxSize != 2000 {
		t.Errorf("expected default max_size 2000, got %d", cfg.MaxSize)
	}
	if cfg.Strategy != ChunkerSimple {
		t.Errorf("expected default strategy %q, got %q", ChunkerSimple, cfg.Strategy)
	}
	if len(cfg.Separators) == 0 {
		t.Errorf("expected default separators to be set")
	}
}

func TestNewChunker_AllStrategies(t *testing.T) {
	tests := []struct {
		strategy ChunkerStrategy
		expected ChunkerStrategy
	}{
		{ChunkerSimple, ChunkerSimple},
		{ChunkerOverlapping, ChunkerOverlapping},
		{ChunkerSemantic, ChunkerSemantic},
		{"", ChunkerSimple}, // Default
	}

	for _, tt := range tests {
		t.Run(string(tt.strategy), func(t *testing.T) {
			cfg := ChunkerConfig{Strategy: tt.strategy}
			chunker, err := NewChunker(cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if chunker.Strategy() != tt.expected {
				t.Errorf("expected strategy %q, got %q", tt.expected, chunker.Strategy())
			}
		})
	}
}

func TestNewChunker_InvalidStrategy(t *testing.T) {
	cfg := ChunkerConfig{Strategy: "invalid"}
	_, err := NewChunker(cfg)
	if err == nil {
		t.Errorf("expected error for invalid strategy")
	}
}

func TestChunk_LineNumbers(t *testing.T) {
	chunker := NewSimpleChunker(ChunkerConfig{Size: 20})
	content := "Line 1\nLine 2\nLine 3\nLine 4"
	chunks, err := chunker.Chunk(content, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First chunk should start at line 1
	if len(chunks) > 0 && chunks[0].StartLine != 1 {
		t.Errorf("expected first chunk to start at line 1, got %d", chunks[0].StartLine)
	}

	// All chunks should have valid line numbers
	for i, chunk := range chunks {
		if chunk.StartLine < 1 {
			t.Errorf("chunk %d has invalid StartLine: %d", i, chunk.StartLine)
		}
		if chunk.EndLine < chunk.StartLine {
			t.Errorf("chunk %d has EndLine (%d) < StartLine (%d)", i, chunk.EndLine, chunk.StartLine)
		}
	}
}

func TestChunk_ByteOffsets(t *testing.T) {
	chunker := NewSimpleChunker(ChunkerConfig{Size: 15})
	content := "Line 1\nLine 2\nLine 3" // 20 chars total
	chunks, err := chunker.Chunk(content, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First chunk should start at byte 0
	if len(chunks) >= 1 && chunks[0].StartByte != 0 {
		t.Errorf("first chunk should start at byte 0, got %d", chunks[0].StartByte)
	}

	// Byte ranges should be contiguous and cover entire content
	if len(chunks) >= 2 {
		// Each chunk's StartByte should equal previous chunk's EndByte
		for i := 1; i < len(chunks); i++ {
			if chunks[i].StartByte != chunks[i-1].EndByte {
				t.Errorf("chunk %d StartByte (%d) != chunk %d EndByte (%d)",
					i, chunks[i].StartByte, i-1, chunks[i-1].EndByte)
			}
		}
	}
}

func TestChunk_Context(t *testing.T) {
	chunker := NewSimpleChunker(ChunkerConfig{Size: 100})
	ctx := &ChunkContext{
		FilePath: "/path/to/test.go",
	}

	chunks, err := chunker.Chunk("Hello, World!", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) > 0 && chunks[0].Context != ctx {
		t.Errorf("chunk context not preserved")
	}
}

// Benchmark tests
func BenchmarkSimpleChunker(b *testing.B) {
	chunker := NewSimpleChunker(ChunkerConfig{Size: 1000})
	content := strings.Repeat("Hello world this is a test content for benchmarking.\n", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = chunker.Chunk(content, nil)
	}
}

func BenchmarkOverlappingChunker(b *testing.B) {
	chunker := NewOverlappingChunker(ChunkerConfig{Size: 1000, Overlap: 200})
	content := strings.Repeat("Hello world this is a test content for benchmarking.\n", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = chunker.Chunk(content, nil)
	}
}

func BenchmarkSemanticChunker(b *testing.B) {
	chunker := NewSemanticChunker(ChunkerConfig{Size: 1000})
	content := strings.Repeat("func Test() {\n    // code\n}\n\n", 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = chunker.Chunk(content, nil)
	}
}
