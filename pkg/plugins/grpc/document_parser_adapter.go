package grpc

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-plugin"
	"github.com/kadirpekel/hector/pkg/plugins"
	pb "github.com/kadirpekel/hector/pkg/plugins/grpc/proto"
	"google.golang.org/grpc"
)

// ============================================================================
// DOCUMENT PARSER PLUGIN ADAPTER
// ============================================================================

// DocumentParserPluginAdapter adapts a DocumentParserProvider to the Plugin interface
type DocumentParserPluginAdapter struct {
	provider DocumentParserProvider
	manifest *plugins.PluginManifest
	client   *plugin.Client
	status   plugins.PluginStatus
}

// NewDocumentParserPluginAdapter creates a new document parser plugin adapter
func NewDocumentParserPluginAdapter(provider DocumentParserProvider, manifest *plugins.PluginManifest, client *plugin.Client) *DocumentParserPluginAdapter {
	return &DocumentParserPluginAdapter{
		provider: provider,
		manifest: manifest,
		client:   client,
		status:   plugins.StatusReady,
	}
}

// Initialize initializes the document parser plugin
func (a *DocumentParserPluginAdapter) Initialize(ctx context.Context, config map[string]interface{}) error {
	a.status = plugins.StatusLoading

	// Convert config to string map
	stringConfig := make(map[string]string)
	for k, v := range config {
		if str, ok := v.(string); ok {
			stringConfig[k] = str
		} else {
			stringConfig[k] = fmt.Sprintf("%v", v)
		}
	}

	err := a.provider.Initialize(ctx, stringConfig)
	if err != nil {
		a.status = plugins.StatusError
		return err
	}

	a.status = plugins.StatusReady
	return nil
}

// Shutdown shuts down the document parser plugin
func (a *DocumentParserPluginAdapter) Shutdown(ctx context.Context) error {
	a.status = plugins.StatusShutdown
	return a.provider.Shutdown(ctx)
}

// GetManifest returns the plugin manifest
func (a *DocumentParserPluginAdapter) GetManifest() *plugins.PluginManifest {
	return a.manifest
}

// GetStatus returns the current plugin status
func (a *DocumentParserPluginAdapter) GetStatus() plugins.PluginStatus {
	return a.status
}

// Health checks if the document parser plugin is healthy
func (a *DocumentParserPluginAdapter) Health(ctx context.Context) error {
	return a.provider.Health(ctx)
}

// ParseDocument parses a document using the plugin
func (a *DocumentParserPluginAdapter) ParseDocument(ctx context.Context, filePath string, fileSize int64, mimeType string, config map[string]string) (*pb.ParseDocumentResponse, error) {
	return a.provider.ParseDocument(ctx, filePath, fileSize, mimeType, config)
}

// GetSupportedExtensions returns the file extensions supported by this parser
func (a *DocumentParserPluginAdapter) GetSupportedExtensions(ctx context.Context) (*pb.GetSupportedExtensionsResponse, error) {
	return a.provider.GetSupportedExtensions(ctx)
}

// ============================================================================
// DOCUMENT PARSER PROVIDER PLUGIN (gRPC SERVER SIDE)
// ============================================================================

// DocumentParserProviderPlugin implements the gRPC server side for document parser plugins
type DocumentParserProviderPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	impl DocumentParserProvider
}

// GRPCServer returns the gRPC server for document parser plugins
func (p *DocumentParserProviderPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pb.RegisterDocumentParserServiceServer(s, &DocumentParserGRPCServer{
		impl: p.impl,
	})
	return nil
}

// GRPCClient returns the gRPC client for document parser plugins
func (p *DocumentParserProviderPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &DocumentParserGRPCClient{
		client: pb.NewDocumentParserServiceClient(c),
	}, nil
}

// ============================================================================
// GRPC SERVER IMPLEMENTATION
// ============================================================================

// DocumentParserGRPCServer implements the gRPC server for document parser plugins
type DocumentParserGRPCServer struct {
	pb.UnimplementedDocumentParserServiceServer
	impl DocumentParserProvider
}

// ParseDocument implements the ParseDocument gRPC method
func (s *DocumentParserGRPCServer) ParseDocument(ctx context.Context, req *pb.ParseDocumentRequest) (*pb.ParseDocumentResponse, error) {
	return s.impl.ParseDocument(ctx, req.FilePath, req.FileSize, req.MimeType, req.Config)
}

// GetSupportedExtensions implements the GetSupportedExtensions gRPC method
func (s *DocumentParserGRPCServer) GetSupportedExtensions(ctx context.Context, req *pb.GetSupportedExtensionsRequest) (*pb.GetSupportedExtensionsResponse, error) {
	return s.impl.GetSupportedExtensions(ctx)
}

// ============================================================================
// GRPC CLIENT IMPLEMENTATION
// ============================================================================

// DocumentParserGRPCClient implements the gRPC client for document parser plugins
type DocumentParserGRPCClient struct {
	client pb.DocumentParserServiceClient
}

// ParseDocument implements the ParseDocument method via gRPC
func (c *DocumentParserGRPCClient) ParseDocument(ctx context.Context, filePath string, fileSize int64, mimeType string, config map[string]string) (*pb.ParseDocumentResponse, error) {
	req := &pb.ParseDocumentRequest{
		FilePath: filePath,
		FileSize: fileSize,
		MimeType: mimeType,
		Config:   config,
	}
	return c.client.ParseDocument(ctx, req)
}

// GetSupportedExtensions implements the GetSupportedExtensions method via gRPC
func (c *DocumentParserGRPCClient) GetSupportedExtensions(ctx context.Context) (*pb.GetSupportedExtensionsResponse, error) {
	req := &pb.GetSupportedExtensionsRequest{}
	return c.client.GetSupportedExtensions(ctx, req)
}

