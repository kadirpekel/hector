package context

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

// ============================================================================
// NATIVE BINARY DOCUMENT PARSERS
// ============================================================================
// These parsers handle binary file formats that cannot be read as plain text

// NativeParserResult represents the result of native document parsing
type NativeParserResult struct {
	Success          bool              `json:"success"`
	Content          string            `json:"content"`
	Title            string            `json:"title"`
	Author           string            `json:"author"`
	Created          string            `json:"created"`
	Modified         string            `json:"modified"`
	Pages            int32             `json:"pages"`
	WordCount        int32             `json:"word_count"`
	Metadata         map[string]string `json:"metadata"`
	Error            string            `json:"error"`
	ProcessingTimeMs int64             `json:"processing_time_ms"`
}

// NativeParser interface for binary document parsers
type NativeParser interface {
	// CanParse returns true if this parser can handle the given file
	CanParse(filePath string) bool

	// Parse extracts text content from a binary document
	Parse(ctx context.Context, filePath string, fileSize int64) (*NativeParserResult, error)

	// GetSupportedExtensions returns the file extensions this parser supports
	GetSupportedExtensions() []string
}

// ============================================================================
// NATIVE PARSER REGISTRY
// ============================================================================

// NativeParserRegistry manages native binary document parsers
type NativeParserRegistry struct {
	parsers []NativeParser
}

// NewNativeParserRegistry creates a new native parser registry
func NewNativeParserRegistry() *NativeParserRegistry {
	registry := &NativeParserRegistry{
		parsers: make([]NativeParser, 0),
	}

	// Register built-in parsers
	registry.registerBuiltinParsers()

	return registry
}

// registerBuiltinParsers registers all built-in native parsers
func (r *NativeParserRegistry) registerBuiltinParsers() {
	// Register parsers in order of preference (most reliable first)
	r.parsers = append(r.parsers, &PDFParser{})
	r.parsers = append(r.parsers, &OfficeParser{}) // Handles .docx, .xlsx
}

// FindParser finds a parser that can handle the given file
func (r *NativeParserRegistry) FindParser(filePath string) NativeParser {
	for _, parser := range r.parsers {
		if parser.CanParse(filePath) {
			return parser
		}
	}
	return nil
}

// ParseDocument attempts to parse a document using native parsers
func (r *NativeParserRegistry) ParseDocument(ctx context.Context, filePath string, fileSize int64) (*NativeParserResult, error) {
	parser := r.FindParser(filePath)
	if parser == nil {
		return &NativeParserResult{
			Success: false,
			Error:   fmt.Sprintf("no native parser available for file: %s", filepath.Ext(filePath)),
		}, nil
	}

	return parser.Parse(ctx, filePath, fileSize)
}

// GetSupportedExtensions returns all supported file extensions
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

// ============================================================================
// PDF PARSER
// ============================================================================

// PDFParser handles PDF document parsing
type PDFParser struct{}

// CanParse returns true if the file is a PDF
func (p *PDFParser) CanParse(filePath string) bool {
	return strings.ToLower(filepath.Ext(filePath)) == ".pdf"
}

// GetSupportedExtensions returns PDF extensions
func (p *PDFParser) GetSupportedExtensions() []string {
	return []string{".pdf"}
}

// Parse extracts text content from a PDF file
func (p *PDFParser) Parse(ctx context.Context, filePath string, fileSize int64) (*NativeParserResult, error) {
	startTime := time.Now()

	// Read PDF file
	file, err := os.Open(filePath)
	if err != nil {
		return &NativeParserResult{
			Success:          false,
			Error:            fmt.Sprintf("Failed to open PDF file: %v", err),
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		}, nil
	}
	defer file.Close()

	// Parse PDF
	reader, err := pdf.NewReader(file, fileSize)
	if err != nil {
		return &NativeParserResult{
			Success:          false,
			Error:            fmt.Sprintf("Failed to parse PDF: %v", err),
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		}, nil
	}

	// Extract text content
	var contentParts []string
	totalPages := reader.NumPage()

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
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

	// Extract metadata
	metadata := p.extractPDFMetadata(reader, filePath)

	// Calculate word count
	wordCount := int32(len(strings.Fields(content)))

	processingTime := time.Since(startTime).Milliseconds()

	return &NativeParserResult{
		Success:          true,
		Content:          content,
		Title:            metadata["title"],
		Author:           metadata["author"],
		Created:          metadata["created"],
		Modified:         metadata["modified"],
		Pages:            int32(totalPages),
		WordCount:        wordCount,
		Metadata:         metadata,
		ProcessingTimeMs: processingTime,
	}, nil
}

// extractPDFMetadata extracts metadata from PDF
func (p *PDFParser) extractPDFMetadata(reader *pdf.Reader, filePath string) map[string]string {
	metadata := make(map[string]string)

	// Basic metadata
	metadata["pages"] = fmt.Sprintf("%d", reader.NumPage())

	// File metadata
	if fileInfo, err := os.Stat(filePath); err == nil {
		metadata["file_size"] = fmt.Sprintf("%d", fileInfo.Size())
		metadata["file_modified"] = fileInfo.ModTime().Format(time.RFC3339)
	}

	// PDF metadata (if available)
	// Note: The ledongthuc/pdf library doesn't expose metadata directly
	// We'll use the filename as title for now
	metadata["title"] = filepath.Base(filePath)

	return metadata
}

