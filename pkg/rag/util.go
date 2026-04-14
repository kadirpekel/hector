package rag

import (
	"path/filepath"
	"strings"
)

// detectMimeType attempts to detect the MIME type of a file based on its extension.
func detectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "text/x-go"
	case ".md", ".markdown":
		return "text/markdown"
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "application/x-yaml"
	case ".js":
		return "application/javascript"
	case ".ts", ".tsx":
		return "application/typescript"
	case ".py":
		return "text/x-python"
	case ".java":
		return "text/x-java"
	case ".c", ".h":
		return "text/x-c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "text/x-c++"
	case ".rs":
		return "text/x-rust"
	case ".rb":
		return "text/x-ruby"
	case ".php":
		return "text/x-php"
	case ".sh", ".bash":
		return "text/x-shellscript"
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".xml":
		return "application/xml"
	case ".csv":
		return "text/csv"
	case ".sql":
		return "application/sql"
	case ".pdf":
		return "application/pdf"
	case ".doc", ".docx":
		return "application/msword"
	case ".xls", ".xlsx":
		return "application/vnd.ms-excel"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}

// DocumentEvent represents a change in a document.
type DocumentEvent struct {
	Type     DocumentEventType
	Document Document
	Error    error
}

// DocumentEventType indicates the type of change.
type DocumentEventType string

const (
	DocumentEventCreate DocumentEventType = "create"
	DocumentEventUpdate DocumentEventType = "update"
	DocumentEventDelete DocumentEventType = "delete"
	DocumentEventError  DocumentEventType = "error"
)

// DefaultDirectorySourceConfig returns sensible defaults for directory source.
// Includes both text-based source code files and binary document formats
// that can be parsed by native parsers (PDF, DOCX, XLSX).
func DefaultDirectorySourceConfig(path string) DirectorySourceConfig {
	return DirectorySourceConfig{
		Path: path,
		Include: []string{
			// Text-based source code and documentation
			"*.md", "*.txt", "*.rst", "*.adoc",
			// Programming languages
			"*.go", "*.py", "*.java", "*.js", "*.ts", "*.jsx", "*.tsx",
			"*.c", "*.cpp", "*.h", "*.hpp", "*.cs", "*.rb", "*.php",
			"*.rs", "*.swift", "*.kt", "*.scala", "*.lua", "*.r",
			// Config and data files
			"*.json", "*.yaml", "*.yml", "*.toml", "*.xml", "*.html", "*.css",
			// Binary document formats (native parsers)
			"*.pdf", "*.docx", "*.xlsx",
		},
		Exclude:     []string{".git", ".hector", "node_modules", "vendor", "dist", "build", "__pycache__", ".venv", "venv"},
		MaxFileSize: 10 * 1024 * 1024, // 10 MB
	}
}

// DirectorySourceConfig configures a directory data source.
type DirectorySourceConfig struct {
	Path        string
	Include     []string
	Exclude     []string
	MaxFileSize int64 // Max file size in bytes to process (0 for no limit)
}
