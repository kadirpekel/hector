package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

const _ = grpc.SupportPackageIsVersion9

const (
	EmbedderProvider_Initialize_FullMethodName      = "/hector.plugin.EmbedderProvider/Initialize"
	EmbedderProvider_Shutdown_FullMethodName        = "/hector.plugin.EmbedderProvider/Shutdown"
	EmbedderProvider_Health_FullMethodName          = "/hector.plugin.EmbedderProvider/Health"
	EmbedderProvider_GetManifest_FullMethodName     = "/hector.plugin.EmbedderProvider/GetManifest"
	EmbedderProvider_GetStatus_FullMethodName       = "/hector.plugin.EmbedderProvider/GetStatus"
	EmbedderProvider_Embed_FullMethodName           = "/hector.plugin.EmbedderProvider/Embed"
	EmbedderProvider_GetEmbedderInfo_FullMethodName = "/hector.plugin.EmbedderProvider/GetEmbedderInfo"
)

type EmbedderProviderClient interface {
	Initialize(ctx context.Context, in *InitializeRequest, opts ...grpc.CallOption) (*InitializeResponse, error)

	Shutdown(ctx context.Context, in *ShutdownRequest, opts ...grpc.CallOption) (*ShutdownResponse, error)

	Health(ctx context.Context, in *HealthRequest, opts ...grpc.CallOption) (*HealthResponse, error)

	GetManifest(ctx context.Context, in *ManifestRequest, opts ...grpc.CallOption) (*ManifestResponse, error)

	GetStatus(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)

	Embed(ctx context.Context, in *EmbedRequest, opts ...grpc.CallOption) (*EmbedResponse, error)

	GetEmbedderInfo(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*EmbedderInfo, error)
}

type embedderProviderClient struct {
	cc grpc.ClientConnInterface
}

func NewEmbedderProviderClient(cc grpc.ClientConnInterface) EmbedderProviderClient {
	return &embedderProviderClient{cc}
}

