package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

const _ = grpc.SupportPackageIsVersion9

const (
	DatabaseProvider_Initialize_FullMethodName       = "/hector.plugin.DatabaseProvider/Initialize"
	DatabaseProvider_Shutdown_FullMethodName         = "/hector.plugin.DatabaseProvider/Shutdown"
	DatabaseProvider_Health_FullMethodName           = "/hector.plugin.DatabaseProvider/Health"
	DatabaseProvider_GetManifest_FullMethodName      = "/hector.plugin.DatabaseProvider/GetManifest"
	DatabaseProvider_GetStatus_FullMethodName        = "/hector.plugin.DatabaseProvider/GetStatus"
	DatabaseProvider_Upsert_FullMethodName           = "/hector.plugin.DatabaseProvider/Upsert"
	DatabaseProvider_Search_FullMethodName           = "/hector.plugin.DatabaseProvider/Search"
	DatabaseProvider_Delete_FullMethodName           = "/hector.plugin.DatabaseProvider/Delete"
	DatabaseProvider_CreateCollection_FullMethodName = "/hector.plugin.DatabaseProvider/CreateCollection"
	DatabaseProvider_DeleteCollection_FullMethodName = "/hector.plugin.DatabaseProvider/DeleteCollection"
)

type DatabaseProviderClient interface {
	Initialize(ctx context.Context, in *InitializeRequest, opts ...grpc.CallOption) (*InitializeResponse, error)

	Shutdown(ctx context.Context, in *ShutdownRequest, opts ...grpc.CallOption) (*ShutdownResponse, error)

	Health(ctx context.Context, in *HealthRequest, opts ...grpc.CallOption) (*HealthResponse, error)

	GetManifest(ctx context.Context, in *ManifestRequest, opts ...grpc.CallOption) (*ManifestResponse, error)

	GetStatus(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)

	Upsert(ctx context.Context, in *UpsertRequest, opts ...grpc.CallOption) (*UpsertResponse, error)

	Search(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error)

	Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error)

	CreateCollection(ctx context.Context, in *CreateCollectionRequest, opts ...grpc.CallOption) (*CreateCollectionResponse, error)

	DeleteCollection(ctx context.Context, in *DeleteCollectionRequest, opts ...grpc.CallOption) (*DeleteCollectionResponse, error)
}

type databaseProviderClient struct {
	cc grpc.ClientConnInterface
}

func NewDatabaseProviderClient(cc grpc.ClientConnInterface) DatabaseProviderClient {
	return &databaseProviderClient{cc}
}

