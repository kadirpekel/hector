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
func DefaultDirectorySourceConfig(path string) DirectorySourceConfig {
	return DirectorySourceConfig{
		Path:        path,
		Include:     []string{"*.md", "*.txt", "*.go", "*.py", "*.java", "*.js", "*.ts", "*.json", "*.yaml", "*.yml"},
		Exclude:     []string{".git", "node_modules", "vendor", "dist", "build", "__pycache__", ".venv", "venv"},
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
