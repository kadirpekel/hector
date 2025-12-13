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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
	"github.com/xuri/excelize/v2"
)

// NativeParserRegistry manages native document parsers for PDF, DOCX, XLSX.
//
// Ported from legacy pkg/context/native_parsers.go
type NativeParserRegistry struct {
	parsers []nativeParserImpl
}

// nativeParserImpl is the internal interface for individual parsers.
type nativeParserImpl interface {
	CanParse(filePath string) bool
	Parse(ctx context.Context, filePath string, fileSize int64) (*NativeParseResult, error)
	GetSupportedExtensions() []string
}

// NewNativeParserRegistry creates a new native parser registry with built-in parsers.
func NewNativeParserRegistry() *NativeParserRegistry {
	registry := &NativeParserRegistry{
		parsers: make([]nativeParserImpl, 0),
	}

	// Register built-in parsers
	registry.parsers = append(registry.parsers, &pdfParser{})
	registry.parsers = append(registry.parsers, &officeParser{})

	return registry
}

// ParseDocument finds the appropriate parser and extracts content.
// Implements NativeParser interface.
func (r *NativeParserRegistry) ParseDocument(ctx context.Context, filePath string, fileSize int64) (*NativeParseResult, error) {
	parser := r.findParser(filePath)
	if parser == nil {
		return &NativeParseResult{
			Success: false,
			Error:   fmt.Sprintf("no native parser available for file: %s", filepath.Ext(filePath)),
		}, nil
	}

	return parser.Parse(ctx, filePath, fileSize)
}

// findParser returns the appropriate parser for the file.
func (r *NativeParserRegistry) findParser(filePath string) nativeParserImpl {
	for _, parser := range r.parsers {
		if parser.CanParse(filePath) {
			return parser
		}
	}
	return nil
}

// GetSupportedExtensions returns all supported file extensions.
func (r *NativeParserRegistry) GetSupportedExtensions() []string {
	extensions := make(map[string]bool)

	for _, parser := range r.parsers {
		for _, ext := range parser.GetSupportedExtensions() {
			extensions[ext] = true
		}
	}

	result := make([]string, 0, len(extensions))
	for ext := range extensions {
		result = append(result, ext)
	}

	return result
}

// Ensure NativeParserRegistry implements NativeParser.
var _ NativeParser = (*NativeParserRegistry)(nil)

// =============================================================================
// PDF Parser
// =============================================================================

// pdfParser handles PDF document extraction.
type pdfParser struct{}

func (p *pdfParser) CanParse(filePath string) bool {
	return strings.ToLower(filepath.Ext(filePath)) == ".pdf"
}

func (p *pdfParser) GetSupportedExtensions() []string {
	return []string{".pdf"}
}

func (p *pdfParser) Parse(ctx context.Context, filePath string, fileSize int64) (*NativeParseResult, error) {
	startTime := time.Now()

	file, err := os.Open(filePath)
	if err != nil {
		return &NativeParseResult{
			Success:          false,
			Error:            fmt.Sprintf("failed to open PDF file: %v", err),
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		}, nil
	}
	defer file.Close()

	reader, err := pdf.NewReader(file, fileSize)
	if err != nil {
		return &NativeParseResult{
			Success:          false,
			Error:            fmt.Sprintf("failed to parse PDF: %v", err),
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		}, nil
	}

	var contentParts []string
	totalPages := reader.NumPage()

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return &NativeParseResult{
				Success:          false,
				Error:            "context cancelled",
				ProcessingTimeMs: time.Since(startTime).Milliseconds(),
			}, ctx.Err()
		default:
		}

		page := reader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			contentParts = append(contentParts, fmt.Sprintf("--- Page %d (extraction failed: %v) ---", pageNum, err))
			continue
		}

		if strings.TrimSpace(text) != "" {
			contentParts = append(contentParts, fmt.Sprintf("--- Page %d ---\n%s", pageNum, text))
		}
	}

	content := strings.Join(contentParts, "\n\n")
	metadata := p.extractMetadata(reader, filePath)
	metadata["word_count"] = fmt.Sprintf("%d", len(strings.Fields(content)))

	return &NativeParseResult{
		Success:          true,
		Content:          content,
		Title:            metadata["title"],
		Author:           metadata["author"],
		Metadata:         metadata,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	}, nil
}

