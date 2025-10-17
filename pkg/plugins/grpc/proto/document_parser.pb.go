// Code generated manually for document parser plugin.
// TODO: Replace with proper protoc-generated code when protoc is available.
// source: document_parser.proto

package proto

import (
	"context"
)

// ParseDocumentRequest requests parsing of a document
type ParseDocumentRequest struct {
	FilePath       string            `json:"file_path"`
	FileSize       int64             `json:"file_size"`
	MimeType       string            `json:"mime_type"`
	Config         map[string]string `json:"config"`
	TimeoutSeconds int32             `json:"timeout_seconds"`
}

// ParseDocumentResponse returns the parsed document content
type ParseDocumentResponse struct {
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
	Warnings         []string          `json:"warnings"`
	ProcessingTimeMs int64             `json:"processing_time_ms"`
}

// GetSupportedExtensionsRequest requests supported file extensions
type GetSupportedExtensionsRequest struct{}

// GetSupportedExtensionsResponse returns supported extensions and MIME types
type GetSupportedExtensionsResponse struct {
	Extensions []string `json:"extensions"`
	MimeTypes  []string `json:"mime_types"`
}

// DocumentParserServiceClient is a simplified client interface
type DocumentParserServiceClient interface {
	ParseDocument(ctx context.Context, req *ParseDocumentRequest) (*ParseDocumentResponse, error)
	GetSupportedExtensions(ctx context.Context, req *GetSupportedExtensionsRequest) (*GetSupportedExtensionsResponse, error)
}

// DocumentParserServiceServer is a simplified server interface
type DocumentParserServiceServer interface {
	ParseDocument(ctx context.Context, req *ParseDocumentRequest) (*ParseDocumentResponse, error)
	GetSupportedExtensions(ctx context.Context, req *GetSupportedExtensionsRequest) (*GetSupportedExtensionsResponse, error)
}

// UnimplementedDocumentParserServiceServer provides default implementations
type UnimplementedDocumentParserServiceServer struct{}

func (UnimplementedDocumentParserServiceServer) ParseDocument(ctx context.Context, req *ParseDocumentRequest) (*ParseDocumentResponse, error) {
	return &ParseDocumentResponse{
		Success: false,
		Error:   "method ParseDocument not implemented",
	}, nil
}

func (UnimplementedDocumentParserServiceServer) GetSupportedExtensions(ctx context.Context, req *GetSupportedExtensionsRequest) (*GetSupportedExtensionsResponse, error) {
	return &GetSupportedExtensionsResponse{
		Extensions: []string{},
		MimeTypes:  []string{},
	}, nil
}