func (c *embedderProviderClient) Initialize(ctx context.Context, in *InitializeRequest, opts ...grpc.CallOption) (*InitializeResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(InitializeResponse)
	err := c.cc.Invoke(ctx, EmbedderProvider_Initialize_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *embedderProviderClient) Shutdown(ctx context.Context, in *ShutdownRequest, opts ...grpc.CallOption) (*ShutdownResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ShutdownResponse)
	err := c.cc.Invoke(ctx, EmbedderProvider_Shutdown_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *embedderProviderClient) Health(ctx context.Context, in *HealthRequest, opts ...grpc.CallOption) (*HealthResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(HealthResponse)
	err := c.cc.Invoke(ctx, EmbedderProvider_Health_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *embedderProviderClient) GetManifest(ctx context.Context, in *ManifestRequest, opts ...grpc.CallOption) (*ManifestResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ManifestResponse)
	err := c.cc.Invoke(ctx, EmbedderProvider_GetManifest_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *embedderProviderClient) GetStatus(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, EmbedderProvider_GetStatus_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *embedderProviderClient) Embed(ctx context.Context, in *EmbedRequest, opts ...grpc.CallOption) (*EmbedResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(EmbedResponse)
	err := c.cc.Invoke(ctx, EmbedderProvider_Embed_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *embedderProviderClient) GetEmbedderInfo(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*EmbedderInfo, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(EmbedderInfo)
	err := c.cc.Invoke(ctx, EmbedderProvider_GetEmbedderInfo_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type EmbedderProviderServer interface {
	Initialize(context.Context, *InitializeRequest) (*InitializeResponse, error)

	Shutdown(context.Context, *ShutdownRequest) (*ShutdownResponse, error)

	Health(context.Context, *HealthRequest) (*HealthResponse, error)

	GetManifest(context.Context, *ManifestRequest) (*ManifestResponse, error)

	GetStatus(context.Context, *StatusRequest) (*StatusResponse, error)

	Embed(context.Context, *EmbedRequest) (*EmbedResponse, error)

	GetEmbedderInfo(context.Context, *Empty) (*EmbedderInfo, error)
	mustEmbedUnimplementedEmbedderProviderServer()
}

type UnimplementedEmbedderProviderServer struct{}

func (UnimplementedEmbedderProviderServer) Initialize(context.Context, *InitializeRequest) (*InitializeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Initialize not implemented")
}
func (UnimplementedEmbedderProviderServer) Shutdown(context.Context, *ShutdownRequest) (*ShutdownResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Shutdown not implemented")
}
func (UnimplementedEmbedderProviderServer) Health(context.Context, *HealthRequest) (*HealthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Health not implemented")
}
func (UnimplementedEmbedderProviderServer) GetManifest(context.Context, *ManifestRequest) (*ManifestResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetManifest not implemented")
}
func (UnimplementedEmbedderProviderServer) GetStatus(context.Context, *StatusRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStatus not implemented")
}
func (UnimplementedEmbedderProviderServer) Embed(context.Context, *EmbedRequest) (*EmbedResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Embed not implemented")
}
func (UnimplementedEmbedderProviderServer) GetEmbedderInfo(context.Context, *Empty) (*EmbedderInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetEmbedderInfo not implemented")
}
func (UnimplementedEmbedderProviderServer) mustEmbedUnimplementedEmbedderProviderServer() {}
func (UnimplementedEmbedderProviderServer) testEmbeddedByValue()                          {}

type UnsafeEmbedderProviderServer interface {
	mustEmbedUnimplementedEmbedderProviderServer()
}

func RegisterEmbedderProviderServer(s grpc.ServiceRegistrar, srv EmbedderProviderServer) {

	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&EmbedderProvider_ServiceDesc, srv)
}

func _EmbedderProvider_Initialize_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitializeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EmbedderProviderServer).Initialize(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EmbedderProvider_Initialize_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EmbedderProviderServer).Initialize(ctx, req.(*InitializeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EmbedderProvider_Shutdown_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ShutdownRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EmbedderProviderServer).Shutdown(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EmbedderProvider_Shutdown_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EmbedderProviderServer).Shutdown(ctx, req.(*ShutdownRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EmbedderProvider_Health_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HealthRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EmbedderProviderServer).Health(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EmbedderProvider_Health_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EmbedderProviderServer).Health(ctx, req.(*HealthRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EmbedderProvider_GetManifest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ManifestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EmbedderProviderServer).GetManifest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EmbedderProvider_GetManifest_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EmbedderProviderServer).GetManifest(ctx, req.(*ManifestRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EmbedderProvider_GetStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EmbedderProviderServer).GetStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EmbedderProvider_GetStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EmbedderProviderServer).GetStatus(ctx, req.(*StatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EmbedderProvider_Embed_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EmbedRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EmbedderProviderServer).Embed(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EmbedderProvider_Embed_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EmbedderProviderServer).Embed(ctx, req.(*EmbedRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EmbedderProvider_GetEmbedderInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EmbedderProviderServer).GetEmbedderInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EmbedderProvider_GetEmbedderInfo_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EmbedderProviderServer).GetEmbedderInfo(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

var EmbedderProvider_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "hector.plugin.EmbedderProvider",
	HandlerType: (*EmbedderProviderServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Initialize",
			Handler:    _EmbedderProvider_Initialize_Handler,
		},
		{
			MethodName: "Shutdown",
			Handler:    _EmbedderProvider_Shutdown_Handler,
		},
		{
			MethodName: "Health",
			Handler:    _EmbedderProvider_Health_Handler,
		},
		{
			MethodName: "GetManifest",
			Handler:    _EmbedderProvider_GetManifest_Handler,
		},
		{
			MethodName: "GetStatus",
			Handler:    _EmbedderProvider_GetStatus_Handler,
		},
		{
			MethodName: "Embed",
			Handler:    _EmbedderProvider_Embed_Handler,
		},
		{
			MethodName: "GetEmbedderInfo",
			Handler:    _EmbedderProvider_GetEmbedderInfo_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "embedder.proto",
}
