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

type NativeParser interface {
	CanParse(filePath string) bool

	Parse(ctx context.Context, filePath string, fileSize int64) (*NativeParserResult, error)

	GetSupportedExtensions() []string
}

type NativeParserRegistry struct {
	parsers []NativeParser
}

func NewNativeParserRegistry() *NativeParserRegistry {
	registry := &NativeParserRegistry{
		parsers: make([]NativeParser, 0),
	}

	registry.registerBuiltinParsers()

	return registry
}

func (r *NativeParserRegistry) registerBuiltinParsers() {

	r.parsers = append(r.parsers, &PDFParser{})
	r.parsers = append(r.parsers, &OfficeParser{})
}

func (r *NativeParserRegistry) FindParser(filePath string) NativeParser {
	for _, parser := range r.parsers {
		if parser.CanParse(filePath) {
			return parser
		}
	}
	return nil
}

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

type PDFParser struct{}

func (p *PDFParser) CanParse(filePath string) bool {
	return strings.ToLower(filepath.Ext(filePath)) == ".pdf"
}

func (p *PDFParser) GetSupportedExtensions() []string {
	return []string{".pdf"}
}

func (p *PDFParser) Parse(ctx context.Context, filePath string, fileSize int64) (*NativeParserResult, error) {
	startTime := time.Now()

	file, err := os.Open(filePath)
	if err != nil {
		return &NativeParserResult{
			Success:          false,
			Error:            fmt.Sprintf("Failed to open PDF file: %v", err),
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		}, nil
	}
	defer file.Close()

	reader, err := pdf.NewReader(file, fileSize)
	if err != nil {
		return &NativeParserResult{
			Success:          false,
			Error:            fmt.Sprintf("Failed to parse PDF: %v", err),
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		}, nil
	}

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

	metadata := p.extractPDFMetadata(reader, filePath)

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

func (p *PDFParser) extractPDFMetadata(reader *pdf.Reader, filePath string) map[string]string {
	metadata := make(map[string]string)

	metadata["pages"] = fmt.Sprintf("%d", reader.NumPage())

	if fileInfo, err := os.Stat(filePath); err == nil {
		metadata["file_size"] = fmt.Sprintf("%d", fileInfo.Size())
		metadata["file_modified"] = fileInfo.ModTime().Format(time.RFC3339)
	}

	metadata["title"] = filepath.Base(filePath)

	return metadata
}

type OfficeParser struct{}

func (p *OfficeParser) CanParse(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".docx" || ext == ".xlsx"
}

func (p *OfficeParser) GetSupportedExtensions() []string {
	return []string{".docx", ".xlsx"}
}

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

func (p *OfficeParser) parseWordDocument(filePath string) (string, string, string, int32, map[string]string) {

	doc, err := docx.ReadDocxFile(filePath)
	if err != nil {
		return fmt.Sprintf("Error parsing Word document: %v", err), filepath.Base(filePath), "", 0, make(map[string]string)
	}
	defer doc.Close()

	content := doc.Editable().GetContent()
	title := filepath.Base(filePath)
	metadata := make(map[string]string)

	paragraphCount := int32(len(strings.Split(content, "\n\n")))

	metadata["title"] = title
	metadata["type"] = "Word Document"

	return content, title, "", paragraphCount, metadata
}

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

	sheets := f.GetSheetList()
	sheetCount = int32(len(sheets))

	for _, sheetName := range sheets {
		var sheetText strings.Builder
		sheetText.WriteString(fmt.Sprintf("--- Sheet: %s ---\n", sheetName))

		rows, err := f.GetRows(sheetName)
		if err != nil {
			sheetText.WriteString(fmt.Sprintf("Error reading sheet: %v\n", err))
			continue
		}

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

	metadata["title"] = title
	metadata["type"] = "Excel Spreadsheet"

	content := strings.Join(contentParts, "\n\n")
	return content, title, "", sheetCount, metadata
}

func isBinaryFileType(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	binaryExtensions := map[string]bool{
		".pdf":  true,
		".docx": true,
		".xlsx": true,
	}
	return binaryExtensions[ext]
}
