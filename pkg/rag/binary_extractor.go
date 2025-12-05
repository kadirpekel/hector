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
	"context"
	"path/filepath"
	"strings"
	"time"
)

// NativeParser interface for parsing binary documents.
//
// Direct port from legacy pkg/context/extraction/binary_extractor.go
type NativeParser interface {
	ParseDocument(ctx context.Context, filePath string, fileSize int64) (*NativeParseResult, error)
}

// NativeParseResult represents the result from a native parser.
//
// Direct port from legacy pkg/context/extraction/binary_extractor.go
type NativeParseResult struct {
	Success          bool
	Content          string
	Title            string
	Author           string
	Metadata         map[string]string
	Error            string
	ProcessingTimeMs int64
}

// BinaryExtractor handles binary files like PDF, DOCX, XLSX using native parsers.
//
// Direct port from legacy pkg/context/extraction/binary_extractor.go
type BinaryExtractor struct {
	nativeParsers NativeParser
}

// NewBinaryExtractor creates a new binary extractor.
func NewBinaryExtractor(nativeParsers NativeParser) *BinaryExtractor {
	return &BinaryExtractor{
		nativeParsers: nativeParsers,
	}
}

// Name returns the extractor name.
func (be *BinaryExtractor) Name() string {
	return "BinaryExtractor"
}

// CanExtract checks if this extractor can handle the file.
func (be *BinaryExtractor) CanExtract(path string, mimeType string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	binaryExtensions := map[string]bool{
		".pdf":  true,
		".docx": true,
		".xlsx": true,
	}
	return binaryExtensions[ext]
}

// Extract uses native parsers to extract content from binary files.
func (be *BinaryExtractor) Extract(ctx context.Context, path string, fileSize int64) (*ExtractedContent, error) {
	startTime := time.Now()

	result, err := be.nativeParsers.ParseDocument(ctx, path, fileSize)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, nil
	}

	metadata := make(map[string]string)
	if result.Metadata != nil {
		for k, v := range result.Metadata {
			metadata[k] = v
		}
	}

	return &ExtractedContent{
		Content:          result.Content,
		Title:            result.Title,
		Author:           result.Author,
		Metadata:         metadata,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// Priority returns medium priority (5).
func (be *BinaryExtractor) Priority() int {
	return 5
}

// Ensure BinaryExtractor implements ContentExtractor.
var _ ContentExtractor = (*BinaryExtractor)(nil)
