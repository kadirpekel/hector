package proto

import (
	"context"
)

type ParseDocumentRequest struct {
	FilePath       string            `json:"file_path"`
	FileSize       int64             `json:"file_size"`
	MimeType       string            `json:"mime_type"`
	Config         map[string]string `json:"config"`
	TimeoutSeconds int32             `json:"timeout_seconds"`
}

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

type GetSupportedExtensionsRequest struct{}

type GetSupportedExtensionsResponse struct {
	Extensions []string `json:"extensions"`
	MimeTypes  []string `json:"mime_types"`
}

type DocumentParserServiceClient interface {
	ParseDocument(ctx context.Context, req *ParseDocumentRequest) (*ParseDocumentResponse, error)
	GetSupportedExtensions(ctx context.Context, req *GetSupportedExtensionsRequest) (*GetSupportedExtensionsResponse, error)
}

type DocumentParserServiceServer interface {
	ParseDocument(ctx context.Context, req *ParseDocumentRequest) (*ParseDocumentResponse, error)
	GetSupportedExtensions(ctx context.Context, req *GetSupportedExtensionsRequest) (*GetSupportedExtensionsResponse, error)
}

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
