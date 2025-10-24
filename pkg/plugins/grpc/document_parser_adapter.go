package grpc

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-plugin"
	"github.com/kadirpekel/hector/pkg/plugins"
	pb "github.com/kadirpekel/hector/pkg/plugins/grpc/proto"
	"google.golang.org/grpc"
)

type DocumentParserPluginAdapter struct {
	provider DocumentParserProvider
	manifest *plugins.PluginManifest
	client   *plugin.Client
	status   plugins.PluginStatus
}

func NewDocumentParserPluginAdapter(provider DocumentParserProvider, manifest *plugins.PluginManifest, client *plugin.Client) *DocumentParserPluginAdapter {
	return &DocumentParserPluginAdapter{
		provider: provider,
		manifest: manifest,
		client:   client,
		status:   plugins.StatusReady,
	}
}

func (a *DocumentParserPluginAdapter) Initialize(ctx context.Context, config map[string]interface{}) error {
	a.status = plugins.StatusLoading

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

func (a *DocumentParserPluginAdapter) Shutdown(ctx context.Context) error {
	a.status = plugins.StatusShutdown
	return a.provider.Shutdown(ctx)
}

func (a *DocumentParserPluginAdapter) GetManifest() *plugins.PluginManifest {
	return a.manifest
}

func (a *DocumentParserPluginAdapter) GetStatus() plugins.PluginStatus {
	return a.status
}

func (a *DocumentParserPluginAdapter) Health(ctx context.Context) error {
	return a.provider.Health(ctx)
}

func (a *DocumentParserPluginAdapter) ParseDocument(ctx context.Context, filePath string, fileSize int64, mimeType string, config map[string]string) (*pb.ParseDocumentResponse, error) {
	return a.provider.ParseDocument(ctx, filePath, fileSize, mimeType, config)
}

func (a *DocumentParserPluginAdapter) GetSupportedExtensions(ctx context.Context) (*pb.GetSupportedExtensionsResponse, error) {
	return a.provider.GetSupportedExtensions(ctx)
}

type DocumentParserProviderPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	impl DocumentParserProvider
}

func (p *DocumentParserProviderPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pb.RegisterDocumentParserServiceServer(s, &DocumentParserGRPCServer{
		impl: p.impl,
	})
	return nil
}

func (p *DocumentParserProviderPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &DocumentParserGRPCClient{
		client: pb.NewDocumentParserServiceClient(c),
	}, nil
}

type DocumentParserGRPCServer struct {
	pb.UnimplementedDocumentParserServiceServer
	impl DocumentParserProvider
}

func (s *DocumentParserGRPCServer) ParseDocument(ctx context.Context, req *pb.ParseDocumentRequest) (*pb.ParseDocumentResponse, error) {
	return s.impl.ParseDocument(ctx, req.FilePath, req.FileSize, req.MimeType, req.Config)
}

func (s *DocumentParserGRPCServer) GetSupportedExtensions(ctx context.Context, req *pb.GetSupportedExtensionsRequest) (*pb.GetSupportedExtensionsResponse, error) {
	return s.impl.GetSupportedExtensions(ctx)
}

type DocumentParserGRPCClient struct {
	client pb.DocumentParserServiceClient
}

func (c *DocumentParserGRPCClient) ParseDocument(ctx context.Context, filePath string, fileSize int64, mimeType string, config map[string]string) (*pb.ParseDocumentResponse, error) {
	req := &pb.ParseDocumentRequest{
		FilePath: filePath,
		FileSize: fileSize,
		MimeType: mimeType,
		Config:   config,
	}
	return c.client.ParseDocument(ctx, req)
}

func (c *DocumentParserGRPCClient) GetSupportedExtensions(ctx context.Context) (*pb.GetSupportedExtensionsResponse, error) {
	req := &pb.GetSupportedExtensionsRequest{}
	return c.client.GetSupportedExtensions(ctx, req)
}
