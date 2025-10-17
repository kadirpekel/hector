// Code generated manually for document parser plugin.
// TODO: Replace with proper protoc-generated code when protoc is available.
// source: document_parser.proto

package proto

import (
	"context"

	"google.golang.org/grpc"
)

// RegisterDocumentParserServiceServer registers the document parser service
func RegisterDocumentParserServiceServer(s *grpc.Server, srv DocumentParserServiceServer) {
	// This is a simplified registration - in a real gRPC implementation,
	// this would register the service with the gRPC server
}

// NewDocumentParserServiceClient creates a new document parser service client
func NewDocumentParserServiceClient(cc grpc.ClientConnInterface) DocumentParserServiceClient {
	return &documentParserServiceClient{cc}
}

// documentParserServiceClient is a simplified client implementation
type documentParserServiceClient struct {
	cc grpc.ClientConnInterface
}

func (c *documentParserServiceClient) ParseDocument(ctx context.Context, req *ParseDocumentRequest) (*ParseDocumentResponse, error) {
	// This is a simplified implementation - in a real gRPC client,
	// this would make an actual gRPC call
	return &ParseDocumentResponse{
		Success: false,
		Error:   "gRPC client not implemented",
	}, nil
}

func (c *documentParserServiceClient) GetSupportedExtensions(ctx context.Context, req *GetSupportedExtensionsRequest) (*GetSupportedExtensionsResponse, error) {
	// This is a simplified implementation - in a real gRPC client,
	// this would make an actual gRPC call
	return &GetSupportedExtensionsResponse{
		Extensions: []string{},
		MimeTypes:  []string{},
	}, nil
}