// ============================================================================
// OFFICE PARSER (Word, PowerPoint, Excel)
// ============================================================================

// OfficeParser handles Microsoft Office document parsing (.docx, .xlsx)
type OfficeParser struct{}

// CanParse returns true if the file is an Office document
func (p *OfficeParser) CanParse(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".docx" || ext == ".xlsx"
}

// GetSupportedExtensions returns Office document extensions
func (p *OfficeParser) GetSupportedExtensions() []string {
	return []string{".docx", ".xlsx"}
}

// Parse extracts text content from an Office document
func (p *OfficeParser) Parse(ctx context.Context, filePath string, fileSize int64) (*NativeParserResult, error) {
	startTime := time.Now()
	ext := strings.ToLower(filepath.Ext(filePath))

	var content string
	var title string
	var author string
	var pages int32
	var metadata map[string]string

	switch ext {
	case ".docx":
		content, title, author, pages, metadata = p.parseWordDocument(filePath)
	case ".xlsx":
		content, title, author, pages, metadata = p.parseExcelDocument(filePath)
	default:
		return &NativeParserResult{
			Success:          false,
			Error:            fmt.Sprintf("unsupported Office format: %s", ext),
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		}, nil
	}

	// Calculate word count
	wordCount := int32(len(strings.Fields(content)))
	processingTime := time.Since(startTime).Milliseconds()

	return &NativeParserResult{
		Success:          true,
		Content:          content,
		Title:            title,
		Author:           author,
		Created:          metadata["created"],
		Modified:         metadata["modified"],
		Pages:            pages,
		WordCount:        wordCount,
		Metadata:         metadata,
		ProcessingTimeMs: processingTime,
	}, nil
}

// parseWordDocument extracts text from a Word document
func (p *OfficeParser) parseWordDocument(filePath string) (string, string, string, int32, map[string]string) {
	// Open and read the Word document
	doc, err := docx.ReadDocxFile(filePath)
	if err != nil {
		return fmt.Sprintf("Error parsing Word document: %v", err), filepath.Base(filePath), "", 0, make(map[string]string)
	}
	defer doc.Close()

	// Extract text content
	content := doc.Editable().GetContent()
	title := filepath.Base(filePath)
	metadata := make(map[string]string)

	// Count paragraphs (approximate)
	paragraphCount := int32(len(strings.Split(content, "\n\n")))

	// Extract basic metadata
	metadata["title"] = title
	metadata["type"] = "Word Document"

	return content, title, "", paragraphCount, metadata
}

// parseExcelDocument extracts text from an Excel spreadsheet
func (p *OfficeParser) parseExcelDocument(filePath string) (string, string, string, int32, map[string]string) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Sprintf("Error parsing Excel document: %v", err), filepath.Base(filePath), "", 0, make(map[string]string)
	}
	defer f.Close()

	var contentParts []string
	title := filepath.Base(filePath)
	metadata := make(map[string]string)
	var sheetCount int32

	// Get all sheet names
	sheets := f.GetSheetList()
	sheetCount = int32(len(sheets))

	// Extract text from each sheet
	for _, sheetName := range sheets {
		var sheetText strings.Builder
		sheetText.WriteString(fmt.Sprintf("--- Sheet: %s ---\n", sheetName))

		// Get all rows in the sheet
		rows, err := f.GetRows(sheetName)
		if err != nil {
			sheetText.WriteString(fmt.Sprintf("Error reading sheet: %v\n", err))
			continue
		}

		// Extract text from cells (limit to first 1000 cells to avoid huge content)
		cellCount := 0
		for rowIndex, row := range rows {
			if cellCount >= 1000 {
				sheetText.WriteString("... (truncated)\n")
				break
			}
			for colIndex, cell := range row {
				if cellCount >= 1000 {
					break
				}
				if text := strings.TrimSpace(cell); text != "" {
					cellRef := fmt.Sprintf("%s%d", string(rune('A'+colIndex)), rowIndex+1)
					sheetText.WriteString(fmt.Sprintf("%s: %s\n", cellRef, text))
					cellCount++
				}
			}
		}

		if text := strings.TrimSpace(sheetText.String()); text != "" {
			contentParts = append(contentParts, text)
		}
	}

	// Extract basic metadata
	metadata["title"] = title
	metadata["type"] = "Excel Spreadsheet"

	content := strings.Join(contentParts, "\n\n")
	return content, title, "", sheetCount, metadata
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// isBinaryFileType returns true if the file type requires binary parsing
func isBinaryFileType(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	binaryExtensions := map[string]bool{
		".pdf":  true,
		".docx": true,
		".xlsx": true,
	}
	return binaryExtensions[ext]
}

// getFileMimeType returns the MIME type for a file based on its extension
func getFileMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	mimeTypes := map[string]string{
		".pdf":  "application/pdf",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}

	if mimeType, exists := mimeTypes[ext]; exists {
		return mimeType
	}
	return "application/octet-stream"
}