func (p *pdfParser) extractMetadata(reader *pdf.Reader, filePath string) map[string]string {
	metadata := make(map[string]string)

	metadata["pages"] = fmt.Sprintf("%d", reader.NumPage())
	metadata["type"] = "PDF Document"

	if fileInfo, err := os.Stat(filePath); err == nil {
		metadata["file_size"] = fmt.Sprintf("%d", fileInfo.Size())
		metadata["file_modified"] = fileInfo.ModTime().Format(time.RFC3339)
	}

	// Use filename as title
	metadata["title"] = filepath.Base(filePath)

	return metadata
}

// =============================================================================
// Office Parser (DOCX, XLSX)
// =============================================================================

// officeParser handles Word and Excel documents.
type officeParser struct{}

func (p *officeParser) CanParse(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".docx" || ext == ".xlsx"
}

func (p *officeParser) GetSupportedExtensions() []string {
	return []string{".docx", ".xlsx"}
}

func (p *officeParser) Parse(ctx context.Context, filePath string, fileSize int64) (*NativeParseResult, error) {
	startTime := time.Now()
	ext := strings.ToLower(filepath.Ext(filePath))

	var content string
	var title string
	var author string
	var metadata map[string]string

	switch ext {
	case ".docx":
		content, title, author, metadata = p.parseWordDocument(filePath)
	case ".xlsx":
		content, title, author, metadata = p.parseExcelDocument(ctx, filePath)
	default:
		return &NativeParseResult{
			Success:          false,
			Error:            fmt.Sprintf("unsupported Office format: %s", ext),
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		}, nil
	}

	return &NativeParseResult{
		Success:          true,
		Content:          content,
		Title:            title,
		Author:           author,
		Metadata:         metadata,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	}, nil
}

func (p *officeParser) parseWordDocument(filePath string) (string, string, string, map[string]string) {
	doc, err := docx.ReadDocxFile(filePath)
	if err != nil {
		return fmt.Sprintf("Error parsing Word document: %v", err), filepath.Base(filePath), "", make(map[string]string)
	}
	defer doc.Close()

	content := doc.Editable().GetContent()
	title := filepath.Base(filePath)
	metadata := make(map[string]string)

	metadata["title"] = title
	metadata["type"] = "Word Document"
	metadata["paragraphs"] = fmt.Sprintf("%d", len(strings.Split(content, "\n\n")))

	return content, title, "", metadata
}

func (p *officeParser) parseExcelDocument(ctx context.Context, filePath string) (string, string, string, map[string]string) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Sprintf("Error parsing Excel document: %v", err), filepath.Base(filePath), "", make(map[string]string)
	}
	defer f.Close()

	var contentParts []string
	title := filepath.Base(filePath)
	metadata := make(map[string]string)

	sheets := f.GetSheetList()
	metadata["sheets"] = fmt.Sprintf("%d", len(sheets))
	metadata["title"] = title
	metadata["type"] = "Excel Spreadsheet"

	for _, sheetName := range sheets {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return strings.Join(contentParts, "\n\n"), title, "", metadata
		default:
		}

		var sheetText strings.Builder
		sheetText.WriteString(fmt.Sprintf("--- Sheet: %s ---\n", sheetName))

		rows, err := f.GetRows(sheetName)
		if err != nil {
			sheetText.WriteString(fmt.Sprintf("Error reading sheet: %v\n", err))
			continue
		}

		cellCount := 0
		maxCells := 1000 // Limit cells per sheet to avoid huge outputs

		for rowIndex, row := range rows {
			if cellCount >= maxCells {
				sheetText.WriteString("... (truncated)\n")
				break
			}
			for colIndex, cell := range row {
				if cellCount >= maxCells {
					break
				}
				if text := strings.TrimSpace(cell); text != "" {
					cellRef := fmt.Sprintf("%s%d", columnLetter(colIndex), rowIndex+1)
					sheetText.WriteString(fmt.Sprintf("%s: %s\n", cellRef, text))
					cellCount++
				}
			}
		}

		if text := strings.TrimSpace(sheetText.String()); text != "" {
			contentParts = append(contentParts, text)
		}
	}

	content := strings.Join(contentParts, "\n\n")
	return content, title, "", metadata
}

// columnLetter converts a 0-based column index to Excel column letter (A, B, ..., Z, AA, AB, ...).
func columnLetter(index int) string {
	result := ""
	for {
		result = string(rune('A'+index%26)) + result
		index = index/26 - 1
		if index < 0 {
			break
		}
	}
	return result
}
