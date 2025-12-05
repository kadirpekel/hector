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
	"fmt"
	"time"
)

// DocumentStoreError represents an error in document store operations.
//
// Inspired by legacy pkg/context/document_store.go error handling
type DocumentStoreError struct {
	StoreName string    // Name of the document store
	Operation string    // Operation that failed
	Message   string    // Error message
	FilePath  string    // File path if applicable
	Err       error     // Underlying error
	Timestamp time.Time // When the error occurred
}

// Error implements the error interface.
func (e *DocumentStoreError) Error() string {
	msg := fmt.Sprintf("[%s] %s: %s", e.StoreName, e.Operation, e.Message)
	if e.FilePath != "" {
		msg += fmt.Sprintf(" (file: %s)", e.FilePath)
	}
	if e.Err != nil {
		msg += fmt.Sprintf(": %v", e.Err)
	}
	return msg
}

// Unwrap returns the underlying error.
func (e *DocumentStoreError) Unwrap() error {
	return e.Err
}

// NewDocumentStoreError creates a new DocumentStoreError.
func NewDocumentStoreError(storeName, operation, message, filePath string, err error) *DocumentStoreError {
	return &DocumentStoreError{
		StoreName: storeName,
		Operation: operation,
		Message:   message,
		FilePath:  filePath,
		Err:       err,
		Timestamp: time.Now(),
	}
}

// SearchError represents an error during search operations.
//
// Inspired by legacy pkg/context error handling
type SearchError struct {
	Component string // Component that failed (e.g., "embedder", "vector_db", "reranker")
	Operation string // Operation that failed
	Message   string // Error message
	Query     string // Query that caused the error
	Err       error  // Underlying error
}

// Error implements the error interface.
func (e *SearchError) Error() string {
	msg := fmt.Sprintf("[%s] %s: %s", e.Component, e.Operation, e.Message)
	if e.Query != "" {
		// Truncate query if too long
		query := e.Query
		if len(query) > 50 {
			query = query[:50] + "..."
		}
		msg += fmt.Sprintf(" (query: %q)", query)
	}
	if e.Err != nil {
		msg += fmt.Sprintf(": %v", e.Err)
	}
	return msg
}

// Unwrap returns the underlying error.
func (e *SearchError) Unwrap() error {
	return e.Err
}

// NewSearchError creates a new SearchError.
func NewSearchError(component, operation, message, query string, err error) *SearchError {
	return &SearchError{
		Component: component,
		Operation: operation,
		Message:   message,
		Query:     query,
		Err:       err,
	}
}

// ExtractionError represents an error during content extraction.
type ExtractionError struct {
	Extractor string // Extractor name
	FilePath  string // File path
	Message   string // Error message
	Err       error  // Underlying error
}

// Error implements the error interface.
func (e *ExtractionError) Error() string {
	msg := fmt.Sprintf("[%s] extraction failed for %s: %s", e.Extractor, e.FilePath, e.Message)
	if e.Err != nil {
		msg += fmt.Sprintf(": %v", e.Err)
	}
	return msg
}

// Unwrap returns the underlying error.
func (e *ExtractionError) Unwrap() error {
	return e.Err
}

// NewExtractionError creates a new ExtractionError.
func NewExtractionError(extractor, filePath, message string, err error) *ExtractionError {
	return &ExtractionError{
		Extractor: extractor,
		FilePath:  filePath,
		Message:   message,
		Err:       err,
	}
}

// ChunkingError represents an error during document chunking.
type ChunkingError struct {
	Strategy   string // Chunking strategy
	DocumentID string // Document ID
	Message    string // Error message
	Err        error  // Underlying error
}

// Error implements the error interface.
func (e *ChunkingError) Error() string {
	msg := fmt.Sprintf("[%s] chunking failed for %s: %s", e.Strategy, e.DocumentID, e.Message)
	if e.Err != nil {
		msg += fmt.Sprintf(": %v", e.Err)
	}
	return msg
}

// Unwrap returns the underlying error.
func (e *ChunkingError) Unwrap() error {
	return e.Err
}

// NewChunkingError creates a new ChunkingError.
func NewChunkingError(strategy, documentID, message string, err error) *ChunkingError {
	return &ChunkingError{
		Strategy:   strategy,
		DocumentID: documentID,
		Message:    message,
		Err:        err,
	}
}

// IndexError represents an error during indexing operations.
type IndexError struct {
	StoreName  string // Document store name
	DocumentID string // Document ID
	Operation  string // Operation (e.g., "embed", "upsert", "delete")
	Message    string // Error message
	Err        error  // Underlying error
}

// Error implements the error interface.
func (e *IndexError) Error() string {
	msg := fmt.Sprintf("[%s] index %s failed for %s: %s", e.StoreName, e.Operation, e.DocumentID, e.Message)
	if e.Err != nil {
		msg += fmt.Sprintf(": %v", e.Err)
	}
	return msg
}

// Unwrap returns the underlying error.
func (e *IndexError) Unwrap() error {
	return e.Err
}

// NewIndexError creates a new IndexError.
func NewIndexError(storeName, documentID, operation, message string, err error) *IndexError {
	return &IndexError{
		StoreName:  storeName,
		DocumentID: documentID,
		Operation:  operation,
		Message:    message,
		Err:        err,
	}
}