func (c *databaseProviderClient) Initialize(ctx context.Context, in *InitializeRequest, opts ...grpc.CallOption) (*InitializeResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(InitializeResponse)
	err := c.cc.Invoke(ctx, DatabaseProvider_Initialize_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *databaseProviderClient) Shutdown(ctx context.Context, in *ShutdownRequest, opts ...grpc.CallOption) (*ShutdownResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ShutdownResponse)
	err := c.cc.Invoke(ctx, DatabaseProvider_Shutdown_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *databaseProviderClient) Health(ctx context.Context, in *HealthRequest, opts ...grpc.CallOption) (*HealthResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(HealthResponse)
	err := c.cc.Invoke(ctx, DatabaseProvider_Health_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *databaseProviderClient) GetManifest(ctx context.Context, in *ManifestRequest, opts ...grpc.CallOption) (*ManifestResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ManifestResponse)
	err := c.cc.Invoke(ctx, DatabaseProvider_GetManifest_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *databaseProviderClient) GetStatus(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, DatabaseProvider_GetStatus_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *databaseProviderClient) Upsert(ctx context.Context, in *UpsertRequest, opts ...grpc.CallOption) (*UpsertResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UpsertResponse)
	err := c.cc.Invoke(ctx, DatabaseProvider_Upsert_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *databaseProviderClient) Search(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SearchResponse)
	err := c.cc.Invoke(ctx, DatabaseProvider_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *databaseProviderClient) Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DeleteResponse)
	err := c.cc.Invoke(ctx, DatabaseProvider_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *databaseProviderClient) CreateCollection(ctx context.Context, in *CreateCollectionRequest, opts ...grpc.CallOption) (*CreateCollectionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CreateCollectionResponse)
	err := c.cc.Invoke(ctx, DatabaseProvider_CreateCollection_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *databaseProviderClient) DeleteCollection(ctx context.Context, in *DeleteCollectionRequest, opts ...grpc.CallOption) (*DeleteCollectionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DeleteCollectionResponse)
	err := c.cc.Invoke(ctx, DatabaseProvider_DeleteCollection_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type DatabaseProviderServer interface {
	Initialize(context.Context, *InitializeRequest) (*InitializeResponse, error)

	Shutdown(context.Context, *ShutdownRequest) (*ShutdownResponse, error)

	Health(context.Context, *HealthRequest) (*HealthResponse, error)

	GetManifest(context.Context, *ManifestRequest) (*ManifestResponse, error)

	GetStatus(context.Context, *StatusRequest) (*StatusResponse, error)

	Upsert(context.Context, *UpsertRequest) (*UpsertResponse, error)

	Search(context.Context, *SearchRequest) (*SearchResponse, error)

	Delete(context.Context, *DeleteRequest) (*DeleteResponse, error)

	CreateCollection(context.Context, *CreateCollectionRequest) (*CreateCollectionResponse, error)

	DeleteCollection(context.Context, *DeleteCollectionRequest) (*DeleteCollectionResponse, error)
	mustEmbedUnimplementedDatabaseProviderServer()
}

type UnimplementedDatabaseProviderServer struct{}

func (UnimplementedDatabaseProviderServer) Initialize(context.Context, *InitializeRequest) (*InitializeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Initialize not implemented")
}
func (UnimplementedDatabaseProviderServer) Shutdown(context.Context, *ShutdownRequest) (*ShutdownResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Shutdown not implemented")
}
func (UnimplementedDatabaseProviderServer) Health(context.Context, *HealthRequest) (*HealthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Health not implemented")
}
func (UnimplementedDatabaseProviderServer) GetManifest(context.Context, *ManifestRequest) (*ManifestResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetManifest not implemented")
}
func (UnimplementedDatabaseProviderServer) GetStatus(context.Context, *StatusRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStatus not implemented")
}
func (UnimplementedDatabaseProviderServer) Upsert(context.Context, *UpsertRequest) (*UpsertResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Upsert not implemented")
}
func (UnimplementedDatabaseProviderServer) Search(context.Context, *SearchRequest) (*SearchResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedDatabaseProviderServer) Delete(context.Context, *DeleteRequest) (*DeleteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedDatabaseProviderServer) CreateCollection(context.Context, *CreateCollectionRequest) (*CreateCollectionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateCollection not implemented")
}
func (UnimplementedDatabaseProviderServer) DeleteCollection(context.Context, *DeleteCollectionRequest) (*DeleteCollectionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteCollection not implemented")
}
func (UnimplementedDatabaseProviderServer) mustEmbedUnimplementedDatabaseProviderServer() {}
func (UnimplementedDatabaseProviderServer) testEmbeddedByValue()                          {}

type UnsafeDatabaseProviderServer interface {
	mustEmbedUnimplementedDatabaseProviderServer()
}

func RegisterDatabaseProviderServer(s grpc.ServiceRegistrar, srv DatabaseProviderServer) {

	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&DatabaseProvider_ServiceDesc, srv)
}

func _DatabaseProvider_Initialize_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitializeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatabaseProviderServer).Initialize(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DatabaseProvider_Initialize_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatabaseProviderServer).Initialize(ctx, req.(*InitializeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DatabaseProvider_Shutdown_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ShutdownRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatabaseProviderServer).Shutdown(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DatabaseProvider_Shutdown_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatabaseProviderServer).Shutdown(ctx, req.(*ShutdownRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DatabaseProvider_Health_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HealthRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatabaseProviderServer).Health(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DatabaseProvider_Health_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatabaseProviderServer).Health(ctx, req.(*HealthRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DatabaseProvider_GetManifest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ManifestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatabaseProviderServer).GetManifest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DatabaseProvider_GetManifest_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatabaseProviderServer).GetManifest(ctx, req.(*ManifestRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DatabaseProvider_GetStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatabaseProviderServer).GetStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DatabaseProvider_GetStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatabaseProviderServer).GetStatus(ctx, req.(*StatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DatabaseProvider_Upsert_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpsertRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatabaseProviderServer).Upsert(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DatabaseProvider_Upsert_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatabaseProviderServer).Upsert(ctx, req.(*UpsertRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DatabaseProvider_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SearchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatabaseProviderServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DatabaseProvider_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatabaseProviderServer).Search(ctx, req.(*SearchRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DatabaseProvider_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatabaseProviderServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DatabaseProvider_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatabaseProviderServer).Delete(ctx, req.(*DeleteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DatabaseProvider_CreateCollection_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateCollectionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatabaseProviderServer).CreateCollection(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DatabaseProvider_CreateCollection_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatabaseProviderServer).CreateCollection(ctx, req.(*CreateCollectionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DatabaseProvider_DeleteCollection_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteCollectionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatabaseProviderServer).DeleteCollection(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DatabaseProvider_DeleteCollection_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatabaseProviderServer).DeleteCollection(ctx, req.(*DeleteCollectionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var DatabaseProvider_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "hector.plugin.DatabaseProvider",
	HandlerType: (*DatabaseProviderServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Initialize",
			Handler:    _DatabaseProvider_Initialize_Handler,
		},
		{
			MethodName: "Shutdown",
			Handler:    _DatabaseProvider_Shutdown_Handler,
		},
		{
			MethodName: "Health",
			Handler:    _DatabaseProvider_Health_Handler,
		},
		{
			MethodName: "GetManifest",
			Handler:    _DatabaseProvider_GetManifest_Handler,
		},
		{
			MethodName: "GetStatus",
			Handler:    _DatabaseProvider_GetStatus_Handler,
		},
		{
			MethodName: "Upsert",
			Handler:    _DatabaseProvider_Upsert_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _DatabaseProvider_Search_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _DatabaseProvider_Delete_Handler,
		},
		{
			MethodName: "CreateCollection",
			Handler:    _DatabaseProvider_CreateCollection_Handler,
		},
		{
			MethodName: "DeleteCollection",
			Handler:    _DatabaseProvider_DeleteCollection_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "database.proto",
}
