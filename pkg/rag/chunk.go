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

// Package rag provides Retrieval-Augmented Generation (RAG) capabilities.
//
// # Architecture
//
// The RAG package follows a layered architecture:
//
//	┌─────────────────────────────────────────────────────────────────────────┐
//	│  SearchEngine (v2/rag/search.go)                                        │
//	│  • Query processing, retrieval, reranking                               │
//	├─────────────────────────────────────────────────────────────────────────┤
//	│  Chunker (v2/rag/chunker.go)                                            │
//	│  • Content splitting strategies                                         │
//	├─────────────────────────────────────────────────────────────────────────┤
//	│  Shared Foundation                                                       │
//	│  ┌───────────────────────────┐ ┌───────────────────────────┐           │
//	│  │ v2/vector/provider.go    │ │ v2/embedder/embedder.go   │           │
//	│  └───────────────────────────┘ └───────────────────────────┘           │
//	└─────────────────────────────────────────────────────────────────────────┘
//
// # Usage
//
// Basic usage for document ingestion and search:
//
//	// Create search engine
//	engine, _ := rag.NewSearchEngine(rag.SearchEngineConfig{
//	    Provider: vectorProvider,
//	    Embedder: embedder,
//	})
//
//	// Ingest document
//	engine.IngestDocument(ctx, "doc1", "Document content...", metadata)
//
//	// Search
//	results, _ := engine.Search(ctx, "query", 10)
//
// # Integration with Memory
//
// The RAG package shares the same vector.Provider abstraction as the memory
// package, allowing both to use the same vector database backend.
package rag

// Chunk represents a piece of content with position and context information.
//
// Chunks are the fundamental unit of retrieval in RAG systems. Each chunk:
//   - Contains a portion of the original document
//   - Tracks its position within the source
//   - Preserves semantic context for better retrieval
//
// Derived from legacy pkg/context/chunking/chunker.go:Chunk
type Chunk struct {
	// Content is the actual text content of this chunk.
	Content string `json:"content"`

	// Index is the chunk's position within the document (0-based).
	Index int `json:"index"`

	// Total is the total number of chunks for this document.
	Total int `json:"total"`

	// StartLine is the starting line number in the source document (1-based).
	StartLine int `json:"start_line"`

	// EndLine is the ending line number in the source document (1-based).
	EndLine int `json:"end_line"`

	// StartByte is the byte offset where this chunk begins (optional).
	StartByte int `json:"start_byte,omitempty"`

	// EndByte is the byte offset where this chunk ends (optional).
	EndByte int `json:"end_byte,omitempty"`

	// Context provides semantic context for the chunk (function name, type, etc.).
	Context *ChunkContext `json:"context,omitempty"`

	// Metadata contains additional chunk-specific information.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ChunkContext provides semantic context for a chunk.
//
// This is especially useful for code files where understanding
// the function or type a chunk belongs to improves retrieval quality.
type ChunkContext struct {
	// FunctionName is the containing function/method name (for code).
	FunctionName string `json:"function_name,omitempty"`

	// TypeName is the containing type/class name (for code).
	TypeName string `json:"type_name,omitempty"`

	// FilePath is the source file path.
	FilePath string `json:"file_path,omitempty"`

	// Language is the detected programming language (for code).
	Language string `json:"language,omitempty"`

	// Section is the document section name (for prose documents).
	Section string `json:"section,omitempty"`

	// ParentID links to a parent chunk (for hierarchical retrieval).
	ParentID string `json:"parent_id,omitempty"`
}

// Document represents a document to be indexed.
//
// Documents go through the following pipeline:
//  1. Content extraction (if binary)
//  2. Chunking (split into searchable pieces)
//  3. Embedding (convert to vectors)
//  4. Indexing (store in vector database)
type Document struct {
	// ID is the unique identifier for this document.
	ID string `json:"id"`

	// Content is the text content to be indexed.
	Content string `json:"content"`

	// Title is the document title (optional).
	Title string `json:"title,omitempty"`

	// SourcePath is the path to the source file (for file-based documents).
	SourcePath string `json:"source_path,omitempty"`

	// MimeType is the content type (e.g., "text/plain", "text/markdown").
	MimeType string `json:"mime_type,omitempty"`

	// Size is the content size in bytes.
	Size int64 `json:"size"`

	// Metadata contains additional document information.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// SearchResult represents a single search result.
//
// Results are ordered by Score (highest first). The Score semantics
// depend on whether reranking was applied:
//   - Without reranking: vector similarity (0.0 to 1.0)
//   - With reranking: LLM-determined position score
type SearchResult struct {
	// ID is the chunk/document identifier.
	ID string `json:"id"`

	// Content is the matched content.
	Content string `json:"content"`

	// Score represents relevance (higher is better).
	Score float32 `json:"score"`

	// DocumentID is the parent document identifier.
	DocumentID string `json:"document_id,omitempty"`

	// ChunkIndex is the chunk position within the document.
	ChunkIndex int `json:"chunk_index,omitempty"`

	// Metadata contains additional result information.
	Metadata map[string]any `json:"metadata,omitempty"`

	// Highlights contains matched text spans (optional).
	Highlights []string `json:"highlights,omitempty"`
}

// SearchRequest represents a search query.
type SearchRequest struct {
	// Query is the search query text.
	Query string `json:"query"`

	// Collection scopes the search to a specific collection.
	Collection string `json:"collection,omitempty"`

	// TopK is the maximum number of results to return.
	TopK int `json:"top_k,omitempty"`

	// Threshold filters results below this score.
	Threshold float32 `json:"threshold,omitempty"`

	// Filter applies metadata filtering.
	Filter map[string]any `json:"filter,omitempty"`

	// Options contains search-specific options.
	Options *SearchOptions `json:"options,omitempty"`
}

// SearchOptions configures search behavior.
type SearchOptions struct {
	// Mode specifies the search mode: "vector", "keyword", "hybrid".
	Mode string `json:"mode,omitempty"`

	// EnableHyDE enables Hypothetical Document Embeddings.
	EnableHyDE bool `json:"enable_hyde,omitempty"`

	// EnableRerank enables LLM-based reranking.
	EnableRerank bool `json:"enable_rerank,omitempty"`

	// EnableMultiQuery enables query expansion.
	EnableMultiQuery bool `json:"enable_multi_query,omitempty"`

	// NumQueries is the number of query variants for multi-query.
	NumQueries int `json:"num_queries,omitempty"`
}

// SearchResponse contains search results.
type SearchResponse struct {
	// Results contains the matched documents/chunks.
	Results []SearchResult `json:"results"`

	// TotalMatches is the total number of matches (before limit).
	TotalMatches int `json:"total_matches,omitempty"`

	// SearchTimeMs is the search duration in milliseconds.
	SearchTimeMs int64 `json:"search_time_ms,omitempty"`

	// QueryExpansions contains expanded queries (if multi-query enabled).
	QueryExpansions []string `json:"query_expansions,omitempty"`
}

// SetDefaults applies default values to SearchRequest.
func (r *SearchRequest) SetDefaults() {
	if r.TopK <= 0 {
		r.TopK = 10
	}
	if r.Options == nil {
		r.Options = &SearchOptions{}
	}
	if r.Options.Mode == "" {
		r.Options.Mode = "vector"
	}
}
