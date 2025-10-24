package proto

import (
	"context"

	"google.golang.org/grpc"
)

func RegisterDocumentParserServiceServer(s *grpc.Server, srv DocumentParserServiceServer) {

}

func NewDocumentParserServiceClient(cc grpc.ClientConnInterface) DocumentParserServiceClient {
	return &documentParserServiceClient{cc}
}

type documentParserServiceClient struct {
	cc grpc.ClientConnInterface
}

func (c *documentParserServiceClient) ParseDocument(ctx context.Context, req *ParseDocumentRequest) (*ParseDocumentResponse, error) {

	return &ParseDocumentResponse{
		Success: false,
		Error:   "gRPC client not implemented",
	}, nil
}

func (c *documentParserServiceClient) GetSupportedExtensions(ctx context.Context, req *GetSupportedExtensionsRequest) (*GetSupportedExtensionsResponse, error) {

	return &GetSupportedExtensionsResponse{
		Extensions: []string{},
		MimeTypes:  []string{},
	}, nil
}
