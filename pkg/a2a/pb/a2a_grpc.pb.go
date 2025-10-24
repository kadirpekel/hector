

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion9

const (
	A2AService_SendMessage_FullMethodName                      = "/a2a.v1.A2AService/SendMessage"
	A2AService_SendStreamingMessage_FullMethodName             = "/a2a.v1.A2AService/SendStreamingMessage"
	A2AService_GetTask_FullMethodName                          = "/a2a.v1.A2AService/GetTask"
	A2AService_CancelTask_FullMethodName                       = "/a2a.v1.A2AService/CancelTask"
	A2AService_TaskSubscription_FullMethodName                 = "/a2a.v1.A2AService/TaskSubscription"
	A2AService_CreateTaskPushNotificationConfig_FullMethodName = "/a2a.v1.A2AService/CreateTaskPushNotificationConfig"
	A2AService_GetTaskPushNotificationConfig_FullMethodName    = "/a2a.v1.A2AService/GetTaskPushNotificationConfig"
	A2AService_ListTaskPushNotificationConfig_FullMethodName   = "/a2a.v1.A2AService/ListTaskPushNotificationConfig"
	A2AService_GetAgentCard_FullMethodName                     = "/a2a.v1.A2AService/GetAgentCard"
	A2AService_DeleteTaskPushNotificationConfig_FullMethodName = "/a2a.v1.A2AService/DeleteTaskPushNotificationConfig"
)

