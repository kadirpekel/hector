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

// MetadataExtractor defines the interface for extracting metadata from source code.
//
// Direct port from legacy pkg/context/metadata/extractor.go
type MetadataExtractor interface {
	// Name returns the extractor name
	Name() string

	// CanExtract determines if this extractor can handle the given language
	CanExtract(language string) bool

	// Extract extracts metadata from source code
	Extract(content string, filePath string) (*CodeMetadata, error)
}

// CodeMetadata contains extracted code structure information.
//
// Direct port from legacy pkg/context/metadata/extractor.go
type CodeMetadata struct {
	Functions []FunctionInfo         `json:"functions,omitempty"`
	Types     []TypeInfo             `json:"types,omitempty"`
	Imports   []string               `json:"imports,omitempty"`
	Symbols   map[string]interface{} `json:"symbols,omitempty"`
	Custom    map[string]interface{} `json:"custom,omitempty"`
}

// FunctionInfo contains information about a function.
//
// Direct port from legacy pkg/context/metadata/extractor.go
type FunctionInfo struct {
	Name       string `json:"name"`
	Signature  string `json:"signature,omitempty"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
	Receiver   string `json:"receiver,omitempty"` // For methods
	IsExported bool   `json:"is_exported,omitempty"`
	DocComment string `json:"doc_comment,omitempty"`
}

// TypeInfo contains information about a type (struct, interface, etc.).
//
// Direct port from legacy pkg/context/metadata/extractor.go
type TypeInfo struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"` // "struct", "interface", "alias", etc.
	StartLine  int      `json:"start_line"`
	EndLine    int      `json:"end_line"`
	Fields     []string `json:"fields,omitempty"`
	Methods    []string `json:"methods,omitempty"`
	IsExported bool     `json:"is_exported,omitempty"`
	DocComment string   `json:"doc_comment,omitempty"`
}

// MetadataExtractorRegistry manages metadata extractors.
//
// Direct port from legacy pkg/context/metadata/extractor.go
type MetadataExtractorRegistry struct {
	extractors map[string]MetadataExtractor
}

// NewMetadataExtractorRegistry creates a new metadata extractor registry.
func NewMetadataExtractorRegistry() *MetadataExtractorRegistry {
	return &MetadataExtractorRegistry{
		extractors: make(map[string]MetadataExtractor),
	}
}

// Register adds a metadata extractor for specific languages.
func (r *MetadataExtractorRegistry) Register(extractor MetadataExtractor) {
	r.extractors[extractor.Name()] = extractor
}

// ExtractMetadata tries to extract metadata using the appropriate extractor.
func (r *MetadataExtractorRegistry) ExtractMetadata(language string, content string, filePath string) (*CodeMetadata, error) {
	for _, extractor := range r.extractors {
		if extractor.CanExtract(language) {
			return extractor.Extract(content, filePath)
		}
	}

	// No extractor found - return empty metadata
	return &CodeMetadata{
		Functions: []FunctionInfo{},
		Types:     []TypeInfo{},
		Imports:   []string{},
		Symbols:   make(map[string]interface{}),
		Custom:    make(map[string]interface{}),
	}, nil
}

// GetExtractors returns all registered extractors.
func (r *MetadataExtractorRegistry) GetExtractors() []MetadataExtractor {
	extractors := make([]MetadataExtractor, 0, len(r.extractors))
	for _, ext := range r.extractors {
		extractors = append(extractors, ext)
	}
	return extractors
}