type A2AServiceClient interface {
	SendMessage(ctx context.Context, in *SendMessageRequest, opts ...grpc.CallOption) (*SendMessageResponse, error)
	SendStreamingMessage(ctx context.Context, in *SendMessageRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[StreamResponse], error)
	GetTask(ctx context.Context, in *GetTaskRequest, opts ...grpc.CallOption) (*Task, error)
	CancelTask(ctx context.Context, in *CancelTaskRequest, opts ...grpc.CallOption) (*Task, error)
	TaskSubscription(ctx context.Context, in *TaskSubscriptionRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[StreamResponse], error)
	CreateTaskPushNotificationConfig(ctx context.Context, in *CreateTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*TaskPushNotificationConfig, error)
	GetTaskPushNotificationConfig(ctx context.Context, in *GetTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*TaskPushNotificationConfig, error)
	ListTaskPushNotificationConfig(ctx context.Context, in *ListTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*ListTaskPushNotificationConfigResponse, error)
	GetAgentCard(ctx context.Context, in *GetAgentCardRequest, opts ...grpc.CallOption) (*AgentCard, error)
	DeleteTaskPushNotificationConfig(ctx context.Context, in *DeleteTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type a2AServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewA2AServiceClient(cc grpc.ClientConnInterface) A2AServiceClient {
	return &a2AServiceClient{cc}
}

func (c *a2AServiceClient) SendMessage(ctx context.Context, in *SendMessageRequest, opts ...grpc.CallOption) (*SendMessageResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SendMessageResponse)
	err := c.cc.Invoke(ctx, A2AService_SendMessage_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *a2AServiceClient) SendStreamingMessage(ctx context.Context, in *SendMessageRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[StreamResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &A2AService_ServiceDesc.Streams[0], A2AService_SendStreamingMessage_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[SendMessageRequest, StreamResponse]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type A2AService_SendStreamingMessageClient = grpc.ServerStreamingClient[StreamResponse]

func (c *a2AServiceClient) GetTask(ctx context.Context, in *GetTaskRequest, opts ...grpc.CallOption) (*Task, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Task)
	err := c.cc.Invoke(ctx, A2AService_GetTask_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *a2AServiceClient) CancelTask(ctx context.Context, in *CancelTaskRequest, opts ...grpc.CallOption) (*Task, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Task)
	err := c.cc.Invoke(ctx, A2AService_CancelTask_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *a2AServiceClient) TaskSubscription(ctx context.Context, in *TaskSubscriptionRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[StreamResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &A2AService_ServiceDesc.Streams[1], A2AService_TaskSubscription_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[TaskSubscriptionRequest, StreamResponse]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type A2AService_TaskSubscriptionClient = grpc.ServerStreamingClient[StreamResponse]

func (c *a2AServiceClient) CreateTaskPushNotificationConfig(ctx context.Context, in *CreateTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*TaskPushNotificationConfig, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(TaskPushNotificationConfig)
	err := c.cc.Invoke(ctx, A2AService_CreateTaskPushNotificationConfig_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *a2AServiceClient) GetTaskPushNotificationConfig(ctx context.Context, in *GetTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*TaskPushNotificationConfig, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(TaskPushNotificationConfig)
	err := c.cc.Invoke(ctx, A2AService_GetTaskPushNotificationConfig_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *a2AServiceClient) ListTaskPushNotificationConfig(ctx context.Context, in *ListTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*ListTaskPushNotificationConfigResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListTaskPushNotificationConfigResponse)
	err := c.cc.Invoke(ctx, A2AService_ListTaskPushNotificationConfig_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *a2AServiceClient) GetAgentCard(ctx context.Context, in *GetAgentCardRequest, opts ...grpc.CallOption) (*AgentCard, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(AgentCard)
	err := c.cc.Invoke(ctx, A2AService_GetAgentCard_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *a2AServiceClient) DeleteTaskPushNotificationConfig(ctx context.Context, in *DeleteTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, A2AService_DeleteTaskPushNotificationConfig_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type A2AServiceServer interface {
	SendMessage(context.Context, *SendMessageRequest) (*SendMessageResponse, error)
	SendStreamingMessage(*SendMessageRequest, grpc.ServerStreamingServer[StreamResponse]) error
	GetTask(context.Context, *GetTaskRequest) (*Task, error)
	CancelTask(context.Context, *CancelTaskRequest) (*Task, error)
	TaskSubscription(*TaskSubscriptionRequest, grpc.ServerStreamingServer[StreamResponse]) error
	CreateTaskPushNotificationConfig(context.Context, *CreateTaskPushNotificationConfigRequest) (*TaskPushNotificationConfig, error)
	GetTaskPushNotificationConfig(context.Context, *GetTaskPushNotificationConfigRequest) (*TaskPushNotificationConfig, error)
	ListTaskPushNotificationConfig(context.Context, *ListTaskPushNotificationConfigRequest) (*ListTaskPushNotificationConfigResponse, error)
	GetAgentCard(context.Context, *GetAgentCardRequest) (*AgentCard, error)
	DeleteTaskPushNotificationConfig(context.Context, *DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error)
	mustEmbedUnimplementedA2AServiceServer()
}

type UnimplementedA2AServiceServer struct{}

func (UnimplementedA2AServiceServer) SendMessage(context.Context, *SendMessageRequest) (*SendMessageResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SendMessage not implemented")
}
func (UnimplementedA2AServiceServer) SendStreamingMessage(*SendMessageRequest, grpc.ServerStreamingServer[StreamResponse]) error {
	return status.Errorf(codes.Unimplemented, "method SendStreamingMessage not implemented")
}
func (UnimplementedA2AServiceServer) GetTask(context.Context, *GetTaskRequest) (*Task, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTask not implemented")
}
func (UnimplementedA2AServiceServer) CancelTask(context.Context, *CancelTaskRequest) (*Task, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CancelTask not implemented")
}
func (UnimplementedA2AServiceServer) TaskSubscription(*TaskSubscriptionRequest, grpc.ServerStreamingServer[StreamResponse]) error {
	return status.Errorf(codes.Unimplemented, "method TaskSubscription not implemented")
}
func (UnimplementedA2AServiceServer) CreateTaskPushNotificationConfig(context.Context, *CreateTaskPushNotificationConfigRequest) (*TaskPushNotificationConfig, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateTaskPushNotificationConfig not implemented")
}
func (UnimplementedA2AServiceServer) GetTaskPushNotificationConfig(context.Context, *GetTaskPushNotificationConfigRequest) (*TaskPushNotificationConfig, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTaskPushNotificationConfig not implemented")
}
func (UnimplementedA2AServiceServer) ListTaskPushNotificationConfig(context.Context, *ListTaskPushNotificationConfigRequest) (*ListTaskPushNotificationConfigResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListTaskPushNotificationConfig not implemented")
}
func (UnimplementedA2AServiceServer) GetAgentCard(context.Context, *GetAgentCardRequest) (*AgentCard, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAgentCard not implemented")
}
func (UnimplementedA2AServiceServer) DeleteTaskPushNotificationConfig(context.Context, *DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteTaskPushNotificationConfig not implemented")
}
func (UnimplementedA2AServiceServer) mustEmbedUnimplementedA2AServiceServer() {}
func (UnimplementedA2AServiceServer) testEmbeddedByValue()                    {}

type UnsafeA2AServiceServer interface {
	mustEmbedUnimplementedA2AServiceServer()
}

func RegisterA2AServiceServer(s grpc.ServiceRegistrar, srv A2AServiceServer) {
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&A2AService_ServiceDesc, srv)
}

func _A2AService_SendMessage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SendMessageRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(A2AServiceServer).SendMessage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: A2AService_SendMessage_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(A2AServiceServer).SendMessage(ctx, req.(*SendMessageRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _A2AService_SendStreamingMessage_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SendMessageRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(A2AServiceServer).SendStreamingMessage(m, &grpc.GenericServerStream[SendMessageRequest, StreamResponse]{ServerStream: stream})
}

type A2AService_SendStreamingMessageServer = grpc.ServerStreamingServer[StreamResponse]

func _A2AService_GetTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTaskRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(A2AServiceServer).GetTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: A2AService_GetTask_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(A2AServiceServer).GetTask(ctx, req.(*GetTaskRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _A2AService_CancelTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CancelTaskRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(A2AServiceServer).CancelTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: A2AService_CancelTask_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(A2AServiceServer).CancelTask(ctx, req.(*CancelTaskRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _A2AService_TaskSubscription_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(TaskSubscriptionRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(A2AServiceServer).TaskSubscription(m, &grpc.GenericServerStream[TaskSubscriptionRequest, StreamResponse]{ServerStream: stream})
}

type A2AService_TaskSubscriptionServer = grpc.ServerStreamingServer[StreamResponse]

func _A2AService_CreateTaskPushNotificationConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateTaskPushNotificationConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(A2AServiceServer).CreateTaskPushNotificationConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: A2AService_CreateTaskPushNotificationConfig_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(A2AServiceServer).CreateTaskPushNotificationConfig(ctx, req.(*CreateTaskPushNotificationConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _A2AService_GetTaskPushNotificationConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTaskPushNotificationConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(A2AServiceServer).GetTaskPushNotificationConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: A2AService_GetTaskPushNotificationConfig_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(A2AServiceServer).GetTaskPushNotificationConfig(ctx, req.(*GetTaskPushNotificationConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _A2AService_ListTaskPushNotificationConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListTaskPushNotificationConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(A2AServiceServer).ListTaskPushNotificationConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: A2AService_ListTaskPushNotificationConfig_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(A2AServiceServer).ListTaskPushNotificationConfig(ctx, req.(*ListTaskPushNotificationConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _A2AService_GetAgentCard_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetAgentCardRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(A2AServiceServer).GetAgentCard(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: A2AService_GetAgentCard_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(A2AServiceServer).GetAgentCard(ctx, req.(*GetAgentCardRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _A2AService_DeleteTaskPushNotificationConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteTaskPushNotificationConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(A2AServiceServer).DeleteTaskPushNotificationConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: A2AService_DeleteTaskPushNotificationConfig_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(A2AServiceServer).DeleteTaskPushNotificationConfig(ctx, req.(*DeleteTaskPushNotificationConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var A2AService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "a2a.v1.A2AService",
	HandlerType: (*A2AServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SendMessage",
			Handler:    _A2AService_SendMessage_Handler,
		},
		{
			MethodName: "GetTask",
			Handler:    _A2AService_GetTask_Handler,
		},
		{
			MethodName: "CancelTask",
			Handler:    _A2AService_CancelTask_Handler,
		},
		{
			MethodName: "CreateTaskPushNotificationConfig",
			Handler:    _A2AService_CreateTaskPushNotificationConfig_Handler,
		},
		{
			MethodName: "GetTaskPushNotificationConfig",
			Handler:    _A2AService_GetTaskPushNotificationConfig_Handler,
		},
		{
			MethodName: "ListTaskPushNotificationConfig",
			Handler:    _A2AService_ListTaskPushNotificationConfig_Handler,
		},
		{
			MethodName: "GetAgentCard",
			Handler:    _A2AService_GetAgentCard_Handler,
		},
		{
			MethodName: "DeleteTaskPushNotificationConfig",
			Handler:    _A2AService_DeleteTaskPushNotificationConfig_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "SendStreamingMessage",
			Handler:       _A2AService_SendStreamingMessage_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "TaskSubscription",
			Handler:       _A2AService_TaskSubscription_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "a2a.proto",
}
